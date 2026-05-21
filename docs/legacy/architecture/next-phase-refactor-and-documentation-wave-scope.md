# Next Phase: Refactor and Documentation Wave — Scope Definition

**Stage:** S209
**Date:** 2026-03-20
**Status:** Scope definition — not yet executing.

---

## 1. Phase Identity

**Name:** Strategic Refactoring and Documentation Consolidation Phase
**Trigger:** Successful completion of S209 registry + S210 stabilization gate
**Exit condition:** All P0/P1 debt items resolved; documentation target reached; clean build/test/CI baseline confirmed.

---

## 2. Phase Objectives (in priority order)

1. **Close the S205 must-finish items** that remain unverified (MF-1 through MF-7).
2. **Reduce documentation entropy** from ~440 architecture docs to ~120-150 active docs with organized archive.
3. **Resolve high-value technical debt** (P0 and P1 items from the debt registry).
4. **Establish a clean, navigable, maintainable foundation** for future development phases.

---

## 3. What This Phase IS

- Structural cleanup of existing code (handler extraction, signature refactoring, module evaluation).
- Documentation consolidation, archival, and reorganization.
- CI/build verification and hardening.
- Debt registry closure.

## 4. What This Phase IS NOT

- **Not feature work.** No new families, domains, services, or capabilities.
- **Not performance optimization.** Load testing baseline is registered but not prioritized.
- **Not architectural redesign.** Module boundaries may be evaluated but not fundamentally changed.
- **Not TC-02 scope.** State persistence, WAL, cold-start bootstrap remain deferred.
- **Not venue expansion.** Real exchange connectivity remains out of scope.

---

## 5. Entry Prerequisites (Hard Gates)

All must be TRUE before the phase begins:

| # | Prerequisite | Verification Method | Current Status |
|---|-------------|---------------------|----------------|
| 1 | S209 debt registry and entropy map are complete | This document exists | DONE |
| 2 | S210 stabilization gate passes | S210 stage report | PENDING |
| 3 | Repository tagged at stabilization exit | `git tag stabilization-exit-s210` | PENDING |
| 4 | All 13 Go modules build cleanly | `go build ./...` per module | NOT VERIFIED |
| 5 | All unit tests pass | `go test ./...` per module | NOT VERIFIED |
| 6 | CI smoke-analytical job verified | PR-triggered verification | NOT VERIFIED |
| 7 | Codegen integrated check passes all 7 families | `scripts/codegen-integrated-check.sh` | NOT VERIFIED |
| 8 | Codegen cross-spec validation passes | `codegen validate-all` | NOT VERIFIED |

---

## 6. Phase Structure

### Wave 1: Entry Gate Closure (prerequisite completion)

**Goal:** Close all S205 MF items and verify the baseline.

| Task | Reference | Effort |
|------|-----------|--------|
| MF-1: Extract `parseAnalyticalParams()` | TD-01 | Small |
| MF-2: Verify CI smoke-analytical | CI-01 | Small |
| MF-3: Verify codegen integrated check | CI-02 | Small |
| MF-5: Verify all modules build | CI-03 | Small |
| MF-6: Verify all tests pass | CI-04 | Small |
| MF-7: Verify codegen cross-spec validation | CI-05 | Trivial |
| Tag repository | — | Trivial |

**Exit:** All MF items verified. Repository tagged.

### Wave 2: Documentation Cleanup (highest entropy reduction)

**Goal:** Execute the documentation entropy map, following the 12-phase execution order.

| Phase | Action | Files Affected |
|-------|--------|----------------|
| 1 | Create `docs/archive/` structure | 0 |
| 2 | Archive superseded docs | ~7 |
| 3 | Archive gate/readiness docs | ~18 |
| 4 | Consolidate deferred-work docs | ~13 → 1 |
| 5 | Consolidate next-wave docs | 15 → 1 |
| 6 | Consolidate gains/tradeoffs docs | 17 → 1 |
| 7 | Consolidate family lifecycle docs | 34 → 5 |
| 8 | Consolidate Wave B docs | 24 → 6 |
| 9 | Consolidate analytical docs | 28 → 10 |
| 10 | Consolidate codegen docs | 16 → 8 |
| 11 | Reorganize domain docs | ~40 moved |
| 12 | Create stage report index | 1 new |

**Exit:** Architecture docs reduced to target range. Archive populated. Stage index created.

### Wave 3: Code Debt Cleanup (P1 items)

**Goal:** Address high-value structural debt.

| Task | Reference | Effort |
|------|-----------|--------|
| Reader signature refactoring (options pattern) | TD-02, AD-01 | Medium |
| Test hardcoded family counts | TD-03 | Small |
| Module graph evaluation | AD-01 | Medium (analysis only) |
| NATS consumer lag exposure | TD-08 | Small |

**Exit:** All P1 items resolved or reclassified with justification.

### Wave 4: Verification and Exit Gate

**Goal:** Confirm the foundation is clean and stable.

| Check | Method |
|-------|--------|
| All 13 modules build | `go build ./...` |
| All tests pass | `go test ./...` |
| CI gates green | PR verification |
| Codegen drift-free | `scripts/codegen-integrated-check.sh` |
| No new P0 debt | Registry audit |
| Doc count in target range | `ls docs/architecture/ | wc -l` |
| Archive structure complete | Manual review |

**Exit:** Phase complete. Tag repository at `refactoring-phase-exit`.

---

## 7. Constraints and Rules

1. **One commit per logical unit.** No mega-commits mixing code changes with doc moves.
2. **No feature work.** Any feature discovered during cleanup gets added to the debt registry, not implemented.
3. **Archive before delete.** Always.
4. **Test green at every commit.** No commit should break build or tests.
5. **Debt registry is living.** New items discovered during cleanup are added with priority.
6. **No scope creep.** If work exceeds the defined scope, stop and re-evaluate.

---

## 8. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Documentation consolidation loses unique content | Medium | High | Archive originals; review each consolidation; git history as safety net |
| Module consolidation introduces import cycles | Low | High | Analysis-only in this phase; defer execution if complex |
| CI verification reveals pre-existing failures | Medium | Medium | Fix failures as P0 before proceeding |
| Scope creep into feature work | Medium | Medium | Strict scope rules; debt registry for new items |
| Handler extraction (MF-1) is more complex than estimated | Low | Low | Well-scoped since S189; function boundary is clear |

---

## 9. Success Metrics

| Metric | Target |
|--------|--------|
| Architecture docs (active) | 120-150 (from 440) |
| P0 debt items open | 0 |
| P1 debt items open | 0 |
| Go modules building | 13/13 |
| Unit tests passing | 100% |
| CI gates verified | All |
| Archive populated with originals | Yes |
| Stage report index created | Yes |

---

## 10. Relationship to S210

S210 is the **final stabilization gate** before this phase begins. S210 should:
1. Verify the S209 deliverables are complete and accurate.
2. Run the entry prerequisite checks (Section 5).
3. Tag the repository.
4. Formally authorize the refactoring phase.

This phase does not begin until S210 passes.
