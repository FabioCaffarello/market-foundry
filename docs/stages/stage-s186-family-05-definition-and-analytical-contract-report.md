# Stage S186 — Family 05 Definition and Analytical Contract Report

## Stage Identity

| Field | Value |
|-------|-------|
| Stage | S186 |
| Title | Family 05 Definition and Analytical Contract |
| Type | Contract Freeze / Definition |
| Predecessor | S185 (Family 05 Selection Confirmation and Responsibility Fit) |
| Successor | S187 (Family 05 Minimal Implementation) |

---

## 1. Executive Summary

This stage freezes the complete contract for Family 05 (Executions / paper_order) — the sixth and final analytical family under the Wave B manual expansion pattern. Every artifact, boundary, column mapping, endpoint specification, parser, test, and success criterion is defined with zero ambiguity.

The contract covers:
- **20-column schema mapping** (DDL → writer → reader → domain) with per-column type and parser specification
- **One endpoint**: `GET /analytical/execution/history` with 4 required and 5 optional query parameters
- **Two new JSON parsers**: `ParseRiskInputJSON` (struct target) and `ParseFillsJSON` (slice target)
- **Handler projection**: 611–626 lines (at 620-line hard ceiling, extraction authorized if exceeded)
- **10 hard success criteria** and **9 diagnostic ceiling-test metrics**
- **14 explicit non-goals** preventing scope expansion

The implementation at S187 should require zero design decisions — the contract is complete.

---

## 2. Contract and Payload Defined

### Analytical contract

| Aspect | Specification |
|--------|--------------|
| Purpose | Read-only historical query over paper_order execution intents |
| Table | `executions` (ClickHouse) |
| DDL columns | 20 total, 16 selected by reader |
| Response key | `"executions"` (array of `ExecutionIntent`) |
| Source | `"analytical/clickhouse"` |
| Meta | `query_ms`, `row_count` |

### Payload shape

Response returns `ExecutionIntent` domain objects with all fields populated from ClickHouse row scan:

- Core fields: type, source, symbol, timeframe, side, quantity, filled_quantity, status
- JSON fields: risk (RiskInput struct), fills ([]FillRecord), parameters (map), metadata (map)
- Correlation fields: exec_correlation_id, exec_causation_id
- State fields: final (bool), timestamp

### Column coherence

All 20 DDL columns traced through the full path:

```
DDL → Writer mapper (mapExecutionRow, 20 values)
    → ClickHouse storage
    → Reader SELECT (16 columns, 4 event metadata excluded)
    → Reader scan (16 variables)
    → Domain mapping (16 fields → ExecutionIntent)
```

Coherence verified against existing `mapExecutionRow` at `cmd/writer/mappers.go:147-171`.

---

## 3. Boundaries and Responsibilities

### Write path (pre-staged — no changes)

| Component | Status | File |
|-----------|--------|------|
| Mapper | Pre-staged | `cmd/writer/mappers.go:147` |
| Pipeline | Pre-staged | `cmd/writer/pipeline.go` |
| NATS consumer | Pre-staged | Via execution_registry.go |
| **Total changes** | **0** | — |

### Read path (to build)

| Component | File | LOC |
|-----------|------|-----|
| ExecutionReader struct + query | `internal/adapters/clickhouse/execution_reader.go` | ~142 |
| Use case + contracts | `internal/application/analyticalclient/` | ~151 |
| Handler method | `internal/interfaces/http/handlers/analytical.go` | ~96–111 |
| Route registration | `internal/interfaces/http/routes/analytical.go` | ~12 |
| Gateway wiring | `cmd/gateway/analytical_reader.go` + compose | ~13 |
| **Total new code** | | **~414–429** |

### Test artifacts (to build)

| Component | File | LOC |
|-----------|------|-----|
| Reader tests | `internal/adapters/clickhouse/execution_reader_test.go` | ~95 |
| Use case tests | `internal/application/analyticalclient/get_execution_history_test.go` | ~90 |
| Handler tests | `internal/interfaces/http/handlers/analytical_test.go` | ~85 |
| Smoke extension | `scripts/smoke-analytical-e2e.sh` | ~10 |
| HTTP test queries | `tests/http/analytical.http` | ~15 |
| **Total test code** | | **~295** |

### Parsers

| Parser | Status | Target type |
|--------|--------|------------|
| FormatFloat | Reuse (2 calls) | float64 → string |
| ParseMetadataJSON | Reuse (2 calls) | string → map[string]string |
| ParseRiskInputJSON | **New** | string → execution.RiskInput |
| ParseFillsJSON | **New** | string → []execution.FillRecord |
| **Post-Family-05 total** | **8 parsers** | At threshold |

