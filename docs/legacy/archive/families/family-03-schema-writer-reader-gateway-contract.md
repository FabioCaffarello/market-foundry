# Family 03 — Schema, Writer, Reader, Gateway Contract

**Family**: Strategies (`mean_reversion_entry`)
**Artifact count**: 9 (per Wave B pattern v2)
**Pre-staged artifacts**: 3 (migration, mapper, pipeline entry)
**New artifacts needed**: 6 (reader adapter, use case, contracts, handler, route, tests)

---

## 1. 9-Artifact inventory

| #  | Artifact                        | Location                                                  | Status      |
|--- |-------------------------------- |---------------------------------------------------------- |------------ |
| A1 | Schema migration                | `deploy/migrations/004_create_strategies.sql`             | Exists      |
| A2 | Writer mapper                   | `cmd/writer/mappers.go` → `mapStrategyRow()`              | Exists      |
| A3 | Pipeline entry                  | `cmd/writer/pipeline.go` → `mean_reversion_entry`         | Exists      |
| A4 | Reader adapter                  | `internal/adapters/clickhouse/strategy_reader.go`         | **S176**    |
| A5 | Use case                        | `internal/application/analyticalclient/get_strategy_history.go` | **S176** |
| A6 | Contracts (query + reply)       | `internal/application/analyticalclient/contracts.go`      | **S176**    |
| A7 | HTTP handler                    | `internal/interfaces/http/handlers/analytical.go`         | **S176**    |
| A8 | Route registration              | `internal/interfaces/http/routes/analytical.go`           | **S176**    |
| A9 | Integration/smoke test          | `scripts/smoke-analytical-e2e.sh` + `tests/http/analytical.http` | **S176** |

---

## 2. Writer path (exists — zero changes)

### 2.1 Mapper: `mapStrategyRow()`

```go
func mapStrategyRow(e strategy.StrategyResolvedEvent) []any {
    m := e.Metadata
    s := e.Strategy
    return []any{
        m.ID, m.OccurredAt, m.CorrelationID, m.CausationID,
        s.Type, s.Source, s.Symbol, uint32(s.Timeframe),
        string(s.Direction), parseFloat(s.Confidence),
        marshalJSON(s.Decisions), marshalJSON(s.Parameters), marshalJSON(s.Metadata),
        s.Final, s.Timestamp,
    }
}
```

Column order matches DDL exactly. 15 values for 15 columns (excluding `ingested_at` DEFAULT).

### 2.2 Pipeline entry

```go
{
    family:       "mean_reversion_entry",
    consumerName: "writer-strategy-mean-reversion-entry-consumer",
    inserterName: "writer-strategy-mean-reversion-entry-inserter",
    table:        "strategies",
    insertSQL:    "INSERT INTO strategies",
    consumerSpec: adapternats.WriterMeanReversionEntryStrategyConsumer(),
    isEnabled:    func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("mean_reversion_entry") },
    // ...
}
```

**Invariant**: Writer path is active and consuming events. Family 03 only adds the read path.

---

## 3. Reader path (to be implemented in S176)

### 3.1 Reader adapter: `StrategyReader`

**File**: `internal/adapters/clickhouse/strategy_reader.go`

**Structure** (follows `DecisionReader` pattern):

```go
type StrategyReader struct {
    client *Client
    logger *slog.Logger
}

func NewStrategyReader(client *Client, logger *slog.Logger) *StrategyReader

func (r *StrategyReader) QueryStrategyHistory(
    ctx context.Context,
    strategyType, source, symbol string,
    timeframe int,
    direction string,
    since, until int64,
    limit int,
) ([]strategy.Strategy, error)
```

### 3.2 Query builder: `BuildStrategyQuery`

```go
func BuildStrategyQuery(
    strategyType, source, symbol string,
    timeframe int,
    direction string,
    since, until int64,
    limit int,
) (string, []any)
```

**SQL template**:
```sql
SELECT type, source, symbol, timeframe, direction, confidence,
       decisions, parameters, metadata, final, timestamp
FROM strategies
WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?
  [AND direction = ?]
  [AND timestamp >= ?]
  [AND timestamp <= ?]
ORDER BY timestamp DESC
LIMIT ?
```

SELECT returns 11 domain columns. Optional clauses added only when parameter is non-empty/non-zero.

### 3.3 New parser: `ParseDecisionInputsJSON`

