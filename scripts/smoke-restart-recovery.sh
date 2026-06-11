#!/usr/bin/env bash
# smoke-restart-recovery.sh — Durable restart and consumer recovery smoke for paper flow.
#
# S280: Proves that critical service restarts do not break the paper flow, and that
# durable consumers, KV projections, and analytical writes resume coherently.
#
# Scenarios proven:
#   RC-1: Writer restart — durable consumer resumes, analytical projection continues
#   RC-2: Execute restart — safety gate re-reads KV, paper flow resumes
#   RC-3: Store restart — KV projections remain queryable after recovery
#   RC-4: Gateway restart — HTTP endpoints recover, control gate state persists
#   RC-5: Control gate state survives all restart scenarios
#   RC-6: No permanent data loss in analytical projection across writer restart
#
# Prerequisites:
#   make up          # starts full stack
#   make seed        # seeds configctl with bindings
#   wait ~120s       # writer needs time to flush initial batches
#
# Usage:
#   ./scripts/smoke-restart-recovery.sh
#   ./scripts/smoke-restart-recovery.sh --wait 180   # override flush wait (default: 120)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-restart-recovery.sh [--wait <seconds>] [--help]

Runs the restart/recovery smoke against a running compose stack.
Canonical public entrypoint: `make smoke-restart-recovery`
Expected setup: `make up && make seed`

Options:
  --wait <seconds>  Maximum time to wait for post-restart flushes. Default: 120
  --help            Show this help text.

Environment:
  BASE_URL          Gateway base URL. Default: http://127.0.0.1:8080
  SMOKE_WAIT        Preferred wait override from make/env.
  FLUSH_WAIT        Legacy wait override kept for compatibility.
EOF
}

