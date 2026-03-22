# Source Selection and Canonical Integration Contract

> **Stage**: S359 — Source Selection and Canonical Contract
> **Wave**: Strategy/Signal Integration (S358–S363)
> **Status**: Binding
> **Date**: 2026-03-22

---

## 1. Executive Summary

This document selects the canonical signal+strategy pair for the Strategy/Signal Integration Wave and defines the field-level contract that maps strategy output to execution-ready intent. The selection prioritizes simplicity, auditability, and architectural fit over sophistication.

**Selected pair**: RSI signal family + Mean Reversion Entry strategy family.

**Contract boundary**: `StrategyResolvedEvent` on NATS `STRATEGY_EVENTS` stream → `StrategyConsumerActor` in execute scope → `ExecutionIntent` via `PaperOrderEvaluator`.

---

## 2. Source Inventory

### 2.1 Signal Families (6 implemented)

| Family | Type Key | Output | Directional | Complexity | Decision Consumer |
|--------|----------|--------|-------------|------------|-------------------|
| RSI | `rsi` | Single decimal (0–100) | Yes (thresholds) | Low | Yes (RSI evaluator) |
| EMA Crossover | `ema_crossover` | Binary crossover + metadata | Yes (cross direction) | Medium | Yes (EMA evaluator) |
| Bollinger Bands | `bollinger` | Multi-value (upper/middle/lower) | Indirect (band position) | Medium | Yes (squeeze evaluator) |
| MACD | `macd` | Multi-value (macd/signal/histogram) | Yes (crossover) | High | Not yet |
| VWAP | `vwap` | Single decimal | No (reference level) | Low | Not yet |
| ATR | `atr` | Single decimal (volatility) | No | Low | Not yet |

### 2.2 Strategy Families (3 implemented)

| Family | Type Key | Signal Dependency | Direction Logic | Parameters |
|--------|----------|-------------------|-----------------|------------|
| Mean Reversion Entry | `mean_reversion_entry` | RSI, Bollinger | RSI oversold→long, overbought→short | signal_type, strength, entry_level |
| Trend Following Entry | `trend_following_entry` | EMA Crossover | Crossover direction→long/short | trend_direction, momentum, level |
| Squeeze Breakout Entry | `squeeze_breakout_entry` | Bollinger | Squeeze breakout→directional | breakout_direction, momentum, volatility |

### 2.3 Existing Execution Path

The current derive binary chains: Signal → Decision → Strategy → Risk → PaperOrderEvaluator → ExecutionIntent → NATS.

The execute binary consumes `PaperOrderSubmittedEvent` from NATS and routes to `VenueAdapterActor`.

There is **no direct strategy-to-execution consumer** in execute scope today. This is the gap the wave fills.

---

## 3. Selection: Comparative Analysis

### 3.1 Evaluation Criteria

| Criterion | Weight | Description |
|-----------|--------|-------------|
| Simplicity | High | Minimal moving parts for first integration proof |
| Directional clarity | High | Clear, unambiguous mapping from signal to execution side |
| Architectural fit | High | Uses existing domain types without extension |
| Auditability | Medium | Human-readable provenance chain |
| Financial plausibility | Medium | Sensible market logic, even for paper mode |

### 3.2 Candidate Comparison

#### Candidate A: RSI + Mean Reversion Entry

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Simplicity | **High** | RSI is single-value (0–100). Mean reversion maps oversold/overbought directly to long/short. |
| Directional clarity | **High** | RSI < 30 → oversold → long entry. RSI > 70 → overbought → short entry. No ambiguity. |
| Architectural fit | **High** | RSI signal → RSI decision evaluator → mean reversion resolver already wired in derive. Strategy type `mean_reversion_entry` registered in NATS registry. PaperOrderEvaluator accepts strategy direction/confidence as primitives. |
| Auditability | **High** | Single decimal value + threshold = fully explainable. Parameters (signal_type, strength, entry_level) are human-readable. |
| Financial plausibility | **High** | Mean reversion is well-understood: buy low, sell high. |

