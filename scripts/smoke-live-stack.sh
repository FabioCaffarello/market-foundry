#!/usr/bin/env bash
# smoke-live-stack.sh — Canonical live stack smoke for the Live Stack Integration Wave.
#
# Unified operational smoke that validates the full venue path with a live stack:
#   1. Stack readiness (NATS, ClickHouse, writer, gateway)
#   2. NATS stream and consumer health
#   3. ClickHouse analytical data presence
#   4. Gateway HTTP composite surface queries
#   5. Disposition and funnel aggregation surface
#   6. Structural Go test regression gate
#   7. Kill-switch control path validation (S335)
#
# This smoke is the canonical, single-command proof for the Live Stack
# Integration Wave (S332–S336). It exercises:
#   venue path → persistence → gateway read → control path
#
# Prerequisites:
#   make up && make seed   # full stack running with configctl seeded
#
# Usage:
#   ./scripts/smoke-live-stack.sh
#   ./scripts/smoke-live-stack.sh --wait 120   # override flush wait
#   SMOKE_WAIT=180 make smoke-live-stack       # via Makefile
#
# Guard rails:
#   - No pipeline expansion; verifies existing paths only.
#   - No dashboard or alerting; stdout PASS/FAIL only.
#   - No manual steps beyond `make up && make seed`.
#   - Single script, single exit code.
#   - Kill-switch test restores active state on exit (safe for pipelines).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
compose() {
    docker compose -f "${COMPOSE_FILE}" "$@"
}

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-live-stack.sh [--wait <seconds>] [--help]

S318: Live stack smoke and gateway verification.
Validates that the full live stack exercises venue path, analytical persistence,
and the gateway composite read surface in a single reproducible flow.

Options:
  --wait <seconds>  Maximum time to wait for writer flushes. Default: 60
  --help            Show this help text.

Environment:
  BASE_URL          Gateway base URL. Default: http://127.0.0.1:8080
  CLICKHOUSE_DATABASE  ClickHouse database name. Default: market_foundry
EOF
}

FLUSH_WAIT="${SMOKE_WAIT:-${FLUSH_WAIT:-60}}"
CLICKHOUSE_DATABASE="${CLICKHOUSE_DATABASE:-market_foundry}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --wait)
            [[ $# -ge 2 ]] || usage_error "--wait requires a value"
            FLUSH_WAIT="$2"
            shift
            ;;
        -h|--help) usage; exit 0 ;;
        *) usage_error "unknown argument: $1" ;;
    esac
    shift
done

require_commands docker curl python3

ERRORS=0

smoke_banner "Live Stack Smoke (S335 canonical)" "make smoke-live-stack" "make up && make seed" "flush-wait" "${FLUSH_WAIT}"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Stack Readiness"
# ══════════════════════════════════════════════════════════════════════

# 1a. ClickHouse
info "Checking ClickHouse health..."
CH_RESULT=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse --query "SELECT 1" 2>/dev/null || echo "")
if [[ "$CH_RESULT" == "1" ]]; then
    pass "ClickHouse is healthy"
else
    die "ClickHouse unreachable — run: make up"
fi

# 1b. Writer
info "Checking writer readiness..."
WRITER_READY=$(compose exec -T writer wget -q -O - http://127.0.0.1:8085/readyz 2>/dev/null || echo "")
WRITER_STATUS=$(echo "$WRITER_READY" | json_field "status")
if [[ "$WRITER_STATUS" == "ready" ]]; then
    pass "Writer is ready"
else
    die "Writer not ready (status=${WRITER_STATUS:-unreachable}) — run: make logs SERVICE=writer"
fi

# 1c. Gateway
info "Checking gateway readiness..."
GW_CODE=$(http_code "${BASE_URL}/readyz")
if [[ "$GW_CODE" == "200" ]]; then
    pass "Gateway is ready"
else
    die "Gateway not ready (HTTP ${GW_CODE}) — run: make logs SERVICE=gateway"
fi

# 1d. NATS
info "Checking NATS health..."
NATS_CODE=$(compose exec -T nats wget -q -O - http://127.0.0.1:8222/healthz 2>/dev/null && echo "ok" || echo "")
if [[ "$NATS_CODE" == "ok" ]]; then
    pass "NATS is healthy"
else
    die "NATS unreachable — run: make up"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: NATS Stream and Consumer Health"
# ══════════════════════════════════════════════════════════════════════

info "Checking EXECUTION_FILL_EVENTS stream..."
FILL_STREAM=$(compose exec -T nats nats stream info EXECUTION_FILL_EVENTS --json 2>/dev/null || echo "")
if [[ -n "$FILL_STREAM" ]] && echo "$FILL_STREAM" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['config']['name'])" 2>/dev/null | grep -q "EXECUTION_FILL_EVENTS"; then
    FILL_MSG_COUNT=$(echo "$FILL_STREAM" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['state']['messages'])" 2>/dev/null || echo "0")
    pass "EXECUTION_FILL_EVENTS stream exists (messages=${FILL_MSG_COUNT})"
else
    warn "EXECUTION_FILL_EVENTS stream not found — may be created on first fill"
fi

info "Checking writer-execution-venue-fill consumer..."
FILL_CONSUMER=$(compose exec -T nats nats consumer info EXECUTION_FILL_EVENTS writer-execution-venue-fill --json 2>/dev/null || echo "")
if [[ -n "$FILL_CONSUMER" ]] && echo "$FILL_CONSUMER" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['config']['durable_name'])" 2>/dev/null | grep -q "writer-execution-venue-fill"; then
    pass "writer-execution-venue-fill consumer registered"
