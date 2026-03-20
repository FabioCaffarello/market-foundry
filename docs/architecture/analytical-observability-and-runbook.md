# Analytical Observability and Runbook

> Consolidated from 5 source documents (archived in docs/archive/analytical/).
> Sources: analytical-observability-and-diagnostics-hardening.md, analytical-read-path-observability-and-reliability.md, analytical-read-path-runbook-and-signal-interpretation.md, analytical-runtime-runbook-and-signal-interpretation.md, analytical-reader-adapter-test-scope-and-limits.md

---

## 1. Design Principles

1. **Minimal useful signals** -- every counter and gauge answers a specific operational question
2. **No external dependencies** -- all signals via `/statusz`, `/diagz`, and structured logs; no OpenTelemetry, Prometheus, or Grafana
3. **Counter semantics reused for gauges** -- Tracker's `Counter(name)` returns `*atomic.Int64` supporting both `Add()` and `Store()`
4. **Log-first debugging** -- significant state transitions emit structured log entries

---

## 2. Write Path Signal Catalog

### Consumer Trackers (per pipeline family)

| Signal | Type | Meaning |
|--------|------|---------|
| `event_count` | counter | Total events received from NATS |
| `error_count` | counter | Total errors recorded |
| `events_received` | counter | Events decoded and forwarded to inserter |
| `pipeline_restarts` | counter | Supervisor-initiated restart attempts |
| `pipeline_degraded` | counter | Set to 1 when restart budget exhausted |

### Inserter Trackers (per pipeline family)

| Signal | Type | Meaning |
|--------|------|---------|
| `event_count` | counter | Successful flush operations |
| `error_count` | counter | Flush failures after retry exhaustion |
| `events_flushed` | counter | Rows successfully inserted into ClickHouse |
| `flush_total` | counter | Number of successful batch flushes |
| `flush_duration_ms` | gauge | Duration of last flush operation (ms) |
| `flush_failures` | counter | Batch drops after retry exhaustion |
| `events_dropped` | counter | Rows permanently lost (overflow + flush exhaustion) |
| `events_overflowed` | counter | Rows evicted due to buffer overflow |
| `buffer_depth` | gauge | Current number of rows buffered |

### Signal Relationships

```
Consumer                          Inserter
---------                         --------
events_received --> [actor msg] --> buffer_depth
                                    |
                                    +--> events_flushed (success)
                                    |    flush_total
                                    |    flush_duration_ms
                                    |
                                    +--> events_overflowed (buffer full)
                                    |    events_dropped
                                    |
                                    +--> flush_failures (retry exhausted)
                                         events_dropped

Supervisor
----------
pipeline_restarts --> pipeline_degraded (budget exhausted)
```

### Key Invariants

1. `events_received` (consumer) >= `events_flushed` (inserter) -- always true
2. `events_dropped` = overflow losses + flush failure losses
3. `events_overflowed` <= `events_dropped` (overflow is a subset)
4. `flush_total` * average_batch_size ~ `events_flushed`
5. `pipeline_degraded == 0` for all trackers when `phase != "degraded"`

---

## 3. Read Path Signal Catalog

### Layer 1 -- ClickHouse Adapter (`candle_reader.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Query completed | DEBUG | `source`, `symbol`, `timeframe`, `rows`, `elapsed_ms` |
| Query failed | ERROR | `source`, `symbol`, `timeframe`, `elapsed_ms`, `error` |
| Scan failed | ERROR | `source`, `symbol`, `timeframe`, `error` |
| Row iteration failed | ERROR | `source`, `symbol`, `timeframe`, `error` |

### Layer 2 -- Use Case (`get_candle_history.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Query completed | INFO | `source`, `symbol`, `timeframe`, `rows`, `query_ms` |
| Query failed | WARN | `source`, `symbol`, `timeframe`, `elapsed_ms`, `error` |

Populates `QueryMeta` in reply: `query_ms` and `row_count`, flowing through the HTTP response.

### Layer 3 -- HTTP Handler (`analytical.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Request failed | WARN | `source`, `symbol`, `timeframe`, `total_ms`, `problem` |

Adds `Server-Timing` header to successful responses:
```
Server-Timing: total;dur=15, query;dur=12
```

### Response Enrichment

Every successful analytical response includes:
```json
{
  "candles": [...],
  "source": "clickhouse",
  "meta": {
    "query_ms": 12,
    "row_count": 50
  }
}
```

---

## 4. Health Endpoints

| Endpoint | Signal | Meaning |
|----------|--------|---------|
| `/healthz` | 200 | Process alive |
| `/readyz` | checks | NATS + ClickHouse connectivity |
| `/statusz` | `phase` | Aggregate: starting, warming, active, idle, stalled, degraded |
| `/statusz` | `degraded_trackers` | Tracker names with `pipeline_degraded > 0` |
| `/statusz` | per-tracker `counters` | Full counter/gauge snapshot |
| `/diagz` | `readiness_checks` | Pass/fail for NATS and ClickHouse |
| `/diagz` | per-tracker `counters` | Same snapshot as `/statusz` |

