# Stage S286 — ATR Signal Family Report

**Wave:** Signal Evolution (S283–S287)
**Position:** Third of three new signal families
**Status:** Complete
**Predecessor:** S285 (VWAP Signal Family)

## 1. Executive Summary

S286 delivers the ATR (Average True Range) signal family as the consolidation proof for the Signal Evolution Wave. ATR introduces a third structurally distinct input interface (high/low/close — 3 price fields), a pure volatility semantic (direction-agnostic), and Wilder smoothing as a memory model. All integration points — codegen, NATS wiring, settings validation, writer pipeline — absorbed ATR without structural changes, confirming wave robustness.

Total integrated families after S286: **19** (was 18 after S285).
Total signal families: **7** (rsi, ema, ema_crossover, bollinger, macd, vwap, atr).

## 2. What ATR Teaches About the Mesh

### 2.1 Three Distinct Interfaces — No Forced Abstraction

| Family | Method | Evidence consumed |
|---|---|---|
| MACD | `AddClose(close, ts)` | 1 field |
| VWAP | `AddCandle(close, volume, ts)` | 2 fields |
| ATR | `AddCandle(high, low, close, ts)` | 3 fields |

The mesh handles heterogeneous evidence consumption without adapters, wrappers, or a common sampler interface. This is not accidental — it's a design property that scales.

### 2.2 Wilder Smoothing vs. EMA vs. SMA

ATR uses Wilder smoothing (`ATR = (prev × (N-1) + TR) / N`), which is distinct from the exponential smoothing in MACD and the simple moving average in Bollinger. The mesh does not constrain algorithm choice — each family owns its computation entirely.

### 2.3 Warm-up Semantics Vary

| Family | Warm-up | Phases |
|---|---|---|
| Bollinger | 20 candles | Single-phase (fill window) |
| MACD | 34 candles | Two-phase (EMA seed + signal seed) |
| VWAP | 20 candles | Single-phase (fill window) |
| ATR | 15 candles | Two-phase (establish prevClose + SMA seed) |

No family constrains another's warm-up behavior.

### 2.4 Codegen Pipeline Confirmed Stable

Three consecutive family additions (S284, S285, S286) with zero structural changes to the codegen pipeline. The pattern is:
1. Write YAML spec
2. Generate golden snapshots
3. Insert integration markers
4. Register abbreviation (if non-standard casing)
5. Update integrated.yaml manifest

This 5-step recipe is now proven across 8 signal-layer families.

## 3. Artifacts Delivered

### Created
- `internal/application/signal/atr_sampler.go` — ATR computation
- `internal/application/signal/atr_sampler_test.go` — 11 tests, 14 sub-tests
- `codegen/families/atr.yaml` — codegen spec
- `codegen/golden-snapshots/atr/consumer_spec.go.golden` — writer consumer
- `codegen/golden-snapshots/atr/pipeline_entry.go.golden` — pipeline entry
- `docs/architecture/atr-signal-family-design.md` — design document
- `docs/architecture/atr-signal-family-implementation-and-acceptance.md` — acceptance record

### Modified
- `codegen/spec.go` — ATR abbreviation
- `codegen/integrated.yaml` — 2 slice entries
- `internal/adapters/nats/natssignal/registry.go` — ATR event/control specs + consumers
- `cmd/writer/pipeline.go` — ATR pipeline entry
- `internal/shared/settings/schema.go` — ATR in known families + dependency map
- `internal/shared/settings/settings_test.go` — family count update

## 4. Test Results

All 11 ATR tests pass (14 sub-tests including trueRange unit tests).
All existing tests pass — zero regressions.

| Test | Status |
|---|---|
| `TestATRSampler_WarmUp` | Pass |
| `TestATRSampler_HighVolatility` | Pass |
| `TestATRSampler_LowVolatility` | Pass |
| `TestATRSampler_ConstantPrices` | Pass |
| `TestATRSampler_GapUp` | Pass |
| `TestATRSampler_Metadata` | Pass |
| `TestATRSampler_InvalidPrice` | Pass |
| `TestATRSampler_Validate` | Pass |
| `TestATRSampler_MultiSymbol` | Pass |
| `TestATRSampler_ContinuousProduction` | Pass |
| `TestATRSampler_TrueRangeCalculation` | Pass (4 sub-tests) |

## 5. Patterns Confirmed

| Pattern | Evidence |
|---|---|
| Heterogeneous input interfaces scale | 3 distinct interfaces, no common abstraction |
| Codegen pipeline is family-agnostic | Zero structural changes across 3 additions |
| Shared signals table absorbs any signal type | ATR metadata in JSON column, no migration |
| Settings validation scales linearly | Add entry to 2 maps, update 1 test |
| Registry pattern is mechanical | Add 2 specs + switch case + 2 consumers |
| Wave discipline holds | No new decision families, no broad refactors |

## 6. Limits Observed

| Limit | Assessment |
|---|---|
| No runtime sampler registry | Each binary manually wires its sampler — acceptable at current family count |
| Evidence dependency is uniform | All signal families depend on `"candle"` — dependency map adds no value yet |
| No cross-family signal composition | Signals are independent; no family reads another family's output |
| Warm-up window is opaque to downstream | Consumers cannot know when a signal family becomes "ready" |

## 7. Guard Rails Compliance

| Rule | Status |
|---|---|
| No new decision family | Compliant |
| No broad transversal refactor | Compliant — only ATR-specific additions |
| No supergeneric abstractions | Compliant — no common sampler interface introduced |
| Disciplined value delivery | Compliant — ATR is a complete, tested, documented family |

## 8. Recommendations for S287

S287 should close the Signal Evolution Wave with a gate assessment:

1. **Wave gate ceremony**: Assess whether 3 new families (MACD, VWAP, ATR) prove the mesh is ready for the next wave, or whether consolidation is needed first.
2. **Cross-family consistency audit**: Verify metadata key naming conventions are consistent across all 7 signal families.
3. **Codegen equivalence check**: Run `codegen-equivalence-check.sh` across all 19 integrated families to confirm no drift.
4. **Warm-up observability**: Consider whether downstream consumers need a "sampler ready" signal — defer implementation unless there's real pressure.
5. **Evidence dependency enrichment**: MACD depends on `candle` but only uses `close`; VWAP uses `close` + `volume`; ATR uses `high` + `low` + `close`. The dependency map could be enriched to field-level granularity, but only if there's a consumer for that information.
