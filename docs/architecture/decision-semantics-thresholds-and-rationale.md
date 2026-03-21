# Decision Semantics, Thresholds, and Rationale — S234

## Purpose

This document defines the semantic model for decision severity zones, the rationale format, and the threshold boundaries used by the RSI Oversold evaluator. It serves as the reference for interpreting decision output and for future evaluator families.

## Severity Model

### Definition

Severity classifies **how extreme** the evaluated condition is. It is a property of the decision, not a recommendation.

| Severity   | Meaning                                    | RSI Oversold Range |
|------------|--------------------------------------------|--------------------|
| `none`     | Condition not met; neutral zone            | RSI >= 30          |
| `low`      | Condition met with mild intensity          | 20 <= RSI < 30     |
| `moderate` | Condition met with notable intensity       | 10 <= RSI < 20     |
| `high`     | Condition met with extreme intensity       | RSI < 10           |

### Design Constraints

1. Severity is **always `none`** for `not_triggered` outcomes.
2. Severity is **never `none`** for `triggered` outcomes.
3. Severity is monotonically non-decreasing as the signal moves further from threshold.
4. Severity zones use fixed 10-point buckets (not percentile-based), chosen for interpretability and debuggability.

### Why Fixed Zones Instead of Dynamic

Dynamic zone boundaries (e.g., percentile-based on historical data) were considered and rejected:
- They require a warm-up period before producing meaningful severity.
- They add state to what is currently a stateless evaluator (violating DBI-4).
- They make debugging harder — the same RSI value could produce different severities at different times.
- The 10-point bucket approach is transparent, reproducible, and sufficient for the current use case.

## Confidence Model (Unchanged)

For reference, the confidence model remains as established in the first slice:

| RSI Range      | Outcome         | Confidence Formula                          | Range       |
|----------------|-----------------|---------------------------------------------|-------------|
| RSI < 30       | `triggered`     | `0.5 + 0.5 * (30 - RSI) / 30`              | [0.5, 1.0]  |
| RSI >= 30      | `not_triggered` | `0.5 + 0.5 * (RSI - 30) / 70`              | [0.5, ~0.85]|

Confidence and severity are complementary: confidence is a continuous scalar, severity is a discrete zone. A decision with `severity: low` and `confidence: 0.52` means the condition is mildly met with moderate uncertainty.

## Rationale Format

### Structure

The rationale is a single human-readable sentence describing the evaluation:

**Triggered**:
```
RSI {value} below oversold threshold {threshold} (distance {pct}%); severity {severity}
```

**Not Triggered**:
```
RSI {value} above oversold threshold {threshold}; not oversold
```

### Examples

| RSI   | Outcome         | Rationale |
|-------|-----------------|-----------|
| 25.00 | triggered       | `RSI 25.00 below oversold threshold 30.0 (distance 16.7%); severity low` |
| 15.00 | triggered       | `RSI 15.00 below oversold threshold 30.0 (distance 50.0%); severity moderate` |
| 5.00  | triggered       | `RSI 5.00 below oversold threshold 30.0 (distance 83.3%); severity high` |
| 65.00 | not_triggered   | `RSI 65.00 above oversold threshold 30.0; not oversold` |
| 30.00 | not_triggered   | `RSI 30.00 above oversold threshold 30.0; not oversold` |

### Design Principles

1. **Self-contained**: The rationale includes all values needed to reproduce the evaluation.
2. **Deterministic**: Same inputs always produce the same rationale string.
3. **Machine-parseable**: Follows a fixed format that can be regex-matched if needed.
4. **No prescriptive language**: Rationale describes the state, not what to do about it. It says "severity high", not "strong buy signal".

## Metadata Enrichment

The evaluator's metadata map now includes three keys:

| Key            | Example  | Description |
|----------------|----------|-------------|
| `threshold`    | `"30.0"` | The oversold threshold used |
| `rsi_zone`     | `"low"`  | The severity zone label |
| `distance_pct` | `"16.7"` | Percentage distance from threshold (0.0 if not triggered) |

## Non-Goals and Limits

1. **Severity is not a recommendation.** It does not imply position size, urgency, or direction. Those belong to strategy and risk.
2. **No severity for `not_triggered`.** The decision domain does not classify "how not-oversold" the market is beyond confidence.
3. **No composite severity.** Severity comes from a single signal value, not from combining multiple signals. Multi-signal classification belongs to strategy.
4. **No historical severity percentiles.** The evaluator remains stateless per DBI-4.
5. **Rationale is for observability, not for downstream logic.** Strategy resolvers should use outcome and confidence, not parse rationale strings.
