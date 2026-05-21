# Strategy Effectiveness Measurement Wave -- Charter and Scope Freeze

**Wave**: Strategy Effectiveness Measurement
**Charter stage**: S474
**Date**: 2026-03-25
**Predecessor wave**: Strategy-to-Execution Decision Quality (S469--S473, PASS)

---

## 1. Problem Statement

The Decision Quality wave (S469--S473) answered **"why was this order placed and was the decision chain internally consistent?"** The system can now trace any fill back to its originating signals, through decision, strategy, risk, and execution -- with 9 cross-domain consistency checks and structured review surfaces.

The natural next question is: **"was the decision good?"**

Today the Foundry has no concept of decision effectiveness. A triggered decision that led to a filled order is indistinguishable from one that led to a loss. There is no win/loss classification, no outcome attribution to the decisional context, no measurement of whether high-severity signals produce better results than low-severity ones, and no batch evaluation surface to compare decision cohorts.

This wave closes that gap with a small, bounded set of deliverables that define and measure effectiveness using the lineage, review, and consistency infrastructure already in place.

---

## 2. Strategic Context

### 2.1 What the system already has

| Capability | Source | Status |
|-----------|--------|--------|
| Full-chain causal lineage (signal -> fill) | S470 | FULL |
| Decision review bundle (5 sections) | S471 | FULL |
| Cross-domain consistency checks (9 checks) | S472 | FULL |
| Execution lifecycle with fills and fees | S428, S411-S413 | FULL |
| Session audit bundle with verification | S462 | FULL |
| Batch decision review endpoint | S471 | IMPLEMENTED |
| CorrelationID / CausationID traceability | Pre-wave | FULL |
| FillRecord with Price, Quantity, Fee, CostBasis | S428 | FULL |

### 2.2 What the system does NOT have

| Missing capability | Impact |
|-------------------|--------|
| Win/loss classification per decision | Cannot distinguish profitable from unprofitable decisions |
| Outcome attribution to decisional context | Cannot answer "did high-severity signals outperform?" |
| Effectiveness score model | No quantitative quality metric for decisions |
| Batch effectiveness evaluation | Cannot compare cohorts of decisions |
| Comparative analysis surface | Cannot identify which decision types, strategies, or signal families produce better outcomes |

### 2.3 Why now

1. The lineage infrastructure is fresh and well-tested (39 new tests, zero regressions).
2. The review bundle already assembles the full decision chain -- effectiveness extends it rather than replacing it.
3. No new API keys, no new live sessions, no new exchange connectivity required.
4. Effectiveness measurement is the highest-value depth extension before any breadth expansion.

---

## 3. Wave Objective

Define canonical semantics for strategy effectiveness, implement outcome attribution from execution results back to decisional context, and provide read surfaces for batch evaluation and comparative analysis -- all within the boundaries of what is already observable in the pipeline.

---

## 4. Wave Blocks (Ordered)

| Block | Stage | Scope | Depends On |
|-------|-------|-------|------------|
| 1. Canonical effectiveness model and attribution semantics | S475 | Define `EffectivenessOutcome`, win/loss/breakeven classification, P&L attribution per decision, outcome-to-context linking rules, domain types and invariants | -- |
| 2. Measurement read surfaces and batch evaluation | S476 | Implement effectiveness computation from existing fill/fee data, batch evaluation endpoint, effectiveness fields in review bundle, ClickHouse read path | S475 |
| 3. Decision effectiveness review and comparative analysis | S477 | Comparative analysis by decision type, strategy type, severity, timeframe; cohort comparison endpoint; effectiveness summary in review explanation | S476 |
| 4. Evidence gate | S478 | Formal assessment against governing questions, evidence matrix, residual gaps, wave verdict | S477 |

**Estimated stages**: 4 (S475--S478)
**Estimated new tests**: 25--40

---

## 5. Governing Questions

These questions define what PASS means for this wave:

| ID | Question | Answered By |
|----|----------|-------------|
| Q-SE1 | Can the system classify each completed decision chain as win, loss, or breakeven with canonical semantics? | S475 |
| Q-SE2 | Can the system attribute realized P&L (price delta, fee impact) to the originating decision and its causal inputs? | S475, S476 |
| Q-SE3 | Is effectiveness computable from existing fill and fee data without new exchange connectivity? | S476 |
| Q-SE4 | Can the system batch-evaluate effectiveness across a cohort of decisions (by type, timeframe, source, severity)? | S476 |
| Q-SE5 | Can the system surface comparative effectiveness analysis (which decision types or strategies outperform?) | S477 |

---

## 6. Scope Freeze Rules

### 6.1 IN scope

- Win/loss/breakeven outcome classification for completed (filled or cancelled) execution intents.
- P&L attribution per decision chain: entry price, exit price (if applicable within session), fees, cost basis.
- Effectiveness score model: bounded, deterministic, derived from observable fills.
- Batch evaluation endpoint analogous to existing `/decision/reviews`.
- Comparative analysis by: decision type, strategy type, signal severity, timeframe, source.
- Extension of `DecisionReviewBundle` with effectiveness section.
- Domain types in `internal/domain/effectiveness/` or extension of existing execution domain.
- Read-path queries in ClickHouse analytical reader.
- Tests covering effectiveness computation, classification edge cases, and attribution correctness.

