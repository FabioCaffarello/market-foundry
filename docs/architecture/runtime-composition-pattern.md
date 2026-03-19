# Runtime Composition Pattern

## Overview

Every foundry binary follows a canonical composition pattern that separates startup ceremony from service-specific wiring. The pattern is explicit, lightweight, and avoids framework-level abstractions.

## Canonical Structure

Each binary consists of two files:

### `main.go` — Entrypoint Ceremony

```go
package main

import "internal/shared/bootstrap"

func main() {
    bootstrap.Main("service-name", Run)
}
```

`bootstrap.Main` handles:
- `-config` flag parsing (default: `config.jsonc`)
- Config loading from JSONC
- Config validation (including cross-layer pipeline dependency checks)
- Error reporting to stderr with service-name prefix
- Delegation to the service-specific `Run` function

### `run.go` — Service Wiring

The `Run(config settings.AppConfig)` function follows this sequence:

1. **Logger bootstrap** — `bootstrap.BuildLogger(config.Log)` + `slog.SetDefault`
2. **Actor engine creation** — `actorcommon.NewDefaultEngine()`
3. **Service-specific setup** — gateways, adapters, trackers (varies per service)
4. **Actor spawn** — `engine.Spawn(supervisor, "name")`
5. **Health server start** — `srv.StartInBackground()`
6. **Block until signal** — `actorcommon.WaitTillShutdown(engine, pid)`
7. **Graceful shutdown** — `srv.GracefulShutdown(5 * time.Second)`

## Shared Building Blocks

### `bootstrap.Main(serviceName, runFn)`
Canonical entrypoint. Encapsulates flag parsing, config load, validation, and error handling. Every binary uses this.

### `bootstrap.BuildLogger(logConfig)`
Creates a structured `*slog.Logger` from config (JSON or text format, configurable level).

### `bootstrap.LoadAndValidate(path)`
Loads JSONC config, applies defaults, validates schema and cross-layer dependencies.

### `bootstrap.NATSReadinessCheck(config)`
Returns a `healthz.ReadinessCheck` that verifies TCP connectivity to the NATS server. Used by all NATS-dependent services (derive, ingest, store, execute).

### `healthz.NewHealthServer(addr, checks, trackers)`
Provides `/healthz` (liveness), `/readyz` (readiness), `/statusz` (tracker activity). All NATS-dependent services use this.

### `healthz.StartInBackground()`
Starts the health server in a goroutine with error logging. Canonical way to launch alongside actor engine.

### `healthz.GracefulShutdown(timeout)`
Creates timeout context and shuts down the health server. Canonical shutdown companion.

## Service-Specific Responsibilities

Each service's `run.go` is responsible for its own domain wiring:

| Service   | Specific Wiring |
|-----------|----------------|
| configctl | Spawns config supervisor (no health server, no NATS) |
| gateway   | Builds 8 optional NATS gateways, wires use cases, spawns HTTP gateway actor |
| ingest    | Creates configctl gateway, spawns ingest supervisor with publisher tracker |
| derive    | Creates configctl gateway, spawns derive supervisor with publisher tracker |
| store     | Builds dynamic tracker map from pipeline family config, spawns store supervisor |
| execute   | Builds venue adapter via config-driven selection, spawns execute supervisor |

## Intentional Limits

- **No generic `Service` struct or interface.** Each binary owns its wiring explicitly.
- **No lifecycle manager.** The sequence (logger → engine → wiring → spawn → health → wait → shutdown) is written out in each `run.go`. This keeps control flow visible.
- **No dependency injection container.** Dependencies are constructed inline and passed directly.
- **Shared code is limited to pure ceremony** (config load, logger, readiness checks, health server lifecycle). Domain-specific setup stays local.

## When to Add a New Building Block

A function should be promoted to `bootstrap` only when:
1. It is used identically by 3+ binaries
2. It has no domain-specific behavior
3. Its interface is stable and unlikely to diverge across services
