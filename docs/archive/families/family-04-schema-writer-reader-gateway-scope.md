# Family 04 — Schema, Writer, Reader, Gateway Scope

> Stage: S180 · Wave B · `risk_assessments`
> Status: Defined — ready for S181 implementation

---

## 1. Schema

### 1.1 DDL (Pre-Staged)

Migration: `deploy/migrations/005_create_risk_assessments.sql`

```sql
CREATE TABLE IF NOT EXISTS risk_assessments (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    disposition    LowCardinality(String),
    confidence     Float64,
    strategies     String,          -- JSON: []StrategyInput
    constraints    String,          -- JSON: Constraints
    rationale      String,          -- free text
    parameters     String,          -- JSON: map[string]string
    metadata       String,          -- JSON: map[string]string
    final          Bool,
    timestamp      DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
```

### 1.2 Column Inventory

| # | Column | Type | Category | Notes |
|---|--------|------|----------|-------|
| 1 | `event_id` | String | Metadata | Event identity |
| 2 | `occurred_at` | DateTime64(3) | Metadata | Event creation time |
| 3 | `correlation_id` | String | Metadata | Trace correlation |
| 4 | `causation_id` | String | Metadata | Causal chain |
| 5 | `type` | LowCardinality(String) | Domain | e.g., `position_exposure` |
| 6 | `source` | LowCardinality(String) | Domain | Origin identifier |
| 7 | `symbol` | LowCardinality(String) | Domain | Trading pair |
| 8 | `timeframe` | UInt32 | Domain | Candle period (seconds) |
| 9 | `disposition` | LowCardinality(String) | Domain | approved/modified/rejected |
| 10 | `confidence` | Float64 | Domain | Assessment confidence |
| 11 | `strategies` | String | JSON | `[]StrategyInput` array |
| 12 | `constraints` | String | JSON | `Constraints` struct |
| 13 | `rationale` | String | Free text | Human-readable explanation |
| 14 | `parameters` | String | JSON | `map[string]string` |
| 15 | `metadata` | String | JSON | `map[string]string` |
| 16 | `final` | Bool | Domain | Interim vs final flag |
| 17 | `timestamp` | DateTime64(3) | Domain | Primary ordering time |
| — | `ingested_at` | DateTime64(3) | Ingestion | Auto-set, not queried |

**Total:** 17 queryable columns (highest in Wave B).

### 1.3 Complexity Comparison Across Families

| Family | DDL Cols | Domain Cols | JSON Cols | Enum Filter | Free Text |
|--------|----------|-------------|-----------|-------------|-----------|
| Candles | 16 | 12 | 0 | — | — |
| Signals | 12 | 6 | 1 | — | — |
| Decisions | 14 | 9 | 2 | outcome | — |
| Strategies | 15 | 11 | 3 | direction | — |
| **Risk** | **17** | **13** | **4** | **disposition** | **rationale** |

---

## 2. Writer (Pre-Staged — Zero Changes)

### 2.1 Mapper

File: `cmd/writer/mappers.go` — `mapRiskRow()` already exists.

**Column mapping (17 fields):**

| # | Go Expression | DDL Column | Type Conversion |
|---|--------------|------------|-----------------|
| 1 | `m.ID` | `event_id` | string |
| 2 | `m.OccurredAt` | `occurred_at` | time.Time |
| 3 | `m.CorrelationID` | `correlation_id` | string |
| 4 | `m.CausationID` | `causation_id` | string |
| 5 | `r.Type` | `type` | string |
| 6 | `r.Source` | `source` | string |
| 7 | `r.Symbol` | `symbol` | string |
| 8 | `uint32(r.Timeframe)` | `timeframe` | int → uint32 |
| 9 | `string(r.Disposition)` | `disposition` | Disposition → string |
| 10 | `parseFloat(r.Confidence)` | `confidence` | string → float64 |
| 11 | `marshalJSON(r.Strategies)` | `strategies` | []StrategyInput → JSON |
| 12 | `marshalJSON(r.Constraints)` | `constraints` | Constraints → JSON |
| 13 | `r.Rationale` | `rationale` | string (direct) |
| 14 | `marshalJSON(r.Parameters)` | `parameters` | map → JSON |
| 15 | `marshalJSON(r.Metadata)` | `metadata` | map → JSON |
| 16 | `r.Final` | `final` | bool |
| 17 | `r.Timestamp` | `timestamp` | time.Time |

### 2.2 Pipeline Configuration

File: `cmd/writer/pipeline.go` — entry already exists:

```
family:       "position_exposure"
table:        "risk_assessments"
consumerSpec: adapternats.WriterPositionExposureRiskConsumer()
isEnabled:    p.IsRiskFamilyEnabled("position_exposure")
```

