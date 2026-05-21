# Bollinger Signal Derive Path and Ownership

## Ownership Model

The Bollinger signal family follows the canonical ownership model established for all derive families:

| Concern | Owner | Rationale |
|---------|-------|-----------|
| Signal computation (%B, SMA, bands) | `BollingerSampler` (application layer) | Pure application logic, no I/O |
| Squeeze detection (decision) | `BollingerSqueezeEvaluator` (application layer) | Pure application logic, no I/O |
| Actor lifecycle & message routing | `BollingerSignalSamplerActor` / `BollingerSqueezeEvaluatorActor` (actor layer) | Owns actor state, delegates computation to application layer |
| Fan-out routing | `SourceScopeActor` | Canonical scope-level fan-out for all families |
| NATS publishing | `SignalPublisherActor` / `DecisionPublisherActor` | Shared publisher per source scope |
| NATS subject contracts | `natssignal.Registry` / `natsdecision.Registry` | Centralized subject/stream definitions |
| Configuration enablement | `settings.PipelineConfig` | `signal_families` / `decision_families` arrays |
| Supervisor registration | `DeriveSupervisor` | `signalProcessors` / `decisionProcessors` arrays |

## Derive Path (Actor Tree)

```
DeriveSupervisor
  └── SourceScopeActor (per source/exchange)
        ├── EvidencePublisherActor (shared)
        ├── SignalPublisherActor (shared)
        ├── DecisionPublisherActor (shared)
        ├── CandleSamplerActor (per symbol × timeframe)
        ├── BollingerSignalSamplerActor (per symbol × timeframe)
        ├── BollingerSqueezeEvaluatorActor (per symbol × timeframe)
        └── ... (other families)
```

## Actor Naming Convention

Actors are named using the pattern: `{prefix}-{symbol}-{timeframe_seconds}s`

- Signal sampler: `signal-bollinger-btcusdt-60s`
- Decision evaluator: `decision-bollinger-squeeze-btcusdt-60s`

## Registration Points

### Signal Processor (derive_supervisor.go)

```go
{
    Family:      "bollinger",
    ActorPrefix: "signal-bollinger",
    NewActor: func(source, symbol string, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
        return NewBollingerSignalSamplerActor(SignalSamplerConfig{
            Source: source, Symbol: symbol, Timeframe: tf,
            SignalPublisherPID: sigPub, ScopePID: scopePID,
        })
    },
}
```

### Decision Processor (derive_supervisor.go)

```go
{
    Family:      "bollinger_squeeze",
    ActorPrefix: "decision-bollinger-squeeze",
    NewActor: func(source, symbol string, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
        return NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
            Source: source, Symbol: symbol, Timeframe: tf,
            DecisionPublisherPID: decPub, ScopePID: scopePID,
        })
    },
}
```

## NATS Subject Contracts

| Purpose | Subject Pattern | Stream |
|---------|----------------|--------|
| Signal published | `signal.events.bollinger.generated.{source}.{symbol}.{timeframe}` | `SIGNAL_EVENTS` |
| Signal query | `signal.query.bollinger.latest.{source}.{symbol}.{timeframe}` | — |
| Decision published | `decision.events.bollinger_squeeze.evaluated.{source}.{symbol}.{timeframe}` | `DECISION_EVENTS` |
| Decision query | `decision.query.bollinger_squeeze.latest.{source}.{symbol}.{timeframe}` | — |

## Boundary Constraints

1. **No cross-family coupling**: `BollingerSignalSamplerActor` only produces `bollinger` signals. `BollingerSqueezeEvaluatorActor` only consumes `bollinger` signals (type-guarded in evaluator).
2. **No direct actor-to-actor wiring**: All inter-stage routing goes through `SourceScopeActor` fan-out.
3. **Application logic is I/O-free**: `BollingerSampler` and `BollingerSqueezeEvaluator` have no NATS, actor, or external dependencies.
4. **Configuration-gated**: Both families require explicit opt-in via `pipeline.signal_families` and `pipeline.decision_families`.
