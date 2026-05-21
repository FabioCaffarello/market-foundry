# Operational History & Explainability -- Evidence Gate

**Stage**: S456A
**Wave**: Operational History & Explainability (S452A--S455A)
**Date**: 2026-03-24
**Type**: Evidence Gate (Formal Wave Closure)
**Predecessor**: S455A (Session Explainability and Cross-Surface Consistency)

---

## 1. Gate Purpose

This document is the formal evidence gate for the Operational History & Explainability wave. It evaluates whether stages S452A through S455A have sufficiently strengthened the system's operational memory, read surfaces, queryability, and explainability to close the wave with confidence.

The gate does not open new work. It audits evidence already produced.

---

## 2. Wave Charter Recap

The wave opened after S449/S450 exposed that the system can execute but cannot explain what it executed. Five structural findings drove the wave:

| Finding | Problem | Severity |
|---------|---------|----------|
| F3 | 50% persistence gap (12 CH records vs 24 fills) | MEDIUM |
| F4 | `type=paper_order` in live records -- type confusion | MEDIUM |
| F5 | `status=submitted` stuck at derive-side | MEDIUM |
| F7 | Only 2 of 9 PO checks executed | MEDIUM |
| F10 | 11 min manual debugging, undocumented | MEDIUM |

The wave was scoped to 7 capabilities (C1--C7), 6 governing questions (Q1--Q6), and 4 implementation stages plus one evidence gate.

---

## 3. Stage Completion Status

| Stage | Name | Status | Deliverables |
|-------|------|--------|-------------|
| S452A | Charter and Scope Freeze | COMPLETE | Charter, capabilities doc, stage report |
| S453A | Historical Execution Read Model | COMPLETE | Lifecycle history endpoint, 19 tests, 3 arch docs |
| S454A | Operational List Queries | COMPLETE | Execution list + summary + lifecycle list endpoints, 23 tests, 2 arch docs |
| S455A | Session Explainability + Consistency | COMPLETE | Explain endpoint, consistency audit, 6 tests, 2 arch docs |
| S456A | Evidence Gate | THIS STAGE | Gate document, evidence matrix, stage report |

All four implementation stages are complete. No stage was skipped or deferred.

---

## 4. Capability Assessment

### C1 -- Persistence Completeness Invariant

**Charter requirement**: Every KV key must have a corresponding ClickHouse record.

**Assessment: SUBSTANTIAL**

- S453A delivered the `GET /analytical/execution/lifecycle` endpoint that queries across all event types without type filtering, enabling complete lifecycle reconstruction from ClickHouse.
- The existing `executions` table already contained all data -- no schema migration was needed.
- The lifecycle history query removes type from the mandatory WHERE clause, enabling persistence completeness verification.
- **Gap**: No automated background check that enumerates KV keys and verifies CH coverage. The explain endpoint (S455A) performs point-in-time cross-surface checks per partition key, but no batch audit exists.
- **Gap**: The F3 root cause (50% persistence gap) was not investigated with forensic depth in this wave's implementation stages. The wave pivoted to building read surfaces that make such gaps visible rather than surgically fixing the S449-era gap.

### C2 -- Type and Status Disambiguation

**Charter requirement**: Live vs paper vs dry-run unambiguous in all queries.

**Assessment: FULL**

- S453A/S454A endpoints surface the `type` column (paper_order, venue_market_order, venue_rejection) as a first-class response field.
- S455A field-level consistency audit confirmed type values are consistent across KV and CH.
- The `execution list` endpoint (S454A) supports `type` as an optional filter, enabling "show all venue_market_order events" queries.
- No type translation or mapping differences exist between surfaces (Finding A1 in S455A audit).

### C3 -- Session Metadata Persistence

**Charter requirement**: Session entity with ID, timestamps, config, operator, outcome as queryable state.

**Assessment: PARTIAL**

- The wave did not implement a dedicated session metadata model or KV bucket.
- S455A's explain endpoint provides per-partition-key session-scoped views (combining KV + CH for a given source/symbol/timeframe), but there is no first-class "session" entity.
- The charter's original design for C3 (session_id, started_at, stopped_at, config_hash, operator, outcome_summary) was not implemented.
- **Rationale**: The wave correctly prioritized persistence completeness, query ergonomics, and cross-surface consistency. Session metadata persistence is deferred.

