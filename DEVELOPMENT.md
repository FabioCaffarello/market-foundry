# Development Workflow

## Quick Reference

| Target | Description |
|--------|-------------|
| `make tidy` | Run `go mod tidy` across all workspace modules |
| `make test` | Run `go test ./...` across all workspace modules |
| `make build` | Build service binaries into `bin/` |
| `make docker-build` | Build Docker images for services |
| `make up` | Start the stack (nats + configctl + gateway + ingest + derive + store) |
| `make down` | Stop the stack |
| `make logs` | Stream logs (optionally `SERVICE=gateway`) |
| `make ps` | Show service status |
| `make seed` | Seed configctl with single symbol (btcusdt) |
| `make seed-multi` | Seed configctl with multi-symbol (btcusdt + ethusdt) |
| `make smoke` | First-slice E2E smoke test (requires `make up` + `make seed`) |
| `make smoke-multi` | Multi-symbol E2E smoke test (requires `make up` + `make seed-multi`) |
| `make check` | Pre-code guard rail (quality-gate fast) |
| `make verify` | Post-change validation (tests + quality-gate) |
| `make check-deep` | Full quality-gate validation |

## Development Flow

### 1. Pre-Change Guard

```bash
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
| configctl | — | Config lifecycle management (NATS only) |
| gateway | 8080 | HTTP API gateway |
| ingest | — | Market data capture: Binance WS → observation events (NATS only) |
| derive | — | Observation → evidence processing: candle sampling (NATS only) |
| store | — | Read model materialization: NATS KV projections (NATS only) |
| execute | — | Execution control service (NATS only) |

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
cmd/store/           Read model materialization service entrypoint
internal/shared/     Cross-cutting concerns (settings, bootstrap, problem, envelope, events)
internal/domain/     Domain layer (pure business logic: configctl, observation, evidence, signal, decision, strategy, risk, execution)
internal/application/ Application layer (use cases, ports, contracts, client use cases)
internal/actors/     Actor-based service orchestration (supervisors, scopes)
internal/adapters/   Infrastructure adapters (NATS, exchanges, repositories)
internal/interfaces/ Interface layer (HTTP handlers, routes, webserver)
tools/raccoon-cli/   Rust architecture guardian CLI
deploy/              Docker Compose, configs, Dockerfile
scripts/             Utility and smoke-test scripts
tests/http/          HTTP test files
docs/architecture/   Architecture decisions and canonical patterns
docs/stages/         Stage completion reports
docs/tooling/        CLI and tooling documentation
```