```go
func ParseDecisionInputsJSON(raw string) []strategy.DecisionInput {
    if raw == "" || raw == "[]" || raw == "{}" {
        return []strategy.DecisionInput{}
    }
    var inputs []strategy.DecisionInput
    if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
        return []strategy.DecisionInput{}
    }
    return inputs
}
```

Mirrors `ParseSignalInputsJSON` — same empty/error fallback contract.

### 3.4 Row scan mapping

```go
var (
    typ        string
    src        string
    sym        string
    tf         uint32
    dir        string
    confidence float64
    decisions  string
    parameters string
    metadata   string
    final      bool
    timestamp  time.Time
)

rows.Scan(&typ, &src, &sym, &tf, &dir, &confidence, &decisions, &parameters, &metadata, &final, &timestamp)

strategy.Strategy{
    Type:       typ,
    Source:     src,
    Symbol:     sym,
    Timeframe:  int(tf),
    Direction:  strategy.Direction(dir),
    Confidence: FormatFloat(confidence),
    Decisions:  ParseDecisionInputsJSON(decisions),
    Parameters: ParseMetadataJSON(parameters),
    Metadata:   ParseMetadataJSON(metadata),
    Final:      final,
    Timestamp:  timestamp,
}
```

### 3.5 Reader adapter tests

**File**: `internal/adapters/clickhouse/strategy_reader_test.go`

Minimum test cases (follows decision reader test pattern):

| #  | Case                                  | Validates                                  |
|--- |-------------------------------------- |------------------------------------------- |
| T1 | `BuildStrategyQuery` — all required   | Base query with 4 required params          |
| T2 | `BuildStrategyQuery` — with direction | Optional direction clause appended         |
| T3 | `BuildStrategyQuery` — with since     | Timestamp lower bound clause               |
| T4 | `BuildStrategyQuery` — with until     | Timestamp upper bound clause               |
| T5 | `BuildStrategyQuery` — all filters    | All optional clauses combined              |
| T6 | `ParseDecisionInputsJSON` — valid     | Round-trip array deserialization            |
| T7 | `ParseDecisionInputsJSON` — empty     | Empty/nil fallback                         |
| T8 | `ParseDecisionInputsJSON` — malformed | Graceful fallback to empty slice           |

---

## 4. Application layer (to be implemented in S176)

### 4.1 Contracts

**Added to** `internal/application/analyticalclient/contracts.go`:

```go
type StrategyHistoryQuery struct {
    Type      string `json:"type"`
    Source    string `json:"source"`
    Symbol    string `json:"symbol"`
    Timeframe int    `json:"timeframe"`
    Direction string `json:"direction,omitempty"`
    Limit     int    `json:"limit"`
    Since     int64  `json:"since,omitempty"`
    Until     int64  `json:"until,omitempty"`
}

type StrategyHistoryReply struct {
    Strategies []strategy.Strategy `json:"strategies"`
    Source     string              `json:"source"`
    Meta       QueryMeta           `json:"meta"`
}
```

### 4.2 Use case: `GetStrategyHistoryUseCase`

**File**: `internal/application/analyticalclient/get_strategy_history.go`

```go
type StrategyReader interface {
    QueryStrategyHistory(ctx context.Context, strategyType, source, symbol string, timeframe int, direction string, since, until int64, limit int) ([]strategy.Strategy, error)
}

type GetStrategyHistoryUseCase struct {
    reader StrategyReader
    logger *slog.Logger
}

func NewGetStrategyHistoryUseCase(reader StrategyReader, logger *slog.Logger) *GetStrategyHistoryUseCase
func (uc *GetStrategyHistoryUseCase) Execute(ctx context.Context, query StrategyHistoryQuery) (StrategyHistoryReply, *problem.Problem)
```

**Validation rules** (identical to decision use case):
- `type` required
- `source` required
- `symbol` required
- `timeframe` > 0
- `since` >= 0
- `until` >= 0
- if both set: `since` <= `until`
- `limit` default 50, max 500
- `direction` NOT validated (empty = all; invalid = empty result)

### 4.3 Use case tests

**File**: `internal/application/analyticalclient/get_strategy_history_test.go`

| #  | Case                                     | Validates                              |
|--- |----------------------------------------- |--------------------------------------- |
| T1 | nil use case returns unavailable         | Graceful degradation                   |
| T2 | nil reader returns unavailable           | Graceful degradation                   |
| T3 | empty type returns invalid argument      | Required field validation              |
| T4 | empty source returns invalid argument    | Required field validation              |
| T5 | empty symbol returns invalid argument    | Required field validation              |
| T6 | zero timeframe returns invalid argument  | Required field validation              |
| T7 | negative since returns invalid argument  | Range validation                       |
| T8 | since > until returns invalid argument   | Range consistency                      |
| T9 | valid query returns strategies            | Happy path                             |
| T10| empty result returns empty slice         | No nil slice in response               |
| T11| default limit applied                    | Limit defaults to 50                   |
| T12| max limit capped                         | Limit capped at 500                    |

