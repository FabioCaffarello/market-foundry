# Evidence Type 01: Trade Burst — Contracts

> Defines the domain type, event envelope, NATS subjects, and query contracts for the trade burst evidence type.

## Domain Type

```go
type EvidenceTradeBurst struct {
    Source     string    `json:"source"`       // Exchange identifier (e.g., "binancef")
    Symbol     string    `json:"symbol"`       // Lowercase symbol (e.g., "btcusdt")
    Timeframe  int       `json:"timeframe"`    // Window seconds (e.g., 60, 300)
    TradeCount int64     `json:"trade_count"`  // Total trades in window
    BuyVolume  string    `json:"buy_volume"`   // Decimal — volume where buyer is maker
    SellVolume string    `json:"sell_volume"`  // Decimal — volume where buyer is taker
    OpenTime   time.Time `json:"open_time"`    // Window start
    CloseTime  time.Time `json:"close_time"`   // Window end
    Burst      bool      `json:"burst"`        // True if trade_count > 2× previous window
    Final      bool      `json:"final"`        // True = window closed (immutable)
}
```

### Validation Rules

| Field | Rule |
|-------|------|
| Source | Required, non-empty |
| Symbol | Required, non-empty |
| Timeframe | Required, positive integer |
| BuyVolume | Required, non-empty decimal string |
| SellVolume | Required, non-empty decimal string |
| OpenTime | Required, non-zero |
| CloseTime | Required, non-zero, after OpenTime |

### Burst Detection

The `Burst` flag uses the simplest possible anomaly detection:

```
burst = (trade_count > 2.0 × previous_window_trade_count) AND (previous_window_trade_count > 0)
```

- First window: `Burst` is always false (no baseline yet)
- Threshold ratio: 2.0× (hardcoded, sufficient for initial evidence)
- Baseline: previous window's trade count only (no rolling average)

### Buy/Sell Volume

- `BuyVolume`: sum of `price × quantity` for trades where `BuyerMaker=true`
- `SellVolume`: sum of `price × quantity` for trades where `BuyerMaker=false`
- All values as decimal strings (same precision policy as candle OHLCV)

## Event Contract

### TradeBurstSampledEvent

```go
type TradeBurstSampledEvent struct {
    Metadata   events.Metadata    `json:"metadata"`
    TradeBurst EvidenceTradeBurst `json:"trade_burst"`
}
```

**Event name:** `tradeburst.sampled`

## NATS Subjects

### Event Stream

| Aspect | Value |
|--------|-------|
| Stream | `EVIDENCE_EVENTS` (shared with candles) |
| Publish subject | `evidence.events.tradeburst.sampled.{source}.{symbol}.{timeframe}` |
| Dedup key | `burst:{source}:{symbol}:{timeframe}:{open_time_unix}` |
| Encoding | CBOR Envelope |

### Durable Consumer

| Aspect | Value |
|--------|-------|
| Name | `store-trade-burst` |
| Filter | `evidence.events.tradeburst.sampled.>` |
| Ack wait | 30s |
| Max deliver | 5 |

### Query Control

| Aspect | Value |
|--------|-------|
| Subject | `evidence.query.tradeburst.latest` |
| Request type | `evidence.query.v1.trade_burst_latest_request` |
| Reply type | `evidence.query.v1.trade_burst_latest_reply` |
| Queue group | `evidence.query` |

## Query Contracts

### Request

```go
type TradeBurstLatestQuery struct {
    Source    string `json:"source"`
    Symbol   string `json:"symbol"`
    Timeframe int   `json:"timeframe"`
}
```

### Reply

```go
type TradeBurstLatestReply struct {
    TradeBurst *EvidenceTradeBurst `json:"trade_burst,omitempty"`
}
```

## HTTP Endpoint

```
GET /evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60
```

**Response (200 OK):**
```json
{
  "trade_burst": {
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "trade_count": 342,
    "buy_volume": "28451263.50000000",
    "sell_volume": "15234891.20000000",
    "open_time": "2024-03-10T14:00:00Z",
    "close_time": "2024-03-10T14:01:00Z",
    "burst": true,
    "final": true
  }
}
```

**Response (200 OK, no data yet):**
```json
{
  "trade_burst": null
}
```

## KV Projection

| Aspect | Value |
|--------|-------|
| Bucket | `TRADE_BURST_LATEST` |
| Key format | `{source}.{symbol}.{timeframe}` |
| Max bytes | 64 MB |
| Monotonicity | OpenTime guard (same as CANDLE_LATEST) |
| Storage | FileStorage |

## Intentional Limitations

1. **No history bucket** — trade bursts are latest-only for now. History can be added following the candle history pattern when needed.
2. **No configurable threshold** — burst ratio (2.0×) is hardcoded. Sufficient for proving the pattern; parameterizable in a future stage.
3. **Single-window baseline** — burst detection compares only to the immediately previous window, not a rolling average. Simple and predictable.
4. **No burst-specific query** — no endpoint to query "only burst windows". The `burst` field in the response lets clients filter.
