# Refactor Wave Charter and Entry Freeze

**Stage:** S211
**Date:** 2026-03-20
**Status:** ACTIVE — Phase formally opened and frozen.
**Predecessor:** S210 (Pre-Refactor Stabilization Gate — CONDITIONAL PASS)

---

## 1. Charter Declaration

This document is the **governing charter** for the Strategic Refactoring and Documentation Consolidation Phase of market-foundry. It establishes the formal authority, scope, rules, and governance for the entire phase — from entry through exit gate.

**Phase name:** Strategic Refactoring and Documentation Consolidation
**Phase identifier:** Refactor Wave (RW)
**Governing authority:** This charter. All decisions during the phase must be traceable to this document.
**Effective from:** 2026-03-20 (S211 completion)
**Effective until:** Refactor Wave exit gate passes (future stage TBD)

---

## 2. Entry Condition

The S210 gate issued a **CONDITIONAL PASS**. The single outstanding condition is mechanical:

| Condition | Nature | Status |
|-----------|--------|--------|
| Push to remote and verify CI pipeline passes (closes MF-2) | Verification, not implementation | PENDING — first action of phase entry |
| Tag repository at `stabilization-exit-s210` | Mechanical | PENDING — immediately after CI passes |

**Hard rule:** Wave 2 (documentation cleanup) MUST NOT begin until CI verification is confirmed green. If CI fails, the failure must be diagnosed and resolved as P0 before any refactoring work proceeds.

**All other S205 MF items are VERIFIED:**

| ID | Item | Verification |
|----|------|-------------|
| MF-1 | `parseAnalyticalParams()` extraction | DONE — handler at 502 lines |
| MF-3 | Codegen integrated check (all 7 families) | VERIFIED — 14/14 golden, 4/4 integrated |
| MF-4 | Writer binary removed from VCS | DONE — gitignore patterns confirmed |
| MF-5 | All 19 Go modules build cleanly | VERIFIED — zero errors |
| MF-6 | All unit tests pass | VERIFIED — all packages pass |
| MF-7 | Codegen cross-spec validation | VERIFIED — 7/7 valid, no collisions |

---

## 3. Phase Objectives

The Refactor Wave has exactly three objectives. Nothing else is authorized.

### Objective 1: Documentation Consolidation
Reduce architectural document entropy from ~440 files to 120–150 active documents. Execute the 12-phase plan from `documentation-entropy-archive-delete-consolidate-map.md` (S209). Archive superseded content. Create navigable, cross-linked documentation.

### Objective 2: Code Debt Cleanup
Address P0 and P1 items from the technical debt registry (`pre-refactor-technical-debt-registry-and-cleanup-plan.md`, S209). Specifically:
- TD-02: Reader 10-parameter positional signature → options pattern or builder
- AD-01: Module graph evaluation (document findings; execute only if safe)
- TD-03: Test hardcoded family counts → registry-driven
- AD-03: Superseded document marking
- AD-04: Per-family doc boilerplate consolidation
- AD-06: Stage report index

### Objective 3: Verification and Exit
Confirm the system remains fully operational after all structural changes. Full build, test, CI, and codegen verification. Tag repository. Produce exit gate report.

---

## 4. Expansion Freeze

**This is the central governance rule of the entire phase.**

> No functional expansion of any kind is permitted until the Refactor Wave exit gate passes.

This freeze is **absolute and non-negotiable**. It cannot be overridden by urgency, opportunity, or convenience. The only exception is a P0 blocker that prevents the refactoring work itself from proceeding — and such exceptions must be documented as emergency deviations with explicit justification.

### Frozen Items (carried from S205 EF + S210 RF)

| ID | Frozen Item | Origin | Rationale |
|----|-------------|--------|-----------|
| EF-1 | New analytical family expansion | S205 | Phase is structural, not functional |
| EF-2 | Codegen template modification | S205/S193 | Templates frozen; modification requires new stage |
| EF-3 | Codegen spec schema extension | S205/S193 | 14-field schema sufficient; extension requires new stage |
| EF-4 | Retroactive manual-to-generated conversion | S205 | 6 manual families are permanent golden references |
| EF-5 | Tier 2 codegen authorization | S205 | Not designed, not validated |
| EF-8 | New NATS stream definitions | S205 | Infrastructure expansion is not refactoring |
| EF-9 | ClickHouse schema changes | S205 | Schema frozen for current scope |
| EF-11 | Writer pipeline structural changes | S205 | Write path is proven stable |
| EF-12 | New service introduction | S205 | No new `cmd/*` services |
| RF-1 | New domain entities or events | S210 | Refactoring does not add business logic |
| RF-2 | New HTTP endpoints or routes | S210 | Refactoring restructures, not extends |
| RF-3 | Dependency version upgrades | S210 | Version changes introduce untested paths |
| RF-4 | Performance optimization work | S210 | Structural changes must complete first |

