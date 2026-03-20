# Family 05 — Schema, Writer, Reader, Gateway Contract

> Frozen contract specifying the exact schema mapping, writer path, reader path, and gateway boundary for the Executions (paper_order) family. Each layer's responsibility is explicit, bounded, and traceable to implementation artifacts.

---

## 1. Schema Contract

### 1.1 DDL (migration 006)

Table: `executions`
Engine: `MergeTree()`
Partition: `toYYYYMM(timestamp)`
Order: `(source, symbol, timeframe, type, timestamp)`
TTL: `toDateTime(timestamp) + INTERVAL 90 DAY`

### 1.2 Column coherence table

This table specifies the exact mapping from DDL column to writer mapper field to reader scan variable to domain struct field. Every column has exactly one owner at each layer.

| # | DDL Column | DDL Type | Writer mapper (`mapExecutionRow`) | Reader scan variable | Reader → Domain mapping | Domain field |
|---|-----------|----------|----------------------------------|---------------------|------------------------|-------------|
| 1 | event_id | String | `m.ID` | *(not selected)* | — | — |
| 2 | occurred_at | DateTime64(3) | `m.OccurredAt` | *(not selected)* | — | — |
| 3 | correlation_id | String | `m.CorrelationID` | *(not selected)* | — | — |
| 4 | causation_id | String | `m.CausationID` | *(not selected)* | — | — |
| 5 | type | LowCardinality(String) | `x.Type` | `typ string` | Direct assign | `ExecutionIntent.Type` |
| 6 | source | LowCardinality(String) | `x.Source` | `src string` | Direct assign | `ExecutionIntent.Source` |
| 7 | symbol | LowCardinality(String) | `x.Symbol` | `sym string` | Direct assign | `ExecutionIntent.Symbol` |
| 8 | timeframe | UInt32 | `uint32(x.Timeframe)` | `tf uint32` | `int(tf)` | `ExecutionIntent.Timeframe` |
| 9 | side | LowCardinality(String) | `string(x.Side)` | `side string` | `execution.Side(side)` | `ExecutionIntent.Side` |
| 10 | quantity | Float64 | `parseFloat(x.Quantity)` | `quantity float64` | `FormatFloat(quantity)` | `ExecutionIntent.Quantity` |
| 11 | filled_quantity | Float64 | `parseFloat(x.FilledQuantity)` | `filledQty float64` | `FormatFloat(filledQty)` | `ExecutionIntent.FilledQuantity` |
| 12 | status | LowCardinality(String) | `string(x.Status)` | `status string` | `execution.Status(status)` | `ExecutionIntent.Status` |
| 13 | risk | String (JSON) | `marshalJSON(x.Risk)` | `riskJSON string` | `ParseRiskInputJSON(riskJSON)` | `ExecutionIntent.Risk` |
| 14 | fills | String (JSON) | `marshalJSON(x.Fills)` | `fillsJSON string` | `ParseFillsJSON(fillsJSON)` | `ExecutionIntent.Fills` |
| 15 | parameters | String (JSON) | `marshalJSON(x.Parameters)` | `parameters string` | `ParseMetadataJSON(parameters)` | `ExecutionIntent.Parameters` |
| 16 | metadata | String (JSON) | `marshalJSON(x.Metadata)` | `metadata string` | `ParseMetadataJSON(metadata)` | `ExecutionIntent.Metadata` |
| 17 | exec_correlation_id | String | `x.CorrelationID` | `execCorrID string` | Direct assign | `ExecutionIntent.CorrelationID` |
| 18 | exec_causation_id | String | `x.CausationID` | `execCausID string` | Direct assign | `ExecutionIntent.CausationID` |
| 19 | final | Bool | `x.Final` | `final bool` | Direct assign | `ExecutionIntent.Final` |
| 20 | timestamp | DateTime64(3) | `x.Timestamp` | `timestamp time.Time` | Direct assign | `ExecutionIntent.Timestamp` |

**Column count:** 20 DDL, 16 selected by reader (columns 1–4 excluded).
**Scan order:** Must match SELECT order exactly (columns 5–20).

