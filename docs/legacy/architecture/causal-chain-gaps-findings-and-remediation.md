# Causal Chain Gaps: Findings and Remediation

**Stage:** S295
**Date:** 2026-03-21
**Status:** Gaps classified and remediation scoped

## Summary

The causal chain (CorrelationID + CausationID) across the 3 approved slices is **intact at the actor layer**. The DAG linkage is correct and deterministically validated. However, three gaps were identified that affect analytical reconstructability and test coverage uniformity.

## Gap Classification

### G1: ClickHouse Readers Do Not Reconstruct Causal IDs (signal/decision/strategy/risk)

**Severity:** Moderate
**Category:** Read-path analytical gap
**Affected layers:** Signal, Decision, Strategy, Risk ClickHouse readers

**Finding:**
- The write path correctly stores `correlation_id` and `causation_id` in all 5 ClickHouse tables.
- The execution reader correctly reads `exec_correlation_id` and `exec_causation_id` back into `ExecutionIntent`.
- The signal, decision, strategy, and risk readers **skip these columns** in their SELECT queries.
- Domain structs `Signal`, `Decision`, `Strategy`, `RiskAssessment` have **no fields** for CorrelationID/CausationID.

**Impact:**
- Causal chain cannot be reconstructed from ClickHouse for the first 4 layers using existing reader API.
- Analytical queries that need to join events by causation must use raw SQL against ClickHouse directly.
- Execution layer is fully reconstructable (ExecutionIntent carries both IDs).

**Root cause:**
Design decision to keep domain structs pure (only ExecutionIntent, as the output artifact, carries trace IDs). Event-level trace metadata is owned by `events.Metadata`, not by domain structs.

**Remediation options:**
1. **(Recommended for S296)** Add `CorrelationID`/`CausationID` fields to the 4 reader return types via analytical projection structs (not modifying domain structs). Create `SignalWithTrace`, `DecisionWithTrace`, etc. wrappers for analytical reads.
2. **(Alternative)** Add `CorrelationID`/`CausationID` directly to domain structs. Simpler but invasive — touches all code that constructs these types.
3. **(Deferred)** Keep current design, document that causal chain reconstruction requires raw ClickHouse SQL.

**S295 action:** Documented. No code change — this is a read-side gap that does not affect write-path integrity or actor-layer propagation. Remediation deferred to S296.

### G2: Signal Events Have Empty CausationID (by design)

**Severity:** Low (by design)
**Category:** Architectural boundary

**Finding:**
- All 3 signal sampler actors (RSI, EMA crossover, Bollinger) create events with:
  ```go
  meta := events.NewMetadata().WithCorrelationID(msg.CorrelationID)
  ```
- CausationID is not set on the published `SignalGeneratedEvent`.
- The fan-out `signalGeneratedMessage` correctly sets `CausationID: meta.ID`.

**Impact:**
- Signal events in NATS/ClickHouse have `causation_id=""`.
- The causal chain starts at the signal layer with no backward link to evidence.
- Downstream propagation is correct: decision events correctly receive `CausationID = signal.Metadata.ID` via the fan-out message.

**Root cause:**
Signals originate from `candleFinalizedMessage`, which is an internal actor message (not a published domain event). There is no published evidence event ID to use as CausationID.

**Remediation:**
None required. This is a deliberate architectural boundary: signals are the root of the causal chain. If evidence-layer tracing is needed in the future, evidence events would need to be published with Metadata.IDs, and signal samplers would reference them. This is out of scope for the current wave.

**S295 action:** Documented as architectural boundary. No code change.

### G3: Pre-S295 Test Coverage Was Asymmetric

**Severity:** Low (now resolved)
**Category:** Test coverage gap

**Finding:**
- Squeeze slice (S291-4): validated CorrelationID at all 5 stages and CausationID non-emptiness at decision and execution.
- Mean Reversion (S268): validated CorrelationID only at execution level.
- Trend Following (S268): validated CorrelationID only at execution level.
- **No existing test validated CausationID DAG linkage** (i.e., that decision.CausationID == signal.Metadata.ID).

**Impact:**
- Prior to S295, the DAG linkage was correct by code inspection but not proven by deterministic test.
- A regression in any actor's CausationID assignment would not have been caught for the mean reversion or trend following slices.

**Remediation:**
S295 added `causal_chain_validation_test.go` with 3 tests that validate:
- CorrelationID immutability at all stages for all 3 slices.
- CausationID DAG linkage (exact parent ID matching) for all 3 slices.
- Metadata.ID uniqueness (DAG acyclicity) for all 3 slices.
- Fan-out message CausationID correctness for the squeeze slice (full 5-stage).

**S295 action:** Resolved. Test file added.

## Gap Classification Matrix

| Gap | Severity | Category | Chain Integrity | Write Path | Read Path | Test Coverage | S295 Action |
|-----|----------|----------|-----------------|------------|-----------|---------------|-------------|
| G1 | Moderate | Read-path | Intact | Intact | Broken | — | Documented, defer to S296 |
| G2 | Low | By design | Intact (root) | Intact | N/A | — | Documented |
| G3 | Low | Test coverage | Intact | — | — | Now complete | Resolved |

## What Is NOT a Gap

For completeness, the following were investigated and confirmed correct:

1. **CorrelationID propagation** — immutable through all 5 layers, all 3 slices.
2. **CausationID DAG at actor layer** — correct parent linking at every hop.
3. **Dual risk fan-out** — both risk evaluators receive identical CorrelationID/CausationID from strategy fan-out.
4. **ExecutionIntent dual tracking** — both event Metadata and intent fields carry identical trace IDs.
5. **NATS envelope propagation** — all publishers pass both IDs to `EncodeEvent`.
6. **ClickHouse write path** — all 5 tables store `correlation_id` and `causation_id` from event Metadata.
7. **Behavioral roundtrip test** — validates correlation chain preservation through ClickHouse row mappers.
