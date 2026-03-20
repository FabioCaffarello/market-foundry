# cmd/migrate and Migration Catalog

## Purpose

`cmd/migrate` is a standalone deployment utility that manages ClickHouse schema evolution for Market Foundry. It reads SQL migration files from the canonical catalog (`deploy/migrations/`), applies them in version order, and tracks state in a `_migrations` metadata table.

## Architectural Position

- **Standalone CLI tool** — not a long-running service.
- **Single external dependency** — ClickHouse only. No NATS, no domain logic.
- **Deployment artifact** — runs before or independently of operational services.
- **Forward-only** — no down/rollback migrations.

## Commands

| Command              | Purpose                                    | Exit Code     |
|----------------------|--------------------------------------------|---------------|
| `migrate up`         | Apply all pending migrations in order      | 0 success / 1 failure |
| `migrate up --dry-run` | Show pending migrations without applying | 0 always      |
| `migrate status`     | Show applied/pending with checksums        | 0 always      |
| `migrate validate`   | Verify applied checksums match disk files  | 0 clean / 1 drift |

## Configuration

All configuration via environment variables:

| Variable             | Default          | Description                    |
|----------------------|------------------|--------------------------------|
| `CLICKHOUSE_HOST`    | `localhost`      | ClickHouse server host         |
| `CLICKHOUSE_PORT`    | `9000`           | ClickHouse native protocol port |
| `CLICKHOUSE_DATABASE`| `market_foundry` | Target database name           |
| `CLICKHOUSE_USER`    | `default`        | ClickHouse user                |
| `CLICKHOUSE_PASSWORD`| *(empty)*        | ClickHouse password            |
| `MIGRATIONS_DIR`     | `deploy/migrations` | Path to migration catalog   |

## Project Layout

```
cmd/migrate/
└── main.go              # CLI entry point, ClickHouse connection, subcommand dispatch

internal/migrate/
├── migration.go         # Types: Migration, AppliedMigration, MigrationStatus
├── catalog.go           # ReadCatalog: discover, validate, sort migration files
├── checksum.go          # FileChecksum: SHA-256 hex digest
├── runner.go            # Runner: Up, Status, Validate orchestration
├── catalog_test.go      # Unit tests for catalog parsing and sorting
└── checksum_test.go     # Unit tests for checksum computation

deploy/migrations/
├── 000_create_migrations_metadata.sql
├── 001_create_evidence_candles.sql
├── 002_create_signals.sql
├── 003_create_decisions.sql
├── 004_create_strategies.sql
├── 005_create_risk_assessments.sql
└── 006_create_executions.sql
```

## Execution Flow

```
1. Parse CLI flags and environment variables
2. Connect to ClickHouse (default database)
3. CREATE DATABASE IF NOT EXISTS {target}
4. Reconnect to target database
5. CREATE TABLE IF NOT EXISTS _migrations (bootstrap)
6. Read catalog: glob deploy/migrations/*.sql, sort by NNN prefix
7. Read applied: SELECT from _migrations
8. For each pending migration (in order):
   a. Compute SHA-256 checksum
   b. Execute SQL content (or print if --dry-run)
   c. INSERT into _migrations on success
   d. STOP on failure
```

## Metadata Table

```sql
CREATE TABLE IF NOT EXISTS _migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3) DEFAULT now64(3),
    checksum   String
) ENGINE = MergeTree()
ORDER BY version
```

The runner bootstraps this table on every invocation via `CREATE TABLE IF NOT EXISTS`. Migration 000 in the catalog contains the same DDL for documentation completeness.

## Failure Semantics

- If a migration fails, it is **not recorded** in `_migrations`.
- Execution stops immediately — subsequent migrations are not attempted.
- The operator fixes the issue and re-runs `migrate up`.
- All migrations are idempotent (`IF NOT EXISTS` / `IF EXISTS` guards), so re-running is safe.

## Checksum Drift Detection

`migrate validate` computes the current SHA-256 of each applied migration's file and compares it to the checksum stored at apply time. Any mismatch is reported as drift.

Drift indicates that a migration file was modified after being applied — a violation of the immutability rule (V-02).

## Dependency Boundaries

| Package                     | May Import `clickhouse-go`? |
|-----------------------------|---------------------------|
| `cmd/migrate`               | **Yes** (driver import)   |
| `internal/migrate`          | **No** (uses `database/sql` interface) |
| `cmd/gateway`, `cmd/store`, etc. | **No** (operational services) |

`internal/migrate` has zero external dependencies — it operates purely through the `database/sql` interface. Only `cmd/migrate` imports the ClickHouse driver.

## Makefile Integration

```bash
make migrate-up         # Apply pending migrations (sources deploy/envs/local.env)
make migrate-status     # Show migration status
make migrate-validate   # Verify checksums
```

## Relationship to Other Components

| Component    | Relationship                                     |
|-------------|--------------------------------------------------|
| `cmd/writer` | Writer assumes tables exist; migrations run first |
| `cmd/store`  | No relationship — store uses NATS KV only        |
| `cmd/gateway`| No relationship — gateway reads NATS KV only     |
| ClickHouse   | Only external dependency                          |
| Docker Compose | ClickHouse service must be healthy before running |

## Design Decisions

**No down migrations.** ClickHouse MergeTree is append-only; DROP TABLE loses data permanently. Forward-only model: write a new migration to reverse a previous one.

**No external CLI framework.** Standard `flag` package suffices for three subcommands. Avoids dependency bloat.

**`database/sql` interface in `internal/migrate`.** Keeps the core logic driver-agnostic and testable. The ClickHouse driver is imported only in `cmd/migrate`.

**Database bootstrap in `cmd/migrate`.** The CLI creates the target database if it doesn't exist, keeping the migration catalog focused on schema (tables, views).

## Limits

- L-01: No down/rollback migrations.
- L-02: No multi-statement transactions (ClickHouse limitation).
- L-03: No migration generation tooling (files are hand-written).
- L-04: No migration locking (single-operator model at current scale).
- L-05: No conditional/environment-specific migrations.
- L-06: Single ClickHouse instance target.