---

## 4. Success Criteria and Risks

### Hard requirements (10)

1. All 9 artifacts delivered
2. Handler ≤ 620 lines (or extraction applied)
3. New frictions ≤ 2
4. Creative decisions = 0
5. Write path changes = 0
6. Test regressions = 0
7. All 5 existing endpoints pass
8. Execution endpoint returns 200
9. Missing params return 400
10. Unavailable reader returns 503

### Ceiling-test metrics (9)

Handler size, total LOC, parser count, duplication percentages, Float64 handling, two-filter friction, implementation time, smoke test growth — all measured post-implementation and reported in validation findings.

### Key risks

| Risk | Probability | Mitigation |
|------|------------|------------|
| Handler > 620 lines | High | Extract `parseAnalyticalParams()` (~1 hour) |
| Parser count at 8 | Certain | Document, flag for codegen tranche |
| Float64 precision | Low | FormatFloat proven for confidence fields |
| Two-filter interaction | Very low | Independent WHERE clauses |
| Reader signature width (10 params) | Certain | Acceptable for terminal family; codegen fixes for future |
| Fills array parsing | Low | FillRecord is a typed struct; json.Unmarshal handles transparently |

---

## 5. Non-Goals (Explicit)

14 items explicitly out of scope:

1. venue_market_order support
2. Cross-family queries
3. Aggregation/analytics
4. Pagination beyond 500
5. Filter value validation
6. Write-path changes
7. Codegen implementation
8. Proactive handler refactoring
9. Smoke test restructuring
10. CI smoke integration
11. NATS consumer lag visibility
12. Sticky degradation auto-recovery
13. Generic parseJSON[T] parser
14. Family 06 preparation

---

## 6. Preparation for S187

S187 should proceed as **pure implementation** with zero design decisions. Everything is specified:

### Implementation order (recommended)

1. **Reader adapter** — `execution_reader.go` with BuildExecutionQuery, ParseRiskInputJSON, ParseFillsJSON, QueryExecutionHistory. Run reader tests.

2. **Contracts** — Add ExecutionHistoryQuery, ExecutionHistoryReply to `contracts.go`.

3. **Use case** — `get_execution_history.go` with ExecutionReader interface and Execute method. Run use case tests.

4. **Handler** — Add interface, struct field, deps field, response type, GetExecutionHistory method to `analytical.go`. Check line count. If > 620, extract `parseAnalyticalParams()` first, then add method. Run handler tests.

5. **Routes** — Add interface, field, HasAny update, conditional route to `routes/analytical.go`.

6. **Gateway** — Add `newAnalyticalExecutionReader` to `analytical_reader.go`. Wire in compose/run.

7. **Smoke + HTTP** — Extend smoke test, add HTTP test queries.

8. **Verify** — Run full test suite, run smoke, check coherence, measure ceiling-test metrics.

### What S187 must produce beyond code

1. **Validation findings document** — frictions, limits, what the pattern proved/didn't prove
2. **Ceiling-test metrics report** — all 9 diagnostic metrics measured
3. **Post-implementation obligations confirmation** — codegen tranche scope, handler split decision

---

## Deliverables Produced

| Document | Path | Purpose |
|----------|------|---------|
| Definition and analytical contract | `docs/architecture/family-05-definition-and-analytical-contract.md` | Payload, endpoint, parsers, instrumentation, scope |
| Schema, writer, reader, gateway contract | `docs/architecture/family-05-schema-writer-reader-gateway-contract.md` | Per-column coherence, layer responsibilities, test specification |
| Success criteria, risks, and non-goals | `docs/architecture/family-05-success-criteria-risks-and-non-goals.md` | Hard requirements, ceiling metrics, 14 non-goals, trade-offs |
| Stage report | `docs/stages/stage-s186-family-05-definition-and-analytical-contract-report.md` | This document |

---

## Stage Verdict

**S186 COMPLETE.**

- Family 05 contract fully frozen: payload, schema, endpoint, parsers, boundaries, tests, success criteria.
- Zero ambiguity in implementation scope — every artifact, file, LOC estimate, and column mapping specified.
- 14 non-goals explicitly documented to prevent scope expansion.
- Handler ceiling risk identified with authorized mitigation path.
- Parser count trajectory documented (8 — at threshold).
- Diagnostic function preserved — ceiling-test metrics defined for pattern terminal assessment.
- Base ready for S187 (pure implementation — zero design decisions required).
