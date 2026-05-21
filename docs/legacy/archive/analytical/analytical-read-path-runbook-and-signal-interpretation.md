# Analytical Read Path — Runbook and Signal Interpretation

> Status: active | Introduced: S160 | Scope: operational playbooks for the analytical read path

## Quick Health Check

```bash
# Gateway operational?
curl -s http://localhost:8080/healthz

# Analytical endpoint available?
curl -s http://localhost:8080/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60

# Check timing in headers:
curl -si http://localhost:8080/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60 | grep Server-Timing
```

A healthy response includes:
- HTTP 200 with `Server-Timing` header
- `meta.query_ms` in single-digit to low double-digit milliseconds
- `meta.row_count` matching the expected data volume

## Signal Interpretation

### Response Meta Fields

| Field | Healthy Range | Investigate When |
|-------|--------------|-----------------|
| `query_ms` | 1–50ms | > 200ms consistently |
| `row_count` | Matches limit or available data | 0 when data is expected |

### Server-Timing Header

```
Server-Timing: total;dur=15, query;dur=12
```

- **total**: Full HTTP handler duration (includes validation, serialization)
- **query**: Time spent in the ClickHouse adapter only

If `total` >> `query`, the overhead is in validation/serialization (unlikely to be problematic). If `query` is high, investigate ClickHouse performance.

### Log Signals

| Log Message | Level | Meaning |
|---|---|---|
| `"clickhouse not configured, analytical endpoints disabled"` | INFO | Normal: ClickHouse optional and not configured |
| `"clickhouse connected, analytical endpoints enabled"` | INFO | Normal: ClickHouse ready |
| `"clickhouse connection failed, analytical endpoints disabled"` | WARN | ClickHouse configured but unreachable at startup |
| `"analytical query completed"` | INFO | Normal query completion with timing |
| `"analytical query failed"` | WARN | Reader returned an error (ClickHouse timeout, connection lost) |
| `"query failed"` | ERROR | Adapter-level query execution failure |
| `"scan failed"` | ERROR | Row data does not match expected column types |
| `"row iteration failed"` | ERROR | Error during result set traversal |
| `"analytical request failed"` | WARN | HTTP handler caught a problem from the use case |

## Scenario Playbooks

### Scenario 1: Analytical endpoint returns 404

**Symptoms**: `GET /analytical/evidence/candles` returns 404.

**Diagnosis**: The route is not registered. This means ClickHouse was not configured or connection failed at startup.

**Actions**:
1. Check gateway startup logs for `"clickhouse not configured"` or `"clickhouse connection failed"`.
2. Verify `clickhouse.addr` is set in `gateway.jsonc`.
3. Verify ClickHouse is reachable from the gateway container: `curl clickhouse:8123/ping`.

### Scenario 2: Analytical endpoint returns 503

**Symptoms**: Endpoint exists but returns 503 with problem body.

**Diagnosis**: ClickHouse was connected at startup but the query failed at runtime.

**Actions**:
1. Check gateway logs for `"analytical query failed"` with error details.
2. Common causes:
   - ClickHouse service restarted after gateway startup
   - Network partition between gateway and ClickHouse
   - Query timeout (large time range with no LIMIT)
3. Verify ClickHouse health: `curl clickhouse:8123/ping`.
4. If ClickHouse is healthy, check `evidence_candles` table exists: `SELECT count() FROM evidence_candles`.

### Scenario 3: Queries return 0 rows when data is expected

**Symptoms**: HTTP 200 with `"row_count": 0` and empty `candles` array.

**Diagnosis**: Query succeeded but no matching data.

**Actions**:
1. Verify writer is running and flushing: check writer `/statusz` for flush counters.
2. Check filter parameters — wrong `source`, `symbol`, or `timeframe` will return empty.
3. Check time range — `since`/`until` may exclude available data.
4. Query ClickHouse directly: `SELECT count() FROM evidence_candles WHERE source='...' AND symbol='...' AND timeframe=...`.

### Scenario 4: High query latency

**Symptoms**: `query_ms` consistently > 200ms.

**Diagnosis**: ClickHouse query performance degradation.

**Actions**:
1. Check if the table has grown significantly: `SELECT count() FROM evidence_candles`.
2. Verify the `ORDER BY` key alignment — queries filter on `(source, symbol, timeframe)` which must match the table's sort key.
3. Check ClickHouse system tables: `SELECT * FROM system.query_log ORDER BY event_time DESC LIMIT 10`.
4. Consider TTL expiration — stale data beyond 90 days should be auto-purged.

### Scenario 5: Scan errors in logs

**Symptoms**: ERROR logs with `"scan failed"` messages.

**Diagnosis**: Schema drift — the ClickHouse table columns no longer match the reader's expected types.

**Actions**:
1. Compare the reader's column list (12 columns: source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final) against the actual DDL.
2. Check if a migration altered column types without updating the reader.
3. This is a code/schema alignment bug that requires a fix — it will not self-heal.

## What This Runbook Does NOT Cover

- **Writer-side diagnostics**: See `analytical-runtime-runbook-and-signal-interpretation.md` for writer pipeline health.
- **ClickHouse cluster operations**: Replication, sharding, and storage management are outside gateway scope.
- **Load testing**: No concurrency benchmarks or saturation points are established yet.
- **Alerting thresholds**: No automated alerting is configured; all signals require manual log review.
- **Multi-endpoint correlation**: Only one analytical endpoint exists (`/analytical/evidence/candles`). Future endpoints should follow the same diagnostic pattern.
