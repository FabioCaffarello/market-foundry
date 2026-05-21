# Family 05 — Pattern Frictions, Cost, and Scalability Findings

> Concrete findings, frictions, cost analysis, and scalability assessment after completing the fifth and terminal Wave B manual family expansion (Executions/Paper Order).
> This document is the pattern sustainability evidence — it captures what the manual pattern proved, where it reached limits, and what must change before Family 06.

## Findings

### F-1: Two optional filters add no structural friction

Family 05 is the first analytical method with two optional WHERE clauses (side, status). The query builder handles them identically to single-filter methods — each is an independent `AND` clause appended when non-empty. No interaction between filters, no combinatorial concern. The dual-filter pattern is as mechanical as the single-filter pattern.

### F-2: Float64 column reuse is seamless

Family 05 has two Float64 columns (quantity, filled_quantity). Both reuse the existing `FormatFloat()` helper from Family 02. Zero new float handling code was required. FormatFloat now serves 4 columns across 3 families (confidence in F-02/F-03/F-04, quantity and filled_quantity in F-05).

### F-3: ParseRiskInputJSON follows proven struct-target pattern

`ParseRiskInputJSON` deserializes JSON into `execution.RiskInput` — a struct target. This is the same pattern as `ParseConstraintsJSON` (Family 04). Both are ~12 lines, identical shape: empty check → `json.Unmarshal` → fallback. The struct-target parser is now proven twice across different domain types.

### F-4: ParseFillsJSON follows proven slice-target pattern

`ParseFillsJSON` deserializes JSON into `[]execution.FillRecord` — a slice target. This is the same pattern as `ParseSignalInputsJSON`, `ParseDecisionInputsJSON`, and `ParseStrategyInputsJSON`. All four slice-target parsers are ~12 lines, identical shape. The slice-target parser is now proven 4 times across 4 different domain types.

### F-5: ParseMetadataJSON reused 8 times across 5 families

`ParseMetadataJSON()` is now used:
- Signals: metadata (1x)
- Decisions: metadata (1x)
- Strategies: parameters, metadata (2x)
- Risk: parameters, metadata (2x)
- Executions: parameters, metadata (2x)

Total: 8 uses from 1 function. JSON map parsing scales through reuse, not proliferation.

### F-6: Write path still requires zero changes (sixth time)

The writer already consumed execution events from NATS and inserted into ClickHouse before S187. The entire execution family scope was the read path. Six consecutive family expansions (candles, signals, decisions, strategies, risk, executions) with zero writer changes confirms the writer was correctly designed as a multi-family service.

### F-7: Struct-based DI (H-1) continues scaling without friction

Adding `GetExecutionHistory` to `AnalyticalHandlerDeps`, `AnalyticalWebHandler`, and `AnalyticalFamilyDeps` required only field additions — no constructor signature changes, no reordering risk, no impact on existing fields. Sixth family added via struct DI with zero churn.

### F-8: 20 DDL columns verified without tooling pressure

The column alignment verification (DDL → writer → reader) for 20 columns was done manually. Despite being the highest column count in the system, the verification was straightforward because each layer uses explicit column ordering. The manual verification scales linearly with column count.

### F-9: Handler at 615 lines — within ceiling but no margin

The handler file grew from ~515 (post-F-04) to 615 lines (post-F-05) — exactly 100 lines per family. At 615/620, the margin is 5 lines. This confirms the ~100 LOC/family growth rate is consistent and the handler ceiling is real. Family 06 would require ~715 lines — far above the ceiling.

### F-10: Error handling contracts remain consistent across 6 families

All six analytical endpoints return identical HTTP status codes for identical error conditions:
- 400: missing required params, invalid limit, since > until
- 503: ClickHouse unavailable or reader not configured
- 200: always includes `source`, `meta.query_ms`, `meta.row_count`

The smoke test validates all six families against the same error contract.

