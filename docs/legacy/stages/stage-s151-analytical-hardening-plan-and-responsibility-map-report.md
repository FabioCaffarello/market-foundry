# Stage S151 — Analytical Hardening Plan and Responsibility Map Report

## Stage identity

| Field | Value |
|-------|-------|
| Stage | S151 |
| Title | Analytical hardening plan and responsibility map |
| Scope | Decompose S150 gaps into responsibility-bounded hardening fronts with sequencing, scope freeze, and expansion gate |
| Predecessor | S150 (post-analytical runtime entry readiness review) |
| Date | 2026-03-19 |

---

## 1. Objective

Transform the gaps identified in S150 into a structured, actionable hardening plan oriented by clear responsibilities. Define what enters and what does not enter Wave A. Establish sequencing, exit criteria, and expansion blockers. Reduce the risk of premature expansion.

This stage produces no code. It produces a plan, a responsibility map, a scope freeze, and a readiness framework for S152–S156.

---

## 2. Deliverables

| # | Document | Path | Status |
|---|----------|------|--------|
| 1 | Hardening plan (master) | `docs/architecture/analytical-wave-a-hardening-plan.md` | Delivered |
| 2 | Responsibility map | `docs/architecture/analytical-responsibility-map-writer-reader-pipeline-observability.md` | Delivered |
| 3 | Scope, blockers, and non-goals | `docs/architecture/analytical-wave-a-scope-blockers-and-non-goals.md` | Delivered |
| 4 | Stage report (this document) | `docs/stages/stage-s151-analytical-hardening-plan-and-responsibility-map-report.md` | Delivered |

---

## 3. Executive Summary

Wave A is decomposed into 5 responsibility fronts spanning S152–S156:

| Front | Stage | Objective |
|-------|-------|-----------|
| **WC — Writer Correctness** | S152 | Test coverage for mappers, inserter, consumer, reader, + 1 integration test |
| **FH — Failure Handling** | S153 | Align INSERT retry with docs, fix buffer-clear-on-error, make mapper errors visible |
| **PR — Pipeline Recovery** | S154 | Supervisor restarts failed pipelines with backoff; degraded state management |
| **OB — Observability** | S155 | Structured counters on write and read paths; periodic summary logging |
| **EG — Expansion Gate** | S156 | Formal review verifying all fronts; expand/block verdict for Wave B |

The sequencing is: **S152 → (S153, S155 parallel) → S154 → S156**. S152 (test coverage) is the foundation; all other fronts depend on it.

11 expansion blockers are defined. All must pass before Wave B begins. There is no partial-pass path.

26 non-goals are explicitly documented across 5 categories (schema, endpoints, writer, infrastructure, architecture). Each has a rationale for exclusion.

---

## 4. Key Findings from S150 Gap Analysis

### 4.1 Critical gaps (must close in Wave A)

| Gap | Current State | Code Evidence |
|-----|--------------|---------------|
| Writer has zero test coverage | No `_test.go` files in `cmd/writer/` | Directory listing confirms |
| INSERT is single-attempt (docs say retry) | `inserter.go:flush()` calls `InsertBatch()` once; no retry loop | Code diverges from `writer-service-failure-and-delivery-semantics.md` §2.2 |
| Buffer cleared on INSERT failure | `inserter.go:flush()` clears rows regardless of success | Line 111 clears before checking error — data loss |
| No pipeline recovery | `supervisor.go:Receive()` poisons itself on any pipeline error | Lines 36-52: Started error → `Poison(PID)` |
| Reader adapter untested | `analytical_reader.go` has no `_test.go` | Only indirect coverage via use case mocks |

### 4.2 Significant gaps (should close in Wave A)

| Gap | Current State |
|-----|--------------|
| Mapper errors invisible | `parseFloat` discards error; `marshalJSON` returns `"{}"` silently |
| No write-path counters | `events_consumed` not tracked; `batch_latency` not measured |
| No read-path counters | Query latency and row count not logged |
| `diag-check.sh` ignores writer | Script checks operational services only |

### 4.3 Confirmed bug: buffer-clear-on-error

The inserter's `flush()` method clears the batch buffer at the top of the function, before the INSERT attempt. If ClickHouse rejects the INSERT, the rows are already gone. This is not a design choice — it is a bug that causes data loss on any transient ClickHouse failure. Front FH (S153) must fix this.

---

## 5. Responsibility Decomposition

### 5.1 By component

| Component | File(s) | Wave A Fronts |
|-----------|---------|---------------|
| Mappers | `cmd/writer/mappers.go` | WC (tests), FH (error visibility) |
| Inserter | `cmd/writer/inserter.go` | WC (tests), FH (retry + buffer fix), OB (latency) |
| Consumer | `cmd/writer/consumer.go` | WC (tests), OB (counter) |
| Supervisor | `cmd/writer/supervisor.go` | WC (tests), PR (recovery) |
| Pipeline catalog | `cmd/writer/pipeline.go` | WC (indirect) |
| Reader adapter | `cmd/gateway/analytical_reader.go` | WC (tests), OB (latency) |
| ClickHouse client | `internal/adapters/clickhouse/client.go` | FH (retry placement) |
| Use case + handler | `internal/application/analyticalclient/`, `internal/interfaces/http/handlers/` | None (already tested) |

