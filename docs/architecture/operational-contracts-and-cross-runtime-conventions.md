# Operational Contracts and Cross-Runtime Conventions

> Canonical reference for operational contracts shared across market-foundry runtimes.
> Last consolidated: S101.

---

## Purpose

This document formalizes the operational contracts that were previously implicit between runtimes. Every runtime in the Foundry must honor these contracts unless a documented exemption exists.

---

## 1. Composition Root Lifecycle (6-Phase Contract)

Every runtime follows a strict 6-phase startup sequence in its `cmd/*/run.go`:

| Phase | Description | Mandatory |
|-------|-------------|-----------|
| 1. Logger setup | `bootstrap.BuildLogger(config.Log)` + `slog.SetDefault(logger)` | Yes |
| 2. Startup log | `logger.Info("<runtime> starting", ...)` | Yes |
| 3. Actor engine | `actorcommon.NewDefaultEngine()` with `os.Exit(1)` on failure | Yes |
| 4. Runtime-specific wiring | NATS connections, trackers, adapters | Per-runtime |
| 5. Actor spawn + health server | Root supervisor + `healthz.NewHealthServer(...)` | Yes |
| 6. Shutdown sequence | `WaitTillShutdown(engine, pid)` then `srv.GracefulShutdown(5s)` | Yes |

**Invariants:**
- Phase 1 always precedes all other phases.
- Phase 3 failures are terminal (`os.Exit(1)`).
- Phase 6 always stops actors BEFORE the health server.

---

## 2. Health Server Contract

Every runtime MUST expose a health server with three endpoints:

| Endpoint | Semantics | Contract |
|----------|-----------|----------|
| `GET /healthz` | Liveness probe | Always returns `200 {"status": "ok"}`. No business logic checks. |
| `GET /readyz` | Readiness probe | Returns `200` only when all registered `ReadinessCheck` functions pass. Returns `503` with failure details otherwise. |
| `GET /statusz` | Activity status | Returns tracker metrics: event counts, error counts, idle duration, custom counters. |

**Construction pattern:**
```go
srv := healthz.NewHealthServer(
    config.HTTP.Addr,
    []healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
    trackers, // nil is acceptable if no pipeline trackers exist
)
srv.StartInBackground()
```

**Shutdown pattern:**
```go
actorcommon.WaitTillShutdown(engine, pid)
_ = srv.GracefulShutdown(5 * time.Second)
```

**Runtime-specific notes:**
- **Gateway** integrates readiness via HTTP routes (`/readyz` is part of the main HTTP server, not a separate health server). The gateway actor handles its own HTTP lifecycle.
- All other runtimes use `healthz.NewHealthServer` for a dedicated health endpoint.

---

## 3. Signal Handling Contract

All runtimes delegate signal handling to `actorcommon.WaitTillShutdown()`:

- **Signals caught:** `os.Interrupt`, `SIGTERM`, `SIGINT`.
- **Per-actor timeout:** 10 seconds via `engine.PoisonCtx(ctx, pid)`.
- **Concurrent shutdown:** All root PIDs are poisoned in parallel.
- **Final log:** `slog.Info("shutdown complete")` after all actors drain.

**No runtime may install its own signal handler.** The canonical entrypoint is the single owner of signal lifecycle.

---

## 4. Readiness Check Contract

Readiness checks follow the `healthz.ReadinessCheck` contract:

```go
type ReadinessCheck struct {
    Name  string
    Check func(ctx context.Context) error
}
```

**Standard checks:**
- `bootstrap.NATSReadinessCheck(config)` — TCP dial to NATS server with 2-second timeout. Returns error if NATS is disabled in config.

**Rules:**
- Readiness checks MUST be fast (< 5 seconds).
- Readiness checks MUST NOT modify state.
- A failed readiness check signals "not yet ready to serve traffic", not "broken".
- Runtimes that depend on NATS MUST include the NATS readiness check.

---

## 5. Structured Logging Contract

All runtimes use `log/slog` with these conventions:

| Convention | Rule |
|------------|------|
| Default logger | Installed via `slog.SetDefault(logger)` in phase 1 |
| Error key | Always `"error"`, never `"err"` or `"reason"` |
| Component context | Actors attach `slog.Default().With("actor", name, ...)` on first receive |
| Startup message | `logger.Info("<runtime> starting", ...)` — runtime name is the first word |
| Shutdown message | `slog.Info("shutdown complete")` — emitted by `WaitTillShutdown` |
| Health component | `slog.Default().With("component", "healthz")` — set internally by health server |

