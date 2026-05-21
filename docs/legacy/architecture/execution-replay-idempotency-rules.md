# Execution Replay and Idempotency Rules

## Purpose

This document defines the replay safety and idempotency guarantees of the `execution` domain. These properties are critical for operational confidence: restarting consumers, recovering from failures, or reprocessing events must never corrupt materialized state.

## Idempotency Layers

Execution idempotency is enforced at three distinct layers:

### Layer 1: JetStream Deduplication (Publish Side)

**Scope**: Derive → EXECUTION_EVENTS stream

The execution publisher assigns a `MsgID` to every published event using the intent's `DeduplicationKey()`:

```
exec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}
```

JetStream deduplicates messages with the same `MsgID` within the stream's deduplication window. This prevents duplicate events from being written to the stream even if the publisher retries.

**Window**: JetStream default (2 minutes). Sufficient for retry scenarios; not a substitute for downstream guards.

### Layer 2: Durable Consumer (Consume Side)

**Scope**: EXECUTION_EVENTS stream → Store consumer

The execution consumer uses a durable JetStream consumer with explicit ACK policy:

| Setting       | Value                          |
|---------------|--------------------------------|
| Durable       | `store-execution-paper-order`  |
| AckWait       | 30 seconds                     |
| MaxDeliver    | 5                              |
| Subject       | `execution.events.paper_order.submitted.>` |

**Properties**:
- Consumer position is persisted by NATS — restarts resume from last ACK.
- Unacknowledged messages are redelivered up to `MaxDeliver` times.
- After `MaxDeliver` exhaustion, the message is considered terminally failed (not silently dropped — the consumer logs terminal failures).

### Layer 3: KV Monotonicity Guard (Projection Side)

**Scope**: Store projection → KV bucket

The KV store enforces monotonicity on every write:

```
1. Read existing entry for key
2. If existing.Timestamp > intent.Timestamp → PutSkippedStale
3. If existing.Timestamp == intent.Timestamp → PutSkippedDuplicate
4. Otherwise → PutWritten (overwrite)
```

**This is the last line of defense.** Even if the same event is consumed and projected multiple times (due to consumer restart, NAK, or redelivery), the KV guard ensures:
- Stale writes are silently skipped (no data corruption).
- Duplicate writes are silently skipped (no redundant state).
- Only strictly newer timestamps overwrite existing data.

## Replay Safety

### What "replay" means here

Replaying execution events means reprocessing the EXECUTION_EVENTS JetStream stream from an earlier position. This can happen due to:
- Consumer restart with lost ACK state (rare with durable consumers).
- Manual consumer reset for debugging or recovery.
- New consumer creation pointing at an existing stream.

### Replay guarantees

| Guarantee                  | Mechanism                          |
|----------------------------|-----------------------------------|
| No duplicate materialization | KV monotonicity guard (Layer 3)  |
| No stale overwrites        | Timestamp comparison in KV Put    |
| No duplicate events in stream | JetStream MsgID dedup (Layer 1) |
| Resume from last position  | Durable consumer ACK (Layer 2)    |

### Replay limitations

- **Latest-only semantics**: Replay does not reconstruct history. The KV bucket only holds the most recent finalized intent per partition key.
- **No event sourcing**: The KV bucket is a projection, not the source of truth. The EXECUTION_EVENTS stream is the authoritative event log (retained for 72 hours).
- **Stream retention**: Events older than 72 hours are purged. Full replay is only possible within the retention window.

## Invariants Summary

1. **Publish idempotency**: Same intent → same MsgID → deduplicated by JetStream.
2. **Consumer durability**: ACK position survives restarts; redelivery is bounded.
3. **Projection monotonicity**: KV writes are timestamp-ordered; stale/duplicate writes are no-ops.
4. **Stats accounting**: `received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors`.
5. **Sole writer**: Only the projection actor writes to its KV bucket — no concurrent writers.

## Operational Notes

- **Safe to restart**: Both consumer and projection actors can be restarted at any time without data corruption.
- **Safe to redeploy**: New deployments resume from the durable consumer's last ACK position.
- **Not safe to delete the KV bucket**: Deleting the bucket loses all materialized state. Reconstruction requires replaying the stream (within 72h retention).
- **Not safe to share the bucket**: A second writer to the same bucket would violate the sole-writer invariant and risk data corruption.
