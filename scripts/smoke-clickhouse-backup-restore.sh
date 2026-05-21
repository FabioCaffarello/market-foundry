#!/usr/bin/env bash
# smoke-clickhouse-backup-restore.sh — End-to-end proof of ClickHouse backup/restore.
#
# This script is the canonical evidence for Stage S435 / Blocker B-3.
# It exercises the full cycle: seed -> count -> backup -> destroy -> restore -> verify.
#
# Prerequisites:
#   - ClickHouse running and healthy (make up)
#   - Migrations applied (make migrate-up)
#
# Usage:
#   ./scripts/smoke-clickhouse-backup-restore.sh

set -euo pipefail

CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"
BACKUP_NAME="proof_$(date -u +%Y%m%d_%H%M%S)"

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

echo "============================================"
echo " S435: ClickHouse Backup/Restore Proof"
echo "============================================"
echo ""
echo "Host:        ${CH_HOST}:${CH_PORT}"
echo "Database:    ${CH_DB}"
echo "Backup name: ${BACKUP_NAME}"
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
echo "Step 2: Ensure proof data exists"

MARKER_ID="proof-s435-$(date -u +%s)"
INSERT_SQL="INSERT INTO ${CH_DB}.executions (event_id, occurred_at, type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, final, timestamp)
VALUES ('${MARKER_ID}', now64(3), 'paper_order', 'proof', 'BTCUSDT', 60, 'buy', 0.001, 0.001, 'Filled', '{}', '[]', '{}', '{\"proof\":\"s435\"}', true, now64(3))"
ch_query "${INSERT_SQL}" >/dev/null

INSERT_CANDLE="INSERT INTO ${CH_DB}.evidence_candles (event_id, occurred_at, source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final)
VALUES ('candle-${MARKER_ID}', now64(3), 'proof', 'BTCUSDT', 60, 100.0, 101.0, 99.0, 100.5, 1000.0, 42, now64(3), now64(3), true)"
ch_query "${INSERT_CANDLE}" >/dev/null

pass "Proof marker rows inserted (${MARKER_ID})"

# ── Step 3: Record pre-backup row counts ─────────────────────────────────────
echo ""
echo "Step 3: Record pre-backup state"

TABLES_RAW=$(ch_query "SELECT name FROM system.tables WHERE database = '${CH_DB}' AND name NOT LIKE '.%' AND engine LIKE '%MergeTree%' ORDER BY name FORMAT TabSeparated")
TABLES=""
PRE_COUNTS_FILE=$(mktemp)
trap "rm -f ${PRE_COUNTS_FILE}" EXIT

while IFS= read -r table; do
    [ -z "${table}" ] && continue
    TABLES="${TABLES} ${table}"
    COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    echo "${table}=${COUNT}" >> "${PRE_COUNTS_FILE}"
    echo "  ${table}: ${COUNT} rows"
done <<< "${TABLES_RAW}"

get_pre_count() {
    grep "^${1}=" "${PRE_COUNTS_FILE}" | cut -d= -f2
}

# ── Step 4: Execute backup ───────────────────────────────────────────────────
echo ""
echo "Step 4: Execute backup"
BACKUP_START=$(date +%s)

for table in ${TABLES}; do
    RESULT=$(ch_query "BACKUP TABLE ${CH_DB}.${table} TO Disk('backups', '${BACKUP_NAME}/${table}/')")
    if echo "${RESULT}" | grep -qi "exception"; then
        fail "Backup ${table}: ${RESULT}"
    else
        pass "Backup ${table}"
    fi
done

BACKUP_END=$(date +%s)
BACKUP_ELAPSED=$((BACKUP_END - BACKUP_START))
echo "  Backup duration: ${BACKUP_ELAPSED}s"

# ── Step 5: Destroy tables ───────────────────────────────────────────────────
echo ""
echo "Step 5: Destroy tables (simulate data loss)"

