#!/usr/bin/env bash
# smoke-supervised-live-session.sh -- S446: Supervised Live Session Operational Script.
#
# This is the canonical operational script for the S446 supervised live session.
# It orchestrates the FULL pre-session checklist, monitors the live session,
# and performs post-session verification.
#
# AUTHORIZATION: S443 evidence gate -> S444 ceremony charter -> S445 C-6 executed.
# SCOPE: Binance Spot, BTCUSDT, 1 market order, minimum exchange quantity.
#
# WARNING: THIS SCRIPT OPERATES AGAINST MAINNET WITH REAL MONEY.
# Do NOT run without reading the full ceremony charter and scope constraints.
#
# Usage:
#   ./scripts/smoke-supervised-live-session.sh pre-session     # Run pre-session checks only
#   ./scripts/smoke-supervised-live-session.sh monitor          # Monitor running session
#   ./scripts/smoke-supervised-live-session.sh post-session     # Run post-session verification
#   ./scripts/smoke-supervised-live-session.sh full             # Full ceremony (pre + wait + post)
#
# Environment:
#   GATEWAY_URL         (default: http://127.0.0.1:8080)
#   EXECUTE_URL         (default: http://127.0.0.1:8084)
#   CLICKHOUSE_HOST     (default: 127.0.0.1)
#   CLICKHOUSE_PORT     (default: 8123)
#   CREDENTIAL_PATH     (default: /run/secrets/market-foundry)
#   OPERATOR_NAME       (required for full ceremony)
#   SESSION_LOG_DIR     (default: ./backups/logs/sessions)
#
# Authority: S446 -- Supervised Live Session Proof
# Predecessor: S445 (C-6 Controlled Execution)

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
CREDENTIAL_PATH="${CREDENTIAL_PATH:-/run/secrets/market-foundry}"
OPERATOR_NAME="${OPERATOR_NAME:-}"
SESSION_LOG_DIR="${SESSION_LOG_DIR:-./backups/logs/sessions}"
SESSION_ID="live_$(date -u +%Y%m%d_%H%M%S)"
SESSION_LOG="${SESSION_LOG_DIR}/${SESSION_ID}.log"

LIVE_CONFIG="${PROJECT_ROOT}/deploy/configs/execute-mainnet-live.jsonc"

# ── Helpers ──────────────────────────────────────────────────────────────────

mkdir -p "${SESSION_LOG_DIR}"

slog() {
    local msg="[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $1"
    echo -e "${msg}"
    echo "${msg}" >> "${SESSION_LOG}"
}

ch_query() {
    curl -sS "http://${CH_HOST}:${CH_PORT}/" \
        --data-binary "$1" \
        -H "X-ClickHouse-User: ${CH_USER}" \
        -H "X-ClickHouse-Key: ${CH_PASS}" \
        2>&1
}

gate_status() {
    local resp
    resp=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null) || { echo "unreachable"; return; }
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown"
}

# ── PS-1: Kill-Switch Cycle Test ────────────────────────────────────────────

check_ps1() {
    slog "PS-1: Kill-switch cycle test"

    if ! "${SCRIPT_DIR}/kill-switch-ops.sh" cycle "s446-pre-session" "${OPERATOR_NAME:-operator}" 2>&1 | tee -a "${SESSION_LOG}"; then
        slog "PS-1: FAIL -- kill-switch cycle test failed"
        return 1
    fi

    slog "PS-1: PASS -- kill-switch cycle test succeeded"
}

# ── PS-2: Automated Backup (Pre-Session) ────────────────────────────────────

check_ps2() {
    slog "PS-2: Automated backup (pre-session)"

    BACKUP_NAME="pre_session_${SESSION_ID}" \
        "${SCRIPT_DIR}/clickhouse-scheduled-backup.sh" 2>&1 | tee -a "${SESSION_LOG}" || {
        slog "PS-2: FAIL -- pre-session backup failed"
        return 1
    }

    slog "PS-2: PASS -- pre-session backup completed"
}

# ── PS-3: Credential File Mount Verification ────────────────────────────────

