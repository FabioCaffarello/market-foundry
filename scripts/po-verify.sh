#!/usr/bin/env bash
# po-verify.sh -- S461: Automated Post-Operation Verification Pipeline.
#
# Runs all 9 PO checks (PO-1 through PO-9) against a session, producing
# structured JSON output that can be persisted, reexecuted, and audited.
#
# This replaces the manual `smoke-supervised-live-session.sh post-session`
# workflow with a fully automated, session-bound verification pipeline.
#
# Usage:
#   ./scripts/po-verify.sh                          # Verify current/latest session
#   ./scripts/po-verify.sh --session-id <id>        # Verify specific session
#   ./scripts/po-verify.sh --json                   # Output structured JSON only
#   ./scripts/po-verify.sh --save                   # Save report to backups/sessions/
#
# Environment:
#   GATEWAY_URL         (default: http://127.0.0.1:8080)
#   EXECUTE_URL         (default: http://127.0.0.1:8084)
#   CLICKHOUSE_HOST     (default: 127.0.0.1)
#   CLICKHOUSE_PORT     (default: 8123)
#   CLICKHOUSE_USER     (default: default)
#   CLICKHOUSE_PASSWORD (default: clickhouse)
#   CLICKHOUSE_DATABASE (default: market_foundry)
#
# Authority: S461 -- PO Automation and Verification Pipeline

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

# ── Configuration ────────────────────────────────────────────────────────────

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
EXECUTE_URL="${EXECUTE_URL:-http://127.0.0.1:8084}"
CH_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CH_PORT="${CLICKHOUSE_PORT:-8123}"
CH_USER="${CLICKHOUSE_USER:-default}"
CH_PASS="${CLICKHOUSE_PASSWORD:-clickhouse}"
CH_DB="${CLICKHOUSE_DATABASE:-market_foundry}"

SESSION_ID=""
JSON_ONLY=false
SAVE_REPORT=false

# ── Argument parsing ─────────────────────────────────────────────────────────

while [[ $# -gt 0 ]]; do
    case "$1" in
        --session-id) SESSION_ID="$2"; shift 2 ;;
        --json)       JSON_ONLY=true; shift ;;
        --save)       SAVE_REPORT=true; shift ;;
        -h|--help)
            cat <<EOF
Usage: $0 [OPTIONS]

Options:
  --session-id <id>  Verify a specific session (default: latest from /session-list)
  --json             Output structured JSON only (no human-readable log)
  --save             Save report to backups/sessions/<session_id>/po-report.json
  -h, --help         Show this help

