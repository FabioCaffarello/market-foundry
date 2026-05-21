# Source-Driven Queryability, Audit Fields, and Operational Limits

> S361 — Audit and Query Contract Reference

## Query Surfaces

### HTTP Endpoints (Gateway Binary)

| Method | Path | Purpose | S361 Status |
|--------|------|---------|-------------|
| GET | `/activation/surface` | Three-dimensional activation state | Existing (S344) |
| GET | `/execution/control` | Current gate status, reason, updater | Existing (S339) |
| PUT | `/execution/control` | Set gate status (halt/resume) | Existing (S339) |
| GET | `/execution/:type/latest` | Latest intent by type/source/symbol/tf | Existing (S264) |
| GET | `/execution/status/latest` | Composite status (intent+result+gate) | Existing (S277) |
| GET | `/execution/source-explain` | Source path composite explanation | **New (S361)** |
| GET | `/metrics` | Prometheus metrics endpoint | Existing (S354) |
| GET | `/healthz` | Process liveness | Existing |

### NATS Request/Reply (Store Binary)

| Subject | Purpose | S361 Status |
|---------|---------|-------------|
| `execution.query.paper_order.latest` | Latest paper order intent | Existing |
| `execution.query.venue_market_order.latest` | Latest venue fill result | Existing |
| `execution.query.status.latest` | Composite execution status | Existing |
| `execution.query.control.get` | Current execution gate | Existing |
| `execution.query.control.set` | Update execution gate | Existing |
| `execution.query.activation.surface.get` | Full activation surface | Existing (S344) |

### NATS KV Read Models

| Bucket | Key | Content | S361 Status |
|--------|-----|---------|-------------|
| `EXECUTION_CONTROL` | `global` | Gate status, reason, updater, timestamp | Existing |
| `EXECUTION_CONTROL` | `dimensions` | Adapter state, credential state, reporter | Existing (S344) |
| `EXECUTION_PAPER_ORDER` | `{source}.{symbol}.{tf}` | Latest paper order intent | Existing |
| `EXECUTION_VENUE_MARKET_ORDER` | `{source}.{symbol}.{tf}` | Latest venue fill result | Existing |

## Audit Fields on ExecutionIntent

### Core Identity Fields

| Field | Type | Source | Mutability |
|-------|------|--------|------------|
| `type` | string | Evaluator | Immutable |
| `source` | string | Strategy.Source | Immutable |
| `symbol` | string | Strategy.Symbol | Immutable |
| `timeframe` | int | Strategy.Timeframe | Immutable |
| `timestamp` | time.Time | Strategy.Timestamp (INV-5) | Immutable |

### Execution State Fields

| Field | Type | Source | Mutability |
|-------|------|--------|------------|
| `side` | string | Evaluator (INV-2) | Immutable |
| `quantity` | string | Config (max_position_pct) | Immutable |
| `status` | string | Lifecycle progression | Mutable |
| `filled_quantity` | string | Venue response | Set once on fill |
| `final` | bool | Lifecycle terminal check | Set once |
| `fills` | []FillRecord | Venue response | Append-only |

### Causal Chain Fields

| Field | Type | Source | Purpose |
|-------|------|--------|---------|
| `correlation_id` | string | Event.Metadata.CorrelationID (INV-3) | End-to-end trace |
| `causation_id` | string | Event.Metadata.ID (INV-3) | Direct parent event |

### Risk Assessment Fields

| Field | Type | Source | Purpose |
|-------|------|--------|---------|
| `risk.type` | string | `"pass_through"` (INV-4) | Risk evaluation mode |
| `risk.disposition` | string | `"approved"` (INV-4) | Risk decision |
| `risk.confidence` | string | Strategy.Confidence | Confidence value |
| `risk.timeframe` | int | Strategy.Timeframe | Risk evaluation timeframe |
| `risk.strategy_type` | string | Strategy.Type (INV-1) | Strategy family identity |
| `risk.decision_severity` | string | Strategy.Decisions[0].Severity | Decision urgency |

### Explainability Fields (Parameters Map — S361)

| Key | Value Example | Purpose |
|-----|--------------|---------|
| `source_path` | `strategy_consumer.mean_reversion_entry` | Source path identity |
| `evaluation_outcome` | `actionable` / `flat` | Evaluation result category |
| `confidence_threshold` | `0.50` | Configured minimum confidence |
| `strategy_type` | `mean_reversion_entry` | Strategy family |
| `strategy_direction` | `long` / `short` / `flat` | Original strategy direction |
| `strategy_confidence` | `0.8500` | Raw strategy confidence |
| `decision_severity` | `high` | Upstream decision severity |
| `risk_type` | `pass_through` | Risk mode |
| `risk_disposition` | `approved` | Risk disposition |
| `max_position_pct` | `0.01` | Position size cap |

