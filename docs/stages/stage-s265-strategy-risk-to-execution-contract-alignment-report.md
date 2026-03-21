# Stage S265 — Strategy/Risk to Execution Contract Alignment Report

Date: 2026-03-21
Wave: PAPER-EXECUTION-WAVE-1
Type: Boundary Alignment and Wiring Validation
Verdict: PASS — Boundary aligned, three gaps fixed, all tests green

---

## Executive Summary

Stage S265 formally aligns the contract between `strategy/risk` and `execution` domains, fixing three concrete boundary gaps discovered during wiring validation. The `PaperOrderEvaluator` now receives full causal context (`StrategyType`, `DecisionSeverity`) across the risk → execution boundary, and the semantic mismatch where drawdown's `StopDistance` was incorrectly mapped as position size has been corrected to use `MaxExposure`.

All existing tests pass. Two new tests validate the S265 fixes: boundary context preservation and drawdown semantic correctness. The build compiles clean across all 8 binaries. The base is ready for S266 (end-to-end paper loop scenario implementation).

## Formal Assessment

### Were there boundary alignment gaps?

| Gap | Severity | Fix |
|-----|----------|-----|
| `StrategyType` dropped at risk → execution boundary | Medium | Added to `riskAssessedMessage`, `PaperOrderEvaluator.Evaluate()`, `RiskInput`, and `Parameters` |
| `DecisionSeverity` dropped at risk → execution boundary | Medium | Added to `PaperOrderEvaluator.Evaluate()`, `RiskInput`, and `Parameters` (already in `riskAssessedMessage`) |
| Drawdown `StopDistance` mapped as `MaxPositionPct` | High | `DrawdownLimitEvaluatorActor` now sends `Constraints.MaxExposure` instead of `Constraints.StopDistance` |

### Is the boundary now coherent?

| Criterion | Status |
|-----------|--------|
| All causal context survives from risk to execution intent | Yes — `StrategyType`, `DecisionSeverity`, `CorrelationID`, `CausationID` all preserved |
| Semantic correctness of constraint mapping per risk type | Yes — position_exposure uses `MaxPositionSize`, drawdown_limit uses `MaxExposure` |
| Domain isolation maintained | Yes — execution never imports from risk/strategy domains; all fields cross as primitives |
| No execution-side domain logic introduced | Yes — execution is a thin translator, not a decision-maker |
| Safety gates remain functional | Yes — `SafetyGate`, `StalenessGuard`, `ControlGate` unchanged and tested |

## Changes Made

### Code changes

| File | Change | Rationale |
|------|--------|-----------|
| `internal/domain/execution/execution.go` | Added `StrategyType` and `DecisionSeverity` to `RiskInput` struct | Preserve full causal context in execution intent |
| `internal/application/execution/paper_order_evaluator.go` | Extended `Evaluate()` signature with `strategyType`, `decisionSeverity`; propagated to `RiskInput` and `Parameters` | Boundary fields must flow through evaluator |
| `internal/actors/scopes/derive/messages.go` | Added `StrategyType` field to `riskAssessedMessage` | Carry strategy family identity across the boundary |
| `internal/actors/scopes/derive/risk_evaluator_actor.go` | Populate `StrategyType` from `assessment.Strategies[0].Type` in fan-out | Source the field from risk assessment |
| `internal/actors/scopes/derive/drawdown_limit_evaluator_actor.go` | Changed `MaxPositionPct` from `StopDistance` to `MaxExposure`; populate `StrategyType` | Fix semantic mismatch; add strategy type |
| `internal/actors/scopes/derive/execution_evaluator_actor.go` | Pass `msg.StrategyType` and `msg.DecisionSeverity` to `Evaluate()` | Wire new fields through actor |

### Test changes

| File | Change |
|------|--------|
| `internal/application/execution/paper_order_evaluator_test.go` | Updated all `Evaluate()` calls to new 10-param signature; added `TestPaperOrderEvaluator_RiskInput_PreservesStrategyTypeAndSeverity` and `TestPaperOrderEvaluator_DrawdownRisk_ProducesBuyWithExposureQuantity` |
| `internal/application/execution/pipeline_integration_test.go` | Updated all `Evaluate()` calls to new 10-param signature |

### Architecture documents

| Document | Purpose |
|----------|---------|
| `docs/architecture/strategy-risk-to-execution-contract-alignment.md` | Defines the aligned boundary contract, field mappings, and responsibility split |
| `docs/architecture/execution-intent-boundary-and-safety-semantics.md` | Defines execution intent lifecycle, safety gate semantics, and field survival guarantees |

## Gains, Trade-offs, and Debts

### Gains

- **Full causal traceability**: Decision severity and strategy type now survive from their origin through to execution fill events
- **Semantic correctness**: Drawdown-originated execution intents now carry a meaningful position constraint (drawdown tolerance %) instead of a stop distance
- **Explicit boundary documentation**: The risk → execution contract is now formally documented with field-by-field mapping

### Trade-offs

- **`Evaluate()` now has 10 parameters**: Increasing parameter count adds visual complexity. This is acceptable because the function is called in exactly 2 places (actor + tests) and the parameters are all domain primitives with distinct types.
- **`RiskInput` grew by 2 fields**: The struct is still small (6 fields) and the new fields are `omitempty` for backward compatibility.

### Open debts

- **No end-to-end scenario proof yet**: The boundary is aligned but the full loop (decision → strategy → risk → execution → fill → projection) is not yet proven as an integrated scenario. This is S266's deliverable.
- **SafetyGate is not yet integrated into the paper loop test path**: Safety gate checks exist in the publisher actor but are not yet proven in an integrated scenario. This is S267's deliverable.

## Deliverables

| Path | Status |
|------|--------|
| `docs/architecture/strategy-risk-to-execution-contract-alignment.md` | Delivered |
| `docs/architecture/execution-intent-boundary-and-safety-semantics.md` | Delivered |
| `docs/stages/stage-s265-strategy-risk-to-execution-contract-alignment-report.md` | Delivered (this document) |
| Code changes (6 files) | Delivered — build clean, all tests pass |
| Test updates (2 files, 2 new tests) | Delivered — all assertions pass |

## Acceptance Criteria Checklist

- [x] Boundary `strategy/risk → execution` is explicit and coherent
- [x] Fields and semantics needed for paper execution are clear and documented
- [x] Domain vs execution responsibilities remain distinct (no execution-side domain logic)
- [x] Base ready for controlled paper order generation in S266
- [x] No improper semantic coupling introduced
- [x] No venue real opened
- [x] No OMS or generic order engine created
- [x] No domain logic collapsed into execution logic
- [x] No contracts inflated beyond necessity
- [x] Limits and trade-offs documented
