#!/usr/bin/env bash
# clickhouse-restore.sh — Canonical ClickHouse restore for market-foundry.
#
# Restores tables from a native BACKUP created by clickhouse-backup.sh.
# Uses RESTORE TABLE ... FROM Disk('backups', ...).
#
# Usage:
#   ./scripts/clickhouse-restore.sh mf_20260323_120000              # restore all tables from backup
#   ./scripts/clickhouse-restore.sh mf_20260323_120000 executions   # restore single table
#
# Environment:
#   CLICKHOUSE_HOST      (default: 127.0.0.1)
#   CLICKHOUSE_PORT      (default: 8123)         — HTTP port
#   CLICKHOUSE_USER      (default: default)
#   CLICKHOUSE_PASSWORD  (default: clickhouse)
#   CLICKHOUSE_DATABASE  (default: market_foundry)

set -euo pipefail

CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"

BACKUP_NAME="${1:-}"
TABLE_ARG="${2:-}"

if [[ -z "${BACKUP_NAME}" ]]; then
    echo "Usage: $0 <backup_name> [table_name]" >&2
    echo "" >&2
    echo "List available backups:" >&2
    echo "  ls ./backups/clickhouse/" >&2
    exit 1
fi

ch_query() {
    local query="$1"
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$query" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>&1
}

echo "=== ClickHouse Restore ==="
echo "Target:      ${CH_HOST}:${CH_PORT}/${CH_DB}"
echo "Backup name: ${BACKUP_NAME}"
echo ""

# Verify connectivity.
VERSION=$(ch_query "SELECT version()")
echo "ClickHouse version: ${VERSION}"

# Ensure database exists.
ch_query "CREATE DATABASE IF NOT EXISTS ${CH_DB}" >/dev/null

if [ -n "${TABLE_ARG}" ]; then
    TABLES="${TABLE_ARG}"
else
    # Discover tables from backup directory listing inside the container.
    BACKUP_HOST_DIR="./backups/clickhouse/${BACKUP_NAME}"
    if [ ! -d "${BACKUP_HOST_DIR}" ]; then
        echo "ERROR: Backup directory not found: ${BACKUP_HOST_DIR}" >&2
        echo "Available backups:" >&2
        ls ./backups/clickhouse/ 2>/dev/null || echo "  (none)"
        exit 1
    fi
    TABLES=$(ls "${BACKUP_HOST_DIR}" | grep -v '^\.')
fi

echo "Tables to restore: ${TABLES}"
echo ""

OVERALL_START=$(date +%s)
RESULTS=""

for table in ${TABLES}; do
    [ -z "${table}" ] && continue

    echo "--- ${table} ---"

    # Drop existing table so restore can recreate it cleanly.
    echo "  Dropping existing table (if any)..."
    ch_query "DROP TABLE IF EXISTS ${CH_DB}.${table}" >/dev/null

    START=$(date +%s)
    RESTORE_SQL="RESTORE TABLE ${CH_DB}.${table} FROM Disk('backups', '${BACKUP_NAME}/${table}/')"
    RESULT=$(ch_query "${RESTORE_SQL}")
    END=$(date +%s)
    ELAPSED=$((END - START))

    if echo "${RESULT}" | grep -qi "exception"; then
        echo "  FAILED (${ELAPSED}s): ${RESULT}" >&2
        RESULTS="${RESULTS}${table}: FAIL (${ELAPSED}s)\n"
    else
        # Verify row count after restore.
        ROW_COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
        echo "  OK (${ELAPSED}s, ${ROW_COUNT} rows restored)"
        RESULTS="${RESULTS}${table}: OK (${ELAPSED}s, ${ROW_COUNT} rows)\n"
    fi
done

OVERALL_END=$(date +%s)
OVERALL_ELAPSED=$((OVERALL_END - OVERALL_START))

echo ""
echo "=== Restore Summary ==="
echo "Backup name: ${BACKUP_NAME}"
echo "Total time:  ${OVERALL_ELAPSED}s"
printf "  %b" "${RESULTS}"
echo ""
echo "Post-restore verification:"
echo "  Run: make migrate-validate"
echo "  Run: make smoke-analytical"
echo "Done."
