#!/usr/bin/env bash
# smoke-automated-backup-offhost.sh — S440: Proof of automated backup with off-host replication.
#
# Exercises the full automated cycle: backup -> off-host replicate -> verify -> restore -> verify.
# Uses a temp directory as the off-host target to prove the replication path without requiring
# an actual external drive or remote host.
#
# Prerequisites:
#   - ClickHouse running and healthy (make up)
#   - Migrations applied (make migrate-up)
#
# Usage:
#   ./scripts/smoke-automated-backup-offhost.sh

set -euo pipefail

CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"

PASS_COUNT=0
FAIL_COUNT=0

pass() { echo "  PASS: $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { echo "  FAIL: $1" >&2; FAIL_COUNT=$((FAIL_COUNT + 1)); }

ch_query() {
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$1" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>&1
}

# Create a temp directory to simulate off-host target.
OFFHOST_DIR=$(mktemp -d)
trap "rm -rf ${OFFHOST_DIR}" EXIT

BACKUP_NAME="proof_s440_$(date -u +%Y%m%d_%H%M%S)"

echo "============================================"
echo " S440: Automated Backup + Off-Host Proof"
echo "============================================"
echo ""
echo "Host:           ${CH_HOST}:${CH_PORT}"
echo "Database:       ${CH_DB}"
echo "Backup name:    ${BACKUP_NAME}"
echo "Off-host dir:   ${OFFHOST_DIR}"
echo ""

# ── Step 1: Verify connectivity ──────────────────────────────────────────────
echo "Step 1: Verify connectivity"
VERSION=$(ch_query "SELECT version()")
if [ -n "${VERSION}" ] && ! echo "${VERSION}" | grep -qi "error"; then
    pass "ClickHouse reachable, version ${VERSION}"
else
    fail "Cannot connect to ClickHouse"
    exit 1
fi

# ── Step 2: Seed proof data ──────────────────────────────────────────────────
echo ""
echo "Step 2: Seed proof data"

MARKER_ID="proof-s440-$(date -u +%s)"
INSERT_SQL="INSERT INTO ${CH_DB}.executions (event_id, occurred_at, type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, final, timestamp)
VALUES ('${MARKER_ID}', now64(3), 'paper_order', 'proof-s440', 'BTCUSDT', 60, 'buy', 0.001, 0.001, 'Filled', '{}', '[]', '{}', '{\"proof\":\"s440\"}', true, now64(3))"
ch_query "${INSERT_SQL}" >/dev/null

pass "Proof marker inserted (${MARKER_ID})"

# Record pre-backup row counts.
TABLES_RAW=$(ch_query "SELECT name FROM system.tables WHERE database = '${CH_DB}' AND name NOT LIKE '.%' AND engine LIKE '%MergeTree%' ORDER BY name FORMAT TabSeparated")
declare -A PRE_COUNTS
while IFS= read -r table; do
    [ -z "${table}" ] && continue
    COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    PRE_COUNTS["${table}"]="${COUNT}"
    echo "  ${table}: ${COUNT} rows"
done <<< "${TABLES_RAW}"

# ── Step 3: Run automated backup with off-host replication ───────────────────
echo ""
echo "Step 3: Run automated backup (clickhouse-scheduled-backup.sh)"

BACKUP_OUTPUT=$(BACKUP_NAME="${BACKUP_NAME}" \
    BACKUP_OFFHOST_TARGET="${OFFHOST_DIR}" \
    BACKUP_RETAIN_COUNT=3 \
    BACKUP_LOG_DIR="./backups/logs" \
    CLICKHOUSE_HOST="${CH_HOST}" \
    CLICKHOUSE_PORT="${CH_PORT}" \
    CLICKHOUSE_USER="${CH_USER}" \
    CLICKHOUSE_PASSWORD="${CH_PASS}" \
    CLICKHOUSE_DATABASE="${CH_DB}" \
    ./scripts/clickhouse-scheduled-backup.sh 2>&1) || true

echo "${BACKUP_OUTPUT}" | tail -20

if echo "${BACKUP_OUTPUT}" | grep -q "Exit code:   0"; then
    pass "Automated backup completed successfully"
else
    fail "Automated backup reported non-zero exit"
fi

# ── Step 4: Verify local backup exists ───────────────────────────────────────
echo ""
echo "Step 4: Verify local backup"

LOCAL_BACKUP="./backups/clickhouse/${BACKUP_NAME}"
if [ -d "${LOCAL_BACKUP}" ]; then
    LOCAL_FILE_COUNT=$(find "${LOCAL_BACKUP}" -type f | wc -l | tr -d ' ')
    pass "Local backup exists: ${LOCAL_FILE_COUNT} files"
else
    fail "Local backup directory not found: ${LOCAL_BACKUP}"
fi

# ── Step 5: Verify off-host replication ──────────────────────────────────────
echo ""
echo "Step 5: Verify off-host replication"

OFFHOST_BACKUP="${OFFHOST_DIR}/${BACKUP_NAME}"
if [ -d "${OFFHOST_BACKUP}" ]; then
    OFFHOST_FILE_COUNT=$(find "${OFFHOST_BACKUP}" -type f | wc -l | tr -d ' ')
    pass "Off-host backup exists: ${OFFHOST_FILE_COUNT} files"

    if [ "${LOCAL_FILE_COUNT:-0}" = "${OFFHOST_FILE_COUNT}" ]; then
        pass "File counts match: local=${LOCAL_FILE_COUNT}, off-host=${OFFHOST_FILE_COUNT}"
    else
        fail "File count mismatch: local=${LOCAL_FILE_COUNT:-?}, off-host=${OFFHOST_FILE_COUNT}"
    fi
else
    fail "Off-host backup directory not found: ${OFFHOST_BACKUP}"
fi

# ── Step 6: Verify backup log was created ────────────────────────────────────
echo ""
echo "Step 6: Verify backup log"

LOG_FILE="./backups/logs/backup_${BACKUP_NAME}.log"
if [ -f "${LOG_FILE}" ]; then
    LOG_LINES=$(wc -l < "${LOG_FILE}" | tr -d ' ')
    pass "Backup log exists: ${LOG_LINES} lines"
else
    fail "Backup log not found: ${LOG_FILE}"
fi

# ── Step 7: Destroy and restore from off-host copy ──────────────────────────
echo ""
echo "Step 7: Destroy tables (simulate data loss)"

TABLES=""
while IFS= read -r table; do
    [ -z "${table}" ] && continue
    TABLES="${TABLES} ${table}"
    ch_query "DROP TABLE IF EXISTS ${CH_DB}.${table}" >/dev/null
done <<< "${TABLES_RAW}"

for table in ${TABLES}; do
    CHECK=$(ch_query "EXISTS TABLE ${CH_DB}.${table} FORMAT TabSeparated")
    if [ "${CHECK}" = "0" ]; then
        pass "Table ${table} confirmed dropped"
    else
        fail "Table ${table} still exists after drop"
    fi
done

# ── Step 8: Copy off-host backup back to local (simulates recovery) ─────────
echo ""
echo "Step 8: Recover from off-host copy"

# Remove local backup to prove we can recover from off-host only.
rm -rf "${LOCAL_BACKUP}"
if [ ! -d "${LOCAL_BACKUP}" ]; then
    pass "Local backup removed (simulating local loss)"
else
    fail "Could not remove local backup"
fi

# Copy from off-host target back to local backup dir.
rsync -a "${OFFHOST_BACKUP}/" "${LOCAL_BACKUP}/"
if [ -d "${LOCAL_BACKUP}" ]; then
    RECOVERED_FILES=$(find "${LOCAL_BACKUP}" -type f | wc -l | tr -d ' ')
    pass "Recovered from off-host: ${RECOVERED_FILES} files"
else
    fail "Recovery from off-host failed"
fi

# ── Step 9: Execute restore ──────────────────────────────────────────────────
echo ""
echo "Step 9: Execute restore from recovered backup"
RESTORE_START=$(date +%s)

for table in ${TABLES}; do
    RESULT=$(ch_query "RESTORE TABLE ${CH_DB}.${table} FROM Disk('backups', '${BACKUP_NAME}/${table}/')")
    if echo "${RESULT}" | grep -qi "exception"; then
        fail "Restore ${table}: ${RESULT}"
    else
        pass "Restore ${table}"
    fi
done

RESTORE_END=$(date +%s)
RESTORE_ELAPSED=$((RESTORE_END - RESTORE_START))
echo "  Restore duration: ${RESTORE_ELAPSED}s"

# ── Step 10: Verify post-restore data integrity ─────────────────────────────
echo ""
echo "Step 10: Verify post-restore data"

for table in ${TABLES}; do
    POST_COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    PRE_COUNT="${PRE_COUNTS[${table}]:-?}"
    if [ "${POST_COUNT}" = "${PRE_COUNT}" ]; then
        pass "${table}: ${POST_COUNT} rows (matches pre-backup)"
    else
        fail "${table}: expected ${PRE_COUNT} rows, got ${POST_COUNT}"
    fi
done

# ── Step 11: Verify proof marker survived ────────────────────────────────────
echo ""
echo "Step 11: Verify proof marker"

MARKER_CHECK=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE event_id = '${MARKER_ID}' FORMAT TabSeparated")
if [ "${MARKER_CHECK}" = "1" ]; then
    pass "Proof marker found in restored data"
else
    fail "Proof marker NOT found (got ${MARKER_CHECK})"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo " Summary"
echo "============================================"
echo "  Restore duration: ${RESTORE_ELAPSED}s"
echo "  PASS: ${PASS_COUNT}"
echo "  FAIL: ${FAIL_COUNT}"
echo ""

if [ "${FAIL_COUNT}" -eq 0 ]; then
    echo "RESULT: ALL CHECKS PASSED — S440 evidence satisfied."
    echo ""
    echo "Proven capabilities:"
    echo "  1. Automated backup (no manual intervention)"
    echo "  2. Off-host replication (verified file-level parity)"
    echo "  3. Recovery from off-host copy (local backup destroyed, restored from off-host)"
    echo "  4. Data integrity after full cycle (row counts + marker row)"
    echo "  5. Backup logging and auditability"
    exit 0
else
    echo "RESULT: ${FAIL_COUNT} CHECK(S) FAILED — review output above."
    exit 1
fi
