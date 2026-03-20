# Migrations Infrastructure Architecture

> **Stage:** S143 — Migrations and ClickHouse Entry Architecture
> **Status:** Definitive
> **Scope:** Architectural definition only. No implementation.

---

## 1. Purpose

This document defines the architecture of `cmd/migrate` — the single migration tool for Market Foundry — and the operational rules governing the migration catalog at `deploy/migrations/`. It resolves PC-01 (migration tool exists) from the preparation gate by providing a complete design specification.

This document builds on the conventions defined in `future-migration-catalog-organization-guidelines.md` (S141) and elevates them from guidelines to architectural decisions.

**S225 reconciliation note:** the original S143 design described a separate `internal/migrate/` module. After S220, that library was absorbed into `cmd/migrate/migrate/`. The architectural intent is unchanged; the paths below reflect the current code layout.

---

## 2. Architectural Position

```
┌──────────────────────────────────────────────────────────┐
│                    Developer / CI                          │
│                                                           │
│    make migrate-up    make migrate-status    go run ./cmd/migrate up --dry-run │
│         │                    │                     │       │
│         ▼                    ▼                     ▼       │
│    ┌──────────────────────────────────────────────────┐   │
│    │              cmd/migrate                          │   │
│    │                                                   │   │
│    │  Reads: deploy/migrations/*.sql (sorted by NNN)  │   │
│    │  Writes: _migrations table (applied, checksum)   │   │
│    │  Connects: ClickHouse only (no NATS, no other)   │   │
│    └──────────────────────┬───────────────────────────┘   │
│                           │                               │
│                           ▼                               │
│                      ClickHouse                           │
│                    _migrations table                      │
│                    domain tables                          │
└──────────────────────────────────────────────────────────┘
```

**Key property:** `cmd/migrate` is a **standalone tool** with exactly one external dependency: a ClickHouse connection. It has no awareness of NATS, no awareness of domain logic, no awareness of other services. It is a deployment utility, not an application component.

---

## 3. cmd/migrate Design

### 3.1 Commands

| Command | Description | Exit Code |
|---------|-------------|-----------|
| `migrate up` | Apply all pending migrations in order | 0 on success, 1 on failure |
| `migrate up --dry-run` | Show pending migrations without applying | 0 always |
| `migrate status` | Show applied and pending migrations with checksums | 0 always |
| `migrate validate` | Verify checksums of applied migrations match files on disk | 0 if clean, 1 if drift detected |

### 3.2 No Down Migration

`cmd/migrate` does **not** support `down` or rollback commands. This is an intentional architectural decision.

**Rationale:**
- ClickHouse's MergeTree engine is append-only; `DROP TABLE` loses data permanently
- Automatic rollback creates a false sense of safety
- Forward-only migrations (write a new migration to fix the previous) are safer and auditable
- Rollback complexity is unjustified at current scale (single developer, single environment)

If a migration must be reversed, the operator writes a new migration (e.g., `012_drop_table_evidence_candles.sql`) and applies it forward.

### 3.3 Internal Architecture

```
cmd/migrate/
├── main.go                # CLI entry point, connection bootstrap, command dispatch
└── migrate/
    ├── runner.go         # Core migration runner (apply, status, validate)
    ├── catalog.go        # Reads and sorts deploy/migrations/*.sql
    ├── checksum.go       # SHA-256 computation for migration files
    ├── migration.go      # Types: Migration, AppliedMigration, MigrationStatus
    ├── catalog_test.go   # Unit tests for catalog parsing and sorting
    └── checksum_test.go  # Unit tests for checksum computation
```

### 3.4 Configuration

`cmd/migrate` reads ClickHouse connection parameters from environment variables and exposes `--migrations-dir` as a CLI flag:

| Parameter | Env Var | Default | Description |
|-----------|---------|---------|-------------|
| Host | `CLICKHOUSE_HOST` | `localhost` | ClickHouse server address |
| Port | `CLICKHOUSE_PORT` | `9000` | Native protocol port |
| Database | `CLICKHOUSE_DATABASE` | `market_foundry` | Target database |
| User | `CLICKHOUSE_USER` | `default` | ClickHouse username |
| Password | `CLICKHOUSE_PASSWORD` | (none) | ClickHouse password |
| Migrations Dir | `MIGRATIONS_DIR` | `deploy/migrations` | Path to migration catalog |

### 3.5 Execution Semantics

