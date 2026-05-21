# Stage C33 Report: Canonical Workflow Hotspot Assessment And Selection

## 1. Executive Summary

Stage C33 applied the C32 continuous-prioritization model to the current
`market-foundry` development platform and selected the highest-value real
hotspot for the next short Codex wave.

The selected hotspot is:

- canonical operational-proof taxonomy drift across `Makefile`, core workflow
  docs, and harness-governance docs.

This was chosen because it is a real recurring repository-platform weakness,
not another abstract governance gap. The repository's proof surface has grown,
but the canonical owner docs are no longer fully aligned with the actual
`smoke-*` surface.

That misalignment reduces trust in the canonical workflow, increases support
fan-out, and creates avoidable ambiguity exactly where future Foundry
expansion will continue to depend on reliable proof selection.

## 2. Scope Boundaries

### In scope

- review the current development platform through the C32 model;
- identify real recurring hotspots in `Makefile`, scripts, CLI boundaries, docs,
  entrypoints, and proofs/harnesses;
- compare hotspot candidates using the C32 criteria and buckets;
- select one primary hotspot and one reserve hotspot;
- document the selection as preparation for the next short applied wave.

### Out of scope

- changing functional system architecture;
- opening another broad governance-model stage;
- executing a wide refactor of workflow or proof surfaces in C33 itself;
- adding new proof families or new parallel front doors.

## 3. Hotspots Evaluated

The following recurring hotspot candidates were evaluated:

1. canonical operational-proof taxonomy drift;
2. bring-up and proof entrypoint overlap;
3. workflow-document fan-out for canonical command guidance;
4. lightweight guard-rail coverage gap for workflow alignment.

## 4. Evidence Used

The assessment used the live repository surfaces rather than historical
intuition:

- `Makefile`
- `README.md`
- `DEVELOPMENT.md`
- `scripts/README.md`
- `scripts/repository-consistency-check.sh`
- `docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`
- `docs/operations/makefile-targets-reference-and-conventions.md`
- `docs/operations/scripts-catalog-and-usage-guide.md`
- `docs/operations/smoke-and-operational-harness-governance.md`
- related repository-platform governance docs from C28 through C32

The main live mismatch observed was this:

- the real `Makefile` proof surface includes `make smoke-live-stack`,
  `make smoke-activation`, and `make smoke-composed`;
- the detailed script catalog reflects those commands;
- but the core lifecycle/governance docs still present a narrower proof
  inventory, which means the canonical workflow story is no longer fully
  synchronized with the actual command surface.

## 5. C32 Model Application

### Dominant value tested

The highest-value question was:
which hotspot most improves workflow trust, predictability, and maintenance
coherence right now?

### Bucket judgment

- canonical operational-proof taxonomy drift: structural improvement
- bring-up and proof overlap: structural improvement
- workflow-document fan-out: structural improvement
- guard-rail coverage gap: quick win after structural alignment

### Criteria judgment summary

The selected hotspot performed best on the C32 criteria that mattered most for
the next short wave:

- high impact on daily development;
- high friction reduction;
- high entropy-risk reduction;
- high predictability improvement;
- high discoverability gain;
- high environment reliability improvement;
- with only medium implementation cost.

## 6. Selected Primary Hotspot

### Hotspot

Canonical operational-proof taxonomy drift across `Makefile`, root workflow
docs, lifecycle/reference docs, and harness-governance docs.

### Why it wins

- It is already visible in the live repository surface.
- It targets a recurring high-traffic workflow, not a marginal support concern.
- It improves confidence in the canonical path directly.
- It reduces maintenance fan-out by tightening one operational owner story.
- It prepares the repository for future proof-surface growth without creating
  more ambiguity.

### Why it is not just a doc tidy-up

The problem is structural, not cosmetic.

The repository has one real operational-proof surface, but the canonical owner
docs have drifted behind that surface. That is a workflow-trust problem, not
just a wording problem.

## 7. Secondary Reserve Hotspot

Lightweight guard-rail coverage gap for workflow alignment.

This remains valuable, but it is secondary because the repository should first
tighten the proof taxonomy and only then codify the missing alignment check.

## 8. Lightweight Governance Adjustments Applied

Stage C33 added active documentation for:

- hotspot assessment and selection;
- hotspot prioritization rationale;
- stage-history traceability for the C33 decision.

These changes keep the selection discoverable inside the canonical operations
and stage indexes without opening a new governance surface.

## 9. Changes Applied

Created:

- `docs/operations/canonical-workflow-hotspot-assessment-and-selection.md`
- `docs/operations/hotspot-candidates-prioritization-and-selection-rationale.md`
- `docs/stages/stage-c33-canonical-workflow-hotspot-assessment-and-selection-report.md`

Updated lightly:

- `docs/operations/README.md`
- `docs/README.md`
- `docs/operations/repository-platform-governance-health-review-and-sustainability-model.md`
- `docs/stages/INDEX.md`

## 10. Preparation For C34

Recommended C34 shape:

1. treat the proof-surface alignment issue as a short structural-improvement
   wave;
2. align the canonical operational-proof taxonomy across the main owner docs;
3. reduce overlap between summary workflow docs and deeper proof catalogs;
4. add only the minimum lightweight check needed to keep that taxonomy aligned.

Guard rails for C34:

- do not expand into a broad workflow-doc rewrite;
- do not add new proof entrypoints unless a runtime need truly requires it;
- do not move into functional runtime architecture;
- do not turn the wave into another abstract governance exercise.

## 11. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C33 STAGE_SLUG=canonical-workflow-hotspot-assessment-and-selection STAGE_REQUIRE=docs/operations/canonical-workflow-hotspot-assessment-and-selection.md,docs/operations/hotspot-candidates-prioritization-and-selection-rationale.md`

## 12. Final Outcome

The repository now has a formal, evidence-based selection of the most valuable
real hotspot for the next short Codex wave.

C33 did what C32 explicitly asked for:

- it applied the prioritization model to a real recurring support hotspot;
- it selected a structural improvement rather than another abstract governance
  layer;
- and it prepared a focused next step that should improve the development
  platform concretely.
