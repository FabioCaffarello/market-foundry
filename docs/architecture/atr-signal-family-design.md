# ATR Signal Family — Design

**Stage:** S286
**Wave:** Signal Evolution (S283–S287)
**Family position:** Third of three new signal families (after MACD S284, VWAP S285)

## 1. Purpose

Average True Range (ATR) measures market volatility by decomposing price movement into True Range and smoothing it over a configurable period. Unlike MACD (momentum) or VWAP (price-relative), ATR provides a pure volatility signal that is direction-agnostic — it measures how much an asset moves, not which way.

ATR is the canonical consolidation family for the Signal Evolution Wave: if the mesh accommodates three structurally distinct families without degradation, the wave is proven robust.

## 2. Semantic Shape

| Property | Value |
|---|---|
| Type | `atr` |
| Input interface | `AddCandle(high, low, close string, ts time.Time)` — heterogeneous (3 price fields) |
| Output value | ATR as decimal string (smoothed average of true ranges) |
| Memory model | Stateful with Wilder smoothing (cumulative, bounded constant memory after warm-up) |
| Warm-up | `period + 1 = 15` candles (1 to establish prevClose, 14 for initial SMA) |
| Default period | 14 (standard Wilder ATR) |

## 3. True Range Definition

For each candle after the first:

```
True Range = max(
    high − low,           // Current candle range
    |high − prevClose|,   // Gap-up detection
    |low  − prevClose|    // Gap-down detection
)
```

This captures intra-candle volatility and inter-candle gaps.

## 4. ATR Smoothing

- **Initial ATR** (candle 15): SMA of first 14 true ranges
- **Subsequent ATR**: Wilder smoothing — `ATR = (prevATR × (period−1) + TR) / period`

Wilder smoothing provides heavier weighting to recent values than a simple moving average while maintaining computational simplicity and bounded memory (constant state: `prevClose`, `atr`, `period`).

## 5. Metadata Contract

| Key | Type | Description |
|---|---|---|
| `period` | int string | Smoothing period (default "14") |
| `atr` | decimal string | Current ATR value |
| `true_range` | decimal string | Current candle's true range |

## 6. Input Interface Decision

ATR requires `high`, `low`, and `close` — three price fields from candle evidence. This is the third distinct interface pattern in the wave:

| Family | Interface | Evidence fields consumed |
|---|---|---|
| MACD (S284) | `AddClose(close, ts)` | 1 price field |
| VWAP (S285) | `AddCandle(close, volume, ts)` | 1 price + 1 volume field |
| ATR (S286) | `AddCandle(high, low, close, ts)` | 3 price fields |

This confirms the wave finding from S285: signal families are NOT constrained to a common interface. Each family consumes exactly the evidence fields its algorithm requires.

## 7. Codegen Integration

ATR follows the established codegen-first pattern:

- `codegen/families/atr.yaml` — family spec
- `codegen/golden-snapshots/atr/consumer_spec.go.golden` — writer consumer
- `codegen/golden-snapshots/atr/pipeline_entry.go.golden` — pipeline entry
- Integration markers in `natssignal/registry.go` and `cmd/writer/pipeline.go`
- Manifest entry in `codegen/integrated.yaml`

No structural changes to the codegen pipeline are required — only abbreviation registration (`"atr": "ATR"` in `codegen/spec.go`).

## 8. NATS Subject Layout

| Purpose | Subject |
|---|---|
| Publish | `signal.events.atr.generated.{source}.{symbol}.{timeframe}` |
| Event type | `signal.events.v1.atr_generated` |
| Writer consumer | `writer-signal-atr` |
| Store consumer | `store-signal-atr` |
| Latest query | `signal.query.atr.latest` |
| KV bucket | `SIGNAL_ATR_LATEST` |

Shared stream: `SIGNAL_EVENTS` (no new streams required).

## 9. ClickHouse Storage

ATR writes to the shared `signals` table. The `type` column distinguishes ATR rows. The `metadata` JSON column carries ATR-specific fields (`period`, `atr`, `true_range`). No schema migration required.
