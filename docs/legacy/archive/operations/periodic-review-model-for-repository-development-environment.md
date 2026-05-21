# Periodic Review Model For Repository Development Environment

## Purpose

This document defines the lightweight periodic review model for the
`market-foundry` repository as a development environment.

Its job is to answer four practical questions:

- when should the repository support surface be reviewed;
- which surfaces deserve recurring review;
- which review loops should stay informal and continuous;
- when should the repository run a more strategic support-environment review.

The goal is sustainability, not ceremony. The repository should detect support
surface degradation early enough to correct it with small changes.

## Scope

This model governs the repository development environment only.

It covers:

- `Makefile` and public workflow entrypoints;
- scripts and harnesses;
- `raccoon-cli` as a development-governance tool;
- active operational docs and indexes;
- area entrypoints and local READMEs;
- lightweight repository guard rails;
- stage and harness governance support.

It does not change service architecture, domain boundaries, or runtime feature
design.

## Review Layers

The repository should operate with two review layers.

### 1. Continuous informal review

This is the default.

It happens inside normal work whenever a contributor touches:

- a public command surface;
- a script or harness;
- an active operations or tooling doc;
- an area README or navigation index;
- a lightweight guard rail;
- stage-support or harness-governance material.

This layer is cheap and local. It should answer:

- did this change create confusion or overlap;
- is the owning surface still the right one;
- do the entrypoints and docs still agree;
- does the change need a small follow-through update.

