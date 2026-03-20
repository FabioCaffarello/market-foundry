# Analytical Runtime Lifecycle and Recovery

> Consolidated from 6 source documents (archived in docs/archive/analytical/).
> Sources: analytical-pipeline-lifecycle-degraded-dead-recovered.md, analytical-pipeline-recovery-and-supervision.md, analytical-runtime-activation-rules-and-failure-modes.md, analytical-runtime-optionality-rules.md, analytical-retry-backoff-and-loss-semantics.md, analytical-failure-handling-and-overflow-model.md

---

## 1. Activation Model

The analytical runtime is **optional and additive**. It activates only when explicitly configured and does not affect the baseline pipeline (ingest -> derive -> store -> execute).

### Writer Activation

The writer binary activates when:
1. A config file is provided with `clickhouse.addr` set
2. `nats.enabled` is `true` with a valid `nats.url`
3. At least one pipeline family is enabled

If any precondition is not met, the writer **exits immediately** with a structured error message.

### Gateway Analytical Endpoints

The gateway activates analytical endpoints when:
1. `clickhouse.addr` is configured
2. The ClickHouse config passes structural validation
3. A connection to ClickHouse succeeds

If any condition fails, the gateway **continues normally** without analytical endpoints. Baseline readiness is not affected.

### Validation Order

The writer validates configuration in strict order:
```
NATS enabled -> ClickHouse config -> Pipeline config -> Log summary -> Open connections
```

All config errors are reported before any connection attempt. The operator sees the maximum number of actionable issues per restart cycle.

---

## 2. Optionality Rules (Invariants)

These rules ensure ClickHouse remains optional throughout all phases of integration.

| Rule | Description |
|------|-------------|
| R-01 | No operational service depends on ClickHouse (not in depends_on, readiness, imports) |
| R-02 | No readiness check references ClickHouse (except writer) |
| R-03 | No event path blocks on ClickHouse availability |
| R-04 | Writer uses independent consumer names (`writer-*` prefix, never shared with `store-*`) |
| R-05 | Writer tolerates ClickHouse absence (buffers, drops on overflow, resumes on recovery) |
| R-06 | Smoke tests pass without ClickHouse and writer |
| R-07 | No conditional behavior in operational services based on ClickHouse |
| R-08 | Historical endpoints are additive (new routes only, existing routes unchanged) |
| R-09 | Cold-start bootstrap is opportunistic (non-blocking, with timeout) |
| R-10 | Configuration lifecycle has no ClickHouse dependency |

### ClickHouse Awareness by Component

| Binary | ClickHouse | Behavior |
|--------|-----------|----------|
| writer | Required | Hard exit on missing or invalid config |
| gateway | Optional | Graceful degradation; analytical endpoints disabled |
| derive, store, ingest, execute, configctl | Not used | ClickHouse config section ignored |

### Violation Severity

| Violation | Severity |
|-----------|----------|
| Operational service imports ClickHouse driver | Critical -- block merge |
| Operational readiness checks ClickHouse | Critical -- block merge |
| Writer shares consumer name with store | Critical -- block merge |
| Smoke tests fail without ClickHouse | Critical -- block merge |
| Existing route behavior changes with CH state | High -- block merge |

---

## 3. Pipeline Lifecycle States

### Active

Consumer and inserter running normally. Events flow from NATS through consumer, are buffered, and batch-inserted into ClickHouse.

**Observable signals:** `event_count` incrementing, `events_flushed` incrementing, `/statusz` phase: `"active"`.

### Restarting

Consumer failed to start. Supervisor has poisoned old actors and scheduled a restart after backoff delay.

**Observable signals:** `pipeline_restarts` incremented, `error_count` incremented, WARN log with `"pipeline failure -- scheduling restart"`.

**Exit conditions:** Restart succeeds -> Active. Restart fails -> re-enters Restarting (if budget remains) or Degraded.

### Degraded

Family exhausted its restart budget (5 attempts). Supervisor stopped trying. Events accumulate in NATS JetStream (bounded by stream retention, typically 72h).

**Observable signals:** `pipeline_degraded` = 1, ERROR log `"pipeline degraded -- restart budget exhausted"`, `/statusz` phase: `"degraded"`.

