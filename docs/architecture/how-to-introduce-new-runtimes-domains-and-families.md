# How to Introduce New Runtimes, Domains, and Families

## Purpose

Step-by-step guide for expanding the market-foundry monorepo with new bounded contexts, processing families, or deployable services. This document prevents ad-hoc growth and ensures every addition follows established patterns.

## Prerequisites

Before introducing anything new, verify:

1. The addition is justified by a real requirement — not speculative.
2. You have read `system-principles.md` and `naming-conventions-for-domains-families-and-runtimes.md`.
3. You understand the current layer structure (see `monorepo-structure-and-engineering-conventions.md`).

---

## Adding a New Domain

A domain is a bounded context in the domain layer. Current domains: `configctl`, `observation`, `evidence`, `signal`, `decision`, `strategy`, `risk`, `execution`.

### Steps

1. **Create the domain package**:
   ```
   internal/domain/{domain_name}/
   ```
   - Name: `snake_case`, singular (e.g., `liquidity`, not `liquidities`).
   - No new `go.mod` — the domain layer shares a single module (`internal/domain/go.mod`).

2. **Define domain types**:
   - Value objects with `.Validate()` returning `*problem.Problem`.
   - Events implementing the `events.Event` interface with proper metadata.
   - Aggregates with `recordEvent()` / `PullEvents()` if event-sourced.
   - Use `string` for financial/measurement precision. Normalize timestamps to UTC.

3. **Create the port interface**:
   ```
   internal/application/ports/{domain_name}.go
   ```
   - Define the gateway interface (e.g., `LiquidityGateway`).
   - Use command/query/reply contracts as parameters.
   - Return `*problem.Problem`, not `error`.

4. **Create application-level contracts** (if cross-service access is needed):
   ```
   internal/application/{domain_name}client/
   ```
   - One file per use case (e.g., `get_latest_liquidity.go`).
   - Each use case struct takes the gateway via constructor injection.

5. **Create the adapter implementation**:
   - NATS registry: add to `internal/adapters/nats/` (e.g., `liquidity_registry.go`).
   - NATS gateway: implement the port interface via request/reply.
   - KV store: if the domain needs materialized views.

6. **Validate**:
   - `make arch-guard` — verify no layer violations.
   - `make test` — all tests pass.
   - `make check` — quality gate passes.

### What NOT to Do

- Do not create a separate `go.mod` for the new domain.
- Do not import other domains from within the domain layer.
- Do not add infrastructure types to the domain package.

---

## Adding a New Family

A family is a processing specialization within a domain (e.g., `candle` and `volume` are families within `evidence`). See `family-runtime-registration-rules.md` for detailed checklists per runtime.

### Steps

1. **Define the family's domain types** in the parent domain package:
   ```
   internal/domain/{domain_name}/{family_type}.go
   ```

2. **Register in the derive runtime** (if the family produces derived data):
   - Add a `FamilyProcessor` entry to the derive supervisor's processor list.
   - The processor defines: family name, input subject, processing function, output subject.

3. **Register in the store runtime** (if the family needs materialized views):
   - Add a `ProjectionPipeline` entry to the store supervisor's pipeline catalog.
   - The pipeline defines: domain, family, durable consumer name, subject pattern, stream, KV bucket.

4. **Register in the gateway runtime** (if the family needs HTTP exposure):
   - Add route dependencies to `internal/interfaces/http/routes/`.
   - Add handler(s) to `internal/interfaces/http/handlers/`.
   - Wire the gateway connection in `cmd/gateway/compose.go`.

5. **Add NATS infrastructure**:
   - Registry specs (event spec, control spec) in the domain's registry file.
   - Consumer specs with durable names following: `{domain}-{family}-{runtime}`.

6. **Update configuration**:
   - Add the family to relevant service JSONC configs in `deploy/configs/`.
   - Families are enabled/disabled via config — never hardcoded.

7. **Validate**:
   - `make arch-guard` — no layer violations.
   - `make test` — all tests pass.
   - `make verify` — full validation.

### Naming Rules

