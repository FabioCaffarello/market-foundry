# Decision Family 01 — RSI Oversold Contracts

> Canonical contract reference for the first decision family (DF-01).
> Stage: S43

---

## Identity

| Property | Value |
|---|---|
| **Family** | `rsi_oversold` |
| **Decision Type** | `rsi_oversold` |
| **Input Signal** | `rsi` (single-signal family) |
| **Evaluation** | RSI < 30.0 → `triggered` |
| **Phase** | 1 — latest-only |

---

## Domain Contract

```go
// internal/domain/decision/decision.go
Decision{
    Type:       "rsi_oversold",
    Outcome:    OutcomeTriggered | OutcomeNotTriggered | OutcomeInsufficient,
    Confidence: "0.0000" .. "1.0000",
    Signals:    []SignalInput{{Type: "rsi", Value: "<decimal>", Timeframe: <int>}},
    Metadata:   {"threshold": "30.0"},
}
```

---

## Evaluator Contract

```go
// internal/application/decision/rsi_oversold_evaluator.go
func (e *RSIOversoldEvaluator) Evaluate(signalType, signalValue string, signalTimeframe int, ts time.Time) (Decision, bool)
```

- **Pure function**: no I/O, no actor references
- **Threshold**: 30.0 (default, not configurable in Phase 1)
- **Confidence formula**:
  - Triggered: `0.5 + 0.5 * (threshold - rsi) / threshold`
  - Not triggered: `0.5 + 0.5 * (rsi - threshold) / (100 - threshold)`

---

## Event Contract

| Field | Value |
|---|---|
| **Stream** | `DECISION_EVENTS` |
| **Subject** | `decision.events.rsi_oversold.evaluated.{source}.{symbol}.{timeframe}` |
| **Envelope type** | `decision.events.v1.rsi_oversold_evaluated` |
| **Retention** | 72h, file-backed, 2 GB max |
| **Deduplication** | `dec:rsi_oversold:{source}:{symbol}:{timeframe}:{timestamp_unix}` |

---

## Projection Contract

| Field | Value |
|---|---|
| **KV Bucket** | `DECISION_RSI_OVERSOLD_LATEST` |
| **Key format** | `{source}.{symbol}.{timeframe}` |
| **Storage** | File, 64 MB max |
| **Gates** | Final, Validate, Monotonicity |

---

## Consumer Contract

| Field | Value |
|---|---|
| **Durable** | `store-decision-rsi-oversold` |
| **Filter** | `decision.events.rsi_oversold.evaluated.>` |
| **AckWait** | 30s |
| **MaxDeliver** | 5 |

---

## Query Contract

| Field | Value |
|---|---|
| **NATS Subject** | `decision.query.rsi_oversold.latest` |
| **Request type** | `decision.query.v1.rsi_oversold_latest_request` |
| **Reply type** | `decision.query.v1.rsi_oversold_latest_reply` |
| **Queue group** | `decision.query` |
| **HTTP endpoint** | `GET /decision/rsi_oversold/latest?source=X&symbol=Y&timeframe=Z` |

---

## Activation Contract

```jsonc
// derive.jsonc
"pipeline": { "decision_families": ["rsi_oversold"] }

// store.jsonc
"pipeline": { "decision_families": ["rsi_oversold"] }
```

**Prerequisite**: `signal_families` must include `rsi` for the evaluator to receive input.
This is an operational dependency, not a code coupling — config validation does not enforce it.
