# Stage S496 — Continuity Review and Cross-Session Reconciliation

**Status**: COMPLETE
**Wave**: Cross-Session Continuity (S493–S497)
**Predecessor**: S495 (Cross-Session Read Model and Continuity Attribution)
**Successor**: S497 (Evidence Gate)

## Objective

Design, implement, validate, and document a minimal continuity review surface and cross-session reconciliation model that allows operators to review what was carried between sessions, what resolved, what remains open, and how reliable the data is.

## Delivered

### Domain Layer

**File**: `internal/domain/pairing/continuity_reconciliation.go`

1. **Three new reconciliation flags**:
   - `cross_session` — entry and exit originate from different sessions.
   - `boundary_carryover` — round-trip resolved after crossing a session boundary.
   - `cross_session_fee_gap` — fee data missing on cross-session pair.

2. **`ContinuityReconciliationResult`** — extends `ReconciliationResult` with continuity state, session provenance, and `carryover_reliable` assessment.

3. **`ReconcileCrossSessionRoundTrip`** — pure function that applies standard reconciliation + cross-session-specific flags.

4. **`ContinuityReconciliationSummary`** — aggregate statistics including cross-session counts, boundary carryover counts, flag distributions, and reliability counts.

5. **`SummarizeContinuityReconciliation`** — pure aggregation function.

### Application Layer

**Files**:
- `internal/application/analyticalclient/continuity_review_contracts.go`
- `internal/application/analyticalclient/get_continuity_review.go`

1. **`ContinuityReviewQuery`** — request contract with full filtering (continuity, cross_only, flagged, outcome).

2. **`ContinuityReviewReply`** — response contract with:
   - Per-round-trip `ContinuityReviewItem` (pairing + reconciliation + effectiveness)
   - `ContinuitySummary` (resolution rates, cross/intra counts)
   - `ContinuityReconciliationSummary` (flag distributions, reliability)
   - `ContinuityEffectivenessSummary` (P&L split by cross/intra session)

3. **`GetContinuityReviewUseCase`** — 7-step orchestration:
   - Fetch sessions from KV
   - Query chains per session from ClickHouse
   - Apply carry-forward eligibility
   - Sort legs by timestamp (FIFO M4)
   - Match with `MatchFIFO`
   - Annotate with session provenance + continuity
   - Build review items with reconciliation + effectiveness + filters
   - Compute all three summary projections

4. **`ContinuityEffectivenessSummary`** — splits effectiveness outcomes into cross-session and intra-session contributions (wins, losses, P&L per group).

### HTTP Layer

**Files**:
- `internal/interfaces/http/handlers/composite.go` — `GetContinuityReview` handler
- `internal/interfaces/http/routes/analytical.go` — route registration
- `cmd/gateway/compose.go` — use case wiring

**Endpoint**: `GET /analytical/composite/pairing/continuity-review`

Query parameters: `source`, `symbol`, `timeframe`, `since`, `until`, `max_sessions`, `continuity`, `cross_only`, `flagged`, `outcome`.

### Tests

| File | Tests | Coverage |
|------|-------|----------|
| `internal/domain/pairing/s496_continuity_reconciliation_test.go` | 7 | Intra-session clean, cross-session flags, fee gap, unmatched open, summary empty, summary mixed, no duplicate flags |
| `internal/application/analyticalclient/s496_continuity_review_test.go` | 6 | Unavailable, missing fields, no sessions, intra-session paired, cross-session paired, flagged filter |

**Result**: 13 new tests, zero regressions across all existing test suites (80+ pairing, 37+ continuity, 20+ effectiveness, 10+ review).

### Architecture Documentation

| Document | Purpose |
|----------|---------|
| `docs/architecture/continuity-review-and-cross-session-reconciliation.md` | Surface contract, data flow, alignment, limitations |
| `docs/architecture/carryover-boundary-fees-pairing-result-reconciliation-semantics-and-limitations.md` | Reconciliation semantics for carryover fragments |

## What the Review Surface Answers

1. **What was carried?** — `continuity.cross_session_paired_count` + individual review items with `cross_session=true`
2. **What resolved?** — `continuity.resolved_count` + `continuity.resolution_rate`
3. **What remains open?** — `continuity.open_count` + `continuity.artificial_unresolved_count`
4. **Is the data reliable?** — `reconciliation.carryover_reliable_count` vs `reconciliation.cross_session_count`
5. **What P&L did carryover produce?** — `effectiveness.cross_session_pnl`
6. **Which round-trips have quality issues?** — Query with `flagged=true`
7. **How does cross-session P&L compare to intra-session?** — `effectiveness.cross_session_pnl` vs `effectiveness.intra_session_pnl`

## What Remains Outside Scope

- Real-time carryover visibility during active sessions.
- Aggregated position tracking or netting.
- Dashboard or visualization layer.
- Multi-exchange cross-session pairing.
- Automated remediation of data-quality issues.

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Review surface is materially useful | **MET** — single endpoint answers all 7 continuity review questions |
| Cross-session reconciliation improves concretely | **MET** — 3 new flags + carryover reliability assessment |
| Stage increases pairing/effectiveness value | **MET** — effectiveness split by cross/intra session; reconciliation flags enrich review |
| Wave ready for evidence gate (S497) | **MET** — all surfaces operational, tests passing, docs complete |

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No broad dashboard | **COMPLIANT** — single review endpoint only |
| No generalized position analytics | **COMPLIANT** — round-trip review, not position tracking |
| No masking of continuity limits | **COMPLIANT** — limitations documented explicitly |
| No redesign of storage or venue model | **COMPLIANT** — additive only, existing infrastructure |

## Artifacts

### New Files
- `internal/domain/pairing/continuity_reconciliation.go`
- `internal/domain/pairing/s496_continuity_reconciliation_test.go`
- `internal/application/analyticalclient/continuity_review_contracts.go`
- `internal/application/analyticalclient/get_continuity_review.go`
- `internal/application/analyticalclient/s496_continuity_review_test.go`
- `docs/architecture/continuity-review-and-cross-session-reconciliation.md`
- `docs/architecture/carryover-boundary-fees-pairing-result-reconciliation-semantics-and-limitations.md`
- `docs/stages/stage-s496-continuity-review-report.md`

### Modified Files
- `internal/interfaces/http/handlers/composite.go` — added `GetContinuityReview` handler
- `internal/interfaces/http/routes/analytical.go` — added route + deps + interface
- `cmd/gateway/compose.go` — wired `GetContinuityReviewUseCase`

## Metrics

- **New domain types**: 3 (ContinuityReconciliationResult, ContinuityReconciliationSummary, 3 flags)
- **New application types**: 5 (query, reply, review item, effectiveness summary, meta)
- **New functions**: 3 (ReconcileCrossSessionRoundTrip, SummarizeContinuityReconciliation, buildContinuityEffectivenessSummary)
- **New tests**: 13 (7 domain + 6 application)
- **New endpoint**: 1 (`GET /analytical/composite/pairing/continuity-review`)
- **Regressions**: 0
