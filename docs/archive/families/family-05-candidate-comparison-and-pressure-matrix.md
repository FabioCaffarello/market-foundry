# Family 05 — Candidate Comparison and Pressure Matrix

> Formal comparison of all remaining analytical family candidates for Family 05, evaluated against architectural pressure, analytical value, pattern fitness, and risk.

---

## 1. Candidate Universe

After Families 01–04 (Signals, Decisions, Strategies, Risk Assessments), the following candidates remain:

| # | Candidate | Layer | Type | Write Path Ready | Schema Ready | Read Path Exists |
|---|-----------|-------|------|-----------------|-------------|-----------------|
| A | **Executions (paper_order)** | 6 — Execution | New layer | Yes (mapper, pipeline, NATS consumer) | Yes (migration 006) | No |
| B | EMA Crossover (ema_crossover) | 2 — Signals | Within-layer variant | Partial (requires config addition) | Shared (migration 002) | Yes (signal reader handles via `type`) |
| C | Tradeburst (tradeburst) | 1 — Evidence | Within-layer deepening | No (no mapper, no pipeline entry) | No (no migration) | No |
| D | Volume (volume) | 1 — Evidence | Within-layer deepening | No (no mapper, no pipeline entry) | No (no migration) | No |

---

## 2. Evaluation Criteria

Each candidate is scored against seven dimensions relevant to Family 05's role as the **last manual-pattern test**:

1. **Analytical value** — Does the family unlock new operational visibility?
2. **Vertical coverage** — Does it extend the contiguous read path to a new layer?
3. **Pattern pressure** — Does it stress the Wave B template in the right places?
4. **Incremental complexity** — Is the delta from Family 04 bounded and productive?
5. **Baseline contamination risk** — Could it break or distort existing families?
6. **Pre-staging readiness** — Are write-path artifacts already operational?
7. **Ceiling-test value** — Does it generate signals useful for evaluating the pattern's limits?

---

## 3. Detailed Candidate Evaluation

### Candidate A: Executions (paper_order)

| Dimension | Score | Evidence |
|-----------|-------|----------|
| Analytical value | **High** | Completes end-to-end pipeline visibility (evidence → signal → decision → strategy → risk → execution). Operators can trace any trading decision to its execution outcome. |
| Vertical coverage | **High** | Layer 6 — the only remaining layer. Completes the analytical dependency chain. |
| Pattern pressure | **High** | 20 DDL columns (highest), Float64 quantity fields (first), fills JSON array (first array-of-fills), status enum, exec-specific correlation IDs. Tests every dimension the pattern hasn't seen. |
| Incremental complexity | **Bounded** | Delta from Family 04 (17 cols): +3 domain columns, Float64 types (new), side/status enums (pattern-proven), fills array (new structure). Each increment is individually small. |
| Baseline contamination risk | **Minimal** | Self-contained in layer 6. No upstream reader depends on it. Write path already operational and tested. |
| Pre-staging readiness | **Complete** | Migration 006 exists. `mapExecutionRow()` mapper exists. Pipeline config exists. NATS consumer exists. Only the read path needs construction. |
| Ceiling-test value | **Maximum** | Largest table, most column types, first Float64, first fills array. If the pattern absorbs this mechanically, it has proven itself for any reasonable family. If it creates friction, that friction defines the codegen boundary precisely. |

**Pressure surface:**

| Component | Specific pressure from executions |
|-----------|----------------------------------|
| Schema | 20 columns — largest table. Float64 for quantity/filled_quantity. Two extra correlation IDs. |
| Writer | Already operational — zero changes expected (6th consecutive immutable write path). |
| Reader | New parser candidates: fills array (`[]FillEntry` or similar), float scan for quantity fields. May push parser count from 6 to 7–8. |
| Handler | 6th analytical method. Pushes handler to ~595–615 lines. New optional filters: `side`, `status`. |
| Gateway | ~8 LOC additive (struct DI pattern). |
| CI/Smoke | ~30 lines smoke test extension. 6th `validate_analytical_family()` invocation. |
| Tests | ~32 new tests (reader + use case + handler). Projected total: ~277. |

### Candidate B: EMA Crossover (ema_crossover)

| Dimension | Score | Evidence |
|-----------|-------|----------|
| Analytical value | **Low** | Within-layer variant of signals. The signal reader already handles it via `type` parameter — adding events to the writer config makes it queryable immediately. |
| Vertical coverage | **None** | Same layer as Family 01 (Signals). No new layer reached. |
| Pattern pressure | **None** | No new artifacts needed. No new reader, no new handler, no new route. Tests nothing about the expansion pattern. |
| Incremental complexity | **Zero** | Config change only. |
| Baseline contamination risk | **None** | Shares existing signal infrastructure. |
| Pre-staging readiness | **Partial** | Requires writer config addition. No other artifacts needed. |
| Ceiling-test value | **None** | Proves nothing about pattern scalability. |

**Verdict:** Not a family expansion candidate. Enable independently when needed.

### Candidate C: Tradeburst (tradeburst evidence)

