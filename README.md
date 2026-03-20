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
# Build, start, seed, and validate the current live stack
make live

# Or start the stack manually
make up
make seed

# Verify health
curl http://127.0.0.1:8080/healthz

# Run E2E smoke test
make smoke

# Prove the analytical path
make smoke-analytical

# Run quality checks
make check

# Run post-change validation
make verify
```

## Development Workflow

```bash
make check       # Pre-code guard rail
make tdd         # Impact-driven validation guide
make verify      # Tests + quality gate
make arch-guard  # Architecture boundary check
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full workflow reference.

## What Was Removed

This repository was originally cloned from a quality-service. The following components were removed during sanitization:

- Validator service and all validation pipeline logic
- Consumer service and Kafka bridge
- Emulator service and synthetic data generation
- Kafka adapter and all Kafka infrastructure
- Quality-specific HTTP endpoints and contracts
- All `.context/` documentation from the quality-service era

See `docs/architecture/` for detailed audit and decision records.

## Next Phase

The repository is prepared for the next governed slice only after active docs, tooling, and closure evidence stay aligned. See `docs/architecture/market-foundry-evolution-playbook.md`.