### 2.3 Mapper Tests

File: `cmd/writer/mappers_test.go` — tests already exist:

- `TestMapRiskRow_ColumnCount` — verifies 17 columns
- `TestMapRiskRow_DomainFields` — verifies field mapping and type conversion

**Writer track scope: zero new work.**

---

## 3. Reader (New — Primary Implementation Scope)

### 3.1 File

`internal/adapters/clickhouse/risk_reader.go` (~138 lines)

### 3.2 Struct

```go
type RiskReader struct {
    client *Client
    logger *slog.Logger
}

func NewRiskReader(client *Client, logger *slog.Logger) *RiskReader
```

### 3.3 Query Method

```go
func (r *RiskReader) QueryRiskHistory(
    ctx context.Context,
    riskType, source, symbol string,
    timeframe int,
    disposition string,       // optional filter
    since, until int64,
    limit int,
) ([]risk.RiskAssessment, error)
```

### 3.4 SELECT Columns (Reader Query)

The reader queries 13 domain columns (excludes 4 metadata columns: `event_id`, `occurred_at`, `correlation_id`, `causation_id`):

```sql
SELECT type, source, symbol, timeframe, disposition, confidence,
       strategies, constraints, rationale, parameters, metadata,
       final, timestamp
FROM risk_assessments
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
  [AND disposition = ?]
  [AND timestamp >= ?]
  [AND timestamp <= ?]
ORDER BY timestamp DESC
LIMIT ?
```

### 3.5 Query Builder (Exported)

```go
func BuildRiskQuery(
    riskType, source, symbol string,
    timeframe int,
    disposition string,
    since, until int64,
    limit int,
) (string, []any)
```

Follows exact same pattern as `BuildStrategyQuery`: base WHERE with 4 required params, optional filter appended conditionally, time range conditionally, ORDER BY + LIMIT always.

### 3.6 JSON Parsers (New)

Two new parser functions required:

**1. `ParseStrategyInputsJSON`** — array of structs:

```go
func ParseStrategyInputsJSON(raw string) []risk.StrategyInput
```

- Empty/malformed input → `[]risk.StrategyInput{}` (silent fallback)
- Follows exact pattern of `ParseSignalInputsJSON` and `ParseDecisionInputsJSON`

**2. `ParseConstraintsJSON`** — single struct:

```go
func ParseConstraintsJSON(raw string) risk.Constraints
```

- Empty/malformed input → `risk.Constraints{}` (silent fallback)
- New parser shape (struct, not map or array) but trivial: `json.Unmarshal` into struct

**Reused parsers:**
- `ParseMetadataJSON` — for `parameters` and `metadata` columns (both `map[string]string`)
- `FormatFloat` — for `confidence` column

### 3.7 Row Scanning

```go
var (
    typ         string
    src         string
    sym         string
    tf          uint32
    disposition string
    confidence  float64
    strategies  string      // raw JSON
    constraints string      // raw JSON
    rationale   string      // plain text
    parameters  string      // raw JSON
    metadata    string      // raw JSON
    final       bool
    timestamp   time.Time
)
rows.Scan(&typ, &src, &sym, &tf, &disposition, &confidence,
    &strategies, &constraints, &rationale, &parameters, &metadata,
    &final, &timestamp)
```

### 3.8 Domain Mapping

```go
risk.RiskAssessment{
    Type:        typ,
    Source:      src,
    Symbol:      sym,
    Timeframe:   int(tf),
    Disposition: risk.Disposition(disposition),
    Confidence:  FormatFloat(confidence),
    Strategies:  ParseStrategyInputsJSON(strategies),
    Constraints: ParseConstraintsJSON(constraints),
    Rationale:   rationale,                          // direct string
    Parameters:  ParseMetadataJSON(parameters),
    Metadata:    ParseMetadataJSON(metadata),
    Final:       final,
    Timestamp:   timestamp,
}
```

### 3.9 Reader Tests

File: `internal/adapters/clickhouse/risk_reader_test.go`

| Test | Purpose |
|------|---------|
| `TestBuildRiskQuery_RequiredParams` | Base query with 4 required args |
| `TestBuildRiskQuery_WithDisposition` | Optional disposition filter |
| `TestBuildRiskQuery_WithSince` | Time range lower bound |
| `TestBuildRiskQuery_WithUntil` | Time range upper bound |
| `TestBuildRiskQuery_AllFilters` | All optional params combined |
| `TestParseStrategyInputsJSON_ValidArray` | Round-trip of strategy inputs |
| `TestParseStrategyInputsJSON_EmptyString` | Empty → empty slice |
| `TestParseStrategyInputsJSON_MalformedJSON` | Invalid → empty slice |
| `TestParseConstraintsJSON_ValidStruct` | Round-trip of constraints |
| `TestParseConstraintsJSON_EmptyString` | Empty → zero struct |
| `TestParseConstraintsJSON_MalformedJSON` | Invalid → zero struct |

