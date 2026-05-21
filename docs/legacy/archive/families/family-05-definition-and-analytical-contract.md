# Family 05 — Definition and Analytical Contract

> Formal definition of the Executions (paper_order) analytical family. This document freezes the contract, payload shape, endpoint specification, and boundaries before implementation (S187).

---

## 1. Family Identity

| Field | Value |
|-------|-------|
| Family | 05 |
| Domain | Executions |
| Event type | `paper_order` |
| Source event | `execution.PaperOrderSubmittedEvent` |
| Domain struct | `execution.ExecutionIntent` |
| Layer | 6 — Execution (terminal) |
| Source binary | derive |
| ClickHouse table | `executions` |
| Migration | `006_create_executions.sql` |
| NATS stream | `EXECUTION_EVENTS` |
| Pattern | Wave B v2 (9-artifact template) |

---

## 2. Analytical Contract

### 2.1 Purpose

Provide a **read-only historical query** over execution intents (paper orders) stored in ClickHouse. This endpoint answers: "What execution intents were submitted for a given source/symbol/timeframe, optionally filtered by side and status, within a time range?"

This family completes the end-to-end analytical read path:

```
Evidence → Signals → Decisions → Strategies → Risk → Executions
  (L1)      (L2)      (L3)        (L4)       (L5)     (L6)
 Baseline    F-01      F-02        F-03       F-04     F-05
```

### 2.2 Scope

- **In scope:** Historical query of `paper_order` execution intents from `executions` table.
- **Out of scope:** `venue_market_order` events, cross-family joins, real-time streaming, write-path changes.

### 2.3 Contract guarantees

| Guarantee | Detail |
|-----------|--------|
| Read-only | No writes, no mutations, no side effects |
| Additive | No existing endpoint, handler, route, or reader modified |
| Immutable write path | Writer mapper, pipeline config, NATS consumer untouched (6th consecutive) |
| Graceful degradation | If ClickHouse unavailable or reader not configured, returns HTTP 503 |
| Consistent observability | Same instrumentation as 5 predecessors (wall-clock timing, QueryMeta, Server-Timing header) |

---

## 3. Payload Shape

### 3.1 Response payload

The response follows the established analytical contract. The `data` array contains `ExecutionIntent` domain objects serialized to JSON:

```json
{
  "executions": [
    {
      "type": "paper_order",
      "source": "derive",
      "symbol": "BTCUSDT",
      "timeframe": 60,
      "side": "buy",
      "quantity": "0.001",
      "filled_quantity": "0.001",
      "status": "filled",
      "risk": {
        "type": "position_exposure",
        "disposition": "approved",
        "confidence": "0.85",
        "timeframe": 60
      },
      "fills": [
        {
          "price": "67500.00",
          "quantity": "0.001",
          "fee": "0.00",
          "simulated": true,
          "timestamp": "2026-03-20T10:01:00Z"
        }
      ],
      "parameters": { "strategy": "mean_reversion_entry" },
      "metadata": { "version": "1" },
      "correlation_id": "corr-123",
      "causation_id": "cause-456",
      "final": true,
      "timestamp": "2026-03-20T10:01:00Z"
    }
  ],
  "source": "analytical/clickhouse",
  "meta": {
    "query_ms": 15,
    "row_count": 1
  }
}
```

### 3.2 Domain types involved

From `internal/domain/execution/execution.go`:

| Type | Go definition | JSON representation |
|------|--------------|-------------------|
| `ExecutionIntent` | struct | Top-level object in response array |
| `Side` | `string` enum (`"buy"`, `"sell"`, `"none"`) | String value |
| `Status` | `string` enum (`"submitted"`, `"sent"`, `"accepted"`, `"filled"`, `"partially_filled"`, `"rejected"`, `"cancelled"`) | String value |
| `RiskInput` | struct with 4 fields | Nested JSON object |
| `FillRecord` | struct with 5 fields (`price`, `quantity`, `fee`, `simulated`, `timestamp`) | Array of JSON objects |