| Dimension | Score | Evidence |
|-----------|-------|----------|
| Analytical value | **Low-Medium** | Deepens evidence layer. Useful for market-impact analysis but not for pipeline tracing. |
| Vertical coverage | **None** | Same layer as baseline (Evidence). No new layer reached. |
| Pattern pressure | **Uncertain** | Schema not designed. Would require new migration, new mapper, new pipeline entry — but for a within-layer table, not a new layer. Tests write-path extension, not read-path expansion. |
| Incremental complexity | **Unknown** | Schema undefined. Could be simple or could require complex tick-level storage. |
| Baseline contamination risk | **Medium** | First time extending the write path as part of a family expansion. Breaks the "write path is immutable" invariant that has held for 5 expansions. |
| Pre-staging readiness | **None** | No mapper, no pipeline entry, no migration. All artifacts from scratch. |
| Ceiling-test value | **Low** | Tests write-path extension rather than read-path pattern scalability. Not the right pressure point for Family 05. |

**Verdict:** Not ready. Requires write-path work that breaks the proven expansion pattern.

### Candidate D: Volume (volume evidence)

| Dimension | Score | Evidence |
|-----------|-------|----------|
| All dimensions | Identical to Tradeburst | Same layer, same readiness gaps, same contamination risk. |

**Verdict:** Same as Tradeburst. Not ready, not appropriate for Family 05.

---

## 4. Comparison Matrix

| Criterion | Executions | EMA Crossover | Tradeburst | Volume |
|-----------|-----------|---------------|------------|--------|
| Analytical value | **High** | Low | Low-Med | Low-Med |
| Vertical coverage | **Layer 6 (new)** | Layer 2 (existing) | Layer 1 (existing) | Layer 1 (existing) |
| Pattern pressure | **High (productive)** | None | Uncertain | Uncertain |
| Incremental complexity | **Bounded** | Zero | Unknown | Unknown |
| Contamination risk | **Minimal** | None | Medium | Medium |
| Pre-staging | **Complete** | Partial | None | None |
| Ceiling-test value | **Maximum** | None | Low | Low |
| **Overall fit** | **Best** | Not a family | Not ready | Not ready |

---

## 5. Pressure Matrix: What Executions Stresses

The pressure matrix below maps exactly where Executions creates load on the existing pattern, component by component:

| Component | Current state (5 families) | Post-executions (6 families) | Pressure type | Threshold proximity |
|-----------|---------------------------|------------------------------|---------------|-------------------|
| Handler file | 515 lines | ~595–615 lines | Size | Concerning (550–600), near critical (>600) |
| Handler methods | 5 | 6 | Count | Manageable |
| Reader adapters | 5 | 6 | Count | Linear, self-contained |
| Use cases | 5 | 6 | Count | Linear, self-contained |
| JSON parsers | 6 | 7–8 | Count | At limit (healthy ≤6, concerning 7–8) |
| DDL columns (total) | ~75 | ~95 | Count | Under threshold (100+) |
| Analytical tables | 6 | 7 (incl. metadata) | Count | Under threshold (12+) |
| Smoke test | ~750 lines | ~780 lines | Size | Linear, helper absorbs |
| Total analytical LOC | ~3,348 | ~3,943 | Size | Under manual threshold (~4,500) |
| Test count | ~245 | ~277 | Count | Proportional |
| Optional filters | 3 (outcome, direction, disposition) | 5 (+ side, status) | Count | Pattern-proven, mechanical |
| Float64 columns | 0 | 2 (quantity, filled_quantity) | **New type** | First occurrence |
| Fills array | 0 | 1 | **New structure** | First occurrence |

### New pressure points unique to executions

1. **Float64 scan and formatting** — First non-string, non-integer, non-bool column type in the read path. Requires `FormatFloat` reuse or new scan handling. Low risk — `FormatFloat` already exists for confidence fields.

2. **Fills JSON array** — First array of execution fill entries. Structure depends on domain definition. May require a new parser (`ParseFillsJSON`) or reuse `ParseMetadataJSON` if fills are stored as `[]map[string]string`. Increases parser count to 7.

3. **Two optional enum filters (side, status)** — First family with two optional filters. Pattern proven individually (outcome, direction, disposition) but not as a pair in one handler method. Low risk — additive WHERE clauses, no interaction.

4. **Execution-specific correlation IDs** — `exec_correlation_id` and `exec_causation_id` are domain-specific variants of the event metadata pattern. Direct column scan — no special handling needed.

5. **Boolean column (final)** — First boolean column in the analytical read path. Trivial scan — `bool` type directly supported by ClickHouse Go driver.

---

## 6. Conclusion

Executions (paper_order) is the only candidate that satisfies all seven evaluation criteria simultaneously. It is the only family that advances vertical coverage, provides maximum ceiling-test value, has complete pre-staging, and stresses the pattern in precisely the dimensions needed to evaluate whether the manual expansion model has reached its limit.

The remaining candidates are either not family expansions (EMA Crossover), not ready for implementation (Tradeburst, Volume), or test the wrong pressure points for the Wave B terminal assessment.