**Terminal per process lifetime.** Recovery requires fixing root cause and restarting the writer process.

### State Diagram

```
                startup
                  |
                  v
              +--------+
              | Active |<-----------------+
              +---+----+                  |
                  |                       |
          consumer failure          restart succeeds
                  |                       |
                  v                       |
            +-----------+                 |
       +--->|Restarting |---> (success) --+
       |    +-----+-----+
       |          |
 restart fails    | budget exhausted
 (budget remains) |
       |          v
       |    +----------+
       +----| Degraded |  (terminal per process)
            +----------+
```

### Distinguishing Degraded from Dead

| Attribute | Degraded | Dead (process crash) |
|-----------|----------|---------------------|
| Other families | Running | All stopped |
| Health endpoints | Responding | Unreachable |
| Recovery | Fix root cause + restart process | Docker auto-restart |
| Restart budget | Exhausted | Reset on process restart |

---

## 4. Supervision and Recovery

### Supervisor-Managed Restart

The `writerSupervisor` actor owns all pipeline families. On consumer `pipelineFailedMsg`:

1. Records failure in family lifecycle state
2. Increments restart counter in health tracker
3. Poisons the failed consumer and paired inserter
4. Schedules restart after exponential backoff
5. Spawns fresh consumer-inserter pair

### Backoff Schedule (Pipeline Recovery)

| Restart | Delay |
|---------|-------|
| 1 | 2s |
| 2 | 4s |
| 3 | 8s |
| 4 | 16s |
| 5 | 30s (cap) |

Total time from first failure to degraded (all attempts fail): ~60 seconds.

### Why Supervisor-Managed (Not Framework-Level)

Hollywood's built-in restart handles panics via `recover()`, not voluntary shutdown. Consumer startup failures are detected through error returns. Supervisor-managed approach provides:
- Explicit control over backoff policy
- Per-family lifecycle state tracking
- Observability through health tracker counters
- Clean separation: Hollywood handles actor mechanics, supervisor handles pipeline recovery

### Design Constraints

