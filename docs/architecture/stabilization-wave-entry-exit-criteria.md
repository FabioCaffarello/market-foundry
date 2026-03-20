# Stabilization Wave Entry and Exit Criteria

**Stage:** S205
**Date:** 2025-07-24
**Status:** Active

---

## Purpose

This document defines the formal gates for entering and exiting the stabilization wave. No work may begin until entry criteria are met. The stabilization wave does not close until all exit criteria are satisfied. The next phase (strategic refactoring and restructuring) cannot begin until this wave is formally closed.

---

## Entry Criteria

All must be true before stabilization execution begins:

| # | Criterion | Verification Method | Status |
|---|-----------|-------------------|--------|
| EC-1 | S205 scope freeze matrix is published and reviewed | This document and siblings exist in `docs/architecture/` | DONE |
| EC-2 | Must-finish list is finite and bounded (≤10 items) | Matrix contains exactly 7 MF items | DONE |
| EC-3 | No active feature branches with in-flight expansion work | `git branch` shows no expansion branches | VERIFY |
| EC-4 | Freeze boundaries are documented and explicit | 12 EF items documented with rationale | DONE |
| EC-5 | Responsibility map assigns every MF item to a track | All 7 items have owner tracks | DONE |

**Entry verdict:** Stabilization wave may begin once EC-3 is verified.

---

## Exit Criteria

All must be true before the stabilization wave closes and the refactoring phase may begin:

| # | Criterion | Verification Method | Blocking? |
|---|-----------|-------------------|-----------|
| XC-1 | MF-1 complete: `parseAnalyticalParams()` extracted | `grep -c parseAnalyticalParams internal/interfaces/http/handlers/analytical.go` ≥ 7 (1 definition + 6 calls) | YES |
| XC-2 | MF-1 verified: handler file within ceiling | `wc -l internal/interfaces/http/handlers/analytical.go` ≤ 510 | YES |
| XC-3 | MF-2 complete: CI smoke-analytical passes on branch | GitHub Actions shows green for smoke-analytical job | YES |
| XC-4 | MF-3 complete: codegen integrated check passes | `make codegen-integrated` exits 0, all 7 families verified | YES |
| XC-5 | MF-4 complete: writer binary removed | `git ls-files cmd/writer/writer` returns empty | YES |
| XC-6 | MF-5 complete: all modules build | `go build ./...` per module exits 0 for all 13 modules | YES |
| XC-7 | MF-6 complete: all tests pass | `make test` exits 0 | YES |
| XC-8 | MF-7 complete: cross-spec validation passes | `make codegen-validate-all` exits 0 | YES |
| XC-9 | No freeze violations committed | Git log since stabilization entry shows no changes to frozen areas | YES |
| XC-10 | Stage report S205 published | `docs/stages/stage-s205-*-report.md` exists | YES |

**Exit verdict:** Stabilization wave closes when all 10 XC criteria are satisfied. Any single failure blocks exit.

---

## What the Refactoring Phase Inherits

When the stabilization wave closes, the next phase receives:

### Clean Baseline
- All 13 modules compile
- All unit tests pass
- CI pipeline green (unit tests + codegen golden + smoke analytical)
- Handler file within safe line budget (~501 lines, runway for ~2 more families)
- Codegen governance verified (7 specs, 14 goldens, integrated check passing)
- No binaries in version control

### Explicitly Deferred Work (17 MD items)
- Documented in the scope freeze matrix with rationale for each deferral
- None create structural risk during refactoring
- Some may be addressed naturally during refactoring (e.g., MD-10 reader signature, MD-14 test assertions)

### Frozen Boundaries (12 EF items)
- Remain frozen during refactoring unless a refactoring stage explicitly unfreezes them with documented rationale
- Template freeze (EF-2), spec freeze (EF-3), and golden reference freeze (EF-4) carry forward
- Family expansion freeze (EF-1) may be lifted only after refactoring phase completes

### Open Debt Inventory
- All debts cataloged in existing `*gains-tradeoffs-and-open-debts*` documents remain authoritative
- Stabilization does not retire debts — it verifies the foundation is safe for the phase that will address them

---

## Timeline Constraint

The stabilization wave is designed to complete in a **single focused session** (~5–6 hours of execution). If it exceeds one session:
- Re-evaluate whether a must-finish item was misclassified (should it be may-defer?)
- Do not extend by adding work — extend only if existing MF items take longer than expected
- Under no circumstances add new items to the must-finish list during execution

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why Dangerous | Correct Action |
|--------------|---------------|----------------|
| "While I'm here, let me also fix..." | Scope creep disguised as convenience | Log it as a note for refactoring phase |
| "This test is flaky, let me rewrite it" | Rewriting is refactoring, not stabilization | Make it pass; fix root cause in refactoring |
| "The docs are messy, let me reorganize" | Documentation cleanup is next phase | Do not touch docs beyond S205 deliverables |
| "This code is ugly but works" | Aesthetics are refactoring concerns | If it compiles and tests pass, it exits stabilization |
| "Let me add one more family since H-5 is done" | Expansion during stabilization violates EF-1 | Family 06+ is post-refactoring |
