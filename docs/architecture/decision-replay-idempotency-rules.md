# Decision Replay & Idempotency Rules

> Defines the invariants that make decision projections safe under replay,
> reprocessing, and out-of-order delivery.

## Scope

These rules apply to all decision families materialized into KV buckets by the
store binary's `DecisionProjectionActor`.

## Core Invariants

### INV-1: Only finalized decisions enter the read model

The projection actor's first gate drops any event where `Decision.Final == false`.
Non-final decisions never reach the KV store. This ensures the read model contains
only decisions that represent a complete evaluation of the input signals.

### INV-2: Every materialized decision passes domain validation

`Decision.Validate()` is called before any write attempt. Decisions missing required
fields (type, source, symbol, timeframe, outcome, confidence, timestamp) or with
invalid outcome values are rejected and counted in the `rejected` stat.

The valid outcome values are: `triggered`, `not_triggered`, `insufficient`.
Any other value is rejected at the validation gate.

### INV-3: Latest never regresses (monotonicity guard)

The KV store reads the existing entry before writing. The write is skipped if
the existing decision's timestamp is newer than or equal to the incoming decision's
timestamp:

- `existing.Timestamp > incoming.Timestamp` -> `PutSkippedStale`
- `existing.Timestamp == incoming.Timestamp` -> `PutSkippedDuplicate`
- `existing.Timestamp < incoming.Timestamp` -> `PutWritten`

This makes replay safe: replaying old events never overwrites a newer decision.

### INV-4: JetStream deduplication prevents double-publish

Each decision event is published with a message ID derived from `Decision.DeduplicationKey()`:
`dec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`.

JetStream's built-in dedup window (72 hours, matching stream MaxAge) prevents the
same event from being stored in the stream twice, even if the publisher retries
due to timeout.

### INV-5: Durable consumer resumes from last acked position

The store's `DecisionConsumerActor` uses a durable JetStream consumer
(`store-decision-rsi-oversold`). On restart, consumption resumes from the last
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
| Invalid decision replayed | Rejected at gate 2 | `Decision.Validate()` |
| Non-final decision replayed | Dropped at gate 1 | `Final == false` check |
| Unknown outcome value | Rejected at gate 2 | `Decision.Validate()` |

## Write Outcome Enum

```go
PutWritten          // New or newer decision written
PutSkippedStale     // Existing decision is newer
PutSkippedDuplicate // Same timestamp already exists
```

All three outcomes are legitimate and expected during normal operation.
Only KV write errors are counted as `errors`.

## Accepted Limitations

### Ack-before-projection window

The consumer acks the JetStream message before the projection actor writes
to KV. If the process crashes between ack and write, the decision is lost
from the consumer's perspective but bounded to 1 decision per partition key.

On next signal evaluation, a newer decision will be produced, overwriting the gap.
This is acceptable because decisions are continuously re-evaluated — staleness
is bounded by the signal computation frequency (tied to candle finalization).

### Single-writer assumption

The monotonicity guard is sufficient when there is exactly one
`DecisionProjectionActor` per family per deployment. With multiple writers,
the guard prevents regression but redundant writes may occur. This is
harmless (idempotent) but wastes I/O.

### No cross-type atomicity

`rsi_oversold` and future decision types (e.g., `macd_crossover`) are projected
into separate buckets by separate actors. There is no transactional consistency
across decision types. Each type's latest value reflects its own most recent
finalized evaluation.

### No decision history

Unlike evidence candles, decisions do not have a history bucket. The latest-only
model is an intentional design choice (see `decision-projection-pattern.md`).
If history becomes necessary, it should follow the candle history pattern with
embedded timestamps in KV keys and a separate TTL-bound bucket.

## Partition Key Contract

Latest bucket key: `{source}.{symbol}.{timeframe}`

Examples:
- `binancef.btcusdt.60` — RSI oversold decision for BTC/USDT on 1-minute candles
- `binancef.ethusdt.300` — RSI oversold decision for ETH/USDT on 5-minute candles

The key does not include the decision type because each type has its own bucket.

## Deduplication Key Contract

JetStream message ID: `dec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`

The unix-second timestamp provides uniqueness per evaluation cycle.
Two decisions for the same partition key but different timestamps have different
dedup keys and are both accepted by JetStream.

## Decision-Specific Validation

Beyond the structural fields common to all stream families, decisions enforce:

- **Outcome enum validation**: must be one of `triggered`, `not_triggered`, `insufficient`
- **Confidence field**: must not be empty (decimal string, evaluated by application logic)
- **Signals and Metadata**: may be nil (optional context, not required for read model integrity)

These rules ensure the read model never contains a decision with an undefined outcome,
which would be ambiguous for any consumer.
