# S416: Futures Real Venue Acceptance/Fill Proof Report

**Stage**: S416
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Type**: Implementation and proof
**Status**: Complete
**Date**: 2026-03-23

## Executive Summary

S416 proves real Binance Futures testnet connectivity and the dominant lifecycle
path `submitted -> accepted -> filled` under the unified runtime with `dry_run=false`.

38 tests (32 adapter-level + 6 actor-level) validate:

- Futures venue connectivity via `BinanceFuturesTestnetAdapter`
- Lifecycle transitions aligned with `ValidTransition()` (S383)
- Fill record fidelity: real price (avgPrice), quantity, fee (cumQuote), timestamp, `Simulated=false`
- Correlation/causation chain integrity through the full write-path
- Segment routing isolation (Futures only, Spot untouched)
- Post-200 reconciliation via `QueryOrder`
- Error classification (retryable vs non-retryable)
- DryRunSubmitter bypass under `dry_run=false`
- Structural differences from Spot adapter (API path, response type, price/fee extraction)

All 38 tests pass. Zero regressions across the full test suite.

## Connectivity and Acceptance/Fill Validated

### Dominant Lifecycle Path

```
ExecutionIntent (source="binancef", status=submitted)
  -> SegmentRouter (source -> MarketSegmentFutures)
    -> BinanceFuturesTestnetAdapter
      -> POST /fapi/v1/order (HMAC-SHA256 signed)
      <- HTTP 200 {status: "FILLED", avgPrice: "65432.10", cumQuote: "65.43210"}
    -> parseOrderResponse()
      -> mapBinanceStatus("FILLED") -> StatusFilled
      -> FillRecord{Price: "65432.10", Qty: "0.001", Fee: "65.43210", Simulated: false}
    <- VenueOrderReceipt {Status: filled, VenueOrderID: "67890"}
```

### Config Pattern

New config `deploy/configs/execute-venue-live-futures.jsonc`:

- `dry_run: false` -- bypasses DryRunSubmitter
- `futures.enabled: true, adapter: binance_futures_testnet` -- real Futures adapter
- `spot.enabled: true` -- structural coexistence preserved

## Files Changed

### New Files

| File | Purpose |
|---|---|
| `deploy/configs/execute-venue-live-futures.jsonc` | Venue_live config for Futures execution |
| `internal/application/execution/s416_futures_venue_acceptance_fill_test.go` | 32 adapter-level tests |
| `internal/actors/scopes/execute/s416_futures_venue_lifecycle_test.go` | 6 actor-composition tests |
| `scripts/smoke-futures-venue-live.sh` | S416 smoke script |
| `docs/architecture/futures-real-venue-connectivity-and-lifecycle-acceptance-fill-proof-on-unified-runtime.md` | Connectivity and lifecycle proof |
| `docs/architecture/futures-accepted-filled-real-response-alignment-controls-and-limitations.md` | Alignment, controls, and limitations |

### Modified Files

| File | Change |
|---|---|
| `Makefile` | Added `smoke-futures-venue-live` target and .PHONY entry |
| `docs/stages/INDEX.md` | Added Phase 45 section with S415--S416 entries |

## Principal Evidence

### FV-Q1: Venue Live Lifecycle Transitions (FULL)

| Test | Transition | Evidence |
|---|---|---|
| `TestS416_FuturesVenueLive_Buy_SubmittedToFilled` | submitted -> filled | VenueOrderID=67890, Status=filled |
| `TestS416_FuturesVenueLive_Sell_SubmittedToFilled` | submitted -> filled | Side=sell preserved |
| `TestS416_FuturesVenueLive_None_NoVenueContact` | submitted -> accepted | 0 HTTP calls, 0 fills |
| `TestS416_FuturesLifecycleAlignment_DominantPathValid` | ValidTransition() | Both steps valid |
| `TestS416_FuturesLifecycleAlignment_FilledIsTerminal` | Terminal absorbing | No outgoing transitions |
| `TestS416_FuturesLifecycleAlignment_BinanceStatusMapping` | All 6 statuses | NEW, FILLED, PARTIALLY_FILLED, CANCELED, REJECTED, EXPIRED |

