# Raccoon CLI Module Boundaries And Evolution Rules

## Purpose

This document defines the internal guard rails for evolving
`tools/raccoon-cli` after Stage C8.

Stage C13 refined these boundaries further. The current advanced guidance lives
in
`docs/tooling/raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`.

The goal is not to freeze the CLI. The goal is to keep growth legible,
taxonomy-consistent, and bounded to repository-support concerns.

## Boundary Rules

### 1. `main.rs` stays thin

`src/main.rs` may:

- parse the CLI;
- call the application entrypoint;
- exit with the returned process code.

`src/main.rs` must not regain:

- command dispatch logic;
- report rendering logic;
- `git status` heuristics;
- command-specific branching beyond process startup.

### 2. `src/cli/mod.rs` owns the command surface

This module owns:

- Clap structs and enums;
- grouped taxonomy;
- hidden compatibility aliases;
- global flags;
- command help text and examples.

It must not own:

- analyzer execution;
- repository inspection logic;
- process spawning;
- output side effects other than parse semantics.

### 3. `src/application/mod.rs` owns command execution policy

This module owns:

- dispatch from parsed command to use case;
- output-format selection;
- exit-code policy;
- compatibility alias convergence;
- minor command orchestration such as snapshot file writes.

It must not become:

- a generic command framework;
- a second parser layer;
- a place where analyzer logic is duplicated.

### 4. `src/application/change_targets.rs` owns auto-detected change inputs

If a command wants to infer targets from the worktree, it should use this
module.

Do not re-implement:

- `git status` parsing;
- rename normalization;
- deleted-file filtering;
- structural-target filtering;

inside analyzer modules or in `main.rs`.

### 5. `src/analyzers/*` own read-only analysis behavior

Analyzers may:

- inspect files, configs, source, and docs;
- build structured reports;
- compose other analysis helpers when the behavior is inherently analytical.

Analyzers must not:

- parse Clap arguments;
- decide process exit codes;
- write to stdout/stderr;
- call `std::process::exit`;
- own grouped command taxonomy.

### 6. `src/gate/mod.rs` is a bounded application orchestrator

`gate` is allowed to coordinate multiple analyzers and the legacy smoke helper
because that is the nature of the quality-gate use case.

Do not turn `gate` into a generic scheduler for unrelated command flows.

### 7. Support modules stay support modules

| Module | Allowed purpose |
|---|---|
| `src/io/*` | subprocess-backed repository/system IO helpers |
| `src/output/mod.rs` | render shared `Report` values |
| `src/models/mod.rs` | define shared report/finding/status types |
| `src/error/mod.rs` | define CLI-local error transport |
| `src/smoke/*` | legacy runtime smoke helper only |
| `src/lsp/*` | semantic enrichment support |
| `src/codeintel/*` | AST/code index support |

These modules should stay reusable and policy-light.

## Allowed Dependency Direction

The intended internal dependency direction is:

```text
main
  -> cli
  -> application

application
  -> gate
  -> analyzers
  -> smoke
  -> output/models/error
  -> lsp/codeintel/process helpers (indirectly where needed)

gate
  -> analyzers
  -> smoke
  -> output/models/error

analyzers
  -> codeintel/lsp/models/error

support modules
  -> lower-level support only
```

Forbidden growth directions:

- analyzers depending on `cli`;
- analyzers depending on `application`;
- output depending on Clap or command enums;
- `smoke` becoming a canonical runtime control plane.

## Rules For Adding A New Command

When adding a new command:

1. Add the canonical command and any compatibility alias in `src/cli/mod.rs`.
2. Add execution wiring in `src/application/mod.rs`.
3. Put target auto-detection in `src/application/change_targets.rs` if needed.
4. Put analysis logic in a new or existing analyzer if the command is analytical.
5. Keep rendering with the report owner unless the renderer is genuinely shared.
6. Update command-surface docs in `docs/operations/`.
7. Update tooling-internal docs here when the module map or boundaries change.

## Rules For Renderers

Prefer this order:

1. renderer next to the report owner for report-specific formats;
2. `src/output/mod.rs` for shared `Report` rendering;
3. `src/application/renderers.rs` only when the renderer is command-owned and
   not a natural analyzer concern.

Do not add ad hoc renderers back into `main.rs`.

## Compatibility Alias Policy

Flat historical commands are allowed only as compatibility aliases.

Guard rails:

- new docs should prefer grouped commands;
- aliases should forward to the same execution path as canonical commands;
- aliases should not create separate internal modules or separate behavior.

## Anti-Patterns Rejected By C8

Do not reintroduce:

- “just add another branch to `main.rs`” command growth;
- direct analyzer access to Clap types;
- duplicate `git status` heuristics in multiple command handlers;
- fake abstractions that hide one concrete command behind multiple layers;
- new runtime orchestration features under `raccoon-cli` when the Makefile
  already owns the workflow.

## Review Checklist For Future CLI Changes

Before merging a new CLI change, confirm:

1. the command landed in the right layer;
2. the taxonomy stayed grouped and consistent;
3. compatibility aliases did not fork behavior;
4. rendering and exit-code policy stayed centralized;
5. no new runtime-control-plane drift was introduced.
