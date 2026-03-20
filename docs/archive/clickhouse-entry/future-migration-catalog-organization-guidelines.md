# Future Migration Catalog Organization Guidelines

> **Stage:** S141 — Ergonomics and Governance Consolidation
> **Scope:** Conventions only. No implementation.

---

## 1. Purpose

When ClickHouse migrations enter the monorepo, they need a clear, predictable organization. This document defines the naming conventions, directory layout, versioning rules, and lifecycle for the migration catalog — so that when the time comes, there is no improvisation.

---

## 2. Directory Layout

```
deploy/
├── configs/              # Service JSONC configs (existing)
├── compose/              # Docker Compose files (existing)
└── migrations/           # ClickHouse migration catalog (future)
    ├── 001_create_evidence_candles.sql
    ├── 002_create_evidence_tradebursts.sql
    ├── 003_create_evidence_volumes.sql
    ├── 004_create_signals.sql
    ├── 005_create_decisions.sql
    ├── ...
    └── README.md         # Migration catalog documentation
```

### Why `deploy/migrations/`

- Migrations are deployment artifacts, not application code — they belong in `deploy/`
- Single directory, flat structure — no subdirectories by domain or date
- Co-located with configs and compose for operational coherence

### Why NOT alternatives

| Alternative | Rejected because |
|------------|------------------|
| `internal/migrations/` | Migrations are not Go code; they're SQL deployment artifacts |
| `deploy/clickhouse/migrations/` | Over-nesting; if we add Postgres later, we'd have competing trees |
| Per-service directories | Violates single-catalog principle (P-03) |
| `cmd/migrate/migrations/` | Couples catalog to tool; catalog should be tool-agnostic |

---

## 3. Naming Convention

### File Format

```
{NNN}_{action}_{target}.sql
```

| Component | Rule | Example |
|-----------|------|---------|
| `NNN` | Zero-padded 3-digit sequence | `001`, `042`, `100` |
| `action` | Verb describing the change | `create`, `add_column`, `drop_column`, `alter`, `create_mv` |
| `target` | Table or object name | `evidence_candles`, `signals`, `runtime_telemetry` |

### Examples

```
001_create_evidence_candles.sql
002_create_evidence_tradebursts.sql
003_create_evidence_volumes.sql
004_create_signals.sql
005_create_decisions.sql
006_create_strategies.sql
007_create_risk_assessments.sql
008_create_execution_intents.sql
009_create_fills.sql
010_create_runtime_telemetry.sql
011_add_column_signals_confidence.sql
012_create_mv_daily_candle_rollup.sql
```

### Reserved Ranges

| Range | Purpose |
|-------|---------|
| 001–099 | Core domain tables (evidence, signal, decision, strategy, risk, execution) |
| 100–199 | Operational telemetry tables |
| 200–299 | Materialized views and aggregations |
| 300–399 | Schema evolution (ALTER, ADD COLUMN, etc.) |
| 400+ | Unreserved |

---

## 4. Migration File Structure

Each `.sql` file must follow this structure:

```sql
-- Migration: 001_create_evidence_candles
-- Created: 2026-XX-XX
-- Description: Creates the evidence_candles table for historical candle storage.
--
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE evidence_candles)

CREATE TABLE IF NOT EXISTS evidence_candles (
    source       LowCardinality(String),
    symbol       LowCardinality(String),
    timeframe    UInt32,
    open_time    DateTime64(3),
    close_time   DateTime64(3),
    open         Float64,
    high         Float64,
    low          Float64,
    close        Float64,
    volume       Float64,
    trade_count  UInt32,
    final        Bool,
    ingested_at  DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)
TTL open_time + INTERVAL 90 DAY;
```

### Required Header Fields

| Field | Purpose |
|-------|---------|
| `Migration` | Must match filename (without `.sql`) |
| `Created` | Date the migration was written |
| `Description` | One-line purpose |
| `Idempotent` | Must be `Yes` — all migrations must be re-runnable |
| `Reversible` | `Yes` with reverse DDL, or `No` with justification |

---

## 5. Versioning Rules

### V-01: Sequence Is Append-Only

New migrations always get the next available number. Never insert a migration between existing ones. Never renumber existing migrations.

### V-02: Applied Migrations Are Immutable

Once a migration has been applied to any environment, its file content must never change. If a fix is needed, create a new migration.

### V-03: Metadata Table

The migration tool records applied migrations in a ClickHouse table:

```sql
CREATE TABLE IF NOT EXISTS _migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3) DEFAULT now64(3),
    checksum   String
) ENGINE = MergeTree()
ORDER BY version;
```

### V-04: Checksum Verification

The tool computes a SHA-256 checksum of each migration file and stores it. On re-run, if a checksum differs from the recorded value, the tool refuses to proceed (drift detection).

---

## 6. Lifecycle

### Creating a Migration

1. Determine the next sequence number
2. Create the file following the naming convention
3. Write idempotent SQL with the required header
4. Test locally: `cmd/migrate up` against a local ClickHouse
5. Commit with a clear message referencing the migration purpose

### Applying Migrations

```bash
# Apply all pending migrations
cmd/migrate up

# Show migration status
cmd/migrate status

# Dry-run (show what would be applied)
cmd/migrate up --dry-run
```

### Rolling Back

Rolling back is not automatic. If a migration must be reversed:

1. Write a new migration that reverses the change
2. Apply it as the next version
3. Document the reversal reason in the migration header

This is intentional — automatic rollback in ClickHouse is dangerous because of the append-only nature of MergeTree engines.

---

## 7. Retention Policy Guidelines

| Table Category | Suggested TTL | Rationale |
|---------------|---------------|-----------|
| Evidence (candles, bursts, volume) | 90 days | Sufficient for backtesting; can be extended |
| Signals, decisions, strategies | 90 days | Follows evidence lifecycle |
| Risk assessments | 90 days | Follows signal lifecycle |
| Executions, fills | 365 days | Longer for audit/compliance |
| Runtime telemetry | 30 days | Operational data, high volume |
| Materialized views | Follows source table | TTL propagates |

TTL is set per-table in the migration SQL. Changing TTL requires a new migration (`ALTER TABLE ... MODIFY TTL`).

---

## 8. Relationship to `cmd/migrate`

The migration catalog (`deploy/migrations/`) is decoupled from the migration tool (`cmd/migrate/`). The catalog is just SQL files. The tool reads them, applies them, and tracks state.

If `cmd/migrate` is not yet created, migrations can be applied manually in version order — the convention ensures this works. The tool adds automation and safety (checksum, dry-run, status), not capability.

When `cmd/migrate` is introduced:
- It lives in `cmd/migrate/` as a Go binary
- It has no NATS dependency — only ClickHouse
- It reads `deploy/migrations/*.sql` sorted by prefix number
- It maintains state in the `_migrations` table
- It is added to the Makefile as `make migrate-up`, `make migrate-status`
