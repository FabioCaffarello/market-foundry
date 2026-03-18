# market-foundry

Foundation repository for the market-foundry system.

## Status

**Post first-slice phase** — the repository has completed sanitization, first vertical slice (observation → evidence → store → query), and architectural recentralization. Multi-symbol scalability proven via config-only changes.

## What This Repository Contains

- **Go workspace** with isolated internal modules following clean architecture layers
- **Config lifecycle management** (configctl) — create, validate, compile, activate configuration versions
- **HTTP API gateway** (gateway) — stateless HTTP→NATS translation layer
- **Market data capture** (ingest) — WebSocket → observation events
- **Evidence derivation** (derive) — observation → candle sampling → evidence events
- **Read model materialization** (store) — evidence events → NATS KV projections + query serving
- **Actor-based orchestration** using the Hollywood framework
- **NATS messaging** for inter-service communication
- **Rust CLI** (raccoon-cli) for static analysis, architecture enforcement, and quality gates
- **Docker Compose** setup for local development

## Architecture

```
cmd/
  configctl/          Config lifecycle service
  gateway/            HTTP API gateway (stateless HTTP→NATS translator)
  ingest/             Market data capture (WebSocket → observation events)
  derive/             Evidence derivation (observation → candle sampling)
  store/              Read model materialization (evidence → KV projections)

internal/
  domain/             Domain layer (pure business logic: configctl, observation, evidence)
  application/        Use cases and ports
  actors/             Actor-based service orchestration
  adapters/nats/      NATS messaging adapters
  adapters/exchanges/ Exchange protocol adapters (Binance Futures, etc.)
  adapters/repos/     In-memory repositories
  interfaces/http/    HTTP handlers and routing
  shared/             Cross-cutting: settings, bootstrap, problem, envelope, events

tools/raccoon-cli/    Rust CLI for quality enforcement

deploy/
  compose/            Docker Compose (nats + configctl + gateway + ingest + derive + store)
  configs/            Service configuration (JSONC)
  docker/             Dockerfile
  nats/               NATS server config
```

## Quick Start

```bash
# Build and start the stack
make up

# Seed configctl with single symbol
make seed

# Verify health
curl http://127.0.0.1:8080/healthz

# Run E2E smoke test
make smoke

# Run quality checks
make check

# Run tests
make test
```

## Development Workflow

```bash
make check       # Pre-code guard rail
make test        # Run Go tests
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

The repository is prepared for further evolution. See `docs/architecture/next-phase-readiness.md`.