### 3.3 Column-to-field mapping (DDL → Domain)

| DDL column | Type | Domain field | Go type | JSON parser needed |
|------------|------|--------------|---------|--------------------|
| event_id | String | *(not in response — internal)* | — | — |
| occurred_at | DateTime64(3) | *(not in response — internal)* | — | — |
| correlation_id | String | *(event metadata — not domain)* | — | — |
| causation_id | String | *(event metadata — not domain)* | — | — |
| type | LowCardinality(String) | `ExecutionIntent.Type` | `string` | No |
| source | LowCardinality(String) | `ExecutionIntent.Source` | `string` | No |
| symbol | LowCardinality(String) | `ExecutionIntent.Symbol` | `string` | No |
| timeframe | UInt32 | `ExecutionIntent.Timeframe` | `int` (via `uint32` scan) | No |
| side | LowCardinality(String) | `ExecutionIntent.Side` | `Side` (string cast) | No |
| quantity | Float64 | `ExecutionIntent.Quantity` | `string` (via `FormatFloat`) | No (reuse `FormatFloat`) |
| filled_quantity | Float64 | `ExecutionIntent.FilledQuantity` | `string` (via `FormatFloat`) | No (reuse `FormatFloat`) |
| status | LowCardinality(String) | `ExecutionIntent.Status` | `Status` (string cast) | No |
| risk | String (JSON) | `ExecutionIntent.Risk` | `RiskInput` | **Yes — `ParseRiskInputJSON`** |
| fills | String (JSON) | `ExecutionIntent.Fills` | `[]FillRecord` | **Yes — `ParseFillsJSON`** |
| parameters | String (JSON) | `ExecutionIntent.Parameters` | `map[string]string` | No (reuse `ParseMetadataJSON`) |
| metadata | String (JSON) | `ExecutionIntent.Metadata` | `map[string]string` | No (reuse `ParseMetadataJSON`) |
| exec_correlation_id | String | `ExecutionIntent.CorrelationID` | `string` | No |
| exec_causation_id | String | `ExecutionIntent.CausationID` | `string` | No |
| final | Bool | `ExecutionIntent.Final` | `bool` | No |
| timestamp | DateTime64(3) | `ExecutionIntent.Timestamp` | `time.Time` | No |

### 3.4 SELECT column list (reader query)

The reader SELECT must match the domain fields returned — event metadata columns (`event_id`, `occurred_at`, `correlation_id`, `causation_id`) and ingestion metadata (`ingested_at`) are **excluded** from the SELECT, consistent with all prior families.

```sql
SELECT type, source, symbol, timeframe, side, quantity, filled_quantity, status,
       risk, fills, parameters, metadata,
       exec_correlation_id, exec_causation_id, final, timestamp
FROM executions
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
  [AND side = ?]
  [AND status = ?]
  [AND timestamp >= ?]
  [AND timestamp <= ?]
ORDER BY timestamp DESC
LIMIT ?
```

**16 columns in SELECT** (vs 13 for risk). Scan order matches SELECT order exactly.

---

## 4. Endpoint Specification

### 4.1 Route

```
GET /analytical/execution/history
```

### 4.2 Query parameters

| Parameter | Required | Type | Validation | Example |
|-----------|----------|------|------------|---------|
| `type` | Yes | string | Non-empty | `paper_order` |
| `source` | Yes | string | Non-empty | `derive` |
| `symbol` | Yes | string | Non-empty | `BTCUSDT` |
| `timeframe` | Yes | integer | Positive | `60` |
| `side` | No | string | Pass-through (no validation) | `buy` |
| `status` | No | string | Pass-through (no validation) | `filled` |
| `since` | No | int64 | Valid unix timestamp | `1710892800` |
| `until` | No | int64 | Valid unix timestamp, ≥ since | `1710979200` |
| `limit` | No | integer | 1–500, default 50 | `100` |