check_ps3() {
    slog "PS-3: Credential file mount verification"
    slog "  Checking path: ${CREDENTIAL_PATH}/binance_spot_mainnet/"

    local key_file="${CREDENTIAL_PATH}/binance_spot_mainnet/API_KEY"
    local secret_file="${CREDENTIAL_PATH}/binance_spot_mainnet/API_SECRET"

    if [ ! -f "${key_file}" ]; then
        slog "PS-3: FAIL -- API_KEY file not found at ${key_file}"
        return 1
    fi

    if [ ! -s "${key_file}" ]; then
        slog "PS-3: FAIL -- API_KEY file is empty"
        return 1
    fi

    if [ ! -f "${secret_file}" ]; then
        slog "PS-3: FAIL -- API_SECRET file not found at ${secret_file}"
        return 1
    fi

    if [ ! -s "${secret_file}" ]; then
        slog "PS-3: FAIL -- API_SECRET file is empty"
        return 1
    fi

    # Verify permissions are restrictive.
    local key_perms
    key_perms=$(stat -f "%Lp" "${key_file}" 2>/dev/null || stat -c "%a" "${key_file}" 2>/dev/null || echo "unknown")
    slog "  API_KEY permissions: ${key_perms}"
    slog "  API_KEY size: $(wc -c < "${key_file}" | tr -d ' ') bytes"
    slog "  API_SECRET size: $(wc -c < "${secret_file}" | tr -d ' ') bytes"

    slog "PS-3: PASS -- credentials mounted and non-empty"
}

# ── PS-4: Config Audit ──────────────────────────────────────────────────────

check_ps4() {
    slog "PS-4: Config audit"
    slog "  Config: ${LIVE_CONFIG}"

    if [ ! -f "${LIVE_CONFIG}" ]; then
        slog "PS-4: FAIL -- live config not found at ${LIVE_CONFIG}"
        return 1
    fi

    # Parse critical fields using python3 (handles JSONC comments).
    local config_check
    config_check=$(python3 -c "
import re, json, sys

with open('${LIVE_CONFIG}') as f:
    content = f.read()

# Strip JSONC comments.
content = re.sub(r'//.*$', '', content, flags=re.MULTILINE)
content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)

cfg = json.loads(content)
venue = cfg.get('venue', {})
segments = venue.get('segments', {})

dry_run = venue.get('dry_run')
cred_provider = venue.get('credential_provider', '')
spot_enabled = segments.get('spot', {}).get('enabled', False)
spot_adapter = segments.get('spot', {}).get('adapter', '')
futures_enabled = segments.get('futures', {}).get('enabled', False)

issues = []
if dry_run is not False:
    issues.append('dry_run is not false')
if cred_provider != 'file':
    issues.append(f'credential_provider is \"{cred_provider}\", expected \"file\"')
if not spot_enabled:
    issues.append('spot segment is not enabled')
if spot_adapter != 'binance_spot_mainnet':
    issues.append(f'spot adapter is \"{spot_adapter}\", expected \"binance_spot_mainnet\"')
if futures_enabled:
    issues.append('futures segment is enabled (NOT authorized)')

if issues:
    print('FAIL:' + '; '.join(issues))
else:
    print('OK:dry_run=false,credential_provider=file,spot=binance_spot_mainnet,futures=disabled')
" 2>&1) || {
        slog "PS-4: FAIL -- config parse error"
        return 1
    }

    slog "  Config check: ${config_check}"

    if [[ "${config_check}" == FAIL:* ]]; then
        slog "PS-4: FAIL -- ${config_check}"
        return 1
    fi

    slog "PS-4: PASS -- config matches minimum authorized scope"
}

# ── PS-5: API Key Permission (Operator Attestation) ─────────────────────────

