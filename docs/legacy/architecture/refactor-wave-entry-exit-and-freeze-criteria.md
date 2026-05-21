# Refactor Wave: Entry, Exit, and Freeze Criteria

**Stage:** S211
**Date:** 2026-03-20
**Governing document:** `refactor-wave-charter-and-entry-freeze.md`
**Status:** ACTIVE

---

## 1. Purpose

This document defines the formal criteria for entering, exiting, and maintaining the expansion freeze during the Strategic Refactoring and Documentation Consolidation Phase. These criteria are binary — they either pass or they don't. There is no "partial pass" for gate criteria.

---

## 2. Entry Criteria

### 2.1 Hard Prerequisites (ALL must be true)

| ID | Criterion | Verification Method | Current Status |
|----|-----------|---------------------|----------------|
| EC-1 | S210 gate issued CONDITIONAL PASS or better | Read S210 report | **PASS** — CONDITIONAL PASS issued 2026-03-20 |
| EC-2 | All S205 MF items verified (except MF-2) | S210 evidence table | **PASS** — 6 of 7 verified; MF-2 locally verified |
| EC-3 | S209 debt registry exists and is classified | Read S209 deliverable | **PASS** — 31 items, P0–P3 classified |
| EC-4 | S209 documentation entropy map exists with execution plan | Read S209 deliverable | **PASS** — 11 clusters, 12-phase plan |
| EC-5 | S211 charter is published and active | This document + charter | **PASS** — Published 2026-03-20 |
| EC-6 | Permitted vs Prohibited classification is published | Companion document | **PASS** — Published 2026-03-20 |

### 2.2 Conditional Prerequisite (must be resolved before RW-2)

| ID | Criterion | Verification Method | Current Status |
|----|-----------|---------------------|----------------|
| EC-7 | CI pipeline passes on real push (closes MF-2) | CI run result | **PENDING** — First action of RW-1 |
| EC-8 | Repository tagged at `stabilization-exit-s210` | `git tag` output | **PENDING** — Immediately after EC-7 |

**Rule:** RW-1 (Entry Gate Closure) is authorized to begin immediately. RW-2 (Documentation Cleanup) is blocked until EC-7 and EC-8 are confirmed.

---

## 3. Per-Wave Entry Criteria

### 3.1 RW-1 → RW-2 Transition

| Criterion | Required |
|-----------|----------|
| CI pipeline confirmed green (EC-7) | Yes — hard gate |
| Repository tagged `stabilization-exit-s210` (EC-8) | Yes — hard gate |
| No unresolved P0 items in debt registry | Yes |
| Charter and governance docs committed | Yes |

### 3.2 RW-2 → RW-3 Transition

| Criterion | Required |
|-----------|----------|
| Documentation consolidation at stable checkpoint | Yes — all in-progress clusters committed, no half-merged docs |
| All 19 Go modules still build cleanly | Yes — regression check |
| All unit tests still pass | Yes — regression check |
| Codegen gates still passing | Yes — drift check |
| No new P0 items introduced | Yes |

**Note:** RW-2 does not need to be fully complete before RW-3 begins, but all in-progress work must be committed and stable. No partially merged document clusters.

### 3.3 RW-3 → RW-4 Transition

| Criterion | Required |
|-----------|----------|
| All P1 code debt items addressed or formally deferred with justification | Yes |
| Module graph evaluation documented (AD-01) | Yes — evaluation required; execution optional |
| All 19 Go modules build cleanly | Yes |
| All unit tests pass | Yes |
| No regressions in codegen gates | Yes |
| Debt registry updated with completion status for all addressed items | Yes |

---

## 4. Exit Criteria (Phase Completion)

### 4.1 Hard Exit Criteria (ALL must be true to exit)

| ID | Criterion | Target | Measurement |
|----|-----------|--------|-------------|
| XC-1 | Active architecture docs reduced | ≤ 150 files in `docs/architecture/` | File count (excluding archived) |
| XC-2 | P0 debt items | 0 remaining | Debt registry audit |
| XC-3 | P1 debt items | 0 remaining (resolved or formally deferred with justification) | Debt registry audit |
| XC-4 | All 19 Go modules build | Zero errors | `go build ./...` per module |
| XC-5 | All unit tests pass | Zero failures | `make test` |
| XC-6 | CI gates green | All jobs pass | CI pipeline run |
| XC-7 | Codegen gates passing | 4/4 gates | `make codegen-check`, `codegen-integrated`, `codegen-validate-all`, `codegen-test` |
| XC-8 | Archive populated | `docs/archive/` contains superseded content | Directory inspection |
| XC-9 | Stage report index created | Index document exists | File exists |
| XC-10 | No new P0 items introduced during phase | 0 | Debt registry audit |
| XC-11 | Repository tagged | `refactoring-phase-exit` tag applied | `git tag` output |
| XC-12 | No frozen items violated | 0 violations | Charter compliance audit |
| XC-13 | Debt registry fully updated | All items have current status | Registry review |

