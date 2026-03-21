# Stage C4 Report — Raccoon CLI UX, Command Taxonomy, And Guard Rails

## Executive Summary

Stage C4 hardened the Raccoon CLI as a repository support tool.

The CLI now presents a canonical grouped command taxonomy, keeps historical flat commands as hidden compatibility aliases, improves help text and discoverability, tightens flag guard rails, and reduces noisy auto-detected change inputs. The result is a clearer operational surface without turning the CLI into a new platform.

## Initial Diagnosis

The pre-C4 surface was functional but operationally uneven:

1. The help surface was flat, mixing checks, inspections, change-planning tools, baseline utilities, and legacy helpers with no intent grouping.
2. Discoverability depended on memorizing historical names such as `doctor`, `topology-doctor`, `contract-usage-map`, and `rename-safety`.
3. The CLI still surfaced `runtime-smoke` as a normal first-class command even though the repository already prefers Makefile-backed runtime validation.
4. Commands with optional LSP enrichment allowed ambiguous intent (`--lsp` together with `--no-lsp`).
5. `git status` fallback flows could include rename/delete noise or documentation-only paths that were not useful for structural analysis.

## Changes Implemented

### Canonical Taxonomy

Introduced grouped top-level commands:

- `check`
- `inspect`
- `change`
- `legacy`

Retained `snapshot`, `snapshot-diff`, and `baseline-drift` as a coherent top-level baseline family.

### Compatibility Model

Kept the historical flat commands as hidden compatibility aliases:

- `doctor`
- `topology-doctor`
- `contract-audit`
- `runtime-bindings`
- `arch-guard`
- `drift-detect`
- `quality-gate`
- `symbol-trace`
- `lsp-enrich`
- `contract-usage-map`
- `coverage-map`
- `impact-map`
- `tdd`
- `briefing`
- `recommend`
- `rename-safety`
- `runtime-smoke`

This avoids gratuitous breakage while letting new documentation and help text converge on the grouped model.

### Help And Usage Improvements

Updated the root help text to:

- state the CLI identity as repository support tooling;
- explicitly reject product/control-plane drift;
- show the canonical taxonomy;
- document compatibility aliases.

Added grouped help text and examples for:

- `check`
- `inspect`
- `change`
- `legacy`

Preserved example-oriented help for hidden flat aliases that contributors may still invoke directly.

### Guard Rails Added

Added the following guard rails:

1. Conflicting `--lsp` and `--no-lsp` flags now fail fast on parsing.
2. `runtime-smoke` is canonically contained under `legacy`.
3. Deep quality-gate help now points operators back to Makefile runtime flows.
4. Auto-detected change inputs now parse renames correctly, ignore deletions, and filter documentation-only paths when structural targets are also present.

## Documentation Delivered

Created:

- `docs/operations/raccoon-cli-ux-taxonomy-and-guard-rails.md`
- `docs/operations/raccoon-cli-command-reference.md`
- `docs/stages/stage-c4-raccoon-cli-ux-command-taxonomy-and-guard-rails-report.md`

Updated:

- `docs/tooling/cli-overview.md`

## Validation

Validated the CLI with:

```bash
cargo test -q --manifest-path tools/raccoon-cli/Cargo.toml
cargo test -q --manifest-path tools/raccoon-cli/Cargo.toml --test cli_integration
cargo test -q --manifest-path tools/raccoon-cli/Cargo.toml --test validation_matrix
```

Observed result:

- full Rust test suite passed;
- CLI integration suite passed;
- validation matrix suite passed.

## Outcome Against Acceptance Criteria

### Clearer And More Consistent CLI

Met.

The main help surface now leads with grouped intent instead of a flat historical command list.

### More Predictable Subcommands And Flags

Met.

The grouped taxonomy standardizes where users look for checks, inspections, and change-planning helpers. LSP-related flag ambiguity is now rejected instead of silently tolerated.

### Better Discoverability

Met.

The root help is now readable as an operator/developer map instead of a raw command dump.

### Real Support-Tool Value

Met.

The CLI remains aligned with repository operation and maintenance. It gained stronger command semantics and usage guidance without expanding into live-domain orchestration.

## Residual Follow-Ups

Optional next improvements:

1. Move the snapshot family under a dedicated grouped surface if the command family grows beyond three commands.
2. Add targeted JSON-schema contract tests for grouped canonical commands in addition to flat aliases.
3. Consider harmonizing future Makefile help and CLI help wording so both surfaces describe the same support taxonomy verbatim.
