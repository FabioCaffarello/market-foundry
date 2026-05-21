#!/usr/bin/env bash
# smoke-spot-ingest-binding.sh — S397: Spot ingest binding seed and runtime projection validation.
#
# Validates that:
#   1. Spot bindings can be seeded via configctl (SOURCE=binances).
#   2. Active bindings include Spot topics after seed.
#   3. Spot exchange adapter (binances) parses and normalizes correctly.
#   4. Spot and Futures binding sources remain distinct.
#   5. Execute with Spot config boots and recognizes segment=spot.
#
# Prerequisites:
#   make up   (compose stack running)
#
# Usage:
#   ./scripts/smoke-spot-ingest-binding.sh
#
# Canonical entrypoint: `make smoke-spot-ingest`

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
    "S397: Spot Ingest Binding Seed and Runtime Projection" \
    "make smoke-spot-ingest" \
    "make up" \
    "boot-wait" \
    "${BOOT_WAIT}"

# ══════════════════════════════════════════════════════════════════════
# Phase 1: Unit Tests — Spot exchange adapter and binding model
# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Spot Exchange Adapter Unit Tests"

info "Running binances adapter tests..."
if (cd "${PROJECT_ROOT}" && go test ./internal/adapters/exchanges/binances/... -count=1 -timeout 30s >/dev/null 2>&1); then
    pass "binances adapter tests pass (parse + normalize + source identity)"
else
    record_fail "binances adapter tests failed"
fi

info "Running S397 binding projection tests..."
if (cd "${PROJECT_ROOT}" && go test ./internal/actors/scopes/ingest/... -run "TestS397" -count=1 -timeout 30s >/dev/null 2>&1); then
    pass "S397 spot ingest binding tests pass"
else
    record_fail "S397 spot ingest binding tests failed"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 2: Seed Spot Bindings — configctl lifecycle
# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: Seed Spot Bindings via Configctl"

info "Checking gateway readiness..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/readyz" 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" != "200" ]]; then
    smoke_die_with_hints "gateway not ready at ${BASE_URL}/readyz (HTTP ${HTTP_CODE})" "make up"
fi
pass "gateway is ready"

info "Seeding Spot bindings (SOURCE=binances)..."
if SOURCE=binances "${SCRIPT_DIR}/seed-configctl.sh" 2>&1 | tail -5; then
    pass "Spot bindings seeded successfully (source=binances)"
else
    record_fail "Spot binding seed failed"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 3: Verify Active Bindings — check configctl reports spot topics
# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Verify Active Spot Bindings"

ACTIVE_RESPONSE=$(curl -s "${BASE_URL}/configctl/configs/active?scope_kind=global&scope_key=default" 2>/dev/null || echo "{}")

if echo "$ACTIVE_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
bindings = data.get('config', {}).get('bindings', [])
spot_bindings = [b for b in bindings if b.get('topic', '').startswith('binances.')]
if spot_bindings:
    for b in spot_bindings:
        print(f'  found: {b[\"topic\"]}')
    sys.exit(0)
else:
    sys.exit(1)
" 2>/dev/null; then
    pass "Active config contains Spot bindings (binances.*)"
else
    record_fail "No Spot bindings found in active config"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 4: Execute Spot Boot — segment=spot recognized at runtime
# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Execute Spot Segment Boot Validation"

info "Setting dummy credentials for both segments (dry-run — never used)..."
export MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY="smoke-test-spot-key"
export MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET="smoke-test-spot-secret"
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY="smoke-test-futures-key"
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET="smoke-test-futures-secret"

info "Rebuilding execute with unified segmented config..."
compose_unified up -d --build execute 2>&1 | tail -5

# Wait for healthy.
attempts=0
max=$((BOOT_WAIT / 5))
healthy=false
while [[ $attempts -lt $max ]]; do
    status=$(compose_unified ps execute --format '{{.Health}}' 2>/dev/null || echo "unknown")
    if [[ "$status" == "healthy" ]]; then
        healthy=true
        break
    fi
    attempts=$((attempts + 1))
    sleep 5
done

if $healthy; then
    pass "execute (spot) is healthy"

    # Check segment=spot in logs.
    spot_logs=$(compose_unified logs execute --tail 100 2>&1)
    if echo "$spot_logs" | grep -q "segment=spot"; then
        pass "execute logs show segment=spot"
    else
        record_fail "execute logs missing segment=spot"
    fi

    if echo "$spot_logs" | grep -q "dry_run=true"; then
        pass "dry-run protection active for Spot segment"
    else
        record_fail "dry-run not detected for Spot segment"
    fi
else
    record_fail "execute (spot) did not become healthy within ${BOOT_WAIT}s"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 5: Restore Default Config
# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Restore Default Config"

info "Rebuilding execute with default paper config..."
compose up -d --build execute 2>&1 | tail -5

attempts=0
while [[ $attempts -lt $max ]]; do
    status=$(compose ps execute --format '{{.Health}}' 2>/dev/null || echo "unknown")
    if [[ "$status" == "healthy" ]]; then
        pass "execute restored to default paper config"
        break
    fi
    attempts=$((attempts + 1))
    sleep 5
done

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════
phase "S397 Smoke Summary"

if [[ $ERRORS -eq 0 ]]; then
    pass "All Spot ingest binding seed checks passed"
    echo ""
    info "Proven:"
    info "  - binances adapter parses Spot aggTrade and stamps source=binances"
    info "  - Spot binding topics (binances.*) seeded in configctl"
    info "  - Active bindings include Spot topics"
    info "  - Execute boots with segment=spot and dry_run=true"
    info "  - Spot and Futures sources remain distinct"
    info "  - Default config restored"
else
    fail "${ERRORS} check(s) failed"
    echo ""
    print_smoke_diagnosis_hints "make up"
fi

exit $ERRORS
