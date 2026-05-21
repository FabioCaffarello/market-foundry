# Stage S435: ClickHouse Backup/Restore Operational Proof

> Date: 2026-03-23 | Phase: 49 (Mainnet Enablement) | Closes: B-3

## Objective

Prove that the market-foundry ClickHouse analytical store has a canonical, tested, and reproducible backup/restore path. Close Blocker B-3 before any mainnet dry-run authorization.

## Context

After S433 (mainnet adapters, B-1 closed) and S434 (secret manager, B-2 closed), the next explicit blocker was B-3: no ClickHouse backup/restore strategy. The analytical store held the audit trail (executions, candles, signals, decisions, strategies, risk assessments) with 90-day TTL on a persistent Docker volume but no recovery path.

## Strategy Decision

**ClickHouse native `BACKUP TABLE ... TO Disk()` / `RESTORE TABLE ... FROM Disk()`.**

Rationale:
- Zero external dependencies (no `clickhouse-backup` tool, no S3 client)
- Available since ClickHouse 22.x (we run 24.8.8)
- Schema-preserving (DDL, partitions, TTL, order keys included in backup metadata)
- Per-table granularity (selective restore possible)
- Bind-mount to host filesystem for portable backup artifacts

## Deliverables

### Code/Infrastructure

| Artifact | Type | Description |
|----------|------|-------------|
| `deploy/clickhouse/config/backup-disk.xml` | New | ClickHouse storage config for `Disk('backups')` at `/backups/` |
| `deploy/compose/docker-compose.yaml` | Modified | Added backup disk config mount + host bind-mount `backups/clickhouse:/backups` |
| `scripts/clickhouse-backup.sh` | New | Backup all or single table via HTTP API |
| `scripts/clickhouse-restore.sh` | New | Restore from named backup via HTTP API |
| `scripts/smoke-clickhouse-backup-restore.sh` | New | 9-step automated proof (33 checks) |
| `backups/clickhouse/.gitignore` | New | Directory structure in git, data ignored |
| `Makefile` | Modified | Added `ch-backup`, `ch-restore`, `ch-backup-list` targets |

### Documentation

| Document | Description |
|----------|-------------|
| [`clickhouse-backup-restore-proof.md`](../architecture/clickhouse-backup-restore-proof.md) | Strategy, infrastructure changes, proof execution, timing, limitations |
| [`clickhouse-recovery-runbook-rto-risks-and-limitations.md`](../architecture/clickhouse-recovery-runbook-rto-risks-and-limitations.md) | Step-by-step runbook, RTO table, risks, troubleshooting |
| `mainnet-blockers-...register.md` | B-3 status updated from BLOCKER to CLOSED |

## Proof Results

Executed `scripts/smoke-clickhouse-backup-restore.sh` against ClickHouse 24.8.8.17 in Docker:

| Step | Description | Result |
|------|-------------|--------|
| 1 | Connectivity | PASS |
| 2 | Seed proof marker rows | PASS |
| 3 | Record pre-backup state (7 tables, 53 rows) | PASS |
| 4 | Backup all 7 tables | 7/7 PASS (<1s) |
| 5 | DROP all 7 tables | 7/7 confirmed dropped |
| 6 | Restore all 7 tables | 7/7 PASS (<1s) |
| 7 | Post-restore row counts match | 7/7 PASS |
| 8 | Proof marker rows survived | 2/2 PASS |
| 9 | Schema integrity (TTL, partitions) | PASS |

**Total: 33 PASS, 0 FAIL.**

## Timing

| Metric | Value |
|--------|-------|
| Backup (7 tables, 53 rows) | <1s |
| Restore (7 tables, 53 rows) | <1s |
| Estimated RTO at 1M rows | ~15-35s |
| Backup size (53 rows) | 384 KB |

## Limitations (Explicit)

1. **Manual trigger** — no automated schedule (acceptable; `make ch-backup` before risky operations)
2. **Single-node** — no cross-replica coordination (matches current deployment)
3. **No incremental** — full backup each time (acceptable at projected <1GB data volume)
4. **Same-host storage** — backup on same filesystem as data (operator copies to external storage for DR)
5. **Per-table atomicity** — cross-table consistency requires stopping writer (optional, minor cross-table skew is benign)

## Blocker Status

| Blocker | Previous Status | New Status | Evidence |
|---------|----------------|------------|----------|
| B-1 (Mainnet Adapters) | CLOSED (S433) | CLOSED | — |
| B-2 (Credential Management) | CLOSED (S434) | CLOSED | — |
| **B-3 (ClickHouse Backup/Restore)** | **BLOCKER** | **CLOSED** | S435 proof (33/33) |

**All three mainnet blockers are now closed.** The wave is ready for mainnet dry-run authorization in S436.

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Canonical, auditable backup/restore path exists | PASS |
| B-3 is closed or residual objectively reduced | PASS (CLOSED) |
| Analytical store no longer depends on implicit recovery assumption | PASS |
| Wave ready for dry-run mainnet proof in S436 | PASS |