### 6.2 OUT of scope (frozen)

Everything in the non-goals document is frozen. Key exclusions:

- No real-time P&L tracking or streaming effectiveness updates.
- No portfolio-level analytics (cross-symbol, cross-session aggregation beyond single decision chains).
- No Sharpe ratio, Sortino, or risk-adjusted return metrics.
- No predictive models or ML-based signal scoring.
- No new ClickHouse tables (effectiveness computed from existing fill data).
- No UI, dashboards, or visualization layers.
- No OMS expansion.
- No multi-exchange or multi-venue work.
- No strategy family expansion.
- No alerting or notification on effectiveness thresholds.

### 6.3 Scope change protocol

Any change to scope requires explicit re-freeze with documented rationale. The wave owner must evaluate whether the change preserves the 4-stage budget and does not violate non-goals.

---

## 7. Guard Rails

1. **No new exchange connectivity.** Effectiveness is computed from data already in the pipeline.
2. **No new ClickHouse tables.** Effectiveness is derived from existing execution, fill, and fee data via queries.
3. **No portfolio analytics.** Scope is per-decision-chain, not cross-symbol or cross-session.
4. **No risk-adjusted metrics.** Win/loss and raw P&L attribution only. Sharpe/Sortino are a separate wave.
5. **No real-time streaming.** Effectiveness is computed on read, not on write path.
6. **No redesign of existing domains.** Effectiveness extends the review surface; it does not refactor decision, strategy, risk, or execution types.
7. **No UI or dashboard work.** HTTP endpoints only, consistent with existing patterns.
8. **No ML or predictive scoring.** Classification is deterministic and rule-based.
9. **Additive only.** Zero changes to existing behavior. All new types, no modified types.
10. **Test budget.** Each implementation stage must deliver tests. Charter estimates 25--40 total.

---

## 8. Dependencies

### 8.1 Hard dependencies (must be present)

| Dependency | Status |
|-----------|--------|
| `lineage` package with `ValidateChain()` | S470, IMPLEMENTED |
| `DecisionReviewBundle` with 5 sections | S471, IMPLEMENTED |
| `consistency` package with 9 checks | S472, IMPLEMENTED |
| `FillRecord` with Price, Quantity, Fee, CostBasis | S428, IMPLEMENTED |
| Batch decision review endpoint | S471, IMPLEMENTED |
| CorrelationID / CausationID on all events | Pre-wave, IMPLEMENTED |

### 8.2 Soft dependencies (nice to have but not blocking)

| Dependency | Status | Impact if missing |
|-----------|--------|-------------------|
| PriceSource tagging per intent (RG-2 from S473) | NOT IMPLEMENTED | Attribution uses fill price only, not market price delta |
| Consistency checks in verification registry (RG-3) | NOT IMPLEMENTED | No impact on effectiveness |
| Not-triggered decision batch query (RG-4) | NOT IMPLEMENTED | Effectiveness only applies to triggered decisions anyway |

---

## 9. Success Criteria

The wave PASSES if:

1. All 5 governing questions are answered YES or SUBSTANTIAL.
2. Zero regressions across existing test suites.
3. At least 25 new tests covering effectiveness semantics, attribution correctness, and read surface behavior.
4. All guard rails observed (no scope creep into portfolio analytics, risk-adjusted metrics, or exchange connectivity).
5. Effectiveness is computable from existing data without new infrastructure.

---

## 10. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Effectiveness semantics are ambiguous for partial fills | MEDIUM | LOW | Define explicit rules for partial-fill classification in S475 |
| Single-session scope limits attribution for multi-session strategies | MEDIUM | LOW | Document as known limitation; cross-session is a future wave |
| Fill data may lack exit price for open positions | HIGH | MEDIUM | Classify only completed round-trips or use session-end mark; document explicitly |
| Scope inflation toward portfolio analytics | LOW | HIGH | Guard rails + non-goals freeze |
| Read-path performance on large cohorts | LOW | MEDIUM | Leverage existing ClickHouse query patterns |

---

## 11. Timeline and Budget

| Stage | Estimated effort | Type |
|-------|-----------------|------|
| S475 | 1 session | Domain modeling + attribution rules |
| S476 | 1 session | Read surfaces + batch evaluation |
| S477 | 1 session | Comparative analysis + review extension |
| S478 | 1 session | Evidence gate |

**Total**: 4 stages, 4 sessions. No multi-session stages expected.

---

## 12. References

- [S473 Evidence Gate Report](../stages/stage-s473-decision-quality-evidence-gate-report.md)
- [Decision Quality Wave Charter](strategy-to-execution-decision-quality-wave-charter-and-scope-freeze.md)
- [Decision Quality Evidence Matrix](decision-quality-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Capabilities, Questions, and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
- [S474 Charter Report](../stages/stage-s474-strategy-effectiveness-charter-report.md)
