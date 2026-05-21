# Stage S209: Pre-Refactor Technical Debt Registry and Cleanup Plan — Report

**Stage:** S209
**Date:** 2026-03-20
**Status:** COMPLETE
**Predecessor:** S208 (Runtime, Config, and Operational Closure)
**Successor:** S210 (Final Stabilization Gate)

---

## 1. Executive Summary

S209 constructs the formal registry and plan for the next phase of market-foundry: strategic refactoring, documentation consolidation, and architectural cleanup. No cleanup was executed — only mapped, classified, and planned.

The key findings:
- **440 architecture docs** with significant entropy: redundancy clusters, implicit supersessions, scattered deferred work, and per-family boilerplate.
- **205 stage reports** that serve as audit trail but lack a navigable index.
- **17 technical debt items** classified across code, architecture, and CI dimensions.
- **9 architectural debt items** ranging from module graph complexity to documentation scatter.
- **5 CI/build debt items** that are hard prerequisites for the refactoring phase.
- **7 S205 must-finish items**, of which only 1 (MF-4, writer binary) is confirmed done.

The refactoring phase is structured as 4 waves with explicit entry/exit gates.

---

## 2. Deliverables Produced

| # | Deliverable | Path | Content |
|---|------------|------|---------|
| 1 | Technical Debt Registry | `docs/architecture/pre-refactor-technical-debt-registry-and-cleanup-plan.md` | 17 code debt items (TD-01 to TD-17), 9 architectural debt items (AD-01 to AD-09), 5 CI debt items (CI-01 to CI-05), priority classification, cleanup plan structure |
| 2 | Documentation Entropy Map | `docs/architecture/documentation-entropy-archive-delete-consolidate-map.md` | 11 cluster analyses, archive/delete/consolidate recommendations per cluster, proposed archive structure, 12-phase execution order, safety guardrails |
| 3 | Next Phase Scope | `docs/architecture/next-phase-refactor-and-documentation-wave-scope.md` | 4-wave execution plan, entry prerequisites, constraints, risks, success metrics, relationship to S210 |
| 4 | Stage Report | `docs/stages/stage-s209-pre-refactor-technical-debt-registry-and-cleanup-plan-report.md` | This document |

---

## 3. Findings Summary

### 3.1 Documentation Entropy

| Cluster | Files | Recommendation | Reduction |
|---------|-------|----------------|-----------|
| Next-wave recommendations | 15 | Consolidate to 1 timeline | -14 |
| Per-family lifecycle (03-06) | 34 | Consolidate to 4 + 1 template | -29 |
| Wave B pattern/iteration | 24 | Consolidate to 6 + archive | -18 |
| Analytical infrastructure | 28 | Consolidate to 10 + archive | -18 |
| Codegen | 16 | Consolidate to 8 | -8 |
| Gains/tradeoffs/debts | 17 | Consolidate to 1 timeline | -16 |
| Deferred/triggered refactors | 13 | Consolidate to 1 registry | -12 |
| Gate/readiness reviews | 18 | Archive all | -18 |
| Superseded documents | 7 | Archive | -7 |
| **Total addressable** | **172** | | **-140** |

**Projected outcome:** ~440 → ~120-150 active architecture docs + organized archive.

### 3.2 Technical Debt Profile

| Priority | Count | Items |
|----------|-------|-------|
| P0 (Structural Blocker) | 6 | TD-01 (handler extraction), CI-01 through CI-05 (verification gates) |
| P1 (High-Value) | 5 | TD-02 (reader signature), AD-01 (module graph), AD-03 (supersession markers), AD-04 (family doc boilerplate), AD-06 (stage index) |
| P2 (Moderate) | 8 | TD-03, TD-08, TD-10, TD-11, TD-14, TD-15, TD-16, AD-07 |
| P3 (Cosmetic) | 7 | TD-04, TD-06, TD-07, TD-09, TD-12, TD-17, AD-09 |

### 3.3 S205 Must-Finish Status

| Status | Count | Items |
|--------|-------|-------|
| DONE | 1 | MF-4 (writer binary removal) |
| NOT VERIFIED | 5 | MF-2, MF-3, MF-5, MF-6, MF-7 |
| NOT DONE | 1 | MF-1 (handler extraction) |

---

## 4. Key Decisions

| Decision | Rationale |
|----------|-----------|
| Archive as default over delete | Git history preserves everything, but having originals in `docs/archive/` makes the safety net visible and accessible without git commands |
| One cluster at a time for consolidation | Reduces merge conflict risk, makes each step reviewable, maintains clean git history |
| Domain docs reorganize (move) rather than consolidate | Domain-specific content is not redundant — it is scattered. Moving to subdirectories preserves content while improving navigability |
| Stage reports kept untouched | They are the audit trail. Only an index is added. |
| Module consolidation is analysis-only in this phase | Import cycle risk means module merges need careful analysis before execution |

---

## 5. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Consolidation loses unique content | Archive originals first; review each merge; git history as final safety net |
| S205 MF items reveal hidden failures | Treat as P0 blockers; fix before refactoring entry |
| Scope creep into feature work | Strict scope definition; debt registry for new discoveries |
| Documentation cleanup takes longer than expected | Phased execution allows stopping at any wave boundary |
| CI verification exposes pre-existing test failures | Part of the entry gate; must be resolved, not bypassed |

---

## 6. Preparation for S210

S210 should verify:
1. All four S209 deliverables exist and are internally consistent.
2. The debt registry covers all known items (no obvious omissions).
3. The entropy map's recommendations are actionable (no vague "maybe consolidate" items).
4. The next-phase scope has clear entry/exit criteria.
5. S205 MF items are either done or have a clear plan for completion before refactoring entry.

S210 is the **authorization gate**. Until it passes, the refactoring phase does not begin.

---

## 7. What Was NOT Done (by design)

- No files were deleted, archived, or moved.
- No code was refactored.
- No CI jobs were run or verified.
- No module consolidation was attempted.
- No new features were designed or implemented.

This stage is purely a **planning and registry** stage. All execution is deferred to the refactoring phase, gated by S210.
