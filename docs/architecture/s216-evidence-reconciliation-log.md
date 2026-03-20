# S216 Evidence Reconciliation Log

**Date:** 2026-03-20
**Stage:** S217 — Exit Gate Closure and Evidence Reconciliation
**Purpose:** Document every discrepancy found between S216 gate documentation and actual repository state, and record the corrections applied.

---

## Methodology

S217 performed systematic code inspection against every claim in the S216 exit gate, gains/tradeoffs, and next-wave recommendations documents. Each item was verified by reading actual source files, counting actual documents, and checking git state.

---

## Discrepancies Found and Corrected

### DISC-01: MF-1 Handler Extraction — MAJOR

| Aspect | S216 Claim | Actual State |
|--------|------------|--------------|
| `parseAnalyticalParams()` | NOT DONE | **EXISTS** at `internal/interfaces/http/handlers/analytical.go:90-122` |
| `analytical.go` line count | 615/620 (at ceiling) | **502 lines** (well under ceiling) |
| XC-2 status | FAIL (1 P0 remaining) | **PASS** (0 P0 remaining) |

**Root cause:** S216 review relied on documentation claims from S189/S205/S209 without verifying against current code. The extraction was implemented at some point between S189 (when it was scoped) and S216 (when it was reviewed), but no stage report recorded its completion.

**Impact:** XC-2 flips from FAIL to PASS. MF-1 is no longer a blocker. The closing tranche scope shrinks from 5 items to 3.

**Corrections applied:**
- `post-refactor-and-documentation-exit-gate.md` — XC-2, MF-1, Section 4.2, Section 5, Section 6, Section 7 updated
- `refactor-wave-gains-tradeoffs-and-open-debts.md` — Process debts table, debt disposition table updated
- `next-wave-recommendations-after-post-refactor-and-documentation-gate.md` — Closing tranche scope updated
- `pre-refactor-technical-debt-registry-and-cleanup-plan.md` — TD-01 marked RESOLVED

---

### DISC-02: Active Architecture Doc Count — MINOR

| Aspect | S216 Claim | Actual State |
|--------|------------|--------------|
| Active docs in `docs/architecture/` | 240 | **243** |
| Stage reports in `docs/stages/` | 214 (INDEX claim: 212) | **214** (INDEX was 2 behind) |

**Root cause:** S215 counted docs at the moment of consolidation. S216 added 3 new documents (exit gate, gains/tradeoffs, next-wave recommendations) without incrementing the count. INDEX.md was created during S215 before S215's own report and S216 reports were added.

**Impact:** Minor. XC-1 gap is 93 docs (not 90). No status change.

**Corrections applied:**
- `documentation-canonical-map-after-consolidation.md` — counts corrected to 243/214, note added
- `post-refactor-and-documentation-exit-gate.md` — XC-1 actual corrected to 243
- `refactor-wave-gains-tradeoffs-and-open-debts.md` — doc count references corrected

---

### DISC-03: P1 Items Status — MODERATE

| Item | S216 Claim | Actual State |
|------|------------|--------------|
| AD-03 (superseded docs unmarked) | ~3 unresolved | **RESOLVED** (S215 archived 245 docs) |
| AD-04 (per-family doc boilerplate) | ~3 unresolved | **RESOLVED** (S215 consolidated family lifecycle docs) |
| AD-06 (stage report index missing) | ~3 unresolved | **RESOLVED** (S215 created INDEX.md) |
| TD-02 (reader 10-param signature) | Unresolved | **Formally deferred** (trigger: Family 07) |
| AD-01 (module count) | Unresolved | **Formally deferred** (tracked as H-06) |

**Root cause:** S216 counted ~3 P1 items as unresolved without checking which ones had been addressed by S215 consolidation.

**Impact:** XC-3 flips from PARTIAL to PASS. All P1 items are now either resolved or formally deferred with justification.

**Corrections applied:**
- `post-refactor-and-documentation-exit-gate.md` — XC-3 status updated
- `pre-refactor-technical-debt-registry-and-cleanup-plan.md` — AD-03, AD-04, AD-05, AD-06 marked resolved; TD-02, AD-01 formally deferred

---

### DISC-04: XC-13 Debt Registry — MINOR

