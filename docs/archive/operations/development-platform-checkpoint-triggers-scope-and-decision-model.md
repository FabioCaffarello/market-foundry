# Development Platform Checkpoint Triggers, Scope, And Decision Model

## Purpose

This document defines when strategic checkpoints for the `market-foundry`
development platform should happen, what each checkpoint should cover, and how
the output should guide proportionate decisions.

It is the operational companion to
[`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md).

## Trigger Model

Strategic checkpoints should happen at natural platform moments, not on a
standing schedule.

## Natural Triggers

### 1. Support-heavy wave closure

Trigger when a wave changes several repository-platform surfaces, such as:

- operations docs;
- root entrypoints;
- `Makefile` workflows;
- scripts or harness governance;
- `raccoon-cli` governance or lifecycle;
- stage-support or support-surface lifecycle rules.

Why it matters:
local correctness is not enough when several support surfaces move together.

Default checkpoint type:
lightweight wave-closure checkpoint.

### 2. Proposed support-surface expansion

Trigger before introducing a new durable support surface, especially:

- a new public command family;
- a new recurring doc family;
- a new lifecycle state or support wrapper;
- a new CLI cluster;
- a new automation or stage-support surface.

Why it matters:
the cheapest platform decision is often to absorb the need into an existing
owner.

Default checkpoint type:
lightweight pre-expansion checkpoint.

### 3. Repeated operational friction

Trigger when contributor friction repeats on the same hotspot even if each
incident is individually small.

Typical signals:

- repeated uncertainty about whether to start in `make`, scripts, or CLI;
- recurring doc/index drift;
- recurring mismatch between supported workflow and documented workflow;
- recurring lifecycle confusion about whether a support surface is canonical,
  auxiliary, or legacy.

Why it matters:
repetition is the signal that the problem is structural rather than local.

Default checkpoint type:
lightweight hotspot checkpoint.

### 4. Multi-surface trigger cluster

Trigger when more than one C26 trigger appears together:

- growth plus duplication;
- drift plus reliability erosion;
- discoverability degradation plus operational confusion.

Why it matters:
clustered triggers usually indicate that a platform-level decision is needed.

Default checkpoint type:
deeper strategic review if one local correction is unlikely to contain the
problem.

### 5. Next-wave readiness decision

Trigger before opening a wave expected to put more load on the development
platform.

Typical cases:

- a wave likely to introduce new support docs or workflow entrypoints;
- a wave likely to add CLI capability or expert tooling;
- a wave likely to widen stage-support expectations;
- a wave that depends on a platform hotspot already known to be fragile.

Why it matters:
expansion should not compound known platform weakness when one small
prerequisite fix would reduce risk.

Default checkpoint type:
lightweight readiness-for-next-wave checkpoint.

## Scope Model

Strategic checkpoints should cover six executive dimensions every time:

1. repository health
2. CLI reliability as a development tool
3. entrypoint coherence
4. docs and stages governance
5. structural cost
6. workflow sustainability

Checkpoint scope rule:
review all six dimensions quickly, then spend depth only where the real drift
appears.

## Lightweight Checkpoint Scope

A lightweight checkpoint should answer these questions:

1. what triggered the checkpoint;
2. which one or two dimensions are actually under pressure;
3. which canonical surface owns the correction;
4. whether the smallest response is clarify, align, consolidate, guard, or do
   nothing.

Expected shape:

- one short pass;
- one hotspot at a time;
- one owner surface;
- one proportional decision.

## Deeper Strategic Review Scope

Escalate scope only when one lightweight checkpoint is not enough.

A deeper strategic review should answer:

1. which triggers are clustering;
2. which dimensions are degrading together;
3. whether the platform problem is local, cross-surface, or lifecycle-driven;
4. whether the next best move is editorial, structural, lifecycle, or guarded;
5. whether a support-focused follow-up stage is justified.

Expected shape:

- limited to the active hotspot cluster;
- explicitly tied to real repository use;
- closed by one recommendation for the next step.

This deeper review is still not a standing audit.

## Decision Model

Use this order for every strategic checkpoint:

1. define the trigger in plain operational terms;
2. name the affected executive dimensions;
3. identify the current owner surface;
4. decide if the problem is clarity, alignment, overlap, lifecycle, trust, or
   cost;
5. choose the smallest durable response;
6. decide whether the repository should proceed unchanged, correct locally, or
   open one narrow follow-up.

## Response Ladder

The expected response ladder stays proportional:

1. do nothing
2. clarify
3. align
4. consolidate
5. guard
6. governed follow-up

Use `do nothing` explicitly when the checkpoint confirms the platform is still
healthy enough and the perceived friction is not durable.

Escalation rules:

- prefer clarify before align when ownership is already correct;
- prefer align before consolidate when the surfaces are distinct but drifted;
- prefer consolidate before admitting a new sibling surface;
- prefer guard only for objective, silent-drift, low-cost invariants;
- prefer a governed follow-up only when repeated evidence shows local action is
  no longer enough.

## Decision Outputs

Strategic checkpoints should produce one of these decision outputs.

| Output | Meaning | Typical consequence |
|---|---|---|
| continue | current platform shape is healthy enough | no extra work beyond normal maintenance |
| continue with local correction | one surface needs a small fix | update one doc, index, wrapper, or lifecycle label |
| consolidate hotspot | overlap or cost has become the main issue | merge, demote, or retire one support surface |
| guard invariant | drift is objective and recurring | add one light consistency protection |
| open narrow follow-up | the hotspot exceeds a local fix | define one support-focused stage with bounded scope |

## Decision-Proportionality Rules

Strategic checkpoints should stay useful by enforcing these rules:

- the output should be smaller than the problem if a small correction works;
- checkpoint recommendations should strengthen canonical ownership, not create
  a parallel governance layer;
- a checkpoint should never recommend a broad support program without a proven
  hotspot cluster;
- if the recommendation creates more maintenance fan-out than it removes, the
  checkpoint should step back to a smaller response.

## Relationship To C26 And C28

Use C26 for trigger language and C28 for the operating contract.

Use this C29 companion document when the repository needs an executive answer
to:

- should a strategic checkpoint happen now;
- how broad should it be;
- what decision should come out of it.

## Related Documents

- [`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md)
- [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md)
- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
- [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md)
