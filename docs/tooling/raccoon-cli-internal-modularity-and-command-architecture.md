# Raccoon CLI Internal Modularity And Command Architecture

## Purpose

This document describes the internal architecture of `tools/raccoon-cli` after
Stage C8.

Stage C13 refined the internals further without changing the core taxonomy.
Read
`docs/tooling/raccoon-cli-advanced-architecture-refinement.md`
for the current refinement pass and updated ownership notes.

The CLI remains a repository support product inside `market-foundry`. Its job is
to validate, inspect, and guide safe repository change. It is not a runtime
operator for the system itself.

## Architectural Goal

The C8 pass focused on three outcomes:

1. make the command taxonomy explicit in code, not only in help text;
2. separate command parsing from command execution and support helpers;
3. give future CLI growth clear places to land without turning the CLI into a
   generic framework.

## Final Module Map

| Layer | Module(s) | Responsibility |
|---|---|---|
| entrypoint | `src/main.rs` | thin process entrypoint only |
| command layer | `src/cli/mod.rs` | Clap parsing, grouped taxonomy, flags, compatibility aliases, help text |
| application layer | `src/application/mod.rs` | command dispatch, exit-code policy, renderer selection, snapshot file writes, bounded optional-LSP orchestration |
| application helpers | `src/application/change_targets.rs` | change-target auto-detection |
| orchestration use case | `src/gate/mod.rs` | multi-step quality-gate orchestration |
| analysis services | `src/analyzers/*` | repository checks, structural analysis, drift, impact, planning, snapshot, recommendations |
| support engines | `src/codeintel/*`, `src/lsp/*` | AST indexing, semantic enrichment, and LSP-owned rendering |
| support infrastructure | `src/io/*`, `src/output/mod.rs`, `src/models/mod.rs`, `src/error/mod.rs`, `src/smoke/*` | subprocess-backed CLI IO, report rendering, common report types, CLI errors, legacy smoke helper |

## Command Taxonomy To Internal Routing

### `check`

- `check repo` -> `analyzers::doctor`
- `check topology` -> `analyzers::topology`
- `check contracts` -> `analyzers::contracts`
- `check bindings` -> `analyzers::runtime_bindings`
- `check arch` -> `analyzers::arch_guard`
- `check drift` -> `analyzers::drift_detect`
- `check gate` -> `gate::run`

### `inspect`

- `inspect symbol` -> `analyzers::symbol_trace`
- `inspect lsp` -> `lsp::GoplsBridge` plus `application::renderers`
- `inspect contract-usage` -> `analyzers::contract_usage_map`
- `inspect coverage` -> `analyzers::coverage_map`

### `change`

- `change impact` -> `analyzers::impact_map`
- `change tdd` -> `analyzers::tdd`
- `change briefing` -> `analyzers::briefing`
- `change recommend` -> `analyzers::recommend`
- `change rename` -> `analyzers::rename_safety`

All change-oriented commands now share the same target-resolution helper in
`src/application/change_targets.rs` instead of each path re-implementing `git
status` heuristics.

### Snapshot Family

- `snapshot` -> `analyzers::snapshot`
- `snapshot-diff` -> `analyzers::snapshot_diff`
- `baseline-drift` -> `analyzers::baseline_drift`

### `legacy`

- `legacy runtime-smoke` -> `smoke::run`

The legacy route is still present for compatibility, but it stays isolated from
the main grouped support surface.

## Execution Flow

1. `main.rs` parses CLI arguments through `cli::Cli`.
2. `application::run` builds an execution context from the parsed CLI.
3. `application::execute` dispatches one canonical or compatibility command.
4. The application layer invokes one analyzer, gate flow, or legacy helper.
5. Rendering is selected once, using either:
   - `output::render` for shared `Report` values;
   - report-specific renderers owned by analyzers;
   - `application::renderers` for command-owned presentation gaps.
6. Exit codes are assigned centrally:
   - `0` for successful checks or informational commands;
   - `1` for failed validation/drift verdicts;
   - `2` for runtime or rendering errors.

## What Changed In C8

### Before C8

`src/main.rs` contained:

- all Clap command declarations;
- compatibility alias handling;
- direct command dispatch;
- output and exit-code policy;
- change-target auto-detection;
- an LSP-specific human renderer;
- internal CLI parsing tests.

That made `main.rs` both the UX surface and the execution center.

### After C8

`src/main.rs` now acts only as an entrypoint.

The execution center moved into `src/application/mod.rs`, while the UX surface
and taxonomy live in `src/cli/mod.rs`. Shared change-target heuristics and the
only non-analyzer renderer were pulled into dedicated helper modules.

This removed the previous “everything lands in main” growth pattern without
inventing a generic plugin system or abstract command framework.

## Why This Shape Is Intentionally Bounded

The CLI is still a relatively small internal product. C8 did not add command
registries, trait-object plugin loaders, or synthetic adapter layers for every
report type.

Instead, the internal shape now follows a simple rule:

- parse in `cli`;
- orchestrate in `application`;
- analyze in `analyzers` or `gate`;
- render in the report owner or shared output modules;
- keep `main` thin.

That is enough structure to support growth without turning the CLI into a
framework.

## Practical Reading Order For Contributors

When changing the CLI internals, read in this order:

1. `src/cli/mod.rs`
2. `src/application/mod.rs`
3. `src/application/change_targets.rs`
4. the analyzer or gate module you are extending
5. `docs/tooling/raccoon-cli-module-boundaries-and-evolution-rules.md`
