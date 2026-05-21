# Full Closed-Loop Scenario Validation

## Purpose

This document defines the closed-loop validation strategy for the Market Foundry domain pipeline. A "closed loop" means a complete traversal from signal intelligence through decision, strategy, risk assessment, and paper execution — with every intermediate stage producing observable, auditable domain events.

## Distinction from Prior Stages

| Stage | Question Answered |
|-------|-------------------|
| S252 (Behavioral Scenarios) | Does severity influence downstream behavior? |
| S266 (Paper Order Generation) | Does a paper order come out of the chain? |
| **S268 (Closed-Loop Validation)** | **Is the full loop coherent, observable, and operationally meaningful at every stage?** |

S268 treats the pipeline as a single operational unit. Each scenario validates not just the final output, but the complete causal narrative: what the signal said, how the decision interpreted it, what the strategy decided, how risk constrained it, and what execution produced.

## Scenario Design Principles

1. **Small and representative** — each scenario covers one meaningful behavioral path, not an exhaustive matrix.
2. **Full observability** — every intermediate domain event (decision, strategy, risk, execution) is captured and asserted.
3. **Operationally meaningful** — scenarios map to real trading situations (strong signal, weak signal, no signal, counter-trend vs pro-trend).
4. **Paper mode only** — no real venue, no real money. All fills are simulated.
5. **Auditable** — CorrelationID and CausationID survive every stage boundary, enabling full trace reconstruction.

## Validated Scenarios

### Closed Loop A: Mean Reversion Full Observability

**Signal**: RSI 10 (extreme oversold)
**Path**: rsi → rsi_oversold (triggered, severity=high) → mean_reversion_entry (long, target×1.5) → [position_exposure + drawdown_limit] (both approved) → paper buy order (filled)

**Observable outputs at each stage**:
- Decision: outcome=triggered, severity=high, rationale includes distance metric
- Strategy: direction=long, confidence=0.8333, target_offset=0.03 (severity-scaled)
- Risk/exposure: disposition=approved, max_position=0.0192, strategy-type factor=0.90
- Risk/drawdown: disposition=approved, stop_distance=0.0212
- Execution: side=buy, status=filled, 1 simulated fill, correlation preserved

### Closed Loop B: Trend Following Full Observability

**Signal**: EMA crossover bullish
**Path**: ema_crossover → ema_crossover (triggered, severity=moderate) → trend_following_entry (long) → [position_exposure + drawdown_limit] → paper buy order (filled)

**Observable outputs at each stage**:
- Decision: outcome=triggered, severity=moderate
- Strategy: direction=long, trailing_stop_pct=0.03, take_profit_pct=0.05
- Risk/exposure: disposition=approved, strategy-type factor=0.95 (pro-trend)
- Risk/drawdown: disposition=approved, stop_distance scaled for trend following
- Execution: side=buy, status=filled, strategy type and severity preserved

### Closed Loop C: Severity Behavioral Contrast

**Signals**: RSI 10 (high severity) vs RSI 25 (low severity)
**Observation**: Every stage produces observably different outputs

| Stage | High Severity | Low Severity |
|-------|--------------|-------------|
| Decision severity | high | low |
| Strategy confidence | 0.8333 | 0.4666 |
| Strategy target_offset | 0.03 | 0.01 |
| Risk max_position | 0.0192 | 0.0075 |
| Execution quantity | 0.0192 | 0.0075 |

The 2.56× quantity ratio between high and low severity demonstrates that domain intelligence actively shapes operational behavior at every stage.

### Closed Loop D: No-Signal Suppression

**Signal**: RSI 75 (not oversold)
**Path**: rsi → rsi_oversold (not_triggered, severity=none) → mean_reversion_entry (flat) → position_exposure (approved, no position) → paper order (side=none, qty=0)

**Observable outputs at each stage**:
- Decision: outcome=not_triggered, severity=none
- Strategy: direction=flat (no trade intent)
- Risk: disposition=approved (trivially — no position needed)
- Execution: side=none, quantity=0, status=submitted (no fill), no fills array

This scenario proves the system safely produces no operational output when conditions are not met, while still generating observable events at every stage.

### Closed Loop E: Cross-Chain Behavioral Distinction

**Chains**: Mean reversion (RSI 10) vs Trend following (EMA bullish)
**Observation**: Semantically distinct intermediate outputs at every stage

| Stage | Mean Reversion | Trend Following |
|-------|---------------|-----------------|
| Decision type | rsi_oversold | ema_crossover |
| Strategy type | mean_reversion_entry | trend_following_entry |
| Strategy parameters | target_offset, stop_offset | trailing_stop_pct, take_profit_pct |
| Risk confidence factor | 0.90 (counter-trend) | 0.95 (pro-trend) |
| Execution strategy_type | mean_reversion_entry | trend_following_entry |

## Causal Trace Preservation

All scenarios verify that:
- **CorrelationID** (signal origin) survives all 4 stage boundaries
- **CausationID** (upstream event) is set at each stage
- **Domain Validation** (`Validate()`) passes for every intermediate and final output
- **Metadata** carries strategy_type, decision_severity, confidence_factors through the pipeline

## Guard Rails Validated

- Paper mode only — all orders are `type: "paper_order"`, all fills are `Simulated: true`
- Risk-gated quantities — execution cannot override risk constraints
- Disposition-gated sides — rejected/flat → no operational output
- Domain validation — all intents pass `Validate()` before publishing
- Per-symbol isolation — partition keys enforce `source.symbol.timeframe`

## Test Location

All closed-loop tests are in:
```
internal/actors/scopes/derive/closed_loop_end_to_end_test.go
```

5 test functions, each validating a complete scenario with intermediate observability:
- `TestClosedLoop_MeanReversion_FullObservability`
- `TestClosedLoop_TrendFollowing_FullObservability`
- `TestClosedLoop_SeverityContrast_EveryStage`
- `TestClosedLoop_NoSignal_Suppression_FullChain`
- `TestClosedLoop_CrossChain_BehavioralDistinction`
