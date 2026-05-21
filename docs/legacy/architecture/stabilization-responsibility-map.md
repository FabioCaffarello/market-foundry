# Stabilization Responsibility Map

**Stage:** S205
**Date:** 2025-07-24
**Status:** Active during stabilization wave

---

## Purpose

This document maps every must-finish item and every freeze boundary to a conceptual owner track. The owner track is not a person — it is the subsystem or concern area responsible for the item. This prevents orphaned work and ensures every obligation has a clear home.

---

## Owner Tracks

### Track 1: Gateway / Handlers

**Scope:** HTTP handler layer, analytical endpoints, request parsing, response formatting.

| Item | Type | Action Required |
|------|------|-----------------|
| MF-1: H-5 `parseAnalyticalParams()` extraction | Must Finish | Extract repeated limit/since/until parsing into shared helper. Reduce handler from 615 → ~501 lines. |
| EF-1: No Family 06+ expansion | Freeze | Do not add new handler methods. Handler file at ceiling. |

**Verification:** `wc -l internal/interfaces/http/handlers/analytical.go` ≤ 510 after MF-1.

**Dependencies:** None. Self-contained change within `analytical.go`.

---

### Track 2: CI / Build Verification

**Scope:** GitHub Actions pipeline, build gates, test execution, smoke tests.

| Item | Type | Action Required |
|------|------|-----------------|
| MF-2: CI smoke-analytical stability | Must Finish | Trigger CI pipeline on a test branch. Verify smoke-analytical job completes successfully. Document result. |
| MF-5: All modules build cleanly | Must Finish | Run `go build ./...` for each of 13 modules. Fix any compilation errors. |
| MF-6: All unit tests pass | Must Finish | Run `make test`. Fix any pre-existing failures. |

**Verification:** CI green on stabilization branch. `make test` exits 0 locally.

**Dependencies:** MF-5 before MF-6 (must compile before testing). MF-2 independent.

---

### Track 3: Codegen Governance

**Scope:** Codegen engine, specs, golden snapshots, integrated check, cross-spec validation.

| Item | Type | Action Required |
|------|------|-----------------|
| MF-3: Integrated check verification | Must Finish | Run `make codegen-integrated`. Verify all 7 families pass golden→target comparison. |
| MF-7: Cross-spec validation | Must Finish | Run `make codegen-validate-all`. Verify zero collisions across 7 specs. |
| EF-2: No template modification | Freeze | Templates frozen per S193. |
| EF-3: No spec schema extension | Freeze | 14-field schema frozen. |
| EF-4: No retroactive conversion | Freeze | 6 manual families are permanent golden references. |
| EF-5: No Tier 2 authorization | Freeze | Read-path generation not authorized. |
| EF-10: No batch generation | Freeze | One-at-a-time validation required. |

**Verification:** `make codegen-check` and `make codegen-integrated` both exit 0.

**Dependencies:** None. Codegen operates independently.

---

### Track 4: Repository Hygiene

**Scope:** Git state, ignored files, binary artifacts.

| Item | Type | Action Required |
|------|------|-----------------|
| MF-4: Remove `cmd/writer/writer` binary | Must Finish | Add `cmd/writer/writer` to `.gitignore`. Remove from staging. |

**Verification:** `git ls-files cmd/writer/writer` returns empty.

**Dependencies:** None.

---

### Track 5: Write Path (Writer / Pipeline)

**Scope:** Writer service, NATS consumers, ClickHouse inserters, pipeline supervision.

| Item | Type | Action Required |
|------|------|-----------------|
| EF-11: No pipeline structural changes | Freeze | Write path is proven immutable across 5 expansions. Do not modify. |
| EF-8: No new NATS streams | Freeze | Stream definitions frozen. |
| EF-9: No ClickHouse schema changes | Freeze | 7 migrations cover all 6 families. |

**Verification:** No diff in `cmd/writer/` core files (consumer.go, inserter.go, supervisor.go, pipeline.go structure).

**Dependencies:** None. Freeze only.

---

### Track 6: Infrastructure / Deployment

**Scope:** Docker compose, configs, migrations, services.

| Item | Type | Action Required |
|------|------|-----------------|
| EF-12: No new services | Freeze | 8 services cover current scope. |

**Verification:** No new directories under `cmd/`.

**Dependencies:** None. Freeze only.

---

### Track 7: Documentation

**Scope:** Architecture docs, stage reports, runbooks.

| Item | Type | Action Required |
|------|------|-----------------|
| EF-6: No massive cleanup/archival | Freeze | Documentation restructuring is next phase. |

**Verification:** No bulk deletion or reorganization of `docs/`.

**Note:** S205 deliverables (this document and siblings) are the only documentation produced during stabilization. They are triaging artifacts, not expansion artifacts.

---

## Cross-Track Dependencies

```
Track 2 (CI) depends on Track 1 (MF-1) for handler test stability
Track 2 (CI) depends on Track 3 (MF-3, MF-7) for codegen gates
Track 2 (CI) depends on Track 4 (MF-4) for clean git state

All other tracks are independent.
```

---

## Execution Order

Recommended execution sequence to minimize rework:

1. **MF-4** (hygiene) — Remove binary, update gitignore. No dependencies.
2. **MF-1** (handler extraction) — Self-contained, enables handler test stability.
3. **MF-5** (build verification) — Verify all modules compile.
4. **MF-6** (test verification) — Verify all tests pass (depends on MF-5).
5. **MF-7** (cross-spec validation) — Independent codegen verification.
6. **MF-3** (integrated check) — Independent codegen verification.
7. **MF-2** (CI verification) — Final gate: push branch, verify CI green.

---

## Freeze Enforcement

During the stabilization wave, any PR that touches freeze-scoped areas must be rejected unless it is:
1. A must-finish item from this matrix, OR
2. A bug fix for a regression discovered during stabilization verification.

No other changes are authorized.
