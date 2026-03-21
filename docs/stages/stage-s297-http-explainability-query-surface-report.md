# Stage S297 — HTTP Explainability Query Surface Report

> Wave: Composite Execution Observability (S294–S299)
> Block: 3 of 5
> Status: **Complete**
> Predecessor: S296 (Composite Execution Read Model)

## Objective

Design, implement, validate, and document a canonical HTTP query surface that exposes the composite execution read model (S296) for operational explainability, enabling governing questions Q1–Q5 to be answered via HTTP without manual correlation ID reconstruction.

## Deliverables

### Code

| File | Type | Description |
|------|------|-------------|
| `internal/interfaces/http/handlers/composite.go` | New | CompositeWebHandler with GetChain and GetChains methods |
| `internal/interfaces/http/handlers/composite_test.go` | New | 8 unit tests covering success, validation, errors, and nil safety |
| `internal/interfaces/http/routes/analytical.go` | Modified | Added GetCompositeChain to AnalyticalFamilyDeps, route registration |
| `cmd/gateway/analytical_reader.go` | Modified | Added newAnalyticalCompositeReader factory |
| `cmd/gateway/compose.go` | Modified | Wired CompositeReader → use case → route deps |

### Documentation

| File | Description |
|------|-------------|
| `docs/architecture/http-explainability-query-surface-for-q1-q7.md` | Query surface design, Q1–Q7 coverage matrix, endpoint specifications |
| `docs/architecture/explainability-http-contracts-payloads-and-usage-examples.md` | Contracts, payload schemas, usage examples for all three answerable questions |
| `docs/stages/stage-s297-http-explainability-query-surface-report.md` | This report |

## Endpoints Delivered

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/analytical/composite/chain?correlation_id=...` | Single chain reconstruction by correlation_id |
| GET | `/analytical/composite/chains?source=...&symbol=...&timeframe=...` | Batch chain lookup with optional time range and limit |

## Governing Questions Status

| Question | Status | Notes |
|----------|--------|-------|
| Q1 — Why executed? | **Answerable** | Full chain visible via single endpoint |
| Q2 — Why rejected? | **Partially answerable** | Risk disposition + constraints visible; structured attribution → S298 |
| Q3 — Signal inputs? | **Answerable** | Decision.signals + signal stage with values |
| Q4 — Confidence flow? | **Answerable** | Each stage carries confidence/severity through chain |
| Q5 — Pipeline break? | **Partially answerable** | missing_stages reveals breakpoints; batch is execution-rooted |
| Q6 — Block/approve counts? | **Deferred** | Requires aggregation → S298 |
| Q7 — Conversion rate? | **Deferred** | Requires aggregation → S298 |

## Test Results

### Handler Tests (8/8 pass)

| Test | Validates |
|------|-----------|
| `TestCompositeGetChain_Success` | Full chain response, status 200, Server-Timing header |
| `TestCompositeGetChain_MissingCorrelationID` | Parameter validation returns 400 |
| `TestCompositeGetChain_UseCaseError` | Use case error returns 503 |
| `TestCompositeGetChain_NilHandler` | Nil handler returns 503 |
| `TestCompositeGetChains_Success` | Batch response with 2 chains, meta.chain_count |
| `TestCompositeGetChains_MissingTimeframe` | Missing timeframe returns 400 |
| `TestCompositeGetChains_InvalidLimit` | Non-integer limit returns 400 |
| `TestCompositeGetChains_NilHandler` | Nil handler returns 503 |

### Use Case Tests (9/9 pass, from S296)

All existing composite use case tests continue to pass without modification.

### Integration Tests (6/6 criteria, from S296)

All existing ClickHouse integration test criteria continue to pass.

## Architectural Decisions

1. **Separate handler, shared route group.** The CompositeWebHandler is a new struct (not added to AnalyticalWebHandler) to keep composite concerns isolated. It is registered within the existing `Analytical()` route group because it shares the same ClickHouse dependency chain and conditional availability pattern.

2. **Two endpoints, not one.** Single-chain and batch are separate paths (`/chain` vs `/chains`) rather than one endpoint with mode switching. This makes the API self-documenting and avoids ambiguity about required parameters.

3. **Reuse existing param parsers.** Batch endpoint reuses `parseQueryKeyParams()` and `parseAnalyticalParams()` from the shared handler utilities. No new parsing logic was needed.

4. **No new use case code.** The `GetCompositeChainUseCase` from S296 was already designed for both single and batch modes. S297 adds only the HTTP transport layer.

5. **Response shape matches use case contract.** The `compositeChainResponse` struct mirrors `CompositeChainReply` exactly, avoiding any translation layer between use case and HTTP.

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No front-end | Compliant — HTTP-only, no UI |
| No observability portal | Compliant — plain REST endpoints |
| No real-time streaming | Compliant — request/response only |
| No speculative endpoints | Compliant — exactly 2 endpoints, both backed by proven use case |
| No S294 non-goals | Compliant — read-side only, no write changes, no new infra |
| Stop condition: max 2 new endpoints per stage | Compliant — exactly 2 endpoints |

## Known Limitations

1. **Batch is execution-rooted.** Chains that never reached the execution stage are not discoverable via batch mode. Operators must use the single-chain endpoint with a known correlation_id, or use per-family analytical history endpoints.

2. **No structured rejection attribution.** The risk stage shows `disposition` and `constraints` but does not identify which specific constraint caused a rejection. This is the primary gap for Q2, addressed in S298.

3. **No aggregation.** Q6 (block/approve counts) and Q7 (stage conversion rates) require counting and grouping across chains. This capability is planned for S298.

4. **Eventual consistency.** The five independent queries may not see the most recently written events. Typical delay is sub-second.

## S298 Preparation

S298 (Attribution of Blockages, Rejections, and Reductions) should:

1. **Add structured rejection attribution** to the risk stage, identifying which constraint(s) caused each rejection. This completes Q2.
2. **Add aggregation endpoints** for counting blocked vs approved executions (Q6) and computing stage conversion rates (Q7).
3. **Consider extending batch mode** to discover chains that broke before reaching execution, addressing the current execution-rooted limitation.
4. **Evaluate whether risk `constraints` need schema enrichment** to support attribution without write-side changes.

## Metrics

- New files: 2 (handler + test)
- Modified files: 3 (routes, reader factory, compose)
- New endpoints: 2
- New tests: 8
- Lines of production code: ~120
- Lines of test code: ~170
- Zero write-side changes
- Zero new dependencies
