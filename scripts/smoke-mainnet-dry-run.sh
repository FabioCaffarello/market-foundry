#!/usr/bin/env bash
# smoke-mainnet-dry-run.sh — S436: Mainnet Dry-Run Proof.
#
# Proves the platform can talk to Binance mainnet endpoints (Spot + Futures)
# in strict dry-run mode with zero risk of real order submission.
#
# Phases:
#   1. S436 config validation tests (dry_run enforcement for mainnet)
#   2. S436 live mainnet connectivity (DNS, TLS, /ping) — requires network
#   3. DryRunSubmitter interception proof against mainnet adapters
#   4. Audit marker and pipeline chain verification
#
# Build tags:
#   Phases 1: no build tags (standard unit tests)
#   Phases 2-4: livemainnet build tag (requires outbound HTTPS)
#
# Prerequisites:
#   Phase 1: None (pure unit tests)
#   Phases 2-4: Outbound HTTPS access to api.binance.com and fapi.binance.com
#
# Usage:
#   ./scripts/smoke-mainnet-dry-run.sh
#   ./scripts/smoke-mainnet-dry-run.sh --skip-live   (phases 1 only)
#
# Canonical entrypoint: `make smoke-mainnet-dry-run`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

ERRORS=0
SKIP_LIVE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-live) SKIP_LIVE=true ;;
        -h|--help)
            cat <<'EOF'
Usage: ./scripts/smoke-mainnet-dry-run.sh [--skip-live] [--help]

S436: Mainnet Dry-Run Proof.
Proves mainnet connectivity and dry-run interception with zero real order risk.

Options:
  --skip-live   Skip phases requiring network access (run config tests only)
  --help        Show this help text
EOF
            exit 0 ;;
        *) echo "unknown argument: $1" >&2; exit 1 ;;
    esac
    shift
done

smoke_banner \
    "S436: Mainnet Dry-Run Proof (Connectivity + DryRunSubmitter Interception)" \
    "make smoke-mainnet-dry-run" \
    "phase 1: none; phases 2-4: outbound HTTPS" \
    "none" \
    "n/a"

# ── Phase 1: Config validation (dry_run enforcement for mainnet) ──────
phase "Phase 1: S436 config validation — mainnet dry_run enforcement"

if go test -v -count=1 -run "TestS436_" ./internal/shared/settings/... 2>&1; then
    pass "S436 config validation tests passed"
else
    record_fail "S436 config validation tests failed"
fi

if [[ "${SKIP_LIVE}" == "true" ]]; then
    echo ""
    echo "════════════════════════════════════════════════════════════"
    echo " --skip-live: skipping phases 2-4 (network-dependent)"
    echo "════════════════════════════════════════════════════════════"
    echo ""
    if [ "${ERRORS}" -eq 0 ]; then
        pass "S436 SMOKE (phase 1 only): ALL CHECKS PASSED"
    else
        fail "S436 SMOKE (phase 1 only): ${ERRORS} PHASE(S) FAILED"
    fi
    exit "${ERRORS}"
fi

# ── Phase 2: Live mainnet connectivity (DNS, TLS, /ping) ─────────────
phase "Phase 2: S436 live mainnet connectivity (DNS + TLS + /ping)"

if go test -tags=livemainnet -v -count=1 \
    -run "TestMainnetDryRun_(DNS|TLS|PublicEndpoint)" \
    ./internal/application/execution/... 2>&1; then
    pass "Mainnet connectivity proven (Spot + Futures)"
else
    record_fail "Mainnet connectivity failed"
fi

# ── Phase 3: DryRunSubmitter interception proof ───────────────────────
phase "Phase 3: S436 DryRunSubmitter interception against mainnet adapters"

if go test -tags=livemainnet -v -count=1 \
    -run "TestMainnetDryRun_(Spot|Futures)DryRunInterception" \
    ./internal/application/execution/... 2>&1; then
    pass "DryRunSubmitter interception confirmed (Spot + Futures)"
else
    record_fail "DryRunSubmitter interception tests failed"
fi

# ── Phase 4: Audit markers and pipeline chain ─────────────────────────
phase "Phase 4: S436 audit markers and pipeline chain composition"

if go test -tags=livemainnet -v -count=1 \
    -run "TestMainnetDryRun_(AuditMarker|Credential|Pipeline)" \
    ./internal/application/execution/... 2>&1; then
    pass "Audit markers and pipeline chain verified"
else
    record_fail "Audit marker or pipeline chain tests failed"
fi

# ── Summary ───────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "${ERRORS}" -eq 0 ]; then
    echo "  S436 SMOKE: ALL PHASES PASSED"
else
    echo "  S436 SMOKE: ${ERRORS} PHASE(S) FAILED"
fi
echo "═══════════════════════════════════════════════════════════"

exit "${ERRORS}"