### F-11: 10-parameter reader signature is the practical limit

`QueryExecutionHistory` has 10 positional parameters — the widest reader in the codebase. Go handles positional args adequately, but readability decreases at this count. A query-object pattern (struct parameter) would improve readability for 11+ parameters.

### F-12: FillRecord struct nesting in JSON is transparent

The `fills` JSON column stores `[]execution.FillRecord` — a struct array with 5 typed fields (Price, Quantity, Fee string; Simulated bool; Timestamp time.Time). `json.Unmarshal` handles this transparently, including the bool field (first occurrence in a nested JSON struct). This proves the pattern handles any struct nesting `json.Unmarshal` can deserialize.

## Pattern Frictions

### PF-1: Handler file at ceiling — split mandatory before Family 06 (ESCALATED)

**Friction:** The handler file is 615 lines (5 below 620-line hard ceiling). Each family adds ~100 lines. Family 06 would require ~715 lines. The handler cannot absorb another family without restructuring.

**Impact:** Critical — this is a hard blocker for Family 06.

**Options:**
1. **Extract `parseAnalyticalParams()` helper** (~1 hour) — reduces per-method code from ~90 to ~30 lines. Buys ~3 more families.
2. **Split handler by domain** (~2 hours) — separate files per family handler. Unlimited scaling.
3. **Codegen handler** — absorbs both concerns. Unlimited scaling by construction.

**Recommendation:** Option 1 as immediate unblock if codegen is not ready. Option 3 as the strategic solution.

### PF-2: 8 parser functions — at threshold for generic parser (ESCALATED)

**Friction:** The ClickHouse adapter now has 8 parser/helper functions:
1. `FormatFloat` — float64 → string (used 4x across 3 families)
2. `ParseMetadataJSON` — string → map[string]string (used 8x across 5 families)
3. `ParseSignalInputsJSON` — string → []signal.SignalInput
4. `ParseDecisionInputsJSON` — string → []strategy.DecisionInput
5. `ParseStrategyInputsJSON` — string → []risk.StrategyInput
6. `ParseConstraintsJSON` — string → risk.Constraints
7. `ParseRiskInputJSON` — string → execution.RiskInput
8. `ParseFillsJSON` — string → []execution.FillRecord

Each parser is ~10–12 lines with identical shape. At 9+ parsers, a generic `parseJSON[T any]` function or codegen template would eliminate the duplication.

**Impact:** Low (each function is trivial) but growing.

**Recommendation:** Accept for now. Codegen would generate per-type parsers from templates, making the count irrelevant.

### PF-3: Smoke test at 651 lines — restructuring candidate (CARRIED)

**Friction:** The smoke test covers 6 analytical families plus infrastructure, migration, writer, error handling, and observability phases. Each family adds ~30–40 lines. The `validate_analytical_family()` reusable function (from H-2 hardening) keeps additions mechanical.

**Impact:** Medium — the script works but is approaching maintainability limits.

**Recommendation:** If Family 06 proceeds, consider splitting per-family validation into sourced files.

### PF-4: No CI integration for analytical smoke test (CARRIED — 5th time)

**Friction:** The smoke test still runs manually. CI runs unit tests but not the integration smoke.

**Impact:** High — regressions in the analytical integration path can ship undetected. Flagged in F-01 through F-05.

**Status:** Carried forward (fifth time). Adding compose-based integration requires CI infrastructure work.

### PF-5: Side and status filters are case-sensitive and unvalidated (same pattern as all previous families)

**Friction:** Both `side` and `status` are passed as-is to ClickHouse WHERE clause. Querying `?side=BUY` returns 0 rows (domain values are lowercase). No error returned for invalid values.

**Impact:** Low — ClickHouse returns empty results. No security risk. Consistent with all previous family filters.

### PF-6: No pagination beyond limit=500 (CARRIED from F-02)

**Friction:** All six analytical endpoints hard-cap at 500 rows. Cursor-based pagination not available.

