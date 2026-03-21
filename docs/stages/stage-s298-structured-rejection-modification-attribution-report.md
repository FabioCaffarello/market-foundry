# Stage S298 — Structured Rejection/Modification Attribution Report

> Wave: Composite Execution Observability (S294–S299)
> Block: 4 of 5
> Status: **Complete**
> Predecessor: S297 (HTTP Explainability Query Surface)

## Objective

Complete Q2 with structured risk attribution, unlock Q6 with disposition aggregation, unlock Q7 with pipeline funnel counts, and improve Q5 with stage-level visibility — all without write-side changes.

## Deliverables

### Code

| File | Type | Description |
|------|------|-------------|
| `internal/application/analyticalclient/composite_contracts.go` | Modified | Added RiskAttribution, AttributionStrategyContext, PipelineFunnelQuery/Reply, DispositionBreakdownQuery/Reply, StageFunnelCount, DispositionCount |
| `internal/application/analyticalclient/get_composite_chain.go` | Modified | Added computeAttribution(), called on single and batch chains |
| `internal/application/analyticalclient/get_pipeline_funnel.go` | New | GetPipelineFunnelUseCase with AggregationReader interface |
| `internal/application/analyticalclient/get_disposition_breakdown.go` | New | GetDispositionBreakdownUseCase with percentage computation |
| `internal/application/analyticalclient/get_pipeline_funnel_test.go` | New | 6 unit tests |
| `internal/application/analyticalclient/get_disposition_breakdown_test.go` | New | 5 unit tests |
| `internal/application/analyticalclient/get_composite_chain_test.go` | Modified | Added 3 attribution tests, enriched fullChain fixture |
| `internal/adapters/clickhouse/composite_reader.go` | Modified | Added QueryPipelineFunnel() and QueryDispositionBreakdown() methods |
| `internal/interfaces/http/handlers/composite.go` | Modified | Added GetFunnel, GetDispositions methods; struct-based DI via CompositeHandlerDeps |
| `internal/interfaces/http/handlers/composite_test.go` | Modified | Added 7 new tests (funnel, dispositions, attribution); updated to struct DI |
| `internal/interfaces/http/routes/analytical.go` | Modified | Added funnel/disposition deps and route registration |
| `cmd/gateway/analytical_reader.go` | Modified | Return concrete *CompositeReader for dual-interface satisfaction |
| `cmd/gateway/compose.go` | Modified | Wired funnel + disposition use cases |

### Documentation

| File | Description |
|------|-------------|
| `docs/architecture/structured-rejection-modification-attribution-and-aggregate-explainability.md` | Attribution design, aggregation patterns, architectural rationale |
| `docs/architecture/risk-constraint-attribution-aggregation-and-operational-limits.md` | Constraint model, endpoint specs, operational limits |
| `docs/stages/stage-s298-structured-rejection-modification-attribution-report.md` | This report |

## Endpoints

| Method | Path | Purpose | Governs |
|--------|------|---------|---------|
| GET | `/analytical/composite/chain` | Now includes `attribution` field | Q2 |
| GET | `/analytical/composite/chains` | Now includes `attribution` field | Q2 |
| GET | `/analytical/composite/funnel` | Pipeline stage counts | Q5, Q7 |
| GET | `/analytical/composite/dispositions` | Risk disposition breakdown | Q6 |

## Governing Questions Final Status

| Question | S297 Status | S298 Status | How |
|----------|-------------|-------------|-----|
| Q1 — Why executed? | Full | **Full** | Chain endpoint (unchanged) |
| Q2 — Why rejected/modified? | Partial | **Full** | `attribution` field on every chain: disposition + rationale + constraints + strategy context |
| Q3 — Signal inputs? | Full | **Full** | Chain endpoint (unchanged) |
| Q4 — Confidence flow? | Full | **Full** | Chain endpoint (unchanged) |
| Q5 — Pipeline break? | Partial | **Substantially improved** | Funnel endpoint shows stage counts; largest drop-off identifies bottleneck |
| Q6 — Blocked vs approved? | Deferred | **Full** | Dispositions endpoint: counts + percentages per disposition |
| Q7 — Conversion rate? | Deferred | **Full** | Funnel endpoint: counts per stage; consumer divides to get conversion rates |

