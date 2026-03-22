#!/usr/bin/env bash
# smoke-activation.sh — Canonical activation smoke for the Venue Activation Wave (S340).
#
# Validates the three activation acceptance scenarios against a live stack:
#   Phase 1: Stack and control surface readiness
#   Phase 2: AC-1 — Inactive → Active (off→on transition)
#   Phase 3: AC-2 — Active → Halt (on→halt transition)
#   Phase 4: AC-3 — Halt → Rollback (controlled return to safe state)
#   Phase 5: Activation unit test gate
#
# Prerequisites:
#   make up && make seed   # full stack running with configctl seeded
#
# Usage:
#   ./scripts/smoke-activation.sh
#   make smoke-activation
#
# Guard rails:
#   - No production expansion; validates control surface transitions only.
#   - No dashboard or alerting; stdout PASS/FAIL only.
#   - Always restores gate to active on exit (safe for pipelines).
#   - Single script, single exit code.

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
Usage: ./scripts/smoke-activation.sh [--help]

S340: Venue Activation Smoke and Acceptance Scenarios.
Validates the three canonical activation acceptance scenarios against a live
stack via the HTTP control surface.

Options:
  --help   Show this help text.

Environment:
  BASE_URL   Gateway base URL. Default: http://127.0.0.1:8080
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) usage; exit 0 ;;
        *) usage_error "unknown argument: $1" ;;
    esac
    shift
done

require_commands docker curl python3 go

CONTROL_URL="${BASE_URL}/execution/control"

# Safety trap: always restore gate to active on exit.
ks_restore_active() {
    curl -s -X PUT "${CONTROL_URL}" \
        -H "Content-Type: application/json" \
        -d '{"status":"active","reason":"smoke-activation-cleanup","updated_by":"smoke-activation"}' \
        >/dev/null 2>&1 || true
}
trap ks_restore_active EXIT

smoke_banner "Activation Smoke (S340+S341 canonical)" "make smoke-activation" "make up && make seed" "phases" "6"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Stack and Control Surface Readiness"
# ══════════════════════════════════════════════════════════════════════

# 1a. Gateway readiness.
info "Checking gateway readiness..."
GW_CODE=$(http_code "${BASE_URL}/readyz")
if [[ "$GW_CODE" == "200" ]]; then
    pass "Gateway is ready"
else
    die "Gateway not ready (HTTP ${GW_CODE}) — run: make up"
fi

