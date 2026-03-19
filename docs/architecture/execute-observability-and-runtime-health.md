# Execute: Observability and Runtime Health

> Stage S87 — Documents the observability surface and health monitoring model for the `execute` binary.

## Health Endpoints

The execute binary exposes three HTTP endpoints on its configured health port (default `:8084`):

### GET /healthz — Liveness Probe

Always returns `200 OK` with `{"status": "ok"}`. Used by docker-compose health checks and orchestrators to determine if the process is alive.

### GET /readyz — Readiness Probe

Returns `200 OK` with `{"status": "ready"}` when all readiness checks pass. Returns `503 Service Unavailable` when any check fails.

**Configured checks:**

| Check | Description |
|-------|-------------|
| `nats` | TCP dial to NATS URL with 2s timeout. Fails if NATS is unreachable or disabled. |

### GET /statusz — Activity Status

Returns `200 OK` with a JSON payload containing tracker activity for all registered components:

```json
{
  "status": "ok",
  "trackers": [
    {
      "name": "venue-adapter",
      "last_event_at": "2026-03-19T12:00:00Z",
      "event_count": 42,
      "error_count": 0,
      "idle_seconds": 5,
      "idle_warning": false,
      "counters": {
        "processed": 50,
        "filled": 42,
        "skipped_stale": 6,
        "skipped_halt": 2
      }
    },
    {
      "name": "venue-consumer",
      "last_event_at": "2026-03-19T12:00:00Z",
      "event_count": 50,
      "error_count": 0,
      "idle_seconds": 5,
      "idle_warning": false
    }
  ]
}
```

## Trackers

Execute registers two health trackers:

| Tracker | Owner | Records |
|---------|-------|---------|
| `venue-adapter` | VenueAdapterActor | Fill events (RecordEvent), submit/publish errors (RecordError) |
| `venue-consumer` | ExecutionConsumer | Consumed execution intents from NATS |

### Custom Counters (venue-adapter)

The venue adapter exposes domain-specific counters via the tracker's counter mechanism:

| Counter | Description |
|---------|-------------|
| `processed` | Total intents received from consumer |
| `filled` | Successfully submitted and published fills |
| `skipped_stale` | Intents rejected by staleness guard |
| `skipped_halt` | Intents blocked by kill switch |

**Invariant:** `processed == filled + skipped_stale + skipped_halt + error_count`

## Idle Heartbeat Monitor

The health server runs a background heartbeat loop (30s interval) that logs warnings when any tracker has been idle beyond the threshold (default: 2 minutes).

```
WARN component idle tracker=venue-adapter idle_seconds=180 last_event=... event_count=42 error_count=0
```

This provides proactive alerting in logs when the execution pipeline stalls.

## Structured Logging

All execute components use `slog` with structured fields:

| Actor | Log fields |
|-------|-----------|
| ExecuteSupervisor | `consumer_durable`, `consumer_subject`, `venue_type`, `control_gate` |
| VenueAdapterActor | `source`, `symbol`, `timeframe`, `correlation_id`, `venue_order_id`, `side`, `quantity` |

### Shutdown Stats

On graceful shutdown, `VenueAdapterActor` logs a final stats summary:

```
INFO venue adapter stats processed=50 filled=42 skipped_stale=6 skipped_halt=2 errors=0
```

## Docker Compose Health Check

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget -q -O - http://127.0.0.1:8084/readyz | grep -q 'ready'"]
  interval: 10s
  timeout: 3s
  retries: 6
  start_period: 10s
```

The compose health check uses `/readyz` (not `/healthz`) to ensure the service is not only alive but can reach NATS before dependent services start.

## Operational Diagnostics Workflow

1. **Is execute alive?** `curl :8084/healthz` — process liveness
2. **Is execute connected?** `curl :8084/readyz` — NATS connectivity
3. **Is execute processing?** `curl :8084/statusz` — event counts, idle time, gate stats
4. **Is execute halted?** `curl :8080/execution/control` — kill switch state via gateway
5. **Are fills materializing?** `curl :8080/execution/venue_market_order/latest?...` — store projection via gateway

## What Remains Outside Scope

- Prometheus `/metrics` endpoint (structured logging + /statusz is sufficient for paper phase)
- Distributed tracing integration (OpenTelemetry)
- External alerting rules (PagerDuty, Grafana)
- Per-symbol breakdown in /statusz (aggregate counters are sufficient for paper phase)
