# Stage S440: Automated Backup with Off-Host Replication

> Date: 2026-03-24 | Phase: 50 (Live Trading Authorization) | Predecessor: S439

## Objective

Eliminate the manual/same-host backup fragility identified as a condition for live trading authorization. Deliver automated, auditable ClickHouse backup with off-host replication and a proven recovery path.

## Context

S435 delivered the first backup/restore proof for the ClickHouse analytical store, closing Blocker B-3 in the Mainnet Enablement wave. However, S435 explicitly documented two residuals incompatible with serious live authorization:

1. **Manual trigger only** -- no automated schedule, operator must remember to run backup.
2. **Same-host storage only** -- backups on the same filesystem as data, no resilience to host-level failure.

The S437 evidence gate listed these as conditions for the Live Trading Authorization Ceremony. S440 closes both.

## Delivery

### Capabilities Delivered

| ID | Capability | Status | Evidence |
|----|-----------|--------|----------|
| C-1 | Automated backup orchestration (single entry point, 4-phase pipeline) | FULL | `scripts/clickhouse-scheduled-backup.sh` |
| C-2 | Off-host replication via rsync (local path or remote SSH) | FULL | Phase 3 of orchestrator, configurable via `BACKUP_OFFHOST_TARGET` |
| C-3 | Post-replication verification (file-count parity check) | FULL | Built into Phase 3 |
| C-4 | Fail-closed on backup errors (no replication of bad state) | FULL | Phase 2 exits on any table backup failure |
| C-5 | Automatic retention pruning (local + local off-host) | FULL | Phase 4, configurable `BACKUP_RETAIN_COUNT` |
| C-6 | Per-run audit logging | FULL | `./backups/logs/backup_<name>.log` |
| C-7 | Recovery from off-host copy (proven: destroy local, restore from off-host) | FULL | `smoke-automated-backup-offhost.sh` Steps 7-9 |
| C-8 | Data integrity after full off-host recovery cycle | FULL | Row counts + marker row verification in smoke proof |

**8/8 capabilities delivered at FULL rating.**

### Artifacts

| Artifact | Type | Purpose |
|----------|------|---------|
| `scripts/clickhouse-scheduled-backup.sh` | Script | Automated backup orchestrator: preflight, backup, replicate, prune |
| `scripts/smoke-automated-backup-offhost.sh` | Script | S440 proof: full cycle including off-host recovery |
| `backups/logs/.gitignore` | Config | Keeps log directory in git |
| `Makefile` | Modified | Added `ch-backup-auto`, `smoke-backup-offhost` targets |
| `docs/architecture/automated-backup-with-off-host-replication.md` | Architecture | Strategy, design decisions, configuration reference |
| `docs/architecture/off-host-backup-restore-runbook-rto-rpo-risks-and-limitations.md` | Runbook | Operational procedures, RTO/RPO, risks, monitoring checklist |

### RTO/RPO

| Metric | Value | Basis |
|--------|-------|-------|
| RPO | Configurable (default: 24h with daily cron) | Bounded by backup frequency |
| RTO (current scale) | ~15s | rsync + RESTORE + healthcheck |
| RTO (projected 1M rows) | ~1-2 minutes | Extrapolated from ClickHouse benchmarks |
| RTO (projected 10M rows) | ~3-7 minutes | Extrapolated from ClickHouse benchmarks |

## What Changed from S435

| Dimension | S435 (Before) | S440 (After) |
|-----------|---------------|--------------|
| Trigger | Manual only (`make ch-backup`) | Automated (`make ch-backup-auto`, cron-ready) |
| Storage | Same host (`./backups/clickhouse/`) | Same host + off-host replication |
| Recovery from host failure | Not possible (backup lost with host) | Possible (restore from off-host copy) |
| Auditability | None | Per-run log files with timestamps and exit codes |
| Retention | Manual cleanup | Automatic pruning (configurable) |
| Proof scope | Backup + restore on same host | Full cycle: backup, replicate, destroy local, recover from off-host, restore, verify |

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Manual/same-host backup risk materially reduced | PASS -- automated backup with off-host replication eliminates single-point-of-failure |
| Auditable automated backup path exists | PASS -- `ch-backup-auto` with per-run log files |
| Auditable restore path from off-host exists | PASS -- proven in smoke proof (Steps 7-9) |
| RTO/RPO documented with honest assumptions | PASS -- runbook Section 1 |
| Limitations documented without vagueness | PASS -- architecture doc Section 8, runbook Section 7 |
| Stage closes a live authorization condition | PASS -- eliminates "no automated schedule" and "same-host only" residuals from S435/S437 |
| Ready for S441 soak/authenticated proof | PASS -- automated backup can run during soak; off-host replication provides safety net |

## Regressions

None. All existing backup/restore functionality (`make ch-backup`, `make ch-restore`, `make smoke-backup-restore`) is unchanged and continues to work.

## Residual Gaps

| Gap | Severity | Impact on Live Authorization |
|-----|----------|------------------------------|
| No push alerting on backup failure | Low | Operator must check logs; acceptable for single-operator deployment |
| No S3/GCS integration | Low | rsync to local/SSH targets sufficient for current topology |
| No point-in-time recovery | Low | Full snapshots at backup frequency; acceptable for analytical data |
| No backup encryption at rest | Low | ClickHouse stores market data, not credentials |
| Remote retention not automated | Low | Operator responsibility, documented in runbook |

**No medium+ severity gaps. All residuals are documented and appropriate for current deployment scale.**

## Next

S441 can now run authenticated soak tests with automated backup as a safety net. The backup system is ready for continuous operation during extended live-system validation.
