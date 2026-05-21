# First Slice Contracts — Market Foundry

> Canonical document. Defines the minimum event contracts, domain types, and message schemas for the first vertical slice.
> Designed: 2026-03-16. All types defined here are the source of truth for implementation.

---

## 1. Domain Types

### 1.1 Observation Domain (`internal/domain/observation`)

#### ObservationTrade

The canonical representation of a single trade captured from an external source.

```go
package observation

import (
    "time"
    "internal/shared/events"
)

// ObservationTrade represents a normalized trade event from an external market source.
type ObservationTrade struct {
    Source    string    `json:"source"`     // Exchange identifier (e.g., "binancef")
    Symbol   string    `json:"symbol"`     // Instrument symbol (e.g., "btcusdt")
    Price    string    `json:"price"`      // Decimal string — no float precision loss
    Quantity string    `json:"quantity"`   // Decimal string
    TradeID  string    `json:"trade_id"`   // Source-assigned trade identifier
    BuyerMaker bool   `json:"buyer_maker"` // True if the buyer is the maker
    Timestamp time.Time `json:"timestamp"` // Exchange-reported trade time
}
```

**Design decisions:**
- `Price` and `Quantity` are strings to avoid IEEE 754 precision loss. Domain consumers parse as needed.
- `TradeID` is source-assigned, used for deduplication at the JetStream level (`Nats-Msg-Id`).
- `Timestamp` is the exchange timestamp, not ingest receipt time.
- `Source` is lowercase, matching the subject key segment convention.
- `Symbol` is lowercase, matching the subject key segment convention.

#### Observation Events

```go
const (
    EventTradeReceived events.Name = "market.trade_received"
)

type TradeReceivedEvent struct {
    Metadata events.Metadata  `json:"metadata"`
    Trade    ObservationTrade  `json:"trade"`
}

func (e TradeReceivedEvent) EventName() events.Name        { return EventTradeReceived }
func (e TradeReceivedEvent) EventMetadata() events.Metadata { return e.Metadata }
```

---

### 1.2 Evidence Domain (`internal/domain/evidence`)

#### EvidenceCandle

The canonical representation of a sampled OHLCV candle.

```go
package evidence

import (
    "time"
    "internal/shared/events"
)

// EvidenceCandle represents a sampled OHLCV candle for a specific symbol and timeframe.
type EvidenceCandle struct {
    Source     string    `json:"source"`      // Exchange identifier (e.g., "binancef")
    Symbol    string    `json:"symbol"`      // Instrument symbol (e.g., "btcusdt")
    Timeframe int       `json:"timeframe"`   // Window duration in seconds (60, 300)
    Open      string    `json:"open"`        // Decimal string
    High      string    `json:"high"`        // Decimal string
    Low       string    `json:"low"`         // Decimal string
    Close     string    `json:"close"`       // Decimal string
    Volume    string    `json:"volume"`      // Decimal string — total traded volume
    TradeCount int64    `json:"trade_count"` // Number of trades in the window
    OpenTime  time.Time `json:"open_time"`   // Window start (floor of first trade)
    CloseTime time.Time `json:"close_time"`  // Window end (open_time + timeframe)
    Final     bool      `json:"final"`       // True = window closed; false = interim/realtime
}
```

**Design decisions:**
- `Timeframe` is an integer (seconds), not a duration string. 60 = 1 minute, 300 = 5 minutes.
- `Open/High/Low/Close/Volume` are decimal strings. Same rationale as ObservationTrade.
- `Final` flag distinguishes finalized (window closed) from interim (in-progress) candles. Both use the same subject; consumers filter by `Final` field.
- `OpenTime` is always `floor(first_trade_timestamp / timeframe) * timeframe`.
- `CloseTime` is always `OpenTime + timeframe_duration`.

#### Evidence Events

```go
const (
    EventCandleSampled events.Name = "candle.sampled"
)

type CandleSampledEvent struct {
    Metadata events.Metadata `json:"metadata"`
    Candle   EvidenceCandle  `json:"candle"`
}

func (e CandleSampledEvent) EventName() events.Name        { return EventCandleSampled }
func (e CandleSampledEvent) EventMetadata() events.Metadata { return e.Metadata }
```

---

## 2. Envelope Contracts

All messages use `Envelope[T]` from `internal/shared/envelope`. The `Type` field identifies the schema version.

### 2.1 Observation Events (ingest → JetStream)

| Envelope Field | Value |
|----------------|-------|
| **Kind** | `event` |
| **Type** | `observation.events.v1.trade_received` |
| **Source** | `ingest` |
| **Subject** | `observation.events.market.trade.{source}` |
| **Payload** | `TradeReceivedEvent` |
| **Nats-Msg-Id** | `{source}:{trade_id}` (deduplication) |