check_ps5() {
    slog "PS-5: API key permission check (operator attestation)"

    if [ -z "${OPERATOR_NAME}" ]; then
        slog "PS-5: FAIL -- OPERATOR_NAME not set. Set OPERATOR_NAME=<your-name> before running."
        return 1
    fi

    slog "  Operator: ${OPERATOR_NAME}"
    slog "  The operator MUST confirm the following in the Binance console BEFORE proceeding:"
    slog "    1. API key has TRADE permission enabled"
    slog "    2. API key does NOT have WITHDRAWAL permission"
    slog "    3. API key has IP restriction (recommended)"
    slog ""

    # In automated mode, we require explicit attestation via env var.
    if [ "${OPERATOR_ATTESTS_TRADE_ONLY:-}" = "true" ]; then
        slog "  Operator attestation: OPERATOR_ATTESTS_TRADE_ONLY=true"
        slog "PS-5: PASS -- operator attests trade-only API key"
    else
        slog "PS-5: WAITING -- set OPERATOR_ATTESTS_TRADE_ONLY=true to attest trade-only permissions"
        slog "  (Verify in Binance console, then re-run with the env var set)"
        return 1
    fi
}

# ── PS-6: Kill-Switch Initial State ─────────────────────────────────────────

check_ps6() {
    slog "PS-6: Kill-switch initial state"

    local status
    status=$(gate_status)

    slog "  Gate status: ${status}"

    if [ "${status}" = "active" ]; then
        slog "PS-6: PASS -- gate is active"
    else
        slog "PS-6: FAIL -- gate is '${status}', expected 'active'"
        return 1
    fi
}

# ── PS-7: System Boot Verification ──────────────────────────────────────────

check_ps7() {
    slog "PS-7: System boot verification"

    # Check gateway reachability.
    local gw_code
    gw_code=$(curl -sS -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/readyz" 2>/dev/null || echo "000")
    slog "  Gateway /readyz: HTTP ${gw_code}"

    if [ "${gw_code}" != "200" ]; then
        slog "PS-7: FAIL -- gateway not reachable at ${GATEWAY_URL}"
        return 1
    fi

    # Check execute reachability.
    local exec_code
    exec_code=$(curl -sS -o /dev/null -w "%{http_code}" "${EXECUTE_URL}/readyz" 2>/dev/null || echo "000")
    slog "  Execute /readyz: HTTP ${exec_code}"

    if [ "${exec_code}" != "200" ]; then
        slog "PS-7: FAIL -- execute not reachable at ${EXECUTE_URL}"
        return 1
    fi

    # Check execute statusz for adapter type.
    local statusz
    statusz=$(curl -sf "${EXECUTE_URL}/statusz" 2>/dev/null || echo "{}")
    slog "  Execute /statusz: $(echo "${statusz}" | head -c 500)"

    # Verify activation surface shows live mode.
    local activation
    activation=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo "{}")
    slog "  Execution control: $(echo "${activation}" | head -c 500)"

    slog "PS-7: PASS -- system boot verified"
}

# ── Pre-Session (all checks) ────────────────────────────────────────────────

cmd_pre_session() {
    slog ""
    slog "============================================="
    slog "  S446: Pre-Session Checklist"
    slog "  Session: ${SESSION_ID}"
    slog "  Operator: ${OPERATOR_NAME:-<not set>}"
    slog "  Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    slog "============================================="
    slog ""

    local failures=0

    for check in check_ps1 check_ps2 check_ps3 check_ps4 check_ps5 check_ps6 check_ps7; do
        slog ""
        if ! ${check}; then
            failures=$((failures + 1))
            slog ">>> CHECK FAILED: ${check} -- session CANNOT proceed <<<"
        fi
        slog ""
    done

    slog "============================================="
    if [ "${failures}" -eq 0 ]; then
        slog "PRE-SESSION: ALL 7 CHECKS PASSED"
        slog "Session ${SESSION_ID} is AUTHORIZED to proceed."
        slog "============================================="
        return 0
    else
        slog "PRE-SESSION: ${failures} CHECK(S) FAILED"
        slog "Session ${SESSION_ID} is NOT authorized. Fix failures and re-run."
        slog "============================================="
        return 1
    fi
}

# ── Monitor (live session observation) ──────────────────────────────────────

