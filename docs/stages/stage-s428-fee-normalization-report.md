# Stage S428: Fee Normalization and Cross-Segment Consistency

> Status: complete | Date: 2026-03-23

## Objective

Design, implement, validate, and document a canonical fee normalization model for Spot and Futures, guaranteeing consistency between persistence, read-path, queryability, and operational semantics.

## Executive Summary

S428 resolved a critical semantic divergence in the `FillRecord.Fee` field: Spot stored actual trading commission while Futures stored the cumulative notional value (`cumQuote`). This made cross-segment fee queries unreliable and created a latent source of operational confusion.

The fix introduces two new fields (`FeeAsset`, `CostBasis`) to `FillRecord` and corrects the semantics so that `Fee` always means "trading commission" and `CostBasis` always means "notional value". The change is backwards-compatible at the JSON level and requires no DDL migration.

## Changes Made

### Domain Layer
- **`internal/domain/execution/execution.go`**: Added `FeeAsset` (string, omitempty) and `CostBasis` (string, omitempty) to `FillRecord`. Updated documentation to specify per-segment semantics.

### Adapter Layer
- **`internal/application/execution/binance_futures_testnet_adapter.go`**: Changed fill construction to set `Fee="0"`, `CostBasis=cumQuote` instead of `Fee=cumQuote`.
- **`internal/application/execution/binance_spot_testnet_adapter.go`**: Updated `computeSpotFillAggregates` to return `feeAsset`. Fill construction now sets `FeeAsset` and `CostBasis=cummulativeQuoteQty`.

### Test Layer
- **New**: `internal/application/execution/s428_fee_normalization_test.go` — 9 tests covering Spot single/multi-fill, Futures fill/partial-fill, Paper, DryRun, cross-segment invariant, JSON round-trip, and omitempty behavior.
- **Updated** (9 files): All Futures tests that previously asserted `Fee==cumQuote` now assert `Fee=="0"` and `CostBasis==cumQuote`. Spot tests enhanced with FeeAsset and CostBasis assertions.

### Documentation
- `docs/architecture/fee-normalization-model-and-cross-segment-consistency.md`
- `docs/architecture/fees-commission-assets-cross-segment-semantics-and-limitations.md`

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/execution/execution.go` | +FeeAsset, +CostBasis fields |
| `internal/application/execution/binance_futures_testnet_adapter.go` | Fee="0", CostBasis=cumQuote |
| `internal/application/execution/binance_spot_testnet_adapter.go` | +FeeAsset, +CostBasis, updated computeSpotFillAggregates |
| `internal/application/execution/s428_fee_normalization_test.go` | New: 9 tests |
| `internal/application/execution/binance_futures_testnet_adapter_test.go` | +Fee/CostBasis/FeeAsset assertions |
| `internal/application/execution/binance_spot_testnet_adapter_test.go` | +Fee/FeeAsset/CostBasis assertions |
| `internal/application/execution/s416_futures_venue_acceptance_fill_test.go` | Fee→0, +CostBasis |
| `internal/application/execution/s417_futures_rejection_partial_fill_test.go` | Fee→0, +CostBasis |
| `internal/application/execution/s418_futures_read_path_audit_test.go` | FillRecord structs normalized |
| `internal/application/execution/s422_futures_venue_connectivity_fill_test.go` | Fee→0, +CostBasis |
| `internal/application/execution/s423_futures_rejection_partial_fill_test.go` | Fee→0, +CostBasis |
| `internal/application/execution/s424_futures_read_path_consolidation_test.go` | FillRecord structs + assertions |
| `internal/actors/scopes/execute/s405_spot_venue_lifecycle_test.go` | +FeeAsset/CostBasis assertions |
| `internal/actors/scopes/execute/s416_futures_venue_lifecycle_test.go` | Fee→0, +CostBasis |
| `internal/actors/scopes/execute/s417_futures_rejection_partial_fill_test.go` | Fee→0, +CostBasis |
| `internal/actors/scopes/execute/s419_unified_compose_e2e_futures_test.go` | Fee→0, +CostBasis |
| `internal/actors/scopes/execute/s425_unified_compose_e2e_futures_test.go` | Fee→0, +CostBasis, struct literals |

## Test Evidence

All tests pass across affected packages:
- `internal/domain/execution` — 0 failures
- `internal/application/execution` — 0 failures (including 9 new S428 tests)
- `internal/actors/scopes/execute` — 0 failures
- `internal/adapters/clickhouse` — 0 failures
- `internal/adapters/clickhouse/writerpipeline` — 0 failures

## Remaining Limitations

1. **Futures commission unavailable**: The Binance Futures RESULT response type does not include commission. Real Futures commission requires a separate `/fapi/v1/userTrades` API call, which is out of scope.
2. **No historical backfill**: Pre-S428 Futures fills in ClickHouse still have cumQuote in the Fee field. Distinguishable by empty `fee_asset` and `cost_basis`.
3. **No accounting precision**: Values use venue-provided decimal strings, not arbitrary-precision types.
4. **Paper/DryRun have no CostBasis**: Simulated fills don't compute notional value.

## Preparation for S429

The normalized fee model provides a clean foundation for health/readiness signals:
- **Fee presence as health signal**: Spot fills with Fee > 0 confirm venue commission extraction is working.
- **CostBasis as volume metric**: Enables notional-volume-based readiness thresholds.
- **FeeAsset as configuration signal**: Non-empty FeeAsset confirms the venue returns commission metadata.
- The cross-segment semantic alignment enables unified dashboards and alerts that previously required per-segment special-casing.
