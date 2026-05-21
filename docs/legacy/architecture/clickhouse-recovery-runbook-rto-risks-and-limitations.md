# ClickHouse Recovery Runbook

> Stage: S435 | Date: 2026-03-23 | Type: Operational Runbook

## Overview

This runbook covers the canonical backup and restore procedure for the market-foundry ClickHouse analytical store. It is the operational companion to the [Backup/Restore Proof](clickhouse-backup-restore-proof.md).

---

## 1. Prerequisites

- Docker running with the market-foundry compose stack
- ClickHouse container healthy (`docker compose ps` shows `healthy`)
- HTTP port 8123 accessible on `127.0.0.1`
- Credentials: user `default`, password from `CLICKHOUSE_PASSWORD` env var (default: `clickhouse`)

---

## 2. Backup Procedure

### Full Backup (All Tables)

```bash
make ch-backup
```

Or with explicit environment:

```bash
CLICKHOUSE_PASSWORD=clickhouse ./scripts/clickhouse-backup.sh
```

### Single Table Backup

```bash
make ch-backup TABLE=executions
```

### Custom Backup Name

```bash
BACKUP_NAME=pre_deploy_v2 make ch-backup
```

### What Happens

1. Script connects to ClickHouse via HTTP API
2. Discovers all MergeTree tables in `market_foundry` database
3. Executes `BACKUP TABLE <db>.<table> TO Disk('backups', '<name>/<table>/')` for each
4. Reports per-table row counts and timing
5. Backup is written to `./backups/clickhouse/<name>/` on the host

### Backup Output Structure

```
backups/clickhouse/<name>/
  <table>/
    .backup
    metadata/market_foundry/<table>.sql    # Full CREATE TABLE DDL
    data/market_foundry/<table>/           # MergeTree data parts
      <partition>_<part>/
        data.bin, primary.cidx, checksums.txt, ...
```

---

## 3. Restore Procedure

### Full Restore

```bash
make ch-restore BACKUP=mf_20260323_120000
```

Or with explicit environment:

```bash
CLICKHOUSE_PASSWORD=clickhouse ./scripts/clickhouse-restore.sh mf_20260323_120000
```

### Single Table Restore

```bash
make ch-restore BACKUP=mf_20260323_120000 TABLE=executions
```

### What Happens

1. Script connects to ClickHouse via HTTP API
2. Ensures target database exists (`CREATE DATABASE IF NOT EXISTS`)
3. For each table: `DROP TABLE IF EXISTS` then `RESTORE TABLE ... FROM Disk('backups', ...)`
4. Reports per-table row counts and timing
5. Schema (DDL, partitions, TTL, order keys) is recreated from backup metadata

### Post-Restore Verification

```bash
# Verify migration checksums match
make migrate-validate

# Verify analytical pipeline (if writer is running)
make smoke-analytical
```

---

## 4. List Available Backups

```bash
make ch-backup-list
```

Or:

```bash
ls -1 backups/clickhouse/
```

---

## 5. Recovery Time Objective (RTO)

| Data Scale | Backup Time | Restore Time | Healthcheck | Total RTO |
|-----------|-------------|--------------|-------------|-----------|
| Current (~50 rows) | <1s | <1s | ~5s | ~5s |
| 100K rows | ~2–5s | ~2–5s | ~5s | ~10–15s |
| 1M rows | ~5–15s | ~5–15s | ~5s | ~15–35s |
| 10M rows | ~30–90s | ~30–90s | ~10s | ~70–190s |

**Note:** These are estimates based on ClickHouse benchmarks for MergeTree tables. Actual RTO depends on disk I/O, compression ratio, and column count. The 90-day TTL caps maximum data volume.

---

## 6. When to Backup

| Trigger | Priority |
|---------|----------|
| Before any deployment that changes ClickHouse schema | Required |
| Before running `migrate up` with destructive migrations | Required |
| Before mainnet dry-run or production activation | Required |
| After significant data ingestion milestone | Recommended |
| Periodic (daily/weekly) during active operation | Recommended |

---

## 7. Consistent Backup (Optional)

For point-in-time consistency across all tables, stop the writer before backup:

```bash
# Stop writer to quiesce writes
docker compose -f deploy/compose/docker-compose.yaml stop writer

# Take backup
make ch-backup

# Restart writer
docker compose -f deploy/compose/docker-compose.yaml start writer
```

This is optional — the writer is append-only with idempotent event IDs, so a backup taken during writes will be consistent per-table but may have minor cross-table timing differences.

---

## 8. Off-Host Backup Copy

The `backups/clickhouse/` directory is on the local host filesystem. For true disaster recovery, copy to external storage:

```bash
# Example: copy to external drive or S3-compatible storage
tar czf "mf_backup_$(date -u +%Y%m%d).tar.gz" backups/clickhouse/<backup_name>/

# Example: rsync to remote host
rsync -avz backups/clickhouse/<backup_name>/ user@backup-host:/backups/market-foundry/
```

---

## 9. Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Backup not taken before destructive operation | High | Make `ch-backup` part of deployment checklist |
| Backup on same disk as data | Medium | Copy to external storage after backup |
| Writer active during backup | Low | Per-table atomic; minor cross-table skew acceptable |
| ClickHouse upgrade changes backup format | Low | Test restore after any ClickHouse version change |
| Backup directory fills disk | Low | 90-day TTL caps data size; prune old backups periodically |

---

## 10. Troubleshooting

### Backup fails with "disk not found"

The `backup-disk.xml` config is not mounted. Verify:
```bash
docker exec market-foundry-clickhouse cat /etc/clickhouse-server/config.d/backup-disk.xml
```

### Backup fails with "permission denied"

The `/backups` directory inside the container needs write access:
```bash
docker exec market-foundry-clickhouse ls -la /backups/
```

### Restore fails with "table already exists"

The restore script drops tables before restoring. If this fails, manually drop:
```bash
curl -sS "http://127.0.0.1:8123/" \
  --data-binary "DROP TABLE IF EXISTS market_foundry.<table>" \
  -H "X-ClickHouse-User: default" \
  -H "X-ClickHouse-Key: clickhouse"
```

### Post-restore migration validation fails

If `make migrate-validate` fails after restore, the `_migrations` table checksums don't match the current migration files. This means the backup was taken with different migration files. Re-run `make migrate-up` to reconcile.
