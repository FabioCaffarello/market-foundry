# Criteria For Opening, Containing, Or Rejecting New Support Surfaces

## Purpose

This document defines the executive criteria for deciding when the
`market-foundry` repository should:

- open a new support surface;
- contain a request inside an existing surface;
- consolidate overlap into one current owner;
- reject expansion and solve the need with convention or documentation only.

The goal is disciplined repository-platform growth. The repository should
improve as a development environment without turning every recurring discomfort
into a new command, script, check, wrapper, or document.

Use this document together with:

- [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md)
- [`support-surface-expansion-decision-rules-and-examples.md`](support-surface-expansion-decision-rules-and-examples.md)

## Executive Problem

The repository already has enough baseline support capability:

- `make` as the canonical public workflow surface;
- `scripts/` as the harness layer;
- `raccoon-cli` as the structural-analysis and governance surface;
- `docs/operations/` as the current support-rule home;
- lightweight checks for objective silent drift.

At this stage, the main risk is no longer missing support tooling. The main
risk is opportunistic support-surface growth:

- one more target for a narrow case;
- one more helper script for a nearby variant;
- one more document for a rule that already has a home;
- one more check for a concern that is not stable enough to enforce;
- one more wrapper that competes with the existing public path.

The discipline of this stage is therefore:
default to containment, extension, or consolidation, and open a new surface
only when the gain is durable and the owner is obvious.

## Historical Growth Pattern In This Repository

Recent repository-platform growth followed four broad patterns.

### 1. Surface openings that added durable value

These openings solved a recurring repository problem that did not yet have a
credible owner surface.

Examples:

- `make stage-status` opened a real continuity surface between
  `stage-scaffold` and `stage-check`. It closed a recurring gap in governed
  stage execution and stayed advisory rather than bureaucratic.
- `make smoke-restart-recovery` and `make codegen-equivalence` promoted
  already-real flows that were previously hidden behind scripts. This improved
  discoverability without inventing a second workflow model.
- `docs/operations/README.md` and the operations namespace created a canonical
  home for active support rules that were previously buried in architecture
  history.

Common trait:
the new surface reduced search cost and ambiguity more than it increased
maintenance fan-out.

### 2. Needs that were better absorbed by an existing owner

Some useful changes did not justify opening a new family or a new top-level
entrypoint.

Examples:

- `lint`, `test-unit`, and `stack-*` were admitted as discoverability aliases,
  but the canonical contracts stayed `check`, `test`, `up`, `down`, and other
  established targets.
- `make docs` improved orientation, but only as a curated entrypoint to
  existing documentation owners rather than a new documentation subsystem.
- CLI evolution in C23 favored grouped command maturity and hidden
  compatibility aliases instead of promoting a second flat taxonomy.

Common trait:
the need was real, but the right answer was extension or clarification inside
an existing surface.

### 3. Needs that should be solved by docs or convention only

Some friction came from unclear ownership, stale naming, or uncertainty about
which surface was canonical.

Examples:

- repeated clarification that direct `scripts/*.sh` usage is debug-oriented and
  should not compete with `make`;
- lifecycle labeling of `raccoon-cli legacy runtime-smoke` as compatibility,
  not as a current operational proof surface;
- repeated documentation that stage reports are historical evidence and not the
  owner of active support policy.

Common trait:
the problem was interpretation drift, not missing execution capability.

### 4. Expansion requests that should be rejected

The repository history also points to patterns that would add entropy faster
than value.

Examples:

- adding new scripts that differ mainly by waits, narrow flags, or adjacent
  runtime variants instead of extending an existing harness;
- adding lightweight checks for subjective, stage-local, or historically
  interesting concerns;
- opening new public workflow families when an existing family can absorb the
  task with one clearer target, one flag, or one documentation update.

Common trait:
the proposed surface would mostly preserve convenience at creation time while
moving long-term cost into docs, help text, index updates, and drift control.

## Decision States

Every support-surface proposal should end in exactly one of these states.

| Decision | Meaning | Default move |
|---|---|---|
| open | a new durable surface is truly needed | add the smallest new surface with one owner |
| contain | the need is real but belongs to an existing owner | extend existing target, script, doc, or CLI group |
| consolidate | overlap or ambiguity is now the main problem | fold sibling surfaces into one current path |
| reject | no durable surface should be added | solve with docs, convention, or nothing |

## Executive Criteria

Evaluate every proposal against these seven criteria.

### 1. Recurrence

Ask:

- will this need recur across more than one stage, wave, or contributor
  session?
- is the friction part of normal repository use rather than one active change?

Interpretation:

- open only when recurrence is credible;
- contain when recurrence is real but already sits under an existing owner;
- reject when the need is one-off, stage-local, or preference-driven.

### 2. Owner clarity

Ask:

- which current surface should own this repository question today?
- would a new surface clarify ownership or split it?

Interpretation:

- open only when the current owner is missing or structurally wrong;
- contain when the owner already exists;
- consolidate when two current surfaces already answer the same question;
- reject when ownership is vague.

### 3. Differentiated value

Ask:

