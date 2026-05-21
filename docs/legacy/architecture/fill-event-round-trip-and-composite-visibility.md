# Fill Event Round-Trip and Composite Visibility

> S334 — Architecture document.
> Describes the canonical fill event round-trip from venue adapter to composite read surface.

## Overview

The fill event round-trip is the critical path that proves a real venue fill
traverses the entire market-foundry pipeline and becomes visible through the
composite query surface exposed by the gateway.

## Canonical Fill Path

```
Signal → Decision → Strategy → Risk → Execute → Fill → Persist → Read
```

### Detailed flow

```
1. Derive publishes PaperOrderSubmittedEvent to EXECUTION_EVENTS
     ↓
2. ExecuteSupervisor (durable consumer: execute-venue-market-order-intake)
     ↓ (intentReceivedMessage → VenueAdapterActor)
3. Safety gates: kill switch + staleness guard
     ↓
4. venue.SubmitOrder() (decorated pipeline: Post200Reconciler + RetrySubmitter)
     ↓
5. VenueOrderFilledEvent published to EXECUTION_FILL_EVENTS
   Subject: execution.fill.venue_market_order.{source}.{symbol}.{timeframe}
     ↓
6a. Writer consumer (writer-execution-venue-fill)
     ↓ mapVenueFillRow()
     ↓ ClickHouse INSERT INTO executions (20 columns)
     ↓
6b. Store consumer (store-execution-venue-market-order-fill)
     ↓ FillProjectionActor
     ↓ KV bucket: EXECUTION_VENUE_MARKET_ORDER_LATEST
     ↓
7. Gateway composite reader
     ↓ queryExecutionByCorrelation(correlation_id, symbol)
     ↓ ParseFillsJSON → []FillRecord
     ↓
8. HTTP response: GET /analytical/composite/chain?correlation_id=...&symbol=...
```

## ClickHouse Persistence

Both paper orders and venue fills write to the same `executions` table using
identical 20-column layouts. The distinguishing fields are:

| Field | Paper Order | Venue Fill |
|-------|-------------|------------|
| type | paper_order | venue_market_order |
| status | submitted | filled |
| filled_quantity | 0 | matches quantity (full fill) |
| fills | `[]` (empty) | `[{price, quantity, fee, simulated: false, timestamp}]` |

## Composite Read Semantics

The composite reader uses `ORDER BY timestamp DESC LIMIT 1` when querying
executions by correlation_id. This means:

- When both a paper_order and venue_fill exist for the same correlation_id, the
  **venue fill wins** because it has a later timestamp.
- The composite chain's execution stage always shows the **latest state** of the
  execution lifecycle.
- Fill data (price, quantity, fee, simulated flag) is deserialized via
  `ParseFillsJSON` and returned in the `ExecutionWithTrace.Fills` field.

## Correlation Chain

```
CorrelationID (immutable across entire chain):
  Signal.correlation_id
    = Decision.correlation_id
    = Strategy.correlation_id
    = Risk.correlation_id
    = PaperOrderSubmittedEvent.Metadata.CorrelationID
    = VenueOrderFilledEvent.Metadata.CorrelationID
    = executions.correlation_id (ClickHouse)

CausationID (links to parent event):
  Decision.causation_id = Signal.event_id
  Strategy.causation_id = Decision.event_id
  Risk.causation_id = Strategy.event_id
  Execution.event_causation_id = Risk.event_id
```

## Deduplication

- JetStream message-level: `fill:{venue_order_id}:{timestamp_unix}`
- ClickHouse: no native dedup — relies on NATS-level dedup and writer idempotency
- KV bucket: monotonicity guard (timestamp-based) in FillProjectionActor

## Test Evidence

| Test | Location | What it proves |
|------|----------|----------------|
| BRT-18: VenueFill_RealFillData | writerpipeline/behavioral_roundtrip_test.go | mapVenueFillRow preserves all fill fields |
| BRT-19: VenueFill_PaperOrderColumnAlignment | writerpipeline/behavioral_roundtrip_test.go | Paper and venue rows use identical column layouts |
| CRI-7: VenueFillChain | composite_reader_integration_test.go | Full 5-stage chain with venue fill visible in composite |
| CRI-8: VenueFillWinsOverPaperOrder | composite_reader_integration_test.go | Latest execution (venue fill) returned when both exist |
| CRI-9: BatchWithVenueFills | composite_reader_integration_test.go | Batch lookup includes venue fill data |
| S317 structural tests | venue_round_trip_test.go | JSON serialization and mapper compatibility |
| LF-1 through LF-4 | live_consumer_flow_test.go | NATS consumer → actor → fill publication |
