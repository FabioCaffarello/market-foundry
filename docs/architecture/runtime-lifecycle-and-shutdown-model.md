# Runtime Lifecycle and Shutdown Model

## Overview

Foundry binaries follow a deterministic lifecycle with ordered startup phases and cooperative shutdown. The model ensures that no component starts before its dependencies and no component is abandoned during shutdown.

## Lifecycle Phases

### Phase 1: Config Load and Validation (pre-runtime)

Handled by `bootstrap.Main` before the service-specific `Run` function is called.

```
process start
  → flag.Parse()
  → bootstrap.LoadAndValidate(configPath)
    → settings.Load()      [read JSONC, strip comments, apply defaults]
    → cfg.Validate()        [cross-layer family dependency checks]
  → on error: stderr + os.Exit(1)
  → Run(cfg)
```

Failures at this phase produce a clear error message and exit immediately. No logger, no goroutines, no resources to clean up.

### Phase 2: Logger Bootstrap

```
Run(config)
  → bootstrap.BuildLogger(config.Log)   [JSON or text, configurable level]
  → slog.SetDefault(logger)
```

From this point, all structured logging is available.

### Phase 3: Actor Engine Creation

```
  → actorcommon.NewDefaultEngine()
  → on error: logger.Error + os.Exit(1)
```

The actor engine is the runtime's process supervisor. It must be created before any actors are spawned.

### Phase 4: Service-Specific Wiring

Each service constructs its domain dependencies:
- NATS clients and gateways
- Health trackers
- Venue adapters (execute only)
- Use cases (gateway only)

Resource cleanup is registered via `defer` at point of creation (e.g., `defer client.Close()`).

### Phase 5: Actor Spawn

```
  → pid := engine.Spawn(supervisor, "name")
```

The supervisor actor starts its child actors. The actor tree is now live.

### Phase 6: Health Server Start

```
  → srv := healthz.NewHealthServer(addr, checks, trackers)
  → srv.StartInBackground()
```

The health server starts after actor spawn so that:
- `/readyz` checks reflect actual infrastructure readiness (e.g., NATS connectivity)
- `/statusz` trackers are already registered and ready to receive events

### Phase 7: Steady State

```
  → actorcommon.WaitTillShutdown(engine, pid)
```

The process blocks on OS signals (`SIGINT`, `SIGTERM`). During steady state:
- The actor tree processes messages
- The health server responds to probes
- The heartbeat monitor logs warnings for idle trackers

### Phase 8: Graceful Shutdown

Triggered by `SIGINT` or `SIGTERM`:

```
signal received
  → actorcommon.WaitTillShutdown unblocks
    → PoisonCtx(10s timeout) for each actor PID
    → wait for all actors to drain
  → srv.GracefulShutdown(5s)
    → stop heartbeat monitor
    → HTTP server shutdown with timeout
  → deferred cleanups run (NATS client close, etc.)
  → process exits
```

### Shutdown Order

1. **Actors drain first** — the actor engine sends poison pills and waits up to 10 seconds for graceful stop
2. **Health server stops** — stops accepting probes, drains in-flight requests (5-second timeout)
3. **Deferred cleanups** — NATS connections, gateway clients close in LIFO order
4. **Process exits** — `slog.Info("shutdown complete")`

This order ensures:
- No new messages are processed after signal
- Health probes report unavailability before the process exits
- External connections are released cleanly

## Health Endpoints

| Endpoint   | Purpose | Behavior |
|-----------|---------|----------|
| `/healthz` | Liveness probe | Always returns 200 OK |
| `/readyz`  | Readiness probe | Runs all registered checks; 200 if all pass, 503 if any fail |
| `/statusz` | Operational visibility | JSON snapshot of all tracker activity, event counts, error counts, idle warnings |

### Readiness Checks

The canonical readiness check is `bootstrap.NATSReadinessCheck(config)`, which performs a TCP dial to the NATS server. Services can add additional checks by appending to the check slice.

### Idle Monitoring

The health server runs a heartbeat loop (every 30 seconds) that logs warnings when any tracker has been idle longer than the threshold (default: 2 minutes). This provides passive operational alerting without external monitoring infrastructure.

## Error Handling

| Phase | Error Strategy |
|-------|---------------|
| Config load/validation | stderr + os.Exit(1) |
| Engine creation | logger.Error + os.Exit(1) |
| Service wiring (critical) | logger.Error + os.Exit(1) |
| Service wiring (optional) | logger.Warn + continue (graceful degradation) |
| Health server start failure | Logged by health server logger; process continues |
| Shutdown timeout | Best-effort; process exits after timeouts |

## Invariants

1. No goroutines are spawned before the logger is configured
2. No actors are spawned before the engine is created
3. The health server starts after actor spawn
4. Actors drain before the health server stops
5. All resource cleanup is registered via `defer` at point of creation
6. Shutdown completes within a bounded time (10s actors + 5s health = 15s max)
