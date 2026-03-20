# Core Schema Application — Validation Notes

## Environment

- **ClickHouse version:** 24.8.8 (clickhouse/clickhouse-server:24.8.8)
- **Protocol:** Native (port 9000)
- **Database:** `market_foundry` (created by `cmd/migrate` bootstrap)
- **User:** `default` (password from `deploy/envs/local.env`)
- **Go driver:** `clickhouse-go/v2` v2.43.0

## Validation Procedure

### Step 1: Start ClickHouse

```bash
docker compose -f deploy/compose/docker-compose.yaml up -d clickhouse
```

**Finding:** The ClickHouse container initially crash-looped because `deploy/envs/local.env` sets `CLICKHOUSE_PASSWORD=clickhouse`, which triggers the entrypoint to rewrite `/etc/clickhouse-server/users.d/default-user.xml`. This file was mounted as `:ro`, causing a write failure.

**Fix:** Removed `:ro` from the users XML volume mount in `docker-compose.yaml`. The config XML mount (`listen.xml`) remains `:ro` as it is not modified by the entrypoint.

### Step 2: Apply Migrations

```bash
CLICKHOUSE_PASSWORD=clickhouse go run ./cmd/migrate up
```

**Output:**
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

All 7 migrations applied in sequence without error.

### Step 3: Status Verification

```bash
CLICKHOUSE_PASSWORD=clickhouse go run ./cmd/migrate status
```

**Output:** 7 applied, 0 pending. All checksums recorded.

### Step 4: Checksum Validation

```bash
CLICKHOUSE_PASSWORD=clickhouse go run ./cmd/migrate validate
```

**Output:** `all checksums valid`

### Step 5: Idempotency Proof

```bash
CLICKHOUSE_PASSWORD=clickhouse go run ./cmd/migrate up
```

**Output:** `no pending migrations`

Re-running `up` after all migrations are applied is a safe no-op. This confirms the runner correctly skips already-applied migrations.

### Step 6: Schema Verification

All 6 domain tables verified via `SHOW CREATE TABLE`:

- Columns match S144 design exactly (including `source`, `final`, all JSON columns)
- ENGINE = MergeTree for all tables
- PARTITION BY matches design (evidence: `(timeframe, toYYYYMM(open_time))`; pipeline: `toYYYYMM(timestamp)`)
- ORDER BY includes `source` as leading key in all tables
- TTL uses `toDateTime()` cast (ClickHouse 24.8 compatibility)

### Step 7: Insert Proof

Sample rows inserted into all 6 domain tables with representative data matching the Go struct shapes. All inserts succeeded. Test data was truncated after verification.

## Issues Found and Resolved

### Issue 1: ClickHouse Container Crash Loop

**Symptom:** Container restarting with `Read-only file system` error.
**Root cause:** `CLICKHOUSE_PASSWORD` env var triggers entrypoint to rewrite users XML, which was mounted `:ro`.
**Fix:** Removed `:ro` from users XML volume mount.
**Impact:** No impact on security — the file is writable only within the container, and the mount still provides the initial user configuration.

### Issue 2: TTL on DateTime64 Columns

**Symptom:** `code: 450, message: TTL expression result column should have DateTime or Date type, but has DateTime64(3)`
**Root cause:** ClickHouse 24.8.8 does not accept `DateTime64` directly in TTL expressions.
**Fix:** All TTL expressions use `toDateTime(<column>) + INTERVAL 90 DAY`.
**Impact:** The subsecond precision loss in the TTL conversion is irrelevant — TTL cleanup operates at the part level (hours/days granularity), not at the row level.

### Issue 3: S146 Migration Divergence from S144 Design

**Symptom:** 6 migration files created in S146 were missing columns (`source`, `final`, `confidence`, `metadata`, etc.) and had incorrect ORDER BY clauses.
**Root cause:** S146 migrations were drafted without strict verification against the S144 design document and Go struct source of truth.
**Fix:** All 6 migration files rewritten to match S144 DDL exactly before first application. No V-02 violation since migrations had never been applied.
**Impact:** None — the corrected files are the first and only version ever applied.

## What Was Proved

| Property | Evidence |
|----------|---------|
| Migrations apply in order | 000 → 001 → ... → 006 sequence observed |
| Idempotent re-runs are safe | Second `migrate up` returns "no pending migrations" |
| Checksums are stable | `migrate validate` confirms no drift |
| Schema matches Go structs | Column-by-column comparison against S144 design |
| Tables accept domain-shaped data | Sample INSERT for all 6 tables succeeded |
| `_migrations` tracks state correctly | 7 entries with version, name, checksum, applied_at |
| Database auto-bootstraps | `market_foundry` database created on first run |

## What Was NOT Proved

| Property | Why | When |
|----------|-----|------|
| Writer can populate tables | No `cmd/writer` yet | S148 |
| Schema survives restart | Not tested (container restart + re-validate) | S148 |
| TTL actually expires data | 90-day window; no data old enough | After 90 days or manual TTL trigger |
| Multi-migration evolution (ALTER) | Only CREATE migrations so far | When first ALTER migration is needed |
| Concurrent migration safety | Single-operator model; no locking tested | If multi-operator scenarios arise |
| Performance under load | No volume data; empty tables | After writer populates at runtime scale |

## Operational Notes

### Running Migrations

```bash
# From repository root, with ClickHouse running:
make migrate-up         # Sources deploy/envs/local.env automatically
make migrate-status     # Check current state
make migrate-validate   # Verify checksums
```

### Resetting Schema (Development Only)

```bash
# Drop the entire database (loses all data):
docker exec market-foundry-clickhouse clickhouse-client \
  --port 9000 --user default --password clickhouse \
  --query "DROP DATABASE IF EXISTS market_foundry"

# Re-apply all migrations:
make migrate-up
```

### Viewing Table DDL

```bash
docker exec market-foundry-clickhouse clickhouse-client \
  --port 9000 --user default --password clickhouse \
  --database market_foundry \
  --query "SHOW CREATE TABLE <table_name>"
```
