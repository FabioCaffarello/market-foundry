# Round-Trip Pairing Wave -- Charter and Scope Freeze

**Wave**: Round-Trip Pairing
**Charter stage**: S479
**Date**: 2026-03-26
**Predecessor wave**: Strategy Effectiveness Measurement (S474--S478, PASS)

---

## 1. Problem Statement

The Strategy Effectiveness Measurement wave (S474--S478) answered **"was the decision good?"** with canonical win/loss/breakeven/unresolved classification, P&L attribution, batch evaluation, and comparative analysis -- 45 tests, 7 FULL capabilities, zero regressions.

However, the evidence gate exposed a structural bottleneck: **G-SE1 (MEDIUM) -- single-leg fills dominate outcomes.** Most evaluation chains return `unresolved` because the pipeline processes individual orders, not round-trip trade pairs. The entry fill has no paired exit within the same processing unit, so `Classify()` cannot determine win or loss.

The consequence is concrete: effectiveness metrics exist but operate on a small subsample of resolved chains. Win rate, average P&L, and cohort comparisons are statistically thin. The measurement infrastructure is sound; the input to that infrastructure is incomplete.

This wave addresses the problem directly: **transform outcomes that are today single-leg and unresolved into paired round-trips that can be classified and measured.**

The `ClassifyPair(entry, exit)` function already exists in domain code (S476). What is missing is:
1. A canonical model for what constitutes a round-trip and how legs are matched.
2. A read model that finds and pairs entry/exit legs from existing data.
3. Integration with the effectiveness attribution pipeline.
4. An HTTP surface that exposes paired outcomes for review and reconciliation.

---

## 2. Strategic Context

### 2.1 What the system already has

| Capability | Source | Status |
|-----------|--------|--------|
| `ClassifyPair(entry, exit)` domain function | S476 | IMPLEMENTED |
| Win/loss/breakeven/unresolved classification | S476 | FULL |
| P&L attribution per decision chain | S476 | FULL |
| Batch effectiveness evaluation (3 endpoints) | S476, S477 | FULL |
| Comparative analysis by 4 dimensions | S477 | FULL |
| Full-chain causal lineage (signal -> fill) | S470 | FULL |
| Decision review bundle (5 sections + effectiveness) | S471, S476 | FULL |
| Cross-domain consistency checks (9 checks) | S472 | FULL |
| Session audit bundle with verification | S462 | FULL |
| CorrelationID / CausationID traceability | Pre-wave | FULL |
| FillRecord with Price, Quantity, Fee, CostBasis | S428 | FULL |
| Execution lifecycle with fills and fees | S411--S413, S428 | FULL |
| Session metadata model and persistence | S460 | FULL |

### 2.2 What the system does NOT have

| Missing capability | Impact |
|-------------------|--------|
| Canonical round-trip definition | No formal model for what constitutes a paired trade |
| Entry/exit leg matching logic | Cannot automatically find the exit that completes an entry |
| Paired effectiveness read model | No query surface for round-trip outcomes |
| Paired attribution integration | `ClassifyPair()` exists but is not wired to batch evaluation or review surfaces |
| Outcome reconciliation surface | No HTTP endpoint to review paired outcomes, flag mismatches, or inspect unresolved reasons |

### 2.3 Why now

1. **G-SE1 is the single highest-value gap** across the entire analytical depth stack. It is the only MEDIUM gap remaining.
2. **`ClassifyPair()` already exists** -- the domain logic for round-trip P&L is tested and ready. This wave wires it, not invents it.
3. **No new API keys, no new live sessions, no new exchange connectivity required.** All data exists in ClickHouse and KV stores.
4. **The effectiveness read surfaces (S476--S477) are the natural consumer.** Pairing feeds directly into win rate improvement without changing the measurement model.
5. **The wave is small by design.** 4 stages, bounded scope, clear gate criteria.

---

## 3. Wave Objective

Define canonical semantics for round-trip trade pairing, implement automated entry/exit leg matching from existing execution data, integrate paired outcomes into the effectiveness attribution pipeline, and provide a reconciliation surface -- all without expanding the OMS, adding exchange connectivity, or redesigning the execution model.

---

## 4. Wave Blocks (Ordered)

