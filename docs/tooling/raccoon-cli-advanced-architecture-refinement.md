# Raccoon CLI Advanced Architecture Refinement

## Purpose

This document records the Stage C13 refinement pass for
`tools/raccoon-cli`.

The objective of C13 was not to redesign the CLI. It was to tighten the
remaining internal seams after the previous modularization waves so the CLI can
keep growing as an internal repository-support product with lower entropy risk.

## Scope Of The C13 Pass

The pass focused on four bounded improvements:

1. remove residual parser-to-orchestrator coupling;
2. reduce repeated command orchestration in the application layer;
3. make subprocess and command-line IO boundaries explicit;
4. relocate report-specific presentation to the module that owns the report.

## Residual Structural Frictions Found Before C13

### 1. `cli` still knew about `gate`

The grouped command parser in `src/cli/mod.rs` still imported `crate::gate` to
convert `GateProfile` into `gate::Profile`.

That created a wrong dependency direction:

- the parser layer should describe user intent only;
- the application/orchestration layer should translate parsed intent into
  runtime policy.

### 2. `application::mod` still repeated the same optional-LSP lifecycle

Several command handlers repeated the same flow:

1. decide whether LSP is enabled;
2. create a `GoplsBridge`;
3. run the LSP or non-LSP analyzer path;
4. shut the bridge down.

This was not a correctness bug, but it kept command execution policy more
centralized and repetitive than necessary.

### 3. Process execution was split across unrelated places

Before C13:

- `src/process_utils.rs` held timeout-driven subprocess helpers;
- `src/application/change_targets.rs` directly spawned `git status`;
- `src/analyzers/snapshot.rs` directly spawned `date`;
- `src/smoke/compose.rs` used the generic helper.

The result was a fuzzy IO/process boundary. The CLI had a support module for
subprocesses, but command-oriented modules still reached for `std::process`
directly for common repository/tooling IO.

### 4. LSP human rendering lived in `application`

`src/application/renderers.rs` contained the human renderer for
`lsp::EnrichedSymbol`.

That renderer is not really application policy. It is the natural presentation
surface of the LSP enrichment output, so keeping it under `application`
preserved an unnecessary “central helper” pattern.

## Refactors Applied In C13

### 1. Introduced an explicit CLI IO boundary

New module family:

- `tools/raccoon-cli/src/io/mod.rs`
- `tools/raccoon-cli/src/io/git.rs`
- `tools/raccoon-cli/src/io/process.rs`
- `tools/raccoon-cli/src/io/system.rs`

This module family now owns subprocess-oriented support behavior:

- `git` porcelain status collection;
- timeout-based process execution;
- lightweight system timestamp retrieval.

Code moved or rewired:

- deleted `src/process_utils.rs`;
- `application/change_targets.rs` now uses `io::status_porcelain_paths`;
- `smoke/compose.rs` now uses `io::run_command_with_timeout`;
- `analyzers/snapshot.rs` now uses `io::utc_timestamp`.

The gain is not abstraction for abstraction’s sake. The gain is a clearer rule:
if the CLI needs subprocess-backed repository/system IO, it should land under
`src/io/*`, not be reintroduced ad hoc from application or analyzer modules.

### 2. Removed `cli -> gate` coupling

`GateProfile` remains parsed in `src/cli/mod.rs`, but conversion into
`gate::Profile` now happens in `src/application/mod.rs`.

That puts translation policy where it belongs:

- `cli` parses flags and arguments;
- `application` converts parsed intent into use-case configuration.

### 3. Consolidated optional-LSP orchestration in application helpers

`src/application/mod.rs` now owns two internal helpers:

- `with_optional_lsp`
- `with_lsp_mode`

They centralize the lifecycle contract for commands that optionally enrich
analysis with `gopls`.

Commands simplified by this:

- `inspect contract-usage`
- `change rename`
- `inspect symbol`
- `change briefing`
- `change impact`
- `inspect lsp`

This keeps the command handlers thin without inventing a framework or a command
registry.

### 4. Moved LSP rendering next to the LSP module

New file:

- `tools/raccoon-cli/src/lsp/render.rs`

Deleted:

- `tools/raccoon-cli/src/application/renderers.rs`

`render_enriched_human` is now exported by `crate::lsp`, which matches the
ownership model more cleanly:

- the LSP module owns the enriched symbol type;
- the LSP module owns the default human rendering for that type;
- the application layer only chooses format and dispatches output.

## Refined Internal Module Map After C13

| Layer | Module(s) | Responsibility |
|---|---|---|
| entrypoint | `src/main.rs` | process entry only |
| command surface | `src/cli/mod.rs` | Clap taxonomy, aliases, global flags, help text |
| application policy | `src/application/mod.rs`, `src/application/change_targets.rs` | command dispatch, exit-code policy, command-level orchestration, change target resolution |
| analysis services | `src/analyzers/*` | repository checks, drift analysis, impact analysis, snapshots, recommendations |
| bounded orchestration | `src/gate/mod.rs`, `src/smoke/*` | multi-step guard rails and legacy smoke flow |
| semantic support | `src/codeintel/*`, `src/lsp/*` | AST indexing, semantic enrichment, LSP-owned output for enriched symbols |
| CLI IO support | `src/io/*` | subprocess-backed repository/system IO and timeout helpers |
| common support | `src/output/mod.rs`, `src/models/mod.rs`, `src/error/mod.rs` | shared report rendering, common report types, error transport |

## Final Dependency Direction

```text
main
  -> cli
  -> application

application
  -> analyzers
  -> gate
  -> smoke
  -> lsp
  -> io
  -> output/models/error

gate
  -> analyzers
  -> smoke
  -> output/models/error

analyzers
  -> codeintel
  -> lsp
  -> io (only when analysis truly needs subprocess-backed support)
  -> models/error

smoke
  -> io
  -> models/error

cli
  -> output (format enum only)
```

## What C13 Explicitly Did Not Do

C13 deliberately did not:

- turn the CLI into a plugin framework;
- add command registries or trait-object dispatch;
- move domain/runtime concerns into the CLI;
- reclassify the legacy smoke flow as a canonical runtime control plane;
- rename modules broadly for cosmetic reasons.

## Architectural Outcome

After C13, the CLI is still recognizably the same product, but with clearer
internal ownership:

- parser concerns stop at parsing;
- application concerns own execution policy and format selection;
- LSP owns LSP output;
- subprocess-backed support has a clear home;
- repeated orchestration logic is reduced without hiding behavior behind generic
  abstractions.

That is the intended maturity target for this stage: more discipline, less
central friction, and better extensibility without a rewrite.
