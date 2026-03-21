# Documentation System Hardening

## Purpose

This document is the canonical map for the `market-foundry` documentation
system after Stage C11.

It exists to make the repository documentation navigable as a system rather
than a large set of individually valid files.

## Executive Summary

`market-foundry` already had strong documentation depth, especially in
`docs/architecture/` and `docs/stages/`. The main weakness was not missing
content, but system behavior:

- multiple entrypoints overlapped in role;
- taxonomy rules existed, but not as one clear canonical governance surface;
- operations, tooling, architecture, stage evidence, and archive boundaries were
  documented, but not linked together as one map;
- older documentation-governance docs were still useful, but they competed with
  newer indexes for authority.

Stage C11 hardens the system by clarifying canonical entrypoints, strengthening
taxonomy, reducing governance duplication, and wiring the new model into the
repository guard rails.

## Current Documentary Shape

| Surface | Markdown volume | Canonical role |
|---|---:|---|
| `README.md` and `DEVELOPMENT.md` | 2 files | Repository-level orientation and daily workflow |
| `docs/operations/` | 19 files | Operational support, documentation governance, entrypoints, and user-facing support surfaces |
| `docs/tooling/` | 18 files | Tool-internal references and analyzer rules |
| `docs/architecture/` | 439 files | Canonical architecture and governance corpus |
| `docs/stages/` | 313 files | Immutable delivery and evolution evidence |
| `docs/archive/` | 246 files in nested archive sets | Historical and superseded material |

The volume concentration in architecture and stages is not a problem by itself.
The system only becomes entropic when contributors lack a small set of stable
entrypoints and clear rules about where lasting guidance belongs.

## Canonical Documentation Entry Chain

Use this chain in order when locating or updating documentation:

1. `README.md` for repository identity and quick orientation.
2. `DEVELOPMENT.md` for the daily workflow and validation loop.
3. [`documentation-system-hardening.md`](documentation-system-hardening.md) for the documentation-system map.
4. [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md) for placement, naming, and maintenance rules.
5. Area indexes:
   - [`README.md`](README.md) in `docs/operations/`
   - [`../tooling/README.md`](../tooling/README.md)
   - [`../architecture/README.md`](../architecture/README.md)
   - [`../stages/INDEX.md`](../stages/INDEX.md)
   - [`../archive/README.md`](../archive/README.md)

## Cross-Surface Relationship Map

| If the question is... | Canonical home | Supporting surface | Historical surface |
|---|---|---|---|
| How do I work in this repository day to day? | `DEVELOPMENT.md` and `docs/operations/README.md` | `docs/operations/*` | `docs/stages/*` only for why it changed |
| How is the documentation system organized? | `docs/operations/documentation-system-hardening.md` | `docs/README.md` | C5/C11 stage reports |
| Where should a new doc live? | `docs/operations/documentation-governance-entrypoints-and-taxonomy.md` | `docs/architecture/monorepo-documentation-and-stage-governance.md` | Older taxonomy docs |
| What does the tooling enforce? | `docs/tooling/README.md` | `docs/operations/raccoon-cli-command-reference.md` | Stage reports about tooling changes |
| What is architecturally canonical? | `docs/architecture/README.md` | `docs/operations/README.md` for runbook links | `docs/archive/` and `docs/stages/` |
| What changed in a bounded stage? | `docs/stages/INDEX.md` | relevant canonical docs | none |
| What was superseded? | `docs/archive/README.md` | relevant active canonical doc | archive subtree |

## Entropy Risks Addressed In C11

### 1. Competing documentation-governance surfaces

Before C11, governance for documentation was spread across:

- `docs/README.md`
- `docs/operations/README.md`
- `docs/operations/documentation-taxonomy-and-authoring-conventions.md`
- `docs/architecture/monorepo-documentation-and-stage-governance.md`
- `docs/operations/documentation-reorganization-and-operational-navigation.md`

Those docs were individually reasonable, but together they made it too easy to
ask "which one is the real authority?"

### 2. Entry-point overlap

The repository already had multiple useful landing pages, but not one explicit
table that mapped document type to canonical entrypoint. That made the system
depend on repository familiarity.

### 3. Weak linkage between current guidance and historical evidence

The repository had strong stage evidence, but the links between active guidance,
historical change rationale, and archived material were implicit more often than
they were explicit.

### 4. Governance not fully reinforced by checks

The earlier documentation-system improvement stages were present, but the new
documentation-governance surfaces were not part of lightweight entrypoint checks.

## Hardening Applied

### Entry points

- Added a canonical documentation-system map:
  [`documentation-system-hardening.md`](documentation-system-hardening.md).
- Added a canonical governance and taxonomy policy:
  [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md).
- Updated root and area indexes to point to these two documents directly.

### Taxonomy

- Clarified that `docs/operations/` owns documentation-system governance as an
  operational concern.
- Kept `docs/architecture/` as the canonical home for architecture and
  repository-governance rules that are structurally binding.
- Preserved `docs/stages/` and `docs/archive/` as historical surfaces only.

### Duplication control

- Marked the C5 documentation-governance docs as valid historical baseline,
  while moving current canonical authority to the new C11 docs.
- Reduced the need to repeat taxonomy guidance across `README.md`,
  `DEVELOPMENT.md`, `docs/README.md`, and `docs/operations/README.md`.

### Guard rails

- Added the new C11 docs and stage report to
  `scripts/repository-consistency-check.sh`.
- Added the new C11 docs to `scripts/bootstrap-check.sh`.
- Added the new C11 docs to `make docs`.

## What Did Not Change

C11 deliberately did not:

- move architecture documents in bulk;
- rename large historical stage/report inventories;
- relocate archive material;
- rewrite major architecture content that was already functioning as canonical
  system design.

## Maintenance Rules

- Prefer index and cross-link improvements over document moves.
- Create a new document only when the repository gains a new durable concern.
- Treat `docs/stages/` as evidence, not as current policy.
- Treat `docs/archive/` as historical comparison material, not as active
  reference.
- Keep the documentation-system map and taxonomy policy in `docs/operations/`
  current whenever the doc tree or repository entrypoints materially change.

## Related Documents

- [`../../README.md`](../../README.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`../README.md`](../README.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`documentation-taxonomy-and-authoring-conventions.md`](documentation-taxonomy-and-authoring-conventions.md)
- [`documentation-reorganization-and-operational-navigation.md`](documentation-reorganization-and-operational-navigation.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
