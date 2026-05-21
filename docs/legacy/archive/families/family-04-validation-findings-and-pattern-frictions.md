# Family 04 — Risk Assessments: Validation Findings and Pattern Frictions

> Concrete findings, frictions, and observations from validating the fourth Wave B family (Risk Assessments/Position Exposure) end-to-end.
> This document captures what worked, what created friction, and what the pattern needs before the fifth family.

## Findings

### F-1: Four JSON columns add no structural friction

The Risk Assessments family is the first to have four JSON-encoded columns (`strategies`, `constraints`, `parameters`, `metadata`). Each column is parsed independently — two via the existing `ParseMetadataJSON()`, one via the new `ParseStrategyInputsJSON()`, and one via the new `ParseConstraintsJSON()`. No cross-dependency between any JSON fields. The pattern handles 4 JSON columns as easily as 1, 2, or 3.

The ceiling concern raised in Family 03's validation ("4 is a ceiling not yet tested") is now resolved. **4 JSON columns is proven.**

### F-2: Struct-target parser (ParseConstraintsJSON) is simpler than array parsers

`ParseConstraintsJSON` is the first parser that deserializes into a struct rather than a slice or map. Structurally, it's the simplest parser yet — `json.Unmarshal` directly into `risk.Constraints`. The fallback to zero-value struct on error is trivial. This proves that the parser pattern scales to any Go type `json.Unmarshal` can handle.

### F-3: Free-text column (rationale) is the simplest column type

The `rationale` column was the first free-text column in the analytical layer. Its handling was trivially simple:
- Writer: direct string pass-through (`r.Rationale`)
- Reader: standard `string` scan, no parsing
- Handler: direct JSON serialization via `encoding/json`

Free-text columns are simpler than JSON columns, simpler than enum columns, and simpler than float columns. No new patterns, no special handling. This proves free-text is a non-concern for future families.

### F-4: ParseMetadataJSON reused 6 times across 4 families

`ParseMetadataJSON()` originally built for the signals family's single `metadata` column is now reused:
- Signals: metadata (1x)
- Decisions: metadata (1x)
- Strategies: parameters, metadata (2x)
- Risk: parameters, metadata (2x)

Total: 6 uses from 1 function. Zero new parsing code for 6 of the family's column mappings. JSON parsing scales through reuse, not proliferation.

### F-5: Disposition filter integrates identically to outcome and direction

The `disposition` parameter follows the same integration pattern as `outcome` (Family 02) and `direction` (Family 03): one WHERE clause in the query builder, one passthrough in each layer, no validation against known values. The pattern for domain-specific optional filters is now proven three times — it's mechanical and predictable.

### F-6: Write path still requires zero changes (fifth time)

Identical to findings in previous families: the writer already consumed risk events from NATS and inserted into ClickHouse. The entire S181 scope for the risk family was the read path. Five consecutive family expansions (candles, signals, decisions, strategies, risk) with zero writer changes confirms the writer was correctly designed as a multi-family service.

### F-7: Struct-based DI (H-1) continues scaling without friction

Adding `GetRiskHistory` to `AnalyticalHandlerDeps`, `AnalyticalWebHandler`, and `AnalyticalFamilyDeps` required only field additions — no constructor signature changes, no reordering risk, no impact on existing fields. Fifth family added via struct DI with zero churn.

### F-8: Observability parity achieved mechanically (fifth time)

Risk read path has identical observability to all previous families: wall-clock timing in adapter, `QueryMeta` in use case, `Server-Timing` header in handler, structured logging at every layer. Achieved by copying the pattern. The observability tax per family remains zero.

### F-9: Error handling contracts remain consistent across 5 families

All five analytical endpoints return identical HTTP status codes for identical error conditions:
- 400: missing required params, invalid limit, since > until
- 503: ClickHouse unavailable or reader not configured
- 200: always includes `source`, `meta.query_ms`, `meta.row_count`

The smoke test validates all five families against the same error contract.

### F-10: 17 DDL columns verified without tooling pressure

The column alignment verification (DDL -> writer -> reader) for 17 columns was done manually. Despite being the highest column count, the verification was straightforward because each layer uses explicit column ordering — no ORM, no reflection, no magic mapping. The manual verification scales linearly with column count and remains practical.

### F-11: StrategyInput struct nesting in JSON is transparent

