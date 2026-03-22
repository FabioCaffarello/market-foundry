# Raccoon CLI Command Lifecycle And Deprecation Strategy

## Purpose

This document defines how `raccoon-cli` commands are introduced, matured,
stabilized, consolidated, marked as legacy, and eventually removed.

The CLI is a repository development tool for `market-foundry`. It must evolve
with lifecycle discipline, not by accreting one-off commands.

## Strategic Position

- `make` remains the canonical public workflow surface for repository
  development and runtime proof.
- `raccoon-cli` is the expert support surface for repository analysis,
  structural validation, and change guidance.
- The CLI must not become a second operational platform, a generic toolbox, or
  a dumping ground for narrow one-off wrappers.

## Lifecycle States

| State | Meaning | Documentation posture | Removal expectation |
|---|---|---|---|
| `stable core` | Canonical, recurring command surface used in normal development | visible in top-level help, command reference, and examples | no planned removal without migration path |
| `stable utility` | Durable but narrower support commands used for focused analysis | documented, but positioned as expert or situational tooling | removable only through explicit replacement or consolidation |
| `experimental` | Bounded proving surface still gathering evidence | explicitly labeled experimental and kept out of default promotion | expected to be promoted, consolidated, or retired |
| `legacy` | Compatibility-only or deprecated surface | visibly labeled as legacy or compatibility-only | expected eventual retirement when callers are gone |

## State Assignment Rules

### Stable core

Use `stable core` only when a command:

- serves a recurring repository workflow;
- has a clear canonical owner in the CLI taxonomy;
- has help text and examples aligned with current workflow rules;
- is reliable enough for normal contributor use;
- is not redundant with an existing command or `make` target.

### Stable utility

Use `stable utility` when a command:

- solves a real recurring expert need;
- is narrower than the main taxonomy but still durable;
- benefits from remaining directly callable rather than being folded into a
  larger grouped command;
- does not confuse the main workflow surface.

### Experimental

Use `experimental` when a command:

- is proving a new support pattern or analysis capability;
- still has open questions around usefulness, naming, or scope;
- should not yet shape contributor expectations;
- can be retired without causing workflow breakage.

Experimental commands should be clearly labeled in help and docs, avoid taking
ownership of critical workflows, and come with explicit promotion criteria.

### Legacy

Use `legacy` when a command:

- exists mainly to preserve compatibility or migration continuity;
- has been superseded by a better canonical surface;
- is known to be fragile, narrowly scoped, or misaligned with current strategy;
- should remain behaviorally frozen except for safety and compatibility fixes.

## Birth Criteria For New Commands

A new command is justified only when all of the following are true:

- the need is recurring, not a one-off stage convenience;
- the task does not already fit an existing command, flag, subcommand, or
  output mode;
- the task should live in expert tooling rather than `make`, scripts, or docs;
- the output can be kept trustworthy and actionable;
- the command name fits the grouped taxonomy or has a strong reason to remain a
  stable utility outside it.

Before adding a command, evaluate these alternatives in order:

1. extend an existing subcommand;
2. add a targeted flag or output mode;
3. add a new grouped subcommand;
4. add a stable utility command;
5. add a temporary experimental command.

If a need is primarily runtime orchestration, developer lifecycle ergonomics, or
proof-of-record execution, it likely belongs in `make`, not in `raccoon-cli`.

## Promotion Criteria For Experimental Commands

Promote an experimental command only when it has:

- repeated use across more than one stage or repository scenario;
- stable naming and scope;
- clear operator value that is not already covered elsewhere;
- consistent output semantics and test coverage;
- documentation ready for public help and command-reference inclusion.

Promotion options:

- `experimental` -> `stable core` when it becomes part of recurring workflows;
- `experimental` -> `stable utility` when it remains specialist but durable;
- `experimental` -> removed when evidence does not justify keeping it.

## Consolidation Rules

Command consolidation is required when two surfaces:

- answer the same developer question with slightly different names;
- differ only by output verbosity or minor filters;
- compete for the same place in examples or docs;
- cause uncertainty about which command is canonical.

Preferred consolidation moves:

1. keep one canonical command;
2. preserve old entrypoints as hidden compatibility aliases when needed;
3. update docs and help so the canonical surface is unmistakable;
4. remove duplicate logic paths so aliases dispatch to the same implementation.

## Deprecation Strategy

Deprecation must be gradual and explicit.

### Marking a command as legacy

When a command is superseded:

- label it `legacy` in help text and docs;
- point to the canonical replacement in help and examples;
- stop promoting it in top-level docs;
- keep behavior stable unless safety fixes are needed.

### Hidden compatibility aliases

Aliases are the first deprecation layer when naming changes but behavior stays
the same.

Rules:

- aliases must remain thin dispatch wrappers with no behavior fork;
- aliases should be hidden from main help once the grouped command is canonical;
- docs should mention them only in compatibility sections, not as first-choice
  examples.

### Retirement

Remove a legacy command only when:

- the canonical replacement has been established and documented;
- the remaining compatibility value is low;
- tests and docs no longer depend on the old surface;
- removal will not abruptly break the repository workflow contract.

## Documentation And Help Rules

- top-level help should distinguish stable core, stable utility, experimental,
  and legacy surfaces;
- grouped command docs should use canonical commands in examples;
- utility commands should be documented as specialist tools, not as the default
  workflow;
- legacy commands must carry an explicit replacement or containment note;
- when no experimental commands are promoted, docs should say so rather than
  leaving the state ambiguous.

## Current Application To Market Foundry

Current policy application:

- `check`, `inspect`, and `change` are `stable core`;
- `snapshot`, `snapshot-diff`, and `baseline-drift` are `stable utility`;
- no command is currently promoted as `experimental`;
- `legacy runtime-smoke` and hidden flat aliases are `legacy`.

The key lifecycle concern today is not command explosion inside the grouped
taxonomy. It is preventing hidden aliases and historical runtime helpers from
being mistaken for canonical workflow surfaces.

## Governance Heuristics

When evaluating a command change, ask:

1. Does this make the CLI clearer, or just larger?
2. Does this belong in expert tooling, or in `make`?
3. Is there already a nearby command that should absorb this need?
4. Will contributors know whether this surface is canonical, specialist, or
   legacy?
5. If this command still exists in a year, will that be a sign of value or of
   drift?

If those answers are weak, the command should not be born, or should remain
experimental until the case becomes stronger.
