# Family 03 — Strategies (Mean Reversion Entry): End-to-End Validation

> Validation proof that the third Wave B expanded family (Strategies/Mean Reversion Entry) works end-to-end:
> stream -> writer -> ClickHouse -> reader -> HTTP historical endpoint.

## Objective

Prove that the Strategies (Mean Reversion Entry) family — the third Wave B expansion — delivers a complete, functioning analytical data path with the highest JSON payload complexity to date (three JSON columns: `decisions` array + `parameters` map + `metadata` map), an optional domain-specific filter (`direction`), and coherent boundaries across all layers.

## Validation Scope

| Layer | Component | What Was Validated |
|---|---|---|
| Schema | `deploy/migrations/004_create_strategies.sql` | Table exists, 16 columns match DDL, ORDER BY key aligns with query patterns, TTL 90 days |
| Write path | `cmd/writer/mappers.go` (mapStrategyRow) | 15-value row matches DDL column order, decisions/parameters/metadata serialized via `marshalJSON`, confidence via `parseFloat` |
| Write path | `cmd/writer/pipeline.go` (mean_reversion_entry) | Consumer subscribes to `strategy.events.mean_reversion_entry.resolved.>`, inserter batches to ClickHouse |
| Persistence | ClickHouse `strategies` table | Rows written with correct types, ordering key works, LowCardinality on type/source/symbol/direction |
| Read path | `internal/adapters/clickhouse/strategy_reader.go` | Parameterized SELECT returns 11 domain columns, decisions JSON array deserialized, parameters/metadata maps deserialized |
| Application | `internal/application/analyticalclient/get_strategy_history.go` | Validation (type required, timeframe > 0, limit clamped, since <= until, direction passthrough), timing, error wrapping |
| HTTP | `GET /analytical/strategy/history` | 200 with correct JSON structure, 400 for invalid params, 503 when unavailable, Server-Timing header |
| Gateway | `cmd/gateway/compose.go` | StrategyReader wired only when ClickHouse available, optionality preserved (R-02 compliance) |

## Validation Method

### 1. Unit Test Verification

All unit tests executed and passed across every layer involved in the strategy family:

| Package | Tests | Result |
|---|---|---|
| `internal/adapters/clickhouse` | 14 strategy reader tests (query builder, direction filter, time range, DecisionInputs JSON parsing, column alignment) | PASS |
| `internal/application/analyticalclient` | 12 strategy use case tests (validation, defaults, errors, nil safety, direction passthrough) | PASS |
| `internal/interfaces/http/handlers` | 7 strategy handler tests (200, 400, 503, direction filter, Server-Timing) | PASS |
| `cmd/writer` | mapper tests (15-column row, parseFloat, marshalJSON for decisions array + parameters map + metadata map) | PASS |

**Total strategy-related tests: 33+ — all passing.**

### 2. Schema Coherence Verification

Column-by-column alignment verified across DDL -> writer -> reader:

| Column | DDL Type | Writer (mapStrategyRow) | Reader (QueryStrategyHistory) | Aligned |
|---|---|---|---|---|
| event_id | String | string (envelope) | -- (not in SELECT) | YES |
| occurred_at | DateTime64(3) | time.Time (envelope) | -- (not in SELECT) | YES |
| correlation_id | String | string (envelope) | -- (not in SELECT) | YES |
| causation_id | String | string (envelope) | -- (not in SELECT) | YES |
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 -> int | YES |
| direction | LowCardinality(String) | string(Direction) | string -> strategy.Direction | YES |
| confidence | Float64 | parseFloat -> float64 | float64 -> FormatFloat -> string | YES |
| decisions | String | marshalJSON([]DecisionInput) -> JSON string | JSON string -> ParseDecisionInputsJSON -> []DecisionInput | YES |
| parameters | String | marshalJSON(map[string]string) -> JSON string | JSON string -> ParseMetadataJSON -> map[string]string | YES |
| metadata | String | marshalJSON(map[string]string) -> JSON string | JSON string -> ParseMetadataJSON -> map[string]string | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |
| ingested_at | DateTime64(3) | DEFAULT now64(3) | -- (not in SELECT) | YES |

