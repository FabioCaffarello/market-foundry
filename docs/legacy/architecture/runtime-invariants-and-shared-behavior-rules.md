# Runtime Invariants and Shared Behavior Rules

> Invariants that hold across all market-foundry runtimes. Violation of any invariant is a bug.
> Last consolidated: S101.

---

## Purpose

This document distinguishes between **invariants** (must always hold) and **conventions** (should hold unless a documented exemption applies). It also defines **local behaviors** that runtimes own independently.

---

## Part 1 — Cross-Runtime Invariants

These invariants are non-negotiable. They hold for every runtime and every code path.

### INV-1: Single Entrypoint

Every binary enters via `bootstrap.Main(serviceName, Run)`. There is exactly one `main()` function per binary, and it delegates to `bootstrap.Main` immediately.

**Why:** Ensures consistent config loading, flag parsing, and error reporting.

### INV-2: Logger Installed Before Any Work

`slog.SetDefault(logger)` is called in phase 1 of the composition root, before any actor creation, NATS connection, or goroutine launch.

**Why:** Guarantees all log output is structured and level-controlled from the first line.

### INV-3: Engine Failure Is Terminal

If `actorcommon.NewDefaultEngine()` returns an error, the runtime calls `os.Exit(1)`. No recovery, no retry.

**Why:** The actor engine is the process-level runtime. If it cannot start, nothing else can function.

### INV-4: Actors Stop Before Health Server

`actorcommon.WaitTillShutdown(engine, pid)` completes before `srv.GracefulShutdown()` is called. This ordering is mandatory.

**Why:** The health server must remain available to report "shutting down" while actors drain. Stopping the health server first would create a blind spot during shutdown.

### INV-5: Error Key Is "error"

All structured log calls use `"error"` as the key for error values. Never `"err"`, `"reason"`, `"cause"`, or variants.

**Why:** Consistent log parsing and alerting. Log aggregation queries depend on a single key.

### INV-6: No init() Registration

No package uses `init()` functions for registration, side effects, or global state mutation. All wiring happens explicitly in composition roots.

**Why:** `init()` creates invisible coupling, untestable dependencies, and import-order sensitivity.

### INV-7: No Cross-Domain Imports in Domain Layer

Packages under `internal/domain/` never import other domain packages. Domain isolation is absolute.

**Why:** Domains are bounded contexts. Cross-domain communication goes through ports/adapters.

### INV-8: problem.Problem Across Layer Boundaries

All functions that cross architectural boundaries (port → adapter, application → port) return `*problem.Problem`, not `error`. Only infrastructure-level code (NATS connections, HTTP listeners) uses raw `error`.

**Why:** `*problem.Problem` carries classification, retryability, and structured details that raw errors cannot.

### INV-9: Compile-Time Interface Proof

Every adapter that implements a port interface includes a compile-time assertion:

```go
var _ ports.SomeGateway = (*AdapterImpl)(nil)
```

**Why:** Catches interface drift at compile time, not at runtime.

### INV-10: Graceful Shutdown Timeout Consistency

| Component | Timeout | Rationale |
|-----------|---------|-----------|
| Actor poison pill | 10 seconds | Actors may need to flush buffers, close streams |
| Health server shutdown | 5 seconds | HTTP drain is fast; no business logic |
| Gateway HTTP server | 5 seconds | Matches health server; in-flight requests drain |

These timeouts are hardcoded constants, not configurable. They represent operational contracts with deployment infrastructure (e.g., Kubernetes `terminationGracePeriodSeconds`).

---

## Part 2 — Shared Behavior Rules

These are strong conventions. A runtime may deviate only with a documented exemption.

### BHV-1: Startup Log Format

Every runtime emits a startup log at `Info` level as its first log message:

```
logger.Info("<runtime> starting", ...optional context fields...)
```

The runtime name is always the first word. Optional fields carry runtime-specific context (e.g., `"addr"` for gateway, `"timeframes"` for derive).

### BHV-2: NATS Readiness Check

Every runtime that depends on NATS includes `bootstrap.NATSReadinessCheck(config)` in its health server readiness checks.

**Exemption:** Gateway uses custom readiness logic that subsumes the NATS check.

### BHV-3: Health Server on config.HTTP.Addr

All health servers bind to the address specified in `config.HTTP.Addr`. No runtime uses a separate port for health endpoints.

**Exemption:** Gateway's health endpoints are part of the main application server.

### BHV-4: Tracker Naming Convention

Trackers follow the pattern `<family>-<role>`:
- `candle-projection`, `candle-consumer` (store)
- `evidence-publisher` (derive)
- `observation-publisher` (ingest)
- `venue-adapter`, `venue-consumer` (execute)

