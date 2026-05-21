# Strategy/Risk to Execution Contract Alignment

Stage: S265
Charter: PAPER-EXECUTION-WAVE-1
Date: 2026-03-21
Status: Aligned

---

## 1. Purpose

This document formalizes the contract boundary between the strategy/risk domains and the execution domain. It defines which fields cross the boundary, when execution intent is produced, and where domain responsibility ends and operational responsibility begins.

## 2. Boundary Before S265

Prior to S265, the risk → execution boundary had three alignment gaps:

| Gap | Impact |
|-----|--------|
| `StrategyType` dropped at boundary | Execution intents lost the originating strategy family identity; no way to distinguish `mean_reversion_entry` from `trend_following_entry` at execution layer |
| `DecisionSeverity` dropped at boundary | Behavioral context from the originating decision was lost; execution could not observe or trace severity-based scaling |
| Drawdown `StopDistance` mapped to `MaxPositionPct` | Semantic mismatch: stop distance (stop-loss ceiling) was used as position size, producing intents with nonsensical quantities for drawdown-originated assessments |

## 3. Boundary After S265

### 3.1 Fields That Cross the Boundary

The `riskAssessedMessage` is the canonical contract between risk evaluator actors and execution evaluator actors. After S265:

| Field | Source | Semantic | Used by execution |
|-------|--------|----------|-------------------|
| `RiskType` | `assessment.Type` | Risk family name (e.g., `position_exposure`, `drawdown_limit`) | Stored in `RiskInput.Type` and `Parameters` |
| `RiskDisposition` | `assessment.Disposition` | `approved`, `modified`, or `rejected` | Determines side (buy/sell/none) |
| `RiskConfidence` | `assessment.Confidence` | Risk-adjusted confidence (decimal string) | Stored in `RiskInput.Confidence` |
| `MaxPositionPct` | `assessment.Constraints.MaxPositionSize` (position_exposure) or `assessment.Constraints.MaxExposure` (drawdown_limit) | Position size constraint as decimal percentage | Used as `Quantity` for actionable intents |
| `StrategyDirection` | `assessment.Strategies[0].Direction` | `long`, `short`, or `flat` | Determines side (buy/sell/none) |
| `StrategyConfidence` | `assessment.Strategies[0].Confidence` | Strategy confidence (decimal string) | Stored in `Parameters` for observability |
| `StrategyType` | `assessment.Strategies[0].Type` | Strategy family name (e.g., `mean_reversion_entry`) | **S265: now stored in `RiskInput.StrategyType` and `Parameters`** |
| `DecisionSeverity` | `assessment.Strategies[0].DecisionSeverity` | `high`, `moderate`, `low`, or empty | **S265: now stored in `RiskInput.DecisionSeverity` and `Parameters`** |
| `Timeframe` | `assessment.Timeframe` | Candle timeframe in seconds | Stored in `RiskInput.Timeframe` and intent `Timeframe` |
| `Timestamp` | `assessment.Timestamp` | Assessment timestamp | Used as intent `Timestamp`; checked by `StalenessGuard` |
| `CorrelationID` | event metadata | Causal trace root | Preserved in intent and event metadata |
| `CausationID` | event metadata ID | Immediate causal parent | Preserved in intent and event metadata |

### 3.2 When Execution Intent is Produced

Execution intent is produced for **every** risk assessment that reaches the execution evaluator actor, including:

| Disposition + Direction | Side | Quantity | Rationale |
|------------------------|------|----------|-----------|
| `rejected` + any | `none` | `0` | Risk rejected — no execution |
| any + `flat` | `none` | `0` | Flat strategy — no position |
| `approved`/`modified` + `long` | `buy` | `MaxPositionPct` | Long entry at risk-constrained size |
| `approved`/`modified` + `short` | `sell` | `MaxPositionPct` | Short entry at risk-constrained size |
| unknown | `none` | `0` | Unknown disposition/direction — safe default |

### 3.3 Semantic Fix: Drawdown Constraint Mapping

| Risk type | Constraint field used as `MaxPositionPct` | Before S265 | After S265 |
|-----------|------------------------------------------|-------------|------------|
| `position_exposure` | `Constraints.MaxPositionSize` | Correct | Correct (unchanged) |
| `drawdown_limit` | `Constraints.MaxExposure` | **Wrong** (`StopDistance` was used) | **Fixed** (`MaxExposure` = drawdown tolerance %) |

`StopDistance` represents a stop-loss ceiling (e.g., `0.0255` = 2.55% stop). Using it as position size produced intents where `Quantity = 0.0255` meant "2.55% position" when it actually represented "2.55% stop distance." `MaxExposure` represents the maximum drawdown tolerance percentage, which is semantically appropriate as a position constraint.

## 4. Responsibility Split

### Domain Responsibility (strategy + risk)

| Concern | Owner |
|---------|-------|
| Severity-based confidence scaling | Strategy resolvers (S250) |
| Strategy-type-aware risk multipliers | Risk evaluators (S251) |
| Position sizing and exposure limits | Risk evaluators |
| Drawdown tolerance and stop distance | Risk evaluators |
| Disposition decision (approved/modified/rejected) | Risk evaluators |
| Constraint values (max position, max exposure, stop distance) | Risk evaluators |

### Execution Responsibility (execution)

| Concern | Owner |
|---------|-------|
| Translating disposition + direction into side (buy/sell/none) | `PaperOrderEvaluator` |
| Applying constraint value as quantity | `PaperOrderEvaluator` |
| Simulating instant paper fills | `PaperFillSimulator` |
| Kill switch enforcement | `SafetyGate` via `ControlGate` |
| Staleness rejection | `SafetyGate` via `StalenessGuard` |
| Event publishing and retry | `ExecutionPublisherActor` |
| KV materialization and monotonicity | `ExecutionProjectionActor` |

### NOT Execution's Responsibility

| Concern | Why not |
|---------|---------|
| Choosing position size | Risk domain already computed `MaxPositionSize`/`MaxExposure` |
| Deciding whether to execute | Risk domain already decided via `Disposition` |
| Applying severity scaling | Strategy/risk domains already applied scaling factors |
| Managing order lifecycle beyond submitted/filled | Paper mode is instant; no pending/partial state management |

## 5. RiskInput After S265

```go
type RiskInput struct {
    Type             string `json:"type"`               // risk family name
    Disposition      string `json:"disposition"`         // approved/modified/rejected
    Confidence       string `json:"confidence"`          // risk-adjusted confidence
    Timeframe        int    `json:"timeframe"`           // candle timeframe
    StrategyType     string `json:"strategy_type"`       // S265: strategy family name
    DecisionSeverity string `json:"decision_severity"`   // S265: originating severity
}
```

This structure is **execution-owned** — it does not import from the risk or strategy domains. The field names match the source fields but the types are execution-local primitives (strings and ints).

## 6. Contract Invariants

1. **Domain isolation**: Execution never imports `risk.RiskAssessment` or `strategy.Strategy` — all data crosses as primitives via `riskAssessedMessage`
2. **Causal trace**: `CorrelationID` and `CausationID` must survive from decision through execution fill
3. **Behavioral context**: `StrategyType` and `DecisionSeverity` now survive into `ExecutionIntent.RiskInput` and `Parameters`
4. **Semantic correctness**: Each risk type maps its appropriate constraint field to `MaxPositionPct` (position → `MaxPositionSize`, drawdown → `MaxExposure`)
5. **No execution-side domain logic**: Execution does not interpret severity, scale confidence, or adjust position sizes — it receives final values from risk