else
    warn "writer-execution-venue-fill consumer not registered yet"
fi

# Also check paper execution stream for data flow evidence.
info "Checking EXECUTION_EVENTS stream..."
EXEC_STREAM=$(compose exec -T nats nats stream info EXECUTION_EVENTS --json 2>/dev/null || echo "")
if [[ -n "$EXEC_STREAM" ]] && echo "$EXEC_STREAM" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['config']['name'])" 2>/dev/null | grep -q "EXECUTION_EVENTS"; then
    EXEC_MSG_COUNT=$(echo "$EXEC_STREAM" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['state']['messages'])" 2>/dev/null || echo "0")
    pass "EXECUTION_EVENTS stream exists (messages=${EXEC_MSG_COUNT})"
else
    warn "EXECUTION_EVENTS stream not found"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: ClickHouse Analytical Data"
# ══════════════════════════════════════════════════════════════════════

ch_query() {
    compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse --database "${CLICKHOUSE_DATABASE}" --query "$1" 2>/dev/null || echo "0"
}

# 3a. Executions table
info "Checking executions table..."
EXEC_COUNT=$(ch_query "SELECT count() FROM executions")
if [[ "$EXEC_COUNT" -gt 0 ]]; then
    pass "executions table: ${EXEC_COUNT} rows"
else
    warn "executions table is empty"
fi

# 3b. Filled rows specifically
FILLED_COUNT=$(ch_query "SELECT count() FROM executions WHERE status = 'filled'")
if [[ "$FILLED_COUNT" -gt 0 ]]; then
    pass "venue-filled rows: ${FILLED_COUNT}"
else
    warn "no venue-filled rows yet"
fi

# 3c. Check other analytical tables for data presence.
for TABLE in evidence signals decisions strategies risk_assessments; do
    TABLE_COUNT=$(ch_query "SELECT count() FROM ${TABLE}")
    if [[ "$TABLE_COUNT" -gt 0 ]]; then
        pass "${TABLE}: ${TABLE_COUNT} rows"
    else
        warn "${TABLE}: empty"
    fi
done

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Gateway Composite HTTP Surface"
# ══════════════════════════════════════════════════════════════════════

# 4a. Composite chains endpoint
info "Querying composite chains..."
CHAINS_URL="${BASE_URL}/analytical/composite/chains?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60&limit=5"
CHAINS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$CHAINS_URL")

