# Writer Service: Failure and Delivery Semantics

> Defines failure modes, retry policy, delivery guarantees, and diagnostic visibility for the writer service.
> Stage: S145 — Writer Service Architecture Decision.

## 1. Delivery Guarantee

**At-least-once delivery with eventual ClickHouse consistency.**

The writer provides at-least-once semantics from NATS to ClickHouse:
- Events are acked to NATS **only after** successful ClickHouse INSERT.
- If the writer crashes mid-batch, unacked events are re-delivered by NATS on restart.
- Duplicate rows in ClickHouse are possible and tolerated (MergeTree, no dedup engine).

**Not provided:**
- Exactly-once delivery (would require transactional coordination between NATS and ClickHouse — unjustified complexity).
- Ordering guarantees across families (each family has an independent consumer cursor).
- Gap-free projection (the drop policy explicitly allows gaps during ClickHouse outage).

## 2. Failure Taxonomy

### 2.1 ClickHouse Unavailable (Connection Failure)

**Trigger:** ClickHouse is down, unreachable, or rejecting connections.

**Behavior:**
1. Inserter actor detects flush failure.
2. Events remain in the batch buffer.
3. New events continue arriving from the consumer actor.
4. Buffer grows until `max_pending` (default: 10,000 events).
5. Beyond `max_pending`, **oldest events are dropped** (FIFO eviction).
6. Each drop is logged and counted (`events_dropped` counter).
7. NATS consumer **continues advancing** — it does not pause or back-pressure.

**Recovery:**
- When ClickHouse becomes available, the next flush attempt succeeds.
- Buffered events are written; normal operation resumes.
- Dropped events are lost from the writer's perspective.
- Dropped events **remain in NATS** for stream retention (72h default) and can be replayed by resetting the consumer cursor.

**Rationale for drop policy:**
- Blocking the NATS consumer would cause consumer lag to grow unboundedly.
- Back-pressure to NATS serves no purpose — the operational pipeline must not be affected.
- At current scale (2 symbols × 4 timeframes), even a 10K buffer covers hours of events.
- Manual replay from NATS stream retention is the recovery path for gaps.

### 2.2 ClickHouse INSERT Failure (Partial/Schema Error)

**Trigger:** ClickHouse is reachable but rejects the INSERT (schema mismatch, type error, disk full).

**Behavior:**
1. Inserter logs the full error with batch metadata (family, batch size, first/last event timestamps).
2. Inserter increments `flush_errors` counter.
3. Inserter retries with **exponential backoff**: 1s, 2s, 4s, 8s, 16s (capped at 30s).
4. After 5 consecutive failures for the same batch, the batch is **dropped** and the error is logged at ERROR level.
5. Consumer continues; new events accumulate in a fresh buffer.

**Rationale for bounded retry:**
- Schema mismatch errors will not self-resolve — infinite retry wastes resources.
- Disk-full conditions may resolve but slowly — bounded retry prevents infinite loops.
- The 5-attempt limit gives transient issues time to resolve while bounding failure impact.

### 2.3 NATS Unavailable

**Trigger:** NATS connection lost.

**Behavior:**
1. Consumer actor detects connection loss.
2. Consumer actor logs the event and stops receiving messages.
3. Inserter actor may still have buffered events — it attempts to flush them (best-effort).
4. Writer's `/readyz` returns 503 (NATS unreachable).
5. NATS client's built-in reconnection logic handles reconnection.
6. On reconnect, the durable consumer resumes from last-acked position.

**No special writer logic needed** — NATS reconnection is handled by the shared NATS adapter, identical to all other runtimes.

### 2.4 Deserialization Failure

**Trigger:** A NATS message cannot be deserialized into the expected Go struct.

**Behavior:**
1. Consumer actor logs the error at WARN level with message subject, sequence number, and raw payload sample (truncated).
2. Message is **terminated** (`msg.Term()`) — it will not be re-delivered.
3. `RecordError()` called on the consumer tracker.
4. Processing continues with the next message.

**Rationale:** Deserialization failures indicate a producer-side bug or schema evolution mismatch. Re-delivery will not fix them. Terminating the message prevents infinite reprocessing.

### 2.5 Writer Process Crash

**Trigger:** Unrecoverable panic, OOM kill, or forced termination.

**Behavior:**
1. In-flight buffer is lost (not flushed to ClickHouse).
2. Unacked messages remain in NATS durable consumer — they will be re-delivered on restart.
3. `restart: unless-stopped` in docker-compose triggers automatic restart.
4. On restart, the writer resumes from last-acked position per consumer.

**Gap analysis:**
- Events that were buffered but not flushed are re-delivered by NATS → no gap.
- Events that were flushed and acked before crash → already in ClickHouse → no gap.
- Result: **at-most one batch worth of duplicate rows** on crash recovery.

## 3. Retry Policy Summary