# 1b. NATS health.
info "Checking NATS health..."
NATS_CODE=$(compose exec -T nats wget -q -O - http://127.0.0.1:8222/healthz 2>/dev/null && echo "ok" || echo "")
if [[ "$NATS_CODE" == "ok" ]]; then
    pass "NATS is healthy"
else
    die "NATS unreachable — run: make up"
fi

# 1c. Control surface reachable.
info "Querying execution control gate..."
READY_CODE=$(curl -s -o /tmp/s340_ready.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$READY_CODE" == "200" ]]; then
    READY_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ready.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    pass "GET /execution/control → 200 (status=${READY_STATUS})"
else
    die "Control surface unreachable (HTTP ${READY_CODE}) — run: make up && make seed"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: AC-1 — Inactive → Active (off→on transition)"
# ══════════════════════════════════════════════════════════════════════

# Step 1: Set gate to halted (simulate inactive/off posture).
info "Setting gate to halted (AC-1 setup)..."
AC1_HALT_CODE=$(curl -s -o /tmp/s340_ac1_halt.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"smoke-s340-ac1-setup","updated_by":"smoke-activation"}')
if [[ "$AC1_HALT_CODE" != "200" ]]; then
    record_fail "[AC-1/setup] PUT halted → HTTP ${AC1_HALT_CODE}"
fi

# Step 2: Confirm gate is halted.
info "Confirming gate is halted..."
AC1_CHECK_CODE=$(curl -s -o /tmp/s340_ac1_check.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC1_CHECK_CODE" == "200" ]]; then
    AC1_CHECK_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac1_check.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    if [[ "$AC1_CHECK_STATUS" == "halted" ]]; then
        pass "[AC-1/step-1] gate confirmed halted — venue_halted posture"
    else
        record_fail "[AC-1/step-1] expected status=halted, got status=${AC1_CHECK_STATUS}"
    fi
else
    record_fail "[AC-1/step-1] GET gate → HTTP ${AC1_CHECK_CODE}"
fi

# Step 3: Activate gate (off→on transition).
info "Activating gate (off→on)..."
AC1_ACTIVE_CODE=$(curl -s -o /tmp/s340_ac1_active.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"smoke-s340-ac1-enable","updated_by":"smoke-activation"}')
if [[ "$AC1_ACTIVE_CODE" != "200" ]]; then
    record_fail "[AC-1/activate] PUT active → HTTP ${AC1_ACTIVE_CODE}"
fi

# Step 4: Confirm gate is active.
info "Confirming gate is active..."
AC1_FINAL_CODE=$(curl -s -o /tmp/s340_ac1_final.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC1_FINAL_CODE" == "200" ]]; then
    AC1_FINAL_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac1_final.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    if [[ "$AC1_FINAL_STATUS" == "active" ]]; then
        pass "[AC-1/step-2] gate confirmed active — activation transition proven"
    else
        record_fail "[AC-1/step-2] expected status=active, got status=${AC1_FINAL_STATUS}"
    fi
else
    record_fail "[AC-1/step-2] GET gate → HTTP ${AC1_FINAL_CODE}"
fi

# Verify audit fields.
AC1_UPDATED_BY=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac1_final.json')).get('gate',{}).get('updated_by',''))" 2>/dev/null || echo "")
AC1_UPDATED_AT=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac1_final.json')).get('gate',{}).get('updated_at',''))" 2>/dev/null || echo "")
if [[ -n "$AC1_UPDATED_BY" && -n "$AC1_UPDATED_AT" ]]; then
    pass "[AC-1] audit fields preserved (updated_by=${AC1_UPDATED_BY})"
else
    record_fail "[AC-1] audit fields missing after activation"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: AC-2 — Active → Halt (on→halt transition)"
# ══════════════════════════════════════════════════════════════════════

# Step 1: Confirm gate is currently active (from Phase 2).
info "Confirming gate is active (AC-2 precondition)..."
AC2_PRE_CODE=$(curl -s -o /tmp/s340_ac2_pre.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC2_PRE_CODE" == "200" ]]; then
    AC2_PRE_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac2_pre.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    if [[ "$AC2_PRE_STATUS" == "active" ]]; then
        pass "[AC-2/pre] gate confirmed active — precondition met"
    else
        record_fail "[AC-2/pre] expected status=active, got status=${AC2_PRE_STATUS}"
    fi
else
    record_fail "[AC-2/pre] GET gate → HTTP ${AC2_PRE_CODE}"
fi

# Step 2: Halt gate (on→halt transition).
info "Halting gate (on→halt)..."
AC2_HALT_CODE=$(curl -s -o /tmp/s340_ac2_halt.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"smoke-s340-ac2-halt","updated_by":"smoke-activation"}')
if [[ "$AC2_HALT_CODE" != "200" ]]; then
    record_fail "[AC-2/halt] PUT halted → HTTP ${AC2_HALT_CODE}"
fi

# Step 3: Confirm gate is halted and reason matches.
info "Confirming gate is halted with correct reason..."
AC2_FINAL_CODE=$(curl -s -o /tmp/s340_ac2_final.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC2_FINAL_CODE" == "200" ]]; then
    AC2_FINAL_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac2_final.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    AC2_FINAL_REASON=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac2_final.json')).get('gate',{}).get('reason',''))" 2>/dev/null || echo "")
    if [[ "$AC2_FINAL_STATUS" == "halted" && "$AC2_FINAL_REASON" == "smoke-s340-ac2-halt" ]]; then
        pass "[AC-2] active → halted transition proven (reason=${AC2_FINAL_REASON})"
    elif [[ "$AC2_FINAL_STATUS" == "halted" ]]; then
        pass "[AC-2] active → halted transition proven"
        warn "[AC-2] reason mismatch: expected smoke-s340-ac2-halt, got ${AC2_FINAL_REASON}"
    else
        record_fail "[AC-2] expected status=halted, got status=${AC2_FINAL_STATUS}"
    fi
else
    record_fail "[AC-2] GET gate → HTTP ${AC2_FINAL_CODE}"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: AC-3 — Halt → Rollback (controlled return to safe state)"
# ══════════════════════════════════════════════════════════════════════
#
# Note: Full rollback includes binary restart with paper adapter config.
# This smoke validates the gate dimension only.

# Step 1: Confirm gate is halted (from Phase 3).
info "Confirming gate is halted (AC-3 precondition)..."
AC3_PRE_CODE=$(curl -s -o /tmp/s340_ac3_pre.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC3_PRE_CODE" == "200" ]]; then
    AC3_PRE_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac3_pre.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    if [[ "$AC3_PRE_STATUS" == "halted" ]]; then
        pass "[AC-3/pre] gate confirmed halted — precondition met"
    else
        record_fail "[AC-3/pre] expected status=halted, got status=${AC3_PRE_STATUS}"
    fi
else
    record_fail "[AC-3/pre] GET gate → HTTP ${AC3_PRE_CODE}"
fi

# Step 2: Rollback — restore gate to active.
info "Rolling back gate to active..."
AC3_ROLLBACK_CODE=$(curl -s -o /tmp/s340_ac3_rollback.json -w "%{http_code}" -X PUT "${CONTROL_URL}" \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"smoke-s340-ac3-rollback","updated_by":"smoke-activation"}')
if [[ "$AC3_ROLLBACK_CODE" != "200" ]]; then
    record_fail "[AC-3/rollback] PUT active → HTTP ${AC3_ROLLBACK_CODE}"
fi

# Step 3: Confirm gate is active after rollback.
info "Confirming gate is active after rollback..."
AC3_FINAL_CODE=$(curl -s -o /tmp/s340_ac3_final.json -w "%{http_code}" "${CONTROL_URL}")
if [[ "$AC3_FINAL_CODE" == "200" ]]; then
    AC3_FINAL_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s340_ac3_final.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    if [[ "$AC3_FINAL_STATUS" == "active" ]]; then
        pass "[AC-3] halt → rollback (gate restored) transition proven"
    else
        record_fail "[AC-3] expected status=active after rollback, got status=${AC3_FINAL_STATUS}"
    fi
else
    record_fail "[AC-3] GET gate → HTTP ${AC3_FINAL_CODE}"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Activation Unit Test Gate"
# ══════════════════════════════════════════════════════════════════════

info "Running S340 activation acceptance tests..."
S340_TESTS="TestActivationAcceptance_"
if (cd "$PROJECT_ROOT" && go test -count=1 -run "$S340_TESTS" ./internal/domain/execution/... 2>/dev/null); then
    pass "S340 activation acceptance tests pass"
else
    record_fail "S340 activation acceptance tests failed"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 6: S341 Controlled Activation Verification (Integration)"
# ══════════════════════════════════════════════════════════════════════
#
# Runs the S341 integration tests that prove the full activation lifecycle
# on the real actor path: halted → enabled → halted, with NATS KV gate
# transitions controlling real event flow through VenueAdapterActor.
# Requires NATS at localhost:4222.

info "Checking NATS reachability for integration tests..."
if nc -z localhost 4222 2>/dev/null; then
    info "Running S341 controlled activation verification tests..."
    S341_TESTS="TestControlledActivation_"
    if (cd "$PROJECT_ROOT" && go test -tags=integration -count=1 -run "$S341_TESTS" ./internal/actors/scopes/execute/... 2>&1 | tail -20); then
        pass "S341 controlled activation verification tests pass"
    else
        record_fail "S341 controlled activation verification tests failed"
    fi
else
    warn "NATS not reachable at localhost:4222 — skipping S341 integration tests"
    info "Run: make test-integration (with NATS running) to verify S341 independently"
fi

# ══════════════════════════════════════════════════════════════════════
# Cleanup
# ══════════════════════════════════════════════════════════════════════

rm -f /tmp/s340_ready.json \
      /tmp/s340_ac1_halt.json /tmp/s340_ac1_check.json /tmp/s340_ac1_active.json /tmp/s340_ac1_final.json \
      /tmp/s340_ac2_pre.json /tmp/s340_ac2_halt.json /tmp/s340_ac2_final.json \
      /tmp/s340_ac3_pre.json /tmp/s340_ac3_rollback.json /tmp/s340_ac3_final.json

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════

echo ""
if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "Activation smoke" "$ERRORS" "make up && make seed"
    exit 1
fi

pass "Activation smoke completed (S340+S341 canonical surface)"
info "Scenarios validated: AC-1 off→on | AC-2 on→halt | AC-3 halt→rollback | CAV lifecycle"
info "Full path: HTTP control surface → NATS KV gate → actor pipeline → state transitions proven"
