# Decision Quality -- Capabilities, Questions, and Non-Goals

**Wave**: Strategy-to-Execution Decision Quality
**Charter**: [strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md](strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md)
**Date**: 2026-03-25

---

## 1. Capabilities Matrix

### 1.1 Pre-Wave State (inherited from S468)

| ID | Capability | State | Evidence |
|----|-----------|-------|----------|
| C-DQ1 | Event correlation chain (CorrelationID/CausationID) | EXISTS | Metadata struct in `shared/events/event.go`; propagated through all domain events |
| C-DQ2 | Risk context in ExecutionIntent (StrategyType, DecisionSeverity) | EXISTS | `RiskInput` struct embedded in `ExecutionIntent` since S265 |
| C-DQ3 | Decision-signal linkage in domain types | EXISTS | `Decision.Signals[]` with `SignalInput`; `Strategy.Decisions[]` with `DecisionInput` |
| C-DQ4 | Rejection auditability | EXISTS | `VenueOrderRejectedEvent` with RejectionCode/Reason; persisted in KV |
| C-DQ5 | Session audit bundle | EXISTS | `SessionAuditBundle` with lifecycle, fees, consistency checks (S462, S467) |
| C-DQ6 | Source path explainability | EXISTS | `SourcePathExplanation` with activation, gate status, config, intent, result (S361) |
| C-DQ7 | Full-chain lineage query | MISSING | No endpoint or assembler traces fill -> signal |
| C-DQ8 | Decision context in audit bundle | MISSING | Bundle has lifecycle + fees but not decision rationale |
| C-DQ9 | Cross-domain consistency validation | MISSING | Severity, type, correlation consistency assumed but not checked |
| C-DQ10 | PriceSource traceability | MISSING | DryRunSubmitter/PaperVenueAdapter use PriceSource but don't record which source |

### 1.2 Target State (post-wave)

| ID | Capability | Target | Block |
|----|-----------|--------|-------|
| C-DQ7 | Full-chain lineage query | FULL | S470 (lineage model) + S471 (HTTP surface) |
| C-DQ8 | Decision context in audit bundle | FULL | S471 (decision summary in bundle) |
| C-DQ9 | Cross-domain consistency validation | FULL | S472 (consistency checks) |
| C-DQ10 | PriceSource traceability | FULL | S472 (metadata recording) |

---

## 2. Governing Questions

These questions guide every implementation decision in the wave. Each stage must advance at least one question toward YES.

| ID | Question | Answered by |
|----|----------|-------------|
| Q-DQ1 | Can an operator trace any execution intent back to its originating signal chain via a single query? | S470 (model), S471 (HTTP endpoint) |
| Q-DQ2 | Does the session audit bundle explain *why* orders were placed, not just *what* happened? | S471 (DecisionSummary in bundle) |
| Q-DQ3 | Is the decision severity that reaches execution provably consistent with the severity recorded at decision evaluation time? | S472 (severity consistency check) |
| Q-DQ4 | Is the price used for fill simulation traceable to a specific PriceSource read? | S472 (PriceSource metadata recording) |
| Q-DQ5 | Do cross-domain consistency checks integrate with the existing session verification framework? | S472 (integration with S461 verification) |

---

## 3. Non-Goals

These items are explicitly excluded from this wave. Each non-goal includes the rationale for exclusion and the condition under which it could be reconsidered.

### 3.1 OMS Expansion

| ID | Non-Goal | Rationale | Reconsider When |
|----|----------|-----------|-----------------|
| NG-DQ1 | Limit order support | Requires new order type, venue adapter changes, partial-fill semantics expansion | Dedicated OMS expansion wave |
| NG-DQ2 | Cancel/modify path | Requires new lifecycle transitions, venue API integration, idempotency model | Dedicated OMS expansion wave |
| NG-DQ3 | Multi-order strategy execution | Requires order group model, coordination semantics | Strategy execution wave |
| NG-DQ4 | Order amendment lifecycle | Requires venue-specific amendment API integration | Dedicated OMS expansion wave |

### 3.2 Multi-Exchange and Breadth

| ID | Non-Goal | Rationale | Reconsider When |
|----|----------|-----------|-----------------|
| NG-DQ5 | Multi-exchange support | Requires adapter factory, credential multiplexing, segment expansion | Multi-exchange wave |
| NG-DQ6 | New instrument families (options, perpetuals beyond current) | Requires domain model expansion, venue-specific semantics | Instrument expansion wave |
| NG-DQ7 | New trading pairs beyond current configuration | Orthogonal to decision quality; config change only | Operator decision |
| NG-DQ8 | Cross-exchange arbitrage or correlation | Requires multi-exchange foundation first | Post multi-exchange |

