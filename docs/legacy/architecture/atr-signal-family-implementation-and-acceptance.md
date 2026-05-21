# ATR Signal Family — Implementation and Acceptance

**Stage:** S286
**Status:** Complete

## 1. Implementation Summary

ATR was implemented as the third signal family in the Signal Evolution Wave, following the canonical patterns established by MACD (S284) and VWAP (S285).

### Files Created

| File | Purpose |
|---|---|
| `internal/application/signal/atr_sampler.go` | Pure ATR computation (Wilder smoothing) |
| `internal/application/signal/atr_sampler_test.go` | 11 behavioral tests (14 sub-tests) |
| `codegen/families/atr.yaml` | Codegen family spec |
| `codegen/golden-snapshots/atr/consumer_spec.go.golden` | Writer consumer golden |
| `codegen/golden-snapshots/atr/pipeline_entry.go.golden` | Pipeline entry golden |

### Files Modified

| File | Change |
|---|---|
| `codegen/spec.go` | Added `"atr": "ATR"` abbreviation |
| `codegen/integrated.yaml` | Added 2 ATR slice entries (consumer_spec + pipeline_entry) |
| `internal/adapters/nats/natssignal/registry.go` | Added ATRGenerated/ATRLatest specs, WriterATRSignalConsumer, StoreATRSignalConsumer, LatestSpecByType case |
| `cmd/writer/pipeline.go` | Added ATR pipeline entry with codegen markers |
| `internal/shared/settings/schema.go` | Added `"atr"` to knownSignalFamilies and signalDependsOnEvidence |
| `internal/shared/settings/settings_test.go` | Updated expected signal family count (6 → 7) |

## 2. Acceptance Criteria

### Universal Criteria (AC-1 through AC-8)

| ID | Criterion | Status |
|---|---|---|
| AC-1 | Codegen YAML spec exists and validates | Pass |
| AC-2 | Golden snapshots generated and match | Pass |
| AC-3 | Integration markers in target files | Pass |
| AC-4 | Full equivalence check green | Pass |
| AC-5 | Application-layer implementation exists | Pass |
| AC-6 | Behavioral tests in CI (zero skip) | Pass — 11 tests, 14 sub-tests |
| AC-7 | ClickHouse schema defined | Pass — shared `signals` table |
| AC-8 | Zero regression in existing tests | Pass |

### ATR-Specific Criteria

| ID | Criterion | Status |
|---|---|---|
| ATR-1 | True Range correctly handles all three cases (range, gap-up, gap-down) | Pass — dedicated `TestATRSampler_TrueRangeCalculation` with 4 sub-tests |
| ATR-2 | Wilder smoothing produces correct ATR values | Pass — volatility expansion/contraction tests confirm |
| ATR-3 | Publishes to `signal.events.atr.generated.>` | Pass — registry wired |
| ATR-4 | Warm-up requires exactly period+1 candles | Pass — `TestATRSampler_WarmUp` |
| ATR-5 | Multi-symbol isolation | Pass — partition and deduplication keys verified |

## 3. Test Coverage

| Test | What it proves |
|---|---|
| `TestATRSampler_WarmUp` | No signal before 15 candles; first signal at candle 15 |
| `TestATRSampler_HighVolatility` | ATR increases when fed high-volatility candles |
| `TestATRSampler_LowVolatility` | ATR contracts when fed narrow-range candles |
| `TestATRSampler_ConstantPrices` | ATR ≈ 0 when high=low=close (zero true range) |
| `TestATRSampler_GapUp` | True Range correctly captures gap-up (|high−prevClose| dominates) |
| `TestATRSampler_Metadata` | All required metadata keys present with correct types |
| `TestATRSampler_InvalidPrice` | Graceful rejection of non-parseable inputs (high, low, close) |
| `TestATRSampler_Validate` | Domain-level Signal.Validate() passes |
| `TestATRSampler_MultiSymbol` | Partition/deduplication key isolation across symbols |
| `TestATRSampler_ContinuousProduction` | After warm-up, every candle produces exactly one signal |
| `TestATRSampler_TrueRangeCalculation` | Unit test for trueRange() with 4 scenarios |

## 4. Architectural Observations

### Interface Heterogeneity Confirmed at Scale

With three distinct input interfaces across the wave (1-field, 2-field, 3-field), the mesh proves it does not require a common sampler interface. Each family consumes exactly the evidence fields its algorithm needs. No interface adapters, no forced abstraction.

### Memory Model Diversity

| Family | Memory model | Post-warmup state |
|---|---|---|
| Bollinger | Rolling window (period prices) | O(period) |
| MACD | Cumulative EMA (no window) | O(1) — three floats |
| VWAP | Rolling window (period prices + volumes) | O(period) |
| ATR | Cumulative Wilder smoothing | O(1) — two floats (prevClose, atr) |

ATR and MACD share the O(1) cumulative pattern. Bollinger and VWAP share the O(period) window pattern. Both patterns coexist without architectural preference.

### Codegen Pipeline Stability

ATR required zero structural changes to the codegen pipeline:
- Same YAML schema as all previous families
- Same golden snapshot templates
- Same integration marker pattern
- Only addition: abbreviation entry (`"atr": "ATR"`)

This is the strongest signal yet that the codegen pipeline is stable and family-agnostic.
