# Runtime Assembly Guidelines

> How foundry runtimes bootstrap, compose dependencies, and shut down.

## Canonical Runtime Structure

Every foundry binary follows the same lifecycle:

```
main.go
  └─ bootstrap.Main(serviceName, Run)
       ├─ flag parsing + config load + validation
       └─ Run(config AppConfig)
            ├─ Phase 1: Infrastructure (logger, actor engine)
            ├─ Phase 2: Dependency composition (connections, adapters)
            ├─ Phase 3: Use case / route wiring
            ├─ Phase 4: Actor spawn (root supervisor)
            ├─ Phase 5: Health server start
            └─ Phase 6: WaitTillShutdown → graceful teardown
```

### Phase separation matters

Each phase has a clear responsibility. Do not mix phases:

- **Phase 2 creates resources** — NATS clients, venue adapters, health trackers. Resources with lifecycle return closer functions.
- **Phase 3 wires application logic** — use cases, route dependencies, readiness checkers. No I/O happens here.
- **Phase 4 spawns actors** — the root supervisor receives its dependencies via constructor. Child actors are spawned by the supervisor, not by `Run()`.

### File organization

| Runtime   | Composition root | Infrastructure factories | Readiness |
|-----------|-----------------|-------------------------|-----------|
| gateway   | `run.go` + `compose.go` | `gateway.go` | `readiness.go` |
| store     | `run.go` | (trackers built inline) | via `bootstrap.NATSReadinessCheck` |
| execute   | `run.go` | `buildVenueAdapter` inline | via `bootstrap.NATSReadinessCheck` |
| derive    | `run.go` | configctl client inline | via `bootstrap.NATSReadinessCheck` |

When a runtime's `Run()` function exceeds ~80 lines of wiring, extract composition helpers into a `compose.go` file in the same package. The gateway runtime demonstrates this pattern.

## Pipeline Declaration Pattern (Store)

The store supervisor uses a declarative pipeline catalog:

```go
allPipelines, registries := declarePipelines()
```

Each `Pipeline` entry is self-contained:
- **Scope** — which domain family (evidence, signal, ...) for registry injection
- **IsEnabled** — config predicate for activation
- **NewProjection / NewConsumer** — actor factories with registry bound via closure

Adding a new pipeline type means:
1. Add one `Pipeline` entry in `declarePipelines()`
2. Add one `trackerDef` entry in `cmd/store/run.go`
3. Implement the projection and consumer actors

The supervisor's `start()` method is a single filter-and-spawn loop.

## Health Tracker Convention

Each projection pipeline gets exactly two trackers:
- `{name}-projection` — tracks materialization activity
- `{name}-consumer` — tracks event delivery

Trackers are created in `Run()` and passed to the supervisor. The supervisor distributes them to child actors during spawn. This ensures health visibility is established before any actor starts processing.

## Shutdown Discipline

1. `WaitTillShutdown(engine, pid)` blocks on SIGINT/SIGTERM
2. Actor engine poisons the root supervisor (10s timeout)
3. Health server gracefully shuts down (5s timeout)
4. Gateway connections close via deferred `Close()`

Closers registered via `defer` in `Run()` execute in LIFO order. The composition root owns all cleanup — actors do not manage connection lifecycle.

## Adding a New Runtime

1. Create `cmd/<name>/main.go` with `bootstrap.Main("<name>", Run)`
2. Create `cmd/<name>/run.go` following the 6-phase structure
3. If wiring exceeds ~80 lines, extract into `compose.go`
4. Add health server with `NATSReadinessCheck` if NATS-dependent
5. Register in `go.work` and `deploy/compose/docker-compose.yaml`
