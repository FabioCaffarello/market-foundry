# Support-Surface Sunset, Consolidation, And Retirement Strategy

## Purpose

This document defines how `market-foundry` should keep its support surfaces
sustainable as the development environment grows.

It covers the lifecycle of commands, scripts, operational docs, entrypoints,
indexes, wrappers, and lightweight helper surfaces.

The goal is not removal for its own sake. The goal is to prevent low-grade
entropy, overlapping ownership, and avoidable maintenance cost.

Use this document when deciding whether a support surface should:

- remain active as-is;
- be consolidated into a stronger existing surface;
- be marked as legacy or compatibility-only;
- be retired.

## Strategic Position

- `make` remains the canonical public workflow surface.
- `scripts/` remains the harness and lower-level debug layer.
- `docs/operations/` remains the canonical home for current support rules and
  workflow navigation.
- `docs/tooling/` remains the home for tool-internal rule sets.
- `docs/stages/` remains historical evidence, not the owner of active support
  policy.

Sunset and consolidation decisions must preserve those ownership boundaries.

## Lifecycle States

| State | Meaning | Typical examples | Default action |
|---|---|---|---|
| active canonical | The preferred current surface for a recurring repository need | `make check`, `make verify`, canonical operations docs, area READMEs | keep aligned, indexed, and trusted |
| active auxiliary | A justified supporting surface that does not compete with the canonical path | script debug flags, expert CLI commands, discoverability aliases | keep only while the differentiated value remains real |
| legacy | A tolerated compatibility or historical surface that should not be promoted | compatibility aliases, legacy wrappers, superseded helper paths | label clearly, freeze scope, route users to replacement |
| retired | A removed surface whose purpose is fully absorbed or no longer justified | stale wrapper, redundant doc, obsolete helper | keep replacement and indexing coherent |

## Core Strategy

### 1. Keep one canonical answer per recurring repository question

The repository should avoid multiple first-choice answers to the same durable
question.

Examples:

- one canonical public workflow entrypoint;
- one canonical operational proof family;
- one canonical document for a support rule;
- one canonical index per active doc area.

Healthy redundancy is allowed only when it improves usability without splitting
ownership.

### 2. Prefer consolidation before retirement

Most support-surface entropy should first be corrected by:

1. clarifying ownership;
2. aligning docs and help text;
3. folding nearby surfaces together;
4. demoting a surface to auxiliary or legacy.

Retirement is appropriate only when the repository no longer gets meaningful
value from keeping the older surface around.

### 3. Distinguish healthy redundancy from costly redundancy

Healthy redundancy:

- a `make` target and its underlying script, when the target is public and the
  script preserves expert flags;
- a curated root entrypoint and a richer area README, when one orients and the
  other catalogs;
- a discoverability alias that helps scanning without splitting examples or
  ownership.

Costly redundancy:

- two active docs both acting as the practical start point for the same topic;
- multiple wrappers that teach the same workflow as equally canonical;
- helper surfaces whose differences are mainly naming, waits, or local habit;
- indices that summarize the same current surface in parallel.

### 4. Use lifecycle labels intentionally

Support surfaces should not silently age.

When a surface is no longer canonical but still useful, label it:

- alias;
- wrapper;
- debug-only;
- compatibility-only;
- legacy.

An unlabeled non-canonical path is the fastest route to accidental drift.

## Decision Framework

Evaluate each support surface against these dimensions:

| Dimension | Keep active when | Consolidate when | Mark legacy when | Retire when |
|---|---|---|---|---|
| usage | routine or strategically recurring | usage splits across nearby surfaces | residual usage is mostly compatibility-driven | little or no current workflow value remains |
| clarity | the role is obvious to contributors | sibling surfaces cause ambiguity | current role is transitional or historical | the surface mainly confuses |
| maintenance cost | updates stay local and low-fan-out | same change must touch too many nearby owners | only compatibility-safe maintenance is justified | keeping it adds avoidable upkeep |
| discoverability | the surface improves findability | indexing or help is duplicated | it must remain findable only as a migration bridge | discovery value is negligible |
| responsibility overlap | ownership is narrow and coherent | nearby surfaces answer the same durable question | replacement ownership is already established | overlap has already been resolved elsewhere |
| drift risk | the surface is easy to keep aligned | drift keeps appearing between peers | frozen behavior is safer than ongoing evolution | drift risk outweighs compatibility benefit |

