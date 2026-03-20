# Migration Naming, Ordering, and Versioning Rules

## Naming Convention

Every migration file follows the pattern:

```
{NNN}_{action}_{target}.sql
```

| Component | Rule                       | Examples                              |
|-----------|----------------------------|---------------------------------------|
| `NNN`     | Zero-padded 3-digit number | `001`, `042`, `200`                   |
| `action`  | Verb describing the change | `create`, `add_column`, `drop_column`, `alter`, `create_mv` |
| `target`  | Table or object affected   | `evidence_candles`, `signals`, `daily_candle_rollup` |

The filename regex enforced by the catalog reader:

```
^(\d{3})_.+\.sql$
```

### Examples

```
000_create_migrations_metadata.sql
001_create_evidence_candles.sql
002_create_signals.sql
011_add_column_signals_confidence.sql
200_create_mv_daily_candle_rollup.sql
301_alter_executions_add_fee.sql
```

## Reserved Version Ranges

| Range   | Purpose                              | Current Usage     |
|---------|--------------------------------------|-------------------|
| 000     | Metadata bootstrap (`_migrations`)   | 000 (used)        |
| 001–099 | Core domain tables                   | 001–006 (used), 007–099 (reserved) |
| 100–199 | Operational telemetry tables         | (reserved)        |
| 200–299 | Materialized views                   | (reserved)        |
| 300–399 | Schema evolution (ALTER, ADD, DROP)   | (reserved)        |
| 400+    | Unreserved                           | (available)       |

## Versioning Rules

### V-01: Sequence Is Append-Only

New migrations always receive the next available number. Never insert between existing migrations. Never renumber applied migrations.

**Rationale:** Stable ordering is the foundation of deterministic schema state. Inserting or renumbering creates ambiguity about what was applied in which order.

### V-02: Applied Migrations Are Immutable

Once a migration has been applied (recorded in `_migrations`), its file content must never change. If a fix is needed, write a new migration with the next available number.

**Rationale:** Checksum validation detects tampering. Immutability ensures that the schema state is reproducible from the catalog.

### V-03: One Catalog, One Tool

All migrations live in `deploy/migrations/`. All schema changes go through `cmd/migrate`. No competing tools, no per-service directories, no ad-hoc DDL.

**Rationale:** Single source of truth prevents version conflicts and split-brain schema states.

### V-04: Idempotency Mandatory

Every migration must use `IF NOT EXISTS` (for CREATE) or `IF EXISTS` (for DROP/ALTER) guards. Running a migration twice must produce no error and no side effects.

**Rationale:** Safe re-runs after partial failures. The runner stops on error; the operator fixes the issue and re-runs. Idempotency guarantees that already-applied statements don't break.

### V-05: Forward-Only

There are no down migrations. If a change needs to be reversed, write a new forward migration that performs the reverse operation.

**Rationale:** ClickHouse MergeTree is append-only. DROP TABLE loses data. Forward-only avoids accidental data loss and simplifies the tool.

### V-06: One DDL Statement Per File

Each migration file contains exactly one DDL statement. ClickHouse does not support multi-statement transactions; a failure mid-file would leave the schema in an ambiguous state.

**Rationale:** Atomic success/failure tracking. Each file either succeeds completely or fails completely, and the `_migrations` record reflects that.

## Required File Header

Every `.sql` file must include a standardized header:

```sql
-- Migration: {NNN}_{action}_{target}
-- Created: YYYY-MM-DD
-- Description: One-line purpose.
-- Source: internal/domain/{package}/{file}.go ({struct name})
-- Idempotent: Yes ({mechanism})
-- Reversible: Yes ({reverse DDL}) | No ({justification})
```

| Field        | Required | Notes                                           |
|-------------|----------|--------------------------------------------------|
| Migration   | Yes      | Must match filename (without `.sql`)             |
| Created     | Yes      | Date the migration was written                   |
| Description | Yes      | One-line purpose                                  |
| Source      | No       | Go struct that defines the event schema          |
| Idempotent  | Yes      | Must be `Yes` with mechanism explanation         |
| Reversible  | Yes      | `Yes` with reverse DDL, or `No` with reason      |

## Checksum Tracking

Each migration's SHA-256 hex digest is computed from the full file content (including header comments) and stored in `_migrations.checksum` at apply time.

`migrate validate` recomputes checksums from current disk files and compares against stored values. Any mismatch is reported as **drift**.

## Ordering Semantics

The catalog reader:
1. Globs `deploy/migrations/*.sql`
2. Validates each filename against `^(\d{3})_.+\.sql$`
3. Parses the 3-digit version number
4. Rejects duplicate version numbers
5. Sorts ascending by version number
6. Returns the ordered list

The runner applies pending migrations in this sorted order. A migration is "pending" if its version number is not present in `_migrations`.

## Creating a New Migration

1. Determine the next available version number (check `deploy/migrations/`).
2. Create the file following the naming convention.
3. Write idempotent SQL with the required header.
4. Test locally: `make migrate-up` (requires ClickHouse running).
5. Verify: `make migrate-status` and `make migrate-validate`.
6. Commit with a clear message referencing the migration.

## Catalog Location

```
deploy/
├── configs/       # Service JSONC configs
├── compose/       # Docker Compose files
├── envs/          # Environment variable files
├── migrations/    # ClickHouse migration catalog (this)
├── clickhouse/    # ClickHouse server config
└── nats/          # NATS server config
```

**Why `deploy/migrations/`:**
- Migrations are deployment artifacts, not application code.
- Co-located with configs and compose for operational coherence.
- Single flat directory — no per-service subdirectories.

**Why NOT:**
- `internal/migrations/` — migrations are SQL, not Go code.
- `deploy/clickhouse/migrations/` — unnecessary nesting.
- Per-service dirs — violates single-catalog principle (V-03).
- `cmd/migrate/migrations/` — couples catalog to tool.
