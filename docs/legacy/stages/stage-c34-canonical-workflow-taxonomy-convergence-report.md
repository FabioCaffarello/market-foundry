# Stage C34 Report: Canonical Workflow Taxonomy Convergence

## Summary

Stage C34 tightened the canonical workflow and operational-proof taxonomy so
the owner docs now describe the same proof surface that the `Makefile`
actually exposes.

The stage stayed intentionally narrow:

- align the proof taxonomy in the owner docs;
- reduce ambiguity about which surfaces are exhaustive versus curated;
- add one lightweight guard rail to catch future proof-taxonomy drift;
- correct one direct-script claim that referenced a nonexistent Make target.

## Diagnosis

The repository had one real proof surface and multiple partially overlapping
descriptions of it.

The concrete drift was:

- `Makefile` exposes `make smoke-round-trip`, `make smoke-live-stack`,
  `make smoke-activation`, and `make smoke-composed`;
- the detailed script catalog already reflected those targets;
- but several owner docs still described a narrower proof family;
- one direct smoke script claimed a canonical `make smoke-venue` entrypoint
  that does not exist.

That combination reduced predictability and trust because the compact docs that
should answer “which proof exists and when should I use it?” were less current
than the implementation-facing surfaces.

## Scope Boundaries

### In scope

- compare the real `Makefile` smoke surface against the owner docs;
- converge the canonical smoke/proof taxonomy in the owner docs;
- add only the minimum lightweight check needed to keep those docs aligned;
- record the result in stage history.

### Out of scope

- adding new Make targets;
- changing runtime architecture or smoke semantics;
- rewriting the broader documentation system;
- promoting direct expert scripts into new public workflow surfaces.

## Decisions

### Decision 1. Keep `README.md` curated, not exhaustive

The root entrypoints remain intentionally short and discoverability-oriented.
Exhaustive proof taxonomy belongs in the operations owner docs, not in every
top-level summary surface.

### Decision 2. Treat four operations docs as the proof-taxonomy owner set

The canonical proof inventory now has a clear owner set:

- `docs/operations/developer-workflow-unification.md`
- `docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`
- `docs/operations/makefile-targets-reference-and-conventions.md`
- `docs/operations/operational-proof-entrypoints-and-ownership.md`
- `docs/operations/smoke-and-operational-harness-governance.md`

`docs/operations/smoke-ux-and-proof-execution-ergonomics.md` was also updated
because its “Current Public Surface” table must not present a stale subset as
if it were the whole surface.

### Decision 3. Guard the taxonomy at the repo-consistency layer

The lightweight repository check now verifies that canonical smoke-taxonomy
docs mention every real `smoke*` Make target and that smoke scripts do not
claim nonexistent Make entrypoints.

This is the smallest check that protects workflow trust without creating a new
heavier governance mechanism.

## Changes Applied

- aligned specialized smoke targets across the owner docs:
  `smoke-round-trip`, `smoke-live-stack`, `smoke-activation`,
  `smoke-composed`;
- added explicit links from the workflow summary doc to the detailed proof
  inventory/ownership docs;
- corrected `scripts/smoke-venue-integration.sh` so it no longer claims a
  nonexistent canonical Make target;
- extended `scripts/repository-consistency-check.sh` with:
  - canonical smoke-taxonomy alignment coverage;
  - smoke-script claimed-entrypoint validation.

## Non-Goals And Limits

- no attempt was made to make every workflow doc exhaustive;
- no smoke script was rewritten for style or semantics;
- no new wrapper was added for the venue-integration script because there is no
  material need to expand the public command surface for this stage;
- no broader documentation consolidation was attempted beyond the proof-taxonomy
  hotspot selected in C33.

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C34 STAGE_SLUG=canonical-workflow-taxonomy-convergence STAGE_REQUIRE=docs/stages/stage-c34-canonical-workflow-taxonomy-convergence-report.md`

## Preparation For Next Stage

1. Keep future proof-surface changes inside the existing `smoke*` family unless
   a strong exception is justified by C31 rules.
2. Use the new repo-consistency checks as the default early warning before more
   drift accumulates across owner docs.
3. If workflow fan-out becomes costly again, prefer narrowing or cross-linking
   owner docs before creating new summary surfaces.
