# Stage C13 Advanced Raccoon CLI Architecture Refinement Report

## 1. Executive Summary

Stage C13 completed a surgical refinement pass over `tools/raccoon-cli`.

The main outcome is a cleaner internal split between parsing, application
orchestration, subprocess-backed IO, semantic enrichment, and output ownership,
without redesigning the CLI or moving it closer to the system runtime domain.

The highest-value changes were:

- removal of the residual `cli -> gate` dependency;
- introduction of an explicit `src/io/*` boundary for subprocess-backed support;
- consolidation of repeated optional-LSP orchestration in the application
  layer;
- relocation of LSP-specific human rendering from `application` to `lsp`.

The CLI is now more cohesive and less centrally tangled while remaining small,
direct, and intentionally non-framework-like.

## 2. Remaining Structural Points Found

### Residual point A: parser layer knew too much about gate orchestration

`src/cli/mod.rs` still converted `GateProfile` directly into `gate::Profile`.
That leaked application policy into the parsing layer.

### Residual point B: optional-LSP command handlers repeated the same lifecycle

Several handlers recreated the same `GoplsBridge` setup/shutdown pattern,
keeping orchestration duplicated in `src/application/mod.rs`.

### Residual point C: subprocess-backed support lacked a clear home

Process execution was split between a generic helper, direct `git` shell-outs,
and direct `date` shell-outs, leaving the IO boundary under-defined.

### Residual point D: report-specific output was still too centralized

The human renderer for enriched LSP output lived in `application`, although the
owning module is `lsp`.

## 3. Refactors Applied

### Refactor 1: introduced `src/io/*`

Created:

- `tools/raccoon-cli/src/io/mod.rs`
- `tools/raccoon-cli/src/io/git.rs`
- `tools/raccoon-cli/src/io/process.rs`
- `tools/raccoon-cli/src/io/system.rs`

Applied rewiring:

- `application/change_targets.rs` now uses `io::status_porcelain_paths`;
- `smoke/compose.rs` now uses `io::run_command_with_timeout`;
- `analyzers/snapshot.rs` now uses `io::utc_timestamp`;
- removed `src/process_utils.rs`.

### Refactor 2: moved gate-profile translation into application policy

`GateProfile` parsing stayed in `src/cli/mod.rs`, but conversion to
`gate::Profile` now happens in `src/application/mod.rs`.

### Refactor 3: consolidated optional-LSP orchestration

Added application-local orchestration helpers:

- `with_optional_lsp`
- `with_lsp_mode`

These now back the LSP-aware command paths instead of repeating bridge
lifecycle logic in each handler.

### Refactor 4: moved enriched-symbol human rendering into `lsp`

Created:

- `tools/raccoon-cli/src/lsp/render.rs`

Removed:

- `tools/raccoon-cli/src/application/renderers.rs`

The application layer now selects the output mode only; the LSP module owns the
default human representation of LSP enrichment.

## 4. Refined Final Architecture

The refined module ownership is:

- `main`: process entrypoint only.
- `cli`: command taxonomy, flags, aliases, help text.
- `application`: command dispatch, exit codes, format selection, bounded
  execution helpers.
- `analyzers`: read-only analytical behavior and report production.
- `gate`: bounded guard-rail orchestration.
- `smoke`: legacy smoke helper flow only.
- `lsp`: semantic enrichment types, bridge, and LSP-owned rendering.
- `io`: subprocess-backed repository/system IO helpers.
- `output/models/error`: shared support primitives.

Dependency discipline now more clearly follows:

- parser intent flows into application policy;
- application policy calls analyzers, gate, smoke, lsp, and io;
- analyzers no longer need to be mixed with application-owned helpers just to
  reach common subprocess behavior.

## 5. Future Evolution Rules

### Rule 1

Do not let `src/cli/mod.rs` depend on orchestration modules again.

### Rule 2

Put new subprocess-backed support in `src/io/*`, not in ad hoc command handlers.

### Rule 3

Keep repeated command orchestration in small bounded helpers, not in generic
framework layers.

### Rule 4

Keep report-specific renderers with the module that owns the report or semantic
type.

### Rule 5

Do not let the CLI drift into a runtime control plane; `make` remains the
canonical operational surface.

Detailed rules are captured in
`docs/tooling/raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`.

## 6. Recommended Preparation For C14

The next safe refinement frontier is not a broad rewrite. C14 should stay
incremental and focus on one of these bounded follow-ups:

1. extract a small command-spec inventory or documentation generator only if
   duplicated help/reference maintenance becomes painful in practice;
2. tighten test coverage around `src/io/*` and command-to-module ownership for
   newly added commands;
3. review whether some analyzer-local renderers still deserve relocation to
   their owning modules where application policy is still accidentally involved;
4. assess whether the legacy smoke surface should be further isolated in docs
   and tests, without removing compatibility.

## Validation

Validated with:

- `cargo fmt --manifest-path tools/raccoon-cli/Cargo.toml`
- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml`

Observed result:

- unit tests: `846 passed`
- CLI integration tests: `68 passed`
- validation matrix tests: `97 passed`
