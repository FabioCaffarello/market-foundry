# Stage S482 — Round-Trip Review and Outcome Reconciliation

**Wave**: Round-Trip Pairing (S479--S483)
**Stage type**: Review surface + reconciliation layer
**Date**: 2026-03-26
**Predecessor**: S481 (Pairing Read Model and Attribution Integration)

---

## 1. Objective

Design, implement, validate, and document a minimal review surface per round-trip and an explicit reconciliation between fills, fees, pairing, and outcome effectiveness — enabling operators to understand what closed, what didn't, and how reliable the data is.

---

## 2. What Was Done

### 2.1 Codebase Analysis

Mapped the complete data flow from ClickHouse fills through pairing to effectiveness:
- `internal/domain/pairing/pairing.go` — MatchFIFO, IntentToLeg, RoundTrip, Leg types (S480)
- `internal/domain/effectiveness/effectiveness.go` — Classify, ClassifyPair, Attribution (S476)
- `internal/application/analyticalclient/get_pairing.go` — GetPairingUseCase with attribution (S481)
- `internal/application/analyticalclient/pairing_contracts.go` — PairingQuery/Reply (S481)

**Key finding**: The pairing surface (S481) provides structural matching and P&L attribution but does not surface data-quality signals. An operator cannot tell whether a "win" round-trip has reliable fee data or whether an "unresolved" outcome is due to zero cost basis (paper) vs. missing exit. The reconciliation layer closes this gap.

### 2.2 Domain Reconciliation Layer

Created `internal/domain/pairing/reconciliation.go`:

| Type | Purpose |
|------|---------|
| `ReconciliationFlag` | Named data-quality condition (fee_gap, cost_basis_zero, simulated, etc.) |
| `ReconciliationResult` | Aggregated flags + clean/fee_reliable/pnl_reliable signals per round-trip |
| `ReconcileRoundTrip()` | Pure function: examines RoundTrip + Attribution → ReconciliationResult |

**8 reconciliation flags:**

| Flag | Condition | Impact |
|------|-----------|--------|
| `fee_gap` | Zero fees on one/both legs | Net P&L may overstate return |
| `cost_basis_zero` | Zero cost basis on one/both legs | P&L unclassifiable |
| `simulated` | Paper/dry-run fill | Not real-money attribution |
| `unmatched_open` | Entry without exit | Position open, no realized P&L |
| `orphan_exit` | Exit without entry | Data gap |
| `fee_asset_mismatch` | Different fee denomination on legs | Fee comparison approximate |
| `outcome_unresolved` | Paired but outcome still unresolved | Cost-basis issue on paired trade |
| `partial_remainder` | Result of partial-fill quantity split | (reserved for future detection) |

### 2.3 Application Layer: Review Use Case

Created `internal/application/analyticalclient/get_roundtrip_review.go`:

| Feature | Implementation |
|---------|---------------|
| Batch review | Fetches chains, converts to legs, runs MatchFIFO, classifies, reconciles |
| Single-chain review | Shows one chain's round-trip with reconciliation |
| Outcome filter | Filter to win/loss/breakeven/unresolved |
| Flagged filter | Only return round-trips with data-quality flags |
| State/side filters | Same as pairing surface |
| Summary | Pairing counts + effectiveness breakdown + reconciliation aggregates |

### 2.4 Contracts

Created `internal/application/analyticalclient/review_contracts.go`:

| Type | Purpose |
|------|---------|
| `RoundTripReviewQuery` | Request: extends pairing query with outcome and flagged filters |
| `RoundTripReviewReply` | Response: review items + summary + meta |
| `RoundTripReviewItem` | RoundTrip + Attribution + ReconciliationResult |
| `ReviewSummary` | Pairing stats + effectiveness + flag_counts + reliability counts |
| `ReviewMeta` | Diagnostic: total_ms, chains_scanned, round_trips, reviewed |

### 2.5 HTTP Endpoints

| Endpoint | Method | Handler |
|----------|--------|---------|
| `/analytical/composite/pairing/review` | GET | `CompositeWebHandler.GetRoundTripReview` |
| `/analytical/composite/pairing/review/chain` | GET | `CompositeWebHandler.GetRoundTripReviewSingle` |

Wired through:
- `handlers/composite.go` — Handler methods with Server-Timing headers
- `routes/analytical.go` — Route registration with nil-check gating
- `cmd/gateway/compose.go` — Use case instantiation from CompositeReader

### 2.6 Test Coverage

**Domain tests** (`internal/domain/pairing/reconciliation_test.go`):

| Test | What It Validates |
|------|-------------------|
| `TestReconcileRoundTrip_CleanPair` | Paired with fees and cost basis → clean, fee_reliable, pnl_reliable |
| `TestReconcileRoundTrip_FeeGap` | Zero fees → fee_gap flag, fee_reliable=false |
| `TestReconcileRoundTrip_CostBasisZero` | Zero cost basis → cost_basis_zero + fee_gap + outcome_unresolved |
| `TestReconcileRoundTrip_Simulated` | Paper fill → simulated flag |
| `TestReconcileRoundTrip_UnmatchedEntry` | No exit → unmatched_open flag |
| `TestReconcileRoundTrip_OrphanExit` | No entry → orphan_exit flag |
| `TestReconcileRoundTrip_FeeAssetMismatch` | Different fee assets → fee_asset_mismatch flag |

