# Strategy-Risk-Decision-Execution Consistency Invariants, Findings, and Limitations

> S472 | Introduced 2026-03-25

## Invariant Catalog

This document catalogs all identified cross-domain invariants between the four decision pipeline domains, whether they are currently checked, and what remains uncovered.

### Checked Invariants (S472)

#### I1: Severity-Outcome Consistency (Decision)

- **Rule**: `outcome=triggered` requires `severity in {low, moderate, high}`. `outcome in {not_triggered, insufficient}` requires `severity=none`.
- **Rationale**: Severity quantifies signal strength; a triggered decision without severity is semantically incomplete. A non-triggered decision with severity is contradictory.
- **Evidence**: Domain enums show `SeverityNone` paired with `OutcomeNotTriggered` in all evaluators. The invariant was implicit in evaluator logic but never enforced cross-domain.

#### I2: Direction-Side Mapping (Strategy -> Execution)

- **Rule**: `direction=long -> side=buy`, `direction=short -> side=sell`, `direction=flat -> side=none`. Risk `disposition=rejected` overrides to `side=none`.
- **Rationale**: The direction-to-side mapping is the semantic bridge between strategy intent and execution action. Divergence here means the system is acting contrary to its own resolution.
- **Evidence**: `PaperOrderEvaluator.Evaluate()` implements this mapping. The check validates the result matches the input.

#### I3: Disposition-Action Consistency (Risk -> Execution)

- **Rule**: `disposition=rejected` must produce `side=none, quantity=0`. `disposition=approved/modified` with non-flat direction should produce actionable side.
- **Rationale**: Risk rejection is a hard gate. If execution proceeds despite rejection, the risk layer is effectively bypassed.
- **Evidence**: `PaperOrderEvaluator` checks `riskDisposition == "rejected"` and sets `side=SideNone`. The invariant formalizes this.

#### I4-I6: Symbol/Source/Timeframe Coherence (All Stages)

- **Rule**: These partition key components must be identical across all stages in a causal chain.
- **Rationale**: Each stage operates on the same instrument/exchange/timeframe. A mismatch means events from different partitions were incorrectly correlated.
- **Evidence**: Partition keys use `{source}.{symbol}.{timeframe}` consistently across all domains.

#### I7: Confidence Progression (Strategy -> Risk)

- **Rule**: `risk_confidence <= strategy_confidence` (soft invariant, warning only).
- **Rationale**: Risk evaluators apply a confidence discount factor (e.g., x0.95 for pro-trend, x0.90 for counter-trend). Risk confidence exceeding strategy confidence violates this expectation.
- **Evidence**: `PositionExposureEvaluator` applies `confidence * confidenceFactor` where factors are <= 1.0.

#### I8: Disposition Propagation (Risk -> Execution)

- **Rule**: `execution.risk.disposition` must equal `risk.disposition` from the originating assessment.
- **Rationale**: The execution domain records the risk gate outcome for traceability. If this value diverges, audit trails are unreliable.
- **Evidence**: `PaperOrderEvaluator` copies `riskDisposition` directly into `RiskInput.Disposition`.

#### I9: Direction Propagation (Strategy -> Risk)

- **Rule**: `risk.strategies[0].direction` must equal `strategy.direction`.
- **Rationale**: Risk evaluates the strategy's direction to determine position sizing and exposure. If the propagated direction differs, risk decisions are based on wrong inputs.
- **Evidence**: `PositionExposureEvaluator` receives `strategyDirection` as a parameter and copies it into `StrategyInput.Direction`.

### Unchecked Invariants (Known Gaps)

#### G1: Signal-Decision Input Consistency

- **Rule**: Decision's `SignalInput` types should correspond to signal types actually present in the chain.
- **Status**: Not checked. Would require the consistency layer to understand signal type taxonomy.
- **Risk**: Low. Signal-decision wiring is tightly controlled by evaluator actors.

#### G2: Quantity-Constraint Alignment

- **Rule**: Execution quantity should be <= risk MaxPositionSize.
- **Status**: Not checked. Would require decimal parsing of string values.
- **Risk**: Medium. The evaluator enforces this, but a bug could silently over-size.

#### G3: Timestamp Monotonicity

- **Rule**: Timestamps should be non-decreasing along the causal chain (decision.ts <= strategy.ts <= risk.ts <= execution.ts).
- **Status**: Not checked. Clock skew between stages could produce legitimate violations.
- **Risk**: Low. Events flow unidirectionally through the actor graph.

#### G4: Lineage EventID Chain

- **Rule**: Each stage's CausationID should match the previous stage's EventID.
- **Status**: Covered by `lineage.ValidateChain()` (S470). Not duplicated here.

#### G5: DecisionInput Fidelity in Strategy

- **Rule**: Strategy's `DecisionInput` fields (type, outcome, confidence, severity) should match the originating decision's actual values.
- **Status**: Not checked. Would require comparing values from two different stage snapshots.
- **Risk**: Medium. The resolver copies these manually.

#### G6: Multi-Strategy / Multi-Risk Fan-Out

- **Rule**: When a single decision produces multiple strategies, each should be independently assessed.
- **Status**: Not modeled. Current snapshot assumes 1:1 stage relationships.
- **Risk**: Low. Current architecture uses 1:1 fan-out per strategy family.

## Findings Summary

| Category | Count | Status |
|----------|-------|--------|
| Checked invariants (hard) | 7 | Enforced as violations |
| Checked invariants (soft) | 2 | Enforced as warnings |
| Known unchecked gaps | 6 | Documented, not yet implemented |

## Limitations

1. The consistency package is a **read-side check**, not a write-side guard. It does not prevent inconsistent data from being persisted.
2. Adding write-side guards (rejecting inconsistent events) would require careful consideration of false positives and backward compatibility.
3. The `ChainSnapshot` model is deliberately flat. Rich checks (e.g., quantity vs constraints) would need numeric parsing that adds complexity.
4. Checks are intentionally scoped to the four decision-chain domains. OMS, venue lifecycle, and fill semantics are out of scope.
