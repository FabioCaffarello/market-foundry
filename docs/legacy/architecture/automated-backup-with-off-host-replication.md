# Automated Backup with Off-Host Replication

> Stage: S440 | Date: 2026-03-24 | Type: Architecture Decision + Implementation

## Purpose

This document describes the automated backup strategy with off-host replication for the market-foundry ClickHouse analytical store. It evolves the manual backup proof from S435 into an automated, auditable pipeline that eliminates the same-host single-point-of-failure identified as a blocker for live trading authorization.

---

## 1. Problem Statement

S435 delivered a proven manual backup/restore path using ClickHouse native `BACKUP TABLE`/`RESTORE TABLE`. However, the S435 residuals explicitly identified two fragilities incompatible with live authorization:

| Fragility | Risk for Live Authorization |
|-----------|----------------------------|
| Manual trigger only | Operator must remember to run backup; no guarantee of recent backup existing when needed |
| Same-host storage only | Backups at `./backups/clickhouse/` live on the same filesystem as the data; a host-level failure (disk, OS, Docker volume corruption) loses both data and backups simultaneously |

Both must be addressed before any serious live trading authorization gate.

---

## 2. Design Decisions

### 2.1 Automation Model

| Decision | Rationale |
|----------|-----------|
| Single orchestration script (`clickhouse-scheduled-backup.sh`) | One entry point for cron, launchd, systemd timers, or manual invocation. Avoids fragmented automation. |
| 4-phase pipeline: preflight -> backup -> replicate -> prune | Each phase is independently observable and fail-safe. Replication is skipped (not failed) if target is not configured. |
| Fail-closed on backup errors | If any table backup fails, replication is skipped entirely. No propagation of corrupt state to off-host. |
| Log every run to `./backups/logs/` | Every automated run produces an auditable log file with timestamps, table counts, replication status, and exit code. |

### 2.2 Off-Host Replication

| Decision | Rationale |
|----------|-----------|
| `rsync` as transport | Available on all Unix systems, handles local paths and remote SSH targets identically, supports incremental transfer. No new dependencies. |
| `BACKUP_OFFHOST_TARGET` env var | Single configuration point. Supports local paths (external drive, NAS mount) and remote SSH targets (`user@host:/path`). Empty = no replication (graceful skip). |
| File-count verification after replication | Post-rsync check compares local and off-host file counts. Detects silent rsync failures. |
| Per-backup directory structure preserved | Off-host target mirrors the exact `<backup_name>/<table>/` structure. Allows direct restore without path translation. |

### 2.3 Retention

| Decision | Rationale |
|----------|-----------|
| `BACKUP_RETAIN_COUNT=7` default | 7 backups covers a week of daily backups. Configurable for different frequencies. |
| Automatic pruning of local and local off-host targets | Prevents unbounded disk growth. Remote SSH targets are not pruned (noted as operational responsibility). |
| Log retention: 30 files | Sufficient audit trail without disk bloat. |

---

## 3. Architecture

```
 Operator / Cron / launchd
        |
        v
 clickhouse-scheduled-backup.sh
        |
        +--- Phase 1: Preflight
        |      Verify ClickHouse connectivity
        |      Discover MergeTree tables
        |
        +--- Phase 2: Backup
        |      BACKUP TABLE ... TO Disk('backups', '<name>/<table>/')
        |      Per-table, fail-closed on any error
        |
        +--- Phase 3: Off-Host Replication
        |      rsync to BACKUP_OFFHOST_TARGET (if set)
        |      Verify file-count parity
        |
        +--- Phase 4: Retention
        |      Prune old backups beyond BACKUP_RETAIN_COUNT
        |      Prune old log files beyond 30
        |
        +--- Summary + Log
               ./backups/logs/backup_<name>.log
```

### Data Flow

```
ClickHouse data (Docker volume)
    |
    | BACKUP TABLE ... TO Disk('backups')
    v
./backups/clickhouse/<name>/     (host bind-mount, same host)
    |
    | rsync
    v
BACKUP_OFFHOST_TARGET/<name>/    (external drive / NAS / remote host)
```

### Recovery Flow

