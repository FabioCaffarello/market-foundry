# Decision-to-Strategy Behavior Activation

**Stage:** S250
**Charter:** BEHAVIORAL-WAVE-1
**Date:** 2026-03-21
**Status:** Active

---

## 1. Purpose

This document describes the behavioral activation of the decision→strategy boundary, implemented in S250. Prior to this activation, strategy resolvers consumed decision context (severity, rationale, type) purely for traceability — the fields flowed through but did not influence resolution logic. After this activation, decision severity directly influences strategy confidence, parameters, and rationale.

---

## 2. What Changed

### 2.1 Before S250 (Traceability-Only)

```
Decision (severity=high, confidence=0.9000)
  → Strategy (confidence=0.9000, target_offset=0.02, stop_offset=0.01)
    [severity stored in metadata but ignored by logic]
```

Strategy confidence was a direct copy of decision confidence. Parameters were static constants. Severity and rationale were carried forward in `DecisionInput` and metadata for observability only.

### 2.2 After S250 (Behavioral)

```
Decision (severity=high, confidence=0.9000)
  → Strategy (confidence=0.9000, target_offset=0.03, stop_offset=0.01)
    [severity actively scales confidence and adjusts parameters]

Decision (severity=low, confidence=0.9000)
  → Strategy (confidence=0.7200, target_offset=0.01, stop_offset=0.01)
    [weak signal → reduced confidence, smaller targets]
```

Strategy confidence is scaled by severity. Parameters are adjusted by severity. The strategy rationale explains the behavioral adjustments made.

---

## 3. Behavioral Semantics

### 3.1 Confidence Scaling

Decision severity applies a multiplicative scaling factor to the raw decision confidence:

| Severity | Scaling Factor | Semantic Meaning |
|----------|---------------|------------------|
| `high` | ×1.00 | Full confidence — extreme signal confirms conviction |
| `moderate` | ×0.90 | Slight reduction — normal signal strength |
| `low` | ×0.80 | Meaningful reduction — weak signal, lower conviction |
| unknown/empty | ×1.00 | Neutral — no scaling applied (backward compatible) |

The scaling factors are **identical across both resolvers** for consistency. They are defined per-resolver to allow future type-specific tuning without cross-resolver coupling.

### 3.2 Parameter Adjustment

Each resolver adjusts its type-specific parameters based on severity:

#### mean_reversion_entry

| Parameter | Base Value | High (×) | Moderate (×) | Low (×) | Semantic |
|-----------|-----------|----------|-------------|---------|----------|
| `target_offset` | 0.02 | 1.50 | 1.00 | 0.75 | Extreme oversold → expect bigger reversion |
| `stop_offset` | 0.01 | 0.75 | 1.00 | 1.50 | Extreme oversold → tighter stop (higher conviction) |

#### trend_following_entry

| Parameter | Base Value | High (×) | Moderate (×) | Low (×) | Semantic |
|-----------|-----------|----------|-------------|---------|----------|
| `trailing_stop_pct` | 0.03 | 0.75 | 1.00 | 1.50 | Strong trend → ride closer (higher conviction) |
| `take_profit_pct` | 0.05 | 1.50 | 1.00 | 0.75 | Strong trend → expect bigger move |

### 3.3 Strategy Rationale

Each strategy now produces a human-readable rationale explaining the behavioral adjustments:

```
# Triggered with severity adjustment:
"mean_reversion_entry triggered by rsi_oversold (severity high); confidence 0.9000→0.9000; params adjusted [0.03, 0.01]"

# Triggered without severity:
"mean_reversion_entry triggered by rsi_oversold; confidence 0.9000 (no severity adjustment); params [0.02, 0.01]"

# Not triggered:
"decision rsi_oversold not_triggered; no entry signal for mean reversion"
```

---

## 4. Decision Context in Strategy Metadata

After S250, strategy metadata carries enriched decision context:

| Key | When Present | Value |
|-----|-------------|-------|
| `decision_type` | Always | The decision type that triggered this strategy (e.g., `rsi_oversold`) |
| `decision_severity` | Always | The decision severity (e.g., `high`, `moderate`, `low`, `none`, `""`) |
| `decision_rationale` | When non-empty | The decision's human-readable rationale |
| `rationale` | Always | The strategy's own rationale explaining behavioral adjustments |
| `reason` | Only for `insufficient` | `"insufficient_data"` |

---

## 5. Design Decisions

### 5.1 Why Multiplicative Scaling

Multiplicative scaling (confidence × factor) preserves the relative ordering of confidence values within a severity tier. A decision with 0.90 confidence at moderate severity always produces higher strategy confidence than a decision with 0.60 confidence at the same severity.

### 5.2 Why Same Scaling Across Resolvers

Both resolvers use the same confidence scaling factors (1.00/0.90/0.80) for consistency. The parameter multipliers differ because mean reversion and trend following have opposite risk profiles (counter-trend vs. pro-trend).

### 5.3 Why Unknown Severity Defaults to Neutral

Unknown or empty severity values produce a ×1.00 scaling factor (no change). This ensures backward compatibility — if a future decision type doesn't set severity, the strategy resolver produces identical behavior to the pre-S250 baseline.

### 5.4 Why Rationale in Metadata (Not a Strategy Field)

The `Strategy` domain struct does not have a `Rationale` field. Adding one would be a domain model change that affects all consumers (store, writer, gateway, ClickHouse schema). Storing rationale in `metadata["rationale"]` achieves the same observability without schema changes.

### 5.5 Why DecisionInput Preserves Raw Confidence

`DecisionInput.Confidence` stores the original (raw) decision confidence, not the scaled value. This preserves the audit trail: the reader can see what the decision produced and what the strategy did with it.

---

## 6. Invariants Preserved

| Invariant | Status |
|-----------|--------|
| Domain isolation (no cross-domain imports) | Preserved — all data flows as primitive strings |
| Single-writer per stream | Preserved — no new streams |
| Acyclic data flow | Preserved — no feedback loops |
| Pure application logic (no I/O in resolvers) | Preserved — all changes are pure functions |
| Envelope uniformity | Preserved — no envelope changes |
| Configctl drives activation | Preserved — no activation changes |

---

## 7. Relationship to Charter

This document fulfills Tier 1 of the BEHAVIORAL-WAVE-1 charter: "Decision → Strategy Integration." The specific behavioral targets achieved:

- Decision severity influences strategy confidence (**confidence scaling**)
- Decision severity influences strategy parameters (**parameter adjustment**)
- Decision type is explicit in strategy metadata (**type awareness**)
- Strategy produces its own rationale (**behavioral explainability**)
- Existing 1:1 chains continue to work (**backward compatibility**)

Multi-decision strategy input (consuming decisions from multiple evaluators) is deferred to a future stage if the charter requires it.
