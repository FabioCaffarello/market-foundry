# Family 03 — Selection and Responsibility Fit Review

> Formal review of the Family 03 selection for Wave B, evaluating architectural responsibility alignment, pattern maturity, and expansion readiness.

---

## 1. Review Context

| Item | Detail |
|------|--------|
| Stage | S174 |
| Gate authority | S173 (Post-Hardening Wave B Gate) — PASS |
| Families delivered | Candles (baseline), Signals/RSI (F01), Decisions/RSI Oversold (F02) |
| Pattern version | v2 (hardened: H-1, H-2, H-3 verified) |
| Active debts | 9 (none blocking) |
| Selection | **Strategies (mean_reversion_entry)** |

---

## 2. Responsibility Map: Current State

### Write Path (complete — 6 layers)

```
Evidence → Signal → Decision → Strategy → Risk → Execution
   ✓          ✓         ✓          ✓         ✓        ✓
candle      rsi    rsi_oversold  mean_rev  pos_exp  paper_order
```

All six layers have active writer pipelines. The write path is a complete, multi-family service with no gaps.

### Read Path (partial — 3 of 6 layers)

```
Evidence → Signal → Decision → Strategy → Risk → Execution
   ✓          ✓         ✓          ✗         ✗        ✗
candle      rsi    rsi_oversold    —         —        —
```

The read path covers the first three layers of the analytical dependency chain. Layers 4-6 have data in ClickHouse but no query surface.

### Read Path After Family 03

```
Evidence → Signal → Decision → Strategy → Risk → Execution
   ✓          ✓         ✓          ✓         ✗        ✗
candle      rsi    rsi_oversold  mean_rev    —        —
```

Family 03 extends the contiguous read path to layer 4, completing the "evaluate → decide → resolve" analytical chain.

---

## 3. Responsibility Fit Analysis

### 3.1 Strategy Domain Responsibilities

The strategy layer has a single, clear responsibility: **resolve whether a strategy condition is met, given upstream decision outcomes**.

In the operational path:
- Receives decision evaluation events
- Resolves strategy conditions (direction, confidence, parameters)
- Emits strategy resolution events

In the analytical path (to be built):
- Stores strategy resolution history in ClickHouse (already done by writer)
- Exposes historical strategy queries via HTTP endpoint (Family 03 scope)

**Responsibility boundary:** The strategy reader does NOT evaluate, re-compute, or aggregate strategies. It serves historical records exactly as they were resolved by the operational runtime.

### 3.2 Boundary Alignment

| Boundary | Strategy fit | Notes |
|----------|-------------|-------|
| Writer → ClickHouse | **Clean** | `mapStrategyRow()` maps event to table row without transformation |
| ClickHouse → Reader | **Clean** | SELECT domain columns, scan to domain struct, return |
| Reader → Use case | **Clean** | Query params in, domain slice out, wall-clock meta |
| Use case → Handler | **Clean** | HTTP params parsed, use case called, JSON response |
| Handler → Route | **Clean** | Conditional registration when ClickHouse is configured |

No boundary crossing is ambiguous. Each layer has a single responsibility and a single direction of data flow.

### 3.3 What Strategy Does NOT Introduce

- No cross-family queries (strategy ← decision join)
- No aggregation or materialized views
- No new infrastructure dependencies
- No operational state coupling
- No schema evolution (additive-only, new reader uses existing table)
- No writer changes (pipeline already active)

---

## 4. Pattern Fit Assessment

### 4.1 Does Strategy Fit the 9-Artifact Template?

| Artifact | Fit | Detail |
|----------|-----|--------|
| 1. Schema | **Pre-staged** | Migration 004 exists, table operational |
| 2. Writer mapper | **Pre-staged** | `mapStrategyRow()` in `cmd/writer/mappers.go` |
| 3. Pipeline entry | **Pre-staged** | Strategy pipeline registered, consuming events |
| 4. Reader adapter | **To build** | Follows signal/decision reader pattern |
| 5. Use case | **To build** | Follows signal/decision use case pattern |
| 6. Handler + route | **To build** | Follows decision handler pattern (with `direction` filter) |
| 7. Integration test | **To build** | Follows decision test pattern |
| 8. Smoke test | **To build** | Single `validate_analytical_family()` call |
| 9. Documentation | **To build** | Coherence table, endpoint spec, limits, frictions |

