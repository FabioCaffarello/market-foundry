# Next-Wave Recommendations After Post-Refactor and Documentation Gate

**Date:** 2026-03-20
**Gate:** S216 — Post-Refactor and Documentation Exit Gate
**Gate verdict:** CONDITIONAL PASS (S217 reconciled — remaining blockers reduced from 5 to 3)
**S221 Reconciliation:** Path B executed in S218–S220. H-01, H-04, H-06 completed. Recommendations below reflect pre-execution state; see S221 reconciliation log for current status.

---

## 1. Immediate Next Action: Closing Tranche (Mandatory)

Before any expansion or new functional work, one short focused tranche must close the gate cleanly.

### Scope (strictly bounded)

| # | Item | Effort | Exit criterion addressed |
|---|------|--------|--------------------------|
| 1 | Archive ~93 docs from `docs/architecture/` to reach ≤150 active | 1 session | XC-1 |
| 2 | ~~Extract `parseAnalyticalParams()` from `analytical.go`~~ | ~~Small~~ | ~~MF-1, XC-2~~ — **S217: confirmed already done** |
| 3 | Push and verify CI pipeline green | Mechanical | EC-7, XC-6 |
| 4 | ~~Update debt registry to reflect S211–S215 outcomes~~ | ~~Small~~ | ~~XC-13~~ — **S217: done in reconciliation** |
| 5 | Tag repository `refactoring-phase-exit-s216` | Mechanical | XC-11 |

### What this tranche does NOT permit
- New families, endpoints, services, schemas (freeze still active)
- NATS restructuring, module consolidation, actor migration
- Codegen template or spec changes
- Performance work or dependency upgrades

### Exit condition
All 13 XC criteria PASS. Tag created. Freeze lifts.

---

## 2. After Gate Closes: Three Viable Paths

The Foundry has three defensible options after the closing tranche. The recommendation depends on strategic priorities.

### Path A: Resume Controlled Analytical/Functional Expansion
**When:** If the priority is adding capability (new families, new domains, new query surfaces).

**Prerequisites:**
- All XC criteria PASS
- Debt registry current
- CI verified green

**First actions:**
1. Family 06 trigger assessment (was deferred at S191)
2. Or: new domain expansion (observation, or venue execution hardening)
3. Or: codegen expansion to additional families

**Risk:** Structural debt (H-01, H-04, H-06) will increase friction per family added. The family-expansion blast radius is reduced (consumer spec factory, query builder) but not eliminated (NATS flat, actors not migrated, module count high).

**Mitigation:** Set a hard cap — e.g., "if adding Family 07 requires touching >12 files, trigger H-01/H-04 before proceeding."

### Path B: Execute Remaining HIGH Structural Refactoring — **COMPLETED (S218–S220)**
**Status:** EXECUTED in S218–S220 tranche.

**What was delivered:**
1. **H-01 (S218):** NATS adapter sub-packaging — flat structure → 8 domain sub-packages (`natskit`, `natsconfigctl`, `natsdecision`, `natsevidence`, `natsexecution`, `natsobservation`, `natsrisk`, `natssignal`, `natsstrategy`)
2. **H-04 (S219):** Per-family actor migration — 9 consumer actors → 1 `GenericConsumerActor`, 8 files deleted, ~510 lines recovered
3. **H-06 (S220):** Module graph simplification — 19→17 modules, 2 absorbed (`internal/migrate`, `internal/adapters/repositories`), zero new dependency edges

**Outcome:** All 3 HIGH structural items completed. Family-expansion blast radius reduced as projected. Next path: Path A (controlled expansion) or Path C (pause).

### Path C: Pause and Wait for External Signal
**When:** If there's no immediate pressure to expand functionality and structural state is acceptable.

**First actions:**
1. Close the exit tranche
2. Tag the repository
3. Document the pause reason
4. Resume when business need dictates direction

**Risk:** Minimal. The codebase is stable and tested. Debt doesn't grow if nothing is added.

---

## 3. Recommendation

**Path B first, then Path A.** — **S221 update: Path B is now complete.**

S218–S220 executed the recommended Path B. The Foundry is now in the post-Path-B state:
- NATS adapter is domain-organized (8 sub-packages)
- Store consumer actors are unified (1 generic actor)
- Module graph simplified (17 modules, 2 absorbed)

**Current recommendation:** Path A (controlled expansion) is now viable at reduced blast radius. Alternatively, Path C (pause) remains defensible if no immediate pressure.

---

## 4. Items Explicitly NOT Recommended

| Item | Reason |
|------|--------|
| Deep documentation rewrite for conciseness | Diminishing returns. 150-doc target is sufficient. |
| Domain subdirectory reorganization | Optional cosmetic. No functional benefit. |
| Module graph simplification without evidence | H-06 should be evaluated, not assumed. |
| Golden snapshot drift fix | Requires codegen template change (frozen). Fix when templates are next touched. |
| Performance optimization | No evidence of performance problems. Premature. |
| Dependency upgrades | No security issues flagged. Defer to dedicated maintenance window. |

---

## 5. Guard Rails for Next Phase (Regardless of Path)

1. **New charter required** — Do not reuse S211 charter. Each wave gets its own governance.
2. **New exit criteria** — Defined before work begins, not after.
3. **Expansion freeze model proven** — Reuse the 17-item freeze matrix pattern. It worked.
4. **Blast radius monitoring** — Track files-touched-per-family as a leading indicator.
5. **CI verification mandatory** — No phase opens without green CI on real push.
6. **Debt registry maintenance** — Update after each stage, not just at gates.

---

## 6. Timeline Projection

| Phase | Estimated Sessions | Depends On |
|-------|-------------------|------------|
| Closing tranche (mandatory) | 1 | Nothing — start immediately |
| Path B: Remaining HIGH refactoring | 2–3 | Closing tranche complete |
| Path A: Next functional expansion | Open-ended | Path B complete (recommended) or closing tranche (minimum) |
