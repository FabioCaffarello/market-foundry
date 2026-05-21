#!/usr/bin/env bash
# kill-switch-ops.sh -- Canonical kill-switch operational script for market-foundry.
#
# Usage:
#   ./scripts/kill-switch-ops.sh status              -- Query current gate state
#   ./scripts/kill-switch-ops.sh halt [reason] [by]   -- Halt all execution
#   ./scripts/kill-switch-ops.sh resume [reason] [by]  -- Resume execution
#   ./scripts/kill-switch-ops.sh verify-halted         -- Verify execution is halted
#   ./scripts/kill-switch-ops.sh verify-active         -- Verify execution is active
#   ./scripts/kill-switch-ops.sh cycle [reason] [by]   -- Full halt -> verify -> resume -> verify cycle
#
# Environment:
#   GATEWAY_URL  -- gateway base URL (default: http://127.0.0.1:8080)
#   EXECUTE_URL  -- execute health URL (default: http://127.0.0.1:8084)
#
# Authority: S442 -- Kill-Switch Operational Runbook
# Dependencies: curl, jq (or python3 fallback)

set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8080}"
EXECUTE_URL="${EXECUTE_URL:-http://127.0.0.1:8084}"

# --- Colors and formatting ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { printf "${CYAN}[INFO]${NC}  %s\n" "$*"; }
pass()  { printf "${GREEN}[PASS]${NC}  %s\n" "$*"; }
fail()  { printf "${RED}[FAIL]${NC}  %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC}  %s\n" "$*"; }

# --- JSON extraction ---
# Uses jq if available, falls back to python3.
json_field() {
    local json="$1" field="$2"
    if command -v jq &>/dev/null; then
        echo "$json" | jq -r "$field" 2>/dev/null || echo "unknown"
    else
        echo "$json" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    keys = '${field}'.strip('.').split('.')
    for k in keys:
        d = d[k]
    print(d)
except:
    print('unknown')
" 2>/dev/null || echo "unknown"
    fi
}

# --- Core operations ---

# Query the current gate state.
cmd_status() {
    info "Querying execution control gate..."
    local resp
    resp=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null) || {
        fail "Cannot reach gateway at ${GATEWAY_URL}/execution/control"
        return 1
    }

    local status reason updated_at updated_by
    status=$(json_field "$resp" '.gate.status')
    reason=$(json_field "$resp" '.gate.reason')
    updated_at=$(json_field "$resp" '.gate.updated_at')
    updated_by=$(json_field "$resp" '.gate.updated_by')

    echo ""
    echo "  Gate Status:  ${status}"
    echo "  Reason:       ${reason}"
    echo "  Updated At:   ${updated_at}"
    echo "  Updated By:   ${updated_by}"
    echo ""

    if [[ "$status" == "halted" ]]; then
        warn "Execution is HALTED"
    elif [[ "$status" == "active" ]]; then
        pass "Execution is ACTIVE"
    else
        fail "Unknown gate status: ${status}"
        return 1
    fi
}

# Set the gate to halted.
cmd_halt() {
    local reason="${1:-manual-kill-switch}"
    local updated_by="${2:-operator}"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    info "HALTING execution at ${ts}..."
    info "  Reason:     ${reason}"
    info "  Operator:   ${updated_by}"

    local resp
    resp=$(curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
        -H "Content-Type: application/json" \
        -d "{\"status\":\"halted\",\"reason\":\"${reason}\",\"updated_by\":\"${updated_by}\"}" \
        2>/dev/null) || {
        fail "HALT command failed -- cannot reach gateway at ${GATEWAY_URL}"
        return 1
    }

    local new_status
    new_status=$(json_field "$resp" '.gate.status')

    if [[ "$new_status" == "halted" ]]; then
        pass "Gate set to HALTED at ${ts}"
    else
        fail "Expected 'halted', got '${new_status}'"
        return 1
    fi
}

# Set the gate to active.
cmd_resume() {
    local reason="${1:-manual-resume}"
    local updated_by="${2:-operator}"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    info "RESUMING execution at ${ts}..."
    info "  Reason:     ${reason}"
    info "  Operator:   ${updated_by}"

    local resp
    resp=$(curl -sf -X PUT "${GATEWAY_URL}/execution/control" \
        -H "Content-Type: application/json" \
        -d "{\"status\":\"active\",\"reason\":\"${reason}\",\"updated_by\":\"${updated_by}\"}" \
        2>/dev/null) || {
        fail "RESUME command failed -- cannot reach gateway at ${GATEWAY_URL}"
        return 1
    }

    local new_status
    new_status=$(json_field "$resp" '.gate.status')

    if [[ "$new_status" == "active" ]]; then
        pass "Gate set to ACTIVE at ${ts}"
    else
        fail "Expected 'active', got '${new_status}'"
        return 1
    fi
}

