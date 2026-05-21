# market-foundry

Foundation repository for the market-foundry system.

## Status

**Post first-slice phase** — the repository has completed sanitization, first vertical slice (observation → evidence → store → query), and architectural recentralization. Multi-symbol scalability proven via config-only changes.

## What This Repository Contains

- **Go workspace** with isolated internal modules following clean architecture layers
- **Config lifecycle management** (configctl) — create, validate, compile, activate configuration versions
- **HTTP API gateway** (gateway) — stateless HTTP→NATS translation layer
- **Market data capture** (ingest) — WebSocket → observation events
- **Evidence and decision derivation** (derive) — observation → evidence/signal/decision/strategy/risk/execution events
- **Read model materialization** (store) — domain events → NATS KV projections + query serving
- **Execution control** (execute) — controlled execution intake and fill-state handling
- **Analytical write path** (writer + ClickHouse) — domain events → analytical storage
- **Schema migration utility** (`cmd/migrate`) — forward-only ClickHouse schema management
- **Actor-based orchestration** using the Hollywood framework
- **NATS messaging** for inter-service communication
- **Rust CLI** (raccoon-cli) for strategic repository intelligence: inspection, impact analysis, architecture enforcement, and quality gates
- **Docker Compose** setup for local development

## Architecture

```
cmd/
  configctl/          Config lifecycle service
  execute/            Execution control service
  gateway/            HTTP API gateway (stateless HTTP→NATS translator)
  ingest/             Market data capture (WebSocket → observation events)
  derive/             Derivation pipeline (observation → downstream domain events)
  migrate/            Forward-only ClickHouse migration tool
  store/              Read model materialization (domain events → KV projections)
  writer/             Analytical writer (domain events → ClickHouse rows)

internal/
  domain/             Domain layer (configctl, observation, evidence, signal, decision, strategy, risk, execution)
  application/        Use cases and ports
  actors/             Actor-based service orchestration
  adapters/clickhouse/ Analytical read adapters
  adapters/nats/      NATS messaging adapters
  adapters/exchanges/ Exchange protocol adapters (Binance Futures, etc.)
  application/configctl/memoryrepo/ In-memory configctl repository
  interfaces/http/    HTTP handlers and routing
  shared/             Cross-cutting: settings, bootstrap, problem, envelope, events

tools/raccoon-cli/    Rust CLI for quality enforcement

deploy/
  compose/            Docker Compose (nats + clickhouse + configctl + gateway + ingest + derive + store + execute + writer)
  configs/            Service configuration (JSONC)
  docker/             Dockerfile
  migrations/         ClickHouse migration catalog
  nats/               NATS server config
```

## Quick Start

```bash
# Discover available workflows first
make help

# Validate local prerequisites and canonical entrypoints
make bootstrap

# Fastest official bring-up path
make live

# Choose the right proof and see common overrides
make smoke-help

# Canonical baseline proof after bring-up
make smoke

# Controlled manual path when you need finer runtime control
make up
make seed
make smoke

# Daily validation loop
make check
make tdd
make verify

# Stage support for governed waves
make stage-help
make stage-status STAGE_ID=C20 STAGE_SLUG=automation-support-for-waves-execution-continuity-and-repo-sustainability
make stage-check STAGE_ID=C15 STAGE_SLUG=stage-tooling-and-execution-governance-support

# Troubleshooting entrypoint
make diag

# Show the primary workflow/tooling docs
make docs
```

## Development Workflow