All 9 artifacts are achievable. 3 are pre-staged, 6 follow proven patterns.

### 4.2 Where Strategy Pressures the Pattern

| Pressure point | Description | Risk |
|----------------|-------------|------|
| Third JSON column | Strategies have `decisions`, `parameters`, and `metadata` — three JSON columns to scan and parse | **Low** — each JSON column is parsed independently, same `json.Unmarshal` pattern |
| Second enum-like filter | `direction` as optional query parameter (buy/sell/hold) alongside universal key params | **Low** — follows `outcome` filter pattern from decisions |
| Four struct DI fields | `AnalyticalHandlerDeps` grows from 3 to 4 fields | **None** — struct DI scales linearly |
| Four smoke validations | Fourth `validate_analytical_family()` call | **None** — parameterized function scales linearly |

**Assessment:** Pattern pressure is moderate and productive. Strategy reveals whether the JSON column pattern scales beyond two, which directly informs Family 04 (risk, with four JSON columns) feasibility.

---

## 5. Constraint Compliance Check

| Constraint | Compliance | Evidence |
|------------|-----------|----------|
| C-1: One family at a time | **Yes** | Only strategies selected; risk, executions deferred |
| C-2: No partial units | **Yes** | All 9 artifacts required before merge |
| C-3: CI before merge | **Yes** | GitHub Actions must pass |
| C-4: No operational regressions | **Yes** | Strategy read path is additive; no operational changes |
| C-5: Schema additive only | **Yes** | No DDL changes; table already exists |
| C-6: Optionality preserved | **Yes** | Gateway starts without ClickHouse; analytical returns 503 |
| C-7: Observability parity | **Yes** | Inserter counters automatic; reader timing, Server-Timing header manual but mechanical |
| C-8: Documentation mandatory | **Yes** | 4 sections required per pattern v2 |
| C-9: Additive only | **Yes** | No existing code modified except to wire new artifacts |

---

## 6. Risk Assessment

### Identified Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| R-1: Three JSON columns reveal scan/parse bottleneck | Low | Low | Each column parsed independently; no interaction between them |
| R-2: `direction` filter semantics differ from `outcome` | Low | Low | Both are LowCardinality strings with predefined values; same pattern |
| R-3: DecisionInput JSON array parsing fails | Very Low | Low | Same pattern as SignalInput in decisions — already proven |
| R-4: Schema coherence error across 4 families | Low | Medium | Unit test assertions on row length and column count; review-enforced |
| R-5: Friction count exceeds threshold | Low | Medium | If >2 new frictions, triggers hardening pause per S167 rule |

### Risk verdict

No identified risk is blocking. The highest-impact risk (R-4: schema coherence at 4 families) is mitigated by existing unit test assertions and has not manifested in 3 prior families. Total risk profile is **lower** than decisions (Family 02), which introduced JSON arrays and optional filters for the first time.

---

## 7. What Remains Out of Scope

- **Family 04 (risk) implementation** — requires strategies gate review first
- **Family 05 (executions) implementation** — requires risk gate review first
- **Within-layer variants** (ema_crossover, tradeburst, volume) — not family expansions
- **Cross-family queries** — not in Wave B scope
- **Codegen** — evaluation deferred to Family 04 (D-4 trigger)
- **CI smoke integration** — tracked gap (PF-5), separate initiative
- **Schema evolution** — no ALTER TABLE, no column additions, no retention changes
- **Materialized views or aggregation** — not in Wave B scope
- **Horizontal refactoring** — writer, reader adapter layer structure unchanged

---

## 8. Preparation for S175

S175 should be the **Strategies Family Definition** stage, establishing:

1. Full schema coherence table (DDL ↔ mapper ↔ reader column alignment)
2. Endpoint specification (`GET /analytical/strategy/history` — params, response contract)
3. Query parameter design (`direction` as optional filter, case sensitivity decision)
4. Known limits and simplifications
5. Success criteria and non-goals

This follows the precedent of S163 (Signal family definition) and S168 (Decisions family definition).
