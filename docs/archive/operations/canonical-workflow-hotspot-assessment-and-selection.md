# Canonical Workflow Hotspot Assessment And Selection

## Purpose

This document applies the C32 continuous-prioritization model to the current
`market-foundry` development platform and selects the highest-value real hotspot
for the next short Codex wave.

The intent is not to open another abstract governance tranche. The intent is to
choose one recurring repository-platform problem whose correction would:

- increase trust in the canonical workflow;
- reduce maintenance fan-out across support surfaces;
- and improve readiness for the next Foundry expansion.

## C33 Framing

### Problem being solved

The repository already has a defined canonical workflow, but the operational
proof surface has expanded faster than the canonical owner docs and guard rails
that are supposed to keep it legible.

### Affected surfaces

- `Makefile`
- `scripts/*.sh`
- root workflow docs
- `docs/operations/` workflow and proof docs
- lightweight repository governance checks

### Change shape

This is an assessment-and-selection stage. It does not perform the structural
improvement itself. It identifies the best target for the next short wave.

## Repository Signals Reviewed

The current repository shows five recurring signals that matter for hotspot
selection:

1. The operational proof surface is now materially broader than the original
   baseline.
2. Proof-related guidance is spread across several active workflow documents.
3. Some newer proof entrypoints are present in `Makefile` and script docs but
   not reflected consistently in the core lifecycle/governance docs.
4. The lightweight repository check protects indexing, link validity, and
   wrapper existence, but it does not fully protect canonical proof-taxonomy
   alignment across the main workflow docs.
5. The next wave should improve the repository as a development platform, not
   open another broad governance abstraction.

## Hotspot Candidates

### Candidate A. Canonical operational-proof taxonomy drift

Problem:
the public proof surface has grown, but the canonical owner docs no longer tell
 the same story with the same level of completeness.

Observed signals:

- `Makefile` exposes `make smoke-live-stack`, `make smoke-activation`, and
  `make smoke-composed`.
- `docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`
  still inventories `make smoke-live-stack` but omits `make smoke-activation`
  and `make smoke-composed`.
- `docs/operations/makefile-targets-reference-and-conventions.md` lists
  `make smoke-live-stack` but omits `make smoke-activation` and
  `make smoke-composed`.
- `docs/operations/smoke-and-operational-harness-governance.md` still presents
  a narrower canonical smoke family than the one now present in `Makefile`.
- `docs/operations/scripts-catalog-and-usage-guide.md` does include the newer
  proof targets, which means the detailed catalog is ahead of the canonical
  lifecycle/governance narrative.

Why it matters:

- contributors can see the commands in `make help`, but cannot rely on a single
  compact canonical doc set to understand which specialized proofs are active,
  why they exist, and how they relate to the proof-of-record model;
- support-surface trust drops when detailed docs are more current than the
  canonical summary docs that are supposed to own the workflow story.

### Candidate B. Bring-up and proof entrypoint overlap

Problem:
the repository has valid distinctions between `make live*`, `make up` +
`make seed*`, and `make smoke*`, but the relationship still depends on repeated
explanation across several documents.

Observed signals:

- the hierarchy is stated in `README.md`, `DEVELOPMENT.md`, lifecycle docs,
  harness governance docs, and the script catalog;
- the distinction is sound, but it relies on repeated wording rather than one
  compact operational owner surface plus stronger alignment checks.

Why it matters:

- repeated explanation creates upkeep cost;
- if the wording drifts, `make live*` can start looking like a competing proof
  surface again.

### Candidate C. Workflow-document fan-out for canonical command guidance

Problem:
the same operational workflow story is distributed across root docs plus many
active operations docs.

Observed signals:

- `README.md`, `DEVELOPMENT.md`, `docs/operations/README.md`,
  `developer-workflow-unification.md`,
  `development-lifecycle-entrypoints-and-canonical-flows.md`,
  `makefile-targets-reference-and-conventions.md`,
  `scripts-catalog-and-usage-guide.md`,
  `smoke-and-operational-harness-governance.md`,
  and other support docs all participate in the same workflow narrative.

Why it matters:

- fan-out raises maintenance cost;
- but much of this duplication is intentionally layered for navigation rather
  than a direct defect in the workflow itself.

### Candidate D. Lightweight guard-rail coverage gap for workflow alignment

Problem:
the repository already checks for index coverage, link validity, and script
wrapper mapping, but not for full alignment of the active canonical proof
taxonomy.

Observed signals:

- `scripts/repository-consistency-check.sh` validates that canonical docs point
  only to real Make targets and that Makefile script wrappers resolve to real
  scripts;
- it does not currently detect that the canonical proof inventory in the main
  lifecycle/governance docs can lag behind the real `smoke-*` surface.

Why it matters:

- this lets workflow drift survive until a human notices a mismatch between
  `Makefile` and the main owner docs.

## Selection

### Primary hotspot

Select Candidate A:
canonical operational-proof taxonomy drift across `Makefile`, core workflow
docs, and harness governance docs.

### Why this is the best next short wave

It is the strongest fit with the C32 recommendation for a short structural
improvement because it:

- addresses a real recurring repository hotspot rather than a governance idea;
- improves confidence in the canonical workflow directly;
- reduces fan-out by giving the proof surface one better-aligned owner story;
- creates a clean follow-on opportunity to add only the smallest missing guard
  rail after the taxonomy is tightened;
- prepares the repository for future expansion by making specialized proofs
  easier to find, classify, and maintain without creating more front doors.

This hotspot is also better than a pure documentation-cleanup wave because the
underlying problem is structural: support-surface growth is outpacing the
alignment layer that keeps canonical workflow guidance trustworthy.

### Secondary reserve hotspot

Candidate D is the reserve:
lightweight guard-rail coverage gap for workflow alignment.

Reason:
it is valuable, but it should follow the structural correction of the proof
taxonomy rather than lead it. Guarding an incoherent surface too early would
just encode current drift.

## Operational Interpretation For C34

The next short wave should focus on one bounded structural theme:

- consolidate and re-align the canonical operational-proof taxonomy;
- make one compact owner surface authoritative for specialized proof selection;
- reduce repeated explanation across layered docs;
- and add only the minimum lightweight check needed to keep the aligned surface
  from drifting again.

## Non-Goals

- no new proof families in C33;
- no broad rewrite of runtime architecture;
- no conversion of the repository into another governance-heavy tranche;
- no wide refactor of every workflow doc in advance of selecting the hotspot.

## Related Documents

- [`continuous-prioritization-model-for-the-development-platform.md`](continuous-prioritization-model-for-the-development-platform.md)
- [`prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md`](prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md)
- [`hotspot-candidates-prioritization-and-selection-rationale.md`](hotspot-candidates-prioritization-and-selection-rationale.md)
- [`../stages/stage-c33-canonical-workflow-hotspot-assessment-and-selection-report.md`](../stages/stage-c33-canonical-workflow-hotspot-assessment-and-selection-report.md)
