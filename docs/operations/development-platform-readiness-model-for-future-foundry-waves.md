# Development Platform Readiness Model For Future Foundry Waves

## Purpose

This document defines how `market-foundry` should judge whether the repository
is ready to absorb a new wave from the perspective of the development platform.

This is not product readiness, domain readiness, or runtime readiness. It is a
repository-platform readiness model focused on whether contributors can extend
the Foundry safely without degrading workflow trust, support-surface coherence,
or governance discipline.

Use this document before opening a new wave that is likely to add code,
tooling, docs, workflow entrypoints, stage support, or other durable support
surface load.

## Readiness Boundary

### Development-platform readiness means:

- the canonical workflow remains predictable;
- public entrypoints still have clear ownership;
- active docs remain current and navigable;
- lightweight guard rails are trusted and actionable;
- support-surface growth stays bounded enough to avoid drift;
- stage/proof/governance discipline remains usable during expansion.

### Development-platform readiness does not mean:

- the product is feature-complete;
- the runtime is production-ready;
- every functional backlog item is closed;
- every architectural debt is resolved;
- every future support concern is already automated.

The C30 question is narrower:
can the repository platform absorb another wave without making normal
development meaningfully harder, noisier, or less governable?

## Current Diagnosis

The repository starts C30 from a relatively strong platform baseline:

- `make` is already the canonical workflow surface;
- root docs and operations indexes form a real entrypoint stack;
- `scripts/` remains a harness layer rather than the public default;
- `raccoon-cli` is bounded as structural-analysis and governance tooling;
- `make repo-consistency-check`, `make stage-status`, and `make stage-check`
  already provide lightweight platform discipline;
- C25 through C29 already define health, review, lifecycle, operating model,
  and strategic checkpoints.

The remaining gap was practical readiness language for opening future waves.
The repository already knows how to observe health and how to trigger a
checkpoint, but it still needed an explicit model that answers:

- when the development platform is ready for wave expansion;
- which signals mean "safe to proceed";
- which signals mean "stabilize first";
- which correction is proportionate before opening more surface area.

## Readiness Dimensions

Use seven dimensions for development-platform readiness.

### 1. Workflow Predictability

Core question:
can contributors still execute the normal engineering loop without repository
memory or ad hoc branching?

Healthy when:

- `make bootstrap`, `make check`, `make tdd`, and `make verify` still match
  documented use;
- proof-of-record flows remain clear;
- support-heavy waves do not create alternate public workflows by accident.

### 2. Entrypoint Reliability

Core question:
do the canonical entrypoints still work as promised?

Healthy when:

- root docs, `make`, wrappers, and support docs agree on the same path;
- bootstrap and lightweight checks fail on actionable drift rather than noise;
- stage and proof helpers remain usable without manual repair.

### 3. Documentation Clarity

Core question:
can contributors still find the current rule or owner doc quickly?

Healthy when:

- active guidance lives in active docs rather than in stage history;
- README indexes remain current;
- new platform rules are promoted into canonical docs in the same wave that
  depends on them.

### 4. Tooling And CLI Trust

Core question:
does tooling still feel bounded, intentional, and safe to rely on?

Healthy when:

- `raccoon-cli` stays in the governance/inspection lane;
- Make wrappers still map to real supported behavior;
- helper surfaces are extended only when an existing owner cannot absorb the
  need cheaply.

### 5. Governance Of Proofs And Stages

Core question:
can the repository still prove and narrate waves coherently?

Healthy when:

- proof entrypoints remain explicit and non-overlapping;
- stage reports remain historical evidence, not accidental current policy;
- stage continuity and closure surfaces still keep wave evidence aligned.

### 6. Maintenance Cost And Structural Load

Core question:
is repository change still cheap enough to govern?

Healthy when:

- support changes usually touch one owner doc and one index, not many mirrored
  surfaces;
- new docs or wrappers remove ambiguity faster than they create upkeep;
- drift is corrected mainly by consolidation rather than by layering.

### 7. Expansion Capacity Without Drift

Core question:
can the repository absorb a new surface or broader wave without losing shape?

Healthy when:

- canonical owners are not already crowded or ambiguous;
- known hotspots are bounded enough that expansion will not amplify them;
- new wave scope can be attached to existing platform contracts instead of
  inventing new support structure.

