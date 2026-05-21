# Stage S303: Composite Observability Under Multi-Symbol Load

> Phase 29 — Multi-Symbol Operational Scaling Wave
> Date: 2026-03-21
> Status: **COMPLETE**

## Objective

Validate and harden the composite explainability surfaces (chain, funnel, dispositions, attribution) to ensure they remain correct, readable, and operationally useful when multiple symbols coexist simultaneously in the analytical store.

## Governing Constraint

This stage is a VALIDATION of existing surfaces under multi-symbol pressure, NOT a new observability platform. No new endpoints, no dashboards, no real-time extensions.

## Surfaces Validated

| Surface | Endpoint | Multi-Symbol Behavior | Result |
|---------|----------|-----------------------|--------|
| Single Chain | `/composite/chain` | Symbol-isolated causal chain | CORRECT |
| Batch Chains | `/composite/chains` | Symbol-scoped batch with mixed dispositions | CORRECT |
| Pipeline Funnel | `/composite/funnel` | Per-symbol monotonic stage counts | CORRECT |
| Dispositions | `/composite/dispositions` | Per-symbol breakdown, percentages sum to 100% | CORRECT |
| Attribution | Embedded in chain | Per-symbol disposition, rationale, constraints, strategy context | CORRECT |

## Test Evidence

### Use Case Layer — 18/18 PASS

File: `internal/application/analyticalclient/composite_observability_multi_symbol_test.go`

| Test | Subtests | Status |
|------|----------|--------|
| TestS303_OBS1_FunnelChainConsistency | btcusdt, ethusdt, solusdt | PASS |
| TestS303_OBS2_DispositionAttributionCoherence | btcusdt, ethusdt, solusdt | PASS |
| TestS303_OBS3_CausalMetadataIntegrity | btcusdt, ethusdt, solusdt | PASS |
| TestS303_OBS4_FilterSpecificity | btcusdt, ethusdt, solusdt | PASS |
| TestS303_OBS5_AttributionReadability | btcusdt, ethusdt, solusdt | PASS |
| TestS303_OBS6_BatchExplainability | btcusdt, ethusdt | PASS |

### Handler Layer — 9/9 PASS

File: `internal/interfaces/http/handlers/composite_observability_multi_symbol_test.go`

| Test | Subtests | Status |
|------|----------|--------|
| TestS303_HTTP_OBS1_CrossSurfaceStructure | btcusdt, ethusdt, solusdt | PASS |
| TestS303_HTTP_OBS2_AttributionCompleteness | btcusdt, ethusdt, solusdt | PASS |
| TestS303_HTTP_OBS3_SequentialQueryIndependence | btcusdt, ethusdt | PASS |

### Regression — Zero

All pre-existing tests in `analyticalclient` and `handlers` continue to pass.

## Files Changed

| File | Change |
|------|--------|
| `internal/application/analyticalclient/composite_observability_multi_symbol_test.go` | NEW — 6 test functions, 18 subtests |
| `internal/interfaces/http/handlers/composite_observability_multi_symbol_test.go` | NEW — 3 test functions, 9 subtests |
| `docs/architecture/composite-observability-under-multi-symbol-load.md` | NEW — surface validation findings |
| `docs/architecture/multi-symbol-explainability-findings-and-limitations.md` | NEW — confirmed strengths and residual limitations |
| `docs/stages/stage-s303-composite-observability-under-multi-symbol-load-report.md` | NEW — this report |

## Key Findings

### Confirmed

1. **Symbol isolation is complete** — every query path enforces `WHERE symbol = ?` at ClickHouse level.
2. **Cross-surface consistency holds** — funnel counts, disposition totals, and chain completeness are mutually consistent per symbol.
3. **Causal DAG is internally valid** — causation_id references form a correct parent→child chain within each symbol.
4. **Attribution is fully readable** — all fields (disposition, rationale, constraints, strategy context) populated per symbol.
5. **S294–S299 wave is valid under multi-symbol** — no code changes required.

### Residual Limitations

| ID | Limitation | Severity | Status |
|----|-----------|----------|--------|
| L1 | No cross-symbol aggregate view | Low | By design (NG-3) |
| L2 | Batch discovery is execution-rooted | Medium | Known since S299 (GAP-Q5-A) |
| L3 | `type` parameter creates query fragmentation | Low | Semantically correct |
| L4 | Per-constraint trigger identification missing | Low | Known since S299 (GAP-Q2-A) |
| L5 | No sub-millisecond ordering stress test | Very Low | Paper mode insufficient rate |
| L6 | Three symbols only | Very Low | Architecture is symbol-count agnostic |

## MQ Coverage Contribution

| MQ | S303 Status |
|----|-------------|
| MQ1 (Symbol isolation) | FULL — confirmed |
| MQ2 (Chain correctness) | FULL — confirmed |
| MQ3 (Batch scoping) | FULL — confirmed |
| MQ4 (Funnel accuracy) | FULL — confirmed |
| MQ5 (Disposition accuracy) | FULL — confirmed |
| MQ6 (Ordering consistency) | SUBSTANTIAL — causal DAG validated, sub-ms gap noted |
| MQ7 (Resource scaling) | DEFERRED — S304 target |

## Non-Goals Respected

- No new HTTP endpoints added.
- No dashboard or Grafana integration.
- No real-time streaming.
- No write-side schema changes.
- No cross-symbol aggregation.
- No performance optimization.

## Preparation for S304

S303 confirms that the **read-side surfaces are functionally correct** under multi-symbol load. S304 should focus on:

1. **Resource scaling measurement** — goroutine count, memory allocation, and ClickHouse query latency under 3× symbol load vs single-symbol baseline.
2. **Batch query N+1 characterization** — the batch path issues 1 + (5 × N) queries for N chains. Measure whether this remains acceptable at realistic batch sizes (20, 50, 100) across 3 symbols.
3. **MQ7 closure** — provide quantitative evidence that resource consumption scales proportionally (not exponentially) with symbol count.
