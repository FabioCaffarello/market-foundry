# Wave B Family 02 — Decisions — Schema, Writer, Reader, Gateway Scope

**Stage:** S168
**Family:** Decisions (RSI Oversold)
**Predecessor:** Family 01 (Signals/RSI)

---

## 1. Data Flow

```
NATS JetStream (decision.events.rsi_oversold.evaluated.<symbol>)
  → writerConsumer → mapDecisionRow() → INSERT INTO decisions
  → decisions MergeTree table
  → DecisionReader.QueryDecisionHistory() ← GetDecisionHistoryUseCase
  → GET /analytical/decision/history ← AnalyticalWebHandler.GetDecisionHistory()
  → JSON response with Server-Timing header
```

**Write path:** Already active — zero changes required. `mapDecisionRow()` exists, pipeline entry exists, consumer is consuming.

**Read path:** New — must be built following the Signal reader pattern.

---

## 2. Schema (Existing — Migration 003)

Table `decisions` already exists via `deploy/migrations/003_create_decisions.sql`.

### 2.1 Column Mapping: DDL → Writer → Reader

| # | Column | DDL Type | Writer Source | In SELECT | Reader Go Type |
|---|--------|----------|---------------|-----------|----------------|
| 1 | event_id | String | m.ID | No | — |
| 2 | occurred_at | DateTime64(3) | m.OccurredAt | No | — |
| 3 | correlation_id | String | m.CorrelationID | No | — |
| 4 | causation_id | String | m.CausationID | No | — |
| 5 | type | LowCardinality(String) | d.Type | **Yes** | string |
| 6 | source | LowCardinality(String) | d.Source | **Yes** | string |
| 7 | symbol | LowCardinality(String) | d.Symbol | **Yes** | string |
| 8 | timeframe | UInt32 | uint32(d.Timeframe) | **Yes** | int |
| 9 | outcome | LowCardinality(String) | string(d.Outcome) | **Yes** | decision.Outcome |
| 10 | confidence | Float64 | parseFloat(d.Confidence) | **Yes** | float64 → string |
| 11 | signals | String | marshalJSON(d.Signals) | **Yes** | []decision.SignalInput |
| 12 | metadata | String | marshalJSON(d.Metadata) | **Yes** | map[string]string |
| 13 | final | Bool | d.Final | **Yes** | bool |
| 14 | timestamp | DateTime64(3) | d.Timestamp | **Yes** | time.Time |
| 15 | ingested_at | DateTime64(3) | DEFAULT now64(3) | No | — |

**SELECT columns:** 10 domain columns (type, source, symbol, timeframe, outcome, confidence, signals, metadata, final, timestamp).

**Event metadata columns (4):** Written for provenance but NOT exposed in the analytical read path — consistent with candles and signals.

### 2.2 Schema Coherence Assertions

The following must be verified in reader unit tests:

1. Query builder SELECT list contains exactly 10 columns.
2. Column names match DDL domain fields exactly.
3. Column order in scan matches SELECT order.
4. `signals` JSON deserialization produces `[]SignalInput` or empty slice.
5. `metadata` JSON deserialization produces `map[string]string` or empty map.
6. `confidence` is read as Float64 and converted to string for domain type compatibility.
7. `outcome` is cast to `decision.Outcome` type.

---

## 3. Writer Path (No Changes)

The writer path is fully operational:

- **Mapper:** `mapDecisionRow()` in `cmd/writer/mappers.go` (lines 66–88)
- **Pipeline entry:** Already registered in writer catalog for `rsi_oversold` family
- **Consumer:** Durable `writer-decision-rsi-oversold` on `decision.events.rsi_oversold.evaluated.>`
- **Batch settings:** 1000 rows / 5s flush / 10000 max pending / 5 retries

**Assertion:** S169 implementation MUST NOT modify any writer code. If writer changes are needed, the iteration stops and escalates.

---

## 4. Reader Path (New — 9 Artifacts)

### 4.1 Artifact 1: Reader Adapter

**File:** `internal/adapters/clickhouse/decision_reader.go`

```go
type DecisionReader struct {
    client *Client
    logger *slog.Logger
}

func NewDecisionReader(client *Client, logger *slog.Logger) *DecisionReader

func (r *DecisionReader) QueryDecisionHistory(
    ctx context.Context,
    decisionType, source, symbol string,
    timeframe int,
    outcome string, // optional filter — empty means all outcomes
    since, until int64,
    limit int,
) ([]decision.Decision, error)

func BuildDecisionQuery(
    decisionType, source, symbol string,
    timeframe int,
    outcome string,
    since, until int64,
    limit int,
) (string, []any)
```

**Key design decisions:**

