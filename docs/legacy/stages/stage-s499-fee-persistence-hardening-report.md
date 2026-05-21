# S499 — Fee Persistence and Reconciliation Hardening

**Wave**: Operational Hardening (S498+)
**Predecessor**: S498 (Operational Hardening Charter), S428 (Fee Normalization), S482 (Reconciliation)
**Status**: Complete

## Objective

Harden fee persistence, provenance, and reconciliation to reduce fragility between venue outcomes, write-path, read surfaces, pairing/effectiveness, and operational review.

## Scope

- Fee provenance tracking (FeeSource)
- Fee ratio anomaly detection
- Segment-aware verification
- Cost basis symmetry in effectiveness
- Fee coverage metrics in review surfaces
- FeeAsset uniformity validation

## Deliverables

### Code Changes

| Area | File | Change |
|------|------|--------|
| Domain: Execution | `internal/domain/execution/execution.go` | `FeeSource` type with 4 values, `FillRecord.FeeSource` field |
| Domain: Pairing | `internal/domain/pairing/pairing.go` | `Leg.FeeSource` field, propagation in `IntentToLeg` |
| Domain: Reconciliation | `internal/domain/pairing/reconciliation.go` | FeeSource-aware `FeeReliable`, `FlagFeeRatioAnomaly`, `FlagFeeSourceFallback`, `isFeeReliableLeg` |
| Domain: Effectiveness | `internal/domain/effectiveness/effectiveness.go` | `Attribution.ExitCostBasis` field |
| Adapter: Spot | `internal/application/execution/binance_spot_testnet_adapter.go` | `FeeSourceVenue`, `FeeSourceFallback`, mixed FeeAsset detection |
| Adapter: Futures | `internal/application/execution/binance_futures_testnet_adapter.go` | `FeeSourceUnavailable` |
| Adapter: Paper | `internal/application/execution/paper_venue_adapter.go` | `FeeSourceSimulated` |
| Adapter: DryRun | `internal/application/execution/dry_run_submitter.go` | `FeeSourceSimulated` |
| Adapter: Simulator | `internal/application/execution/paper_fill_simulator.go` | `FeeSourceSimulated` |
| UseCase: Verify | `internal/application/executionclient/verify_session.go` | Segment-aware `checkFeeFields` |
| UseCase: Review | `internal/application/analyticalclient/review_contracts.go` | `TotalCostBasis`, `FeeCoverageRatio` |
| UseCase: Review | `internal/application/analyticalclient/get_roundtrip_review.go` | Fee coverage computation |

### Tests

| File | New Tests |
|------|-----------|
| `internal/domain/pairing/reconciliation_test.go` | `FuturesFeeSourceUnavailableIsReliable`, `FeeRatioAnomaly`, `FeeRatioNormal`, `FeeSourceFallback` |
| `internal/domain/effectiveness/effectiveness_test.go` | `ExitCostBasisPopulated`, `SingleLeg_ExitCostBasisIsZero` |

**Total new tests**: 6
**Existing tests**: All passing, zero regressions

### Documentation

| Document | Purpose |
|----------|---------|
| `docs/architecture/fee-persistence-and-reconciliation-hardening.md` | Technical design and change inventory |
| `docs/architecture/fees-costs-commission-persistence-reconciliation-and-limitations.md` | Canonical fee flow reference with limitations |
| `docs/stages/stage-s499-fee-persistence-hardening-report.md` | This report |

## Key Design Decisions

### D1: FeeSource as Provenance Tag

FeeSource is carried on FillRecord and propagated to Leg, enabling all downstream surfaces to reason about why a fee has its current value. This is strictly additive — no existing behavior changes, empty FeeSource = legacy data.

### D2: FeeReliable Considers FeeSourceUnavailable

Futures round-trips with `FeeSource=unavailable` are now considered fee-reliable because the system explicitly acknowledges the zero is an API limitation, not a data gap. This prevents Futures round-trips from being permanently flagged as unreliable.

### D3: Fee Ratio Anomaly Threshold at 10%

The 10% threshold is conservative — normal Binance fees are 0.01%-0.1%. Anything above 10% strongly suggests data corruption. The threshold is a named constant for future adjustment.

### D4: Segment-Aware Verification

`checkFeeFields` now classifies fills by FeeSource before judging whether missing fees are a problem. This eliminates false positives for Futures and Paper sessions.

## What Was NOT Changed

- No ClickHouse DDL changes (FeeSource persists via JSON in existing `fills` column)
- No new NATS subjects or consumers
- No write-path logic changes beyond setting FeeSource
- No accounting, ledger, or portfolio expansion
- No Futures commission fetching (remains a known gap)
- No historical backfill of FeeSource

## Residual Gaps

| Gap | Severity | Mitigation |
|-----|----------|------------|
| Futures commission unknown | Medium | Flagged via `fee_gap`, FeeSource=`unavailable` acknowledged. Future: fetch from `GET /fapi/v1/userTrades` |
| Legacy data has no FeeSource | Low | Empty string treated as unknown; operators filter by date for reliable data |
| Fee ratio threshold static | Low | Named constant, adjustable per-stage |
| No cross-symbol fee aggregation | Low | Out of scope per guard rails |

## Acceptance Criteria

- [x] Fees/costs more persistently tracked with provenance (FeeSource)
- [x] Reconciliation is more precise (FeeSource-aware reliability, anomaly detection)
- [x] Pairing/effectiveness have more complete cost basis data (ExitCostBasis)
- [x] Verification produces fewer false positives (segment-aware fee checks)
- [x] Review surface exposes fee coverage metrics (TotalCostBasis, FeeCoverageRatio)
- [x] Base ready for lifecycle close hardening in S500

## Guard Rails Compliance

- [x] No accounting/ledger expansion
- [x] No analytics broadening
- [x] No masking of real asymmetries (Futures fee gap still flagged)
- [x] No portfolio/risk model expansion
