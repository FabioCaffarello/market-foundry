# Stage C7 Repository Architecture Convergence Report

## Summary

Stage C7 executed a repository-support convergence pass focused strictly on the
support architecture of `market-foundry`.

The work stayed outside bounded contexts and business/domain evolution. It
targeted only the repository support layer:

- `Makefile`;
- scripts and harnesses;
- `raccoon-cli`;
- support/documentation surfaces;
- support-surface ownership and entrypoint rules.

## Baseline

The repository already had substantial support improvements from C1 through C6:

- a grouped `Makefile` help surface;
- normalized scripts and wrappers;
- a grouped `raccoon-cli` taxonomy;
- a dedicated `docs/operations/` area;
- a lightweight repository consistency guard rail.

What remained weak was convergence across those pieces. The repository had good
parts, but the system-level reading of those parts was still spread across
multiple documents and some tooling output still contradicted the current
surface.

## Findings

### 1. Canonical versus auxiliary surfaces were still implicit

`Makefile`, direct scripts, direct `raccoon-cli` usage, and raw substrate
commands all existed, but the repository did not provide one canonical
support-architecture model explaining precedence and ownership.

### 2. The docs split existed, but the boundary still needed explicit policy

`docs/operations/` and `docs/tooling/` existed, but the repository still needed
a clear statement that:

- `docs/operations/` owns support-surface usage and workflow coexistence;
- `docs/tooling/` owns tooling-internal behavior and analyzer policy.

### 3. Tooling still emitted stale operational guidance

`raccoon-cli` still suggested `make up-dataplane` in multiple places, even
though that entrypoint does not exist in the current repository shape.

### 4. Some runtime-proof guidance still implied parallel CLI routes

The coverage map still pointed to a nonexistent CLI smoke command instead of the
canonical Makefile-backed smoke surface.

## Changes Applied

### New canonical documents

- Added `docs/operations/repository-support-surface-canonical-model.md`
- Added `docs/operations/repository-architecture-convergence.md`
- Added this stage report

### Existing-document convergence updates

- Updated `README.md`
- Updated `DEVELOPMENT.md`
- Updated `docs/operations/README.md`
- Updated `docs/operations/makefile-targets-reference-and-conventions.md`
- Updated `docs/operations/scripts-catalog-and-usage-guide.md`
- Updated `docs/operations/documentation-taxonomy-and-authoring-conventions.md`
- Updated `docs/tooling/README.md`
- Updated `docs/tooling/cli-overview.md`
- Updated `docs/tooling/cli-architecture-guardrails.md`
- Updated `docs/stages/INDEX.md`

### Tooling updates

- Updated `make docs` so the new C7 canonical docs are part of the primary support-document set.
- Extended `scripts/repository-consistency-check.sh` so the C7 documents are required repository entrypoints.
- Removed stale `make up-dataplane` guidance from active `raccoon-cli` output paths and tests.
- Realigned coverage-map runtime-proof guidance to `make smoke`.
- Hardened `Makefile` module-aware Go execution so `make verify` and related targets no longer mask failures from earlier modules.

## Validation

Validation executed during C7:

- `make check` before changes
- `make raccoon-test`
- `make check`
- `make verify`

Observed validation outcome:

- support-surface validation passed (`make raccoon-test`, `make check`);
- final repository validation passed (`make verify`);
- during validation, C7 also hardened the module-aware Makefile Go loops so canonical targets fail reliably instead of depending on the last module's exit status.

## Outcome

After C7, the repository support architecture has a clearer reading:

- `make` is the public workflow contract;
- scripts are harness implementations;
- direct `raccoon-cli` usage is the expert tooling surface;
- raw compose/go/cargo stay below the repository workflow contract;
- `docs/operations/` now explicitly owns the coexistence model for support surfaces.

This improves coherence without touching the functional system architecture.

## Recommended Preparation For C8

- Audit the remaining historical CLI residue around scenario-oriented hints and legacy command text, but keep the scope bounded to support-surface correctness.
- Decide whether any additional `raccoon-cli` expert outputs should render canonical Make targets instead of historical internal command names.
- If script growth becomes the next pain point, charter a narrow harness-internal cleanup stage rather than widening C7 retroactively.
