# MACD Signal Family — Implementation and Acceptance

**Stage:** S284
**Date:** 2026-03-21

## 1. Implementation Summary

The MACD signal family was delivered end-to-end following the 8-item acceptance
checklist defined in the Signal Evolution Wave charter (S283).

## 2. Acceptance Checklist

| # | Item                                        | Status |
|---|---------------------------------------------|--------|
| AC-1 | Codegen YAML spec                        | PASS — `codegen/families/macd.yaml` |
| AC-2 | Golden snapshots (consumer_spec + pipeline_entry) | PASS — `codegen/golden-snapshots/macd/` |
| AC-3 | Integration markers in target files      | PASS — `natssignal/registry.go`, `cmd/writer/pipeline.go` |
| AC-4 | Application-layer implementation         | PASS — `macd_sampler.go` |
| AC-5 | Behavioral tests (non-skipping)          | PASS — 9 tests, all passing |
| AC-6 | ClickHouse schema                        | PASS — uses existing `signals` table |
| AC-7 | Equivalence check PASS                   | PASS — 24/24 slices (12 families) |
| AC-8 | CI green (zero regressions)              | PASS — `make test` clean |

## 3. Family-Specific Acceptance Criteria

| # | Criterion                                      | Status |
|---|------------------------------------------------|--------|
| MACD-1 | Computes MACD line (fast EMA − slow EMA)  | PASS — verified in bullish/bearish tests |
| MACD-2 | Computes signal line (EMA of MACD line)   | PASS — metadata includes `signal_line` |
| MACD-3 | Publishes to `signal.events.macd`         | PASS — subject `signal.events.macd.generated.>` |
| MACD-4 | Follows severity scaling pattern          | PASS — histogram value is downstream-consumable |

## 4. Files Created

| File | Purpose |
|------|---------|
| `codegen/families/macd.yaml` | Codegen family spec |
| `codegen/golden-snapshots/macd/consumer_spec.go.golden` | Writer consumer golden |
| `codegen/golden-snapshots/macd/pipeline_entry.go.golden` | Pipeline entry golden |
| `internal/application/signal/macd_sampler.go` | MACD computation |
| `internal/application/signal/macd_sampler_test.go` | 9 behavioral tests |

## 5. Files Modified

| File | Change |
|------|--------|
| `internal/adapters/nats/natssignal/registry.go` | Added MACDGenerated, MACDLatest, WriterMACDSignalConsumer, StoreMACDSignalConsumer, LatestSpecByType case |
| `cmd/writer/pipeline.go` | Added MACD pipeline entry with codegen markers |
| `internal/shared/settings/schema.go` | Added `"macd"` to knownSignalFamilies |
| `internal/shared/settings/settings_test.go` | Updated family count (4→5) and unknown family test value |
| `codegen/spec.go` | Added `"macd": "MACD"` to knownAbbreviations |
| `codegen/integrated.yaml` | Added 2 MACD slice entries |

## 6. Test Coverage

9 tests covering:

1. **WarmUp** — verifies no signal before candle 34, first signal at candle 34
2. **BullishDivergence** — sustained rising prices → positive histogram
3. **BearishDivergence** — sustained falling prices → negative histogram
4. **ConstantPrices** — flat input → histogram ≈ 0
5. **Metadata** — all 8 metadata keys present with correct periods
6. **InvalidPrice** — non-numeric input handled gracefully
7. **Validate** — domain validation passes on produced signals
8. **MultiSymbol** — two symbols produce isolated partition/deduplication keys
9. **ContinuousProduction** — exactly 27 signals from 60 candles (34 warm-up)

## 7. Codegen Validation Results

```
check-all:     24 passed, 0 failed (12 families × 2 artifacts)
validate-all:  12 valid, 0 invalid (cross-spec uniqueness OK)
integrated:    24 passed, 0 failed (24 slices)
```

## 8. Multi-Symbol Verification

The `TestMACDSampler_MultiSymbol` test proves:
- Two independent sampler instances (btcusdt, ethusdt) produce signals
  with distinct partition keys and deduplication keys.
- No hardcoded symbols in the sampler.
- JetStream subject hierarchy provides stream-level isolation.

## 9. Regression Impact

- Zero test failures in existing families.
- `make test` passes clean.
- Writer binary compiles successfully.
- All 11 pre-existing codegen families unaffected.
