# Projection Writer Pattern

> Canonical reference for how the store materializes domain events into read-optimized projections.

## Overview

The store service follows a single-writer projection pattern: one `CandleProjectionActor` owns all writes to both the latest and history KV buckets. There are no concurrent writers. This simplifies consistency reasoning — monotonicity and idempotency guards operate without locks or CAS.

## Projection Pipeline

```
EVIDENCE_EVENTS stream
  → EvidenceConsumerActor (durable JetStream consumer)
    → candleReceivedMessage (actor message)
      → CandleProjectionActor
        → Gate 1: Final=true filter
        → Gate 2: candle.Validate()
        → Write: CANDLE_LATEST (monotonicity guard)
        → Write: CANDLE_HISTORY (idempotent by key)
```

## Projection Targets

### CANDLE_LATEST

| Property | Value |
|----------|-------|
| Bucket | `CANDLE_LATEST` |
| Key format | `{source}.{symbol}.{timeframe}` |
| Write semantics | **Last-writer-wins with monotonicity guard** |
| Guard | Existing candle's OpenTime must be strictly older than incoming candle |
| On stale replay | Write skipped, result = `PutSkippedStale` |
| On duplicate replay | Write skipped, result = `PutSkippedDuplicate` |
| On first write | Written unconditionally, result = `PutWritten` |

The monotonicity guard reads the existing value before writing. Since there is exactly one writer, this read-then-write is safe without CAS. The guard prevents regression during replay or reprocessing: a stale candle never overwrites a newer one.

### CANDLE_HISTORY

| Property | Value |
|----------|-------|
| Bucket | `CANDLE_HISTORY` |
| Key format | `{source}.{symbol}.{timeframe}.{open_time_unix}` |
| Write semantics | **Idempotent by key design** |
| Retention | 24h TTL, 256MB max |
| On replay | Same key written with identical data — NATS KV overwrites silently |
| Dedup mechanism | The key embeds `OpenTime` as unix timestamp. Same candle → same key → same value. |

No application-level guard is needed. The key design provides natural dedup.

## Writer Guarantees

### Single-Writer Invariant

There is exactly one `CandleProjectionActor` per store process. The `StoreSupervisor` spawns it once. No horizontal scaling of the writer — scaling happens on the query responder side (via NATS queue groups).

If multiple store instances run concurrently (e.g., during rolling deploy), each has its own durable consumer position. They will process different events and converge to the same state because:
- Latest: monotonicity guard ensures forward-only progression
- History: idempotent key-based writes

### Validation Before Write

Every candle passes `candle.Validate()` before any KV operation. This prevents malformed data from entering the read model. Rejected candles are counted and logged at WARN level.

### Write Ordering

For each candle, the projection writes LATEST first, then HISTORY. If the HISTORY write fails, the LATEST write has already succeeded. This is intentional: LATEST represents the most recent state, while HISTORY is a bounded append log. A gap in history is recoverable (events are still in the JetStream stream), but serving a stale latest would be misleading.

## Observability

`CandleProjectionActor` tracks six counters via `projectionStats`:

| Counter | Meaning |
|---------|---------|
| `materialized` | Candles written to both latest and history |
| `skipped_stale` | Latest write skipped: existing candle is newer |
| `skipped_dedup` | Latest write skipped: same OpenTime already exists |
| `skipped_non_final` | Candles dropped because `Final=false` |
| `rejected` | Candles rejected by domain validation |
| `errors` | KV write errors |

Stats are logged at shutdown (`actor.Stopped`) and can be used to detect replay anomalies:
- High `skipped_stale` after restart → consumer replayed a range of old events (expected)
- High `rejected` → upstream derive is producing invalid candles (investigate)
- High `errors` → NATS KV connectivity issue
