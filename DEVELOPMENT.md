# Development Workflow

## Quick Reference

| Target | Description |
|--------|-------------|
| `make help` | Show grouped targets and common variables |
| `make docs` | Show the primary workflow and tooling docs |
| `make bootstrap` | Validate local prerequisites and repository entrypoints for the official workflow |
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
| `make smoke-help` | Show smoke/proof selection, prerequisites, and common diagnosis commands |
| `make seed` | Seed configctl with single symbol (btcusdt) |
| `make seed-multi` | Seed configctl with multi-symbol (btcusdt + ethusdt) |
| `make smoke` | First-slice E2E smoke test (requires `make up` + `make seed`) |
| `make smoke-multi` | Multi-symbol E2E smoke test (requires `make up` + `make seed-multi`) |
| `make smoke-analytical` | Analytical path proof (NATS → writer → ClickHouse → reader → gateway) |
| `make smoke-round-trip` | Full persistence round-trip proof (adapter → NATS → ClickHouse → HTTP) |
| `make smoke-live-stack` | Live stack smoke and gateway verification |
| `make smoke-composed` | Composed pipeline smoke without the full stack |
| `make smoke-operational` | OS-process/container operational proof (halt/resume plus read-path checks) |
| `make smoke-restart-recovery` | Restart/recovery smoke for durable consumers and projections |
| `make check` | Pre-code guard rail (repo consistency + quality-gate fast) |
| `make repo-consistency-check` | Lightweight repository consistency checks for docs, naming, links, and script wrappers |
| `make stage-help` | Show the lightweight stage tooling surface |
| `make stage-scaffold` | Scaffold a stage report (`STAGE_ID`, `STAGE_SLUG`, `STAGE_TITLE`) |
| `make stage-status` | Show continuity status and next actions for one active stage |
| `make stage-check` | Validate one active stage report and its required artifacts |
| `make tdd` | Impact-driven testing guide for the current change set |
| `make verify` | Post-change validation (tests + repo consistency + quality-gate) |
| `make check-deep` | Full quality-gate validation |
| `make codegen-equivalence` | Cross-artifact codegen equivalence wrapper |
| `make migrate-up` | Apply pending ClickHouse migrations |
| `make migrate-status` | Show migration status |
| `make migrate-validate` | Verify migration checksums |

## Official Workflow

### 1. Bootstrap Or Revalidate The Machine

```bash
make bootstrap
```

Use this when onboarding a new machine, after toolchain changes, or when the
repository starts failing in ways that look environmental rather than code-related.

### 2. Choose The Runtime Bring-Up Path

Fastest path:

```bash
make live
```

Selection refresher:

```bash
make smoke-help
```

Controlled manual path:

```bash
make up
make seed       # or make seed-multi
make smoke      # or the narrowest relevant make smoke*
```

Rule:

- prefer `make live` when you want the repository to orchestrate bring-up for you;
- prefer `make up` + `make seed*` when you need to inspect or control each step;
- use `make smoke*` as the proof-of-record surface in both cases.

## Daily Change Loop

### 1. Pre-Change Guard

```bash
make help
make check
```

Runs the raccoon-cli fast quality-gate to verify repository structure, topology, contracts, and architecture boundaries before you start coding.
The target now starts with a lightweight repository consistency pass so broken
support-doc links, stage index drift, naming drift, and missing script wrappers
fail before deeper analysis runs.

When you are resuming or closing a governed stage, inspect continuity first:

```bash
make stage-status STAGE_ID=C20 STAGE_SLUG=automation-support-for-waves-execution-continuity-and-repo-sustainability
```

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
The same lightweight repository consistency pass from `make check` also runs here.

### 5. Deep Check (when needed)

```bash
make check-deep
```

Full validation including all quality-gate checks.

## Smoke Selection

| Need To Prove | Command |
|---------|---------|
| Baseline single-symbol runtime flow | `make smoke` |
| Broader multi-symbol runtime flow | `make smoke-multi` |
| Analytical writer/reader path | `make smoke-analytical` |
| Full persistence round-trip path | `make smoke-round-trip` |
| Live stack plus gateway verification | `make smoke-live-stack` |
| Composed pipeline proof without the full stack | `make smoke-composed` |
| Process/container operational behavior | `make smoke-operational` |
| Restart and recovery resilience | `make smoke-restart-recovery` |

Choose the narrowest smoke that proves the behavior you changed. `make live*`
does not replace `make smoke*` as proof-of-record.

## First-Line Troubleshooting

Use this order before dropping into direct scripts or raw substrate commands:

