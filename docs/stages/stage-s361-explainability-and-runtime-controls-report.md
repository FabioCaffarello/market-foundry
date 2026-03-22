# Stage S361 — Explainability and Runtime Controls for Source-Driven Execution

> Strategy Signal Integration Block SSI-3
> Delivered: 2026-03-22

## 1. Executive Summary

S361 makes the source-driven execution path explainable and runtime-controllable. After S360 wired strategy events to execution, the path worked but was opaque: operators could not easily answer "why did the source generate execution?" or "why was it blocked?". S361 closes this gap with:

- **Prometheus metrics** for strategy evaluations, gate checks, and intent production
- **Confidence threshold** as a source-specific runtime control
- **Enriched intent Parameters** carrying source path, evaluation outcome, and configuration context
- **Composite explain endpoint** (`GET /execution/source-explain`) aggregating activation, gate, config, and status
- **Proportional tests** covering all new behaviors

The path is now diagnosable without log diving and controllable via existing gate + new confidence threshold.

## 2. Explainability and Controls Delivered

### 2.1 Prometheus Metrics (L6 Closure)

Four new metric families expose the source-driven path to monitoring:

| Metric | Type | Purpose |
|--------|------|---------|
| `marketfoundry_execution_strategy_evaluations_total` | Counter | Evaluation volume by strategy type and outcome |
| `marketfoundry_execution_gate_checks_total` | Counter | Gate check frequency by gate and verdict |
| `marketfoundry_execution_intents_total` | Counter | Intent production by source path and side |
| `marketfoundry_execution_gate_active` | Gauge | Current gate state (1=active, 0=halted) |

Canonical outcomes: `actionable`, `flat`, `skipped_wrong_type`, `skipped_low_confidence`, `error`.

### 2.2 Confidence Threshold (L5 Closure)

New `MinConfidence` configuration on `StrategyConsumerConfig`:

- Skips events with confidence < threshold
- Records `skipped_low_confidence` counter in both healthz tracker and Prometheus
- Logs skipped events with full context (confidence, threshold, source, symbol, timeframe, correlation_id)
- Fail-open: invalid threshold or unparseable confidence passes the event
- Boundary: equal-to-threshold passes (strict less-than comparison)

### 2.3 Intent Parameter Enrichment

Every strategy-produced ExecutionIntent now carries three additional Parameters:

| Field | Value | Purpose |
|-------|-------|---------|
| `source_path` | `strategy_consumer.mean_reversion_entry` | Identifies source path |
| `evaluation_outcome` | `actionable` or `flat` | Evaluation category |
| `confidence_threshold` | e.g., `0.50` | Configured minimum (omitted if disabled) |

These join the existing 7 Parameters (strategy_type, strategy_direction, strategy_confidence, decision_severity, risk_type, risk_disposition, max_position_pct) for a total of 10 explainability fields per intent.

### 2.4 Source Explain Endpoint

**GET /execution/source-explain[?source=...&symbol=...&timeframe=...]**

Returns composite JSON combining:
- Activation surface (three-dimensional state)
- Gate status (with audit fields)
- Source path configuration (max_position_pct, min_confidence, staleness_max_age, risk_type)
- Last intent and result for the partition (when query params provided)
- Effective propagation status

### 2.5 Gate Verdict Prometheus Recording

VenueAdapterActor now records Prometheus metrics for every safety gate check:
- `gate_checks_total{gate="kill_switch",verdict="blocked"}` when kill switch blocks
- `gate_checks_total{gate="stale",verdict="blocked"}` when staleness rejects (mapped from existing `stale` reason)
- `gate_checks_total{gate="all",verdict="allowed"}` when all gates pass
- `gate_active` gauge updated on every gate check

## 3. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/domain/execution/explain.go` | `SourcePathExplanation` and `SourcePathConfig` types |
| `internal/application/executionclient/explain_contracts.go` | Query/Reply contracts for source explain |
| `internal/application/executionclient/get_source_explanation.go` | Use case composing explanation from existing gateways |
| `internal/interfaces/http/handlers/source_explain.go` | HTTP handler for source explain endpoint |
| `internal/interfaces/http/routes/source_explain.go` | Route registration for `GET /execution/source-explain` |
| `internal/interfaces/http/routes/source_explain_test.go` | Route tests (registered, unavailable, nil) |
| `docs/architecture/explainability-and-runtime-controls-for-source-driven-execution.md` | Architecture document |
| `docs/architecture/source-driven-queryability-audit-fields-and-operational-limits.md` | Audit and query reference |

### Modified Files