The `strategies` JSON column stores `[]risk.StrategyInput` — a struct array with 4 typed fields (Type, Direction, Confidence, Timeframe). `json.Unmarshal` handles this transparently. This is the second nested struct array (after `[]strategy.DecisionInput` in Family 03), confirming the pattern handles arbitrary struct nesting.

### F-12: Total parser function count (6) remains manageable

After Family 04, the analytical adapter has 6 parser functions:
1. `FormatFloat` — float64 → string (used by decisions, strategies, risk)
2. `ParseMetadataJSON` — string → map[string]string (used 6x across 4 families)
3. `ParseSignalInputsJSON` — string → []signal.SignalInput
4. `ParseDecisionInputsJSON` — string → []strategy.DecisionInput
5. `ParseStrategyInputsJSON` — string → []risk.StrategyInput
6. `ParseConstraintsJSON` — string → risk.Constraints

Each parser is ~10 lines. All follow the same shape: empty check → unmarshal → fallback. No abstraction needed — the repetition is trivial and grep-safe.

## Pattern Frictions

### PF-1: Handler method duplication at ~90 lines per family (escalating)

**Friction:** Each analytical handler method follows the same structure: nil check → parse type → parse key params → parse optional filter → parse limit → parse since/until → execute → set Server-Timing → write response. ~90 lines each. 5 methods × 90 lines = ~450 lines of largely identical code.

**Impact:** Medium — readable and auditable, but any change to the shared pattern requires updating 5 methods. The handler file is now ~515 lines (under the 600-line threshold flagged in S181, but approaching it).

**Recommendation:** Accept for now. If Family 05 is confirmed, the handler file will cross 600 lines. At that point, extracting `parseAnalyticalParams()` helper for the common prefix (type/source/symbol/timeframe/limit/since/until parsing) would reduce per-method duplication from ~90 to ~30 lines without adding abstraction complexity. This is a triggered refactor candidate, not a blocker.

### PF-2: Smoke test approaching ~750 lines with 5 families

**Friction:** The smoke test now covers 5 analytical families plus infrastructure, migration, writer, and error handling phases. Each family adds ~30 lines of validation calls plus filter-specific checks.

**Impact:** Medium — the `validate_analytical_family()` function (from H-2 hardening) keeps per-family additions mechanical, but the script is growing.

**Recommendation:** If Family 05 is confirmed, consider splitting per-family validation into separate functions or files invoked by a coordinator.

### PF-3: Disposition filter is case-sensitive and unvalidated (same pattern as PF-3/F-03, PF-4/F-02)

**Friction:** The `disposition` parameter is passed as-is to ClickHouse WHERE clause. Querying `?disposition=APPROVED` returns 0 rows (domain values are lowercase). No error returned for invalid values.

**Impact:** Low — ClickHouse returns empty results. No security risk. Consistent with `outcome` and `direction` behavior.

**Recommendation:** Accept. Document in runbook that disposition values are lowercase by convention (`approved`, `modified`, `rejected`).

### PF-4: No CI integration for analytical smoke test (carried from F-01 → F-02 → F-03)

**Friction:** The smoke test (`smoke-analytical-e2e.sh`) still runs manually. CI runs unit tests but not the integration smoke.

**Impact:** High — regressions in the analytical integration path can ship undetected. This was flagged in F-01, F-02, F-03, and now F-04.

**Status:** Carried forward (fourth time). The CI workflow runs unit tests. Adding a compose-based integration step requires infrastructure work.

### PF-5: No pagination beyond limit=500 (carried from F-02)

**Friction:** All five analytical endpoints hard-cap at 500 rows. Cursor-based pagination is not available.

**Impact:** Low for current usage. Consistent across all 5 families.

**Recommendation:** Defer. Current limit is sufficient for dashboard-style queries.

### PF-6: `validate_analytical_family` doesn't verify JSON column contents (carried from F-03)

**Friction:** The smoke test checks field presence in the response but does not verify the structure of JSON columns (e.g., that `strategies` contains well-formed `StrategyInput` objects, or that `constraints` is a valid struct).

**Impact:** Low — unit tests cover JSON round-trip. The smoke test proves the HTTP-level contract.

**Recommendation:** Accept. Adding JSON schema validation to the smoke test would create maintenance burden without proportional value.

### PF-7: 6 parser functions with identical shape — extraction candidate