**Impact:** Low for current usage. Consistent across all families.

### PF-7: Reader method parameter count has no compile-time protection (NEW)

**Friction:** With 10 positional parameters on QueryExecutionHistory, swapping two string parameters at the call site (e.g., `source` and `symbol`) would compile but produce incorrect queries. Earlier families with fewer params had lower risk.

**Impact:** Low — the composition root is the only call site, and integration tests would catch transposition. But the risk increases with parameter count.

**Recommendation:** Query-object pattern (struct parameter) in codegen would eliminate this risk class.

## Incremental Cost Analysis

### Per-Family Cost (Actual, Measured Across F-01 to F-05)

| Cost item | F-01 | F-02 | F-03 | F-04 | F-05 | Trend |
|---|---|---|---|---|---|---|
| New files | 2 | 2 | 2 | 2 | 2 | Flat |
| Modified files | 6 | 6 | 6 | 6 | 6 | Flat |
| New impl LOC | ~270 | ~310 | ~330 | ~350 | ~350 | Slightly increasing, plateauing |
| New test LOC | ~380 | ~410 | ~420 | ~430 | ~450 | Slightly increasing, plateauing |
| Total new LOC | ~650 | ~720 | ~750 | ~780 | ~800 | ~780 average |
| Creative decisions | 0 | 0 | 0 | 0 | 0 | Zero throughout |
| Write path changes | 0 | 0 | 0 | 0 | 0 | Zero throughout |
| New frictions | 1 | 2 | 1 | 2 | 2 | Low, stable |
| Handler growth | +100 | +100 | +100 | +100 | +100 | Linear, predictable |

### Cumulative Cost (6 Families Total: Baseline + F-01 through F-05)

| Metric | Total | Per-family average |
|---|---|---|
| Total analytical impl LOC | ~2,100 | ~350 |
| Total analytical test LOC | ~1,850 | ~310 |
| Total new files (readers + tests) | 12 (2 per family) | — |
| Total modified files | ~30 (6 per family) | — |
| Handler file growth | 0 → 615 lines | ~100/family |
| Smoke test growth | ~300 → 651 lines | ~60/family |
| Parser function count | 0 → 8 | ~1.3/family |

### Artisanal Residual

The following repetitive work was performed manually for every family and would be eliminated by codegen:

1. **Copy reader file** — adapt struct name, query columns, scan variables, JSON parsers, WHERE clauses
2. **Copy reader test file** — adapt expected SQL, test table, filter names
3. **Copy use case file** — adapt interface, query struct, reply struct, validation
4. **Copy use case test file** — adapt mock, test scenarios
5. **Add handler method** — copy previous method, adapt type parsing, filter params, response struct
6. **Add handler test cases** — copy previous test block, adapt assertions
7. **Add route registration** — copy conditional block
8. **Add gateway factory** — copy one-liner
9. **Add composition wiring** — copy reader/use-case instantiation
10. **Add smoke test phase** — copy `validate_analytical_family` call with new params
11. **Add HTTP test file entries** — copy query blocks with new endpoint
12. **Update contracts.go** — add query/reply structs

**Estimated manual effort per family:** ~45 minutes of copy-adapt work.
**Estimated codegen effort per family (once templates exist):** ~2 minutes (schema definition + generate).

## Limits Observed

1. **Only paper_order execution type tested** — `venue_market_order` events are out of scope.
2. **No load testing** — execution queries tested with development-scale data only.
3. **No concurrent query testing** — sequential queries only.
4. **RiskInput struct schema not validated** — reader deserializes without checking for expected fields.
5. **FillRecord struct schema not validated** — reader deserializes without checking for expected fields.
6. **Side values not validated** — any string accepted, including empty or nonsensical.
7. **Status values not validated** — any string accepted.
8. **Quantity precision** — float64 round-trip may alter decimal representation (cosmetic).
9. **Handler file at hard ceiling** — 615/620 lines. No margin for further expansion.
10. **Reader signature at 10 params** — practical limit for positional arguments.
11. **Parser count at 8** — at threshold for generic consolidation.
12. **No fill validation** — fills JSON accepted as-is; no count, amount, or consistency checks.

