# Stage S185 — Family 05 Selection Confirmation and Responsibility Fit Report

## Stage Identity

| Field | Value |
|-------|-------|
| Stage | S185 |
| Title | Family 05 Selection Confirmation and Responsibility Fit |
| Type | Selection / Decision |
| Predecessor | S183 (Family 05 Trigger Assessment) |
| Successor | S186 (Family 05 Definition and Contract Freeze) |

---

## 1. Executive Summary

This stage formally confirms **Executions (paper_order)** as Family 05 — the sixth and final analytical family in the Wave B manual expansion pattern. The selection is based on a systematic comparison of all remaining candidates against architectural fitness, analytical value, pattern pressure, and implementation readiness.

Executions is the only candidate that advances the analytical read path to an uncovered layer (layer 6), provides maximum ceiling-test value for the manual pattern, has fully pre-staged write-path artifacts, and presents zero contamination risk to the existing 5-family baseline.

Family 05 is explicitly positioned as the **terminal test of the manual expansion model**. Its implementation will produce diagnostic signals — handler file size, friction count, Float64 handling, multi-filter complexity, parser trajectory — that determine the scope and urgency of the codegen/hardening tranche required before Family 06.

---

## 2. Candidates Evaluated

| Candidate | Layer | Type | Verdict |
|-----------|-------|------|---------|
| **Executions (paper_order)** | 6 — Execution | New layer expansion | **Confirmed as Family 05** |
| EMA Crossover (ema_crossover) | 2 — Signals | Within-layer variant | Deferred — not a family expansion |
| Tradeburst (tradeburst) | 1 — Evidence | Within-layer deepening | Deferred — missing infrastructure |
| Volume (volume) | 1 — Evidence | Within-layer deepening | Deferred — missing infrastructure |

No other candidates exist in the NATS event registries or domain definitions.

---

## 3. Family 05 Confirmed: Executions (paper_order)

### Selection rationale (summary)

1. **Only candidate that advances vertical coverage** — layer 6 is the sole uncovered layer.
2. **Maximum ceiling-test value** — 20 DDL columns, first Float64 in read path, first boolean, first fills array, first two-filter handler method. Tests every dimension the pattern hasn't encountered.
3. **Complete pre-staging** — migration 006, `mapExecutionRow()` mapper, pipeline config, NATS consumer all operational. Write path remains immutable (6th consecutive expansion).
4. **Zero contamination risk** — terminal position in dependency chain, no upstream dependency.
5. **Highest analytical value** — completes end-to-end pipeline tracing (evidence → execution).
6. **Diagnostic function** — generates quantitative signals defining the codegen/hardening boundary.

### Schema profile

- 20 DDL columns (highest of any family)
- 4 JSON columns: risk, fills, parameters, metadata
- 2 Float64 columns: quantity, filled_quantity (new type in read path)
- 1 Boolean column: final (new type in read path)
- 2 enum-like filters: side, status
- 2 execution-specific correlation IDs

### Responsibility map

| Artifact | Status |
|----------|--------|
| Migration (006) | Pre-staged |
| Writer mapper | Pre-staged |
| Pipeline entry | Pre-staged |
| Reader adapter | To build (~143 LOC) |
| Use case + contracts | To build (~158 LOC) |
| Handler method | To extend (~80–100 LOC) |
| Route registration | To extend (~5 LOC) |
| Gateway wiring | To extend (~8 LOC) |
| Tests (reader + use case + handler) | To build (~270 LOC) |
| Smoke extension | To build (~30 LOC) |

---

## 4. Deferred Candidates and Rationale

### EMA Crossover — Not a family expansion

The signal reader already handles EMA Crossover via the `type` query parameter. No new artifacts are required. Enabling it is a writer config change, not a pattern expansion. Using it as a "family" would produce zero diagnostic value and dilute the family expansion concept.

**When it becomes relevant:** Post-vertical-coverage, when horizontal deepening within layers is prioritized.

### Tradeburst — Missing infrastructure

