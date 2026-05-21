# Raccoon CLI UX Taxonomy And Guard Rails

## Intent

The Raccoon CLI exists to support repository operation, maintenance, validation, and safe change execution inside `market-foundry`.

This document defines the UX model for the CLI after the C4 hardening pass. The CLI remains strictly in the tooling/support lane. It does not become a second operational product surface and it does not own live domain workflows that already belong to the Makefile or service runtime.

## Diagnosis Of The Pre-C4 Surface

Before this pass, the CLI exposed nearly every command at the top level.

That shape worked, but it had four recurring UX problems:

1. Command discoverability was weak because checks, inspection tools, change-planning helpers, baseline utilities, and legacy helpers all appeared in one flat list.
2. Naming conventions were inconsistent because the user had to remember whether a command was a `*-doctor`, `*-map`, `*-safety`, or `*-gate` command without any grouping context.
3. Guard rails around fragile behavior were too soft, especially around `runtime-smoke` and around ambiguous flag combinations such as `--lsp` with `--no-lsp`.
4. Auto-detected change workflows (`impact-map`, `tdd`, `briefing`, `recommend`) could produce noisy results when `git status` included documentation-only changes or rename/delete entries.

## Canonical UX Model

The canonical surface is now grouped by intent.

### `check`

Repository guard rails and audits.

| Canonical command | Compatibility alias | Purpose |
|---|---|---|
| `raccoon-cli check repo` | `doctor` | Validate repository structure and required support paths |
| `raccoon-cli check topology` | `topology-doctor` | Validate topology across source, configs, and compose |
| `raccoon-cli check contracts` | `contract-audit` | Audit messaging and contract invariants |
| `raccoon-cli check bindings` | `runtime-bindings` | Validate config-to-routing binding alignment |
| `raccoon-cli check arch` | `arch-guard` | Enforce architectural dependency boundaries |
| `raccoon-cli check drift` | `drift-detect` | Detect repo drift between docs, source, and runtime declarations |
| `raccoon-cli check gate` | `quality-gate` | Run the consolidated guard-rail profile |

### `inspect`

Read-only structural and contract analysis.

| Canonical command | Compatibility alias | Purpose |
|---|---|---|
| `raccoon-cli inspect symbol <SYMBOL>` | `symbol-trace` | Trace a symbol across definitions, references, and contracts |
| `raccoon-cli inspect lsp <SYMBOL>` | `lsp-enrich` | Enrich structural output with `gopls` data |
| `raccoon-cli inspect contract-usage` | `contract-usage-map` | Map definition, propagation, and consumption of contracts |
| `raccoon-cli inspect coverage` | `coverage-map` | Show guard-rail/test coverage across sensitive areas |

### `change`

Impact mapping and validation guidance for pending work.

| Canonical command | Compatibility alias | Purpose |
|---|---|---|
| `raccoon-cli change impact [TARGET...]` | `impact-map` | Estimate structural blast radius |
| `raccoon-cli change tdd [TARGET...]` | `tdd` | Generate disciplined before/after validation guidance |
| `raccoon-cli change briefing [TARGET...]` | `briefing` | Produce a dense operator/developer briefing |
| `raccoon-cli change recommend [TARGET...]` | `recommend` | Recommend checks, scenarios, and gate depth |
| `raccoon-cli change rename <SYMBOL>` | `rename-safety` | Evaluate rename risk before editing shared surfaces |

### Snapshot Family

These remain top-level because they are already coherent as a small baseline/diff family and do not benefit from additional nesting.

| Command | Purpose |
|---|---|
| `raccoon-cli snapshot` | Capture a deterministic repository baseline |
| `raccoon-cli snapshot-diff` | Compare two baselines |
| `raccoon-cli baseline-drift` | Detect structural drift against a saved baseline |

### `legacy`

Deprecated or fragile helper flows.

| Canonical command | Compatibility alias | Purpose |
|---|---|---|
| `raccoon-cli legacy runtime-smoke` | `runtime-smoke` | Historical runtime smoke wrapper kept only for compatibility |

## UX Conventions

### Command Naming

- Canonical commands should read as `group + action/object`.
- New documentation should prefer grouped commands over flat aliases.
- Flat aliases remain for backward compatibility, not as the primary UX.

### Global Flags

Global flags remain consistent across the CLI:

- `--project-root`
- `--json`
- `--verbose`

### LSP Flags

Commands that support enrichment now reject `--lsp` and `--no-lsp` when both are supplied together. The CLI now treats that as an invalid invocation instead of silently picking one.

### Auto-Detected Change Inputs

For change-oriented commands that fall back to `git status`, the CLI now:

1. Parses rename entries into the current path instead of returning `old -> new`.
2. Ignores deleted entries for structural analysis.
3. Filters out documentation-only changes when structural targets are also present.

This keeps `impact`, `tdd`, `briefing`, and `recommend` focused on actionable structural inputs without hiding explicit user-provided targets.

## Guard Rails Added In C4

### Support-Surface Guard Rail

The root help text now describes the CLI as a repository support tool and explicitly states that it must not become a product control plane.

### Legacy Containment Guard Rail

`runtime-smoke` is no longer promoted in the main help surface. It is canonically documented under `legacy runtime-smoke`, and the deep quality-gate help now points contributors toward Makefile-backed operational flows for actual runtime proof.

### Ambiguous-Flag Guard Rail

Commands with optional LSP enrichment now fail fast on conflicting `--lsp` and `--no-lsp` usage.

### Discoverability Guard Rail

The main help surface is now organized by user intent instead of by historical command accretion:

- `check`
- `inspect`
- `change`
- `snapshot` family
- `legacy`

### Change-Detection Guard Rail

Auto-detected change flows now avoid polluting structural analysis with documentation-only files when code/config targets already exist in the same worktree.

## Non-Goals

This taxonomy does not authorize:

- replacing Makefile-backed runtime flows with new CLI product-style orchestration;
- turning the CLI into a runtime operator for the live domain;
- introducing gratuitous breaking changes to existing ergonomics;
- expanding the CLI into unrelated feature work outside repository support.
