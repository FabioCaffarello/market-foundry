# Wave B Family 02 — Decisions (RSI Oversold): Implementation Notes

## Overview

This document records the implementation details and technical decisions made during the S169 stage — the second controlled Wave B family expansion, implementing Decisions (RSI Oversold) end-to-end.

## Schema Coherence Table

| Column     | DDL Type               | Writer Type   | Reader Type   | Domain Type            | Status   |
|-----------|------------------------|---------------|---------------|------------------------|----------|
| type       | LowCardinality(String) | string        | string        | string                 | VERIFIED |
| source     | LowCardinality(String) | string        | string        | string                 | VERIFIED |
| symbol     | LowCardinality(String) | string        | string        | string                 | VERIFIED |
| timeframe  | UInt32                 | uint32        | uint32        | int                    | VERIFIED |
| outcome    | LowCardinality(String) | string        | string        | decision.Outcome       | VERIFIED |
| confidence | Float64                | float64       | float64       | string (FormatFloat)   | VERIFIED |
| signals    | String                 | JSON-string   | JSON-string   | []decision.SignalInput | VERIFIED |
| metadata   | String                 | JSON-string   | JSON-string   | map[string]string      | VERIFIED |
| final      | Bool                   | bool          | bool          | bool                   | VERIFIED |
| timestamp  | DateTime64(3)          | time.Time     | time.Time     | time.Time              | VERIFIED |

### Event metadata columns (write-only, not exposed in read path)

| Column         | DDL Type      | Writer Type | Read Path |
|---------------|---------------|-------------|-----------|
| event_id       | String        | string      | NOT READ  |
| occurred_at    | DateTime64(3) | time.Time   | NOT READ  |
| correlation_id | String        | string      | NOT READ  |
| causation_id   | String        | string      | NOT READ  |
| ingested_at    | DateTime64(3) | DEFAULT     | NOT READ  |

## Data Flow

```
NATS JetStream
  → writerConsumer (cmd/writer/consumer.go — existing, unchanged)
  → mapDecisionRow() (cmd/writer/mappers.go — existing, unchanged)
  → INSERT INTO decisions (cmd/writer/inserter.go — existing, unchanged)
  → decisions table (ClickHouse — migration 003, pre-existing)
  → DecisionReader.QueryDecisionHistory() (internal/adapters/clickhouse/decision_reader.go — NEW)
  → GetDecisionHistoryUseCase.Execute() (internal/application/analyticalclient/get_decision_history.go — NEW)
  → GET /analytical/decision/history (internal/interfaces/http/handlers/analytical.go — EXTENDED)
```

## Endpoint Specification

```
GET /analytical/decision/history

Required parameters:
  type       string   Decision type (e.g., "rsi_oversold")
  source     string   Data source (e.g., "binancef")
  symbol     string   Trading pair (e.g., "btcusdt")
  timeframe  int      Timeframe in seconds (e.g., 60)

Optional parameters:
  outcome    string   Filter by outcome ("triggered", "not_triggered", "insufficient")
  limit      int      Max results (1–500, default 50)
  since      int64    Unix seconds, inclusive lower bound (0 = unset)
  until      int64    Unix seconds, inclusive upper bound (0 = unset)

Response: 200 OK
{
  "decisions": [...],
  "source": "clickhouse",
  "meta": { "query_ms": N, "row_count": N }
}

Headers:
  Server-Timing: total;dur=N, query;dur=N

Errors:
  400  Missing/invalid required params, limit out of range, since > until
  503  ClickHouse unavailable or decision reader not configured
```

## Artifacts Produced (9-Artifact Pattern)

| # | Artifact                       | File                                                     | Status |
|---|-------------------------------|----------------------------------------------------------|--------|
| 1 | Migration DDL                 | deploy/migrations/003_create_decisions.sql               | PRE-EXISTING |
| 2 | Writer mapper                 | cmd/writer/mappers.go (mapDecisionRow)                   | PRE-EXISTING |
| 3 | Writer pipeline entry         | cmd/writer/pipeline.go (rsi_oversold)                    | PRE-EXISTING |
| 4 | Reader adapter                | internal/adapters/clickhouse/decision_reader.go          | NEW |
| 5 | Application use case + contracts | internal/application/analyticalclient/get_decision_history.go, contracts.go | NEW |
| 6 | HTTP handler + route          | internal/interfaces/http/handlers/analytical.go, routes/analytical.go | EXTENDED |
| 7 | Gateway composition           | cmd/gateway/analytical_reader.go, compose.go             | EXTENDED |
| 8 | Integration test (HTTP)       | tests/http/analytical.http                               | EXTENDED |
| 9 | Smoke-analytical-e2e section  | scripts/smoke-analytical-e2e.sh (Phase 5c)               | EXTENDED |

## Complexity Delta vs Family 01 (Signals)

| Dimension            | Signals (F-01)   | Decisions (F-02)         | Delta       |
|---------------------|------------------|--------------------------|-------------|
| Domain columns       | 8                | 10                       | +2          |
| JSON columns         | 1 (metadata)     | 2 (signals, metadata)    | +1          |
| Enum-like columns    | 0                | 1 (outcome)              | +1          |
| Family-specific param| 0                | 1 (outcome filter)       | +1          |
| Query builder args   | 7                | 8                        | +1          |
| Constructor args     | 3                | 3 (at AnalyticalWebHandler level) | 0 → now 4 total |

## JSON Deserialization Notes

### signals column ([]SignalInput)

The `signals` column stores a JSON array of `SignalInput` structs. The read path uses `ParseSignalInputsJSON()` which:
- Returns `[]decision.SignalInput{}` for empty string, `[]`, or `{}`
- Gracefully falls back to empty slice on invalid JSON
- Mirrors the write-path `marshalJSON()` fallback strategy

This is the first time the read path deserializes a JSON array (not just a map). No friction was observed — `json.Unmarshal` handles both cases identically.

### metadata column (map[string]string)

Reuses the existing `ParseMetadataJSON()` from the signal reader. No changes needed.

### confidence column (Float64 → string)

The DDL stores confidence as `Float64`. The writer converts from `string` via `parseFloat()`. The reader converts back via `FormatFloat()`. Round-trip preserves reasonable precision.

## Known Limits

1. **Outcome filtering is case-sensitive** — the query passes `outcome` as-is to ClickHouse WHERE clause. Domain values are lowercase by convention ("triggered", "not_triggered", "insufficient").
2. **No aggregation** — the endpoint returns raw decision events, not aggregated statistics.
3. **No signal drill-down** — the `signals` array is returned as-is; no cross-family join to the signals table.
4. **Constructor growth** — `NewAnalyticalWebHandler` now takes 4 positional arguments (candle, signal, decision, logger). Documented as H-1 hardening pre-commitment for Family 03.
