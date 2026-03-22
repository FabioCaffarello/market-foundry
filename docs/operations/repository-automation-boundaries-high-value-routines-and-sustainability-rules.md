# Repository Automation Boundaries, High-Value Routines, And Sustainability Rules

## Purpose

This document defines where repository automation in `market-foundry` is
desirable and where it becomes counterproductive.

It exists to keep C20 aligned with the repository's operating philosophy:
automation should reduce friction and preserve continuity, not obscure reality
or create a control plane.

## High-Value Routines Worth Automating

Automate a repository routine when most of the following are true:

- contributors perform it repeatedly across waves or stages;
- the failure mode is usually omission, drift, or inconsistency;
- the automation can stay transparent about what it checks or runs;
- the routine touches repository governance, validation, or navigation rather
  than domain semantics;
- the support surface can live behind `make` or a simple script without needing
  a stateful platform.

Current high-value examples:

| Routine | Why it is worth automation | Surface |
|---|---|---|
| Fast repository consistency pass | catches common support-surface drift early | `make repo-consistency-check` |
| Stage report scaffolding | removes repeated document setup | `make stage-scaffold` |
| Stage continuity inspection | reduces context-recovery and closure friction | `make stage-status` |
| Stage closure floor | catches missing report/index/artifact discipline | `make stage-check` |
| Canonical dev validation loop | keeps the normal guard-rail path obvious | `make check`, `make tdd`, `make verify` |

## Routines That Should Stay Manual

Keep a routine manual when it depends on judgment more than repetition.

Examples:

- deciding whether a new wave or stage should exist at all;
- choosing the real scope boundary for a stage;
- deciding which architecture or operations docs deserve promotion;
- determining what evidence is sufficient to close a wave;
- interpreting failures from smoke or integration proof surfaces.

If automation would replace reasoning with templated output, it is the wrong
automation for this repository.

## Anti-Patterns

Do not introduce:

- a workflow engine for stages or waves;
- hidden wrappers that mask the real commands or artifacts being used;
- automatic report, index, or governance edits that contributors stop reading;
- a second metadata registry for stages, waves, or checkpoints;
- automation for rare or one-off repository chores.

## Practical Boundary Rules

### Prefer advisory automation first

If a recurring routine mostly needs context recovery, start with a status or
diagnostic helper before adding enforcement.

### Enforce only stable, objective rules

Use checks for things like:

- file presence;
- naming conventions;
- index alignment;
- local link resolution;
- minimum report/document shape.

Do not enforce subjective writing style, strategy, or wave planning logic.

### Protect support surfaces through existing checks

If a new automation changes the public workflow, integrate it into the existing
docs and `make`/consistency surfaces instead of creating new registries or
background services.

### Keep outputs legible

A helper should tell the contributor:

1. what it inspected;
2. what is missing or drifting;
3. what to do next.

If the output needs another manual to explain it, the helper is too heavy.

## Sustainability Rules

- Every lasting automation entrypoint should be reachable from `make` or a
  clearly documented script.
- Every lasting automation entrypoint should have one canonical doc owner in
  `docs/operations/`.
- Every lasting automation rule should be cheap enough to run routinely.
- Support automation must not modify domain behavior or cross architecture
  boundaries.
- Prefer deleting stale automation over preserving decorative support surfaces.

## Review Questions For Future Automation Work

Before adding a new helper, answer:

1. Which recurring friction does it remove?
2. Why is an existing `make` target or script not enough?
3. Is the output advisory or enforcing, and why?
4. What real artifact or workflow becomes easier to sustain?
5. What manual judgment still remains intentionally manual?

If these questions do not have crisp answers, the automation is probably not
worth adding.

## Related Documents

- [`automation-support-for-waves-execution-continuity-and-repo-sustainability.md`](automation-support-for-waves-execution-continuity-and-repo-sustainability.md)
- [`stage-tooling-and-execution-governance-support.md`](stage-tooling-and-execution-governance-support.md)
- [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
