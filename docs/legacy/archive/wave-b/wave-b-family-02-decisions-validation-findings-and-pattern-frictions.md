# Wave B Family 02 — Decisions: Validation Findings and Pattern Frictions

> Concrete findings, frictions, and observations from validating the second Wave B family (Decisions/RSI Oversold) end-to-end.
> This document captures what worked, what created friction, and what the pattern needs before the third family.

## Findings

### F-1: JSON array deserialization adds no structural friction

The Decisions family is the first to introduce a JSON array column (`signals` storing `[]SignalInput`). The read path uses `ParseSignalInputsJSON()` which follows the same fallback-to-empty pattern as `ParseMetadataJSON()`. `json.Unmarshal` handles arrays and maps identically. No new infrastructure, no special handling required. The pattern scales from `map[string]string` to `[]struct` without modification.

### F-2: Two JSON columns do not compound complexity

Having both `signals` (array) and `metadata` (map) in the same row does not create combinatorial complexity. Each column is parsed independently — the reader scans both as strings and parses separately. No cross-dependency between the two JSON fields.

### F-3: Domain-specific optional filter (outcome) integrates cleanly

The `outcome` parameter is the first optional domain-specific filter beyond the universal key params (source, symbol, timeframe). It required exactly one additional WHERE clause in the query builder and one parameter passthrough in each layer. The pattern absorbs optional filters without structural changes.

### F-4: Write path still required zero changes

Identical to F-01 findings: the writer already consumed decision events from NATS and inserted into ClickHouse. The entire S169 + S170 scope for the decision family was the read path. This confirms the writer was correctly designed as a multi-family service that pre-stages families before the read path is built.

### F-5: Observability parity achieved mechanically (third time)

Decision read path has identical observability to candles and signals: wall-clock timing in adapter, `QueryMeta` in use case, `Server-Timing` header in handler, structured logging at every layer. Achieved by copying the pattern. The observability tax per family is zero — it comes free with the pattern.

### F-6: Error handling contracts remain consistent across 3 families

All three analytical endpoints (candles, signals, decisions) return identical HTTP status codes for identical error conditions:
- 400: missing required params, invalid limit, since > until
- 503: ClickHouse unavailable or reader not configured
- 200: always includes `source`, `meta.query_ms`, `meta.row_count`

The smoke test validates all three families against the same error contract.

### F-7: Confidence float64 round-trip works but has precision implications

The `confidence` field follows: string -> `parseFloat()` -> float64 (DDL) -> float64 (Scan) -> `FormatFloat()` -> string (domain). The round-trip preserves reasonable precision but may alter representation (e.g., "0.80" becomes "0.8", "1.0" becomes "1"). This is cosmetic, not functional, and matches the pattern used for candle OHLCV values.

### F-8: Schema ORDER BY key includes `type` for decisions

The decisions ORDER BY key is `(source, symbol, timeframe, type, timestamp)` — different from candles `(source, symbol, timeframe, open_time)` but matching signals `(source, symbol, timeframe, type, timestamp)`. The `type` dimension is appropriate for typed families. All queries include type in WHERE, so the key is always utilized.

## Pattern Frictions

### PF-1: Constructor with 4 positional args (confirmed, escalated)

**Friction:** `NewAnalyticalWebHandler()` now takes 4 positional arguments (candle, signal, decision, logger). This was identified in F-01 validation (PF-2) as a future risk and is now confirmed as an active friction.

**Impact:** Medium — a fifth argument (Family 03) will make the constructor fragile and error-prone.

**Status:** Committed as H-1 hardening for Family 03 scope. Switch to struct-based DI:
```go
type AnalyticalHandlerDeps struct {
    CandleHistory   getAnalyticalCandleHistoryUseCase
    SignalHistory    getAnalyticalSignalHistoryUseCase
    DecisionHistory getAnalyticalDecisionHistoryUseCase
}
```

### PF-2: `parseEvidenceKeyParams()` naming residue (confirmed, now urgent)

**Friction:** The shared function `parseEvidenceKeyParams()` is now used by 3 families (candles, signals, decisions). The "evidence" prefix is misleading for signals and decisions.

**Impact:** Low-medium — code reads misleadingly across 3 call sites. The function signature and behavior are correct.

**Status:** Should be renamed to `parseAnalyticalKeyParams()` as part of H-1 hardening for Family 03. Three consumers now justify the rename.

