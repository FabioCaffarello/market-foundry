# Next-Wave Recommendations After Post-Refactor and Documentation Gate

**Date:** 2026-03-20
**Gate:** S216 — Post-Refactor and Documentation Exit Gate
**Gate verdict:** CONDITIONAL PASS

---

## 1. Immediate Next Action: Closing Tranche (Mandatory)

Before any expansion or new functional work, one short focused tranche must close the gate cleanly.

### Scope (strictly bounded)

| # | Item | Effort | Exit criterion addressed |
|---|------|--------|--------------------------|
| 1 | Archive ~90 docs from `docs/architecture/` to reach ≤150 active | 1 session | XC-1 |
| 2 | Extract `parseAnalyticalParams()` from `analytical.go` | Small | MF-1, XC-2 |
| 3 | Push and verify CI pipeline green | Mechanical | EC-7, XC-6 |
| 4 | Update debt registry to reflect S211–S215 outcomes | Small | XC-13 |
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

### Path B: Execute Remaining HIGH Structural Refactoring
**When:** If the priority is reducing long-term evolution cost before adding more families.

**Prerequisites:**
- All XC criteria PASS
- Same freeze model as S211 (new charter, new exit criteria)

**Scope:**
1. H-01: NATS adapter sub-packaging (73 files → organized sub-packages)
2. H-04 completion: Per-family actor migration to `GenericConsumerActor` (~1,800 lines recovered)
3. H-06: Module graph evaluation and simplification (19 → ~10 modules)

**Duration:** 2–3 focused sessions.

**Value:** Eliminates the three largest remaining duplication/complexity clusters. After this, adding a new family becomes a 5-file, single-package operation instead of a 15-file, 8-package operation.

**Risk:** Low — infrastructure for H-04 already exists. H-01 is organizational. H-06 requires evaluation first.

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

**Path B first, then Path A.**

Rationale:
- The S212 census identified NATS adapter scale, actor duplication, and module count as the three largest structural costs. Two of three have infrastructure ready (H-04) or clear mechanical steps (H-01).
- Adding families on top of the current 19-module, 73-file NATS adapter will compound the blast radius problem.
- Path B is bounded (2–3 sessions) and has clear exit criteria (same model as S211).
- After Path B, Path A becomes cheaper per family added.

If business pressure requires immediate capability expansion, Path A is viable — but set the blast radius cap described above.

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
