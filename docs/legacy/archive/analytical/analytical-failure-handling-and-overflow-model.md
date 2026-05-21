# Analytical Failure Handling and Overflow Model

Status: canonical (S153)
Scope: writer service failure semantics after S153 hardening

## 1. Design Principles

The writer is a **lateral, append-only projection** of domain events into ClickHouse.
It is not part of the operational critical path. Core pipeline services do not depend on writer success.

Failure handling follows three principles:

1. **No silent loss.** Every dropped event must be logged and counted.
2. **Bounded retry, not infinite.** Transient failures get a fair retry window; persistent failures are surfaced, not hidden.
3. **Bounded memory.** The buffer has a hard ceiling. Overflow is an explicit, observable data-loss event.

## 2. Failure Categories

### 2.1 ClickHouse INSERT Failure

When `InsertBatch` fails, the inserter retries with exponential backoff.

| Parameter | Default | Config key |
|-----------|---------|------------|
| Max retries | 5 | `clickhouse.max_retries` |
| Initial backoff | 1s | `clickhouse.initial_backoff` |
| Backoff cap | 30s | hardcoded |
| Per-attempt timeout | 30s | hardcoded |

Backoff progression: 1s, 2s, 4s, 8s, 16s (capped at 30s).

**On retry exhaustion:** the batch is dropped. An ERROR log is emitted with `rows_dropped`, `attempts`, and the last error. The `events_dropped` and `flush_failures` counters are incremented.

**Buffer retention during retry:** the buffer is NOT cleared until the batch succeeds or all retries are exhausted. This prevents the data-loss bug where a failed INSERT would silently discard the batch (pre-S153 behavior).

### 2.2 Buffer Overflow (FIFO Eviction)

When the in-memory buffer exceeds `max_pending` (default 10,000 rows), the oldest rows are evicted to make room. This is a **permanent data-loss event** from the analytical projection's perspective.

Overflow behavior:
- Evicts oldest rows (FIFO) until buffer is at `max_pending`
- Logs at **ERROR** level with `evicted` count and `buffer_depth`
- Increments `events_dropped` (aggregate loss counter)
- Increments `events_overflowed` (overflow-specific counter)

Overflow does NOT block the NATS consumer. Blocking would cause consumer lag to grow unboundedly, which is worse than bounded data loss in a lateral projection.

### 2.3 Event Decode / Mapping Failure

If the consumer cannot decode a NATS event, the row is skipped and logged. The event remains durably in NATS JetStream and can theoretically be replayed.

If mapper helper functions (`parseFloat`, `marshalJSON`) encounter invalid input:
- `parseFloat("")` or `parseFloat("invalid")`: returns 0, logs WARN
- `marshalJSON(v)` on marshal error: returns `"{}"`, logs WARN

These produce degraded-but-insertable rows. The WARN log makes the degradation visible to operators without blocking ingestion.

### 2.4 Consumer Startup Failure

If a NATS consumer fails to start (e.g., stream not found), the actor is poisoned. Events remain durably in NATS. Recovery requires process restart (handled by docker-compose `restart: unless-stopped`).

Pipeline-level recovery (supervisor restarts individual failed pipelines) is deferred to S154.

### 2.5 ClickHouse Connection Loss

If ClickHouse becomes unreachable, every flush attempt will fail and retry. The retry window (up to ~61s with 5 retries at exponential backoff) provides tolerance for brief outages. If ClickHouse remains down beyond the retry window, batches are dropped with ERROR logs.

The `/readyz` endpoint checks ClickHouse connectivity. Persistent connection loss will cause readiness to fail, which is visible to orchestration.

## 3. What is Guaranteed

- Every dropped event is logged (ERROR for overflow and retry exhaustion, WARN for mapper fallbacks).
- Every dropped event is counted in `events_dropped`.
- Overflow and INSERT failure drops are distinguishable via `events_overflowed` and `flush_failures`.
- Retry with backoff covers transient ClickHouse failures up to the configured retry budget.
- Buffer is retained during retry — no premature clearing.
- Successful batches are all-or-nothing (ClickHouse batch protocol).

## 4. What is Best Effort

- Mapper fallback values (0 for bad floats, `"{}"` for nil) degrade data quality but allow ingestion to continue. Operator must monitor WARN logs to catch data quality issues.
- Shutdown drain attempts to flush remaining buffer but is bounded by shutdown timeout.
- Events that fail all retries are permanently lost from the analytical projection. They remain in NATS JetStream for the retention window but there is no automatic replay mechanism.

## 5. What is Not Guaranteed (Out of Scope)

- **Dead-letter queue.** Failed batches are not persisted for later retry.
- **Automatic replay.** There is no mechanism to re-consume events from NATS after loss.
- **Pipeline-level recovery.** A failed consumer-inserter pair is not automatically restarted (deferred to S154).
- **Cross-batch deduplication.** If a batch partially succeeded at the ClickHouse level before returning an error, rows may be duplicated on retry. ClickHouse's append-only semantics tolerate duplicates for analytical queries.
- **Rate limiting.** The writer does not throttle ingestion rate.

## 6. Counter Reference

| Counter | Meaning | Incremented by |
|---------|---------|----------------|
| `events_flushed` | Rows successfully inserted | flush (on success) |
| `events_dropped` | Rows permanently lost (any cause) | overflow eviction, retry exhaustion |
| `events_overflowed` | Rows lost specifically to buffer overflow | overflow eviction |
| `flush_failures` | Flush operations that exhausted all retries | retry exhaustion |

All counters are per-pipeline-family and visible via `/statusz` and `/diagz`.
