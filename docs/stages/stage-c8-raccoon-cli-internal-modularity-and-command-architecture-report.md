# Stage C8 Report — Raccoon CLI Internal Modularity And Command Architecture

## 1. Executive Summary

Stage C8 hardened the internal architecture of `tools/raccoon-cli` as an
internal repository-support product.

The CLI already had a better external taxonomy after C4, but its internal shape
still concentrated parsing, dispatch, rendering policy, compatibility handling,
and change-target heuristics in `src/main.rs`. C8 corrected that by separating:

- command layer;
- application dispatch layer;
- support helpers for command-owned behavior;
- analyzer and orchestration modules.

The result is a more modular CLI without changing its support-surface role or
turning it into a generic framework.

## 2. Diagnosis Of The Current CLI

Before C8, the CLI had two different states at once:

1. Externally, it already presented a grouped taxonomy (`check`, `inspect`,
   `change`, `legacy` plus the snapshot family).
2. Internally, it still behaved like an accreted single-file binary, because
   `src/main.rs` owned almost all command wiring and several support concerns.

Observed internal composition before the refactor:

- analyzers already existed as distinct modules in `src/analyzers/*`;
- `gate` already existed as a real orchestration use case;
- output, models, LSP, codeintel, smoke, and subprocess helpers already existed
  as support modules;
- but the command layer and the application layer were not explicit modules.

That meant the codebase had useful pieces, but the central CLI flow was still
organized around one high-gravity file.

## 3. Structural Problems Found

### Main-entrypoint overload

`src/main.rs` mixed:

- Clap taxonomy and aliases;
- command dispatch;
- renderer selection;
- exit-code policy;
- `git status` target detection;
- an LSP-specific human renderer;
- internal CLI parsing tests.

That shape made every new command or compatibility rule gravitate back to the
entrypoint.

### Taxonomy existed in UX but not as a stable internal boundary

The grouped command taxonomy was real for users, but the code still treated it
mostly as a parser concern instead of a first-class module boundary.

### Change-target heuristics were opportunistic

The `git status` fallback rules for `impact`, `tdd`, `briefing`, and
`recommend` were implemented as local helpers in `main.rs`, even though they
are shared application behavior.

### Command-owned rendering had no real home

One renderer (`render_enriched_human`) existed outside the LSP support modules
and outside the shared `output` layer, so it also accumulated in `main.rs`.

## 4. Changes Applied

### Code changes in `tools/raccoon-cli`

Added:

- `src/cli/mod.rs`
- `src/application/mod.rs`
- `src/application/change_targets.rs`
- `src/application/renderers.rs`

Changed:

- `src/main.rs` now only parses and delegates
- `tools/raccoon-cli/README.md`

### Internal command-layer extraction

Moved the entire Clap surface into `src/cli/mod.rs`, including:

- grouped command enums;
- argument structs;
- compatibility aliases;
- help text and examples;
- CLI parsing tests.

### Application-layer introduction

Added `src/application/mod.rs` as the single execution/orchestration entry for
parsed commands.

This layer now owns:

- dispatch from commands to analyzers/gate/smoke;
- centralized exit-code policy;
- output-format selection;
- renderer invocation;
- snapshot write-to-file behavior.

### Shared change-target helper

Moved the `git status` fallback logic into
`src/application/change_targets.rs` so all change-oriented commands reuse the
same heuristics.

### Command-owned renderer extraction

Moved the LSP-enriched human renderer into
`src/application/renderers.rs`, keeping it out of both `main.rs` and the shared
report-output module.

### Taxonomy residue cleanup

Removed residual user-facing references to the obsolete `scenario-smoke`
command from CLI guidance and recommendations.

Change-planning and recommendation output now points back to the canonical
Makefile runtime-proof surface (`make smoke`) instead of suggesting a command
that is no longer part of the supported taxonomy.

### Documentation delivered

Created:

- `docs/tooling/raccoon-cli-internal-modularity-and-command-architecture.md`
- `docs/tooling/raccoon-cli-module-boundaries-and-evolution-rules.md`
- `docs/stages/stage-c8-raccoon-cli-internal-modularity-and-command-architecture-report.md`

Updated:

- `tools/raccoon-cli/README.md`
- `docs/tooling/README.md`
- `docs/tooling/cli-overview.md`
- `docs/tooling/cli-architecture-guardrails.md`
- `docs/stages/INDEX.md`

## 5. Final CLI Architecture

The final internal architecture is:

```text
main
  -> cli
  -> application

application
  -> gate
  -> analyzers
  -> smoke
  -> output/models/error
  -> lsp/codeintel support

gate
  -> analyzers + smoke

analyzers
  -> codeintel/lsp/models/error
```

Operationally:

- `main.rs` is now a thin entrypoint;
- `cli` owns taxonomy and compatibility aliases;
- `application` owns dispatch and command execution policy;
- analyzers remain the primary analysis/use-case implementations;
- support modules remain reusable and policy-light.

This improves modularity and readability without redesigning the CLI into a new
platform.

## 6. Recommended Preparation For C9

1. Keep future command additions on the same path: `cli` for parsing and
   `application` for dispatch, never back into `main.rs`.
2. Audit whether more report-specific renderers should live with their owning
   analyzers so `application::renderers` stays small.
3. If the snapshot family grows materially, consider a bounded grouped surface
   for that family, but only if it improves clarity rather than symmetry.
4. If quality-gate grows beyond its current scope, review `gate` as a bounded
   orchestration module instead of letting orchestration leak into unrelated
   analyzers.
5. Preserve the repository-support boundary: new CLI growth should continue to
   assist `make`, docs, and safe change workflows, not compete with the system's
   runtime control surfaces.

## Validation

Validation executed for C8:

- `cargo fmt --manifest-path tools/raccoon-cli/Cargo.toml`
- `cargo test -q --manifest-path tools/raccoon-cli/Cargo.toml`
- `make tdd`
- `make check`
- `make verify`

Observed result:

- the `raccoon-cli` Rust suite passed after the refactor;
- `make tdd` passed and now emits canonical `make smoke` guidance instead of
  the stale `scenario-smoke` recommendation;
- `make check` passed;
- `make verify` failed outside the C8 scope because the current worktree
  already contains unrelated Go test/build failures in:
  - `internal/application/execution/multi_symbol_concurrency_test.go`
  - `internal/application/risk/multi_symbol_concurrency_test.go`

Those `make verify` failures are real and should be fixed, but they are not
caused by the CLI modularity changes delivered in C8.
