# Decision Domain Deepening — S234

## Overview

S234 deepens the decision domain by adding two first-class semantic fields — **Severity** and **Rationale** — and enriching evaluator metadata with zone classification and distance metrics. This evolution keeps the domain small, controlled, and fully explainable.

## Changes

### 1. Domain Model (`decision.Decision`)

Two new fields added to the shared Decision struct:

| Field       | Type     | Purpose                                                        |
|-------------|----------|----------------------------------------------------------------|
| `Severity`  | Severity | Graduated classification of how extreme the evaluated condition is |
| `Rationale` | string   | Human-readable explanation of why this outcome was reached      |

Severity is an enum with four values: `none`, `low`, `moderate`, `high`.

- **Triggered decisions** always carry a non-none severity.
- **Not-triggered decisions** always carry `severity: none`.
- The field is validated at the domain level (unknown values rejected).
- Empty severity is allowed for backward compatibility during migration.

### 2. Evaluator Enrichment (`RSIOversoldEvaluator`)

The evaluator now produces:

- **Severity**: Based on distance from threshold in 10-point zones:
  - `low`: 0–10 points below threshold (RSI 20–30)
  - `moderate`: 10–20 points below threshold (RSI 10–20)
  - `high`: 20+ points below threshold (RSI < 10)
- **Rationale**: Structured sentence explaining the evaluation, e.g.:
  - `"RSI 25.00 below oversold threshold 30.0 (distance 16.7%); severity low"`
  - `"RSI 65.00 above oversold threshold 30.0; not oversold"`
- **Enriched metadata**:
  - `rsi_zone`: Same as severity label (mirrors the zone classification)
  - `distance_pct`: Percentage distance from threshold (0.0 for not-triggered)
  - `threshold`: (existing) The threshold value used

### 3. ClickHouse Schema

New migration `007_add_decision_severity_rationale.sql`:
- `severity LowCardinality(String) DEFAULT ''` — after `confidence`
- `rationale String DEFAULT ''` — after `severity`

Both columns use `DEFAULT ''` for backward compatibility with existing rows.

### 4. Pipeline Alignment

Updated across the full derive → store → read path:

| Component | Change |
|-----------|--------|
| Writer pipeline `mapDecisionRow` | Emits severity and rationale columns |
| ClickHouse reader `QueryDecisionHistory` | Scans severity and rationale |
| `BuildDecisionQuery` | SELECT includes severity and rationale |
| KV store (NATS) | Automatic via JSON marshal/unmarshal |
| HTTP response | Automatic via JSON struct tags |

### 5. Codegen

Updated `rsi_oversold.yaml` columns list and golden snapshot to include `severity, rationale`.

## What Did NOT Change

- No new evaluator families were introduced.
- No new NATS streams or subjects were created.
- The Outcome enum (`triggered`, `not_triggered`, `insufficient`) is unchanged.
- The confidence calculation is unchanged.
- Signal → Decision boundary (DBI-9) is preserved.
- Strategy resolvers continue consuming decision primitives — they do not use severity or rationale.

## Downstream Impact

- **Strategy resolvers**: Unaffected. They consume `decisionEvaluatedMessage` which carries outcome/confidence/type as primitives. Severity and rationale are not forwarded to strategy.
- **HTTP consumers**: Will see `severity` and `rationale` in JSON responses immediately.
- **ClickHouse queries**: Can filter by severity once the migration is applied.
- **KV store**: New decisions will include severity and rationale. Old entries remain valid (empty strings on new fields).

## Precedent

The `Rationale` field follows the pattern already established by `risk.RiskAssessment.Rationale`, which carries a human-readable explanation of risk assessment outcomes.
