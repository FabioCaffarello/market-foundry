# observation — Raw market data

The `observation` domain models market data after normalization from
exchange WebSocket feeds. It is the **smallest** family domain (~68
production lines), reflecting its narrow responsibility.

---

## What this domain models

When the `ingest` binary connects to a Binance WebSocket (Spot or
Futures, per configuration) and receives raw trade messages, those
messages are normalized into a uniform `ObservationTrade` representation
and published as `TradeReceivedEvent` on the `OBSERVATION_EVENTS` stream.

Downstream binaries (only `derive` currently) consume these events to
produce evidence.

The intentional minimalism here reflects a deliberate design: keep
observation as thin as possible. The heavy lifting (parsing exchange
formats, reconnect logic, deduplication) lives in adapters
(`internal/adapters/exchanges/binances/`,
`internal/adapters/exchanges/binancef/`), not in the domain.

---

## Core types

### `ObservationTrade`

The normalized trade representation. Fields (from `trade.go`):

| Field | Type | Notes |
|---|---|---|
| `Source` | string | Exchange identifier (e.g., `"binancef"`) |
| `Symbol` | string | Instrument symbol, lowercase (e.g., `"btcusdt"`) |
| `Price` | string | Decimal string — IEEE 754 avoided for precision |
| `Quantity` | string | Decimal string |
| `TradeID` | string | Source-assigned trade identifier |
| `BuyerMaker` | bool | True if the buyer is the maker |
| `Timestamp` | time.Time | Exchange-reported trade time |

Methods:
- `Validate() *problem.Problem` — rejects empty `Source`, `Symbol`,
  `Price`, `Quantity`, `TradeID`, and zero timestamps.
- `DeduplicationKey() string` — returns `"{Source}:{TradeID}"`. Used as
  the JetStream `Msg-Id` for stream-level deduplication.

### `TradeReceivedEvent`

Envelope payload published on `OBSERVATION_EVENTS`. Carries:

- `Metadata events.Metadata` — common envelope metadata
- `Trade ObservationTrade` — the normalized trade

Implements the `DomainEvent` interface
(`EventName() → "market.trade_received"`, `EventMetadata()`).

---

## Event flow

### Streams

- **Writer:** `ingest` binary
- **Stream:** `OBSERVATION_EVENTS`
- **Consumer:** `derive` (via `derive-observation` durable)

There is exactly one event type: `TradeReceivedEvent`
(`events.Name = "market.trade_received"`).

### Subject

The subject filter used by the `derive-observation` consumer matches
trades scoped by source/symbol. The canonical NATS subject in code is
`observation.events.market.trade` (single observation event; partition
key is encoded in the message subject suffix). For the exact form,
consult `internal/adapters/nats/natsobservation/registry.go`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsobservation/` | Stream + publisher + consumer spec for `derive-observation` |
| Application | _none_ in `internal/application/observation/` | Producers (samplers that consume observations and emit evidence) live under `internal/application/derive/` |
| Exchange | `internal/adapters/exchanges/binances/`, `binancef/` | Binance Spot and Futures WebSocket clients with reconnect, parsing |
| ClickHouse | _none_ | Observations are not persisted analytically |

Note the absence of an `internal/application/observation/` package.
Observation has no use cases beyond "publish what arrived". The
consumers (samplers that read observations and emit evidence) live in
the derive application package (`internal/application/derive/sampler.go`,
`internal/application/derive/volume_sampler.go`,
`internal/application/derive/trade_burst_sampler.go`) because
derivation is the consumer's responsibility.

---

## HTTP surface

Observation has **no dedicated HTTP endpoints**. The data flows through
the stream mesh and is queryable downstream via evidence/signal/decision
endpoints.

---

## Known anomalies and patterns

### Absence of operational projection

Unlike most family domains, observation does **not** have an
`OBSERVATION_LATEST` KV bucket. Latest values are not directly
queryable through gateway — only via downstream derivations (candles
in `CANDLE_LATEST`, etc.).

This is by design: raw trade-by-trade values are too noisy to project
operationally. Aggregations are the useful operational surface.

### Smallest domain in the system

68 production LOC + 107 test LOC is the floor among family domains.
If a new family domain looks smaller than this, it likely doesn't need
to be a domain at all — consider folding into an existing one or
adding only the adapter.

### Decimal strings instead of float64

`Price` and `Quantity` are stored as decimal strings, not `float64`,
to avoid IEEE 754 precision loss across the pipeline. Downstream code
must parse as needed; the domain does not perform arithmetic on these
values.

---

## Reading further

| If you want | Go to |
|---|---|
| Where observations are produced | `internal/adapters/exchanges/binances/` |
| Where observations are consumed | `internal/application/derive/` |
| The derived data | [evidence.md](evidence.md) |
| Stream and consumer durables | [`../RUNTIME.md`](../RUNTIME.md) |
