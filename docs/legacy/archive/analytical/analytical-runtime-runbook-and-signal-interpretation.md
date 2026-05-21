# Analytical Runtime Runbook and Signal Interpretation

## Purpose

Operational runbook for diagnosing and responding to analytical layer issues using the signals available from health endpoints and structured logs.

## Quick Health Check

```bash
# All runtimes including writer
./scripts/diag-check.sh --local

# Writer only
curl -s http://localhost:8085/statusz | jq '{phase, degraded_trackers, trackers: [.trackers[] | {name, event_count, error_count, counters}]}'
```

## Phase Interpretation

| Phase | Meaning | Action |
|---|---|---|
| `starting` | Process started <30s ago, no events yet | Wait. Normal during cold start. |
| `warming` | At least one pipeline awaiting first event | Check if upstream runtimes are active. If persists >2min, investigate NATS connectivity. |
| `active` | All pipelines receiving and processing events | Healthy. No action. |
| `idle` | At least one pipeline idle beyond threshold (2min) | Check if upstream events stopped for that family. May be normal if no market data for that family. |
| `stalled` | All pipelines idle beyond threshold | Likely upstream issue. Check ingest/derive/store runtimes. Check NATS connectivity via `/readyz`. |
| `degraded` | At least one pipeline has exhausted restart budget | Investigate `degraded_trackers`. Check logs for root cause. Restart writer after fixing. |

## Scenario Playbooks

### Scenario: Pipeline Degraded

**Symptoms**: `/statusz` shows `phase: "degraded"`, `degraded_trackers` lists affected families.

**Diagnosis**:
1. Check `/statusz` for which trackers have `pipeline_degraded > 0`
2. Check logs: `grep "pipeline degraded" <writer-logs>` to find the `last_error`
3. Common causes:
   - NATS connection failure (consumer can't subscribe)
   - Invalid consumer configuration
   - Stream not found

**Resolution**:
1. Fix the root cause (NATS connectivity, stream configuration)
2. Restart the writer process — restart budget resets on process start

### Scenario: Buffer Overflow (Data Loss)

**Symptoms**: Inserter tracker shows `events_overflowed > 0`, `events_dropped > 0`.

**Diagnosis**:
1. Check `buffer_depth` — if consistently at or near `max_pending` (default 10,000), ClickHouse can't keep up
2. Check `flush_duration_ms` — high values indicate ClickHouse latency
3. Check `flush_failures` — non-zero means ClickHouse INSERT failures
4. Check ClickHouse readiness: `curl -s http://localhost:8085/readyz`

**Resolution**:
- If flush_duration_ms is high: investigate ClickHouse performance (disk I/O, memory, query load)
- If flush_failures > 0: check ClickHouse logs for INSERT errors (schema mismatch, disk full)
- If sustained overflow: consider increasing `max_pending` or `batch_size` in writer config
- Data lost to overflow is permanent — it remains in NATS for the retention window but there is no automatic replay

### Scenario: Flush Failures Without Overflow

**Symptoms**: `flush_failures > 0` but `events_overflowed == 0`.

**Diagnosis**:
1. Transient ClickHouse issue that resolved within retry window
2. Check `events_dropped` — these rows were lost after retry exhaustion
3. Check logs: `grep "flush failed" <writer-logs>` for the specific ClickHouse error

**Resolution**:
- If intermittent: likely transient. Monitor `flush_failures` trend.
- If persistent: check ClickHouse health, disk space, schema compatibility
- Increase `max_retries` or `initial_backoff` if ClickHouse restarts are slow

### Scenario: Events Received But Not Flushed

**Symptoms**: Consumer tracker `events_received` growing but inserter `events_flushed` is zero or flat.

**Diagnosis**:
1. Check `buffer_depth` — rows are accumulating but not flushing
2. If `buffer_depth` < `batch_size` and `flush_interval` hasn't elapsed: normal, waiting for batch or timer
3. If `buffer_depth` >= `batch_size`: flush is failing silently (check `flush_failures`)

**Resolution**:
- If new pipeline: wait for first batch to fill or flush interval to trigger
- If established: check ClickHouse connectivity and inserter logs

### Scenario: Writer Stalled

**Symptoms**: `/statusz` shows `phase: "stalled"`, all trackers idle.

**Diagnosis**:
1. Check upstream: are ingest/derive/store runtimes active?
2. Check `/readyz` — are NATS and ClickHouse reachable?
3. Check if market data feed is active (no candles = no downstream events)

**Resolution**:
- If upstream is stalled: fix upstream first
- If NATS is unreachable: fix NATS connectivity
- If market is closed: stalled is expected, no action needed

### Scenario: Writer Unreachable

**Symptoms**: `diag-check.sh` shows writer `/readyz` unreachable.

**Diagnosis**:
1. Is the writer container running? `docker compose ps writer`
2. Check writer logs for startup errors
3. Common causes: ClickHouse not available at startup (writer exits on connection failure)

**Resolution**:
1. Ensure ClickHouse is running and reachable
2. Ensure NATS is running
3. Start/restart writer container

## Signal Relationships

```
Consumer                          Inserter
─────────                         ────────
events_received ──→ [actor msg] ──→ buffer_depth
                                    │
                                    ├──→ events_flushed (success)
                                    │    flush_total
                                    │    flush_duration_ms
                                    │
                                    ├──→ events_overflowed (buffer full)
                                    │    events_dropped
                                    │
                                    └──→ flush_failures (retry exhausted)
                                         events_dropped

Supervisor
──────────
pipeline_restarts ──→ pipeline_degraded (budget exhausted)
```

## Key Invariants for Validation

1. `events_received` (consumer) >= `events_flushed` (inserter) — always true
2. `events_dropped` = events lost to overflow + events lost to flush failure
3. `events_overflowed` <= `events_dropped` — overflow is a subset of drops
4. `flush_total` * average_batch_size ≈ `events_flushed` — rough consistency check
5. `pipeline_degraded == 0` for all trackers when `phase != "degraded"`
6. `buffer_depth == 0` after a successful flush with no concurrent inserts

## Monitoring Without External Tools

For operators without Prometheus/Grafana, periodic polling provides trend data:

```bash
# Poll writer status every 30s and log to file
while true; do
  echo "$(date -Iseconds) $(curl -s http://localhost:8085/statusz | jq -c '{phase, trackers: [.trackers[] | {name, counters}]}')" >> /tmp/writer-trend.log
  sleep 30
done
```

This log can be inspected later to identify when counters changed and correlate with incidents.

## Limits of Current Observability

- **No historical trend data** — counters reset on process restart; no time-series storage
- **No alerting** — signals are pull-only via HTTP; no push-based alerting
- **No NATS consumer lag** — can't tell if writer is behind the NATS stream tip
- **No cross-service correlation** — no distributed tracing between writer and upstream
- **No per-symbol breakdown** — counters are per-pipeline-family, not per-symbol
- **No ClickHouse query performance** — reader adapter has no instrumentation yet
