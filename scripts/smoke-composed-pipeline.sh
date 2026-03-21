#!/usr/bin/env bash
# smoke-composed-pipeline.sh — S330: Live smoke after production wiring.
#
# Reproduzível operational smoke proving the final composed venue pipeline
# (Post200Reconciler → RetrySubmitter → rawAdapter) remains functional,
# auditable, and stable after the production wiring tranche.
#
# This smoke exercises:
#   1. Build verification (go vet)
#   2. Supervisor composition tests (SC-01..SC-07)
#   3. Venue path verification tests (VP-01..VP-09)
#   4. Venue error code classification tests
#   5. Full execution package regression gate
#
# No running stack required — this is a Go-test-only smoke that validates
# the composed pipeline at the application layer.
#
# Prerequisites:
#   go 1.23+ installed
#
# Usage:
#   ./scripts/smoke-composed-pipeline.sh
#   ./scripts/smoke-composed-pipeline.sh --help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-composed-pipeline.sh [--help]

S330: Live smoke after production wiring.
Validates the final composed venue pipeline via reproducible Go test suites.
No running stack required.

Options:
  --help    Show this help text.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h) usage; exit 0 ;;
        *)         usage_error "unknown argument: $1" ;;
    esac
done

require_commands go

ERRORS=0

phase "S330 Composed Pipeline Smoke"
info "Canonical entrypoint: make smoke-composed"
info "Validates: decorator composition + venue path + observability + safety gate"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Build Verification"
# ══════════════════════════════════════════════════════════════════════

info "Running go vet on execution package..."
(cd "$PROJECT_ROOT" && go vet ./internal/application/execution/...) || die "go vet failed on execution package"
pass "go vet: execution package clean"

info "Running go vet on execute actor package..."
(cd "$PROJECT_ROOT" && go vet ./internal/actors/scopes/execute/...) || die "go vet failed on execute actor package"
pass "go vet: execute actor package clean"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: Supervisor Composition Tests (SC-01..SC-07)"
# ══════════════════════════════════════════════════════════════════════

info "Running S328 composition tests..."
SC_TESTS="TestSC01|TestSC02|TestSC03|TestSC04|TestSC05|TestSC06|TestSC07"
(cd "$PROJECT_ROOT" && go test -v -count=1 -run "$SC_TESTS" ./internal/application/execution/... 2>&1) || {
    record_fail "supervisor composition tests (SC-01..SC-07) failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "SC-01..SC-07: decorator composition verified"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Venue Path Verification Tests (VP-01..VP-09)"
# ══════════════════════════════════════════════════════════════════════

info "Running S329 venue path verification tests..."
VP_TESTS="TestVP01|TestVP02|TestVP03|TestVP04|TestVP05|TestVP06|TestVP07|TestVP08|TestVP09"
(cd "$PROJECT_ROOT" && go test -v -count=1 -run "$VP_TESTS" ./internal/application/execution/... 2>&1) || {
    record_fail "venue path verification tests (VP-01..VP-09) failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "VP-01..VP-09: composed venue path verified"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Venue Error Code Classification Tests"
# ══════════════════════════════════════════════════════════════════════

info "Running S325 error code classification tests..."
EC_TESTS="TestEC_S325"
(cd "$PROJECT_ROOT" && go test -v -count=1 -run "$EC_TESTS" ./internal/application/execution/... 2>&1) || {
    record_fail "venue error code classification tests failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "error code classification: venue-aware classification verified"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Full Execution Package Regression Gate"
# ══════════════════════════════════════════════════════════════════════

info "Running full execution package test suite..."
(cd "$PROJECT_ROOT" && go test -count=1 -timeout 120s ./internal/application/execution/... 2>&1) || {
    record_fail "regression: execution package tests failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "regression gate: all execution package tests pass"
fi

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════

echo ""
if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "S330 composed pipeline smoke" "$ERRORS" "ensure Go toolchain is installed"
    exit 1
fi

pass "S330 composed pipeline smoke completed successfully"
info "Pipeline validated: composition (SC) + venue path (VP) + error classification + regression gate"
info "Composed path: Post200Reconciler -> RetrySubmitter(+hooks) -> rawAdapter"