- `outcome` parameter is optional (empty string = no filter). This is the only family-specific query parameter beyond the shared key params.
- `BuildDecisionQuery` is exported for deterministic testing without ClickHouse.
- `signals` column deserialized via `ParseSignalInputsJSON()` — new helper alongside `ParseMetadataJSON()`.
- `metadata` column deserialized via existing `ParseMetadataJSON()`.
- `confidence` read as Float64, converted to string via `strconv.FormatFloat()` for domain type fidelity.

### 4.2 Artifact 2: Reader Tests

**File:** `internal/adapters/clickhouse/decision_reader_test.go`

Expected test cases:

| # | Test | Purpose |
|---|------|---------|
| 1 | Query with all required params | Base query construction |
| 2 | Query with since filter | Time range lower bound |
| 3 | Query with until filter | Time range upper bound |
| 4 | Query with since + until | Full range |
| 5 | Query with outcome filter | Family-specific filtering |
| 6 | Query without outcome filter | No outcome in WHERE clause |
| 7 | Column count matches DDL | 10 SELECT columns |
| 8 | Parameter count matches WHERE clauses | Dynamic parameter correctness |
| 9 | ParseSignalInputsJSON valid | JSON array deserialization |
| 10 | ParseSignalInputsJSON empty | Empty string fallback |
| 11 | ParseSignalInputsJSON malformed | Invalid JSON fallback |
| 12 | ParseMetadataJSON reuse | Existing helper works for decisions |

### 4.3 Artifact 3: Application Use Case

**File:** `internal/application/analyticalclient/get_decision_history.go`

```go
type GetDecisionHistoryUseCase struct {
    reader DecisionReader
    logger *slog.Logger
}

func (uc *GetDecisionHistoryUseCase) Execute(
    ctx context.Context,
    query DecisionHistoryQuery,
) (DecisionHistoryReply, *problem.Problem)
```

Validation rules (consistent with SignalHistoryQuery):
- `type` required (string, non-empty)
- `source` required (string, non-empty)
- `symbol` required (string, non-empty)
- `timeframe` > 0
- `since` ≥ 0, `until` ≥ 0, `since` ≤ `until` if both set
- `limit` ∈ [1, 500], default 50
- `outcome` optional — if provided, must be one of: `triggered`, `not_triggered`, `insufficient`

### 4.4 Artifact 4: Use Case Tests

**File:** `internal/application/analyticalclient/get_decision_history_test.go`

Expected test cases:

| # | Test | Purpose |
|---|------|---------|
| 1 | Valid query with all params | Happy path |
| 2 | Missing type | Validation failure |
| 3 | Missing source | Validation failure |
| 4 | Missing symbol | Validation failure |
| 5 | Zero timeframe | Validation failure |
| 6 | Invalid outcome | Validation failure |
| 7 | Valid outcome filter | Passes validation |
| 8 | Default limit | 50 when unset |
| 9 | Limit boundary (500) | Max accepted |
| 10 | Limit boundary (501) | Rejected |
| 11 | Reader returns error | Maps to Unavailable |
| 12 | Since > Until | Validation failure |

### 4.5 Artifact 5: Contracts

**File:** `internal/application/analyticalclient/contracts.go` (append)

```go
type DecisionHistoryQuery struct {
    Type      string `json:"type"`
    Source    string `json:"source"`
    Symbol    string `json:"symbol"`
    Timeframe int    `json:"timeframe"`
    Outcome   string `json:"outcome,omitempty"` // optional: triggered|not_triggered|insufficient
    Limit     int    `json:"limit"`
    Since     int64  `json:"since,omitempty"`
    Until     int64  `json:"until,omitempty"`
}

type DecisionHistoryReply struct {
    Decisions []decision.Decision `json:"decisions"`
    Source    string              `json:"source"` // always "clickhouse"
    Meta      QueryMeta           `json:"meta"`
}
```

### 4.6 Artifact 6: HTTP Handler

**File:** `internal/interfaces/http/handlers/analytical.go` (extend)

New method: `GetDecisionHistory(w http.ResponseWriter, r *http.Request)`

Handler flow:
1. Extract `type` query param (required)
2. Extract `outcome` query param (optional)
3. Extract shared key params via `parseEvidenceKeyParams()`
4. Extract `limit`, `since`, `until` (same pattern as Signal handler)
5. Validate `outcome` against known values if provided
6. Call `getDecisionHistory.Execute()`
7. Set `Server-Timing` header
8. Write JSON response

Constructor extends to 3 use cases:
```go
func NewAnalyticalWebHandler(
    getCandleHistory ...,
    getSignalHistory ...,
    getDecisionHistory ...,   // NEW
    logger ...,
)
```

### 4.7 Artifact 7: Route Registration

**File:** `internal/interfaces/http/routes/analytical.go` (extend)

```go
// Add to AnalyticalFamilyDeps:
GetDecisionHistory handlersGetAnalyticalDecisionHistoryUseCase

// Add to HasAny():
|| a.GetDecisionHistory != nil

// Add route:
if deps.GetDecisionHistory != nil {
    routes = append(routes, webserver.Route{
        Method:  http.MethodGet,
        Path:    "/analytical/decision/history",
        Handler: handler.GetDecisionHistory,
    })
}
```

