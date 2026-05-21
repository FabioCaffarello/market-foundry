# Family 05 — Selection Rationale and Deferred Candidates

> Formal record of the Family 05 selection decision: why Executions was confirmed, why all other candidates were deferred, and what role Family 05 plays as the terminal test of the Wave B manual expansion pattern.

---

## 1. Selected: Executions (paper_order)

### Why Executions

**Primary rationale:** Executions is the only candidate that completes the analytical dependency chain end-to-end, tests the manual pattern at its maximum stress point, and does so with fully pre-staged write-path artifacts and zero contamination risk to existing families.

**Detailed justification:**

1. **Terminal layer in the dependency chain.** The analytical read path covers layers 1–5 (evidence, signals, decisions, strategies, risk). Executions is layer 6 — the only uncovered layer. After Family 05, every stage of the trading pipeline is analytically visible. No other candidate advances vertical coverage.

2. **Maximum ceiling-test value.** Executions has the highest DDL column count (20), introduces the first Float64 columns in the read path, the first boolean column, the first fills array, and the first two-filter handler method. This is the most complex family by every metric. If the manual pattern absorbs it mechanically, the pattern has proven itself at its natural limit. If friction appears, it defines exactly what codegen must address.

3. **Complete pre-staging.** Migration 006, `mapExecutionRow()` mapper, pipeline config entry, and NATS consumer are all operational. Only the read path needs construction — exactly matching the proven expansion pattern that has held for 5 consecutive families with zero write-path changes.

4. **Isolated responsibility.** Executions sits at the terminal end of the dependency chain. No existing family depends on it. No reader, handler, or use case references execution data. Adding it is purely additive — the lowest possible contamination risk.

5. **Highest analytical value per unit of pattern work.** End-to-end pipeline tracing (evidence → execution) is the single most valuable analytical capability for operators. Every other candidate either deepens existing coverage (marginal value) or is not ready for implementation.

6. **Diagnostic function.** Family 05 is not just an expansion — it is a measurement. Its implementation will produce quantitative signals (handler size, friction count, parser count, implementation time) that determine whether the manual pattern can scale further or has reached its limit. This diagnostic role is only meaningful at the pattern's stress boundary, which executions uniquely occupies.

### What Executions tests about the pattern

| Dimension | What's tested | Prior coverage |
|-----------|---------------|----------------|
| DDL column count | 20 columns (highest) | 17 (Family 04) |
| Float64 columns | 2 (quantity, filled_quantity) | 0 — first occurrence |
| Boolean columns | 1 (final) | 0 — first occurrence |
| JSON column count | 4 (risk, fills, parameters, metadata) | 4 (Family 04) — proven ceiling |
| Fills array structure | First array of fill entries | Signal/decision/strategy input arrays |
| Optional filter count | 2 per handler method (side, status) | 1 per method (outcome, direction, disposition) |
| Handler file at boundary | ~595–615 lines post-expansion | 515 lines (Family 04) |
| Parser count trajectory | 7–8 parsers | 6 parsers (at healthy limit) |
| Total analytical LOC | ~3,943 | ~3,348 |
| Vertical coverage | 6/6 layers | 5/6 layers |

---

## 2. Deferred Candidates

### Deferred: EMA Crossover (ema_crossover signal)

**Status:** Not a family expansion candidate. Not scheduled.

**Why deferred:**

1. **Not a new layer.** EMA Crossover is a signal variant within layer 2. The signal reader already handles it via the `type` query parameter. No new read-path artifacts are needed.

2. **Tests nothing about the pattern.** Adding EMA Crossover requires no new reader, handler, use case, or route. It is a writer config change. Using it as a "family" would dilute the meaning of family expansion and produce zero diagnostic value.

3. **No gate required.** EMA Crossover can be enabled at any time by adding `"ema_crossover"` to the writer's signal families config. It does not require a formal gate or selection review.

**What would make it relevant:** When horizontal deepening (multiple event types per layer) becomes a priority — likely post-vertical-coverage (after Family 05) and post-codegen tranche. At that point, enabling additional signal types is a config/operational task, not an architectural one.

### Deferred: Tradeburst (tradeburst evidence)

**Status:** Not ready for implementation. Not scheduled.

**Why deferred:**

1. **Missing infrastructure.** No writer mapper, no pipeline entry, no ClickHouse migration. Every other family expansion has relied on pre-staged write-path artifacts. Tradeburst would be the first family requiring both write-path and read-path construction — breaking the proven invariant.

2. **Schema not designed.** The ClickHouse table for tradeburst evidence is undefined. Schema design decisions (tick-level granularity, aggregation strategy, retention) are non-trivial and outside the scope of the mechanical expansion pattern.

3. **Within-layer deepening, not vertical extension.** Tradeburst adds another evidence type to layer 1 without advancing the read path to a new layer. Vertical coverage (completing all 6 layers) takes precedence over horizontal depth.

4. **Baseline contamination risk.** Modifying the write path — which has remained immutable for 5 consecutive family expansions — introduces a new risk category. The write path's stability is one of the strongest confidence signals in the system. Breaking it for a within-layer variant is not justified when a vertical extension (executions) is available.

