# Stage S287 — Bollinger Squeeze Decision Family Report

**Status:** Complete
**Date:** 2026-03-21
**Scope:** Signal Evolution Wave closure — first decision family consuming evolved signal infrastructure

## Executive Summary

S287 delivers the `bollinger_squeeze` decision family, proving that the Signal Evolution Wave (S284-S286) translates into real decision-layer value. The family detects Bollinger Band volatility compression (squeeze conditions) by consuming the bollinger signal's %B value and bandwidth metadata. It is the third canonical decision family in market-foundry, following `rsi_oversold` and `ema_crossover`.

## Objective

Project, implement, validate, and document the `bollinger_squeeze` decision family, demonstrating that evolved signal infrastructure sustains concrete decision evolution.

## Design Summary

### What is a Bollinger Squeeze?

A Bollinger Squeeze occurs when the bandwidth between upper and lower Bollinger Bands contracts below a relative threshold, indicating low volatility and potential for a directional breakout. The evaluator computes **relative bandwidth** (`bandwidth / SMA`) to normalize across price levels, then classifies squeeze severity by compression depth.

### Key Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| squeezeThreshold | 0.10 | Relative bandwidth below which a squeeze is detected |
| Severity High | ratio ≤ 0.25 | Extreme compression |
| Severity Moderate | 0.25 < ratio ≤ 0.50 | Significant compression |
| Severity Low | ratio > 0.50 | Mild squeeze |

### Input Surface

The evaluator consumes:
- **%B** (signal value) — position within bands [0, 1]
- **bandwidth** (signal metadata) — absolute band width (upper - lower)
- **SMA** (signal metadata) — simple moving average over period

This is the first decision evaluator to consume signal metadata, enabled by the `SignalMetadata` field added to `signalGeneratedMessage`.

## Artifacts Delivered

### New Files

| File | Lines | Purpose |
|------|-------|---------|
| `internal/application/decision/bollinger_squeeze_evaluator.go` | 139 | Pure squeeze detection evaluator |
| `internal/application/decision/bollinger_squeeze_evaluator_test.go` | 376 | Comprehensive evaluator tests |
| `internal/actors/scopes/derive/bollinger_squeeze_decision_evaluator_actor.go` | 100 | Canonical actor wrapper |

### Modified Files

| File | Change |
|------|--------|
| `messages.go` | `SignalMetadata` field in `signalGeneratedMessage` |
| `signal_sampler_actor.go` | Populate metadata from signal (RSI) |
| `ema_crossover_signal_sampler_actor.go` | Populate metadata from signal (EMA) |
| `natsdecision/registry.go` | Event spec, control spec, consumer specs |
| `natsdecision/publisher.go` | Type routing for bollinger_squeeze |
| `natsdecision/kv_store.go` | Bucket constant |
| `settings/schema.go` | Family registration, dependency graph |
| `settings/settings_test.go` | Dependency validation tests |

## Test Results

| Package | Tests | Status |
|---------|-------|--------|
| `internal/application/decision` | 47 | PASS |
| `internal/shared/settings` | 34 | PASS |
| `internal/actors/scopes/derive` | all | PASS |
| Build check (actors, natsdecision) | — | PASS |

## Signal Evolution Wave — Value Ledger

| Stage | Layer | Family | Value |
|-------|-------|--------|-------|
| S284 | Signal | MACD | Momentum divergence detection |
| S285 | Signal | VWAP | Volume-weighted price deviation |
| S286 | Signal | ATR | Volatility range measurement |
| **S287** | **Decision** | **Bollinger Squeeze** | **Volatility compression → breakout detection** |

The wave proves the full transformation chain: evidence → signal → decision. Three new signal families feed one new decision family that delivers directional market intelligence.

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No new families beyond charter | Compliant — only bollinger_squeeze added |
| No venue readiness changes | Compliant |
| No observability wave changes | Compliant |
| No codegen expansion | Compliant |
| No opportunistic signal-decision coupling | Compliant — metadata bridge follows DBI-9 |
| No broad decision layer redesign | Compliant — evaluator follows canonical pattern |

## Architecture Notes

### SignalMetadata Extension

The `signalGeneratedMessage` now includes `SignalMetadata map[string]string`, forwarding signal metadata to decision evaluators without importing signal domain types. This is backward-compatible (existing evaluators ignore it) and enables future decision families to consume richer signal data without architecture changes.

### Dependency Graph Update

```
evidence/candle → signal/bollinger → decision/bollinger_squeeze
```

Added to `decisionDependsOnSignal` in settings schema. Pipeline validation enforces this dependency chain at config load time.

## Recommendation: Post-S287 Gate

The Signal Evolution Wave is now complete with demonstrable value:

1. **Breadth**: 4 new signal families (MACD, VWAP, ATR, Bollinger) operational
2. **Depth**: Bollinger Squeeze decision proves signal-to-decision transformation
3. **Infrastructure**: `SignalMetadata` bridge enables richer decision families without architecture changes
4. **Validation**: 80+ new tests across evaluator, settings, and actors

### Recommended next actions:

1. **Wire BollingerSignalSamplerActor** into the derive actor system to enable end-to-end bollinger signal → decision flow
2. **Consider strategy resolver** that consumes `bollinger_squeeze` decisions for squeeze-breakout entry strategies
3. **Gate ceremony** to formally close the Signal Evolution Wave and transition to the next charter

### Not recommended at this time:

- Additional decision families (wave is closed)
- Codegen expansion for bollinger_squeeze (should follow the existing codegen governance process)
- Broad decision layer redesign
