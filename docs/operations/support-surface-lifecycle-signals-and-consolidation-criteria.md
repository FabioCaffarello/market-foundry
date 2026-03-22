# Support-Surface Lifecycle Signals And Consolidation Criteria

## Purpose

This document turns the C27 lifecycle strategy into practical signals and
criteria for repository support surfaces.

Use it to decide whether a command, script, doc, entrypoint, index, wrapper,
or helper should:

- remain active;
- be consolidated;
- be marked as legacy;
- be retired.

This document is the decision companion to
[`support-surface-sunset-consolidation-and-retirement-strategy.md`](support-surface-sunset-consolidation-and-retirement-strategy.md).

## Decision States

| Decision | Meaning | Typical response |
|---|---|---|
| remain | The surface still pays for itself | keep aligned and documented |
| consolidate | The need is real, but the current shape is too fragmented | merge, fold, demote, or parameterize |
| mark legacy | Compatibility still matters, but promotion should stop | label clearly and freeze scope |
| retire | Current value no longer justifies upkeep | remove and clean up entrypoints |

## Primary Signals

### 1. Usage signal

Ask:

- Is this surface used in current recurring repository work?
- Is the usage durable, or only historical?
- Does it serve a distinct expert/debug role, or only repeat a canonical path?

Interpretation:

- keep when usage is recurring and distinct;
- consolidate when usage is split across sibling surfaces;
- mark legacy when remaining usage is mostly migration or compatibility-driven;
- retire when meaningful current usage is absent.

### 2. Clarity signal

Ask:

- Would a contributor understand when to choose this surface?
- Is its role obvious from its name, help text, and docs?
- Does it compete with a nearby surface for the same first-choice role?

Interpretation:

- keep when the role is obvious;
- consolidate when ambiguity comes from overlapping peers;
- mark legacy when the role is historical but still needs explicit containment;
- retire when the surface mainly adds confusion.

### 3. Maintenance-cost signal

Ask:

- How many docs, wrappers, indexes, or checks must change to keep this surface
  accurate?
- Is the maintenance fan-out proportional to the value provided?
- Does the surface force repeated synchronization with nearby owners?

Interpretation:

- keep when upkeep is local and cheap;
- consolidate when the same change keeps touching several sibling surfaces;
- mark legacy when only compatibility-safe maintenance is still justified;
- retire when the maintenance burden clearly exceeds the differentiated value.

### 4. Discoverability signal

Ask:

- Does this surface help contributors find the right path faster?
- Is it indexed in the correct owning entrypoint?
- Is it making navigation clearer, or just broader?

Interpretation:

- keep when it materially improves navigation;
- consolidate when several surfaces teach the same navigation role;
- mark legacy when it must stay discoverable only as a containment bridge;
- retire when discovery value is negligible or misleading.

### 5. Responsibility-overlap signal

Ask:

- Which durable repository question does this surface answer?
- Do nearby surfaces answer the same question?
- Is the overlap intentional and differentiated, or accidental and costly?

Interpretation:

- keep when ownership is narrow and non-competing;
- consolidate when overlap is active and current;
- mark legacy when overlap is resolved but compatibility remains temporarily
  useful;
- retire when the ownership is already fully absorbed elsewhere.

### 6. Drift-risk signal

Ask:

- How likely is this surface to stop matching its neighbors?
- Has this kind of drift already happened?
- Would drift be silent and confusing, or obvious and harmless?

Interpretation:

- keep when alignment is easy;
- consolidate when multiple sibling surfaces repeatedly drift;
- mark legacy when a frozen compatibility posture is safer than active
  evolution;
- retire when drift risk remains high and the value is low.

## Criteria Matrix

