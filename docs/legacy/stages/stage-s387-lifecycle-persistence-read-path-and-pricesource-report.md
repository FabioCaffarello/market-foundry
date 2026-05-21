# Stage S387 — Lifecycle Persistence, Read-Path, and PriceSource Report

> Wave: OMS Foundation (S382–S387)
> Status: Complete
> Predecessor: S386 (Rejection Event Path and Write-Path Observability)

## Executive Summary

S387 closes the OMS foundation wave by delivering three structural alignments:

1. **PriceSource wiring via NATS KV** — the G1 gap from S384 is now closed in production. DryRunSubmitter and PaperVenueAdapter receive realistic fill prices from the `CANDLE_LATEST` KV bucket instead of defaulting to `"0"`.

2. **Rejection KV projection** — the persistence gap from S386 is closed. Venue rejection events now materialize to `EXECUTION_VENUE_REJECTION_LATEST` KV via a dedicated store pipeline, making rejections queryable.

3. **Read-path enhancement** — the composite status query (`execution.query.status.latest`) now includes rejection state alongside intent, fill, and gate, providing complete lifecycle visibility through a single endpoint.

## Deliverables

### Code Changes

| File | Change | Purpose |
|------|--------|---------|
| `internal/adapters/nats/natsevidence/price_source.go` | **New** | `CandleKVPriceSource` implements `ports.PriceSource` reading Close from CANDLE_LATEST |
| `internal/adapters/nats/natsexecution/rejection_consumer.go` | **New** | Durable JetStream consumer for `VenueOrderRejectedEvent` |
| `internal/adapters/nats/natsexecution/registry.go` | Modified | Added `VenueRejectionLatestBucket` constant |
| `internal/actors/scopes/store/rejection_projection_actor.go` | **New** | Sole writer for `EXECUTION_VENUE_REJECTION_LATEST` KV bucket |
| `internal/actors/scopes/store/messages.go` | Modified | Added `rejectionReceivedMessage` type |
| `internal/actors/scopes/store/store_supervisor.go` | Modified | Added `venue_rejection` pipeline entry |
| `internal/actors/scopes/store/query_responder_actor.go` | Modified | Added rejection KV store + read-path in status query |
| `internal/application/executionclient/contracts.go` | Modified | Added `Rejection` field to `ExecutionStatusReply`; updated `DeriveEffectivePropagation` to 3-arg |
| `cmd/execute/run.go` | Modified | Wire `CandleKVPriceSource` into DryRunSubmitter and PaperVenueAdapter |
| `internal/application/execution/pipeline_integration_test.go` | Modified | Updated `DeriveEffectivePropagation` call to 3-arg signature |

### Test Files

| File | Tests | Result |
|------|-------|--------|
| `internal/adapters/nats/natsevidence/s387_price_source_test.go` | 4 tests: nil store fallback, partition key alignment, fallback semantics | PASS |
| `internal/application/execution/s387_lifecycle_persistence_test.go` | 12 tests: propagation with rejection (7 combos), rejection field presence (2), price source wiring (3) | PASS |

### Architecture Documents

| File | Content |
|------|---------|
| `docs/architecture/lifecycle-persistence-read-path-alignment-and-pricesource-wiring.md` | Persistence model, wiring details, invariants preserved |
| `docs/architecture/executionintent-queryability-correlation-pricesource-runtime-and-limitations.md` | Queryability model, correlation chain, PriceSource runtime, limitations |

## Persistence Model After S387

```
derive binary
  → PaperOrderSubmittedEvent → EXECUTION_EVENTS stream
    → store: EXECUTION_PAPER_ORDER_LATEST KV (intent)

execute binary
  → VenueOrderFilledEvent → EXECUTION_FILL_EVENTS stream
    → store: EXECUTION_VENUE_MARKET_ORDER_LATEST KV (fill)
  → VenueOrderRejectedEvent → EXECUTION_REJECTION_EVENTS stream
    → store: EXECUTION_VENUE_REJECTION_LATEST KV (rejection)  ← NEW (S387)
```

All three KV projections use the same partition key `{source}.{symbol}.{timeframe}` and the same monotonicity guard pattern.

## Read-Path After S387

The composite status query returns:

```
Intent     — paper_order latest (derive output)
Result     — venue fill latest (execute output)
Rejection  — venue rejection latest (execute output)  ← NEW (S387)
Gate       — execution control gate
Propagation — derived from most-recent(result, rejection) > intent > "none"
```

## PriceSource Runtime After S387

```
CANDLE_LATEST KV → CandleKVPriceSource → DryRunSubmitter.resolvePrice()
                                        → PaperVenueAdapter.resolvePrice()
```

Fill prices now reflect actual market data from the candle projection instead of hardcoded `"0"`.

## Evidence and Validation

### Compilation

All binaries compile cleanly: execute, store, gateway, derive, writer.

### Test Results

- **S387 propagation tests**: 7/7 PASS — all combinations of intent/result/rejection including timestamp-based priority
- **S387 rejection field tests**: 2/2 PASS — nil and present states
- **S387 price source wiring tests**: 3/3 PASS — realistic prices, fallback on error
- **S387 price source adapter tests**: 4/4 PASS — nil store, partition alignment, fallback semantics
- **Existing test regression**: 0 failures across all execution tests (32s suite)

### Backward Compatibility

- `ExecutionStatusReply.Rejection` is additive (null when absent) — no breaking changes to existing consumers
- `DeriveEffectivePropagation` signature changed from 2 to 3 args — single call site updated (query responder + existing test)
- PriceSource injection is optional (graceful degradation to `"0"`)

## What Is NOT Covered

| Item | Reason |
|------|--------|
| OMS complete | Out of scope — this stage is persistence alignment, not order management |
| ClickHouse writer for rejections | Writer consumer specs exist (S386) but actor wiring deferred |
| Lifecycle history KV | Latest-only model sufficient for foundation; history available via JetStream streams |
| Cross-partition queries | KV model is per-partition; aggregation is a future concern |
| Dashboards | No observability tooling in scope |
| Venue-mode price source | Venue live orders get prices from the exchange, not PriceSource |

## Wave Status: OMS Foundation (S382–S387)

| Stage | Scope | Status |
|-------|-------|--------|
| S382 | OMS Foundation Charter | Complete |
| S383 | Canonical Order Model and Lifecycle State Machine | Complete |
| S384 | Lifecycle Invariant Coverage and Price Realism Hardening | Complete |
| S385 | Write-Path Integration Tests by Execution Mode | Complete |
| S386 | Rejection Event Path and Write-Path Observability | Complete |
| S387 | Lifecycle Persistence, Read-Path, and PriceSource Wiring | **Complete** |

The OMS foundation wave is now complete. All lifecycle states are persisted, queryable, and correlated. The PriceSource is wired for realistic fill pricing. The read-path surfaces the complete lifecycle through a single composite endpoint.

## Recommended Preparation for S388

The natural next steps after the OMS foundation wave closure:

1. **Evidence gate for OMS foundation** — validate the wave deliverables with a formal evidence matrix (pattern established by S375/S381).
2. **Writer wiring for rejections** — connect the ClickHouse writer consumer to persist rejection events beyond JetStream retention.
3. **Quantity enforcement at domain level** — S384 tests prove invariants but the domain model does not enforce them; consider runtime validation.
4. **Lifecycle history projection** — if audit requirements expand beyond latest-only KV, introduce a history bucket or consider the event streams as the authoritative history.