```bash
make bootstrap   # validate local prerequisites once per machine / environment change
make live        # fastest official bring-up path
make smoke-help  # choose the right smoke/proof and see common overrides
make smoke       # canonical baseline operational proof
make diag        # first troubleshooting stop for a running stack
make help        # Discover the supported target surface
make check       # Pre-code guard rail (repo consistency + quality gate)
make repo-consistency-check  # Lightweight naming/docs/support-surface checks
make stage-status STAGE_ID=C20 STAGE_SLUG=automation-support-for-waves-execution-continuity-and-repo-sustainability
make tdd         # Impact-driven validation guide
make verify      # Tests + repo consistency + quality gate
make arch-guard  # Architecture boundary check
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full workflow reference.
The documentation entrypoint is [`docs/README.md`](docs/README.md).
The product surface lives in [`docs/product/README.md`](docs/product/README.md).
The development surface lives in [`docs/development/README.md`](docs/development/README.md).
Direct tooling references live in [`docs/tooling/README.md`](docs/tooling/README.md).

## Repository Navigation

Use these entrypoints when you need to navigate the physical repository shape
rather than the documentation taxonomy:

- [`cmd/README.md`](cmd/README.md) - service and binary entrypoints
- [`internal/README.md`](internal/README.md) - architecture layers and implementation map
- [`deploy/README.md`](deploy/README.md) - runtime assets, configs, compose, and migrations
- [`scripts/README.md`](scripts/README.md) - script catalog and wrapper rules
- [`tests/README.md`](tests/README.md) - test surfaces and when to use them
- [`tools/raccoon-cli/README.md`](tools/raccoon-cli/README.md) - tooling workspace entrypoint
- [`docs/development/repository-map.md`](docs/development/repository-map.md) - cross-repository map that connects these areas

## Documentation Map

- [`docs/README.md`](docs/README.md) - human documentation entrypoint
- [`docs/product/README.md`](docs/product/README.md) - product and runtime context
- [`docs/product/owners.md`](docs/product/owners.md) - product-facing owner docs
- [`docs/development/README.md`](docs/development/README.md) - contributor workflow and navigation
- [`docs/development/owners.md`](docs/development/owners.md) - development-facing owner docs
- [`docs/tooling/README.md`](docs/tooling/README.md) - `raccoon-cli` guardrails, drift rules, and topology references
- [`docs/architecture/README.md`](docs/architecture/README.md) - deep canonical architecture reference
- [`docs/stages/INDEX.md`](docs/stages/INDEX.md) - historical stage reports
- [`docs/archive/README.md`](docs/archive/README.md) - archived and superseded docs

## Support Surface Hierarchy

- `make` is the canonical public entrypoint for repository workflows: validation, stack lifecycle, smoke flows, codegen checks, and migrations.
- `make bootstrap` is the canonical setup check for a new machine or changed local environment.
- `make live` is the fastest official bring-up path; `make up` + `make seed*` remains the controlled manual path when you need finer runtime control.
- `make smoke-help` is the fastest way to choose the right proof and recall the common setup/diagnosis commands.
- The `make smoke*` family is the canonical operational-proof surface; choose the narrowest smoke that proves the runtime behavior you changed.
- `make diag`, `make ps`, and `make logs SERVICE=...` are the first-line troubleshooting entrypoints for a running stack.
- `make live*` and `stack-*` targets are ergonomic wrappers around canonical runtime workflows, not competing proof-of-record surfaces.
- `scripts/*.sh` are auxiliary harnesses behind `make`; call them directly only when you need debugging detail or flags that the public Make target intentionally hides.
- Direct `raccoon-cli` usage is the expert tooling surface for inspection and governance; it complements `make`, owns strategic intelligence tasks, and should not replace runtime/operator flows that already have Makefile entrypoints.
- Raw `docker compose`, `go`, and `cargo` commands are substrate-level interfaces for working on those layers directly, not the primary repository workflow surface.

## What Was Removed

This repository was originally cloned from a quality-service. The following components were removed during sanitization:

- Validator service and all validation pipeline logic
- Consumer service and Kafka bridge
- Emulator service and synthetic data generation
- Kafka adapter and all Kafka infrastructure
- Quality-specific HTTP endpoints and contracts
- All `.context/` documentation from the quality-service era

See [`docs/product/README.md`](docs/product/README.md) and
[`docs/architecture/README.md`](docs/architecture/README.md) for current
system context and deep technical records.

## Next Phase

The repository is prepared for the next governed slice only after active docs,
tooling, and closure evidence stay aligned. See
[`docs/architecture/market-foundry-evolution-playbook.md`](docs/architecture/market-foundry-evolution-playbook.md).
