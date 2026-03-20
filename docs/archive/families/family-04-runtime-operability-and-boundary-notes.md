# Family 04 — Runtime, Operability, and Boundary Notes

> Stage: S181 · Wave B · `risk_assessments`
> Status: Complete

---

## 1. Runtime Activation

### 1.1 ClickHouse Optionality Preserved

Risk reader activation follows the same conditional path as all previous families:

```
gateway startup
  → buildAnalyticalClient() → chClient (nil if ClickHouse not configured)
  → if chClient != nil: create risk reader, wire use case, register route
  → if chClient == nil: /analytical/risk/history returns 503
```

No new activation logic. No new config keys required.

### 1.2 Writer Pipeline

The risk pipeline is pre-staged and activates via the existing `IsRiskFamilyEnabled("position_exposure")` check in `cmd/writer/pipeline.go`. No writer changes were made in S181.

### 1.3 Migration

`deploy/migrations/005_create_risk_assessments.sql` is pre-staged and applied by `cmd/migrate` during normal startup. The smoke test verifies table existence.

---

## 2. Observability

### 2.1 Existing Instrumentation (No Changes)

| Signal | Location | Purpose |
|--------|----------|---------|
| `Server-Timing` header | Handler response | `total;dur=X, query;dur=Y` — wall-clock and ClickHouse timing |
| `QueryMeta` in JSON response | Response body | `query_ms` and `row_count` per request |
| `slog.Error` on query failure | Reader adapter | Structured log with risk_type, source, symbol, timeframe, error |
| `slog.Warn` on use case failure | Use case | Structured log with elapsed_ms and problem code |
| `slog.Info` on success | Use case | Row count and query timing |
| `slog.Debug` on query complete | Reader adapter | Detailed timing for debugging |

### 2.2 Writer Observability

Writer-side observability for risk assessments is pre-staged:
- Tracker visible in `/statusz` and `/diagz` endpoints
- Event count, flush count, error count, degradation state all tracked
- No new observability was needed

---

## 3. Boundary Review

### 3.1 Component Boundaries

| Component | Boundary | Verified |
|-----------|----------|----------|
| Reader → Use Case | `RiskReader` interface in `contracts.go` | ✓ Strict contract |
| Use Case → Handler | `getAnalyticalRiskHistoryUseCase` private interface | ✓ Handler-local |
| Handler → Route | `AnalyticalFamilyDeps.GetRiskHistory` field | ✓ Struct DI |
| Route → Gateway | `analyticalclient.RiskReader` interface | ✓ Adapter boundary |

### 3.2 No Cross-Reader Dependencies

The risk reader is fully self-contained. It uses two shared utility functions (`FormatFloat`, `ParseMetadataJSON`) that are stateless and shared across all readers. No coupling between readers.

### 3.3 Import Graph

```
cmd/gateway → internal/application/analyticalclient (RiskReader interface)
cmd/gateway → internal/adapters/clickhouse (NewRiskReader concrete)
internal/adapters/clickhouse → internal/domain/risk
internal/application/analyticalclient → internal/domain/risk
internal/interfaces/http/handlers → internal/application/analyticalclient (contracts only)
```

No circular dependencies. No new import paths beyond the expected pattern.

---

## 4. Graceful Degradation

| Scenario | Behavior | HTTP Response |
|----------|----------|--------------|
| ClickHouse not configured | Route not registered, handler nil | 503 |
| ClickHouse down at query time | Reader returns error, use case wraps as problem | 503 |
| Empty results | Empty array returned | 200 with `[]` |
| Missing required params | Validation in handler | 400 |

All degradation paths are identical to existing families.

---

## 5. Ceiling Test Measurements

| Measurement | Value | Threshold | Status |
|-------------|-------|-----------|--------|
| Handler file size | **515 lines** | <550 healthy, 550-600 concerning, >600 critical | **Healthy** |
| Reader file size | **161 lines** | <150 healthy, 150-180 concerning, >180 critical | **Concerning** |
| New frictions | **0** | ≤2 | **Healthy** |
| Creative decisions | **0** | 0 | **Healthy** |
| JSON parser count | **6 total** | ≤6 healthy, 7-8 concerning | **Healthy (at limit)** |
| Test count (Family 04) | **~32 new** | 27±5 | **Healthy** |

### 5.1 Interpretation

- **Handler file (515 lines):** Comfortably under the 550-line healthy threshold. One more family (~80 lines) would push it to ~595, entering the "concerning" zone. DEF-C3 (handler split) is confirmed as necessary before Family 06.

- **Reader file (161 lines):** Slightly above the 150-line "healthy" threshold, entering "concerning." The 13-column scan and 4 JSON parsers contributed ~20 lines more than the Strategy reader (138 lines). This is proportional to the added complexity and not a friction.

- **JSON parser count (6):** At the healthy upper limit. Family 05 (executions) has simpler JSON columns and may not need new parsers. The codegen trigger (D-4) remains justified but non-urgent.

### 5.2 Gate Output

> **The Wave B manual expansion pattern can sustain at least one more family (executions) without structural intervention.** Codegen remains justified but not blocking before Family 05. Handler split (DEF-C3) must execute before Family 06.

---

## 6. Runbook Entry

### `/analytical/risk/history` — Operational Notes

**Endpoint:** `GET /analytical/risk/history?type={riskType}&source={source}&symbol={symbol}&timeframe={timeframe}[&disposition={disposition}][&since={unix}][&until={unix}][&limit={n}]`

**Check if working:**
```bash
curl -s "http://localhost:8080/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60&limit=1" | python3 -m json.tool
```

**Expected healthy response:** HTTP 200, `source: "clickhouse"`, `risk_assessments` array (may be empty if no data yet).

**Common issues:**
- **503:** ClickHouse not configured or down. Check `deploy/configs/gateway.jsonc` and ClickHouse connectivity.
- **400:** Missing required params (type, source, symbol, timeframe) or invalid limit.
- **Empty results:** Pipeline may not have produced risk assessment events yet. Check writer `/statusz` for the position_exposure tracker.

**Filters:**
- `disposition`: `approved`, `modified`, `rejected` — optional, case-sensitive.
- `since`/`until`: Unix seconds, inclusive bounds.
- `limit`: Default 50, max 500.