### New Freeze Items for S211

| ID | Frozen Item | Rationale |
|----|-------------|-----------|
| RF-5 | Mass documentation deletion without archive | Content must be preserved before removal |
| RF-6 | Module boundary changes (merge/split) without documented evaluation | AD-01 requires evaluation first; execution only if risk is assessed and accepted |
| RF-7 | New architecture decision records | Phase is executing existing decisions, not making new ones |
| RF-8 | Changes to CI job definitions beyond bug fixes | CI is the safety net; do not modify the net during the work it protects |

---

## 5. Wave Structure

The Refactor Wave follows the 4-wave structure defined in S209:

| Wave | Name | Scope | Entry Condition |
|------|------|-------|-----------------|
| **RW-1** | Entry Gate Closure | CI verification, repository tag, charter activation | S211 complete |
| **RW-2** | Documentation Cleanup | 12-phase entropy reduction per S209 plan | CI green, tag applied |
| **RW-3** | Code Debt Cleanup | P1 debt items from registry | RW-2 complete or at stable checkpoint |
| **RW-4** | Verification and Exit Gate | Full system verification, exit report | RW-3 complete |

**Sequential rule:** Waves are sequential. RW-2 must not begin before RW-1 closes. RW-3 must not begin before RW-2 reaches a stable checkpoint (not necessarily full completion, but all in-progress clusters committed). RW-4 must not begin before RW-3 completes.

---

## 6. Governance Model

### Decision Authority
- **In scope (per this charter):** proceed without additional authorization.
- **Ambiguous (not clearly in scope):** stop, document the question, defer to next conversation for a ruling. Do not guess.
- **Out of scope:** prohibited. If it is truly needed, it requires a formal charter amendment with justification.

### Change Control
- New debt items discovered during refactoring: add to the debt registry with priority classification.
- New debt items classified as P0: may be addressed immediately if they block refactoring work. Must be documented.
- New debt items classified as P1+: registered but deferred to the appropriate wave. No opportunistic fixes.
- Charter amendments: require explicit user authorization and a new section in this document recording the amendment, date, and justification.

### Progress Tracking
- Each stage within the Refactor Wave produces a stage report.
- The debt registry (`pre-refactor-technical-debt-registry-and-cleanup-plan.md`) is the single source of truth for item status.
- Items completed during the wave are marked DONE with stage number and date.
- No item may be silently dropped. Items can only be DONE, DEFERRED (with justification), or SUPERSEDED (with pointer to replacement).

### Rollback Point
The `stabilization-exit-s210` tag is the rollback point for the entire phase. If the refactoring phase must be abandoned, the system can be restored to this tag.

---

## 7. Success Criteria

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Active architecture docs | 120–150 (from ~440) | File count in `docs/architecture/` excluding `docs/archive/` |
| P0 debt items remaining | 0 | Debt registry |
| P1 debt items remaining | 0 | Debt registry |
| All 19 Go modules building | Yes | `go build ./...` per module |
| All unit tests passing | Yes | `make test` |
| CI gates green | Yes | CI pipeline run |
| Codegen gates passing | Yes | 4 codegen CI gates |
| Archive populated with superseded content | Yes | `docs/archive/` exists and contains consolidated originals |
| Stage report index created | Yes | Index document exists |
| No new P0 items introduced | Yes | Debt registry audit |
| Repository tagged at exit | Yes | `refactoring-phase-exit` tag |
| No frozen items violated | Yes | Charter compliance audit |

---

## 8. What This Phase Is NOT

- **Not a feature sprint.** No new capabilities, families, endpoints, or services.
- **Not a performance phase.** No optimization, no load testing execution (baseline is registered debt, not phase work).
- **Not an infrastructure phase.** No ClickHouse changes, no NATS changes, no new Docker services.
- **Not a design phase.** No new architecture decisions. The phase executes the existing S209 plan.
- **Not unbounded.** The phase has explicit exit criteria. It ends when those criteria are met.

---

## 9. Operating Documents

The following documents are the operating references for this phase:

| Document | Purpose | Path |
|----------|---------|------|
| This charter | Governance, scope, freeze rules | `docs/architecture/refactor-wave-charter-and-entry-freeze.md` |
| Permitted vs Prohibited | Detailed change classification | `docs/architecture/refactor-wave-permitted-vs-prohibited-changes.md` |
| Entry/Exit/Freeze Criteria | Formal criteria definitions | `docs/architecture/refactor-wave-entry-exit-and-freeze-criteria.md` |
| Technical Debt Registry | Item tracking | `docs/architecture/pre-refactor-technical-debt-registry-and-cleanup-plan.md` |
| Documentation Entropy Map | Cleanup execution plan | `docs/architecture/documentation-entropy-archive-delete-consolidate-map.md` |
| S210 Gate Review | Entry authorization | `docs/architecture/pre-refactor-stabilization-gate.md` |
