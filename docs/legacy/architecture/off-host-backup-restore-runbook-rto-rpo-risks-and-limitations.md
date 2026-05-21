# Off-Host Backup & Restore Runbook

> Stage: S440 | Date: 2026-03-24 | Type: Operational Runbook

## Overview

This runbook covers the automated backup with off-host replication and the recovery procedure for the market-foundry ClickHouse analytical store. It supersedes the recovery sections of the [S435 runbook](clickhouse-recovery-runbook-rto-risks-and-limitations.md) for automated/off-host scenarios while the S435 manual procedures remain valid.

---

## 1. RTO/RPO Summary

| Metric | Value | Assumption |
|--------|-------|------------|
| **RPO (Recovery Point Objective)** | Frequency of scheduled backup (default: 24h if daily cron) | Data written after the last completed backup is lost. RPO = time since last successful backup. |
| **RTO (Recovery Restore Time)** | 1-5 minutes at current scale; up to ~10 minutes at 10M rows | Includes: copy from off-host + ClickHouse RESTORE + healthcheck. Does not include: diagnosing the failure, provisioning new host. |
| **RTO breakdown (current scale)** | rsync recovery: ~5s, RESTORE: <1s, healthcheck: ~5s | Total ~15s for <100 rows. |
| **RTO breakdown (projected 1M rows)** | rsync recovery: ~30-60s, RESTORE: ~5-15s, healthcheck: ~5s | Total ~1-2 minutes. |
| **RTO breakdown (projected 10M rows)** | rsync recovery: ~2-5min, RESTORE: ~30-90s, healthcheck: ~10s | Total ~3-7 minutes. |

### What These Numbers Mean

- **RPO is bounded by backup frequency.** Daily backups = up to 24h of data loss. Increase frequency to reduce RPO.
- **RTO assumes the off-host backup is intact and accessible.** If the off-host target is also lost, recovery is not possible from this system alone.
- **RTO does not include root-cause investigation.** The numbers above are pure technical recovery time after the decision to restore has been made.

---

## 2. Automated Backup

### 2.1 Run Automated Backup

```bash
# Without off-host replication (local backup only):
make ch-backup-auto

# With off-host replication to external drive:
BACKUP_OFFHOST_TARGET=/Volumes/ExternalDrive/backups/mf make ch-backup-auto

# With off-host replication to remote host:
BACKUP_OFFHOST_TARGET=user@backup-host:/backups/mf make ch-backup-auto
```

### 2.2 Schedule with Cron

```bash
# Daily at 02:00 UTC, replicating to external drive
# Add to crontab: crontab -e
0 2 * * * cd /path/to/market-foundry && \
  BACKUP_OFFHOST_TARGET=/Volumes/ExternalDrive/backups/mf \
  ./scripts/clickhouse-scheduled-backup.sh >> /var/log/mf-backup-cron.log 2>&1
```

### 2.3 Schedule with macOS launchd

Create `~/Library/LaunchAgents/com.market-foundry.backup.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.market-foundry.backup</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>-c</string>
        <string>cd /path/to/market-foundry &amp;&amp; BACKUP_OFFHOST_TARGET=/Volumes/ExternalDrive/backups/mf ./scripts/clickhouse-scheduled-backup.sh</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>2</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/mf-backup.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/mf-backup-err.log</string>
</dict>
</plist>
```

Load: `launchctl load ~/Library/LaunchAgents/com.market-foundry.backup.plist`

### 2.4 Verify Backup Ran

```bash
# Check most recent log:
ls -lt backups/logs/ | head -5

# Read the log:
cat backups/logs/backup_auto_*.log | tail -20

# Check exit code in log:
grep "Exit code:" backups/logs/backup_auto_*.log | tail -1
```

---

## 3. Restore Procedures

### 3.1 Restore from Local Backup (Normal Case)

When the local backup directory is intact:

```bash
# List available backups:
make ch-backup-list

# Restore all tables from a specific backup:
make ch-restore BACKUP=auto_20260324_020000

# Restore a single table:
make ch-restore BACKUP=auto_20260324_020000 TABLE=executions
```

### 3.2 Restore from Off-Host Backup (After Local Loss)

When the local backup is lost (disk failure, accidental deletion):

```bash
# Step 1: Copy backup from off-host to local backup directory.
# From external drive:
rsync -a /Volumes/ExternalDrive/backups/mf/<backup_name>/ ./backups/clickhouse/<backup_name>/

# From remote host:
rsync -az user@backup-host:/backups/mf/<backup_name>/ ./backups/clickhouse/<backup_name>/

# Step 2: Restore using the standard procedure.
make ch-restore BACKUP=<backup_name>

# Step 3: Verify.
make migrate-validate
```

