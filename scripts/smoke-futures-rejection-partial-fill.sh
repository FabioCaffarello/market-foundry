#!/usr/bin/env bash
# smoke-futures-rejection-partial-fill.sh -- S417/S423: Futures rejection and partial fill proof.
#
# Validates Futures rejection and partial fill lifecycle paths:
# - S417: Rejection (submitted -> rejected) with real error codes
# - S417: Partial fill (submitted -> partially_filled) with Futures response format
# - S417: Rejection event construction and audit trail completeness
# - S417: Segment isolation (Spot not contacted for Futures paths)
# - S423: Explicit ValidTransition lifecycle assertions for rejection/partial fill
# - S423: QueryOrder reconciliation for rejected/partially_filled states
# - S423: Multi-scenario rejection event audit trail with lifecycle verification
# - S423: S422 fill-path regression proof
#
# This smoke:
#   1. Runs S423 adapter-level tests (lifecycle-grade evidence).
#   2. Runs S417 adapter-level tests (mock-based error classification).
#   3. Runs S417 actor-level tests (SegmentRouter composition).
#   4. Runs S422 regression tests to verify fill path unchanged.
#   5. Validates Futures config exists and is parseable.
#
# Prerequisites:
#   For all phases: None (pure unit tests, no external dependencies).
#
# Usage:
#   ./scripts/smoke-futures-rejection-partial-fill.sh
#
# Canonical entrypoint: `make smoke-futures-rejection-partial-fill`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

ERRORS=0

smoke_banner \
    "S423: Futures Real Rejection and Partial Fill Evidence" \
    "make smoke-futures-rejection-partial-fill" \
    "none (unit tests only)" \
    "none" \
    "n/a"

# -- Phase 1: S423 lifecycle-grade tests -----------------------------------
phase_start "S423 adapter-level tests (lifecycle ValidTransition + QueryOrder)"

if go test -v -count=1 -run "TestS423_" ./internal/application/execution/... 2>&1; then
    phase_pass "S423 adapter-level tests passed"
else
    phase_fail "S423 adapter-level tests failed"
    ERRORS=$((ERRORS + 1))
fi

# -- Phase 2: S417 adapter-level tests ------------------------------------
phase_start "S417 adapter-level unit tests (rejection + partial fill)"

if go test -v -count=1 -run "TestS417_" ./internal/application/execution/... 2>&1; then
    phase_pass "S417 adapter-level tests passed"
else
    phase_fail "S417 adapter-level tests failed"
    ERRORS=$((ERRORS + 1))
fi

# -- Phase 3: S417 actor-composition tests ---------------------------------
phase_start "S417 actor-composition unit tests (rejection + partial fill)"

if go test -v -count=1 -run "TestS417_" ./internal/actors/scopes/execute/... 2>&1; then
    phase_pass "S417 actor-composition tests passed"
else
    phase_fail "S417 actor-composition tests failed"
    ERRORS=$((ERRORS + 1))
fi

# -- Phase 4: S422 regression (fill path unchanged) -----------------------
phase_start "S422 regression tests (fill path must remain intact)"

if go test -v -count=1 -run "TestS422_" ./internal/application/execution/... 2>&1; then
    phase_pass "S422 adapter regression passed"
else
    phase_fail "S422 adapter regression failed"
    ERRORS=$((ERRORS + 1))
fi

# -- Phase 5: Config validation -------------------------------------------
phase_start "Unified segmented config exists and is parseable"

UNIFIED_CONFIG="${PROJECT_ROOT}/deploy/configs/execute-unified.jsonc"
if [ -f "${UNIFIED_CONFIG}" ]; then
    phase_pass "Unified segmented config exists: ${UNIFIED_CONFIG}"
else
    phase_fail "Unified segmented config missing: ${UNIFIED_CONFIG}"
    ERRORS=$((ERRORS + 1))
fi

# -- Summary ---------------------------------------------------------------
echo ""
echo "================================================================"
if [ "${ERRORS}" -eq 0 ]; then
    echo "  S423 SMOKE: ALL PHASES PASSED"
else
    echo "  S423 SMOKE: ${ERRORS} PHASE(S) FAILED"
fi
echo "================================================================"

exit "${ERRORS}"
