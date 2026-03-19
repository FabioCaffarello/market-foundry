# Execution Fill Projection Pattern

## Purpose

Defines how venue order fill events emitted by the `execute` binary are consumed, validated, and materialized by the `store` binary into a canonical read-side KV bucket.

## Context

The `execute` binary publishes `VenueOrderFilledEvent` to the `EXECUTION_FILL_EVENTS` JetStream stream after the venue adapter (paper or real) produces a fill. Without a corresponding store-side projection, the fill data has no canonical read model — it exists only transiently in the stream.

This pattern closes that gap by adding a dedicated fill projection pipeline in the store, following the same structural contract as all other projection pipelines (evidence, signal, decision, strategy, risk, execution intent).

## Design

### Stream and Subject

| Property | Value |
|----------|-------|
| Stream | `EXECUTION_FILL_EVENTS` |
| Subject filter | `execution.fill.venue_market_order.>` |
| Event type | `execution.fill.v1.venue_market_order_filled` |
| Durable consumer | `store-execution-venue-market-order-fill` |

### KV Bucket

| Property | Value |
|----------|-------|
| Bucket | `EXECUTION_VENUE_MARKET_ORDER_LATEST` |
| Key format | `{source}.{symbol}.{timeframe}` |
| Semantics | Latest-only (no history) |
| Max size | 64 MB |
| Storage | FileStorage |

### Pipeline Components

```
EXECUTION_FILL_EVENTS stream
    ↓
FillConsumer (durable: "store-execution-venue-market-order-fill")
    ↓ decodes VenueOrderFilledEvent
FillConsumerActor
    ↓ sends fillReceivedMessage to projection PID
FillProjectionActor
    ├─ Gate 1: Skip non-final intents (intent.Final == false)
    ├─ Gate 2: Validate ExecutionIntent (domain rules)
    └─ Gate 3: Monotonicity guard (timestamp-based, via ExecutionKVStore)
         ↓
EXECUTION_VENUE_MARKET_ORDER_LATEST KV bucket
```

### Query Path

```
Gateway HTTP: GET /execution/venue_market_order/latest?source=X&symbol=Y&timeframe=Z
    ↓
ExecutionGateway → NATS request to "execution.query.venue_market_order.latest"
    ↓
QueryResponderActor → executionVenueMarketOrderStore.Get()
    ↓
Returns ExecutionLatestReply { ExecutionIntent }
```

## Gating Sequence

The fill projection applies the same three-gate pipeline as the execution intent projection:

1. **Finality gate**: Only `Final == true` intents materialize. Non-final intents are counted as `skippedNonFinal`.
2. **Validation gate**: `intent.Validate()` enforces all domain invariants (type, source, symbol, timeframe, side, status). Invalid intents are counted as `rejected`.
3. **Monotonicity gate**: The KV adapter compares timestamps. Stale (out-of-order) and duplicate (same timestamp) writes are skipped. This prevents regression to older fill states.

## Separation of Concerns

| Bucket | Scope | What it stores |
|--------|-------|---------------|
| `EXECUTION_PAPER_ORDER_LATEST` | Derive intent | The execution intent as evaluated by derive — reflects what derive *wanted* to happen |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Execute fill | The execution intent as returned by the venue adapter — reflects what *actually* happened |

These two buckets are intentionally separate. The paper order bucket captures the derive-side intent; the venue market order bucket captures the execute-side fill result. A query consumer can compare both to detect divergence between intent and outcome.

## Stats Invariant

```
received == materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors
```

Checked on actor shutdown. Violations are logged at ERROR level.

## Limitations

- **Latest-only**: No fill history is retained. Each partition key holds only the most recent fill.
- **No aggregation**: Fill records are not aggregated across time or symbols. Each partition is independent.
- **Paper mode only**: The current implementation handles paper fills exclusively. Real venue fills will follow the same pattern but may require additional validation gates.