**`migrate up` algorithm:**

```
1. Connect to ClickHouse
2. Ensure _migrations table exists (bootstrap DDL)
3. Read catalog: glob deploy/migrations/*.sql, sort by NNN prefix
4. Read applied: SELECT version, name, checksum FROM _migrations ORDER BY version
5. For each catalog entry not in applied:
   a. Compute SHA-256 checksum of file contents
   b. If --dry-run: print "[PENDING] NNN_name.sql" and continue
   c. Execute SQL contents against ClickHouse
   d. On success: INSERT INTO _migrations (version, name, checksum)
   e. On failure: print error, STOP (do not continue to next migration)
6. Print summary: N applied, M already applied, K total
```

**`migrate validate` algorithm:**

```
1. Connect to ClickHouse
2. Read applied migrations from _migrations table
3. For each applied migration:
   a. Find corresponding file in deploy/migrations/
   b. Compute SHA-256 of file contents
   c. Compare with stored checksum
   d. If mismatch: report drift (file, expected checksum, actual checksum)
   e. If file missing: report missing migration file
4. Exit 0 if all clean, exit 1 if any drift or missing files
```

**Failure semantics:**
- If a migration fails mid-execution, it is NOT recorded in `_migrations`
- The operator must fix the issue and re-run `migrate up`
- Since all migrations are idempotent (`IF NOT EXISTS`), re-running is safe
- No partial state tracking (a migration either fully applied or not recorded)

---

## 4. Migration Catalog Architecture

### 4.1 Directory Layout

```
deploy/
└── migrations/
    ├── 000_create_migrations_metadata.sql    # Bootstrap: _migrations table itself
    ├── 001_create_evidence_candles.sql
    ├── 002_create_signals.sql
    ├── 003_create_decisions.sql
    ├── 004_create_strategies.sql
    ├── 005_create_risk_assessments.sql
    ├── 006_create_executions.sql
    └── ...
```

### 4.2 Migration 000: Self-Bootstrap

The `_migrations` table is itself managed as migration 000. The migrate tool has a special bootstrap path:

1. On first run, check if `_migrations` table exists
2. If not: execute `000_create_migrations_metadata.sql` directly
3. Record migration 000 in the newly created table
4. Proceed with normal `migrate up` flow

This avoids chicken-and-egg: the metadata table is a migration like any other, but the tool knows to bootstrap it first.

### 4.3 _migrations Table DDL

```sql
-- Migration: 000_create_migrations_metadata
-- Created: 2026-03-19
-- Description: Bootstrap metadata table for migration tracking.
--
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: No (dropping this table loses all migration history)

CREATE TABLE IF NOT EXISTS _migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3) DEFAULT now64(3),
    checksum   String
) ENGINE = MergeTree()
ORDER BY version;
```

### 4.4 Naming Convention (Restated from S141)

```
{NNN}_{action}_{target}.sql
```

| Component | Rule | Example |
|-----------|------|---------|
| `NNN` | Zero-padded 3-digit sequence | `001`, `042`, `100` |
| `action` | Verb describing the change | `create`, `add_column`, `alter`, `create_mv` |
| `target` | Table or object name | `evidence_candles`, `signals` |

### 4.5 Reserved Ranges (Restated from S141)

| Range | Purpose |
|-------|---------|
| 000 | Metadata bootstrap |
| 001–099 | Core domain tables |
| 100–199 | Operational telemetry tables |
| 200–299 | Materialized views and aggregations |
| 300–399 | Schema evolution (ALTER, ADD COLUMN) |
| 400+ | Unreserved |

### 4.6 File Structure Requirements

Every migration file MUST include:

```sql
-- Migration: {NNN}_{action}_{target}
-- Created: YYYY-MM-DD
-- Description: One-line purpose.
--
-- Idempotent: Yes (explain mechanism)
-- Reversible: Yes (reverse DDL) | No (justification)

<SQL statements>
```

---

## 5. Makefile Integration

```makefile
# Migration targets
migrate-up:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

migrate-validate:
	go run ./cmd/migrate validate
```

These targets are developer-facing. They assume ClickHouse is running (via `docker compose up clickhouse`).

---

## 6. Versioning Rules

### V-01: Sequence Is Append-Only

New migrations always get the next available number. Never insert between existing migrations. Never renumber.

### V-02: Applied Migrations Are Immutable