**Example serialized envelope:**
```json
{
  "id": "01JQXX...",
  "kind": "event",
  "type": "observation.events.v1.trade_received",
  "source": "ingest",
  "subject": "observation.events.market.trade.binancef",
  "timestamp": "2026-03-16T14:30:00.123Z",
  "payload": {
    "metadata": {
      "id": "01JQXX...",
      "name": "market.trade_received",
      "occurred_at": "2026-03-16T14:30:00.100Z"
    },
    "trade": {
      "source": "binancef",
      "symbol": "btcusdt",
      "price": "84521.30",
      "quantity": "0.150",
      "trade_id": "4839201",
      "buyer_maker": false,
      "timestamp": "2026-03-16T14:30:00.098Z"
    }
  }
}
```

### 2.2 Evidence Events (derive → JetStream)

| Envelope Field | Value |
|----------------|-------|
| **Kind** | `event` |
| **Type** | `evidence.events.v1.candle_sampled` |
| **Source** | `derive` |
| **Subject** | `evidence.events.candle.sampled.{source}.{symbol}.{timeframe}` |
| **Payload** | `CandleSampledEvent` |
| **CausationID** | ID of the observation event that triggered the window close |

**Example serialized envelope:**
```json
{
  "id": "01JQXY...",
  "kind": "event",
  "type": "evidence.events.v1.candle_sampled",
  "source": "derive",
  "subject": "evidence.events.candle.sampled.binancef.btcusdt.60",
  "causation_id": "01JQXX...",
  "timestamp": "2026-03-16T14:31:00.005Z",
  "payload": {
    "metadata": {
      "id": "01JQXY...",
      "name": "candle.sampled",
      "occurred_at": "2026-03-16T14:31:00.001Z"
    },
    "candle": {
      "source": "binancef",
      "symbol": "btcusdt",
      "timeframe": 60,
      "open": "84521.30",
      "high": "84589.90",
      "low": "84510.00",
      "close": "84575.40",
      "volume": "12.345",
      "trade_count": 87,
      "open_time": "2026-03-16T14:30:00Z",
      "close_time": "2026-03-16T14:31:00Z",
      "final": true
    }
  }
}
```

### 2.3 Evidence Query (gateway → derive, request/reply)

**Request:**

| Envelope Field | Value |
|----------------|-------|
| **Kind** | `request` |
| **Type** | `evidence.query.v1.candle_latest_request` |
| **Source** | `gateway` |
| **Subject** | `evidence.query.candle.latest` |
| **Payload** | `CandleLatestQuery` |

```go
type CandleLatestQuery struct {
    Source    string `json:"source"`
    Symbol   string `json:"symbol"`
    Timeframe int   `json:"timeframe"`
}
```

**Reply:**

| Envelope Field | Value |
|----------------|-------|
| **Kind** | `reply` |
| **Type** | `evidence.query.v1.candle_latest_reply` |
| **Source** | `derive` |
| **Payload** | `CandleLatestReply` |

```go
type CandleLatestReply struct {
    Candle *EvidenceCandle `json:"candle,omitempty"` // nil if no candle available
}
```

---

## 3. NATS Registry Definitions

### 3.1 ObservationRegistry (`internal/adapters/nats`)

```go
type ObservationRegistry struct {
    TradeReceived EventSpec
}

func DefaultObservationRegistry() ObservationRegistry {
    eventStream := StreamSpec{
        Name:     "OBSERVATION_EVENTS",
        Subjects: []string{"observation.events.market.>"},
        Storage:  jetstream.FileStorage,
        MaxAge:   6 * time.Hour,
        MaxBytes: 1 * 1024 * 1024 * 1024,
    }

    return ObservationRegistry{
        TradeReceived: EventSpec{
            Subject: "observation.events.market.trade",
            Type:    "observation.events.v1.trade_received",
            Stream:  eventStream,
        },
    }
}
```

**Note:** The base subject `observation.events.market.trade` is extended with `.{source}` at publish time.

### 3.2 EvidenceRegistry (`internal/adapters/nats`)

```go
type EvidenceRegistry struct {
    CandleSampled EventSpec
    CandleLatest  ControlSpec
}

func DefaultEvidenceRegistry() EvidenceRegistry {
    eventStream := StreamSpec{
        Name:     "EVIDENCE_EVENTS",
        Subjects: []string{"evidence.events.candle.>"},
        Storage:  jetstream.FileStorage,
        MaxAge:   72 * time.Hour,
        MaxBytes: 2 * 1024 * 1024 * 1024,
    }

    return EvidenceRegistry{
        CandleSampled: EventSpec{
            Subject: "evidence.events.candle.sampled",
            Type:    "evidence.events.v1.candle_sampled",
            Stream:  eventStream,
        },
        CandleLatest: ControlSpec{
            Subject:     "evidence.query.candle.latest",
            RequestType: "evidence.query.v1.candle_latest_request",
            ReplyType:   "evidence.query.v1.candle_latest_reply",
            QueueGroup:  "evidence.query",
        },
    }
}
```