---

## 4. Application Layer (New)

### 4.1 Contracts

Add to `internal/application/analyticalclient/contracts.go`:

**Interface:**

```go
type RiskReader interface {
    QueryRiskHistory(
        ctx context.Context,
        riskType, source, symbol string,
        timeframe int,
        disposition string,
        since, until int64,
        limit int,
    ) ([]risk.RiskAssessment, error)
}
```

**Query:**

```go
type RiskHistoryQuery struct {
    Type        string `json:"type"`
    Source      string `json:"source"`
    Symbol      string `json:"symbol"`
    Timeframe   int    `json:"timeframe"`
    Disposition string `json:"disposition,omitempty"`
    Limit       int    `json:"limit"`
    Since       int64  `json:"since,omitempty"`
    Until       int64  `json:"until,omitempty"`
}
```

**Reply:**

```go
type RiskHistoryReply struct {
    RiskAssessments []risk.RiskAssessment `json:"risk_assessments"`
    Source          string                `json:"source"`
    Meta            QueryMeta             `json:"meta"`
}
```

### 4.2 Use Case

File: `internal/application/analyticalclient/get_risk_history.go` (~60 lines)

```go
type GetRiskHistoryUseCase struct {
    reader RiskReader
    logger *slog.Logger
}

func NewGetRiskHistoryUseCase(reader RiskReader, logger *slog.Logger) *GetRiskHistoryUseCase

func (uc *GetRiskHistoryUseCase) Execute(
    ctx context.Context,
    query RiskHistoryQuery,
) (RiskHistoryReply, *problem.Problem)
```

**Validation (identical to all families):**
- `type` required
- `source` required
- `symbol` required
- `timeframe` > 0
- `since` >= 0
- `until` >= 0
- if both set: `since` <= `until`
- `limit` defaults to 50, capped at 500
- `disposition` not validated — passthrough

### 4.3 Use Case Tests

File: `internal/application/analyticalclient/get_risk_history_test.go`

| Test | Purpose |
|------|---------|
| `TestGetRiskHistoryUseCase_NilUseCase` | Nil receiver returns problem |
| `TestGetRiskHistoryUseCase_NilReader` | Missing reader returns unavailable |
| `TestGetRiskHistoryUseCase_EmptyType` | Validation: type required |
| `TestGetRiskHistoryUseCase_EmptySource` | Validation: source required |
| `TestGetRiskHistoryUseCase_EmptySymbol` | Validation: symbol required |
| `TestGetRiskHistoryUseCase_ZeroTimeframe` | Validation: timeframe > 0 |
| `TestGetRiskHistoryUseCase_NegativeSince` | Validation: since >= 0 |
| `TestGetRiskHistoryUseCase_SinceGreaterThanUntil` | Validation: range consistency |
| `TestGetRiskHistoryUseCase_LimitDefaulting` | Default limit = 50 |
| `TestGetRiskHistoryUseCase_LimitCapping` | Max limit = 500 |
| `TestGetRiskHistoryUseCase_ValidQuery` | Success path with timing |
| `TestGetRiskHistoryUseCase_EmptyResult` | Empty result is not an error |
| `TestGetRiskHistoryUseCase_DispositionPassthrough` | Optional filter forwarded |

---

## 5. HTTP Layer (Extend Existing)

### 5.1 Endpoint

```
GET /analytical/risk/history
```

### 5.2 Parameters

| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `type` | Yes | string | e.g., `position_exposure` |
| `source` | Yes | string | Source identifier |
| `symbol` | Yes | string | Trading pair |
| `timeframe` | Yes | int | Candle period in seconds |
| `disposition` | No | string | approved/modified/rejected |
| `limit` | No | int | Default 50, max 500 |
| `since` | No | int64 | Unix seconds, lower bound |
| `until` | No | int64 | Unix seconds, upper bound |

### 5.3 Response Shape

```json
{
  "risk_assessments": [
    {
      "type": "position_exposure",
      "source": "binance",
      "symbol": "BTCUSD",
      "timeframe": 60,
      "disposition": "approved",
      "confidence": "0.82",
      "strategies": [
        {"type": "mean_reversion_entry", "direction": "long", "confidence": "0.75", "timeframe": 60}
      ],
      "constraints": {
        "max_position_size": "0.1",
        "max_exposure": "1000.00"
      },
      "rationale": "Position within exposure limits, strategy confidence above threshold",
      "parameters": {"risk_model": "basic"},
      "metadata": {},
      "final": true,
      "timestamp": "2026-03-20T12:00:00.000Z"
    }
  ],
  "source": "clickhouse",
  "meta": {
    "query_ms": 12,
    "row_count": 1
  }
}
```