| Block | Stage | Scope | Depends On |
|-------|-------|-------|------------|
| 1. Canonical round-trip and leg-pairing model | S480 | Define `RoundTrip` domain type, leg-matching rules (same symbol, same segment, opposite side, temporal ordering), pairing strategies (FIFO, session-scoped), handling of partial fills, unmatched legs, and edge cases. Domain types and invariants in `internal/domain/pairing/`. | -- |
| 2. Pairing read model and attribution integration | S481 | Implement leg-matching query from existing ClickHouse execution data, wire matched pairs through `ClassifyPair()`, produce paired effectiveness attributions, integrate into batch evaluation pipeline. | S480 |
| 3. Round-trip review and outcome reconciliation | S482 | HTTP endpoint for paired outcome review, reconciliation surface for unresolved/unmatched legs, extension of DecisionReviewBundle with pairing section, batch paired effectiveness endpoint. | S481 |
| 4. Evidence gate | S483 | Formal assessment against governing questions, evidence matrix, residual gaps, wave verdict. | S482 |

**Estimated stages**: 4 (S480--S483)
**Estimated new tests**: 25--40

---

## 5. Governing Questions

These questions define what PASS means for this wave:

| ID | Question | Answered By |
|----|----------|-------------|
| Q-RT1 | Can the system identify and pair entry/exit legs of a round-trip trade from existing execution data with canonical matching rules? | S480, S481 |
| Q-RT2 | Does automated pairing increase the resolved rate (reduce `unresolved` outcomes) compared to the pre-wave single-leg baseline? | S481 |
| Q-RT3 | Are paired round-trip outcomes correctly classified (win/loss/breakeven) with accurate P&L attribution including fees? | S480, S481 |
| Q-RT4 | Can the system surface paired outcomes through HTTP for review, and flag unmatched/unresolved legs with clear reasons? | S482 |
| Q-RT5 | Is round-trip pairing computable from existing data without new exchange connectivity, new ClickHouse tables, or OMS expansion? | S481 |

---

## 6. Scope Freeze Rules

### 6.1 IN scope

- Canonical `RoundTrip` type with entry leg, exit leg, matching metadata, and lifecycle state.
- Leg-matching rules: same symbol, same segment, opposite side (buy/sell), temporal ordering (entry before exit), session-scoped or cross-correlation-ID scoped.
- FIFO matching strategy as the default (earliest unmatched entry pairs with earliest available exit).
- Partial-fill pairing: proportional matching when quantities differ.
- Integration with existing `ClassifyPair()` for P&L computation.
- Paired effectiveness attribution: same `Attribution` struct, now with `resolved` outcomes instead of `unresolved`.
- Extension of batch effectiveness evaluation to include paired results.
- HTTP endpoint for paired outcome review and reconciliation.
- Extension of `DecisionReviewBundle` with pairing section (entry leg, exit leg, match quality, P&L).
- Reconciliation surface: list unmatched legs with reason codes (no exit found, quantity mismatch, session boundary).
- Domain types in `internal/domain/pairing/`.
- Read-path queries from existing ClickHouse execution data.
- Tests covering pairing logic, edge cases, attribution correctness, and reconciliation behavior.

### 6.2 OUT of scope (frozen)

Everything in the non-goals document is frozen. Key exclusions:

- No OMS expansion (no new order types, no position tracking, no portfolio management).
- No real-time pairing or streaming updates (read-path only).
- No new ClickHouse tables (pairing computed from existing execution data).
- No multi-exchange or cross-venue pairing.
- No portfolio-level P&L aggregation across symbols.
- No UI, dashboards, or visualization.
- No write-path changes to execution pipeline.
- No strategy family expansion or new signal types.
- No risk-adjusted metrics (Sharpe, Sortino, etc.).
- No position or risk engine.
- No cross-session pairing beyond correlation-ID scope.
- No advanced derivatives handling (funding rates, liquidation).

### 6.3 Scope change protocol

Any change to scope requires explicit re-freeze with documented rationale. The wave owner must evaluate whether the change preserves the 4-stage budget and does not violate non-goals.

---

## 7. Guard Rails

1. **No OMS expansion.** Pairing is a read-path operation on existing execution data. No new order types, no position model, no portfolio engine.
2. **No new ClickHouse tables.** Round-trip pairing is computed from existing execution and fill records via queries.
3. **No new exchange connectivity.** All data already exists in the pipeline.
4. **No write-path changes.** Pairing does not modify how orders are submitted, filled, or recorded.
5. **No portfolio analytics.** Pairing is per-symbol, per-segment. No cross-symbol aggregation.
6. **No real-time streaming.** Pairing is computed on read, not on write path.
7. **No domain type refactoring.** Pairing adds new types (`RoundTrip`, pairing logic). It does not modify existing `ExecutionIntent`, `FillRecord`, or `Attribution` types.
8. **No UI or dashboard work.** HTTP endpoints only, consistent with existing gateway patterns.
9. **No risk engine or position engine.** Pairing determines trade outcomes, not ongoing risk exposure.
10. **Additive only.** Zero changes to existing behavior. All new code, no modified behavior. Existing effectiveness endpoints continue to work unchanged.

