# Stage C10 Developer Workflow Unification Report

## Summary

Stage C10 unified the developer workflow of `market-foundry` around one
official operational journey:

- bootstrap the machine with `make bootstrap`;
- bring up the stack with `make live` or the controlled manual path;
- validate changes with `make check`, `make tdd`, and `make verify`;
- prove runtime behavior with the narrowest relevant `make smoke*`;
- troubleshoot first with `make diag`, `make ps`, `make logs`, and `make restart`.

The stage stayed inside repository DX/workflow surfaces. It did not alter
business logic, bounded-context behavior, or functional architecture.

## Diagnosis

Before C10, the repository already had a strong support surface, but the
developer journey still had practical friction:

1. there was no explicit bootstrap/setup entrypoint for a new machine or changed local environment;
2. the difference between `make live` and `make up` + `make seed` was documented, but not presented as one official hierarchy in one place;
3. onboarding and troubleshooting guidance was spread across `README.md`, `DEVELOPMENT.md`, `docs/operations/README.md`, and narrower governance documents;
4. contributors had to infer the real day-to-day path from multiple support documents rather than following one operational runbook;
5. the repository had operational governance, but it still needed a more teachable developer workflow contract.

## Changes Applied

### Entrypoints and ergonomics

- Added `make bootstrap` as the canonical setup-validation entrypoint.
- Added `scripts/bootstrap-check.sh` to validate host tools, Docker availability, compose renderability, canonical repository entrypoints, and required local env files.
- Updated `make docs` so the new workflow and onboarding documents are part of the primary documentation set.

### Documentation alignment

- Added `docs/operations/developer-workflow-unification.md`.
- Added `docs/operations/developer-onboarding-and-troubleshooting-guide.md`.
- Updated `README.md` quick-start and hierarchy guidance.
- Updated `DEVELOPMENT.md` to reflect the official setup, bring-up, smoke-selection, and troubleshooting paths.
- Updated `docs/README.md` and `docs/operations/README.md` so the new documents are first-class entrypoints.
- Updated `docs/operations/makefile-targets-reference-and-conventions.md` and `docs/operations/scripts-catalog-and-usage-guide.md` to include the bootstrap surface.

### Repository guard rails

- Extended `scripts/repository-consistency-check.sh` so the new C10 documents and stage report are required repository artifacts.
- Added C10 to `docs/stages/INDEX.md`.

## Validation

Validation executed for C10:

- `make repo-consistency-check`

Observed outcome:

- repository consistency passed before and during the workflow-unification work;
- the new C10 documents and entrypoints were wired into the same support-surface guard rails that protect earlier C-stage documentation.

## Outcome

After C10, the repository has a clearer and more teachable workflow:

- setup now has a canonical entrypoint instead of implicit prerequisites;
- bring-up has one explicit hierarchy: `make live` for fastest bring-up, `make up` + `make seed*` for controlled manual bring-up;
- runtime proof still belongs to `make smoke*`, now within a clearer end-to-end developer journey;
- troubleshooting starts from one small set of official commands before escalating to expert/debug routes;
- onboarding guidance is more concrete without introducing new functional behavior or parallel workflow surfaces.

## Recommended Preparation For C11

1. measure whether `make bootstrap` should grow a narrow optional check for local credential or env drift, but only if real friction appears;
2. keep reducing duplication between top-level docs by pushing detailed operational guidance into the new C10 documents rather than re-expanding `README.md` or `DEVELOPMENT.md`;
3. if harness-specific friction continues, charter C11 around targeted runtime/operator ergonomics or smoke-harness maintainability, not around broader workflow redesign.
