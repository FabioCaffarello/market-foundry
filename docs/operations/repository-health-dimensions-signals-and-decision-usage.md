# Repository Health Dimensions, Signals, And Decision Usage

## Purpose

This document turns the C25 strategic health model into a practical review
framework for `market-foundry`.

Use it to decide:

- which repository-health dimensions matter for a change or problem;
- which signals are worth observing;
- which signals are useful enough to influence action;
- which apparent metrics should be ignored because they add bureaucracy without
  improving decisions.

## How To Use This Framework

For each dimension:

1. look for the health signals below;
2. judge whether the signal is isolated or recurring;
3. decide whether the right move is clarify, consolidate, guard, or do nothing;
4. prefer the smallest durable correction.

Do not convert this into mandatory scoring. The value is in steering decisions,
not in producing status numbers.

## Dimension 1. Discoverability

### Why it matters

A healthy developer environment lets contributors answer "where do I start?"
quickly without scanning the tree or reading stage history.

### Useful signals

- `README.md`, `DEVELOPMENT.md`, `docs/README.md`, and
  `docs/operations/README.md` still point to distinct, non-overlapping starting
  points.
- new active docs are indexed in the owning README.
- contributors can reach the canonical workflow by `make help`, `make docs`,
  or the root docs instead of ad hoc search.
- area `README.md` files reduce blind directory scanning in `cmd/`, `internal/`,
  `deploy/`, `scripts/`, and `tests/`.

### Bureaucratic signals to avoid

- counting total docs in a directory;
- measuring click depth mechanically when navigation remains obvious;
- requiring every doc to appear in root docs.

### Decision usage

- improve or trim entrypoint docs when contributors are choosing the wrong
  surface;
- update the owning README when a durable doc becomes active;
- avoid adding new bridge docs if an existing entrypoint can be clarified.

## Dimension 2. Operational Reliability

### Why it matters

The environment is unhealthy if canonical flows exist on paper but cannot be
used predictably in practice.

### Useful signals

- `make bootstrap`, `make check`, `make verify`, and relevant `make smoke*`
  remain the real normal path.
- troubleshooting first lines stay coherent: `make diag`, `make ps`,
  `make logs SERVICE=...`, `make restart`.
- lightweight checks fail on actionable drift, not on historical noise.
- wrappers, docs, and harnesses describe the same supported flow.

### Bureaucratic signals to avoid

- tracking command execution frequency as a proxy for trust;
- inventing uptime-style metrics for local developer commands;
- requiring extra proof runs when a narrower canonical proof already exists.

### Decision usage

- fix contradictions between docs, wrappers, and scripts before adding more
  workflow surface;
- strengthen only cheap guard rails that catch silent drift;
- keep the proof-of-record hierarchy explicit instead of multiplying proofs.

## Dimension 3. Entrypoint Coherence

### Why it matters

Health drops when multiple public surfaces claim authority over the same task.

### Useful signals

- `make` remains the canonical public workflow surface.
- scripts stay implementation-facing or expert-facing rather than being taught
  as competing public APIs.
- `raccoon-cli` remains in the structural-analysis lane and does not drift into
  runtime control-plane behavior.
- aliases improve discovery without obscuring which command is canonical.

### Bureaucratic signals to avoid

- minimizing command count as a goal by itself;
- banning aliases categorically;
- promoting every useful internal helper to the public surface.

### Decision usage

- route recurring repository workflows through `make`;
- keep raw scripts and direct CLI usage as bounded expert surfaces;
- consolidate sibling entrypoints before adding new ones.

## Dimension 4. Navigability

### Why it matters

The repository should be physically explorable without requiring prior
knowledge of the monorepo history.

### Useful signals

- task-to-area mapping remains clear through the repository navigation docs.
- top-level areas with real developer traffic have local `README.md` files.
- contributors can locate runtime entrypoints, harnesses, configs, and tests
  from canonical maps instead of guessing directory names.
- stage evidence is navigated through `docs/stages/INDEX.md` rather than used as
  an ad hoc current index.

### Bureaucratic signals to avoid

- directory-count thresholds;
- arbitrary maximums for tree depth;
- forcing README files into low-value directories that have no navigation need.

### Decision usage

- add or refine area entrypoints where physical-tree search cost is real;
- prefer navigation maps over directory churn when the tree itself is sound;
- keep historical indexes historical.

