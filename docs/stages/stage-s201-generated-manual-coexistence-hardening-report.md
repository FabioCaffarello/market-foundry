# Stage S201: Generated/Manual Coexistence Hardening — Report

**Date:** 2026-03-20
**Predecessor:** S200 (First Generated Slice Integration)
**Objective:** Harden the coexistence between manual and generated artifacts, making ownership, drift detection, and regeneration operationally safe before the first codegen-first family.

---

## 1. Executive Summary

S201 closes the ambiguity gap between manual and generated artifacts. Before this stage, the project had a working first governed slice (RSI A1+A2) but lacked cross-spec validation, had a hardcoded integrated check script, and had no developer-facing governance visibility. After S201:

- Cross-spec uniqueness is enforced in CI (no duplicate durables, subjects, or family names)
- The integrated check script is manifest-driven (adding a slice requires only a YAML entry, not script edits)
- Marker integrity is validated (begin/end presence, non-empty regions)
- Governance status is visible via `make codegen-status`
- Ownership zones, editing rules, and review checklists are documented
- The CI pipeline validates specs before attempting golden comparison (fail-fast ordering)

The base is now ready for a codegen-first family decision without risk of ownership ambiguity.

## 2. Coexistence Hardened

### Before S201

| Dimension | State |
|-----------|-------|
| Cross-spec uniqueness | Not checked; collision possible |
| Integrated check | Hardcoded RSI slices in script |
| Marker validation | Only content comparison; missing markers → confusing errors |
| Governance visibility | No developer-facing tool |
| CI ordering | Golden check before spec validation |
| Ownership documentation | Spread across multiple S193–S200 docs |
| Editing rules | Implicit in architecture decisions, not consolidated |
| Review checklist | None formalized |

### After S201

| Dimension | State |
|-----------|-------|
| Cross-spec uniqueness | `codegen validate-all` enforces in CI |
| Integrated check | Manifest-driven from `codegen/integrated.yaml` |
| Marker validation | Begin/end presence + non-empty region + content match |
| Governance visibility | `make codegen-status` shows GOVERNED/MANUAL per family |
| CI ordering | validate-all → check-all → test → integrated (fail-fast) |
| Ownership documentation | Consolidated in `generated-manual-coexistence-hardening.md` |
| Editing rules | Consolidated in `generated-artifact-editing-regeneration-and-review-rules.md` |
| Review checklist | Formalized in editing/review rules doc |

## 3. Files Changed

### Codegen Engine

| File | Change |
|------|--------|
| `codegen/spec.go` | Added `LoadAllSpecs()`, `ValidateCrossSpec()` — cross-spec uniqueness validation |
| `codegen/main.go` | Added `validate-all` command; added `cmdValidateAll()` function |
| `codegen/spec_test.go` | Added 5 tests: `TestValidateCrossSpec_NoDuplicates`, `_DuplicateDurable`, `_DuplicateSubject`, `_DuplicateName`, `TestLoadAllSpecs` |

### Scripts

| File | Change |
|------|--------|
| `scripts/codegen-integrated-check.sh` | Rewritten: manifest-driven parsing from `codegen/integrated.yaml`; added marker presence validation; macOS awk compatible |

### Build/CI

| File | Change |
|------|--------|
| `Makefile` | Added targets: `codegen-validate-all`, `codegen-status` |
| `.github/workflows/ci.yml` | Added `validate-all` step before golden comparison in `codegen-golden` job |

### Architecture Documentation

| File | Content |
|------|---------|
| `docs/architecture/generated-manual-coexistence-hardening.md` | Three-zone ownership model, marker format standard, manifest, manual family rationale, cross-spec rules |
| `docs/architecture/generated-artifact-editing-regeneration-and-review-rules.md` | Editing rules per zone, regeneration procedures, PR review checklist, revocation policy, contributor quick reference |
| `docs/architecture/generated-path-drift-detection-and-ci-hardening.md` | Three drift types, CI pipeline chain, structural comparison rules, manifest-driven check, marker integrity, known gaps |

## 4. Rules and Checks

### CI Validation Chain (Post-S201)

