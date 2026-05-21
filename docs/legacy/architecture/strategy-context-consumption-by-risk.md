# Strategy Context Consumption by Risk

> Stage S251 — Behavioral Wave 1

## Purpose

This document specifies the contract between strategy output and risk evaluation,
detailing exactly which fields risk consumes, how they influence risk behavior, and
what risk must NOT do with strategy context.

## Context Flow

```
Strategy Resolver
  ↓ strategyResolvedMessage
  ↓  ├─ StrategyType       → risk confidence factor, stop distance factor
  ↓  ├─ StrategyDirection   → core logic: flat/long/short path selection
  ↓  ├─ StrategyConfidence  → position sizing, stop distance calculation
  ↓  ├─ DecisionSeverity    → position limit scaling, drawdown tolerance
  ↓  └─ DecisionRationale   → metadata/traceability only
  ↓
Risk Evaluator (PositionExposure or DrawdownLimit)
```

## Field Consumption Matrix

### Position Exposure Evaluator

| Field | Usage Category | Behavior |
|---|---|---|
| `StrategyType` | **Behavioral (S251)** | Selects confidence multiplier: mean_reversion→×0.90, trend_following→×0.95 |
| `StrategyDirection` | **Core Logic** | Determines evaluation path: flat→auto-approve, long/short→size position |
| `StrategyConfidence` | **Core Logic** | Drives position sizing: `confidence × effectiveMaxPosition` |
| `DecisionSeverity` | **Behavioral (S251)** | Scales effective position limit: high→×1.15, moderate→×1.00, low→×0.80 |
| `DecisionRationale` | **Traceability** | Stored in metadata for audit trail |

### Drawdown Limit Evaluator

| Field | Usage Category | Behavior |
|---|---|---|
| `StrategyType` | **Behavioral (S251)** | Selects confidence multiplier (×0.85/×0.92) AND stop distance base (×0.85/×1.15) |
| `StrategyDirection` | **Core Logic** | Determines evaluation path: flat→auto-approve, long/short→compute stop |
| `StrategyConfidence` | **Core Logic** | Drives stop distance: `effectiveStopBase × confidence` |
| `DecisionSeverity` | **Behavioral (S251)** | Scales max drawdown tolerance: high→×1.15, moderate→×1.00, low→×0.80 |
| `DecisionRationale` | **Traceability** | Stored in metadata for audit trail |

## What Risk Must NOT Do

1. **Import strategy domain package** — risk uses primitive strings only.
2. **Reference strategy.Direction or strategy.Strategy types** — risk owns its own
   StrategyInput struct.
3. **Recompute or modify strategy fields** — risk trusts the upstream values.
4. **Assume specific strategy types** — unknown types get neutral defaults, never fail.
5. **Build a configurable rules engine** — scaling maps are package-level constants.
6. **Reject based on strategy type alone** — type only influences scaling, not disposition.

## StrategyInput (Risk-Owned)

```go
type StrategyInput struct {
    Type              string `json:"type"`
    Direction         string `json:"direction"`
    Confidence        string `json:"confidence"`
    Timeframe         int    `json:"timeframe"`
    DecisionSeverity  string `json:"decision_severity,omitempty"`
    DecisionRationale string `json:"decision_rationale,omitempty"`
}
```

This type is defined in `internal/domain/risk/risk.go` and belongs to the risk domain.
It mirrors strategy output fields as primitive types, preserving domain isolation.

## Behavioral Activation Timeline

| Stage | Field | Usage |
|---|---|---|
| Pre-S251 | `StrategyType` | Stored in StrategyInput but not used in logic |
| Pre-S251 | `DecisionSeverity` | Stored in metadata/rationale only |
| **S251** | `StrategyType` | **Active**: drives confidence factor and stop distance factor |
| **S251** | `DecisionSeverity` | **Active**: drives position limit and drawdown tolerance |

## Backward Compatibility

- **Unknown strategy types** → neutral confidence factors (0.92 for position, 0.88 for
  drawdown) and neutral stop factor (×1.00). Risk continues to function for any future
  strategy type without code changes.
- **Empty/none severity** → ×1.00 factor on all severity-scaled dimensions. Pre-S251
  chains produce identical behavior.
- **No new message fields** — all data was already flowing; S251 activates consumption.

## Downstream Impact

Risk assessments now carry richer context for execution evaluators:

- `Metadata["strategy_type"]` — enables execution to also differentiate by strategy family
- `Parameters["effective_*"]` — shows the actual limits applied, not just base config
- `Rationale` — explains exactly which factors influenced the assessment

The `riskAssessedMessage` to execution evaluators is unchanged in structure — it still
carries `StrategyDirection`, `StrategyConfidence`, and `DecisionSeverity`.
