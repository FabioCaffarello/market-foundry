# Stage S469 -- Strategy-to-Execution Decision Quality Wave Charter Report

**Stage**: S469
**Type**: Charter and Scope Freeze
**Status**: COMPLETE
**Date**: 2026-03-25
**Wave**: Strategy-to-Execution Decision Quality (S469--S473)
**Predecessor**: S468 (Session Access & Verification Evidence Gate)

---

## 1. Executive Summary

S469 opens the Strategy-to-Execution Decision Quality wave -- a short, disciplined wave that strengthens the traceability, auditability, and internal consistency of the signal-to-execution decision chain.

The Foundry runtime, venue path, session surfaces, and operational explainability reached maturity through waves S382--S468. The systemic gap that remains is **decision quality**: the ability to answer "why was this order placed?" with the same rigor that the system already answers "what happened?"

This stage freezes the wave scope to 4 blocks (S470--S473), defines 5 governing questions, registers 20 non-goals, and orders the execution sequence. No new API keys, live sessions, or structural refactoring is required.

---

## 2. Pre-Wave State Assessment

### 2.1 What Exists

The pipeline `Signal -> Decision -> Strategy -> Risk -> ExecutionIntent -> Venue` is fully operational with:

- **Event correlation chain**: `CorrelationID` and `CausationID` in `shared/events/Metadata` propagated through all domain events.
- **Risk context preservation**: `RiskInput` in `ExecutionIntent` carries `StrategyType` and `DecisionSeverity` since S265.
- **Domain linkage**: `Strategy.Decisions[]` with `DecisionInput` (outcome, severity, rationale); `Decision.Signals[]` with `SignalInput`.
- **Session audit bundle**: lifecycle, fees, consistency checks, check index, batch audit (S462, S467).
- **Source path explainability**: `SourcePathExplanation` with activation, gate status, config, intent, result (S361).
- **Rejection auditability**: `VenueOrderRejectedEvent` with code/reason, persisted in KV (S386).

### 2.2 What is Missing

| Gap | Impact |
|-----|--------|
| No lineage query endpoint | Operator must manually reconstruct fill -> signal chain across KV buckets |
| No decision context in audit bundle | Bundle answers "what happened" but not "why" |
| No cross-domain consistency checks | Severity/type consistency between decision and execution assumed but unverified |
| PriceSource not recorded in ExecutionIntent | Fill price origin opaque; price realism not auditable |

### 2.3 Consolidated Capability Baseline

| Capability | State |
|-----------|-------|
| Runtime execution (spot + futures, testnet + mainnet) | MATURE |
| Venue adapter pipeline (submit, retry, reconcile) | MATURE |
| Session metadata and lifecycle tracking | MATURE |
| Session audit bundle (HTTP accessible) | MATURE |
| Source path explainability | MATURE |
| Kill switch and operational controls | MATURE |
| Event correlation/causation metadata | EXISTS but not queryable end-to-end |
| Decision lineage traceability | MISSING |
| Decision context in audit surface | MISSING |
| Cross-domain consistency validation | MISSING |

---

## 3. Wave Structure

### 3.1 Blocks

| Block | Stage | Scope | Dependencies |
|-------|-------|-------|-------------|
| 1 | S470 | Decision lineage and causality model | None (reads existing KV) |
| 2 | S471 | Decision review surface and evidence bundle | S470 (lineage model) |
| 3 | S472 | Cross-domain consistency checks | S471 (verification framework integration) |
| 4 | S473 | Evidence gate | S470, S471, S472 |

### 3.2 Execution Order

```
S469 (charter) ── frozen
  │
  └─► S470 (lineage model)
        │
        └─► S471 (review surface + bundle)
              │
              └─► S472 (consistency checks)
                    │
                    └─► S473 (evidence gate)
```

Strictly sequential. Each block builds on the prior.

---

## 4. Governing Questions

