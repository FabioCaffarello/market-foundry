# Stage S471 — Decision Review Surface and Evidence Bundle

> Date: 2026-03-25
> Status: Complete
> Predecessor: S470 (Decision Lineage and Causality Model)
> Successor: S472 (Consistency Checks)

## Objective

Design, implement, validate, and document a minimal decision review surface and evidence bundle that connects signal inputs, decision transforms, strategy resolution, risk constraints, and execution output into a single auditable view.

## Strategic Context

After S470 established the causal lineage model (EventID on all Input types, lineage validation), the next step was to transform that infrastructure into a surface actually useful for decision review. This stage is NOT about dashboard/BI — it is about a focused review surface for understanding, auditing, and comparing how decisions were formed.

## Findings

### Pre-S471 State

The composite chain surface (`/analytical/composite/chain`) already reconstructed full causal chains from signal through execution. The infrastructure was solid:

1. **CompositeExecutionChain** with 5 optional stages (signal, decision, strategy, risk, execution)
2. **RiskAttribution** (S298) computed from the risk stage for Q2 explainability
3. **WithTrace types** carrying EventID/CorrelationID/CausationID from ClickHouse
4. **S470 EventID wiring** on all Input types for domain-level lineage

### Identified Gap

The composite chain was **execution-centric** — it starts from execution events and walks backward. An operator asking "why did this decision trigger, and what happened downstream?" had to mentally reproject the chain from the decision's perspective. There was no dedicated surface for:
- Decision-anchored review with structured inputs/transform/constraints/output
- Human-readable explanation synthesizing the full chain from the decision's viewpoint
- Batch comparison of decisions filtered by outcome

### Resolution

A pure read-side projection over the existing CompositeReader. No new ClickHouse queries, no new write-side changes, no new domain types. The `DecisionReviewBundle` re-projects the `CompositeExecutionChain` through five review sections.

## Changes

### New Files

| File | Purpose |
|---|---|
| `internal/application/analyticalclient/decision_review_contracts.go` | DecisionReviewQuery, DecisionReviewReply, DecisionReviewBundle, and five review section types |
| `internal/application/analyticalclient/get_decision_review.go` | GetDecisionReviewUseCase — projects chains to review bundles, builds explanations |
| `internal/application/analyticalclient/get_decision_review_test.go` | 7 unit tests covering full chain, partial chain, missing decision, outcome filter, validation, nil reader |

### Modified Files

| File | Change |
|---|---|
| `internal/interfaces/http/handlers/composite.go` | Added `getDecisionReviewUseCase` interface, `GetDecisionReview` and `GetDecisionReviews` handler methods |
| `internal/interfaces/http/routes/analytical.go` | Added `GetDecisionReview` to `AnalyticalFamilyDeps`, registered two routes |
| `cmd/gateway/compose.go` | Wired `GetDecisionReview` use case from existing `compositeReader` |

### New Architecture Documents

| File | Purpose |
|---|---|
| `docs/architecture/decision-review-surface-and-evidence-bundle.md` | Surface design, endpoints, data flow, limitations |
| `docs/architecture/decision-inputs-transforms-constraints-output-and-review-semantics.md` | Semantic model of each review section, what it shows vs. what it doesn't |

## HTTP Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/analytical/composite/decision/review` | Single decision review by correlation_id + symbol |
| GET | `/analytical/composite/decision/reviews` | Batch decision reviews by source/symbol/timeframe with optional outcome filter |

## DecisionReviewBundle Structure

```
DecisionReviewBundle
  +-- Inputs (signal evidence as recorded by decision)
  |     signals: []SignalInput
  |     event_id, at
  +-- Transform (decision evaluation — the core)
  |     type, outcome, severity, confidence, rationale, metadata
  |     event_id, at
  +-- Resolution (strategy resolved from decision)
  |     type, direction, confidence, decision_inputs, parameters
  |     event_id, at
  +-- Constraints (risk assessment applied)
  |     type, disposition, confidence, rationale, limits, strategy_context
  |     event_id, at
  +-- Output (execution intent and lifecycle)
  |     type, side, quantity, filled_quantity, status, final
  |     event_id, at
  +-- explanation (human-readable synthesis)
  +-- stage_count, chain_complete, missing_stages
```

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestGetDecisionReview_SingleChain_FullBundle` | Full 5-stage chain produces complete bundle with all sections |
| `TestGetDecisionReview_SingleChain_NoDecision_EmptyResult` | Chain without decision produces empty result (decision is mandatory anchor) |
| `TestGetDecisionReview_SingleChain_MissingSymbol` | Missing symbol returns InvalidArgument (S301 isolation) |
| `TestGetDecisionReview_Batch_OutcomeFilter` | Outcome filter correctly narrows results |
| `TestGetDecisionReview_Batch_ValidationErrors` | Missing source/symbol/timeframe each return InvalidArgument |
| `TestGetDecisionReview_NilReader` | Nil reader returns Unavailable |
| `TestGetDecisionReview_PartialChain_DecisionOnly` | Decision-only chain has nil Inputs/Resolution/Constraints/Output |

All 7 tests pass. All 3 binaries (gateway, execute, writer) build cleanly. No regressions in existing test suites.

## Acceptance Criteria Evaluation

| Criterion | Status | Evidence |
|---|---|---|
| Decision review is materially clearer and more usable | PASS | Single endpoint returns structured bundle with 5 sections and human-readable explanation |
| Evidence bundle reduces manual reconstruction | PASS | Operator no longer needs to correlate across 3-5 endpoints to review a decision |
| Stage improves explainability of core decisioning | PASS | Bundle includes rationale, severity, confidence, signal inputs, risk constraints, and a synthesis explanation |
| Base ready for consistency checks in S472 | PASS | Bundle structure with per-section event_ids and timestamps enables cross-section consistency validation |

## Limitations

1. **Batch mode is execution-rooted**: Decisions that never reached execution (e.g., `not_triggered`) are not returned in batch mode. This is inherited from the composite reader design — the batch starts from the executions table. Single-chain lookup works for any decision.

2. **No decision-first batch query**: Starting batch queries from the decisions table (rather than executions) would require a new ClickHouse query path. Deferred to a future stage if operator demand warrants it.

3. **No diff/comparison computation**: The surface returns individual bundles. Comparison across decisions is left to the consumer.

4. **No evaluator config exposure**: The bundle shows the decision's output but not the evaluator's thresholds or configuration. These live in the actor runtime, not in persisted events.

## Alignment

- **S296/S298**: Reuses composite chain and attribution infrastructure directly.
- **S301**: Symbol isolation enforced on all single-chain lookups.
- **S470**: EventID fields on Input types are surfaced in each bundle section's `event_id` field.
- **S455A**: Complements the session explain surface (execution/lifecycle-centric) without overlap.
- **Guard rails**: No dashboard/BI, no complex UI, no masking of explainability limits.

## Next Stage Direction

S472 should introduce consistency checks that validate cross-section invariants within a DecisionReviewBundle:
- Signal event_id in Inputs matches decision's SignalInput.EventID
- Strategy DecisionInput references match the Transform section
- Risk StrategyInput references match the Resolution section
- Execution CausationID traces back through the chain
- Temporal ordering (signal.at <= decision.at <= strategy.at <= risk.at <= execution.at)
