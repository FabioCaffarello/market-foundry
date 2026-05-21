# Stage S272 — Execution Analytical Round-Trip Proof — Report

**Stage**: S272
**Objective**: Prove the analytical round-trip for execution paper_order events.
**Status**: COMPLETE
**Date**: 2026-03-21

## Executive Summary

S272 closes the analytical debt registered in S269: the execution paper serialization round-trip
through the NATS → writer → ClickHouse → reader → HTTP path is now proved at the write→read
serialization level. Nine new test scenarios were added to the existing `behavioral_roundtrip_test.go`,
extending the four-stage chain proof from `decision → strategy → risk` to the full
`decision → strategy → risk → execution` path. Zero regressions.

## Round-Trip Validated

```
PaperOrderSubmittedEvent
  → mapExecutionRow()        [write: domain → 20-column []any row]
  → FormatFloat / marshalJSON [serialization]
  → ParseRiskInputJSON / ParseFillsJSON / ParseMetadataJSON / FormatFloat [deserialization]
  → ExecutionIntent          [read: reconstructed domain type]
```

**Proven**: all 20 columns, 3 Side enum values, 7 Status enum values, RiskInput with strategy-type-aware
fields, FillRecord arrays, quantity precision, and the full four-stage causation chain survive the round-trip.

## Files Changed

| File | Change |
|------|--------|
| `internal/adapters/clickhouse/writerpipeline/behavioral_roundtrip_test.go` | Added 9 execution round-trip test scenarios (Scenarios 9–17) |
| `docs/architecture/execution-analytical-round-trip-proof.md` | New: detailed proof documentation |
| `docs/architecture/execution-analytical-surface-findings-and-queryability.md` | New: surface description and queryability findings |
| `docs/stages/stage-s272-execution-analytical-round-trip-proof-report.md` | New: this report |

## Evidence and Key Findings

### Test Results

```
=== RUN   TestBehavioralRoundTrip_Execution_BasicPaperOrder          — PASS
=== RUN   TestBehavioralRoundTrip_Execution_SideEnumValues           — PASS (3 subtests)
=== RUN   TestBehavioralRoundTrip_Execution_StatusEnumValues         — PASS (7 subtests)
=== RUN   TestBehavioralRoundTrip_Execution_RiskCausalContext_CounterTrend — PASS
=== RUN   TestBehavioralRoundTrip_Execution_RiskCausalContext_ProTrend    — PASS
=== RUN   TestBehavioralRoundTrip_Execution_MultipleFills            — PASS
=== RUN   TestBehavioralRoundTrip_Execution_EmptyFills               — PASS
=== RUN   TestBehavioralRoundTrip_Execution_QuantityPrecision        — PASS (6 subtests)
=== RUN   TestBehavioralRoundTrip_FullChain_DecisionToExecution      — PASS
=== RUN   TestBehavioralRoundTrip_Execution_RejectedOrder            — PASS
```

Total: 9 new test functions, 26 sub-test cases, all PASS.

### Key Findings

1. **Dual ID channels work correctly**: Execution has both envelope `correlation_id`/`causation_id` (shared event metadata) and domain-specific `exec_correlation_id`/`exec_causation_id`. Both are independently serialized and survive the round-trip.

2. **RiskInput S265 fields survive**: The `strategy_type` and `decision_severity` fields added in S265 for cross-boundary causal traceability are correctly serialized in the `risk` JSON column and deserialized by `ParseRiskInputJSON`.

3. **Full four-stage chain is coherent**: The `decision → strategy → risk → execution` causation chain maintains:
   - Shared correlation ID across all four stages
   - Proper causation chain (each stage's causation_id = previous stage's event_id)
   - Confidence ordering: `risk ≤ strategy ≤ decision`
   - Decision severity and strategy type propagation end-to-end

4. **No code changes required**: The existing `mapExecutionRow`, `ParseRiskInputJSON`, `ParseFillsJSON`, and `FormatFloat` functions were already correct. S272 only added tests to prove what was already working but unverified.

## Remaining Limits

- **Live integration**: The round-trip is proved at the serialization level, not against a running ClickHouse. Live proof exists in `smoke-analytical-e2e.sh` Phase 5.8.
- **Venue fills**: `VenueOrderFilledEvent` and its separate NATS stream are not covered by the analytical round-trip. This is a separate family not yet in the writer pipeline.
- **Aggregation**: No GROUP BY, time-bucket, or statistical aggregation queries are proved. The surface provides row-level queryability.
- **Cross-domain joins**: SQL-level correlation_id joins across `executions`, `risk_assessments`, `strategies`, `decisions` tables were not tested.

## Preparation for S273

With the analytical round-trip now closed for all four core pipeline stages
(decision, strategy, risk, execution), recommended next steps:

1. **Closed-loop observability**: Consider whether the existing `smoke-analytical-e2e.sh`
   provides sufficient live coverage or if a dedicated analytical integration test
   should run in CI.
2. **Venue fill analytical path**: If venue execution becomes a priority, extend the
   writer pipeline and round-trip tests to `VenueOrderFilledEvent`.
3. **Cross-domain queryability**: Prove that correlation_id-based joins across tables
   reconstruct the full chain at the SQL level.
4. **Transition gate**: Evaluate whether the paper execution wave debts are now
   sufficiently closed for a transition gate to the next wave.
