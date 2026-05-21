#!/usr/bin/env bash
# clickhouse-backup.sh — Canonical ClickHouse backup for market-foundry.
#
# Uses ClickHouse native BACKUP TABLE ... TO Disk('backups', ...) which writes
# to the server-side path configured in deploy/clickhouse/config/backup-disk.xml.
# The compose stack bind-mounts that path to ./backups/clickhouse on the host.
#
# Usage:
#   ./scripts/clickhouse-backup.sh              # backup all tables
#   ./scripts/clickhouse-backup.sh executions   # backup single table
#
# Environment:
#   CLICKHOUSE_HOST      (default: 127.0.0.1)
#   CLICKHOUSE_PORT      (default: 8123)         — HTTP port
#   CLICKHOUSE_USER      (default: default)
#   CLICKHOUSE_PASSWORD  (default: clickhouse)
#   CLICKHOUSE_DATABASE  (default: market_foundry)
#   BACKUP_NAME          (default: mf_<timestamp>)

set -euo pipefail

CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"
BACKUP_NAME="${BACKUP_NAME:-mf_$(date -u +%Y%m%d_%H%M%S)}"

TABLES_ARG="${1:-}"

ch_query() {
    local query="$1"
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$query" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>&1
}

echo "=== ClickHouse Backup ==="
echo "Target: ${CH_HOST}:${CH_PORT}/${CH_DB}"
echo "Backup name: ${BACKUP_NAME}"
echo ""

# Verify connectivity.
VERSION=$(ch_query "SELECT version()")
echo "ClickHouse version: ${VERSION}"

if [ -n "${TABLES_ARG}" ]; then
    TABLES="${TABLES_ARG}"
else
    # Discover all tables in the database.
    TABLES=$(ch_query "SELECT name FROM system.tables WHERE database = '${CH_DB}' AND name NOT LIKE '.%' ORDER BY name FORMAT TabSeparated")
    if [ -z "${TABLES}" ]; then
        echo "ERROR: No tables found in database ${CH_DB}" >&2
        exit 1
    fi
fi

echo "Tables to backup: ${TABLES}"
echo ""

OVERALL_START=$(date +%s)
RESULTS=""

for table in ${TABLES}; do
    [ -z "${table}" ] && continue

    # Get row count before backup.
    ROW_COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    echo "--- ${table} (${ROW_COUNT} rows) ---"

    START=$(date +%s)
    BACKUP_SQL="BACKUP TABLE ${CH_DB}.${table} TO Disk('backups', '${BACKUP_NAME}/${table}/')"
    RESULT=$(ch_query "${BACKUP_SQL}")
    END=$(date +%s)
    ELAPSED=$((END - START))

    if echo "${RESULT}" | grep -qi "exception"; then
        echo "  FAILED (${ELAPSED}s): ${RESULT}" >&2
        RESULTS="${RESULTS}${table}: FAIL (${ELAPSED}s)\n"
    else
        echo "  OK (${ELAPSED}s)"
        RESULTS="${RESULTS}${table}: OK (${ELAPSED}s, ${ROW_COUNT} rows)\n"
    fi
done

OVERALL_END=$(date +%s)
OVERALL_ELAPSED=$((OVERALL_END - OVERALL_START))

echo ""
echo "=== Backup Summary ==="
echo "Backup name: ${BACKUP_NAME}"
echo "Total time:  ${OVERALL_ELAPSED}s"
printf "  %b" "${RESULTS}"

# Verify backup exists on disk.
BACKUP_CHECK=$(ch_query "SELECT count() FROM system.backups WHERE name = '${BACKUP_NAME}' OR status = 'BACKUP_CREATED'" 2>/dev/null || echo "n/a")
echo ""
echo "Backup verification (system.backups): ${BACKUP_CHECK}"
echo ""
echo "Host-side backup location: ./backups/clickhouse/${BACKUP_NAME}/"
echo "Done."
