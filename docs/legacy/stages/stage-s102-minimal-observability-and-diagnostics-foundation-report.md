# Stage S102 -- Minimal Observability and Diagnostics Foundation Report

> Establishing baseline observability across all market-foundry runtimes without introducing external telemetry dependencies.

---

## Stage Identifier

- **Stage:** S102
- **Date:** 2026-03-19
- **Predecessor:** S101 (Operational Contracts and Cross-Runtime Conventions)
- **Focus:** Observability, diagnostics, structured logging, runtime signals

---

## 1. Executive Summary

S102 introduces a minimal observability foundation across all 6 runtimes (gateway, configctl, ingest, derive, store, execute). The changes add runtime identity to every structured log line, shutdown signal logging, enhanced `/statusz` with runtime metadata, a new `/diagz` diagnostic endpoint, and consistent startup summary logs. No external observability infrastructure is introduced -- all changes use Go stdlib `log/slog` and the existing `healthz` package.

**Key outcome:** Every runtime now produces structured, filterable logs with runtime identity, exposes its internal state via four HTTP diagnostic endpoints (`/healthz`, `/readyz`, `/statusz`, `/diagz`), and logs its lifecycle transitions (startup, shutdown signal, shutdown complete) in a consistent format.

---

## 2. Changes Made

### Infrastructure Changes

| File | Change | Reason |
|------|--------|--------|
| `internal/shared/bootstrap/logger.go` | `BuildLogger` now accepts a `runtime string` parameter; adds `runtime` field to every log line via `slog.Logger.With` | Runtime identity in every log line enables cross-runtime log filtering and aggregation |
| `internal/actors/common/entrypoint.go` | `WaitTillShutdown` now logs the received signal type (`SIGTERM`/`SIGINT`) before beginning actor poison | Distinguishes operator-initiated shutdowns from crashes in post-mortem analysis |
| `internal/shared/healthz/healthz.go` | Added `WithRuntime` option; `/statusz` now includes `runtime`, `started_at`, `uptime_seconds` fields | Runtime metadata in diagnostic responses enables identification without external context |
| `internal/shared/healthz/healthz.go` | Added `/diagz` endpoint handler; registered in `Start()` | Single-request diagnostic summary combining readiness checks and tracker health |

### Runtime Composition Root Changes

| File | Change | Reason |
|------|--------|--------|
| `cmd/gateway/run.go` | Updated `BuildLogger` call to pass `"gateway"` as runtime name | Runtime identity field |
| `cmd/store/run.go` | Updated `BuildLogger` call to pass `"store"` as runtime name | Runtime identity field |
| `cmd/derive/run.go` | Updated `BuildLogger` call to pass `"derive"` as runtime name | Runtime identity field |
| `cmd/ingest/run.go` | Updated `BuildLogger` call to pass `"ingest"` as runtime name | Runtime identity field |
| `cmd/execute/run.go` | Updated `BuildLogger` call to pass `"execute"` as runtime name | Runtime identity field |
| `cmd/configctl/run.go` | Updated `BuildLogger` call to pass `"configctl"` as runtime name | Runtime identity field |

### Supervisor Changes

| File | Change | Reason |
|------|--------|--------|
| `internal/actors/scopes/store/store_supervisor.go` | Added startup summary log (enabled families, stream/consumer count) | Consistent supervisor startup visibility |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added startup summary log (enabled families, source count, timeframes) | Consistent supervisor startup visibility |

### Architecture Documents Created

| File | Purpose |
|------|---------|
| `docs/architecture/minimal-observability-foundation.md` | Canonical reference for observability philosophy, log field conventions, diagnostic surfaces, and extension patterns |
| `docs/architecture/diagnostic-surfaces-and-runtime-signals.md` | Detailed catalog of HTTP diagnostic endpoints, lifecycle signals, tracker behavior, and debugging workflows |

---

## 3. Observability Foundation Introduced

### Structured Log Field Conventions

Six standard fields are established across all runtimes:

| Field | Source | Present In |
|-------|--------|------------|
| `runtime` | Logger default attribute | Every log line |
| `actor` | Actor log calls | Actor lifecycle and processing logs |
| `component` | Infrastructure log calls | Health server, NATS client logs |
| `source` | Ingestion actor log calls | Venue/exchange-related logs |
| `family` | Supervisor log calls | Domain family-related logs |
| `error` | Error log calls (INV-5) | All error logs |

### Log Level Discipline

| Level | Policy |
|-------|--------|
| INFO | Lifecycle events, startup summaries, configuration confirmations |
| WARN | Idle components, unknown messages, degraded states |
| ERROR | Failures affecting correctness or availability |
| DEBUG | Reserved for future per-event tracing; not emitted in production |

Successful event processing does not emit log lines. Activity is tracked via `Tracker.RecordEvent()` and exposed through `/statusz`.

### Diagnostic HTTP Surface

| Endpoint | Purpose | Added/Enhanced |
|----------|---------|----------------|
| `/healthz` | Liveness probe (always 200) | Existing |
| `/readyz` | Readiness probe (200 or 503) | Existing |
| `/statusz` | Activity status with tracker details | Enhanced with runtime metadata |
| `/diagz` | Combined readiness + tracker summary | New |