### Audit Fields on ControlGate

| Field | Type | Purpose |
|-------|------|---------|
| `status` | GateStatus | `active` or `halted` |
| `reason` | string | Why the gate is in this state |
| `updated_at` | time.Time | When last changed |
| `updated_by` | string | Who changed it |

### Audit Fields on ActivationSurface

| Field | Type | Purpose |
|-------|------|---------|
| `adapter` | AdapterState | `paper` or `venue` (immutable per process) |
| `gate` | ControlGate | Current gate with full audit fields |
| `credentials` | CredentialState | `present` or `absent` (immutable per process) |
| `effective` | EffectiveMode | Derived truth: paper, venue_halted, venue_live, venue_degraded |
| `observed_at` | time.Time | When the surface was computed |

## Prometheus Metrics (S361)

### Strategy Evaluation Metrics

```
# HELP marketfoundry_execution_strategy_evaluations_total Total strategy evaluations by strategy type and outcome.
# TYPE marketfoundry_execution_strategy_evaluations_total counter
marketfoundry_execution_strategy_evaluations_total{strategy_type="mean_reversion_entry",outcome="actionable"} 42
marketfoundry_execution_strategy_evaluations_total{strategy_type="mean_reversion_entry",outcome="flat"} 18
marketfoundry_execution_strategy_evaluations_total{strategy_type="mean_reversion_entry",outcome="skipped_low_confidence"} 3
marketfoundry_execution_strategy_evaluations_total{strategy_type="mean_reversion_entry",outcome="skipped_wrong_type"} 0
marketfoundry_execution_strategy_evaluations_total{strategy_type="mean_reversion_entry",outcome="error"} 0
```

### Gate Check Metrics

```
# HELP marketfoundry_execution_gate_checks_total Total pre-submit gate checks by gate and verdict.
# TYPE marketfoundry_execution_gate_checks_total counter
marketfoundry_execution_gate_checks_total{gate="all",verdict="allowed"} 60
marketfoundry_execution_gate_checks_total{gate="kill_switch",verdict="blocked"} 0
marketfoundry_execution_gate_checks_total{gate="stale",verdict="blocked"} 2
```

### Intent Production Metrics

```
# HELP marketfoundry_execution_intents_total Total execution intents produced by source path and side.
# TYPE marketfoundry_execution_intents_total counter
marketfoundry_execution_intents_total{source_path="strategy_consumer.mean_reversion_entry",side="buy"} 25
marketfoundry_execution_intents_total{source_path="strategy_consumer.mean_reversion_entry",side="sell"} 17
marketfoundry_execution_intents_total{source_path="strategy_consumer.mean_reversion_entry",side="none"} 18
```

### Gate Status Gauge

```
# HELP marketfoundry_execution_gate_active Whether the execution gate is active (1) or halted (0).
# TYPE marketfoundry_execution_gate_active gauge
marketfoundry_execution_gate_active 1
```

## Operational Limits

### What IS Queryable

- Current activation state (three-dimensional)
- Current gate status with reason and audit fields
- Latest intent per partition (source/symbol/timeframe)
- Latest result per partition
- Composite status (intent + result + gate + propagation)
- Source path composite explanation (activation + gate + config + status)
- Prometheus counters for strategy evaluations, gate checks, intents

### What IS NOT Queryable

- **Gate change history**: Only the current gate state is stored. No revision history in KV.
- **Rejected intent stream**: Skipped intents (low confidence, wrong type, stale, halted) are counted in Prometheus but not published as events.
- **Full correlation trace via HTTP**: Cannot query all events sharing a correlation_id through the HTTP surface. Must use structured logs.
- **Historical execution status**: Only the latest intent/result per partition is materialized. Time-series queries require ClickHouse (future).
- **Per-strategy gate state**: No strategy-type-specific halt capability. Global gate only.
- **Confidence threshold at runtime**: Threshold is set at startup config, not mutable via HTTP.

### Label Cardinality Bounds

| Metric | Label | Cardinality | Bound |
|--------|-------|-------------|-------|
| strategy_evaluations | strategy_type | 1 currently (mean_reversion_entry) | Max ~5 strategy types |
| strategy_evaluations | outcome | 5 canonical values | Fixed |
| gate_checks | gate | 3 canonical values | Fixed |
| gate_checks | verdict | 2 canonical values | Fixed |
| intents | source_path | 1 currently | Max ~5 source paths |
| intents | side | 3 canonical values | Fixed |

Total label combinations: bounded and safe for Prometheus.