### 5.4 Response Headers

```
Server-Timing: total;dur=15, query;dur=12
Content-Type: application/json
```

### 5.5 Handler Changes

File: `internal/interfaces/http/handlers/analytical.go`

- Add `getRiskHistory` private interface type
- Add field to `AnalyticalHandlerDeps` and `AnalyticalWebHandler`
- Add `GetRiskHistory` method (~80 lines)
- Estimated file growth: ~500 → ~580 lines

### 5.6 Route Changes

File: `internal/interfaces/http/routes/analytical.go`

- Add `GetRiskHistory` to `AnalyticalFamilyDeps`
- Update `HasAny()` to include risk
- Add conditional route registration

### 5.7 Handler Tests

File: `internal/interfaces/http/handlers/analytical_test.go` (extend)

| Test | Purpose |
|------|---------|
| `TestGetRiskHistory_MissingHandler` | 503 when handler nil |
| `TestGetRiskHistory_MissingType` | 400 on missing type |
| `TestGetRiskHistory_MissingSource` | 400 on missing source |
| `TestGetRiskHistory_MissingSymbol` | 400 on missing symbol |
| `TestGetRiskHistory_MissingTimeframe` | 400 on missing timeframe |
| `TestGetRiskHistory_InvalidLimit` | 400 on non-numeric limit |
| `TestGetRiskHistory_DispositionFilter` | Disposition forwarded to use case |
| `TestGetRiskHistory_ServerTimingHeader` | Header present on success |

---

## 6. Gateway Composition (Extend Existing)

### 6.1 Reader Factory

File: `cmd/gateway/analytical_reader.go`

Add:

```go
func newAnalyticalRiskReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.RiskReader {
    return clickhouse.NewRiskReader(client, logger)
}
```

### 6.2 Composition Wiring

File: `cmd/gateway/compose.go`

In `buildRouteDependencies`, inside the `if chClient != nil` block:

```go
riskReader := newAnalyticalRiskReader(chClient, logger)

analytical = routes.AnalyticalFamilyDeps{
    // ... existing fields ...
    GetRiskHistory: analyticalclient.NewGetRiskHistoryUseCase(riskReader, logger),
}
```

### 6.3 Configuration

File: `deploy/configs/gateway.jsonc`

No gateway config changes needed — ClickHouse connection settings already configured. Risk reader uses the same shared ClickHouse client.

---

## 7. Integration & Smoke Testing

### 7.1 Smoke Test Extension

File: `scripts/smoke-analytical-e2e.sh`

Add:

```bash
validate_analytical_family "risk_assessments" \
    "http://gateway:8080/analytical/risk/history" \
    "type=position_exposure&source=binance&symbol=BTCUSD&timeframe=60"
```

### 7.2 HTTP Test File

Extend `tests/http/analytical.http` with risk assessment queries:

```http
### Risk History — required params
GET http://localhost:8080/analytical/risk/history?type=position_exposure&source=binance&symbol=BTCUSD&timeframe=60

### Risk History — with disposition filter
GET http://localhost:8080/analytical/risk/history?type=position_exposure&source=binance&symbol=BTCUSD&timeframe=60&disposition=approved

### Risk History — with time range
GET http://localhost:8080/analytical/risk/history?type=position_exposure&source=binance&symbol=BTCUSD&timeframe=60&since=1710000000&until=1710100000&limit=100
```

---

## 8. Implementation Artifact Checklist

| # | Artifact | File | Status |
|---|----------|------|--------|
| 1 | Schema migration | `deploy/migrations/005_create_risk_assessments.sql` | Pre-staged |
| 2 | Writer mapper | `cmd/writer/mappers.go` | Pre-staged |
| 3 | Pipeline config | `cmd/writer/pipeline.go` | Pre-staged |
| 4 | ClickHouse reader | `internal/adapters/clickhouse/risk_reader.go` | **S181** |
| 5 | Application use case | `internal/application/analyticalclient/get_risk_history.go` | **S181** |
| 6 | Contract types | `internal/application/analyticalclient/contracts.go` | **S181** |
| 7 | HTTP handler method | `internal/interfaces/http/handlers/analytical.go` | **S181** |
| 8 | Route registration | `internal/interfaces/http/routes/analytical.go` | **S181** |
| 9 | Smoke test | `scripts/smoke-analytical-e2e.sh` | **S181** |

**Pre-staged:** 3/9 (migration, mapper, pipeline)
**New in S181:** 6/9 (reader, use case, contracts, handler, routes, smoke)
