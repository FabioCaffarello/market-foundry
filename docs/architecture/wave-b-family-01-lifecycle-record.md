# Wave B Family 01 Lifecycle Record -- Signals (RSI)

**Family:** Signals (RSI)
**Wave B Iteration:** 01
**Stages:** S163-S167
**Status:** Complete -- proven end-to-end

---

## Definition

**Selected:** Signals (RSI) -- the first Wave B expansion beyond the baseline candle family.

**Selection rationale:**
1. Write path already active -- writer consumes `signal.events.rsi.generated` and inserts via `mapSignalRow()`. Only the read path needed building.
2. Simplest domain type after candles -- 8 domain fields vs 16+ for evidence candles. Metadata (`map[string]string` as JSON) is the only new concern.
3. Schema already exists -- migration `002_create_signals.sql` applied in S147. No new DDL required.
4. Dependency chain root -- signals are Layer 1 (depends on evidence). Validates the simplest non-evidence family before more complex ones.

**Complexity profile:** 8 domain columns, 1 JSON column (metadata), 1 enum-like column (type), 1 float column (value).

---

## Schema & Gateway

### DDL Columns (`deploy/migrations/002_create_signals.sql`)

| Column | DDL Type | Writer (mapSignalRow) | Reader (QuerySignalHistory) | Aligned |
|---|---|---|---|---|
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 | YES |
| value | Float64 | float64 (parseFloat) | float64 (FormatFloat) | YES |
| metadata | String | string (marshalJSON) | string (ParseMetadataJSON) | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |

Event metadata columns (event_id, occurred_at, correlation_id, causation_id) are write-only provenance -- not exposed in the read path.

**Schema coherence: 8/8 domain columns verified across DDL, writer, and reader.**

### Data Flow

```
NATS JetStream (signal.events.rsi.generated)
  -> writerConsumer -> mapSignalRow() -> INSERT INTO signals (batch)
  -> signals MergeTree table (partitioned by toYYYYMM, TTL 90 days)
  -> SignalReader.QuerySignalHistory() (parameterized SELECT)
  -> GetSignalHistoryUseCase (validation, timing)
  -> GET /analytical/signal/history -> 200 JSON + Server-Timing
```

### Endpoint Specification

```
GET /analytical/signal/history
  Required: type, source, symbol, timeframe
  Optional: limit (1-500, default 50), since, until (unix seconds)
  Response: { signals: [...], source: "clickhouse", meta: { query_ms, row_count } }
  Headers: Server-Timing: total;dur=N, query;dur=M
  Errors: 400 (invalid params), 503 (ClickHouse unavailable)
```

### Gateway Composition

SignalReader wired only when ClickHouse is available. Optionality preserved -- gateway starts without ClickHouse; analytical routes return 503.

---

## Implementation

### New Artifacts (4 files)

| Artifact | File |
|---|---|
| Reader adapter | `internal/adapters/clickhouse/signal_reader.go` |
| Reader tests | `internal/adapters/clickhouse/signal_reader_test.go` |
| Use case | `internal/application/analyticalclient/get_signal_history.go` |
| Use case tests | `internal/application/analyticalclient/get_signal_history_test.go` |

### Modified Artifacts (8 files)

- `internal/application/analyticalclient/contracts.go` -- added SignalHistoryQuery, SignalHistoryReply
- `internal/interfaces/http/handlers/analytical.go` -- added GetSignalHistory handler
- `internal/interfaces/http/handlers/analytical_test.go` -- added 6 signal handler tests
- `internal/interfaces/http/routes/analytical.go` -- added signal route
- `cmd/gateway/analytical_reader.go` -- added newAnalyticalSignalReader() factory
- `cmd/gateway/analytical_reader_test.go` -- compile-time interface assertion
- `cmd/gateway/compose.go` -- wired SignalReader
- `tests/http/analytical.http` -- added 7 signal HTTP test requests

### Design Decisions

