# Squeeze Path: Paper Execution Integration

Status: **Closed** (S290)
Scope: How the squeeze breakout slice integrates with the mature paper execution loop.

---

## Integration Architecture

The squeeze breakout slice reaches paper execution through the same canonical path as mean_reversion_entry and trend_following_entry:

```
bollinger signal → bollinger_squeeze decision → squeeze_breakout_entry strategy
    → [position_exposure | drawdown_limit] risk → paper_order execution
```

No new actors, messages, publishers, or KV stores were introduced. The integration is achieved entirely through:
1. Actor routing (already generic — fans strategies to all risk evaluators)
2. Risk scaling factors (newly added in S290)
3. Execution evaluation (already strategy-type-agnostic)

---

## Data Flow Through Paper Execution

### Stage 1: Strategy Resolution

The `SqueezeBreakoutEntryResolverActor` receives `decisionEvaluatedMessage` from the bollinger squeeze decision evaluator and produces:

- **Published event**: `StrategyResolvedEvent` on NATS subject `strategy.events.squeeze_breakout_entry.resolved.{source}.{symbol}.{timeframe}`
- **Internal message**: `strategyResolvedMessage` sent to `ScopePID` for fan-out to risk evaluators

Key fields carried forward:
- `StrategyType = "squeeze_breakout_entry"`
- `StrategyDirection = "long"` (on triggered) or `"flat"` (on not_triggered/insufficient)
- `StrategyConfidence` = severity-scaled confidence
- `DecisionSeverity` = originating decision severity
- `DecisionRationale` = originating decision rationale

### Stage 2: Risk Assessment (Fan-Out)

The `SourceScopeActor.routeStrategyToRisk()` fans `strategyResolvedMessage` to ALL enabled risk evaluators for the symbol. Both `position_exposure` and `drawdown_limit` receive and independently assess the squeeze breakout strategy.

**Position Exposure** produces:
- `RiskAssessment.Disposition` = approved/modified/rejected
- `RiskAssessment.Constraints.MaxPositionSize` = severity-adjusted, confidence-scaled position size
- `RiskAssessment.Confidence` = strategy confidence x 0.93

**Drawdown Limit** produces:
- `RiskAssessment.Disposition` = approved/modified/rejected
- `RiskAssessment.Constraints.StopDistance` = type-adjusted, confidence-scaled stop distance
- `RiskAssessment.Confidence` = strategy confidence x 0.90

Each risk assessment is published independently and routed to execution.

### Stage 3: Paper Order Generation

The `PaperOrderEvaluatorActor` receives `riskAssessedMessage` from each risk evaluator. For squeeze breakout:

- **Approved long** → Side=buy, Quantity=maxPositionPct
- **Modified long** → Side=buy, Quantity=cappedMaxPositionPct
- **Rejected** → Side=none, Quantity=0
- **Flat** → Side=none, Quantity=0

The paper fill simulator then applies simulated fill logic (immediate fill at market for paper mode).

### Stage 4: Publication

The execution intent is published to:
- **Stream**: `EXECUTION_EVENTS`
- **Subject**: `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`
- **KV**: `EXECUTION_PAPER_ORDER_LATEST` with key `{source}.{symbol}.{timeframe}`

---

## Causal Trace Preservation

The full causal chain is preserved through the pipeline:

```
Decision:  correlation_id=C1, causation_id=<signal_event_id>
           decision_severity="high", decision_rationale="Bollinger squeeze detected"

Strategy:  correlation_id=C1, causation_id=<decision_event_id>
           strategy_type="squeeze_breakout_entry", decision_severity="high"

Risk:      correlation_id=C1, causation_id=<strategy_event_id>
           strategy_input.type="squeeze_breakout_entry"
           strategy_input.decision_severity="high"

Execution: correlation_id=C1, causation_id=<risk_event_id>
           risk.strategy_type="squeeze_breakout_entry"
           risk.decision_severity="high"
           parameters.strategy_type="squeeze_breakout_entry"
```

---

## Paper Mode Guarantees

The squeeze breakout path inherits all paper mode guarantees established in prior stages:

1. **No real orders**: All execution intents have `type = "paper_order"` and are processed by `PaperFillSimulator`
2. **Monotonicity**: KV stores reject stale timestamps at every layer
3. **Deduplication**: NATS dedup keys prevent duplicate processing
4. **Idempotency**: Each stage produces deterministic output for identical inputs
5. **Staleness guard**: Execution staleness max age (120s) prevents acting on old risk assessments

---

## Configuration Example

Minimal pipeline config for squeeze breakout paper execution:

```json
{
  "pipeline": {
    "families": ["candle"],
    "signal_families": ["bollinger"],
    "decision_families": ["bollinger_squeeze"],
    "strategy_families": ["squeeze_breakout_entry"],
    "risk_families": ["position_exposure", "drawdown_limit"],
    "execution_families": ["paper_order"]
  }
}
```

This config is validated by `PipelineConfig.ValidatePipeline()` with the "at least one" dependency semantics.

---

## What Was Not Changed

| Component | Reason |
|-----------|--------|
| Actor wiring in `derive_supervisor.go` | Already generic — `squeeze_breakout_entry` was registered in S289 |
| `source_scope_actor.go` routing | Already fans all strategies to all risk/execution evaluators |
| NATS publishers/registries | Already type-parameterized — no squeeze-specific subjects needed |
| Domain models (risk, execution) | Already generic — accept any strategy type |
| KV stores | Already type-parameterized via family registration |
| Writer pipeline | Already codegen-governed for strategy/risk/execution layers |
| Paper fill simulator | Strategy-type-agnostic — simulates fill regardless of origin |
