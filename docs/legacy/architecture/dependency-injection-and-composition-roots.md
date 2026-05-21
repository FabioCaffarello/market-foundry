# Dependency Injection and Composition Roots

> Canonical reference for DI patterns accepted in market-foundry.

## Principles

1. **No external DI framework.** All composition is explicit, constructor-based, and visible in source.
2. **Composition roots are the only places where dependencies are created and wired.** Domain, application, and adapter code receive dependencies — they never create them.
3. **Each runtime has exactly one composition root** — the `Run()` function in `cmd/<service>/run.go`, optionally delegating to composition helpers in `compose.go`.
4. **Registries are value objects.** NATS registries (`EvidenceRegistry`, `SignalRegistry`, etc.) are stateless structs passed by value. They declare stream/subject contracts, not runtime state.
5. **Config drives activation, not code.** Which families/pipelines/gateways are enabled is determined by `settings.AppConfig`, never by compile-time flags or conditional imports.

## Accepted Patterns

### Constructor injection

All dependencies are passed via constructor parameters. No `SetX()` methods, no global state, no init() side effects.

```go
func NewSomeActor(config SomeConfig, gateway ports.SomeGateway, tracker *healthz.Tracker) actor.Producer
```

### Factory functions for infrastructure

Infrastructure objects with lifecycle (NATS connections, HTTP clients) are created by factory functions that return `(resource, closerFunc, error)`:

```go
func newEvidenceGateway(config settings.AppConfig) (ports.EvidenceGateway, func() error, *problem.Problem)
```

The composition root collects all closer functions and calls them on shutdown.

### Closure-bound registries

When multiple pipeline types share the same structural shape but differ in registry type, the registry is captured via closure at declaration time:

```go
NewConsumer: func(natsURL string, spec ConsumerSpec, projPID *actor.PID, tracker *Tracker) actor.Producer {
    return NewSignalConsumerActor(SignalConsumerConfig{
        Registry: sigRegistry,  // captured from outer scope
        // ...
    })
},
```

This eliminates the need for separate struct types per registry kind.

### Config-driven activation predicates

Pipelines carry an `IsEnabled` function that evaluates against `PipelineConfig`:

```go
IsEnabled: func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("rsi") },
```

This keeps activation logic co-located with pipeline declaration rather than scattered across supervisor code.

### Graceful degradation for optional dependencies

Optional gateways (evidence, signal, etc.) return `nil` when unavailable. Use case wiring checks for `nil` before creating use cases. Route registration checks `HasAny()` before adding routes.

## What Must Stay Manual

- **Actor constructor wiring.** Hollywood actors use the Producer pattern (`func() actor.Receiver`). Each actor's dependencies are bound at spawn time. This is intentional — actors own their lifecycle.
- **NATS connection creation.** Each gateway/publisher/consumer manages its own NATS connection. Connection pooling would hide failure boundaries.
- **Venue adapter selection.** The `buildVenueAdapter()` switch is an activation gate with security implications. It must remain explicit.

## What Must NOT Be Done

- **Do not introduce a DI container or service locator.** The codebase is small enough that explicit wiring is both readable and maintainable.
- **Do not use `init()` for registration.** All registration happens in composition roots or declarative catalogs.
- **Do not pass `AppConfig` through the entire call chain.** Extract the specific fields each component needs into its own config struct.
- **Do not create "god" assembler objects.** Each composition root is scoped to one runtime. Shared helpers live in `bootstrap/`.