---

## 8. Dependencies

### 8.1 Hard dependencies (must be present)

| Dependency | Status |
|-----------|--------|
| `ClassifyPair(entry, exit)` function | S476, IMPLEMENTED |
| `Classify()` function with `unresolved` handling | S476, IMPLEMENTED |
| `Attribution` struct with P&L fields | S476, IMPLEMENTED |
| `FillRecord` with Price, Quantity, Fee, CostBasis | S428, IMPLEMENTED |
| Batch effectiveness evaluation endpoint | S476, IMPLEMENTED |
| `DecisionReviewBundle` with effectiveness section | S476, IMPLEMENTED |
| CorrelationID / CausationID on all events | Pre-wave, IMPLEMENTED |
| ClickHouse execution data with fills | S411--S413, IMPLEMENTED |
| CompositeReader for analytical queries | Pre-wave, IMPLEMENTED |

### 8.2 Soft dependencies (nice to have but not blocking)

| Dependency | Status | Impact if missing |
|-----------|--------|-------------------|
| PriceSource tagging per intent (RG-2 from S473) | NOT IMPLEMENTED | Cannot compare fill price vs market price at decision time; pairing still works on fill data |
| Futures fee normalization (G-SE3) | INCOMPLETE | Futures round-trip P&L understates fee impact; spot is accurate |
| Statistical significance on cohorts (G-SE2) | NOT IMPLEMENTED | Pairing improves resolved rate but does not add significance testing |

---

## 9. Success Criteria

The wave PASSES if:

1. All 5 governing questions are answered YES or SUBSTANTIAL.
2. Zero regressions across existing test suites.
3. At least 25 new tests covering pairing semantics, matching logic, attribution correctness, and reconciliation behavior.
4. The resolved rate measurably increases compared to the pre-wave baseline (fewer `unresolved` outcomes in batch evaluation).
5. All guard rails observed (no scope creep into OMS, portfolio analytics, or write-path changes).
6. Pairing is computable from existing data without new infrastructure.

---

## 10. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Insufficient exit data -- few sessions have both entry and exit fills | MEDIUM | MEDIUM | Document actual resolved rate improvement; even modest improvement validates the model. Unmatched legs get explicit reason codes. |
| Matching ambiguity when multiple entries exist for same symbol | MEDIUM | LOW | FIFO matching with clear tie-breaking rules. Document limitations for complex scenarios. |
| Partial-fill quantity mismatches complicate pairing | MEDIUM | LOW | Proportional matching: pair available quantity, leave remainder as unmatched leg with reason. |
| Scope inflation toward position tracking | LOW | HIGH | Guard rails + non-goals freeze. Pairing determines trade outcomes, not ongoing positions. |
| Read-path performance with pairing computation | LOW | MEDIUM | Leverage existing ClickHouse query patterns and scan limits. |
| Cross-session pairing demand | MEDIUM | LOW | Explicitly scoped to correlation-ID boundary. Cross-session is a documented non-goal for this wave. |

---

## 11. Timeline and Budget

| Stage | Estimated effort | Type |
|-------|-----------------|------|
| S480 | 1 session | Domain modeling + matching rules |
| S481 | 1 session | Read model + attribution integration |
| S482 | 1 session | Review surface + reconciliation |
| S483 | 1 session | Evidence gate |

**Total**: 4 stages, 4 sessions. No multi-session stages expected.

---

## 12. References

- [S478 Evidence Gate Report](../stages/stage-s478-strategy-effectiveness-evidence-gate-report.md)
- [Strategy Effectiveness Wave Charter](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [Effectiveness Evidence Matrix and Residual Gaps](strategy-effectiveness-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Effectiveness Capabilities and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
- [Measurement Read Surfaces](measurement-read-surfaces-and-batch-evaluation.md)
- [Capabilities, Questions, and Non-Goals](round-trip-pairing-capabilities-questions-and-non-goals.md)
- [S479 Charter Report](../stages/stage-s479-round-trip-pairing-charter-report.md)
