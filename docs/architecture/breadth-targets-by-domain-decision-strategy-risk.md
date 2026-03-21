# Breadth Targets by Domain: Decision, Strategy, Risk

**Charter:** BREADTH-WAVE-1
**Reference:** breadth-charter-and-scope-freeze.md

---

## 1. Current State (Baseline)

| Domain | Type Count | Existing Type | Family YAML | Signal Source |
|--------|-----------|--------------|-------------|---------------|
| Decision | 1 | `rsi_oversold` | `rsi_oversold.yaml` | RSI signal |
| Strategy | 1 | `mean_reversion_entry` | `mean_reversion_entry.yaml` | RSI oversold decision |
| Risk | 1 | `position_exposure` | `position_exposure.yaml` | Mean reversion strategy |

All three domains have exactly one evaluator/resolver with full production coverage: domain validation, application logic, actor integration, codegen family, and CI-proven tests.

---

## 2. Decision Domain — Target: `ema_crossover`

### 2.1 What It Is

An evaluator that consumes EMA (Exponential Moving Average) signal pairs and produces a decision based on crossover detection — whether a fast EMA has crossed above or below a slow EMA.

### 2.2 Why This Candidate

1. **Infrastructure already exists.** The EMA signal family is defined (`codegen/families/ema.yaml`), the EMA sampler actor exists (`ema_crossover_signal_sampler_actor.go`), and the NATS event type is registered. No signal-layer work is needed.
2. **Genuinely distinct logic.** RSI oversold evaluates a single oscillator value against a threshold. EMA crossover evaluates the relative position and direction of two moving averages — a fundamentally different analytical model.
3. **Different signal source.** Consumes EMA signals, not RSI signals. This proves the decision domain can handle multiple signal types, validating the `SignalInput` abstraction.
4. **Natural pairing.** EMA crossover decisions feed naturally into trend-following strategies, creating a second complete analytical chain parallel to the RSI→mean-reversion path.

### 2.3 Candidates Considered and Rejected

| Candidate | Reason for Rejection |
|-----------|---------------------|
| Volume spike evaluator | Requires a new signal type (`volume`) not yet defined; adds signal-layer scope |
| Price momentum evaluator | Overlaps conceptually with EMA crossover; less distinct logic differentiation |
| Multi-timeframe RSI evaluator | Still RSI-based; constitutes depth (variant of existing type), not breadth |

### 2.4 Expected Implementation Shape

- **Family type:** `ema_crossover`
- **Input:** Two EMA values (fast and slow periods) via `SignalInput`
- **Outcome logic:**
  - Fast EMA > Slow EMA (bullish crossover) → `OutcomeTriggered`
  - Fast EMA ≤ Slow EMA (no crossover) → `OutcomeNotTriggered`
  - Insufficient data → `OutcomeInsufficient`
- **Severity:** Based on crossover magnitude (distance between EMAs as percentage)
- **Confidence:** Based on consistency/strength of the crossover signal
- **Delivery stage:** S241

---

## 3. Strategy Domain — Target: `trend_following_entry`

### 3.1 What It Is

A resolver that consumes EMA crossover decisions and produces a strategy with trend-following entry parameters — distinct from mean-reversion logic which enters against the trend.

### 3.2 Why This Candidate

1. **Logically paired with `ema_crossover`.** EMA crossover signals trend direction; a trend-following resolver acts on that signal by entering *with* the trend — the opposite philosophy from mean-reversion (which enters *against* the trend).
2. **Genuinely distinct resolution logic.** Mean reversion maps `triggered` → `DirectionLong` (buy the dip). Trend following maps `triggered` → direction determined by the crossover direction (long if bullish crossover, short if bearish). Different parameter model: trail stops instead of fixed target offsets.
3. **Validates fan-out.** Two resolvers consuming different decision types proves the `DecisionInput` abstraction and actor fan-out routing work for multiple strategy families.

### 3.3 Candidates Considered and Rejected

| Candidate | Reason for Rejection |
|-----------|---------------------|
| Breakout entry resolver | Requires price-level analysis not available from current signal types |
| Momentum continuation resolver | Overlaps with trend following; less architecturally distinct |
| Mean reversion exit resolver | Exit logic is a different concern (lifecycle, not entry); out of breadth scope |

### 3.4 Expected Implementation Shape

