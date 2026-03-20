# Stage S212: Repository Architecture Census and Refactor Map — Report

> Date: 2026-03-20
> Status: Complete
> Phase: Strategic Refactoring Preparation
> Predecessor: S211 (Refactor Wave Charter and Entry Freeze)
> Successor: S213 (Structural Refactoring Execution)

---

## 1. Executive Summary

S212 produced a complete architectural census of the market-foundry repository and a prioritized refactoring map. The census covers all 8 runtimes, 19 modules, 6 architectural layers, and 80+ adapter files. The analysis identified 10 structural duplication clusters totaling ~10,100 lines of recoverable duplication (~11% of the 57,289-line codebase), 5 coupling issues, 5 naming debt items, and 6 structural smells.

The refactoring map assigns 6 items to HIGH priority (execute in S213), 7 items to MEDIUM priority (execute if time permits), and 6 items to LOW priority (defer). Expected outcome after S213 HIGH items: 67% reduction in documentation entropy, 80% reduction in NATS consumer spec duplication, and 60–75% reduction in ClickHouse reader and store actor duplication.

---

## 2. Deliverables Produced

| Deliverable | Location | Content |
|-------------|----------|---------|
| Architecture Census | `docs/architecture/repository-architecture-census-and-refactor-map.md` | Complete runtime, module, package, boundary, codegen, and documentation census |
| Boundaries and Smells | `docs/architecture/repository-boundaries-coupling-duplication-and-smells.md` | 10 structural duplications, 5 coupling issues, 5 naming debts, 6 smells, 5 non-issues |
| Priority Map | `docs/architecture/refactor-priority-map-high-medium-low.md` | 6 HIGH, 7 MEDIUM, 6 LOW items with scoring, execution order, and expected outcomes |
| Stage Report | `docs/stages/stage-s212-repository-architecture-census-and-refactor-map-report.md` | This document |

---

## 3. Key Findings

### 3.1 Largest Duplication Clusters

| Rank | Area | Lines | Dup % | Priority |
|------|------|-------|-------|----------|
| 1 | NATS consumers + publishers | 2,400 | 55% | M-05 |
| 2 | Store consumer + projection actors | 2,400 | 60% | H-04 |
| 3 | NATS KV stores | 1,400 | 60% | M-04 |
| 4 | NATS registry consumer specs | 1,008 | 75% | H-02 |
| 5 | ClickHouse readers | 884 | 70% | H-03 |
| 6 | HTTP handlers | 740 | 65% | M-02 |
| 7 | Analytical use cases | 584 | 85% | M-01 |
| 8 | Writer pipeline + mappers | 451 | 88% | M-03 |
| 9 | Gateway compose | 246 | 70% | M-06 |

### 3.2 Primary Architectural Pressure Point

**Family Addition Blast Radius (CP-01):** Adding a new pipeline family currently requires coordinated changes across 15+ files spanning 8 packages. This is the single highest-cost operation in the system and the primary motivator for most HIGH-priority refactorings.

### 3.3 Clear vs Blurred Boundaries

**Clear:** Domain ↔ Infrastructure, Application ↔ Adapters (via ports), Service ↔ Service (via NATS), Operational ↔ Analytical, Generated ↔ Manual.

**Blurred:** NATS adapter flat package (10,110 lines), Gateway compose mixing concerns, Settings schema monolith.

### 3.4 Non-Issues Confirmed

- Domain input types (e.g., Decision.SignalInput) are intentional decoupling, not duplication.
- Per-family actor pairs in store are correct fault isolation, not redundancy.
- Separate operational/analytical query paths are correct CQRS boundary.
- Settings OrDefault() methods are idiomatic Go, not bloat.

---

## 4. Decisions Made

### D-01: Structural vs Cosmetic Classification

Every identified item was explicitly classified as structural (affects evolution cost) or cosmetic (noise, not worth addressing). This prevents the refactoring phase from expanding into a cleanup-everything exercise.

### D-02: Priority Scoring Method

Three-factor weighted scoring (evolution cost 40%, blast radius 30%, risk 30% inverted) provides a defensible, reproducible ranking. Every item has a numeric score.

### D-03: Execution Wave Structure

S213 execution is organized into 5 sequential waves:
1. Foundations (modules, docs)
2. NATS layer
3. Analytical path
4. Store + handlers
5. Writer + gateway

This order ensures each wave's output enables the next.

### D-04: Guard Rails

7 explicit guard rails prevent the refactoring from introducing regressions or scope creep. Most critical: preserve NATS durable names, preserve ClickHouse queries, preserve HTTP contracts.

---

## 5. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| NATS consumer name change breaks offsets | Low | High | Guard rail: factory must generate identical durable names |
| Module consolidation breaks CI | Medium | Medium | Evaluate first, execute incrementally, CI green gate |
| ClickHouse query regression | Low | High | Guard rail: identical SQL output verification |
| Refactoring scope creep | Medium | Medium | Priority map + freeze rules from S211 |
| Documentation consolidation loses canonical decisions | Low | Medium | Archive, don't delete. Explicit consolidation targets. |

---

## 6. Items Explicitly Deferred

| Item | Reason | When |
|------|--------|------|
| EMA naming fix (L-01) | Requires NATS consumer migration + codegen unfreeze | After codegen freeze lifts |
| Evidence/candle terminology (L-02) | Cosmetic, no functional impact | Broader naming pass |
| Float formatting (L-03) | Cosmetic, both implementations correct | Opportunistic |
| Store supervisor wiring (L-04) | Becomes trivial after H-04 | After store actor generics |
| Codegen family naming (L-05) | Codegen frozen | After codegen freeze lifts |
| Test hardcoded counts (L-06) | Family expansion frozen | When expansion resumes |

---

## 7. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|---------|
| Clear repository census exists | **PASS** | Census document covers all runtimes, modules, packages, boundaries |
| Principal smells and couplings are explicit | **PASS** | 10 structural duplications, 5 coupling issues, 6 smells cataloged |
| Defensible refactoring prioritization | **PASS** | Weighted scoring with 3 factors; every item has numeric priority |
| Phase no longer depends on subjective impressions | **PASS** | All items quantified by line count, duplication %, and blast radius |
| Base is ready for S213 structural execution | **PASS** | Execution order defined, guard rails set, dependencies mapped |

---

## 8. Preparation for S213

### S213 Entry Conditions (all must be TRUE)

1. S212 documents committed and reviewed
2. S211 governance documents in place (refactor-wave-charter, entry-exit criteria, permitted changes)
3. CI pipeline passes (MF-2 — pending from S210)
4. Repository tagged `stabilization-exit-s210` (pending from S210)

### S213 Scope

Execute HIGH priority items (H-01 through H-06) from the refactor priority map. MEDIUM items are stretch goals. LOW items are explicitly out of scope.

### S213 Success Criteria

- All 6 HIGH items complete with CI green
- No regressions in existing tests
- No NATS durable name changes
- No ClickHouse query changes
- No HTTP contract changes
- Active architecture docs ≤ 150
- Refactoring items tracked with before/after line counts

---

## 9. Relationship to Prior Work

| Document | Relationship |
|----------|-------------|
| S209 technical-debt-registry | **Subsumed** — debt items TD-01 through TD-16 and AD-01 through AD-06 are now classified in the priority map |
| S209 documentation-entropy-map | **Referenced** — H-05 executes the 12-phase plan from this document |
| S210 stabilization-gate | **Predecessor** — S212 builds on the conditional pass from S210 |
| S211 refactor-wave-charter | **Governance** — S212 operates under the freeze rules and permitted changes from S211 |
