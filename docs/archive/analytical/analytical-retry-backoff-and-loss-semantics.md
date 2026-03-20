# Analytical Retry, Backoff, and Loss Semantics

Status: canonical (S153)
Scope: retry and backoff behavior for the writer service INSERT path

## 1. Retry Model

The writer inserter retries failed ClickHouse INSERT operations with exponential backoff.

```
Attempt 1: immediate
Attempt 2: wait 1s
Attempt 3: wait 2s
Attempt 4: wait 4s
Attempt 5: wait 8s
         → total worst-case wait: 15s (plus INSERT timeouts)
```

Each attempt has a 30s timeout. Worst-case total time for a single flush with 5 retries:
`5 * 30s (timeouts) + 15s (backoff) = 165s`.

In practice, transient failures return quickly. The 30s timeout is a safety net for hung connections.

### Configuration

| Parameter | Config key | Default | Range |
|-----------|-----------|---------|-------|
| Max retries | `clickhouse.max_retries` | 5 | 1+ (0 treated as 1) |
| Initial backoff | `clickhouse.initial_backoff` | 1s | valid Go duration |
| Backoff cap | — | 30s | hardcoded |
| Per-attempt timeout | — | 30s | hardcoded |

### Backoff Progression

Starting from `initial_backoff`, each subsequent wait doubles, capped at 30s:

| initial_backoff | Attempt 2 | Attempt 3 | Attempt 4 | Attempt 5 |
|-----------------|-----------|-----------|-----------|-----------|
| 500ms | 500ms | 1s | 2s | 4s |
| 1s (default) | 1s | 2s | 4s | 8s |
| 2s | 2s | 4s | 8s | 16s |
| 5s | 5s | 10s | 20s | 30s |

## 2. Buffer Retention

**Critical invariant:** the buffer is not cleared until either:
- INSERT succeeds (buffer cleared, `events_flushed` incremented), or
- All retries are exhausted (buffer cleared, `events_dropped` incremented, ERROR logged).

This was a bug fix in S153. Pre-S153, the buffer was cleared before the INSERT attempt, meaning any failure would permanently lose the rows with no retry opportunity.

### During Retry

While the inserter is retrying:
- The actor is blocked (actor model — sequential message processing).
- New `insertRowMsg` messages queue in the actor mailbox.
- The buffer is not modified.
- No new rows are appended until the retry sequence completes.

This is acceptable because:
- The inserter is already doing blocking I/O.
- Messages queue naturally in the actor mailbox.
- NATS consumer backpressure is handled by JetStream's durable consumer semantics.

## 3. Loss Semantics

### Loss is explicit, not silent

Every data loss event produces:
1. A structured log entry at ERROR level.
2. An increment to the `events_dropped` counter.
3. A cause-specific counter (`events_overflowed` or `flush_failures`).

### Loss categories

| Category | Cause | Severity | Recoverable? |
|----------|-------|----------|--------------|
| Overflow eviction | Buffer > `max_pending` | ERROR | No (from projection) |
| Retry exhaustion | All INSERT attempts failed | ERROR | No (from projection) |
| Mapper fallback | Invalid float/JSON in event | WARN | Degraded row inserted |
| Decode skip | Undecodable NATS event | WARN | Event stays in NATS |

### At-least-once semantics

The writer's delivery guarantee is **at-least-once with bounded retry**:
- Events are consumed from NATS JetStream durably.
- INSERT is retried up to `max_retries` times.
- If all retries fail, the event is lost from the analytical projection but remains in NATS for the stream retention period.

**Caveat:** if ClickHouse accepts a batch but the response is lost (network partition), the next retry may insert duplicates. This is acceptable for analytical workloads where duplicates can be handled at query time (e.g., `DISTINCT`, `argMax`).

## 4. Interaction with Other Failure Modes

### Buffer overflow during retry

If the buffer overflows while a flush is in progress, the overflow happens at the next `insertRowMsg` processing — which is after the current flush completes (actor sequential processing). This means overflow cannot happen mid-retry.

### Shutdown during retry

If a shutdown signal arrives while a retry is in progress, the current retry attempt continues (bounded by its 30s timeout). After the current attempt completes:
- If successful, the batch is flushed.
- If unsuccessful and retries remain, no further retries are attempted (shutdown takes priority over retry budget).
- Remaining buffer is subject to the shutdown drain attempt.

### Multiple pipeline families

Each pipeline family (candle, rsi, etc.) has its own inserter actor. Retry in one family does not block or affect other families. Counters are per-family.

## 5. Tuning Guidelines

| Scenario | Recommendation |
|----------|---------------|
| ClickHouse frequently briefly unavailable | Increase `max_retries` to 7-10 |
| Tight memory budget | Decrease `max_pending` (trades data loss for memory) |
| High-throughput ingestion | Increase `batch_size` (fewer, larger flushes) |
| Very stable ClickHouse | Decrease `max_retries` to 2-3 (fail fast) |

## 6. Decision Record

**Why retry at the inserter level (not the adapter)?**
The ClickHouse adapter (`internal/adapters/clickhouse`) is a thin wrapper with no retry logic. Retry belongs in the inserter because:
- The inserter owns the buffer and knows whether to retain or drop rows.
- Different callers (reader vs. writer) have different retry requirements.
- Keeping the adapter stateless simplifies testing and reuse.

**Why exponential backoff (not fixed interval)?**
Exponential backoff with a cap provides:
- Fast recovery for transient blips (1s first retry).
- Graceful backing off for longer outages.
- Bounded total retry time (predictable worst case).

**Why drop after exhaustion (not block indefinitely)?**
The writer is a lateral projection. Blocking indefinitely would:
- Cause NATS consumer lag to grow unboundedly.
- Prevent other pipeline families from making progress.
- Provide no benefit if ClickHouse is persistently down.
Dropping with explicit ERROR logging is the correct trade-off for a best-effort analytical layer.
