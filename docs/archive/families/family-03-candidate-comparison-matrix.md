# Family 03 — Candidate Comparison Matrix

> Formal comparison of all viable candidates for the third Wave B analytical family expansion.
> Evaluated against architectural fit, value, complexity, risk, and pattern pressure.

---

## 1. Evaluation Criteria

| # | Criterion | Weight | Description |
|---|-----------|--------|-------------|
| C1 | Architectural boundary fit | High | Does the candidate extend the read path into a layer not yet covered, or deepen an existing layer? |
| C2 | Incremental complexity | High | Does the candidate add meaningful but controlled complexity over Family 02? |
| C3 | Pattern v2 pressure | Medium | Does the candidate test the pattern in a new dimension without requiring redesign? |
| C4 | Value to analytical surface | Medium | Does the candidate unlock meaningful historical queries for operators or downstream consumers? |
| C5 | Risk to operational baseline | High | Can the candidate be added without touching operational paths or writer infrastructure? |
| C6 | Dependency chain position | Medium | Where in the pipeline dependency graph does the candidate sit? |
| C7 | Existing infrastructure readiness | Medium | How much of the 9-artifact checklist is already pre-staged? |

---

## 2. Candidate Roster

### Candidate A: Strategies (mean_reversion_entry)

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| C1: Boundary fit | **Strong** | Opens the **4th analytical layer** (strategy). Read path currently covers evidence, signal, decision — strategy is the next natural step downstream. |
| C2: Complexity | **Healthy** | 15 columns (vs 14 for decisions). Adds `direction` (enum), `decisions` (JSON array of DecisionInput), `parameters` (JSON map), and `metadata` (JSON map). Three JSON columns vs decisions' two — a meaningful but bounded step. |
| C3: Pattern pressure | **Moderate** | Tests whether the pattern handles a third JSON column and a second enum-like filter cleanly. Does NOT require structural changes. |
| C4: Analytical value | **High** | Strategy resolution history is the bridge between signal/decision evaluation and risk/execution. Queries like "show all mean_reversion_entry resolutions for BTCUSD in the last 24h" are high-value for operational review. |
| C5: Operational risk | **None** | Write path already active. Strategy table exists. NATS consumer running. Only read path is new. |
| C6: Dependency position | **Layer 4 of 6** | Depends on decisions (covered). Upstream of risk and execution (not yet covered in read path). |
| C7: Readiness | **High** | Migration 004 exists. Writer mapper exists. Pipeline entry exists. Consumer active. Only reader, use case, handler, route, tests, smoke, docs needed. |

### Candidate B: Risk Assessments (position_exposure)

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| C1: Boundary fit | **Strong** | Opens the **5th analytical layer** (risk). Skips strategy layer in read path — creates a coverage gap. |
| C2: Complexity | **Elevated** | 17 columns (vs 14 for decisions, 15 for strategies). Adds `disposition` (enum), `constraints` (JSON), `rationale` (free text string), `strategies` (JSON array), `parameters` (JSON), `metadata` (JSON) — four JSON columns. |
| C3: Pattern pressure | **High** | Tests four JSON columns, free-text string field, and a potential domain-specific filter (`disposition`). Significant pattern pressure. |
| C4: Analytical value | **High** | Risk assessment history is valuable for compliance review and operational audit. |
| C5: Operational risk | **None** | Write path already active. Risk table exists. |
| C6: Dependency position | **Layer 5 of 6** | Depends on strategies (not yet covered in read path). Skipping to risk creates an incomplete analytical chain. |
| C7: Readiness | **High** | Migration 005 exists. Writer mapper exists. Pipeline entry exists. |