- Fixed restart budget per process lifetime (counter does not reset after successful restarts)
- No cross-family coupling (one family's failure never affects others)
- Budget is hardcoded (5); configurable limits deferred
- Inserter restart is not independently managed (stopped and respawned alongside consumer)

---

## 5. Retry, Backoff, and Loss Semantics (INSERT Path)

### Retry Model

The inserter retries failed ClickHouse INSERT operations with exponential backoff:

```
Attempt 1: immediate
Attempt 2: wait 1s
Attempt 3: wait 2s
Attempt 4: wait 4s
Attempt 5: wait 8s
         -> total worst-case wait: 15s (plus INSERT timeouts)
```

Each attempt has a 30s timeout. Worst-case total: `5 * 30s + 15s = 165s`.

### Configuration

| Parameter | Config key | Default |
|-----------|-----------|---------|
| Max retries | `clickhouse.max_retries` | 5 |
| Initial backoff | `clickhouse.initial_backoff` | 1s |
| Backoff cap | -- | 30s (hardcoded) |
| Per-attempt timeout | -- | 30s (hardcoded) |

### Buffer Retention During Retry

**Critical invariant:** the buffer is not cleared until either:
- INSERT succeeds (buffer cleared, `events_flushed` incremented), or
- All retries exhausted (buffer cleared, `events_dropped` incremented, ERROR logged)

While retrying: the actor is blocked (sequential message processing), new messages queue in mailbox, buffer is not modified.

### Loss Semantics

Every data loss event produces:
1. A structured log entry at ERROR level
2. An increment to `events_dropped`
3. A cause-specific counter (`events_overflowed` or `flush_failures`)

| Category | Cause | Recoverable? |
|----------|-------|--------------|
| Overflow eviction | Buffer > `max_pending` | No (from projection) |
| Retry exhaustion | All INSERT attempts failed | No (from projection) |
| Mapper fallback | Invalid float/JSON in event | Degraded row inserted |
| Decode skip | Undecodable NATS event | Event stays in NATS |

### At-Least-Once Delivery

- Events consumed from NATS JetStream durably
- INSERT retried up to `max_retries` times
- If all retries fail, event lost from analytical projection but remains in NATS for retention period
- Duplicates possible if batch accepted but response lost (acceptable for analytical workloads)

---

## 6. Failure Handling and Overflow

### Design Principles

1. **No silent loss.** Every dropped event is logged and counted.
2. **Bounded retry, not infinite.** Transient failures get a fair retry window; persistent failures are surfaced.
3. **Bounded memory.** Buffer has a hard ceiling. Overflow is explicit and observable.

### Buffer Overflow (FIFO Eviction)

When buffer exceeds `max_pending` (default 10,000 rows):
- Oldest rows evicted (FIFO)
- ERROR log with `evicted` count and `buffer_depth`
- `events_dropped` and `events_overflowed` incremented

Overflow does NOT block the NATS consumer.

### Event Decode / Mapping Failure

- `parseFloat("")` or `parseFloat("invalid")`: returns 0, logs WARN
- `marshalJSON(v)` on error: returns `"{}"`, logs WARN
- Produces degraded-but-insertable rows

### Shutdown During Retry

Current retry attempt continues (bounded by 30s timeout). After completion:
- If successful, batch flushed
- If unsuccessful with retries remaining, no further retries (shutdown priority)
- Remaining buffer subject to shutdown drain attempt

---

## 7. Failure Modes Catalog

### Writer Startup Failures

| ID | Trigger | Behavior |
|----|---------|----------|
| F-01 | `clickhouse.addr` empty | Exit with code 1 |
| F-02 | `clickhouse.database` empty | Exit with field-level error |
| F-03 | `clickhouse.username` empty | Exit with field-level error |
| F-04 | Negative batch_size/max_pending/max_retries | Exit with all issues |
| F-05 | `nats.enabled` false | Exit |
| F-06 | No families configured | Exit |
| F-07 | Pipeline dependency violation | Exit with all dependency issues |
| F-08 | Connection failure | Exit with addr |

### Gateway Degradation

| ID | Trigger | Behavior |
|----|---------|----------|
| F-09 | Invalid ClickHouse config | Log warning, disable analytical endpoints, continue |
| F-10 | Connection failure | Log warning, disable analytical endpoints, continue |

### Runtime Failures

| ID | Trigger | Behavior |
|----|---------|----------|
| F-11 | Consumer startup failure exceeds restart budget | Family degraded; other families continue |

---

## 8. Automatic vs Manual Recovery

### Recovers Automatically

| Scenario | Recovery |
|----------|----------|
| Consumer startup fails (NATS temporarily unavailable) | Supervisor retries up to 5 times with exponential backoff |
| ClickHouse INSERT fails | Inserter retries up to 5 times with exponential backoff |
| NATS connection drops during operation | NATS client reconnects; durable consumer resumes |
| Brief ClickHouse outage (<60s) | Inserter buffers rows, retries, flushes on recovery |

### Requires Manual Recovery

| Scenario | Recovery Path |
|----------|---------------|
| Consumer fails 5+ times on startup | Restart writer process |
| ClickHouse schema mismatch | Fix schema, restart writer |
| NATS stream deleted | Recreate stream, restart writer |
| Writer process crash | Docker `restart: unless-stopped` |

---

## 9. What is NOT Guaranteed (Explicit Non-Goals)

- **Dead-letter queue:** Failed batches are not persisted for later retry
- **Automatic replay:** No mechanism to re-consume events from NATS after loss
- **Cross-batch deduplication:** Duplicates possible on retry after partial success
- **Rate limiting:** Writer does not throttle ingestion rate
- **Auto-recovery from degraded:** Requires process restart (intentional simplicity)
- **Dynamic family registration:** Families are statically configured
- **Backfill mechanism:** No historical NATS replay

---

## 10. Tuning Guidelines

| Scenario | Recommendation |
|----------|---------------|
| ClickHouse frequently briefly unavailable | Increase `max_retries` to 7-10 |
| Tight memory budget | Decrease `max_pending` (trades data loss for memory) |
| High-throughput ingestion | Increase `batch_size` (fewer, larger flushes) |
| Very stable ClickHouse | Decrease `max_retries` to 2-3 (fail fast) |
