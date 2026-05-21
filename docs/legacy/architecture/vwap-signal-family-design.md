# VWAP Signal Family Design

**Stage:** S285 — Signal Evolution Wave
**Family:** `vwap` (Volume Weighted Average Price)
**Layer:** Signal · Tier 1
**Date:** 2026-03-21

---

## Purpose

VWAP measures the average price of an asset weighted by trading volume over a rolling window. Unlike momentum indicators (MACD) or volatility indicators (Bollinger), VWAP combines price and volume to reveal where institutional interest has been concentrated.

VWAP was selected as the second Signal Evolution Wave family specifically because it pressures the architecture with a fundamentally different input shape: it requires **both close price and volume** from candles, while all prior signal families consume only close price.

## Indicator Specification

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Period | 20 candles | Rolling window, consistent with Bollinger default |
| Input | Close price + Volume | First signal to require multi-field evidence consumption |
| Warm-up | 20 candles | Matches period; first signal at candle 20 |
| Output value | Deviation ratio: `(close − VWAP) / VWAP` | Normalized, direction-agnostic |
| Precision | 6 decimal places | Higher precision than other signals due to small ratio values |

### Formula

```
VWAP = Σ(close_i × volume_i) / Σ(volume_i)    for i in rolling window
deviation = (close − VWAP) / VWAP
```

### Output Semantics

- **Positive deviation** → price above VWAP → potential resistance / overextension
- **Negative deviation** → price below VWAP → potential support / undervaluation
- **Zero deviation** → price at VWAP → equilibrium
- **Zero total volume** → deviation = 0 (degenerate case, safe default)

## Architectural Differentiation from MACD

| Dimension | MACD | VWAP |
|-----------|------|------|
| Input shape | Close price only | Close price + Volume |
| Warm-up phases | Two-phase (EMA seed + signal seed) | Single-phase (rolling window fill) |
| State model | Stateful EMAs (cumulative) | Rolling window (bounded memory) |
| Output nature | Momentum histogram | Price-relative deviation ratio |
| Method signature | `AddClose(price, ts)` | `AddCandle(close, volume, ts)` |
| Edge case | Constant price → zero | Zero volume → zero deviation |

This differentiation proves that the signal family contract is flexible enough to support heterogeneous evidence consumption without requiring a forced common interface.

## Domain Integration

VWAP uses the canonical `signal.Signal` domain type shared by all signal families:
- `Type`: `"vwap"`
- `Value`: deviation ratio as decimal string
- `Metadata`: `period`, `vwap`, `close`, `total_volume`, `deviation`
- Evidence dependency: `candle` (same as all other signal families)

No new domain types, streams, or tables are needed. VWAP publishes to the shared `SIGNAL_EVENTS` stream and persists to the shared `signals` ClickHouse table.

## NATS Topology

| Contract | Value |
|----------|-------|
| Publish subject | `signal.events.vwap.generated.{source}.{symbol}.{timeframe}` |
| Event type | `signal.events.v1.vwap_generated` |
| Stream | `SIGNAL_EVENTS` (shared) |
| Writer durable | `writer-signal-vwap` |
| Store durable | `store-signal-vwap` |
| Latest query | `signal.query.vwap.latest` |

## Multi-Symbol Isolation

Each `VWAPSampler` instance is scoped to `(source, symbol, timeframe)`. JetStream subject hierarchy ensures stream-level isolation. Partition keys and deduplication keys are structurally distinct across symbols — validated in tests.