### 1.3 Column type distribution

| Type category | Columns | Pattern precedent |
|--------------|---------|-------------------|
| String (direct) | type, source, symbol, side, status, exec_correlation_id, exec_causation_id | All families |
| UInt32 | timeframe | All families |
| Float64 | quantity, filled_quantity | **New — first Float64 in read path** |
| Bool | final | **New — first Bool in read path** |
| DateTime64(3) | timestamp | All families |
| String (JSON → struct) | risk | ParseConstraintsJSON pattern (F-04) |
| String (JSON → slice) | fills | ParseStrategyInputsJSON pattern (F-04) |
| String (JSON → map) | parameters, metadata | ParseMetadataJSON pattern (all families) |

---

## 2. Writer Contract

### 2.1 Status: Pre-staged — no changes required

The writer path for executions is fully operational:

| Artifact | File | Status |
|----------|------|--------|
| Mapper function | `cmd/writer/mappers.go:147` (`mapExecutionRow`) | Exists — 20 column mapping |
| Pipeline entry | `cmd/writer/pipeline.go` | Exists — `execution_events` consumer |
| NATS consumer | Via `internal/adapters/nats/execution_registry.go` | Exists — `paper_order` family |

### 2.2 Write-path invariant

**The write path MUST NOT be modified during Family 05 implementation.** This is the 6th consecutive family expansion with zero write-path changes. The invariant holds because:

1. `mapExecutionRow` already maps all 20 DDL columns.
2. The pipeline already consumes `PaperOrderSubmittedEvent` from NATS.
3. The inserter already handles batch insertion for the `executions` table.

### 2.3 Mapper column alignment verification

The mapper at `cmd/writer/mappers.go:147-171` produces `[]any` with 20 elements matching the DDL column order exactly:

```
m.ID → event_id
m.OccurredAt → occurred_at
m.CorrelationID → correlation_id
m.CausationID → causation_id
x.Type → type
x.Source → source
x.Symbol → symbol
uint32(x.Timeframe) → timeframe
string(x.Side) → side
parseFloat(x.Quantity) → quantity
parseFloat(x.FilledQuantity) → filled_quantity
string(x.Status) → status
marshalJSON(x.Risk) → risk
marshalJSON(x.Fills) → fills
marshalJSON(x.Parameters) → parameters
marshalJSON(x.Metadata) → metadata
x.CorrelationID → exec_correlation_id
x.CausationID → exec_causation_id
x.Final → final
x.Timestamp → timestamp
```

**Alignment confirmed.** No gaps, no reordering, no missing columns.

---

## 3. Reader Contract

### 3.1 Artifacts to build

| Artifact | File | Estimated LOC |
|----------|------|---------------|
| `ExecutionReader` struct | `internal/adapters/clickhouse/execution_reader.go` | ~15 |
| `NewExecutionReader` constructor | Same file | ~8 |
| `QueryExecutionHistory` method | Same file | ~65 |
| `BuildExecutionQuery` function | Same file | ~30 |
| `ParseRiskInputJSON` function | Same file | ~12 |
| `ParseFillsJSON` function | Same file | ~12 |
| **Total** | | **~142** |

### 3.2 Reader struct

```go
type ExecutionReader struct {
    client *Client
    logger *slog.Logger
}
```

Pattern: Identical to `RiskReader`, `StrategyReader`, etc.

### 3.3 Query method signature

```go
func (r *ExecutionReader) QueryExecutionHistory(
    ctx context.Context,
    execType, source, symbol string,
    timeframe int,
    side, status string,
    since, until int64,
    limit int,
) ([]execution.ExecutionIntent, error)
```

**10 parameters** (vs 8 for risk). The two additional are `side` and `status` optional filters.

### 3.4 Query builder

```go
func BuildExecutionQuery(
    execType, source, symbol string,
    timeframe int,
    side, status string,
    since, until int64,
    limit int,
) (string, []any)
```

