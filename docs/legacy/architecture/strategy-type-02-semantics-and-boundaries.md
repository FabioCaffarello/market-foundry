# Strategy Type 02: `trend_following_entry` — Semantics and Boundaries

## Identity

- **Type name**: `trend_following_entry`
- **Domain**: strategy
- **Tier**: 1 (core pipeline)
- **Primary decision source**: `ema_crossover`

## Semantic Definition

`trend_following_entry` resolves a directional entry strategy based on trend confirmation from EMA crossover signals. It is a **pro-trend** strategy: when the fast EMA crosses above the slow EMA (bullish), it enters long expecting price continuation in the direction of the trend.

### Contrast with `mean_reversion_entry`

| Dimension          | mean_reversion_entry           | trend_following_entry          |
|--------------------|--------------------------------|--------------------------------|
| Philosophy         | Counter-trend (buy the dip)    | Pro-trend (ride momentum)      |
| Entry trigger      | RSI oversold condition         | EMA bullish crossover          |
| Exit mechanism     | Fixed target + stop offsets    | Trailing stop + take profit %  |
| Signal source      | RSI indicator                  | EMA crossover indicator        |
| Decision type      | rsi_oversold                   | ema_crossover                  |
| Direction bias     | Long only (on oversold)        | Long only (on bullish)         |
| Risk profile       | Quick reversion expected       | Sustained move expected        |

## Outcome Mapping

| Decision Outcome | Direction | Confidence      | Parameters                                                      |
|------------------|-----------|-----------------|-----------------------------------------------------------------|
| triggered        | long      | from decision   | entry=market, trailing_stop_pct=0.03, take_profit_pct=0.05     |
| not_triggered    | flat      | 0.0000          | none                                                            |
| insufficient     | flat      | 0.0000          | none (metadata: reason=insufficient_data)                       |

## Domain Boundaries

### What `trend_following_entry` owns
- Resolution logic: mapping decision outcomes to directional strategies
- Parameter selection: trailing stop and take profit percentages
- Metadata propagation: decision rationale into strategy metadata

### What `trend_following_entry` does NOT own
- Signal computation (owned by `ema_crossover` signal sampler)
- Decision evaluation (owned by `ema_crossover` decision evaluator)
- Risk assessment (owned by `position_exposure` risk evaluator)
- Execution intent (owned by `paper_order` execution evaluator)

### Domain Isolation

- The resolver receives primitive data (`string`, `int`, `time.Time`) per DBI-9
- It does NOT import from the decision domain
- Strategy owns `DecisionInput` as a value type for traceability
- Risk receives `strategyResolvedMessage` with primitive fields, not `strategy.Strategy`

## Infrastructure Contracts

| Artifact                | Value                                                        |
|-------------------------|--------------------------------------------------------------|
| NATS event subject      | `strategy.events.trend_following_entry.resolved.{source}.{symbol}.{timeframe}` |
| NATS event type         | `strategy.events.v1.trend_following_entry_resolved`          |
| NATS stream             | `STRATEGY_EVENTS` (shared with mean_reversion_entry)         |
| KV bucket               | `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST`                      |
| Writer durable          | `writer-strategy-trend-following-entry`                      |
| Store durable           | `store-strategy-trend-following-entry`                       |
| ClickHouse table        | `strategies` (shared, type column differentiates)            |
| Query subject           | `strategy.query.trend_following_entry.latest`                |
| Config key              | `pipeline.strategy_families: [trend_following_entry]`        |

## Simplifications and Trade-offs

1. **Fixed parameters**: Trailing stop (3%) and take profit (5%) are compile-time constants. No runtime configuration or per-symbol tuning yet.

2. **Long-only**: Like `mean_reversion_entry`, this type only resolves to `long` or `flat`. Short entries are deferred to a future stage.

3. **Single-decision input**: Each strategy resolves from exactly one decision. Multi-decision aggregation (e.g., requiring both RSI oversold AND EMA bullish) is explicitly out of scope.

4. **No decision-type filtering**: The resolver processes all `decisionEvaluatedMessage` fan-outs. It resolves on any outcome (`triggered`/`not_triggered`/`insufficient`) regardless of the originating decision type. This keeps the resolver simple and stateless.

## Preparation for S243 (Risk Breadth)

The `trend_following_entry` strategy flows through the same risk evaluation path as `mean_reversion_entry`. The `position_exposure` risk evaluator treats all strategy types uniformly — it evaluates direction, confidence, and position sizing without knowing which strategy type produced the input.

When S243 adds a second risk type, it will naturally receive strategies from both `mean_reversion_entry` and `trend_following_entry`, providing real breadth at the risk input surface.
