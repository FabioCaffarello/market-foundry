# Bollinger Signal End-to-End Wiring

## Overview

This document describes the complete end-to-end wiring path for the Bollinger signal family within the market-foundry derive actor system. As of S288, all wiring gaps have been closed and the Bollinger signal flows from candle evidence through to decision evaluation via the standard actor chain.

## Signal Flow Path

```
Trade → CandleSamplerActor → candleFinalizedMessage
  → BollingerSignalSamplerActor → publishSignalMessage → NATS (signal.events.bollinger.generated.>)
                                → signalGeneratedMessage → SourceScopeActor fan-out
  → BollingerSqueezeEvaluatorActor → publishDecisionMessage → NATS (decision.events.bollinger_squeeze.evaluated.>)
                                   → decisionEvaluatedMessage → SourceScopeActor fan-out
  → [downstream strategy/risk/execution — not yet wired for this family]
```

## Component Inventory

| Layer | Component | File | Status |
|-------|-----------|------|--------|
| Application | `BollingerSampler` | `internal/application/signal/bollinger_sampler.go` | Complete |
| Application | `BollingerSqueezeEvaluator` | `internal/application/decision/bollinger_squeeze_evaluator.go` | Complete |
| Actor | `BollingerSignalSamplerActor` | `internal/actors/scopes/derive/bollinger_signal_sampler_actor.go` | Complete (S288) |
| Actor | `BollingerSqueezeEvaluatorActor` | `internal/actors/scopes/derive/bollinger_squeeze_decision_evaluator_actor.go` | Complete (S287) |
| Supervisor | Signal processor registration | `internal/actors/scopes/derive/derive_supervisor.go` | Complete (S288) |
| Supervisor | Decision processor registration | `internal/actors/scopes/derive/derive_supervisor.go` | Complete (S288) |
| Registry | `BollingerGenerated` / `BollingerLatest` | `internal/adapters/nats/natssignal/registry.go` | Complete |
| Registry | `BollingerSqueezeEvaluated` / `BollingerSqueezeLatest` | `internal/adapters/nats/natsdecision/registry.go` | Complete |
| Writer | `WriterBollingerSignalConsumer` | `internal/adapters/nats/natssignal/registry.go` | Complete |
| Writer | `WriterBollingerSqueezeDecisionConsumer` | `internal/adapters/nats/natsdecision/registry.go` | Complete |
| Writer Pipeline | Bollinger signal pipeline entry | `cmd/writer/pipeline.go` | Complete |
| Config | `knownSignalFamilies["bollinger"]` | `internal/shared/settings/schema.go` | Complete |
| Config | `knownDecisionFamilies["bollinger_squeeze"]` | `internal/shared/settings/schema.go` | Complete |
| Config | `decisionDependsOnSignal["bollinger_squeeze"] = {"bollinger"}` | `internal/shared/settings/schema.go` | Complete |

## Message Flow Detail

### Stage 1: Candle → Signal

1. `CandleSamplerActor` emits `candleFinalizedMessage{Symbol, ClosePrice, Timestamp, CorrelationID}`
2. `SourceScopeActor.routeCandleToSignal()` fans out to all signal samplers for the symbol
3. `BollingerSignalSamplerActor.onCandleFinalized()`:
   - Calls `BollingerSampler.AddClose(closePrice, ts)`
   - Sampler accumulates 20 candles (period), then emits signal on every subsequent candle
   - Signal contains: `Type="bollinger"`, `Value=%B`, metadata `{period, k, sma, upper, lower, bandwidth}`
   - Publishes `publishSignalMessage` to signal publisher (→ NATS)
   - Sends `signalGeneratedMessage` to scope for decision fan-out

### Stage 2: Signal → Decision

1. `SourceScopeActor.routeSignalToDecision()` fans out to all decision evaluators for the symbol
2. `BollingerSqueezeEvaluatorActor.onSignalGenerated()`:
   - Calls `BollingerSqueezeEvaluator.Evaluate(signalType, signalValue, timeframe, ts, metadata)`
   - Evaluator checks `bandwidth/SMA < 0.10` threshold for squeeze detection
   - Classifies severity (high/moderate/low) based on relative bandwidth tightness
   - Classifies zone (lower/middle/upper) based on %B position
   - Publishes `publishDecisionMessage` to decision publisher (→ NATS)
   - Sends `decisionEvaluatedMessage` to scope for strategy fan-out

## Configuration

Enable Bollinger in pipeline config:

```yaml
pipeline:
  signal_families:
    - bollinger
  decision_families:
    - bollinger_squeeze
```

The dependency graph (`decisionDependsOnSignal`) ensures that enabling `bollinger_squeeze` without `bollinger` is detected as a configuration validation warning.

## What Remains Out of Scope

- No strategy resolver is yet wired to consume `bollinger_squeeze` decisions
- No risk evaluator is yet paired with a Bollinger-driven strategy
- MACD, VWAP, ATR signal families have application logic but no actor wiring (same gap Bollinger had before S288)