### 4.2 Non-Exit Criteria (NOT required for exit)

These are desirable outcomes but are explicitly not gate criteria:

| Item | Why Not Required |
|------|-----------------|
| All P2 items resolved | P2 items are moderate cleanup; deferral is acceptable |
| All P3 items resolved | P3 items are cosmetic; may never be addressed |
| Load testing baseline established | Post-refactoring concern (TD-10) |
| Module consolidation executed | Evaluation is required (AD-01); execution is conditional on findings |
| clickhouse-go version upgraded | Deferred past refactoring phase |
| Remaining 5 families codegen-integrated | Post-refactoring expansion decision |

---

## 5. Freeze Criteria

### 5.1 Freeze Activation

The expansion freeze is **active from the moment S211 is complete** and remains active until the exit gate passes.

### 5.2 Freeze Verification Points

The freeze is verified at every wave transition:

| Checkpoint | Verification |
|------------|-------------|
| RW-1 → RW-2 | Confirm no frozen items were touched during entry gate work |
| RW-2 → RW-3 | Confirm documentation work did not introduce new functionality, endpoints, or schema |
| RW-3 → RW-4 | Confirm code refactoring did not add new capabilities, only restructured existing ones |
| RW-4 (exit) | Full freeze compliance audit across all changes since `stabilization-exit-s210` tag |

### 5.3 Freeze Violation Protocol

If a frozen item is discovered to have been modified:

1. **Stop** the current work immediately.
2. **Document** the violation: what was changed, why, and by which stage.
3. **Assess** whether the change can be cleanly reverted.
4. **If revertible:** revert and proceed.
5. **If not revertible:** escalate to charter amendment process. The change must be formally accepted or the phase must be rolled back to the last clean checkpoint.

### 5.4 Freeze Exceptions

There is exactly one class of legitimate freeze exception:

> A **critical security vulnerability** (CVE severity HIGH or CRITICAL) in a direct dependency that is exploitable in the project's deployment context.

This exception:
- Requires documentation of the CVE.
- Requires assessment of exploitability.
- Permits only a targeted version bump of the affected dependency.
- Must be recorded as a charter amendment.
- Does not open the door for other dependency upgrades.

---

## 6. Criteria Traceability

| Criterion ID | Source | Stage Defined |
|--------------|--------|---------------|
| EC-1 through EC-6 | S210 gate conditions + S211 charter | S211 |
| EC-7, EC-8 | S210 conditional pass outstanding item | S210/S211 |
| XC-1 | S209 entropy map target | S209 |
| XC-2, XC-3 | S209 debt registry priorities | S209 |
| XC-4 through XC-7 | S205 MF items (build/test/codegen baseline) | S205 |
| XC-8 | S209 archive requirement | S209 |
| XC-9 | AD-06 in debt registry | S209 |
| XC-10 | Phase governance (no regression) | S211 |
| XC-11 | S210 recommendation | S210 |
| XC-12 | S211 charter freeze rules | S211 |
| XC-13 | S209 registry maintenance rules | S209 |
| EF-1 through EF-12 | S205 scope freeze | S205 |
| RF-1 through RF-8 | S210 + S211 refactoring freeze | S210/S211 |

---

## 7. Decision Record: Why These Criteria

### Entry is deliberately lightweight
The stabilization wave (S205–S210) already performed extensive verification. The entry criteria for the Refactor Wave confirm that verification exists — they do not re-verify. The one exception (EC-7, CI on real push) is the single gap the stabilization gate identified.

### Exit is deliberately strict
The refactoring phase modifies structural foundations (documentation, module boundaries, constructor signatures). A strict exit gate ensures that structural changes haven't introduced regressions that would cascade into future work.

### Freeze is deliberately absolute
Prior waves in this project suffered from scope creep — "one small addition" compounding into expanded scope. The absolute freeze prevents this pattern. The only exception (CVE) is bounded and requires formal process.

### P2/P3 items are deliberately excluded from exit criteria
Requiring all debt resolution would create perverse incentives to either rush low-value changes or game the registry. P1+ is the quality bar. P2/P3 items are tracked but optional.
