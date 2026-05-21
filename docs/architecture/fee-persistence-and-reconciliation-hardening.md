# Fee Persistence and Reconciliation Hardening

S499: Hardening of fee persistence, provenance, and reconciliation across Spot, Futures, and Paper/DryRun execution paths.

## Problem Statement

After S428 (fee normalization), the system correctly separates Fee, FeeAsset, and CostBasis across segments. However, several fragilities remained:

1. **No fee provenance** — downstream surfaces could not distinguish "fee=0 because paper" from "fee=0 because Futures API limitation" from "fee=0 unexpectedly for Spot".
2. **Verification was segment-blind** — `checkFeeFields` always warned for Futures sessions (false positive).
3. **No fee ratio anomaly detection** — data corruption in fee/cost_basis went undetected.
4. **Missing ExitCostBasis** in effectiveness Attribution — operators couldn't verify cost basis symmetry.
5. **No fee coverage metrics** in review surface — available in audit bundle but not in round-trip review.
6. **No FeeAsset uniformity validation** — multi-fill Spot orders assumed uniform CommissionAsset without checking.

## Changes

### 1. FeeSource Provenance (FillRecord)

Added `FeeSource` field to `FillRecord` with four values:

| FeeSource | Meaning | Set By |
|-----------|---------|--------|
| `venue` | Real commission from exchange | Spot adapter (fills[] present) |
| `unavailable` | Venue API does not return commission | Futures adapter (RESULT response) |
| `simulated` | Paper/dry-run — no real fee | PaperVenueAdapter, DryRunSubmitter, PaperFillSimulator |
| `fallback` | Spot without fills[] array (unexpected) | Spot adapter fallback path |

Propagated through:
- `execution.FillRecord` → NATS events → ClickHouse writer → read surfaces
- `pairing.Leg` → round-trip matching → reconciliation

### 2. FeeSource-Aware Reconciliation

Updated `ReconcileRoundTrip`:
- **FeeReliable** now considers `FeeSourceUnavailable` as reliable (acknowledged gap, not data corruption).
- **FlagFeeRatioAnomaly**: fee > 10% of cost_basis on either leg signals data corruption.
- **FlagFeeSourceFallback**: leg used the unexpected Spot fallback path.

### 3. ExitCostBasis in Attribution

Added `ExitCostBasis` to `effectiveness.Attribution`, populated in `ClassifyPair`. Enables:
- Cost basis symmetry verification
- Fee-to-volume ratio computation per round-trip

### 4. Segment-Aware Fee Verification

Updated `checkFeeFields` in session verification to classify fills by `FeeSource`:
- `venue` fills: expected to have non-zero fee → warn if zero
- `unavailable` / `simulated` fills: expected zero → pass
- `fallback` fills: unexpected → warn

This eliminates false positives for Futures sessions.

### 5. Fee Coverage in Review Summary

Added to `ReviewSummary`:
- `TotalCostBasis`: sum of entry+exit cost basis across paired round-trips
- `FeeCoverageRatio`: "N/M" string showing fills with fee / total fills

### 6. FeeAsset Uniformity Validation

`computeSpotFillAggregates` now returns a `mixed` boolean when fills have different CommissionAssets. The caller can use this for logging/flagging.

## Reconciliation Flag Inventory (Post-S499)

| Flag | Condition | Segment Impact |
|------|-----------|---------------|
| `fee_gap` | One or both legs have fee=0 | Futures (always), Paper (always), Spot fallback |
| `cost_basis_zero` | One or both legs have cost_basis=0 | Paper/DryRun |
| `simulated` | At least one leg is paper/dry-run | Paper/DryRun |
| `fee_asset_mismatch` | Entry and exit have different fee assets | Spot (rare) |
| `outcome_unresolved` | Paired but outcome unclassifiable | Any |
| `partial_remainder` | Quantity split from partial match | Any |
| `unmatched_open` | Entry without exit | Any |
| `orphan_exit` | Exit without entry | Any |
| `fee_ratio_anomaly` | Fee/cost_basis > 10% | Spot (data corruption) |
| `fee_source_fallback` | Spot fill without fills[] array | Spot (unexpected) |

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/execution/execution.go` | FeeSource type + constants, FillRecord.FeeSource field |
| `internal/domain/pairing/pairing.go` | Leg.FeeSource field, IntentToLeg propagation |
| `internal/domain/pairing/reconciliation.go` | FeeSource-aware FeeReliable, FlagFeeRatioAnomaly, FlagFeeSourceFallback |
| `internal/domain/effectiveness/effectiveness.go` | ExitCostBasis field in Attribution |
| `internal/application/execution/binance_spot_testnet_adapter.go` | FeeSourceVenue, FeeSourceFallback, mixed FeeAsset detection |
| `internal/application/execution/binance_futures_testnet_adapter.go` | FeeSourceUnavailable |
| `internal/application/execution/paper_venue_adapter.go` | FeeSourceSimulated |
| `internal/application/execution/dry_run_submitter.go` | FeeSourceSimulated |
| `internal/application/execution/paper_fill_simulator.go` | FeeSourceSimulated |
| `internal/application/executionclient/verify_session.go` | Segment-aware checkFeeFields |
| `internal/application/analyticalclient/review_contracts.go` | TotalCostBasis, FeeCoverageRatio in ReviewSummary |
| `internal/application/analyticalclient/get_roundtrip_review.go` | Fee coverage computation in buildReviewSummary |

## Tests Added

| File | Tests |
|------|-------|
| `internal/domain/pairing/reconciliation_test.go` | FuturesFeeSourceUnavailableIsReliable, FeeRatioAnomaly, FeeRatioNormal, FeeSourceFallback |
| `internal/domain/effectiveness/effectiveness_test.go` | ExitCostBasisPopulated, SingleLeg_ExitCostBasisIsZero |

## Guard Rails

- No new ClickHouse tables or DDL changes.
- No accounting/ledger expansion.
- No portfolio analytics.
- FeeSource is additive; existing data without FeeSource continues to work (empty string = legacy).
- Reconciliation remains a pure read-path computation.
