# Family 04 — Implementation Notes

> Stage: S181 · Wave B · `risk_assessments`
> Status: Complete
> Pattern: Wave B v2 — 9-artifact template, struct DI, manual reader/handler/use-case

---

## 1. Implementation Summary

Family 04 (`risk_assessments`) was implemented following the exact Wave B v2 pattern. All 9 artifacts were completed: 3 pre-staged (migration, mapper, pipeline) and 6 new (reader, use case, contracts, handler, routes, smoke test).

**Zero creative decisions were required.** The implementation was fully mechanical, following the strategy reader (Family 03) as the closest template.

---

## 2. Artifacts Implemented

| # | Artifact | File | Status | Lines |
|---|----------|------|--------|-------|
| 1 | Schema migration | `deploy/migrations/005_create_risk_assessments.sql` | Pre-staged | — |
| 2 | Writer mapper | `cmd/writer/mappers.go` (`mapRiskRow`) | Pre-staged | — |
| 3 | Pipeline config | `cmd/writer/pipeline.go` | Pre-staged | — |
| 4 | ClickHouse reader | `internal/adapters/clickhouse/risk_reader.go` | **New** | 161 |
| 5 | Application use case | `internal/application/analyticalclient/get_risk_history.go` | **New** | 93 |
| 6 | Contract types | `internal/application/analyticalclient/contracts.go` | **Extended** | +24 |
| 7 | HTTP handler method | `internal/interfaces/http/handlers/analytical.go` | **Extended** | +98 |
| 8 | Route registration | `internal/interfaces/http/routes/analytical.go` | **Extended** | +14 |
| 9 | Smoke test | `scripts/smoke-analytical-e2e.sh` | **Extended** | +28 |

### Additional artifacts (test + integration)

| Artifact | File | Lines |
|----------|------|-------|
| Reader tests | `internal/adapters/clickhouse/risk_reader_test.go` | 211 |
| Use case tests | `internal/application/analyticalclient/get_risk_history_test.go` | 182 |
| Handler tests | `internal/interfaces/http/handlers/analytical_test.go` | +118 |
| HTTP test queries | `tests/http/analytical.http` | +45 |
| Gateway reader factory | `cmd/gateway/analytical_reader.go` | +7 |
| Gateway composition | `cmd/gateway/compose.go` | +2 |

---

## 3. New JSON Parsers

Two new parsers were introduced, both following the established pattern:

| Parser | Target Type | Shape | Fallback |
|--------|------------|-------|----------|
| `ParseStrategyInputsJSON` | `[]risk.StrategyInput` | Array of structs | Empty slice |
| `ParseConstraintsJSON` | `risk.Constraints` | Single struct | Zero-value struct |

`ParseConstraintsJSON` is the first struct-target parser (previous parsers target slices or maps). This is structurally simpler — just `json.Unmarshal` into a struct — and required no deviation from the pattern.

**Total parser function count after Family 04: 6** (FormatFloat, ParseMetadataJSON, ParseSignalInputsJSON, ParseDecisionInputsJSON, ParseStrategyInputsJSON, ParseConstraintsJSON).

---

## 4. Column Alignment

All 13 domain columns are consistent across DDL → mapper → reader → handler:

```
type, source, symbol, timeframe, disposition, confidence,
strategies, constraints, rationale, parameters, metadata,
final, timestamp
```

4 metadata columns (`event_id`, `occurred_at`, `correlation_id`, `causation_id`) are written by the mapper but not queried by the reader — consistent with all families.

---

## 5. `rationale` Free-Text Column

The `rationale` column was the first free-text column in the analytical layer. Its handling was the simplest of all column types:

- **Writer:** Direct string pass-through (`r.Rationale`)
- **Reader:** Standard `string` scan, no parsing
- **Handler:** Direct JSON serialization via Go's `encoding/json`

No encoding issues, no special handling, no new patterns. This confirms that free-text columns are simpler than JSON columns in the analytical layer.

---

## 6. `disposition` Enum Filter

The `disposition` filter follows the exact pattern from `outcome` (Family 02) and `direction` (Family 03):

- Optional query parameter — empty means no filter
- No validation at handler level (passthrough)
- Added as conditional `AND disposition = ?` in query builder
- Tested for passthrough in handler and query builder tests

---

## 7. Implementation Order

The implementation followed the S180-prescribed order exactly:

1. Reader adapter (`risk_reader.go` + tests) ✓
2. Contracts (extend `contracts.go`) ✓
3. Use case (`get_risk_history.go` + tests) ✓
4. Handler method (extend `analytical.go` + tests) ✓
5. Routes (extend routes `analytical.go`) ✓
6. Gateway wiring (extend `compose.go` + `analytical_reader.go`) ✓
7. Smoke test (extend `smoke-analytical-e2e.sh`) ✓
8. HTTP test queries (extend `analytical.http`) ✓

---

## 8. Simplifications Adopted

| # | Simplification | Rationale |
|---|---------------|-----------|
| 1 | No `disposition` enum validation | Pattern rule: enum filters are passthrough (consistent with `outcome`, `direction`) |
| 2 | No `rationale` content validation | Free text is written and read as-is; Go handles JSON escaping |
| 3 | No cross-family correlation | Explicit non-goal across all Wave B families |
| 4 | No pagination beyond `limit=500` | No demand at current data volumes |
