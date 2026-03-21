# VWAP Signal Family — Implementation and Acceptance

**Stage:** S285 — Signal Evolution Wave
**Family:** `vwap`
**Date:** 2026-03-21

---

## Delivery Checklist (per S283 Charter)

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Codegen YAML spec | PASS | `codegen/families/vwap.yaml` |
| 2 | Golden snapshots | PASS | `codegen/golden-snapshots/vwap/consumer_spec.go.golden`, `pipeline_entry.go.golden` |
| 3 | Integrated markers in target files | PASS | `registry.go` + `pipeline.go` codegen markers |
| 4 | Application-layer implementation | PASS | `internal/application/signal/vwap_sampler.go` |
| 5 | Behavioral tests | PASS | 13 tests covering warm-up, volume weighting, edge cases, multi-symbol |
| 6 | ClickHouse schema integration | PASS | Uses shared `signals` table — no schema change needed |
| 7 | Equivalence check compatibility | PASS | Golden snapshots match codegen markers |
| 8 | Build + test green | PASS | `go test ./internal/application/signal/...` + `go build ./cmd/writer/...` |

## Test Coverage

| Test | What it validates |
|------|-------------------|
| `TestVWAPSampler_WarmUp` | 20-candle warm-up, correct field values on first signal |
| `TestVWAPSampler_ConstantPrices` | Deviation ≈ 0 when price is flat |
| `TestVWAPSampler_PriceAboveVWAP` | Positive deviation after sustained rise |
| `TestVWAPSampler_PriceBelowVWAP` | Negative deviation after sustained fall |
| `TestVWAPSampler_VolumeWeighting` | VWAP pulled toward high-volume candles |
| `TestVWAPSampler_ZeroVolume` | Safe zero deviation with zero volume |
| `TestVWAPSampler_Metadata` | All 5 metadata keys present with correct values |
| `TestVWAPSampler_InvalidPrice` | Graceful rejection of non-numeric price |
| `TestVWAPSampler_InvalidVolume` | Graceful rejection of non-numeric volume |
| `TestVWAPSampler_Validate` | Domain-level signal validation passes |
| `TestVWAPSampler_MultiSymbol` | Symbol isolation via partition/deduplication keys |
| `TestVWAPSampler_ContinuousProduction` | Every candle after warm-up produces a signal (21 of 40) |
| `TestVWAPSampler_RollingWindow` | Old prices correctly drop out after window rotation |

## Files Changed

### New files
- `internal/application/signal/vwap_sampler.go` — VWAPSampler implementation
- `internal/application/signal/vwap_sampler_test.go` — 13 behavioral tests
- `codegen/families/vwap.yaml` — codegen family spec
- `codegen/golden-snapshots/vwap/consumer_spec.go.golden` — writer consumer golden
- `codegen/golden-snapshots/vwap/pipeline_entry.go.golden` — pipeline entry golden

### Modified files
- `internal/adapters/nats/natssignal/registry.go` — VWAP registry fields + codegen consumer + store consumer
- `cmd/writer/pipeline.go` — VWAP pipeline entry with codegen markers
- `internal/shared/settings/schema.go` — `vwap` in knownSignalFamilies + signalDependsOnEvidence
- `internal/shared/settings/settings_test.go` — signal family count 5 → 6
- `codegen/spec.go` — `vwap` → `VWAP` in knownAbbreviations
- `codegen/integrated.yaml` — VWAP consumer_spec + pipeline_entry entries (S285)

## Architectural Findings

### Finding 1: Input Shape Heterogeneity
VWAP is the first signal family to require more than close price. Its `AddCandle(close, volume, ts)` signature differs from the `AddClose(price, ts)` used by RSI, EMA, Bollinger, and MACD. This proves that:
- No forced common sampler interface is needed
- Each sampler can define its own input contract
- The evidence consumption model is flexible without premature abstraction

### Finding 2: Rolling Window Convergence
VWAP, Bollinger, and RSI all use rolling windows, but each computes different aggregates (volume-weighted price, standard deviation, gain/loss averages). The pattern is structurally similar but semantically distinct — correctly kept as separate implementations rather than abstracted.

### Finding 3: Precision Requirements
VWAP deviation ratios are typically small values (e.g., 0.003 = 0.3% above VWAP). The family uses 6-decimal precision compared to the 4-decimal standard of other signals. This is a natural consequence of the ratio output — not an architecture issue.

### Finding 4: Zero-Volume Degeneracy
When total volume is zero across the window, VWAP is undefined. The sampler returns deviation=0 as a safe default. This edge case does not exist in price-only families and represents a new class of degeneracy that volume-consuming families must handle.

## Acceptance Verdict

**PASS** — VWAP is delivered as a coherent, well-tested signal family that pressures the architecture with a fundamentally different input shape (close + volume) while demonstrating that the signal contract supports this without degradation or forced abstraction.