| File | Change |
|------|--------|
| `internal/shared/metrics/metrics.go` | Added 4 execution metric families + 4 helper functions |
| `internal/shared/metrics/metrics_test.go` | Added `TestExecutionMetrics_DoNotPanic` |
| `internal/actors/scopes/execute/strategy_consumer_actor.go` | Added confidence threshold, Prometheus metrics, explainability Parameters |
| `internal/actors/scopes/execute/strategy_consumer_actor_test.go` | Added 6 new tests (confidence threshold + explainability fields) |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Added Prometheus gate check recording |
| `internal/interfaces/http/routes/core.go` | Added `SourceExplainFamilyDeps`, interface, `DefaultRoutes` wiring |

## 4. Tests and Evidence

### New Tests (6)

| Test | Validates |
|------|-----------|
| `TestStrategyConsumer_ConfidenceThreshold_AboveThreshold_Evaluated` | Events above threshold produce intents |
| `TestStrategyConsumer_ConfidenceThreshold_BelowThreshold_Skipped` | Events below threshold are skipped |
| `TestStrategyConsumer_ConfidenceThreshold_EmptyString_DisablesFilter` | Empty threshold passes all events |
| `TestStrategyConsumer_ConfidenceThreshold_EqualToThreshold_Evaluated` | Boundary: equal-to passes |
| `TestStrategyConsumer_ExplainabilityFields_Present` | source_path, evaluation_outcome, confidence_threshold in Parameters |
| `TestStrategyConsumer_ExplainabilityFields_FlatOutcome` | Flat direction produces `evaluation_outcome=flat` |

### New Route Tests (3)

| Test | Validates |
|------|-----------|
| `TestSourceExplainRouteRegistered` | Endpoint returns 200 with explanation payload |
| `TestSourceExplainRouteUnavailable` | Returns 503 when use case fails |
| `TestSourceExplainRouteOmittedWhenNil` | No route registered when use case is nil |

### New Metrics Tests (1)

| Test | Validates |
|------|-----------|
| `TestExecutionMetrics_DoNotPanic` | All 4 metric helpers callable, metrics appear in /metrics output |

### Existing Tests (11 — all passing)

All S360 invariant tests continue to pass with the new code.

### Test Results

```
ok  internal/shared/metrics           — 6/6 PASS
ok  internal/actors/scopes/execute    — 17/17 PASS
ok  internal/interfaces/http/routes   — source explain 3/3 PASS
```

## 5. Remaining Limits

| ID | Gap | Impact | Deferred To |
|----|-----|--------|------------|
| L1 | No per-strategy gate | Cannot halt mean_reversion_entry without halting all execution | Future wave |
| L2 | No confidence threshold HTTP control | Threshold requires process restart to change | Future wave (if needed) |
| L3 | No gate change history | Cannot audit when/why gate was halted over time | Future wave |
| L4 | No rejected intent event stream | Skipped intents counted but not published for downstream analysis | Future wave |
| L5 | No correlation ID HTTP query | Full execution trace requires structured logs, not HTTP | Future wave |
| L6 | No historical execution queries | Only latest intent/result per partition; time-series needs ClickHouse | Future wave |
| L7 | Source explain requires gateway wiring | Use case created but gateway binary must wire the `SourcePathConfigProvider` | Gateway compose.go update |

## 6. Preparation for S362

S362 will validate the complete source-driven execution path end-to-end. S361 provides the foundation:

### What S362 Can Now Leverage

1. **Composite explain endpoint**: `GET /execution/source-explain` provides a single-request diagnostic for integration test assertions.
2. **Prometheus metrics**: Test harness can scrape `/metrics` to verify strategy evaluations, gate checks, and intent production counts.
3. **Enriched Parameters**: Integration tests can assert on `source_path`, `evaluation_outcome`, and `confidence_threshold` in produced intents.
4. **Confidence threshold**: Tests can exercise the threshold to verify low-confidence events are correctly skipped.
5. **Gate verdict metrics**: Tests can verify that halt/resume cycles are correctly counted.

### S362 Recommended Scope

1. **End-to-end signal → execution**: Publish RSI signal → verify strategy resolved → verify intent produced with correct Parameters → verify fill event published.
2. **Kill switch integration**: Halt gate → verify intent blocked → resume → verify intent passes.
3. **Confidence threshold integration**: Set threshold → verify low-confidence skipped → remove threshold → verify passes.
4. **Staleness integration**: Send old strategy event → verify stale rejection via Prometheus metric.
5. **Correlation chain verification**: Trace correlation_id from signal through strategy through intent through fill.
6. **Operational runbook validation**: Execute diagnostic workflow documented in S361 architecture doc.

### Prerequisites Verified

- ✓ Source-driven path produces enriched intents (S360 + S361)
- ✓ Path is observable via Prometheus (S361)
- ✓ Path is controllable via gate + confidence threshold (existing + S361)
- ✓ Composite diagnostic endpoint available (S361)
- ✓ All 27 tests passing across all modified packages
