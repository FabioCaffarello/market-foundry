# Minimal Observability Foundation

> What market-foundry observes, how it exposes diagnostics, and what it deliberately omits.

---

## Rationale

Distributed systems fail in ways that are invisible without observability. However, observability infrastructure is itself a source of complexity, operational cost, and coupling. market-foundry follows a minimal-first approach: observe what is needed to debug production issues today, defer what requires infrastructure that does not yet exist.

The goals are:

1. **Reduce debugging cost** -- every log line identifies which runtime produced it; every diagnostic endpoint reveals internal state without attaching a debugger.
2. **Increase operational confidence** -- startup summaries confirm what is active; idle monitoring detects pipeline stalls; readiness probes prevent traffic to unready instances.
3. **Avoid premature infrastructure** -- no telemetry collectors, no dashboards, no distributed tracing agents. These are valid future investments, not current necessities.

---

## What Is Observed

### Runtime Identity in Every Log Line

Every structured log line carries a `runtime` field set to the binary name (e.g., `"gateway"`, `"store"`, `"derive"`). This is injected once at logger construction via `BuildLogger(cfg, runtimeName)` and propagated to all downstream log calls automatically through `slog.Logger.With`.

When logs from multiple runtimes are aggregated into a single stream (container orchestrator logs, centralized log sink), the `runtime` field enables filtering without parsing message text.

### Lifecycle Signals

- **Startup**: each runtime emits an INFO log on entry (e.g., `"store starting"`) with relevant configuration context.
- **Shutdown signal**: when SIGTERM or SIGINT is received, the signal type is logged before shutdown begins.
- **Shutdown complete**: after all actors are poisoned and stopped, a final `"shutdown complete"` log confirms clean exit.

### Component Activity via Health Trackers

Health trackers (`healthz.Tracker`) record event counts, error counts, and last-event timestamps for named components. They power both the `/statusz` HTTP endpoint and the idle heartbeat monitor. Trackers are created in composition roots and injected into actors -- actors never create their own trackers.

### Readiness State

Readiness checks (`healthz.ReadinessCheck`) verify that external dependencies (NATS connections, venue adapters) are reachable. The `/readyz` endpoint runs all checks on each request and returns the first failure, enabling orchestrators to hold traffic until the runtime is ready.

---

## Structured Log Field Conventions

All structured log fields follow consistent naming across runtimes. These are conventions, not enforced by type system -- adherence is verified by review and architecture guardian rules.

| Field | Type | Meaning | Example |
|-------|------|---------|---------|
| `runtime` | string | Binary name, set once at logger creation | `"store"`, `"gateway"` |
| `actor` | string | Actor identity within the engine | `"evidence-projection"` |
| `component` | string | Infrastructure component emitting the log | `"healthz"`, `"nats"` |
| `source` | string | External data source or exchange | `"binance"`, `"kraken"` |
| `family` | string | Domain family being processed | `"evidence"`, `"signal"` |
| `error` | string/error | Error details (always keyed as `"error"`, never `"err"`) | `"connection refused"` |
| `addr` | string | Network address | `":8080"` |
| `tracker` | string | Health tracker name in idle warnings | `"evidence-projection"` |
| `idle_seconds` | int | Seconds since last tracker event | `180` |
| `signal` | string | OS signal received at shutdown | `"SIGTERM"` |

### Key rule

The error field key is always `"error"`. This is INV-5 from S101. Using `"err"` or any variant is a violation.

---

## Log Level Semantics

| Level | Used For | Examples |
|-------|----------|---------|
| **INFO** | Lifecycle events, startup summaries, configuration confirmations | `"store starting"`, `"enabled families"`, `"shutdown complete"` |
| **WARN** | Idle components, unknown actor messages, degraded but functional states | `"component idle"`, `"unknown message type"` |
| **ERROR** | Failures that affect correctness or availability | `"create actor engine"`, `"NATS connection failed"` |
| **DEBUG** | Reserved for future per-event tracing; not currently emitted in production | (none currently) |

### Noise discipline

- Successful event processing does NOT emit a log line. Activity is tracked via `Tracker.RecordEvent()` and visible through `/statusz`.
- Periodic heartbeat checks (every 30s) only log when a tracker exceeds the idle threshold. Silent heartbeats produce no output.
- Startup logs are bounded: one line per runtime, one summary per supervisor. No per-actor startup flood.

---

## Diagnostic HTTP Surfaces

