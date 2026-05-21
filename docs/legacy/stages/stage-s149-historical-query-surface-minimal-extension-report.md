# Stage S149 — Historical Query Surface Minimal Extension: Report

> **Stage:** S149
> **Status:** Complete
> **Predecessor:** S148 (Writer Service Minimal Append-Only Implementation)
> **Scope:** Expose the smallest useful historical query via gateway, backed by ClickHouse.

---

## 1. Executive Summary

S149 implements the minimal historical query surface for the Market Foundry gateway. A single analytical endpoint (`GET /analytical/evidence/candles`) queries ClickHouse for historical candle data, proving the read path from the analytical store through the gateway to HTTP consumers.

The implementation is strictly additive: no existing endpoint, configuration, or behavior is modified. ClickHouse remains optional — the gateway starts and operates normally without it. When configured, analytical endpoints activate; when ClickHouse is unavailable, they return 503 without affecting operational queries.

---

## 2. Historical Query Implemented

### Endpoint

```
GET /analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60&limit=50&since=...&until=...
```

### Parameters

| Parameter | Required | Default | Max |
|-----------|----------|---------|-----|
| source | Yes | — | — |
| symbol | Yes | — | — |
| timeframe | Yes | — | — |
| limit | No | 50 | 500 |
| since | No | 0 (unset) | — |
| until | No | 0 (unset) | — |

### Response

```json
{
  "candles": [...],
  "source": "clickhouse"
}
```

Results are ordered newest-first. The `source` field distinguishes analytical results from operational queries.

---

## 3. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/adapters/clickhouse/reader.go` | Generic `Query` method on ClickHouse Client |
| `internal/application/analyticalclient/contracts.go` | Query/Reply contracts for analytical candle history |
| `internal/application/analyticalclient/get_candle_history.go` | Use case with validation and limit clamping |
| `internal/application/analyticalclient/get_candle_history_test.go` | Use case unit tests (8 cases) |
| `internal/interfaces/http/handlers/analytical.go` | HTTP handler for analytical queries |
| `internal/interfaces/http/handlers/analytical_test.go` | Handler unit tests (5 cases) |
| `internal/interfaces/http/routes/analytical.go` | Route registration under `/analytical/` prefix |
| `cmd/gateway/analytical_reader.go` | ClickHouse → EvidenceCandle row mapper |
| `tests/http/analytical.http` | Manual test file for analytical endpoints |
| `docs/architecture/historical-query-surface-minimal-extension.md` | Architecture: what, why, how |
| `docs/architecture/operational-vs-analytical-query-boundaries.md` | Architecture: boundary rules |

### Modified Files

| File | Change |
|------|--------|
| `internal/interfaces/http/routes/core.go` | Added `Analytical` field to `Dependencies`; conditional route registration |
| `cmd/gateway/compose.go` | Added `buildAnalyticalClient()`; wired analytical use cases |
| `cmd/gateway/run.go` | Added Phase 2a: optional ClickHouse client creation |
| `deploy/configs/gateway.jsonc` | Added optional `clickhouse` configuration section |

### Unchanged Files (Verified)

| File | Status |
|------|--------|
| `cmd/gateway/readiness.go` | No ClickHouse in readiness check |
| `deploy/compose/docker-compose.yaml` | No `clickhouse` in gateway `depends_on` |
| All operational handlers and routes | Zero modifications |
| Smoke test scripts | Zero modifications |

---

## 4. Boundaries Preserved

### 4.1 Optionality Rules Compliance

| Rule | Status | Verification |
|------|--------|--------------|
| R-01: No operational service depends on ClickHouse | **PASS** | Gateway `depends_on` unchanged in docker-compose |
| R-02: No readiness check references ClickHouse (except writer) | **PASS** | `readiness.go` unchanged |
| R-03: No event path blocks on ClickHouse | **PASS** | Analytical queries are HTTP request-scoped, not event handlers |
| R-06: Smoke tests pass without ClickHouse | **PASS** | Smoke scripts unchanged; no analytical assertions |
| R-07: No conditional behavior in operational services | **PASS** | All changes are additive; existing code paths untouched |
| R-08: Historical endpoints are additive | **PASS** | New `/analytical/*` route; existing routes unchanged |