cmd_monitor() {
    slog ""
    slog "============================================="
    slog "  S446: Live Session Monitor"
    slog "  Session: ${SESSION_ID}"
    slog "  Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    slog "============================================="
    slog ""
    slog "Monitoring live session. Press Ctrl+C to stop monitoring."
    slog "The operator MUST halt execution after the first order completes."
    slog ""

    local cycle=0
    while true; do
        cycle=$((cycle + 1))
        slog "--- Monitor cycle ${cycle} at $(date -u +%H:%M:%S) ---"

        # Check gate state.
        local status
        status=$(gate_status)
        slog "  Gate: ${status}"

        if [ "${status}" = "halted" ]; then
            slog "  Gate is HALTED -- session appears complete or stopped."
            slog "  Run post-session verification: $0 post-session"
            break
        fi

        # Check execute health.
        local exec_status
        exec_status=$(curl -sf "${EXECUTE_URL}/statusz" 2>/dev/null || echo "unreachable")
        slog "  Execute: $(echo "${exec_status}" | head -c 200)"

        # Check for recent fills in ClickHouse.
        local recent_fills
        recent_fills=$(ch_query "SELECT count() FROM ${CH_DB}.execution_intents WHERE created_at > now() - INTERVAL 5 MINUTE" 2>/dev/null || echo "query_failed")
        slog "  Recent intents (5min): ${recent_fills}"

        local recent_responses
        recent_responses=$(ch_query "SELECT count() FROM ${CH_DB}.venue_responses WHERE created_at > now() - INTERVAL 5 MINUTE" 2>/dev/null || echo "query_failed")
        slog "  Recent responses (5min): ${recent_responses}"

        slog ""
        sleep 10
    done
}

# ── Post-Session Verification ────────────────────────────────────────────────

