# raccoon-cli

Strategic repository-intelligence toolkit for `market-foundry`. Fully isolated from the Go runtime — reads files, configs, and source; executes subprocesses only for bounded support checks.

## Build & Test

```sh
cd tools/raccoon-cli
cargo build --release
cargo test
```

## Quick Start

```sh
# From the project root:
raccoon-cli check repo                           # stable core: repository structure check
raccoon-cli check gate                           # stable core: fast static checks (default)
raccoon-cli inspect symbol ConfigSet --lsp       # stable core: expert inspection
raccoon-cli change tdd                           # stable core: change-planning guidance
raccoon-cli snapshot --output baseline.json      # stable utility: baseline capture
```

Prefer `make check`, `make tdd`, `make verify`, and `make smoke*` for the
repository workflow contract. Use direct `raccoon-cli` commands when you need
expert inspection depth or when working on the tooling layer itself.
The deep quality-gate profile and `runtime-smoke` compatibility helper are not
the canonical operational-proof surface; the proof-of-record runtime entrypoints
remain the `make smoke*` targets.

Contract summary:

- `make` owns the stable public workflow and runtime/proof entrypoints.
- `raccoon-cli` owns expert inspection, impact analysis, TDD guidance, drift detection, and architecture safety.
- `scripts/*.sh` stay behind `make` unless you are debugging harness behavior.

## Internal Structure

After Stage C8, the CLI is organized around explicit internal layers:

- `src/cli/mod.rs` — command layer, taxonomy, aliases, and help text
- `src/application/mod.rs` — command dispatch, exit-code policy, and renderer selection
- `src/application/change_targets.rs` — shared `git status` target detection for change-oriented commands
- `src/analyzers/*` and `src/gate/mod.rs` — analysis and orchestration logic
- `src/output/mod.rs`, `src/models/mod.rs`, `src/error/mod.rs`, `src/lsp/*`, `src/codeintel/*`, `src/smoke/*` — support modules

See:

- `docs/tooling/raccoon-cli-internal-modularity-and-command-architecture.md`
- `docs/tooling/raccoon-cli-module-boundaries-and-evolution-rules.md`

## Command Lifecycle

The CLI is governed as a development tool, not as an ever-growing product
surface.

| Lifecycle state | Meaning | Current surface |
|---------|---------|---------|
| `stable core` | Default supported commands for recurring repository work | `check`, `inspect`, `change` |
| `stable utility` | Narrow but durable support flows used for focused analysis | `snapshot`, `snapshot-diff`, `baseline-drift` |
| `experimental` | Proving-only commands not yet ready for broad promotion | none currently promoted |
| `legacy` | Compatibility-only or deprecated helper flows | `legacy runtime-smoke`, hidden flat aliases |

Prefer the grouped taxonomy. Hidden flat commands remain only as compatibility
aliases and should not be used in new docs or examples.

## Canonical Commands

### `check` — stable core guard rails

| Command | Purpose |
|---------|---------|
| `raccoon-cli check repo` | Validate project structure (go.work, modules, configs, compose) |
| `raccoon-cli check topology` | Validate service topology (configs, compose, streams, subjects) |
| `raccoon-cli check contracts` | Audit messaging contracts (NATS subjects, event types, envelope) |
| `raccoon-cli check bindings` | Validate runtime binding alignment |
| `raccoon-cli check arch` | Enforce architecture layer boundaries (11 rules) |
| `raccoon-cli check drift` | Detect cross-layer semantic drift (naming, docs, config, compose) |
| `raccoon-cli check gate` | Run the consolidated guard-rail profile behind `make check*` |

### `inspect` — stable core expert inspection

| Command | Purpose |
|---------|---------|
| `raccoon-cli inspect symbol <SYMBOL>` | Trace symbol definitions, references, and contracts |
| `raccoon-cli inspect lsp <SYMBOL>` | Semantic enrichment via `gopls` |
| `raccoon-cli inspect contract-usage` | Map contract definition, propagation, and consumption |
| `raccoon-cli inspect coverage` | Show quality coverage map and gaps |

### `change` — stable core change guidance

| Command | Purpose |
|---------|---------|
| `raccoon-cli change impact [TARGET...]` | Map change impact across modules |
| `raccoon-cli change tdd [TARGET...]` | TDD guide for current changes and validation sequence |
| `raccoon-cli change briefing [TARGET...]` | Generate briefing for targets or active diff |
| `raccoon-cli change recommend [TARGET...]` | Recommend validation after a change |
| `raccoon-cli change rename <SYMBOL>` | Assess rename risk before editing |

### Stable utility commands

| Command | Purpose |
|---------|---------|
| `raccoon-cli snapshot` | Generate code intelligence snapshot (JSON) |
| `raccoon-cli snapshot-diff` | Compare two snapshots |
| `raccoon-cli baseline-drift` | Detect drift from baseline snapshot |

### Legacy compatibility

| Command | Purpose |
|---------|---------|
| `raccoon-cli legacy runtime-smoke` | Deprecated helper retained only for compatibility |
| flat aliases such as `doctor`, `quality-gate`, `symbol-trace` | Hidden compatibility entrypoints for existing consumers |

## Architecture Guardrails

The CLI enforces these architectural invariants:

1. **Layer boundaries** — domain has no infrastructure imports; dependencies flow inward
2. **Service topology** — configs, compose, and source code agree on services, streams, subjects
3. **Naming identity** — no residual "server" references where "gateway" is canonical
4. **Contract alignment** — NATS registry specs match domain event definitions
5. **Docs-reality alignment** — architecture docs match actual binary/service structure

## Current Structural Assumptions

The current checks are aligned to the post-S218/S219/S220 repository shape:

- NATS adapters are organized by sub-package under `internal/adapters/nats/` (`natsevidence/`, `natssignal/`, `natsdecision/`, `natsstrategy/`, `natsrisk/`, `natsexecution/`, `natsobservation/`, `natsconfigctl/`, plus `natskit/`)
- Registry discovery accepts both legacy `*_registry.go` files and the current `*/registry.go` layout
- Durable consumer discovery recognizes both explicit `ConsumerSpec{...}` blocks and `natskit.NewConsumerSpec(...)` factory calls
- Store-side consumer wiring is validated through `internal/actors/scopes/store/generic_consumer_actor.go` and `internal/actors/scopes/store/store_supervisor.go`, not deleted per-domain consumer actor wrappers

## Output Formats

All commands support `--json` for machine-readable output and `-v` for verbose mode.

## Deprecation Notes

- `raccoon-cli legacy runtime-smoke` is retained only as a bounded compatibility
  helper and must not be promoted as the runtime proof-of-record surface.
- Flat aliases remain hidden compatibility entrypoints and should not appear in
  new guidance except when documenting migration or compatibility behavior.
