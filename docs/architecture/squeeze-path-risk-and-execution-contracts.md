# Squeeze Path: Risk and Execution Contracts

Status: **Closed** (S290)
Scope: Contracts governing how `squeeze_breakout_entry` strategies are evaluated by risk and transformed into execution intents.

---

## Contract Summary

The squeeze breakout path reuses the canonical strategy-to-risk-to-execution pipeline without introducing new message types, new evaluator actors, or new domain structures. All integration is achieved through:

1. Explicit scaling factors in `risk_scaling.go`
2. Corrected dependency graph in `settings/schema.go`
3. No changes to actor wiring, NATS adapters, or domain models

---

## Risk Evaluation Contracts

### Position Exposure Evaluator

The `PositionExposureEvaluator` accepts `squeeze_breakout_entry` as a `strategyType` input. It applies the following scaling factors:

| Factor | Value | Rationale |
|--------|-------|-----------|
| Confidence multiplier | 0.93 | Momentum/volatility strategy — moderate false-breakout risk. Between counter-trend (0.90) and pro-trend (0.95). |
| Severity position limit | Uses shared severity map | high=1.15x, moderate=1.0x, low=0.80x — same as all strategies. |

**Behavioral semantics:**
- A high-severity squeeze with 0.90 confidence yields risk confidence 0.8370 and effective position limit 0.0230 (2.3%).
- A low-severity squeeze with 0.85 confidence yields risk confidence 0.7905 and effective position limit 0.0160 (1.6%).
- Flat strategies (not_triggered/insufficient) are always approved with confidence 1.0 and no constraints.
- Zero or negative confidence produces a rejected disposition.

### Drawdown Limit Evaluator

The `DrawdownLimitEvaluator` accepts `squeeze_breakout_entry` as a `strategyType` input. It applies the following scaling factors:

| Factor | Value | Rationale |
|--------|-------|-----------|
| Confidence multiplier | 0.90 | Between counter-trend (0.85) and pro-trend (0.92). |
| Stop distance multiplier | 1.05 | Slightly wider than neutral — breakouts need room to develop but less than trend following (1.15). |
| Severity drawdown tolerance | Uses shared severity map | high=1.15x, moderate=1.0x, low=0.80x — same as all strategies. |

**Behavioral semantics:**
- Effective stop base = 0.03 x 1.05 = 0.0315 (3.15%).
- High-severity squeeze allows up to 0.0575 (5.75%) drawdown tolerance.
- Stop distance is confidence-scaled: `effectiveStopBase x confidence`, floored at 0.0050.

---

## Execution Contract

### Paper Order Evaluator

The `PaperOrderEvaluator` is strategy-type-agnostic. It receives risk assessments and produces execution intents based on:

| Input | Squeeze Breakout Behavior |
|-------|--------------------------|
| `riskDisposition = "approved"` + `strategyDirection = "long"` | Side = buy, quantity = maxPositionPct |
| `riskDisposition = "modified"` + `strategyDirection = "long"` | Side = buy, quantity = cappedMaxPositionPct |
| `riskDisposition = "rejected"` | Side = none, quantity = 0 |
| `strategyDirection = "flat"` | Side = none, quantity = 0 |

**Causal traceability:**
- `ExecutionIntent.Risk.StrategyType` = `"squeeze_breakout_entry"`
- `ExecutionIntent.Risk.DecisionSeverity` = originating decision severity
- `ExecutionIntent.Parameters["strategy_type"]` = `"squeeze_breakout_entry"`

---

## Dependency Graph

Updated dependency rules in `settings/schema.go`:

```
riskDependsOnStrategy:
  position_exposure → [mean_reversion_entry, trend_following_entry, squeeze_breakout_entry]
  drawdown_limit    → [mean_reversion_entry, trend_following_entry, squeeze_breakout_entry]

executionDependsOnRisk:
  paper_order        → [position_exposure, drawdown_limit]
  venue_market_order → [position_exposure, drawdown_limit]
```

**Validation semantics changed to "at least one":** A risk family requires at least one compatible strategy to be enabled, not all of them. This allows configurations like `[squeeze_breakout_entry] + [position_exposure]` without requiring `mean_reversion_entry` or `trend_following_entry`.

---

## Ownership Boundaries

| Component | Owner | Change in S290 |
|-----------|-------|----------------|
| `risk_scaling.go` | Risk domain | Added squeeze_breakout_entry factors |
| `schema.go` | Settings | Added drawdown_limit to known families; corrected dependency graph |
| `position_exposure_evaluator.go` | Risk domain | No change — already strategy-type-generic |
| `drawdown_limit_evaluator.go` | Risk domain | No change — already strategy-type-generic |
| `paper_order_evaluator.go` | Execution domain | No change — already strategy-type-agnostic |
| Actor wiring | Derive supervisor | No change — already fans all strategies to all risk evaluators |
| NATS adapters | Adapter layer | No change — publisher subjects are type-parameterized |

---

## What Remains Out of Scope

- Venue-specific order routing (OMS/router/portfolio)
- Real execution venue integration
- Risk evaluator refactoring (not needed — the generic design accommodates new strategies)
- Codegen integration for squeeze_breakout_entry in risk/execution layers
- Multi-strategy aggregation (risk evaluates each strategy independently)
