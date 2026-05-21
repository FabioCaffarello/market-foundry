# Stage C21 Report: Repository Maintainability Economics And Structural Cost Control

## Summary

Stage C21 analyzed the repository support surface through the lens of
maintenance economics and reduced accidental structural cost in the highest-ROI
areas: root-doc duplication, inflated documentation entrypoints, and
over-accumulated lightweight guard rails.

## Objective

Make the repository support surface cheaper to evolve by reducing avoidable
edit fan-out and documenting explicit structural-cost control rules.

## Scope Boundaries

### In scope

- support-surface entrypoints in root docs and operations indexes
- lightweight repository guard rails
- structural-cost documentation for docs, scripts, CLI, checks, and stage
  support surfaces

### Out of scope

- bounded-context refactors
- runtime functional behavior
- broad CLI or harness redesign

### Not changed

- service runtime topology
- architecture-layer contracts
- governed stage model and stage-history inventory

## Hotspots Found

- Root entrypoints were carrying too much duplicated support-document detail.
- `make docs` had grown into a long secondary documentation index instead of a
  quick entrypoint surface.
- `scripts/repository-consistency-check.sh` still treated several historical
  support-stage reports as required artifacts, causing linear upkeep growth.
- Ownership boundaries between root docs and the operations index were not
  explicit enough to prevent future duplication drift.

## Changes Applied

- Curated `make docs` down to primary workflow, navigation, tooling, and C21
  structural-cost entrypoints.
- Reduced support-document duplication in `README.md` and `DEVELOPMENT.md`,
  keeping those files focused on orientation and daily workflow.
- Updated `docs/operations/README.md` to remain the canonical detailed catalog
  and to carry the explicit structural-cost rule for support-doc indexing.
- Narrowed the lightweight required-document guard rail so it protects current
  canonical support surfaces instead of accumulating historical stage reports as
  permanent required files.
- Added explicit C21 operations docs covering maintainability economics,
  hotspots, and cost-reduction principles.

## Artifacts Added Or Updated

| Artifact | Purpose |
|---|---|
| `docs/operations/repository-maintainability-economics-and-structural-cost-control.md` | structural-cost model and control rules |
| `docs/operations/repository-maintenance-hotspots-and-cost-reduction-principles.md` | hotspot inventory and reduction principles |
| `docs/stages/stage-c21-repository-maintainability-economics-and-structural-cost-control-report.md` | stage completion record |
| `Makefile` | curated `make docs` surface |
| `README.md` | root entrypoint simplification |
| `DEVELOPMENT.md` | workflow doc simplification |
| `docs/operations/README.md` | canonical detailed support-doc index |
| `scripts/repository-consistency-check.sh` | reduced accidental guard-rail growth |

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C21 STAGE_SLUG=repository-maintainability-economics-and-structural-cost-control STAGE_REQUIRE=docs/operations/repository-maintainability-economics-and-structural-cost-control.md,docs/operations/repository-maintenance-hotspots-and-cost-reduction-principles.md`

## Limits And Deferred Follow-Ups

- The operations index remains intentionally detailed; C21 reduced duplicate
  catalogs elsewhere rather than flattening the canonical index itself.
- The lightweight guard rail still contains a hand-maintained canonical-doc set;
  that is acceptable while the set stays small and invariant-bearing.
- Large smoke scripts and some support docs remain costly to evolve, but they
  were not the safest high-ROI change for this stage.

## Preparation For Next Stage

- Use the new structural-cost decision test before adding support-stage docs or
  new helper commands in C22.
- Prefer improvements that reduce support-surface drift around active runtime
  waves rather than adding more governance-only layers.
