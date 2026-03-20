# Historical Query Surface: Minimal Extension

> **Stage:** S149 — Historical Query Surface Minimal Extension
> **Status:** Definitive
> **Scope:** Defines the smallest useful historical query surface exposed via gateway, backed by ClickHouse.

---

## 1. Purpose

This document defines what the minimal historical query surface is, why it was chosen, and how it integrates with the existing gateway without breaking operational boundaries.

The operational pipeline already serves latest-value queries from NATS KV. The analytical layer (ClickHouse) holds historical data written by the writer service. This stage exposes the smallest useful read path from ClickHouse through the gateway.

---

## 2. Minimal Query Surface

### 2.1 Single Endpoint

| Method | Path | Source | Description |
|--------|------|--------|-------------|
| GET | `/analytical/evidence/candles` | ClickHouse | Historical candle query with time range and limit |

### 2.2 Query Parameters

| Parameter | Type | Required | Default | Constraints |
|-----------|------|----------|---------|-------------|
| `source` | string | Yes | — | Exchange identifier (e.g., `binancef`) |
| `symbol` | string | Yes | — | Instrument symbol (e.g., `btcusdt`) |
| `timeframe` | int | Yes | — | Window duration in seconds (e.g., `60`) |
| `since` | int64 | No | 0 (unset) | Unix seconds, inclusive lower bound |
| `until` | int64 | No | 0 (unset) | Unix seconds, inclusive upper bound |
| `limit` | int | No | 50 | Range: 1–500 |

### 2.3 Response Format

```json
{
  "candles": [
    {
      "source": "binancef",
      "symbol": "btcusdt",
      "timeframe": 60,
      "open": "100.50",
      "high": "101.20",
      "low": "99.80",
      "close": "100.90",
      "volume": "1234.56",
      "trade_count": 42,
      "open_time": "2026-03-19T10:00:00Z",
      "close_time": "2026-03-19T10:01:00Z",
      "final": true
    }
  ],
  "source": "clickhouse"
}
```

The `source` field in the response is always `"clickhouse"`, distinguishing analytical results from operational queries which serve from NATS KV.

### 2.4 Result Ordering

Results are returned newest-first (descending by `open_time`).

---

## 3. Why This Is the Minimum

### 3.1 Candles Are the Foundation

Every other domain layer (signals, decisions, strategies, risk, executions) derives from evidence candles. A historical candle query is the single most useful analytical query because:

- It enables visual inspection of market data windows
- It enables basic backtesting verification
- It validates the entire write path (ingest → NATS → writer → ClickHouse)
- It proves the read path works (gateway → ClickHouse → HTTP response)

### 3.2 What Is Explicitly Not Included

| Excluded | Reason |
|----------|--------|
| Historical signals, decisions, strategies, risk, executions | Premature — candle query proves the pattern; others follow the same shape |
| Aggregation queries (OHLCV rollups, averages) | Analytics scope — not a basic historical query |
| Cross-symbol queries | Multi-key queries add complexity without proving the pattern |
| Pagination (cursor/offset) | Limit + time range is sufficient for the minimal surface |
| Real-time streaming from ClickHouse | Streaming is a different concern entirely |

### 3.3 Extension Path

Once the candle query proves stable, adding historical queries for other domains follows the same pattern:

1. Add a reader method to the ClickHouse client (or analytical reader)
2. Add contracts in `analyticalclient/`
3. Add use case
4. Add handler method
5. Add route in `routes/analytical.go`
6. Wire in `compose.go`

Each addition is additive and independent.

---

## 4. Gateway Integration

### 4.1 ClickHouse Connection

The gateway creates an optional ClickHouse client during composition:

```
Phase 1:  NATS gateway connections (existing — unchanged)
Phase 2a: Optional ClickHouse client (new — no-op if unconfigured)
Phase 2b: Wire use cases including analytical (new)
Phase 3:  Assemble routes and spawn gateway actor (existing — unchanged)
```

### 4.2 Configuration

Gateway config gains an optional `clickhouse` section:

