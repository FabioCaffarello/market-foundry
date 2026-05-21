# Derive Family Processor Pattern

> Canonical pattern for registering and spawning evidence family processors in the derive binary.

## Problem

The derive binary produces multiple evidence types (candle, trade burst, and future types). Before this pattern, adding a new evidence type required modifying `SourceScopeActor.onActivateSampler` to hardcode the new sampler spawn logic alongside existing types. This violated the open-closed principle and made it unclear where new families should be registered.

## Solution: FamilyProcessor

A `FamilyProcessor` is a declarative struct that describes one evidence family's processing pipeline within derive. It is **not** an interface, **not** a generic framework — it is a simple data declaration.

```go
type FamilyProcessor struct {
    Family      string    // canonical family name: "candle", "tradeburst"
    ActorPrefix string    // name prefix for actor naming: "sampler", "burst-sampler"
    NewActor    func(source, symbol string, timeframe time.Duration, publisherPID *actor.PID) actor.Producer
}
```

### Registration Point

All family processors are registered in `DeriveSupervisor.start()`. This is the **single point of truth** for which evidence families the derive binary produces:

```go
s.processors = []FamilyProcessor{
    {
        Family:      "candle",
        ActorPrefix: "sampler",
        NewActor: func(source, symbol string, tf time.Duration, pub *actor.PID) actor.Producer {
            return NewSamplerActor(SamplerConfig{...})
        },
    },
    {
        Family:      "tradeburst",
        ActorPrefix: "burst-sampler",
        NewActor: func(source, symbol string, tf time.Duration, pub *actor.PID) actor.Producer {
            return NewTradeBurstSamplerActor(TradeBurstSamplerConfig{...})
        },
    },
}
```

### Spawning

`SourceScopeActor.onActivateSampler` iterates over the registered processors. For each processor and each configured timeframe, it spawns one sampler actor:

```
SourceScopeActor (source=binancef)
├── publisher
├── sampler-btcusdt-60s        (candle family, 60s timeframe)
├── burst-sampler-btcusdt-60s  (tradeburst family, 60s timeframe)
├── sampler-ethusdt-60s        (candle family, 60s timeframe)
└── burst-sampler-ethusdt-60s  (tradeburst family, 60s timeframe)
```

The actor naming convention is: `{ActorPrefix}-{symbol}-{timeframe}s`.

## What Each Layer Owns

### DeriveSupervisor

- **Registers** which family processors exist (the `processors` list)
- **Routes** trades to the correct SourceScopeActor by source
- **Passes** the processors list to each SourceScopeActor

### SourceScopeActor

- **Spawns** one sampler actor per (processor × symbol × timeframe)
- **Owns** the evidence publisher shared by all samplers in this source scope
- **Routes** trades to all samplers for the relevant symbol

### Sampler Actors (per family)

- **Own** the pure application logic (CandleSampler, TradeBurstSampler)
- **Receive** trades via `tradeReceivedMessage`
- **Send** publish messages to the shared publisher actor
- Each sampler actor is a self-contained unit with no cross-family dependencies

### EvidencePublisherActor

- **Owns** the NATS JetStream connection for all evidence types in this source scope
- **Handles** per-type publish messages (publishCandleMessage, publishTradeBurstMessage)
- Remains explicitly typed — no generic publish interface

## Adding a New Evidence Type

To add a new evidence family (e.g., `volume`), follow these steps:

### Step 1: Domain type

Create the domain type in `internal/domain/evidence/`:
- `volume.go` — `EvidenceVolume` struct with validation
- Add `VolumeSampledEvent` to `events.go`

### Step 2: Application logic

Create the sampler in `internal/application/derive/`:
- `volume_sampler.go` — `VolumeSampler` with `AddTrade()` and `WindowFor()`
- `volume_sampler_test.go` — comprehensive tests

### Step 3: Sampler actor

Create the actor in `internal/actors/scopes/derive/`:
- `volume_sampler_actor.go` — `VolumeSamplerActor` following the same pattern as `SamplerActor`

### Step 4: Publish message

Add to `messages.go`:
```go
type publishVolumeMessage struct {
    Event evidence.VolumeSampledEvent
}
```

### Step 5: Publisher handling

Add a case to `EvidencePublisherActor.Receive`:
```go
case publishVolumeMessage:
    // publish via adapter
```

Add `PublishVolume` to the NATS `EvidencePublisher` adapter.

### Step 6: Register the processor

Add one entry to the `processors` list in `DeriveSupervisor.start()`:
```go
{
    Family:      "volume",
    ActorPrefix: "volume-sampler",
    NewActor: func(source, symbol string, tf time.Duration, pub *actor.PID) actor.Producer {
        return NewVolumeSamplerActor(VolumeSamplerConfig{...})
    },
},
```

### Step 7: Registry spec

Add `VolumeSampled` to `EvidenceRegistry` in the NATS adapter.

**That's it.** No modification to SourceScopeActor, DeriveSupervisor routing, or ConsumerActor.

## What This Pattern Is NOT

- **Not a plugin system.** Processors are compiled in, not loaded dynamically.
- **Not a generic framework.** Each sampler actor is a concrete type with explicit message handling. There is no `Sampler` interface.
- **Not an abstraction over the publisher.** The publisher remains explicitly typed per evidence family. Type safety at the NATS boundary is preserved.
- **Not configurable at runtime.** The processor list is fixed at compile time. Config-driven evidence type activation is a separate concern (see signal readiness review).

## Design Rationale

### Why not a `Sampler` interface?

A generic interface would unify the actor layer but would:
- Lose type safety on publish messages
- Require type assertions in the publisher
- Make each sampler's specific logic harder to trace
- Add indirection without reducing code

The current approach has one file per family at the actor layer. This is explicit, greppable, and each file is self-contained.

### Why not per-family publishers?

One publisher per evidence type would mean N NATS connections per source scope (where N = number of families). The current shared publisher with per-type message handling uses one connection and one JetStream context, which is more efficient and simpler to manage.

### Why `func` instead of `interface` for NewActor?

A factory function is the simplest way to parameterize actor creation. It avoids defining an interface that would need to be implemented by each sampler config type. The function captures the config type at registration time, keeping everything type-safe.
