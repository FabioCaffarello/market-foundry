# Signal Family SF-01 — RSI Contracts

> Canonical contract reference for the RSI signal family.
> Phase 1: single-evidence (candle-only), latest-only projection.

## Domain Type

```go
type Signal struct {
    Type      string            `json:"type"`       // "rsi"
    Source    string            `json:"source"`     // e.g., "binancef"
    Symbol   string            `json:"symbol"`     // e.g., "btcusdt"
    Timeframe int              `json:"timeframe"`  // seconds
    Value     string            `json:"value"`      // RSI value (0–100 decimal string)
    Metadata  map[string]string `json:"metadata"`   // { "period", "avg_gain", "avg_loss" }
    Final     bool              `json:"final"`      // true = finalized
    Timestamp time.Time         `json:"timestamp"`  // computation time
}
```

## Event Contract

| Field | Value |
|-------|-------|
| Stream | `SIGNAL_EVENTS` |
| Subject | `signal.events.rsi.generated.{source}.{symbol}.{timeframe}` |
| Envelope type | `signal.events.v1.rsi_generated` |
| Durable consumer | `store-signal-rsi` |
| Filter subject | `signal.events.rsi.generated.>` |

## Query Contract

| Field | Value |
|-------|-------|
| NATS subject | `signal.query.rsi.latest` |
| Request type | `signal.query.v1.rsi_latest_request` |
| Reply type | `signal.query.v1.rsi_latest_reply` |
| Queue group | `signal.query` |

## HTTP Contract

| Method | Path | Query Params |
|--------|------|-------------|
| GET | `/signal/rsi/latest` | `source`, `symbol`, `timeframe` (required) |

### Response (200 OK)

```json
{
  "signal": {
    "type": "rsi",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "value": "65.3200",
    "metadata": {
      "period": "14",
      "avg_gain": "1.20000000",
      "avg_loss": "0.64000000"
    },
    "final": true,
    "timestamp": "2026-03-17T12:00:00Z"
  }
}
```

### Response (not found — 200 OK, null signal)

```json
{
  "signal": null
}
```

## KV Projection

| Field | Value |
|-------|-------|
| Bucket | `SIGNAL_RSI_LATEST` |
| Key format | `{source}.{symbol}.{timeframe}` |
| Storage | FileStorage |
| Max bytes | 64 MB |

## Projection Gates

1. **Final gate** — only `Final=true` signals are written.
2. **Validate gate** — `Signal.Validate()` must pass.
3. **Monotonicity guard** — reject writes where `Timestamp <= existing.Timestamp`.

## Sampler Specification

| Property | Value |
|----------|-------|
| Algorithm | Wilder's smoothed moving average (RSI) |
| Default period | 14 |
| Warm-up candles | 15 (period + 1) |
| Input | Finalized candle close prices |
| Statefulness | Stateful — maintains `avgGain`, `avgLoss`, `prevClose` |
| I/O | None — pure application logic |

## Ownership

| Concern | Binary | Actor |
|---------|--------|-------|
| RSI computation | derive | `SignalSamplerActor[rsi/...]` |
| Signal event publishing | derive | `SignalPublisherActor` |
| RSI event consumption | store | `SignalConsumerActor[rsi]` |
| RSI KV projection | store | `SignalProjectionActor[rsi]` |
| RSI query serving | store | `QueryResponderActor` |
| RSI HTTP translation | gateway | `SignalWebHandler.GetLatestSignal` |
