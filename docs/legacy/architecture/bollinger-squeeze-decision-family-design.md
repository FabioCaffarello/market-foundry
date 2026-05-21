# Bollinger Squeeze Decision Family — Design

## Purpose

The `bollinger_squeeze` decision family evaluates whether Bollinger Band signals indicate a volatility compression (squeeze) condition. A squeeze occurs when the bandwidth between the upper and lower Bollinger Bands contracts below a relative threshold, signaling potential breakout conditions. This family closes the Signal Evolution Wave by proving that signal infrastructure built in S284-S286 translates into concrete decision-layer value.

## Domain Concept

A **Bollinger Squeeze** is a well-known technical analysis pattern:

- Bollinger Bands measure volatility using a moving average (SMA) plus/minus K standard deviations.
- When price action compresses, the bands narrow — bandwidth decreases relative to the SMA.
- Sustained low bandwidth (squeeze) often precedes a directional breakout.
- The squeeze itself is direction-agnostic; the %B value indicates where price sits within the bands.

## Input Contract

| Field | Source | Description |
|-------|--------|-------------|
| signalType | `"bollinger"` | From Bollinger signal sampler |
| signalValue | %B decimal string | Position within bands: `(price - lower) / (upper - lower)` |
| signalMetadata["bandwidth"] | Band absolute width | `upper - lower` |
| signalMetadata["sma"] | Simple moving average | Rolling SMA over period |
| signalTimeframe | int (seconds) | Evidence window duration |

## Evaluation Logic

### Relative Bandwidth

The evaluator computes **relative bandwidth** = `bandwidth / SMA`. This normalizes band width across different price levels (a 200-point band on BTC at 50000 is very different from 200 points at 2000).

### Squeeze Detection

- `relativeBW < squeezeThreshold (0.10)` → **OutcomeTriggered** — squeeze detected
- `relativeBW >= squeezeThreshold` → **OutcomeNotTriggered** — normal or wide bands

### Severity Classification

For triggered decisions, severity is based on compression depth (ratio = relativeBW / threshold):

| Ratio Range | Severity | Interpretation |
|-------------|----------|----------------|
| ratio <= 0.25 | High | Extreme compression (< 25% of threshold) |
| 0.25 < ratio <= 0.50 | Moderate | Significant compression (25-50% of threshold) |
| ratio > 0.50 | Low | Mild squeeze (50-100% of threshold) |

### Confidence Calculation

**Triggered:** Confidence ∈ [0.5, 1.0], increasing as bandwidth moves further below threshold.

```
confidence = 0.5 + 0.5 * (threshold - relativeBW) / threshold
```

**Not triggered:** Confidence ∈ [0.5, 1.0), increasing as bandwidth moves further above threshold.

```
excess = relativeBW - threshold
confidence = 0.5 + 0.5 * excess / (excess + threshold)
```

### %B Zone Classification

The evaluator enriches metadata with the %B zone:

| %B Range | Zone | Interpretation |
|----------|------|----------------|
| < 0.20 | lower | Near or below lower band |
| 0.20 - 0.80 | middle | Between bands |
| > 0.80 | upper | Near or above upper band |

## Output Contract

The evaluator produces a standard `decision.Decision`:

```
Type:       "bollinger_squeeze"
Outcome:    triggered | not_triggered
Severity:   none | low | moderate | high
Confidence: "0.5000" - "1.0000"
Signals:    [{type: "bollinger", value: %B, timeframe: N}]
Metadata:   {squeeze_threshold, relative_bandwidth, bandwidth, sma, pct_b, pct_b_zone}
```

## Dependency Graph

```
evidence/candle → signal/bollinger → decision/bollinger_squeeze
```

The `bollinger_squeeze` decision family requires the `bollinger` signal family to be enabled. The `bollinger` signal family requires the `candle` evidence family.

## Architecture Integration Points

### Signal-to-Decision Bridge

The `signalGeneratedMessage` carries `SignalMetadata map[string]string`, enabling the Bollinger Squeeze evaluator to access bandwidth and SMA from the signal without importing signal domain types. This follows DBI-9 (primitive data across boundaries).

### Actor System

`BollingerSqueezeEvaluatorActor` follows the canonical decision evaluator actor pattern:
1. Receives `signalGeneratedMessage` from scope fan-out
2. Delegates to `BollingerSqueezeEvaluator.Evaluate()`
3. Publishes `DecisionEvaluatedEvent` via decision publisher
4. Fans out `decisionEvaluatedMessage` to strategy resolvers via scope

### NATS Infrastructure

- Event spec: `decision.events.bollinger_squeeze.evaluated`
- Control spec: `decision.query.bollinger_squeeze.latest`
- Writer consumer: `writer-decision-bollinger-squeeze`
- Store consumer: `store-decision-bollinger-squeeze`
- KV bucket: `DECISION_BOLLINGER_SQUEEZE_LATEST`

### Configuration

Registered in `settings/schema.go`:
- `knownDecisionFamilies["bollinger_squeeze"]`
- `decisionDependsOnSignal["bollinger_squeeze"] = ["bollinger"]`

## Design Decisions

1. **Relative bandwidth over absolute**: Absolute bandwidth is price-level-dependent. A 100-point band on a 50000 asset is 0.2%, while on a 100 asset it's 100%. Relative bandwidth normalizes this.

2. **Metadata-aware evaluator**: Unlike RSI and EMA crossover evaluators that work on signal value alone, Bollinger Squeeze requires both %B (value) and bandwidth+SMA (metadata). The evaluator accepts an additional `metadata map[string]string` parameter.

3. **Threshold at 10%**: The default squeeze threshold of 0.10 (10% relative bandwidth) is a well-established empirical level in Bollinger Band analysis. It balances sensitivity with noise rejection.

4. **Zone enrichment**: The %B zone in metadata provides directional context without coupling the decision outcome to direction. Strategy resolvers downstream can use zone information to determine entry direction.