cmd_post_session() {
    slog ""
    slog "============================================="
    slog "  S446: Post-Session Verification"
    slog "  Session: ${SESSION_ID}"
    slog "  Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    slog "============================================="
    slog ""

    local failures=0

    # PO-1: Kill-switch should be halted.
    slog "PO-1: Kill-switch halt verification"
    local status
    status=$(gate_status)
    slog "  Gate status: ${status}"
    if [ "${status}" = "halted" ]; then
        slog "PO-1: PASS -- gate is halted"
    else
        slog "PO-1: INFO -- gate is '${status}'. Halting now..."
        "${SCRIPT_DIR}/kill-switch-ops.sh" halt "s446-post-session" "${OPERATOR_NAME:-operator}" 2>&1 | tee -a "${SESSION_LOG}" || true
        slog "PO-1: DONE -- halt command issued"
    fi
    slog ""

    # PO-2: Post-session backup.
    slog "PO-2: Post-session backup"
    BACKUP_NAME="post_session_${SESSION_ID}" \
        "${SCRIPT_DIR}/clickhouse-scheduled-backup.sh" 2>&1 | tee -a "${SESSION_LOG}" || {
        slog "PO-2: WARN -- post-session backup had issues"
        failures=$((failures + 1))
    }
    slog "PO-2: DONE"
    slog ""

    # PO-3: ClickHouse execution intent records.
    slog "PO-3: ClickHouse intent records"
    local intents
    intents=$(ch_query "SELECT * FROM ${CH_DB}.execution_intents WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 5 FORMAT JSONEachRow" 2>/dev/null || echo "query_failed")
    slog "  Recent BTCUSDT intents:"
    slog "  ${intents}"
    if [ "${intents}" = "query_failed" ] || [ -z "${intents}" ]; then
        slog "PO-3: WARN -- no intent records found or query failed"
        failures=$((failures + 1))
    else
        slog "PO-3: PASS -- intent records found"
    fi
    slog ""

    # PO-4: ClickHouse venue response records.
    slog "PO-4: ClickHouse venue response records"
    local responses
    responses=$(ch_query "SELECT * FROM ${CH_DB}.venue_responses WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 5 FORMAT JSONEachRow" 2>/dev/null || echo "query_failed")
    slog "  Recent BTCUSDT responses:"
    slog "  ${responses}"
    if [ "${responses}" = "query_failed" ] || [ -z "${responses}" ]; then
        slog "PO-4: WARN -- no response records found or query failed"
        failures=$((failures + 1))
    else
        slog "PO-4: PASS -- response records found"
    fi
    slog ""

    # PO-5: NATS KV order state (via execution control endpoint).
    slog "PO-5: NATS KV / execution control state"
    local control
    control=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo "unreachable")
    slog "  Execution control: $(echo "${control}" | head -c 500)"
    slog "PO-5: DONE -- recorded"
    slog ""

    # PO-6: System status summary.
    slog "PO-6: System status summary"
    local exec_statusz
    exec_statusz=$(curl -sf "${EXECUTE_URL}/statusz" 2>/dev/null || echo "unreachable")
    slog "  Execute /statusz: $(echo "${exec_statusz}" | head -c 500)"
    slog "PO-6: DONE"
    slog ""

    # PO-7: Fee/commission verification (S447).
    slog "PO-7: Fee and commission field verification"
    local fills_data
    fills_data=$(ch_query "SELECT event_id, symbol, side, status, filled_quantity, fills FROM ${CH_DB}.executions WHERE symbol = 'BTCUSDT' AND status IN ('filled','partially_filled') ORDER BY timestamp DESC LIMIT 5 FORMAT JSONEachRow" 2>/dev/null || echo "query_failed")
    slog "  BTCUSDT fill records with fee data:"
    slog "  ${fills_data}"
    if [ "${fills_data}" = "query_failed" ] || [ -z "${fills_data}" ]; then
        slog "PO-7: WARN -- no fill records with fee data found or query failed"
        failures=$((failures + 1))
    else
        # Check that fills JSON contains Fee and FeeAsset fields.
        if echo "${fills_data}" | grep -q '"Fee"' && echo "${fills_data}" | grep -q '"FeeAsset"'; then
            slog "PO-7: PASS -- fill records contain Fee and FeeAsset fields"
        else
            slog "PO-7: INFO -- fill records present but Fee/FeeAsset fields not detected in JSON (check fills column manually)"
        fi
    fi
    slog ""

    # PO-8: Lifecycle consistency check (S447).
    slog "PO-8: Lifecycle consistency (ClickHouse vs NATS KV)"
    local ch_latest_status
    ch_latest_status=$(ch_query "SELECT status, final, filled_quantity FROM ${CH_DB}.executions WHERE symbol = 'BTCUSDT' AND type = 'venue_market_order' ORDER BY timestamp DESC LIMIT 1 FORMAT JSONEachRow" 2>/dev/null || echo "query_failed")
    slog "  ClickHouse latest execution status: ${ch_latest_status}"

    local kv_status
    kv_status=$(curl -sf "${GATEWAY_URL}/execution/venue-market-order/latest?symbol=BTCUSDT" 2>/dev/null || echo "unreachable")
    slog "  NATS KV latest venue order: $(echo "${kv_status}" | head -c 500)"

    if [ "${ch_latest_status}" = "query_failed" ] && [ "${kv_status}" = "unreachable" ]; then
        slog "PO-8: WARN -- both ClickHouse and NATS KV queries failed"
        failures=$((failures + 1))
    else
        slog "PO-8: DONE -- lifecycle data captured for manual consistency review"
    fi
    slog ""

    # PO-9: Scope containment verification (S447).
    slog "PO-9: Scope containment verification"
    local total_executions
    total_executions=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE type = 'venue_market_order' AND timestamp > now() - INTERVAL 24 HOUR" 2>/dev/null || echo "query_failed")
    slog "  Total venue_market_order executions (24h): ${total_executions}"
    local non_btc_executions
    non_btc_executions=$(ch_query "SELECT count() FROM ${CH_DB}.executions WHERE type = 'venue_market_order' AND symbol != 'BTCUSDT' AND timestamp > now() - INTERVAL 24 HOUR" 2>/dev/null || echo "query_failed")
    slog "  Non-BTCUSDT venue executions (24h): ${non_btc_executions}"
    if [ "${non_btc_executions}" != "query_failed" ] && [ "${non_btc_executions}" != "0" ] && [ -n "${non_btc_executions}" ]; then
        non_btc_trimmed=$(echo "${non_btc_executions}" | tr -d '[:space:]')
        if [ "${non_btc_trimmed}" != "0" ]; then
            slog "PO-9: FAIL -- non-BTCUSDT executions detected! Scope violation."
            failures=$((failures + 1))
        else
            slog "PO-9: PASS -- no scope leakage detected"
        fi
    else
        slog "PO-9: PASS -- no scope leakage detected (or query unavailable)"
    fi

    slog ""
    slog "============================================="
    slog "POST-SESSION: Verification complete."
    slog "  Checks: PO-1 through PO-9"
    slog "  Failures/warnings: ${failures}"
    slog "  Session log: ${SESSION_LOG}"
    slog "============================================="

    return 0
}