**Application tests** (`internal/application/analyticalclient/s482_roundtrip_review_test.go`):

| Test | What It Validates |
|------|-------------------|
| `TestGetRoundTripReview_Batch_PairedClean` | Clean pair → clean=true, pnl_reliable=true, win attribution |
| `TestGetRoundTripReview_Batch_FeeGapFlagged` | Zero fees → flagged with fee_gap |
| `TestGetRoundTripReview_Batch_UnmatchedEntryFlagged` | Lone buy → unmatched_open flag |
| `TestGetRoundTripReview_OutcomeFilter` | outcome=win filters to only winning round-trips |
| `TestGetRoundTripReview_FlaggedFilter` | flagged=true filters to only flagged round-trips |
| `TestGetRoundTripReview_ValidationErrors` | Missing source/symbol/timeframe → InvalidArgument |
| `TestGetRoundTripReview_NilUseCase` | Nil use case → Unavailable |
| `TestGetRoundTripReview_SimulatedFlagged` | Paper fills → simulated flag |
| `TestGetRoundTripReview_SummaryReconciliationCounts` | Mixed clean/flagged → correct summary counts |
| `TestGetRoundTripReview_RejectedExcluded` | Rejected chains excluded from review |

---

## 3. Architecture Docs Produced

| Document | What It Covers |
|----------|---------------|
| `round-trip-review-and-outcome-reconciliation.md` | Review surface design, endpoints, flags, reliability signals, relationship to existing surfaces |
| `fills-fees-pairing-result-reconciliation-semantics-and-limitations.md` | Canonical reference for fill aggregation, fee semantics by segment, reconciliation rules, cross-surface consistency invariants |

---

## 4. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/domain/pairing/reconciliation.go` | ReconciliationFlag, ReconciliationResult, ReconcileRoundTrip |
| `internal/domain/pairing/reconciliation_test.go` | 7 domain reconciliation tests |
| `internal/application/analyticalclient/review_contracts.go` | RoundTripReviewQuery, RoundTripReviewReply, RoundTripReviewItem, ReviewSummary |
| `internal/application/analyticalclient/get_roundtrip_review.go` | GetRoundTripReviewUseCase |
| `internal/application/analyticalclient/s482_roundtrip_review_test.go` | 10 application-level tests |
| `docs/architecture/round-trip-review-and-outcome-reconciliation.md` | Review surface architecture |
| `docs/architecture/fills-fees-pairing-result-reconciliation-semantics-and-limitations.md` | Reconciliation semantics |

### Modified Files

| File | Change |
|------|--------|
| `internal/interfaces/http/handlers/composite.go` | Added getRoundTripReviewUseCase interface, handler struct field, deps, GetRoundTripReview and GetRoundTripReviewSingle handlers |
| `internal/interfaces/http/routes/analytical.go` | Added GetRoundTripReview to AnalyticalFamilyDeps, route registration for review endpoints |
| `cmd/gateway/compose.go` | Wired GetRoundTripReviewUseCase from CompositeReader |

---

## 5. What the Surface Answers

| Question | Answer Mechanism |
|----------|-----------------|
| What round-trips closed in this period? | `state=paired` filter → paired round-trips with P&L |
| What positions are still open? | `state=unmatched_entry` → entries without exits |
| Is this round-trip's P&L reliable? | `reconciliation.pnl_reliable` field |
| Are the fees trustworthy? | `reconciliation.fee_reliable` field |
| Which round-trips have data quality issues? | `flagged=true` filter → only flagged items |
| What types of issues exist in the cohort? | `summary.flag_counts` → count per flag type |
| How does fee gap affect my win rate? | Compare `pnl_reliable_count` vs `paired_count` |

---

## 6. What Remains Outside

| Topic | Why It's Out |
|-------|-------------|
| Cross-session position reconciliation | Requires position tracking (non-goal for this wave) |
| Futures fee recovery | Venue API limitation; cannot be solved at application layer |
| Fee-asset currency conversion | Requires price feeds for BNB/USDT conversion (out of scope) |
| Historical correction log | Reconciliation is computed at read time; no write-path audit trail |
| Dashboard/visualization | Guard rail: no broad dashboard in this stage |

---

## 7. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Review surface per round-trip is materially useful | **Met** — operators can filter by outcome, state, flagged; see P&L reliability per item |
| Reconciliation between pairing and outcome improves concretely | **Met** — 8 explicit flags, 3 reliability signals, per-item and per-cohort |
| Stage increases measurement layer value | **Met** — review surface adds data-quality dimension absent from pairing and effectiveness |
| Wave ready for evidence gate in S483 | **Met** — pairing model (S480), read model (S481), and review+reconciliation (S482) complete |

---

## 8. Guard Rails Compliance

| Guard Rail | Compliance |
|------------|-----------|
| No broad dashboard | Only round-trip review endpoints; no analytics platform |
| No generalized trade analytics | Scoped to round-trip reconciliation only |
| No masking of pairing limits | Flags explicitly surface every known data gap |
| No redesign of storage or venue model | Zero ClickHouse changes; zero write-path changes |