## Surface-Specific Guidance

### Make targets

Keep a target active when it is a stable public contract with a short,
recurring purpose.

Consolidate when:

- aliases multiply faster than capabilities;
- a target family answers the same workflow with cosmetic variants;
- the same user intent is taught through several top-level targets.

Mark as legacy when compatibility matters but the target should stop being
promoted.

Retire when the workflow is fully absorbed elsewhere and current docs no longer
depend on the older target.

### Scripts and wrappers

Keep a script active when it has a harness/debug role distinct from the public
Make surface.

Consolidate when:

- adjacent scripts differ mostly by waits, symbols, or small harness options;
- a new script is proposed where a flag on an existing script would do;
- wrappers and scripts are both taught as normal first-choice paths.

Mark as legacy when a script remains only to avoid abrupt breakage for existing
operators or narrow tooling hooks.

Retire when the script is no longer needed for debugging, compatibility, or
implementation structure.

### Operational docs and runbooks

Keep a doc active when it owns a durable support rule or navigation concern.

Consolidate when:

- two docs both summarize the same current policy;
- one doc exists mainly to repeat another with slightly different framing;
- the same lifecycle rule appears in several active places.

Mark as legacy only rarely for docs; prefer archive or retirement once a clear
replacement exists.

Retire when the rule is fully captured by another canonical active doc or the
topic is no longer part of the active support model.

### Entrypoints and indexes

Keep an entrypoint or index active when it shortens navigation without trying
to re-own every detail.

Consolidate when:

- root docs start mirroring area indexes;
- more than one active index claims the same navigational role;
- an area README grows into a second documentation map.

Retire when an index is no longer a real entrypoint and merely duplicates a
better-maintained navigation surface.

### Lightweight checks

Keep a check active when it protects a stable high-signal invariant.

Consolidate when:

- several checks validate nearby editorial concerns that could be guarded by
  one simpler invariant;
- a check grows to protect historical completeness rather than active policy.

Mark as legacy only in exceptional cases. Usually checks should either stay
active or be removed.

Retire when the invariant is no longer active, objective, or worth the routine
cost.

## Sunset Process

Use this lightweight sequence when a surface shows lifecycle stress:

1. identify the canonical owner and replacement path;
2. decide whether the issue is clarify, align, consolidate, legacy, or retire;
3. update help text or docs so the non-canonical status is explicit;
4. remove the surface from curated entrypoints if it is no longer first-choice;
5. keep only the minimum compatibility needed;
6. retire once current active usage and indexing no longer justify keeping it.

This process should be applied proportionally. Most cases should end at
clarify, align, or consolidate.

## Current Hotspots In Market Foundry

Current support-surface areas that warrant ongoing lifecycle discipline:

- Make aliases versus canonical target names;
- `make live*` orchestration versus `make smoke*` proof-of-record ownership;
- direct script invocation versus public `make` workflows;
- broad operational governance docs versus overlapping summaries;
- repository indexes, area READMEs, and curated root entrypoints;
- compatibility-only CLI helper paths.

These are not all removal targets. They are the highest-likelihood drift and
overlap hotspots.

## Review Questions

Before keeping or adding a support surface, answer:

1. What recurring repository question does this surface own?
2. Which nearby surface could absorb this instead?
3. Is this improving discovery, or just adding another place to look?
4. If the surface became frozen tomorrow, would it still justify its upkeep?
5. If a new contributor saw both surfaces, would canonical ownership still be
   obvious?

Weak answers are a consolidation signal.

## Sustainability Rule

Support surfaces should age intentionally.

The repository should tolerate:

- a small amount of purposeful redundancy;
- gradual compatibility bridges;
- narrow auxiliary surfaces.

The repository should resist:

- parallel first-choice entrypoints;
- unlabeled legacy paths;
- repeated summaries of the same active rule;
- helpers whose upkeep exceeds their differentiated value.

## Related Documents

- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`../tooling/raccoon-cli-command-lifecycle-and-deprecation-strategy.md`](../tooling/raccoon-cli-command-lifecycle-and-deprecation-strategy.md)
