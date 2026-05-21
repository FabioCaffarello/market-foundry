# Signal Replay & Idempotency Rules

> Defines the invariants that make signal projections safe under replay,
> reprocessing, and out-of-order delivery.

## Scope

These rules apply to all signal families materialized into KV buckets by the
store binary's `SignalProjectionActor`.

## Core Invariants

### INV-1: Only finalized signals enter the read model

The projection actor's first gate drops any event where `Signal.Final == false`.
Non-final signals never reach the KV store. This ensures the read model contains
only signals that have completed their computation window.

### INV-2: Every materialized signal passes domain validation

`Signal.Validate()` is called before any write attempt. Signals missing required
fields (type, source, symbol, timeframe, value, timestamp) are rejected and
counted in the `rejected` stat.

### INV-3: Latest never regresses (monotonicity guard)

The KV store reads the existing entry before writing. The write is skipped if
the existing signal's timestamp is newer than or equal to the incoming signal's
timestamp:

- `existing.Timestamp > incoming.Timestamp` -> `PutSkippedStale`
- `existing.Timestamp == incoming.Timestamp` -> `PutSkippedDuplicate`
- `existing.Timestamp < incoming.Timestamp` -> `PutWritten`

This makes replay safe: replaying old events never overwrites a newer signal.

### INV-4: JetStream deduplication prevents double-publish

Each signal event is published with a message ID derived from `Signal.DeduplicationKey()`:
`sig:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`.

JetStream's built-in dedup window prevents the same event from being stored in
the stream twice, even if the publisher retries due to timeout.

### INV-5: Durable consumer resumes from last acked position

The store's `SignalConsumerActor` uses a durable JetStream consumer
(`store-signal-rsi`). On restart, consumption resumes from the last
acknowledged message — no gap, no full replay unless the consumer is
manually reset.

## Replay Safety Matrix

| Scenario | Outcome | Guard |
|----------|---------|-------|
| Process restart | Resumes from last ack | Durable consumer |
| Duplicate event in stream | Dropped by dedup | JetStream MsgID |
| Old event replayed | Skipped by monotonicity | KV read-before-write |
| Same-timestamp event | Skipped as duplicate | KV timestamp comparison |
| New event, fresh bucket | Written (first entry) | ErrKeyNotFound -> write |
| Invalid signal replayed | Rejected at gate 2 | `Signal.Validate()` |
| Non-final signal replayed | Dropped at gate 1 | `Final == false` check |

## Write Outcome Enum

```go
PutWritten         // New or newer signal written
PutSkippedStale    // Existing signal is newer
PutSkippedDuplicate // Same timestamp already exists
```

All three outcomes are legitimate and expected during normal operation.
Only KV write errors are counted as `errors`.

## Accepted Limitations

### Ack-before-projection window

The consumer acks the JetStream message before the projection actor writes
to KV. If the process crashes between ack and write, the signal is lost
from the consumer's perspective but bounded to 1 signal per partition key.

On next candle close, a newer signal will be produced, overwriting the gap.
This is acceptable because signals are continuously refreshed — staleness
is bounded by the evidence window duration (timeframe).

### Single-writer assumption

The monotonicity guard is sufficient when there is exactly one
`SignalProjectionActor` per family per deployment. With multiple writers,
the guard prevents regression but redundant writes may occur. This is
harmless (idempotent) but wastes I/O.

### No cross-type atomicity

RSI and future signal types (MACD, etc.) are projected into separate buckets
by separate actors. There is no transactional consistency across signal types.
Each type's latest value reflects its own most recent finalized computation.

## Partition Key Contract

Latest bucket key: `{source}.{symbol}.{timeframe}`

Examples:
- `binancef.btcusdt.300` — RSI for BTC/USDT on 5-minute candles
- `binancef.ethusdt.60` — RSI for ETH/USDT on 1-minute candles

The key does not include the signal type because each type has its own bucket.

## Deduplication Key Contract

JetStream message ID: `sig:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`

The unix-second timestamp provides uniqueness per computation window.
Two signals for the same partition key but different timestamps have different
dedup keys and are both accepted by JetStream.
