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
- **Rust CLI** (raccoon-cli) for static analysis, architecture enforcement, and quality gates
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
make tdd         # Impact-driven validation guide
make verify      # Tests + repo consistency + quality gate
make arch-guard  # Architecture boundary check
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full workflow reference.
The documentation entrypoint is [`docs/README.md`](docs/README.md).
The canonical documentation-system hardening map lives in
[`docs/operations/documentation-system-hardening.md`](docs/operations/documentation-system-hardening.md).
The canonical documentation governance, entrypoint, and taxonomy rules live in
[`docs/operations/documentation-governance-entrypoints-and-taxonomy.md`](docs/operations/documentation-governance-entrypoints-and-taxonomy.md).
The unified operational journey is documented in
[`docs/operations/developer-workflow-unification.md`](docs/operations/developer-workflow-unification.md).
The canonical developer-environment architecture and lifecycle now live in
[`docs/operations/development-environment-architecture-and-lifecycle.md`](docs/operations/development-environment-architecture-and-lifecycle.md)
and
[`docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`](docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md).
The onboarding and troubleshooting runbook lives in
[`docs/operations/developer-onboarding-and-troubleshooting-guide.md`](docs/operations/developer-onboarding-and-troubleshooting-guide.md).
The smoke/proof UX guidance lives in
[`docs/operations/smoke-ux-and-proof-execution-ergonomics.md`](docs/operations/smoke-ux-and-proof-execution-ergonomics.md),
and the proof failure-diagnosis flow lives in
[`docs/operations/proof-execution-user-flows-and-failure-diagnosis.md`](docs/operations/proof-execution-user-flows-and-failure-diagnosis.md).
Operational conventions for the command surface live in
[`docs/operations/README.md`](docs/operations/README.md), and direct tooling
references live in [`docs/tooling/README.md`](docs/tooling/README.md).
The canonical support-surface model is documented in
[`docs/operations/repository-support-surface-canonical-model.md`](docs/operations/repository-support-surface-canonical-model.md).
Operational proof governance and ownership now live in
[`docs/operations/smoke-and-operational-harness-governance.md`](docs/operations/smoke-and-operational-harness-governance.md)
and
[`docs/operations/operational-proof-entrypoints-and-ownership.md`](docs/operations/operational-proof-entrypoints-and-ownership.md).
Stage-support workflow guidance now lives in
[`docs/operations/stage-tooling-and-execution-governance-support.md`](docs/operations/stage-tooling-and-execution-governance-support.md)
and
[`docs/operations/stage-artifacts-conventions-and-support-model.md`](docs/operations/stage-artifacts-conventions-and-support-model.md).

## Documentation Map

- [`docs/README.md`](docs/README.md) - top-level documentation navigation
- [`docs/operations/README.md`](docs/operations/README.md) - daily workflow,
  command surface, scripts, and documentation conventions
- [`docs/operations/development-environment-architecture-and-lifecycle.md`](docs/operations/development-environment-architecture-and-lifecycle.md) - canonical developer environment architecture and lifecycle model
- [`docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`](docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md) - canonical entrypoints and flow-by-flow operating paths
- [`docs/operations/smoke-ux-and-proof-execution-ergonomics.md`](docs/operations/smoke-ux-and-proof-execution-ergonomics.md) - smoke/proof UX model, selection guidance, and operator ergonomics
- [`docs/operations/proof-execution-user-flows-and-failure-diagnosis.md`](docs/operations/proof-execution-user-flows-and-failure-diagnosis.md) - operational flows, failure interpretation, and diagnosis paths
- [`docs/operations/documentation-system-hardening.md`](docs/operations/documentation-system-hardening.md) - canonical documentation-system map and cross-surface links
- [`docs/operations/documentation-governance-entrypoints-and-taxonomy.md`](docs/operations/documentation-governance-entrypoints-and-taxonomy.md) - canonical taxonomy, naming, and maintenance rules
- [`docs/operations/stage-tooling-and-execution-governance-support.md`](docs/operations/stage-tooling-and-execution-governance-support.md) - active stage-support workflow and lightweight checks
- [`docs/operations/stage-artifacts-conventions-and-support-model.md`](docs/operations/stage-artifacts-conventions-and-support-model.md) - naming, placement, and minimum completeness for stage artifacts
- [`docs/tooling/README.md`](docs/tooling/README.md) - `raccoon-cli` guardrails,
  drift rules, and topology references
- [`docs/architecture/README.md`](docs/architecture/README.md) - canonical
  architecture and governance entrypoint
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
- Direct `raccoon-cli` usage is the expert tooling surface for inspection and governance; it complements `make` and should not replace runtime/operator flows that already have Makefile entrypoints.
- Raw `docker compose`, `go`, and `cargo` commands are substrate-level interfaces for working on those layers directly, not the primary repository workflow surface.

## What Was Removed

This repository was originally cloned from a quality-service. The following components were removed during sanitization:

- Validator service and all validation pipeline logic
- Consumer service and Kafka bridge
- Emulator service and synthetic data generation
- Kafka adapter and all Kafka infrastructure
- Quality-specific HTTP endpoints and contracts
- All `.context/` documentation from the quality-service era

See [`docs/architecture/README.md`](docs/architecture/README.md) for detailed
audit and decision records.

## Next Phase

The repository is prepared for the next governed slice only after active docs,
tooling, and closure evidence stay aligned. See
[`docs/architecture/market-foundry-evolution-playbook.md`](docs/architecture/market-foundry-evolution-playbook.md).