**Log levels:**
- `Error` — Operation failed, needs attention.
- `Warn` — Degraded state (optional dependency unavailable, component idle, redelivery).
- `Info` — Lifecycle events (starting, stopping, venue selected).
- `Debug` — Diagnostic detail (processing steps, values).

---

## 6. Error Handling Contract

All runtimes use `*problem.Problem` as the canonical error type across layer boundaries:

| Property | Semantics |
|----------|-----------|
| `Code` | Stable error classifier (`VAL_*`, `SYS_*`, `CFG_*`) |
| `Message` | Human-readable description |
| `Retryable` | Whether the caller should retry |
| `Cause` | Underlying `error` for `errors.Is`/`errors.As` |
| `Details` | Structured metadata (field, value) |

**Consumer message handling rules:**
- Decode failure with `InvalidArgument` → `msg.Term()` (terminal, do not retry).
- Decode failure with other codes → `msg.Nak()` (retry via redelivery).
- Successful handling → `msg.Ack()`.

---

## 7. Configuration Contract

All runtimes load configuration via:

```go
bootstrap.Main(serviceName, Run) // in main.go
```

- Config format: JSONC (JSON with comments).
- Flag: `-config` (default: `config.jsonc`).
- Lifecycle: `Load → ApplyDefaults → Validate → Run`.
- Validation includes cross-layer dependency checks (e.g., signal families require their evidence dependencies).

**Shared config sections:**
- `log` — Level + format (all runtimes).
- `http` — Address + timeouts (all runtimes with HTTP endpoints).
- `nats` — Connection URL + timeout (all NATS-dependent runtimes).
- `pipeline` — Family enablement + timeframes (store, derive, ingest).
- `venue` — Adapter selection (execute only).

---

## 8. NATS Connection Contract

Runtimes that communicate via NATS follow these patterns:

**Request/Reply (synchronous query):**
- Client: `NATSRequestClient` with configurable timeout.
- Responder: Queue-group subscription for load balancing.
- Payload: CBOR-encoded `Envelope[T]` with correlation ID.

**JetStream (event streaming):**
- Publisher: Creates stream if not exists (10s timeout), publishes with message ID.
- Consumer: Durable, explicit ack, configurable max-deliver and ack-wait.
- Redelivery detection: Check `msg.Metadata().NumDelivered`.

**Connection lifecycle:**
- Connections created in composition root (phase 4).
- Cleanup via `defer client.Close()` or `defer conns.Close(logger)`.
- No connection pooling — one connection per gateway/client.

---

## 9. Actor Lifecycle Contract

All actors follow the Hollywood framework conventions:

| Message | Required Handling |
|---------|-------------------|
| `actor.Initialized` | Ignore (via `ShouldIgnoreLifecycleMessage`) |
| `actor.Started` | Initialize resources, spawn children |
| `actor.Stopped` | Cleanup resources, close connections |
| Unknown messages | Log as warning unless `ShouldIgnoreLifecycleMessage` returns true |

**Supervisor rules:**
- Root supervisors own the lifecycle of their children.
- Children are spawned via `ctx.SpawnChild(producer, name)`.
- Poison pill cascades: poisoning a supervisor poisons all children.

---

## 10. Tracker Contract

Health trackers provide operational visibility without affecting business logic:

- `tracker.RecordEvent()` — Called on successful event processing.
- `tracker.RecordError()` — Called on processing errors (keeps tracker alive).
- `tracker.Counter(name)` — Domain-specific counters (e.g., "filled", "skipped_stale").

**Idle detection:**
- Heartbeat loop runs every 30 seconds.
- Warning logged when tracker idle > 2 minutes (configurable via `WithIdleThreshold`).
- Trackers with zero events are skipped (not yet active).

---

## Exemptions and Local Responsibilities

| Runtime | Exemption | Reason |
|---------|-----------|--------|
| Gateway | No separate health server | Readiness is integrated into the main HTTP server via route handlers. The gateway actor manages its own HTTP lifecycle. |
| Gateway | Custom readiness logic | Probes configctl (required) + evidence store (non-blocking). Domain-specific degradation rules. |
| Configctl | No pipeline trackers | Configctl manages configuration lifecycle, not data pipelines. Passes `nil` trackers to health server. |
| Execute | Config-driven venue adapter | Security-sensitive activation stays in explicit switch statement, not registry-driven. |
