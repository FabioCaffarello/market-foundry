# Multi-Symbol Explainability Findings and Limitations

> Phase 29 — Multi-Symbol Operational Scaling Wave
> Stage: S303
> Date: 2026-03-21

## Purpose

This document catalogs every finding, confirmed strength, and residual limitation discovered during S303 validation of the composite explainability surfaces under multi-symbol load.

## Confirmed Strengths

### S1: Symbol Isolation Is Complete

Every query path — single chain, batch, funnel, dispositions — enforces `WHERE symbol = ?` at the ClickHouse query level. The S301 fix is verified to hold under multi-symbol concurrent data.

**Evidence:** OBS-3 (causal metadata integrity), OBS-4 (filter specificity), and all S302 integration test criteria.

### S2: Attribution Is Symbol-Specific

Each symbol's attribution carries its own disposition, rationale, constraints, and strategy context. No field contains data from another symbol. Operators reading attribution for btcusdt see only btcusdt's risk gate outcome.

**Evidence:** OBS-5 (attribution readability) validates all attribution fields per symbol.

### S3: Cross-Surface Consistency Holds

Chain completeness, funnel counts, and disposition totals are mutually consistent per symbol:
- A symbol with 3 approved chains shows execution_count=3 in funnel and approved_count=3 in dispositions.
- A symbol with 1 rejected chain shows risk_count > execution_count in funnel and rejected_count=1 in dispositions.

**Evidence:** OBS-1 (funnel-chain consistency), OBS-2 (disposition-attribution coherence).

### S4: Causal DAG Is Internally Consistent

Within each chain, the causation_id references form a valid parent→child DAG:
```
signal (root) → decision → strategy → risk → execution
```
No cross-chain or cross-symbol causation_id references exist.

**Evidence:** OBS-3 tests validate the complete causal chain for all 3 symbols.

### S5: JSON Serialization Is Structurally Complete

HTTP responses include all expected fields: `chains`, `source`, `meta`, and within each chain: `correlation_id`, `stage_count`, `chain_complete`, `missing_stages`, `attribution`. Raw JSON inspection confirms no fields are silently dropped.

**Evidence:** HTTP-OBS-2 (attribution completeness) inspects raw JSON field presence.

## Residual Limitations

### L1: No Cross-Symbol Aggregate View

**Description:** There is no endpoint that returns aggregated data across all symbols (e.g., "total executions across btcusdt + ethusdt + solusdt"). Each query is scoped to exactly one symbol.

**Impact:** An operator wanting a cross-symbol overview must issue N queries (one per symbol) and aggregate client-side.

**Severity:** Low. This is a deliberate design choice — cross-symbol aggregation risks hiding per-symbol anomalies. The current design forces per-symbol investigation.

**Remediation:** NOT RECOMMENDED. Cross-symbol aggregation is a Phase 29 non-goal (NG-3: no portfolio-level aggregation).

### L2: Batch Discovery Is Execution-Rooted

**Description:** `/chains` starts from the executions table. Chains terminated before execution (rejected at risk stage) are not discoverable via batch queries. They require either:
- Single-chain lookup by known correlation_id.
- Indirect detection via funnel (risk_count > execution_count).

**Impact:** Moderate for ad-hoc investigation of rejected chains without known correlation_ids.

**Severity:** Medium. Compensated by funnel aggregate view.

**Remediation:** A future signal-rooted or risk-rooted batch endpoint could address this. Deferred — not in scope for S300–S305.

### L3: `type` Parameter Creates Query Fragmentation

**Description:** Funnel and disposition queries require a `type` parameter (signal type). When multiple signal types are active for the same symbol (e.g., "rsi" and "bollinger" both triggering for btcusdt), the operator must issue separate queries per type.

**Impact:** Low with current 3 signal families. Increases linearly with new signal types.

**Severity:** Low. Each type/source/symbol/timeframe combination represents a distinct pipeline path — mixing them in one funnel would be semantically incorrect.

**Remediation:** None needed. The fragmentation is semantically correct.

### L4: Per-Constraint Trigger Identification Missing

**Description:** `attribution.active_constraints` shows all constraints that were active during risk assessment, but does not identify which constraint caused a rejection or modification. The `rationale` field is free text.

**Impact:** Low with 3 constraints (max_position_size, max_exposure, stop_distance). Higher with future constraint additions.

**Severity:** Low. The rationale field typically names the triggering constraint in practice.

**Remediation:** Requires write-side `triggering_constraints` field on risk.RiskAssessment. Deferred since S299 (GAP-Q2-A).

### L5: No Temporal Ordering Stress Under Sub-Millisecond Concurrency

**Description:** All tests use `time.Now()` for timestamps, which does not stress sub-millisecond ordering of events from different symbols arriving at the same instant.

**Impact:** Low. ClickHouse MergeTree ORDER BY `(source, symbol, timeframe, type, timestamp)` provides deterministic ordering within a partition. Sub-millisecond collisions within the same symbol/type combination are resolved by insertion order.

**Severity:** Very low. Paper mode does not produce sub-millisecond event rates.

**Remediation:** S304 resource scaling measurement may indirectly surface ordering anomalies under load. No dedicated stress test needed at this stage.

### L6: Three Symbols Only

**Description:** Validation covers btcusdt, ethusdt, solusdt. The system has not been tested with >3 symbols.

**Impact:** Low. The isolation mechanism (WHERE symbol = ?) is symbol-count agnostic. Adding a 4th symbol does not change the query pattern.

**Severity:** Very low. The architecture scales linearly — each additional symbol adds independent rows, not cross-symbol complexity.

**Remediation:** Future waves may expand to more symbols as needed.

## Governing Question Coverage (MQ1–MQ7)

| MQ | Question | S303 Contribution | Status |
|----|----------|--------------------|--------|
| MQ1 | Symbol isolation | Confirmed via OBS-3, OBS-4 | FULL |
| MQ2 | Chain correctness per symbol | Confirmed via OBS-2, OBS-5 | FULL |
| MQ3 | Batch query symbol scoping | Confirmed via OBS-6 | FULL |
| MQ4 | Funnel accuracy per symbol | Confirmed via OBS-1, OBS-4 | FULL |
| MQ5 | Disposition accuracy per symbol | Confirmed via OBS-1, OBS-4 | FULL |
| MQ6 | Ordering and consistency | Confirmed via OBS-3 (causal DAG); L5 noted for sub-ms | SUBSTANTIAL |
| MQ7 | Resource scaling | Out of S303 scope (S304 target) | DEFERRED |

## Summary

The S294–S299 composite observability wave is **validated under multi-symbol load**. All explainability surfaces produce correct, isolated, and readable results. Six residual limitations are documented — none require immediate code changes, and none compromise the integrity of the existing surfaces.