---

## 5. Structured Log Signals

### Write Path

| Level | Event | Key Fields |
|-------|-------|------------|
| DEBUG | `batch flushed` | family, table, rows, flush_ms |
| WARN | `flush attempt failed` | error, family, table, rows, attempt, max_retries |
| WARN | `component idle` | tracker, idle_seconds, event_count, error_count |
| WARN | `pipeline failure -- scheduling restart` | family, error, restart, max_restarts, backoff |
| ERROR | `buffer overflow` | family, evicted, buffer_depth |
| ERROR | `flush failed -- retries exhausted` | error, family, rows_dropped, attempts, flush_ms |
| ERROR | `pipeline degraded` | family, restarts, last_error |

### Read Path

| Level | Event | Key Fields |
|-------|-------|------------|
| INFO | `clickhouse connected, analytical endpoints enabled` | addr, database |
| INFO | `clickhouse not configured, analytical endpoints disabled` | -- |
| INFO | `analytical query completed` | source, symbol, timeframe, rows, query_ms |
| WARN | `clickhouse connection failed, analytical endpoints disabled` | addr |
| WARN | `analytical query failed` | source, symbol, timeframe, elapsed_ms, error |
| WARN | `analytical request failed` | source, symbol, timeframe, total_ms, problem |
| ERROR | `query failed` | source, symbol, timeframe, elapsed_ms, error |
| ERROR | `scan failed` | source, symbol, timeframe, error |

---

## 6. Operational Questions Answered

| Question | How to Answer |
|----------|---------------|
| Is the writer alive? | `GET /healthz` -> 200 |
| Is ClickHouse reachable? | `GET /readyz` -> check `clickhouse` status |
| Are all pipelines healthy? | `GET /statusz` -> `phase != "degraded"` and `degraded_trackers` empty |
| Which pipeline is degraded? | `GET /statusz` -> `degraded_trackers` array |
| Is data flowing? | Compare `events_received` (consumer) vs `events_flushed` (inserter) |
| Is backpressure building? | Check `buffer_depth` on inserter trackers |
| Are we losing data? | Check `events_dropped` and `events_overflowed` |
| How fast are flushes? | Check `flush_duration_ms` gauge |
| How fast are queries? | Check `meta.query_ms` in response or `Server-Timing` header |

---

## 7. Phase Interpretation

| Phase | Meaning | Action |
|-------|---------|--------|
| `starting` | Process started <30s ago, no events yet | Wait. Normal cold start. |
| `warming` | At least one pipeline awaiting first event | Check upstream runtimes. Investigate if >2min. |
| `active` | All pipelines receiving and processing | Healthy. No action. |
| `idle` | At least one pipeline idle beyond threshold (2min) | Check upstream events. May be normal. |
| `stalled` | All pipelines idle beyond threshold | Likely upstream issue. Check ingest/derive/store. |
| `degraded` | At least one pipeline exhausted restart budget | Investigate. Fix root cause. Restart writer. |

---

## 8. Scenario Playbooks

### Writer: Pipeline Degraded

**Symptoms:** `/statusz` shows `phase: "degraded"`, `degraded_trackers` lists affected families.

**Actions:**
1. Check `/statusz` for trackers with `pipeline_degraded > 0`
2. Check logs: `grep "pipeline degraded"` for `last_error`
3. Common causes: NATS connection failure, invalid consumer config, stream not found
4. Fix root cause, restart writer process (budget resets on restart)

### Writer: Buffer Overflow (Data Loss)

**Symptoms:** `events_overflowed > 0`, `events_dropped > 0`.

**Actions:**
1. Check `buffer_depth` -- if at/near `max_pending`, ClickHouse can't keep up
2. Check `flush_duration_ms` -- high values indicate ClickHouse latency
3. Check `flush_failures` -- non-zero means INSERT failures
4. Resolution: investigate ClickHouse performance; consider increasing `max_pending` or `batch_size`
5. Data lost to overflow is permanent (remains in NATS for retention window, no auto-replay)

### Writer: Events Received But Not Flushed

**Symptoms:** Consumer `events_received` growing but inserter `events_flushed` is flat.

**Actions:**
1. Check `buffer_depth` -- rows accumulating but not flushing
2. If depth < `batch_size` and flush_interval hasn't elapsed: normal, waiting for trigger
3. If depth >= `batch_size`: flush failing (check `flush_failures`)

### Writer: Stalled

**Symptoms:** `/statusz` shows `phase: "stalled"`, all trackers idle.

**Actions:**
1. Check upstream: are ingest/derive/store active?
2. Check `/readyz` for NATS and ClickHouse reachability
3. If market closed: stalled is expected