Use the routines in
[`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
for this layer.

### 2. Periodic strategic review

This layer is less frequent and broader.

It exists because some repository-environment problems do not show up inside
one isolated change:

- many small helpers accumulate until the surface feels fragmented;
- docs remain locally correct but the overall navigation stack becomes crowded;
- harnesses stay functional but begin overlapping in responsibility;
- low-grade trust erosion appears across checks, entrypoints, and support docs.

The strategic review is a short structured pass over the support environment,
not a governance event for every change.

## Surfaces That Require Recurring Review

### CLI

Review because:

- direct `raccoon-cli` use is legitimate but intentionally bounded;
- command growth can quietly blur the boundary between expert tooling and
  public workflow;
- discoverability and lifecycle clarity degrade if commands accumulate without
  role discipline.

Review for:

- command-role clarity;
- overlap with `make`;
- lifecycle hygiene and deprecation needs;
- whether command growth still reflects expert inspection value.

### Scripts

Review because:

- scripts are where harness entropy accumulates fastest;
- direct-script usage can start competing with the intended public workflow;
- low-frequency helpers can survive past their useful life.

Review for:

- mapping to a clear public owner surface;
- overlap between harnesses;
- debugging-only vs routine-use clarity;
- size, duplication, and support burden.

### Makefile

Review because:

- `make` is the canonical public workflow surface;
- alias growth and wrapper growth can erode canonicality if left unchecked.

Review for:

- whether target families remain coherent;
- whether aliases help discoverability without obscuring canonical targets;
- whether targets still match the documented workflow;
- whether wrappers still point to real, trusted behavior.

### Operational docs

Review because:

- support guidance drifts through accumulation more often than through explicit
  design mistakes;
- root docs and operations docs can start mirroring each other if not checked.

Review for:

- active-doc discoverability;
- canonical ownership;
- root-doc shallowness;
- overlap or accidental sibling docs.

### Indexes and navigation surfaces

Review because:

- support docs become expensive to use once indexes lag behind the active tree;
- discoverability degrades before anything is technically broken.

Review for:

- `docs/README.md`, `docs/operations/README.md`, `docs/tooling/README.md`, and
  `docs/stages/INDEX.md` remaining faithful to their intended roles;
- area READMEs still helping real navigation;
- navigation maps still matching the physical tree.

### Entrypoints

Review because:

- contributor confusion usually appears first at the entrypoint layer, not deep
  in implementation;
- crowded or overlapping entrypoints increase search cost and merge hotspots.

Review for:

- role separation across `README.md`, `DEVELOPMENT.md`, `docs/README.md`, and
  `docs/operations/README.md`;
- whether area READMEs still reduce blind scanning;
- whether root docs remain curated instead of becoming catalogs.

### Lightweight guard rails

Review because:

- trusted checks are high-value only while they remain small, objective, and
  understandable;
- guard rails that grow for historical reasons become noise and lose trust.

Review for:

- objective invariant fit;
- failure clarity;
- silent-drift protection value;
- whether a rule belongs in documentation rather than in code.

### Harness governance and stage support

Review because:

- the repository now has explicit governance support for proofs, stage
  execution, and support-surface continuity;
- this area can become procedural if not kept intentionally light.

Review for:

- whether stage helpers still remove friction without becoming a workflow
  engine;
- whether proof-of-record rules remain explicit;
- whether support-stage output is promoted into active docs instead of left in
  historical reports.

## Review Types

### Type A. Lightweight recurring review

Use this review:

- during ordinary changes;
- at stage closure when support surfaces were touched;
- when a new doc, script, target, wrapper, or guard rail is proposed.

Characteristics:

- local to the changed surface;
- fast enough to fit normal repository work;
- focused on coherence, discoverability, and ownership;
- usually resolved by one small correction.

Typical outputs:

- clarify one canonical doc;
- index one active doc;
- trim an overlapping explanation;
- reject one unnecessary helper;
- add one small guard rail only if drift would otherwise stay silent.

### Type B. Strategic periodic review

Use this review:

- at the end of a support-heavy wave;
- when multiple weak signals point to the same support-surface hotspot;
- when the environment feels harder to navigate or trust even though local
  artifacts are still individually valid.

Characteristics:

- cross-surface, not tied to one file;
- concerned with cost, drift, and coherence trends;
- still short and decision-oriented;
- intended to produce proportionate cleanup, not a new governance program.

Typical outputs:

- choose one hotspot for consolidation;
- retire or demote one low-value surface;
- rewrite one index or entrypoint boundary;
- authorize one narrowly-scoped follow-up stage when local fixes are no longer
  enough.

## Suggested Cadence

Use the lightest cadence that preserves support-surface health.

### Continuous

Run lightweight recurring review whenever a change touches:

- `Makefile`;
- `scripts/`;
- `docs/operations/`;
- `docs/tooling/`;
- `README.md`, `DEVELOPMENT.md`, `docs/README.md`;
- stage-support or proof-governance surfaces;
- repository guard rails.

### Per stage closure

Run a short review when closing any stage that changed repository-support
surfaces.

The question is not "did we finish everything?" The question is:

- what lasting support rule or entrypoint changed;
- where is that rule now canonically owned;
- what small cleanup prevents local drift from becoming repository drift.

### Per support-heavy wave

Run one strategic review pass near wave closure when the wave introduced or
modified multiple support surfaces.

This is the default periodic strategic review for the repository.

### On signal, outside the normal cadence

Run a strategic review earlier when the trigger model in the companion document
shows recurring drift or support-surface confusion.

## Minimum Strategic Review Agenda

A strategic review should stay short and answer these questions:

1. Which support surfaces accumulated the most friction since the last pass?
2. Which symptoms are local noise and which are recurring structural signals?
3. Which hotspot is high-value and low-cost to correct now?
4. What is the smallest follow-through action that improves trust or
   discoverability?
5. Does the result belong in a doc update, a consolidation, a guard rail, or a
   small governed follow-up stage?

If the review cannot produce a bounded action, it should not produce a broad
program instead.

## High-Value, Low-Cost Review Priorities

The repository should prefer recurring review on surfaces where small
adjustments have broad payoff:

- root and operations entrypoints;
- `Makefile` target clarity;
- script-to-owner mapping;
- active-doc indexing;
- harness role boundaries;
- guard rails that contributors use habitually;
- stage-support docs that influence many future waves.

These are high-value because they shape normal repository use, and low-cost
because most corrections are editorial, consolidating, or narrowly procedural.

## Relationship To Existing Governance

This model builds on the previous repository-governance stages:

- C19 established repository navigation and indexes;
- C20 defined where automation helps continuity;
- C21 defined structural-cost control;
- C22 defined support-surface extension discipline;
- C23 defined CLI lifecycle discipline;
- C24 defined sustainability review routines;
- C25 defined the strategic health model and health dimensions.

C26 adds the missing cadence layer:

- when to review;
- what surfaces deserve recurring review;
- how to separate continuous informal review from periodic strategic review.

## Companion Document

Use
[`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
for the trigger model and action-selection rules.

This document defines the cadence model.
The companion document defines how signals turn into proportionate follow-through.

## Related Documents

- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
- [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
