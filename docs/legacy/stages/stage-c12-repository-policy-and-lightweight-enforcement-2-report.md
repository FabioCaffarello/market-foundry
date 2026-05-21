# Stage C12 Repository Policy And Lightweight Enforcement 2 Report

## Summary

Stage C12 implemented a second-generation lightweight repository-policy layer
for the cleaned `market-foundry` support surface.

The work did not introduce a new policy framework. Instead, it extended the
existing repository consistency guard rail so the repository now protects a
small set of cheap, high-value invariants around:

- documentation entrypoints;
- docs/index presence;
- bootstrap alignment;
- Makefile/script/catalog consistency;
- public script hygiene;
- CLI governance-surface stability.

## Delivered Changes

- Extended `scripts/repository-consistency-check.sh` with new checks for:
  - docs-area entrypoints;
  - public script self-description (`#!/usr/bin/env bash` + `--help`);
  - bootstrap governed-entrypoint alignment;
  - Makefile-to-scripts-catalog alignment;
  - CLI governance-surface alignment across source and canonical docs.
- Updated `scripts/bootstrap-check.sh` so the bootstrap path now expects the new
  C12 governance docs and the stage index entrypoint.
- Updated `Makefile` `docs` output so the new governance docs are part of the
  published navigation surface.
- Added the new C12 canonical operations docs:
  - `docs/operations/repository-policy-and-lightweight-enforcement-2.md`
  - `docs/operations/repository-invariants-check-matrix-and-enforcement-policy.md`
- Updated documentation indexes so the new policy surface is discoverable from
  the active operations/doc entrypoints.

## Invariants Selected

The chosen invariants were limited to rules with good cost-benefit:

1. missing canonical docs and index entrypoints;
2. drift between bootstrap expectations and the active support surface;
3. undocumented Makefile-backed script wrappers;
4. public script entrypoints that stop being self-describing;
5. erosion of the `raccoon-cli` support-only grouped taxonomy.

These were selected because they are cheap to evaluate, objective, and hard to
police reliably in review alone.

## What Stayed Out Of Enforcement

C12 intentionally did not enforce:

- prose or editorial style;
- full-corpus broken-link validation;
- deep shell style rules;
- subjective completeness of documents;
- runtime or domain architecture concerns already owned by `raccoon-cli`,
  tests, and smoke harnesses.

## Flow Integration

The new enforcement remains integrated through the existing repository flow:

- `make repo-consistency-check`
- `make check`
- `make verify`
- `make check-deep`
- `make bootstrap`
- `make docs`

This keeps governance visible in the default workflow without creating a second
parallel validation path.

## Validation

Validation for C12 is:

- `make repo-consistency-check`

The expected result is a passing lightweight policy scan over the governed
support surface.

## Outcome

C12 closes the cleanup wave with practical repository governance.

The repository now has lightweight automated protection against support-surface
drift, while keeping the enforcement small enough to stay credible and
sustainable in day-to-day engineering work.
