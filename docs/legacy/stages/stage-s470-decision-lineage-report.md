# Stage S470 — Decision Lineage and Causality Model

> Date: 2026-03-25
> Status: Complete
> Predecessor: S469
> Successor: S471 (review surface)

## Objective

Design, implement, validate, and document a canonical model of decision lineage and causality, making explicit the relationships between signal inputs, strategic transformation, resulting decision, and execution intent throughout the pipeline.

## Strategic Context

After S469, the first priority of this wave is to make the causal lineage between signal, strategy, decision, and execution more explicit and auditable. This stage is NOT about new domain capabilities — it is about LINEAGE MODEL AND DECISION CAUSALITY.

## Findings

### Pre-S470 State

The pipeline already had a solid causality infrastructure:

1. **CorrelationID/CausationID** in `events.Metadata` flowed through all five stages
2. **Semantic depth forwarding** carried DecisionSeverity and StrategyType across boundaries
3. **CompositeExecutionChain** reconstructed full chains from ClickHouse via CorrelationID
4. **ExecutionIntent** was the only domain type with explicit CorrelationID/CausationID fields

### Identified Gaps

| Gap | Description | Impact |
|---|---|---|
| G1 | Input types (SignalInput, DecisionInput, StrategyInput, RiskInput) had no EventID reference to parent stage | Domain-level lineage was implicit — required event metadata to trace causation |
| G2 | No formal lineage model or vocabulary | Chain validation was ad-hoc; no canonical invariant definitions |
| G3 | Signal/Decision/Strategy/Risk domain types lacked CorrelationID/CausationID | Only event envelopes carried trace IDs; domain objects were causally opaque |

### Resolution

G1 was fully resolved by adding `EventID` to all four Input types and wiring them through the actor layer. G2 was resolved by creating the `lineage` package with formal validation. G3 was intentionally NOT resolved — adding CorrelationID/CausationID to all domain types would duplicate event metadata and require ClickHouse schema changes. The Input-level EventID provides equivalent traceability with minimal disruption.

## Changes

### Domain Types

| File | Change |
|---|---|
| `internal/domain/decision/decision.go` | Added `EventID string` to `SignalInput` |
| `internal/domain/strategy/strategy.go` | Added `EventID string` to `DecisionInput` |
| `internal/domain/risk/risk.go` | Added `EventID string` to `StrategyInput` |
| `internal/domain/execution/execution.go` | Added `EventID string` to `RiskInput` |

### Lineage Package (new)

| File | Purpose |
|---|---|
| `internal/domain/lineage/lineage.go` | Stage constants, ChainLink/Chain types, ValidateChain(), IsComplete(), MissingStages() |
| `internal/domain/lineage/lineage_test.go` | 9 test cases: ordering, completeness, broken causation, correlation mismatch |

### Actor Wiring (11 files)

All decision evaluators (3), strategy resolvers (3), risk evaluators (2), and the execution evaluator (1) were updated to enrich Input types with the parent event's ID after the pure evaluator returns.

| Actor | Enrichment |
|---|---|
| RSIOversoldEvaluatorActor | `dec.Signals[i].EventID = msg.CausationID` |
| EMACrossoverEvaluatorActor | `dec.Signals[i].EventID = msg.CausationID` |
| BollingerSqueezeEvaluatorActor | `dec.Signals[i].EventID = msg.CausationID` |
| MeanReversionEntryResolverActor | `strat.Decisions[i].EventID = msg.CausationID` |
| SqueezeBreakoutEntryResolverActor | `strat.Decisions[i].EventID = msg.CausationID` |
| TrendFollowingEntryResolverActor | `strat.Decisions[i].EventID = msg.CausationID` |
| PositionExposureEvaluatorActor | `assessment.Strategies[i].EventID = msg.CausationID` |
| DrawdownLimitEvaluatorActor | `assessment.Strategies[i].EventID = msg.CausationID` |
| PaperOrderEvaluatorActor | `intent.Risk.EventID = msg.CausationID` |

### Tests

| File | Tests | Status |
|---|---|---|
| `internal/domain/lineage/lineage_test.go` | 9 tests | PASS |
| `internal/actors/scopes/derive/s470_lineage_causality_test.go` | 5 tests (per-stage + full chain) | PASS |

### Documentation

| File | Purpose |
|---|---|
| `docs/architecture/decision-lineage-and-causality-model.md` | Canonical lineage model, invariants, data flow |
| `docs/architecture/signal-strategy-decision-execution-lineage-semantics-ownership-and-limitations.md` | Stage semantics, ownership, limitations, trade-offs |

## Evidence

### Test Results

```
internal/domain/lineage      — 9/9 PASS
internal/actors/scopes/derive — all PASS (including 5 S470-specific)
internal/domain/*            — all PASS (zero regressions)
internal/application/*       — all PASS (zero regressions)
```

### Key Test Cases

1. **TestS470_DecisionCarriesSignalEventID** — RSI oversold evaluator enriches SignalInput.EventID with the signal event ID
2. **TestS470_StrategyCarriesDecisionEventID** — Mean reversion resolver enriches DecisionInput.EventID with the decision event ID
3. **TestS470_RiskCarriesStrategyEventID** — Position exposure evaluator enriches StrategyInput.EventID with the strategy event ID
4. **TestS470_ExecutionCarriesRiskEventID** — Paper order evaluator enriches RiskInput.EventID with the risk event ID
5. **TestS470_FullChainLineagePreservation** — Full 5-stage chain validates all EventIDs form a proper causal chain

## Acceptance Criteria Assessment

| Criterion | Status | Evidence |
|---|---|---|
| Causal chain is more explicit and auditable | Met | EventID in all Input types makes domain-level causation visible |
| Ambiguity between pipeline domains reduced | Met | Formal lineage package defines stages, invariants, and validation |
| Traceability quality improves materially | Met | Chain reconstruction no longer requires event metadata traversal for Input references |
| Base ready for S471 review surface | Met | lineage.ValidateChain() and Input EventIDs provide the data needed for review surfaces |

## Guard Rails Compliance

| Guard Rail | Status |
|---|---|
| No broad pipeline redesign | Compliant — only added fields and a thin validation package |
| No generic knowledge graph | Compliant — lineage model is minimal and pipeline-specific |
| No masking of real gaps | Compliant — L1-L6 limitations explicitly documented |
| No new execution/OMS capabilities | Compliant — no behavioral changes to execution path |

## Known Limitations (documented in detail)

- **L1**: Evidence-to-Signal causation gap remains (by design — signals derive from in-memory state)
- **L2**: Single-parent causation assumption (no cross-symbol chains)
- **L3**: No retroactive chain validation in ClickHouse
- **L4**: KV latest does not index by EventID
- **L5**: Historical data lacks Input EventIDs (best-effort after deployment)
- **L6**: Rejection/fill events are lifecycle extensions, not new causal stages

## What Changed vs What Didn't

### Changed
- 4 domain Input types gained EventID field
- 9 actor handlers now enrich Input types with parent event references
- New lineage package with formal chain model and validation
- 14 new test cases across 2 test files

### Did NOT Change
- Event metadata model (CorrelationID/CausationID unchanged)
- NATS stream topology
- ClickHouse schema (EventIDs flow via existing JSON columns)
- Actor message types
- Application evaluator/resolver signatures
- Execution behavior or lifecycle semantics
