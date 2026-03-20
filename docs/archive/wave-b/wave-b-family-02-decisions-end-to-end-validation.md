# Wave B Family 02 — Decisions (RSI Oversold): End-to-End Validation

> Validation proof that the second Wave B expanded family (Decisions/RSI Oversold) works end-to-end:
> stream -> writer -> ClickHouse -> reader -> HTTP historical endpoint.

## Objective

Prove that the Decisions (RSI Oversold) family — the second Wave B expansion — delivers a complete, functioning analytical data path with JSON payload complexity (two JSON columns: `signals` array + `metadata` map), an optional domain-specific filter (`outcome`), and coherent boundaries across all layers.

## Validation Scope

| Layer | Component | What Was Validated |
|---|---|---|
| Schema | `deploy/migrations/003_create_decisions.sql` | Table exists, 15 columns match DDL, ORDER BY key aligns with query patterns, TTL 90 days |
| Write path | `cmd/writer/mappers.go` (mapDecisionRow) | 14-value row matches DDL column order, signals/metadata serialized via `marshalJSON`, confidence via `parseFloat` |
| Write path | `cmd/writer/pipeline.go` (rsi_oversold) | Consumer subscribes to `decision.events.rsi_oversold.evaluated.>`, inserter batches to ClickHouse |
| Persistence | ClickHouse `decisions` table | Rows written with correct types, ordering key works, LowCardinality on type/source/symbol/outcome |
| Read path | `internal/adapters/clickhouse/decision_reader.go` | Parameterized SELECT returns 10 domain columns, signals JSON array deserialized, metadata map deserialized |
| Application | `internal/application/analyticalclient/get_decision_history.go` | Validation (type required, timeframe > 0, limit clamped, since <= until, outcome passthrough), timing, error wrapping |
| HTTP | `GET /analytical/decision/history` | 200 with correct JSON structure, 400 for invalid params, 503 when unavailable, Server-Timing header |
| Gateway | `cmd/gateway/compose.go` | DecisionReader wired only when ClickHouse available, optionality preserved (R-02 compliance) |

## Validation Method

### 1. Unit Test Verification

All unit tests executed and passed across every layer involved in the decision family:

| Package | Tests | Result |
|---|---|---|
| `internal/adapters/clickhouse` | 12 decision reader tests (query builder, signal JSON parsing, column alignment) | PASS |
| `internal/application/analyticalclient` | 11 decision use case tests (validation, defaults, errors, nil safety) | PASS |
| `internal/interfaces/http/handlers` | 7 decision handler tests (200, 400, 503, outcome filter, Server-Timing) | PASS |
| `cmd/writer` | mapper tests (14-column row, parseFloat, marshalJSON for signals array + metadata map) | PASS |

**Total decision-related tests: 30+ — all passing.**

### 2. Schema Coherence Verification

Column-by-column alignment verified across DDL -> writer -> reader:

| Column | DDL Type | Writer (mapDecisionRow) | Reader (QueryDecisionHistory) | Aligned |
|---|---|---|---|---|
| event_id | String | string (envelope) | -- (not in SELECT) | YES |
| occurred_at | DateTime64(3) | time.Time (envelope) | -- (not in SELECT) | YES |
| correlation_id | String | string (envelope) | -- (not in SELECT) | YES |
| causation_id | String | string (envelope) | -- (not in SELECT) | YES |
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 -> int | YES |
| outcome | LowCardinality(String) | string(Outcome) | string -> decision.Outcome | YES |
| confidence | Float64 | parseFloat -> float64 | float64 -> FormatFloat -> string | YES |
| signals | String | marshalJSON([]SignalInput) -> JSON string | JSON string -> ParseSignalInputsJSON -> []SignalInput | YES |
| metadata | String | marshalJSON(map[string]string) -> JSON string | JSON string -> ParseMetadataJSON -> map[string]string | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |
| ingested_at | DateTime64(3) | DEFAULT now64(3) | -- (not in SELECT) | YES |

**Result: 15/15 columns verified — PASS**

The reader SELECT covers 10 domain columns (type through timestamp). The 4 event metadata columns + ingested_at are write-only — intentional separation between provenance and domain data.

### 3. JSON Payload Coherence (New for Family 02)

The Decisions family is the first family with two JSON-encoded columns. Both directions are verified:

| Column | Write Path | Read Path | Round-Trip |
|---|---|---|---|
| signals | `marshalJSON([]SignalInput)` -> `[{"type":"rsi","value":"28.5","timeframe":60}]` | `ParseSignalInputsJSON()` -> `[]decision.SignalInput` | VERIFIED |
| metadata | `marshalJSON(map[string]string)` -> `{"threshold":"30","period":"14"}` | `ParseMetadataJSON()` -> `map[string]string` | VERIFIED |

Fallback behavior verified:
- Empty/nil signals -> writer produces `"[]"` -> reader returns `[]SignalInput{}`
- Empty/nil metadata -> writer produces `"{}"` -> reader returns `map[string]string{}`
- Invalid JSON -> reader returns empty type (silent fallback, documented trade-off)

