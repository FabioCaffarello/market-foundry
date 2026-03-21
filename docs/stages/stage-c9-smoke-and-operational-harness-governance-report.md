# Stage C9 Smoke And Operational Harness Governance Report

## Summary

Stage C9 consolidated the repository governance around smoke tests and
operational harnesses without changing domain behavior.

The outcome is explicit:

- `make smoke*` is now the canonical operational-proof surface;
- `make live*` and `stack-*` are explicitly classified as ergonomic wrappers;
- direct scripts and CLI runtime helpers are contained as tolerated legacy or debugging routes;
- repository docs, Make help text, script help text, and tooling wording now agree on that model.

## Diagnosis

Before C9, the repository already had enough operational support surface to
justify stricter governance:

- multiple smoke targets with different depth and scope;
- activation wrappers that also performed validation;
- direct shell harnesses with richer flags than their Make wrappers;
- a deep tooling gate that still included a legacy runtime smoke helper;
- documentation that broadly preferred `make`, but still left room to read
  `live*`, direct scripts, and deep tooling checks as parallel proof routes.

The main governance problems were:

1. proof-of-record entrypoints were implied rather than explicitly classified;
2. `make live*` risked being read as equivalent to the `make smoke*` proof surface;
3. `make check-deep` and `runtime-smoke` still had language that could be read as operational proof;
4. script help text did not consistently point back to the canonical Make target;
5. ownership across Makefile, scripts, docs, and CLI compatibility helpers was not captured in one operational governance artifact.

## Applied Changes

### Public surface alignment

- Updated `Makefile` target descriptions to distinguish canonical smoke proofs
  from ergonomic `live*` wrappers.
- Updated `README.md` and `DEVELOPMENT.md` so `make smoke*` is the stated
  operational proof-of-record surface.
- Added `make docs` references to the new operational governance documents.

### Script and CLI alignment

- Added canonical-entrypoint guidance to the relevant script `--help` output.
- Normalized `scripts/smoke-analytical-e2e.sh` argument handling so it now
  supports explicit `--help` and validated `--wait` parsing like the other
  smoke harnesses.
- Updated `raccoon-cli` help and deep-gate messaging so the CLI no longer
  suggests `make check-deep` as the runtime proof-of-record path.

### Governance documentation

- Added `docs/operations/smoke-and-operational-harness-governance.md`.
- Added `docs/operations/operational-proof-entrypoints-and-ownership.md`.
- Updated `docs/operations/README.md` and related operational/tooling docs to
  link and reflect the new governance model.
- Extended `scripts/repository-consistency-check.sh` so these documents and the
  C9 stage report are now required repository support artifacts.

## Final Model

The repository now uses this operational-proof model:

- canonical proofs: `make smoke*`;
- ergonomic wrappers: `make live*`, `stack-*`;
- tolerated legacy/debugging routes: direct scripts, direct runtime-smoke CLI
  helper, deep gate runtime helper, raw substrate commands;
- discontinued-as-primary guidance: any new documentation or ownership model
  that treats those tolerated routes as first-choice proof entrypoints.

That model keeps one authoritative proof surface while still preserving
pragmatic debugging paths.

## Residual Limits

C9 intentionally did not:

- rewrite the large smoke harnesses;
- replace the legacy CLI runtime helper;
- merge all operational proofs into one mega-harness;
- touch domain/runtime logic to make proofs easier.

Those would be broader scope changes than this governance pass.

## Recommended Preparation For C10

The next stage should stay narrow and structural as well:

1. decide whether the legacy `runtime-smoke` helper should remain merely frozen
   or move to a harder deprecation path;
2. consider extracting shared shell proof utilities from the largest smoke
   scripts if harness maintenance friction continues to grow;
3. add a small validation matrix or ownership check if future stages add new
   `smoke-*` targets, so canonicality rules stay enforced automatically.
