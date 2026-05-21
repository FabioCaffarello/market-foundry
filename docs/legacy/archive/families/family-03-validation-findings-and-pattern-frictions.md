# Family 03 — Strategies: Validation Findings and Pattern Frictions

> Concrete findings, frictions, and observations from validating the third Wave B family (Strategies/Mean Reversion Entry) end-to-end.
> This document captures what worked, what created friction, and what the pattern needs before the fourth family.

## Findings

### F-1: Three JSON columns add no structural friction

The Strategies family is the first to have three JSON-encoded columns (`decisions`, `parameters`, `metadata`). Each column is parsed independently — two via the existing `ParseMetadataJSON()` and one via the new `ParseDecisionInputsJSON()`. No cross-dependency between any JSON fields. The pattern handles 3 JSON columns as easily as 1 or 2.

### F-2: ParseMetadataJSON reuse confirms JSON parsing scalability

`ParseMetadataJSON()` was originally built for the signals family's single `metadata` map column. The strategies family reuses it for both `parameters` and `metadata` — both are `map[string]string`. Zero new parsing code for 2 of the 3 JSON columns. This confirms that the JSON parsing approach scales through reuse, not proliferation.

### F-3: ParseDecisionInputsJSON follows established array pattern

`ParseDecisionInputsJSON()` for the `[]DecisionInput` array follows the exact same shape as `ParseSignalInputsJSON()` from Family 02. The pattern — unmarshal to typed slice, fallback to empty on error — is now established for all array-type JSON columns. No novel parsing required.

### F-4: Direction filter integrates identically to outcome filter

The `direction` parameter follows the same integration pattern as `outcome` in Family 02: one WHERE clause in the query builder, one passthrough in each layer, no validation against known values. The pattern for domain-specific optional filters is now proven twice — it's mechanical and predictable.

### F-5: Write path still required zero changes (fourth time)

Identical to findings in F-01 and F-02: the writer already consumed strategy events from NATS and inserted into ClickHouse. The entire S176 scope for the strategy family was the read path. Four consecutive family expansions (candles, signals, decisions, strategies) with zero writer changes confirms the writer was correctly designed as a multi-family service.

### F-6: Struct-based DI (H-1) proven in first real use

Adding `GetStrategyHistory` to `AnalyticalHandlerDeps`, `AnalyticalWebHandler`, and `AnalyticalFamilyDeps` required only field additions — no constructor signature changes, no reordering risk, no impact on existing fields. The H-1 hardening completed in S172 is validated. Family 04 can be added with the same zero-churn approach.

### F-7: Observability parity achieved mechanically (fourth time)

Strategy read path has identical observability to candles, signals, and decisions: wall-clock timing in adapter, `QueryMeta` in use case, `Server-Timing` header in handler, structured logging at every layer. Achieved by copying the pattern. The observability tax per family remains zero.

### F-8: Error handling contracts remain consistent across 4 families

All four analytical endpoints (candles, signals, decisions, strategies) return identical HTTP status codes for identical error conditions:
- 400: missing required params, invalid limit, since > until
- 503: ClickHouse unavailable or reader not configured
- 200: always includes `source`, `meta.query_ms`, `meta.row_count`

The smoke test validates all four families against the same error contract.

### F-9: Confidence float64 round-trip consistent with Family 02

The `confidence` field follows the same path as in decisions: string -> `parseFloat()` -> float64 (DDL) -> float64 (Scan) -> `FormatFloat()` -> string (domain). Cosmetic representation changes (e.g., "0.80" -> "0.8") are identical to Family 02 behavior. No new concern.

### F-10: DecisionInput struct nesting does not require special handling

The `decisions` JSON column stores `[]DecisionInput` — a struct array with typed fields (Type, Outcome, Confidence, Timeframe). `json.Unmarshal` handles this transparently. No special struct-aware parsing, no custom deserialization. The Go standard library's JSON support is sufficient for the current nesting depth.

## Pattern Frictions

### PF-1: Handler method duplication at ~80 lines per family (escalating)

**Friction:** Each analytical handler method (GetCandleHistory, GetSignalHistory, GetDecisionHistory, GetStrategyHistory) follows the same structure: nil check → parse type → parse key params → parse optional filter → parse limit → parse since/until → execute → set Server-Timing → write response. ~80 lines each. 4 methods × 80 lines = 320 lines of largely identical code.

**Impact:** Medium — readable and auditable, but the duplication means any change to the shared pattern (e.g., adding a new common parameter) requires updating 4 methods.

**Recommendation:** Accept for now. Extracting a generic handler would require generics or interface gymnastics that adds more complexity than the duplication. The duplication is mechanical and grep-safe. Revisit if Family 05+ is confirmed.

### PF-2: Smoke test approaching ~700 lines with 4 families

**Friction:** The smoke test now covers 4 analytical families plus infrastructure, migration, writer, and error handling phases. Each family adds ~30 lines of validation calls plus filter-specific checks.

**Impact:** Medium — the `validate_analytical_family()` function (from H-2 hardening) keeps per-family additions mechanical, but the overall script length and summary section grow linearly.