| Failure Type | Retry | Max Attempts | Backoff | On Exhaustion |
|-------------|-------|-------------|---------|---------------|
| ClickHouse connection lost | Yes | Unbounded (buffer + drop) | N/A | Drop oldest beyond max_pending |
| ClickHouse INSERT rejected | Yes | 5 | Exponential (1s–30s) | Drop batch, log ERROR |
| NATS connection lost | Yes | Unbounded (NATS client) | Built-in | Resume on reconnect |
| Deserialization failure | No | 0 | N/A | Term message, log WARN |
| Writer process crash | N/A | N/A | N/A | Docker restarts; NATS re-delivers |

## 4. Idempotency

### 4.1 What Idempotency Means Here

The writer does **not** provide insert-level idempotency. The same event may be written to ClickHouse more than once (after crash recovery or NATS re-delivery). This is acceptable because:

1. ClickHouse is an analytical projection, not a source of truth.
2. Duplicates are rare (only on crash or restart boundaries).
3. At current scale, duplicate volume is negligible.
4. Query-time deduplication (`GROUP BY event_id` or `SELECT DISTINCT`) is trivial and sufficient.

### 4.2 Future Idempotency Path

If deduplication becomes necessary at scale:
- **Option A:** Switch to `ReplacingMergeTree(occurred_at)` with `event_id` in ORDER BY. Dedup is eventual (at merge time) but automatic.
- **Option B:** Maintain a ClickHouse-side dedup set (last N event_ids per table). Adds write-path complexity.
- **Option C:** Query-time `FINAL` modifier on ReplacingMergeTree. Simplest migration path.

**Current decision:** None of these are needed now. The guard rail is explicit: do not add deduplication infrastructure until duplicates are measured and proven problematic.

## 5. Diagnostic Visibility

### 5.1 Structured Logging

Every failure event produces a structured log entry with:

| Field | Description |
|-------|-------------|
| `family` | Pipeline family (candle, signal, etc.) |
| `error` | Error message |
| `batch_size` | Number of events in the failed batch |
| `buffer_depth` | Current buffer depth at time of failure |
| `retry_attempt` | Current retry count (for INSERT failures) |
| `events_dropped` | Cumulative drop count (for connection failures) |

### 5.2 Health Tracker Counters

Exposed via `/statusz`:

| Counter | Failure Signal |
|---------|----------------|
| `flush_errors > 0` | ClickHouse write failures occurring |
| `events_dropped > 0` | Buffer overflow; events lost |
| `error_count > 0` | Deserialization or processing errors |
| `idle_seconds` increasing | Consumer not receiving events (NATS issue or no traffic) |

### 5.3 Alerting Surface

The writer exposes enough signal for external monitoring (future Grafana/alerting integration):

| Metric | Threshold | Meaning |
|--------|-----------|---------|
| `events_dropped` increasing | > 0/min | ClickHouse likely down; buffer overflowing |
| `flush_errors` increasing | > 0/min | ClickHouse rejecting writes |
| `readyz` returning 503 | > 30s | NATS or ClickHouse unreachable |
| `idle_seconds` | > 120s | No events flowing through writer |

## 6. Event Replay

### 6.1 When Replay Is Needed

Replay is needed when:
- ClickHouse was down long enough to exhaust the buffer and drop events.
- A schema migration requires re-ingesting events with new column mappings.
- The writer consumer cursor needs to be reset after a bug fix.

### 6.2 Replay Mechanism

NATS JetStream retains events for the configured stream retention period (72h default). Replay is achieved by:

1. Stop the writer.
2. Delete the `writer-{family}-consumer` durable consumer from NATS.
3. Optionally truncate the target ClickHouse table (`TRUNCATE TABLE {table}`).
4. Start the writer — it creates a new durable consumer starting from the stream's earliest available message.

**Constraint:** Replay window is bounded by NATS stream retention. Events older than retention are irrecoverable from NATS (but this is acceptable — ClickHouse data older than the gap is still present).

### 6.3 Replay and Duplicates

Replay inherently creates duplicates (events already in ClickHouse are re-inserted). This is handled by:
- Query-time dedup if needed.
- Or pre-replay table truncation for clean re-ingestion.

## 7. Anti-Patterns

| Anti-Pattern | Why It's Wrong | Correct Approach |
|-------------|----------------|------------------|
| Blocking NATS consumer on ClickHouse failure | Couples analytical failure to operational pipeline | Buffer + drop; never block |
| Infinite retry on schema mismatch | Schema errors don't self-resolve | Bounded retry (5 attempts), then drop |
| Acking before ClickHouse write | Loses events on crash | Ack only after successful INSERT |
| Per-event INSERT | Destroys ClickHouse performance | Batch INSERT only |
| Writer publishing errors to NATS | Violates INV-03 (writer never publishes) | Log + counter only |
| Dedup infrastructure at current scale | Over-engineering for rare duplicates | Query-time dedup if needed |
