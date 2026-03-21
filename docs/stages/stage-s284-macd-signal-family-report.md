# Stage S284 — MACD Signal Family Report

**Stage:** S284
**Wave:** Signal Evolution (S283–S288)
**Date:** 2026-03-21
**Status:** COMPLETE

## 1. Executive Summary

S284 delivers the MACD (Moving Average Convergence Divergence) signal family
as the first family in the Signal Evolution Wave. The delivery validates that
the platform can generate domain value through the codegen-first pipeline,
not just infrastructure. All 8 universal acceptance criteria and all 4
family-specific criteria are met.

## 2. What Was Delivered

### Application Layer
- `MACDSampler` — pure stateful computation implementing standard MACD (12, 26, 9).
  Two-phase warm-up (26 candles for EMA seeding, 8 more for signal line EMA).
  Emits histogram as primary value with 8 metadata fields.

### Codegen Artifacts
- Family spec (`macd.yaml`), 2 golden snapshots, 2 integrated slices.
- `MACD` abbreviation added to codegen `knownAbbreviations`.
- Total codegen surface: 12 families, 24 slices, 0 failures.

### NATS Wiring
- EventSpec and ControlSpec for MACD in signal registry.
- Codegen-governed writer consumer (`writer-signal-macd`).
- Manual-owned store consumer (`store-signal-macd`).
- LatestSpecByType switch case for query gateway.

### Writer Pipeline
- Pipeline entry with codegen markers writing to existing `signals` table.

### Settings
- `"macd"` registered in `knownSignalFamilies`.
- Settings tests updated (family count 4→5).

## 3. Test Results

| Suite | Result |
|-------|--------|
| MACD sampler tests (9) | ALL PASS |
| Settings tests | ALL PASS |
| Full `make test` | ALL PASS (zero regressions) |
| Codegen check-all | 24/24 PASS |
| Codegen validate-all | 12/12 VALID |
| Codegen integrated | 24/24 PASS |

## 4. Acceptance Criteria

### Universal (AC-1 through AC-8): ALL PASS

### Family-Specific:
- MACD-1: MACD line computed correctly (fast EMA − slow EMA) — PASS
- MACD-2: Signal line computed correctly (EMA of MACD line) — PASS
- MACD-3: Publishes to `signal.events.macd.generated.>` — PASS
- MACD-4: Follows severity scaling pattern (histogram consumable downstream) — PASS

### Multi-Symbol: PASS
- Independent sampler instances per symbol.
- Partition and deduplication keys isolated.
- No hardcoded symbols.

## 5. Guard Rails Compliance

| Rule | Status |
|------|--------|
| No additional families opened | COMPLIANT — only MACD delivered |
| No codegen expansion as primary objective | COMPLIANT — added 1 abbreviation only |
| No architectural shortcuts | COMPLIANT — follows all canonical patterns |
| No observability platform inflation | COMPLIANT — no new metrics in this stage |

## 6. Architecture Alignment

MACD follows identical patterns to RSI, EMA Crossover, and Bollinger:
- Same Signal domain type with type-specific metadata.
- Same AddClose(closePrice, timestamp) sampler interface.
- Same NATS subject hierarchy and stream.
- Same ClickHouse signals table.
- Same codegen governance (consumer_spec + pipeline_entry).
- Same settings registration and validation.

No architectural deviations or exceptions were needed.

## 7. Limitations and Open Items

1. **KV store bucket** — `SIGNAL_MACD_LATEST` bucket creation is handled by the
   store binary at startup. No code changes needed (store uses ControlSpec from
   registry).
2. **Derive actor wiring** — the derive binary instantiates samplers based on
   configured signal families. MACD will be picked up when added to the pipeline
   config's `signal_families` array.
3. **Pipeline counter metric** — S283 charter lists this as interleaved for S284
   but scoped as "must not consume >20% effort". Deferred to be woven into the
   existing pipeline loop when ATR (S285) is delivered, as the metric applies
   to all families collectively, not per-family.

## 8. Preparation for S285 (ATR)

S285 delivers the ATR (Average True Range) signal family. Key differences from MACD:

1. **Evidence dependency** — ATR uses high, low, close (not just close). The
   sampler interface will need `AddCandle(high, low, close, ts)` or similar.
2. **No EMA dependency** — ATR uses Wilder's smoothing (same as RSI), not dual EMA.
3. **Value semantics** — ATR output is a volatility measure (always positive),
   not a momentum divergence.
4. **Codegen** — identical codegen surface (YAML spec, 2 goldens, 2 integrated
   slices). May need `"atr": "ATR"` in knownAbbreviations.
5. **Signal→Risk composition** — ATR is frequently used for position sizing in
   risk evaluators. S285 acceptance should verify this path.

## 9. Wave Progress

| Family            | Stage | Status   |
|-------------------|-------|----------|
| MACD              | S284  | COMPLETE |
| ATR               | S285  | PENDING  |
| Bollinger Squeeze | S286  | PENDING  |
| VWAP              | S287  | PENDING  |
| Post-wave gate    | S288  | PENDING  |
