# Decision First Slice — Market Foundry

> Canonical implementation reference for the first vertical slice of the `decision` domain.
> Stage: S43
> Approved: 2026-03-17

---

## 1. Family: RSI Oversold (DF-01)

The first decision family evaluates whether the RSI signal indicates an oversold condition:

- **Input**: RSI signal value (from `rsi` signal family)
- **Threshold**: RSI < 30.0 → `triggered`; RSI >= 30.0 → `not_triggered`
- **Confidence**: Graduated based on distance from threshold
- **Scope**: Per-symbol, per-timeframe, latest-only

---

## 2. Pipeline Flow

```
observation → evidence (candle) → signal (RSI) → decision (RSI Oversold)
                                                         ↓
                                                   DECISION_EVENTS
                                                         ↓
                                                  store (projection)
                                                         ↓
                                                   NATS KV latest
                                                         ↓
                                                 gateway (HTTP query)
```

### Derive Binary

```
SourceScopeActor
├── SamplerActor (candle)           → candleFinalizedMessage
├── RSISignalSamplerActor           → signalGeneratedMessage (to scope)
├── RSIOversoldEvaluatorActor       → publishDecisionMessage
├── EvidencePublisherActor
├── SignalPublisherActor
└── DecisionPublisherActor          ← NEW
```

Signal samplers notify the SourceScopeActor via `signalGeneratedMessage` (primitive data, not signal.Signal — per DBI-9). The scope fans out to decision evaluators for the matching symbol.

### Store Binary

```
StoreSupervisor
├── DecisionProjectionActor         ← NEW (materializes to DECISION_RSI_OVERSOLD_LATEST)
├── DecisionConsumerActor           ← NEW (durable consumer on DECISION_EVENTS)
└── QueryResponderActor             ← EXTENDED (serves decision.query.rsi_oversold.latest)
```

---

## 3. Domain Model

```go
type Decision struct {
    Type       string            // "rsi_oversold"
    Source     string            // "binancef"
    Symbol     string            // "btcusdt"
    Timeframe  int               // 60
    Outcome    Outcome           // triggered | not_triggered | insufficient
    Confidence string            // "0.8500"
    Signals    []SignalInput      // [{Type: "rsi", Value: "28.50", Timeframe: 60}]
    Metadata   map[string]string // {"threshold": "30.0"}
    Final      bool              // true
    Timestamp  time.Time
}
```

---

## 4. NATS Contracts

| Artifact | Value |
|---|---|
| **Stream** | `DECISION_EVENTS` |
| **Subject** | `decision.events.rsi_oversold.evaluated.{source}.{symbol}.{timeframe}` |
| **Envelope** | `decision.events.v1.rsi_oversold_evaluated` |
| **KV Bucket** | `DECISION_RSI_OVERSOLD_LATEST` |
| **Key Format** | `{source}.{symbol}.{timeframe}` |
| **Query Subject** | `decision.query.rsi_oversold.latest` |
| **Durable Consumer** | `store-decision-rsi-oversold` |

---

## 5. HTTP Query Surface

```
GET /decision/{type}/latest?source=X&symbol=Y&timeframe=Z
```

Response:
```json
{
  "decision": {
    "type": "rsi_oversold",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "outcome": "triggered",
    "confidence": "0.8500",
    "signals": [{"type": "rsi", "value": "28.50", "timeframe": 60}],
    "metadata": {"threshold": "30.0"},
    "final": true,
    "timestamp": "2026-03-17T12:00:00Z"
  }
}
```

---

## 6. Activation

### Config-driven (requires restart)

```jsonc
// derive.jsonc + store.jsonc
"pipeline": {
    "decision_families": ["rsi_oversold"]
}
```

### Runtime binding (via BindingWatcher)

Decision evaluators are spawned per binding when the family is enabled. No change to BindingWatcher — it already spawns all configured processors per binding.

---

## 7. Projection Gates

| Gate | Rule |
|---|---|
| **Final** | Only `Final=true` decisions enter the read model |
| **Validate** | `decision.Validate()` must pass |
| **Monotonicity** | Latest bucket only advances forward in time |

---

## 8. What Is Deferred

| Item | Deferred To |
|---|---|
| Decision history bucket | S44+ |
| Decision history query | S44+ |
| Multi-signal confluence families | S44+ |
| MACD crossover family | S44+ (requires MACD signal) |
| Strategy/risk/execution/portfolio | Phase 3+ |
| Raccoon-CLI decision governance | S44+ |