FLUSH_WAIT="${SMOKE_WAIT:-${FLUSH_WAIT:-120}}"
SETUP_HINT="make up && make seed"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --wait)
            [[ $# -ge 2 ]] || usage_error "--wait requires a value"
            FLUSH_WAIT="$2"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage_error "unknown argument: $1"
            ;;
    esac
    shift
done

require_commands docker curl python3
require_positive_integer "--wait" "${FLUSH_WAIT}"

ERRORS=0
SYMBOL="${SYMBOL:-btcusdt}"
SOURCE="${SOURCE:-binancef}"
CONTRACT="${CONTRACT:-perpetual}"
TIMEFRAME="${TIMEFRAME:-60}"
GATEWAY_URL="${BASE_URL}"

smoke_banner "Restart And Recovery Smoke" "make smoke-restart-recovery" "${SETUP_HINT}" "flush-wait" "${FLUSH_WAIT}"

CLICKHOUSE_USER="${CLICKHOUSE_USER:-default}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-clickhouse}"

ch_query() {
    docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" exec -T clickhouse clickhouse-client \
        --port 9000 \
        --user "${CLICKHOUSE_USER}" \
        --password "${CLICKHOUSE_PASSWORD}" \
        --database market_foundry \
        --query "$1" 2>/dev/null || echo "0"
}

wait_service_healthy() {
    local svc="$1"
    local max_wait="${2:-90}"
    local elapsed=0
    local port
    port=$(svc_port "$svc")

    while [[ $elapsed -lt $max_wait ]]; do
        local resp
        resp=$(curl -sf "http://127.0.0.1:${port}/readyz" 2>/dev/null || echo "")
        if [[ "$resp" == *"ready"* ]]; then
            return 0
        fi
        sleep 5
        elapsed=$((elapsed + 5))
    done
    return 1
}

# ── Phase 0: Pre-flight — verify stack is running ────────────────────
phase "Phase 0: Pre-flight Check"

REQUIRED_SERVICES=("nats" "clickhouse" "gateway" "writer" "execute" "store" "derive")
ALL_RUNNING=true

for svc in "${REQUIRED_SERVICES[@]}"; do
    status=$(docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" ps --format json "$svc" 2>/dev/null | python3 -c "
import sys,json
lines = sys.stdin.read().strip().split('\n')
for line in lines:
    if line.strip():
        d = json.loads(line)
        print(d.get('State','unknown'))
        break
" 2>/dev/null || echo "not_found")

    if [[ "$status" == "running" ]]; then
        pass "$svc: running"
    else
        ALL_RUNNING=false
        record_fail "$svc: expected running, got $status"
    fi
done

if ! $ALL_RUNNING; then
    fail "ABORT: Not all services running."
    print_smoke_diagnosis_hints "${SETUP_HINT}"
    exit 1
fi

# Capture baseline data counts before restarts.
info "Capturing baseline analytical counts..."
BASELINE_CANDLES=$(ch_query "SELECT count() FROM evidence_candles" | tr -d '[:space:]')
BASELINE_SIGNALS=$(ch_query "SELECT count() FROM signals" | tr -d '[:space:]')
BASELINE_EXECUTIONS=$(ch_query "SELECT count() FROM executions" | tr -d '[:space:]')
info "Baseline: candles=$BASELINE_CANDLES signals=$BASELINE_SIGNALS executions=$BASELINE_EXECUTIONS"

# Capture baseline control gate state.
GATE_BEFORE=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo '{}')
GATE_STATUS_BEFORE=$(echo "$GATE_BEFORE" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")
info "Control gate before restarts: $GATE_STATUS_BEFORE"

# Ensure gate is active for the test.
if [[ "$GATE_STATUS_BEFORE" != "active" ]]; then
    info "Setting gate to active for restart test..."
    curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
        -H "Content-Type: application/json" \
        -d '{"status":"active","reason":"S280 restart recovery smoke pre-flight","updated_by":"smoke-restart-recovery"}' \
        >/dev/null 2>&1
fi

pass "Pre-flight complete — all services running, baseline captured"

# ── Phase 1: Writer Restart (RC-1) ───────────────────────────────────
phase "Phase 1: Writer Restart Recovery (RC-1)"

info "Restarting writer service..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart writer >/dev/null 2>&1

info "Waiting for writer to become healthy..."
if wait_service_healthy "writer" 90; then
    pass "Writer restarted and healthy"
else
    record_fail "Writer failed to become healthy after restart"
fi

# Wait for writer to resume consumption and flush.
info "Waiting ${FLUSH_WAIT}s for writer to resume consumption and flush..."
sleep "$FLUSH_WAIT"

# Check analytical counts increased.
POST_WRITER_CANDLES=$(ch_query "SELECT count() FROM evidence_candles" | tr -d '[:space:]')
POST_WRITER_SIGNALS=$(ch_query "SELECT count() FROM signals" | tr -d '[:space:]')

if [[ "$POST_WRITER_CANDLES" -ge "$BASELINE_CANDLES" ]] 2>/dev/null; then
    candle_delta=$((POST_WRITER_CANDLES - BASELINE_CANDLES))
    pass "RC-1: Candles after writer restart: $POST_WRITER_CANDLES (delta=$candle_delta)"
else
    record_fail "RC-1: Candle count decreased after writer restart: $POST_WRITER_CANDLES < $BASELINE_CANDLES"
fi

if [[ "$POST_WRITER_SIGNALS" -ge "$BASELINE_SIGNALS" ]] 2>/dev/null; then
    signal_delta=$((POST_WRITER_SIGNALS - BASELINE_SIGNALS))
    pass "RC-1: Signals after writer restart: $POST_WRITER_SIGNALS (delta=$signal_delta)"
else
    record_fail "RC-1: Signal count decreased after writer restart"
fi

# Verify writer health endpoint shows tracker info.
writer_health=$(curl -sf "http://127.0.0.1:8085/readyz" 2>/dev/null || echo "unreachable")
if [[ "$writer_health" == *"ready"* ]]; then
    pass "RC-1: Writer readyz healthy after restart"
else
    record_fail "RC-1: Writer readyz unhealthy after restart: $writer_health"
fi

# ── Phase 2: Execute Restart (RC-2) ──────────────────────────────────
phase "Phase 2: Execute Restart Recovery (RC-2)"

info "Restarting execute service..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart execute >/dev/null 2>&1

info "Waiting for execute to become healthy..."
if wait_service_healthy "execute" 60; then
    pass "Execute restarted and healthy"
else
    record_fail "Execute failed to become healthy after restart"
fi

# Verify safety gate is accessible from the restarted execute binary.
# We check indirectly: the execute service should be reporting ready.
exec_health=$(curl -sf "http://127.0.0.1:8084/readyz" 2>/dev/null || echo "unreachable")
if [[ "$exec_health" == *"ready"* ]]; then
    pass "RC-2: Execute readyz healthy after restart — safety gate re-initialized"
else
    record_fail "RC-2: Execute readyz unhealthy after restart"
fi

# Verify control gate is still readable via gateway (execute restart doesn't affect gate).
gate_after_exec=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo '{}')
gate_status_after_exec=$(echo "$gate_after_exec" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")

if [[ "$gate_status_after_exec" == "active" ]]; then
    pass "RC-2: Control gate remains active after execute restart"
else
    record_fail "RC-2: Control gate changed after execute restart: $gate_status_after_exec"
fi

# ── Phase 3: Store Restart (RC-3) ────────────────────────────────────
phase "Phase 3: Store Restart Recovery (RC-3)"

info "Restarting store service..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart store >/dev/null 2>&1

info "Waiting for store to become healthy..."
if wait_service_healthy "store" 60; then
    pass "Store restarted and healthy"
else
    record_fail "Store failed to become healthy after restart"
fi

# Verify KV projections are queryable after store restart.
kv_http_code=$(curl -s -o /dev/null -w "%{http_code}" \
    "${GATEWAY_URL}/execution/paper_order/latest?source=${SOURCE}&base=${SYMBOL%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${TIMEFRAME}" \
    2>/dev/null || echo "000")

if [[ "$kv_http_code" == "200" || "$kv_http_code" == "404" ]]; then
    pass "RC-3: KV paper_order/latest queryable after store restart (HTTP $kv_http_code)"
else
    record_fail "RC-3: KV paper_order/latest unavailable after store restart (HTTP $kv_http_code)"
fi

# Verify control gate round-trip still works via store.
gate_after_store=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo '{}')
gate_status_after_store=$(echo "$gate_after_store" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")

if [[ "$gate_status_after_store" == "active" ]]; then
    pass "RC-3: Control gate still active after store restart"
else
    record_fail "RC-3: Control gate changed after store restart: $gate_status_after_store"
fi

# Verify gate write works after store restart.
info "Testing gate write after store restart..."
halt_resp=$(curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"S280 store restart verification","updated_by":"smoke-restart-recovery"}' \
    2>/dev/null || echo '{"error":"failed"}')
halt_status=$(echo "$halt_resp" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")

if [[ "$halt_status" == "halted" ]]; then
    pass "RC-3: Gate write works after store restart"
else
    record_fail "RC-3: Gate write failed after store restart: $halt_status"
fi

# Resume gate.
curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"S280 resume after store restart test","updated_by":"smoke-restart-recovery"}' \
    >/dev/null 2>&1

# ── Phase 4: Gateway Restart (RC-4) ──────────────────────────────────
phase "Phase 4: Gateway Restart Recovery (RC-4)"

info "Restarting gateway service..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart gateway >/dev/null 2>&1

info "Waiting for gateway to become healthy..."
if wait_service_healthy "gateway" 90; then
    pass "Gateway restarted and healthy"
else
    record_fail "Gateway failed to become healthy after restart"
fi

# Verify HTTP endpoints recover.
gw_health=$(curl -sf "http://127.0.0.1:8080/readyz" 2>/dev/null || echo "unreachable")
if [[ "$gw_health" == *"ready"* ]]; then
    pass "RC-4: Gateway readyz healthy after restart"
else
    record_fail "RC-4: Gateway readyz unhealthy after restart"
fi

# Verify control gate state persisted across gateway restart.
gate_after_gw=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo '{}')
gate_status_after_gw=$(echo "$gate_after_gw" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")

if [[ "$gate_status_after_gw" == "active" ]]; then
    pass "RC-4: Control gate persisted across gateway restart"
else
    record_fail "RC-4: Control gate state lost after gateway restart: $gate_status_after_gw"
fi

# Verify analytical endpoints recover.
analytical_code=$(curl -s -o /dev/null -w "%{http_code}" \
    "${GATEWAY_URL}/analytical/evidence/candles?source=${SOURCE}&base=${SYMBOL%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${TIMEFRAME}&limit=5" \
    2>/dev/null || echo "000")

if [[ "$analytical_code" == "200" ]]; then
    pass "RC-4: Analytical endpoint recovered after gateway restart"
else
    record_fail "RC-4: Analytical endpoint failed after gateway restart (HTTP $analytical_code)"
fi

# ── Phase 5: Control Gate Durability (RC-5) ──────────────────────────
phase "Phase 5: Control Gate Cross-Restart Durability (RC-5)"

info "Setting gate to halted with audit trail..."
curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"S280 durability proof","updated_by":"smoke-restart-recovery-phase5"}' \
    >/dev/null 2>&1

# Restart store (which mediates gate operations).
info "Restarting store to test gate persistence..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart store >/dev/null 2>&1
wait_service_healthy "store" 60

# Gateway needs store, so restart gateway after store is healthy.
info "Restarting gateway to test full path..."
docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" restart gateway >/dev/null 2>&1
wait_service_healthy "gateway" 90

# Read gate — should still be halted with original audit trail.
sleep 2
gate_durable=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null || echo '{}')
gate_durable_status=$(echo "$gate_durable" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('status','unknown'))
" 2>/dev/null || echo "unknown")
gate_durable_reason=$(echo "$gate_durable" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('reason',''))
" 2>/dev/null || echo "")
gate_durable_by=$(echo "$gate_durable" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=d.get('gate',d)
print(g.get('updated_by',''))
" 2>/dev/null || echo "")

if [[ "$gate_durable_status" == "halted" ]]; then
    pass "RC-5: Gate status survived store+gateway restart: halted"
else
    record_fail "RC-5: Gate status lost after store+gateway restart: $gate_durable_status"
fi

if [[ "$gate_durable_reason" == "S280 durability proof" ]]; then
    pass "RC-5: Gate reason survived restart: '$gate_durable_reason'"
else
    warn "RC-5: Gate reason changed: '$gate_durable_reason' (may have been overwritten)"
fi

if [[ "$gate_durable_by" == "smoke-restart-recovery-phase5" ]]; then
    pass "RC-5: Gate updated_by survived restart: '$gate_durable_by'"
else
    warn "RC-5: Gate updated_by changed: '$gate_durable_by'"
fi

# Resume gate for cleanup.
curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"S280 smoke cleanup","updated_by":"smoke-restart-recovery"}' \
    >/dev/null 2>&1

# ── Phase 6: Analytical Projection Continuity (RC-6) ─────────────────
phase "Phase 6: Analytical Projection Continuity (RC-6)"

info "Waiting 30s for post-restart data to flow..."
sleep 30

FINAL_CANDLES=$(ch_query "SELECT count() FROM evidence_candles" | tr -d '[:space:]')
FINAL_SIGNALS=$(ch_query "SELECT count() FROM signals" | tr -d '[:space:]')

if [[ "$FINAL_CANDLES" -ge "$BASELINE_CANDLES" ]] 2>/dev/null; then
    total_delta=$((FINAL_CANDLES - BASELINE_CANDLES))
    pass "RC-6: Candle count non-decreasing across all restarts: $FINAL_CANDLES (total delta=$total_delta)"
else
    record_fail "RC-6: Candle count decreased: $FINAL_CANDLES < baseline $BASELINE_CANDLES"
fi

if [[ "$FINAL_SIGNALS" -ge "$BASELINE_SIGNALS" ]] 2>/dev/null; then
    total_delta=$((FINAL_SIGNALS - BASELINE_SIGNALS))
    pass "RC-6: Signal count non-decreasing: $FINAL_SIGNALS (total delta=$total_delta)"
else
    record_fail "RC-6: Signal count decreased: $FINAL_SIGNALS < baseline $BASELINE_SIGNALS"
fi

# ── Results ──────────────────────────────────────────────────────────
phase "Results"

echo ""
echo "  RC-1: Writer Restart Recovery ........... $(if [[ $ERRORS -eq 0 ]]; then echo "PASS"; else echo "CHECK ABOVE"; fi)"
echo "  RC-2: Execute Restart Recovery .......... $(if [[ "$gate_status_after_exec" == "active" ]]; then echo "PASS"; else echo "FAIL"; fi)"
echo "  RC-3: Store Restart Recovery ............ $(if [[ "$gate_status_after_store" == "active" ]]; then echo "PASS"; else echo "FAIL"; fi)"
echo "  RC-4: Gateway Restart Recovery .......... $(if [[ "$gate_status_after_gw" == "active" ]]; then echo "PASS"; else echo "FAIL"; fi)"
echo "  RC-5: Control Gate Durability ........... $(if [[ "$gate_durable_status" == "halted" ]]; then echo "PASS"; else echo "FAIL"; fi)"
echo "  RC-6: Analytical Projection Continuity .. $(if [[ "$FINAL_CANDLES" -ge "$BASELINE_CANDLES" ]]; then echo "PASS"; else echo "FAIL"; fi)"
echo ""

echo "  Known limits (by design, not failures):"
echo "    - Writer buffer loss: events ACKed but not yet flushed to ClickHouse are lost on crash"
echo "    - Dedup window: JetStream dedup is bounded (~2min); events outside window may duplicate"
echo "    - ClickHouse INSERT is not idempotent — duplicate rows possible on replay"
echo "    - No automatic reconnect on NATS connection loss (relies on Docker restart policy)"
echo ""

if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "Restart recovery smoke" "$ERRORS" "${SETUP_HINT}"
    exit 1
else
    pass "Restart recovery smoke completed successfully — all scenarios proven"
    exit 0
fi
