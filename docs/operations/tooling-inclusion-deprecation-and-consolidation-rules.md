# Tooling Inclusion, Deprecation, And Consolidation Rules

## Purpose

This document turns the C22 extension-discipline model into concrete operating
rules for adding, consolidating, deprecating, or removing support tooling in
`market-foundry`.

Use it when a new helper, target, script, doc, or lightweight check is being
considered.

## Inclusion Rules

### Add a new surface only when the need is recurring

A new surface is justified when the same friction is likely to recur across
more than one stage, wave, or contributor session.

Do not add a durable surface for:

- one-off migration support;
- temporary stage-local ceremony;
- narrow personal preference;
- alternate spelling of an existing workflow.

### Choose the owning surface before implementing anything

Use this inclusion matrix:

| Need | Preferred surface | Inclusion test |
|---|---|---|
| normal human workflow | `Makefile` | routine, stable, short public contract |
| harness logic or debug flags | `scripts/` | lower-level than `make`, shell-heavy, implementation detail |
| structural analysis or machine-readable governance | `raccoon-cli` | tooling-specific, read-heavy, reusable analysis |
| lasting rule or operating model | `docs/operations/` or `docs/tooling/` | canonical explanation needed, no new execution surface required |
| cheap objective invariant | lightweight check | high-signal drift, local and routine |
| isolated clarification | existing canonical doc only | no new surface needed |

### Prefer extension over addition

Before adding a new item, check in this order:

1. Can an existing canonical doc absorb the rule?
2. Can an existing Make family absorb the workflow?
3. Can an existing script accept one more explicit flag?
4. Can an existing CLI group own the analysis?
5. Is the right answer to document the boundary and add nothing else?

Create something new only after these options are rejected for clear reasons.

## Specific Inclusion Criteria By Surface

### New Make target

Add a target only when:

- the workflow belongs on the public support surface;
- the name will remain valid after the current stage closes;
- the behavior is coherent enough to deserve a stable invocation contract;
- the target can be documented in one sentence in `make help`.

Reject the target when:

- it is debug-only;
- it merely forwards to a one-off script for a rare case;
- it competes with an existing canonical target;
- it exists only because a current script is large or awkward.

### New script

Add a script only when:

- the logic is harness-oriented and would make the `Makefile` opaque or brittle;
- the behavior benefits from direct flags for expert use;
- the script has a credible owner surface in `make` or a documented debug role.

Reject the script when:

- it answers the same workflow as another script with slightly different waits
  or symbols;
- it would be taught as a first-class user flow instead of a harness layer;
- the change is better expressed as a flag or mode on an existing script.

### New CLI command

Add a CLI command only when:

- it extends the existing grouped taxonomy cleanly;
- the output is valuable outside a single stage;
- the behavior belongs to repository analysis, validation, or change-planning;
- `make` would be too coarse or too human-oriented for the use case.

Reject the command when:

- the request is operational runtime orchestration;
- the same need is already satisfied by `make check`, `make tdd`, `make verify`,
  or `make smoke*`;
- the command would be mostly a wrapper around an operational script.

### New operations/tooling document

Add a document only when:

- the topic needs a canonical home that will outlive the current stage;
- the topic cannot fit cleanly into an existing canonical document;
- the document reduces ambiguity about ownership, lifecycle, or selection.

Reject the document when:

- it mostly restates another canonical doc;
- the stage report already captures the one-time rationale sufficiently;
- the topic is still too fluid to deserve its own home.

### New lightweight check

Add a check only when:

- the invariant is binary and objective;
- failures are easy to understand and fix locally;
- the invariant protects a canonical workflow, entrypoint, or support asset;
- the runtime cost stays small enough for habitual execution.

Reject the check when:

- it evaluates subjective quality;
- it enforces historical completeness rather than active invariants;
- it is only useful during the current stage.

## Consolidation Rules

Consolidate before adding when any of these signals appear:

- more than one public entrypoint is offered for the same repository question;
- two docs both claim to be the practical start point for the same topic;
- a support family grows through synonyms instead of capability;
- a script split would merely hide duplication across sibling files;
- a check list grows by appending every new support artifact forever.

Preferred consolidation moves:

1. extend an existing Make family
2. add an explicit flag or mode to an existing script
3. merge overlapping docs and leave one bridge link
4. de-emphasize aliases while keeping compatibility if needed
5. remove low-value invariants from lightweight checks

## Deprecation Rules

Deprecate a support surface when it is still referenced enough that abrupt
removal would cause confusion, but it is no longer canonical.

The deprecation bar is met when:

- there is a documented canonical replacement;
- the old surface adds confusion if presented as current;
- compatibility is still temporarily useful.

Deprecation actions:

1. mark the old surface as non-canonical in the owning doc or help text
2. point directly to the replacement
3. remove it from curated entrypoints such as `make docs` and root-doc examples
4. keep compatibility only as long as active usage or historical references make
   that worthwhile

## Removal Rules

Remove a surface instead of deprecating it when:

- it is no longer referenced by current docs or public workflows;
- it has no real compatibility value;
- keeping it would encourage drift or duplicate maintenance.

Removal should normally happen in the same change that:

- updates the owning canonical doc;
- updates any affected check or index;
- clarifies the replacement path if one exists.

## Ownership Rules

Every lasting addition must answer these ownership questions:

| Question | Expected answer |
|---|---|
| Who owns the public contract? | `Makefile`, `docs/operations/`, or `docs/tooling/` |
| Who owns the implementation? | script, CLI module, or existing substrate |
| Who owns the canonical explanation? | one current document, not a stage report |
| Who keeps it coherent? | the same change that edits behavior updates docs and guard rails as needed |

If ownership is vague, the addition is not ready.

## Lifecycle Rules For Support Changes

Apply this sequence when a change is justified:

1. extend the smallest existing surface that can own the need
2. update the owning canonical doc
3. update entrypoint docs only if contributor orientation changes materially
4. update lightweight checks only if a new stable invariant was introduced
5. avoid leaving behind an unlabeled legacy path

This sequence preserves agility while preventing opportunistic sprawl.

## Naming And Canonicality Rules

- One workflow should have one canonical public entrypoint.
- A discoverability alias must not be documented as a competing canonical path.
- A script name should describe the harness responsibility, not stage-specific
  temporary intent.
- A document title should describe a repository-specific rule or model, not a
  vague governance theme.
- Compatibility helpers should be visibly labeled as legacy, wrapper, alias, or
  debug-only support.

## Review Questions

Before landing a support-tooling change, answer:

1. What surface owns this now?
2. What existing surface did we decide not to extend, and why?
3. What future maintenance cost does this add?
4. What duplicate or drifting surface did this change avoid or retire?
5. Is the result still lightweight enough for this repository's current scale?

## Related Documents

- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md)
- [`repository-maintenance-hotspots-and-cost-reduction-principles.md`](repository-maintenance-hotspots-and-cost-reduction-principles.md)
- [`repository-consistency-invariants-and-check-policy.md`](repository-consistency-invariants-and-check-policy.md)
- [`../tooling/cli-overview.md`](../tooling/cli-overview.md)
