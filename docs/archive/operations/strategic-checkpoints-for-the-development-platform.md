# Strategic Checkpoints For The Development Platform

## Purpose

This document defines the lightweight strategic-checkpoint model for operating
the `market-foundry` repository as the Foundry development platform.

The goal is executive clarity, not ceremony. Strategic checkpoints should help
contributors decide when the development platform needs a short review, what
must be evaluated, and what kind of decision is proportionate.

## Why Strategic Checkpoints Exist

Earlier support-platform waves already established:

- repository health dimensions;
- periodic review cadence and trigger rules;
- support-surface lifecycle discipline;
- structural-cost control;
- a unified strategic operating model for the repository platform.

What remained implicit was the executive checkpoint layer:

- when the repository most needs a platform-level pause;
- which dimensions should always be revisited at that moment;
- how to keep the response light unless evidence shows a deeper hotspot.

Strategic checkpoints close that gap.

## Design Rules

Strategic checkpoints must stay:

- light enough to fit normal wave closure or support-heavy follow-through;
- tied to real repository moments, not arbitrary calendar ritual;
- grounded in canonical surfaces already used by contributors;
- oriented to one proportional decision, not to exhaustive auditing;
- narrow enough to preserve focus on platform health rather than runtime
  feature design.

Strategic checkpoints must not become:

- recurring audits without a trigger;
- scorecards or dashboards;
- standing governance meetings;
- a second workflow layered on top of the normal development loop.

## Executive Dimensions Revisited At Every Checkpoint

Use these dimensions together as the executive lens.

| Dimension | Executive question |
|---|---|
| repository health | does the platform still feel discoverable, reliable, and governable as a whole? |
| CLI reliability as a dev tool | can `raccoon-cli` still be trusted as a bounded structural-analysis surface without becoming a parallel workflow entrypoint? |
| entrypoint coherence | do `README.md`, `DEVELOPMENT.md`, `make`, docs indexes, scripts, and CLI still point contributors toward one obvious path? |
| docs and stages governance | are active rules living in active docs, with stage reports remaining historical evidence rather than accidental source of truth? |
| structural cost | is support-surface change increasing edit fan-out, overlap, or maintenance burden faster than value? |
| workflow sustainability | can contributors still execute the normal loop and support workflows without repository memory or ad hoc recovery paths? |

These dimensions are not scored. They are revisited to identify which one
currently deserves the next corrective decision.

## Checkpoints Defined

The repository should use four natural strategic checkpoints.

### 1. Wave-Closure Checkpoint

Use when a support-heavy Codex wave or repository-governance stage is closing.

Evaluate:

- whether the new support surfaces stayed inside the existing operating model;
- whether any newly promoted doc, wrapper, or helper created overlap;
- whether the change reduced or increased structural cost;
- whether future contributors still have one obvious canonical path.

Expected output:
a short closure judgment plus one recommendation: keep as-is, tighten
alignment, consolidate one hotspot, or queue one narrow follow-up stage.

### 2. Pre-Expansion Checkpoint

Use before adding a new durable support surface such as:

- a new public `make` workflow family;
- a new recurring operational document;
- a new `raccoon-cli` command family;
- a new stage-support or automation surface.

Evaluate:

- whether an existing owner surface can absorb the need;
- whether the problem is actually discoverability, alignment, or lifecycle
  drift;
- whether the addition removes more cost than it creates.

Expected output:
extend current owner, reject, or admit a narrowly-scoped new surface with an
explicit owner.

### 3. Hotspot Checkpoint

Use when recurring friction clusters around one platform hotspot:

- CLI trust or overlap;
- entrypoint confusion;
- script sprawl;
- docs governance drift;
- repeated support-surface lifecycle uncertainty.

Evaluate:

- which health dimensions are degrading together;
- whether the hotspot is local or cross-surface;
- whether the smallest durable response is clarify, align, consolidate, or
  guard.

Expected output:
one hotspot-level decision, not a broad repository-improvement program.

### 4. Readiness-For-Next-Wave Checkpoint

Use when the repository is about to support a new wave that is likely to add
tooling, docs, workflows, CLI capability, or support surfaces.

Evaluate:

- whether the current platform is healthy enough to absorb more change;
- whether a known hotspot should be corrected before expansion;
- whether the next wave should stay functional, support-focused, or mixed;
- whether any canonical entrypoint or governance boundary would become too
  crowded if expansion continues unchanged.

Expected output:
proceed, proceed with one prerequisite correction, or open one small platform
follow-up before broader expansion.

## Lightweight vs Deeper Checkpoints

Most strategic checkpoints should stay lightweight.

Use a lightweight checkpoint when:

- one wave closes with support-surface changes;
- a proposed addition is still narrow;
- a hotspot is visible but likely containable with one corrective action;
- the repository needs a quick readiness judgment before the next wave.

Escalate to a deeper strategic review only when:

- multiple health dimensions are degrading at the same time;
- the same trigger returns after one local correction;
- several active surfaces now overlap on the same durable question;
- the next platform decision would otherwise be guesswork.

The deeper review is still time-boxed and hotspot-oriented. It is not an open
ended audit.

## What A Checkpoint Must Produce

Every strategic checkpoint should produce one of these outcomes:

- no action, because the platform remains coherent;
- one local alignment or clarification;
- one consolidation or lifecycle correction;
- one lightweight guard for an objective silent-drift invariant;
- one narrowly-scoped follow-up stage for a proven hotspot.

Checkpoint rule:
produce one decision with one owner whenever possible.

## Decision Orientation

A good strategic checkpoint asks:

1. what changed or which recurring friction triggered the checkpoint;
2. which executive dimensions are actually affected;
3. which current owner surface should answer the problem;
4. whether the smallest valid response is enough;
5. whether the proposed response strengthens the repository as a development
   platform rather than adding another support burden.

Use the companion document
[`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md)
for the trigger interpretation, scope boundaries, and response ladder.

## Relationship To Existing Governance

Strategic checkpoints extend the existing model rather than replacing it:

- C25 supplies the repository-health dimensions;
- C26 supplies periodic review triggers and proportional follow-through;
- C27 supplies support-surface lifecycle discipline;
- C28 supplies the strategic operating model and applied governance model.

C29 adds the executive checkpoint layer that says when those tools should be
used together to guide the next platform decision.

## Related Documents

- [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md)
- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
- [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`support-surface-sunset-consolidation-and-retirement-strategy.md`](support-surface-sunset-consolidation-and-retirement-strategy.md)
- [`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md)
