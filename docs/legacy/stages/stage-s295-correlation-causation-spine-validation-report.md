# Stage S295: Correlation/Causation Spine Validation Report

**Date:** 2026-03-21
**Status:** Complete
**Predecessor:** S294 (Composite Execution Observability Charter)

## Objective

Validate and document the integrity of the CorrelationID/CausationID causal chain across the 3 approved vertical slices, identify gaps, and close what is compatible with the current wave scope.

## Executive Summary

The causal chain is **intact at the actor layer** across all 3 slices. The CausationID DAG linkage is correct: each event's CausationID points to its parent event's Metadata.ID, forming a non-cyclic directed graph from signal through execution. CorrelationID is immutable from injection to terminal output.

Three gaps were identified:
- **G1 (Moderate):** ClickHouse readers for signal/decision/strategy/risk do not reconstruct causal IDs. Data is stored but not readable through existing API. Deferred to S296.
- **G2 (Low, by design):** Signal events have empty CausationID — they are the root of the chain.
- **G3 (Low, resolved):** Pre-S295 tests did not validate DAG linkage. Now covered.

## Deliverables

| Deliverable | Path | Status |
|-------------|------|--------|
| Causal chain validation test | `internal/actors/scopes/derive/causal_chain_validation_test.go` | Added |
| Spine validation document | `docs/architecture/correlation-causation-spine-validation-across-existing-slices.md` | Added |
| Gaps and remediation document | `docs/architecture/causal-chain-gaps-findings-and-remediation.md` | Added |
| Stage report | `docs/stages/stage-s295-correlation-causation-spine-validation-report.md` | This file |

## Validation Methodology

1. **Deterministic injection** — each slice was exercised with a known CorrelationID and CausationID (or candle sequence for the squeeze slice).
2. **Stage-by-stage collection** — every published event and every fan-out message was captured by message collectors.
3. **DAG linkage assertion** — for each stage, `event.Metadata.CausationID == parent_event.Metadata.ID` was asserted.
4. **Immutability assertion** — CorrelationID was checked at every stage, including both event Metadata and ExecutionIntent fields.
5. **Uniqueness assertion** — all Metadata.IDs were verified unique across the chain (DAG acyclicity).

## Results by Slice

### Slice 1: Mean Reversion (RSI → mean_reversion_entry)
- **Chain:** decision → strategy → risk → execution (4 stages)
- **CorrelationID:** Immutable across all stages
- **CausationID DAG:** signal-root-001 → decision.ID → strategy.ID → risk.ID
- **Verdict:** INTACT

### Slice 2: Trend Following (EMA → trend_following_entry)
- **Chain:** decision → strategy → risk → execution (4 stages)
- **CorrelationID:** Immutable across all stages
- **CausationID DAG:** signal-root-002 → decision.ID → strategy.ID → risk.ID
- **Verdict:** INTACT

### Slice 3: Squeeze Breakout (Bollinger → squeeze_breakout_entry)
- **Chain:** signal → decision → strategy → risk → execution (5 stages)
- **CorrelationID:** Immutable across all 5 stages
- **CausationID DAG:** "" (root) → signal.ID → decision.ID → strategy.ID → risk.ID
- **Fan-out CausationIDs:** Validated at all 3 hops (signal→decision, decision→strategy, strategy→risk)
- **Verdict:** INTACT

## Gaps Found

| ID | Description | Severity | Action |
|----|-------------|----------|--------|
| G1 | ClickHouse readers skip correlation_id/causation_id for signal/decision/strategy/risk | Moderate | Deferred to S296 |
| G2 | Signal events have empty CausationID (root of chain) | Low | By design, documented |
| G3 | Pre-S295 tests lacked DAG linkage validation | Low | Resolved (test added) |

## Acceptance Criteria Checklist

- [x] Causal chain of 3 slices validated with deterministic data
- [x] Gaps identified and classified with rigor
- [x] Minimal corrections made without inflating scope (test file added, no write-side changes)
- [x] Stage prepares auditable base for composition in S296

## Guard Rails Compliance

- [x] Did not redesign the spine
- [x] Did not open non-goals from S294
- [x] Did not create a new tracing platform
- [x] Did not make broad write-side changes
- [x] Did not mask gaps with generic documentation

## Limitations

1. **ClickHouse analytical reconstruction** — causal chain can only be reconstructed from ClickHouse via raw SQL (joining on correlation_id/causation_id columns). The reader API does not expose these fields for signal/decision/strategy/risk. This is the primary gap to address in S296.
2. **Evidence-to-signal link** — the causal chain starts at signal. There is no CausationID link from signal back to evidence (candle) events, because evidence events are not published as domain events with Metadata.IDs in the current architecture.
3. **NATS replay reconstruction** — causal chain reconstruction from NATS replay was not tested (requires live NATS infrastructure). The envelope correctly carries both IDs per code inspection and publisher tests.

## Preparation for S296

The validated causal chain provides a solid base for S296 (Composite Execution Observability). Recommended focus:

1. **Analytical projection types** — create `SignalWithTrace`, `DecisionWithTrace`, etc. wrappers that include CorrelationID/CausationID for ClickHouse reads.
2. **Causal chain query** — build a query that reconstructs the full chain from ClickHouse by joining across tables on causation_id.
3. **Observability dashboard foundation** — the CorrelationID can serve as the grouping key for a full execution trace view.

## Test Evidence

```
=== RUN   TestCausalChain_MeanReversion_DAGLinkage
    [correlation] immutable across all stages
    [decision] CausationID=signal-root-001 (links to signal)
    [strategy] CausationID={decision.ID} (links to decision)
    [risk] CausationID={strategy.ID} (links to strategy)
    [execution] CausationID={risk.ID} (links to risk)
    [causal-chain/mean-reversion] PASS — full DAG linkage validated
--- PASS: TestCausalChain_MeanReversion_DAGLinkage (0.05s)

=== RUN   TestCausalChain_TrendFollowing_DAGLinkage
    [causal-chain/trend-following] PASS — full DAG linkage validated
--- PASS: TestCausalChain_TrendFollowing_DAGLinkage (0.05s)

=== RUN   TestCausalChain_SqueezeBreakout_DAGLinkage
    [signal] CorrelationID=s295-sq-causal CausationID="" (root, empty by design)
    [causal-chain/squeeze-breakout] PASS — full 5-stage DAG linkage validated
--- PASS: TestCausalChain_SqueezeBreakout_DAGLinkage (0.05s)
```