### 4.3 Response codes

| Code | Condition |
|------|-----------|
| 200 | Success — always includes `source`, `meta.query_ms`, `meta.row_count` |
| 400 | Missing required param, invalid limit, invalid timestamp, since > until |
| 503 | ClickHouse unavailable or execution reader not configured |

### 4.4 Response headers

| Header | Format | Example |
|--------|--------|---------|
| `Content-Type` | `application/json` | — |
| `Server-Timing` | `total;dur=N, query;dur=M` | `total;dur=18, query;dur=12` |

---

## 5. New Parsers Required

Two new JSON parsers are needed. Both follow the established pattern (empty check → `json.Unmarshal` → fallback to zero-value):

### 5.1 ParseRiskInputJSON

```go
// ParseRiskInputJSON deserializes a JSON string into execution.RiskInput.
// Returns a zero-value RiskInput on parse failure.
func ParseRiskInputJSON(raw string) execution.RiskInput {
    if raw == "" || raw == "{}" {
        return execution.RiskInput{}
    }
    var r execution.RiskInput
    if err := json.Unmarshal([]byte(raw), &r); err != nil {
        return execution.RiskInput{}
    }
    return r
}
```

**Pattern precedent:** Identical shape to `ParseConstraintsJSON` (struct target, Family 04).

### 5.2 ParseFillsJSON

```go
// ParseFillsJSON deserializes a JSON string into []execution.FillRecord.
// Returns an empty slice on parse failure.
func ParseFillsJSON(raw string) []execution.FillRecord {
    if raw == "" || raw == "[]" || raw == "{}" {
        return []execution.FillRecord{}
    }
    var fills []execution.FillRecord
    if err := json.Unmarshal([]byte(raw), &fills); err != nil {
        return []execution.FillRecord{}
    }
    return fills
}
```

**Pattern precedent:** Identical shape to `ParseStrategyInputsJSON` (slice target, Family 04).

### 5.3 Parser count trajectory

| State | Count | Parsers |
|-------|-------|---------|
| Pre-Family-05 | 6 | FormatFloat, ParseMetadataJSON, ParseSignalInputsJSON, ParseDecisionInputsJSON, ParseStrategyInputsJSON, ParseConstraintsJSON |
| Post-Family-05 | **8** | + ParseRiskInputJSON, ParseFillsJSON |
| Threshold | ≤8 healthy, >8 concerning | At threshold — generic `parseJSON[T]` recommended if Family 06 adds more |

---

## 6. Contracts (Application Layer)

### 6.1 Query contract

```go
type ExecutionHistoryQuery struct {
    Type      string `json:"type"`                // execution type (e.g., "paper_order")
    Source    string `json:"source"`
    Symbol    string `json:"symbol"`
    Timeframe int    `json:"timeframe"`
    Side      string `json:"side,omitempty"`      // optional side filter
    Status    string `json:"status,omitempty"`     // optional status filter
    Limit     int    `json:"limit"`
    Since     int64  `json:"since,omitempty"`      // unix seconds, inclusive lower bound (0 = unset)
    Until     int64  `json:"until,omitempty"`      // unix seconds, inclusive upper bound (0 = unset)
}
```

**Note:** This is the first query contract with **two** optional filters. Prior families have 0 (candles) or 1 (signals: none, decisions: outcome, strategies: direction, risk: disposition).

### 6.2 Reply contract

```go
type ExecutionHistoryReply struct {
    Executions []execution.ExecutionIntent `json:"executions"`
    Source     string                      `json:"source"`     // always "clickhouse"
    Meta       QueryMeta                   `json:"meta"`
}
```

### 6.3 Reader interface

```go
type ExecutionReader interface {
    QueryExecutionHistory(ctx context.Context, execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error)
}
```

