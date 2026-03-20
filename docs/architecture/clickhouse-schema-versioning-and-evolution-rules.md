# ClickHouse Schema Versioning and Evolution Rules

> **Stage:** S144 — Core Analytical Schema Design
> **Status:** Definitive
> **Scope:** Rules governing how the ClickHouse schema evolves over time.

---

## 1. Purpose

This document defines the rules for versioning and evolving the ClickHouse schema in Market Foundry. It bridges the gap between the migration tool design (S143, `migrations-infrastructure-architecture.md`) and the practical realities of schema change in an event-driven analytical system.

The core tension: the Go event structs are the source of truth, but ClickHouse tables are a projection of those structs. When the Go structs change, the ClickHouse schema must follow — but not automatically, and not immediately.

---

## 2. Schema Version Model

### 2.1 There Is No Schema Version Number

The ClickHouse schema does not carry an explicit version number (no `schema_version` column, no version table). Instead, the schema version is implicit in the migration catalog: the set of applied migrations defines the current schema state.

**Rationale:**
- A version number adds a column to every table that must be maintained and checked
- The migration catalog (`_migrations` table) already tracks what has been applied
- `migrate status` shows the current schema state
- Adding a version number creates a second source of truth that can drift from the actual applied migrations

### 2.2 Migration Sequence Is the Version History

```
Schema state = set of applied migrations
Schema version at time T = max(version) in _migrations at time T
```

The migration catalog is append-only and forward-only. This means:
- The schema can only move forward (new migrations)
- The full version history is in `_migrations`
- Any two environments can be compared by diffing their `_migrations` contents

---

## 3. Evolution Rules

### EV-01: Go Struct Changes Trigger Schema Review, Not Automatic Migration

**Rule:** When a Go event struct changes, the developer must evaluate whether a ClickHouse migration is needed. Not all struct changes require schema changes.

| Struct Change | Schema Impact | Action Required |
|---------------|---------------|-----------------|
| New field added (scalar) | New column needed | Write `ADD COLUMN` migration |
| New field added (to a map/JSON field) | No schema impact | None — JSON absorbs new keys |
| Field type changed (e.g., `int` → `int64`) | Column type may need change | Write `MODIFY COLUMN` migration |
| Field removed | Column becomes unused | Evaluate: drop column migration or leave dormant |
| Field renamed | Column name mismatch | Write migration (add new, drop old) — or update writer only |
| New enum value added | No schema impact | None — LowCardinality(String) accepts new values |
| Nested struct changed | No schema impact if stored as JSON | None — JSON absorbs structural changes |

**Key insight:** Storing nested structures as JSON strings (signals, constraints, fills, etc.) provides a buffer against schema churn. Only top-level typed columns require migrations when Go structs change.

### EV-02: Additive Changes Are Preferred

**Rule:** Prefer `ADD COLUMN` over `MODIFY COLUMN` or `DROP COLUMN`.

- Adding a column is non-destructive and backward-compatible
- The new column has a DEFAULT value for existing rows
- The writer starts populating it immediately
- Old rows with the default are distinguishable from new rows with real values

```sql
-- Preferred: additive
ALTER TABLE signals ADD COLUMN IF NOT EXISTS spread Float64 DEFAULT 0;

-- Avoid: destructive (data loss)
ALTER TABLE signals DROP COLUMN metadata;

-- Avoid when possible: type change (requires data rewrite)
ALTER TABLE signals MODIFY COLUMN value Decimal128(18);
```

### EV-03: Column Removal Is a Two-Phase Process

**Rule:** Removing a column follows a two-phase approach across separate migrations:

1. **Phase 1 — Stop writing:** Update the writer to stop populating the column. The column remains in the schema but receives only DEFAULT values. This is a code change, not a migration.
2. **Phase 2 — Drop column:** After confirming no queries depend on the column, write a `DROP COLUMN` migration.

**Rationale:** Immediate column removal breaks any in-flight queries and loses historical data. The two-phase approach provides a grace period.

### EV-04: One DDL Statement per Migration File

**Rule:** Each migration file should contain exactly one DDL statement (one `CREATE TABLE`, one `ALTER TABLE`, one `DROP TABLE`).

**Rationale:**
- ClickHouse does not support multi-statement transactions
- If a migration with 3 statements fails on statement 2, the tool cannot roll back statement 1
- Single-statement migrations have atomic success/failure semantics
- Exception: `CREATE TABLE` followed by a comment-only operation is acceptable

### EV-05: JSON Columns Are Schema-Flexible Buffers

**Rule:** Columns stored as `String` (containing JSON) do not require migrations when the JSON structure changes. The writer simply serializes the new structure.

This means:
- Adding a new key to `Signal.Metadata` map → no migration needed
- Adding a new field to `Constraints` struct → no migration needed (stored as JSON in `risk_assessments.constraints`)
- Removing a key from a JSON column → no migration needed (old keys remain in historical rows)

**When a JSON column SHOULD become typed columns:**

If a specific JSON field is queried frequently (>50% of queries on that table extract the same field), it should be promoted to a materialized column:

```sql
ALTER TABLE signals
    ADD COLUMN IF NOT EXISTS rsi_period UInt32
    MATERIALIZED toUInt32OrZero(JSONExtractString(metadata, 'period'));
```

This is an optimization migration, not a schema evolution.

---

## 4. Migration Categories

### 4.1 Schema Creation (Range 001–099)

Initial table creation. Defined in S144, applied by `cmd/migrate`.

| Migration | Purpose |
|-----------|---------|
| 001–006 | Core 6 event tables |
| 007–009 | Reserved for supplementary evidence tables (tradebursts, volumes, fills) |
| 010–099 | Reserved for future core domain tables |