### 5.2 By front

| Front | Components Touched | Estimated Scope |
|-------|-------------------|-----------------|
| WC | mappers, inserter, consumer, supervisor, reader adapter | ~15–20 test functions + 1 integration test |
| FH | inserter, mappers, ClickHouse client, architecture docs | ~3–5 code changes + doc update |
| PR | supervisor | ~1 structural change (restart loop + degraded state) |
| OB | inserter, consumer, reader adapter, diag-check.sh | ~6–8 counter additions + 1 periodic logger |
| EG | none (review only) | 1 document |

---

## 6. Sequencing Rationale

```
S152 (WC) ──┬──→ S153 (FH) ──→ S154 (PR)
             │                       │
             └──→ S155 (OB) ────────┘
                                     ▼
                               S156 (EG)
```

**Why S152 first:** You cannot safely change failure handling or add recovery logic without tests to catch regressions. Tests are the foundation.

**Why S153 before S154:** Recovery logic must be built on correct failure semantics. If INSERT retry is wrong, restart-on-failure will exercise the wrong code path.

**Why S155 can parallel S153:** Observability counters are additive. They don't change control flow. They can be added independently of failure handling changes (though counters should reflect the final FH behavior).

**Why S156 last:** The gate is meaningless unless all fronts are complete. Combining it with the last hardening stage removes the checkpoint.

---

## 7. Decision Required at S153 Start

Before implementing Front FH, one architectural decision must be made:

| Question | Option A | Option B |
|----------|----------|----------|
| INSERT failure retry policy | Implement exponential backoff (1s–30s, 5 attempts) as documented in S145 | Accept single-attempt INSERT; update docs to match |

**Recommendation:** Option A (implement retry). Rationale:
- The documented behavior was a deliberate architectural decision in S145.
- Single-attempt INSERT with data loss on transient failure is weaker than the at-least-once guarantee the writer claims.
- Retry implementation is bounded (5 attempts) and low effort.
- Accepting single-attempt would require updating multiple architecture documents.

This decision should be made at S153 start and recorded in the S153 report.

---

## 8. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Hardening creeps into redesign | Medium | High | Each front has explicit not-in-scope list; review at stage boundaries |
| Integration test infra delays S152 | Medium | Medium | Accept compose-based tests; no custom harness; local-only is fine |
| FH retry decision stalls | Low | Medium | Decision made at S153 start, not debated across stages |
| OB scope expands to metrics platform | Low | High | Structured logs only; rule RB-04 enforced |
| Recovery logic introduces new failure modes | Medium | Medium | Bounded restarts; degraded state is the safety valve; tested in S154 |
| Wave A takes too long, pressure to skip gate | Medium | High | Gate is mandatory; cannot be combined with S155; 1-stage overhead |

---

## 9. Acceptance Criteria Verification

| Criterion | Met? | Evidence |
|-----------|------|----------|
| Wave A decomposed by responsibilities | Yes | 5 fronts with clear ownership, scope, and exit criteria |
| Scope frozen without ambiguity | Yes | 16 in-scope items, 26 non-goals, each with rationale |
| Success criteria explicit | Yes | 11 exit criteria in hardening plan; 11 expansion blockers |
| Blocking criteria explicit | Yes | BLK-01 through BLK-11 with verification method |
| Base ready for disciplined execution | Yes | Sequencing defined; dependencies mapped; parallel opportunities identified |
| Expansion risk reduced | Yes | Anti-patterns documented; scope creep resistance rules defined; gate is mandatory |

---

## 10. Guard Rail Compliance

| Guard rail | Status |
|------------|--------|
| No new endpoints/families opened | Compliant — explicitly listed as non-goals |
| No cold-start/bootstrap opened | Compliant — deferred to Wave C |
| Not a redesign | Compliant — all changes are additive (tests, counters, recovery) or corrective (buffer bug, retry alignment) |
| No implementation mixed with planning | Compliant — S151 produces documents only; code starts in S152 |
| Out-of-scope clearly documented | Compliant — 26 non-goals with rationales across 5 categories |

---

## 11. Preparation for S152

S152 (Writer Correctness) should begin with:

1. **Read all writer source files** to understand testable interfaces and dependencies.
2. **Identify mock boundaries**: ClickHouse client interface, NATS consumer interface, actor engine interface.
3. **Establish test file structure**: `cmd/writer/mappers_test.go`, `cmd/writer/inserter_test.go`, `cmd/writer/consumer_test.go`, `cmd/writer/supervisor_test.go`.
4. **Start with mappers** — pure functions, no dependencies, highest test density per effort.
5. **End with integration test** — requires compose stack; validates full path; highest confidence per test.

The reader adapter test (`analytical_reader_test.go`) can be written in parallel with writer tests — it has no dependency on writer changes.

---

## 12. Stage Disposition

**S151 complete.** The Wave A hardening plan is defined, responsibility-mapped, scope-frozen, and gate-protected. S152 (Writer Correctness) is ready to begin.
