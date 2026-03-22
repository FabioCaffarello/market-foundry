# Stage C22 Report: Tooling Evolution Patterns And Repository Extension Discipline

## Summary

Stage C22 defined how support tooling in `market-foundry` should evolve after
the repository already achieved a sufficient baseline of scripts, Makefile
entrypoints, operational docs, and lightweight automation.

The stage focused on disciplined extension rather than new capability volume:
which surface should absorb future needs, when support additions are justified,
and how to prefer consolidation over opportunistic growth.

## Objective

Establish repository-specific rules for tooling evolution so future support
changes strengthen coherence instead of increasing maintenance drift.

## Scope Boundaries

### In scope

- recent evolution of `Makefile`, `scripts/`, `docs/operations/`,
  `docs/tooling/`, and lightweight support automation
- criteria for adding commands, scripts, docs, checks, or no new surface
- inclusion, consolidation, deprecation, and retirement rules for support
  tooling
- light hardening of canonical docs and consistency protection for the new model

### Out of scope

- runtime/domain architecture changes
- large harness refactors
- new workflow engines or metadata systems
- heavy process bureaucracy for small support edits

### Not changed

- the public workflow centered on `make`
- the current `raccoon-cli` grouped taxonomy
- the role of `scripts/*.sh` as harness implementations behind `make`
- the stage model and historical evidence structure

## Growth Pattern Reviewed

The recent support-tooling history showed a healthy but now risk-bearing
sequence:

1. root `Makefile` expansion to expose canonical workflows
2. harness growth in `scripts/` for smoke, bring-up, diagnostics, and stage
   support
3. operations/tooling documentation convergence
4. lightweight repository consistency checks to keep the new surfaces aligned

This growth solved real repository problems, but it also raised the risk of
surface inflation, duplicate entrypoints, monolithic harnesses, and support-doc
fragmentation if extension remains purely additive.

## Findings

### Good patterns

- `make` is already the canonical public workflow contract.
- `scripts/` mostly acts as implementation detail behind that contract.
- `raccoon-cli` remains in the structural-analysis and governance lane.
- `docs/operations/README.md` is the detailed canonical support index.
- `make repo-consistency-check` protects objective support-surface invariants
  without becoming a general policy engine.

### Weak-discipline areas

- additive helper growth can now happen faster than public-surface review
- some large smoke harnesses are tempting places to fork instead of consolidate
- the repository had cost-control guidance and automation-boundary guidance, but
  lacked one explicit extension model covering Make targets, scripts, docs,
  checks, and "do nothing"
- the canonical docs did not yet expose one concrete decision model for
  inclusion, deprecation, and consolidation

## Changes Applied

- Added `docs/operations/tooling-evolution-patterns-and-repository-extension-discipline.md`
  as the canonical model for future support-surface growth.
- Added `docs/operations/tooling-inclusion-deprecation-and-consolidation-rules.md`
  as the concrete operating rules and review filter for future additions.
- Updated `docs/operations/README.md` so both new documents are part of the
  canonical support index.
- Updated `docs/tooling/README.md` so CLI/tooling work now explicitly routes
  through the C22 extension-discipline model before adding new tooling
  surfaces.
- Updated `scripts/README.md` to reinforce that new scripts should remain
  harness-level and should be promoted into `make` only when they become routine
  public workflows.
- Updated `Makefile` `make docs` output to surface the new primary C22
  extension-discipline doc.
- Extended `scripts/repository-consistency-check.sh` so the new canonical C22
  docs are protected as required support documents.
- Indexed this stage in `docs/stages/INDEX.md`.

## Criteria Defined

Stage C22 established explicit rules for deciding when a new need should become:

- a `make` target
- a lower-level script
- a `raccoon-cli` command
- documentation only
- a lightweight check
- or no new surface at all

It also established repository-specific criteria for:

- inclusion
- consolidation
- deprecation
- retirement
- naming and canonicality
- ownership and lifecycle

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C22 STAGE_SLUG=tooling-evolution-patterns-and-repository-extension-discipline STAGE_REQUIRE=docs/operations/tooling-evolution-patterns-and-repository-extension-discipline.md,docs/operations/tooling-inclusion-deprecation-and-consolidation-rules.md`

## Limits And Deferred Follow-Ups

- C22 did not refactor the large smoke scripts; it only documented how future
  work should approach them.
- C22 did not add new workflow engines, metadata registries, or approval
  mechanics.
- C22 keeps the lightweight consistency pass intentionally narrow; future
  invariants should still be admitted sparingly.

## Preparation For C23

- Use the C22 inclusion rules before adding any new support entrypoint.
- Prefer consolidation work where support growth now manifests as sibling
  scripts, duplicate docs, or overlapping command surfaces.
- If C23 targets a support surface, start by identifying what can be retired or
  folded into an existing family before proposing anything new.