```
BACKUP_OFFHOST_TARGET/<name>/    (off-host copy, survives host failure)
    |
    | rsync back to local (or direct mount)
    v
./backups/clickhouse/<name>/     (local, accessible by ClickHouse container)
    |
    | RESTORE TABLE ... FROM Disk('backups')
    v
ClickHouse data (Docker volume)
```

---

## 4. Artifacts

| Artifact | Purpose |
|----------|---------|
| `scripts/clickhouse-scheduled-backup.sh` | Automated backup orchestrator. Single entry point for all automated backup runs. |
| `scripts/smoke-automated-backup-offhost.sh` | S440 proof script. Full cycle: seed -> backup -> replicate -> destroy -> recover from off-host -> restore -> verify. |
| `backups/logs/.gitignore` | Keeps log directory in git, ignores log files. |
| `Makefile` targets `ch-backup-auto`, `smoke-backup-offhost` | Ergonomic entry points. |

---

## 5. Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_OFFHOST_TARGET` | (empty) | Off-host replication target. Local path or `user@host:/path`. Empty = skip replication. |
| `BACKUP_RETAIN_COUNT` | 7 | Number of backups to keep locally (and at local off-host targets). |
| `BACKUP_LOG_DIR` | `./backups/logs` | Directory for per-run log files. |
| `BACKUP_NAME` | `auto_<timestamp>` | Override backup name. Normally auto-generated. |
| `CLICKHOUSE_HOST` | `127.0.0.1` | ClickHouse HTTP host. |
| `CLICKHOUSE_PORT` | `8123` | ClickHouse HTTP port. |
| `CLICKHOUSE_USER` | `default` | ClickHouse user. |
| `CLICKHOUSE_PASSWORD` | `clickhouse` | ClickHouse password. |
| `CLICKHOUSE_DATABASE` | `market_foundry` | Target database. |

---

## 6. Typical Deployment Patterns

### 6.1 Daily Cron (macOS launchd or Linux cron)

```bash
# crontab entry: daily at 02:00 UTC
0 2 * * * cd /path/to/market-foundry && BACKUP_OFFHOST_TARGET=/Volumes/ExternalDrive/backups/mf/ ./scripts/clickhouse-scheduled-backup.sh >> /var/log/mf-backup.log 2>&1
```

### 6.2 Before Deployment (Manual)

```bash
make ch-backup-auto
# or with off-host:
BACKUP_OFFHOST_TARGET=/Volumes/Backup/mf make ch-backup-auto
```

### 6.3 Remote Replication via SSH

```bash
BACKUP_OFFHOST_TARGET=backupuser@nas.local:/volume1/market-foundry/backups make ch-backup-auto
```

---

## 7. Relationship to S435

S440 builds on S435 without replacing it:

| Capability | S435 | S440 |
|-----------|------|------|
| Manual backup | `make ch-backup` | Unchanged, still works |
| Manual restore | `make ch-restore BACKUP=<name>` | Unchanged, still works |
| Automated backup | Not available | `make ch-backup-auto` |
| Off-host replication | Not available | `BACKUP_OFFHOST_TARGET=... make ch-backup-auto` |
| Backup logging | Not available | `./backups/logs/backup_<name>.log` |
| Retention pruning | Not available | Automatic, configurable |
| Smoke proof: backup/restore | `make smoke-backup-restore` | Unchanged |
| Smoke proof: automated + off-host | Not available | `make smoke-backup-offhost` |

---

## 8. Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| No cloud storage integration (S3/GCS) | Off-host replication limited to rsync-reachable targets | Sufficient for current topology; S3 is a future enhancement if needed |
| No encryption at rest for backups | Backup files are plaintext on disk | Acceptable for analytical data; credentials are NOT stored in ClickHouse |
| No backup integrity checksums beyond file-count | A bit-flip in a backup file would not be detected | ClickHouse native backup includes internal checksums per data part |
| Remote retention not automated | Old backups on SSH targets must be pruned manually | Documented in runbook as operator responsibility |
| Per-table atomicity, not database-wide snapshot | Minor cross-table timing skew possible under active writes | Acceptable for append-only analytical data with idempotent event IDs |
| Single-node ClickHouse only | No replica coordination | Matches current deployment topology |
| rsync requires SSH key setup for remote targets | Operator must configure SSH access | Standard operational setup, documented in runbook |