**Recommendation:** Acceptable through Family 04. If Family 05+ is confirmed, consider splitting into per-family smoke scripts invoked by a coordinator, or parameterizing via a family manifest file.

### PF-3: Direction filter is case-sensitive and unvalidated (same as PF-4 from F-02)

**Friction:** The `direction` parameter is passed as-is to ClickHouse WHERE clause. Querying `?direction=LONG` returns 0 rows (domain values are lowercase). No error is returned for invalid direction values.

**Impact:** Low — ClickHouse returns empty results. No security risk. Consistent with `outcome` behavior in Family 02.

**Recommendation:** Accept. Document in runbook that direction values are lowercase by convention (`long`, `short`, `flat`).

### PF-4: No CI integration for analytical smoke test (carried from F-01 → F-02)

**Friction:** The smoke test (`smoke-analytical-e2e.sh`) still runs manually. CI runs unit tests but not the integration smoke.

**Impact:** High — regressions in the analytical integration path can ship undetected. This was flagged in F-01 (PF-6) and F-02 (PF-5) and remains unresolved.

**Status:** Carried forward. Three consecutive validation stages have flagged this. The CI workflow (`.github/workflows/ci.yml`) runs unit tests. Adding a compose-based integration step requires either Docker-in-Docker or a dedicated integration job.

### PF-5: No pagination beyond limit=500 (carried from F-02)

**Friction:** All four analytical endpoints hard-cap at 500 rows. Cursor-based pagination is not available.

**Impact:** Low for current usage. Consistent across all 4 families.

**Recommendation:** Defer. Current limit is sufficient for dashboard-style queries.

### PF-6: `validate_analytical_family` doesn't verify JSON column contents

**Friction:** The smoke test's `validate_analytical_family()` function checks field presence in the response but does not verify the contents of JSON columns (e.g., that `decisions` contains well-formed `DecisionInput` objects, or that `parameters` is a valid map).

**Impact:** Low — unit tests cover JSON round-trip. The smoke test proves the HTTP-level contract.

**Recommendation:** Accept. Adding JSON schema validation to the smoke test would create maintenance burden without proportional value.

## Limits Observed

1. **Only mean_reversion_entry strategies tested** — other strategy types (if any) are not validated.
2. **No load testing** — strategy queries tested with development-scale data only.
3. **No concurrent query testing** — sequential queries only.
4. **DecisionInput array schema not validated** — reader deserializes `[]DecisionInput` without checking for expected fields.
5. **Parameters/metadata key schema not validated** — reader deserializes `map[string]string` without checking for expected keys.
6. **Direction values not validated** — any string accepted, including empty or nonsensical values.
7. **Confidence precision** — float64 round-trip may alter decimal representation (cosmetic, same as Family 02).
8. **3 JSON columns is the new ceiling** — Family 04 (Risk Assessments) will have 4 JSON columns. The pattern works for 3; 4 should be monitored.

## What the Pattern Proves (After 3 Family Expansions)

1. **JSON column count scales linearly** — 1 (F-01) → 2 (F-02) → 3 (F-03), all handled identically.
2. **JSON parsing functions compose through reuse** — `ParseMetadataJSON` reused 4 times across 3 families, no proliferation.
3. **Domain-specific optional filters are mechanical** — `outcome` (F-02) and `direction` (F-03) follow identical integration pattern.
4. **Write path remains immutable** — four family expansions with zero writer changes.
5. **Struct-based DI (H-1) eliminates constructor churn** — proven in first real use after hardening.
6. **Observability parity is free** — identical instrumentation in every family, no per-family effort.
7. **Error handling contracts are stable** — same HTTP codes, same validation rules, same response structure across 4 families.
8. **Schema coherence is testable offline** — no running ClickHouse needed for verification.
9. **The 9-artifact pattern produces consistent, auditable results** — third expansion, third confirmation.
10. **H-1..H-4 hardening was correctly scoped** — no new structural friction discovered in Family 03.

## What the Pattern Does NOT Yet Prove

1. That 4 JSON columns (Family 04: Risk Assessments) create no new friction.
2. That the smoke test scales beyond 4 families without restructuring.
3. That CI integration for smoke tests is achievable with the current pipeline.
4. That cross-family queries (e.g., strategy with contributing decisions) are feasible.
5. That handler method duplication (PF-1) remains acceptable beyond 5 families.
6. That pagination beyond 500 rows is unnecessary for production use.

## Assessment: Readiness for Family 04 Evaluation

| Criterion | Status |
|---|---|
| Pattern friction | No new structural friction — all frictions carried forward or cosmetic |
| JSON column ceiling | 3 JSON columns proven; 4 is next step (Risk Assessments) |
| Struct DI scalability | Proven — Family 04 will add 1 more field, zero churn |
| Smoke test scalability | Acceptable through 4 families; may need restructuring for 5+ |
| Write path stability | Immutable through 4 expansions — high confidence for Family 04 |
| CI gap | Unresolved — 3 stages have flagged this; decision needed before Family 05 |

**Verdict:** The pattern is ready for a Family 04 evaluation. No blocking friction exists. The decision on whether Family 04 proceeds or requires another hardening tranche depends on whether the CI gap (PF-4) and handler duplication (PF-1) are acceptable risks.