## Test Results

### New Tests (S298)

| Test Suite | Tests | Status |
|------------|-------|--------|
| Handler: funnel | 3 (success, missing type, nil handler) | Pass |
| Handler: dispositions | 3 (success, missing type, nil handler) | Pass |
| Handler: attribution | 1 (rejection with full attribution) | Pass |
| Use case: pipeline funnel | 6 (success, missing fields, reader error, nil) | Pass |
| Use case: disposition breakdown | 5 (success, empty, missing type, reader error, nil) | Pass |
| Use case: attribution | 3 (single, no-risk, batch) | Pass |

### Existing Tests (S296/S297)

All 9 composite use case tests and 8 composite handler tests continue to pass.

**Total composite test count: 36** (15 handler + 21 use case)

## Architectural Decisions

1. **Read-side only.** Attribution is computed from existing risk fields at read time. Zero write-side changes.

2. **Use case layer attribution.** `computeAttribution()` runs in the use case layer, not the adapter. The reader stays pure data assembly.

3. **Dual-interface adapter.** `*clickhouse.CompositeReader` satisfies both `CompositeReader` (chain queries) and `AggregationReader` (funnel/disposition). One struct, two interfaces.

4. **Resilient funnel queries.** If one stage's count query fails, the stage returns 0 and the failure is logged. The overall funnel succeeds.

5. **Percentages computed in use case.** Disposition percentages are computed in `GetDispositionBreakdownUseCase`, not in ClickHouse. This keeps SQL simple and testing deterministic.

6. **No batch extension.** Batch lookup still starts from executions. The funnel endpoint provides aggregate visibility into pre-execution pipeline breaks, which is more operationally useful than listing individual broken chains.

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No dashboards | Compliant — REST endpoints only |
| No streaming realtime | Compliant — request/response only |
| No risk redesign | Compliant — zero changes to risk domain or evaluators |
| No excessive taxonomy | Compliant — attribution reuses existing domain types |
| No analytics expansion beyond Q6–Q7 | Compliant — exactly 2 new endpoints |
| No write-side changes | Compliant — read-side projections and aggregations only |

## Known Limitations

1. **No per-constraint triggering.** The attribution shows which constraints were active but not which one specifically caused the rejection. The write-side would need a `triggered_constraints` field for this.

2. **Funnel counts are not chained.** The funnel counts events independently per table. An event in stage N is not guaranteed to have a corresponding event in stage N-1 with the same correlation_id.

3. **No cross-symbol aggregation.** Funnel and disposition queries are scoped to one type/source/symbol/timeframe.

4. **Conversion rates are raw counts.** The consumer must divide to compute percentages.

## S299 Preparation

S299 (Post-Composite-Observability-Wave Gate) should:

1. **Validate all Q1–Q7 are answerable** through the delivered endpoints with real data.
2. **Verify zero regression** across the three vertical slices (EMA, Trend, Squeeze).
3. **Assess whether per-constraint triggering** (limitation #1) is operationally necessary or can be deferred.
4. **Confirm the wave is complete** and recommend the next direction (MACD vertical slice per S293 assessment).
5. **Document any gaps** discovered during validation for future waves.

## Metrics

- New files: 4 (2 use cases + 2 test files)
- Modified files: 9
- New endpoints: 2 (funnel, dispositions)
- Enhanced endpoints: 2 (chain, chains — now with attribution)
- New tests: 21
- Total composite tests: 36
- Lines of production code: ~350
- Lines of test code: ~400
- Zero write-side changes
- Zero new dependencies
- Zero new ClickHouse tables or columns
