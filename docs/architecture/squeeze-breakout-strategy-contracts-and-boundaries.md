# Squeeze Breakout Strategy: Contracts and Boundaries

## Purpose

This document defines the contracts, ownership, and layer boundaries for the squeeze breakout strategy slice, ensuring decision and strategy layers remain cleanly separated.

## Contract: Decision → Strategy

### Input Contract

The `squeeze_breakout_entry` resolver receives decision output via `decisionEvaluatedMessage`:

| Field | Type | Source |
|---|---|---|
| DecisionType | string | `"bollinger_squeeze"` |
| DecisionOutcome | string | `"triggered"` / `"not_triggered"` / `"insufficient"` |
| DecisionConfidence | string | `"0.0000"` to `"1.0000"` |
| DecisionSeverity | string | `"none"` / `"low"` / `"moderate"` / `"high"` |
| DecisionRationale | string | Human-readable explanation |
| Timeframe | int | Seconds |
| Timestamp | time.Time | Event time |
| CorrelationID | string | Trace propagation |
| CausationID | string | Parent event ID |

### Output Contract

The resolver produces a `Strategy` domain object published as `StrategyResolvedEvent`:

| Field | Contract |
|---|---|
| Type | Always `"squeeze_breakout_entry"` |
| Direction | `"long"` when triggered, `"flat"` otherwise |
| Confidence | Severity-scaled, 4-decimal string, clamped [0, 1] |
| Parameters | `entry`, `breakout_target_pct`, `breakout_stop_pct` (only on triggered) |
| Decisions | Exactly one DecisionInput preserving raw confidence |
| Final | Always `true` |

### NATS Contract

| Property | Value |
|---|---|
| Stream | `STRATEGY_EVENTS` |
| Subject | `strategy.events.squeeze_breakout_entry.resolved.{source}.{symbol}.{timeframe}` |
| Event type | `strategy.events.v1.squeeze_breakout_entry_resolved` |
| Writer durable | `writer-strategy-squeeze-breakout-entry` |
| Store durable | `store-strategy-squeeze-breakout-entry` |

## Boundary: Decision vs. Strategy

### Decision Layer Responsibility (bollinger_squeeze)

- Detects whether a Bollinger squeeze exists (bandwidth below threshold).
- Classifies severity based on how far below the threshold.
- Determines %B zone (lower/middle/upper).
- Produces `triggered` / `not_triggered` outcome.
- Does NOT determine positional direction.
- Does NOT decide entry type or parameter values.
- Does NOT consider portfolio context.

### Strategy Layer Responsibility (squeeze_breakout_entry)

- Translates squeeze detection into positional intent (`long`).
- Applies severity-based parameter adjustment.
- Scales confidence for downstream risk consumption.
- Produces actionable parameters: entry type, target, and stop.
- Preserves full decision context for traceability.
- Does NOT evaluate signal data directly.
- Does NOT modify or re-evaluate the decision outcome.

### Invariants

1. **One-way data flow**: Decision → Strategy. Strategy never calls back into Decision.
2. **Primitive interface**: Strategy receives strings, not Decision domain objects.
3. **Raw confidence preservation**: DecisionInput.Confidence carries the unscaled value; Strategy.Confidence is severity-scaled.
4. **No decision logic in strategy**: The resolver does not re-check Bollinger bandwidth or %B zone.
5. **No strategy logic in decision**: The evaluator does not compute targets, stops, or direction.

## Dependency Graph

```
candle (evidence)
  └→ bollinger (signal)
       └→ bollinger_squeeze (decision)
            └→ squeeze_breakout_entry (strategy)
```

Configuration validation enforces:
- `squeeze_breakout_entry` requires `bollinger_squeeze` in `decision_families`.
- `bollinger_squeeze` requires `bollinger` in `signal_families`.
- `bollinger` requires `candle` in `families`.

## Ownership

| Artifact | Owner |
|---|---|
| `squeeze_breakout_entry_resolver.go` | `internal/application/strategy` |
| `squeeze_breakout_entry_resolver_actor.go` | `internal/actors/scopes/derive` |
| NATS registry entries | `internal/adapters/nats/natsstrategy` |
| Settings registration | `internal/shared/settings/schema.go` |
| Writer pipeline entry | `cmd/writer/pipeline.go` |

## Limits

- **Long-only**: The resolver currently produces only `long` direction on squeeze trigger. Short-side breakout is not modeled and requires a separate strategy family if needed.
- **No risk/execution coupling**: This stage intentionally stops at strategy. Risk evaluators and execution families are not modified.
- **No multi-decision composition**: The resolver consumes exactly one decision type. Multi-signal strategies (e.g., squeeze + trend confirmation) are a future concern.