### 4.8 Artifact 8: Integration Test

**File:** `tests/http/analytical.http` (extend)

```http
### Decision history — happy path
GET {{host}}/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&limit=5

### Decision history — with outcome filter
GET {{host}}/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&outcome=triggered&limit=5

### Decision history — missing type
GET {{host}}/analytical/decision/history?source=binancef&symbol=btcusdt&timeframe=60

### Decision history — invalid outcome
GET {{host}}/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&outcome=invalid
```

### 4.9 Artifact 9: Smoke Test Extension

**File:** `scripts/smoke-analytical-e2e.sh` (extend)

Add Phase 5c (Decision validation):
- Verify `decisions` table has rows in ClickHouse
- HTTP 200 with correct JSON structure (`decisions` array)
- All 10 domain fields present in response
- `signals` field is JSON array (not null/empty string)
- `metadata` field is JSON object (not null/empty string)
- Server-Timing header present
- 400 for missing params
- Optional: outcome filter returns subset

---

## 5. Endpoint Specification

### `GET /analytical/decision/history`

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| type | string | Yes | — | Decision type (e.g., `rsi_oversold`) |
| source | string | Yes | — | Data source (e.g., `binancef`) |
| symbol | string | Yes | — | Trading symbol (e.g., `btcusdt`) |
| timeframe | int | Yes | — | Candle timeframe in seconds |
| outcome | string | No | — | Filter by outcome: `triggered`, `not_triggered`, `insufficient` |
| limit | int | No | 50 | Max results (1–500) |
| since | int64 | No | — | Unix seconds, inclusive lower bound |
| until | int64 | No | — | Unix seconds, inclusive upper bound |

### Response (200 OK)

```json
{
  "decisions": [
    {
      "type": "rsi_oversold",
      "source": "binancef",
      "symbol": "btcusdt",
      "timeframe": 60,
      "outcome": "triggered",
      "confidence": "0.85",
      "signals": [{"type": "rsi", "value": "28.5", "timeframe": 60}],
      "metadata": {"threshold": "30", "period": "14"},
      "final": true,
      "timestamp": "2026-03-19T12:00:00.000Z"
    }
  ],
  "source": "clickhouse",
  "meta": {
    "query_ms": 12,
    "row_count": 1
  }
}
```

### Error Responses

| Status | Condition |
|--------|-----------|
| 400 | Missing required params, invalid limit/since/until, invalid outcome value |
| 503 | ClickHouse unavailable |

---

## 6. Gateway Composition

### `cmd/gateway/compose.go` Changes

1. Create `DecisionReader` from ClickHouse client
2. Create `GetDecisionHistoryUseCase` with reader
3. Pass use case to `AnalyticalFamilyDeps.GetDecisionHistory`
4. No changes to non-analytical composition

### `cmd/gateway/analytical_reader.go` Changes

1. Add `DecisionReader` interface (if reader interface is defined here)
2. Wire `NewDecisionReader()` in the analytical client factory

---

## 7. Observability (Mechanical Parity)

All observability follows the established pattern — no new instrumentation required beyond copy-and-adapt:

| Layer | Signal | Level |
|-------|--------|-------|
| Reader adapter | Query timing | DEBUG |
| Reader adapter | Query errors | ERROR |
| Use case | Completion + row count + query_ms | INFO |
| Use case | Failure | WARN |
| Handler | Server-Timing header | — |
| Handler | JSON response with QueryMeta | — |
| Writer (existing) | events_flushed, events_dropped, buffer_depth | via healthz |

---

## 8. Files Modified / Created Summary

| Action | File | Artifact # |
|--------|------|------------|
| CREATE | `internal/adapters/clickhouse/decision_reader.go` | 1 |
| CREATE | `internal/adapters/clickhouse/decision_reader_test.go` | 2 |
| CREATE | `internal/application/analyticalclient/get_decision_history.go` | 3 |
| CREATE | `internal/application/analyticalclient/get_decision_history_test.go` | 4 |
| MODIFY | `internal/application/analyticalclient/contracts.go` | 5 |
| MODIFY | `internal/interfaces/http/handlers/analytical.go` | 6 |
| MODIFY | `internal/interfaces/http/handlers/analytical_test.go` | 6 |
| MODIFY | `internal/interfaces/http/routes/analytical.go` | 7 |
| MODIFY | `internal/interfaces/http/routes/core.go` | 7 |
| MODIFY | `cmd/gateway/compose.go` | 7 |
| MODIFY | `cmd/gateway/analytical_reader.go` | 7 |
| MODIFY | `tests/http/analytical.http` | 8 |
| MODIFY | `scripts/smoke-analytical-e2e.sh` | 9 |