| Surface type | Remain when | Consolidate when | Mark legacy when | Retire when |
|---|---|---|---|---|
| Make target | public, recurring, unambiguous | aliases or siblings crowd the same workflow | replacement exists but compatibility still helps | docs and workflows no longer depend on it |
| script | harness/debug role is distinct | sibling scripts differ mostly by narrow variants | retained only for historical callers or debugging continuity | no distinct harness value remains |
| wrapper | it shortens the public path materially | it only re-expresses a canonical path with small differences | compatibility wrapper still avoids abrupt breakage | it adds no meaningful route value |
| operations doc | owns one durable support rule | active docs overlap in topic or start-point role | rare; only if transitional references still matter | another active doc fully owns the rule |
| index/README | shortens navigation with clear ownership | multiple active maps teach the same route | almost never; better to simplify | it only mirrors a better index |
| lightweight check | protects a stable objective invariant | several checks can collapse into one better invariant | rarely useful | invariant is stale, subjective, or low-value |

## Sunset Signals

Treat these as signs that a support surface should enter sunset review:

- contributors ask which of two nearby paths is canonical;
- examples in active docs split between several equivalent entrypoints;
- the same update keeps requiring edits in many neighboring docs or wrappers;
- a helper survives mainly because it once existed, not because it still has a
  clear role;
- discoverability depends on repository memory rather than current entrypoints;
- the surface is only safe because contributors already know to ignore it;
- compatibility value is cited, but no current docs or workflows actually rely
  on it.

A sunset signal does not mean immediate removal. It means the surface should be
evaluated against the lifecycle states deliberately.

## Healthy Redundancy Tests

Redundancy is healthy only if all of these remain true:

1. the canonical owner is still obvious;
2. the secondary surface has a distinct job;
3. docs do not teach both as interchangeable defaults;
4. maintenance fan-out stays modest;
5. the secondary surface would be missed if removed.

If any of those fail, the redundancy is probably becoming expensive.

## Consolidation Moves

When consolidation is chosen, prefer these moves in order:

1. clarify one canonical owner and demote the others;
2. merge overlapping docs into one active canonical document;
3. turn a sibling script into a flag or mode on an existing script;
4. keep an alias but stop promoting it in curated entrypoints;
5. remove repeated summaries from root docs and keep the richer area index.

Consolidation should usually reduce maintenance fan-out immediately.

## Legacy Marking Rules

Mark a surface as legacy only when all are true:

- a better canonical replacement already exists;
- abrupt removal would still create avoidable confusion or breakage;
- the surface should stop evolving except for safety or compatibility needs;
- docs can point directly to the replacement.

Legacy is a temporary containment state, not a permanent category for neglected
surfaces.

## Retirement Rules

Retire a support surface when all are true:

- current recurring workflows no longer depend on it;
- active docs and curated entrypoints no longer need to promote it;
- the replacement path is already canonical and discoverable;
- keeping it would mainly preserve maintenance burden or ambiguity.

Retirement should normally be accompanied by:

- doc/help cleanup;
- index cleanup;
- guard-rail updates if they referenced the retired surface.

## Applying The Criteria In Market Foundry

Current examples of proportional lifecycle interpretation:

- `make smoke*` should remain active canonical because they own proof-of-record
  runtime validation;
- `make live*` should remain active auxiliary because they orchestrate bring-up
  but should not compete with proof ownership;
- direct `scripts/*.sh` invocation should remain auxiliary and debug-oriented
  when it preserves flags or harness maintenance value;
- compatibility-only CLI helper paths should stay explicitly legacy until their
  callers disappear;
- broad operational summary docs should be consolidated if they start repeating
  the same lifecycle or navigation rules.

## Review Questions

Before landing a lifecycle decision, answer:

1. Is the problem lack of value, or lack of differentiation?
2. Would clarifying ownership solve this without structural change?
3. If kept, what ongoing cost are we accepting?
4. If demoted to legacy, what replacement are we teaching?
5. If retired, what confusion or breakage remains?

## Related Documents

- [`support-surface-sunset-consolidation-and-retirement-strategy.md`](support-surface-sunset-consolidation-and-retirement-strategy.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
