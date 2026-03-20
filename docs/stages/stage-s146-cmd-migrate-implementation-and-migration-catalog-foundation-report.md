# Stage S146 — cmd/migrate Implementation and Migration Catalog Foundation

**Status:** Complete
**Date:** 2026-03-19

## Objective

Implement `cmd/migrate` and the foundational migration catalog, establishing a robust, organized, and predictable base for schema evolution in Market Foundry.

## Context

- S142 identified `cmd/migrate` as a critical missing pre-condition for ClickHouse adoption.
- S143 defined the migrations infrastructure architecture.
- S144 designed the core analytical schema (6 tables).
- S145 decided the writer service architecture (standalone).
- S146 institutionalizes schema evolution infrastructure before depending on it.

## Deliverables

### 1. `cmd/migrate` Implementation

Standalone CLI tool with three commands:

| Command | Behavior |
|---------|----------|
| `migrate up` | Apply all pending migrations in version order |
| `migrate up --dry-run` | Show pending migrations without applying |
| `migrate status` | Display applied/pending state with checksums |
| `migrate validate` | Verify checksums of applied migrations against disk |

**Technical decisions:**

- Environment-variable configuration (no config file — deployment utility, not service).
- `database/sql` interface in `internal/migrate` — zero external dependencies in core logic.
- ClickHouse driver (`clickhouse-go/v2`) imported only in `cmd/migrate`.
- Automatic database bootstrap (`CREATE DATABASE IF NOT EXISTS`).
- Automatic `_migrations` table bootstrap on every invocation.
- Forward-only model — no down migrations.
- Stop-on-failure semantics — partial catalogs are never left in ambiguous state.

### 2. Migration Catalog

Seven initial migrations in `deploy/migrations/`:

| Version | Name | Purpose |
|---------|------|---------|
| 000 | `create_migrations_metadata` | Bootstrap `_migrations` tracking table |
| 001 | `create_evidence_candles` | Candle events — partitioned by `(timeframe, toYYYYMM(open_time))` |
| 002 | `create_signals` | Signal events (RSI, EMA crossover) |
| 003 | `create_decisions` | Decision evaluation events |
| 004 | `create_strategies` | Strategy resolution events |
| 005 | `create_risk_assessments` | Risk assessment events |
| 006 | `create_executions` | Execution events (paper orders, venue fills) — 365-day TTL |

**Schema highlights:**

- All tables use MergeTree engine.
- Uniform metadata columns: `event_id`, `occurred_at`, `correlation_id`, `causation_id`, `ingested_at`.
- `LowCardinality(String)` for bounded enum fields (`symbol`, `type`, `outcome`, `direction`, `disposition`, `status`, `side`).
- `Float64` for decimal values (acceptable precision for paper trading).
- `String DEFAULT ''` over `Nullable` (avoids null bitmap overhead).
- 90-day TTL on pipeline tables; 365-day TTL on executions (audit).
- All migrations idempotent (`CREATE TABLE IF NOT EXISTS`).

### 3. `internal/migrate` Package

Core migration logic with zero external dependencies:

| File | Responsibility |
|------|---------------|
| `migration.go` | Type definitions: `Migration`, `AppliedMigration`, `MigrationStatus` |
| `catalog.go` | File discovery, filename validation, version parsing, sorting |
| `checksum.go` | SHA-256 hex digest computation |
| `runner.go` | Orchestration: `Up()`, `Status()`, `Validate()` with metadata management |
| `catalog_test.go` | Unit tests: sorting, empty dir, duplicate version, invalid filename |
| `checksum_test.go` | Unit tests: determinism, different content, missing file |

### 4. Makefile Integration

Three new targets sourcing `deploy/envs/local.env` for ClickHouse credentials:

```
make migrate-up         # Apply pending migrations
make migrate-status     # Show migration status
make migrate-validate   # Verify checksums
```

`migrate` added to `BUILDABLE_SERVICES` for `make build`.

### 5. Workspace Integration