Base query:
```sql
SELECT type, source, symbol, timeframe, side, quantity, filled_quantity, status,
       risk, fills, parameters, metadata,
       exec_correlation_id, exec_causation_id, final, timestamp
FROM executions
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
```

Optional clauses (additive, independent):
```sql
AND side = ?       -- if side != ""
AND status = ?     -- if status != ""
AND timestamp >= ? -- if since > 0
AND timestamp <= ? -- if until > 0
```

Suffix:
```sql
ORDER BY timestamp DESC LIMIT ?
```

### 3.5 Row scan → domain mapping

The scan produces 16 local variables that are assembled into `execution.ExecutionIntent`:

```go
execution.ExecutionIntent{
    Type:           typ,
    Source:         src,
    Symbol:         sym,
    Timeframe:     int(tf),
    Side:           execution.Side(side),
    Quantity:       FormatFloat(quantity),
    FilledQuantity: FormatFloat(filledQty),
    Status:         execution.Status(status),
    Risk:           ParseRiskInputJSON(riskJSON),
    Fills:          ParseFillsJSON(fillsJSON),
    Parameters:     ParseMetadataJSON(parameters),
    Metadata:       ParseMetadataJSON(metadata),
    CorrelationID:  execCorrID,
    CausationID:    execCausID,
    Final:          final,
    Timestamp:      timestamp,
}
```

### 3.6 Parser reuse and new parsers

| Parser | Target type | Status | Precedent |
|--------|------------|--------|-----------|
| `FormatFloat` | `float64` → `string` | Reuse (2 calls: quantity, filled_quantity) | Used in decisions, strategies, risk |
| `ParseMetadataJSON` | `string` → `map[string]string` | Reuse (2 calls: parameters, metadata) | Used in all families |
| `ParseRiskInputJSON` | `string` → `execution.RiskInput` | **New** | Same shape as `ParseConstraintsJSON` (struct target) |
| `ParseFillsJSON` | `string` → `[]execution.FillRecord` | **New** | Same shape as `ParseStrategyInputsJSON` (slice target) |

Post-Family-05 parser count: **8** (at healthy threshold, generic parser recommended if exceeded).

---

## 4. Use Case Contract

### 4.1 Artifacts to build

| Artifact | File | Estimated LOC |
|----------|------|---------------|
| `ExecutionReader` interface | `internal/application/analyticalclient/get_execution_history.go` | ~3 |
| `GetExecutionHistoryUseCase` struct | Same file | ~5 |
| `NewGetExecutionHistoryUseCase` constructor | Same file | ~8 |
| `Execute` method | Same file | ~75 |
| Query/Reply contracts | `internal/application/analyticalclient/contracts.go` | ~30 |
| **Total** | | **~121** |

### 4.2 Validation rules

Identical to all prior families plus two optional filters:

| Field | Rule | Error |
|-------|------|-------|
| Type | Non-empty | `"type is required"` |
| Source | Non-empty | `"source is required"` |
| Symbol | Non-empty | `"symbol is required"` |
| Timeframe | > 0 | `"timeframe must be positive"` |
| Since | ≥ 0 | `"since must be a non-negative unix timestamp"` |
| Until | ≥ 0 | `"until must be a non-negative unix timestamp"` |
| Since/Until | since ≤ until (when both > 0) | `"since must not be after until"` |
| Limit | Default 50, cap 500 | Applied silently |
| Side | No validation | Pass-through to reader |
| Status | No validation | Pass-through to reader |

---

## 5. Handler Contract

### 5.1 Artifacts to modify

| Artifact | File | Change type | Estimated LOC added |
|----------|------|------------|-------------------|
| Interface | `internal/interfaces/http/handlers/analytical.go` | New interface type | ~3 |
| Struct field | Same file (`AnalyticalWebHandler`) | New field | ~1 |
| Deps field | Same file (`AnalyticalHandlerDeps`) | New field | ~1 |
| Constructor | Same file (`NewAnalyticalWebHandler`) | Wire new field | ~1 |
| Response type | Same file | New struct | ~5 |
| Handler method | Same file (`GetExecutionHistory`) | **New method** | ~85–100 |
| **Total addition** | | | **~96–111** |