| Aspect | S216 Claim | Actual State |
|--------|------------|--------------|
| XC-13 | PARTIAL | **PASS** (S217 updated all items in registry) |

**Root cause:** S216 noted the registry needed updating but didn't perform the update.

**Impact:** XC-13 flips from PARTIAL to PASS.

**Corrections applied:**
- `pre-refactor-technical-debt-registry-and-cleanup-plan.md` — TD-01, AD-02, AD-03, AD-04, AD-05, AD-06 statuses updated; TD-02, AD-01 formally deferred
- `post-refactor-and-documentation-exit-gate.md` — XC-13 status updated

---

## Items Verified as Correct (No Drift)

| Item | S216 Claim | S217 Verification |
|------|------------|-------------------|
| Consumer spec factory (H-02) | EXISTS, used by all 6 registries | **CONFIRMED** — `newConsumerSpec()` in all 6 registry files |
| Query builder (H-03) | EXISTS, used by readers | **CONFIRMED** — `BuildQuery()` used by candle_reader.go, signal_reader.go |
| Generic actor (H-04) | EXISTS, not yet adopted | **CONFIRMED** — `GenericConsumerActor` exists; store supervisor still uses per-family actors |
| Ownership annotations | 7 files annotated | **CONFIRMED** — `manual:owned` in 6 registry files + cmd/writer/pipeline.go |
| Golden snapshot drift | RSI/EMA use literal, not factory | **CONFIRMED** — consumer_spec.go.golden shows `ConsumerSpec{}` literal |
| Archive structure | 245 docs in 16 categories | **CONFIRMED** — 245 files across 16 subdirectories |
| Expansion freeze | Zero violations | **CONFIRMED** — no new families, endpoints, services, schemas |
| CI workflow | Exists with 3 jobs | **CONFIRMED** — unit-tests, codegen-golden, smoke-analytical |
| Repository tags | Neither tag exists | **CONFIRMED** — `stabilization-exit-s210` and `refactoring-phase-exit` not in git |

---

## Reconciled Exit Criteria Summary (Post-S217)

| ID | Target | S216 Status | S217 Status | Change |
|----|--------|-------------|-------------|--------|
| XC-1 | ≤150 docs | FAIL (240) | **FAIL** (243) | Count corrected, still fails |
| XC-2 | 0 P0 | FAIL (1) | **PASS** (0) | MF-1 confirmed done |
| XC-3 | 0 P1 or deferred | PARTIAL (~3) | **PASS** (3 resolved, 2 deferred) | Items reconciled |
| XC-4 | Build | PASS | **PASS** | No change |
| XC-5 | Tests | PASS | **PASS** | No change |
| XC-6 | CI green | PENDING | **PENDING** | No change — requires push |
| XC-7 | Codegen | PASS | **PASS** | No change |
| XC-8 | Archive | PASS | **PASS** | No change |
| XC-9 | Index | PASS | **PASS** | No change |
| XC-10 | No new P0 | PASS | **PASS** | No change |
| XC-11 | Tag | PENDING | **PENDING** | No change — requires push first |
| XC-12 | Freeze | PASS | **PASS** | No change |
| XC-13 | Registry | PARTIAL | **PASS** | Registry updated in S217 |

**S216 score:** 6 PASS, 2 PENDING, 2 PARTIAL, 2 FAIL
**S217 score:** 9 PASS, 2 PENDING, 0 PARTIAL, 1 FAIL

---

## Documents Modified in S217

| File | Changes |
|------|---------|
| `docs/architecture/post-refactor-and-documentation-exit-gate.md` | XC-1/2/3/13 status, MF-1 status, Section 4.2, Section 5, Section 6, Section 7 |
| `docs/architecture/refactor-wave-gains-tradeoffs-and-open-debts.md` | Doc counts, MF-1 status, debt disposition |
| `docs/architecture/next-wave-recommendations-after-post-refactor-and-documentation-gate.md` | Closing tranche scope reduced |
| `docs/architecture/documentation-canonical-map-after-consolidation.md` | Counts corrected (243/214) |
| `docs/architecture/pre-refactor-technical-debt-registry-and-cleanup-plan.md` | TD-01, AD-02, AD-03, AD-04, AD-05, AD-06, TD-02, AD-01 statuses updated |
| `docs/stages/INDEX.md` | S215, S216, S217 entries added |
