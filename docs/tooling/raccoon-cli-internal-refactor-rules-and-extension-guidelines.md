# Raccoon CLI Internal Refactor Rules And Extension Guidelines

## Purpose

This document defines the post-C13 rules for extending or refactoring
`tools/raccoon-cli`.

The CLI is an internal repository-support product. Its architecture must remain
modular, bounded, and intentionally non-generic.

## Core Rules

### 1. Keep parsing separate from execution policy

`src/cli/mod.rs` may own:

- Clap types;
- grouped taxonomy;
- compatibility aliases;
- examples and help text;
- parse-time validation only.

`src/cli/mod.rs` must not own:

- analyzer dispatch;
- gate configuration logic;
- subprocess access;
- output side effects;
- orchestrator-specific conversions.

If a parsed value needs translation into runtime policy, do it in
`src/application/*`.

### 2. Keep `application` thin, but authoritative

`src/application/*` owns:

- command dispatch;
- compatibility alias convergence;
- exit-code policy;
- output format selection;
- light orchestration helpers such as optional-LSP handling and target
  auto-detection.

`src/application/*` must not become:

- a second analyzer layer;
- a framework for generic command middleware;
- a dumping ground for report-specific rendering;
- a place where subprocess calls proliferate again.

### 3. Put subprocess-backed support under `src/io/*`

Use `src/io/*` when the CLI needs:

- `git` shell-outs;
- timeout-based subprocess execution;
- system utility calls needed by the CLI itself.

Do not add new direct `std::process::Command` usage to:

- `main.rs`;
- `cli/mod.rs`;
- arbitrary application command handlers;
- analyzer modules that are only formatting or composing data.

If a new subprocess behavior is reused or represents a stable support concern,
promote it into `src/io/*` immediately.

### 4. Keep report-specific rendering with the report owner

Preferred ownership order:

1. the module that owns the report type;
2. `src/output/mod.rs` for shared `Report` rendering only;
3. `src/application/*` only for genuinely command-owned presentation policy.

Examples:

- `lsp::EnrichedSymbol` rendering belongs under `src/lsp/*`;
- analyzer-specific reports should render in their analyzer module;
- shared guard-rail `Report` rendering belongs in `src/output/mod.rs`.

### 5. Use bounded helpers, not frameworks

Small helpers are allowed when they remove repeated policy.

Examples of good helpers:

- optional-LSP lifecycle wrappers;
- common change-target resolution;
- IO wrappers for stable subprocess patterns.

Rejected patterns:

- command registries;
- trait-object command plugins;
- generic middleware chains for a fixed internal tool;
- abstraction layers that hide simple match-based routing.

### 6. Preserve command taxonomy discipline

New commands must fit one of the existing groups unless a strong architectural
reason exists:

- `check`
- `inspect`
- `change`
- `snapshot`
- `legacy`

Compatibility aliases are allowed only to preserve historical entrypoints.
Aliases must converge into the same execution path as the canonical command.

### 7. Keep the CLI out of the runtime control plane

The CLI may:

- inspect repository structure;
- inspect source, docs, configs, and contracts;
- provide analysis and validation guidance;
- retain clearly-labeled legacy compatibility helpers.

The CLI must not become:

- the primary runtime operator for the system;
- a replacement for `make smoke*` operational flows;
- a coordinator for application runtime lifecycle management.

## Extension Guidelines

### Adding a new analytical command

1. Add the grouped command and help text in `src/cli/mod.rs`.
2. Route it in `src/application/mod.rs`.
3. Put analytical behavior in `src/analyzers/*`.
4. Put any reusable subprocess-backed support in `src/io/*`.
5. Keep rendering with the analyzer unless the output truly belongs elsewhere.
6. Add or update CLI integration tests.
7. Update tooling docs if a boundary or module map changed.

### Adding optional LSP enrichment

If a command supports AST-only and LSP-enriched modes:

1. keep flags in `src/cli/mod.rs`;
2. use the application-level optional-LSP orchestration helpers;
3. keep semantic data types and rendering inside `src/lsp/*` or the owning
   analyzer;
4. avoid duplicating bridge setup/teardown inside each command handler.

### Adding new IO/process support

When you need a new shell-out pattern:

1. ask whether it is CLI support infrastructure or command-local behavior;
2. if it is support infrastructure, place it in `src/io/*`;
3. keep the function focused on one stable concern;
4. return data, not stdout/stderr side effects;
5. let `application` decide user-facing behavior and exit codes.

### Refactoring existing command handlers

A refactor is justified when it:

- removes repeated orchestration policy;
- clarifies ownership boundaries;
- reduces cross-layer knowledge;
- improves testability without increasing abstraction drag.

A refactor is not justified when it only:

- renames modules without changing responsibility;
- adds indirection to hide a single direct call;
- creates future-proofing abstractions without present pressure.

## Review Checklist

Before merging a CLI change, confirm:

1. parsing still stops at `src/cli/mod.rs`;
2. execution policy still lives in `src/application/*`;
3. subprocess-backed support did not leak out of `src/io/*` again;
4. report-specific rendering stayed with the report owner;
5. compatibility aliases did not fork behavior;
6. the CLI still behaves as repository tooling, not system runtime control.