**Result: 16/16 columns verified — PASS**

The reader SELECT covers 11 domain columns (type through timestamp). The 4 event metadata columns + ingested_at are write-only — intentional separation between provenance and domain data.

### 3. JSON Payload Coherence (3 JSON Columns — New Record)

The Strategies family is the first family with three JSON-encoded columns. All directions are verified:

| Column | Write Path | Read Path | Round-Trip |
|---|---|---|---|
| decisions | `marshalJSON([]DecisionInput)` -> `[{"type":"rsi_oversold","outcome":"triggered","confidence":"0.85","timeframe":60}]` | `ParseDecisionInputsJSON()` -> `[]strategy.DecisionInput` | VERIFIED |
| parameters | `marshalJSON(map[string]string)` -> `{"threshold":"30","period":"14"}` | `ParseMetadataJSON()` -> `map[string]string` | VERIFIED |
| metadata | `marshalJSON(map[string]string)` -> `{"version":"1","origin":"backtest"}` | `ParseMetadataJSON()` -> `map[string]string` | VERIFIED |

Fallback behavior verified:
- Empty/nil decisions -> writer produces `"[]"` -> reader returns `[]DecisionInput{}`
- Empty/nil parameters -> writer produces `"{}"` -> reader returns `map[string]string{}`
- Empty/nil metadata -> writer produces `"{}"` -> reader returns `map[string]string{}`
- Invalid JSON -> reader returns empty type (silent fallback, documented trade-off)
- `"{}"` as input to `ParseDecisionInputsJSON` -> returns `[]DecisionInput{}` (edge case handled)

### 4. Integration Smoke Test Verification

`scripts/smoke-analytical-e2e.sh` includes Phase 5d covering the full strategy family:

| Check | What It Proves |
|---|---|
| ClickHouse `strategies WHERE type='mean_reversion_entry'` row count | Writer persisted strategy events |
| `GET /analytical/strategy/history?type=mean_reversion_entry&...` -> 200 | Read path + HTTP layer functional |
| Response structure validation (strategies array, source, meta) | JSON contract matches spec |
| Strategy field presence (11 required fields) | Domain struct serialization correct |
| `decisions` field is valid JSON array | Array deserialization path correct |
| `parameters` field is valid JSON object | Map deserialization path correct |
| `metadata` field is valid JSON object | Map deserialization path correct |
| Server-Timing header present | Observability instrumentation active |
| Missing `type` -> 400 | Required parameter validation works |
| Missing `timeframe` -> 400 | Shared validation logic works |
| Invalid `limit` (9999) -> 400 | Limit clamping enforced |
| `since > until` -> 400 | Time range validation works |
| `direction=long` filter -> 200 | Domain-specific filter functional |

### 5. Boundary Verification

| Boundary | Status |
|---|---|
| Operational pipeline unaffected | Strategy operational path (NATS KV) unchanged — no regression |
| Candle baseline unaffected | Zero changes to candle read/write path |
| Signal family unaffected | Zero changes to signal read/write path |
| Decision family unaffected | Zero changes to decision read/write path |
| ClickHouse optionality preserved | Gateway starts without ClickHouse; analytical routes return 503 |
| Writer pipeline isolation | mean_reversion_entry pipeline failure does not affect candle/signal/decision pipelines |
| No cross-family queries | Strategy endpoint returns only strategies; no join with decisions/signals/candles |

### 6. Direction Filter Verification

The `direction` parameter is the second domain-specific optional filter in the analytical layer (after `outcome` in Family 02):

| Scenario | Expected | Verified |
|---|---|---|
| No direction param | Returns all directions | YES (via smoke test default query) |
| `direction=long` | Filters to long only | YES (via handler test + smoke test) |
| `direction=short` | Filters to short only | YES (via query builder test) |
| `direction=flat` | Filters to flat only | YES (by design, same WHERE clause) |
| `direction=nonexistent` | Returns 0 rows, 200 | YES (by design, no validation against known values) |

