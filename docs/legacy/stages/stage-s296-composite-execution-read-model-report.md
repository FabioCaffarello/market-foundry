# Stage S296: Composite Execution Read Model Report

**Date:** 2026-03-21
**Status:** Complete
**Predecessor:** S295 (Correlation/Causation Spine Validation)

## Objective

Design, implement, and validate a composite execution read model that unifies the 5 ClickHouse domain tables (signals, decisions, strategies, risk_assessments, executions) through the correlation_id spine, enabling coherent reconstruction of any execution's causal chain.

## Executive Summary

S296 delivers the composite execution read model as application-side composition over 5 independent ClickHouse queries. The model closes S295 gap G1 (readers skipping causal metadata) by introducing `*WithTrace` projection types that extend domain structs with event_id, correlation_id, causation_id, and occurred_at.

The implementation is minimal and pragmatic: one new reader (5-table composition), one use case (validation + dispatch), one contract file (canonical types), and 13 unit tests + 6 integration test criteria. No ClickHouse JOINs, no materialized views, no write-side changes.

## Deliverables

| Deliverable | Path | Status |
|-------------|------|--------|
| Composite reader | `internal/adapters/clickhouse/composite_reader.go` | Added |
| Composite contracts | `internal/application/analyticalclient/composite_contracts.go` | Added |
| Composite use case | `internal/application/analyticalclient/get_composite_chain.go` | Added |
| Use case unit tests | `internal/application/analyticalclient/get_composite_chain_test.go` | Added (9 tests) |
| Reader unit tests | `internal/adapters/clickhouse/composite_reader_test.go` | Added (4 tests) |
| Integration tests | `internal/adapters/clickhouse/composite_reader_integration_test.go` | Added (6 criteria) |
| Architecture doc | `docs/architecture/composite-execution-read-model-over-five-clickhouse-tables.md` | Added |
| Semantics doc | `docs/architecture/composite-read-model-semantics-ordering-and-limitations.md` | Added |
| Stage report | `docs/stages/stage-s296-composite-execution-read-model-report.md` | This file |

## Design Decisions

### Application-Side Composition (Not ClickHouse JOINs)

**Chosen:** 5 independent queries by correlation_id, assembled in Go.

**Rationale:**
1. Each query is a simple point lookup — no complex JOIN plans.
2. Partial chain assembly is natural — a failed query for one stage does not prevent results.
3. No coupling between tables at the SQL level.
4. Aligns with ClickHouse's strengths (fast point reads, not OLTP JOINs).

### WithTrace Projection Types

**Chosen:** New `SignalWithTrace`, `DecisionWithTrace`, etc. that embed the domain struct + add causal metadata.

**Rationale:**
- Closes S295 gap G1 without modifying existing domain structs.
- Domain structs remain pure (no trace concerns).
- Composite reader is the only consumer of trace metadata at the read layer.

### LIMIT 1 Per Stage

Each stage query returns the most recent event for a correlation_id. This assumes 1:1 cardinality, which holds across all 3 proven slices. Documented as a known limitation if fan-out is introduced.

## Test Results

### Unit Tests (13 total, all pass)

```
=== RUN   TestGetCompositeChain_Single_FullChain       --- PASS
=== RUN   TestGetCompositeChain_Single_EmptyResult     --- PASS
=== RUN   TestGetCompositeChain_Single_ReaderError     --- PASS
=== RUN   TestGetCompositeChain_Batch_Success          --- PASS
=== RUN   TestGetCompositeChain_Batch_MissingSource    --- PASS
=== RUN   TestGetCompositeChain_Batch_MissingSymbol    --- PASS
=== RUN   TestGetCompositeChain_Batch_InvalidTimeframe --- PASS
=== RUN   TestGetCompositeChain_Batch_LimitClamping    --- PASS
=== RUN   TestGetCompositeChain_NilUseCase             --- PASS
=== RUN   TestComputeChainCompleteness_AllPresent      --- PASS
=== RUN   TestComputeChainCompleteness_PartialChain    --- PASS
=== RUN   TestComputeChainCompleteness_Empty           --- PASS
=== RUN   TestComputeChainCompleteness_RiskRejected    --- PASS
```