### 5.2 Handler method structure

The `GetExecutionHistory` method follows the exact structure of `GetRiskHistory` with one addition — a second optional filter:

1. Nil check (`h == nil || h.getExecutionHistory == nil`)
2. Parse `type` (required)
3. Parse key params via `parseQueryKeyParams(r)` (source, symbol, timeframe)
4. Parse `side` (optional — `r.URL.Query().Get("side")`)
5. Parse `status` (optional — `r.URL.Query().Get("status")`)
6. Parse `limit` (optional, default 50, max 500)
7. Parse `since` / `until` (optional timestamps)
8. Execute use case
9. Set `Server-Timing` header
10. Write JSON response

### 5.3 Handler file size projection

| State | Lines | Threshold |
|-------|-------|-----------|
| Pre-Family-05 | 515 | — |
| Post-Family-05 (projected) | **611–626** | 620 hard ceiling |

**Risk:** The handler may exceed 620 lines because the execution method has ~6 lines more than risk (two optional filter reads instead of one). If actual line count exceeds 620:

- **Immediate action:** Extract `parseAnalyticalParams()` helper for the common prefix (type/source/symbol/timeframe/limit/since/until). Reduces each method from ~90 to ~30 lines. Estimated effort: ~1 hour.
- **This extraction is allowed during Family 05** — it is a triggered refactor, not a proactive redesign.

---

## 6. Gateway Contract

### 6.1 Artifacts to modify

| Artifact | File | Change type | Estimated LOC added |
|----------|------|------------|-------------------|
| Factory function | `cmd/gateway/analytical_reader.go` | New function | ~8 |
| Route deps field | `internal/interfaces/http/routes/analytical.go` | New field + interface + HasAny update | ~12 |
| Composition wiring | `cmd/gateway/compose.go` or `cmd/gateway/run.go` | Wire execution reader | ~5 |
| **Total addition** | | | **~25** |

### 6.2 Gateway factory function

```go
func newAnalyticalExecutionReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.ExecutionReader {
    return clickhouse.NewExecutionReader(client, logger)
}
```

Pattern: Identical to `newAnalyticalRiskReader`.

### 6.3 Route registration

```go
if deps.GetExecutionHistory != nil {
    routes = append(routes, webserver.Route{
        Method:  http.MethodGet,
        Path:    "/analytical/execution/history",
        Handler: handler.GetExecutionHistory,
    })
}
```

Pattern: Identical to all prior families.

### 6.4 AnalyticalFamilyDeps extension

```go
type AnalyticalFamilyDeps struct {
    GetCandleHistory    handlersGetAnalyticalCandleHistoryUseCase
    GetSignalHistory    handlersGetAnalyticalSignalHistoryUseCase
    GetDecisionHistory  handlersGetAnalyticalDecisionHistoryUseCase
    GetStrategyHistory  handlersGetAnalyticalStrategyHistoryUseCase
    GetRiskHistory      handlersGetAnalyticalRiskHistoryUseCase
    GetExecutionHistory handlersGetAnalyticalExecutionHistoryUseCase  // NEW
}
```

`HasAny()` updated to include `|| a.GetExecutionHistory != nil`.

---

## 7. Test Contract

### 7.1 Reader tests

File: `internal/adapters/clickhouse/execution_reader_test.go`

| Test | Purpose |
|------|---------|
| `TestBuildExecutionQuery_BasicFilters` | Required params produce correct WHERE clause |
| `TestBuildExecutionQuery_SideFilter` | Optional side filter adds WHERE clause |
| `TestBuildExecutionQuery_StatusFilter` | Optional status filter adds WHERE clause |
| `TestBuildExecutionQuery_BothFilters` | Both side + status produce two additional WHERE clauses |
| `TestBuildExecutionQuery_SinceUntil` | Time range filters work correctly |
| `TestBuildExecutionQuery_AllFilters` | All optional filters combine correctly |
| `TestParseRiskInputJSON_Valid` | Valid JSON produces correct RiskInput struct |
| `TestParseRiskInputJSON_Empty` | Empty/default strings produce zero-value struct |
| `TestParseRiskInputJSON_Invalid` | Malformed JSON produces zero-value struct |
| `TestParseFillsJSON_Valid` | Valid JSON produces correct FillRecord slice |
| `TestParseFillsJSON_Empty` | Empty/default strings produce empty slice |
| `TestParseFillsJSON_Invalid` | Malformed JSON produces empty slice |
| `TestParseFillsJSON_MultipleFills` | Multiple fill entries deserialize correctly |

