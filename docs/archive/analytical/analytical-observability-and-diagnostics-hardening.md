# Analytical Observability and Diagnostics Hardening

## Purpose

This document defines the minimal observability and diagnostic signals introduced for the analytical layer (writer service, reader adapter, health endpoints) to make the layer operationally diagnosable without heavy observability tooling.

## Design Principles

1. **Minimal useful signals** — every counter and gauge must answer a specific operational question.
2. **No external dependencies** — all signals are exposed via existing `/statusz` and `/diagz` endpoints plus structured logs. No OpenTelemetry, Prometheus, or Grafana required.
3. **Counter semantics reused for gauges** — the Tracker's `Counter(name)` returns `*atomic.Int64`, which supports both `Add()` (monotonic counter) and `Store()` (point-in-time gauge). No new abstraction needed.
4. **Log-first debugging** — all significant state transitions emit structured log entries that can be grepped from container logs.

## Signal Catalog

### Consumer Trackers (per pipeline family)

| Signal | Type | Source | Meaning |
|---|---|---|---|
| `event_count` | counter | `RecordEvent()` | Total events received from NATS |
| `error_count` | counter | `RecordError()` | Total errors recorded |
| `events_received` | counter | consumer callback | Events decoded and forwarded to inserter |
| `pipeline_restarts` | counter | supervisor | Number of restart attempts for this family |
| `pipeline_degraded` | counter | supervisor | Set to 1 when restart budget exhausted |

### Inserter Trackers (per pipeline family)

| Signal | Type | Source | Meaning |
|---|---|---|---|
| `event_count` | counter | `RecordEvent()` | Successful flush operations |
| `error_count` | counter | `RecordError()` | Flush failures after retry exhaustion |
| `events_flushed` | counter | flush success | Total rows successfully inserted into ClickHouse |
| `flush_total` | counter | flush success | Total number of successful batch flushes |
| `flush_duration_ms` | gauge | flush complete | Duration of last flush operation (ms) |
| `flush_failures` | counter | flush exhaustion | Number of batch drops after retry exhaustion |
| `events_dropped` | counter | overflow + flush exhaustion | Total rows permanently lost |
| `events_overflowed` | counter | buffer overflow | Rows evicted due to buffer overflow |
| `buffer_depth` | gauge | row insert / flush / overflow | Current number of rows buffered |

### Health Endpoint Signals

| Endpoint | Signal | Meaning |
|---|---|---|
| `/statusz` | `phase` | Aggregate phase: starting, warming, active, idle, stalled, degraded |
| `/statusz` | `degraded_trackers` | List of tracker names with `pipeline_degraded > 0` |
| `/statusz` | per-tracker `counters` | Full counter/gauge snapshot per tracker |
| `/diagz` | `readiness_checks` | Pass/fail for NATS and ClickHouse connectivity |
| `/diagz` | per-tracker `counters` | Same counter snapshot as `/statusz` |

### Structured Log Signals

| Level | Event | Key Fields |
|---|---|---|
| DEBUG | `batch flushed` | family, table, rows, flush_ms |
| WARN | `flush attempt failed` | error, family, table, rows, attempt, max_retries |
| WARN | `component idle` | tracker, idle_seconds, event_count, error_count, + all counters |
| ERROR | `buffer overflow` | family, evicted, buffer_depth |
| ERROR | `flush failed — retries exhausted` | error, family, rows_dropped, attempts, flush_ms |
| ERROR | `pipeline degraded` | family, restarts, last_error |
| WARN | `pipeline failure — scheduling restart` | family, error, restart, max_restarts, backoff |

## Operational Questions Answered

| Question | How to Answer |
|---|---|
| Is the writer alive? | `GET /healthz` → 200 |
| Is ClickHouse reachable? | `GET /readyz` → check `clickhouse` status |
| Are all pipelines healthy? | `GET /statusz` → `phase != "degraded"` and `degraded_trackers` is empty |
| Which pipeline is degraded? | `GET /statusz` → `degraded_trackers` array |
| Is data flowing? | Compare `events_received` (consumer) vs `events_flushed` (inserter) |
| Is backpressure building? | Check `buffer_depth` on inserter trackers |
| Are we losing data? | Check `events_dropped` and `events_overflowed` counters |
| How fast are flushes? | Check `flush_duration_ms` gauge |
| How many batches succeeded? | Check `flush_total` counter |
| Did a pipeline restart? | Check `pipeline_restarts` counter |

## Diagnostic Tooling

### diag-check.sh

The diagnostic script now includes the writer runtime alongside all operational runtimes. It queries `/readyz`, `/statusz`, and `/diagz` on port 8085 and includes writer tracker details in its output.

### Manual Inspection

```bash
# Writer health
curl -s http://localhost:8085/healthz | jq .

# Writer readiness (NATS + ClickHouse)
curl -s http://localhost:8085/readyz | jq .

# Writer activity and counters
curl -s http://localhost:8085/statusz | jq .

# Writer full diagnostic
curl -s http://localhost:8085/diagz | jq .
```

## What Is NOT Observable (Current Limits)

| Gap | Reason | Future Resolution |
|---|---|---|
| Per-row insert latency | Only batch-level timing tracked | Would require sampling, deferred |
| NATS consumer lag | Requires JetStream admin API queries | Future NATS admin integration |
| ClickHouse disk usage | Requires ClickHouse system tables query | Future infra monitoring |
| Cross-pipeline event correlation | No distributed tracing | OpenTelemetry in future wave |
| Historical counter trends | Counters are in-memory, reset on restart | Future Prometheus/time-series export |
| Individual row failures within a batch | ClickHouse batch is all-or-nothing | By design; no partial batch tracking |
| Reader query latency | No instrumentation on analytical reader | Future gateway middleware |