**Note:** The base subject `evidence.events.candle.sampled` is extended with `.{source}.{symbol}.{timeframe}` at publish time.

### 3.3 Consumer Definitions

```go
// derive consuming observation trades
var DeriveObservationConsumer = ConsumerSpec{
    Durable: "derive-observation",
    Event: EventSpec{
        Subject: "observation.events.market.trade.>",
        Type:    "observation.events.v1.trade_received",
        Stream: StreamSpec{
            Name: "OBSERVATION_EVENTS",
        },
    },
    AckWait:    30 * time.Second,
    MaxDeliver: 5,
}
```

---

## 4. Binance Futures Adapter Contract

### 4.1 WebSocket Connection

```
URL: wss://fstream.binance.com/ws/btcusdt@aggTrade
```

### 4.2 aggTrade Payload (Binance → ingest)

```json
{
  "e": "aggTrade",
  "E": 1710600600098,
  "s": "BTCUSDT",
  "a": 4839201,
  "p": "84521.30",
  "q": "0.150",
  "f": 12345678,
  "l": 12345680,
  "T": 1710600600098,
  "m": false
}
```

### 4.3 Mapping: aggTrade → ObservationTrade

| aggTrade field | ObservationTrade field | Transformation |
|----------------|----------------------|----------------|
| (hardcoded) | `Source` | `"binancef"` |
| `s` | `Symbol` | `strings.ToLower(s)` |
| `p` | `Price` | Direct string copy |
| `q` | `Quantity` | Direct string copy |
| `a` | `TradeID` | `strconv.FormatInt(a, 10)` |
| `m` | `BuyerMaker` | Direct bool copy |
| `T` | `Timestamp` | `time.UnixMilli(T).UTC()` |

### 4.4 Deduplication Key

```
Nats-Msg-Id = "binancef:" + strconv.FormatInt(aggTrade.a, 10)
```

JetStream deduplication window prevents re-publishing the same trade on reconnection.

---

## 5. HTTP API Contract (gateway extension)

### GET /evidence/candles/latest

**Request:**
```
GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `source` | string | yes | Exchange identifier |
| `symbol` | string | yes | Instrument symbol |
| `timeframe` | int | yes | Candle duration in seconds |

**Response (200 OK):**
```json
{
  "candle": {
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "open": "84521.30",
    "high": "84589.90",
    "low": "84510.00",
    "close": "84575.40",
    "volume": "12.345",
    "trade_count": 87,
    "open_time": "2026-03-16T14:30:00Z",
    "close_time": "2026-03-16T14:31:00Z",
    "final": true
  }
}
```

**Response (404 Not Found):**
```json
{
  "type": "not_found",
  "title": "No candle available",
  "detail": "No candle data available for source=binancef symbol=btcusdt timeframe=60"
}
```

**Response (400 Bad Request):**
```json
{
  "type": "invalid_argument",
  "title": "Invalid query parameters",
  "validation_issues": [
    {"field": "timeframe", "message": "must be a positive integer"}
  ]
}
```

Error responses follow the existing `problem.Problem` pattern from `internal/shared/problem`.

---

## 6. Contract Invariants

These invariants must hold for every message in the slice:

1. **Every event is wrapped in `Envelope[T]`.** No raw struct publishing.
2. **Every envelope has a unique ID.** Generated by `envelope.New()`.
3. **Every event payload implements `events.Event`.** Has `EventName()` and `EventMetadata()`.
4. **Observation events use trade timestamp, not system clock.** `Timestamp` field reflects exchange time.
5. **Evidence candle `OpenTime` is deterministic.** `floor(first_trade.Timestamp / timeframe) * timeframe`.
6. **Price and volume are decimal strings.** No float64 in domain types for monetary values.
7. **Subject keys are lowercase.** `binancef`, `btcusdt` — never `BINANCEF` or `BTCUSDT`.
8. **Deduplication uses source-assigned IDs.** `Nats-Msg-Id` for observation; envelope ID for evidence.
9. **CausationID links evidence to observation.** The candle's causation ID is the trade that triggered finalization.
10. **Finalized candles are immutable.** Once `final: true` is published, the same `open_time + source + symbol + timeframe` is never re-published.