**Signature note:** 10 parameters (vs 8 for risk reader). The two additional parameters are `side` and `status` optional filters. This is the widest reader signature yet.

---

## 7. Instrumentation

Identical to all prior families — no new instrumentation patterns:

| Layer | Instrumentation | Detail |
|-------|----------------|--------|
| Reader adapter | Wall-clock timing | `time.Since(start)` around query execution |
| Reader adapter | Structured logging | Debug on success (type, source, symbol, timeframe, side, status, rows, elapsed_ms), Error on failure |
| Use case | Timing passthrough | `elapsed.Milliseconds()` into `QueryMeta.QueryMs` |
| Use case | Structured logging | Info on success, Warn on failure |
| Handler | Server-Timing header | `total;dur=N, query;dur=M` |
| Handler | Structured logging | Warn on failure with all params |

---

## 8. Smoke Test Extension

Add execution family to the existing smoke test (`scripts/smoke-analytical-e2e.sh`):

```bash
validate_analytical_family "execution" "history" \
    "type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60" \
    "executions" "type" "source" "symbol" "timeframe" "side" "quantity" \
    "filled_quantity" "status" "final" "timestamp"
```

**Estimated addition:** ~8–10 lines (validation call + optional filter checks for `side` and `status`).

### Smoke test scope

| Check | Method |
|-------|--------|
| Endpoint returns 200 | HTTP GET with valid params |
| Response has `executions` array | JSON field presence |
| Response has `source` = `"analytical/clickhouse"` | Field value check |
| Response has `meta.query_ms` and `meta.row_count` | Field presence |
| Domain fields present | `type`, `source`, `symbol`, `timeframe`, `side`, `quantity`, `filled_quantity`, `status`, `final`, `timestamp` |
| Optional filter works | `?side=buy` returns subset or empty |
| Missing required param returns 400 | Omit `source` |

---

## 9. What This Contract Is Sufficient For

This contract is designed to measure the manual pattern's ceiling. It is **sufficient** because:

1. **Largest schema.** 20 DDL columns, 16 SELECT columns, 4 JSON columns. If the pattern absorbs this mechanically, schema complexity is a non-issue.

2. **New column types.** Float64 (quantity, filled_quantity) and Bool (final) are first occurrences in the read path. FormatFloat reuse and direct bool scan are expected — any friction is a signal.

3. **Two new parsers.** ParseRiskInputJSON (struct target) and ParseFillsJSON (slice target). Both follow proven shapes. Parser count goes from 6 to 8 (at threshold).

4. **Two optional filters.** First handler method with two optional WHERE clauses (side, status). Prior maximum was one. Additive WHERE clauses should have no interaction — any interaction is a signal.

5. **Handler file at boundary.** Current: 515 lines. Expected post-expansion: ~595–615 lines. The 620-line hard ceiling tests whether the handler can absorb one more family without extraction.

6. **Pattern convergence.** All other dimensions (struct DI, route registration, gateway wiring, smoke extension, test structure) are proven 5 times. The only novelty is in the dimensions listed above.

---

## 10. What This Contract Does NOT Cover

1. **`venue_market_order` events.** Only `paper_order` is in scope. Venue fills require a separate gate.
2. **Cross-family queries.** No execution-to-risk or pipeline trace joins.
3. **Aggregation or analytics.** Raw row-level data only — no SUM, AVG, GROUP BY.
4. **Pagination.** Hard cap at `limit=500`. No cursor-based pagination.
5. **Filter validation.** `side` and `status` values are not validated against known enums. Invalid values return empty results (consistent with `outcome`, `direction`, `disposition` in prior families).
6. **Write-path changes.** Writer mapper, pipeline, NATS consumer are pre-staged and untouched.
7. **Codegen.** No template generation during Family 05. Codegen is a post-implementation obligation.
8. **Handler refactoring.** No extraction unless handler exceeds 620 lines.