### Integration Test Criteria (requires live ClickHouse)

| ID | Criterion | Status |
|----|-----------|--------|
| CRI-1 | Full chain reconstruction by correlation_id | Covered |
| CRI-2 | Causal metadata preservation across all stages | Covered |
| CRI-3 | Domain fields survive composite round-trip | Covered |
| CRI-4 | Partial chain (risk-rejected) correctly represented | Covered |
| CRI-5 | Batch lookup ordered by execution timestamp DESC | Covered |
| CRI-6 | Missing correlation_id returns empty chain | Covered |

### Regression Check

All existing tests pass with zero regressions:
- `internal/adapters/clickhouse` — PASS
- `internal/application/analyticalclient` — PASS
- All other modules — unaffected (no imports changed)
- Gateway binary compiles cleanly

## Gap Closure

| Gap | Source | Status |
|-----|--------|--------|
| G1: Readers skip causal metadata | S295 | **Closed** — `*WithTrace` types expose event_id, correlation_id, causation_id, occurred_at |
| G2: Signal empty causation_id | S295 | Unchanged — by design (signal is chain root) |

## Governing Questions Progress

| ID | Question | S296 Status |
|----|----------|-------------|
| Q1 | Why was execution X submitted? | **Answerable** via single chain lookup |
| Q2 | Why was execution X rejected? | **Partially answerable** — risk stage shows disposition + constraints |
| Q3 | Which signals contributed to decision D? | **Answerable** via decision.signals + causal chain |
| Q4 | Confidence/severity flow? | **Answerable** — each stage carries confidence; decision carries severity |
| Q5 | Why did symbol stop? | Deferred to S297/S298 |
| Q6 | Blocked vs approved count? | Deferred to S298 |
| Q7 | Conversion rate per stage? | Deferred to S298 |

## Acceptance Criteria Checklist

- [x] Composite read model exists and is canonical
- [x] Composition over 5 tables works via application layer
- [x] Causal metadata (event_id, correlation_id, causation_id) is exposed at read time
- [x] Full and partial chains are correctly assembled
- [x] Batch lookup returns ordered results
- [x] Solution is small, pragmatic, and proportional
- [x] Zero regressions in existing tests

## Guard Rails Compliance

- [x] Did not open BI/reporting
- [x] Did not create generic observability platform
- [x] Did not hide ordering/consistency gaps (documented in semantics doc)
- [x] Did not introduce write-side changes
- [x] Did not open non-goals from S294

## Limitations

1. **No HTTP endpoint yet** — the composite model is accessible via Go API only. HTTP exposure is S297.
2. **No aggregation** — individual chains only, not funnel metrics. Aggregation is S298.
3. **Eventual consistency** — write lag between tables may cause temporarily incomplete chains.
4. **1:1 cardinality assumed** — fan-out not modeled.
5. **No evidence-to-signal link** — chain starts at signal (architectural boundary).

## Preparation for S297

S297 (Explainability Query Surface) should:

1. **Wire the composite reader + use case into the gateway** — add `newAnalyticalCompositeReader` to `cmd/gateway/analytical_reader.go` and wire through `compose.go`.
2. **Add HTTP endpoint** — `GET /analytical/composite/chain?correlation_id=...` and `GET /analytical/composite/chains?source=...&symbol=...&timeframe=...`.
3. **Add the composite use case to AnalyticalFamilyDeps** and route registration.
4. **Response format** — nest the full `CompositeExecutionChain` as the JSON response body.
5. **Filter extensions** — consider adding `outcome` (for Q2) and `disposition` (for risk-filtered views) to the batch query.