## What the Pattern Proves (After 5 Family Expansions + 1 Baseline = 6 Total)

1. **The 9-artifact template produces consistent results** — 5 expansions, 5 identical outcomes, zero creative decisions.
2. **Write path is genuinely immutable** — 6 consecutive read-path expansions with zero writer changes.
3. **JSON complexity is a non-concern** — 1, 2, 3, 4 JSON columns all handled identically. Struct, slice, and map targets all work.
4. **Optional filters are mechanical** — 1 or 2 per method, no interaction, additive WHERE clauses.
5. **Float64 handling is solved** — FormatFloat reused across 3 families, 4 columns.
6. **Struct DI eliminates constructor churn** — 6 families, zero signature changes.
7. **Observability is free** — identical instrumentation in every family.
8. **Error contracts are stable** — same behavior across all 6 families.
9. **The pattern is fully learnable** — anyone who reads one family can implement the next.
10. **Full vertical coverage achieved** — Evidence → Signals → Decisions → Strategies → Risk → Executions.

## What the Pattern Does NOT Prove

1. That the handler can absorb Family 06 without refactoring (it cannot — 615/620 ceiling).
2. That codegen can replace manual effort (untested; evidence supports it but not proven).
3. That CI integration for smoke tests is achievable with current infrastructure.
4. That cross-family queries are feasible or needed.
5. That pagination beyond 500 rows is unnecessary for production.
6. That the manual pattern is preferable to codegen at 7+ families.

## Scalability Assessment — Manual Pattern Ceiling

### Pattern is at ceiling. Evidence:

| Signal | Measurement | Verdict |
|---|---|---|
| Handler file | 615/620 lines | **Hard block at F-06** |
| Per-family effort | ~780 LOC, ~45 min | Sustainable but artisanal |
| Parser count | 8 | At threshold |
| Reader params | 10 | At practical limit |
| Total analytical LOC | ~3,950 | Manageable but growing linearly |
| Smoke test | 651 lines | Approaching restructuring threshold |
| Creative decisions | 0 across 5 families | Confirms codegen viability |

### Mandatory prerequisites before Family 06:

1. **Handler refactoring** — extract `parseAnalyticalParams()` or split handler file. Without this, Family 06 cannot be added.
2. **Codegen evaluation** — the zero-creative-decision record across 5 families proves that the pattern is 100% templatable. A codegen tranche definition should scope the effort.
3. **CI smoke integration** — five stages have flagged this as high-severity unresolved friction. Decision required.

### Optional improvements (triggered, not blocking):

4. Generic JSON parser (`parseJSON[T any]`) — replaces 7 identical-shape parsers.
5. Query-object pattern for readers — replaces positional params with struct.
6. Smoke test restructuring — split per-family validation into sourced files.

## Assessment: What This Means for the Wave B Strategy

The manual Wave B pattern has proven its thesis:

- **6 families delivered** with zero structural friction, zero creative decisions, zero write-path changes.
- **Full vertical analytical coverage** achieved: every pipeline layer from Evidence through Executions has a ClickHouse-backed historical query endpoint.
- **The pattern is 100% mechanical** — making it a clear candidate for code generation.

The manual pattern has also proven its limits:

- **Handler file at physical ceiling** — cannot absorb another method.
- **Artisanal cost is measurable** — ~780 LOC and ~45 minutes per family.
- **Codegen would reduce this to ~2 minutes** per family.

**The Wave B manual expansion phase is complete.** The next expansion, if triggered, should use codegen or require a mandatory hardening tranche to address the handler ceiling, parser threshold, and CI gap.
