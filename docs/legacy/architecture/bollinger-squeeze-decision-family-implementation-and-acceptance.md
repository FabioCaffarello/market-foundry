# Bollinger Squeeze Decision Family — Implementation and Acceptance

## Implementation Summary

The `bollinger_squeeze` decision family was implemented as the third canonical decision family in market-foundry, following `rsi_oversold` and `ema_crossover`. It proves that the Signal Evolution Wave (S284-S286) delivers concrete decision-layer value.

## Files Created

| File | Purpose |
|------|---------|
| `internal/application/decision/bollinger_squeeze_evaluator.go` | Pure evaluator logic — squeeze detection via relative bandwidth |
| `internal/application/decision/bollinger_squeeze_evaluator_test.go` | 25+ test cases covering triggered/not-triggered, severity, confidence, metadata |
| `internal/actors/scopes/derive/bollinger_squeeze_decision_evaluator_actor.go` | Actor wrapper following canonical decision evaluator pattern |

## Files Modified

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/messages.go` | Added `SignalMetadata map[string]string` to `signalGeneratedMessage` |
| `internal/actors/scopes/derive/signal_sampler_actor.go` | Populate `SignalMetadata` from signal in RSI sampler actor |
| `internal/actors/scopes/derive/ema_crossover_signal_sampler_actor.go` | Populate `SignalMetadata` from signal in EMA crossover sampler actor |
| `internal/adapters/nats/natsdecision/registry.go` | Added `BollingerSqueezeEvaluated` event spec, `BollingerSqueezeLatest` control spec, writer and store consumer specs |
| `internal/adapters/nats/natsdecision/publisher.go` | Added routing for `"bollinger_squeeze"` type to event spec |
| `internal/adapters/nats/natsdecision/kv_store.go` | Added `BollingerSqueezeLatestBucket` constant |
| `internal/shared/settings/schema.go` | Registered `bollinger_squeeze` in `knownDecisionFamilies` and `decisionDependsOnSignal` |
| `internal/shared/settings/settings_test.go` | Added dependency validation tests for bollinger_squeeze family |

## Acceptance Criteria Verification

### 1. Bollinger Squeeze exists as canonical decision family

- Evaluator implements the standard decision evaluation pattern (pure logic, no I/O)
- Actor follows the canonical decision evaluator actor pattern (receive signal, evaluate, publish, fan-out)
- NATS registry, publisher, kv_store, and consumer specs fully registered
- Configuration schema recognizes `bollinger_squeeze` with dependency on `bollinger` signal

### 2. Family integrates signals cleanly and usefully

- Consumes `bollinger` signal's %B value and bandwidth/SMA metadata
- Uses relative bandwidth normalization for price-level-independent squeeze detection
- Does not import signal domain types — uses primitive data only (DBI-9)
- `SignalMetadata` extension to `signalGeneratedMessage` is backward-compatible

### 3. Signal Evolution Wave closes with domain value

- S284 (MACD) → momentum divergence signal
- S285 (VWAP) → volume-weighted price deviation signal
- S286 (ATR) → volatility range signal
- S287 (Bollinger Squeeze) → **decision-layer consumer** proving signals drive real decisions

The wave demonstrates the full signal→decision transformation path.

### 4. Infrastructure sustains real decision evolution

- The `signalGeneratedMessage` now carries metadata, enabling richer decision families without architecture changes
- The decision publisher, consumer, and KV patterns scale identically for all decision families
- Configuration validation enforces dependency chains automatically

## Test Coverage

### Evaluator Tests (25 cases)

| Category | Tests | Status |
|----------|-------|--------|
| Triggered/Not-triggered | 4 | PASS |
| Invalid inputs (value, metadata, SMA) | 5 | PASS |
| Domain validation | 1 | PASS |
| Confidence bounds | 5 subtests | PASS |
| Confidence monotonicity | 2 | PASS |
| Severity (low/moderate/high/none) | 4 | PASS |
| Severity monotonicity | 1 | PASS |
| Rationale content | 2 | PASS |
| Metadata enrichment | 4 | PASS |
| Timestamp/signal preservation | 2 | PASS |

### Settings Tests (2 new cases)

| Test | Status |
|------|--------|
| Rejects bollinger_squeeze without bollinger signal | PASS |
| Accepts bollinger_squeeze with bollinger signal | PASS |

### Existing Tests (regression)

| Package | Status |
|---------|--------|
| `internal/application/decision` | PASS (all 22 RSI + 25 Bollinger) |
| `internal/shared/settings` | PASS (all 32 tests) |
| `internal/actors/scopes/derive` | PASS (all 6.3s) |
| `internal/adapters/nats/natsdecision` | PASS (build-check) |

## Architectural Conformance

- No new domain types added — uses existing `decision.Decision`, `decision.SignalInput`
- No cross-domain imports — evaluator consumes primitive data only
- No codegen-governed region violations
- No new families opened beyond charter scope
- No venue readiness, observability, or codegen expansion changes