```
1. codegen validate-all      → per-spec + cross-spec uniqueness     [NEW]
2. codegen check-all         → spec→golden structural equivalence
3. codegen test              → engine unit tests
4. codegen integrated-check  → golden→target match (manifest-driven) [IMPROVED]
5. unit-tests (Go)           → compilation + logic tests
6. smoke-analytical          → runtime E2E proof
```

### New Invariants Enforced

| Invariant | Check | Gate |
|-----------|-------|------|
| No duplicate `family.name` across specs | `validate-all` | CI, blocks merge |
| No duplicate `nats.durable` across specs | `validate-all` | CI, blocks merge |
| No duplicate `nats.subject` across specs | `validate-all` | CI, blocks merge |
| All 14 spec fields present per family | `validate-all` | CI, blocks merge |
| Begin marker present in target file | `integrated-check` | CI, blocks merge |
| End marker present with matching family | `integrated-check` | CI, blocks merge |
| Non-empty region between markers | `integrated-check` | CI, blocks merge |

### Make Targets Available

```
make codegen-validate-all   # cross-spec validation (per-spec + uniqueness)
make codegen-check          # spec→golden equivalence (all families × all artifacts)
make codegen-test           # engine unit tests
make codegen-integrated     # golden→target match (manifest-driven)
make codegen-status         # governance status (GOVERNED vs MANUAL per family)
```

## 5. Remaining Limits

### Intentionally Deferred

| Item | Why Deferred | When It Should Close |
|------|-------------|---------------------|
| Automated file patching | Complexity outweighs benefit at current scale (1 governed family) | When ≥3 governed families make manual copy-paste a friction bottleneck |
| Mapper generation (A3) | Requires `domain.columns` spec extension not yet designed | After first codegen-first family validates A1+A2 pattern at scale |
| Template version tracking | Templates are frozen; no version to track | If templates are ever unfrozen (requires authorization stage) |
| Batch generation | One-family-at-a-time is safer during expansion | After pattern is proven with ≥3 codegen-first families |
| Pre-commit hooks for codegen | CI gate is sufficient; pre-commit adds local dev friction | Only if CI detection latency becomes a problem |

### Known Gaps (Acknowledged, Not Blocked)

| Gap | Risk Level | Mitigation |
|-----|-----------|------------|
| Derivation logic change invalidates all goldens | Low (logic is simple, tested) | `check-all` catches it; all 12 goldens would fail |
| Spec field constraints not enforced (e.g., mapper must be known function) | Low | Go compilation catches invalid references |
| Manual families may structurally diverge from goldens over time | None (by design) | Manual families are baseline, never governed |

## 6. Preparation for S202

S201 leaves the project ready for the first codegen-first family. The recommended S202 scope:

1. **Select the first codegen-first family** — a new family that will be authored spec-first, never manually coded
2. **Author the YAML spec** — using the frozen 14-field schema
3. **Generate goldens** and **place markers** in target files
4. **Author manual artifacts** (mapper, tests, config, smoke)
5. **Add manifest entry** and verify full CI chain
6. **Validate the pattern** — does the governed process work end-to-end for a family that was never manual?

### Decision criteria for S202 family selection:
- Should be in a layer that already has a manual family (to avoid new-layer complexity)
- Should be simple enough that A1+A2 generation covers the mechanical parts
- Should exercise the cross-spec validation (second family in same layer is ideal)

### What S202 should NOT do:
- Do not generate A3 (mappers) — not yet authorized
- Do not modify templates — still frozen
- Do not batch-generate multiple families — one at a time
- Do not retroactively govern existing manual families

---

**Acceptance criteria met:**
- [x] Ownership zones are explicit and documented
- [x] Drift detection covers all three drift types
- [x] CI pipeline validates specs, goldens, and governed targets
- [x] Integrated check is manifest-driven (no script edits for new slices)
- [x] Cross-spec uniqueness is enforced
- [x] Governance status is visible to developers
- [x] Review checklist formalized
- [x] Base is ready for first codegen-first family decision
- [x] No ambiguous ownership remains