#### Candidate B: EMA Crossover + Trend Following Entry

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Simplicity | Medium | Crossover detection requires two values (fast/slow EMA) and directional logic. |
| Directional clarity | **High** | Bullish crossover → long, bearish crossover → short. |
| Architectural fit | **High** | Full chain exists in derive. NATS registry has `trend_following_entry`. |
| Auditability | Medium | Crossover logic requires understanding of EMA periods. |
| Financial plausibility | **High** | Trend following is well-understood. |

#### Candidate C: Bollinger + Squeeze Breakout Entry

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| Simplicity | Low | Multi-value signal (3 bands), squeeze detection, breakout direction. |
| Directional clarity | Medium | Breakout direction depends on price relative to bands — more complex. |
| Architectural fit | **High** | Full chain exists. NATS registry has `squeeze_breakout_entry`. |
| Auditability | Low | Squeeze detection requires understanding band width, period, and breakout conditions. |
| Financial plausibility | **High** | Squeeze breakout is a valid pattern. |

### 3.3 Selection Decision

**Selected**: Candidate A — **RSI signal + Mean Reversion Entry strategy**.

**Rationale**: RSI provides the simplest, most directionally unambiguous signal. Mean Reversion Entry translates that signal into execution intent via the most straightforward mapping possible. This pair minimizes integration risk while exercising every domain boundary (signal → decision → strategy → execution). The pair also has the highest auditability, which is critical for the wave's explainability goal (SSI-3).

**Rejected alternatives are not inferior** — they are deferred. Once the RSI + Mean Reversion pattern is proven, the same wiring pattern applies to any signal+strategy pair.

---

## 4. Canonical Integration Contract

### 4.1 Contract Identity

| Property | Value |
|----------|-------|
| Source domain | Strategy |
| Source type | `mean_reversion_entry` |
| Source event | `StrategyResolvedEvent` |
| Source NATS subject | `strategy.events.mean_reversion_entry.resolved.>` |
| Source NATS stream | `STRATEGY_EVENTS` |
| Target domain | Execution |
| Target type | `paper_order` |
| Target evaluator | `PaperOrderEvaluator` |
| Consumer actor | `StrategyConsumerActor` (to be implemented in SSI-2) |
| Consumer scope | `internal/actors/scopes/execute/` |

### 4.2 Event Envelope: StrategyResolvedEvent

The source event arrives as a NATS JetStream message with this JSON structure:

```json
{
  "metadata": {
    "id": "uuid-v4",
    "name": "strategy_resolved",
    "correlation_id": "uuid-v4",
    "causation_id": "uuid-v4",
    "timestamp": "2026-03-22T10:00:00Z"
  },
  "strategy": {
    "type": "mean_reversion_entry",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "direction": "long",
    "confidence": "0.85",
    "decisions": [
      {
        "type": "rsi_oversold_signal",
        "outcome": "triggered",
        "confidence": "0.90",
        "severity": "high",
        "rationale": "RSI at 22.5, below lower threshold 30",
        "timeframe": 60
      }
    ],
    "parameters": {
      "signal_type": "rsi",
      "strength": "strong",
      "entry_level": "22.5"
    },
    "metadata": {},
    "final": true,
    "timestamp": "2026-03-22T10:00:00Z"
  }
}
```

### 4.3 Field-Level Contract: Strategy → ExecutionIntent

The `StrategyConsumerActor` transforms a `StrategyResolvedEvent` into a `PaperOrderEvaluator.Evaluate()` call. Since the wave excludes risk domain changes (NG-4), risk inputs use pass-through defaults.

#### Required Field Mapping

