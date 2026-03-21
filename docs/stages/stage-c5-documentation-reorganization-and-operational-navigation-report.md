# Stage C5 - Documentation Reorganization And Operational Navigation Report

## Objective

Reorganize the operational and support documentation surfaces so the repository is
easier to execute, review, and maintain without rewriting the main architecture
corpus.

## Scope

C5 was limited to documentation structure, navigation, and conventions:

- add useful landing pages and indexes
- clarify the taxonomy between architecture, operations, tooling, stages, and archive
- reduce navigation entropy in the current docs tree
- update repository entrypoints to the new navigation model

Non-goals:

- broad architectural rewrites
- archive surgery
- renaming historical stage artifacts
- moving large clusters of architecture docs

## Diagnosis

The repository already had strong documentation depth but weak navigation:

- no `docs/` landing page
- no README-style index for `docs/operations/`, `docs/tooling/`,
  `docs/architecture/`, or `docs/archive/`
- operational runbooks remained discoverable only through the architecture corpus
- the stage index did not include the C-series support/documentation stages
- documentation governance still described an older docs tree

## Changes Made

### New indexes and entrypoints

Created:

- `docs/README.md`
- `docs/operations/README.md`
- `docs/tooling/README.md`
- `docs/architecture/README.md`
- `docs/archive/README.md`

### New canonical operational documents

Created:

- `docs/operations/documentation-reorganization-and-operational-navigation.md`
- `docs/operations/documentation-taxonomy-and-authoring-conventions.md`

### Updated repository entrypoints

Updated:

- `README.md`
- `DEVELOPMENT.md`
- `Makefile`

These now point to the new documentation entrypoints instead of isolated deep docs.

### Updated documentation-governance record

Updated:

- `docs/architecture/monorepo-documentation-and-stage-governance.md`

This change was surgical: it only aligned the canonical governance description with
the current documentation structure.

### Updated stage navigation

Updated:

- `docs/stages/INDEX.md`

This now:

- explains that the stage index is historical rather than the daily workflow entry
- includes C1-C5 support/documentation stages

## Decisions

### 1. Prefer re-indexing over mass moves

Large physical moves would create more churn than clarity because many existing
documents already reference the current paths. C5 therefore improved discoverability
through landing pages, cross-links, and placement rules.

### 2. Keep canonical runbooks in architecture when they are also architecture

Current runbooks such as `current-baseline-runbook.md` remain in
`docs/architecture/` because they are canonical architecture records. C5 surfaced
them through `docs/operations/README.md` instead of relocating them.

### 3. Separate user-facing tooling usage from tooling internals

`docs/operations/` is now the home for user-facing CLI usage and support-surface
navigation, while `docs/tooling/` remains the home for analyzer guardrails and drift
rules.

## Validation

Executed:

- `make check`

Result:

- passed

Additional validation:

- verified the current worktree before edits to avoid trampling unrelated in-flight
  work
- checked the docs directory inventory and counts to anchor the reorganization

## Outcome

C5 delivered a more sustainable documentation structure with minimal path churn:

- the docs tree now has real entrypoints by area
- operations and tooling docs are easier to locate
- canonical vs historical surfaces are clearer
- future documentation growth now has explicit placement and naming rules

## Follow-Up Recommendations

1. Keep the new area README files updated whenever new docs are added.
2. Use the taxonomy conventions before adding new support or workflow docs.
3. Consider a later targeted pass for filename harmonization in ambiguous tooling
   docs if that ambiguity becomes operationally costly.