### 4.2 Operational vs. Analytical Boundary

| Dimension | Operational | Analytical |
|-----------|------------|------------|
| Route prefix | `/evidence/*`, `/signal/*`, etc. | `/analytical/*` |
| Data source | NATS KV (via store) | ClickHouse (direct) |
| In readiness check | Yes | No |
| Failure isolation | Independent | Independent |
| ClickHouse dependency | None | Required |

### 4.3 Build Verification

```
go build cmd/gateway       ✓  (includes ClickHouse driver via analytical reader)
go build cmd/writer        ✓  (unchanged)
go build cmd/store         ✓  (no ClickHouse imports)
go build cmd/ingest        ✓  (no ClickHouse imports)
go build cmd/derive        ✓  (no ClickHouse imports)
go build cmd/execute       ✓  (no ClickHouse imports)
go build cmd/configctl     ✓  (no ClickHouse imports)
go build cmd/migrate       ✓  (unchanged)
```

### 4.4 Test Results

```
go test ./internal/application/analyticalclient/...  ✓  (8 tests)
go test ./internal/interfaces/http/handlers/...      ✓  (all existing + 5 new)
go test ./internal/interfaces/http/routes/...        ✓  (all existing pass)
```

---

## 5. Remaining Limits

| Limit | Description | When to Address |
|-------|-------------|-----------------|
| Candles only | No historical queries for signals, decisions, strategies, risk, executions | When candle query is proven stable in production use |
| No aggregation | Raw rows only; no OHLCV rollups or statistical queries | When analytics scope is explicitly opened |
| No pagination | Time range + limit only; no cursor-based pagination | When single-request limits prove insufficient |
| Float64 precision | ClickHouse stores Float64; decimal precision may differ from original string | Acceptable for analytical (not settlement) use |
| No cross-symbol | Each query targets one source+symbol+timeframe triple | When cross-asset analysis is prioritized |
| No gateway-side caching | Every request hits ClickHouse | When query volume warrants caching |
| No docker-compose depends_on | Gateway doesn't wait for ClickHouse at startup | By design — analytical is optional |

---

## 6. Trade-offs Accepted

| Decision | Trade-off | Rationale |
|----------|-----------|-----------|
| Gateway imports ClickHouse driver | Binary size increase; new dependency | Architecture explicitly allows this (R-08 exception in optionality rules) |
| Direct ClickHouse connection | No intermediate service (reader/analytics) | Follows the architecture's "direct consumer" model for gateway historical endpoints |
| Optional config section | Empty section = no analytical routes | Simplest optionality mechanism; no feature flags |
| Single endpoint | Proves the pattern but doesn't deliver full historical coverage | Deliberate scope constraint per S149 objectives |

---

## 7. Preparation for S150

The following topics are natural candidates for S150 or subsequent stages:

| Topic | Readiness |
|-------|-----------|
| **Historical signal queries** | Pattern proven; add `QuerySignalHistory` reader + handler + route |
| **Historical decision queries** | Same pattern; depends on signal query working |
| **End-to-end analytical smoke test** | Needs ClickHouse in CI or a separate smoke profile |
| **Gateway → ClickHouse connection pooling** | Current single connection is sufficient for low-volume queries |
| **Analytical query timeout** | Currently uses context deadline from HTTP handler; may need explicit timeout |
| **ClickHouse availability monitoring** | Gateway logs connection status but doesn't expose it via `/statusz` |

### Recommended S150 Scope

1. Expand historical query surface to signals and decisions (two most useful after candles)
2. Add an analytical smoke test script with its own compose profile
3. Document the complete analytical query catalog
4. Review whether gateway needs a `/analytical/status` health endpoint (separate from `/readyz`)