### Lifecycle Signal Coverage

| Signal | When | Log Level |
|--------|------|-----------|
| Startup | Runtime enters `Run()` | INFO |
| Startup summary | Supervisor spawns children | INFO |
| Shutdown signal received | SIGTERM/SIGINT caught | INFO |
| Shutdown complete | All actors poisoned | INFO |
| Component idle | Heartbeat detects idle tracker | WARN |

---

## 4. Diagnostic Gains

### Before S102

- Log lines did not carry runtime identity. Filtering aggregated logs required parsing message text.
- Shutdown cause (SIGTERM vs SIGINT vs crash) was indistinguishable in logs.
- `/statusz` reported tracker data but not which runtime produced it or how long it had been running.
- No single-request diagnostic overview existed. Debugging required checking `/readyz` and `/statusz` separately.
- Supervisor startup logs were inconsistent -- some logged summaries, others did not.

### After S102

- Every log line carries `runtime=<name>`, enabling instant filtering across aggregated streams.
- Shutdown signal type is logged, distinguishing orchestrator shutdowns from manual interrupts and crashes.
- `/statusz` includes `runtime`, `started_at`, and `uptime_seconds` -- self-identifying without external context.
- `/diagz` provides a single-request overview: readiness check results + tracker health summary.
- All supervisors emit consistent startup summaries with enabled families and resource counts.

---

## 5. Limits Maintained

The following capabilities were explicitly excluded from this stage:

| Excluded Capability | Rationale |
|---------------------|-----------|
| OpenTelemetry / Jaeger | Requires collector infrastructure, SDK integration, and span propagation. Premature given current deployment model. |
| Prometheus metrics | Requires metrics backend and careful label cardinality design. Health trackers provide sufficient aggregate visibility. |
| Grafana dashboards | No aggregation layer exists. `/statusz` and `/diagz` serve as the current visualization surface. |
| Correlation ID in log lines | Correlation IDs exist in domain events but are not yet injected into slog attributes per-request. Requires middleware or per-handler enrichment. |
| Per-event log lines | Would produce excessive noise. Event activity is tracked via counters, not individual log entries. |
| Alerting integration | No alerting system is integrated. Idle warnings are logged but not routed externally. |
| Custom log sampling | All runtimes use the same log level. Per-component level control is a future enhancement. |

These exclusions are deliberate engineering decisions, not oversights. Each can be introduced independently when the operational need justifies the infrastructure cost.

---

## 6. Preparation for S103

Based on the observability foundation established in S102, the following areas are recommended for S103:

1. **Test infrastructure for diagnostic endpoints** -- The `/healthz`, `/readyz`, `/statusz`, and `/diagz` endpoints have no integration tests verifying their response format or status codes under various conditions (NATS down, trackers idle, trackers active). A lightweight test suite using `httptest.Server` would prevent regression.

2. **Raccoon-cli observability rules** -- The architecture guardian can enforce: (a) `BuildLogger` calls include runtime name, (b) error log keys use `"error"` not `"err"`, (c) supervisor actors emit startup summary logs.

3. **Correlation ID propagation to logs** -- `requestctx.CorrelationID` is available in domain contexts. A `slog` middleware or handler wrapper could inject it into log attributes for request-scoped tracing without external infrastructure.

4. **Structured error classification** -- Error logs currently use free-form strings. A classification scheme (connectivity, validation, timeout, internal) would enable automated error categorization without metrics infrastructure.

---

## 7. Files Changed

### Code

- `internal/shared/bootstrap/logger.go`
- `internal/actors/common/entrypoint.go`
- `internal/shared/healthz/healthz.go`
- `cmd/gateway/run.go`
- `cmd/store/run.go`
- `cmd/derive/run.go`
- `cmd/ingest/run.go`
- `cmd/execute/run.go`
- `cmd/configctl/run.go`
- `internal/actors/scopes/store/store_supervisor.go`
- `internal/actors/scopes/derive/derive_supervisor.go`

### Documentation

- `docs/architecture/minimal-observability-foundation.md`
- `docs/architecture/diagnostic-surfaces-and-runtime-signals.md`
- `docs/stages/stage-s102-minimal-observability-and-diagnostics-foundation-report.md`

---

## Guard Rails -- Compliance

| Guard rail | Compliance |
|------------|------------|
| No OpenTelemetry/Jaeger/Prometheus/Grafana | Fulfilled -- all observability uses stdlib slog and in-process healthz package |
| No excessive log noise | Fulfilled -- successful events are counted not logged; heartbeat is silent when healthy |
| No coupling diagnostics to domain logic | Fulfilled -- trackers are injected from composition roots; actors only call RecordEvent/RecordError |
| No vague diagnostic endpoints | Fulfilled -- /diagz has defined schema; /statusz has defined fields; both documented with examples |
| Error key is always "error" (INV-5) | Maintained |
| Single-phase logger construction | Maintained -- BuildLogger is called once with runtime name in Phase 1 |