---

## 5. Gateway boundary (to be implemented in S176)

### 5.1 Handler additions

**File**: `internal/interfaces/http/handlers/analytical.go`

Add to `AnalyticalWebHandler`:
```go
type getAnalyticalStrategyHistoryUseCase interface {
    Execute(context.Context, analyticalclient.StrategyHistoryQuery) (analyticalclient.StrategyHistoryReply, *problem.Problem)
}

// Add field to struct:
getStrategyHistory getAnalyticalStrategyHistoryUseCase

// Add to AnalyticalHandlerDeps:
GetStrategyHistory getAnalyticalStrategyHistoryUseCase
```

New method: `GetStrategyHistory(w http.ResponseWriter, r *http.Request)`
- Parses `type`, `source`, `symbol`, `timeframe` (required), `direction`, `limit`, `since`, `until` (optional)
- Follows exact same structure as `GetDecisionHistory` with `direction` replacing `outcome`

### 5.2 Route registration

**File**: `internal/interfaces/http/routes/analytical.go`

Add to `AnalyticalFamilyDeps`:
```go
GetStrategyHistory handlersGetAnalyticalStrategyHistoryUseCase
```

Add interface:
```go
type handlersGetAnalyticalStrategyHistoryUseCase interface {
    Execute(context.Context, analyticalclient.StrategyHistoryQuery) (analyticalclient.StrategyHistoryReply, *problem.Problem)
}
```

Add route:
```go
if deps.GetStrategyHistory != nil {
    routes = append(routes, webserver.Route{
        Method:  http.MethodGet,
        Path:    "/analytical/strategy/history",
        Handler: handler.GetStrategyHistory,
    })
}
```

Update `HasAny()` to include `a.GetStrategyHistory != nil`.

### 5.3 Gateway composition

**File**: `cmd/gateway/analytical_reader.go`

Add:
```go
func newAnalyticalStrategyReader(client *clickhouse.Client, logger *slog.Logger) analyticalclient.StrategyReader {
    return clickhouse.NewStrategyReader(client, logger)
}
```

**File**: `cmd/gateway/compose.go`

Wire into `buildRouteDependencies`:
```go
strategyReader := newAnalyticalStrategyReader(chClient, logger)
deps.Analytical = routes.AnalyticalFamilyDeps{
    GetCandleHistory:   ...,
    GetSignalHistory:   ...,
    GetDecisionHistory: ...,
    GetStrategyHistory: analyticalclient.NewGetStrategyHistoryUseCase(strategyReader, logger),
}
```

---

## 6. Integration and smoke (to be extended in S176)

### 6.1 HTTP integration test

**File**: `tests/http/analytical.http`

Add:
```http
### Strategy history — analytical
GET {{host}}/analytical/strategy/history?type=mean_reversion_entry&source=binance&symbol=BTCUSD&timeframe=60
```

### 6.2 Smoke test extension

**File**: `scripts/smoke-analytical-e2e.sh`

Add strategy table verification and endpoint validation phase:
1. Verify `strategies` table exists in ClickHouse
2. Call `GET /analytical/strategy/history` with test parameters
3. Validate response shape (`strategies` array, `source`, `meta`)

### 6.3 CI smoke

No changes to CI workflow structure. The existing smoke pipeline runs `smoke-analytical-e2e.sh`, which will exercise the new endpoint after extension.

---

## 7. Data flow summary

```
NATS (strategy.resolved.mean_reversion_entry)
  → writerConsumer (mean_reversion_entry pipeline)
    → mapStrategyRow()
      → INSERT INTO strategies (15 columns)
        → ClickHouse MergeTree (partitioned by month, ordered by source/symbol/timeframe/type/timestamp)

GET /analytical/strategy/history?type=...&source=...&symbol=...&timeframe=...
  → handler.GetStrategyHistory()
    → GetStrategyHistoryUseCase.Execute()
      → StrategyReader.QueryStrategyHistory()
        → BuildStrategyQuery() → SELECT 11 domain columns
          → scan + ParseDecisionInputsJSON + ParseMetadataJSON (×2) + FormatFloat
            → []strategy.Strategy
              → StrategyHistoryReply → JSON response
```