**Friction:** All 6 parser functions in the ClickHouse adapter follow the same shape: empty/default check → `json.Unmarshal` → fallback. The only differences are the target type and the empty-check values.

**Impact:** Low — each function is ~10 lines, and the duplication is trivial. However, if Family 05+ adds more JSON columns, the count will grow.

**Recommendation:** Accept. A generic `parseJSON[T any](raw string, defaults T) T` function could replace all 5 JSON parsers (excluding FormatFloat), but Go generics in this context add cognitive overhead for minimal savings. Revisit if parser count exceeds 8.

## Limits Observed

1. **Only position_exposure risk type tested** — other risk types (if any) are not validated.
2. **No load testing** — risk queries tested with development-scale data only.
3. **No concurrent query testing** — sequential queries only.
4. **StrategyInput array schema not validated** — reader deserializes `[]StrategyInput` without checking for expected fields.
5. **Constraints struct schema not validated** — reader deserializes into `Constraints` without checking for expected fields; extra fields are silently ignored.
6. **Parameters/metadata key schema not validated** — reader deserializes `map[string]string` without checking for expected keys.
7. **Disposition values not validated** — any string accepted, including empty or nonsensical values.
8. **Rationale content not validated** — free-text is written and read as-is; no length or content checks.
9. **Confidence precision** — float64 round-trip may alter decimal representation (cosmetic, same as Family 02 and 03).
10. **4 JSON columns + 1 free-text is the new ceiling** — Family 05 (Executions) will define the next complexity level.

## What the Pattern Proves (After 4 Family Expansions)

1. **JSON column count scales linearly** — 1 (F-01) → 2 (F-02) → 3 (F-03) → 4 (F-04), all handled identically.
2. **JSON parser types are flexible** — slices (3 parsers), maps (1 parser reused 6x), structs (1 parser). All work.
3. **Free-text columns are trivial** — simpler than any other column type. No new patterns needed.
4. **Domain-specific optional filters are mechanical** — `outcome` (F-02), `direction` (F-03), `disposition` (F-04) follow identical pattern.
5. **Write path remains immutable** — five family expansions with zero writer changes.
6. **Struct-based DI (H-1) eliminates constructor churn** — five families added without signature changes.
7. **Observability parity is free** — identical instrumentation in every family, no per-family effort.
8. **Error handling contracts are stable** — same HTTP codes, same validation rules, same response structure across 5 families.
9. **Schema coherence is testable offline** — no running ClickHouse needed for verification.
10. **The 9-artifact pattern produces consistent, auditable results** — fourth expansion, fourth confirmation.
11. **17 DDL columns verified without pressure** — explicit column ordering scales to any reasonable count.
12. **Pattern ceiling tested and passed** — highest complexity family (4 JSON, 1 free-text, 17 DDL cols, struct-target parser) absorbed mechanically.

## What the Pattern Does NOT Yet Prove

1. That the handler file scales beyond 600 lines without refactoring (approaching threshold).
2. That the smoke test scales beyond 5 families without restructuring.
3. That CI integration for smoke tests is achievable with the current pipeline.
4. That cross-family queries (e.g., risk with contributing strategies) are feasible.
5. That Family 05 (Executions) can be added without handler refactoring.
6. That pagination beyond 500 rows is unnecessary for production use.

## Assessment: Readiness for Family 05 Evaluation

| Criterion | Status |
|---|---|
| Pattern friction | No new structural friction — all frictions carried forward, cosmetic, or triggered |
| JSON column ceiling | 4 JSON columns proven; pattern handles any count |
| Free-text ceiling | 1 free-text column proven; trivially simple |
| Struct DI scalability | Proven through 5 families — zero churn |
| Smoke test scalability | Acceptable through 5 families; may need restructuring for 6+ |
| Handler file size | ~515 lines — approaching 600-line threshold; refactoring triggered if F-05 confirmed |
| Write path stability | Immutable through 5 expansions — high confidence for any future family |
| CI gap | Unresolved — 4 stages have flagged this; decision needed before Family 06 |

**Verdict:** The pattern is ready for a Family 05 evaluation. No blocking friction exists. The handler refactoring (PF-1) is a triggered candidate — if Family 05 proceeds, extracting the common parameter-parsing prefix is the right sequencing. The CI gap (PF-4) remains the only high-severity friction and is orthogonal to family expansion.
