# Stage C37 Report: Development-Platform Convergence Light Enforcement

## Summary

Stage C37 added one final light enforcement pass to protect the development
platform's convergence without opening a new governance layer.

The stage stayed deliberately narrow:

- define only the invariants that materially protect central workflow trust;
- add proportional checks for drift between workflow docs, Make wrappers,
  scripts catalog rows, and the public `raccoon-cli` taxonomy;
- document the protected contract where contributors already look for it;
- avoid cosmetic lint, expansive doc freezing, or new support surfaces.

## Scope

### In scope

- workflow owner docs that define the public developer loop;
- `Makefile` wrappers that promote grouped `raccoon-cli` commands;
- `docs/operations/scripts-catalog-and-usage-guide.md` as the canonical
  script-backed wrapper catalog;
- `docs/tooling/cli-overview.md` and the Make/CLI contract doc as the
  public taxonomy/boundary record;
- lightweight enforcement in `scripts/repository-consistency-check.sh`.

### Out of scope

- adding new Make targets, scripts, or CLI families;
- broad documentation cleanup outside the protected contract;
- linting wording, formatting, or exhaustive target inventories everywhere;
- changing smoke semantics, runtime behavior, or stage workflow semantics.

## Protected Invariants

The protected set is intentionally small:

1. the workflow owner docs keep the same minimal public loop:
   `make bootstrap`, `make check`, `make tdd`, `make verify`, then the
   relevant `make smoke*`;
2. Make wrappers that delegate to `raccoon-cli` keep using the grouped
   taxonomy instead of flat compatibility aliases;
3. the scripts catalog records the real `make <target>` to `scripts/*.sh`
   mapping for script-backed public wrappers;
4. the CLI taxonomy remains centered on `check`, `inspect`, `change`, and
   `legacy`, while runtime proof remains owned by `make smoke*`.

These invariants were selected because drift here directly changes what
contributors run, how they discover the supported path, or whether the same
workflow question starts getting multiple incompatible answers.

## Changes Applied

### Enforcement

- extended `scripts/repository-consistency-check.sh` with:
  - workflow owner-loop alignment checks;
  - Make-to-`raccoon-cli` wrapper contract checks;
  - scripts-catalog contract checks based on real target-to-script mappings.

### Source of truth support

- extended `tools/raccoon-cli/src/command_refs.rs` with grouped command strings
  for `inspect coverage` and `change briefing`, so wrapper checks can rely on
  one CLI wording source instead of duplicating strings ad hoc.

### Documentation

- updated `docs/operations/make-and-raccoon-cli-contract.md` with the protected
  convergence contract;
- updated `docs/tooling/cli-overview.md` with the protected taxonomy contract;
- indexed this stage in `docs/stages/INDEX.md`.

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C37 STAGE_SLUG=development-platform-convergence-light-enforcement STAGE_REQUIRE=docs/stages/stage-c37-development-platform-convergence-light-enforcement-report.md,docs/operations/make-and-raccoon-cli-contract.md,docs/tooling/cli-overview.md`

## Limitations

- the checks do not try to make every workflow doc exhaustive;
- the checks do not freeze CLI internal help text beyond the public taxonomy and
  grouped-wrapper contract;
- the scripts catalog check validates the canonical mapping, not every prose
  example around it;
- compatibility aliases still exist intentionally, so this stage protects the
  promoted surface rather than banning legacy paths outright.

## Follow-Through Rules

1. when a public Make wrapper changes its delegated grouped CLI command, update
   `tools/raccoon-cli/src/command_refs.rs`, the Makefile wrapper, and the
   relevant contract/reference docs in the same change;
2. when a new script-backed public wrapper is added, update the scripts catalog
   row in the same change instead of relying on later cleanup;
3. when workflow guidance changes, preserve the minimal public loop in the
   owner docs before updating deeper reference material;
4. if a future need requires more enforcement, justify it against C31 and keep
   the bias toward convergence protection over surface expansion.

## Preparation For Next Stage

1. keep future workflow-surface edits coupled: owner docs, wrapper, catalog,
   and taxonomy should move together when they change at all;
2. prefer extending the existing convergence checks only when drift has already
   become recurrent or materially misleading in the public workflow;
3. if a future wave touches only one of these central surfaces, use
   `make repo-consistency-check` early to avoid reintroducing silent drift.
