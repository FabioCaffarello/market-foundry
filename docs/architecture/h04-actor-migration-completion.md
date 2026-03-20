# H-04: Actor Migration Completion

> Stage S219 — Completing the migration of store consumer actors to the generic infrastructure.

## Context

Stage S216 identified H-04 (generic actor infrastructure adoption) as **partial**: the `GenericConsumerActor` had been designed and implemented, but none of the 9 domain-specific consumer actors in the store scope had been migrated to use it. This left the system in a hybrid state where the infrastructure existed but delivered no value.

## Problem

The store scope contained 9 structurally identical consumer actor implementations:

| Actor File | Domain | NATS Consumer Type |
|---|---|---|
| `evidence_consumer_actor.go` | evidence/candle | `*natsevidence.CandleConsumer` |
| `trade_burst_consumer_actor.go` | evidence/tradeburst | `*natsevidence.TradeBurstConsumer` |
| `volume_consumer_actor.go` | evidence/volume | `*natsevidence.VolumeConsumer` |
| `signal_consumer_actor.go` | signal | `*natssignal.Consumer` |
| `decision_consumer_actor.go` | decision | `*natsdecision.Consumer` |
| `strategy_consumer_actor.go` | strategy | `*natsstrategy.Consumer` |
| `risk_consumer_actor.go` | risk | `*natsrisk.Consumer` |
| `execution_consumer_actor.go` | execution | `*natsexecution.Consumer` |
| `fill_consumer_actor.go` | execution/fill | `*natsexecution.FillConsumer` |

Each actor followed an identical pattern:
1. Config struct holding URL, ConsumerSpec, Registry, ProjectionPID, Tracker
2. Actor struct holding config, logger, domain-specific consumer
3. `Receive()` with Started/Stopped/default lifecycle
4. `start()` creating consumer with handler closure, calling Start(), handling error

The **only variance** between these 9 implementations was:
- Which NATS consumer constructor to call
- Which domain event type to handle in the callback
- Which message type to send to the projection actor

## Solution

Migrated all 9 consumer actors to use the existing `GenericConsumerActor` via `ConsumerStartFn` closures declared in `declarePipelines()`.

### How it works

`GenericConsumerActor` accepts a `ConsumerStartFn` — a closure that:
1. Creates the domain-specific NATS consumer with the appropriate handler
2. Calls `Start()` on it
3. Returns it as `io.Closer`

The closure captures the registry and message routing at declaration time, so `GenericConsumerActor` handles all lifecycle management (Started → start consumer, Stopped → close consumer) without knowing the domain type.

### Example: candle pipeline before vs after

**Before** (separate file + separate types):
```go
// evidence_consumer_actor.go — 93 lines
type EvidenceConsumerConfig struct { ... }
type EvidenceConsumerActor struct { ... }
func NewEvidenceConsumerActor(cfg EvidenceConsumerConfig) actor.Producer { ... }
func (a *EvidenceConsumerActor) Receive(c *actor.Context) { ... }
func (a *EvidenceConsumerActor) start(c *actor.Context) { ... }
```

**After** (inline closure in declarePipelines):
```go
NewConsumer: startConsumer("candle", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
    c := natsevidence.NewCandleConsumer(url, spec, reg.evidence, func(event evidence.CandleSampledEvent) {
        if tracker != nil { tracker.RecordEvent() }
        actorCtx.Send(projPID, candleReceivedMessage{Event: event})
    }, logger)
    return c, c.Start()
}),
```

## What was migrated

- 9 consumer actor files deleted (9 Config types, 9 Actor types, 9 constructors, 9 Receive methods, 9 start methods)
- All 10 pipeline entries in `declarePipelines()` updated to use `GenericConsumerActor` via `startConsumer` helper
- Zero behavioral changes — all message types, routing, and lifecycle remain identical

## What was NOT migrated (and why)

### Projection actors (9 implementations)
While projection actors share structural similarity (stats tracking, validation gates, lifecycle), they contain domain-specific logic that differs meaningfully:
- **Candle**: dual-bucket write (latest + history), candle-specific validation
- **Fill**: intent bucket cross-reference
- **Signal/Decision/Strategy/Risk**: different domain objects, different validation, different log fields

Generalizing projection actors would require either Go generics with complex constraints or interface-based dispatch that obscures the domain logic. The net reduction in code would not justify the increase in abstraction complexity.

### Derive scope publisher actors (5 implementations)
Publisher actors in the derive scope follow a similar pattern but are outside the scope of this migration. They are candidates for a future consolidation if the pattern proves stable.

## Verification

- `go build internal/actors/scopes/store` — clean
- `go vet internal/actors/scopes/store/...` — clean
- `go test internal/actors/scopes/store/...` — all tests pass
