# Raccoon CLI Overview

**Status**: Architecture guardian for market-foundry

## Identity

The CLI is named `raccoon-cli` and lives in `tools/raccoon-cli/`. It is a Rust binary that performs static analysis, architecture enforcement, and code intelligence over the Go workspace without importing any Go runtime code.

## Architecture Enforcement Commands

| Command | Purpose | Guard Rail |
|---------|---------|-----------|
| `doctor` | Validate repository structure | go.work, modules, configs, compose |
| `topology-doctor` | Validate service topology | configs, compose, streams, subjects |
| `contract-audit` | Audit messaging contracts | NATS subjects, event types, envelope |
| `runtime-bindings` | Validate runtime bindings | config-to-stream routing alignment |
| `arch-guard` | Enforce layer boundaries | 11 rules: domain purity, import direction, port leaks |
| `drift-detect` | Detect architectural drift | naming, docs, config, compose, binary, signal alignment |

## Coverage and Planning Commands

| Command | Purpose |
|---------|---------|
| `coverage-map` | Show quality coverage map and identify gaps |
| `tdd` | TDD guide — what to validate for current changes |
| `impact-map` | Map impact of changes across modules |
| `recommend` | Smart recommendations from diff/baseline analysis |
| `briefing` | Generate briefing for specified targets |

## Code Intelligence Commands

| Command | Purpose |
|---------|---------|
| `symbol-trace` | Trace symbol definitions, references, contracts |
| `contract-usage-map` | Map contract definition, propagation, consumption |
| `rename-safety` | Assess rename risk before executing |
| `lsp-enrich` | Semantic enrichment via gopls |

## Change Analysis Commands

| Command | Purpose |
|---------|---------|
| `snapshot` | Generate golden snapshot of code intelligence (JSON) |
| `snapshot-diff` | Compare two snapshots |
| `baseline-drift` | Detect drift against a baseline snapshot |

## Quality Orchestration

| Command | Purpose |
|---------|---------|
| `quality-gate` | Run consolidated quality checks (profiles: fast, ci, deep) |

## Deprecated Commands

These commands are legacy quality-service artifacts and are no longer functional:

| Command | Replacement |
|---------|------------|
| `runtime-smoke` | `make smoke` / `make smoke-multi` |
| `scenario-smoke` | `make smoke` / `make smoke-multi` |
| `results-inspect` | No replacement (validator removed) |
| `trace-pack` | No replacement |

## Build and Test

```bash
make raccoon-build    # Build release binary
make raccoon-test     # Run Rust tests
```

## Workflow Integration

```bash
make check            # Pre-code guard rail (quality-gate fast)
make verify           # Post-change: Go tests + quality-gate
make check-deep       # Full validation
make arch-guard       # Architecture boundary check
make drift-detect     # Cross-layer drift detection
make tdd              # TDD guide
make recommend        # Smart recommendations
```

## Architectural Invariants Protected

The CLI enforces these invariants from the canonical architecture documents:

1. **Five-binary ceiling** — only configctl, gateway, ingest, derive, store exist as service binaries
2. **Layer sovereignty** — domain imports nothing from infrastructure; dependencies flow inward only
3. **Gateway is stateless** — no domain logic, no repositories, no event publishing in gateway
4. **Single stream ownership** — each JetStream stream has exactly one producer binary
5. **Naming identity** — no "server" where "gateway" is canonical; no old service names in active code
6. **Docs-code alignment** — architecture docs, compose, configs, and source code agree
7. **Signal domain governance** — signal subjects, durables, KV buckets, adapters, actors, docs, and config symmetry are audited (see [cli-signal-guardrails.md](cli-signal-guardrails.md))