Once a migration is applied to any environment, its file MUST NOT change. `migrate validate` enforces this via checksum comparison. If a fix is needed, create a new migration.

### V-03: One Catalog, One Tool

There is exactly one migration catalog (`deploy/migrations/`) and one tool (`cmd/migrate`). No per-service directories. No competing tools. No shell scripts that apply DDL directly.

### V-04: Idempotency Is Mandatory

Every migration uses `IF NOT EXISTS`, `IF EXISTS`, or equivalent guards. Re-running `migrate up` on a fully-applied catalog is a no-op (each migration checked against `_migrations` table).

### V-05: Forward-Only

No automatic rollback. Corrections are new migrations. This is simpler, safer, and auditable.

---

## 7. Relationship to Other Components

### 7.1 cmd/migrate ↔ cmd/writer

`cmd/migrate` creates and evolves the ClickHouse schema. `cmd/writer` populates the tables. They share no code and have no runtime dependency on each other. The operational contract between them is the table DDL.

```
cmd/migrate creates tables  →  cmd/writer inserts rows
```

`cmd/migrate` runs before `cmd/writer` starts (or at any time to apply new migrations). `cmd/writer` assumes tables exist; if a table is missing, the inserter logs an error and skips.

### 7.2 cmd/migrate ↔ deploy/configs/

No relationship. Migrations are schema artifacts. Configs are runtime behavior. They are in the same `deploy/` parent directory for operational co-location, not because they interact.

### 7.3 cmd/migrate ↔ NATS

No relationship. `cmd/migrate` does not connect to NATS, does not consume events, does not publish anything. It is a pure ClickHouse schema management tool.

### 7.4 cmd/migrate ↔ CI (Future)

When CI exists, `migrate validate` should run on every PR that touches `deploy/migrations/`. This prevents checksum drift from accidental edits to applied migrations.

---

## 8. Anti-Patterns

| Anti-Pattern | Why It's Dangerous | Prevention |
|---|---|---|
| Manual `CREATE TABLE` in ClickHouse shell | Untracked state; `migrate validate` will report drift | All DDL goes through migration files |
| Editing an applied migration file | Breaks checksum integrity | `migrate validate` in CI catches this |
| Migration with side effects (data insertion) | Migrations are schema-only; data comes from writer | Code review: no INSERT of domain data in migrations |
| Conditional logic in SQL migrations | ClickHouse SQL has limited procedural support; complexity leads to failures | Keep migrations simple: one DDL statement per file when possible |
| Coupling migration tool to NATS or domain logic | Violates standalone principle; complicates testing | `cmd/migrate` imports only `cmd/migrate/migrate` and ClickHouse driver |
| Using `migrate up` as part of service startup | Creates ordering dependency; complicates compose | Migrations run explicitly before starting services |

---

## 9. Open Design Questions for Implementation

These questions are identified but intentionally **not resolved** at the architecture level. They are implementation decisions for the stage that builds `cmd/migrate`:

| Question | Options | Recommendation |
|----------|---------|----------------|
| ClickHouse Go driver | `clickhouse-go` v2 (official) vs `ch-go` (low-level) | `clickhouse-go` v2 — well-maintained, supports native protocol |
| CLI framework | `flag` (stdlib) vs `cobra` | `flag` — minimal tool, no subcommand complexity needed |
| Config loading | Env vars only vs JSONC config | Env vars only — consistent with compose, no config file to manage |
| Transaction semantics | Per-migration vs per-batch | Per-migration — ClickHouse has limited transaction support anyway |

---

## 10. Success Criteria for cmd/migrate Implementation

When `cmd/migrate` is built (future stage), it must satisfy:

| Criterion | Verification |
|-----------|-------------|
| `migrate up` applies pending migrations in order | Run against empty CH, verify tables created |
| `migrate up` is idempotent | Run twice, second run applies nothing |
| `migrate up --dry-run` shows pending without applying | Run, verify no tables created |
| `migrate status` shows applied and pending | Run after partial apply, verify output |
| `migrate validate` detects checksum drift | Modify an applied migration file, verify exit code 1 |
| `migrate validate` detects missing files | Delete a migration file, verify exit code 1 |
| No NATS dependency | Run with NATS stopped, verify success |
| Failure stops execution | Introduce a broken migration, verify subsequent migrations not applied |
| _migrations table is self-bootstrapping | Run against fresh CH with no tables, verify _migrations created |
