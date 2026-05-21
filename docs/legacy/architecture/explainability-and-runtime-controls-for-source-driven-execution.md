# Explainability and Runtime Controls for Source-Driven Execution

> S361 — Strategy Signal Integration Block SSI-3

## Purpose

This document defines the explainability surfaces and runtime controls that make the source-driven execution path diagnosable and operationally controllable. The scope is minimal: explain why a source generated active execution or was blocked, and provide clear enable/disable/halt controls.

## Source-Driven Execution Path Overview

The canonical source-driven path flows:

```
StrategyResolvedEvent (NATS STRATEGY_EVENTS)
  → StrategyConsumerActor evaluation
    → ExecutionIntent production
      → VenueAdapterActor safety gates
        → Venue submission → Fill publication
```

The path is identified by `source_path = "strategy_consumer.mean_reversion_entry"`.

## Explainability Surfaces

### 1. Intent-Level Explainability (Parameters Map)

Every ExecutionIntent produced by the strategy consumer carries these explainability fields in `Parameters`:

| Field | Value | Purpose |
|-------|-------|---------|
| `source_path` | `strategy_consumer.mean_reversion_entry` | Identifies which source path produced this intent |
| `evaluation_outcome` | `actionable` or `flat` | Whether the strategy produced a directional or flat evaluation |
| `confidence_threshold` | e.g., `0.50` | The minimum confidence configured (empty if disabled) |
| `strategy_type` | `mean_reversion_entry` | Strategy family identity (INV-1) |
| `strategy_direction` | `long`, `short`, `flat` | Original strategy direction |
| `strategy_confidence` | e.g., `0.8500` | Strategy confidence value |
| `decision_severity` | e.g., `high` | Decision severity from upstream |
| `risk_type` | `pass_through` | Risk evaluation mode |
| `risk_disposition` | `approved` | Risk disposition |
| `max_position_pct` | `0.01` | Position size cap |

These fields answer: "What did the strategy say, and how was it evaluated?"

### 2. Composite Source Explain Endpoint

**GET /execution/source-explain**

Returns a single JSON response combining activation state, gate status, source path configuration, and last execution status.

Optional query parameters: `source`, `symbol`, `timeframe` — when provided, includes the last intent and result for that partition.

Response schema:

```json
{
  "explanation": {
    "source_path": "strategy_consumer.mean_reversion_entry",
    "strategy_type": "mean_reversion_entry",
    "activation": {
      "adapter": "paper",
      "gate": { "status": "active", "reason": "", "updated_at": "...", "updated_by": "" },
      "credentials": "present",
      "effective": "paper",
      "observed_at": "..."
    },
    "gate": { "status": "active", "reason": "", "updated_at": "...", "updated_by": "" },
    "config": {
      "max_position_pct": "0.01",
      "min_confidence": "0.50",
      "staleness_max_age": "120s",
      "risk_type": "pass_through"
    },
    "last_intent": null,
    "last_result": null,
    "propagation": "none",
    "observed_at": "..."
  }
}
```

This endpoint answers:
- **Is the source path active?** → `activation.effective` + `gate.status`
- **What configuration governs it?** → `config.*`
- **What was the last execution?** → `last_intent` + `last_result` + `propagation`
- **Why was it blocked?** → `gate.status=halted` + `gate.reason`

### 3. Prometheus Metrics

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `marketfoundry_execution_strategy_evaluations_total` | Counter | `strategy_type`, `outcome` | Strategy evaluation volume and outcomes |
| `marketfoundry_execution_gate_checks_total` | Counter | `gate`, `verdict` | Gate check frequency and block rate |
| `marketfoundry_execution_intents_total` | Counter | `source_path`, `side` | Intent production volume by source and side |
| `marketfoundry_execution_gate_active` | Gauge | — | Current gate state (1=active, 0=halted) |

Canonical `outcome` values: `actionable`, `flat`, `skipped_wrong_type`, `skipped_low_confidence`, `error`.
Canonical `verdict` values: `allowed`, `blocked`.
Canonical `gate` values: `kill_switch`, `staleness`, `all`.

## Runtime Controls

### 1. Global Kill Switch (Existing)

- **Authority**: NATS KV `EXECUTION_CONTROL/global`
- **HTTP**: `PUT /execution/control` with `{"status": "halted", "reason": "...", "updated_by": "..."}`
- **Scope**: Blocks ALL intents — both derive-path and strategy-path
- **Fail-open**: KV unavailability allows execution to proceed

### 2. Confidence Threshold (New — S361)

- **Configuration**: `StrategyConsumerConfig.MinConfidence`
- **Behavior**: Events with confidence below threshold are skipped with counter `skipped_low_confidence`
- **Default**: Empty string (disabled — all events evaluated)
- **Fail-open**: Invalid threshold config passes all events
- **Boundary**: Equal-to-threshold passes (strict less-than comparison)

### 3. Staleness Guard (Existing)

- **Configuration**: `VenueAdapterConfig.StalenessMaxAge` (default: 120s)
- **Clock source**: `intent.Timestamp` vs `time.Now().UTC()`
- **Scope**: Rejects old intents regardless of source path

### Control Composition

The three controls compose in sequence:

```
Strategy Event arrives
  → [Confidence Threshold] — skip if below min_confidence
    → [Evaluation] — produce ExecutionIntent
      → [Kill Switch] — block if gate halted
        → [Staleness Guard] — block if intent too old
          → [Venue Submit]
```

Confidence threshold is checked in StrategyConsumerActor (pre-evaluation).
Kill switch and staleness are checked in VenueAdapterActor (pre-submission).

## Diagnostic Workflow

### "Why is the source not generating execution?"

1. `GET /execution/source-explain` — check `activation.effective` and `gate.status`
2. If gate is halted → `PUT /execution/control {"status": "active", "reason": "...", "updated_by": "..."}`
3. If no recent intents → check strategy consumer logs for `skipped_low_confidence` or `skipped_wrong_type`
4. Check Prometheus: `marketfoundry_execution_strategy_evaluations_total{outcome="skipped_low_confidence"}`

### "Why was this specific intent blocked?"

1. `GET /execution/paper_order/latest?source=...&symbol=...&timeframe=...` — check intent Parameters
2. Look for `evaluation_outcome`, `source_path`, `confidence_threshold`
3. Check Prometheus: `marketfoundry_execution_gate_checks_total{verdict="blocked"}`

### "Is the source-driven path healthy?"

1. `GET /execution/source-explain` — composite view
2. `GET /metrics` — check `strategy_evaluations_total` counters
3. `GET /healthz` — process liveness

## Limitations

- **No per-strategy gate**: Kill switch is global only. Per-strategy-type halt deferred to future wave.
- **No confidence threshold HTTP control**: Threshold is set at startup config, not runtime-mutable.
- **No gate change history**: Only current gate state is queryable, not historical changes.
- **No rejection event stream**: Skipped intents are counted but not published as events.
- **No correlation ID query**: Cannot trace full execution journey via HTTP (must use logs).