### C4 -- Order Narrative Query (Full Lifecycle Trace)

**Charter requirement**: Full lifecycle trace from signal through fill via correlation chain.

**Assessment: SUBSTANTIAL**

- S453A delivers reverse-chronological lifecycle timeline per partition key (all execution event types).
- S455A's explain endpoint combines KV latest state + CH history + consistency checks + human-readable explanation into a single response.
- Correlation ID is preserved and surfaced in both KV and CH read paths.
- **Gap**: The charter envisioned a cross-table join (signals -> decisions -> strategies -> risk_assessments -> executions). The implemented surface covers the execution layer only. Upstream pipeline stages (signal, decision, strategy, risk) are not included in the lifecycle trace.

### C5 -- List Query Ergonomics

**Charter requirement**: Filtering by time, status, segment, mode; summary aggregations.

**Assessment: FULL**

- S454A delivered three new endpoints:
  - `GET /analytical/execution/list` -- relaxed-filter query (any combination of type, status, source, symbol, timeframe, side, time range).
  - `GET /analytical/execution/summary` -- GROUP BY (type, status) with counts and latest timestamp.
  - `GET /execution/lifecycle/list` -- KV-backed lifecycle enumeration exposed via HTTP.
- All ClickHouse query dimensions match the original scope.
- 23 tests cover query builder logic, use case validation, and error handling.

### C6 -- KV-to-ClickHouse Consistency Audit

**Charter requirement**: Automated divergence detection and reporting.

**Assessment: SUBSTANTIAL**

- S455A implements per-partition-key consistency checks in the explain endpoint: compares KV intent/fill/rejection status with CH-derived statuses and flags divergent, consistent, or unavailable.
- Field-level consistency matrix documented across 15 fields with findings categorized as consistent, representation differences, or structural differences.
- Parity gaps corrected (Risk + Parameters fields added to LifecycleHistoryEntry).
- **Gap**: No batch consistency audit that enumerates all KV keys and checks CH coverage. The explain endpoint operates on one partition key at a time.
- **Gap**: No background monitoring or alerting for persistent divergences.

### C7 -- Post-Session Verification Automation

**Charter requirement**: All 9 PO checks executable as automated validations.

**Assessment: PARTIAL**

- The wave did not codify the 9 PO checks from S447 as a dedicated automation harness or script.
- The new query surfaces (lifecycle history, execution list, explain) provide the data foundation needed to implement PO checks, but the checks themselves are not automated.
- **Gap**: PO check automation was originally assigned to the evidence gate stage (S452E in the charter). The wave's renumbered stages (S453A--S455A) did not include this deliverable.

---

## 5. Governing Question Assessment

| ID | Question | Answer | Evidence |
|----|----------|--------|---------|
| Q1 | Why did 50% of execution events fail to reach ClickHouse? | PARTIALLY ANSWERED | The wave did not perform forensic root cause investigation of the S449 gap. However, the new lifecycle history surface enables exact comparison of KV vs CH records per partition key. The explain endpoint detects divergences in real time. |
| Q2 | Can read surfaces distinguish live from paper execution? | ANSWERED | Type field (paper_order, venue_market_order, venue_rejection) is exposed in all query surfaces. Consistency audit confirms no translation differences. |
| Q3 | Can an operator reconstruct full lifecycle of any intent? | SUBSTANTIALLY ANSWERED | Within the execution domain: yes -- lifecycle history + explain endpoint reconstruct the full execution trajectory. Across the full pipeline (signal -> fill): no -- upstream stages not included. |
| Q4 | Can the system detect KV/CH divergence? | SUBSTANTIALLY ANSWERED | Per-partition-key: yes, via explain endpoint. Batch/all-keys: no automated check exists. |
| Q5 | Can post-session verification run without manual intervention? | NOT YET | Data surfaces exist but PO checks are not codified as automated validations. |
| Q6 | Does session-level metadata exist as queryable state? | NOT YET | No first-class session entity. Per-partition explain endpoint provides session-scoped views but session metadata (start, stop, config, operator) is not persisted. |

---

## 6. Regression Verification

