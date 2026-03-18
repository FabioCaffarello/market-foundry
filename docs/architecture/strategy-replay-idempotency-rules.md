# Strategy Replay & Idempotency Rules

> Defines the invariants that make strategy projections safe under replay,
> reprocessing, and out-of-order delivery.

## Scope

These rules apply to all strategy families materialized into KV buckets by the
store binary's `StrategyProjectionActor`.

## Core Invariants

### INV-1: Only finalized strategies enter the read model

The projection actor's first gate drops any event where `Strategy.Final == false`.
Non-final strategies never reach the KV store. This ensures the read model contains
only strategies that represent a complete resolution of the input decisions.

### INV-2: Every materialized strategy passes domain validation

`Strategy.Validate()` is called before any write attempt. Strategies missing required
fields (type, source, symbol, timeframe, direction, confidence, timestamp, decisions)
or with invalid direction values are rejected and counted in the `rejected` stat.

The valid direction values are: `long`, `short`, `flat`.
Any other value is rejected at the validation gate.

### INV-3: Latest never regresses (monotonicity guard)

The KV store reads the existing entry before writing. The write is skipped if
the existing strategy's timestamp is newer than or equal to the incoming strategy's
timestamp:

- `existing.Timestamp > incoming.Timestamp` -> `PutSkippedStale`
- `existing.Timestamp == incoming.Timestamp` -> `PutSkippedDuplicate`
- `existing.Timestamp < incoming.Timestamp` -> `PutWritten`

This makes replay safe: replaying old events never overwrites a newer strategy.

### INV-4: JetStream deduplication prevents double-publish

Each strategy event is published with a message ID derived from `Strategy.DeduplicationKey()`:
`strat:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`.

JetStream's built-in dedup window (72 hours, matching stream MaxAge) prevents the
same event from being stored in the stream twice, even if the publisher retries
due to timeout.

### INV-5: Durable consumer resumes from last acked position

The store's `StrategyConsumerActor` uses a durable JetStream consumer
(`store-strategy-mean-reversion-entry`). On restart, consumption resumes from the last
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
| Invalid strategy replayed | Rejected at gate 2 | `Strategy.Validate()` |
| Non-final strategy replayed | Dropped at gate 1 | `Final == false` check |
| Invalid direction value | Rejected at gate 2 | `Strategy.Validate()` |

## Write Outcome Enum

```go
PutWritten          // New or newer strategy written
PutSkippedStale     // Existing strategy is newer
PutSkippedDuplicate // Same timestamp already exists
```

All three outcomes are legitimate and expected during normal operation.
Only KV write errors are counted as `errors`.

## Accepted Limitations

### Ack-before-projection window

The consumer acks the JetStream message before the projection actor writes
to KV. If the process crashes between ack and write, the strategy is lost
from the consumer's perspective but bounded to 1 strategy per partition key.

On next decision evaluation, a newer strategy will be produced, overwriting the gap.
This is acceptable because strategies are continuously re-resolved — staleness
is bounded by the decision computation frequency (tied to signal/candle finalization).

### Single-writer assumption

The monotonicity guard is sufficient when there is exactly one
`StrategyProjectionActor` per family per deployment. With multiple writers,
the guard prevents regression but redundant writes may occur. This is
harmless (idempotent) but wastes I/O.

### No cross-type atomicity

`mean_reversion_entry` and future strategy types (e.g., `macd_momentum_entry`,
`confluence_entry`) are projected into separate buckets by separate actors. There
is no transactional consistency across strategy types. Each type's latest value
reflects its own most recent finalized resolution.

### No strategy history

Unlike evidence candles, strategies do not have a history bucket. The latest-only
model is an intentional design choice (see `strategy-projection-pattern.md`).
If history becomes necessary, it should follow the candle history pattern with
embedded timestamps in KV keys and a separate TTL-bound bucket.

## Partition Key Contract

Latest bucket key: `{source}.{symbol}.{timeframe}`

Examples:
- `binancef.btcusdt.60` — Mean reversion entry strategy for BTC/USDT on 1-minute candles
- `binancef.ethusdt.300` — Mean reversion entry strategy for ETH/USDT on 5-minute candles

The key does not include the strategy type because each type has its own bucket.

## Deduplication Key Contract

JetStream message ID: `strat:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`

The unix-second timestamp provides uniqueness per resolution cycle.
Two strategies for the same partition key but different timestamps have different
dedup keys and are both accepted by JetStream.

## Strategy-Specific Validation

Beyond the structural fields common to all stream families, strategies enforce:

- **Direction enum validation**: must be one of `long`, `short`, `flat`
- **Confidence field**: must not be empty (decimal string, evaluated by application logic)
- **Decisions array**: at least one `DecisionInput` required (proves the strategy was derived from a decision)
- **Parameters and Metadata**: may be nil (optional context, not required for read model integrity)

These rules ensure the read model never contains a strategy with an undefined direction,
which would be ambiguous for any downstream consumer (risk, execution).

## Monotonicity Guard Implementation

```go
func (s *StrategyKVStore) Put(ctx, strat) (PutResult, error):
  1. Nil/uninitialized guard (returns Unavailable)
  2. Read existing entry by PartitionKey()
  3. Compare timestamps:
     - existing.Timestamp > incoming → PutSkippedStale
     - existing.Timestamp == incoming → PutSkippedDuplicate
     - ErrKeyNotFound → proceed to write (first entry)
  4. Marshal and write only if monotonic advance
```

The read-before-write pattern is the same across all stream families (evidence,
signal, decision, strategy). This consistency makes the replay behavior predictable
and auditable.
