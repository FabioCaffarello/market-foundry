# Backups

market-foundry persists data in two stores:

| Store | Volume | Critical? |
|---|---|---|
| **NATS JetStream** | `market-foundry-nats-data` | Yes — contains stream messages and KV state |
| **ClickHouse** | `market-foundry-clickhouse-data` | Yes — contains all analytical history |
| ClickHouse logs | `market-foundry-clickhouse-logs` | No — operational diagnostics only |

The backup story focuses on **ClickHouse**. NATS state is reproducible
from the upstream (Binance WebSocket) plus a fresh configctl config;
ClickHouse history is not.

---

## ClickHouse backup

### Available targets

| Target | Purpose | Notes |
|---|---|---|
| `make ch-backup` | On-demand snapshot of all tables (or `TABLE=<name>` for one) | Does not stop services. Writes to local backup directory. |
| `make ch-restore BACKUP=<name>` | Restore from a specific backup. Optional `TABLE=<name>` to restore one table | Requires ClickHouse running. |
| `make ch-backup-list` | List available backups | Reads the backup directory contents. |
| `make ch-backup-auto` | Automated backup + optional off-host replication. Set `BACKUP_OFFHOST_TARGET=...` to enable replication. | Cron-friendly. Logs to its own log directory. Skips off-host replication if `BACKUP_OFFHOST_TARGET` is unset. |
| `make smoke-backup-restore` | Smoke test of the backup/restore round-trip (S435) | Doesn't touch production data. |
| `make smoke-backup-offhost` | Smoke test of automated backup + off-host replication (S440) | Validates the `ch-backup-auto` pipeline including the replication step. |

The underlying scripts live in `scripts/`:
- `clickhouse-backup.sh` — on-demand backup
- `clickhouse-restore.sh` — restore
- `clickhouse-scheduled-backup.sh` — used by `make ch-backup-auto`
- `smoke-clickhouse-backup-restore.sh` — backup/restore smoke
- `smoke-automated-backup-offhost.sh` — automated + off-host smoke

### Typical usage

For an on-demand backup before destructive changes:

```bash
make ch-backup
# ... do something risky ...
# If it goes wrong:
make ch-backup-list                      # see available snapshots
make ch-restore BACKUP=mf_20260322_143000
```

For scheduled backups (recommended for mainnet deployments):

```bash
# crontab -e
0 */6 * * * cd /path/to/market-foundry && make ch-backup-auto
```

To enable off-host replication on schedule, export the target before
the cron job runs:

```bash
0 */6 * * * cd /path/to/market-foundry && \
    BACKUP_OFFHOST_TARGET="user@backup-host:/srv/mf-backups" make ch-backup-auto
```

To validate the backup pipeline (without touching production data):

```bash
make smoke-backup-restore
```

This smoke verifies the backup-restore round-trip works on a sample
of data.

### Storage location

By default, on-host backups are written to `./backups/clickhouse/`
relative to the repository root. The ClickHouse container bind-mounts
this path via `deploy/clickhouse/config/backup-disk.xml`.

Backup names follow a timestamp-based pattern (e.g.,
`mf_20260322_143000`) so they sort chronologically and never collide.

For off-host replication, set `BACKUP_OFFHOST_TARGET` to an rsync
destination (e.g., `user@host:/path` or a local path on a different
filesystem). The scheduled-backup script handles the rsync step
itself; the smoke verifies the round-trip.

For long-term retention beyond the local directory, copy or sync to
external storage (S3, NAS, etc.) via your own scheduling.

### `backups/` directory tracking (shim pattern)

The `backups/` directory exists in the working tree with this layout:

```
backups/
├── clickhouse/
│   ├── .gitignore       (TRACKED, content: `*` then `!.gitignore`)
│   └── .gitkeep         (TRACKED, preserves empty dir)
├── logs/.gitignore      (TRACKED, same shim pattern)
└── sessions/.gitignore  (TRACKED, same shim pattern)
```

The `.gitignore` shims use the pattern `*` followed by `!.gitignore`,
meaning: **ignore all files in this dir except `.gitignore` itself**.

This preserves the empty directory layout in git (so Makefile targets
like `make ch-backup-list`, `make ch-backup-auto`, and the scheduled-
backup script find them when checking out fresh), while keeping backup
contents local-only — they are operator artifacts, not source.

If you see these shims missing from your working tree, restore them
from git (`git checkout -- backups/`) — they're load-bearing for the
backup workflow.

**Do NOT add `backups/` itself to `.gitignore`.** Doing so prevents
git from tracking the shims and breaks the operational pattern. (This
was almost recommended by an external audit in P4.0 pre-work; pause-
and-report caught it before execution. The audit had observed a
transient zip-state where the shims were locally deleted, but did not
cross-reference `Makefile` or this document.)

---

## NATS backup

NATS JetStream state is **not actively backed up**. The reasoning:

1. JetStream messages are derivable: re-running ingest against the
   same Binance time range plus replaying configctl will reproduce
   the stream.
2. KV state is derived from streams via store's durable consumers;
   replaying streams rebuilds KV.
3. The only state that **cannot** be reconstructed from upstream is
   in-flight execution intents that haven't reached terminal state.

For mainnet deployments where in-flight intents matter, consider:

- Periodic snapshot of the `nats_data` volume (via `docker run`
  with the volume mounted, then `tar`).
- Monitoring of in-flight intents and reconciliation against venue
  state.

There is currently no scripted target for NATS snapshots.

---

## Recovery scenarios

### Recovery 1: ClickHouse data loss (corruption, accidental drop)

```bash
make ch-backup-list                       # confirm what's available
make ch-restore BACKUP=<name>             # restore from a specific snapshot
make ps                                   # verify writer reconnects
```

After restore, run `make smoke-analytical` to confirm reads work.

### Recovery 2: NATS state loss (volume corrupted, deleted)

```bash
make down
docker volume rm market-foundry-nats-data
make up                                   # NATS comes up empty
make seed                                 # re-apply configctl configs
make smoke                                # verify operational path
```

This loses operational state but ClickHouse history remains. Reads
of `/{domain}/{type}/latest` will fail until producers re-publish;
reads of `/analytical/{domain}/history` continue working.

### Recovery 3: Full system loss (host wipe)

1. Restore the host environment (Docker, Make, scripts).
2. Clone the repository.
3. Restore the latest ClickHouse snapshot to `./backups/clickhouse/`
   from off-host storage.
4. `make up` and re-seed.
5. Re-publish in-flight execution state from venue (manual
   reconciliation).

There is no fully automated full-system recovery procedure. This is
acceptable for single-operator deployments; high-availability
deployments would require more sophistication.

---

## Validation

After any restore or recovery, validate with:

```bash
make smoke-analytical                     # historical reads work
make smoke                                # operational path works
curl -fsS http://127.0.0.1:8080/readyz   # gateway and family ready
```

If `/readyz` returns 503, see [troubleshooting.md](troubleshooting.md).

---

## Reading further

| If you want | Go to |
|---|---|
| What to back up before destructive recovery | [troubleshooting.md](troubleshooting.md) → "When all else fails" |
| ClickHouse migration handling | [deployment.md](deployment.md) → "Configuration files" |
| Persistent volumes overview | [`../RUNTIME.md`](../RUNTIME.md) |
| What state is reproducible vs critical | This document, top section |
