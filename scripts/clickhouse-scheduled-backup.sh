#!/usr/bin/env bash
# clickhouse-scheduled-backup.sh — Automated ClickHouse backup with off-host replication.
#
# This script is the canonical automated backup entry point for market-foundry.
# It orchestrates: backup -> replicate off-host -> prune old backups -> log summary.
#
# Designed to run unattended (cron, launchd, systemd timer, or manual).
#
# Usage:
#   ./scripts/clickhouse-scheduled-backup.sh                # full automated cycle
#   BACKUP_OFFHOST_TARGET="" ./scripts/clickhouse-scheduled-backup.sh   # skip replication
#
# Environment:
#   CLICKHOUSE_HOST       (default: 127.0.0.1)
#   CLICKHOUSE_PORT       (default: 8123)
#   CLICKHOUSE_USER       (default: default)
#   CLICKHOUSE_PASSWORD   (default: clickhouse)
#   CLICKHOUSE_DATABASE   (default: market_foundry)
#   BACKUP_NAME           (default: auto_<timestamp>)
#   BACKUP_OFFHOST_TARGET (default: unset — replication skipped if empty)
#                          Examples:
#                            /Volumes/ExternalDrive/backups/market-foundry/
#                            user@remote:/backups/market-foundry/
#   BACKUP_RETAIN_COUNT   (default: 7 — number of local backups to keep)
#   BACKUP_LOG_DIR        (default: ./backups/logs)

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────

CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"
BACKUP_NAME="${BACKUP_NAME:-auto_$(date -u +%Y%m%d_%H%M%S)}"
OFFHOST_TARGET="${BACKUP_OFFHOST_TARGET:-}"
RETAIN_COUNT="${BACKUP_RETAIN_COUNT:-7}"
LOG_DIR="${BACKUP_LOG_DIR:-./backups/logs}"
LOCAL_BACKUP_DIR="./backups/clickhouse"

TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LOG_FILE="${LOG_DIR}/backup_${BACKUP_NAME}.log"
EXIT_CODE=0

# ── Helpers ──────────────────────────────────────────────────────────────────

mkdir -p "${LOG_DIR}" "${LOCAL_BACKUP_DIR}"

log() {
    local msg="[$(date -u +%H:%M:%S)] $1"
    echo "${msg}"
    echo "${msg}" >> "${LOG_FILE}"
}

ch_query() {
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$1" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>&1
}

# ── Phase 1: Preflight ──────────────────────────────────────────────────────

log "=== Automated Backup: ${BACKUP_NAME} ==="
log "Timestamp: ${TIMESTAMP}"
log "Target: ${CH_HOST}:${CH_PORT}/${CH_DB}"
log "Off-host target: ${OFFHOST_TARGET:-<none — replication skipped>}"
log "Retention: keep last ${RETAIN_COUNT} backups"
log ""

VERSION=$(ch_query "SELECT version()" 2>/dev/null || echo "")
if [ -z "${VERSION}" ] || echo "${VERSION}" | grep -qi "error\|exception"; then
    log "FATAL: Cannot connect to ClickHouse at ${CH_HOST}:${CH_PORT}"
    exit 1
fi
log "ClickHouse version: ${VERSION}"

TABLES=$(ch_query "SELECT name FROM system.tables WHERE database = '${CH_DB}' AND name NOT LIKE '.%' AND engine LIKE '%MergeTree%' ORDER BY name FORMAT TabSeparated")
if [ -z "${TABLES}" ]; then
    log "FATAL: No MergeTree tables found in database ${CH_DB}"
    exit 1
fi

TABLE_COUNT=$(echo "${TABLES}" | wc -l | tr -d ' ')
log "Tables discovered: ${TABLE_COUNT}"

# ── Phase 2: Execute Backup ─────────────────────────────────────────────────

log ""
log "--- Phase 2: Backup ---"
BACKUP_START=$(date +%s)
BACKUP_OK=0
BACKUP_FAIL=0

while IFS= read -r table; do
    [ -z "${table}" ] && continue

    ROW_COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    RESULT=$(ch_query "BACKUP TABLE ${CH_DB}.${table} TO Disk('backups', '${BACKUP_NAME}/${table}/')")

    if echo "${RESULT}" | grep -qi "exception"; then
        log "  FAIL: ${table} — ${RESULT}"
        BACKUP_FAIL=$((BACKUP_FAIL + 1))
        EXIT_CODE=1
    else
        log "  OK:   ${table} (${ROW_COUNT} rows)"
        BACKUP_OK=$((BACKUP_OK + 1))
    fi
done <<< "${TABLES}"

BACKUP_END=$(date +%s)
BACKUP_ELAPSED=$((BACKUP_END - BACKUP_START))
log "Backup: ${BACKUP_OK} OK, ${BACKUP_FAIL} FAIL in ${BACKUP_ELAPSED}s"

if [ "${BACKUP_FAIL}" -gt 0 ]; then
    log "WARNING: Backup had failures — skipping replication to avoid propagating bad state"
    exit 1
fi

# Compute backup size.
BACKUP_SIZE=$(du -sh "${LOCAL_BACKUP_DIR}/${BACKUP_NAME}" 2>/dev/null | cut -f1 || echo "unknown")
log "Backup size: ${BACKUP_SIZE}"

# ── Phase 3: Off-Host Replication ────────────────────────────────────────────

log ""
log "--- Phase 3: Off-Host Replication ---"

if [ -z "${OFFHOST_TARGET}" ]; then
    log "SKIP: BACKUP_OFFHOST_TARGET not set — no off-host replication"