**What would trigger reconsideration:** Completion of vertical coverage (all 6 layers), codegen tranche, and a deliberate decision to begin horizontal deepening. At that point, tradeburst and volume become the natural first candidates for evidence layer enrichment.

### Deferred: Volume (volume evidence)

**Status:** Same as Tradeburst — not ready, not scheduled.

**Why deferred:** Identical reasoning to Tradeburst. No mapper, no migration, no pipeline entry. Within-layer deepening with incomplete infrastructure. Same contamination risk from write-path modification.

**What would trigger reconsideration:** Same triggers as Tradeburst. Volume and tradeburst should be evaluated together as part of a horizontal-deepening initiative.

---

## 3. Why No Other Candidate Was Considered

The candidate universe is exhaustive. The analytical layer tracks events from 6 pipeline layers:

| Layer | Event type(s) | Family status |
|-------|--------------|---------------|
| 1. Evidence | candle_sampled | Baseline (complete) |
| 2. Signals | rsi (+ ema_crossover deferred) | Family 01 (complete) |
| 3. Decisions | rsi_oversold | Family 02 (complete) |
| 4. Strategies | mean_reversion_entry | Family 03 (complete) |
| 5. Risk | position_exposure | Family 04 (complete) |
| 6. Executions | paper_order (+ venue_market_order deferred) | **Family 05 (confirmed)** |

There are no uncovered layers beyond layer 6. The only remaining candidates are within-layer variants:
- `ema_crossover` (layer 2) — handled by existing signal reader
- `venue_market_order` (layer 6) — could share execution reader via `type` filter, similar to EMA Crossover's relationship with the signal reader
- `tradeburst`, `volume` (layer 1) — no infrastructure

No candidate outside this list exists in the NATS event registries or domain definitions.

---

## 4. Ordering Rationale: Why Executions Is Last

### Vertical coverage principle (maintained through all 5 selections)

```
Layer 1: Evidence (candles)           ← Baseline    — foundation
Layer 2: Signals (RSI)                ← Family 01   — first expansion
Layer 3: Decisions (RSI Oversold)     ← Family 02   — first enum filter
Layer 4: Strategies (mean_reversion)  ← Family 03   — 3 JSON columns
Layer 5: Risk (position_exposure)     ← Family 04   — ceiling test (4 JSON, free-text, struct parser)
Layer 6: Executions (paper_order)     ← Family 05   — terminal test (Float64, boolean, fills, 20 cols)
```

### Monotonic complexity gradient (maintained)

| Family | Domain columns | JSON columns | New column types |
|--------|---------------|-------------|-----------------|
| Baseline | 11 | 0 | — |
| F-01 | 8 | 1 | JSON array |
| F-02 | 9 | 2 | Enum filter |
| F-03 | 10 | 3 | Second enum |
| F-04 | 12 | 4 | Free-text, struct parser |
| **F-05** | **15** | **4** | **Float64, boolean, fills array, two filters** |

Each step increases complexity by a bounded amount. The gradient is smooth — no family is a discontinuous jump from its predecessor.

---

## 5. What This Selection Makes Explicit About Family 06

Family 05 closes the vertical coverage chapter. Whatever comes after — whether Family 06 (venue_market_order), horizontal deepening (tradeburst, volume, ema_crossover), or cross-family queries — operates under different conditions:

1. **Codegen is a prerequisite.** The manual pattern reaches ~3,943 LOC at Family 05. Adding a 7th analytical surface manually would push past ~4,538 LOC with >80% structural identity. Template generation is no longer optional.

2. **Handler file must be split.** At ~595–615 lines post-Family-05, a 7th handler method would exceed 700 lines. The file must be split (by family, by generation, or via shared param extraction) before any new analytical method is added.

3. **The expansion pattern changes.** Families 01–05 followed a consistent 9-artifact manual template. Family 06+ will follow a codegen-driven template. This is not a failure of the manual pattern — it is its designed end-of-life. The manual pattern was always intended to prove the structure; codegen automates it.

4. **Horizontal deepening requires a new framework.** Within-layer variants (ema_crossover, venue_market_order) do not follow the 9-artifact template. They require config-level or template-level activation, not full family expansion. The framework for horizontal deepening is undefined and out of scope for Wave B.

---

## 6. Summary

| Decision | Rationale |
|----------|-----------|
| **Executions confirmed as Family 05** | Only candidate advancing vertical coverage; maximum ceiling-test value; complete pre-staging; isolated responsibility |
| **EMA Crossover deferred** | Not a family expansion; handled by existing signal reader; tests nothing about the pattern |
| **Tradeburst deferred** | Missing infrastructure; within-layer; would break write-path immutability invariant |
| **Volume deferred** | Same as Tradeburst |
| **No other candidates exist** | All 6 pipeline layers are covered or pre-staged; remaining variants are within-layer |
| **Family 05 is the terminal manual expansion** | Codegen, handler split, and pattern transition are mandatory before Family 06 |
