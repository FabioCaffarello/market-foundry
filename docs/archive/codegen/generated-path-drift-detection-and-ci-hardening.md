# Generated Path Drift Detection and CI Hardening

> S201 — How drift is detected, what the CI pipeline enforces, and what gaps remain.

## Drift Types

There are three distinct drift types in the codegen pipeline:

### Type 1: Spec → Golden Drift

**Definition:** The golden snapshot no longer matches what the engine generates from the current spec.

**Cause:** Spec edited without regenerating goldens, or engine logic changed.

**Detection:** `make codegen-check` (runs `codegen check-all`).

**CI gate:** `codegen-golden` job, step "Run codegen golden comparison."

**Resolution:** Regenerate goldens from spec.

### Type 2: Golden → Target Drift

**Definition:** The governed region in a target file no longer matches its golden snapshot.

**Cause:** Manual edit inside markers, or golden regenerated without updating target.

**Detection:** `make codegen-integrated` (runs `codegen-integrated-check.sh`).

**CI gate:** `codegen-golden` job, step "Verify integrated slices match golden snapshots."

**Resolution:** Copy golden content into target file between markers.

### Type 3: Cross-Spec Collision

**Definition:** Two or more specs share a value that must be unique (durable consumer name, NATS subject, family name).

**Cause:** Copy-paste error when creating new spec, or naming convention collision.

**Detection:** `make codegen-validate-all` (runs `codegen validate-all`).

**CI gate:** `codegen-golden` job, step "Validate all specs."

**Resolution:** Fix the conflicting spec.

## CI Pipeline: Codegen Validation Chain

The `codegen-golden` CI job runs these steps in order:

```
Step 1: codegen validate-all     → cross-spec uniqueness + per-spec validation
Step 2: codegen check-all        → spec→golden structural equivalence (12 checks)
Step 3: codegen test             → engine unit tests (derivation, comparison, rendering)
Step 4: codegen-integrated-check → golden→target match for governed slices (manifest-driven)
```

All four must pass for the job to succeed. The job blocks merge on failure.

### Why This Order Matters

1. **validate-all first:** Catches spec errors before attempting generation. Prevents cryptic failures downstream.
2. **check-all second:** Ensures goldens are in sync with specs. If this fails, integrated check would also fail.
3. **test third:** Engine logic tests. If check-all passes but tests fail, it signals a test gap.
4. **integrated last:** This is the final proof that governed code in target files matches the source of truth.

## Structural Comparison Rules

Both `codegen/compare.go` and `scripts/codegen-integrated-check.sh` apply identical normalization:

1. Strip single-line comments (`// ...`)
2. Normalize tabs to single space
3. Trim leading/trailing whitespace per line
4. Remove empty lines

This means:
- **Caught:** Added/removed fields, changed function names, different SQL, reordered struct fields
- **Not caught:** Comment-only changes, whitespace-only changes, import reordering

This is intentional — structural equivalence is what matters for correctness. Cosmetic differences are tolerated.

## Manifest-Driven Check (S201 Improvement)

Before S201, `codegen-integrated-check.sh` had hardcoded slice entries. Now it reads from `codegen/integrated.yaml` automatically:

```
┌──────────────────────┐
│ codegen/integrated.yaml │  ← single manifest, human-maintained
└──────────┬───────────┘
           │ parsed by
           ▼
┌──────────────────────────────────┐
│ scripts/codegen-integrated-check.sh │  ← no edits needed when adding slices
└──────────┬───────────────────────┘
           │ for each slice:
           ├── verify markers exist in target
           ├── extract region between markers
           ├── normalize both golden + extracted
           └── compare → PASS/FAIL
```

Adding a new governed slice now requires only:
1. Add entry to `codegen/integrated.yaml`
2. Place markers in target file
3. CI picks it up automatically

## Marker Integrity Checks (S201 Improvement)

The integrated check script now validates:

1. **Begin marker exists** in target file — catches accidentally deleted markers
2. **End marker exists** with matching artifact and family — catches orphaned begins
3. **Region is non-empty** between markers — catches empty governed regions
4. **Content matches golden** — the core drift check

Previously only step 4 was checked. Steps 1–3 catch marker-level problems that would otherwise produce confusing failures.

## Cross-Spec Validation (S201 Improvement)

The new `codegen validate-all` command enforces invariants that individual spec validation cannot catch:

| Invariant | What It Prevents |
|-----------|-----------------|
| Unique `family.name` | Two specs claiming the same family |
| Unique `nats.durable` | Two consumers competing for the same durable name at runtime |
| Unique `nats.subject` | Two consumers processing the same subject (would cause duplicate writes) |

This runs in CI before golden comparison, catching problems at the earliest possible point.

## What CI Does NOT Catch (Known Gaps)

| Gap | Risk | Mitigation | When It Closes |
|-----|------|------------|----------------|
| Mapper function existence | Generated A2 references a mapper by name; if mapper doesn't exist, build fails | Go compilation gate in `unit-tests` job | Compilation is sufficient |
| Config entry presence | Missing config entry means family is disabled at runtime | Smoke test checks runtime activation | `smoke-analytical` job |
| Manual families' structural match to goldens | Manual families may diverge from what codegen would produce | Manual families are the baseline, not governed | By design — never closes |
| Template version tracking | Template change without full regeneration | Template is frozen; any change requires authorization stage | Future stage if template changes are authorized |

## Developer Workflow

```bash
# After any codegen-related change:
make codegen-validate-all    # cross-spec uniqueness
make codegen-check           # spec→golden equivalence
make codegen-test            # engine tests
make codegen-integrated      # golden→target match

# Quick governance overview:
make codegen-status          # shows GOVERNED vs MANUAL per family
```

## Validation Chain Summary

```
                    ┌──────────────────────┐
                    │   YAML Spec (human)  │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 1  │  validate-all        │  cross-spec uniqueness
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 2  │  check-all           │  spec → golden match
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 3  │  codegen test        │  engine unit tests
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 4  │  integrated-check    │  golden → target match
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 5  │  unit-tests (Go)     │  compilation + logic
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
         CI Step 6  │  smoke-analytical    │  runtime participation
                    └──────────────────────┘
```

Every layer catches a different class of problem. No single check is sufficient alone.
