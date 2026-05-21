# Documentation Reorganization And Operational Navigation

## Purpose

This document records the C5 documentation reorganization for operational and
support surfaces in `market-foundry`.

The goal is to reduce documentation entropy, improve navigation, and make daily
repository execution easier without rewriting the main architecture corpus.

## Status

This document is the historical C5 reorganization record. For the current
canonical documentation-system map and governance rules, use:

- [`documentation-system-hardening.md`](documentation-system-hardening.md)
- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)

## Executive Summary

Before C5, the repository had strong architectural depth but weak documentation
navigation:

- there was no `docs/` landing page
- `docs/operations/`, `docs/tooling/`, `docs/architecture/`, and `docs/archive/`
  had no README-style entrypoints
- daily operational runbooks were hard to discover because they lived inside the
  architecture corpus
- the stage index existed, but support/documentation mini-stages were not indexed
- the canonical governance doc for documentation still described an older tree

C5 fixes navigation and conventions while keeping canonical content in place.

## Diagnosis

### 1. Daily-entry docs were fragmented

The repository entrypoints existed, but they did not form a complete navigation
system:

- `README.md` covered overview
- `DEVELOPMENT.md` covered workflow
- `docs/stages/INDEX.md` covered stage history
- everything else required repository folklore

That left no clear answer to "where do I go for operations, tooling, or
documentation rules?"

### 2. Operational guidance was discoverable only through architecture history

Several current runbook-style docs lived in `docs/architecture/`, including:

- `minimal-operational-baseline.md`
- `current-baseline-runbook.md`
- `current-baseline-operational-diagnostics.md`
- `operational-smoke-ci-and-runbook-closure.md`
- `analytical-observability-and-runbook.md`

Those are legitimate architecture records, but they were not surfaced through an
operator-first index.

### 3. Tooling docs had no area index

`docs/tooling/` already contained 15 documents, but there was no directory-level
README explaining:

- where to start
- the difference between user-facing CLI docs and tool-internal rule docs
- how to interpret similarly named files such as `cli-execution-*` and
  `cli-execute-*`

### 4. Documentation governance described an outdated docs tree

`docs/architecture/monorepo-documentation-and-stage-governance.md` still modeled
the docs tree as:

- `docs/architecture/`
- `docs/stages/`
- `docs/tooling/`

That omitted the now-important operational and archive surfaces.

### 5. Stage history was indexed, but support/documentation stages were not

`docs/stages/INDEX.md` indexed major numbered phases but did not include the C1-C4
repo-support/documentation stages. That made recent support work harder to find.

## Reorganization Applied

### New landing pages

Added area entrypoints:

- `docs/README.md`
- `docs/operations/README.md`
- `docs/tooling/README.md`
- `docs/architecture/README.md`
- `docs/archive/README.md`

These pages provide stable navigation without moving the canonical documents that
other docs already reference.

### New operational governance documents

Added:

- `documentation-reorganization-and-operational-navigation.md`
- `documentation-taxonomy-and-authoring-conventions.md`

Together they define the navigation model and the authoring rules for future docs.

### Updated repository entrypoints

Updated:

- `README.md`
- `DEVELOPMENT.md`
- `Makefile` (`make docs`)

These entrypoints now point to documentation indexes instead of isolated deep links.

### Updated stage navigation

Updated `docs/stages/INDEX.md` to:

- explain that the index is historical, not the primary daily workflow entrypoint
- include the C1-C5 support/documentation stages

### Updated canonical documentation-governance doc

Updated `docs/architecture/monorepo-documentation-and-stage-governance.md` so its
described structure matches the actual repository:

- `docs/operations/` is now recognized as the support and operational surface
- `docs/archive/` is now recognized as the historical non-canonical surface

## Canonical Navigation Model

| Question | Canonical entrypoint |
|---|---|
| "What is this repository?" | `README.md` |
| "How do I work here day to day?" | `DEVELOPMENT.md` |
| "Where are operational docs and runbook links?" | `docs/operations/README.md` |
| "Where are tooling rules and analyzer references?" | `docs/tooling/README.md` |
| "What is the canonical architecture?" | `docs/architecture/README.md` |
| "Where is the stage history?" | `docs/stages/INDEX.md` |
| "Where do archived docs live?" | `docs/archive/README.md` |

## Docs That Were Hard To Find And How C5 Resolved Them

| Document or surface | Problem before C5 | Resolution |
|---|---|---|
| `current-baseline-runbook.md` | Buried inside `docs/architecture/` | Linked from `docs/operations/README.md` |
| `current-baseline-operational-diagnostics.md` | Buried inside `docs/architecture/` | Linked from `docs/operations/README.md` |
| `analytical-observability-and-runbook.md` | Easy to miss unless you knew the analytical tranche history | Linked from `docs/operations/README.md` and `docs/architecture/README.md` |
| `docs/tooling/*` as a set | No area index | `docs/tooling/README.md` |
| `docs/archive/*` as a set | No explanation of archive status | `docs/archive/README.md` |
| C1-C4 support stages | Not present in stage index | Added to `docs/stages/INDEX.md` |

## Explicit Non-Goals

C5 deliberately did not:

- rewrite the main architecture documents
- relocate large parts of `docs/architecture/`
- rename historical stage files
- create a deep nested docs taxonomy
- treat stage reports as canonical workflow documents

## Expected Outcomes

- contributors can find operational and tooling docs without knowing stage history
- operators can reach current runbooks through `docs/operations/README.md`
- new docs have a placement model that reduces future entropy
- the docs tree is more legible without disrupting canonical architecture paths