```jsonc
{
  // ... existing config unchanged ...
  "clickhouse": {
    "addr": "clickhouse:9000",
    "database": "default",
    "username": "default",
    "password": "clickhouse"
  }
}
```

When `addr` is empty or the section is absent, analytical endpoints are not registered. The gateway starts normally with only operational endpoints.

### 4.3 No Readiness Impact

The gateway `/readyz` endpoint does **not** check ClickHouse. This preserves:

- **R-02:** No readiness check references ClickHouse except writer
- **R-07:** No conditional behavior in operational services (analytical routes are additive, not conditional modifications)

### 4.4 Failure Behavior

| ClickHouse State | Analytical Endpoint Behavior | Operational Endpoints |
|-----------------|-----------------------------|-----------------------|
| Not configured | Routes not registered (404) | Unchanged |
| Configured + healthy | Normal: returns candle data | Unchanged |
| Configured + unreachable | Returns 503 | Unchanged |

---

## 5. Implementation Architecture

### 5.1 New Components

```
internal/adapters/clickhouse/reader.go       — Query method on Client (generic)
internal/application/analyticalclient/       — Contracts + use case
internal/interfaces/http/handlers/analytical.go — HTTP handler
internal/interfaces/http/routes/analytical.go   — Route registration
cmd/gateway/analytical_reader.go             — ClickHouse → EvidenceCandle mapper
```

### 5.2 Data Flow

```
HTTP request
  → AnalyticalWebHandler.GetCandleHistory()
    → GetCandleHistoryUseCase.Execute()
      → analyticalCandleReader.QueryCandleHistory()
        → clickhouse.Client.Query()
          → ClickHouse SELECT evidence_candles
        ← Rows → scan → []EvidenceCandle
      ← []EvidenceCandle
    ← CandleHistoryReply{Candles, Source: "clickhouse"}
  ← JSON response
```

### 5.3 SQL Query

```sql
SELECT source, symbol, timeframe, open, high, low, close, volume,
       trade_count, open_time, close_time, final
FROM evidence_candles
WHERE source = ? AND symbol = ? AND timeframe = ?
  AND open_time >= ?  -- when since > 0
  AND open_time <= ?  -- when until > 0
ORDER BY open_time DESC
LIMIT ?
```

The query uses the table's primary key ordering `(source, symbol, timeframe, open_time)` which ensures efficient index utilization.

---

## 6. Limits and Constraints

| Constraint | Value | Rationale |
|-----------|-------|-----------|
| Max limit per request | 500 | Prevents unbounded result sets |
| Default limit | 50 | Reasonable default for inspection |
| TTL | 90 days (ClickHouse table TTL) | Historical window is bounded by table definition |
| Precision | Float64 → string conversion | Analytical layer uses Float64; response converts back to decimal string |
| No aggregation | Raw rows only | Aggregation is analytics scope, not historical query scope |

---

## 7. Optionality Compliance

| Rule | Status | Evidence |
|------|--------|----------|
| R-01 | Compliant | Gateway does not depend on ClickHouse in docker-compose |
| R-02 | Compliant | Gateway `/readyz` does not check ClickHouse |
| R-03 | Compliant | No event path blocks on ClickHouse |
| R-06 | Compliant | Smoke tests do not test analytical endpoints |
| R-07 | Compliant | No conditional behavior in existing operational code paths |
| R-08 | Compliant | `/analytical/evidence/candles` is a new route, not a modification |

---

## 8. Trade-offs

| Decision | Trade-off |
|----------|-----------|
| Gateway imports ClickHouse driver | Adds ClickHouse dependency to gateway binary — accepted because the architecture explicitly allows this for historical endpoints (R-08 exception) |
| Float64 precision loss | ClickHouse stores prices as Float64; converting back to string may lose trailing precision — acceptable for analytical (not settlement) use |
| No pagination | Time range + limit is sufficient for the minimal surface; pagination adds complexity |
| Single endpoint | Only candles — proving the pattern before expanding to other domains |
