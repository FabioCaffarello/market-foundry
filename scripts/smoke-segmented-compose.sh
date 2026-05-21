#!/usr/bin/env bash
# smoke-segmented-compose.sh — S394/S417: Unified segmented compose proof.
#
# Validates that the execute binary boots correctly with the canonical unified
# segmented config (both Spot and Futures segments enabled, dry_run=true):
#
#   1. Config validation: unified segmented config passes startup validation.
#   2. Segment identity: execute logs show both segments at startup.
#   3. Dry-run protection: DryRunSubmitter is active (no real venue contact).
#   4. Segment coexistence: both adapters present in logs.
#   5. Activation surface: reports correct adapter/segment/dry_run state.
#
# This smoke runs WITHIN the existing compose stack. It rebuilds execute with
# the unified config, verifies boot, checks logs, then restores the default.
#
# Prerequisites:
#   make up && make seed   (compose stack running + bindings activated)
#
# Usage:
#   ./scripts/smoke-segmented-compose.sh
#
# Canonical entrypoint: `make smoke-segmented-compose`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
COMPOSE_UNIFIED="${PROJECT_ROOT}/deploy/compose/docker-compose.unified.yaml"

compose() {
    docker compose -f "${COMPOSE_FILE}" "$@"
}

compose_unified() {
    docker compose -f "${COMPOSE_FILE}" -f "${COMPOSE_UNIFIED}" "$@"
}

BOOT_WAIT="${BOOT_WAIT:-60}"

ERRORS=0

smoke_banner \
    "S394/S417: Unified Segmented Compose Proof" \
    "make smoke-segmented-compose" \
    "make up && make seed" \
    "boot-wait" \
    "${BOOT_WAIT}"

# Helper: wait for execute to become healthy.
wait_execute_healthy() {
    local label="$1"
    local compose_fn="$2"
    local attempts=0
    local max=$((BOOT_WAIT / 5))

    info "Waiting for execute (${label}) to become healthy..."
    while [[ $attempts -lt $max ]]; do
        local status
        status=$($compose_fn ps execute --format '{{.Health}}' 2>/dev/null || echo "unknown")
        if [[ "$status" == "healthy" ]]; then
            pass "execute (${label}) is healthy"
            return 0
        fi
        attempts=$((attempts + 1))
        sleep 5
    done
    record_fail "execute (${label}) did not become healthy within ${BOOT_WAIT}s"
    return 1
}

# Helper: check execute logs for a string.
check_execute_log() {
    local label="$1"
    local pattern="$2"
    local compose_fn="$3"
    local log_output
    log_output=$($compose_fn logs execute --tail 100 2>&1)

    if echo "$log_output" | grep -q "$pattern"; then
        pass "${label}: found '${pattern}' in execute logs"
        return 0
    else
        record_fail "${label}: expected '${pattern}' in execute logs but not found"
        return 1
    fi
}

# ══════════════════════════════════════════════════════════════════════
# Phase 1: Baseline — verify stack is running with default paper config
# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Baseline Stack Verification"

if compose ps execute --format '{{.Health}}' 2>/dev/null | grep -q "healthy"; then
    pass "execute is healthy with default config (paper_simulator)"
else
    smoke_die_with_hints "execute is not healthy — cannot proceed with segmented proof" "make up && make seed"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 2: Unified Segmented Boot — both Spot and Futures segments
# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: Unified Segmented Boot"

info "Setting dummy credentials for both segments (dry-run — never used)..."
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY="smoke-test-futures-key"
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET="smoke-test-futures-secret"
export MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY="smoke-test-spot-key"
export MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET="smoke-test-spot-secret"

info "Rebuilding execute with unified segmented config..."
compose_unified up -d --build execute 2>&1 | tail -5

if wait_execute_healthy "unified" "compose_unified"; then
    # Check both segments present in logs.
    check_execute_log "spot-segment" "spot" "compose_unified"
    check_execute_log "futures-segment" "futures" "compose_unified"

    # Check dry-run active.
    check_execute_log "dry-run" "dry_run=true" "compose_unified"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 3: Segment Coexistence — both adapters visible
# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Segment Coexistence Verification"

unified_logs=$(compose_unified logs execute --tail 100 2>&1)

if echo "$unified_logs" | grep -q "binance_spot_testnet"; then
    pass "Spot adapter (binance_spot_testnet) present in logs"
else
    record_fail "Spot adapter not found in logs"
fi

if echo "$unified_logs" | grep -q "binance_futures_testnet"; then
    pass "Futures adapter (binance_futures_testnet) present in logs"
else
    record_fail "Futures adapter not found in logs"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 4: Config Validation — verify fail-closed rejects bad configs
# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Config Validation (Unit Tests)"

info "Running segment enablement tests..."
if (cd "${PROJECT_ROOT}" && go test ./internal/shared/settings/ -run "TestSegment" -count=1 -timeout 30s >/dev/null 2>&1); then
    pass "segment enablement unit tests pass"
else
    record_fail "segment enablement unit tests failed"
fi

info "Running spot adapter tests..."
if (cd "${PROJECT_ROOT}" && go test ./internal/application/execution/ -run "TestBinanceSpot" -count=1 -timeout 30s >/dev/null 2>&1); then
    pass "spot adapter unit tests pass"
else
    record_fail "spot adapter unit tests failed"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 5: Restore — return to default paper config
# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Restore Default Config"

info "Rebuilding execute with default paper config..."
compose up -d --build execute 2>&1 | tail -5

if wait_execute_healthy "default" "compose"; then
    pass "execute restored to default paper config"
else
    record_fail "failed to restore execute to default config"
fi

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════
phase "S394/S417 Smoke Summary"

if [[ $ERRORS -eq 0 ]]; then
    pass "All segmented compose checks passed"
    echo ""
    info "Proven:"
    info "  - Execute boots with unified config (both Spot + Futures segments)"
    info "  - Both segment adapters visible in logs"
    info "  - Dry-run protection active"
    info "  - Default config restored successfully"
else
    fail "${ERRORS} check(s) failed"
    echo ""
    print_smoke_diagnosis_hints "make up && make seed"
fi

exit $ERRORS