1. **Signal type as query filter (not path parameter)** -- analytical endpoints use a flat `/analytical/` namespace. Type is mandatory in the query to scope to a specific signal family.
2. **Metadata deserialization with silent fallback** -- `ParseMetadataJSON` returns empty map on invalid JSON, matching write-path fallback. Corruption is silent at read time.
3. **Shared query parameter parsing** -- both candle and signal handlers reuse `parseEvidenceKeyParams()`. Naming residue accepted under C-9 additive-only constraint.
4. **No signal-type validation** -- reader accepts any type string; ClickHouse returns empty for unknown types. Avoids coupling read path to settings registry.

---

## Validation

### Unit Tests: 29+ signal-related tests -- all passing

| Package | Tests |
|---|---|
| `internal/adapters/clickhouse` | 12 signal reader tests |
| `internal/application/analyticalclient` | 10 signal use case tests |
| `internal/interfaces/http/handlers` | 6 signal handler tests |
| `cmd/writer` | mapper tests |

### Integration Smoke

`scripts/smoke-analytical-e2e.sh` Phase 5b:
- ClickHouse row count for `signals WHERE type='rsi'`
- HTTP 200 with correct JSON structure
- All 8 domain fields present in response
- Server-Timing header present
- 400 for missing type, missing timeframe, invalid limit, since > until

### Boundary Verification

- Operational pipeline (NATS KV) unchanged -- no regression
- ClickHouse optionality preserved -- 503 when unavailable
- Writer pipeline isolation -- RSI signal pipeline failure does not affect candle pipeline
- No cross-family queries

---

## Runtime & Operability

### Health Verification

```bash
# Writer pipeline active?
curl -s http://127.0.0.1:8085/statusz | jq '.pipelines.rsi'

# Data reaching ClickHouse?
docker exec -it clickhouse clickhouse-client --query \
  "SELECT count(), min(timestamp), max(timestamp) FROM signals WHERE type='rsi'"

# Endpoint responding?
curl -s "http://127.0.0.1:8080/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60&limit=5" | jq '.meta'
```

### Failure Scenarios

| Failure | Symptom | Recovery |
|---|---|---|
| ClickHouse down | 503 on endpoint | Restart gateway after ClickHouse available (sticky degradation) |
| Empty results despite active pipeline | 200 with 0 rows | Check query filters, wait for batch flush, check writer /statusz |
| Metadata field empty | metadata: {} | Check writer marshalJSON output, verify raw ClickHouse data |
| Slow queries | Server-Timing query;dur > 100ms | Reduce limit, verify time range filters set, check ClickHouse resources |

### Observability

Identical model across adapter (slog.Debug timing, slog.Error), use case (slog.Info timing + row count, slog.Warn failure), handler (Server-Timing header, QueryMeta JSON). No Prometheus/OpenTelemetry -- structured logging only.

---

## Findings

### What Worked
- F-1: Wave B expansion pattern works as designed -- 9-artifact pattern produced a functioning family with no structural surprises.
- F-2: Schema coherence provable through unit tests alone -- no runtime ClickHouse needed.
- F-3: Write path required zero changes -- writer correctly designed as multi-family service from inception.
- F-4: Observability parity achieved mechanically -- copy pattern, get same instrumentation.
- F-5: Error handling contracts consistent across families.
- F-6: Metadata JSON adds exactly one new concern (JSON deserialization beyond primitives).

### Pattern Frictions Identified
- PF-1: `parseEvidenceKeyParams()` naming residue -- accepted, rename at third family.
- PF-2: Constructor accumulation in AnalyticalWebHandler (2 args) -- manageable, struct DI at third family.
- PF-3: ~80% mechanical duplication across families -- accepted through 3 families.
- PF-4: No signal-type validation against known families -- accepted (empty results, no risk).
- PF-5: Smoke test grows linearly with families -- restructure after third family.
- PF-6: No automated CI integration for analytical smoke test -- blocking prerequisite before second family.

### Limits
- Only RSI signals tested (EMA crossover depends on actor config).
- No load testing, concurrent query testing, or pagination beyond 500 rows.
- Metadata schema not validated at read time.

---

*Consolidated from: wave-b-family-01-end-to-end-validation.md, wave-b-family-01-implementation-notes.md, wave-b-family-01-runbook-and-operability-notes.md, wave-b-family-01-schema-writer-reader-gateway-path.md, wave-b-family-01-validation-findings-and-pattern-frictions.md*
