# Development Workflow

## Quick Reference

| Target | Description |
|--------|-------------|
| `make help` | Show grouped targets and common variables |
| `make docs` | Show the primary workflow and tooling docs |
| `make tidy` | Run `go mod tidy` across all workspace modules |
| `make test` | Run `go test ./...` across all workspace modules |
| `make lint` | Alias for `make check` |
| `make build` | Build service binaries into `bin/` |
| `make docker-build` | Build Docker images for services |
| `make up` | Start the stack (nats + clickhouse + configctl + gateway + ingest + derive + store + execute + writer) |
| `make down` | Stop the stack |
| `make logs` | Stream logs (optionally `SERVICE=gateway`) |
| `make ps` | Show service status |
| `make live` | Build, start, seed, and validate the single-symbol live stack |
| `make seed` | Seed configctl with single symbol (btcusdt) |
| `make seed-multi` | Seed configctl with multi-symbol (btcusdt + ethusdt) |
| `make smoke` | First-slice E2E smoke test (requires `make up` + `make seed`) |
| `make smoke-multi` | Multi-symbol E2E smoke test (requires `make up` + `make seed-multi`) |
| `make smoke-analytical` | Analytical path proof (NATS → writer → ClickHouse → reader → gateway) |
| `make smoke-restart-recovery` | Restart/recovery smoke for durable consumers and projections |
| `make check` | Pre-code guard rail (quality-gate fast) |
| `make tdd` | Impact-driven testing guide for the current change set |
| `make verify` | Post-change validation (tests + quality-gate) |
| `make check-deep` | Full quality-gate validation |
| `make codegen-equivalence` | Cross-artifact codegen equivalence wrapper |
| `make migrate-up` | Apply pending ClickHouse migrations |
| `make migrate-status` | Show migration status |
| `make migrate-validate` | Verify migration checksums |

## Development Flow

### 1. Pre-Change Guard

```bash
make help
make check
```

Runs the raccoon-cli fast quality-gate to verify repository structure, topology, contracts, and architecture boundaries before you start coding.

### 2. TDD

```bash
make tdd
```

Get a TDD guide showing what to validate for your current changes.

### 3. Implement

Make the smallest correct change.

### 4. Post-Change Validation

```bash
make verify
```

Runs all Go tests across workspace modules, then runs the quality-gate.

### 5. Deep Check (when needed)

```bash
make check-deep
```

Full validation including all quality-gate checks.

## Quality Gate Profiles

| Profile | Scope | When to Use |
|---------|-------|-------------|
| fast | Structure, topology, contracts | Default for `make check` |
| ci | Strict static (JSON output) | CI pipelines |
| deep | Full validation | Before merging significant changes |

## Architecture Enforcement

```bash
make arch-guard      # Check layer boundary violations
make drift-detect    # Detect cross-layer semantic drift
```

## Discoverability And Conventions

The root `Makefile` now follows these conventions:

- `make check` remains the canonical fast guard rail; `make lint` is a discoverability alias for teams expecting lint-style naming.
- Existing operational targets such as `make up`, `make smoke`, and `make live` remain canonical; `stack-*` aliases exist only to make the runtime surface more obvious.
- Hidden but real support flows are wrapped and promoted through `make`, notably `make smoke-restart-recovery` and `make codegen-equivalence`.
- `make docs` points to the current workflow and tooling documents instead of requiring contributors to search stage history.

See:

- [`docs/operations/makefile-command-ergonomics-and-hardening.md`](docs/operations/makefile-command-ergonomics-and-hardening.md)
- [`docs/operations/makefile-targets-reference-and-conventions.md`](docs/operations/makefile-targets-reference-and-conventions.md)
- [`docs/tooling/cli-overview.md`](docs/tooling/cli-overview.md)

## Change Analysis

```bash
make snapshot                          # Generate code intelligence snapshot
make snapshot-diff SNAP1=a.json SNAP2=b.json  # Compare snapshots
make baseline-drift BASELINE=b.json    # Detect drift from baseline
make recommend                         # Smart recommendations
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| nats | 4222 (client), 8222 (monitor) | Message bus |
| clickhouse | 8123 (HTTP), 9000 (native) | Analytical storage |
| configctl | — | Config lifecycle management (NATS only) |
| gateway | 8080 | HTTP API gateway |
| ingest | — | Market data capture: Binance WS → observation events (NATS only) |
| derive | — | Observation → downstream domain events (NATS only) |
| store | — | Read model materialization: NATS KV projections (NATS only) |
| execute | — | Execution control service (NATS only) |
| writer | — | Analytical writer: NATS events → ClickHouse rows (NATS + ClickHouse) |

## Module Scoping

```bash
MODULE=./internal/shared make test    # Test single module
SERVICE=gateway make build             # Build single service
SERVICE=gateway make logs              # Logs for single service
```

## Project Structure

```
cmd/configctl/       Config lifecycle service entrypoint
cmd/derive/          Evidence derivation service entrypoint
cmd/execute/         Execution control service entrypoint
cmd/gateway/         HTTP API gateway entrypoint
cmd/ingest/          Market data capture service entrypoint
cmd/migrate/         ClickHouse migration CLI entrypoint
cmd/store/           Read model materialization service entrypoint
cmd/writer/          Analytical writer service entrypoint
cmd/migrate/engine/  Binary-local migration library (renamed after Go 1.25 path collision)
internal/shared/     Cross-cutting concerns (settings, bootstrap, problem, envelope, events)
internal/domain/     Domain layer (pure business logic: configctl, observation, evidence, signal, decision, strategy, risk, execution)
internal/application/ Application layer (use cases, ports, contracts, client use cases, configctl memory repo)
internal/actors/     Actor-based service orchestration (supervisors, scopes)
internal/adapters/   Infrastructure adapters (NATS, exchanges, ClickHouse)
internal/interfaces/ Interface layer (HTTP handlers, routes)
internal/shared/webserver/ Shared HTTP server bootstrap and lifecycle wiring
tools/raccoon-cli/   Rust architecture guardian CLI
deploy/              Docker Compose, configs, Dockerfile
scripts/             Utility and smoke-test scripts
tests/http/          HTTP test files
docs/architecture/   Architecture decisions and canonical patterns
docs/stages/         Stage completion reports
docs/tooling/        CLI and tooling documentation
```

Current workspace baseline: `go.work` contains 17 modules after the S220 simplification.