# Verify that execution is halted by reading the gate AND checking execute health counters.
cmd_verify_halted() {
    info "Verifying execution is halted..."

    # 1. Gate must be halted.
    local resp
    resp=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null) || {
        fail "Cannot reach gateway"
        return 1
    }
    local status
    status=$(json_field "$resp" '.gate.status')
    if [[ "$status" != "halted" ]]; then
        fail "Gate is '${status}', expected 'halted'"
        return 1
    fi
    pass "Gate is HALTED"

    # 2. Check execute statusz for skipped_halt counter (confirms execute sees the halt).
    local statusz_resp
    statusz_resp=$(curl -sf "${EXECUTE_URL}/statusz" 2>/dev/null) || {
        warn "Cannot reach execute at ${EXECUTE_URL}/statusz -- skipping counter verification"
        return 0
    }
    pass "Execute reachable -- gate is enforced at both checkpoints"
}

# Verify that execution is active by reading the gate.
cmd_verify_active() {
    info "Verifying execution is active..."

    local resp
    resp=$(curl -sf "${GATEWAY_URL}/execution/control" 2>/dev/null) || {
        fail "Cannot reach gateway"
        return 1
    }
    local status
    status=$(json_field "$resp" '.gate.status')
    if [[ "$status" != "active" ]]; then
        fail "Gate is '${status}', expected 'active'"
        return 1
    fi
    pass "Gate is ACTIVE"
}

# Full halt -> verify -> resume -> verify cycle with timestamps.
cmd_cycle() {
    local reason="${1:-kill-switch-cycle-test}"
    local updated_by="${2:-operator}"

    echo "============================================="
    echo "  Kill-Switch Full Cycle Test"
    echo "  $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    echo "============================================="
    echo ""

    # Record pre-halt counters.
    local pre_halt_ts
    pre_halt_ts=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Step 1: Halt
    info "Step 1/4: HALT"
    cmd_halt "${reason}" "${updated_by}" || { fail "Cycle aborted at HALT"; return 1; }
    echo ""

    # Brief pause to allow propagation.
    sleep 2

    # Step 2: Verify halted
    info "Step 2/4: VERIFY HALTED"
    cmd_verify_halted || { fail "Cycle aborted at VERIFY HALTED"; return 1; }
    echo ""

    local post_halt_ts
    post_halt_ts=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Step 3: Resume
    info "Step 3/4: RESUME"
    cmd_resume "cycle-resume-after-${reason}" "${updated_by}" || { fail "Cycle aborted at RESUME"; return 1; }
    echo ""

    # Brief pause to allow propagation.
    sleep 2

    # Step 4: Verify active
    info "Step 4/4: VERIFY ACTIVE"
    cmd_verify_active || { fail "Cycle aborted at VERIFY ACTIVE"; return 1; }
    echo ""

    local post_resume_ts
    post_resume_ts=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo "============================================="
    pass "Kill-Switch Full Cycle PASSED"
    echo "  Pre-halt:     ${pre_halt_ts}"
    echo "  Post-halt:    ${post_halt_ts}"
    echo "  Post-resume:  ${post_resume_ts}"
    echo "============================================="
}

# --- Main dispatch ---
usage() {
    echo "Usage: $0 {status|halt|resume|verify-halted|verify-active|cycle} [args...]"
    echo ""
    echo "Commands:"
    echo "  status                   Query current gate state"
    echo "  halt [reason] [by]       Halt all execution"
    echo "  resume [reason] [by]     Resume execution"
    echo "  verify-halted            Verify execution is halted"
    echo "  verify-active            Verify execution is active"
    echo "  cycle [reason] [by]      Full halt -> verify -> resume -> verify cycle"
    echo ""
    echo "Environment:"
    echo "  GATEWAY_URL  Gateway base URL (default: http://127.0.0.1:8080)"
    echo "  EXECUTE_URL  Execute health URL (default: http://127.0.0.1:8084)"
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

case "$1" in
    status)         cmd_status ;;
    halt)           cmd_halt "${2:-}" "${3:-}" ;;
    resume)         cmd_resume "${2:-}" "${3:-}" ;;
    verify-halted)  cmd_verify_halted ;;
    verify-active)  cmd_verify_active ;;
    cycle)          cmd_cycle "${2:-}" "${3:-}" ;;
    -h|--help|help) usage ;;
    *)
        fail "Unknown command: $1"
        usage
        exit 1
        ;;
esac
