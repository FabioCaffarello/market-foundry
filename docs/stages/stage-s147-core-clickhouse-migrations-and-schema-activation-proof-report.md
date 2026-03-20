# Stage S147 — Core ClickHouse Migrations and Schema Activation Proof

**Status:** Complete
**Date:** 2026-03-19

## Objective

Create and validate the migrations for the 6 core analytical tables, proving that the schema evolution mechanism designed in S143–S146 works predictably against a real ClickHouse instance.

## Context

- S144 defined the canonical schema design (6 tables, column mappings, partition/order keys, TTLs).
- S146 implemented `cmd/migrate` and created draft migration files.
- S147 corrected the migrations to match S144 exactly and proved activation against ClickHouse 24.8.8.

## Summary of Work

### 1. Migration Corrections

The S146 draft migrations diverged from the S144 canonical design. All 6 files (001–006) were rewritten before first application:

**Columns restored across all tables:**
- `source` (`LowCardinality(String)`) — exchange identifier, part of ORDER BY
- `final` (`Bool`) — window closed/finalized state

**Per-table columns restored:**

| Table | Restored Columns |
|-------|-----------------|
| decisions | `confidence` (Float64), `metadata` (String) |
| strategies | `confidence` (Float64), `decisions` (String) |
| risk_assessments | `confidence` (Float64), `strategies` (String), `rationale` (String), `parameters` (String), `metadata` (String) |
| executions | `filled_quantity` (Float64, was incorrectly `price`), `risk` (String), `parameters` (String), `metadata` (String) |

**ORDER BY corrected:** All tables now lead with `source` per S144 design.

**executions TTL corrected:** 365 DAY → 90 DAY (uniform with all core tables per S144).

### 2. Runtime Compatibility Fix

**TTL on DateTime64:** ClickHouse 24.8.8 rejects TTL expressions on `DateTime64` columns directly. All migrations now use `TTL toDateTime(<column>) + INTERVAL 90 DAY`. Precision loss is irrelevant at TTL cleanup granularity.

### 3. Docker Compose Fix

**ClickHouse container crash loop:** The `CLICKHOUSE_PASSWORD` env var triggers the entrypoint to rewrite the users XML, which was mounted `:ro`. Removed `:ro` from the users XML volume mount.

### 4. Activation Proof

All 7 migrations applied successfully against ClickHouse 24.8.8:

```
migrate up
  applying 000_create_migrations_metadata ... OK
  applying 001_create_evidence_candles ... OK
  applying 002_create_signals ... OK
  applying 003_create_decisions ... OK
  applying 004_create_strategies ... OK
  applying 005_create_risk_assessments ... OK
  applying 006_create_executions ... OK
```

**Validation results:**

| Check | Result |
|-------|--------|
| `migrate status` — 7 applied, 0 pending | Pass |
| `migrate validate` — all checksums valid | Pass |
| `migrate up` re-run — idempotent no-op | Pass |
| `SHOW TABLES` — 7 tables in `market_foundry` | Pass |
| `SHOW CREATE TABLE` — DDL matches S144 for all tables | Pass |
| Sample INSERT into all 6 domain tables | Pass |
| Go test suite (`make test`) — no regressions | Pass |

## Files Modified

| Path | Change |
|------|--------|
| `deploy/migrations/001_create_evidence_candles.sql` | Rewritten: added `source`, `final`; fixed ORDER BY; fixed TTL |
| `deploy/migrations/002_create_signals.sql` | Rewritten: added `source`, `final`; fixed ORDER BY; fixed TTL |
| `deploy/migrations/003_create_decisions.sql` | Rewritten: added `source`, `confidence`, `metadata`, `final`; fixed ORDER BY; fixed TTL |
| `deploy/migrations/004_create_strategies.sql` | Rewritten: added `source`, `confidence`, `decisions`, `final`; fixed ORDER BY; fixed TTL |
| `deploy/migrations/005_create_risk_assessments.sql` | Rewritten: added `source`, `confidence`, `strategies`, `rationale`, `parameters`, `metadata`, `final`; fixed ORDER BY; fixed TTL |
| `deploy/migrations/006_create_executions.sql` | Rewritten: replaced `price` with `filled_quantity`, added `source`, `risk`, `parameters`, `metadata`, `final`; fixed ORDER BY and TTL |
| `deploy/compose/docker-compose.yaml` | Removed `:ro` from ClickHouse users XML mount |

## Files Created

| Path | Type |
|------|------|
| `docs/architecture/core-clickhouse-migrations-and-activation-proof.md` | Architecture doc |
| `docs/architecture/core-schema-application-validation-notes.md` | Architecture doc |

## What Was Proved

1. **Schema evolution works.** `cmd/migrate` applies migrations in order, tracks state, and detects drift.
2. **S144 design is materializable.** All column types, partition keys, order keys, and TTLs from the design doc translate to working ClickHouse DDL.
3. **Idempotency holds.** Re-running `migrate up` is a safe no-op.
4. **Tables accept domain-shaped data.** Sample inserts matching Go struct shapes succeed.
5. **Operational pipeline is unaffected.** No changes to operational services; ClickHouse remains optional.

## What Was NOT Proved

| Property | Deferred To |
|----------|------------|
| Writer populates tables from NATS events | S148 (cmd/writer implementation) |
| Schema survives ClickHouse restart | S148 (operational hardening) |
| TTL expiration works at 90 days | Natural — will occur after 90 days |
| ALTER migration path (schema evolution) | First ALTER migration needed |
| Multi-operator migration safety | If scenario arises |

## Guard Rails Preserved

| Guard Rail | Status |
|-----------|--------|
| No query history opened | Respected — tables created but no read endpoints |
| ClickHouse not coupled to operational baseline | Respected — no operational service changes |
| Schema not inflated beyond S144 | Respected — exact 6 tables, exact columns |
| Migration limitations not masked | Respected — TTL and DateTime64 findings documented |
| Proven vs unproven explicitly documented | Respected — see tables above |

## Preparation for S148

With the core schema materialized and proven, S148 can:

1. **Implement `cmd/writer`** — the standalone ClickHouse writer service. Tables are ready to receive data.
2. **Wire writer into docker-compose** — as an optional service depending on `clickhouse` and `nats`.
3. **Create `smoke-writer.sh`** — validates writer → ClickHouse flow independently, without affecting `smoke-first-slice.sh`.
4. **Test schema persistence** — verify tables survive ClickHouse container restart.
5. **Validate TTL behavior** — insert data with past timestamps, trigger `OPTIMIZE TABLE` to verify TTL cleanup.

The critical gate — "can the schema be created, tracked, and validated via governed infrastructure?" — is now passed.
