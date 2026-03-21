# Controlled Paper Order Generation

## Purpose

This document describes how market-foundry generates paper orders from domain intelligence in a controlled, auditable manner. Paper order generation is the activation of the decision → strategy → risk → execution pipeline, producing simulated execution intents that are observable, traceable, and bounded by explicit guard rails.

## What Changed

Prior to S266, the actor chain was fully wired from signal through risk assessment, but the risk → execution leg was not exercised end-to-end. The `PaperOrderEvaluatorActor` existed and was capable of receiving `riskAssessedMessage` from risk evaluators via `ScopePID` fan-out, but no end-to-end proof existed that domain intelligence actually produced paper orders.

S266 activates this path by:
1. Confirming that risk evaluator actors send `riskAssessedMessage` to their `ScopePID` (execution evaluator)
2. Proving the full chain end-to-end across 7 behavioral scenarios
3. Documenting the generation semantics, guard rails, and boundaries

## Generation Path

```
Signal → Decision Evaluator → Strategy Resolver → Risk Evaluator(s) → Paper Order Evaluator → Publisher
                                                        ↓ ScopePID fan-out
                                                   riskAssessedMessage
                                                        ↓
                                                PaperOrderEvaluatorActor
                                                        ↓
                                              PaperOrderEvaluator.Evaluate()
                                                        ↓
                                              PaperFillSimulator.SimulateFill()
                                                        ↓
                                            publishExecutionMessage → Publisher
```

### Step-by-Step

1. **Risk evaluator** produces a `RiskAssessment` and, if `ScopePID` is set, constructs a `riskAssessedMessage` with primitive data (risk type, disposition, confidence, max position %, strategy direction/confidence/type, decision severity, timeframe, timestamp, trace IDs).

2. **PaperOrderEvaluatorActor** receives the message and calls `PaperOrderEvaluator.Evaluate()`:
   - `rejected` disposition → `SideNone`, quantity `"0"`
   - `flat` strategy direction → `SideNone`, quantity `"0"`
   - `long` + `approved`/`modified` → `SideBuy`, quantity = `maxPositionPct`
   - `short` + `approved`/`modified` → `SideSell`, quantity = `maxPositionPct`

3. **Causal trace** is attached: `CorrelationID` (original signal trace) and `CausationID` (risk event ID).

4. **PaperFillSimulator.SimulateFill()** transitions actionable intents:
   - `SideNone` → unchanged (stays `StatusSubmitted`, no fills)
   - `SideBuy`/`SideSell` → `StatusFilled`, one `FillRecord` with `Simulated: true`

5. **Domain validation** (`intent.Validate()`) ensures all required fields are present.

6. **PaperOrderSubmittedEvent** is constructed with event metadata and sent to the execution publisher actor.

## Dual Risk Fan-Out

Each strategy fans out to both risk evaluators (position_exposure and drawdown_limit). Each risk evaluator independently produces a `riskAssessedMessage` to its respective execution evaluator. This means a single triggered signal can produce **two** independent paper orders — one per risk assessment type.

This is by design: each risk evaluator applies different constraints (position sizing vs drawdown tolerance), and the execution layer records which risk type originated each order.

## Observability Fields

Every paper order carries full causal context:

| Field | Source | Purpose |
|-------|--------|---------|
| `Risk.Type` | Risk evaluator | Which risk assessment produced this order |
| `Risk.Disposition` | Risk evaluator | Whether risk approved, modified, or rejected |
| `Risk.Confidence` | Risk evaluator | Risk-adjusted confidence |
| `Risk.StrategyType` | Strategy resolver | Which strategy family (S265) |
| `Risk.DecisionSeverity` | Decision evaluator | Originating severity level (S265) |
| `Parameters["strategy_type"]` | Strategy resolver | Observability duplicate of strategy type |
| `Parameters["decision_severity"]` | Decision evaluator | Observability duplicate of severity |
| `Parameters["risk_disposition"]` | Risk evaluator | Observability duplicate of disposition |
| `Parameters["max_position_pct"]` | Risk evaluator | Position constraint applied |
| `CorrelationID` | Signal | End-to-end trace identity |
| `CausationID` | Risk event | Immediate causal parent |

## Severity Influence on Order Quantity

Decision severity propagates through the full chain and produces observably different paper order sizes:

- **High severity** (e.g., RSI 10.0, distance=20): larger position size → larger order quantity
- **Moderate severity**: baseline position size → baseline order quantity
- **Low severity** (e.g., RSI 25.0, distance=5): smaller position size → smaller order quantity

This is not controlled by the execution layer — it is a natural consequence of severity-aware risk scaling (S251) flowing through to the `maxPositionPct` value in `riskAssessedMessage`.

## Non-Objectives

- No real venue interaction
- No portfolio tracking or aggregation
- No OMS (order management system)
- No multi-venue routing
- No real money or real fills
- No SafetyGate integration (deferred to S267)
