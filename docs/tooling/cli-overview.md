# Raccoon CLI Overview

## Purpose

`raccoon-cli` is the repository support CLI for `market-foundry`.

It exists to help contributors validate architecture, inspect structural relationships, and plan safe repository changes. It is the repository's strategic intelligence layer, not a product-facing surface, and must not become a parallel operational platform for the live tranche.

## Support-Surface Boundary

- `make` remains the canonical public entrypoint for repository workflows such as `make check`, `make tdd`, `make verify`, `make smoke`, and `make up`.
- Direct `raccoon-cli` usage is the expert support surface when you need narrower inspection, JSON output, tooling-specific work, or explicit impact/drift/architecture intelligence.
- Direct CLI usage should complement, not replace, Makefile-backed runtime and operational flows.
- The operational proof-of-record surface is `make smoke*`; `quality-gate --profile deep` and `legacy runtime-smoke` remain tolerated compatibility helpers only.

Operational contract:

- `make` owns public workflow and proof entrypoints.
- `raccoon-cli` owns inspection, impact analysis, TDD guidance, drift detection, and architecture safety.
- `scripts/*.sh` stay as execution detail behind `make`.

## Canonical Taxonomy

| Group | Scope | Canonical examples |
|---|---|---|
| `check` | Repository guard rails and audits | `raccoon-cli check repo`, `raccoon-cli check gate --profile ci` |
| `inspect` | Read-only structural and contract analysis | `raccoon-cli inspect symbol ConfigSet`, `raccoon-cli inspect coverage` |
| `change` | Impact mapping and validation guidance | `raccoon-cli change tdd`, `raccoon-cli change recommend` |
| `snapshot` family | Baselines, diffs, and drift | `raccoon-cli snapshot -o baseline.json`, `raccoon-cli snapshot-diff before.json after.json` |
| `legacy` | Deprecated or fragile helpers | `raccoon-cli legacy runtime-smoke` |

## Command Lifecycle

The taxonomy answers "what kind of task is this?". The lifecycle answers "how
durable and promotable is this command surface?".

| Lifecycle state | Meaning | Current Raccoon CLI surface |
|---|---|---|
| `stable core` | Default, documented surface for recurring repository workflows | `check`, `inspect`, `change` |
| `stable utility` | Durable but narrower support surface for focused expert work | `snapshot`, `snapshot-diff`, `baseline-drift` |
| `experimental` | Proving-only surface with bounded scope and explicit promotion criteria | none currently promoted |
| `legacy` | Compatibility-only or deprecated surface retained to avoid abrupt breaks | `legacy runtime-smoke`, hidden flat aliases |

The public help surface should make these distinctions visible enough that
contributors can tell which commands are normal, which are specialist tools,
and which are only tolerated for compatibility.

## Compatibility

Historic flat commands such as `doctor`, `quality-gate`, `symbol-trace`, `impact-map`, `tdd`, and `runtime-smoke` remain supported as hidden compatibility aliases. New documentation and examples should prefer the canonical grouped taxonomy, and should not present those aliases as the canonical operational-proof surface.

## Protected Taxonomy Contract

The taxonomy is protected only at the level that materially affects workflow
convergence:

- top-level user-facing groups remain `check`, `inspect`, `change`, and
  `legacy`;
- Make-backed public wrappers should keep pointing to grouped commands instead
  of compatibility aliases;
- runtime proof remains documented through `make smoke*`, not through
  `quality-gate --profile deep` or `legacy runtime-smoke`.

This is a convergence guard, not a freeze on internal CLI implementation or on
specialized analyzer text.

## Internal Architecture

After Stage C8, the internal CLI shape is explicitly layered:

- `src/cli/mod.rs` owns Clap parsing, grouped taxonomy, and compatibility aliases.
- `src/application/mod.rs` owns command dispatch, output policy, and exit-code policy.
- `src/application/change_targets.rs` owns shared auto-detected change-target heuristics.
- `src/analyzers/*` and `src/gate/mod.rs` own repository analysis and guard-rail orchestration.
- support modules (`output`, `models`, `error`, `lsp`, `codeintel`, `smoke`, `process_utils`) stay below those command/application concerns.

This keeps the CLI sustainable as an internal product without turning it into a generic command framework.

## Primary References

- [`docs/operations/raccoon-cli-ux-taxonomy-and-guard-rails.md`](../operations/raccoon-cli-ux-taxonomy-and-guard-rails.md)
- [`docs/operations/raccoon-cli-command-reference.md`](../operations/raccoon-cli-command-reference.md)
- [`docs/operations/make-and-raccoon-cli-contract.md`](../operations/make-and-raccoon-cli-contract.md)
- [`raccoon-cli-command-lifecycle-and-deprecation-strategy.md`](raccoon-cli-command-lifecycle-and-deprecation-strategy.md)
- [`raccoon-cli-command-catalog-maturity-model-and-governance.md`](raccoon-cli-command-catalog-maturity-model-and-governance.md)
- [`docs/tooling/development-cli-reliability-and-command-testing-strategy.md`](development-cli-reliability-and-command-testing-strategy.md)
- [`docs/tooling/raccoon-cli-command-trustworthiness-and-error-semantics.md`](raccoon-cli-command-trustworthiness-and-error-semantics.md)
- [`docs/tooling/raccoon-cli-internal-modularity-and-command-architecture.md`](raccoon-cli-internal-modularity-and-command-architecture.md)
- [`docs/tooling/raccoon-cli-module-boundaries-and-evolution-rules.md`](raccoon-cli-module-boundaries-and-evolution-rules.md)
- [`docs/stages/stage-c4-raccoon-cli-ux-command-taxonomy-and-guard-rails-report.md`](../stages/stage-c4-raccoon-cli-ux-command-taxonomy-and-guard-rails-report.md)
