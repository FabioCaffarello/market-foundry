# Stage S455A: Session Explainability and Cross-Surface Consistency

**Status:** Complete
**Date:** 2026-03-24
**Wave:** Operational History and Explainability (post-S453A/S454A)

## Objective

Design, implement, and validate a minimal session explainability surface and audit cross-surface consistency between KV, ClickHouse, and gateway read paths.

## Context

After S453A (historical lifecycle read model) and S454A (operational list queries), the wave had historical depth and retrieval capabilities. The next step was connecting these surfaces into a single explainability view and auditing whether the surfaces tell a consistent story.

## Deliverables

### Code Changes

| File | Change | Purpose |
|------|--------|---------|
| `internal/application/analyticalclient/contracts.go` | Added `SessionExplainQuery`, `SessionExplainReply`, `ConsistencyCheck`, `intentToLifecycleEntry()`, `Risk`/`Parameters` fields to `LifecycleHistoryEntry` | Contracts for explain endpoint + field parity fix |
| `internal/application/analyticalclient/get_session_explain.go` | New file | Use case combining KV + CH + consistency checks |
| `internal/application/analyticalclient/s455a_session_explain_test.go` | New file | 6 test cases covering consistent, divergent, unavailable, validation, rejection scenarios |
| `internal/application/analyticalclient/get_lifecycle_history.go` | Refactored to use `intentToLifecycleEntry()` | DRY conversion |
| `internal/application/analyticalclient/get_execution_list.go` | Refactored to use `intentToLifecycleEntry()` | DRY conversion |
| `internal/interfaces/http/handlers/analytical.go` | Added `GetSessionExplain` handler + deps | HTTP handler |
| `internal/interfaces/http/routes/analytical.go` | Added route, interface, deps field | Route registration |
| `cmd/gateway/compose.go` | Wired `GetSessionExplainUseCase` with both CH reader and KV reader | Composition root |

### New Endpoint

**`GET /analytical/execution/explain?source=...&symbol=...&timeframe=...&limit=...`**

Returns:
- KV latest state (intent, fill, rejection, propagation)
- ClickHouse history (recent lifecycle events)
- Cross-surface consistency checks (per-field comparison)
- Human-readable explanation

### Documentation

| Document | Path |
|----------|------|
| Explainability surface design + audit | `docs/architecture/session-explainability-surface-and-cross-surface-consistency-audit.md` |
| KV/CH/Gateway consistency findings | `docs/architecture/kv-clickhouse-gateway-lifecycle-consistency-findings-and-limitations.md` |
| Stage report | `docs/stages/stage-s455a-session-explainability-and-consistency-report.md` |

## Consistency Audit Results

### Consistent (No Issues)

- Status enum values (same across KV and CH)
- Rejection metadata embedding (identical logic in both paths)
- Propagation derivation (same function)
- Event type taxonomy (paper_order, venue_market_order, venue_rejection)
- Partition key scheme ({source}.{symbol}.{timeframe})
- JSON-encoded fields (risk, fills, parameters, metadata) round-trip correctly

### Known Limitations (Not Divergences)

- **Quantity precision**: KV string vs CH Float64 → trailing zero differences (numeric equivalence preserved)
- **Dual correlation ID**: CH has both envelope and intent correlation IDs (reader correctly maps intent ID)
- **Timestamp serialization**: RFC3339 in HTTP responses drops sub-second precision (preserved in storage)

### Parity Gaps Corrected

- **LifecycleHistoryEntry**: Added `Risk` and `Parameters` fields that were present in ExecutionIntent but missing from lifecycle entries
- **Conversion centralization**: Introduced `intentToLifecycleEntry()` to prevent future field drift

### Remaining Gaps

- No automated consistency monitoring or alerting
- No reconciliation mechanism for persistent divergences
- KV lifecycle list is O(3N) — CH summary queries are more efficient for aggregates
- No sub-second timestamp precision in HTTP lifecycle responses

## Test Results

```
=== RUN   TestSessionExplain_ConsistentFilled       --- PASS
=== RUN   TestSessionExplain_Divergent               --- PASS
=== RUN   TestSessionExplain_KVUnavailable           --- PASS
=== RUN   TestSessionExplain_NilKVReader             --- PASS
=== RUN   TestSessionExplain_ValidationErrors        --- PASS (3 subtests)
=== RUN   TestSessionExplain_RejectionConsistency    --- PASS
```

All existing S453A and S454A tests continue to pass.

## Acceptance Criteria Assessment

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Explainability operacional improves materially | Met | Single endpoint combines KV state + CH history + consistency checks + narrative |
| Consistency between surfaces is audited and clearer | Met | Field-level audit documented; parity gaps corrected; explain endpoint exposes consistency live |
| Stage increases operational confidence without live session | Met | Explain endpoint works purely from read paths; no write-side changes |
| Wave ready for evidence gate in S456A | Met | Explainability surface operational; consistency documented; remaining gaps are known and bounded |

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No broad observability platform | Compliant — single focused endpoint |
| No distributed tracing/telemetry | Compliant — no tracing infrastructure added |
| No masking real inconsistencies | Compliant — all findings documented including limitations |
| No broad storage redesign | Compliant — no schema changes; only additive read surface |