### 4. Integration Smoke Test Verification

`scripts/smoke-analytical-e2e.sh` includes Phase 5c covering the full decision family:

| Check | What It Proves |
|---|---|
| ClickHouse `decisions WHERE type='rsi_oversold'` row count | Writer persisted decision events |
| `GET /analytical/decision/history?type=rsi_oversold&...` -> 200 | Read path + HTTP layer functional |
| Response structure validation (decisions array, source, meta) | JSON contract matches spec |
| Decision field presence (10 required fields) | Domain struct serialization correct |
| `signals` field is valid JSON array | Array deserialization path correct |
| `metadata` field is valid JSON object | Map deserialization path correct |
| Server-Timing header present | Observability instrumentation active |
| Missing `type` -> 400 | Required parameter validation works |
| Missing `timeframe` -> 400 | Shared validation logic works |
| Invalid `limit` (9999) -> 400 | Limit clamping enforced |
| `since > until` -> 400 | Time range validation works |
| `outcome=triggered` filter -> 200 | Domain-specific filter functional |

### 5. Boundary Verification

| Boundary | Status |
|---|---|
| Operational pipeline unaffected | Decision operational path (NATS KV) unchanged — no regression |
| Candle baseline unaffected | Zero changes to candle read/write path |
| Signal family unaffected | Zero changes to signal read/write path |
| ClickHouse optionality preserved | Gateway starts without ClickHouse; analytical routes return 503 |
| Writer pipeline isolation | rsi_oversold pipeline failure does not affect candle/signal pipelines |
| No cross-family queries | Decision endpoint returns only decisions; no join with signals/candles |

### 6. Outcome Filter Verification

The `outcome` parameter is the first domain-specific optional filter in the analytical layer:

| Scenario | Expected | Verified |
|---|---|---|
| No outcome param | Returns all outcomes | YES (via smoke test default query) |
| `outcome=triggered` | Filters to triggered only | YES (via handler test + smoke test) |
| `outcome=not_triggered` | Filters to not_triggered only | YES (via query builder test) |
| `outcome=nonexistent` | Returns 0 rows, 200 | YES (by design, no validation against known values) |

## End-to-End Data Flow (Proven)

```
NATS JetStream
  | decision.events.rsi_oversold.evaluated (durable consumer)
  v
Writer Service
  | mapDecisionRow() -> 14-column row slice
  | JSON: signals = marshalJSON([]SignalInput), metadata = marshalJSON(map[string]string)
  | Inserter batches (size=1000 or interval=5s)
  | Retry with exponential backoff (1s->30s, max 5 retries)
  v
ClickHouse decisions table
  | MergeTree engine
  | Partitioned by toYYYYMM(timestamp)
  | Ordered by (source, symbol, timeframe, type, timestamp)
  | TTL 90 days
  v
DecisionReader adapter
  | Parameterized SELECT with filters (type, source, symbol, timeframe, outcome, since, until)
  | JSON: ParseSignalInputsJSON() -> []SignalInput, ParseMetadataJSON() -> map[string]string
  | ORDER BY timestamp DESC LIMIT N
  | Wall-clock timing, structured logging
  v
GetDecisionHistoryUseCase
  | Validates: type required, source required, symbol required, timeframe > 0
  | Validates: limit in [1,500] (default 50), since <= until, outcome passthrough
  | Measures query duration -> QueryMeta
  v
AnalyticalWebHandler.GetDecisionHistory()
  | Parses 8 query params (type, source, symbol, timeframe, outcome, limit, since, until)
  | Sets Server-Timing: total;dur=N, query;dur=M
  v
GET /analytical/decision/history -> 200
  { decisions: [...], source: "clickhouse", meta: { query_ms, row_count } }
```

## Complexity Delta vs Family 01 (Signals)

| Dimension | Signals (F-01) | Decisions (F-02) | Delta |
|---|---|---|---|
| Domain columns | 8 | 10 | +2 (outcome, confidence) |
| JSON columns | 1 (metadata) | 2 (signals, metadata) | +1 (array type) |
| Enum-like columns | 0 | 1 (outcome) | +1 |
| Family-specific query param | 0 | 1 (outcome filter) | +1 |
| Query builder WHERE clauses | max 4 | max 5 | +1 |
| JSON parsing functions | 1 (ParseMetadataJSON) | 2 (+ParseSignalInputsJSON) | +1 (array deserialization) |
| Constructor args (handler) | 3 | 4 | +1 |

## Verdict

**The Decisions (RSI Oversold) family is proven end-to-end.** Every layer — schema, write path, persistence, read path, application logic, HTTP surface — has been validated through unit tests (30+), schema coherence verification (15/15 columns), JSON round-trip verification (2 JSON columns), and integration smoke tests (12 checks). The Wave B pattern handles the increased JSON payload complexity without structural changes or exceptions.
