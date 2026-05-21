# Decision Type 02: EMA Crossover — Semantics, Rationale, and Boundaries

**Stage:** S241
**Charter:** BREADTH-WAVE-1
**Date:** 2026-03-21

---

## 1. Semantic Identity

| Property | Value |
|----------|-------|
| **Type name** | `ema_crossover` |
| **Signal source** | `ema_crossover` signal (from `EMACrossoverSampler`) |
| **Analytical model** | Trend detection via dual EMA comparison |
| **Core question** | "Is the fast EMA above the slow EMA (bullish trend)?" |
| **Decision philosophy** | Trend confirmation — enters *with* the trend |

### Contrast with Type 01 (`rsi_oversold`)

| Dimension | `rsi_oversold` | `ema_crossover` |
|-----------|----------------|-----------------|
| Signal type | Continuous oscillator (0–100) | Categorical direction (bullish/bearish/neutral) |
| Analytical model | Mean reversion (counter-trend) | Trend following (with-trend) |
| Trigger condition | RSI < threshold (oversold) | Fast EMA > slow EMA (bullish crossover) |
| Severity source | Distance below threshold | Fixed baseline (categorical input) |
| Confidence model | Proportional to threshold distance | Fixed by direction category |

---

## 2. Input/Output Contract

### Input

The evaluator receives a `signalGeneratedMessage` with:
- `SignalType`: `"ema_crossover"`
- `SignalValue`: one of `"bullish"`, `"bearish"`, `"neutral"`
- `Timeframe`: candle duration in seconds
- `Timestamp`: signal generation time

### Output

A `Decision` struct with:

| Field | Bullish | Bearish | Neutral |
|-------|---------|---------|---------|
| `Type` | `ema_crossover` | `ema_crossover` | `ema_crossover` |
| `Outcome` | `triggered` | `not_triggered` | `not_triggered` |
| `Severity` | `moderate` | `none` | `none` |
| `Confidence` | `0.7500` | `0.7500` | `0.5000` |
| `Rationale` | "EMA crossover bullish: fast EMA above slow EMA..." | "EMA crossover bearish: ..." | "EMA crossover neutral: ..." |
| `Metadata.crossover_direction` | `bullish` | `bearish` | `neutral` |

---

## 3. Design Rationale

### 3.1 Why Categorical Evaluation

The EMA crossover signal sampler already classifies the EMA relationship into categories. The evaluator respects this categorization rather than reimplementing EMA math. This keeps the evaluator's responsibility clear: it decides whether the signal category constitutes a triggering condition, not whether the EMAs have crossed.

### 3.2 Why Fixed Severity/Confidence

The `signalGeneratedMessage` contract carries only primitive values (type, value, timeframe, timestamp) per DBI-9. Signal metadata (fast_ema, slow_ema, spread) is not propagated through the actor chain. Rather than:

1. **Widening the message contract** (infrastructure scope increase, cross-cutting change), or
2. **Re-computing EMAs in the evaluator** (duplicating signal logic, violating separation of concerns)

We chose **baseline values** that are semantically valid:
- `0.75` confidence for directional signals (the crossover is confirmed but magnitude unknown)
- `0.50` confidence for neutral signals (insufficient information to judge)
- `moderate` severity for bullish (crossover detected but strength undetermined)

This is explicitly a breadth delivery. Depth enrichment (carrying spread magnitude for graduated severity/confidence) is deferred.

### 3.3 Why Only Bullish Triggers

The evaluator treats only `"bullish"` as triggering and `"bearish"` as not-triggered. This is a design choice aligned with the S242 target: the `trend_following_entry` strategy will look for bullish entry signals. A future variant could treat bearish as triggering for short-entry strategies, but that adds a second evaluator type (e.g., `ema_crossover_bearish`) rather than overloading this one.

---

## 4. Boundaries and Non-Goals

### What This Type Does
- Converts EMA crossover direction into a structured decision
- Produces rationale explaining the crossover state
- Outputs metadata for downstream strategy consumption
- Validates through the standard `Decision.Validate()` pipeline

### What This Type Does NOT Do
- Does not compute EMAs (that's the signal sampler's job)
- Does not evaluate multiple signal types simultaneously
- Does not consider historical crossover patterns (single-event evaluation)
- Does not carry signal-level numeric metadata (spread, EMA values)
- Does not produce `OutcomeInsufficient` (the signal sampler handles warm-up gating)

### Scope Constraints
- No new ClickHouse columns or migration needed (reuses existing `decisions` schema)
- No new HTTP endpoints (reuses `GET /decision/:type/latest`)
- No changes to downstream strategy or risk evaluators
- No multi-timeframe crossover detection

---

## 5. Observability

The EMA crossover decision is fully observable through:

1. **NATS events**: Subject `decision.events.ema_crossover.evaluated.{source}.{symbol}.{timeframe}`
2. **KV latest state**: Bucket `DECISION_EMA_CROSSOVER_LATEST`, key `{source}.{symbol}.{timeframe}`
3. **ClickHouse analytical**: `SELECT * FROM decisions WHERE type = 'ema_crossover'`
4. **HTTP query**: `GET /decision/ema_crossover/latest?source=...&symbol=...&timeframe=...`
5. **Actor logs**: Tagged with `actor=ema-crossover-evaluator`

---

## 6. Future Depth Opportunities (Not In Scope)

If signal metadata propagation becomes available (via widened `signalGeneratedMessage` or side-channel lookup):

- **Graduated severity**: `spread >= 2%` → high, `>= 1%` → moderate, `< 1%` → low
- **Proportional confidence**: `0.5 + 0.5 * (spread / max_expected_spread)`
- **Richer metadata**: `fast_ema`, `slow_ema`, `spread_pct` in decision metadata
- **Crossover freshness**: Track if this is the initial crossover candle vs. continuation
