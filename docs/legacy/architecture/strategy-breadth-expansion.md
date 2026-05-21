# Strategy Breadth Expansion

## Purpose

This document records the breadth expansion of the `strategy` domain from one resolver/type (`mean_reversion_entry`) to two (`mean_reversion_entry` + `trend_following_entry`). This satisfies the charter requirement of achieving real breadth (≥2 types/resolvers) in every domain before exiting the current wave.

## Motivation

The strategy domain had a mature end-to-end pipeline for `mean_reversion_entry`, but only one type. The charter mandates breadth parity across `decision`, `strategy`, and `risk`. Since `decision` already achieved breadth in S241 (`rsi_oversold` + `ema_crossover`), strategy needed a second type to stay coherent with the pipeline.

## Strategy Type Selection: `trend_following_entry`

The second strategy type was chosen based on:

1. **Semantic complementarity**: `mean_reversion_entry` is counter-trend (enters on oversold conditions expecting reversion). `trend_following_entry` is pro-trend (enters on bullish crossover expecting continuation). These represent fundamentally different trading philosophies.

2. **Decision alignment**: `ema_crossover` decisions were already flowing through the pipeline but had no dedicated strategy consumer. `trend_following_entry` provides the natural pairing.

3. **Controlled scope**: Both strategies share the same domain model (`Strategy` struct), the same event type (`StrategyResolvedEvent`), and the same infrastructure (NATS stream, ClickHouse table). The addition is purely additive.

## Decision → Strategy Pairing

| Decision Type   | Strategy Type            | Semantic Intent           |
|-----------------|--------------------------|---------------------------|
| rsi_oversold    | mean_reversion_entry     | Counter-trend: buy dip    |
| ema_crossover   | trend_following_entry    | Pro-trend: ride momentum  |

Both strategy types receive all `decisionEvaluatedMessage` fan-outs. Each resolver acts on the outcomes it understands (`triggered`, `not_triggered`, `insufficient`) regardless of which decision type produced them. The resolver logic is outcome-driven, not decision-type-specific.

## Implementation Pattern

The breadth expansion follows the same formulaic pattern established in S241 for decision breadth:

1. **Application layer**: Pure resolver function (`TrendFollowingEntryResolver.Resolve()`)
2. **Actor layer**: Thin actor wrapper (`TrendFollowingEntryResolverActor`)
3. **Supervisor registration**: One entry in `strategyProcessors` list
4. **NATS registry**: `TrendFollowingEntryResolved` EventSpec + `TrendFollowingEntryLatest` ControlSpec
5. **Publisher routing**: Switch case in `specForType()`
6. **KV bucket**: `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST`
7. **Store pipeline**: Projection + consumer entry in `declarePipelines()`
8. **Writer pipeline**: Consumer-inserter pair writing to `strategies` table
9. **Codegen family**: `trend_following_entry.yaml`

No shared code was modified. The `Strategy` domain model, projection actor, and consumer are fully reusable across types.

## Parameter Semantics

### mean_reversion_entry (existing)
```
entry:         "market"
target_offset: "0.02"   (2% target)
stop_offset:   "0.01"   (1% stop)
```

### trend_following_entry (new)
```
entry:            "market"
trailing_stop_pct: "0.03"  (3% trailing stop)
take_profit_pct:   "0.05"  (5% take profit)
```

The parameter difference reflects the semantic distinction: mean reversion uses fixed offsets (expecting quick reversion), while trend following uses percentage-based trailing mechanisms (expecting sustained moves).

## Non-Goals

- No multi-decision aggregation (each strategy consumes exactly one decision).
- No short-side strategy types in this stage.
- No runtime parameter tuning — parameters are compile-time defaults.
- No cross-strategy interaction or combination logic.