## Dimension 5. Documentation Governance

### Why it matters

Healthy environments keep active rules in active docs and historical rationale
in historical docs.

### Useful signals

- durable rules are promoted out of stage reports into `docs/operations/`,
  `docs/tooling/`, or `docs/architecture/`.
- root docs stay curated instead of becoming shadow catalogs.
- the operations/tooling READMEs remain the discoverability owner for active
  docs in their area.
- new support docs answer a durable question instead of capturing a transient
  stage activity.

### Bureaucratic signals to avoid

- requiring every historical rationale to be restated in active docs;
- promoting stage reports to required reading for normal workflow;
- measuring health by total document count or section count.

### Decision usage

- clarify canonical ownership before writing a new doc;
- link instead of restating when overlap already has a canonical owner;
- reject stage-local docs that do not create a lasting home for a durable rule.

## Dimension 6. Tooling Sustainability

### Why it matters

Tooling should remain trustworthy, bounded, and cheap to evolve.

### Useful signals

- new needs are absorbed by the smallest correct surface: `make`, script, doc,
  CLI, or nothing.
- `raccoon-cli` command taxonomy and lifecycle remain legible.
- wrapper targets still map to real executable behavior.
- low-frequency helpers have a clear owner surface and documentation path.

### Bureaucratic signals to avoid

- adding commands just to avoid documentation;
- requiring machine-readable output for workflows that are primarily human;
- using CLI growth itself as evidence of repository maturity.

### Decision usage

- extend `make` for normal workflows, scripts for harness mechanics, and CLI for
  structural analysis;
- retire or demote helpers whose role is no longer clear;
- prefer parameterization and consolidation over near-duplicate helpers.

## Dimension 7. Maintenance Cost Control

### Why it matters

A repository can look organized while still becoming expensive to maintain.
Health requires low edit fan-out and resistance to support-surface sprawl.

### Useful signals

- adding one support rule usually changes one canonical doc and one index, not
  many root files.
- lightweight checks protect active invariants rather than historical volume.
- new support changes rarely require touching `README.md`, `DEVELOPMENT.md`,
  `docs/README.md`, and `Makefile` all at once.
- hotspots are corrected by consolidation rather than by layering more
  documentation or wrappers on top.

### Bureaucratic signals to avoid

- forcing every change into a cost worksheet;
- measuring file size alone without regard to ownership and churn;
- expanding required-file lists in checks just because a stage created new
  artifacts.

### Decision usage

- prefer one canonical catalog over mirrored catalogs;
- keep root entrypoints shallow and high-signal;
- add checks only when silent drift is objective and likely.

## Cross-Dimension Interpretation Rules

Some signals matter across more than one dimension:

- a doc missing from `docs/operations/README.md` is both discoverability drift
  and documentation-governance drift;
- a raw script taught as normal workflow is both entrypoint-coherence drift and
  tooling-sustainability drift;
- a growing required-doc list in a lightweight check is both maintenance-cost
  drift and operational-reliability risk if trust in the guard rail drops.

When one issue hits several dimensions, prefer the fix that restores clear
ownership rather than adding more explanatory surface.

## Recommended Decision Heuristics

### Clarify

Use when the surface is correct but confusing:

- tighten wording in a canonical doc;
- add or improve a link in the owning README;
- make the canonical-vs-auxiliary boundary explicit.

### Consolidate

Use when two surfaces answer the same question:

- merge guidance into the canonical owner;
- demote or trim the overlapping surface;
- avoid preserving duplicate public entrypoints "for convenience."

### Guard

Use when drift is objective, silent, and cheap to detect:

- missing index entries;
- broken local links;
- missing wrapper-to-script alignment;
- missing required canonical docs.

### Do nothing

Use when the signal is real but not recurring, durable, or repository-wide.
Not every friction point deserves a new surface or a new check.

## Recommended Review Situations

Apply this framework during:

- support-stage planning and closure;
- changes to root docs or `Makefile`;
- addition of a script, wrapper, or CLI command;
- promotion of a stage-local rule into active docs;
- proposals to extend lightweight checks.

## Relationship To The Strategic Model

This document is the applied companion to:

- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)

Read the strategic model first when the question is "what counts as repository
health?" Read this document when the question is "what signals should shape the
next decision?"

## Related Documents

- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md)
- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