- `go.work` updated with `./cmd/migrate` and `./internal/migrate`.
- `cmd/migrate/go.mod` — depends on `clickhouse-go/v2` (resolved via `go mod tidy`).
- `internal/migrate/go.mod` — pure stdlib, no external dependencies.

## Files Created

| Path | Type |
|------|------|
| `cmd/migrate/main.go` | Go source |
| `internal/migrate/migration.go` | Go source |
| `internal/migrate/catalog.go` | Go source |
| `internal/migrate/checksum.go` | Go source |
| `internal/migrate/runner.go` | Go source |
| `internal/migrate/catalog_test.go` | Go test |
| `internal/migrate/checksum_test.go` | Go test |
| `internal/migrate/go.mod` | Go module |
| `cmd/migrate/go.mod` | Go module |
| `cmd/migrate/go.sum` | Go checksum |
| `deploy/migrations/000_create_migrations_metadata.sql` | SQL migration |
| `deploy/migrations/001_create_evidence_candles.sql` | SQL migration |
| `deploy/migrations/002_create_signals.sql` | SQL migration |
| `deploy/migrations/003_create_decisions.sql` | SQL migration |
| `deploy/migrations/004_create_strategies.sql` | SQL migration |
| `deploy/migrations/005_create_risk_assessments.sql` | SQL migration |
| `deploy/migrations/006_create_executions.sql` | SQL migration |
| `docs/architecture/cmd-migrate-and-migration-catalog.md` | Architecture doc |
| `docs/architecture/migration-naming-ordering-and-versioning-rules.md` | Architecture doc |

## Files Modified

| Path | Change |
|------|--------|
| `go.work` | Added `./cmd/migrate` and `./internal/migrate` |
| `Makefile` | Added `migrate` to BUILDABLE_SERVICES; added `migrate-up`, `migrate-status`, `migrate-validate` targets |

## Naming and Versioning Conventions Established

1. **Filename:** `{NNN}_{action}_{target}.sql` — enforced by catalog reader regex.
2. **Version ranges:** 000 metadata, 001–099 core tables, 100–199 telemetry, 200–299 MVs, 300–399 schema evolution.
3. **Append-only versioning** — never insert between, never renumber.
4. **Immutable after apply** — checksum drift detection enforces this.
5. **One DDL per file** — ClickHouse lacks multi-statement transactions.
6. **Required header** — Migration, Created, Description, Idempotent, Reversible fields.
7. **Forward-only** — no down migrations; corrections are new forward migrations.

## Guard Rails Preserved

| Guard Rail | Status |
|-----------|--------|
| No analytics implementation | Respected — tables created but no writer/query code |
| No migration/business config mixing | Respected — migrations are pure DDL |
| No feature inflation in cmd/migrate | Respected — 3 commands, env var config only |
| No coupling to operational baseline | Respected — zero NATS awareness, no domain logic |
| ClickHouse remains optional | Respected — no operational service imports clickhouse-go |
| Documented limits and operations | Respected — architecture docs explicit about scope |

## Validation

| Check | Result |
|-------|--------|
| `go build ./cmd/migrate` | Passes |
| `go test ./internal/migrate/...` | All tests pass |
| `migrate help` | Correct usage output |
| Catalog reader sorts correctly | Verified by unit tests |
| Checksum deterministic | Verified by unit tests |
| Duplicate version rejected | Verified by unit tests |
| Invalid filename rejected | Verified by unit tests |

## Preparation for S147

With `cmd/migrate` and the catalog in place, the next stage can:

1. **Implement `cmd/writer`** — the standalone ClickHouse writer service consuming NATS events.
2. **Run `make migrate-up`** against a live ClickHouse to create all 6 core tables.
3. **Add a `smoke-writer.sh`** test that validates writer → ClickHouse flow independently.
4. **Extend the catalog** as needed (e.g., telemetry tables in the 100–199 range).

The migration infrastructure is ready to receive real schema changes. The critical path item — a governed, versioned, forward-only schema evolution mechanism — is now institutionalized.
