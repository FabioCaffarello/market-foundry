# Risk First Slice — Position Exposure

## Overview

Risk is the sixth domain layer in the market-foundry pipeline:

```
observation → evidence → signal → decision → strategy → risk
```

The risk domain evaluates strategy outputs and produces disposition assessments (approved, modified, rejected) that govern whether a strategy recommendation should proceed. Risk never imports upstream domain types directly; it defines its own input contracts at the domain boundary.

## First Family: Position Exposure (RF-01)

Position Exposure is a stateless, rule-based evaluator. Given a strategy input (symbol, direction, suggested size, confidence), it assesses whether the proposed exposure falls within acceptable limits and emits a disposition.

**Dispositions:**

| Disposition | Meaning |
|-------------|---------|
| `approved`  | Exposure within limits, strategy may proceed unchanged |
| `modified`  | Exposure partially acceptable, size adjusted downward |
| `rejected`  | Exposure exceeds limits, strategy blocked |

The evaluator is pure: no external state, no side effects, deterministic output for a given input.

## Actor Ownership

### Derive Service

| Actor | Responsibility |
|-------|---------------|
| `PositionExposureEvaluatorActor` | Receives resolved strategy messages, runs the position exposure evaluator, forwards risk events to the publisher |
| `RiskPublisherActor` | Publishes assessed risk events to the RISK_EVENTS stream |

The strategy resolver actor fans out resolved strategy messages to risk evaluator actors via `ScopePID`. Each risk family processor is instantiated per source scope, following the same pattern as signal, decision, and strategy families.

### Store Service

| Actor | Responsibility |
|-------|---------------|
| `RiskConsumerActor` | Consumes from the RISK_EVENTS stream via durable consumer |
| `RiskProjectionActor` | Projects risk events into the KV latest bucket |

## Stream and Storage

- **Stream:** `RISK_EVENTS`
- **Subject pattern:** `risk.events.position_exposure.assessed.{source}.{symbol}.{timeframe}`
- **KV bucket:** `RISK_POSITION_EXPOSURE_LATEST`
- **Key format:** `{source}.{symbol}.{timeframe}`

## Query Surface

- **Endpoint:** `GET /risk/{type}/latest`
- **Query subject:** `risk.query.position_exposure.latest`
- **Request/Reply** over NATS with queue group `risk.query`

## Activation Model

Risk uses the same two-layer activation as all other domain families:

1. **pipeline.risk_families** — static configuration in `derive.jsonc` and `store.jsonc` listing enabled risk families and their parameters
2. **Binding watcher** — dynamic activation via configctl bindings; risk evaluator actors are spawned only when a binding includes an active risk family for the given source/symbol/timeframe

## Dependency Chain

```
risk → strategy → decision → signal → evidence
```

Risk evaluator actors are spawned after strategy resolver actors. A risk event is produced only when a strategy resolved event is received. If strategy is not active for a binding, no risk evaluation occurs.

## Domain Boundary

The risk domain defines its own `StrategyInput` struct. It never imports the `strategy` domain package. This preserves domain isolation and prevents coupling drift. The adapter layer translates strategy resolved events into the risk domain's input type.

## Deferred

The following are explicitly out of scope for the first slice:

- History projections (`RISK_POSITION_EXPOSURE_HISTORY` bucket)
- Multi-strategy risk evaluation (current slice handles single `StrategyInput` only)
- Drawdown Guard family (RF-02) — requires execution/portfolio state
- Correlation Limit family (RF-03) — requires multi-symbol portfolio view
- Volatility Scaler family (RF-04) — requires volatility evidence stream
- ClickHouse risk analytics
- Portfolio-level exposure aggregation
