# Next Wave Recommendations After Pre-Refactor Stabilization Gate

**Stage:** S210
**Date:** 2026-03-20
**Gate result:** CONDITIONAL PASS
**Predecessor recommendations:** `next-wave-recommendations-after-post-generated-family-gate.md` (S204)

---

## 1. Recommended Next Wave

**Strategic Refactoring and Documentation Consolidation Phase**

This is the only recommended next action. No expansion, no new features, no new services.

---

## 2. Entry Sequence (ordered, not parallelizable)

### Step 1: Push and verify CI (closes MF-2)

Push the current state to origin and verify the CI pipeline passes, including the `smoke-analytical` job. This is the single outstanding verification from the stabilization gate.

**Blocked by:** nothing — ready to execute immediately.
**Blocks:** all subsequent steps.

### Step 2: Tag repository

```
git tag stabilization-exit-s210
```

This tag is the rollback point for the entire refactoring phase.

### Step 3: Enter Wave 1 — Entry Gate Closure

All S205 MF items are now verified (Step 1 closes MF-2). No further implementation work is needed before beginning Wave 2.

If CI reveals failures, fix them as P0 before proceeding. Do not skip.

### Step 4: Enter Wave 2 — Documentation Cleanup

Follow the 12-phase execution order from `documentation-entropy-archive-delete-consolidate-map.md` (S209). Execute one cluster at a time, one commit per logical unit.

**Expected outcome:** ~440 → 120-150 active architecture docs.

### Step 5: Enter Wave 3 — Code Debt Cleanup

Address P1 items from `pre-refactor-technical-debt-registry-and-cleanup-plan.md` (S209):
- Reader signature refactoring (TD-02)
- Module graph evaluation (AD-01)
- Test hardcoded family counts (TD-03)

### Step 6: Enter Wave 4 — Verification and Exit Gate

Full build + test + CI + codegen verification. Tag repository at `refactoring-phase-exit`. Produce exit gate report.

---

## 3. What Must NOT Happen During the Refactoring Phase

These items from S205 EF (Explicitly Frozen) remain frozen:

| ID | Frozen Item | Why Still Frozen |
|----|-------------|-----------------|
| EF-1 | New analytical family expansion | Refactoring phase is structural, not functional |
| EF-2 | Codegen template modification | Templates frozen per S193; modification requires new stage |
| EF-3 | Codegen spec schema extension | 14-field schema sufficient; extension requires new stage |
| EF-4 | Retroactive manual-to-generated conversion | 6 manual families are permanent golden references |
| EF-5 | Tier 2 codegen authorization | Not designed, not validated |
| EF-8 | New NATS stream definitions | Infrastructure expansion is not refactoring |
| EF-9 | ClickHouse schema changes | Schema frozen for current scope |
| EF-11 | Writer pipeline structural changes | Write path is proven stable |
| EF-12 | New service introduction | No new `cmd/*` services |

**New freeze items for refactoring phase:**

| ID | Frozen Item | Why |
|----|-------------|-----|
| RF-1 | New domain entities or events | Refactoring does not add business logic |
| RF-2 | New HTTP endpoints or routes | Refactoring restructures, not extends |
| RF-3 | Dependency version upgrades | Version changes introduce untested paths; address post-refactoring |
| RF-4 | Performance optimization work | Structural changes must complete before performance tuning |

---

## 4. Conditional Recommendations (trigger-based)

### If CI smoke-analytical fails on real PR:
- Investigate immediately. Do not enter Wave 2 until resolved.
- Likely causes: Docker compose timing, ClickHouse initialization, service port conflicts.
- Fix, re-push, verify green.

### If documentation consolidation discovers lost content:
- Stop the current cluster consolidation.
- Verify originals are in `docs/archive/`.
- Review the consolidation for content gaps before proceeding.

### If module graph evaluation reveals tight coupling:
- Document findings but do not execute module merges in this phase.
- Module boundary changes are high-risk and require their own stage.
- Add to debt registry as P1 for post-refactoring evaluation.

### If new debt is discovered during refactoring:
- Add to `pre-refactor-technical-debt-registry-and-cleanup-plan.md` with priority classification.
- Do not implement fixes for newly discovered items unless they are P0 blockers.

---

## 5. Post-Refactoring Phase Recommendations

After the refactoring phase exit gate passes, the following are the candidate next waves (in priority order):

1. **Production readiness assessment** — Load testing baseline, operational monitoring, alerting setup.
2. **Codegen expansion evaluation** — Decide whether to integrate remaining 5 families, based on refactoring phase experience.
3. **TC-02 planning** — State persistence, WAL, cold-start bootstrap architecture.
4. **Venue expansion** — Real exchange connectivity beyond paper trading.

These are not authorized. They are candidates for evaluation at the post-refactoring gate.

---

## 6. Success Criteria for Refactoring Phase Exit

| Criteria | Target |
|----------|--------|
| Active architecture docs | 120-150 (from 440) |
| P0 debt items | 0 |
| P1 debt items | 0 |
| All 19 modules building | Yes |
| All unit tests passing | Yes |
| CI gates green | Yes |
| Archive populated | Yes |
| Stage report index created | Yes |
| No new P0 items introduced | Yes |
| Repository tagged at exit | Yes |

---

## 7. Timeline Expectation

No timeline is prescribed. The refactoring phase should proceed at whatever pace maintains quality. Rushing documentation consolidation risks content loss. Rushing code cleanup risks regressions.

The wave structure (entry → docs → code → verification) allows stopping at any boundary if priorities change.
