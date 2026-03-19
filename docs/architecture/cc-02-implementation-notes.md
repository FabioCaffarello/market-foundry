# CC-02 Implementation Notes — EMA Crossover Signal Family

## Overview

CC-02 adds the `ema_crossover` signal family to the market-foundry pipeline as a minimal extensibility proof. The implementation follows Playbook 1 (new signal family) exactly, reusing every existing architectural component without modifications.

## Implementation Decisions

### 1. Domain Model Unchanged (EX-01 confirmed)

The existing `signal.Signal` struct is fully family-agnostic. EMA crossover uses:
- `Value`: categorical string (`"bullish"`, `"bearish"`, `"neutral"`) instead of RSI's numeric decimal — confirms the `string` type is flexible enough.
- `Metadata`: multiple parameters (`fast_period`, `slow_period`, `fast_ema`, `slow_ema`, `spread`) — confirms the `map[string]string` design handles multi-parameter families.

No domain types were added or modified.

### 2. EMA Computation — Warm-up Strategy

The sampler seeds both EMAs using simple moving averages over their respective windows once `slowPeriod` (21) candles are collected. After warm-up, standard EMA smoothing is applied:

```
EMA_new = price × k + EMA_prev × (1 − k)
k = 2 / (period + 1)
```

The fast EMA (period 9) reacts faster to price changes than the slow EMA (period 21). The spread (`fast_ema − slow_ema`) determines direction.

### 3. Crossover Direction Tolerance

A tolerance of `1e-8` is applied to the spread to prevent noise-level oscillations between bullish/bearish when EMAs are near-equal. This produces `"neutral"` only during exact convergence — in practice, after warm-up, signals are almost always `"bullish"` or `"bearish"`.

### 4. Reused Components (Zero New Infrastructure)

| Component | Reused? | Notes |
|-----------|---------|-------|
| `signal.Signal` domain struct | Yes | No changes |
| `signal.SignalGeneratedEvent` | Yes | No changes |
| `SignalPublisherActor` | Yes | Shared across all signal families |
| `SignalPublisher` (NATS adapter) | Yes | Added `case "ema_crossover"` routing |
| `SignalProjectionActor` | Yes | Family-agnostic, bucket injected |
| `SignalConsumerActor` | Yes | Family-agnostic, consumer spec injected |
| `SIGNAL_EVENTS` stream | Yes | Wildcard `signal.events.>` covers new family |
| `SignalKVStore` | Yes | Bucket name injected at construction |
| Signal HTTP route | Yes | `/signal/:type/latest` is type-parameterized |
| `SignalGateway` | Yes | `LatestSpecByType()` dispatches by type |
| `SignalSamplerConfig` | Yes | Shared config struct for all signal sampler actors |

### 5. Actor Wiring — Copy-Paste Pattern (CF-08 evidence)

The `EMACrossoverSignalSamplerActor` follows the exact same structure as `RSISignalSamplerActor`:
- Receives `candleFinalizedMessage`
- Delegates to pure sampler
- Sends `publishSignalMessage` to publisher
- Sends `signalGeneratedMessage` to scope for decision fan-out

The structural duplication between the two actor files is ~95% identical. This is expected CF-08 boilerplate — documented as a friction trigger, not resolved in this stage.

### 6. Correlation ID Propagation — Copy-Paste (CF-03 evidence)

The correlation ID flows through the actor identically to RSI:
```go
meta := events.NewMetadata().WithCorrelationID(msg.CorrelationID)
```

This is the same copy-paste pattern observed in every actor. CF-03 trigger confirmed.

## Simplifications Adopted

1. **Fixed periods (9/21)**: The fast and slow EMA periods are hardcoded. Configurable periods would require a per-family parameter mechanism that does not yet exist. This is sufficient for the extensibility proof.

2. **No downstream families**: No decision, strategy, risk, or execution families consume `ema_crossover` signals. The signal flows to the scope PID for decision fan-out, but no decision family is registered to consume it. This is by design — CC-02 validates signal-layer extensibility only.

3. **No multi-timeframe correlation**: Each sampler operates independently per timeframe. Cross-timeframe analysis is out of scope.

## Files Changed

### New Files (3)
| File | Lines | Purpose |
|------|-------|---------|
| `internal/application/signal/ema_crossover_sampler.go` | ~110 | Pure EMA computation logic |
| `internal/application/signal/ema_crossover_sampler_test.go` | ~130 | 6 unit tests |
| `internal/actors/scopes/derive/ema_crossover_signal_sampler_actor.go` | ~80 | Actor wrapper |

### Modified Files (7)
| File | Change | Lines Added |
|------|--------|-------------|
| `internal/shared/settings/schema.go` | Register `ema_crossover` in `knownSignalFamilies` and `signalDependsOnEvidence` | +2 |
| `internal/adapters/nats/signal_registry.go` | Add `EMACrossoverGenerated`, `EMACrossoverLatest`, `StoreEMACrossoverSignalConsumer()` | +30 |
| `internal/adapters/nats/signal_publisher.go` | Add `case "ema_crossover"` in `specForType()` | +3 |
| `internal/adapters/nats/signal_kv_store.go` | Add `SignalEMACrossoverLatestBucket` constant | +1 |
| `internal/actors/scopes/derive/derive_supervisor.go` | Register `ema_crossover` processor in signal processor slice | +10 |
| `internal/actors/scopes/store/store_supervisor.go` | Register `ema_crossover` pipeline in `declarePipelines()` | +15 |
| `internal/shared/settings/settings_test.go` | Update expected signal family count from 1 to 2 | +1 (modified) |