| ID | Question | Target Block |
|----|----------|-------------|
| Q-DQ1 | Can an operator trace any execution intent back to its originating signal chain via a single query? | S470 + S471 |
| Q-DQ2 | Does the session audit bundle explain *why* orders were placed, not just *what* happened? | S471 |
| Q-DQ3 | Is the decision severity that reaches execution provably consistent with the severity recorded at decision evaluation time? | S472 |
| Q-DQ4 | Is the price used for fill simulation traceable to a specific PriceSource read? | S472 |
| Q-DQ5 | Do cross-domain consistency checks integrate with the existing session verification framework? | S472 |

---

## 5. Non-Goals (Frozen)

20 non-goals registered across 5 categories. Full list in [decision-quality-capabilities-questions-and-non-goals.md](../architecture/decision-quality-capabilities-questions-and-non-goals.md).

**Key exclusions**:

| Category | Excluded Items |
|----------|---------------|
| OMS expansion | Limit orders, cancel path, modify path, multi-order strategies |
| Multi-exchange | New exchanges, cross-exchange arbitrage, credential multiplexing |
| Infrastructure | New persistence backends, alerting system, Grafana dashboards, ML scoring |
| Structural | Domain event hierarchy refactoring, derive binary changes, NATS topology changes, actor refactoring |
| Enforcement | Fail-closed consistency blocking, automated policy engine, decision replay |

---

## 6. Artifacts Produced

| Artifact | Type | Path |
|----------|------|------|
| Wave charter and scope freeze | Architecture | `docs/architecture/strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, and non-goals | Architecture | `docs/architecture/decision-quality-capabilities-questions-and-non-goals.md` |
| Stage report (this document) | Stage report | `docs/stages/stage-s469-decision-quality-charter-report.md` |

---

## 7. Guard Rails Compliance

| Guard Rail | Status |
|-----------|--------|
| No OMS expansion opened | COMPLIANT -- NG-DQ1 through NG-DQ4 frozen |
| No multi-exchange opened | COMPLIANT -- NG-DQ5 through NG-DQ8 frozen |
| No structural redesign | COMPLIANT -- NG-DQ14 through NG-DQ17 frozen; additive-only constraint |
| No observability platform inflation | COMPLIANT -- NG-DQ10, NG-DQ11, NG-DQ13 frozen |

---

## 8. S470 Preparation

The next stage (S470: Decision Lineage and Causality Model) should:

1. **Read** the existing domain types:
   - `internal/domain/execution/execution.go` -- ExecutionIntent, RiskInput
   - `internal/domain/strategy/strategy.go` -- Strategy, DecisionInput
   - `internal/domain/decision/decision.go` -- Decision, SignalInput
   - `internal/domain/signal/signal.go` -- Signal
   - `internal/shared/events/event.go` -- Metadata (CorrelationID, CausationID)

2. **Read** the existing KV bucket access patterns:
   - `internal/adapters/nats/natsexecution/kv_store.go` -- execution KV operations
   - `internal/adapters/nats/natsexecution/registry.go` -- KV bucket registry

3. **Define** the `DecisionLineage` type in `internal/domain/execution/` as a read-only projection containing:
   - The ExecutionIntent (root of the query)
   - The referenced RiskAssessment (if available via KV)
   - The referenced Strategy (if available via KV)
   - The referenced Decision(s) (if available via KV)
   - The referenced Signal(s) (if available via KV)
   - Gap markers for any missing chain links

4. **Implement** a lineage assembler (application layer) that:
   - Takes a CorrelationID or ExecutionIntent
   - Reads from existing KV buckets
   - Returns a `DecisionLineage` with graceful degradation for missing data

5. **Test** with unit tests covering:
   - Complete chain assembly
   - Partial chain (missing decision data)
   - Flat/no-op intent (side=none)
   - Rejection chain
   - Empty/invalid correlation ID

**Estimated scope**: 1 new domain type, 1 assembler, 6-8 tests.

---

## 9. References

- [Wave Charter](../architecture/strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](../architecture/decision-quality-capabilities-questions-and-non-goals.md)
- [S468 Evidence Gate Report](stage-s468-session-access-verification-evidence-gate-report.md)
- [S463 Session Intelligence Evidence Gate](stage-s463-session-intelligence-evidence-gate-report.md)
- [S382 OMS Foundation Charter](stage-s382-oms-foundation-charter-report.md)
