# Raccoon CLI Command Reference

## Purpose

This reference describes the canonical command surface for the repository support CLI in `market-foundry`.

The preferred model is grouped usage. Historic flat commands remain available as compatibility aliases.

Boundary rule:

- `make` is the public workflow contract.
- `raccoon-cli` is the expert strategic-intelligence surface behind and beside that workflow.
- Runtime proof still belongs to `make smoke*`, not to CLI deep/legacy helpers.

## Lifecycle At A Glance

| Lifecycle state | Meaning | Current surface |
|---|---|---|
| `stable core` | Default supported surface for recurring repository work | `check`, `inspect`, `change` |
| `stable utility` | Narrow but durable support flows | `snapshot`, `snapshot-diff`, `baseline-drift` |
| `experimental` | Bounded proving surface not yet promoted in public help | none currently promoted |
| `legacy` | Deprecated or compatibility-only surface | `legacy runtime-smoke`, hidden flat aliases |

## Global Usage

```bash
raccoon-cli [--project-root PATH] [--json] [--verbose] <command>
```

### Global Flags

| Flag | Meaning |
|---|---|
| `--project-root PATH` | Analyze a different repository root |
| `--json` | Emit JSON instead of human-readable output |
| `--verbose`, `-v` | Show expanded detail in human output |

## `check`

```bash
raccoon-cli check <subcommand>
```

| Subcommand | Alias | Use when |
|---|---|---|
| `repo` | `doctor` | You need a fast repository structure sanity check |
| `topology` | `topology-doctor` | You changed topology, compose, service declarations, or stream ownership |
| `contracts` | `contract-audit` | You touched message contracts, subjects, or envelope usage |
| `bindings` | `runtime-bindings` | You changed config-to-runtime routing or config-bound consumers |
| `arch` | `arch-guard` | You changed package boundaries, imports, or layering |
| `drift` | `drift-detect` | You need docs/config/source drift detection |
| `gate` | `quality-gate` | You want the consolidated guard-rail run |

### `check gate`

```bash
raccoon-cli check gate --profile <fast|ci|deep> [--fail-fast] [--base-url URL]
```

- `fast`: default static repository checks
- `ci`: static checks with warnings promoted to failures
- `deep`: static checks plus the legacy runtime smoke helper

Prefer `make smoke` and related Makefile workflows for operational runtime proof. `deep` is for compatibility, not for expanding the CLI into a runtime platform.

## `inspect`

```bash
raccoon-cli inspect <subcommand>
```

| Subcommand | Alias | Use when |
|---|---|---|
| `symbol <SYMBOL>` | `symbol-trace` | You need a structural trace for a symbol |
| `lsp <SYMBOL>` | `lsp-enrich` | You want `gopls`-enriched symbol output |
| `contract-usage` | `contract-usage-map` | You need contract flow visibility |
| `coverage` | `coverage-map` | You want guard-rail and test coverage visibility |

### LSP-Constrained Commands

These commands support optional LSP enrichment:

- `inspect symbol`
- `inspect lsp`
- `inspect contract-usage`
- `change impact`
- `change briefing`
- `change rename`

Guard rail:

- `--lsp` conflicts with `--no-lsp`

## `change`

```bash
raccoon-cli change <subcommand> [TARGET...]
```

| Subcommand | Alias | Use when |
|---|---|---|
| `impact` | `impact-map` | You want structural blast-radius mapping |
| `tdd` | `tdd` | You want before/after validation guidance |
| `briefing` | `briefing` | You want a compact, auditable summary for an area |
| `recommend` | `recommend` | You want prioritized validation suggestions |
| `rename <SYMBOL>` | `rename-safety` | You want pre-edit rename risk assessment |

### Auto-Detection Behavior

When `TARGET...` is omitted for `impact`, `tdd`, `briefing`, or `recommend`, the CLI uses `git status`.

The auto-detected path set now:

- keeps the current path for renames;
- ignores deleted files;
- filters out documentation-only paths when structural targets are also present.

If you want documentation paths analyzed intentionally, pass them explicitly.

## Snapshot And Baseline Commands

| Command | Purpose |
|---|---|
| `raccoon-cli snapshot [-o OUTPUT_JSON]` | Generate a deterministic structural baseline |
| `raccoon-cli snapshot-diff BEFORE_JSON AFTER_JSON` | Compare two baselines |
| `raccoon-cli snapshot-diff BEFORE_JSON --after-live` | Compare a baseline against the current worktree |
| `raccoon-cli baseline-drift BASELINE_JSON` | Detect drift against a saved baseline |

## `legacy`

```bash
raccoon-cli legacy runtime-smoke [--base-url URL]
```

This exists only to preserve a historical helper path.

Prefer:

- `make smoke`
- `make smoke-multi`
- `make smoke-restart-recovery`

## Compatibility Aliases

Hidden flat aliases such as `doctor`, `quality-gate`, `symbol-trace`,
`impact-map`, and `tdd` remain supported so existing local scripts and operator
habits do not break abruptly.

Governance rule:

- new docs and examples should use the grouped canonical commands;
- aliases should remain behaviorally identical to their canonical command;
- aliases are compatibility surfaces, not a second taxonomy.

## Recommended Operator Flows

### Fast Pre-Change Guard Rail

```bash
make check
raccoon-cli check gate
```

### Structural Investigation

```bash
raccoon-cli inspect symbol ConfigSet --lsp
raccoon-cli inspect contract-usage
raccoon-cli inspect coverage
```

### Safe Change Planning

```bash
make tdd
raccoon-cli change impact
raccoon-cli change briefing
raccoon-cli change recommend
```

### Baseline And Drift Review

```bash
raccoon-cli snapshot -o baseline.json
raccoon-cli baseline-drift baseline.json
```

## Related Governance

- [`make-and-raccoon-cli-contract.md`](make-and-raccoon-cli-contract.md)
- [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
- [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md)
