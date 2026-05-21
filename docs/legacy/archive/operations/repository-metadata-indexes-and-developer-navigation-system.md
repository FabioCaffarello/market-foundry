# Repository Metadata, Indexes, And Developer Navigation System

## Purpose

This document defines the lightweight navigation system for `market-foundry`.

The goal is practical discoverability:

- help contributors find the right entrypoint faster;
- reduce dependence on remembered tribal paths;
- keep the navigation layer close to the real repository structure;
- avoid a heavy metadata-management system.

## Navigation Diagnosis

Before C19, the repository already had strong document indexes, but navigation
still had three structural gaps:

1. The main indexes were doc-taxonomy oriented, not repository-shape oriented.
2. Core physical areas such as `cmd/`, `internal/`, `deploy/`, `scripts/`, and
   `tests/` had no local entrypoint file.
3. Contributors could discover canonical documents, but still had to infer
   where the real code, runtime assets, and support surfaces lived.

That made the repository understandable after reading several docs, but less
efficient to scan directly from the tree.

## Design Principles

- Keep metadata close to the area it describes.
- Prefer one short index per important top-level area over one giant central map.
- Index by developer task, not by ornamental taxonomy.
- Link to canonical sources instead of duplicating them.
- Enforce only the smallest set of navigation artifacts needed to keep the
  system alive.

## Navigation Layer

The repository navigation layer now has four levels:

| Level | Artifact type | Purpose |
|---|---|---|
| Root entrypoints | `README.md`, `DEVELOPMENT.md`, `docs/README.md` | orient the contributor to workflow, taxonomy, and repository shape |
| Area entrypoints | `cmd/README.md`, `internal/README.md`, `deploy/README.md`, `scripts/README.md`, `tests/README.md`, `tools/raccoon-cli/README.md` | explain what each top-level area owns and where to start |
| Canonical maps | this document and `repository-navigation-maps-entrypoints-and-maintenance-rules.md` | connect the repository tree to real tasks and rules |
| Guard rails | `scripts/repository-consistency-check.sh` | keep critical navigation artifacts present and linked |

## Metadata Model

This navigation system uses only lightweight metadata:

- directory entrypoints (`README.md`);
- top-level documentation indexes;
- task-oriented tables that map need -> starting point;
- cross-links between workflow docs and physical repository areas.

It intentionally does not introduce:

- generated metadata registries;
- separate YAML or JSON catalogs for docs;
- ownership systems detached from the directory tree;
- synthetic maps that must be manually synchronized with every minor file move.

## Canonical Entrypoints By Question

| Question | Start here | Then go to |
|---|---|---|
| How do I work in this repository day to day? | `DEVELOPMENT.md` | `docs/operations/README.md` |
| Which top-level area owns a concern? | `docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md` | the relevant area `README.md` |
| Which binary starts this behavior? | `cmd/README.md` | `cmd/<service>/` |
| Which layer should hold this code? | `internal/README.md` | relevant package tree |
| Which runtime assets back this service? | `deploy/README.md` | `deploy/configs/`, `deploy/compose/`, or `deploy/migrations/` |
| Which script or wrapper owns this workflow? | `scripts/README.md` | Makefile target or script catalog doc |
| Which docs are current versus historical? | `docs/README.md` | `docs/operations/README.md`, `docs/architecture/README.md`, or `docs/stages/INDEX.md` |

## Sustainability Rules

- Update the closest area entrypoint when a top-level directory gains a new
  responsibility.
- Update root entrypoints only when the change affects repository-wide
  discoverability.
- If a navigation statement becomes architecture policy, promote it to
  `docs/architecture/`; if it remains usage guidance, keep it in
  `docs/operations/`.
- Prefer deleting stale links over preserving broad but misleading indexes.
- Keep maintenance local: a change in `deploy/` should rarely require edits in
  more than `deploy/README.md` plus one root or operations index.

## Enforcement Scope

The consistency guard rail should enforce only:

- existence of core documentation entrypoints;
- existence of the core repository-area entrypoints;
- index alignment for stage reports;
- link validity in primary support docs.

It should not enforce exhaustive documentation coverage for every directory.

## What C19 Adds

- physical repository entrypoints for the most important top-level areas;
- a practical navigation map that connects tasks to directories;
- cross-links from root docs into those area entrypoints;
- consistency checks so the navigation layer remains present.
