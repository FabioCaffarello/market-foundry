# Real Venue Through Stack: Findings and Limitations

**Stage:** S317
**Status:** Complete
**Date:** 2026-03-21

## Findings

### F-1: Writer Gap Was the Only Structural Blocker

The complete execution pipeline — adapter, NATS publishing, store projection, ClickHouse
schema, composite reader, HTTP handler, gateway wiring — was already in place. The only
missing piece was a writer consumer for the `EXECUTION_FILL_EVENTS` stream. Once
`WriterVenueMarketOrderFillConsumer` and `NewVenueFillStarter` were added, venue fills
flow through the entire stack without any further changes.

**Implication:** The pipeline architecture is sound. Each component was independently
correct; the gap was purely at the wiring layer.

### F-2: Unified Executions Table Works for Both Families

Paper orders and venue fills share the same `executions` table schema. The `status` column
and `fills` JSON content naturally differentiate them:

- Paper: `status=submitted`, fills contain `simulated=true`
- Venue: `status=filled`, fills contain `simulated=false` with real prices

No schema changes, no migration, no conditional logic required.

### F-3: Correlation ID Flows End-to-End

The correlation_id set by the derive binary on the original paper order event is preserved
through every hop:

1. `PaperOrderSubmittedEvent.Metadata.CorrelationID` (derive → NATS)
2. `VenueAdapterActor` copies it to `VenueOrderFilledEvent.Metadata.CorrelationID`
3. Writer maps both `Metadata.CorrelationID` → `correlation_id` column and
   `ExecutionIntent.CorrelationID` → `exec_correlation_id` column
4. `CompositeReader` queries by `correlation_id` column to assemble the chain

This dual-column design (event-level + intent-level) provides redundancy and supports
both composite chain assembly and intent-level audit trail.

### F-4: FillConsumer Was Production-Ready

The `natsexecution.FillConsumer` (used by the store binary) was already feature-complete:
redelivery tracking, max-deliver termination, ack/nak semantics, graceful shutdown.
The writer simply reuses it with a different handler function.

### F-5: Batch Insert Semantics Are Additive

The writer's batch inserter does not distinguish between paper and venue rows — both are
just `[]any` slices matching the same INSERT SQL. This means venue fills naturally benefit
from the same batching, retry, and overflow protection already proven for paper orders.

## Limitations

### L-1: Round-Trip Not Proven with Continuous Live Data

The S317 proof validates structural correctness and component wiring. A true continuous
round-trip (market data → full pipeline → venue submission → fill persistence → composite
read) requires:

- Live market data feeding ingest
- Active bindings producing signal → decision → strategy → risk → execution → venue fill
- Writer flushing within observation window

This is an operational concern, not a structural one. The smoke script
(`smoke-round-trip.sh`) validates the infra readiness but cannot inject venue fills
without the execute binary actively connected to a venue.

### L-2: Testnet Fills Are Atomic

Binance Futures testnet fills minimum-size market orders atomically (single fill).
The `StatusPartiallyFilled` code path is exercised in unit tests but has never been
observed with real testnet data. Production venue behavior may differ.

### L-3: Real Commission Endpoint Not Integrated

The adapter uses `cumQuote` from Binance response as a fee proxy. The real
`/fapi/v1/commissionRate` endpoint is available but not yet integrated. Fee accuracy
is sufficient for testnet proof but not for production accounting.

### L-4: Kill Switch Tested with Mock Only

The safety gate (kill switch + staleness guard) has been validated with mock checkers
in integration tests. Live NATS KV `EXECUTION_CONTROL` bucket integration is exercised
by the execute binary at runtime but is not directly observable in the round-trip smoke.

### L-5: Single Venue Only

Only Binance Futures testnet is supported. The `VenuePort` interface allows additional
adapters, but no second venue implementation exists. Multi-venue round-trip is out of scope.

### L-6: No WebSocket / Async Fill Path

All venue interaction is synchronous REST. The Binance WebSocket user data stream
(which would provide real-time fill updates for large or partial orders) is not connected.
This is an explicit guard rail, not a limitation to fix in this stage.

## Residual Gaps for Future Stages

| Gap | Severity | Recommendation |
|-----|----------|----------------|
| Continuous live round-trip proof | Medium | Run full pipeline with testnet credentials for extended period |
| Partial fill handling | Low | Unit tests cover the path; observe in testnet with larger orders |
| Real commission endpoint | Low | Integrate `/fapi/v1/commissionRate` when production accounting needed |
| Multi-venue support | Deferred | Add second venue adapter when business requires |
| WebSocket fill stream | Deferred | Required only for production latency requirements |
