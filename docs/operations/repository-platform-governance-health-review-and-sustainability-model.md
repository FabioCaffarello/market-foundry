# Repository Platform Governance, Health, Review, And Sustainability Model

## Purpose

This document translates the C28 strategic operating model into a practical
governance model for operating the repository as the development platform of
`market-foundry`.

Use it when deciding:

- which repository-platform layer owns a change;
- which reviews should happen continuously or periodically;
- which criteria justify adding, consolidating, guarding, or retiring a support
  surface;
- which follow-through is proportionate to the observed drift.

## Governance Objective

Keep the repository platform light, governable, and durable by combining:

- health interpretation;
- recurring review cadence;
- support-surface lifecycle discipline;
- tooling and documentation sustainability;
- minimal protection against silent entropy.

## Applied Governance Model

### 1. Canonical platform contract

Protect these as the stable operating contract:

- `make` public workflows;
- root/documentation entrypoint stack;
- `scripts/` as harness implementation;
- `raccoon-cli` as structural-analysis and governance tooling;
- operations/tooling/architecture/stage ownership boundaries;
- proof-of-record and lifecycle rules already established in earlier waves.

### 2. Health interpretation

Use C25 to judge where the platform is drifting:

| Dimension | Main question |
|---|---|
| discoverability | can contributors still find the right start point quickly? |
| operational reliability | do canonical workflows still work as documented? |
| entrypoint coherence | are several surfaces claiming the same job? |
| navigability | can contributors reach the right repository area without guesswork? |
| documentation governance | do active rules live in active canonical docs? |
| tooling sustainability | are helper surfaces still bounded and intentional? |
| maintenance cost control | does the change reduce or expand edit fan-out? |

### 3. Review rhythm

Apply three review layers:

| Review layer | When | Output |
|---|---|---|
| local continuous review | any support-surface change | small alignment or clarification |
| periodic hotspot review | recurring triggers or support-heavy wave changes | consolidation, guard, or scoped follow-up |
| strategic platform review | multiple recurring signals at once | next-wave recommendation for highest-value hotspot |

### 4. Follow-through ladder

Use the smallest durable response:

1. clarify;
2. align;
3. consolidate;
4. guard;
5. governed follow-up.

Only move upward when the lower response no longer contains the drift.

## Decision Criteria By Surface

### `Makefile`

Keep or extend a target when it strengthens the public workflow contract.

Review questions:

- is this a recurring repository workflow;
- is `make` the correct public owner;
- does it reduce script-first behavior;
- does it create overlap with an existing target or alias;
- does the new target improve discovery more than it increases upkeep.

### Scripts and harnesses

Keep scripts behind `make` unless expert/debug depth is the main value.

Review questions:

- is the script implementation-facing or becoming a public workflow by accident;
- does a sibling script already solve nearly the same problem;
- can the existing harness absorb the behavior more cheaply than forking;
- is direct invocation being taught as normal use when it should stay expert-only.

### `raccoon-cli`

Keep the CLI in the governance and inspection lane.

Review questions:

- is the behavior structural-analysis oriented rather than runtime-operation oriented;
- can the need be solved by an existing command, flag, or grouped subcommand;
- is the proposed command `stable core`, `stable utility`, or only `legacy`;
- does the addition reduce ambiguity or merely avoid documentation work.

### Documentation and indexes

Treat docs as durable operating surfaces, not passive notes.

Review questions:

- is the topic active and durable;
- does a canonical owner already exist;
- has the active doc been indexed in the owning README;
- is a stage report being used as current guidance by accident;
- will this change force repeated root-doc churn without improving orientation.

### Lightweight checks

Admit checks only for objective, cheap, silent-drift invariants.

Review questions:

- is the invariant stable enough to codify;
- is the failure locally actionable;
- would the drift otherwise stay hidden until it becomes expensive;
- is the check protecting active policy rather than history volume.

### Automation

Use automation only for recurring, transparent, low-judgment routines.

Review questions:

- is the routine frequent and omission-prone;
- is the output legible enough to guide the next step;
- does it fit behind existing `make` or script surfaces;
- does it preserve manual judgment where interpretation still matters.

## Lifecycle Governance

Use C27 states explicitly:

| State | Meaning | Expected treatment |
|---|---|---|
| active canonical | public primary answer | preserve and keep discoverable |
| active auxiliary | helpful but not primary | keep differentiated and bounded |
| legacy | compatibility-only or frozen | stop promoting, keep labeled |
| retired | no longer worth keeping | remove after replacement path is clear |

Lifecycle rule:
when two active surfaces answer the same recurring question, consolidation is
the default response.

## Sustainability Rules

### Promotion rule

If a stage discovers a lasting rule:

1. promote it into the correct canonical doc;
2. link it from the owning index;
3. keep the stage report as rationale only;
4. guard it only if the invariant is objective and drift-prone.

### Index rule

Every active operations or tooling document must appear in its owning README.

### Root-doc rule

Root docs remain curated entrypoints, not exhaustive support catalogs.

### Cost rule

Prefer a change that updates one behavior surface plus one owner doc over a
change that creates broad edit fan-out.

### Review rule

Periodic review exists to spot real entropy patterns, not to create standing
ceremony.

## Lightweight Protection Set

The repository should rely on a narrow set of protections:

- canonical entrypoint documents;
- area README indexes;
- repository consistency checks;
- stage-status and stage-check for continuity and closure hygiene;
- CLI lifecycle and support-surface lifecycle labeling in docs;
- proportional documentation maintenance triggers.

Anything heavier requires strong evidence that the current lightweight model is
failing.

## Practical Operating Loop

When changing the repository platform, use this loop:

1. identify the affected support surface;
2. classify the problem through the health dimensions;
3. choose the owner surface;
4. apply the smallest valid follow-through;
5. update the owning index and stage evidence if required;
6. run the narrowest validation that protects the platform contract.

Typical validation set:

- `make repo-consistency-check`;
- `make stage-check` for the active stage;
- `make check` or `make verify` when the change touches public workflow or
  tooling behavior.

## Long-Term Sustainability Outcome

This model is succeeding when:

- the repository increasingly behaves like one coherent development platform;
- support-focused waves unify previous artifacts instead of accumulating new
  fragments;
- contributors can see which surfaces are canonical, reviewed periodically, or
  intentionally flexible;
- maintenance effort goes primarily into useful surfaces, not into catalog
  sprawl or compatibility ambiguity.

## Relationship To C28 Strategic Model

The strategic operating model defines the platform posture and long-term
contract. This document defines how to govern it in day-to-day and wave-level
practice.

Canonical strategic entrypoint:

- [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md)

## Related Documents

- [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`support-surface-lifecycle-signals-and-consolidation-criteria.md`](support-surface-lifecycle-signals-and-consolidation-criteria.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