| PaperOrderEvaluator Parameter | Source | Mapping Rule |
|-------------------------------|--------|--------------|
| `riskType` | Default | `"pass_through"` — indicates risk layer was bypassed |
| `riskDisposition` | Default | `"approved"` — all strategy intents are approved by default |
| `riskConfidence` | Strategy | `strategy.Confidence` — forwarded as risk confidence |
| `maxPositionPct` | Config | Configurable default (e.g., `"0.01"`) — 1% position size cap |
| `strategyDirection` | Strategy | `string(strategy.Direction)` — "long", "short", or "flat" |
| `strategyConfidence` | Strategy | `strategy.Confidence` |
| `strategyType` | Strategy | `strategy.Type` — "mean_reversion_entry" |
| `decisionSeverity` | Strategy | `strategy.Decisions[0].Severity` — severity of primary decision |
| `riskTimeframe` | Strategy | `strategy.Timeframe` |
| `ts` | Strategy | `strategy.Timestamp` |

#### Output: ExecutionIntent

| ExecutionIntent Field | Source | Value |
|-----------------------|--------|-------|
| `Type` | Constant | `"paper_order"` |
| `Source` | Strategy | `strategy.Source` |
| `Symbol` | Strategy | `strategy.Symbol` |
| `Timeframe` | Strategy | `strategy.Timeframe` |
| `Side` | Evaluator | Derived: `direction=long → buy`, `direction=short → sell`, else `none` |
| `Quantity` | Config | `maxPositionPct` from configuration |
| `FilledQuantity` | Constant | `""` (unfilled at submission) |
| `Status` | Constant | `"submitted"` |
| `Risk.Type` | Default | `"pass_through"` |
| `Risk.Disposition` | Default | `"approved"` |
| `Risk.Confidence` | Strategy | `strategy.Confidence` |
| `Risk.Timeframe` | Strategy | `strategy.Timeframe` |
| `Risk.StrategyType` | Strategy | `strategy.Type` |
| `Risk.DecisionSeverity` | Strategy | `strategy.Decisions[0].Severity` |
| `Parameters["risk_type"]` | Default | `"pass_through"` |
| `Parameters["risk_disposition"]` | Default | `"approved"` |
| `Parameters["strategy_direction"]` | Strategy | `string(strategy.Direction)` |
| `Parameters["strategy_confidence"]` | Strategy | `strategy.Confidence` |
| `Parameters["strategy_type"]` | Strategy | `strategy.Type` |
| `Parameters["decision_severity"]` | Strategy | `strategy.Decisions[0].Severity` |
| `Parameters["max_position_pct"]` | Config | Configurable default |
| `CorrelationID` | Event metadata | `event.Metadata.CorrelationID` |
| `CausationID` | Event metadata | `event.Metadata.ID` |
| `Final` | Evaluator | `true` (paper mode — pre-filled) |
| `Timestamp` | Strategy | `strategy.Timestamp` |

### 4.4 Risk Pass-Through Justification

The wave charter explicitly excludes risk domain changes (NG-4). The `PaperOrderEvaluator` requires risk inputs because it was designed to sit downstream of risk evaluation. Rather than restructuring the evaluator (violating FC-7), the `StrategyConsumerActor` supplies pass-through defaults:

- `riskType = "pass_through"` — makes the bypass explicit and auditable
- `riskDisposition = "approved"` — all strategy intents proceed
- `maxPositionPct` — sourced from configuration, not from risk constraints

This is a **temporary bridge**. When a future wave introduces risk integration, the `StrategyConsumerActor` will be extended to query risk assessments before evaluation. The `pass_through` risk type makes it trivially auditable which intents bypassed risk.

### 4.5 Direction-to-Side Mapping (Invariant)

The `PaperOrderEvaluator` enforces this mapping, which the contract inherits:

| Strategy Direction | Risk Disposition | Execution Side | Quantity |
|--------------------|------------------|----------------|----------|
| `long` | `approved` | `buy` | `maxPositionPct` |
| `short` | `approved` | `sell` | `maxPositionPct` |
| `flat` | any | `none` | `"0"` |
| any | `rejected` | `none` | `"0"` |