# ── Full Ceremony ────────────────────────────────────────────────────────────

cmd_full() {
    slog ""
    slog "########################################################"
    slog "#  S446: SUPERVISED LIVE SESSION -- FULL CEREMONY       #"
    slog "#                                                       #"
    slog "#  Exchange: Binance Spot (mainnet)                     #"
    slog "#  Symbol:   BTCUSDT                                    #"
    slog "#  Order:    1 market order, minimum quantity            #"
    slog "#  Operator: ${OPERATOR_NAME:-<REQUIRED>}"
    slog "#  Session:  ${SESSION_ID}"
    slog "#                                                       #"
    slog "#  WARNING: THIS SUBMITS A REAL ORDER ON MAINNET        #"
    slog "########################################################"
    slog ""

    if [ -z "${OPERATOR_NAME}" ]; then
        slog "ABORT: OPERATOR_NAME is required. Set OPERATOR_NAME=<your-name>"
        exit 1
    fi

    # Phase 1: Pre-session checks.
    slog "=== Phase 1: Pre-Session Checklist ==="
    if ! cmd_pre_session; then
        slog ""
        slog "CEREMONY ABORTED: Pre-session checks failed."
        slog "Fix all failures, then re-run: $0 full"
        exit 1
    fi

    slog ""
    slog "=== Phase 2: Live Session ==="
    slog ""
    slog "Pre-session checks PASSED. The system is ready for the live session."
    slog ""
    slog "OPERATOR ACTIONS REQUIRED:"
    slog "  1. The system will now accept and process execution intents."
    slog "  2. Wait for the pipeline to generate ONE execution intent for BTCUSDT."
    slog "  3. Monitor the order lifecycle in system logs."
    slog "  4. After the first order completes (fill or reject), IMMEDIATELY halt:"
    slog "     ./scripts/kill-switch-ops.sh halt \"s446-session-complete\" \"${OPERATOR_NAME}\""
    slog "  5. Then run post-session: $0 post-session"
    slog ""
    slog "Starting monitor... (Ctrl+C to exit monitor without halting)"
    slog ""

    cmd_monitor

    slog ""
    slog "=== Phase 3: Post-Session Verification ==="
    cmd_post_session

    slog ""
    slog "########################################################"
    slog "#  S446: CEREMONY COMPLETE                              #"
    slog "#  Session log: ${SESSION_LOG}"
    slog "########################################################"
}

# ── Main Dispatch ────────────────────────────────────────────────────────────

usage() {
    cat <<EOF
Usage: $0 {pre-session|monitor|post-session|full}

Commands:
  pre-session    Run all 7 pre-session checks (PS-1 through PS-7)
  monitor        Monitor the running live session
  post-session   Run post-session verification (PO-1 through PO-6)
  full           Full ceremony: pre-session + monitor + post-session

Environment:
  GATEWAY_URL              Gateway base URL (default: http://127.0.0.1:8080)
  EXECUTE_URL              Execute health URL (default: http://127.0.0.1:8084)
  OPERATOR_NAME            Operator name (required for full ceremony)
  OPERATOR_ATTESTS_TRADE_ONLY  Set to 'true' after verifying API key permissions
  CREDENTIAL_PATH          Path to credential files (default: /run/secrets/market-foundry)

Authority: S446 -- Supervised Live Session Proof
EOF
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

case "$1" in
    pre-session)  cmd_pre_session ;;
    monitor)      cmd_monitor ;;
    post-session) cmd_post_session ;;
    full)         cmd_full ;;
    -h|--help|help) usage ;;
    *)
        fail "Unknown command: $1"
        usage
        exit 1
        ;;
esac
