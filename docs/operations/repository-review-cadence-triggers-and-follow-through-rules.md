# Repository Review Cadence Triggers And Follow-Through Rules

## Purpose

This document defines which signals should trigger repository-development-
environment review and how those reviews should produce proportionate actions.

It is the operational companion to
[`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md).

The repository should not escalate every small friction into governance work.
It should react when signals indicate recurring degradation, unclear ownership,
or rising support cost.

## Trigger Categories

### 1. Growth trigger

Use when a surface grows enough that contributors start needing explanation to
understand where to start or which option is canonical.

Common examples:

- new Make aliases or wrapper families;
- more scripts covering adjacent flows;
- additional active docs on the same support topic;
- CLI commands that look like sibling solutions.

What growth means here:

- not raw size alone;
- growth that increases ambiguity, overlap, or maintenance fan-out.

### 2. Drift trigger

Use when the support environment stops agreeing with itself.

Common examples:

- docs and wrappers teach different flows;
- active docs are missing from the owning README;
- area entrypoints lag behind the real tree;
- lightweight checks protect outdated assumptions.

This trigger is high value because drift often remains silent until it creates
trust loss.

### 3. Operational confusion trigger

Use when contributors can technically complete work but the path is harder to
understand than it should be.

Common examples:

- uncertainty about whether to start in `make`, scripts, or direct CLI;
- multiple docs answering the same workflow question;
- root docs or operations docs becoming too broad;
- stage support that requires remembering several disconnected documents.

This trigger matters even without breakage because confusion is a leading
indicator of discoverability decay.

### 4. Duplication trigger

Use when more than one active surface answers the same durable question.

Common examples:

- overlapping docs in `docs/operations/`;
- a script and a Make target both presented as the normal path;
- multiple indexes describing the same navigation concern;
- helper commands that differ more by naming than by capability.

Duplication should usually cause consolidation, not another summary surface.

### 5. Reliability trigger

Use when contributors start doubting whether canonical workflows, harnesses, or
checks can be trusted as described.

Common examples:

- wrappers pointing to stale behavior;
- guard rails failing for noisy or historical reasons;
- stage support that passes mechanically but leaves obvious ambiguity;
- proof-of-record boundaries becoming unclear.

Reliability drift is expensive because it drives contributors toward ad hoc
workarounds.

### 6. Discoverability trigger

Use when finding the right surface becomes materially slower or more dependent
on repository memory.

Common examples:

- active docs are technically present but poorly indexed;
- area READMEs stop reflecting real ownership;
- the docs stack no longer makes role separation obvious;
- the canonical path depends on knowing historical stage names.

Discoverability triggers should usually lead to navigation fixes, not new
parallel guides.

## Trigger Interpretation Rules

Interpret signals with these rules:

1. One-off friction is not automatically a cadence trigger.
2. Repeated friction on the same surface is a real signal even when each
   instance looks small.
3. A signal that crosses several surfaces is more important than a local
   nuisance.
4. Prefer signals tied to contributor behavior and workflow trust over abstract
   repository neatness.
5. If the corrective action would cost more than the drift it prevents, reduce
   scope.

## Action Ladder

The repository should answer triggers with the smallest action that restores
clarity or trust.

### Level 1. Clarify

Use when the owning surface is correct and the problem is mainly readability or
discoverability.

Typical actions:

- tighten wording in a canonical doc;
- add or fix one README/index link;
- make the canonical-vs-auxiliary boundary explicit;
- improve `make docs` or help text curation.

### Level 2. Align

Use when related surfaces have drifted apart but should still exist.

Typical actions:

- reconcile doc wording with wrapper behavior;
- update a script catalog after workflow changes;
- bring an area README back in line with the physical tree;
- update guard-rail assumptions to current active policy.

### Level 3. Consolidate

Use when the signal shows duplication or crowded ownership.

Typical actions:

- merge overlapping docs;
- demote one helper surface;
- route users back to the canonical `make` entrypoint;
- retire low-value aliases or stale support artifacts.

### Level 4. Guard

Use when the drift is objective, likely to recur, and cheap to detect.

Typical actions:

- extend a required-doc list;
- require cross-linking between paired canonical docs;
- validate index presence for active docs;
- protect wrapper-to-script alignment.

Do not use Guard for subjective preference or for rules that still change
frequently.

### Level 5. Governed follow-up

Use only when repeated signals show that local fixes are no longer enough.

Typical actions:

- open a small support stage focused on one hotspot;
- define a narrow consolidation or taxonomy correction;
- add one active canonical doc when there is a durable gap with no clean owner.

This is the highest response level and should stay rare.

## Proportionality Rules

Apply these rules before escalating:

- prefer clarification before consolidation when ownership is already clear;
- prefer consolidation before adding a new surface;
- prefer documentation before automation when the invariant is editorial;
- prefer one hotspot pass before a broad support cleanup stage;
- prefer stage work only when several small fixes need coordinated closure.

If a review outcome reads like a general repository-improvement program, it is
too large for this cadence model.

## Trigger-To-Action Mapping

| Trigger | Default first action | Escalate when |
|---|---|---|
| Growth | clarify or consolidate | growth creates overlapping ownership or repeated maintenance fan-out |
| Drift | align | the same drift keeps returning and becomes cheap to guard |
| Operational confusion | clarify | confusion survives one clarification because ownership itself is crowded |
| Duplication | consolidate | consolidation reveals a durable missing owner concern |
| Reliability | align | objective silent failure deserves a lightweight guard |
| Discoverability | clarify | navigation remains weak because indexes or entrypoints are structurally overlapping |

## What Reviews Should Produce

Every periodic review should produce one of these outcomes:

- no action, because the signal is isolated or not durable;
- one local correction, because the surface is healthy after a small fix;
- one hotspot consolidation, because the signal is recurring and cross-surface;
- one guarded invariant, because silent drift is objective and cheap to catch;
- one narrowly-scoped support follow-up stage, because the hotspot exceeds a
  local fix.

Reviews should not produce:

- recurring status meetings;
- broad scorecards;
- permanent review committees;
- mandatory escalations for minor edits.

## Review Record Expectations

Most review outcomes should be recorded by the repository change itself:

- doc clarification in the canonical owner;
- index update in the owning README;
- small guard-rail extension;
- narrow stage report when a governed support follow-up is justified.

The repository does not need a separate recurring review log.

## Sustainability Outcome

This trigger model keeps the environment sustainable by making degradation
visible in practical terms:

- confusion becomes a discoverability or coherence signal;
- overlap becomes a consolidation signal;
- mistrust becomes a reliability signal;
- repeated silent drift becomes a guard-rail candidate;
- broader recurring friction becomes a bounded governance-stage candidate.

That keeps repository review strategic and operational at the same time, while
avoiding heavy process.

## Related Documents

- [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