- **Family type:** `trend_following_entry`
- **Input:** EMA crossover decision outcome, confidence, severity, rationale via `DecisionInput`
- **Direction logic:**
  - Triggered + bullish crossover metadata → `DirectionLong`
  - Triggered + bearish crossover metadata → `DirectionShort`
  - Not triggered / insufficient → `DirectionFlat`
- **Parameters:** `entry_method: "trend_follow"`, `trail_offset`, `momentum_threshold`
- **Distinct from mean reversion:** Enters with the trend (not against it); uses trailing stops (not fixed targets); confidence scales with trend strength (not oversold distance)
- **Delivery stage:** S242

---

## 4. Risk Domain — Target: `drawdown_limit`

### 4.1 What It Is

An evaluator that assesses strategies against portfolio drawdown limits — rejecting or modifying positions when cumulative drawdown approaches or exceeds configured thresholds.

### 4.2 Why This Candidate

1. **Orthogonal risk dimension.** Position exposure evaluates individual position sizing. Drawdown limit evaluates cumulative portfolio health. These are independent risk axes that can be composed.
2. **Genuinely distinct logic.** Position exposure caps percentages; drawdown limit tracks cumulative loss trajectory and applies circuit-breaker logic (reject all trades if drawdown exceeds threshold).
3. **No new infrastructure required.** Risk domain's `StrategyInput` and `Constraints` structs already support the necessary input/output shape. The `Disposition` enum (approved/modified/rejected) covers all drawdown outcomes.
4. **Industry-standard risk model.** Maximum drawdown limits are a fundamental risk management primitive; their absence from a trading system is a notable gap.

### 4.3 Candidates Considered and Rejected

| Candidate | Reason for Rejection |
|-----------|---------------------|
| Correlation exposure evaluator | Requires multi-asset correlation data not available in current signal pipeline |
| Stop-loss optimizer | Overlaps with position exposure (both constrain individual positions); less distinct |
| Volatility-adjusted sizing evaluator | Requires volatility signal type not yet defined; adds signal-layer scope |

### 4.4 Expected Implementation Shape

- **Family type:** `drawdown_limit`
- **Input:** Strategy direction, confidence, and current portfolio state via `StrategyInput`
- **Disposition logic:**
  - Drawdown below warning threshold → `DispositionApproved` (no constraints added)
  - Drawdown between warning and critical thresholds → `DispositionModified` (reduced position size)
  - Drawdown above critical threshold → `DispositionRejected` (circuit breaker)
- **Constraints:** `max_drawdown_pct`, `warning_threshold_pct`, `critical_threshold_pct`
- **Distinct from position exposure:** Evaluates portfolio-level health, not individual position sizing; applies circuit-breaker pattern; stateful (tracks cumulative drawdown)
- **Delivery stage:** S243

---

## 5. Second Chain Topology

After breadth delivery, two parallel analytical chains will exist:

```
Chain A (Mean Reversion):
  candle → rsi_signal → rsi_oversold → mean_reversion_entry → position_exposure → paper_order

Chain B (Trend Following):
  candle → ema_signal → ema_crossover → trend_following_entry → drawdown_limit → paper_order
```

Both chains share:
- Evidence layer (candle)
- Execution layer (paper_order)
- Domain struct abstractions (SignalInput, DecisionInput, StrategyInput)
- Actor infrastructure (fan-out, publisher, projection)

This validates the architectural promise that domains are truly generic and can host multiple independent analytical families.

---

## 6. Breadth Measurement Matrix

| Domain | Metric | Threshold | How Measured |
|--------|--------|-----------|-------------|
| Decision | Distinct evaluator types | ≥ 2 | Count of files in `internal/application/decision/` with distinct family types |
| Decision | Distinct signal sources consumed | ≥ 2 | RSI + EMA signals feeding separate evaluators |
| Strategy | Distinct resolver types | ≥ 2 | Count of files in `internal/application/strategy/` with distinct family types |
| Strategy | Distinct resolution philosophies | ≥ 2 | Mean reversion (counter-trend) + Trend following (with-trend) |
| Risk | Distinct evaluator types | ≥ 2 | Count of files in `internal/application/risk/` with distinct family types |
| Risk | Distinct risk dimensions | ≥ 2 | Position sizing + Portfolio drawdown |
| All | Family YAML count | ≥ 6 | 3 existing + 3 new in `codegen/families/` |
| All | Chain integration paths | ≥ 2 | Two end-to-end paths exercised in integration tests |