Estimated: **~95 LOC**

### 7.2 Use case tests

File: `internal/application/analyticalclient/get_execution_history_test.go`

| Test | Purpose |
|------|---------|
| `TestGetExecutionHistory_Success` | Valid query returns executions with correct meta |
| `TestGetExecutionHistory_MissingType` | Empty type returns InvalidArgument |
| `TestGetExecutionHistory_MissingSource` | Empty source returns InvalidArgument |
| `TestGetExecutionHistory_MissingSymbol` | Empty symbol returns InvalidArgument |
| `TestGetExecutionHistory_InvalidTimeframe` | Zero/negative timeframe returns InvalidArgument |
| `TestGetExecutionHistory_SinceAfterUntil` | since > until returns InvalidArgument |
| `TestGetExecutionHistory_DefaultLimit` | Zero limit defaults to 50 |
| `TestGetExecutionHistory_MaxLimit` | Limit > 500 capped to 500 |
| `TestGetExecutionHistory_ReaderUnavailable` | Nil reader returns Unavailable |
| `TestGetExecutionHistory_ReaderError` | Reader error wrapped as Unavailable |

Estimated: **~90 LOC**

### 7.3 Handler tests

Added to: `internal/interfaces/http/handlers/analytical_test.go`

| Test | Purpose |
|------|---------|
| `TestGetExecutionHistory_Success` | Valid request returns 200 with correct body |
| `TestGetExecutionHistory_MissingType` | Missing type returns 400 |
| `TestGetExecutionHistory_MissingSource` | Missing source returns 400 |
| `TestGetExecutionHistory_Unavailable` | Nil use case returns 503 |
| `TestGetExecutionHistory_SideFilter` | Side filter passed to use case |
| `TestGetExecutionHistory_StatusFilter` | Status filter passed to use case |

Estimated: **~85 LOC**

### 7.4 Projected test count

| State | Test count |
|-------|-----------|
| Pre-Family-05 | ~245 |
| Post-Family-05 | ~277 (±5) |
| Growth | ~32 tests |

---

## 8. HTTP Test Queries

Added to: `tests/http/analytical.http`

```http
### Execution history — all
GET {{host}}/analytical/execution/history?type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60

### Execution history — with side filter
GET {{host}}/analytical/execution/history?type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60&side=buy

### Execution history — with status filter
GET {{host}}/analytical/execution/history?type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60&status=filled

### Execution history — with both filters
GET {{host}}/analytical/execution/history?type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60&side=buy&status=filled

### Execution history — with time range
GET {{host}}/analytical/execution/history?type=paper_order&source=derive&symbol=BTCUSDT&timeframe=60&since=1710892800&until=1710979200&limit=100

### Execution history — missing source (expect 400)
GET {{host}}/analytical/execution/history?type=paper_order&symbol=BTCUSDT&timeframe=60
```

---

## 9. Boundary Summary

| Boundary | What crosses | What does NOT cross |
|----------|-------------|-------------------|
| NATS → Writer | `PaperOrderSubmittedEvent` (pre-staged) | No changes |
| Writer → ClickHouse | 20-column batch insert (pre-staged) | No changes |
| ClickHouse → Reader | 16-column SELECT | No schema changes |
| Reader → Use Case | `[]execution.ExecutionIntent` | No new domain types |
| Use Case → Handler | `ExecutionHistoryReply` | No new response patterns |
| Handler → HTTP | JSON response + Server-Timing | No new headers |
| Gateway | Struct DI field addition | No constructor changes |
| Routes | Conditional route registration | No existing routes modified |