### 3.3 Full Recovery (New Host / Fresh Docker)

After a catastrophic failure requiring a new deployment:

```bash
# Step 1: Clone the repository and start infrastructure.
make up
# Wait for ClickHouse to be healthy.

# Step 2: Apply migrations (creates empty tables).
make migrate-up

# Step 3: Copy backup from off-host target.
mkdir -p backups/clickhouse
rsync -az /Volumes/ExternalDrive/backups/mf/<backup_name>/ ./backups/clickhouse/<backup_name>/

# Step 4: Restore data from backup (drops + recreates tables from backup).
make ch-restore BACKUP=<backup_name>

# Step 5: Verify schema and data.
make migrate-validate

# Step 6: Restart services to reconnect to restored data.
make restart
```

---

## 4. Verification Commands

```bash
# Smoke test: automated backup + off-host + restore cycle
make smoke-backup-offhost

# Smoke test: basic backup/restore (S435 proof)
make smoke-backup-restore

# Count rows in all tables:
curl -sS "http://127.0.0.1:8123/" \
  --data-binary "SELECT name, total_rows FROM system.tables WHERE database = 'market_foundry' AND name NOT LIKE '.%' ORDER BY name FORMAT Pretty" \
  -H "X-ClickHouse-User: default" \
  -H "X-ClickHouse-Key: clickhouse"
```

---

## 5. Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Cron/launchd not running or misconfigured | High | Medium | Verify backup logs exist with expected frequency; add a simple check: `find backups/logs -name 'backup_auto_*' -mtime -1 \| grep -q .` |
| Off-host target full or unmounted | High | Medium | Script fails loudly (rsync non-zero exit); operator must monitor log or exit code |
| Off-host target is on same physical disk | Critical | Low | Operator responsibility to ensure target is physically separate; external drive or network path |
| ClickHouse not running during scheduled backup | Medium | Low | Script fails at preflight with explicit error; does not produce partial backup |
| Backup taken during heavy writes | Low | Medium | Per-table atomic; idempotent event IDs mean minor cross-table skew is recoverable |
| SSH key expired for remote replication | Medium | Low | rsync fails with auth error; visible in log |
| Backup format changes after ClickHouse upgrade | Medium | Low | Run `make smoke-backup-restore` after any ClickHouse version change |
| Operator restores from wrong backup | Medium | Low | Backup names include timestamps; always verify backup name before restore |

---

## 6. Monitoring Checklist

Since this system has no built-in alerting (out of scope for S440), the operator should periodically verify:

| Check | How | Frequency |
|-------|-----|-----------|
| Backup ran recently | `ls -lt backups/logs/ \| head -3` | Daily |
| Last backup succeeded | `grep "Exit code:" backups/logs/backup_auto_*.log \| tail -1` | Daily |
| Off-host copy exists | `ls <BACKUP_OFFHOST_TARGET>/` | Weekly |
| Restore still works | `make smoke-backup-restore` | After ClickHouse upgrades |
| Full off-host cycle works | `make smoke-backup-offhost` | Monthly or after infra changes |

---

## 7. Limitations (Honest Assessment)

| Limitation | Why It Exists | When It Matters |
|------------|---------------|-----------------|
| No push-based alerting on backup failure | Building alerting infrastructure is out of scope for S440 | If the operator does not check logs, a failed backup goes unnoticed until recovery is needed |
| RPO depends on backup frequency | Continuous replication (CDC, WAL shipping) is disproportionate for this scale | If backup runs daily, up to 24h of data can be lost |
| No point-in-time recovery (PITR) | ClickHouse native backup is full snapshots, not WAL-based | Cannot restore to an arbitrary timestamp between backups |
| No backup encryption | Would require GPG/age integration | Acceptable: ClickHouse stores analytical/market data, not credentials |
| Remote retention not automated | SSH-based pruning adds complexity and security surface | Old backups on remote targets accumulate until manually pruned |
| No cross-table consistency guarantee | Each table is backed up sequentially, not as an atomic snapshot | Under active writes, tables may reflect slightly different points in time (milliseconds apart) |
| Restore requires ClickHouse to be running | Cannot restore into a stopped ClickHouse instance | Standard operational constraint; `make up` must succeed first |
| Backup size scales with data, not with changes | No incremental backup | At projected 90-day TTL window (<1GB), full backups are practical |
