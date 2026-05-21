# Squeeze Breakout Strategy Resolver Design

## Purpose

This document describes the design of the `squeeze_breakout_entry` strategy resolver, which converts `bollinger_squeeze` decision events into canonical strategy resolution for the squeeze breakout use case.

## Context

After S288, the `bollinger` signal and `bollinger_squeeze` decision layers are fully wired. S289 lifts the slice into the strategy layer by providing a dedicated resolver that translates squeeze detection into actionable positional intent.

## Resolver Identity

| Property | Value |
|---|---|
| Strategy type | `squeeze_breakout_entry` |
| Input decision | `bollinger_squeeze` |
| Direction on trigger | `long` |
| Direction on no-trigger | `flat` |
| Entry type | `market` |
| Parameters | `breakout_target_pct`, `breakout_stop_pct` |

## Semantic Distinction

The Foundry has three strategy resolver families, each with distinct market semantics:

- **mean_reversion_entry**: counter-trend; enters on oversold (RSI) conditions expecting price reversion.
- **trend_following_entry**: pro-trend; enters on bullish crossover (EMA) expecting continuation.
- **squeeze_breakout_entry**: volatility-driven; enters on Bollinger squeeze detection, anticipating a sharp directional breakout after a period of compressed volatility.

Squeeze breakout is the first volatility-regime strategy in the Foundry. It does not depend on trend direction or oscillator extremes but on bandwidth compression as a predictor of imminent expansion.

## Severity Behavioral Activation (S250)

Decision severity directly influences strategy parameters:

### Confidence Scaling

| Severity | Multiplier |
|---|---|
| high | 1.00x |
| moderate | 0.90x |
| low | 0.80x |
| unknown/empty | 1.00x (neutral) |

### Parameter Adjustment

| Parameter | Base Value | High (x) | Moderate (x) | Low (x) |
|---|---|---|---|---|
| breakout_target_pct | 0.04 | 1.50 | 1.00 | 0.75 |
| breakout_stop_pct | 0.015 | 0.75 | 1.00 | 1.50 |

Rationale:
- High severity (strong squeeze) → wider target (expect bigger breakout), tighter stop (high conviction).
- Low severity (weak squeeze) → smaller target (conservative expectation), wider stop (allow more noise).
- This mirrors the trend-following pattern where conviction inversely relates to stop width.

## Data Flow

```
bollinger signal
    ↓ signalGeneratedMessage
bollinger_squeeze decision evaluator
    ↓ decisionEvaluatedMessage (type, outcome, confidence, severity, rationale)
squeeze_breakout_entry resolver actor
    ↓ publishStrategyMessage → NATS STRATEGY_EVENTS stream
    ↓ strategyResolvedMessage → SourceScopeActor (fan-out to risk)
```

## Output Shape

```go
Strategy{
    Type:       "squeeze_breakout_entry",
    Source:     "binancef",
    Symbol:     "btcusdt",
    Timeframe:  60,
    Direction:  "long",       // or "flat"
    Confidence: "0.6750",     // severity-scaled
    Decisions:  []DecisionInput{{
        Type:       "bollinger_squeeze",
        Outcome:    "triggered",
        Confidence: "0.7500",  // raw
        Severity:   "moderate",
        Rationale:  "...",
    }},
    Parameters: {
        "entry":               "market",
        "breakout_target_pct": "0.04",
        "breakout_stop_pct":   "0.01",
    },
    Metadata: {
        "decision_type":      "bollinger_squeeze",
        "decision_severity":  "moderate",
        "rationale":          "...",
        "decision_rationale": "...",
    },
    Final: true,
}
```

## Architecture Compliance

- Pure application logic: no I/O, no NATS, no actor references in the resolver.
- Primitive data interface (DBI-9): receives decision as strings, not domain objects.
- Reuses shared severity scaling functions from `severity_scaling.go`.
- Actor follows the established `StrategyResolverConfig` pattern.
- Registered in all canonical integration points: settings schema, dependency graph, NATS registry, derive supervisor, writer pipeline.
