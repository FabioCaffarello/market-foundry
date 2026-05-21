# Repository Architecture Convergence

## Purpose

This document records the C7 convergence pass on the repository support
architecture for `market-foundry`.

The goal is narrow and structural:

- make the support surfaces read as one system;
- reduce parallel entrypoint ambiguity;
- clarify which surfaces are canonical versus auxiliary;
- keep the change proportional and outside domain/business evolution.

## Diagnosis

Before C7, the repository already had many good support pieces, but the
coexistence model was still implicit.

The main convergence problems were:

- `Makefile`, scripts, direct `raccoon-cli`, and raw substrate commands were all documented, but their precedence was not stated in one canonical place;
- `docs/operations/` described user-facing support workflows while `docs/tooling/` described `raccoon-cli`, yet the boundary between wrapped CLI usage and direct expert CLI usage remained diffuse;
- some tooling output still suggested obsolete or nonexistent operational entrypoints such as `make up-dataplane`;
- coverage and guard-rail messaging still implied parallel CLI runtime flows even though the repository had standardized around Makefile-backed runtime proofs.

## Convergence Decisions

### 1. `make` is the public support API

Repository workflows that contributors and operators are expected to use
regularly must front through the root `Makefile`.

That includes:

- validation;
- stack lifecycle;
- smoke and live proofs;
- codegen checks;
- migrations.

### 2. Scripts are harnesses, not competing front doors

Shell scripts remain important, but their architectural role is now explicit:
they implement or deepen public workflows rather than competing with them.

### 3. `raccoon-cli` is an expert tooling surface

Direct CLI usage stays canonical inside the tooling/governance lane:

- structural inspection;
- change analysis;
- machine-readable output;
- tooling development.

It does not become the default operator interface for runtime proof when a Make
workflow already exists.

### 4. Raw substrate interfaces stay below the support contract

Direct `docker compose`, `go`, and `cargo` commands remain valid implementation
tools, but they are not the first-choice repository entrypoints.

## Applied Changes In C7

### Documentation

- Added [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md) as the canonical support-surface model.
- Added this convergence record to state the architectural decisions behind the model.
- Linked the new support model from `README.md`, `DEVELOPMENT.md`, `docs/operations/README.md`, and the `make docs` surface.
- Updated support docs so `docs/operations/` clearly owns support-surface usage and `docs/tooling/` clearly owns tooling-internal behavior.

### Tooling

- Fixed `raccoon-cli` messaging that still suggested nonexistent `make up-dataplane` commands.
- Realigned runtime-proof guidance to the Makefile-backed surface: `make up`, `make live`, and `make smoke*`.
- Fixed the coverage map so its runtime proof command points to `make smoke` instead of a nonexistent CLI command.
- Hardened module-aware Makefile Go loops so canonical targets such as `make verify` stop masking earlier module failures.
- Updated the `raccoon-cli` README and tooling docs to reflect the grouped taxonomy and the support-surface boundary.

### Guard Rails

- Extended `scripts/repository-consistency-check.sh` so the new C7 canonical docs and stage report become part of the required support-document set.

## Target Support Architecture

| Layer | Canonical surface | Why |
|---|---|---|
| orientation | `README.md`, `docs/README.md` | fast entry and navigation |
| daily workflow | `DEVELOPMENT.md`, `Makefile` | one obvious public workflow contract |
| operational support docs | `docs/operations/` | canonical usage rules for support surfaces |
| tooling internals | `docs/tooling/`, direct `raccoon-cli` | expert analysis and enforcement details |
| harness implementation | `scripts/*.sh` | lower-level execution and debugging |
| runtime substrate | `deploy/*`, raw compose/go/cargo | underlying implementation layer |
| architecture governance | `docs/architecture/` | binding system rules outside support usage |
| historical evidence | `docs/stages/` | immutable delivery trail |

## Entry Point Policy

When a contributor asks "where should I start?", the answer should now be
deterministic:

1. if the workflow exists in `make`, start there;
2. if the task is expert inspection or tooling development, use direct `raccoon-cli`;
3. if the task is harness debugging, drop to `scripts/*.sh`;
4. if the task is substrate debugging or implementation work, use raw compose/go/cargo.

## Residual Limits

C7 intentionally did not:

- redesign the runtime harnesses;
- refactor large smoke scripts;
- expand or change domain/runtime behavior;
- turn every old tooling hint into a new public API.

Some deeper CLI historical residue still exists and should be handled only when
there is a bounded charter to keep the change surgical.

## Related Documents

- [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
- [`../stages/stage-c7-repository-architecture-convergence-report.md`](../stages/stage-c7-repository-architecture-convergence-report.md)
