# Correlation/Causation Spine Validation Across Existing Slices

**Stage:** S295
**Date:** 2026-03-21
**Status:** Validated

## Purpose

Validate the integrity of the CorrelationID/CausationID causal chain across the 3 approved vertical slices, following the path signal → decision → strategy → risk → execution.

## Methodology

Each slice was exercised with deterministic data through the full actor chain. At every stage boundary, two properties were asserted:

1. **CorrelationID immutability** — the original trace ID survives unchanged from injection to the terminal execution intent.
2. **CausationID DAG linkage** — each stage's event CausationID equals the parent stage's Metadata.ID, forming a directed acyclic graph.

Tests validated both the published event Metadata and the internal fan-out message CausationID at every hop.

## Slice 1: Mean Reversion (RSI oversold → mean_reversion_entry)

**Path:** `rsi signal → rsi_oversold decision → mean_reversion_entry strategy → position_exposure risk → paper_order execution`

**Injection:** `signalGeneratedMessage` with `CorrelationID="s295-mr-causal"`, `CausationID="signal-root-001"`

| Stage | CorrelationID | CausationID | Links To |
|-------|---------------|-------------|----------|
| Decision | s295-mr-causal | signal-root-001 | Signal (injected) |
| Strategy | s295-mr-causal | {decision.Metadata.ID} | Decision |
| Risk | s295-mr-causal | {strategy.Metadata.ID} | Strategy |
| Execution (event) | s295-mr-causal | {risk.Metadata.ID} | Risk |
| Execution (intent) | s295-mr-causal | {risk.Metadata.ID} | Risk |

**Result:** INTACT — full DAG linkage validated, all Metadata.IDs unique.

## Slice 2: Trend Following (EMA crossover → trend_following_entry)

**Path:** `ema_crossover signal → ema_crossover decision → trend_following_entry strategy → position_exposure risk → paper_order execution`

**Injection:** `signalGeneratedMessage` with `CorrelationID="s295-tf-causal"`, `CausationID="signal-root-002"`

| Stage | CorrelationID | CausationID | Links To |
|-------|---------------|-------------|----------|
| Decision | s295-tf-causal | signal-root-002 | Signal (injected) |
| Strategy | s295-tf-causal | {decision.Metadata.ID} | Decision |
| Risk | s295-tf-causal | {strategy.Metadata.ID} | Strategy |
| Execution (event) | s295-tf-causal | {risk.Metadata.ID} | Risk |
| Execution (intent) | s295-tf-causal | {risk.Metadata.ID} | Risk |

**Result:** INTACT — full DAG linkage validated, all Metadata.IDs unique.

## Slice 3: Squeeze Breakout (Bollinger → squeeze_breakout_entry)

**Path:** `bollinger signal → bollinger_squeeze decision → squeeze_breakout_entry strategy → position_exposure risk → paper_order execution`

**Injection:** 20 tight-range candles via `candleFinalizedMessage` with `CorrelationID="s295-sq-causal"`

| Stage | CorrelationID | CausationID | Links To |
|-------|---------------|-------------|----------|
| Signal | s295-sq-causal | "" (root) | — |
| Decision | s295-sq-causal | {signal.Metadata.ID} | Signal |
| Strategy | s295-sq-causal | {decision.Metadata.ID} | Decision |
| Risk | s295-sq-causal | {strategy.Metadata.ID} | Strategy |
| Execution (event) | s295-sq-causal | {risk.Metadata.ID} | Risk |
| Execution (intent) | s295-sq-causal | {risk.Metadata.ID} | Risk |

**Additional fan-out validation:**

| Fan-out Hop | CausationID | Links To |
|-------------|-------------|----------|
| signal → decision | {signal.Metadata.ID} | Signal event |
| decision → strategy | {decision.Metadata.ID} | Decision event |
| strategy → risk | {strategy.Metadata.ID} | Strategy event |

**Result:** INTACT — full 5-stage DAG linkage validated, all Metadata.IDs unique, fan-out hops correct.

## Propagation Patterns Verified

### CorrelationID (thread identifier)
- Minted at evidence boundary (candle finalization) or injected in test.
- Forwarded unchanged through every actor, message, and event.
- Persisted on both event Metadata and ExecutionIntent domain struct.
- Never invented mid-chain.

### CausationID (parent link)
- Signal events: CausationID is empty (root of causal chain, by design).
- Decision events: CausationID = signal event's Metadata.ID.
- Strategy events: CausationID = decision event's Metadata.ID.
- Risk events: CausationID = strategy event's Metadata.ID.
- Execution events: CausationID = risk event's Metadata.ID.
- Fan-out messages: CausationID = parent event's Metadata.ID (set at fan-out point).

### Dual Tracking at Execution
- Event-level: `Metadata.CorrelationID` and `Metadata.CausationID` on `PaperOrderSubmittedEvent`.
- Domain-level: `ExecutionIntent.CorrelationID` and `ExecutionIntent.CausationID`.
- Both carry identical values, ensuring traceability at transport and domain layers.

## Evidence

Test file: `internal/actors/scopes/derive/causal_chain_validation_test.go`

| Test | Slice | Stages | Status |
|------|-------|--------|--------|
| `TestCausalChain_MeanReversion_DAGLinkage` | Mean Reversion | 4 (decision→execution) | PASS |
| `TestCausalChain_TrendFollowing_DAGLinkage` | Trend Following | 4 (decision→execution) | PASS |
| `TestCausalChain_SqueezeBreakout_DAGLinkage` | Squeeze Breakout | 5 (signal→execution) | PASS |