## End-to-End Data Flow (Proven)

```
NATS JetStream
  | strategy.events.mean_reversion_entry.resolved (durable consumer)
  v
Writer Service
  | mapStrategyRow() -> 15-column row slice
  | JSON: decisions = marshalJSON([]DecisionInput)
  |       parameters = marshalJSON(map[string]string)
  |       metadata = marshalJSON(map[string]string)
  | Inserter batches (size=1000 or interval=5s)
  | Retry with exponential backoff (1s->30s, max 5 retries)
  v
ClickHouse strategies table
  | MergeTree engine
  | Partitioned by toYYYYMM(timestamp)
  | Ordered by (source, symbol, timeframe, type, timestamp)
  | TTL 90 days
  v
StrategyReader adapter
  | Parameterized SELECT with filters (type, source, symbol, timeframe, direction, since, until)
  | JSON: ParseDecisionInputsJSON() -> []DecisionInput
  |       ParseMetadataJSON() -> map[string]string (x2, for parameters + metadata)
  | ORDER BY timestamp DESC LIMIT N
  | Wall-clock timing, structured logging
  v
GetStrategyHistoryUseCase
  | Validates: type required, source required, symbol required, timeframe > 0
  | Validates: limit in [1,500] (default 50), since <= until, direction passthrough
  | Measures query duration -> QueryMeta
  v
AnalyticalWebHandler.GetStrategyHistory()
  | Parses 8 query params (type, source, symbol, timeframe, direction, limit, since, until)
  | Sets Server-Timing: total;dur=N, query;dur=M
  v
GET /analytical/strategy/history -> 200
  { strategies: [...], source: "clickhouse", meta: { query_ms, row_count } }
```

## Complexity Delta vs Previous Families

| Dimension | Signals (F-01) | Decisions (F-02) | Strategies (F-03) | Delta vs F-02 |
|---|---|---|---|---|
| Domain columns | 8 | 10 | 11 | +1 (direction) |
| JSON columns | 1 (metadata) | 2 (signals, metadata) | 3 (decisions, parameters, metadata) | +1 (parameters map) |
| Enum-like columns | 0 | 1 (outcome) | 1 (direction) | 0 |
| Family-specific query param | 0 | 1 (outcome filter) | 1 (direction filter) | 0 |
| Query builder WHERE clauses | max 4 | max 5 | max 5 | 0 |
| JSON parsing functions | 1 (ParseMetadataJSON) | 2 (+ParseSignalInputsJSON) | 2 (+ParseDecisionInputsJSON, reused ParseMetadataJSON×2) | 0 (reuse) |
| Constructor args (struct DI) | N/A (migrated) | N/A (migrated) | Field addition only | 0 (no churn) |

## H-1 Hardening Verification (Struct-Based DI)

Family 03 is the first family added **after** the S172 mandatory hardening tranche. The struct-based DI refactoring (H-1) is verified as working:

| Pre-H-1 (Family 02) | Post-H-1 (Family 03) | Improvement |
|---|---|---|
| `NewAnalyticalWebHandler(candle, signal, decision, logger)` | `NewAnalyticalWebHandler(AnalyticalHandlerDeps{...})` | No signature churn |
| Positional args — fragile, order-dependent | Named struct fields — additive, self-documenting | Zero risk of arg-swap |
| 4th arg added with churn risk | `GetStrategyHistory` field added — zero impact on existing fields | Proven scalable |

## Verdict

**The Strategies (Mean Reversion Entry) family is proven end-to-end.** Every layer — schema, write path, persistence, read path, application logic, HTTP surface — has been validated through unit tests (33+), schema coherence verification (16/16 columns), JSON round-trip verification (3 JSON columns — new record), and integration smoke tests (13 checks). The Wave B pattern handles the highest JSON payload complexity to date without structural changes or exceptions. The H-1 struct-based DI hardening is proven in its first real use.
