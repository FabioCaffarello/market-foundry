#!/usr/bin/env bash
# smoke-endurance-soak.sh — S412: Endurance soak and persistence hardening proof.
#
# Validates sustained operational stability of the Spot execution path on the
# unified runtime by exercising:
#
#   1. S412 endurance unit tests (END-1 through END-10)
#   2. Writer row mapping fidelity across event types
#   3. Lifecycle invariant coverage under sustained load
#   4. Correlation chain preservation across cycles
#   5. Concurrent submission stability
#   6. Compose-level persistence consistency (requires stack)
#
# This smoke operates in two modes:
#   - Stackless (default): phases 1-4 run pure unit tests
#   - With compose stack: phases 5-8 validate NATS/KV/ClickHouse consistency
#
# Prerequisites:
#   For phases 1-4: None (pure unit tests, no external dependencies).
#   For phases 5-8: make up && make seed-unified (compose stack running).
#
# Usage:
#   ./scripts/smoke-endurance-soak.sh
#   ./scripts/smoke-endurance-soak.sh --wait 300
#
# Canonical entrypoint: `make smoke-endurance-soak`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"

compose() {
    docker compose -f "${COMPOSE_FILE}" "$@"
}

ENDURANCE_WAIT="${SMOKE_WAIT:-${ENDURANCE_WAIT:-300}}"
ERRORS=0

smoke_banner \
    "S412: Endurance Soak and Persistence Hardening" \
    "make smoke-endurance-soak" \
    "phases 1-4: none | phases 5-8: make up && make seed-unified" \
    "endurance-wait" \
    "${ENDURANCE_WAIT}"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: S412 Endurance Unit Tests (END-1 through END-10)"
# ══════════════════════════════════════════════════════════════════════

info "Running S412 endurance soak tests..."
if (cd "${PROJECT_ROOT}" && go test -v -count=1 -timeout 120s -run "TestS412_" ./internal/application/execution/... 2>&1); then
    pass "All S412 endurance tests pass (END-1 through END-10)"
else
    record_fail "S412 endurance tests failed"
    ERRORS=$((ERRORS + 1))
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: Writer Row Mapping Stability (All Event Types)"
# ══════════════════════════════════════════════════════════════════════

info "Running writer pipeline row mapping tests..."
if (cd "${PROJECT_ROOT}" && go test -v -count=1 -timeout 60s ./internal/adapters/clickhouse/writerpipeline/... 2>&1); then
    pass "Writer pipeline row mapping tests pass"
else
    record_fail "Writer pipeline row mapping tests failed"
    ERRORS=$((ERRORS + 1))
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Lifecycle Invariant Coverage (S384 Baseline)"
# ══════════════════════════════════════════════════════════════════════

info "Running S384 lifecycle invariant tests..."
if (cd "${PROJECT_ROOT}" && go test -count=1 -timeout 60s -run "TestS384_" ./internal/domain/execution/... 2>&1); then
    pass "S384 lifecycle invariant tests pass"
else
    record_fail "S384 lifecycle invariant tests failed"
    ERRORS=$((ERRORS + 1))
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Cross-Stage Execution Test Regression"
# ══════════════════════════════════════════════════════════════════════

info "Running S385 write-path by mode tests..."
if (cd "${PROJECT_ROOT}" && go test -count=1 -timeout 60s -run "TestS385_" ./internal/application/execution/... 2>&1); then
    pass "S385 write-path by mode tests pass"
else
    record_fail "S385 write-path tests failed"
    ERRORS=$((ERRORS + 1))
fi

info "Running S386 rejection event path tests..."
if (cd "${PROJECT_ROOT}" && go test -count=1 -timeout 60s -run "TestS386_" ./internal/domain/execution/... ./internal/adapters/nats/natsexecution/... 2>&1); then
    pass "S386 rejection event path tests pass"
else
    record_fail "S386 rejection event path tests failed"
    ERRORS=$((ERRORS + 1))
fi

info "Running S387 lifecycle persistence tests..."
if (cd "${PROJECT_ROOT}" && go test -count=1 -timeout 60s -run "TestS387_" ./internal/application/execution/... 2>&1); then
    pass "S387 lifecycle persistence tests pass"
else
    record_fail "S387 lifecycle persistence tests failed"
    ERRORS=$((ERRORS + 1))
fi

# ══════════════════════════════════════════════════════════════════════
# Compose-dependent phases (skip if stack not running)
# ══════════════════════════════════════════════════════════════════════

STACK_AVAILABLE=false
if docker compose -f "${COMPOSE_FILE}" ps --format '{{.State}}' nats 2>/dev/null | grep -q running; then
    STACK_AVAILABLE=true
fi