### 3.3 Infrastructure and Platforms

| ID | Non-Goal | Rationale | Reconsider When |
|----|----------|-----------|-----------------|
| NG-DQ9 | New persistence backends (Postgres, TimescaleDB) | KV sufficient for lineage at current scale | Scale requires it |
| NG-DQ10 | Real-time alerting or notification system | Observational checks only; alerting is a separate concern | Operational alerting wave |
| NG-DQ11 | Grafana/dashboard integration | HTTP API is the output surface; dashboard wiring is separate | Dashboard wave |
| NG-DQ12 | ML-based decision scoring or backtesting | Requires historical data pipeline not in scope | Analytics wave |
| NG-DQ13 | Broad observability platform (distributed tracing, metrics aggregation) | Far exceeds decision quality scope | Observability wave |

### 3.4 Structural and Refactoring

| ID | Non-Goal | Rationale | Reconsider When |
|----|----------|-----------|-----------------|
| NG-DQ14 | Refactoring domain event hierarchy | Existing event types are stable and sufficient | Domain evolution wave |
| NG-DQ15 | Changing derive binary event emission | All work is in execute/gateway; derive is untouched | Derive enhancement wave |
| NG-DQ16 | Changing NATS subject topology | Lineage reads from existing KV buckets | NATS topology wave |
| NG-DQ17 | Refactoring actor hierarchy in execute binary | Actor structure is stable; lineage is a read concern | Actor refactoring wave |

### 3.5 Enforcement and Policy

| ID | Non-Goal | Rationale | Reconsider When |
|----|----------|-----------|-----------------|
| NG-DQ18 | Fail-closed consistency enforcement (blocking execution on check failure) | Checks are observational; blocking requires confidence in check correctness | Checks proven reliable over multiple sessions |
| NG-DQ19 | Automated decision policy engine | Requires policy model, rule engine; far exceeds scope | Policy wave |
| NG-DQ20 | Decision replay or simulation | Requires event sourcing or snapshot replay infrastructure | Simulation wave |

---

## 4. Assumptions

| ID | Assumption | Risk if Wrong |
|----|-----------|---------------|
| A-DQ1 | Existing KV buckets retain decision/strategy data for the duration of a session | Lineage assembler returns partial chains; graceful degradation handles this |
| A-DQ2 | CorrelationID is consistently propagated across all domain events today | If broken, S472 consistency checks will surface the gap |
| A-DQ3 | Session verification framework (S461) is extensible for new check types | If not, S472 may need a parallel check registration mechanism |
| A-DQ4 | PriceSource reads happen at a single point (DryRunSubmitter/PaperVenueAdapter) | If scattered, metadata recording in S472 needs multiple injection points |

---

## 5. Acceptance Criteria per Block

### S470: Decision Lineage and Causality Model

- [ ] `DecisionLineage` type defined in execution domain.
- [ ] Lineage assembler reads from DECISION_LATEST, STRATEGY_LATEST, EXECUTION_INTENT_LATEST KV buckets.
- [ ] Unit tests for: complete chain, partial chain (missing decision), flat/no-op chain, rejection chain.
- [ ] Graceful degradation: missing upstream data produces explicit gap markers, not errors.

### S471: Decision Review Surface and Evidence Bundle

- [ ] `GET /session/{id}/decisions` returns lineage entries for all intents in the session.
- [ ] `SessionAuditBundle.DecisionSummary` populated with outcome counts, severity distribution.
- [ ] Per-intent lifecycle entries include decision context (strategy type, severity, confidence).
- [ ] Tests for: query with data, query with empty session, bundle augmentation, graceful degradation.

### S472: Cross-Domain Consistency Checks

- [ ] Severity consistency check: RiskInput.DecisionSeverity matches source Decision.Severity.
- [ ] StrategyType consistency check: RiskInput.StrategyType matches source Strategy.Type.
- [ ] CorrelationID chain check: unbroken chain from intent to at least one signal.
- [ ] PriceSource recorded in ExecutionIntent.Metadata["price_source"].
- [ ] Checks registered in session verification framework.
- [ ] Tests for each check: pass, fail, degraded.

### S473: Evidence Gate

- [ ] All governing questions assessed.
- [ ] Capability grading for C-DQ7 through C-DQ10.
- [ ] Residual gap register.
- [ ] Verdict with recommendation.

---

## 6. References

- [Wave Charter](strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md)
- [Canonical Order Model](canonical-order-model-and-lifecycle-state-machine.md)
- [Session Intelligence Charter](../stages/stage-s459-session-intelligence-charter-report.md)
- [S468 Evidence Gate](../stages/stage-s468-session-access-verification-evidence-gate-report.md)
