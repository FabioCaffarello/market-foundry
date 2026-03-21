# Full Persistence Round-Trip with Live Stack

**Stage:** S317
**Status:** Complete
**Date:** 2026-03-21

## Context

S316 proved real venue submission, fill receipt, and structural persistence compatibility.
However, it explicitly documented gap **R-S316-1**: the complete round-trip
`adapter → NATS → ClickHouse → HTTP composite surface` was never exercised with real
data flowing through the live stack.

S317 closes this gap.

## Round-Trip Architecture

```
┌──────────────────┐
│ Binance Futures   │
│ Testnet (real)    │
└────────┬─────────┘
         │ REST /fapi/v1/order
         ▼
┌──────────────────┐     VenueOrderFilledEvent
│ VenueAdapterActor │────────────────────────────┐
│ (execute binary)  │                            │
└──────────────────┘                            ▼
                                    ┌───────────────────────┐
                                    │ NATS JetStream         │
                                    │ EXECUTION_FILL_EVENTS  │
                                    │ execution.fill.        │
                                    │   venue_market_order.> │
                                    └───────┬───────────────┘
                                            │
                              ┌─────────────┼──────────────┐
                              ▼                            ▼
                    ┌──────────────────┐        ┌──────────────────┐
                    │ writer binary     │        │ store binary      │
                    │ writer-execution- │        │ store-execution-  │
                    │ venue-fill        │        │ venue-market-     │
                    │ consumer          │        │ order-fill        │
                    └────────┬─────────┘        └────────┬─────────┘
                             │                           │
                             ▼                           ▼
                    ┌──────────────────┐        ┌──────────────────┐
                    │ ClickHouse        │        │ NATS KV           │
                    │ executions table   │        │ EXECUTION_VENUE_  │
                    │ (batch insert)     │        │ MARKET_ORDER_     │
                    └────────┬─────────┘        │ LATEST             │
                             │                  └──────────────────┘
                             ▼
                    ┌──────────────────┐
                    │ CompositeReader   │
                    │ (5-table assembly)│
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │ gateway HTTP      │
                    │ /analytical/      │
                    │   composite/...   │
                    └──────────────────┘
```

## What S317 Added

### 1. Writer Consumer for Venue Fills

**Gap:** The writer binary consumed only `EXECUTION_EVENTS` (paper_order family). Venue fill
events published to `EXECUTION_FILL_EVENTS` had no writer consumer — they reached NATS KV
via the store binary but never reached ClickHouse.

**Fix:** Added `WriterVenueMarketOrderFillConsumer` in the NATS execution registry and a
corresponding pipeline entry in `cmd/writer/pipeline.go`:

- **Consumer spec:** `writer-execution-venue-fill` on `EXECUTION_FILL_EVENTS` stream
- **Subject filter:** `execution.fill.venue_market_order.>`
- **Target table:** `executions` (same table as paper_order — unified schema)
- **Row mapper:** `mapVenueFillRow` in writerpipeline — structurally identical to
  `mapExecutionRow` but typed to `VenueOrderFilledEvent`

### 2. Fill Starter

`NewVenueFillStarter` in writerpipeline creates a `FillConsumer` (already existed in
natsexecution but was only used by the store binary) and wires it to the ClickHouse
batch inserter via the standard `RowEmitter` pattern.

### 3. Pipeline Enablement

The `venue_market_order` family is already registered in `knownExecutionFamilies`.
It is enabled via `pipeline.execution_families: ["paper_order", "venue_market_order"]`
in the runtime config. When enabled, the writer creates the consumer-inserter pair
automatically on startup.

## Data Flow Proof

When the pipeline is active:

1. Execute binary submits market order to Binance Futures testnet
2. Venue returns fill with real price, quantity, fee
3. VenueAdapterActor constructs `VenueOrderFilledEvent` preserving correlation/causation IDs
4. Event published to `EXECUTION_FILL_EVENTS` stream
5. Writer's `writer-execution-venue-fill` consumer receives the event
6. `mapVenueFillRow` extracts 20 columns matching the `executions` DDL
7. Batch inserter flushes to ClickHouse
8. `CompositeReader.QueryChainByCorrelationID` finds the row by `correlation_id + symbol`
9. Gateway HTTP endpoint returns the full chain including the venue execution stage

## Schema Compatibility

The `executions` table schema was designed for both paper and venue families:

| Column | Paper Order | Venue Fill |
|--------|------------|------------|
| type | `paper_order` | `paper_order` (same — intent type, not event type) |
| status | `submitted` | `filled` |
| fills | `[{simulated:true}]` | `[{simulated:false, price:real, fee:real}]` |
| filled_quantity | `"0"` or simulated | real from venue |
| exec_correlation_id | from intent | from intent (preserved through venue) |
| exec_causation_id | from intent | from intent (preserved through venue) |

No schema migration required. Both families coexist in the same table, differentiated by
`status` and fill content.

## Validation

### Structural Tests (no stack required)
- `TestS317_VenueFill_RowMapperCompatibility` — 20-column alignment
- `TestS317_VenueFill_CompositeChainReadability` — correlation_id dual alignment
- `TestS317_VenueFill_DryRun` — JSON round-trip without credentials

### Live Tests (credentials required)
- `TestS317_VenueFill_PersistenceRoundTrip` — real venue fill → event construction → JSON → field preservation

### Stack Smoke (full stack required)
- `scripts/smoke-round-trip.sh` — validates NATS stream, writer consumer, ClickHouse rows, HTTP composite surface

## Entrypoints

| Target | Command |
|--------|---------|
| Structural tests | `go test -run TestS317 ./internal/application/execution/...` |
| Smoke (stack) | `make smoke-round-trip` |
