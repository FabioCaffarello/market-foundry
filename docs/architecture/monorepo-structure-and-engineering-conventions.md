# Monorepo Structure and Engineering Conventions

## Purpose

This document formalizes the structural conventions of the market-foundry monorepo. It serves as the canonical reference for how code, modules, services, and infrastructure are organized and how they must grow.

## Monorepo Layout

```
market-foundry/
├── cmd/{service}/          # Service entry points (composition roots)
├── internal/
│   ├── domain/             # Pure business logic (no external deps)
│   ├── application/        # Use cases, ports, contracts, client use cases
│   ├── adapters/           # Infrastructure implementations
│   │   ├── nats/           # NATS messaging adapter
│   │   ├── exchanges/      # Exchange WebSocket adapters
│   │   └── repositories/   # Data access implementations
│   ├── actors/             # Hollywood actor orchestration
│   │   ├── common/         # Shared engine, lifecycle
│   │   └── scopes/         # Service-specific actor hierarchies
│   ├── interfaces/         # External protocol handlers (HTTP)
│   │   └── http/           # Handlers, routes, webserver
│   └── shared/             # Cross-cutting concerns
├── tools/raccoon-cli/      # Rust architecture guardian
├── deploy/                 # Docker, compose, service configs
│   ├── compose/            # docker-compose.yaml
│   ├── configs/            # Service JSONC configs
│   ├── docker/             # Dockerfile(s)
│   ├── nats/               # NATS server config
│   └── clickhouse/         # ClickHouse config
├── scripts/                # Automation and smoke tests
├── tests/                  # Test fixtures (HTTP files)
├── docs/
│   ├── architecture/       # Canonical architecture decisions
│   ├── stages/             # Stage completion reports
│   └── tooling/            # CLI and tooling documentation
├── go.work                 # Go workspace root
├── Makefile                # Build and workflow automation
├── AGENTS.md               # AI agent operating contract
├── DEVELOPMENT.md          # Developer workflow reference
└── README.md               # Project overview
```

## Go Workspace Conventions

### Module Boundaries

The workspace (`go.work`) declares 14 modules organized by architectural layer:

| Layer | Module Path | Purpose |
|-------|-------------|---------|
| Domain | `internal/domain` | Pure business types, events, aggregates |
| Application | `internal/application` | Use cases, port interfaces, contracts |
| Adapters | `internal/adapters/nats` | NATS messaging implementation |
| Adapters | `internal/adapters/exchanges` | Exchange WebSocket adapters |
| Adapters | `internal/adapters/repositories` | Data persistence implementations |
| Actors | `internal/actors` | Actor supervision, lifecycle, scopes |
| Interfaces | `internal/interfaces/http` | HTTP handlers, routes, webserver |
| Shared | `internal/shared` | Cross-cutting: settings, problem, events, bootstrap |
| Services | `cmd/{service}` | One module per deployable binary (6 total) |

### Module Rules

1. **One go.mod per architectural boundary** — modules align with layers, not features.
2. **Dependencies flow inward only** — domain has zero internal dependencies; cmd depends on everything.
3. **No circular dependencies** — enforced by `make arch-guard`.
4. **Shared is a utility layer** — it may be imported by any internal module but must not import any.

### Dependency Direction

```
domain ← application ← adapters ← actors ← interfaces ← cmd
                                                    ↑
                                              shared (utility)
```

## Service Conventions

### Binary Layout

Each service lives in `cmd/{service}/` and follows a consistent structure:

| File | Purpose |
|------|---------|
| `main.go` | Bootstrap entry via `bootstrap.Main()` |
| `run.go` | Runtime orchestration: infrastructure → composition → actor spawn → health → shutdown |
| `compose.go` | Optional: extracted composition root when `run.go` exceeds ~80 lines |
| `{service}.go` | Optional: service-specific actor spawning logic |

### Service Naming

- Service names are **lowercase, no separators**: `gateway`, `store`, `derive`, `ingest`, `configctl`, `execute`.
- Binary output goes to `bin/{service}`.
- Docker image names match service names.
- Config files live at `deploy/configs/{service}.jsonc`.

### Current Services

| Service | Type | Purpose |
|---------|------|---------|
| configctl | NATS-only | Config lifecycle management |
| gateway | HTTP↔NATS | API gateway (stateless translator) |
| ingest | NATS-only | Market data capture (exchange WebSocket → observations) |
| derive | NATS-only | Evidence derivation (observations → candles, volumes) |
| store | NATS-only | Read model materialization (NATS KV projections) |
| execute | NATS-only | Execution control |

## Package Conventions

### Domain Layer (`internal/domain/`)

- One subdirectory per bounded context: `configctl`, `observation`, `evidence`, `signal`, `decision`, `strategy`, `risk`, `execution`.
- Contains only pure business logic — no infrastructure imports.
- Value objects use validation methods returning `*problem.Problem`.
- Aggregates collect pending events via `recordEvent()` / `PullEvents()`.
- Financial/measurement values stored as `string` to avoid IEEE 754 precision loss.
- All timestamps normalized to UTC.

