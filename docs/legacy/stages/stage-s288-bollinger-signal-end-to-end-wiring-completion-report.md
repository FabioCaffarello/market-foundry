# Stage S288: Bollinger Signal End-to-End Wiring Completion Report

**Date**: 2026-03-21
**Predecessor**: S287 (Bollinger Squeeze Decision Family)
**Scope**: End-to-end wiring closure for Bollinger signal family in derive actor system

## Executive Summary

S288 closes the wiring gap left by S287 by creating the `BollingerSignalSamplerActor`, registering both the Bollinger signal and `bollinger_squeeze` decision processors in the `DeriveSupervisor`, and proving the full chain with integration tests. The Bollinger family is no longer partial — it flows from candle evidence through signal generation to decision evaluation within the derive actor system.

## Initial State Found

Before S288, the Bollinger family had:

| Component | Status |
|-----------|--------|
| `BollingerSampler` (application logic) | Complete (S286) |
| `BollingerSqueezeEvaluator` (application logic) | Complete (S287) |
| `BollingerSqueezeEvaluatorActor` (actor wrapper for decision) | Complete (S287) |
| NATS registry entries (signal + decision) | Complete |
| Writer pipeline consumer specs | Complete |
| Settings/config validation | Complete |
| **`BollingerSignalSamplerActor` (actor wrapper for signal)** | **MISSING** |
| **Signal processor registration in `DeriveSupervisor`** | **MISSING** |
| **Decision processor registration in `DeriveSupervisor`** | **MISSING** |

The chain was broken at the actor layer: application logic existed but was not wired into the supervised actor tree.

## Wiring Completed

### 1. Created `BollingerSignalSamplerActor`

**File**: `internal/actors/scopes/derive/bollinger_signal_sampler_actor.go`

- Follows canonical pattern established by `RSISignalSamplerActor` and `EMACrossoverSignalSamplerActor`
- Receives `candleFinalizedMessage` from candle sampler via scope fan-out
- Delegates computation to `BollingerSampler.AddClose()`
- Publishes `publishSignalMessage` to shared signal publisher (→ NATS)
- Sends `signalGeneratedMessage` to scope for decision evaluator fan-out
- Preserves correlation ID chain

### 2. Registered Bollinger Signal Processor in DeriveSupervisor

**File**: `internal/actors/scopes/derive/derive_supervisor.go`

- Added `{Family: "bollinger", ActorPrefix: "signal-bollinger"}` to `signalProcessors` array
- Wires `NewBollingerSignalSamplerActor` with standard `SignalSamplerConfig`
- Gated by `pipeline.signal_families` configuration (opt-in)

### 3. Registered Bollinger Squeeze Decision Processor in DeriveSupervisor

**File**: `internal/actors/scopes/derive/derive_supervisor.go`

- Added `{Family: "bollinger_squeeze", ActorPrefix: "decision-bollinger-squeeze"}` to `decisionProcessors` array
- Wires `NewBollingerSqueezeEvaluatorActor` with standard `DecisionEvaluatorConfig`
- Gated by `pipeline.decision_families` configuration (opt-in)

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/bollinger_signal_sampler_actor.go` | **Created** — actor wrapper for BollingerSampler |
| `internal/actors/scopes/derive/derive_supervisor.go` | **Modified** — added bollinger signal + decision processor registrations |
| `internal/actors/scopes/derive/bollinger_chain_integration_test.go` | **Created** — 3 integration tests proving the chain |
| `docs/architecture/bollinger-signal-end-to-end-wiring.md` | **Created** — full wiring documentation |
| `docs/architecture/bollinger-signal-derive-path-and-ownership.md` | **Created** — ownership and path documentation |
| `docs/stages/stage-s288-bollinger-signal-end-to-end-wiring-completion-report.md` | **Created** — this report |

## Tests and Evidence

### Integration Tests Added

| Test | Purpose | Result |
|------|---------|--------|
| `TestActorChain_BollingerSignalSampler_ProducesSignal` | Proves 20-candle warmup → signal generation with correct metadata | PASS |
| `TestActorChain_BollingerSignal_To_BollingerSqueezeDecision` | Proves tight bands → squeeze triggered, full signal→decision chain | PASS |
| `TestActorChain_BollingerSignal_WideBands_NotTriggered` | Proves wide bandwidth → not_triggered decision | PASS |

### Regression Check

All existing `TestActorChain_*` tests continue to pass (RSI, EMA Crossover chains unaffected).

### Key Observations from Tests

- BollingerSampler correctly emits after 20-candle warmup period
- Signal metadata includes all expected fields: `period`, `k`, `sma`, `upper`, `lower`, `bandwidth`
- Squeeze detection correctly triggers on tight bands (bandwidth/SMA < 0.10)
- Correlation ID preserved through signal → decision chain
- Causation ID correctly set from signal metadata ID

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| `BollingerSignalSamplerActor` is wired into derive actor system | PASS |
| Bollinger family no longer exists in partial state | PASS |
| End-to-end signal flow proven (candle → signal → decision) | PASS |
| No new breadth opened | PASS |
| No scope inflation beyond wiring | PASS |

## What Remains Out of Scope

1. **No strategy resolver** consumes `bollinger_squeeze` decisions yet — this requires a new strategy family (e.g., squeeze breakout entry) which is new breadth
2. **MACD, VWAP, ATR families** have the same gap Bollinger had (application logic exists, actor wiring missing) — candidates for S289+
3. **Store binary** query serving for Bollinger signal/decision — requires store binary scope (separate concern)

## Recommended Preparation for S289

Options for the next stage, ordered by wiring-closure priority:

1. **MACD/VWAP/ATR signal actor wiring** — apply the same S288 pattern to close remaining signal family gaps (pure wiring, no new domain logic needed)
2. **Bollinger squeeze strategy family** — create a strategy resolver that consumes `bollinger_squeeze` decisions (new breadth, extends the chain deeper)
3. **Signal Evolution Wave gate** — if all signal families are wired, assess readiness for the next wave transition