else
    REPLICATE_START=$(date +%s)

    # Determine replication method: rsync for remote or local path.
    if echo "${OFFHOST_TARGET}" | grep -q ':'; then
        # Remote target (user@host:/path or host:/path).
        log "Replicating to remote: ${OFFHOST_TARGET}"
        rsync -az --delete \
            "${LOCAL_BACKUP_DIR}/${BACKUP_NAME}/" \
            "${OFFHOST_TARGET}/${BACKUP_NAME}/" \
            2>&1 | while read -r line; do log "  rsync: ${line}"; done
    else
        # Local path (external drive, NAS mount, etc.).
        log "Replicating to local path: ${OFFHOST_TARGET}"
        mkdir -p "${OFFHOST_TARGET}/${BACKUP_NAME}"
        rsync -a --delete \
            "${LOCAL_BACKUP_DIR}/${BACKUP_NAME}/" \
            "${OFFHOST_TARGET}/${BACKUP_NAME}/" \
            2>&1 | while read -r line; do log "  rsync: ${line}"; done
    fi

    RSYNC_EXIT=$?
    REPLICATE_END=$(date +%s)
    REPLICATE_ELAPSED=$((REPLICATE_END - REPLICATE_START))

    if [ "${RSYNC_EXIT}" -eq 0 ]; then
        # Verify replication: compare file counts.
        LOCAL_FILES=$(find "${LOCAL_BACKUP_DIR}/${BACKUP_NAME}" -type f | wc -l | tr -d ' ')
        if echo "${OFFHOST_TARGET}" | grep -q ':'; then
            REMOTE_FILES=$(rsync -az --dry-run --stats "${LOCAL_BACKUP_DIR}/${BACKUP_NAME}/" "${OFFHOST_TARGET}/${BACKUP_NAME}/" 2>/dev/null | grep "Number of regular files transferred" | grep -o '[0-9]*' || echo "0")
            # dry-run transfer count of 0 means everything is already synced
            if [ "${REMOTE_FILES}" = "0" ]; then
                log "Replication OK: all ${LOCAL_FILES} files synced (${REPLICATE_ELAPSED}s)"
            else
                log "WARNING: ${REMOTE_FILES} files still need transfer after replication"
                EXIT_CODE=1
            fi
        else
            REMOTE_FILES=$(find "${OFFHOST_TARGET}/${BACKUP_NAME}" -type f | wc -l | tr -d ' ')
            if [ "${LOCAL_FILES}" = "${REMOTE_FILES}" ]; then
                log "Replication OK: ${LOCAL_FILES} files verified at off-host target (${REPLICATE_ELAPSED}s)"
            else
                log "WARNING: File count mismatch — local=${LOCAL_FILES}, off-host=${REMOTE_FILES}"
                EXIT_CODE=1
            fi
        fi
    else
        log "FAIL: rsync exited with code ${RSYNC_EXIT}"
        EXIT_CODE=1
    fi
fi

# ── Phase 4: Retention / Prune Old Backups ───────────────────────────────────

log ""
log "--- Phase 4: Retention ---"

prune_dir() {
    local dir="$1"
    local label="$2"

    if [ ! -d "${dir}" ]; then
        log "  ${label}: directory not found, skipping"
        return
    fi

    # List backups sorted by name (timestamp-based, oldest first), exclude hidden files.
    local all_backups
    all_backups=$(ls -1 "${dir}" | grep -E '^(auto_|mf_|proof_)' | sort)
    local count
    count=$(echo "${all_backups}" | grep -c . || echo "0")

    if [ "${count}" -le "${RETAIN_COUNT}" ]; then
        log "  ${label}: ${count} backups, within retention limit (${RETAIN_COUNT})"
        return
    fi

    local to_prune=$((count - RETAIN_COUNT))
    log "  ${label}: ${count} backups, pruning ${to_prune} oldest"

    echo "${all_backups}" | head -n "${to_prune}" | while read -r old; do
        [ -z "${old}" ] && continue
        log "    Removing: ${old}"
        rm -rf "${dir:?}/${old}"
    done
}

prune_dir "${LOCAL_BACKUP_DIR}" "local"

if [ -n "${OFFHOST_TARGET}" ] && ! echo "${OFFHOST_TARGET}" | grep -q ':'; then
    prune_dir "${OFFHOST_TARGET}" "off-host"
fi

# Remote retention for rsync targets requires SSH — log a reminder instead.
if [ -n "${OFFHOST_TARGET}" ] && echo "${OFFHOST_TARGET}" | grep -q ':'; then
    log "  off-host (remote): retention pruning must be managed on the remote host"
fi

# Also prune old log files (keep last 30).
LOG_COUNT=$(ls -1 "${LOG_DIR}"/backup_*.log 2>/dev/null | wc -l | tr -d ' ')
if [ "${LOG_COUNT}" -gt 30 ]; then
    TO_PRUNE=$((LOG_COUNT - 30))
    ls -1t "${LOG_DIR}"/backup_*.log | tail -n "${TO_PRUNE}" | xargs rm -f
    log "  logs: pruned ${TO_PRUNE} old log files"
fi

# ── Summary ──────────────────────────────────────────────────────────────────

log ""
log "=== Summary ==="
log "Backup:      ${BACKUP_NAME} (${BACKUP_OK}/${TABLE_COUNT} tables, ${BACKUP_ELAPSED}s, ${BACKUP_SIZE})"
if [ -n "${OFFHOST_TARGET}" ]; then
    log "Replication: ${OFFHOST_TARGET}"
fi
log "Exit code:   ${EXIT_CODE}"

exit "${EXIT_CODE}"