All runtimes expose diagnostic endpoints via `healthz.HealthServer`. The gateway runtime integrates health handlers into its main HTTP server; all other runtimes run a dedicated health server on `config.HTTP.Addr`.

### /healthz -- Liveness Probe

| Aspect | Detail |
|--------|--------|
| Method | `GET` |
| Success | `200 OK` with `{"status": "ok"}` |
| Failure | Never fails (if the process is alive, /healthz responds) |
| Use case | Kubernetes liveness probe; restart detection |

### /readyz -- Readiness Probe

| Aspect | Detail |
|--------|--------|
| Method | `GET` |
| Success | `200 OK` with `{"status": "ready"}` |
| Failure | `503 Service Unavailable` with `{"status": "not_ready", "check": "<name>", "error": "<detail>"}` |
| Use case | Kubernetes readiness probe; traffic gating |

Checks are evaluated sequentially. The first failing check short-circuits the response. Typical checks: NATS connectivity, venue adapter availability.

### /statusz -- Activity Status

| Aspect | Detail |
|--------|--------|
| Method | `GET` |
| Success | `200 OK` with tracker summaries, runtime metadata |
| Failure | Always 200 (status reflects tracker state, not endpoint health) |
| Use case | Operator inspection; pipeline stall detection; debugging |

Response includes per-tracker: `name`, `event_count`, `error_count`, `last_event_at` (RFC3339), `idle_seconds`, `idle_warning` (boolean), and custom `counters`. Runtime metadata includes `runtime` (name), `started_at` (RFC3339), and `uptime_seconds`.

### /diagz -- Diagnostic Summary

| Aspect | Detail |
|--------|--------|
| Method | `GET` |
| Success | `200 OK` with combined readiness and tracker summary |
| Failure | Always 200 (reports diagnostic state, does not itself fail) |
| Use case | Single-request health overview; incident triage |

Response combines readiness check results (pass/fail per check) with a condensed tracker summary (active count, idle count, error count). This endpoint is designed for human consumption during incidents -- one curl gives a complete picture.

---

## What Is Intentionally NOT Observed

| Capability | Status | Rationale |
|------------|--------|-----------|
| Distributed tracing (OpenTelemetry, Jaeger) | Deferred | Requires collector infrastructure, SDK integration, and span propagation. The operational need does not yet justify the cost. Correlation IDs exist in domain events (`requestctx.CorrelationID`) but are not propagated to log lines. |
| Per-event metrics (Prometheus, counters per message type) | Deferred | Health trackers provide aggregate counts. Per-event cardinality metrics need a metrics backend and careful label design. |
| Dashboards (Grafana, custom UI) | Deferred | No aggregation layer exists. `/statusz` and `/diagz` serve as the current visualization surface. |
| Correlation ID in log lines | Deferred | Correlation IDs flow through NATS event headers and are available in domain contexts, but are not yet injected into `slog` attributes per-request. Adding this requires either middleware or per-handler logger enrichment. |
| Alerting rules | Deferred | No alerting system is integrated. Idle warnings are logged but not routed to any notification channel. |

These are valid future capabilities. They are excluded now because each requires infrastructure investment (collectors, storage, dashboards) that would be premature given the current deployment model.

---

## How to Extend

### Adding a New Tracker

1. Create the tracker in the composition root (`cmd/<runtime>/run.go`):
   ```go
   tracker := healthz.NewTracker("my-component")
   ```
2. Pass it to the actor that will record events via constructor injection.
3. Include it in the `trackers` slice passed to `NewHealthServer`.
4. The tracker automatically appears in `/statusz` and `/diagz`, and is monitored by the idle heartbeat.

### Adding a Custom Counter to a Tracker

```go
tracker.Counter("my_metric").Add(1)
```

Custom counters appear in the `/statusz` response under the tracker's `counters` map. Use them for domain-specific operational metrics (e.g., `"filled"`, `"skipped_stale"`).

### Adding a New Readiness Check

1. Define the check:
   ```go
   check := healthz.ReadinessCheck{
       Name:  "my-dependency",
       Check: func(ctx context.Context) error { /* verify connectivity */ },
   }
   ```
2. Include it in the `checks` slice passed to `NewHealthServer`.
3. The check automatically gates `/readyz` responses.

### Adding Runtime Metadata to /statusz

Runtime metadata (name, uptime, started_at) is injected via `WithRuntime` option on `HealthServer`. This option is applied once at construction and does not require per-request computation.
