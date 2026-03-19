# Registry-Driven Runtime Assembly

> Canonical reference for when and how to use registry-driven assembly in Market Foundry.

## Principle

Runtime assembly should be **catalog-driven**: each domain scope declares its pipelines, processors, or connections in a single declarative catalog, and the composition root derives everything it needs from that catalog. No secondary lists that must be kept in sync.

## What "Registry-Driven" Means Here

In Market Foundry, "registry" does NOT mean a global service locator or a DI container. It means:

1. **A declarative catalog** — a function or slice that returns all available units (pipelines, processors, gateways) with their metadata and factories.
2. **A composition root** that iterates the catalog, applies config-driven filters, and wires only what's enabled.
3. **Derived artifacts** — health trackers, log fields, readiness checks — computed from the catalog rather than maintained as separate lists.

This is registry-as-data, not registry-as-framework.

## Canonical Patterns

### Pipeline Catalog (Store)

The store binary's `declarePipelines()` is the single source of truth for all projection pipelines. Each `Pipeline` entry carries:

- Identity: `Scope`, `Family`, `ProjectionName`, `ConsumerName`
- Infrastructure: `Buckets`, `ConsumerSpec`
- Activation: `IsEnabled` predicate
- Factories: `NewProjection`, `NewConsumer` (registry bound via closure)

**Derived artifacts:**

| Artifact | Derived From | Previously |
|----------|-------------|------------|
| Health trackers | `PipelineTrackerDefs()` iterates catalog | Separate `allTrackerDefs` list (removed) |
| Active scopes | Collected during pipeline spawn loop | Same |
| Query responder registries | `pipelineRegistries.queryResponderConfig()` | Manual if/else blocks |

**Adding a new pipeline** requires exactly ONE entry in `declarePipelines()`. Trackers, scope tracking, and query responder wiring follow automatically.

### Processor Catalog (Derive)

The derive binary declares processors per scope (evidence, signal, decision, strategy, risk, execution) as slice literals in the supervisor's `start()` method. The generic `filterEnabled[T]` function applies config predicates uniformly:

```go
s.processors = filterEnabled(allProcessors,
    func(p FamilyProcessor) string { return p.Family },
    s.cfg.Pipeline.IsFamilyEnabled, s.logger, "evidence")
```

**Adding a new processor** requires ONE entry in the appropriate slice. Filtering, skip-logging, and family name collection for startup logs are handled by the generic helpers.

### Gateway Connection Catalog (Gateway)

All domain gateways follow the same create-request-client → wrap-with-constructor pattern. The generic `newGatewayConn[T]` function eliminates per-scope boilerplate:

```go
conns.signal, cl, p = newGatewayConn(config, "signal",
    func(rc *adapternats.NATSRequestClient) ports.SignalGateway {
        return adapternats.NewSignalGateway(rc, "gateway.http")
    })
```

**Adding a new gateway** requires ONE call to `newGatewayConn` + ONE field in `gatewayConns`.

## When Registry-Driven Assembly is Appropriate

| Situation | Use Registry? | Reason |
|-----------|:---:|--------|
| Multiple units of the same kind (pipelines, processors, gateways) | Yes | Eliminates list duplication and per-unit boilerplate |
| Derived artifacts (trackers, log fields) that mirror a primary list | Yes | Single source of truth prevents drift |
| Repeated structural pattern with per-unit variation only in factories | Yes | Generic helper + catalog entry |
| Unique one-off wiring (configctl gateway, observation consumer) | No | No duplication to eliminate |
| Security-sensitive activation (venue adapter) | No | Must stay explicit per architectural rules |
| Actor constructor wiring (Hollywood Producer pattern) | No | Already minimal; abstraction adds indirection without gain |

## When NOT to Use Registry-Driven Assembly

1. **When there's only one instance.** A registry of one is just indirection.
2. **When the units have fundamentally different shapes.** The derive processor types (6 types) have different `NewActor` signatures because the pipeline chain requires it. Forcing them into a single type would lose type safety.
3. **When activation has security implications.** Venue adapter selection must remain explicit — never auto-discovered.
4. **When it would hide initialization order.** The 6-phase lifecycle must remain visible in `run.go`.
5. **When it requires `init()` or package-level registration.** All registration must happen in composition roots, never in `init()`.

## Relationship to Existing Patterns

- **Composition roots** remain the only place where wiring happens. Catalogs are declared inside composition roots or their direct callees.
- **Constructor injection** is unchanged. Catalogs provide factories that return producers; the composition root calls them with explicit dependencies.
- **The 6-phase lifecycle** is unchanged. Catalogs are consulted during Phase 2 (dependency composition) and Phase 4 (actor spawn).
- **NATS registries** (EvidenceRegistry, SignalRegistry, etc.) remain stateless value objects. They are captured by closure in pipeline declarations, not injected via a service locator.

## Anti-Patterns

| Anti-Pattern | Why It's Wrong |
|-------------|---------------|
| Global mutable registry that accepts `Register()` calls | Hides what's registered; breaks explicit composition |
| Auto-discovery via reflection or `init()` | Makes the assembly graph invisible; violates composition root ownership |
| Single unified pipeline type for all scopes | Loses type safety; execution processors genuinely differ from evidence processors |
| Registry that manages lifecycle (Start/Stop) | Lifecycle belongs to the actor system, not to a registry |
| Catalog that returns interfaces instead of concrete factories | Adds indirection without gain; the composition root knows the concrete types |