- Family names: `snake_case` (e.g., `candle`, `volume`, `trade_burst`, `rsi`).
- Durable consumer names: `{domain}-{family}-{runtime}` (e.g., `evidence-candle-store`).
- KV bucket names: `{domain}_{family}` (e.g., `evidence_candle`).

---

## Adding a New Runtime (Service)

A runtime is a deployable binary in `cmd/`. Current runtimes: `configctl`, `gateway`, `ingest`, `derive`, `store`, `execute`.

### Pre-Requisites

Adding a new runtime is a significant decision. Verify:

- The new service has a distinct operational concern that cannot be served by an existing runtime.
- The service boundary aligns with a clear domain or infrastructure responsibility.

### Steps

1. **Create the service entry point**:
   ```
   cmd/{runtime}/
   ├── go.mod        # New module with workspace-local dependencies
   ├── main.go       # bootstrap.Main("{runtime}", Run)
   └── run.go        # Runtime orchestration
   ```

2. **Follow the runtime lifecycle** (see `runtime-assembly-guidelines.md`):
   - Phase 1: Infrastructure setup (NATS connection, config load).
   - Phase 2: Composition (registries, gateways, use cases).
   - Phase 3: Use case wiring.
   - Phase 4: Actor spawn (supervisor creation).
   - Phase 5: Health server start.
   - Phase 6: Graceful shutdown.

3. **Create the actor scope** (if the service uses actors):
   ```
   internal/actors/scopes/{runtime}/
   ```
   - Supervisor actor with `Receive()` message routing.
   - Child actors as needed.

4. **Register in the workspace**:
   - Add `./cmd/{runtime}` to `go.work`.

5. **Add deployment infrastructure**:
   - Config: `deploy/configs/{runtime}.jsonc`.
   - Add service to `deploy/compose/docker-compose.yaml`.
   - The shared Dockerfile (`deploy/docker/go-service.Dockerfile`) should work for all Go services.

6. **Update the Makefile**:
   - Add the service name to `BUILDABLE_SERVICES`.

7. **Update documentation**:
   - Add to the services table in `AGENTS.md`.
   - Add to the services table in `DEVELOPMENT.md`.

8. **Validate**:
   - `make build SERVICE={runtime}` — binary builds.
   - `make test` — all tests pass.
   - `make verify` — full validation.

### What NOT to Do

- Do not create a runtime for a single use case — group related concerns.
- Do not duplicate infrastructure setup — use `internal/shared/bootstrap/`.
- Do not skip the health server — all runtimes must be health-checkable.

---

## Adding a New Adapter

Adapters implement port interfaces from the application layer. Current adapter groups: `nats`, `exchanges`, `repositories`.

### Steps

1. **Determine the adapter group**:
   - Messaging → `internal/adapters/nats/`
   - External data sources → `internal/adapters/exchanges/`
   - Persistence → `internal/adapters/repositories/`
   - New technology → create `internal/adapters/{technology}/` with its own `go.mod`.

2. **Implement the port interface** defined in `internal/application/ports/`.

3. **Add to `go.work`** if a new `go.mod` was created.

4. **Validate** with `make arch-guard` and `make test`.

---

## Checklist Summary

| Action | Key Files | Validation |
|--------|-----------|------------|
| New domain | `internal/domain/{name}/`, `internal/application/ports/{name}.go` | `make arch-guard`, `make test` |
| New family | Domain types, supervisor registration, config | `make verify` |
| New runtime | `cmd/{name}/`, `go.work`, Makefile, deploy configs | `make build`, `make verify` |
| New adapter | `internal/adapters/{tech}/`, port implementation | `make arch-guard`, `make test` |

## Related Documents

- `naming-conventions-for-domains-families-and-runtimes.md` — naming rules
- `family-runtime-registration-rules.md` — detailed family registration checklists
- `runtime-assembly-guidelines.md` — runtime lifecycle phases
- `dependency-injection-and-composition-roots.md` — composition root patterns
- `monorepo-structure-and-engineering-conventions.md` — structural conventions
