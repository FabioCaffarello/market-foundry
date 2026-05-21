# S472 -- Cross-Domain Consistency Checks

| Field | Value |
|-------|-------|
| Stage | S472 |
| Title | Cross-domain consistency checks for decision quality |
| Wave | Strategy-to-Execution Decision Quality (S469--S473) |
| Status | Complete |
| Date | 2026-03-25 |

## Objective

Reduce silent divergence between strategy, risk, decision, and execution domains by implementing structured, auditable cross-domain consistency checks.

## Context

After S470 (lineage) and S471 (review surface), the pipeline had causal traceability and a decision-centric review endpoint. However, cross-domain **semantic** invariants were unchecked --- a strategy could theoretically resolve `direction=long` while execution showed `side=sell` without any system raising an alert. S472 closes this gap for the most valuable invariants.

## Delivered

### New Package: `internal/domain/consistency`

Pure domain package with zero I/O dependencies.

- `ChainSnapshot` --- flat struct of primitive values from all four domains
- `Check(snap)` --- runs 9 cross-domain checks, returns structured `Report`
- `Finding` --- individual check result with check ID, severity, domain, message, got/expected
- `Report` --- aggregate with findings, counts, clean/dirty flag

### 9 Cross-Domain Checks

| Check | Boundary | Type |
|-------|----------|------|
| `severity_outcome` | decision | violation |
| `direction_side` | strategy -> execution | violation |
| `disposition_action` | risk -> execution | violation/warning |
| `symbol_coherence` | all stages | violation |
| `source_coherence` | all stages | violation |
| `timeframe_coherence` | all stages | violation |
| `confidence_progression` | strategy -> risk | warning |
| `disposition_propagation` | risk -> execution | violation |
| `direction_propagation` | strategy -> risk | violation |

### Integration: Decision Review Surface

- `DecisionReviewBundle.Consistency` field added (S472)
- `buildChainSnapshot()` projects `CompositeExecutionChain` into `ChainSnapshot`
- `Explanation` text now includes consistency violation/warning counts
- Available via `/api/v1/decision-review` endpoint

### Tests

18 unit tests covering:
- Clean chain (all stages consistent)
- Each check in isolation (positive and negative cases)
- Partial chains (decision-only)
- Direction variants (long/buy, short/sell, flat/none)
- Confidence discount factor (normal and anomalous)
- Disposition and direction propagation mismatches

All tests pass with zero regressions.

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/consistency/consistency.go` | New --- cross-domain consistency checks |
| `internal/domain/consistency/consistency_test.go` | New --- 18 tests |
| `internal/application/analyticalclient/decision_review_contracts.go` | Added `Consistency` field to `DecisionReviewBundle` |
| `internal/application/analyticalclient/get_decision_review.go` | Added `buildChainSnapshot()`, wired checks into `projectChainToReview()` |
| `docs/architecture/cross-domain-consistency-checks-for-decision-quality.md` | New --- architecture doc |
| `docs/architecture/strategy-risk-decision-execution-consistency-invariants-findings-and-limitations.md` | New --- invariant catalog |

## Guard Rails Observed

- **No rules engine inflation**: Checks are simple predicate functions, not a configurable engine.
- **No OMS expansion**: Execution-side checks are limited to side/quantity/disposition; no lifecycle or venue semantics.
- **No contract redesign**: All checks use existing domain types via primitive extraction.
- **Uncovered gaps documented**: 6 known unchecked invariants are explicitly listed with risk assessment.

## Residual Gaps

| Gap | Description | Risk |
|-----|-------------|------|
| G1 | Signal-decision input type consistency | Low |
| G2 | Quantity-constraint numeric alignment | Medium |
| G3 | Timestamp monotonicity across stages | Low |
| G5 | DecisionInput fidelity in strategy | Medium |
| G6 | Multi-strategy fan-out consistency | Low |

G4 (lineage EventID chain) is already covered by S470's `lineage.ValidateChain()`.

## Wave Status

| Stage | Scope | Status |
|-------|-------|--------|
| S469 | Wave charter | Complete |
| S470 | Decision lineage and causality model | Complete |
| S471 | Decision review surface and evidence bundle | Complete |
| S472 | Cross-domain consistency checks | **Complete** |
| S473 | Evidence gate | Pending |