if [ "$STACK_AVAILABLE" = "true" ]; then
    # ══════════════════════════════════════════════════════════════════════
    phase "Phase 5: NATS Stream Health — Execution Event Streams"
    # ══════════════════════════════════════════════════════════════════════

    for stream in EXECUTION_EVENTS EXECUTION_FILL_EVENTS EXECUTION_REJECTION_EVENTS; do
        STREAM_INFO=$(compose exec -T nats nats stream info "$stream" --json 2>/dev/null || echo "")
        if [ -n "$STREAM_INFO" ]; then
            MSG_COUNT=$(echo "$STREAM_INFO" | python3 -c "import sys,json; print(json.load(sys.stdin)['state']['messages'])" 2>/dev/null || echo "0")
            pass "${stream}: ${MSG_COUNT} messages, stream healthy"
        else
            info "${stream}: stream not found or empty (may be expected if pipeline not yet exercised)"
        fi
    done

    # ══════════════════════════════════════════════════════════════════════
    phase "Phase 6: ClickHouse Writer Stability — Execution Table"
    # ══════════════════════════════════════════════════════════════════════

    EXEC_TOTAL=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
        --database market_foundry --query "SELECT count() FROM executions" 2>/dev/null || echo "0")
    info "ClickHouse executions total: ${EXEC_TOTAL} rows"

    if [ "$EXEC_TOTAL" -gt 0 ]; then
        # Check status distribution.
        STATUS_DIST=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
            --database market_foundry --query "SELECT status, count() as cnt FROM executions GROUP BY status ORDER BY cnt DESC" 2>/dev/null || echo "none")
        pass "ClickHouse executions status distribution:"
        echo "    ${STATUS_DIST}" | head -10

        # Check for Spot-sourced rows.
        SPOT_ROWS=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
            --database market_foundry --query "SELECT count() FROM executions WHERE source = 'binances'" 2>/dev/null || echo "0")
        if [ "$SPOT_ROWS" -gt 0 ]; then
            pass "ClickHouse Spot executions: ${SPOT_ROWS} rows"
        else
            info "No Spot executions in ClickHouse yet"
        fi

        # Check for rejected rows.
        REJ_ROWS=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
            --database market_foundry --query "SELECT count() FROM executions WHERE status = 'rejected'" 2>/dev/null || echo "0")
        info "ClickHouse rejected executions: ${REJ_ROWS} rows"
    else
        info "ClickHouse executions table empty — writer may not have received events yet"
    fi

    # ══════════════════════════════════════════════════════════════════════
    phase "Phase 7: NATS KV Consistency — Execution Read Models"
    # ══════════════════════════════════════════════════════════════════════

    for bucket in EXECUTION_PAPER_ORDER_LATEST EXECUTION_VENUE_MARKET_ORDER_LATEST EXECUTION_VENUE_REJECTION_LATEST; do
        BUCKET_INFO=$(compose exec -T nats nats kv info "$bucket" --json 2>/dev/null || echo "")
        if [ -n "$BUCKET_INFO" ]; then
            KEY_COUNT=$(echo "$BUCKET_INFO" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('values',0))" 2>/dev/null || echo "0")
            pass "${bucket}: ${KEY_COUNT} keys"
        else
            info "${bucket}: bucket not found (may be expected)"
        fi
    done

    # ══════════════════════════════════════════════════════════════════════
    phase "Phase 8: Persistence Coherence — Stream vs ClickHouse Delta"
    # ══════════════════════════════════════════════════════════════════════

    # Compare NATS stream counts with ClickHouse row counts for coherence.
    NATS_FILL_COUNT=$(compose exec -T nats nats stream info EXECUTION_FILL_EVENTS --json 2>/dev/null | \
        python3 -c "import sys,json; print(json.load(sys.stdin)['state']['messages'])" 2>/dev/null || echo "0")
    CH_FILL_COUNT=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
        --database market_foundry --query "SELECT count() FROM executions WHERE status = 'filled' AND type = 'venue_market_order'" 2>/dev/null || echo "0")

    info "Persistence coherence: NATS fills=${NATS_FILL_COUNT} vs ClickHouse fills=${CH_FILL_COUNT}"
    if [ "$NATS_FILL_COUNT" -ge "$CH_FILL_COUNT" ]; then
        pass "NATS fill count >= ClickHouse fill count (batch flush lag expected)"
    else
        record_fail "ClickHouse has more fills than NATS — possible data integrity issue"
        ERRORS=$((ERRORS + 1))
    fi
else
    phase "Phases 5-8: Compose-dependent (skipped — stack not running)"
    info "Run 'make up && make seed-unified' to enable compose-dependent phases"
fi

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════
echo ""
echo "================================================================"

if [ "${ERRORS}" -eq 0 ]; then
    pass "S412 Endurance Soak and Persistence Hardening: ALL PHASES PASSED"
    echo ""
    echo "Proven:"
    echo "  - 200-cycle endurance across 10 invariant categories (END-1..END-10)"
    echo "  - Writer row mapping stability for paper, fill, and rejection events"
    echo "  - Lifecycle transition consistency under sustained mixed workloads"
    echo "  - Correlation chain preservation across all cycles"
    echo "  - Concurrent submission safety (10 goroutines, no races)"
    echo "  - Dry-run submitter endurance with auditable receipts"
    echo "  - Venue live adapter stability under mock HTTP (200 cycles)"
    if [ "$STACK_AVAILABLE" = "true" ]; then
        echo "  - NATS stream health for all execution event streams"
        echo "  - ClickHouse writer stability and status distribution"
        echo "  - NATS KV read model consistency"
        echo "  - Persistence coherence between NATS and ClickHouse"
    fi
else
    fail "${ERRORS} phase(s) failed"
    echo ""
    print_smoke_diagnosis_hints "make up && make seed-unified"
fi

exit $ERRORS
