# Stage Tooling And Execution Governance Support

## Purpose

This document defines how `market-foundry` supports disciplined stage execution
in practice.

It does not redefine stage semantics. The Opus, architecture governance, and
stage-definition rules still define what a stage means. This document only
defines the lightweight support surface that helps contributors execute stages
more consistently.

## Problem This Support Surface Solves

The repository already had strong stage discipline at the policy level:

- stage reports were mandatory
- `docs/stages/INDEX.md` was the historical navigation surface
- architecture docs described stage governance and definition of done
- repository consistency checks protected naming and index alignment

The remaining friction was practical:

- no single helper existed to open a new stage report in a predictable shape
- no stage-focused check existed for a contributor working on one active stage
- minimum stage completeness still depended heavily on memory
- stage support rules were spread across architecture docs, operations docs,
  the stage index, and historical reports

## Canonical Support Surface

Use this surface when executing a stage:

| Need | Canonical support entrypoint |
|---|---|
| Understand stage support model | `docs/operations/stage-tooling-and-execution-governance-support.md` |
| Understand artifact naming and minimum completeness | `docs/operations/stage-artifacts-conventions-and-support-model.md` |
| Scaffold a new stage report | `make stage-scaffold STAGE_ID=... STAGE_SLUG=... STAGE_TITLE=...` |
| Inspect continuity for one active stage | `make stage-status STAGE_ID=... STAGE_SLUG=...` |
| Validate one active stage | `make stage-check STAGE_ID=... STAGE_SLUG=...` |
| Get usage help for the stage helper | `make stage-help` |
| Validate repository-wide lightweight governance | `make repo-consistency-check` |
| Review historical stage evidence | `docs/stages/INDEX.md` |

## What The New Helper Does

`scripts/stage-tooling.sh` is intentionally narrow.

It provides two supported actions:

1. `scaffold`
2. `status`
3. `check`

### Scaffold

`make stage-scaffold` creates only the stage report skeleton.

This is deliberate:

- a stage report is the one artifact every governed stage must produce
- operations or architecture docs vary by stage and should remain intentional
- the helper should reduce setup friction, not infer domain meaning

### Check

`make stage-check` validates one active stage report against lightweight
execution discipline:

- report exists in the expected location or explicit path
- filename follows the stage-report convention
- report is indexed in `docs/stages/INDEX.md`
- report contains a minimum working section set
- report contains explicit scope-boundary signals
- report-local markdown links resolve
- optional required artifacts exist when declared through `STAGE_REQUIRE`

This is a stage-local guard rail. It is not a replacement for:

- `make verify`
- `make check`
- `make smoke*`
- architecture reviews

### Status

`make stage-status` is the continuity-oriented companion to `make stage-check`.

It answers the lightweight questions that commonly stall resumed stage work:

- does the report already exist in the expected place;
- is it already indexed in `docs/stages/INDEX.md`;
- are the minimum report signals present or obviously incomplete;
- are the declared durable artifacts already present;
- what are the next recommended commands.

`status` is intentionally advisory. It should help contributors recover context
quickly without promoting a heavyweight stage tracker.

## Operational Flow For A Governed Stage

Use this order:

1. Open or confirm the stage objective and boundaries in architecture/governance docs.
2. Run `make stage-scaffold` if the report does not exist yet.
3. Deliver the minimal tooling/docs/code changes that the stage actually requires.
4. Update canonical docs when a lasting convention or support surface changes.
5. Add the report to `docs/stages/INDEX.md`.
6. Run `make stage-status` when you are resuming work, validating continuity, or preparing to close the stage.
7. Run `make stage-check` for the active stage.
8. Run the narrowest repository validation needed for the change, usually `make repo-consistency-check` and then the normal `make verify` or narrower proof path.
9. Close the report with explicit limits, validation, and next-stage preparation.

## Relationship To Existing Governance

This support model complements existing governance layers instead of replacing
them:

| Layer | Role |
|---|---|
| `docs/architecture/stage-definition-of-done.md` | Defines what a complete stage means |
| `docs/architecture/monorepo-documentation-and-stage-governance.md` | Defines the high-level governance model |
| `docs/stages/INDEX.md` | Historical evidence navigation |
| `make stage-status` | Advisory continuity snapshot for an active stage |
| `make repo-consistency-check` | Repository-wide lightweight support-surface guard rail |
| `make stage-check` | Active-stage completeness and traceability check |

## What Stays Manual On Purpose

The repository should not turn stage execution into a workflow engine.

The following remain intentionally manual:

- deciding whether a stage is justified at all
- defining the real objective and boundaries
- deciding which canonical docs must change
- deciding what evidence is sufficient for closure
- interpreting trade-offs and choosing the next stage

The helper removes repeated mechanical work, not judgment.

## Practical Rules

- Prefer updating an existing canonical doc before creating a new one.
- Every governed stage should leave one obvious current support path behind it.
- Use `STAGE_REQUIRE` only for concrete artifacts the stage is expected to
  produce; do not turn it into a generic checklist dump.
- Prefer `make stage-status` when context recovery is the problem and `make stage-check` when closure discipline is the problem.
- If `make stage-check` passes but the stage still feels unclear, the stage is
  not operationally complete yet. The check is a floor, not a substitute for
  clarity.

## Related Documents

- [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`../architecture/stage-definition-of-done.md`](../architecture/stage-definition-of-done.md)
- [`../architecture/monorepo-documentation-and-stage-governance.md`](../architecture/monorepo-documentation-and-stage-governance.md)
- [`../stages/INDEX.md`](../stages/INDEX.md)