### Application Layer (`internal/application/`)

- `ports/` — one interface file per domain gateway (e.g., `configctl.go`, `evidence.go`).
- `{domain}client/` — gateway client use cases for cross-service calls (used by gateway binary).
- `{domain}/` — domain-specific business logic (used by the domain's own runtime).
- `contracts/` — command, query, and reply types.
- `runtimecontracts/` — type contracts for runtime communication (projection families, pipeline specs).
- Use case structs follow: constructor injection, `Execute()` method, `*problem.Problem` error return.

### Adapter Layer (`internal/adapters/`)

- Organized by technology, not by domain: `nats/`, `exchanges/`, `repositories/`.
- Each adapter subdirectory has its own `go.mod`.
- Registry objects are value objects (not DI containers) — they define subject patterns, consumer specs, stream config.
- Gateway implementations satisfy port interfaces from the application layer.

### Actor Layer (`internal/actors/`)

- `common/` — shared engine creation, lifecycle helpers, entrypoint.
- `scopes/{service}/` — one scope directory per runtime that uses actors.
- Supervisor actors manage child actor lifecycles.
- Scope refers to **actor supervision boundaries** (not domain boundaries).

### Interface Layer (`internal/interfaces/http/`)

- `handlers/` — HTTP handler implementations.
- `routes/` — route groupings with family-grouped dependency structs.
- `webserver/` — HTTP server lifecycle management.

### Shared Layer (`internal/shared/`)

- `bootstrap/` — service startup (Main function, logger config, NATS readiness).
- `problem/` — canonical error type with stable codes.
- `events/` — event interface, metadata, ID generation.
- `settings/` — configuration schema and parsing.
- `envelope/` — NATS message wrapper.
- `healthz/` — health check and tracker infrastructure.
- `memdb/` — in-memory key-value utilities.
- `requestctx/` — HTTP request context.

## Configuration Conventions

- Service configs use **JSONC** format at `deploy/configs/{service}.jsonc`.
- Features are enabled/disabled via config — never via code flags or build tags.
- Environment-specific overrides use `.env` files (gitignored).
- Config schema is defined in `internal/shared/settings/`.

## Build and Workflow Conventions

### Makefile Targets

The Makefile is the single entry point for all development workflows:

| Category | Key Targets |
|----------|-------------|
| Build | `make build`, `make docker-build` |
| Test | `make test`, `make test-integration` |
| Quality | `make check`, `make verify`, `make check-deep` |
| Stack | `make up`, `make down`, `make logs`, `make ps` |
| Seed/Smoke | `make seed`, `make smoke`, `make smoke-multi` |
| Analysis | `make arch-guard`, `make drift-detect`, `make tdd`, `make coverage-map` |
| Snapshots | `make snapshot`, `make snapshot-diff`, `make baseline-drift` |

### Developer Workflow

```
make check → make tdd → implement → make verify → (make check-deep for significant changes)
```

### Scoping

- `MODULE=./internal/shared make test` — run tests for a single module.
- `SERVICE=gateway make build` — build a single service.

## Tooling Conventions

### raccoon-cli

- Lives at `tools/raccoon-cli/` (Rust, built via Cargo).
- Serves as the **architecture guardian** — enforces layer boundaries, naming, contracts, and drift.
- Quality-gate profiles: `fast` (local dev), `ci` (pipeline), `deep` (pre-merge).
- All analysis is driven by `make` targets — developers never invoke `raccoon-cli` directly.

### Scripts

- `scripts/` contains utility and automation scripts.
- `scripts/utils/` contains helper scripts used by the Makefile (e.g., `for-each-module.sh`, `list-modules.sh`).
- Smoke tests and seed scripts live at `scripts/` root level.

## Naming Invariants

These are binding — see `naming-conventions-for-domains-families-and-runtimes.md` for full detail:

| Entity | Convention | Examples |
|--------|-----------|----------|
| Domain | `snake_case`, singular | `evidence`, `signal`, `decision` |
| Family | `snake_case` | `candle`, `volume`, `rsi` |
| Runtime/Service | lowercase, no separators | `gateway`, `store`, `derive` |
| Scope | actor supervision boundary | `source`, `exchange` |
| Module path | matches layer name | `internal/domain`, `internal/application` |
| Package path | matches entity name | `internal/domain/evidence`, `internal/application/evidenceclient` |

## Anti-Patterns

1. **No framework-based DI** — all wiring is explicit constructor injection in composition roots.
2. **No feature flags in code** — use config-driven activation.
3. **No cross-domain imports in the domain layer** — domains are isolated.
4. **No infrastructure types in domain or application layers**.
5. **No direct raccoon-cli invocation** — always go through `make` targets.
6. **No new top-level directories** without documented rationale.

## Related Documents

- `system-vision.md` — foundational identity and purpose
- `system-principles.md` — inviolable architectural principles
- `naming-conventions-for-domains-families-and-runtimes.md` — full naming rules
- `dependency-injection-and-composition-roots.md` — DI and composition patterns
- `boundary-naming-and-interface-hygiene.md` — terminology disambiguation