- what repository question does this surface answer that is not already answered
  nearby?
- if removed, what real capability would be lost?

Interpretation:

- open only when differentiated value is clear;
- contain when the value is simply easier access to an existing capability;
- reject when the new surface is mostly a synonym.

### 4. Structural cost

Ask:

- how many docs, indexes, wrappers, checks, or help surfaces must stay aligned
  after this is added?
- is the added fan-out smaller than the confusion or cost being removed?

Interpretation:

- open only when the maintenance burden remains local and proportionate;
- contain when the new surface would create broad fan-out;
- consolidate when current fan-out is already signaling overlap;
- reject when maintenance cost dominates expected value.

### 5. Canonical-path effect

Ask:

- does this make the public path more obvious or less obvious?
- will contributors become less certain about where to start?

Interpretation:

- open only when the canonical path becomes clearer;
- contain when a child flow belongs under a current family;
- consolidate when the public story is already split;
- reject when the result would create another plausible first-choice path.

### 6. Drift risk

Ask:

- how likely is this surface to drift from its neighbors?
- would that drift be silent and misleading?

Interpretation:

- open only when alignment is cheap or strongly owned;
- contain when a new sibling would repeat the same information in another
  place;
- guard only after a stable invariant exists;
- reject when drift risk is obvious at proposal time.

### 7. Reversibility

Ask:

- if this turns out not to pay for itself, can it be demoted or removed
  cleanly?
- will a temporary convenience accidentally become permanent repository debt?

Interpretation:

- prefer containment and documented convention when reversibility is poor;
- open only when the lifecycle can remain explicit and governable.

## Decision Rules By Surface Type

### New `make` target or family

Open only when all are true:

- the flow is part of normal repository usage;
- the result belongs on the public support surface;
- it cannot be expressed cleanly as an extension of an existing family;
- one sentence in `make help` is enough to explain it.

Contain when:

- the workflow fits an existing family such as `smoke-*`, `stage-*`,
  `codegen-*`, `stack-*`, or the core workflow targets.

Reject when:

- the target would be debug-only, stage-local, or a synonym for an existing
  path.

### New script or wrapper

Open only when all are true:

- the behavior is harness-level or debug-oriented;
- shell orchestration would make the `Makefile` or CLI opaque;
- the script has a clear public owner or an explicit debug-only role.

Contain when:

- the change is better represented as a mode or flag on an existing script.

Consolidate when:

- sibling scripts differ mainly by waits, symbols, or narrow variants of the
  same proof.

Reject when:

- the script would effectively become a parallel public API.

### New CLI command or family

Open only when all are true:

- the need belongs to repository analysis, validation, or change-planning;
- the output is reusable beyond one stage;
- extending an existing CLI group is insufficient.

Contain when:

- the request fits an existing grouped command or existing `make` wrapper.

Reject when:

- the command would orchestrate runtime bring-up, proofs, or other operator
  flows already owned by `make` and `scripts/`.

### New operations doc

Open only when all are true:

- the topic needs a durable canonical home;
- an existing canonical document cannot absorb the rule cleanly;
- the new doc reduces ambiguity more than it increases taxonomy sprawl.

Contain when:

- the rule already belongs to an active owner document.

Consolidate when:

- multiple active docs compete as the practical start point for the same topic.

Reject when:

- the stage report already captures the rationale and no ongoing rule needs a
  new owner.

### New lightweight check

Open only when all are true:

- the invariant is objective and binary;
- failures are understandable and cheap to fix locally;
- the invariant protects a current canonical support surface;
- the runtime cost stays small.

Contain when:

- the invariant can be absorbed by an existing check family.

Reject when:

- the rule is subjective, historically interesting, or useful only during the
  active stage.

## Executive Decision Flow

Use this order before creating anything new.

1. State the recurring problem in one sentence.
2. Name the current owner surface that should answer it.
3. Test whether docs or naming clarification solves it first.
4. Test whether the current owner can absorb it with one small extension.
5. Test whether overlap already exists and consolidation should happen before
   any expansion.
6. Open a new surface only if the need remains unowned after those steps.

Default bias:

1. clarify
2. contain
3. consolidate
4. open

Rejection is correct whenever the proposal cannot beat containment on value,
clarity, and maintenance cost.

## Guard Rails For Non-Bureaucratic Use

These criteria are meant to improve repository judgment, not to create a heavy
approval process.

Rules:

- do not require formal scoring;
- do not block small useful improvements that clearly fit an existing owner;
- do not open a governance stage for every local support correction;
- do not turn this model into a blanket veto against ergonomic improvements;
- do require explicit reasoning when a change wants to create a new durable
  surface.

The threshold is simple:
if the proposal adds a new lasting support surface, it must explain why
containment or consolidation would be inferior.

## Final Executive Model

The repository should now treat support-surface expansion as an exception, not
as the default expression of care.

The durable rule is:

- open when capability is missing and recurrence is clear;
- contain when the owner already exists;
- consolidate when overlap is the real problem;
- reject when docs, naming, or convention are enough.

That is how the repository stays robust as a development platform without
freezing useful improvement.