```bash
make diag
make ps
make logs SERVICE=gateway
SERVICE=gateway make restart
make down
```

Escalate to direct `scripts/*.sh`, `docker compose`, `go`, or `cargo` only when
you are debugging below the repository workflow contract.

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

- `make bootstrap` is the canonical setup check for a machine or changed environment.
- `make check` remains the canonical fast guard rail; `make lint` is a discoverability alias for teams expecting lint-style naming.
- `make live` is the fastest official bring-up path; `make up` + `make seed*` is the controlled manual path.
- `make smoke-help` is the fastest way to choose the right proof target and recall supported wait and URL overrides.
- `make smoke*` is the canonical operational-proof surface; choose the narrowest smoke target that proves the runtime behavior you touched.
- `make live*` and `stack-*` remain ergonomic wrappers and aliases; they do not replace the canonical proof-of-record `make smoke*` surface.
- Hidden but real support flows are wrapped and promoted through `make`, notably `make smoke-restart-recovery` and `make codegen-equivalence`.
- `make stage-status` is the continuity helper for governed stages; it shows missing report/index/artifact signals before `make stage-check`.
- `make docs` points to the current workflow and tooling documents instead of requiring contributors to search stage history.

Repository navigation now also has physical-area entrypoints:

- `cmd/README.md` for runtime and binary ownership
- `internal/README.md` for implementation layers and placement
- `deploy/README.md` for compose/config/env/migration assets
- `scripts/README.md` for harness ownership behind `make`
- `tests/README.md` for shared repository-level test assets
- `docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md` for task-to-directory navigation

See:

- [`docs/README.md`](docs/README.md)
- [`docs/operations/README.md`](docs/operations/README.md)
- [`docs/operations/repository-metadata-indexes-and-developer-navigation-system.md`](docs/operations/repository-metadata-indexes-and-developer-navigation-system.md)
- [`docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`](docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md)
- [`docs/operations/repository-maintainability-economics-and-structural-cost-control.md`](docs/operations/repository-maintainability-economics-and-structural-cost-control.md)
- [`docs/operations/repository-maintenance-hotspots-and-cost-reduction-principles.md`](docs/operations/repository-maintenance-hotspots-and-cost-reduction-principles.md)
- [`docs/tooling/README.md`](docs/tooling/README.md)

Keep this file intentionally shallow. The detailed support-document catalog lives
in [`docs/operations/README.md`](docs/operations/README.md) so root workflow docs
do not need to be edited every time the repository adds another support guide.

## Support Surface Hierarchy

- `make` is the canonical public entrypoint for day-to-day repository workflows.
- `make bootstrap` owns setup validation for the official developer workflow.
- `make live` is the fastest official bring-up path; `make up` + `make seed*` is the controlled manual path.
- `make smoke-help` improves proof discoverability but does not replace the proof-of-record targets.
- `make smoke*` owns operational proof; `make live*` and `stack-*` only compose or alias that surface.
- `make diag`, `make ps`, and `make logs SERVICE=...` are the first-line troubleshooting surface.
- `stage-help`, `stage-scaffold`, `stage-status`, and `stage-check` form the lightweight support surface for governed stage execution and report hygiene.
- `scripts/*.sh` are harness implementations behind `make`, not competing public APIs.
- Direct `raccoon-cli` usage is the expert support surface for structural inspection and governance work.
- `make check-deep` remains a deep tooling gate, not the operational proof-of-record surface.
- Raw `docker compose`, `go`, and `cargo` commands are substrate interfaces to use only when you are working on those layers directly or debugging below the canonical workflow surface.

## Documentation Surfaces

| Surface | Use For |
|---------|---------|
| `README.md` | Project overview and quick orientation |
| `DEVELOPMENT.md` | Daily engineering workflow |
| `docs/operations/` | Operational support docs, unified workflow, onboarding, troubleshooting, command surfaces |
| `docs/operations/documentation-*.md` | Documentation-system hardening, governance, entrypoints, and taxonomy |
| `docs/tooling/` | `raccoon-cli` guardrails and drift-rule references |
| `docs/architecture/` | Canonical architecture and governance |
| `docs/stages/` | Historical stage evidence |
| `docs/archive/` | Archived or superseded material |

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
docs/archive/        Historical and superseded documentation
docs/operations/     Operational support docs and documentation conventions
docs/stages/         Stage completion reports
docs/tooling/        CLI and tooling documentation
```

Current workspace baseline: `go.work` contains 17 modules after the S220 simplification.
