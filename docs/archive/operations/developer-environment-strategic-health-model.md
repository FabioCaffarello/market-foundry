# Developer Environment Strategic Health Model

## Purpose

This document defines how `market-foundry` should evaluate the health of the
repository as a development environment.

The goal is not to build a metrics platform. The goal is to keep one simple,
useful model for judging whether the repository remains easy to enter, safe to
change, and sustainable to operate as a support system for engineering work.

## What "Health" Means Here

Repository health is the condition in which contributors can:

- find the right entrypoint without tribal knowledge;
- run the normal workflow with predictable outcomes;
- trust the checks, scripts, and docs enough to use them habitually;
- extend the support surface without creating structural drag.

This is a strategic model for the developer environment only. It does not score
runtime business behavior and it does not redefine functional architecture.

## Strategic Diagnosis

The repository already has a strong baseline:

- `make` is the canonical public workflow surface;
- `README.md`, `DEVELOPMENT.md`, `docs/README.md`, and
  `docs/operations/README.md` give a clear entrypoint stack;
- `scripts/` and `raccoon-cli` have bounded roles;
- `make repo-consistency-check`, `make stage-status`, and `make stage-check`
  provide lightweight operational governance;
- C19 through C24 established navigation, sustainability, structural-cost, and
  tooling-evolution rules.

The remaining gap was strategic: the repository had governance pieces, but not
one explicit model for deciding whether the environment is healthy overall and
where the next improvement should land.

## Design Principles

### 1. Health is multi-dimensional

No single count or score can represent repository health. The right question is
which dimensions are degrading and whether that degradation harms normal work.

### 2. Signals matter more than dashboards

The model should use a small number of qualitative and operational signals that
contributors can observe during normal work:

- confusion at entrypoints;
- repeated drift in canonical docs;
- failing or mistrusted lightweight checks;
- support changes that require too many reconciliations.

### 3. Canonical surfaces should stay explicit

The model assumes that healthy repositories preserve clear ownership:

- `make` for public workflow;
- `scripts/` for harness implementation;
- `raccoon-cli` for structural analysis and machine-readable support tooling;
- `docs/operations/` for active support policy;
- `docs/tooling/` for tooling-internal governance;
- `docs/stages/` for historical evidence only.

### 4. Improvement should stay proportional

A health model is useful only if it guides small, correct interventions.
Most repository-health issues should be corrected by:

- tightening one index;
- clarifying one canonical doc;
- consolidating one entrypoint;
- rejecting one low-value helper;
- adding one cheap, objective check when silent drift is otherwise likely.

## Health Dimensions

The repository should evaluate health across seven dimensions:

1. discoverability
2. operational reliability
3. entrypoint coherence
4. navigability
5. documentation governance
6. tooling sustainability
7. maintenance cost control

These dimensions were chosen because they map directly to the current
repository shape: root docs, `Makefile`, `scripts/`, `raccoon-cli`,
operations/tooling docs, area READMEs, and lightweight guard rails.

## Health Model Shape

Use the model in three layers:

1. dimension: what property of the environment is being protected;
2. signal: what observable evidence says the property is healthy or drifting;
3. decision use: what kind of repository decision the signal should influence.

The model should stay qualitative-first. A signal is useful when it improves
prioritization or design judgment. A signal is bureaucratic when it creates
counting work without changing real decisions.

## What This Model Rejects

Do not turn repository health into:

- a numeric scorecard;
- a mandatory weekly audit ceremony;
- a broad instrumentation program;
- a reason to add checks for subjective preferences;
- a substitute for engineering judgment.

If a proposed repository-health practice needs persistent scoring, heavy
reporting, or more process than the change it is trying to prevent, it is too
heavy for this repository.

## Operating Pattern

Apply this model when:

- choosing whether a support issue deserves action;
- deciding which surface should absorb a new workflow or rule;
- reviewing support-stage closure;
- prioritizing cleanup work across docs, scripts, CLI, entrypoints, and checks.

A good repository-health decision usually looks like one of these:

- simplify the entrypoint stack instead of adding another guide;
- improve one canonical doc instead of creating a sibling document;
- add a Make wrapper instead of promoting a raw script;
- keep a concern manual if the invariant is not objective enough for a check;
- reject a convenience surface that would increase maintenance fan-out.

## Relationship To Existing Governance

This document does not replace the existing operations governance set. It sits
above it as a decision model:

- C17 defines the developer-environment lifecycle;
- C19 defines repository navigation;
- C20 defines lightweight automation boundaries;
- C21 defines structural-cost control;
- C22 defines extension discipline;
- C23 defines CLI lifecycle discipline;
- C24 defines long-term sustainability and review routines.

C25 adds the missing strategic layer: how to interpret those systems together
as repository health.

## Canonical Companion Document

The practical dimension-by-dimension signals and decision guidance live in:

- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)

Use this document for the strategic model and the companion document for
applied review and prioritization.

## Related Documents

- [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md)
- [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
- [`repository-metadata-indexes-and-developer-navigation-system.md`](repository-metadata-indexes-and-developer-navigation-system.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