### BHV-5: Connection Cleanup via Defer

NATS connections and request clients are cleaned up via `defer` in the composition root. Cleanup happens after `WaitTillShutdown` returns (deferred functions run on function exit).

### BHV-6: Registry as Value Object

NATS registries (`EvidenceRegistry`, `SignalRegistry`, etc.) are stateless value objects created via `Default*Registry()` factory functions. They carry subject/stream/bucket mappings, never lifecycle or state.

### BHV-7: Config-Driven Activation

Pipeline families are activated via `IsEnabled` predicates derived from `PipelineConfig`. Runtimes skip disabled families silently (log at debug level, not warn).

**Exemption:** Venue adapters use explicit switch-case activation (security-sensitive).

---

## Part 3 — Local Behaviors (Runtime-Owned)

These behaviors are intentionally local. Standardizing them would reduce clarity or add artificial constraints.

### LOCAL-1: Gateway Connection Topology

The gateway creates one NATS request client per domain gateway. This is a local optimization — other runtimes share a single connection. The gateway's `gatewayConns` struct and its `Close()` method are gateway-private patterns.

### LOCAL-2: Supervisor Internal Structure

Each supervisor decides its own child topology:
- **Store:** Pipeline catalog with projection + consumer pairs per family.
- **Derive:** Processor catalogs with dynamic SourceScopeActor spawning.
- **Ingest:** Binding watcher + dynamic ExchangeScopeActor spawning.
- **Execute:** VenueAdapter + ExecutionConsumer pair.
- **Configctl:** Config supervisor (single actor).

No unified supervisor framework exists by design.

### LOCAL-3: Tracker Granularity

Each runtime decides how many trackers it needs:
- Store: Two per enabled pipeline family (projection + consumer).
- Derive: One (evidence-publisher).
- Ingest: One (observation-publisher).
- Execute: Two (venue-adapter + venue-consumer).
- Configctl: None (no pipeline processing).

### LOCAL-4: Readiness Check Composition

Each runtime decides which readiness checks to register:
- Store/Derive/Ingest/Execute: NATS TCP connectivity.
- Gateway: Configctl availability + optional evidence store probe.
- Configctl: NATS TCP connectivity.

### LOCAL-5: NATS Consumer Configuration

Each consumer owns its durability, ack policy, max delivery, and filter subjects. These are domain-specific and not standardized.

---

## Part 4 — Verification Checklist

When adding a new runtime or modifying an existing one, verify:

- [ ] Uses `bootstrap.Main(serviceName, Run)` as entrypoint
- [ ] Calls `slog.SetDefault(logger)` before any other work
- [ ] Emits `logger.Info("<runtime> starting", ...)` as first log
- [ ] Creates actor engine with terminal exit on failure
- [ ] Exposes health server (or integrates health into main HTTP server)
- [ ] Stops actors before health server in shutdown sequence
- [ ] Uses `"error"` key in all structured log calls
- [ ] Uses `*problem.Problem` for cross-boundary error returns
- [ ] No `init()` registration or global mutable state
- [ ] NATS connections cleaned up via `defer`
- [ ] Compile-time interface assertions for all port implementations

---

## Trade-offs and Limits

### What this document does NOT standardize

1. **Actor message types** — Each supervisor handles domain-specific messages.
2. **Consumer retry policies** — AckWait, MaxDeliver are domain decisions.
3. **Stream configuration** — Retention, storage type, max age are per-domain.
4. **HTTP route structure** — Only gateway exposes HTTP routes; route organization is gateway-local.
5. **Domain model shapes** — Domains are bounded contexts with independent models.

### Why not a unified lifecycle interface?

A shared `Runtime` interface (with `Start()`, `Stop()`, `Ready()` methods) was considered and rejected:
- Each runtime's startup is meaningfully different (gateway builds HTTP routes, store builds pipeline catalogs, execute selects venue adapters).
- A unified interface would either be too generic to be useful or too specific to be truly shared.
- The 6-phase lifecycle pattern achieves consistency through convention, not through a type constraint.
- At 6 runtimes, the composition roots are small enough to verify by inspection.

### Why hardcoded timeouts?

Configurable shutdown timeouts add complexity without clear value:
- The actor poison timeout (10s) must coordinate with orchestrator `terminationGracePeriodSeconds`.
- Making this configurable invites misconfiguration where the process is killed before actors finish draining.
- If a specific runtime needs different timeouts, it should be discussed as an architectural decision, not a config knob.