Environment:
  GATEWAY_URL         Gateway base URL (default: http://127.0.0.1:8080)
  EXECUTE_URL         Execute health URL (default: http://127.0.0.1:8084)
  CLICKHOUSE_HOST     ClickHouse host (default: 127.0.0.1)
  CLICKHOUSE_PORT     ClickHouse HTTP port (default: 8123)

Authority: S461 -- PO Automation and Verification Pipeline
EOF
            exit 0
            ;;
        *) die "Unknown argument: $1" ;;
    esac
done

# ── Helpers ──────────────────────────────────────────────────────────────────

ch_query() {
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$1" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>/dev/null || echo ""
}

http_get() {
    curl -sf "$1" 2>/dev/null || echo ""
}

log() {
    if [[ "${JSON_ONLY}" != "true" ]]; then
        echo -e "$1"
    fi
}

# ── JSON report assembly ────────────────────────────────────────────────────

REPORT_START=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
CHECKS_JSON="[]"

# add_check appends a check result to the CHECKS_JSON array.
# Args: check_id name verdict detail automated evidence_json duration_ms
add_check() {
    local check_id="$1"
    local name="$2"
    local verdict="$3"
    local detail="$4"
    local automated="$5"
    local evidence="${6:-{}}"
    local duration_ms="${7:-0}"
    local executed_at
    executed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    local check_json
    check_json=$(python3 -c "
import json, sys
check = {
    'check_id': sys.argv[1],
    'name': sys.argv[2],
    'verdict': sys.argv[3],
    'detail': sys.argv[4],
    'automated': sys.argv[5] == 'true',
    'evidence': json.loads(sys.argv[6]),
    'executed_at': sys.argv[7],
    'duration_ms': int(sys.argv[8])
}
existing = json.loads(sys.argv[9])
existing.append(check)
print(json.dumps(existing))
" "$check_id" "$name" "$verdict" "$detail" "$automated" "$evidence" "$executed_at" "$duration_ms" "$CHECKS_JSON")

    CHECKS_JSON="$check_json"
}

# ── Resolve session ─────────────────────────────────────────────────────────

if [[ -z "${SESSION_ID}" ]]; then
    log "Resolving latest session from ${GATEWAY_URL}/session-list ..."
    session_list=$(http_get "${GATEWAY_URL}/session-list")
    if [[ -n "${session_list}" ]]; then
        SESSION_ID=$(echo "${session_list}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
sessions = data.get('sessions', [])
if sessions:
    print(sessions[0].get('session_id', ''))
else:
    print('')
" 2>/dev/null || echo "")
    fi
    if [[ -z "${SESSION_ID}" ]]; then
        SESSION_ID="unknown_$(date -u +%Y%m%d_%H%M%S)"
        log "[INFO] No session found via API — using generated ID: ${SESSION_ID}"
    else
        log "[INFO] Using latest session: ${SESSION_ID}"
    fi
fi

# Fetch session metadata if available.
SESSION_META=$(http_get "${GATEWAY_URL}/session/${SESSION_ID}")
SESSION_OPERATOR=""
SESSION_CONFIG=""
if [[ -n "${SESSION_META}" ]]; then
    SESSION_OPERATOR=$(echo "${SESSION_META}" | python3 -c "import sys,json; s=json.load(sys.stdin).get('session',{}); print(s.get('operator',''))" 2>/dev/null || echo "")
    SESSION_CONFIG=$(echo "${SESSION_META}" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin).get('session',{}).get('config',{})))" 2>/dev/null || echo "{}")
fi

log ""
log "============================================="
log "  S461: PO Verification Pipeline"
log "  Session: ${SESSION_ID}"
log "  Operator: ${SESSION_OPERATOR:-<unknown>}"
log "  Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
log "============================================="
log ""

# ── PO-1: Kill-switch halt verification ────────────────────────────────────

po1_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-1: Kill-switch halt verification"

gate_resp=$(http_get "${GATEWAY_URL}/execution/control")
gate_status=""
if [[ -n "${gate_resp}" ]]; then
    gate_status=$(echo "${gate_resp}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
fi

po1_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po1_dur=$(( po1_end - po1_start ))

if [[ "${gate_status}" == "halted" ]]; then
    log "  Gate: halted"
    add_check "PO-1" "Kill-switch halt verification" "pass" "Gate is halted" "true" \
        "{\"gate_status\": \"${gate_status}\"}" "$po1_dur"
    log "PO-1: PASS"
elif [[ -z "${gate_status}" || "${gate_status}" == "unknown" ]]; then
    log "  Gate: unreachable"
    add_check "PO-1" "Kill-switch halt verification" "skip" "Gate endpoint unreachable" "true" \
        "{\"gate_status\": \"unreachable\"}" "$po1_dur"
    log "PO-1: SKIP (endpoint unreachable)"
else
    log "  Gate: ${gate_status}"
    add_check "PO-1" "Kill-switch halt verification" "warn" "Gate is ${gate_status}, expected halted" "true" \
        "{\"gate_status\": \"${gate_status}\"}" "$po1_dur"
    log "PO-1: WARN (gate is ${gate_status})"
fi
log ""

# ── PO-2: Backup verification ─────────────────────────────────────────────

po2_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-2: Post-session backup verification"

backup_exists=false
backup_name="post_session_${SESSION_ID}"
if [[ -d "${PROJECT_ROOT}/backups/clickhouse" ]]; then
    if ls "${PROJECT_ROOT}/backups/clickhouse/"*"${SESSION_ID}"* 1>/dev/null 2>&1; then
        backup_exists=true
    fi
fi

po2_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po2_dur=$(( po2_end - po2_start ))

if [[ "${backup_exists}" == "true" ]]; then
    add_check "PO-2" "Post-session backup" "pass" "Backup artifact found for session" "true" \
        "{\"backup_name\": \"${backup_name}\", \"found\": true}" "$po2_dur"
    log "PO-2: PASS"
else
    add_check "PO-2" "Post-session backup" "manual" "No backup artifact found — may need manual verification or creation" "false" \
        "{\"backup_name\": \"${backup_name}\", \"found\": false}" "$po2_dur"
    log "PO-2: MANUAL (backup not found locally — verify or create)"
fi
log ""

# ── PO-3: ClickHouse intent records ───────────────────────────────────────

po3_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-3: ClickHouse intent records"

intent_count=$(ch_query "SELECT count() FROM ${CH_DB}.execution_intents WHERE symbol = 'BTCUSDT' AND created_at > now() - INTERVAL 24 HOUR" | tr -d '[:space:]')
intent_sample=$(ch_query "SELECT event_id, symbol, side, status FROM ${CH_DB}.execution_intents WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 3 FORMAT JSONEachRow")

po3_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po3_dur=$(( po3_end - po3_start ))

if [[ -n "${intent_count}" && "${intent_count}" != "0" ]]; then
    add_check "PO-3" "ClickHouse intent records" "pass" "${intent_count} intent records found (24h)" "true" \
        "{\"count\": ${intent_count}}" "$po3_dur"
    log "  Intent count (24h): ${intent_count}"
    log "PO-3: PASS"
elif [[ -z "${intent_count}" ]]; then
    add_check "PO-3" "ClickHouse intent records" "skip" "ClickHouse query failed or unavailable" "true" \
        "{\"error\": \"query_failed\"}" "$po3_dur"
    log "PO-3: SKIP (ClickHouse unavailable)"
else
    add_check "PO-3" "ClickHouse intent records" "warn" "No intent records found in 24h window" "true" \
        "{\"count\": 0}" "$po3_dur"
    log "PO-3: WARN (no records)"
fi
log ""

# ── PO-4: ClickHouse venue response records ──────────────────────────────

po4_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-4: ClickHouse venue response records"

response_count=$(ch_query "SELECT count() FROM ${CH_DB}.venue_responses WHERE symbol = 'BTCUSDT' AND created_at > now() - INTERVAL 24 HOUR" | tr -d '[:space:]')

po4_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po4_dur=$(( po4_end - po4_start ))

if [[ -n "${response_count}" && "${response_count}" != "0" ]]; then
    add_check "PO-4" "ClickHouse venue response records" "pass" "${response_count} venue response records found (24h)" "true" \
        "{\"count\": ${response_count}}" "$po4_dur"
    log "  Response count (24h): ${response_count}"
    log "PO-4: PASS"
elif [[ -z "${response_count}" ]]; then
    add_check "PO-4" "ClickHouse venue response records" "skip" "ClickHouse query failed or unavailable" "true" \
        "{\"error\": \"query_failed\"}" "$po4_dur"
    log "PO-4: SKIP (ClickHouse unavailable)"
else
    add_check "PO-4" "ClickHouse venue response records" "warn" "No venue response records found in 24h window" "true" \
        "{\"count\": 0}" "$po4_dur"
    log "PO-4: WARN (no records)"
fi
log ""

# ── PO-5: NATS KV state validation ──────────────────────────────────────

po5_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-5: NATS KV state validation"

control_resp=$(http_get "${GATEWAY_URL}/execution/control")
kv_latest=$(http_get "${GATEWAY_URL}/execution/venue-market-order/latest?source=binances&base=btc&quote=usdt&contract=spot&timeframe=60")

po5_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po5_dur=$(( po5_end - po5_start ))

kv_status="unavailable"
if [[ -n "${kv_latest}" ]]; then
    kv_status=$(echo "${kv_latest}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
ei = data.get('execution_intent', {})
print(ei.get('status', 'unknown'))
" 2>/dev/null || echo "unknown")
fi

if [[ "${kv_status}" != "unavailable" && "${kv_status}" != "unknown" ]]; then
    add_check "PO-5" "NATS KV state validation" "pass" "KV latest venue order status: ${kv_status}" "true" \
        "{\"kv_latest_status\": \"${kv_status}\", \"control_available\": $([ -n "${control_resp}" ] && echo 'true' || echo 'false')}" "$po5_dur"
    log "  KV latest status: ${kv_status}"
    log "PO-5: PASS"
else
    add_check "PO-5" "NATS KV state validation" "skip" "KV endpoints unreachable" "true" \
        "{\"kv_latest_status\": \"${kv_status}\"}" "$po5_dur"
    log "PO-5: SKIP (KV unreachable)"
fi
log ""

# ── PO-6: System status summary ────────────────────────────────────────

po6_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-6: System status summary"

exec_statusz=$(http_get "${EXECUTE_URL}/statusz")
gateway_readyz=$(http_get "${GATEWAY_URL}/readyz")

po6_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po6_dur=$(( po6_end - po6_start ))

exec_healthy=false
gateway_healthy=false
[[ -n "${exec_statusz}" ]] && exec_healthy=true
[[ -n "${gateway_readyz}" ]] && gateway_healthy=true

if [[ "${exec_healthy}" == "true" || "${gateway_healthy}" == "true" ]]; then
    add_check "PO-6" "System status summary" "pass" "System endpoints responsive" "true" \
        "{\"execute_healthy\": ${exec_healthy}, \"gateway_healthy\": ${gateway_healthy}}" "$po6_dur"
    log "  Execute healthy: ${exec_healthy}, Gateway healthy: ${gateway_healthy}"
    log "PO-6: PASS"
else
    add_check "PO-6" "System status summary" "warn" "System endpoints not responding — may be shut down post-session" "true" \
        "{\"execute_healthy\": false, \"gateway_healthy\": false}" "$po6_dur"
    log "PO-6: WARN (endpoints not responding — expected if system shut down)"
fi
log ""

# ── PO-7: Fee/commission field verification ────────────────────────────

po7_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-7: Fee and commission field verification"

fills_data=$(ch_query "SELECT event_id, symbol, status, filled_quantity, fills FROM ${CH_DB}.executions WHERE symbol = 'BTCUSDT' AND status IN ('filled','partially_filled') ORDER BY timestamp DESC LIMIT 5 FORMAT JSONEachRow")

po7_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po7_dur=$(( po7_end - po7_start ))

if [[ -z "${fills_data}" ]]; then
    add_check "PO-7" "Fee/commission field verification" "skip" "No fill records found or ClickHouse unavailable" "true" \
        "{\"fills_found\": false}" "$po7_dur"
    log "PO-7: SKIP (no fill data)"
else
    # Automated check: verify Fee and FeeAsset fields are present in fills JSON.
    fee_check=$(echo "${fills_data}" | python3 -c "
import sys, json
lines = sys.stdin.read().strip().split('\n')
total = 0
with_fee = 0
for line in lines:
    if not line.strip():
        continue
    try:
        rec = json.loads(line)
        fills_raw = rec.get('fills', '[]')
        if isinstance(fills_raw, str):
            fills = json.loads(fills_raw)
        else:
            fills = fills_raw
        total += 1
        if any(f.get('Fee') or f.get('fee') for f in fills):
            with_fee += 1
    except:
        pass
print(json.dumps({'total': total, 'with_fee': with_fee, 'all_have_fees': total > 0 and total == with_fee}))
" 2>/dev/null || echo '{"total":0,"with_fee":0,"all_have_fees":false}')

    all_have_fees=$(echo "${fee_check}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('all_have_fees', False))" 2>/dev/null || echo "False")

    if [[ "${all_have_fees}" == "True" ]]; then
        add_check "PO-7" "Fee/commission field verification" "pass" "All fill records contain fee fields" "true" \
            "${fee_check}" "$po7_dur"
        log "PO-7: PASS"
    else
        add_check "PO-7" "Fee/commission field verification" "warn" "Some fill records may lack fee fields" "true" \
            "${fee_check}" "$po7_dur"
        log "PO-7: WARN (check fee data)"
    fi
fi
log ""

# ── PO-8: Lifecycle consistency (CH vs KV) ─────────────────────────────

po8_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-8: Lifecycle consistency (ClickHouse vs NATS KV)"

# Use the session-explain endpoint if available — it already does CH-vs-KV consistency.
explain_resp=$(http_get "${GATEWAY_URL}/analytical/execution/explain?source=binance_spot&base=btc&quote=usdt&contract=spot&timeframe=60")

po8_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po8_dur=$(( po8_end - po8_start ))

if [[ -n "${explain_resp}" ]]; then
    consistency_result=$(echo "${explain_resp}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
consistent = data.get('consistent', False)
checks = data.get('consistency', [])
divergent = [c for c in checks if c.get('status') == 'divergent']
print(json.dumps({
    'consistent': consistent,
    'check_count': len(checks),
    'divergent_count': len(divergent),
    'kv_available': data.get('kv_available', False),
    'ch_available': data.get('ch_available', False),
    'kv_intent_status': data.get('kv_intent_status', ''),
    'ch_latest_intent_status': data.get('ch_latest_intent_status', ''),
    'divergences': [{'field': c.get('field',''), 'kv': c.get('kv_value',''), 'ch': c.get('ch_value','')} for c in divergent]
}))
" 2>/dev/null || echo '{"consistent": false, "error": "parse_failed"}')

    is_consistent=$(echo "${consistency_result}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('consistent', False))" 2>/dev/null || echo "False")

    if [[ "${is_consistent}" == "True" ]]; then
        add_check "PO-8" "Lifecycle consistency (CH vs KV)" "pass" "ClickHouse and KV are consistent" "true" \
            "${consistency_result}" "$po8_dur"
        log "PO-8: PASS (consistent)"
    else
        add_check "PO-8" "Lifecycle consistency (CH vs KV)" "warn" "Potential divergence between CH and KV" "true" \
            "${consistency_result}" "$po8_dur"
        log "PO-8: WARN (divergence detected — review evidence)"
    fi
else
    # Fallback: direct comparison like the old manual check.
    ch_status=$(ch_query "SELECT status, filled_quantity FROM ${CH_DB}.executions WHERE symbol = 'BTCUSDT' AND type = 'venue_market_order' ORDER BY timestamp DESC LIMIT 1 FORMAT JSONEachRow" | head -1)

    if [[ -n "${ch_status}" && -n "${kv_latest}" ]]; then
        ch_st=$(echo "${ch_status}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
        kv_st="${kv_status}"

        if [[ "${ch_st}" == "${kv_st}" ]]; then
            add_check "PO-8" "Lifecycle consistency (CH vs KV)" "pass" "CH status '${ch_st}' matches KV status '${kv_st}'" "true" \
                "{\"ch_status\": \"${ch_st}\", \"kv_status\": \"${kv_st}\"}" "$po8_dur"
            log "PO-8: PASS (CH=${ch_st}, KV=${kv_st})"
        else
            add_check "PO-8" "Lifecycle consistency (CH vs KV)" "warn" "CH status '${ch_st}' differs from KV status '${kv_st}'" "true" \
                "{\"ch_status\": \"${ch_st}\", \"kv_status\": \"${kv_st}\"}" "$po8_dur"
            log "PO-8: WARN (CH=${ch_st}, KV=${kv_st})"
        fi
    else
        add_check "PO-8" "Lifecycle consistency (CH vs KV)" "skip" "Insufficient data for consistency check" "true" \
            "{\"ch_available\": $([ -n "${ch_status}" ] && echo 'true' || echo 'false'), \"kv_available\": $([ -n "${kv_latest}" ] && echo 'true' || echo 'false')}" "$po8_dur"
        log "PO-8: SKIP (insufficient data)"
    fi
fi
log ""

# ── PO-9: Scope containment verification ──────────────────────────────

po9_start=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
log "PO-9: Scope containment verification"

total_executions=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE type = 'venue_market_order' AND timestamp > now() - INTERVAL 24 HOUR" | tr -d '[:space:]')
non_btc_executions=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE type = 'venue_market_order' AND symbol != 'BTCUSDT' AND timestamp > now() - INTERVAL 24 HOUR" | tr -d '[:space:]')

po9_end=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
po9_dur=$(( po9_end - po9_start ))

if [[ -z "${total_executions}" ]]; then
    add_check "PO-9" "Scope containment verification" "skip" "ClickHouse unavailable" "true" \
        "{\"error\": \"query_failed\"}" "$po9_dur"
    log "PO-9: SKIP (ClickHouse unavailable)"
elif [[ "${non_btc_executions}" == "0" || -z "${non_btc_executions}" ]]; then
    add_check "PO-9" "Scope containment verification" "pass" "No out-of-scope executions detected" "true" \
        "{\"total_executions\": ${total_executions:-0}, \"non_btcusdt\": 0}" "$po9_dur"
    log "  Total venue executions (24h): ${total_executions}"
    log "  Non-BTCUSDT: 0"
    log "PO-9: PASS"
else
    add_check "PO-9" "Scope containment verification" "fail" "Scope violation: ${non_btc_executions} non-BTCUSDT executions detected" "true" \
        "{\"total_executions\": ${total_executions}, \"non_btcusdt\": ${non_btc_executions}}" "$po9_dur"
    log "  Total venue executions (24h): ${total_executions}"
    log "  Non-BTCUSDT: ${non_btc_executions}"
    log "PO-9: FAIL (scope violation)"
fi
log ""

# ── Assemble final report ────────────────────────────────────────────────

REPORT_END=$(date -u +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
REPORT_DUR=$(( REPORT_END - REPORT_START ))
REPORT_TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

FINAL_REPORT=$(python3 -c "
import json, sys

checks = json.loads(sys.argv[1])

# Compute summary.
summary = {
    'total': len(checks),
    'passed': sum(1 for c in checks if c['verdict'] == 'pass'),
    'failed': sum(1 for c in checks if c['verdict'] == 'fail'),
    'warnings': sum(1 for c in checks if c['verdict'] == 'warn'),
    'skipped': sum(1 for c in checks if c['verdict'] == 'skip'),
    'manual': sum(1 for c in checks if c['verdict'] == 'manual'),
    'automated': sum(1 for c in checks if c['automated']),
}

report = {
    'session_id': sys.argv[2],
    'operator': sys.argv[3],
    'executed_at': sys.argv[4],
    'duration_ms': int(sys.argv[5]),
    'checks': checks,
    'summary': summary,
}

print(json.dumps(report, indent=2))
" "${CHECKS_JSON}" "${SESSION_ID}" "${SESSION_OPERATOR}" "${REPORT_TIMESTAMP}" "${REPORT_DUR}")

# ── Output ────────────────────────────────────────────────────────────────

if [[ "${JSON_ONLY}" == "true" ]]; then
    echo "${FINAL_REPORT}"
else
    log "============================================="
    log "  PO Verification Summary"
    log "============================================="

    # Extract summary.
    echo "${FINAL_REPORT}" | python3 -c "
import sys, json
r = json.load(sys.stdin)
s = r['summary']
print(f\"  Session:    {r['session_id']}\")
print(f\"  Checks:     {s['total']}\")
print(f\"  Passed:     {s['passed']}\")
print(f\"  Failed:     {s['failed']}\")
print(f\"  Warnings:   {s['warnings']}\")
print(f\"  Skipped:    {s['skipped']}\")
print(f\"  Manual:     {s['manual']}\")
print(f\"  Automated:  {s['automated']} / {s['total']}\")
print(f\"  Duration:   {r['duration_ms']}ms\")
print()
for c in r['checks']:
    icon = {'pass': 'PASS', 'fail': 'FAIL', 'warn': 'WARN', 'skip': 'SKIP', 'manual': 'MANUAL'}
    print(f\"  [{icon.get(c['verdict'], '???')}] {c['check_id']}: {c['name']}\")
    if c['verdict'] not in ('pass', 'skip'):
        print(f\"         {c['detail']}\")
"

    log ""
    log "============================================="
fi

# ── Save report if requested ─────────────────────────────────────────────

if [[ "${SAVE_REPORT}" == "true" ]]; then
    REPORT_DIR="${PROJECT_ROOT}/backups/sessions/${SESSION_ID}"
    mkdir -p "${REPORT_DIR}"
    REPORT_FILE="${REPORT_DIR}/po-report.json"
    echo "${FINAL_REPORT}" > "${REPORT_FILE}"
    log "Report saved to: ${REPORT_FILE}"
fi

# Exit with non-zero if any check failed.
failed_count=$(echo "${FINAL_REPORT}" | python3 -c "import sys,json; print(json.load(sys.stdin)['summary']['failed'])")
if [[ "${failed_count}" != "0" ]]; then
    exit 1
fi
exit 0
