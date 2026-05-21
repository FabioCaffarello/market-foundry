# Family 03 — Selection Rationale and Deferred Candidates

> Formal record of the Family 03 selection decision: why Strategies was chosen, why others were deferred, and what triggers future candidates.

---

## 1. Selected: Strategies (mean_reversion_entry)

### Why Strategies

**Primary rationale:** Strategies is the only candidate that simultaneously advances the read path into a new layer, introduces healthy incremental complexity, and respects the dependency chain — all without requiring redesign or skipping coverage.

**Detailed justification:**

1. **Next natural layer.** The current read path covers evidence (candles), signals (RSI), and decisions (RSI oversold). Strategies is layer 4 — the immediate successor in the analytical dependency chain. Adding it completes the "evaluate → decide → resolve" read path, which is the most operationally meaningful segment for pipeline debugging and strategy review.

2. **Healthy complexity delta.** Strategies has 15 columns vs decisions' 14. The increment is:
   - `direction` — a new LowCardinality(String) enum-like field, similar to decisions' `outcome` but with different semantics (buy/sell/hold vs approve/reject)
   - `decisions` — JSON array of `DecisionInput` structs, mirroring decisions' `signals` JSON array pattern
   - `parameters` — JSON map, same as decisions' `metadata`
   - `metadata` — JSON map (second instance)

   This means strategies test **three JSON columns** (vs decisions' two) and a **second domain-specific enum filter** — controlled pressure without a structural leap.

3. **Pattern v2 pressure is productive.** The three-JSON-column case is the first real test of whether the scan/parse pattern scales. If it does, risk assessments (four JSON columns) becomes lower-risk as Family 04. If it reveals friction, the pattern can be hardened at a reasonable scale (4 families, not 6).

4. **Write path is already active.** The `mapStrategyRow()` mapper, pipeline entry, and NATS consumer are all operational. Migration 004 exists. Only the read path needs construction — consistent with the proven expansion pattern.

5. **Operational review value.** Strategy resolution history answers critical operational questions: "What strategies were resolved in the last hour? What directions were they resolving to? What decisions drove them?" This is the analytical layer operators will consult most frequently after decisions.

6. **No coverage gap.** Adding strategies after decisions maintains a contiguous read path through layers 1-4. No analytical query surface has to reach across an unimplemented layer.

### What Strategies tests about the pattern

| Dimension | What's tested | Prior coverage |
|-----------|---------------|----------------|
| JSON column count | 3 columns (decisions, parameters, metadata) | 2 columns (decisions) |
| Enum-like filter | `direction` as optional query param | `outcome` (decisions) |
| JSON array of structs | `[]DecisionInput` | `[]SignalInput` (decisions) |
| Schema coherence at 4 families | 4 simultaneous DDL/mapper/reader alignment checks | 3 |
| Struct DI scaling | 4th field in `AnalyticalHandlerDeps` | 3 fields |
| Smoke parameterization | 4th `validate_analytical_family()` call | 3 calls |

### Expected implementation scope

Following Wave B pattern v2, the 9 artifacts for strategies:

1. Schema — **exists** (migration 004)
2. Writer mapper — **exists** (`mapStrategyRow`)
3. Writer pipeline entry — **exists**
4. Reader adapter — **to build** (`QueryStrategyHistory`, `BuildStrategyQuery`)
5. Application use case — **to build** (`GetStrategyHistory`, contracts)
6. HTTP handler + route — **to build** (`GetStrategyHistory` handler, `/analytical/strategy/history`)
7. Integration test — **to build**
8. Smoke test update — **to build** (~7 lines)
9. Documentation — **to build** (coherence table, endpoint spec, known limits, friction log)

Artifacts 1-3 are pre-staged. Artifacts 4-9 follow the proven pattern from signals and decisions.

---

## 2. Deferred Candidates

### Deferred: Risk Assessments (position_exposure)

**Deferred to:** Family 04 (next natural candidate after strategies)

**Why deferred:**

1. **Coverage gap.** Adding risk before strategies would skip layer 4, creating a gap at strategy resolution — the most operationally relevant analytical layer between decisions and risk.

2. **Complexity jump.** Risk has 17 columns with four JSON columns, a free-text `rationale` field, and a `disposition` enum. This is a larger delta from the current pattern (14 → 17) than strategies provides (14 → 15). Adding this after strategies makes the complexity gradient smoother.

3. **Pattern readiness.** The three-JSON-column case (strategies) should be proven before attempting four JSON columns (risk). If strategies reveals JSON-related friction, fixing it at 4 families is cheaper than at 5.

**What makes it the strong Family 04 candidate:**

- Write path fully operational
- Migration exists (005)
- Extends the contiguous read path to layer 5
- After strategies proves three JSON columns, four becomes incremental
- `disposition` filter follows the established enum-filter pattern
- `rationale` (free-text string) is a new but simple column type that adds minimal pattern pressure

**Trigger for Family 04:** Strategies gate review passes (all 9 criteria from pattern v2).

### Deferred: Executions (paper_order)

**Deferred to:** Family 05 (after risk assessments)

**Why deferred:**

1. **Maximum schema complexity.** 20 columns — the largest table in the system. Introduces quantity fields (`Float64` for financial amounts), execution-specific correlation IDs, fill arrays, and status enum. This is not an incremental step from any current family.

2. **Three-layer coverage gap.** Adding executions now would skip strategies and risk, leaving layers 4 and 5 unqueryable in the analytical surface. Operators seeing execution results without strategy/risk context is misleading.

3. **Closest operational coupling.** Executions represent actual or simulated order state. While the write path is isolated, the read path queries are the most operationally sensitive — showing incomplete or stale execution data creates the highest risk of operator confusion.

4. **Terminal position.** As the final layer in the dependency chain, executions should be the last family added, after all upstream layers are analytically visible.

**What makes it the natural Family 05 candidate:**

- Highest analytical value for end-to-end pipeline visibility
- After risk proves four JSON columns and free-text fields, execution's 20-column schema is incremental
- Completes the analytical read path end-to-end
- Natural capstone for Wave B

**Trigger for Family 05:** Risk assessments gate review passes.

### Deferred: EMA Crossover (ema_crossover signal)

**Deferred to:** Not scheduled (within-layer variant, not a family expansion)

**Why deferred:**

1. **Tests nothing new.** The signal reader already supports type-based discrimination. An `ema_crossover` signal is queryable the moment events flow into the `signals` table — the `type` column differentiates them.

2. **Not a read-path expansion.** EMA crossover shares the signal reader, use case, handler, and route with RSI. No new 9-artifact unit is needed. Enabling it is a writer config change, not a family expansion.

3. **Architectural confusion risk.** Treating within-layer variants as families would dilute the meaning of "family expansion" and set a precedent for counting every new event type as a Wave B iteration.

**How it gets enabled:** Add `"ema_crossover"` to `signal_families` in writer config. The signal reader already handles it via `type` filtering.

### Deferred: Tradeburst (tradeburst evidence)

**Deferred to:** Not scheduled (requires write-path work beyond pattern scope)

**Why deferred:**

1. **Incomplete infrastructure.** Unlike candidates A-C, tradeburst has no writer mapper, no pipeline entry, and no migration. The Wave B read-path expansion pattern assumes the write path is pre-staged.

2. **Schema undefined.** The NATS event registry defines the event type but the ClickHouse table schema is not defined. Creating a new evidence table would require schema design decisions that go beyond the mechanical expansion pattern.

3. **Within-layer deepening.** Like EMA crossover, tradeburst deepens an existing layer (evidence) rather than extending the read path downstream. The current priority is vertical coverage, not horizontal depth.

**Trigger for future consideration:** When the vertical read path is complete (all 6 layers covered), within-layer deepening becomes valuable. Tradeburst and volume would be natural first candidates.

### Deferred: Volume (volume evidence)

**Same reasoning as tradeburst.** No writer mapper, no migration, no pipeline entry. Within-layer variant with no vertical progress.

---

## 3. Ordering Rationale

The recommended family ordering follows two principles:

### Principle 1: Contiguous vertical coverage

```
Layer 1: Evidence (candles)     ← Baseline
Layer 2: Signals (RSI)          ← Family 01
Layer 3: Decisions (RSI Oversold) ← Family 02
Layer 4: Strategies (mean_reversion_entry) ← Family 03 (selected)
Layer 5: Risk (position_exposure) ← Family 04 (candidate)
Layer 6: Executions (paper_order) ← Family 05 (candidate)
```

Each family extends the read path by exactly one layer, maintaining a contiguous analytical surface. No operator ever queries a downstream layer without upstream context being available.

### Principle 2: Monotonic complexity gradient

```
Candles:    11 domain columns, 0 JSON columns
Signals:    8 domain columns, 1 JSON column
Decisions:  9 domain columns, 2 JSON columns
Strategies: 10 domain columns, 3 JSON columns  ← Family 03
Risk:       12 domain columns, 4 JSON columns  ← Family 04
Executions: 15 domain columns, 4 JSON columns + quantities ← Family 05
```

Each step increases schema complexity by a bounded amount. No jump exceeds +3 columns or +1 JSON column over the predecessor.

---

## 4. Triggers This Selection Reveals for Family 04

| Trigger | Condition | Action |
|---------|-----------|--------|
| T-1: Three JSON columns proven | Strategies gate passes without JSON-related friction | Risk assessments can proceed with four JSON columns |
| T-2: Direction filter proven | Strategy `direction` query param works | Risk `disposition` filter follows same pattern |
| T-3: Codegen pressure | D-4 activates at Family 04 per pattern v2 | Evaluate code generation for reader/handler/test boilerplate |
| T-4: Free-text column | Risk introduces `rationale` (String, not JSON) | First non-enum, non-JSON, non-numeric text column in analytical read path |
| T-5: Friction count | If strategies introduces >2 new frictions | Consider hardening pause before Family 04 |