### 4.2 Schema Extension (Range 300–399)

`ADD COLUMN` and `MODIFY COLUMN` operations driven by Go struct evolution.

Naming convention:
```
3XX_add_column_{table}_{column}.sql
3XX_modify_column_{table}_{column}.sql
```

Examples:
```
301_add_column_signals_spread.sql
302_modify_column_evidence_candles_volume_decimal.sql
```

### 4.3 Schema Optimization (Range 200–299)

Materialized views, materialized columns, and index changes driven by query patterns.

Naming convention:
```
2XX_create_mv_{name}.sql
2XX_add_materialized_{table}_{column}.sql
```

### 4.4 Schema Cleanup (Range 300–399, shared with extension)

`DROP COLUMN` operations after the two-phase removal process.

Naming convention:
```
3XX_drop_column_{table}_{column}.sql
```

---

## 5. Compatibility Rules

### CR-01: Writer Must Handle Schema Lag

**Rule:** The writer must tolerate a schema that is older than the Go struct it is writing. If the writer has a field that the table doesn't have a column for, the writer skips that field (does not fail).

**Rationale:** During deployment, the writer binary may be updated before migrations run. The writer must not crash because a column doesn't exist yet.

**Implementation guidance:** The writer should use explicit column lists in INSERT statements, not `INSERT INTO table VALUES (...)`. This way, new fields in the Go struct that don't have corresponding columns are simply omitted.

### CR-02: Queries Must Handle Missing Columns Gracefully

**Rule:** If a query references a column that doesn't exist (because a migration hasn't been applied), ClickHouse will return an error. This is expected behavior — the query should be updated to match the current schema, or the migration should be applied.

There is no automatic schema discovery or column existence checking in queries. Queries are written against the known schema.

### CR-03: Old Data Coexists with New Data

**Rule:** After an `ADD COLUMN` migration, old rows have the column's DEFAULT value. Queries must account for this:

```sql
-- After adding 'spread' column with DEFAULT 0:
-- Old rows: spread = 0
-- New rows: spread = actual value

-- Query that handles both:
SELECT * FROM signals WHERE spread > 0  -- Only rows with real spread values
```

This is the cost of additive evolution. It is acceptable because:
- DEFAULT values are explicit and documented in the migration
- Queries can filter on the presence of real values
- The alternative (backfilling old rows) is expensive and often impossible (the data wasn't captured)

---

## 6. Event Schema Versioning (Deferred)

S142 identified PC-05 (event schema versioning convention) as an important pre-condition. This design **defers** formal event schema versioning.

**Why deferred:**
- At single-developer scale, Go struct changes are visible in the same codebase
- The writer and the structs are compiled together — schema mismatches are caught at compile time
- The ClickHouse schema follows the Go struct via manual review (EV-01), not automated sync
- Formal versioning (e.g., schema registry, version field in NATS envelope) adds overhead with no consumer at current scale

**When to revisit:**
- Multiple developers changing event schemas independently
- Multiple writers consuming different schema versions
- Need for backward-compatible schema evolution (old writer, new schema)

**Interim convention:** The migration file header includes a `-- Source:` line referencing the Go struct:

```sql
-- Migration: 301_add_column_signals_spread
-- Created: 2026-04-15
-- Source: internal/domain/signal/signal.go (Signal struct)
-- Description: Add spread column for bid-ask spread signal support.
```

This creates a traceable link between the ClickHouse migration and the Go struct change that triggered it.

---

## 7. Testing Schema Changes

### 7.1 Before Applying a Migration

```bash
# Show what will be applied
make migrate-dry-run

# Verify current state
make migrate-status
```

### 7.2 After Applying a Migration

```bash
# Apply the migration
make migrate-up

# Verify checksums are clean
make migrate-validate

# Verify the table exists and has expected columns
# (via ClickHouse client or a future smoke test)
```

### 7.3 Detecting Unintended Changes

```bash
# After any manual ClickHouse interaction:
make migrate-validate
# Exit code 1 = someone changed a migration file or applied DDL outside the tool
```

---

## 8. Summary of Rules

| Rule | Statement |
|------|-----------|
| **EV-01** | Go struct changes trigger manual review, not automatic migration |
| **EV-02** | Prefer additive changes (ADD COLUMN) |
| **EV-03** | Column removal is two-phase (stop writing → drop column) |
| **EV-04** | One DDL statement per migration file |
| **EV-05** | JSON columns absorb structural changes without migration |
| **CR-01** | Writer tolerates schema lag (missing columns) |
| **CR-02** | Queries fail explicitly on missing columns |
| **CR-03** | Old data coexists with new data (DEFAULT values) |

---

## 9. Anti-Patterns

| Anti-Pattern | Why Dangerous | Prevention |
|---|---|---|
| Adding `schema_version` column to every table | Second source of truth; drifts from migration catalog | Use `_migrations` as the single version source |
| Automatic migration generation from Go structs | Brittle, context-free; can't reason about data impact | Manual review per EV-01 |
| Backfilling old rows after ADD COLUMN | Expensive, sometimes impossible, rarely necessary | Accept DEFAULT values; filter in queries |
| Migration with `IF NOT EXISTS` on ALTER | ClickHouse ALTERs are not idempotent the same way CREATEs are | Test migrations on a clean copy; document expected state |
| Dropping columns immediately after Go struct change | Loses historical data; breaks in-flight queries | Two-phase removal per EV-03 |
| Storing schema version in NATS event envelope | Adds coupling; NATS is transport, not a schema registry | Deferred — compile-time safety is sufficient for now |
