# MACD Signal Family — Design

**Stage:** S284
**Family:** macd
**Layer:** signal
**Tier:** 1

## 1. Purpose

MACD (Moving Average Convergence Divergence) is a momentum indicator that measures
the relationship between two EMAs of close prices and uses their divergence to
detect trend strength and direction changes. It is the first family delivered
in the Signal Evolution Wave (S283–S288).

## 2. Domain Shape

The MACD family produces signals of type `"macd"` conforming to the canonical
`signal.Signal` domain struct. No new domain types are introduced.

| Field     | Value                                                    |
|-----------|----------------------------------------------------------|
| Type      | `"macd"`                                                 |
| Value     | Histogram (MACD line − signal line) as decimal string    |
| Source    | Exchange identifier (e.g. `"binancef"`)                  |
| Symbol    | Trading pair in lowercase (e.g. `"btcusdt"`)             |
| Timeframe | Candle duration in seconds (e.g. `300`)                  |
| Final     | `true` (computed from finalized candles only)             |

### Metadata Fields

| Key             | Description                                 |
|-----------------|---------------------------------------------|
| `fast_period`   | Fast EMA period (default: 12)               |
| `slow_period`   | Slow EMA period (default: 26)               |
| `signal_period` | Signal line EMA period (default: 9)         |
| `fast_ema`      | Current fast EMA value                      |
| `slow_ema`      | Current slow EMA value                      |
| `macd_line`     | MACD line = fast EMA − slow EMA             |
| `signal_line`   | Signal line = EMA(signal_period) of MACD    |
| `histogram`     | Histogram = MACD line − signal line         |

## 3. Indicator Computation

Standard MACD (12, 26, 9):

1. **Fast EMA** — 12-period EMA of close prices.
2. **Slow EMA** — 26-period EMA of close prices.
3. **MACD Line** — `fast EMA − slow EMA`.
4. **Signal Line** — 9-period EMA of the MACD line.
5. **Histogram** — `MACD line − signal line` (primary output value).

### Warm-up

- **Phase 1 (candles 1–26):** Accumulate `slowPeriod` close prices. Seed both
  EMAs with SMA over their respective windows.
- **Phase 2 (candles 27–34):** Accumulate `signalPeriod` MACD line values.
  Seed signal EMA with SMA of those values.
- **First output at candle 34.** After warm-up, every candle produces a signal.

## 4. NATS Contracts

| Contract          | Value                                           |
|-------------------|-------------------------------------------------|
| Subject (publish) | `signal.events.macd.generated.{source}.{symbol}.{timeframe}` |
| Event type        | `signal.events.v1.macd_generated`               |
| Stream            | `SIGNAL_EVENTS`                                 |
| Writer consumer   | `writer-signal-macd` (codegen-governed)         |
| Store consumer    | `store-signal-macd` (manual-owned)              |
| KV bucket         | `SIGNAL_MACD_LATEST` (via store binary)         |
| Query subject     | `signal.query.macd.latest`                      |

## 5. Writer Pipeline

MACD shares the same `signals` ClickHouse table as all other signal families.
No schema changes required — the existing column set covers all MACD fields
through the `metadata` JSON column.

## 6. Multi-Symbol Support

Each `MACDSampler` instance is scoped to a single `(source, symbol, timeframe)`.
The derive actor topology creates one sampler per stream partition. JetStream
subject hierarchy (`signal.events.macd.generated.{source}.{symbol}.{timeframe}`)
ensures isolation. Partition and deduplication keys are derived from the Signal
domain type and apply identically to MACD.

## 7. Evidence Dependency

MACD depends solely on candle evidence (close prices). No new evidence fields
or types are required.

## 8. Codegen Governance

Two artifacts are codegen-governed (consumer_spec + pipeline_entry):

- Spec: `codegen/families/macd.yaml`
- Golden: `codegen/golden-snapshots/macd/consumer_spec.go.golden`
- Golden: `codegen/golden-snapshots/macd/pipeline_entry.go.golden`
- Integrated markers in `natssignal/registry.go` and `cmd/writer/pipeline.go`

The `MACD` abbreviation was added to `codegen/spec.go` `knownAbbreviations`
to produce correct PascalCase function names (e.g. `WriterMACDSignalConsumer`).

## 9. Downstream Consumption

MACD signals are available for:
- **Decision evaluators** — histogram polarity/crossovers can trigger decision events.
- **Strategy resolvers** — MACD momentum can inform entry/exit strategies.
- **Analytical queries** — ClickHouse signal reader works for all types including MACD.
- **Gateway API** — `GET /signal/macd/latest` (via existing SignalWebHandler).
