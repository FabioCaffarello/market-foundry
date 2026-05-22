# signal — Indicator computations

The `signal` domain models technical indicator outputs computed from
evidence. Each signal is a deterministic function of recent evidence
(typically candles) producing a numeric or categorical value that
strategies and decisions consume.

---

## What this domain models

A signal is a discrete computation result: at moment T for partition
P = (source, symbol, timeframe), the signal value is V. The value
can be a float (e.g., EMA price), a category (e.g., RSI overbought),
or a richer structure (e.g., Bollinger Band envelope).

Signals are derived **pure** from evidence — they have no I/O, no
dependency on configctl beyond their activation binding, and no
side effects. Their entire job is to take a candle stream and emit
a `SignalGeneratedEvent`.

Each signal type has its own sampler in `internal/application/signal/`
following the `FamilyProcessor` pattern: declarative registration,
spawned per (signal-type × source × symbol × timeframe) tuple.

---

## Core types

- `Signal` — the computed value, with metadata (Source, Symbol,
  Timeframe, Timestamp, Type, Value).
- `SignalGeneratedEvent` — the envelope payload carrying a `Signal`.

Both follow the canonical `Validate() *problem.Problem` pattern.

---

## Concrete signal types

The system implements 6 indicator types. Each has its own sampler
file in `internal/application/signal/`:

| Type identifier | Sampler file | Purpose |
|---|---|---|
| `rsi` | `rsi_sampler.go` | Relative Strength Index (overbought/oversold) |
| `ema_crossover` | `ema_crossover_sampler.go` | Exponential moving average direction crossover |
| `macd` | `macd_sampler.go` | Moving Average Convergence/Divergence |
| `bollinger` | `bollinger_sampler.go` | Volatility band envelope |
| `atr` | `atr_sampler.go` | Average True Range (volatility) |
| `vwap` | `vwap_sampler.go` | Volume-Weighted Average Price |

The `{type}` identifier above is the value the `/signal/:type/latest`
HTTP route accepts and the value embedded in the NATS subject.

---

## Event flow

- **Writer:** `derive` binary
- **Stream:** `SIGNAL_EVENTS`
- **Consumers:**
  - `store` — per-type KV projection (only for types with a `_LATEST` bucket; 2 of 6 today)
  - `writer` — per-type ClickHouse persistence; one durable per type
    (`writer-signal-rsi`, `writer-signal-ema`, `writer-signal-bollinger`,
    `writer-signal-macd`, `writer-signal-vwap`, `writer-signal-atr`)

### Subject taxonomy

```
signal.events.{type}.generated
signal.query.{type}.latest
```

Note the order: the **type** comes after `events`, the verb (`generated`)
after the type. For example:
- `signal.events.rsi.generated`
- `signal.events.ema_crossover.generated`
- `signal.query.rsi.latest`

The partition key (source, symbol, timeframe) is encoded in the
subject suffix appended by the publisher. For exact form, see
`internal/adapters/nats/natssignal/registry.go`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natssignal/` | Stream + publisher + 6 store-side consumer specs + 6 writer-side per-type durables |
| Application (producer) | `internal/application/signal/` | 6 per-type samplers, FamilyProcessor pattern |
| Application (reader) | `internal/application/signalclient/` | Read-side client used by gateway |
| ClickHouse | `internal/adapters/clickhouse/signal_reader.go` | `signals` table; all 6 signal types persist here |

---

## KV bucket coverage

Not every signal type has a `_LATEST` KV bucket. Coverage today (verified
in `internal/adapters/nats/natssignal/kv_store.go`):

| Type | KV `_LATEST` bucket | Operational read (`/signal/:type/latest`) |
|---|---|---|
| `rsi` | `SIGNAL_RSI_LATEST` ✓ | works |
| `ema_crossover` | `SIGNAL_EMA_CROSSOVER_LATEST` ✓ | works |
| `macd` | — | returns 404 |
| `bollinger` | — | returns 404 |
| `atr` | — | returns 404 |
| `vwap` | — | returns 404 |

The 4 types without a `_LATEST` bucket are operationally
"analytical-only" — their values persist in ClickHouse (queryable via
`/analytical/signal/history`) but cannot be queried operationally
via `/signal/:type/latest`.

This is the **G2 gap** documented in [`../RESUMPTION.md`](../RESUMPTION.md).
It is unclear whether this is intentional design (some signals are
analytical-only) or oversight. The signal events flow through the
stream regardless; the gap is only in the operational read path.

---

## HTTP surface

One operational route: `GET /signal/:type/latest` (see
[`../HTTP-API.md`](../HTTP-API.md) → Domain latest group).

The `:type` path parameter accepts the type identifier from the
"Concrete signal types" table. If the requested type has no KV
`_LATEST` bucket, the endpoint returns 404 even when the type exists
and is flowing through the stream.

Analytical reads (history) for signals are available at
`GET /analytical/signal/history` with time-range and `type` query
params, served by writer's ClickHouse reader.

---

## Known anomalies and patterns

Follows canonical FamilyProcessor + Pipeline + FamilyDeps patterns
documented in [`../ARCHITECTURE.md`](../ARCHITECTURE.md).

The KV coverage gap noted above is the main domain-specific quirk.
A secondary minor inconsistency: the store-side consumer for
`ema_crossover` is named `store-signal-ema-crossover` (with hyphen)
while the writer-side equivalent is named `writer-signal-ema` (no
suffix). Same underlying signal type, different durable names. See
[`../RUNTIME.md`](../RUNTIME.md) → "Consumer durables" for the full
naming map.

---

## Reading further

| If you want | Go to |
|---|---|
| Producer code (per-type samplers) | `internal/application/signal/` |
| Reader code (gateway-side) | `internal/application/signalclient/` |
| The next layer of derivation | [decision.md](decision.md) |
| KV gap context | [`../RESUMPTION.md`](../RESUMPTION.md) → G2 |
| HTTP endpoints | [`../HTTP-API.md`](../HTTP-API.md) |