if [[ "$CHAINS_CODE" == "200" ]]; then
    pass "GET /analytical/composite/chains → 200"

    CHAINS_RESPONSE=$(curl -s "$CHAINS_URL")
    CHAIN_COUNT=$(echo "$CHAINS_RESPONSE" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    chains = d.get('chains', [])
    print(len(chains))
except:
    print(0)
" 2>/dev/null || echo "0")

    if [[ "$CHAIN_COUNT" -gt 0 ]]; then
        pass "Composite surface returns ${CHAIN_COUNT} chain(s)"

        # Check for execution stage presence.
        HAS_EXEC=$(echo "$CHAINS_RESPONSE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for c in d.get('chains', []):
    if c.get('execution') is not None:
        status = c['execution'].get('status', '')
        corr = c.get('correlation_id', '')
        print(f'correlation_id={corr} status={status}')
        sys.exit(0)
print('none')
" 2>/dev/null || echo "none")

        if [[ "$HAS_EXEC" != "none" ]]; then
            pass "Chain with execution stage: ${HAS_EXEC}"
        else
            warn "No chain contains an execution stage yet"
        fi
    else
        warn "Composite surface returned 0 chains (data may still be propagating)"
    fi
elif [[ "$CHAINS_CODE" == "503" ]]; then
    record_fail "Composite endpoint returns 503 — ClickHouse not connected to gateway"
else
    record_fail "Composite endpoint returns unexpected HTTP ${CHAINS_CODE}"
fi

# 4b. Pipeline funnel endpoint
info "Querying pipeline funnel..."
FUNNEL_URL="${BASE_URL}/analytical/composite/funnel?type=paper_order&source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
FUNNEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FUNNEL_URL")

if [[ "$FUNNEL_CODE" == "200" ]]; then
    FUNNEL_RESPONSE=$(curl -s "$FUNNEL_URL")
    STAGE_COUNT=$(echo "$FUNNEL_RESPONSE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(len(d.get('stages', [])))
" 2>/dev/null || echo "0")
    pass "GET /analytical/composite/funnel → 200 (${STAGE_COUNT} stages)"
else
    record_fail "Pipeline funnel endpoint returns HTTP ${FUNNEL_CODE}"
fi

# 4c. Disposition breakdown endpoint
info "Querying disposition breakdown..."
DISP_URL="${BASE_URL}/analytical/composite/dispositions?type=paper_order&source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
DISP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$DISP_URL")

if [[ "$DISP_CODE" == "200" ]]; then
    DISP_RESPONSE=$(curl -s "$DISP_URL")
    TOTAL=$(echo "$DISP_RESPONSE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d.get('total', 0))
" 2>/dev/null || echo "0")
    pass "GET /analytical/composite/dispositions → 200 (total=${TOTAL})"
else
    record_fail "Disposition breakdown endpoint returns HTTP ${DISP_CODE}"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Single-Family Analytical Endpoints"
# ══════════════════════════════════════════════════════════════════════

ANALYTICAL_ENDPOINTS=(
    "/analytical/evidence/candles?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
    "/analytical/signal/history?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
    "/analytical/decision/history?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
    "/analytical/strategy/history?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
    "/analytical/risk/history?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
    "/analytical/execution/history?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60"
)

for ENDPOINT in "${ANALYTICAL_ENDPOINTS[@]}"; do
    FULL_URL="${BASE_URL}${ENDPOINT}"
    EP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FULL_URL")
    EP_PATH=$(echo "$ENDPOINT" | cut -d'?' -f1)

    if [[ "$EP_CODE" == "200" ]]; then
        pass "GET ${EP_PATH} → 200"
    elif [[ "$EP_CODE" == "503" ]]; then
        record_fail "GET ${EP_PATH} → 503 (ClickHouse not wired)"
    else
        record_fail "GET ${EP_PATH} → unexpected HTTP ${EP_CODE}"
    fi
done

# ══════════════════════════════════════════════════════════════════════
phase "Phase 6: Structural Go Test Gate"
# ══════════════════════════════════════════════════════════════════════

info "Running S317 structural round-trip tests..."
S317_TESTS="TestS317_VenueFill_RowMapperCompatibility|TestS317_VenueFill_CompositeChainReadability|TestS317_VenueFill_DryRun"
if (cd "$PROJECT_ROOT" && go test -count=1 -run "$S317_TESTS" ./internal/application/execution/... 2>/dev/null); then
    pass "S317 structural round-trip tests pass"
else
    record_fail "S317 structural tests failed"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 7: Kill-Switch Control Path (S335)"
# ══════════════════════════════════════════════════════════════════════
#
# Validates the execution control gate (kill-switch) via the gateway HTTP
# surface against live NATS KV. Tests the halt → query → resume cycle.
#
# Safety: always restores the gate to "active" on exit, even on failure.

CONTROL_URL="${BASE_URL}/execution/control"
KS_ERRORS=0

# Trap to ensure gate is restored to active if script exits mid-phase.
ks_restore_active() {
    curl -s -X PUT "${CONTROL_URL}" \
        -H "Content-Type: application/json" \
        -d '{"status":"active","reason":"smoke-cleanup","updated_by":"smoke-live-stack"}' \
        >/dev/null 2>&1 || true
}
trap ks_restore_active EXIT

# 7a. Query current gate state — must succeed.
info "Querying execution control gate..."
KS_GET_CODE=$(curl -s -o /tmp/ks_get.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$KS_GET_CODE" == "200" ]]; then
    KS_CURRENT=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_get.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    pass "GET /execution/control → 200 (status=${KS_CURRENT})"
else
    record_fail "GET /execution/control → HTTP ${KS_GET_CODE} (control surface unreachable)"
    KS_ERRORS=1
fi

if [[ $KS_ERRORS -eq 0 ]]; then
    # 7b. Halt gate — set to halted.
    info "Setting gate to halted..."
    KS_HALT_CODE=$(curl -s -o /tmp/ks_halt.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
        -H "Content-Type: application/json" \
        -d '{"status":"halted","reason":"smoke-s335-kill-switch-proof","updated_by":"smoke-live-stack"}')
    if [[ "$KS_HALT_CODE" == "200" ]]; then
        KS_HALT_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_halt.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
        if [[ "$KS_HALT_STATUS" == "halted" ]]; then
            pass "PUT /execution/control → halted (kill-switch engaged)"
        else
            record_fail "PUT halted returned 200 but status=${KS_HALT_STATUS} (expected halted)"
        fi
    else
        record_fail "PUT /execution/control (halt) → HTTP ${KS_HALT_CODE}"
    fi

    # 7c. Confirm gate reads as halted.
    info "Confirming gate reads halted..."
    KS_CONFIRM_CODE=$(curl -s -o /tmp/ks_confirm.json -w "%{http_code}" "${CONTROL_URL}")
    if [[ "$KS_CONFIRM_CODE" == "200" ]]; then
        KS_CONFIRM_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_confirm.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
        KS_CONFIRM_REASON=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_confirm.json')).get('gate',{}).get('reason',''))" 2>/dev/null || echo "")
        if [[ "$KS_CONFIRM_STATUS" == "halted" ]]; then
            pass "Gate confirmed halted (reason=${KS_CONFIRM_REASON})"
        else
            record_fail "Gate read-after-halt returned status=${KS_CONFIRM_STATUS} (expected halted)"
        fi
    else
        record_fail "GET /execution/control (confirm halt) → HTTP ${KS_CONFIRM_CODE}"
    fi

    # 7d. Resume gate — set to active.
    info "Resuming gate to active..."
    KS_RESUME_CODE=$(curl -s -o /tmp/ks_resume.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
        -H "Content-Type: application/json" \
        -d '{"status":"active","reason":"smoke-s335-resume-after-proof","updated_by":"smoke-live-stack"}')
    if [[ "$KS_RESUME_CODE" == "200" ]]; then
        KS_RESUME_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_resume.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
        if [[ "$KS_RESUME_STATUS" == "active" ]]; then
            pass "PUT /execution/control → active (kill-switch disengaged)"
        else
            record_fail "PUT active returned 200 but status=${KS_RESUME_STATUS} (expected active)"
        fi
    else
        record_fail "PUT /execution/control (resume) → HTTP ${KS_RESUME_CODE}"
    fi

    # 7e. Confirm gate reads as active after resume.
    info "Confirming gate reads active..."
    KS_FINAL_CODE=$(curl -s -o /tmp/ks_final.json -w "%{http_code}" "${CONTROL_URL}")
    if [[ "$KS_FINAL_CODE" == "200" ]]; then
        KS_FINAL_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_final.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
        if [[ "$KS_FINAL_STATUS" == "active" ]]; then
            pass "Gate confirmed active after resume cycle"
        else
            record_fail "Gate read-after-resume returned status=${KS_FINAL_STATUS} (expected active)"
        fi
    else
        record_fail "GET /execution/control (confirm resume) → HTTP ${KS_FINAL_CODE}"
    fi

    # 7f. Verify audit fields survive round-trip.
    KS_UPDATED_BY=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_resume.json')).get('gate',{}).get('updated_by',''))" 2>/dev/null || echo "")
    KS_UPDATED_AT=$(python3 -c "import sys,json; print(json.load(open('/tmp/ks_resume.json')).get('gate',{}).get('updated_at',''))" 2>/dev/null || echo "")
    if [[ -n "$KS_UPDATED_BY" && -n "$KS_UPDATED_AT" ]]; then
        pass "Audit fields preserved (updated_by=${KS_UPDATED_BY})"
    else
        record_fail "Audit fields missing after round-trip"
    fi
fi

# Clean up temp files.
rm -f /tmp/ks_get.json /tmp/ks_halt.json /tmp/ks_confirm.json /tmp/ks_resume.json /tmp/ks_final.json

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════

echo ""
if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "Live stack smoke" "$ERRORS" "make up && make seed"
    exit 1
fi

pass "Live stack smoke completed (S335 canonical surface)"
info "Stack validated: NATS streams | ClickHouse tables | Gateway composite | Analytical endpoints | Kill-switch control path"
info "Full path: venue adapter → NATS → writer → ClickHouse → gateway HTTP → control gate"