Since risk disposition is always `approved` in pass-through mode, the effective mapping simplifies to:

| Strategy Direction | Execution Side |
|--------------------|----------------|
| `long` | `buy` |
| `short` | `sell` |
| `flat` | `none` |

### 4.6 NATS Consumer Spec

The `StrategyConsumerActor` will use this consumer specification:

```
Durable:    "execute-strategy-mean-reversion-entry"
Subject:    "strategy.events.mean_reversion_entry.resolved.>"
Stream:     "STRATEGY_EVENTS"
AckWait:    30s
MaxDeliver: 5
```

This follows the existing consumer naming convention (`{scope}-{domain}-{family}`).

---

## 5. Signal-to-Execution Chain (Full Provenance Path)

For completeness, here is the full chain from signal to execution for the selected pair:

```
RSI Signal (Value: "22.5", Metadata: {period: "14", lower_threshold: "30"})
    │
    ▼
RSI Decision (Outcome: "triggered", Severity: "high", Confidence: "0.90")
    │  SignalInput: {type: "rsi", value: "22.5", timeframe: 60}
    ▼
Mean Reversion Entry Strategy (Direction: "long", Confidence: "0.85")
    │  DecisionInput: {type: "rsi_oversold_signal", outcome: "triggered", severity: "high"}
    │  Parameters: {signal_type: "rsi", strength: "strong", entry_level: "22.5"}
    ▼
    ╔═══════════════════════════════════════════════════════╗
    ║  NATS STRATEGY_EVENTS stream boundary                ║
    ║  Subject: strategy.events.mean_reversion_entry       ║
    ║           .resolved.binancef.btcusdt.60              ║
    ╚═══════════════════════════════════════════════════════╝
    │
    ▼  [StrategyConsumerActor in execute scope — S359 contract]
    │
ExecutionIntent (Side: "buy", Quantity: "0.01", Status: "submitted")
    Risk: {type: "pass_through", disposition: "approved", strategy_type: "mean_reversion_entry"}
    Parameters: {strategy_type: "mean_reversion_entry", strategy_direction: "long",
                 strategy_confidence: "0.85", decision_severity: "high",
                 risk_type: "pass_through", risk_disposition: "approved"}
    CorrelationID: (propagated from signal origin)
    CausationID: (strategy event metadata.id)
```

---

## 6. Ownership and Responsibilities

| Concern | Owner | Boundary |
|---------|-------|----------|
| RSI signal generation | derive (SamplerActor) | Produces `SignalGeneratedEvent` |
| RSI decision evaluation | derive (DecisionEvaluatorActor) | Produces `DecisionEvaluatedEvent` |
| Mean reversion strategy resolution | derive (StrategyResolverActor) | Produces `StrategyResolvedEvent` → NATS |
| Strategy event consumption | execute (StrategyConsumerActor) | Subscribes to `strategy.events.mean_reversion_entry.resolved.>` |
| Intent evaluation | execute (PaperOrderEvaluator) | Pure function: strategy fields → ExecutionIntent |
| Intent submission | execute (VenueAdapterActor) | Existing: kill switch, staleness, paper/venue adapter |
| Read model materialization | store (QueryResponderActor) | Existing: KV projections for all domains |
| HTTP query surface | gateway | Existing: `/strategy/latest`, `/execution/latest`, `/activation/surface` |

---

## 7. What This Contract Does NOT Cover

- Risk evaluation (NG-4) — uses pass-through defaults
- Multiple signal families (NG-1) — RSI only
- Multiple strategy families (NG-2) — mean_reversion_entry only
- Venue submission (NG-7) — paper adapter only
- Strategy parameter optimization (NG-12)
- Multi-timeframe integration (NG-14)
- Correlation ID assignment at signal origin (deferred to SSI-3)
- Per-strategy-type gates and confidence thresholds (deferred to SSI-3)
- Prometheus metrics for strategy-driven execution (deferred to SSI-3/SSI-4)
