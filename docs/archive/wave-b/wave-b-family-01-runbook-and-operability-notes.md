# Wave B Family-01 — Runbook and Operability Notes (Signals)

## Operational Overview

The signal analytical read path is a new additive endpoint that queries the `signals` ClickHouse table. It follows the same operational model as the candle read path established in S149/S160.

## Health Verification

### 1. Is the signal write path active?

Check writer service `/statusz` or `/diagz`:
```bash
curl -s http://127.0.0.1:8085/statusz | jq '.pipelines.rsi'
```

Expected: pipeline state `active`, `events_flushed > 0`.

### 2. Is data reaching ClickHouse?

```bash
docker exec -it clickhouse clickhouse-client --query \
  "SELECT count(), min(timestamp), max(timestamp) FROM signals WHERE type='rsi'"
```

Expected: non-zero count with recent timestamps.

### 3. Is the analytical endpoint responding?

```bash
curl -s "http://127.0.0.1:8080/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60&limit=5" | jq '.meta'
```

Expected: `{"query_ms": <low>, "row_count": <n>}` with `source: "clickhouse"`.

### 4. Is Server-Timing present?

```bash
curl -sI "http://127.0.0.1:8080/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60" | grep Server-Timing
```

Expected: `Server-Timing: total;dur=<ms>, query;dur=<ms>`.

## Failure Scenarios

### Scenario 1: ClickHouse down or unreachable

**Symptom:** GET `/analytical/signal/history` returns 503.
**Cause:** Gateway failed to connect to ClickHouse at startup, or connection dropped.
**Diagnostic:** Check gateway logs for `clickhouse connection failed` or `clickhouse not configured`.
**Recovery:** Restart gateway after ClickHouse is available. The gateway does NOT auto-reconnect (sticky degradation by design).

### Scenario 2: Empty results despite active pipeline

**Symptom:** 200 OK with `signals: []` and `row_count: 0`.
**Possible causes:**
1. **Wrong query filters** — verify `type`, `source`, `symbol`, `timeframe` match what the writer is receiving.
2. **Batch not flushed** — writer batches signals (default: 1000 rows or 5s interval). Wait for flush.
3. **Time range mismatch** — `since`/`until` may exclude all data. Try without time filters.
4. **Pipeline degraded** — check writer `/statusz` for pipeline state.

**Diagnostic:**
```bash
# Check what's in ClickHouse
docker exec -it clickhouse clickhouse-client --query \
  "SELECT type, source, symbol, timeframe, count() FROM signals GROUP BY type, source, symbol, timeframe"
```

### Scenario 3: Metadata field empty or `{}`

**Symptom:** Signals returned with `metadata: {}` despite RSI signals having `period`, `avg_gain`, `avg_loss`.
**Possible causes:**
1. **Write-path issue** — `marshalJSON` returned `"{}"` due to nil metadata on the signal event.
2. **Read-path fallback** — `ParseMetadataJSON` silently returns empty map on invalid JSON.

**Diagnostic:** Check the raw stored data:
```bash
docker exec -it clickhouse clickhouse-client --query \
  "SELECT metadata FROM signals WHERE type='rsi' LIMIT 5"
```

### Scenario 4: Slow queries

**Symptom:** `Server-Timing: query;dur=<high>` consistently > 100ms.
**Possible causes:**
1. **Large result set** — reduce `limit` parameter.
2. **Missing partition pruning** — queries should hit a single partition (timestamp-based). Verify `since`/`until` are set.
3. **ClickHouse under load** — check ClickHouse resource usage.

**Diagnostic:** Check `query_ms` in response `meta` field.

## Observability Parity

The signal read path inherits the same observability model as the candle read path:

| Layer | Observable | Signal Family |
|---|---|---|
| Adapter (signal_reader.go) | Query timing via `slog.Debug` | YES |
| Adapter (signal_reader.go) | Error logging via `slog.Error` | YES |
| Use case (get_signal_history.go) | Timing + row count via `slog.Info` | YES |
| Use case (get_signal_history.go) | Failure logging via `slog.Warn` | YES |
| Handler (analytical.go) | `Server-Timing` response header | YES |
| Handler (analytical.go) | `QueryMeta` in JSON response | YES |
| Writer (pipeline/inserter) | `events_flushed`, `events_dropped`, `buffer_depth` | Pre-existing |

## Known Gaps

1. **No Prometheus/OpenTelemetry** — same as candle path. Structured logging only. Revisit when team grows or volume 10x.
2. **No request counting** — no middleware counts analytical requests. Acceptable at current scale.
3. **No per-signal-type metrics** — all signal types share the same logging. If multiple signal types are active, logs must be filtered by `signal_type` field.
4. **No alerting** — no thresholds on query latency or error rate. Manual monitoring only.