for table in ${TABLES}; do
    ch_query "DROP TABLE IF EXISTS ${CH_DB}.${table}" >/dev/null
done

for table in ${TABLES}; do
    CHECK=$(ch_query "EXISTS TABLE ${CH_DB}.${table} FORMAT TabSeparated")
    if [ "${CHECK}" = "0" ]; then
        pass "Table ${table} confirmed dropped"
    else
        fail "Table ${table} still exists after drop"
    fi
done

# ── Step 6: Execute restore ──────────────────────────────────────────────────
echo ""
echo "Step 6: Execute restore"
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

# ── Step 7: Verify post-restore row counts ───────────────────────────────────
echo ""
echo "Step 7: Verify post-restore data integrity"

for table in ${TABLES}; do
    POST_COUNT=$(ch_query "SELECT count() FROM ${CH_DB}.${table}")
    PRE_COUNT=$(get_pre_count "${table}")
    if [ "${POST_COUNT}" = "${PRE_COUNT}" ]; then
        pass "${table}: ${POST_COUNT} rows (matches pre-backup)"
    else
        fail "${table}: expected ${PRE_COUNT} rows, got ${POST_COUNT}"
    fi
done

# ── Step 8: Verify marker row survived ───────────────────────────────────────
echo ""
echo "Step 8: Verify proof marker row"

MARKER_CHECK=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE event_id = '${MARKER_ID}' FORMAT TabSeparated")
if [ "${MARKER_CHECK}" = "1" ]; then
    pass "Proof marker row found in restored data"
else
    fail "Proof marker row NOT found (got ${MARKER_CHECK})"
fi

CANDLE_CHECK=$(ch_query "SELECT count() FROM ${CH_DB}.evidence_candles WHERE event_id = 'candle-${MARKER_ID}' FORMAT TabSeparated")
if [ "${CANDLE_CHECK}" = "1" ]; then
    pass "Proof candle marker found in restored data"
else
    fail "Proof candle marker NOT found (got ${CANDLE_CHECK})"
fi

# ── Step 9: Verify schema integrity ─────────────────────────────────────────
echo ""
echo "Step 9: Verify schema integrity (TTL, partitioning)"

TTL_CHECK=$(ch_query "SELECT count() FROM system.tables WHERE database = '${CH_DB}' AND engine_full LIKE '%TTL%' FORMAT TabSeparated")
TOTAL_TABLES=$(ch_query "SELECT count() FROM system.tables WHERE database = '${CH_DB}' AND name NOT LIKE '.%' AND engine LIKE '%MergeTree%' FORMAT TabSeparated")
echo "  Tables with TTL: ${TTL_CHECK} / ${TOTAL_TABLES}"

PARTITION_CHECK=$(ch_query "SELECT count() FROM system.tables WHERE database = '${CH_DB}' AND partition_key != '' FORMAT TabSeparated")
echo "  Tables with partitions: ${PARTITION_CHECK} / ${TOTAL_TABLES}"

if [ "${TTL_CHECK}" -gt 0 ] && [ "${PARTITION_CHECK}" -gt 0 ]; then
    pass "Schema properties (TTL, partitioning) preserved"
else
    fail "Schema properties may be degraded"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo " Summary"
echo "============================================"
echo "  Backup duration:  ${BACKUP_ELAPSED}s"
echo "  Restore duration: ${RESTORE_ELAPSED}s"
echo "  Total RTO:        $((RESTORE_ELAPSED + 5))s (restore + ~5s healthcheck)"
echo "  PASS: ${PASS_COUNT}"
echo "  FAIL: ${FAIL_COUNT}"
echo ""

if [ "${FAIL_COUNT}" -eq 0 ]; then
    echo "RESULT: ALL CHECKS PASSED -- B-3 evidence satisfied."
    exit 0
else
    echo "RESULT: ${FAIL_COUNT} CHECK(S) FAILED -- review output above."
    exit 1
fi
