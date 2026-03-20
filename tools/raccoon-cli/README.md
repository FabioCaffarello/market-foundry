# raccoon-cli

Architecture guardian toolkit for `market-foundry`. Fully isolated from the Go runtime — reads files, configs, and source; executes subprocesses only for compose status checks.

## Build & Test

```sh
cd tools/raccoon-cli
cargo build --release
cargo test
```

## Quick Start

```sh
# From the project root:
raccoon-cli doctor                           # project structure check
raccoon-cli quality-gate                     # fast static checks (default)
raccoon-cli quality-gate --profile ci --json # CI pipeline
raccoon-cli quality-gate --profile deep      # full validation
```

## Commands

### Architecture Enforcement

| Command | Purpose |
|---------|---------|
| `doctor` | Validate project structure (go.work, modules, configs, compose) |
| `topology-doctor` | Validate service topology (configs, compose, streams, subjects) |
| `contract-audit` | Audit messaging contracts (NATS subjects, event types, envelope) |
| `runtime-bindings` | Validate runtime binding alignment |
| `arch-guard` | Enforce architecture layer boundaries (11 rules) |
| `drift-detect` | Detect cross-layer semantic drift (naming, docs, config, compose) |

### Coverage and Planning

| Command | Purpose |
|---------|---------|
| `coverage-map` | Show quality coverage map and gaps |
| `tdd` | TDD guide for current changes |
| `impact-map` | Map change impact across modules |
| `recommend` | Smart recommendations from diff/baseline |
| `briefing` | Generate briefing for targets |

### Code Intelligence

| Command | Purpose |
|---------|---------|
| `symbol-trace` | Trace symbol definitions, references, contracts |
| `contract-usage-map` | Map contract definition, propagation, consumption |
| `rename-safety` | Assess rename risk before executing |
| `lsp-enrich` | Semantic enrichment via gopls |

### Change Analysis

| Command | Purpose |
|---------|---------|
| `snapshot` | Generate code intelligence snapshot (JSON) |
| `snapshot-diff` | Compare two snapshots |
| `baseline-drift` | Detect drift from baseline snapshot |

### Quality Orchestration

| Command | Purpose |
|---------|---------|
| `quality-gate` | Run quality checks (profiles: fast, ci, deep) |

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

## Deprecated Commands

The following commands are legacy quality-service artifacts and are no longer functional:
- `runtime-smoke` — replaced by `make smoke` / `make smoke-multi`
- `scenario-smoke` — replaced by `make smoke` / `make smoke-multi`
- `results-inspect` — no longer applicable (validator removed)
- `trace-pack` — no longer applicable