No writer mapper, no pipeline entry, no ClickHouse migration. Would be the first family requiring write-path construction — breaking the immutability invariant that has held for 5 consecutive expansions. Schema not designed. Within-layer deepening of evidence (layer 1), not vertical extension.

**When it becomes relevant:** Post-codegen tranche, when horizontal deepening of evidence layer is prioritized and write-path extension patterns are defined.

### Volume — Same as Tradeburst

Identical gaps and reasoning. Should be evaluated alongside tradeburst as part of evidence layer enrichment.

---

## 5. Risks and Limits

### Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Handler exceeds 620 lines | Medium | Mid-implementation extraction needed | Measure early; `parseAnalyticalParams()` extraction is ~1 hour effort |
| Float64 handling creates friction | Low | New column type requires special formatting | `FormatFloat` already exists; proven for confidence fields |
| Fills array requires complex parser | Low-Medium | New parser shape, potential structural surprise | Fills structure is bounded; worst case adds 1 parser to count (7 total, under 8 threshold) |
| Two-filter method creates interaction bugs | Low | Unexpected WHERE clause behavior | Filters are independent (side, status); no logical interaction |
| Codegen scope creep post-Family-05 | Medium | Codegen tranche expands beyond readers/handlers/use cases | Scope to three highest-duplication artifacts only |

### Limits

1. Only `paper_order` execution type in scope — `venue_market_order` is deferred.
2. No cross-family queries — execution-to-risk or pipeline trace queries are out of scope.
3. No pagination beyond `limit=500` — consistent with all existing families.
4. No write-path changes — writer must remain immutable.
5. No codegen during Family 05 — codegen is a post-implementation obligation.

---

## 6. What Family 05 Must Produce (Diagnostic Signals)

| Signal | Expected value | If exceeded |
|--------|---------------|-------------|
| Handler file size | 595–615 lines | >620 → immediate extraction |
| New frictions | 0–1 | >2 → mandatory hardening |
| JSON parser count | 7 | ≥8 → generic parser evaluation |
| Creative decisions | 0 | >0 → pattern review |
| Float64 handling friction | Zero (FormatFloat reuse) | Any → document for codegen |
| Two-filter method friction | Zero (additive WHERE) | Any → document for codegen |
| Implementation time | Consistent with F-02–F-04 | Slower → document cause |

---

## 7. Preparation for S186

S186 should freeze the Family 05 contract and implementation scope, establishing:

1. **Exact 9-artifact scope** — reader, use case, contracts, handler, route, gateway, tests, smoke, HTTP test queries.
2. **Column-level coherence table** — DDL → mapper → reader alignment for all 20 columns.
3. **Endpoint specification** — `GET /analytical/execution/history` with all query parameters.
4. **Success criteria** — hard requirements (handler ≤620 lines, ≤2 frictions, 0 write-path changes) and ceiling-test metrics.
5. **Post-implementation obligations** — codegen tranche definition, handler split decision, pattern terminal assessment.
6. **Binding constraint** — Family 05 is the last manual expansion. Codegen tranche is a hard gate for Family 06.

---

## Deliverables Produced

| Document | Path | Purpose |
|----------|------|---------|
| Selection confirmation and responsibility fit | `docs/architecture/family-05-selection-confirmation-and-responsibility-fit.md` | Formal confirmation with schema, responsibility map, success criteria |
| Candidate comparison and pressure matrix | `docs/architecture/family-05-candidate-comparison-and-pressure-matrix.md` | Systematic comparison of all candidates |
| Selection rationale and deferred candidates | `docs/architecture/family-05-selection-rationale-and-deferred-candidates.md` | Why executions, why not others |
| Stage report | `docs/stages/stage-s185-family-05-selection-confirmation-and-responsibility-fit-report.md` | This document |

---

## Stage Verdict

**S185 COMPLETE.**

- Family 05 formally confirmed: **Executions (paper_order)**.
- Selection is architecturally defensible across all evaluation dimensions.
- All deferred candidates have clear rationale and future triggers.
- Family 05's role as terminal test of the manual pattern is explicit.
- Diagnostic signals are defined — implementation will produce quantitative data for the codegen/hardening boundary.
- Base is ready for S186 (contract and scope freeze).