### PF-3: Smoke test at ~200 lines for 3 phases

**Friction:** Phase 5c adds ~80 lines for decisions. The smoke script is now approaching 615 lines with the 7-phase structure.

**Impact:** Medium — still maintainable, but a fourth family will push toward restructuring.

**Recommendation:** Extract a `validate_analytical_family()` bash function before Family 03. This was identified in F-01 (PF-5) and is now approaching the threshold.

### PF-4: Outcome filter is case-sensitive and unvalidated

**Friction:** The `outcome` parameter is passed as-is to ClickHouse WHERE clause. Querying `?outcome=TRIGGERED` returns 0 rows (domain values are lowercase). No error is returned for invalid outcome values.

**Impact:** Low — ClickHouse returns empty results. No security risk. The alternative (validating against known outcomes) couples the read adapter to domain constants.

**Recommendation:** Accept. Document in runbook that outcome values are lowercase by convention. Consider adding a `known_outcomes` field to `/diagz` if operators request it.

### PF-5: No CI integration for analytical smoke test (carried from F-01)

**Friction:** The smoke test (`smoke-analytical-e2e.sh`) still runs manually. CI runs unit tests but not the integration smoke.

**Impact:** High — regressions in the analytical integration path can ship undetected. This was flagged as PF-6 in F-01 and remains unresolved.

**Status:** Carried forward. The CI workflow (`.github/workflows/ci.yml`) runs unit tests. Adding a compose-based integration step requires either Docker-in-Docker or a dedicated integration job.

### PF-6: No pagination beyond limit=500

**Friction:** All three analytical endpoints hard-cap at 500 rows. Cursor-based pagination is not available. This was noted in F-01 and remains unchanged.

**Impact:** Low for current usage. Will become a real limitation when operators need to export or scan larger datasets.

**Recommendation:** Defer. Current limit is sufficient for dashboard-style queries. Pagination adds complexity to the contract and the reader.

## Limits Observed

1. **Only RSI Oversold decisions tested** — other decision types (if any) are not validated.
2. **No load testing** — decision queries tested with development-scale data only.
3. **No concurrent query testing** — sequential queries only.
4. **Signals array schema not validated** — reader deserializes `[]SignalInput` without checking for expected fields.
5. **Metadata key schema not validated** — reader deserializes `map[string]string` without checking for expected keys.
6. **Outcome values not validated** — any string accepted, including empty or nonsensical values.
7. **Confidence precision** — float64 round-trip may alter decimal representation (cosmetic).

## What the Pattern Proves (After 2 Family Expansions)

1. **JSON array and JSON map columns follow the same pattern** — `marshalJSON` / `ParseXJSON` with fallback.
2. **Domain-specific optional filters integrate without structural changes** — one WHERE clause, one passthrough per layer.
3. **Write path remains stable** — two family expansions with zero writer changes.
4. **Observability parity is mechanical** — identical instrumentation in every family.
5. **Error handling contracts are stable** — same HTTP codes, same validation rules, same response structure.
6. **Schema coherence is testable offline** — no running ClickHouse needed for verification.
7. **The 9-artifact pattern produces consistent results** — predictable, auditable, no artisanal exceptions.

## What the Pattern Does NOT Yet Prove

1. That H-1 (struct-based DI) resolves constructor accumulation cleanly.
2. That the smoke test scales beyond 3 families without restructuring.
3. That CI integration for smoke tests is achievable with the current pipeline.
4. That cross-family queries (e.g., join decisions with contributing signals) are feasible.
5. That the pattern works for families with fundamentally different data shapes (e.g., nested JSON, variable-schema metadata).
6. That `parseEvidenceKeyParams()` rename to `parseAnalyticalKeyParams()` is the right abstraction boundary.

## Pre-Committed Hardening for Family 03

| ID | Item | Committed In |
|---|---|---|
| H-1 | Refactor `NewAnalyticalWebHandler` to struct-based DI | F-01 PF-2, confirmed F-02 PF-1 |
| H-2 | Parameterize smoke test with `validate_analytical_family()` | F-01 PF-5, approaching threshold |
| H-3 | Rename `parseEvidenceKeyParams()` to `parseAnalyticalKeyParams()` | F-01 PF-1, confirmed F-02 PF-2 |
| H-4 | Review naming consistency (consumer/inserter labels) | S169 report |
