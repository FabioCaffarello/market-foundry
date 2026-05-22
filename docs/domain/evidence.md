# evidence — Aggregated derivations

The `evidence` domain models the first level of aggregation from raw
observations. Where `observation` is trade-by-trade, `evidence` is
time-windowed and structural: candles, volumes, trade bursts.

These aggregations are the foundation for all downstream signals,
decisions, and strategies.

---

## What this domain models

Three sub-types of evidence, all derived from observation events:

| Evidence type | What it captures | Why it matters |
|---|---|---|
| **Candle** | OHLCV (Open/High/Low/Close/Volume) over a time window | Universal market-data primitive; foundation for most indicators |
| **Volume** | Aggregated traded volume in a window | Liquidity proxy, surge detection |
| **TradeBurst** | Notable concentration of trades in a sub-second window | Microstructure event, momentum trigger |

Each is parameterized by a partition key:

```
{source}.{symbol}.{timeframe}
```

For example, `binance_spot.btcusdt.60` (BTC/USDT spot, 1-minute candles).

---

## Core types

### Evidence types (in `internal/domain/evidence/`)

- **`EvidenceCandle`** (`candle.go`): OHLCV over a window;
  `Validate() *problem.Problem`.
- **`EvidenceVolume`** (`volume.go`): aggregated volume in a window;
  `Validate() *problem.Problem`.
- **`EvidenceTradeBurst`** (`trade_burst.go`): trade-burst metrics
  (count, density) within a sub-second window;
  `Validate() *problem.Problem`.

### Event types (in `events.go`)

- **`CandleSampledEvent`** → `events.Name = "EventCandleSampled"`
- **`VolumeSampledEvent`** → `events.Name = "EventVolumeSampled"`
- **`TradeBurstSampledEvent`** → `events.Name = "EventTradeBurstSampled"`

Each event wraps the corresponding evidence struct plus envelope
metadata and implements the `DomainEvent` interface.

All three evidence types follow the canonical pattern
(`Validate() *problem.Problem`).

---

## Event flow

### Streams

- **Writer:** `derive` binary
- **Stream:** `EVIDENCE_EVENTS`
- **Consumers:** multiple, with **per-type filtering**:

| Consumer durable | Owner | Filter subject | Purpose |
|---|---|---|---|
| `store-candle` | store | `evidence.events.candle.sampled.>` | KV projection (CANDLE_LATEST + CANDLE_HISTORY) |
| `store-volume` | store | `evidence.events.volume.sampled.>` | KV projection (VOLUME_LATEST) |
| `store-trade-burst` | store | `evidence.events.tradeburst.sampled.>` | KV projection (TRADE_BURST_LATEST) |
| `writer-candle` | writer | `evidence.events.candle.sampled.>` | ClickHouse persistence (`evidence_candles` table) |

**Asymmetry:** writer has only `writer-candle`. Volumes and trade
bursts are **not** persisted to ClickHouse — they exist as
operational projections only.

### Event subjects

```
evidence.events.candle.sampled.{source}.{symbol}.{timeframe}
evidence.events.volume.sampled.{source}.{symbol}.{timeframe}
evidence.events.tradeburst.sampled.{source}.{symbol}.{timeframe}
```

### Query subjects (request/reply, served by store)

```
evidence.query.candle.latest
evidence.query.candle.history
evidence.query.volume.latest
evidence.query.tradeburst.latest
```

Only candle has a `history` query (matching the only HISTORY KV bucket).

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsevidence/` | Stream, publisher, 3 consumer specs, 3 KV stores, price source |
| Application (producer) | `internal/application/derive/` | Samplers: `sampler.go` (candle), `volume_sampler.go`, `trade_burst_sampler.go` |
| Application (reader) | `internal/application/evidenceclient/` | Read-side use cases (`get_latest_candle`, `get_candle_history`, `get_latest_volume`, `get_latest_trade_burst`) plus contracts |
| ClickHouse | `internal/adapters/clickhouse/candle_reader.go` (+ `writerpipeline/support.go`) | `evidence_candles` table for analytical reads |

The split between producer (`derive/`) and reader (`evidenceclient/`)
is the standard shape for family domains that need both write-side
derivation and read-side query serving.

### KV buckets

Four KV buckets total (verified in
`internal/adapters/nats/natsevidence/*kv_store.go`):

| Bucket | Type | Owner |
|---|---|---|
| `CANDLE_LATEST` | Latest candle per partition | store |
| `CANDLE_HISTORY` | Bounded recent history of candles | store |
| `VOLUME_LATEST` | Latest volume per partition | store |
| `TRADE_BURST_LATEST` | Latest trade burst per partition | store |

There is **no** `VOLUME_HISTORY` or `TRADE_BURST_HISTORY` bucket.
Historical volume and trade-burst data is not retained at all (neither
in KV nor in ClickHouse).

---

## HTTP surface

4 routes documented in [`../HTTP-API.md`](../HTTP-API.md) → "Evidence":

- `GET /evidence/candles/latest` — KV-backed (`CANDLE_LATEST`), low latency
- `GET /evidence/candles/history` — KV-backed (`CANDLE_HISTORY`), bounded window
- `GET /evidence/tradeburst/latest` — KV-backed (`TRADE_BURST_LATEST`)
- `GET /evidence/volume/latest` — KV-backed (`VOLUME_LATEST`)

The analytical group also exposes `GET /analytical/evidence/candles`
for ClickHouse-backed historical reads with arbitrary time ranges.

The asymmetric coverage (only candle has a `history` endpoint and a
ClickHouse table) reflects the relative analytical importance of
candles vs. volume/tradebursts.

---

## Known anomalies and patterns

### Application package split

Evidence has no `internal/application/evidence/` directory.
- Production (sampling from observation) lives in
  `internal/application/derive/`.
- Read-side queries live in `internal/application/evidenceclient/`.

This split is intentional: production is owned by the binary that does
the derivation (`derive`), not by the domain that holds the type.

### Multiple sub-types in one domain

Evidence is unusual in carrying 3 distinct sub-types (Candle, Volume,
TradeBurst) with similar but separate event flows and per-type KV
stores. Most other domains have one principal type. This reflects
evidence's role as the "aggregation layer" — multiple kinds of
aggregation, all foundational.

### Persistence asymmetry by sub-type

- Candle: KV (LATEST + HISTORY) + ClickHouse + HTTP (latest + history + analytical)
- Volume: KV (LATEST only) + HTTP (latest only)
- TradeBurst: KV (LATEST only) + HTTP (latest only)

If you need historical volume or trade-burst data, it is not currently
available anywhere in the system — neither operationally nor
analytically. Adding it would require a new ClickHouse table and a new
writer durable.

### `price_source.go` in the NATS adapter

The natsevidence adapter contains a `price_source.go` (S387) that
provides a reference price source used by other components. This is
unusual for an adapter package but reflects evidence's role as a
"first source of truth" for derived prices.

---

## Reading further

| If you want | Go to |
|---|---|
| How evidence is derived from observations | `internal/application/derive/` |
| Where candle history is stored | `internal/adapters/clickhouse/candle_reader.go` |
| The next layer of derivation | [signal.md](signal.md) |
| HTTP endpoint contracts | [`../HTTP-API.md`](../HTTP-API.md) |
