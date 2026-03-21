# Strategy-to-Risk Behavior Activation

> Stage S251 — Behavioral Wave 1

## Purpose

This document specifies how risk evaluators react to strategy type and decision severity,
replacing fixed risk multipliers with context-aware scaling that reflects the semantic
difference between strategy families and signal strength.

## Problem Statement

Before S251, risk evaluators applied uniform multipliers regardless of strategy type or
decision severity:

- **Position Exposure**: fixed ×0.95 confidence multiplier for all strategy types
- **Drawdown Limit**: fixed ×0.90 confidence multiplier for all strategy types
- Decision severity was carried as metadata only — not used in risk logic

This produced identical risk behavior for fundamentally different strategies (counter-trend
vs pro-trend), missing an opportunity to size risk more precisely.

## Behavioral Activation

### Position Exposure Evaluator

**Strategy-type confidence multiplier** (replaces fixed ×0.95):

| Strategy Type | Factor | Semantic Rationale |
|---|---|---|
| `mean_reversion_entry` | ×0.90 | Counter-trend carries inherently higher risk |
| `trend_following_entry` | ×0.95 | Pro-trend aligns with momentum → lower risk |
| Unknown | ×0.92 | Neutral default |

**Severity-based position limit multiplier**:

| Decision Severity | Factor | Semantic Rationale |
|---|---|---|
| `high` | ×1.15 | Strong signal justifies up to 15% larger position |
| `moderate` | ×1.00 | Neutral |
| `low` | ×0.80 | Weak signal → reduce position limit by 20% |
| `none` / empty | ×1.00 | Backward compatible |

**Formula**:
```
effectiveMaxPosition = maxPositionPct × severityFactor
requestedSize = strategyConfidence × effectiveMaxPosition
riskConfidence = strategyConfidence × strategyTypeConfidenceFactor
```

### Drawdown Limit Evaluator

**Strategy-type confidence multiplier** (replaces fixed ×0.90):

| Strategy Type | Factor | Semantic Rationale |
|---|---|---|
| `mean_reversion_entry` | ×0.85 | Counter-trend needs stricter drawdown assessment |
| `trend_following_entry` | ×0.92 | Pro-trend can tolerate slightly more |
| Unknown | ×0.88 | Neutral default |

**Strategy-type stop distance multiplier**:

| Strategy Type | Factor | Semantic Rationale |
|---|---|---|
| `mean_reversion_entry` | ×0.85 | Counter-trend → tighter stop distance ceiling |
| `trend_following_entry` | ×1.15 | Pro-trend → wider stop, room for trend to develop |
| Unknown | ×1.00 | Neutral default |

**Severity-based drawdown tolerance multiplier**:

| Decision Severity | Factor | Semantic Rationale |
|---|---|---|
| `high` | ×1.15 | Strong signal → tolerate 15% more drawdown |
| `moderate` | ×1.00 | Neutral |
| `low` | ×0.80 | Weak signal → tighten drawdown tolerance by 20% |
| `none` / empty | ×1.00 | Backward compatible |

**Formula**:
```
effectiveStopBase = stopDistancePct × strategyTypeStopFactor
effectiveMaxDrawdown = maxDrawdownPct × severityFactor
stopDistance = effectiveStopBase × strategyConfidence
riskConfidence = strategyConfidence × strategyTypeConfidenceFactor
```

## Rationale Format

Risk rationales now explain both strategy type and severity influence:

```
Position size 0.0136 within exposure limits; mean_reversion_entry (confidence ×0.90); decision severity low (limit ×0.80)
```

```
Stop distance 0.0217 within drawdown limits for long; mean_reversion_entry (confidence ×0.85, stop ×0.85); decision severity low (tolerance ×0.80)
```

When severity is `none` or empty, the severity clause is omitted:

```
Position size 0.0170 within exposure limits; mean_reversion_entry (confidence ×0.90)
```

## Metadata

Risk assessments now record `strategy_type` in metadata alongside `decision_severity` and
`decision_rationale`, providing a complete audit trail for downstream consumers and
observability tooling.

## Parameters

Both evaluators now emit effective (adjusted) parameters alongside base configuration:

**Position Exposure**:
- `max_position_pct` — base config value
- `effective_max_position_pct` — after severity adjustment
- `confidence_factor` — strategy-type-specific multiplier applied
- `severity_limit_factor` — severity-based position limit multiplier

**Drawdown Limit**:
- `stop_distance_pct` — base config value
- `effective_stop_distance_pct` — after strategy-type adjustment
- `max_drawdown_pct` — base config value
- `effective_max_drawdown_pct` — after severity adjustment
- `confidence_factor` — strategy-type-specific multiplier applied
- `stop_type_factor` — strategy-type-specific stop multiplier
- `severity_tolerance_factor` — severity-based drawdown tolerance multiplier

## Design Constraints

- **No cross-domain imports**: Risk receives strategy type and decision severity as
  primitive strings, preserving domain isolation (DBI-9).
- **Pure functions**: All scaling logic is stateless with no I/O.
- **Backward compatible**: Unknown strategy types use neutral defaults; unknown/empty
  severity defaults to ×1.00.
- **No policy engine**: Scaling factors are declared as package-level maps, not a
  configurable rules engine.

## Non-Objectives

- This is NOT a risk policy engine or rules framework.
- Risk types remain `position_exposure` and `drawdown_limit` — no new risk families.
- No new actor topology or message types were introduced.
- Disposition logic (approved/modified/rejected) is unchanged — only the inputs to
  that logic are now context-aware.
