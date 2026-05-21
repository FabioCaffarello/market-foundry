# Risk Breadth Expansion

## Purpose

This document records the breadth expansion of the `risk` domain from one evaluator/type (`position_exposure`) to two (`position_exposure` + `drawdown_limit`), fulfilling the charter's breadth requirement for risk without inflating the domain into a general-purpose risk management subsystem.

## Context

- The S240 charter required breadth across `decision`, `strategy`, and `risk`.
- S241 achieved decision breadth: `rsi_oversold` + `ema_crossover`.
- S242 achieved strategy breadth: `mean_reversion_entry` + `trend_following_entry`.
- S243 completes the trio by adding the second risk evaluator/type.

## Design Rationale

### Why `drawdown_limit`?

The two risk types are semantically complementary:

| Aspect | `position_exposure` | `drawdown_limit` |
|--------|-------------------|-----------------|
| **Question** | "How much capital to allocate?" | "How much loss is acceptable?" |
| **Focus** | Position sizing vs. portfolio limits | Stop-loss distance vs. drawdown limits |
| **Constraints** | `MaxPositionSize`, `MaxExposure` | `StopDistance`, `MaxExposure` |
| **Confidence mapping** | Linear: confidence × max_position_pct | Inverse: lower confidence → tighter stop |
| **Discount factor** | 0.95 | 0.90 |

Together they cover the two fundamental risk questions for any trade:
1. **How big?** (position_exposure)
2. **How far can it fall?** (drawdown_limit)

### Why NOT something else?

- **Volatility-based risk**: would require additional signal data (ATR, historical vol) — violates breadth-only scope.
- **Correlation risk**: would require multi-symbol state — violates single-symbol evaluator pattern.
- **Liquidity risk**: would require order book data — not available in current pipeline.

`drawdown_limit` uses the same inputs as `position_exposure` (strategy primitives) and requires no new data sources, making it the minimal-cost breadth addition.

## Architecture

### Evaluation Logic

The `DrawdownLimitEvaluator` receives the same `strategyResolvedMessage` as `PositionExposureEvaluator`:

1. **Flat strategies**: Always `approved` — no drawdown risk for zero-position intent.
2. **Long/Short strategies**:
   - Stop distance = `stop_distance_pct × confidence` (inversely scaled)
   - Floor: minimum 0.50% stop distance to avoid unrealistic stops
   - Cap: maximum `stop_distance_pct` (3% default)
   - If stop ≤ `max_drawdown_pct` (5% default): `approved`
   - If stop > `max_drawdown_pct`: `modified` (capped to max)

### Data Flow

```
strategyResolvedMessage (primitives)
  ├── PositionExposureEvaluatorActor → publishRiskMessage (type=position_exposure)
  └── DrawdownLimitEvaluatorActor    → publishRiskMessage (type=drawdown_limit)
                                          ↓
                                  RISK_EVENTS stream (shared)
                                          ↓
                            ┌─────────────┴─────────────┐
                      Writer pipeline              Store pipeline
                   (risk_assessments table)    (KV: RISK_DRAWDOWN_LIMIT_LATEST)
```

Both risk types share:
- The same NATS stream (`RISK_EVENTS`)
- The same ClickHouse table (`risk_assessments`)
- The same domain struct (`RiskAssessment`)
- The same actor config (`RiskEvaluatorConfig`)

They differ by:
- NATS subjects (`risk.events.drawdown_limit.assessed.>`)
- KV buckets (`RISK_DRAWDOWN_LIMIT_LATEST`)
- Writer/store consumer durables
- Type field value in domain struct

### Configuration

Activation follows the same opt-in pattern:

```yaml
pipeline:
  risk_families:
    - position_exposure
    - drawdown_limit
```

## Non-Goals

- No aggregation across risk types (no "combined risk score")
- No cross-symbol risk correlation
- No dynamic parameter tuning at runtime
- No rejection disposition — current evaluator only approves or modifies
- No additional signal/indicator inputs beyond strategy primitives

## Trade-offs

| Decision | Benefit | Cost |
|----------|---------|------|
| Fixed parameters | Simplicity, predictability | No runtime adaptation |
| Inverse confidence→stop | Natural: uncertain trades get tight stops | Doesn't consider market volatility |
| Stop distance floor (0.5%) | Prevents unrealistic micro-stops | May be too wide for some instruments |
| Shared RiskAssessment struct | No domain changes needed | All risk types must fit same shape |
| No rejection | Simpler disposition logic | Cannot outright block a trade |
