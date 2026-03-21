#!/usr/bin/env bash
# smoke-venue-integration.sh — E2E smoke test for venue integration proof (S316).
#
# Validates the minimum viable venue integration path:
#   1. Credential loading from environment
#   2. Safety gate check (staleness + kill switch)
#   3. Market order submission to Binance Futures testnet
#   4. Fill receipt validation
#   5. Receipt structural compatibility with persistence layer
#
# This smoke does NOT require a running stack (no NATS, no ClickHouse, no gateway).
# It directly exercises the Go integration test suite against the real testnet.
#
# Guard rails (S316):
#   - No async fills / websocket
#   - No advanced order types (market only)
#   - No mainnet
#   - Single venue only (Binance Futures testnet)
#
# Prerequisites:
#   export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=<your-key>
#   export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=<your-secret>
#
# Usage:
#   ./scripts/smoke-venue-integration.sh
#   ./scripts/smoke-venue-integration.sh --dry-run   # skip real venue tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-venue-integration.sh [--dry-run] [--help]

Runs the venue integration E2E proof against Binance Futures testnet.

Options:
  --dry-run   Run only unit-level safety gate tests (no real venue calls).
  --help      Show this help text.

Environment:
  MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY     Required for real venue tests.
  MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET  Required for real venue tests.
EOF
}

DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run)  DRY_RUN=true; shift ;;
        --help|-h)  usage; exit 0 ;;
        *)          usage_error "unknown argument: $1" ;;
    esac
done

# ── Phase 1: Credential check ────────────────────────────────────────

phase "S316 Venue Integration Proof"
info "Canonical entrypoint: make smoke-venue"

if [[ "$DRY_RUN" == "true" ]]; then
    info "DRY RUN mode — skipping real venue tests"
    info "Running safety gate and structural tests only"
else
    if [[ -z "${MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY:-}" ]]; then
        die "MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY is not set. Set credentials or use --dry-run."
    fi
    if [[ -z "${MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET:-}" ]]; then
        die "MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET is not set. Set credentials or use --dry-run."
    fi
    info "Testnet credentials detected"
fi

# ── Phase 2: Build verification ──────────────────────────────────────

phase "Build verification"

info "Running go vet on execution package..."
(cd "$PROJECT_ROOT" && go vet ./internal/application/execution/...) || die "go vet failed"
pass "go vet clean"

# ── Phase 3: Safety gate tests (no credentials needed) ───────────────

phase "Safety gate tests (VQ6)"

info "Running safety gate and staleness guard tests..."
GATE_TESTS="TestS316_VQ6_SafetyGate_StaleIntent_BlocksVenueSubmit|TestS316_VQ6_SafetyGate_KillSwitch_BlocksVenueSubmit|TestS316_VQ6_SafetyGate_KillSwitchPriority"
(cd "$PROJECT_ROOT" && go test -v -count=1 -run "$GATE_TESTS" ./internal/application/execution/...) || {
    record_fail "safety gate tests failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "safety gate tests: staleness guard + kill switch + priority"
fi

# ── Phase 4: Venue integration tests (credentials required) ──────────

if [[ "$DRY_RUN" == "false" ]]; then
    phase "Venue integration tests (VQ1 + VQ3 + VQ4 + VQ6)"

    info "Submitting market order to Binance Futures testnet..."
    VENUE_TESTS="TestS316_VQ1|TestS316_VQ3|TestS316_VQ4|TestS316_VQ6_SafetyGate_FreshIntent|TestS316_E2E|TestS316_NoAction|TestS316_ClientOrderID"
    (cd "$PROJECT_ROOT" && go test -v -count=1 -timeout 60s -run "$VENUE_TESTS" ./internal/application/execution/...) || {
        record_fail "venue integration tests failed"
    }

    if [[ $ERRORS -eq 0 ]]; then
        pass "venue integration: submit + fill + receipt + persistence compat + safety gate"
    fi
fi

# ── Phase 5: Existing adapter tests (regression check) ───────────────

phase "Regression check"

info "Running full execution package test suite..."
(cd "$PROJECT_ROOT" && go test -count=1 -timeout 120s ./internal/application/execution/...) || {
    record_fail "regression: execution package tests failed"
}

if [[ $ERRORS -eq 0 ]]; then
    pass "regression check: all execution package tests pass"
fi

# ── Summary ──────────────────────────────────────────────────────────

echo ""
if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "S316 venue integration proof" "$ERRORS" "export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=... && export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=..."
    exit 1
fi

pass "S316 venue integration proof completed successfully"
if [[ "$DRY_RUN" == "true" ]]; then
    info "Dry run only — run without --dry-run with real credentials for full proof"
fi
