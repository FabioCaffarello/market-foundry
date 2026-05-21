# Current Baseline — Operational Diagnostics Assessment

> Canonical reference for what diagnostic signals the Foundry baseline exposes,
> what is sufficient, what remains minimal, and where gaps exist.

## Diagnostic Surface per Runtime

Every long-running Foundry binary exposes a dedicated health HTTP server with four endpoints:

| Endpoint   | Purpose                          | Exposed By                                     |
|------------|----------------------------------|-------------------------------------------------|
| `/healthz` | Liveness probe                   | All runtimes (ingest, derive, store, execute, configctl, gateway) |
| `/readyz`  | Readiness probe with named checks| All runtimes                                    |
| `/statusz` | Operational activity status      | ingest, derive, store, execute, configctl       |
| `/diagz`   | Machine-readable diagnostic summary | ingest, derive, store, execute, configctl    |

**Gateway exception:** The gateway runtime exposes `/healthz` and `/readyz` via the main HTTP webserver (shared port with domain routes), not via a separate health server. It does NOT expose `/statusz` or `/diagz`.

## Endpoint Detail

### `/healthz` — Liveness

- Returns `{"status": "ok"}` with HTTP 200.
- No dependency checks. Confirms process is alive.
- **Assessment: sufficient.** This is the correct liveness signal.

### `/readyz` — Readiness

- Runs registered `ReadinessCheck` functions sequentially.
- Returns `{"status": "ready"}` (200) or `{"status": "not_ready", "check": "...", "error": "..."}` (503).
- Registered checks per runtime:
  - **ingest, derive, store, execute, configctl**: NATS TCP dial (2s timeout).
  - **gateway**: NATS enabled check + configctl gateway ping + evidence store probe (non-blocking).
- **Assessment: sufficient for current baseline.** Covers the critical dependency (NATS). Gateway correctly degrades when evidence store is unavailable.

### `/statusz` — Activity Status

- Returns JSON with: `status`, `phase`, `runtime`, `uptime`, `started_at`, `trackers[]`.
- Each tracker reports: `name`, `event_count`, `error_count`, `last_event_at`, `idle_seconds`, `idle_warning`, `counters{}`.
- **Phase classification** (aggregate operational state):
  - `starting` — uptime < 30s and no tracker has recorded an event.
  - `warming` — at least one tracker awaiting its first event.
  - `active` — all trackers receiving events, none idle.
  - `idle` — at least one tracker exceeds idle threshold (default 2 min).
  - `stalled` — all active trackers exceed idle threshold.
- **Idle heartbeat monitor**: runs every 30s, logs `slog.Warn` for idle trackers.
- Registered trackers per runtime:

| Runtime   | Trackers                                                    |
|-----------|-------------------------------------------------------------|
| ingest    | `observation-publisher`                                     |
| derive    | `evidence-publisher`                                        |
| store     | Per enabled family: `{family}-projection`, `{family}-consumer` (e.g., `candle-projection`, `candle-consumer`) |
| execute   | `venue-adapter`, `venue-consumer`                           |
| configctl | (none — no pipeline trackers)                               |

- **Assessment: sufficient.** Phase classification, idle detection, custom counters, and per-tracker event/error counts provide clear operational visibility without heavyweight tracing.

### `/diagz` — Diagnostic Summary

- Returns JSON with: `runtime`, `phase`, `started_at`, `uptime`, `go_version`, `num_goroutines`, `readiness_checks[]`, `trackers[]`.
- Readiness checks are re-evaluated on each request (not cached).
- Trackers include `status: "awaiting_first_event"` when no events recorded yet.
- **Assessment: sufficient.** Goroutine count and Go version aid debugging without adding overhead.

## Structured Logging

- Framework: Go `log/slog` (standard library).
- Format: JSON or text, configurable per service via `log.format` in config.
- All log lines include `"runtime": "<name>"` field automatically.
- Log level configurable via `log.level` in config.
- **Assessment: sufficient.** JSON structured logs with runtime labels enable filtering and aggregation. No additional log enrichment needed at this stage.

## Diagnostic Scripts

| Script                         | Purpose                                          |
|--------------------------------|--------------------------------------------------|
| `scripts/diag-check.sh`       | Lightweight diagnostic snapshot of running stack  |
| `scripts/live-pipeline-activate.sh` | Full pipeline activation with diagnostic validation |
| `scripts/smoke-first-slice.sh` | E2E smoke test for single-symbol slice           |
| `scripts/smoke-multi-symbol.sh`| E2E smoke test for multi-symbol scenario         |
| `scripts/seed-configctl.sh`   | Seed configctl with ingestion bindings           |

## Gaps and Minimal Areas

### Currently Sufficient

1. **Liveness and readiness probes** — clear, correct, tested.
2. **Tracker system** — per-component event/error counting with custom counters.
3. **Phase classification** — aggregate state derivation (starting/warming/active/idle/stalled).
4. **Idle detection** — 30s heartbeat with configurable threshold and log warnings.
5. **Structured logging** — JSON format with runtime labels.
6. **Correlation ID middleware** — `X-Correlation-ID` propagation on gateway HTTP requests.
7. **Error log scanning** — automated in `live-pipeline-activate.sh` and `diag-check.sh`.
8. **Memory usage snapshot** — `docker stats` in pipeline activation script.

### Currently Minimal (Acceptable for Baseline)

1. **Gateway lacks /statusz and /diagz** — acceptable because gateway is a stateless request proxy. Its operational state is fully visible through the upstream services it calls. Adding trackers would require artificial instrumentation with no clear payoff.

2. **No request-level latency metrics** — the gateway does not track per-endpoint latency. Acceptable for current scale; would become important with production traffic or SLO commitments.

3. **No NATS consumer lag visibility** — the system does not expose JetStream consumer pending counts. The tracker system indirectly covers this (idle warnings signal stalled consumption), but direct lag numbers would help diagnose backpressure.

4. **No persistent diagnostic history** — all diagnostics are point-in-time snapshots. There is no time-series storage of tracker counts or phase transitions. This is a natural candidate for ClickHouse (see `future-analytics-signals-candidates-for-clickhouse.md`).

5. **Configctl has no pipeline trackers** — it only has a NATS readiness check. This is correct since configctl handles config lifecycle, not data flow. Its operational health is confirmed by gateway readiness probe.

### Not Needed Now

- Distributed tracing (OpenTelemetry) — premature for single-operator baseline.
- Prometheus metrics endpoint — adds dependency without current consumers.
- Alerting rules — no alerting infrastructure exists yet.
- Dashboard definitions — no Grafana or equivalent deployed.
