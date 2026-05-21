# Stage S359 — Source Selection and Canonical Contract Report

> **Wave**: Strategy/Signal Integration (S358–S363)
> **Block**: SSI-1
> **Predecessor**: S358 (Charter and Scope Freeze)
> **Date**: 2026-03-22

---

## 1. Executive Summary

S359 completes the source selection and canonical integration contract for the Strategy/Signal Integration Wave. After inventorying all available signal and strategy families, comparing architectural fit, complexity, and auditability, the stage selects **RSI signal + Mean Reversion Entry strategy** as the canonical pair. The field-level contract mapping `StrategyResolvedEvent → ExecutionIntent` is fully defined, with 11 binding invariants, explicit boundary rules, and enumerated test assertions.

---

## 2. Source Selection

### 2.1 Inventory

**Signal families evaluated (6)**: RSI, EMA Crossover, Bollinger Bands, MACD, VWAP, ATR.
**Strategy families evaluated (3)**: Mean Reversion Entry, Trend Following Entry, Squeeze Breakout Entry.

All families are fully implemented in the derive scope with NATS event streams, KV projections, and query endpoints. None had architectural blockers.

### 2.2 Decision

**Selected**: RSI signal + Mean Reversion Entry strategy.

**Primary rationale**:
- RSI provides the simplest signal (single decimal value, clear threshold semantics)
- Mean Reversion Entry has the most direct direction mapping (oversold → long, overbought → short)
- The pair exercises every domain boundary without requiring multi-value interpretation
- Highest auditability score among candidates — critical for SSI-3 explainability goals

**Rejected alternatives**: EMA Crossover + Trend Following (medium complexity, deferred), Bollinger + Squeeze Breakout (high complexity, deferred). Both are viable for future waves using the same wiring pattern.

### 2.3 Reference

Full comparative analysis: [source-selection-and-canonical-integration-contract.md](../architecture/source-selection-and-canonical-integration-contract.md), Section 3.

---

## 3. Contract Definition

### 3.1 Contract Boundary

The integration boundary is the NATS `STRATEGY_EVENTS` stream. The execute scope subscribes to `strategy.events.mean_reversion_entry.resolved.>` via a new `StrategyConsumerActor`. The consumer transforms strategy events into `ExecutionIntent` via `PaperOrderEvaluator`.

### 3.2 Key Contract Properties

| Property | Value |
|----------|-------|
| Source event | `StrategyResolvedEvent` |
| Source NATS subject | `strategy.events.mean_reversion_entry.resolved.>` |
| Target type | `ExecutionIntent` (type: `paper_order`) |
| Transformation | `StrategyConsumerActor` → `PaperOrderEvaluator.Evaluate()` |
| Risk handling | Pass-through (`riskType = "pass_through"`, `riskDisposition = "approved"`) |
| Position sizing | Configurable `max_position_pct` (default: `"0.01"`) |
| Correlation | `CorrelationID` from event metadata, `CausationID` from event `Metadata.ID` |

### 3.3 Direction-to-Side Mapping

| Strategy Direction | Execution Side | Quantity |
|--------------------|----------------|----------|
| `long` | `buy` | `max_position_pct` |
| `short` | `sell` | `max_position_pct` |
| `flat` | `none` | `"0"` |

### 3.4 Reference

Full field-level mapping: [source-selection-and-canonical-integration-contract.md](../architecture/source-selection-and-canonical-integration-contract.md), Section 4.3.

---

## 4. Invariants and Boundaries

### 4.1 Invariants Defined (11)

| ID | Invariant |
|----|-----------|
| INV-1 | Strategy type identity preserved in `ExecutionIntent.Risk.StrategyType` |
| INV-2 | Direction-to-side mapping is deterministic (pure function) |
| INV-3 | Causation chain: `CausationID = event.Metadata.ID`, `CorrelationID = event.Metadata.CorrelationID` |
| INV-4 | Pass-through risk explicitly marked (`riskType = "pass_through"`) |
| INV-5 | Timestamp from strategy event, not `time.Now()` |
| INV-6 | Single strategy family per consumer instance |
| INV-7 | Flat direction produces `Side = "none"`, `Quantity = "0"` |
| INV-8 | NATS ack only after successful publication |
| INV-9 | Kill switch applies to strategy-sourced intents |
| INV-10 | Staleness guard applies to strategy-sourced intents |
| INV-11 | Deduplication keys are unique per event |

### 4.2 Domain Boundaries

