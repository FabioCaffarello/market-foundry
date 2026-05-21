# Wave B Family-01 End-to-End Validation — Signals (RSI)

> Validation proof that the first Wave B expanded family (Signals/RSI) works end-to-end:
> stream → writer → ClickHouse → reader → HTTP historical endpoint.

## Objective

Prove that the Signal (RSI) family — the first Wave B expansion beyond the baseline candle family — delivers a complete, functioning analytical data path with no gaps, no silent failures, and coherent boundaries across all layers.

## Validation Scope

| Layer | Component | What Was Validated |
|---|---|---|
| Schema | `deploy/migrations/002_create_signals.sql` | Table exists, columns match DDL, migration recorded in `_migrations` |
| Write path | `cmd/writer/pipeline.go` (rsi pipeline) | Consumer subscribes to `signal.events.rsi.generated`, mapper produces correct row, inserter batches and flushes to ClickHouse |
| Persistence | ClickHouse `signals` table | Rows appear with correct types, ordering key works, TTL defined |
| Read path | `internal/adapters/clickhouse/signal_reader.go` | Parameterized SELECT returns domain structs, metadata JSON deserialized |
| Application | `internal/application/analyticalclient/get_signal_history.go` | Validation (type required, limit clamped, since/until ordering), timing, error handling |
| HTTP | `GET /analytical/signal/history` | 200 with correct JSON structure, 400 for invalid params, Server-Timing header |
| Gateway | `cmd/gateway/compose.go` | SignalReader wired only when ClickHouse is available, optionality preserved |

## Validation Method

### 1. Unit Test Verification

All unit tests executed and passed across every layer involved in the signal family:

| Package | Tests | Result |
|---|---|---|
| `internal/adapters/clickhouse` | 12 signal reader tests (query builder, metadata parsing, column alignment) | PASS |
| `internal/application/analyticalclient` | 10 signal use case tests (validation, defaults, errors, nil) | PASS |
| `internal/interfaces/http/handlers` | 6 signal handler tests (200, 400, 503, Server-Timing) | PASS |
| `cmd/writer` | mapper tests (12-column row, parseFloat, marshalJSON), inserter tests, supervisor tests | PASS |
| `cmd/gateway` | compile-time interface assertion (SignalReader) | PASS |
| `internal/migrate` | migration catalog and checksum tests | PASS |

**Total signal-related tests: 29+ — all passing.**

### 2. Schema Coherence Verification

Column-by-column alignment verified across DDL → writer → reader:

| Column | DDL Type | Writer (mapSignalRow) | Reader (QuerySignalHistory) | Aligned |
|---|---|---|---|---|
| event_id | String | string (envelope) | — (not in SELECT) | YES |
| occurred_at | DateTime64(3) | time.Time (envelope) | — (not in SELECT) | YES |
| correlation_id | String | string (envelope) | — (not in SELECT) | YES |
| causation_id | String | string (envelope) | — (not in SELECT) | YES |
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 | YES |
| value | Float64 | parseFloat→float64 | float64 | YES |
| metadata | String | marshalJSON→string | string→ParseMetadataJSON | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |

**Result: 12/12 columns verified — PASS**

Note: The reader SELECT covers 8 domain columns (type through timestamp). The 4 metadata columns (event_id, occurred_at, correlation_id, causation_id) are written by the writer for provenance but are not exposed in the read path. This is intentional — the analytical read path returns domain-relevant data only.

### 3. Integration Smoke Test

`scripts/smoke-analytical-e2e.sh` extended with Phase 5b covering the full signal family:

| Check | What It Proves |
|---|---|
| ClickHouse `signals WHERE type='rsi'` row count | Writer persisted RSI signal events |
| `GET /analytical/signal/history?type=rsi&...` → 200 | Read path + HTTP layer functional |
| Response structure validation (signals array, source, meta) | JSON contract matches spec |
| Signal field presence (type, source, symbol, timeframe, value, metadata, final, timestamp) | Domain struct serialization correct |
| Server-Timing header present | Observability instrumentation active |
| Missing `type` → 400 | Required parameter validation works |
| Missing `timeframe` → 400 | Shared validation logic works for signals |
| Invalid `limit` → 400 | Limit clamping enforced |
| `since > until` → 400 | Time range validation works |

### 4. Boundary Verification

| Boundary | Status |
|---|---|
| Operational pipeline unaffected | Signal operational path (NATS KV) unchanged — no regression |
| ClickHouse optionality preserved | Gateway starts without ClickHouse; analytical routes return 503 when unavailable |
| Writer pipeline isolation | RSI signal pipeline failure does not affect candle pipeline (supervisor restarts independently) |
| No cross-family queries | Signal endpoint returns only signals; no join with candles or other tables |

## End-to-End Data Flow (Proven)

```
NATS JetStream
  │ signal.events.rsi.generated (durable consumer)
  ▼
Writer Service
  │ mapSignalRow() → 12-column row slice
  │ Inserter batches (size=1000 or interval=5s)
  │ Retry with exponential backoff (1s→30s, max 5 retries)
  ▼
ClickHouse signals table
  │ MergeTree engine
  │ Partitioned by toYYYYMM(timestamp)
  │ Ordered by (source, symbol, timeframe, type, timestamp)
  │ TTL 90 days
  ▼
SignalReader adapter
  │ Parameterized SELECT with filters (type, source, symbol, timeframe, since, until)
  │ ORDER BY timestamp DESC LIMIT N
  │ Wall-clock timing, structured logging
  ▼
GetSignalHistoryUseCase
  │ Validates: type required, limit ∈ [1,500], since ≤ until
  │ Measures query duration → QueryMeta
  ▼
AnalyticalWebHandler.GetSignalHistory()
  │ Parses query params, delegates to use case
  │ Sets Server-Timing: total;dur=N, query;dur=M
  ▼
GET /analytical/signal/history → 200
  { signals: [...], source: "clickhouse", meta: { query_ms, row_count } }
```

## Verdict

**The Signal (RSI) family is proven end-to-end.** Every layer — schema, write path, persistence, read path, application logic, HTTP surface — has been validated through unit tests, schema coherence checks, and integration smoke tests. The pattern established in S163/S164 is confirmed functional and the base is ready for pattern hardening before the second family.
