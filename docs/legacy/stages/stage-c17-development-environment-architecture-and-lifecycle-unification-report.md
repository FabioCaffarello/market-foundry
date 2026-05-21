# Stage C17 Report: Development Environment Architecture And Lifecycle Unification

## Summary

Stage C17 consolidates the repository developer environment into one explicit
architecture and lifecycle model, clarifying how setup, local bring-up,
validation, smoke, troubleshooting, and reset fit together.

## Objective

Define and apply a canonical developer-environment architecture for
`market-foundry` so the repository becomes more predictable to operate without
changing functional runtime behavior.

## Scope Boundaries

### In scope

- developer-environment architecture and lifecycle documentation
- canonical entrypoint hierarchy across Makefile, scripts, docs, and tooling
- lightweight entrypoint hardening to reflect the new canonical model
- stage evidence for the unification work

### Out of scope

- domain behavior or service contract changes
- runtime feature development
- new orchestration frameworks or alternate lifecycle systems
- turning `raccoon-cli` into a repository control plane

### Not changed

- bounded contexts and runtime ownership
- service topology and functional stack behavior
- existing smoke harness semantics

## Diagnosis

The repository already had a strong operational surface, but the lifecycle was
described in fragments:

- workflow guidance lived across `README.md`, `DEVELOPMENT.md`, Makefile help,
  operations docs, and harness-specific docs;
- `live*`, `up`/`seed*`, `smoke*`, and troubleshooting flows were individually
  documented, but the hierarchy between them was not made explicit in one
  canonical lifecycle model;
- cleanup/reset existed in practice, but it was not established as an official
  lifecycle phase;
- `raccoon-cli` was documented well, but its role inside the overall developer
  environment could still be misread as broader than intended.

## Changes Applied

- added [`../operations/development-environment-architecture-and-lifecycle.md`](../operations/development-environment-architecture-and-lifecycle.md) as the canonical architecture and lifecycle model;
- added [`../operations/development-lifecycle-entrypoints-and-canonical-flows.md`](../operations/development-lifecycle-entrypoints-and-canonical-flows.md) as the command-level lifecycle map;
- updated `make docs` to expose the new canonical environment docs;
- updated `scripts/bootstrap-check.sh` so bootstrap validates the presence of the new canonical lifecycle docs and points to them in its next-step guidance;
- updated `README.md`, `DEVELOPMENT.md`, [`../operations/README.md`](../operations/README.md), and [`../tooling/README.md`](../tooling/README.md) to anchor the new lifecycle model into the existing documentation surface;
- updated [`INDEX.md`](INDEX.md) to register C17 in stage history.

## Artifacts Added Or Updated

| Artifact | Purpose |
|---|---|
| `docs/operations/development-environment-architecture-and-lifecycle.md` | Canonical architecture and lifecycle model for the developer environment |
| `docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md` | Canonical operational entrypoints and phase-by-phase flows |
| `docs/stages/stage-c17-development-environment-architecture-and-lifecycle-unification-report.md` | Stage completion record |
| `Makefile` | `make docs` now exposes the canonical lifecycle docs |
| `scripts/bootstrap-check.sh` | Bootstrap now validates and points to the new lifecycle docs |
| `README.md` | Repository overview now links to the canonical developer-environment docs |
| `DEVELOPMENT.md` | Daily workflow doc now links to the canonical lifecycle docs |
| `docs/operations/README.md` | Operations index now exposes the lifecycle docs as first-class entrypoints |
| `docs/tooling/README.md` | Tooling guidance now explicitly defers lifecycle ownership to operations docs |
| `docs/stages/INDEX.md` | Stage history now includes C17 |

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C17 STAGE_SLUG=development-environment-architecture-and-lifecycle-unification STAGE_REQUIRE=docs/operations/development-environment-architecture-and-lifecycle.md,docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`

## Limits And Deferred Follow-Ups

- this stage consolidates lifecycle architecture and navigation, but it does not
  attempt to shrink or refactor the larger smoke harness implementations;
- the existing operations corpus still contains overlapping historical support
  docs, but the canonical hierarchy is now explicit;
- no new Make targets were introduced because the main gap was architectural
  clarity and lifecycle cohesion, not missing command coverage.

## Preparation For Next Stage

- use the new lifecycle model as the canonical filter for any future developer
  environment work;
- prefer tightening runtime reset ergonomics, proof discoverability, or support
  surface simplification only when a gap remains visible against the new
  lifecycle model;
- keep future tooling work subordinate to the repository-level environment
  architecture established here.
