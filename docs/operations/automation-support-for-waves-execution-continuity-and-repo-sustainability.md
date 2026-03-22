# Automation Support For Waves, Execution Continuity, And Repo Sustainability

## Purpose

This document defines the lightweight automation model that supports continuous
execution in `market-foundry`.

The goal is not to automate the repository into a platform. The goal is to
reduce recurring friction around waves, stages, validation, operational
navigation, and repeatable maintenance work.

## Diagnosis

Before C20, the repository already had strong support entrypoints:

- `make` for the public workflow;
- `make check`, `make verify`, and `make smoke*` for validation and proof;
- `make stage-scaffold` and `make stage-check` for governed stage work;
- operational and navigation docs for canonical usage.

The remaining recurring friction was narrower:

1. Resuming a stage still depended too much on manually checking whether the
   report existed, was indexed, linked the right durable docs, and already had
   enough closure shape.
2. Support-only stages often required re-running the same mental checklist
   before `make stage-check` or `make verify`.
3. Wave/stage continuity was documented, but there was no lightweight
   automation surface that translated that continuity into immediate next steps.
4. The boundary between high-value helper automation and low-value workflow
   engine behavior was implicit rather than explicit.

## High-Value Automation Chosen

### 1. Stage continuity status

The repository now exposes:

```bash
make stage-status STAGE_ID=... STAGE_SLUG=...
```

This helper is advisory rather than enforcing. It reports:

- whether the active stage report exists in the expected location;
- whether it is already indexed in `docs/stages/INDEX.md`;
- whether the report has the expected closure signals;
- whether declared durable artifacts already exist;
- which next commands to run.

This is high-value because contributors repeatedly need this information when a
stage spans more than one session or when a wave is being advanced carefully.

### 2. Continuity-aware stage helper integration

The stage helper now has a three-step support flow:

1. `make stage-scaffold`
2. `make stage-status`
3. `make stage-check`

This preserves the existing lightweight model:

- scaffold opens the report;
- status restores context and highlights gaps;
- check enforces the minimum closure floor.

### 3. Sustainability through existing guard rails

The automation-support docs and the C20 stage report are now part of the
required repository documentation set protected by
`scripts/repository-consistency-check.sh`.

That keeps the support model durable without adding a separate automation
registry or another metadata layer.

## Recurring Routines This Model Supports

| Routine | Manual failure mode | Lightweight support now used |
|---|---|---|
| Resume a governed stage | forget report/index/artifact gaps | `make stage-status` |
| Close a governed stage | rely on memory for minimum completeness | `make stage-check` |
| Keep support docs coherent after support-surface changes | drift across README/operations/stage docs | `make repo-consistency-check` |
| Re-enter the standard dev loop after support changes | skip the normal validation sequence | `make check`, `make tdd`, `make verify` |
| Recover navigation to the right operational surface | search through history or scripts ad hoc | `make docs`, `docs/operations/README.md`, area `README.md` entrypoints |

## Operating Model

### For a new or resumed stage

Use this order:

1. confirm the stage objective and scope in the governing docs;
2. run `make stage-status STAGE_ID=... STAGE_SLUG=...`;
3. fill the gaps surfaced there;
4. run `make stage-check STAGE_ID=... STAGE_SLUG=...`;
5. run the narrowest repository validation needed for the change.

### For support-only repository changes

Use this order:

1. `make repo-consistency-check`
2. `make stage-status` when the work belongs to an active governed stage
3. `make stage-check` when the stage is close to closure
4. `make verify` unless a narrower proof is more appropriate

## Why This Stays Lightweight

- No automation writes stage indexes automatically.
- No helper infers stage meaning or wave authorization.
- No workflow engine tracks assignees, approvals, or state transitions.
- No new registry is introduced for waves or checkpoints.
- All helpers remain transparent shell wrappers with readable output.

## Sustainability Rules

- Add automation only when the routine is frequent, error-prone, and easy to
  explain.
- Prefer wrappers that expose real repository artifacts instead of hiding them.
- Extend `make` only when the result becomes part of the public workflow.
- Keep automation composable with existing docs and checks instead of building
  a parallel orchestration layer.
- Protect lasting support automation through existing consistency checks, not
  through bespoke governance subsystems.

## Related Documents

- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`stage-artifacts-conventions-and-support-model.md`](stage-artifacts-conventions-and-support-model.md)
- [`repository-automation-boundaries-high-value-routines-and-sustainability-rules.md`](repository-automation-boundaries-high-value-routines-and-sustainability-rules.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`../stages/stage-c20-automation-support-for-waves-execution-continuity-and-repo-sustainability-report.md`](../stages/stage-c20-automation-support-for-waves-execution-continuity-and-repo-sustainability-report.md)
