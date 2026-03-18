# Replay & Idempotency Rules

> Defines how the store projection layer behaves under replay, restart, and reprocessing scenarios.

## Replay Scenarios

### 1. Normal restart (process crash/restart)

The durable JetStream consumer (`store-evidence`) resumes from the last acked position. Events processed before the crash are not re-delivered. Events that were in-flight (acked by the consumer but not yet projected) may be lost from the projection.

**Impact on latest:** Minimal. The next finalized candle will overwrite whatever was in KV. Monotonicity guard prevents any regression.

**Impact on history:** Minimal. The gap is bounded by in-flight events at crash time (typically 0-1 candles). History is a bounded 24h window — the gap will be covered by subsequent events.

### 2. Consumer reset (intentional replay)

If the durable consumer is deleted and recreated, it starts from the earliest available message in the stream (72h retention). All events are replayed.

**Impact on latest:** Safe. The monotonicity guard ensures only forward progression. A full replay produces the correct final state (latest = newest candle) regardless of event order within the replay.

**Impact on history:** Safe. Each replayed candle writes to its unique key (`{source}.{symbol}.{tf}.{open_time_unix}`). KV `Put` is idempotent — same key, same value. History only retains 24h of data, so events older than 24h are written then immediately expire.

### 3. Duplicate delivery

JetStream may deliver the same event twice (at-least-once semantics). The projection handles this without special logic:

- **Latest:** Monotonicity guard returns `PutSkippedDuplicate` for same OpenTime → no write.
- **History:** Same key written with same value → NATS KV overwrites silently.

### 4. Out-of-order delivery

Events may arrive out of chronological order during replay or concurrent processing.

- **Latest:** Monotonicity guard returns `PutSkippedStale` for older candles → the projection never regresses.
- **History:** Each candle has its own time-indexed key → ordering doesn't affect correctness. Query results are sorted by the reader, not by write order.

## Idempotency Rules

### Rule 1: History is idempotent by key design

The history key `{source}.{symbol}.{tf}.{open_time_unix}` is a natural dedup key. A candle for btcusdt/60s starting at 1710000000 always writes to the same key, regardless of how many times the event is delivered.

**Invariant:** For any given candle, `PutHistory(candle)` called N times produces the same state as calling it once.

### Rule 2: Latest is idempotent via monotonicity guard

The latest projection reads the existing value before writing. Three outcomes:

| Existing state | Incoming candle | Result |
|----------------|-----------------|--------|
| No entry | Any | `PutWritten` |
| OpenTime < incoming | Any | `PutWritten` |
| OpenTime = incoming | Same candle (replay) | `PutSkippedDuplicate` |
| OpenTime > incoming | Stale candle | `PutSkippedStale` |

**Invariant:** After processing a set of candle events (in any order, with any number of duplicates), `CANDLE_LATEST` contains the candle with the most recent OpenTime from that set.

### Rule 3: Non-final candles are never materialized

Events with `Final=false` are dropped at the first gate. This means interim/realtime candles never enter the read model. The projection only materializes closed windows.

**Invariant:** Every candle in `CANDLE_LATEST` and `CANDLE_HISTORY` has `Final=true`.

### Rule 4: Domain validation precedes all writes

`candle.Validate()` runs before any KV operation. A candle that fails validation is counted and logged but never written.

**Invariant:** Every candle in KV passes `EvidenceCandle.Validate()`.

## Deduplication Summary

| Bucket | Dedup mechanism | Scope |
|--------|-----------------|-------|
| CANDLE_LATEST | Monotonicity guard (read-then-compare-then-write) | Per source/symbol/timeframe |
| CANDLE_HISTORY | Natural key dedup (open_time in key) | Per source/symbol/timeframe/open_time |

## Known Limitations

### Ack-before-projection window

The evidence consumer acks the JetStream message before the projection actor processes it (the consumer sends an actor message, then acks). On crash between ack and projection write, that candle is lost from the projection.

**Why this is acceptable:**
- The gap is at most 1 candle per source/symbol/timeframe.
- The next finalized candle will update the latest projection correctly.
- History may have a 1-candle gap, but this is a bounded window (24h) for operational queries, not a compliance archive.
- Making ack synchronous with projection would require the consumer to wait for the projection actor to respond, adding latency and complexity for marginal benefit.

**Mitigation if needed (future):**
- Switch to ack-after-projection by having the projection actor send back a confirmation message.
- Or use a write-ahead-log pattern where the consumer writes to a local buffer before acking.

### No cross-bucket atomicity

LATEST and HISTORY are written sequentially, not atomically. If the process crashes between the two writes, LATEST may be updated while HISTORY is not. This is acceptable because:
- Both are eventually consistent views of the same event stream.
- The next event will fill the history gap.
- Queries never join across the two buckets.

### Single-writer assumption

The monotonicity guard is a read-then-write, not a CAS. It relies on the single-writer invariant. If two store instances write to the same KV buckets concurrently, the guard may produce incorrect results. The system prevents this by design (one store process per deployment), but does not enforce it at the KV level.