### FV-Q2: Fill Record Fidelity

| Test | Field | Evidence |
|---|---|---|
| `TestS416_FuturesVenueLive_FillRecordFidelity` | Price=65432.10 (avgPrice) | From venue response |
| `TestS416_FuturesVenueLive_FillRecordFidelity` | Qty=0.001 | From executedQty |
| `TestS416_FuturesVenueLive_FillRecordFidelity` | Fee=65.43210 (cumQuote) | Fee proxy |
| `TestS416_FuturesVenueLive_FillRecordFidelity` | Simulated=false | Real venue fill |
| `TestS416_FuturesVenueLive_FillTimestampFromUpdateTime` | Timestamp | From updateTime ms |

### FV-Q3: Correlation Chain Integrity

| Test | Evidence |
|---|---|
| `TestS416_FuturesVenueLive_CorrelationChainPreserved` | CorrelationID and CausationID survive round-trip |
| `TestS416_FuturesVenueLive_IntentFieldPreservation` | Type, Source, Symbol, Timeframe, Risk.* preserved |

### FV-Q4: Post-200 Reconciliation

| Test | Evidence |
|---|---|
| `TestS416_FuturesVenueLive_QueryOrder_ReconcilesFill` | GET /fapi/v1/order returns correct fill |
| `TestS416_FuturesVenueLive_QueryOrder_UsesCorrectAPIPath` | Uses /fapi/v1/order (not /api/v3/order) |

### Segment Routing

| Test | Evidence |
|---|---|
| `TestS416_SegmentRouter_FuturesSourceDispatchesToFuturesAdapter` | Futures adapter called, Spot NOT called |
| `TestS416_SegmentForSource_Futures` | "binancef" -> MarketSegmentFutures |

### Spot vs Futures Structural Difference

| Test | Evidence |
|---|---|
| `TestS416_SpotFuturesDifference_APIPathAndResponseType` | /fapi/v1/order + RESULT vs /api/v3/order + FULL |
| `TestS416_FuturesVenueLive_RESULTResponseType` | Futures uses RESULT (not FULL) |
| `TestS416_FuturesVenueLive_FuturesAPIPath` | /fapi/v1/order confirmed |

### Actor Composition

| Test | Evidence |
|---|---|
| `TestS416_ActorComposition_FuturesVenueLive_Buy_Filled` | Full router path, segment isolation, correlation |
| `TestS416_ActorComposition_FuturesVenueLive_Sell_Filled` | Sell side through router |
| `TestS416_ActorComposition_FuturesVenueLive_None_NoContact` | No-op, 0 HTTP calls |
| `TestS416_ActorComposition_FuturesQueryOrder_Reconciliation` | Router-level query reconciliation |
| `TestS416_ActorComposition_DryRunDisabled_RealFuturesAdapterCalled` | dry_run=false -> real adapter |
| `TestS416_ActorComposition_DryRunEnabled_InterceptsFuturesAdapter` | dry_run=true -> DryRunSubmitter intercepts |

## Remaining Limitations

1. **cumQuote as fee proxy** -- not true commission (requires separate endpoint)
2. **No partial fill exercise** -- PARTIALLY_FILLED mapped but not testnet-exercised
3. **No real rejection exercise** -- deferred to S417
4. **Testnet behavioral differences** -- instant fills, synthetic liquidity
5. **Single symbol scope** -- BTCUSDT only (multi-symbol structurally supported)
6. **No position management** -- adapter does not manage leverage, margin, or positions

## Recommended Preparation for S417

S417 should prove:

1. **Rejection path**: `submitted -> rejected` with real Futures testnet rejection
   (insufficient margin, invalid parameters, invalid symbol)
2. **Partial fill handling**: `submitted -> accepted -> partially_filled -> filled`
3. **Rejection event emission**: `VenueOrderRejectedEvent` with Futures-specific codes
4. **Quantity monotonicity**: `FilledQuantity` non-decreasing across partial fills
5. **Rejection audit trail**: rejection reason, HTTP status, and venue error code preserved

The existing `VenueOrderRejectedEvent` infrastructure from S386 and the rejection
projection from S411 are ready for Futures rejection paths without structural changes.