### Writer: Unreachable

**Symptoms:** `diag-check.sh` shows writer unreachable.

**Actions:**
1. Check container: `docker compose ps writer`
2. Check startup logs for errors
3. Ensure ClickHouse and NATS are running

### Reader: Analytical Endpoint Returns 404

**Symptoms:** `GET /analytical/evidence/candles` returns 404.

**Diagnosis:** Route not registered -- ClickHouse not configured or connection failed at startup.

**Actions:**
1. Check gateway logs for `"clickhouse not configured"` or `"clickhouse connection failed"`
2. Verify `clickhouse.addr` in `gateway.jsonc`
3. Verify ClickHouse reachable: `curl clickhouse:8123/ping`

### Reader: Analytical Endpoint Returns 503

**Diagnosis:** ClickHouse connected at startup but query failed at runtime.

**Actions:**
1. Check logs for `"analytical query failed"` with error details
2. Common: ClickHouse restarted, network partition, query timeout
3. Verify ClickHouse health and table existence

### Reader: Queries Return 0 Rows

**Actions:**
1. Verify writer is running and flushing (check writer `/statusz`)
2. Check filter parameters (source, symbol, timeframe)
3. Check time range (since/until)
4. Query ClickHouse directly to confirm data exists

### Reader: High Query Latency

**Symptoms:** `query_ms` consistently > 200ms.

**Actions:**
1. Check table size: `SELECT count() FROM evidence_candles`
2. Verify ORDER BY key alignment with query filters
3. Check ClickHouse system tables: `system.query_log`

### Reader: Scan Errors

**Diagnosis:** Schema drift -- table columns no longer match reader's expected types.

**Actions:**
1. Compare reader's column list against actual DDL
2. Check if migration altered column types without updating reader
3. Code/schema alignment bug -- requires a fix

---

## 9. Read Path Failure Visibility

| Failure Mode | HTTP Status | Log Level | Signal |
|---|---|---|---|
| ClickHouse not configured | 404 (route absent) | INFO at startup | "clickhouse not configured" |
| ClickHouse unreachable | 503 | WARN | "analytical query failed" |
| Query timeout | 503 | ERROR+WARN | adapter logs elapsed_ms |
| Scan/type mismatch | 503 | ERROR | "scan failed" |
| Validation failure | 400 | -- | client-side error |

---

## 10. Diagnostic Tooling

### Quick Health Check

```bash
# All runtimes including writer
./scripts/diag-check.sh --local

# Writer only
curl -s http://localhost:8085/statusz | jq '{phase, degraded_trackers, trackers: [.trackers[] | {name, event_count, error_count, counters}]}'

# Gateway analytical endpoint
curl -si http://localhost:8080/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60 | grep Server-Timing
```

### Periodic Polling (Without External Tools)

```bash
while true; do
  echo "$(date -Iseconds) $(curl -s http://localhost:8085/statusz | jq -c '{phase, trackers: [.trackers[] | {name, counters}]}')" >> /tmp/writer-trend.log
  sleep 30
done
```

---

## 11. Read Path Test Coverage

### Covered

| Layer | Tests | Status |
|-------|-------|--------|
| Use case validation | 9 tests (missing source/symbol, invalid timeframe, since>until, default limit, limit clamped, reader error, nil reader, nil use case) | Comprehensive |
| HTTP handler | 5 tests (happy path, missing timeframe, limit OOB, nil handler, use case errors) | Adequate |
| Query builder + float formatting | 8 tests (basic filters, time filters, select columns, float precision) | Comprehensive |

### Outside Coverage

| Component | Reason | Mitigation |
|-----------|--------|------------|
| `QueryCandleHistory` full path | Requires live ClickHouse | Smoke tests validate E2E |
| Row scanning | Requires real result rows | Type alignment via DDL review |
| Connection errors / retries | Infrastructure-dependent | ClickHouse client handles reconnection |

### Round-Trip Fidelity

Write path: `"0.1"` -> `parseFloat` -> `0.1` (float64). Read path: `0.1` -> `formatFloat` -> `"0.1"`. Not lossless for all inputs (IEEE 754), but acceptable for analytical use. Exact decimal fidelity should use the operational (NATS KV) path.

---

## 12. Current Observability Limits

| Gap | Reason |
|-----|--------|
| Per-row insert latency | Only batch-level timing tracked |
| NATS consumer lag | Requires JetStream admin API |
| ClickHouse disk usage | Requires system tables query |
| Cross-pipeline event correlation | No distributed tracing |
| Historical counter trends | In-memory counters reset on restart |
| Per-symbol breakdown | Counters are per-family, not per-symbol |
| Query plan analysis | Requires ClickHouse-side tooling |
| Connection pool state | Not exposed; degradation surfaces as increased query_ms |
| Alerting | Signals are pull-only; no push-based alerting |