## Readiness Classes

Use three practical classes.

### Ready

Meaning:
the platform can absorb the next wave with normal maintenance and no platform
prerequisite beyond ordinary closure hygiene.

Typical shape:

- canonical workflow is stable;
- no critical hotspot is repeating;
- support-surface growth remains proportionate;
- new wave scope fits inside existing owners and docs.

### Conditionally Ready

Meaning:
the platform can absorb the next wave only after one local correction or one
small support-focused prerequisite.

Typical shape:

- the repository is broadly healthy;
- one known hotspot would become riskier if ignored;
- the required correction is clear, bounded, and low-cost.

### Not Ready

Meaning:
opening a new wave now would likely amplify platform drift, trust erosion, or
support-surface sprawl.

Typical shape:

- more than one readiness dimension is degrading together;
- canonical ownership is already ambiguous;
- guard rails or docs are no longer trusted enough to support expansion;
- the next wave would force platform decisions that are currently unresolved.

## Ready-To-Open Signals

Treat these as positive readiness signals.

- Contributors still have one obvious workflow path from root docs into `make`.
- `make check`, `make verify`, and the relevant proof helpers remain credible
  enough to be the default validation story.
- New durable support rules can be promoted into an existing canonical doc.
- `docs/operations/README.md`, `docs/README.md`, and `make docs` remain
  readable entrypoints instead of sprawling catalogs.
- `raccoon-cli`, scripts, and Make wrappers still have visibly different roles.
- Stage continuity and stage closure can be maintained without manual
  archaeology.
- The next wave adds mostly domain/product surface while platform overhead
  stays bounded.

## Not-Yet Signals

Treat these as "do not open yet" signals.

- Contributors must choose between multiple public entrypoints for the same
  recurring task.
- Important current guidance is living in stage reports instead of active docs.
- Lightweight checks are mistrusted because they are noisy, stale, or unclear.
- Support-heavy changes require repeated edits across root docs, indexes,
  scripts, and wrappers just to stay aligned.
- A known hotspot around docs, CLI, stage governance, or entrypoints is
  already repeating and the next wave would add more load to it.
- The proposed wave implicitly requires a new support surface, but its owner is
  still unclear.
- The repository can still ship domain changes, but the platform can no longer
  explain or validate them cleanly.

## Saturation Interpretation

Development-platform saturation does not mean the repository is "full." It
means the current support model is close to losing coherence under additional
load.

The practical saturation pattern is usually one of these:

- too many sibling surfaces answering the same question;
- too much edit fan-out to keep support docs aligned;
- too much trust erosion in checks or CLI behavior;
- too much stage/proof governance friction for normal wave closure.

When saturation signs appear, the default response is not to stop growth
indefinitely. The default response is to apply the smallest stabilizing
correction before opening more surface area.

## Decision Model Before Opening A Wave

Use this sequence:

1. confirm that the question is development-platform readiness, not product
   readiness;
2. identify whether the next wave adds platform load directly or indirectly;
3. scan the seven readiness dimensions quickly;
4. classify the repository as ready, conditionally ready, or not ready;
5. if not fully ready, choose the smallest prerequisite correction;
6. only open a support-focused follow-up stage when local correction is no
   longer enough.

## Default Prerequisite Types

When the platform is not fully ready, prefer one of these prerequisite moves:

- clarify one canonical doc or owner boundary;
- align one entrypoint stack inconsistency;
- consolidate one overlapping support surface;
- guard one objective silent-drift invariant;
- open one small support-focused wave only when the hotspot exceeds local fix
  scope.

## Relationship To Existing Governance

This readiness model does not replace earlier governance layers.

- C25 explains how to interpret repository health.
- C26 explains recurring review triggers and proportional follow-through.
- C27 explains lifecycle discipline for support surfaces.
- C28 explains the repository-platform operating contract.
- C29 explains when strategic checkpoints should happen.

C30 adds the final operational question:
is the development platform itself ready to absorb the next wave, and if not,
what is the smallest correction that should come first?

## Related Documents

- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
- [`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md)
- [`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md)
- [`readiness-signals-saturation-signals-and-wave-opening-rules.md`](readiness-signals-saturation-signals-and-wave-opening-rules.md)
- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
