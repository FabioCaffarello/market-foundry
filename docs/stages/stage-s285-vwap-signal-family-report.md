# Stage S285 — VWAP Signal Family Report

**Wave:** Signal Evolution (S283 charter)
**Family:** `vwap` — Volume Weighted Average Price
**Date:** 2026-03-21
**Status:** COMPLETE

---

## Executive Summary

S285 delivers the VWAP signal family as the second family of the Signal Evolution Wave. VWAP was chosen specifically to pressure the architecture with a semantically and structurally different indicator from MACD (S284). Where MACD consumes only close prices via a two-phase EMA warm-up to produce momentum histograms, VWAP consumes both close price and volume via a rolling window to produce price-relative deviation ratios.

The delivery confirms that the signal family architecture supports heterogeneous evidence consumption without forced abstraction, degradation, or special-casing.

## Design Summary

- **Indicator:** Rolling VWAP = Σ(close × volume) / Σ(volume) over 20-candle window
- **Output:** Deviation ratio `(close − VWAP) / VWAP` — positive means above VWAP, negative means below
- **Input shape:** `AddCandle(close, volume, ts)` — first family to require volume
- **Warm-up:** 20 candles (single-phase, rolling window fill)
- **Domain type:** Canonical `signal.Signal` — no new types
- **NATS:** Shared `SIGNAL_EVENTS` stream, unique `writer-signal-vwap` durable

## Artifacts Delivered

| Artifact | Path |
|----------|------|
| Sampler implementation | `internal/application/signal/vwap_sampler.go` |
| Behavioral tests (13) | `internal/application/signal/vwap_sampler_test.go` |
| Codegen YAML spec | `codegen/families/vwap.yaml` |
| Golden snapshot (consumer) | `codegen/golden-snapshots/vwap/consumer_spec.go.golden` |
| Golden snapshot (pipeline) | `codegen/golden-snapshots/vwap/pipeline_entry.go.golden` |
| Registry integration | `internal/adapters/nats/natssignal/registry.go` |
| Pipeline integration | `cmd/writer/pipeline.go` |
| Settings registration | `internal/shared/settings/schema.go` |
| Codegen manifest | `codegen/integrated.yaml` |
| Design document | `docs/architecture/vwap-signal-family-design.md` |
| Acceptance document | `docs/architecture/vwap-signal-family-implementation-and-acceptance.md` |

## Architectural Findings

### 1. Input Shape Heterogeneity (confirmed)
VWAP proves that signal families are not constrained to a single input interface. The `AddCandle(close, volume, ts)` signature coexists with `AddClose(price, ts)` without requiring a common interface. This is structurally correct — forcing a common interface would either over-generalize (passing unused fields) or under-specify (losing type safety).

### 2. VWAP vs MACD — Architectural Pressure Matrix

| Dimension | MACD (S284) | VWAP (S285) | Pressure applied |
|-----------|-------------|-------------|------------------|
| Evidence fields | Close only | Close + Volume | Multi-field consumption |
| Warm-up model | Two-phase (26+9=34) | Single-phase (20) | Phase diversity |
| State shape | Cumulative EMAs | Bounded rolling window | Memory model |
| Output semantics | Momentum histogram | Price-relative ratio | Semantic diversity |
| Edge case class | Constant price → zero | Zero volume → degenerate | New degeneracy class |
| Method signature | `AddClose` | `AddCandle` | Interface flexibility |

### 3. Codegen Generality
VWAP integrates through the same codegen pipeline as all other signal families: YAML spec → golden snapshots → integrated markers. The `VWAP` abbreviation was added to `knownAbbreviations` in `spec.go` to produce correct PascalCase (`WriterVWAPSignalConsumer`). No codegen structural changes were needed.

### 4. Signal Family Count
With VWAP, the signal layer now has **6 families**: RSI, EMA, EMA Crossover, Bollinger, MACD, VWAP. Total codegen-integrated families across all layers: **17** (exceeding the S283 charter target of ≥15).

## Test Results

```
=== RUN   TestVWAPSampler_WarmUp             --- PASS
=== RUN   TestVWAPSampler_ConstantPrices      --- PASS
=== RUN   TestVWAPSampler_PriceAboveVWAP      --- PASS
=== RUN   TestVWAPSampler_PriceBelowVWAP      --- PASS
=== RUN   TestVWAPSampler_VolumeWeighting     --- PASS
=== RUN   TestVWAPSampler_ZeroVolume          --- PASS
=== RUN   TestVWAPSampler_Metadata            --- PASS
=== RUN   TestVWAPSampler_InvalidPrice        --- PASS
=== RUN   TestVWAPSampler_InvalidVolume       --- PASS
=== RUN   TestVWAPSampler_Validate            --- PASS
=== RUN   TestVWAPSampler_MultiSymbol         --- PASS
=== RUN   TestVWAPSampler_ContinuousProduction --- PASS
=== RUN   TestVWAPSampler_RollingWindow       --- PASS
PASS — 13/13 tests, zero regressions in signal and settings suites
```

## Regression Check

- `go test ./internal/application/signal/...` — PASS (all existing + new tests)
- `go test ./internal/shared/settings/...` — PASS (family count updated 5→6)
- `go build ./cmd/writer/...` — PASS (pipeline compiles with VWAP entry)

## S286 Preparation Recommendations

The Signal Evolution Wave charter (S283) defines 4 families:
1. **MACD** — S284 COMPLETE
2. **VWAP** — S285 COMPLETE
3. **ATR** (Average True Range) — S286 candidate

ATR would add further pressure:
- Requires **high, low, close** — three evidence fields, expanding the input shape beyond VWAP's two
- True Range calculation: `max(high−low, |high−prevClose|, |low−prevClose|)` — needs previous candle state
- Output is an absolute volatility measure (not a ratio), testing yet another output semantic
- Warm-up: N+1 candles (Wilder's smoothing, similar to RSI)

After ATR (S286), the charter calls for one decision family (Bollinger Squeeze, S287), completing the wave.

## Verdict

**S285 COMPLETE** — VWAP delivered as a coherent, well-tested signal family with fundamentally different semantics from MACD. The architecture supports 6 signal families without degradation. The wave is on track for ATR (S286) as the next family.
