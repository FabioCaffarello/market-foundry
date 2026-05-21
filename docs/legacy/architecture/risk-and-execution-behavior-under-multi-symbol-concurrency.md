# Risk and Execution Behavior Under Multi-Symbol Concurrency

Stage: S304
Status: Validated
Date: 2026-03-21

## Purpose

Document and validate that risk evaluation (position_exposure, drawdown_limit) and execution paper (paper_order, paper_fill) maintain correct, isolated, and explainable behavior when multiple symbols traverse the pipeline simultaneously.

## Architecture Summary

### Risk Evaluation Layer

Each symbol is evaluated by independent evaluator instances scoped to `(source, symbol, timeframe)`. No shared state exists between evaluator instances:

```
PositionExposureEvaluator("binancef", "btcusdt", 60)  → isolated assessment
PositionExposureEvaluator("binancef", "ethusdt", 60)  → isolated assessment
DrawdownLimitEvaluator("binancef", "solusdt", 60)     → isolated assessment
```

Strategy-type-aware scaling factors are applied per symbol's strategy type:
- `mean_reversion_entry`: confidence ×0.90, stop ×0.85 (counter-trend → conservative)
- `trend_following_entry`: confidence ×0.95, stop ×1.15 (pro-trend → tolerant)
- `squeeze_breakout_entry`: confidence ×0.93, stop ×1.05 (momentum → moderate)

Severity-based position/drawdown adjustment is per symbol's decision severity:
- `high`: limit ×1.15 (strong signal → more room)
- `moderate`: limit ×1.00 (neutral)
- `low`: limit ×0.80 (weak signal → tighter)

### Execution Paper Layer

Paper order evaluation maps risk outcomes to execution intents per symbol:

| Risk Disposition | Strategy Direction | Execution Side | Quantity |
|---|---|---|---|
| approved | long | buy | maxPositionPct |
| approved | short | sell | maxPositionPct |
| modified | long | buy | risk-adjusted maxPositionPct |
| modified | short | sell | risk-adjusted maxPositionPct |
| rejected | any | none | 0 |
| any | flat | none | 0 |

Paper fill simulation transitions actionable intents through: submitted → filled (instant, simulated). No-action intents (side=none) remain unchanged.

### Isolation Mechanisms

1. **Evaluator scoping**: each evaluator instance is constructed with a specific symbol — no cross-symbol state.
2. **Partition keys**: `{source}.{symbol}.{timeframe}` ensures KV and deduplication isolation.
3. **Actor scoping**: each actor processes messages for a single (source, symbol, timeframe) tuple.
4. **Composite read model**: all queries filter by `WHERE symbol = ?` (S301).

## Validated Behaviors

### Risk Layer (RE-1 through RE-6)

- **RE-1**: Same strategy type with different severities across 3 symbols produces distinct effective position limits (high > moderate > low). No cross-symbol leakage in rationale or metadata.
- **RE-2**: Different strategy types produce strategy-specific confidence scaling. Risk confidence = `baseConf × strategyTypeFactor`. Metadata carries correct strategy_type per symbol.
- **RE-3**: Mixed dispositions (approved, rejected via zero/negative confidence) coexist across symbols. Rejected assessments carry per-symbol rationale. Approved assessments carry constraints.
- **RE-4**: Drawdown evaluator produces strategy-type-specific stop distances. mean_reversion (×0.85) yields tighter stops than squeeze_breakout (×1.05). Each symbol's rationale references its own strategy type.
- **RE-5**: Position exposure and drawdown limit evaluators agree on symbol ownership, strategy type, and decision severity for the same symbol. Both pass domain validation independently.
- **RE-6**: Flat direction across all symbols produces approved disposition with confidence 1.0 and no position constraints. No cross-symbol leakage.

### Execution Layer (EX-1 through EX-6)

- **EX-1**: Three symbols with approved/rejected/approved risk produce buy/none/sell sides correctly. Quantity reflects risk-constrained maxPositionPct per symbol.
- **EX-2**: Full paper lifecycle (evaluate → simulate fill) per symbol transitions submitted → filled. Fill records are simulated. Symbol survives lifecycle transitions.
- **EX-3**: Rejected risk blocks execution across all symbols independently. No-action intents have side=none, quantity=0, and no fill records.
- **EX-4**: Modified disposition preserves risk-adjusted quantity per symbol. Fill simulation applies to modified intents and fills with capped quantity.
- **EX-5**: Strategy type and decision severity preserved through risk→execution boundary in RiskInput and Parameters per symbol.
- **EX-6**: Paper venue adapter produces unique venue order IDs per symbol. Receipt carries correct symbol and filled status.

### Composite Pipeline (RX-1 through RX-5)

- **RX-1**: Three symbols with approved/rejected/modified risk produce correct chain completeness (5/4/5 stages), correct attribution disposition, and correct execution presence/absence.
- **RX-2**: Risk attribution carries unique rationale, constraints, and strategy context per symbol.
- **RX-3**: Execution status coherence: approved→filled (5 stages), rejected→blocked (4 stages, missing execution), modified→filled (5 stages, capped constraints).
- **RX-4**: Cross-surface alignment: funnel execution count = approved+modified disposition count. Disposition total = risk count. Monotonic decrease across stages.
- **RX-5**: position_exposure and drawdown_limit risk types coexist in the composite pipeline across symbols without interference.

## Constraints and Parameters

All evaluators use these defaults (configurable at construction time):

| Parameter | Default | Risk Type |
|---|---|---|
| maxPositionPct | 0.02 (2%) | position_exposure |
| maxPortfolioExposurePct | 0.10 (10%) | position_exposure |
| maxDrawdownPct | 0.05 (5%) | drawdown_limit |
| stopDistancePct | 0.03 (3%) | drawdown_limit |

Stop distance floor: 0.5% (prevents unrealistic stops at very low confidence).

## Causal Metadata Flow

```
Signal (root, no causation)
  → Decision (causation = signal.event_id)
    → Strategy (causation = decision.event_id)
      → Risk Assessment (causation = strategy.event_id)
        → Execution Intent (causation = risk.event_id)
```

All stages share a single `correlation_id`. The composite read model enforces `symbol` in every query, preventing cross-symbol result contamination.

## Operational Implications

1. **No shared state between symbols**: each evaluator is stateless and symbol-scoped. Adding a new symbol requires no changes to risk or execution code.
2. **Paper mode fills are instant and simulated**: no market impact, no venue latency. Fill price = "0", fee = "0", simulated = true.
3. **Attribution is fully deterministic**: given the same inputs, the same symbol always produces the same risk assessment and execution intent.
4. **Composite read model reflects write-side behavior**: the analytical pipeline faithfully mirrors the live domain events.