| Check | Result |
|-------|--------|
| `go build ./...` (all 17 modules) | PASS -- zero errors |
| `go vet ./...` | PASS -- zero warnings |
| Adapter tests (clickhouse, writerpipeline) | PASS -- 334 test cases |
| Use case tests (analyticalclient) | PASS |
| Interface tests (http routes) | PASS |
| All existing S382--S448 tests | No regressions detected |

**Total new tests in wave**: 48 (19 S453A + 23 S454A + 6 S455A)

---

## 7. Architecture Documents Produced

| Document | Stage | Content |
|----------|-------|---------|
| Wave Charter and Scope Freeze | S452A | Problem statement, capabilities, questions, stage sequence |
| Capabilities, Questions, Non-Goals | S452A | Detailed C1--C7, Q1--Q6, NG1--NG10 |
| Historical Execution and Lifecycle Read Model | S453A | Design, SQL projection, endpoint, consistency model |
| Execution Lifecycle History Sources and Limitations | S453A | Projection semantics, limitations |
| Operational List Queries and Retrieval Ergonomics | S454A | Design, endpoints, filter semantics |
| Listing Filters Query Semantics and Limitations | S454A | Operator usage, valid/invalid combinations |
| Session Explainability Surface and Consistency Audit | S455A | Explain endpoint, cross-surface audit |
| KV/CH/Gateway Consistency Findings and Limitations | S455A | Field-level matrix, findings, gaps |

**Total**: 8 architecture documents + 4 stage reports.

---

## 8. Guard Rails Compliance

| Guard Rail | Status | Evidence |
|------------|--------|---------|
| No lakehouse/analytics inflation | COMPLIANT | Single table reused; no new infrastructure |
| No broad dashboards or UI | COMPLIANT | Query APIs only; no presentation layer |
| No OMS expansion | COMPLIANT | Existing event model unchanged |
| No exchange connectivity | COMPLIANT | Zero API keys, zero live sessions |
| No structural storage redesign | COMPLIANT | No schema migration; additive columns only |
| No real-time streaming | COMPLIANT | All request-reply (pull model) |
| No scope beyond wave | COMPLIANT | All work within operational history/explainability boundary |

---

## 9. Formal Verdict

### Wave Classification: SUBSTANTIALLY COMPLETE

The Operational History & Explainability wave has delivered its core value proposition: the system can now explain what it executed through unified read surfaces, relaxed-filter queries, cross-surface consistency checks, and a structured explain endpoint.

**Capabilities achieved**:
- 2 FULL (C2, C5)
- 3 SUBSTANTIAL (C1, C4, C6)
- 2 PARTIAL (C3, C7)

**Questions answered**:
- 1 FULLY ANSWERED (Q2)
- 2 SUBSTANTIALLY ANSWERED (Q3, Q4)
- 1 PARTIALLY ANSWERED (Q1)
- 2 NOT YET (Q5, Q6)

### Gate Decision: CLOSE WITH NOTED GAPS

The wave is authorized to close. The delivered surfaces materially improve operational confidence. The gaps are bounded and documented:

1. **Session metadata persistence (C3, Q6)**: Deferred. Not blocking for operational use.
2. **PO check automation (C7, Q5)**: Deferred. Data foundation exists; automation is a scripting exercise.
3. **Batch consistency audit**: Deferred. Per-key checking is operational; batch checking is a quality-of-life improvement.
4. **F3 forensic root cause**: Not surgically resolved, but the new surfaces make similar gaps immediately detectable.

### Conditions for Full Closure

None of the gaps require a closure micro-stage. They can be addressed in a future wave or as incremental improvements when operationally motivated.

---

## 10. Next Macro-Front Recommendation

The wave closes having strengthened the read side of the system. The recommended next direction should be chosen based on the most pressing operational need:

| Direction | Motivation | Dependency |
|-----------|-----------|-----------|
| **Second supervised live session** | Validate execution pipeline under real market conditions with improved observability | Requires S452-S455 live stabilization track |
| **Automated operational verification** | Codify PO checks + batch consistency audit + session metadata | Low risk, high confidence improvement |
| **Performance and resilience hardening** | Stress test query surfaces under load; validate writer pipeline under sustained throughput | Useful before scaling to multi-symbol live |

The evidence from this gate suggests the system is ready for the next supervised live session with significantly better post-session audit capability than S449 had.