Five domain layers maintain strict isolation:
- **Signal**: evidence interpretation only
- **Decision**: condition evaluation only
- **Strategy**: directional intent only
- **Control**: halt/resume only
- **Execution**: side determination, sizing, venue submission only

No layer imports types from downstream layers. Each layer owns its input structs (e.g., `DecisionInput` is strategy-owned, not decision-owned).

### 4.3 Reference

Full boundary rules and responsibility matrix: [source-to-execution-contract-boundaries-invariants-and-limits.md](../architecture/source-to-execution-contract-boundaries-invariants-and-limits.md).

---

## 5. Explicit Limits

### 5.1 What S359 Guarantees

- Canonical source pair selected with justified rationale
- Field-level contract fully specified
- Every `PaperOrderEvaluator` parameter has a documented source
- All invariants are testable
- Error handling for every failure mode is specified
- Configuration surface is minimal and documented

### 5.2 What S359 Does NOT Deliver

- No code implementation (deferred to S360/SSI-2)
- No risk evaluation (NG-4 — pass-through only)
- No confidence thresholds (deferred to SSI-3)
- No per-strategy gates (deferred to SSI-3)
- No Prometheus metrics (deferred to SSI-3)
- No integration tests (deferred to SSI-4)

### 5.3 Known Simplifications

| Simplification | Resolution |
|----------------|------------|
| Risk pass-through | Future wave adds risk query |
| Fixed position size | Risk integration provides dynamic sizing |
| No confidence threshold | SSI-3 adds configurable minimum |
| No per-strategy gate | SSI-3 adds strategy-type KV gate |
| Single decision severity (`Decisions[0]`) | Mean reversion uses exactly one decision; multi-decision strategies need aggregation |

---

## 6. Governing Questions Addressed

| Question | Status | Evidence |
|----------|--------|----------|
| SSIQ-1: Is one signal+strategy pair selected with justified contract? | **Answered** | RSI + mean_reversion_entry selected. Comparative analysis in Section 3 of source-selection document. |
| SSIQ-2: Is there a formal field-level contract mapping strategy → ExecutionIntent? | **Answered** | Complete field mapping in Section 4.3 of source-selection document. Direction-to-side mapping, risk pass-through defaults, correlation propagation all specified. |

---

## 7. Wave Constraint Compliance

| Constraint | Status |
|------------|--------|
| FC-1: No new blocks after S358 | Compliant |
| FC-2: Block scope frozen | Compliant |
| FC-4: Only selected pair | Compliant — RSI + mean_reversion_entry |
| FC-5: Paper adapter only | Compliant |
| FC-7: Domain types extend, not restructure | Compliant — no domain changes |
| NG-1: No multiple signals | Compliant |
| NG-2: No multiple strategies | Compliant |
| NG-4: No risk domain changes | Compliant |
| NG-11: No new domain types | Compliant |

---

## 8. Preparation for S360

S360 (SSI-2: Controlled Source-to-Execution Wiring) can proceed with:

1. **Consumer spec ready**: `execute-strategy-mean-reversion-entry` subscribing to `strategy.events.mean_reversion_entry.resolved.>`
2. **Field mapping ready**: every `PaperOrderEvaluator.Evaluate()` parameter has a documented source and default
3. **Error handling ready**: deserialization, validation, wrong type, evaluation, and publication failures all specified
4. **Unit tests ready**: 10 test assertions enumerated in boundaries document Section 8.1
5. **Integration tests ready**: 4 integration test assertions enumerated in boundaries document Section 8.2
6. **Invariants ready**: 11 invariants serve as acceptance criteria for implementation

**Implementation scope for S360**:
- `internal/actors/scopes/execute/strategy_consumer_actor.go` — new actor
- `cmd/execute/run.go` — wire actor into topology
- `internal/adapters/nats/natsexecution/registry.go` — add consumer spec (or reference natsstrategy)
- Unit tests validating all 10 assertions
- Health tracker integration

---

## 9. Deliverables

| # | Deliverable | Path | Status |
|---|-------------|------|--------|
| 1 | Source selection and canonical integration contract | `docs/architecture/source-selection-and-canonical-integration-contract.md` | Delivered |
| 2 | Boundaries, invariants, and limits | `docs/architecture/source-to-execution-contract-boundaries-invariants-and-limits.md` | Delivered |
| 3 | Stage report (this document) | `docs/stages/stage-s359-source-selection-and-canonical-contract-report.md` | Delivered |

---

## 10. Verdict

**S359 is COMPLETE.** The canonical source pair is selected, the integration contract is fully specified at field level, invariants are defined and testable, and the implementation scope for S360 is clear. The wave proceeds to SSI-2 wiring.
