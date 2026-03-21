# Stage S252 — Scenario-Based End-to-End Domain Validation Report

**Date:** 2026-03-21
**Status:** Complete
**Predecessor:** S250 (decision-to-strategy behavior activation), S251 (strategy-to-risk behavior activation)
**Successor:** S253 (integration/CI hardening)

## Objective

Validate that the `decision -> strategy -> risk` chain produces coherent, observable behavioral output across representative domain scenarios — proving that the breadth delivered in S241–S244 and the behavioral enrichment in S250–S251 compose into a working end-to-end system.

## Delivered

### Test Artifacts

| File | Content |
|------|---------|
| `internal/actors/scopes/derive/scenario_end_to_end_test.go` | 6 end-to-end scenario tests with helper functions |

### Documentation Artifacts

| File | Content |
|------|---------|
| `docs/architecture/scenario-based-end-to-end-domain-validation.md` | Scenario design, selection criteria, validation architecture |
| `docs/architecture/domain-scenarios-results-and-behavioral-findings.md` | Quantitative results, behavioral findings, before/after comparison |

## Scenarios Validated

| # | Scenario | Chain | Assertion Focus |
|---|----------|-------|-----------------|
| 1 | RSI Oversold → Mean Reversion → Dual Risk | A | Dual-risk coherence, severity-adjusted params |
| 2 | EMA Crossover → Trend Following → Dual Risk | B | Pro-trend chain, strategy-type factors |
| 3 | Severity Contrast (High vs Low) | A×2 | Quantitative behavioral divergence |
| 4 | Cross-Chain Risk Profile | A vs B | Strategy-type-aware asymmetric risk |
| 5 | Not-Triggered (Both Chains) | A + B | Clean negative path |
| 6 | Context Preservation | A | Rationale survives 6 checkpoints |

**All 6 scenarios pass.** Total test suite runtime: ~690ms (including all existing derive tests).

## Key Findings

1. **Severity is behavioral, not decorative.** High severity produces 2.56× larger position size and 1.79× higher risk confidence compared to low severity through the same chain.

2. **Strategy type drives asymmetric risk.** Counter-trend (mean_reversion) receives 5.3% lower position confidence and 26.1% tighter stop ceiling than pro-trend (trend_following) from the same evaluator.

3. **Dual-risk fan-out is coherent.** A single strategy resolution produces two independent, valid risk assessments with distinct constraint types (MaxPositionSize vs StopDistance).

4. **Context preservation is complete.** Decision rationale text survives unchanged through all 6 pipeline stages from decision to risk metadata.

5. **Non-triggered paths are clean.** Both chains handle non-triggered decisions without errors, producing safe flat/approved defaults.

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| End-to-end scenarios exist and pass | Done | 6 tests in `scenario_end_to_end_test.go`, all green |
| Chain `decision -> strategy -> risk` is observable and coherent | Done | Scenarios 1, 2 validate full chain with dual-risk |
| Wave transcends local-domain enrichment | Done | Scenarios 3, 4 prove cross-domain behavioral composition |
| Base ready for CI hardening in S253 | Done | Tests are deterministic, fast (~690ms), no external dependencies |
| Scenarios are small, useful, and auditable | Done | 6 scenarios, each with clear assertions and t.Logf diagnostics |

## Guard Rails Compliance

| Guard rail | Status |
|-----------|--------|
| No large scenario matrix | 6 targeted scenarios, not exhaustive combinatorics |
| No new observability infrastructure | Uses existing msgCollector pattern |
| No artificial gap masking | All findings documented including simplifications |
| No new breadth expansion | Only validation of existing breadth |
| Limits documented | See findings document, section "Observed Simplifications" |

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/scenario_end_to_end_test.go` | **New** — 6 scenario tests + 2 chain helper functions |
| `docs/architecture/scenario-based-end-to-end-domain-validation.md` | **New** — scenario design document |
| `docs/architecture/domain-scenarios-results-and-behavioral-findings.md` | **New** — results and findings document |
| `docs/stages/stage-s252-scenario-based-end-to-end-domain-validation-report.md` | **New** — this report |

## Remaining Limits

1. **No real infrastructure tests.** Scenarios validate behavioral logic via actor messages, not NATS/ClickHouse round-trips.
2. **No SourceScopeActor integration.** Fan-out is simulated manually; full scope-level routing is a natural S253 target.
3. **No execution stage.** Paper order evaluation is excluded — chain validation ends at risk assessment.
4. **EMA severity is fixed.** `EMACrossoverEvaluator` always returns `SeverityModerate` for bullish; no graduated severity contrast available for Chain B.
5. **FormatParam precision.** 2-decimal formatting rounds sub-percent parameter differences (e.g., 0.0075 → 0.01).

## Preparation for S253

S253 can build on this foundation for integration/CI hardening:

1. **CI regression anchors.** The 6 scenario tests are deterministic and fast — add them to the CI pipeline as regression guards.
2. **Scope-level integration.** Consider a test that wires `SourceScopeActor` directly, validating real fan-out routing (not simulated).
3. **Writer/store round-trip.** Validate that published events are correctly consumed and stored by writer/store pipelines (requires test NATS + ClickHouse or mocks).
4. **HTTP projection.** Validate that risk assessments materialized in KV buckets are queryable via the gateway HTTP API.