### Candidate C: Executions (paper_order)

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| C1: Boundary fit | **Terminal** | Opens the **6th analytical layer** (execution). Skips strategy and risk layers — creates a two-layer coverage gap. |
| C2: Complexity | **High** | 20 columns (vs 14 for decisions). Adds `side`, `quantity`, `filled_quantity`, `status`, `risk` (JSON), `fills` (JSON array), `parameters` (JSON), `metadata` (JSON), `exec_correlation_id`, `exec_causation_id`. Significant column count jump. |
| C3: Pattern pressure | **Very High** | Tests numeric quantity fields, execution-specific correlation IDs, fill arrays, and the largest schema in the system. Risk of revealing pattern weaknesses that should be discovered incrementally. |
| C4: Analytical value | **Very High** | Execution history is the most directly actionable analytical query for operators. |
| C5: Operational risk | **Low** | Write path active, but execution domain has the closest coupling to operational state (fills, orders). |
| C6: Dependency position | **Layer 6 of 6** | Terminal node. Depends on risk (not covered) and strategy (not covered). Three-layer gap in read path. |
| C7: Readiness | **High** | Migration 006 exists. Writer mapper exists. Pipeline entry exists. |

### Candidate D: EMA Crossover (ema_crossover signal)

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| C1: Boundary fit | **Deepening** | Adds a second variant within signal layer. Does NOT open a new layer. |
| C2: Complexity | **Minimal** | Identical domain structure to RSI — same `Signal` struct, same column mapping, different `type` value. No new columns, no new JSON shapes, no new filters. |
| C3: Pattern pressure | **None** | Tests nothing new. The existing signal reader already handles `ema_crossover` by type discrimination — it just needs the signal events to flow. |
| C4: Analytical value | **Low-Moderate** | Adds historical visibility into a second signal type, but the query surface is already proven for signals. |
| C5: Operational risk | **None** | Write path already mapped. Signal reader already supports type-based filtering. |
| C6: Dependency position | **Layer 2 of 6** | Same layer as RSI. No new dependency traversal. |
| C7: Readiness | **Overqualified** | The existing signal reader already supports querying ema_crossover signals. Adding this family to the writer config is the only real action. No new reader, handler, or route needed. |

### Candidate E: Tradeburst (tradeburst evidence)

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| C1: Boundary fit | **Deepening** | Adds a second variant within evidence layer. Does NOT open a new layer. |
| C2: Complexity | **Low** | Likely similar to candles (sampled event with trade metrics). Would share the `evidence_candles` table schema approach or require a new table. |
| C3: Pattern pressure | **Low** | Tests whether a second evidence family creates friction, but evidence is the simplest domain. |
| C4: Analytical value | **Moderate** | Trade burst detection is useful for volatility analysis but less operationally critical than strategy or risk history. |
| C5: Operational risk | **None** | Event registry defined but no writer mapper yet — requires both write and read path implementation. |
| C6: Dependency position | **Layer 1 of 6** | No dependencies. Does not advance the read path coverage downstream. |
| C7: Readiness | **Partial** | NATS registry defined but no writer mapper, no migration (would need a new table), no pipeline entry. More work than candidates A-C. |

---

## 3. Comparison Summary

| Criterion | Strategies (A) | Risk (B) | Executions (C) | EMA Crossover (D) | Tradeburst (E) |
|-----------|:-:|:-:|:-:|:-:|:-:|
| C1: Boundary fit | **Strong** | Strong | Terminal | Deepening | Deepening |
| C2: Complexity | **Healthy** | Elevated | High | Minimal | Low |
| C3: Pattern pressure | **Moderate** | High | Very High | None | Low |
| C4: Analytical value | High | High | Very High | Low-Moderate | Moderate |
| C5: Operational risk | **None** | None | Low | None | None |
| C6: Dependency chain | **Layer 4** | Layer 5 (gap) | Layer 6 (gap) | Layer 2 (same) | Layer 1 (same) |
| C7: Readiness | **High** | High | High | Overqualified | Partial |
| **Overall fit** | **Best** | Good, premature | Good, premature | Too shallow | Incomplete |

---

## 4. Ranking

| Rank | Candidate | Verdict |
|------|-----------|---------|
| **1** | **Strategies (mean_reversion_entry)** | Best fit — extends read path to next natural layer, healthy complexity increment, moderate pattern pressure, high readiness |
| 2 | Risk Assessments (position_exposure) | Strong candidate but premature — should follow strategies to avoid coverage gap |
| 3 | Executions (paper_order) | Highest value but highest risk — skips two layers, maximum schema complexity |
| 4 | EMA Crossover | Too shallow — tests nothing new, existing infrastructure already handles it |
| 5 | Tradeburst | Incomplete infrastructure — requires write path work beyond read-path expansion pattern |
