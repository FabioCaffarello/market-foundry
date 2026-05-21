#!/usr/bin/env bash
# smoke-authenticated-mainnet-soak.sh — S441: Authenticated Mainnet Proof & Sustained Soak.
#
# Proves the platform can make authenticated API calls to Binance mainnet (Spot + Futures)
# with valid credentials, sustain stable operation over a defined soak window,
# and maintain DryRunSubmitter interception at 100% throughout.
#
# Phases:
#   1. Authenticated connectivity proof (Spot + Futures account status)
#   2. DryRunSubmitter integrity after authenticated calls
#   3. Sustained soak with authenticated calls over time window
#   4. DryRunSubmitter stability throughout soak
#
# SAFETY: All tests use GET-only endpoints (read). DryRunSubmitter blocks all order submission.
#
# Build tag: livemainnet
#
# Prerequisites:
#   - Outbound HTTPS to api.binance.com and fapi.binance.com
#   - Valid Binance API credentials (read-only permissions sufficient):
#       MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY
#       MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET
#       MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY
#       MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET
#
# Usage:
#   ./scripts/smoke-authenticated-mainnet-soak.sh
#   ./scripts/smoke-authenticated-mainnet-soak.sh --quick       (30s soak instead of 5m)
#   ./scripts/smoke-authenticated-mainnet-soak.sh --skip-soak   (auth proof only, no soak)
#
# Canonical entrypoint: `make smoke-authenticated-mainnet-soak`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

ERRORS=0
SKIP_SOAK=false
SOAK_DURATION=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --quick) SOAK_DURATION="30s" ;;
        --skip-soak) SKIP_SOAK=true ;;
        -h|--help)
            cat <<'EOF'
Usage: ./scripts/smoke-authenticated-mainnet-soak.sh [--quick] [--skip-soak] [--help]

S441: Authenticated Mainnet Proof & Sustained Soak.
Proves authenticated connectivity and sustained operational stability.

Options:
  --quick       Use 30-second soak window instead of 5 minutes
  --skip-soak   Skip sustained soak phases (auth proof only)
  --help        Show this help text

Environment:
  MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY       Binance Spot API key
  MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET    Binance Spot API secret
  MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY    Binance Futures API key
  MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET Binance Futures API secret
  MF_SOAK_DURATION                            Override soak duration (e.g., "2m")
EOF
            exit 0 ;;
        *) echo "unknown argument: $1" >&2; exit 1 ;;
    esac
    shift
done

# Validate credentials are set.
CREDS_OK=true
for var in MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET \
           MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET; do
    if [ -z "${!var:-}" ]; then
        echo "ERROR: ${var} is not set" >&2
        CREDS_OK=false
    fi
done
if [ "${CREDS_OK}" != "true" ]; then
    echo ""
    echo "All four credential environment variables must be set for authenticated proof."
    echo "These should be read-only API keys (no trading permissions required)."
    exit 1
fi

smoke_banner \
    "S441: Authenticated Mainnet Proof & Sustained Soak" \
    "make smoke-authenticated-mainnet-soak" \
    "outbound HTTPS + valid Binance API credentials" \
    "none" \
    "n/a"

# ── Phase 1: Authenticated connectivity proof ────────────────────────
phase "Phase 1: S441 authenticated connectivity — Spot + Futures account status"

if go test -tags=livemainnet -v -count=1 -timeout=60s \
    -run "TestAuthenticatedMainnet_(Spot|Futures)AccountStatus" \
    ./internal/application/execution/... 2>&1; then
    pass "Authenticated connectivity proven (Spot + Futures)"
else
    record_fail "Authenticated connectivity failed"
fi

# ── Phase 2: DryRunSubmitter integrity ───────────────────────────────
phase "Phase 2: S441 DryRunSubmitter integrity after authenticated calls"

if go test -tags=livemainnet -v -count=1 -timeout=60s \
    -run "TestAuthenticatedMainnet_(DryRunIntact|PipelineChain)" \
    ./internal/application/execution/... 2>&1; then
    pass "DryRunSubmitter integrity confirmed after authenticated calls"
else
    record_fail "DryRunSubmitter integrity check failed"
fi

if [[ "${SKIP_SOAK}" == "true" ]]; then
    echo ""
    echo "════════════════════════════════════════════════════════════"
    echo " --skip-soak: skipping phases 3-4 (sustained soak)"
    echo "════════════════════════════════════════════════════════════"
    echo ""
    if [ "${ERRORS}" -eq 0 ]; then
        pass "S441 SMOKE (auth proof only): ALL CHECKS PASSED"
    else
        fail "S441 SMOKE (auth proof only): ${ERRORS} PHASE(S) FAILED"
    fi
    exit "${ERRORS}"
fi

# ── Phase 3: Sustained soak ──────────────────────────────────────────
SOAK_ENV=""
if [ -n "${SOAK_DURATION}" ]; then
    SOAK_ENV="MF_SOAK_DURATION=${SOAK_DURATION}"
fi

TIMEOUT="400s"
if [ -n "${SOAK_DURATION}" ]; then
    TIMEOUT="120s"
fi

phase "Phase 3: S441 sustained soak — repeated authenticated calls (duration=${SOAK_DURATION:-5m})"

if env ${SOAK_ENV} go test -tags=livemainnet -v -count=1 -timeout="${TIMEOUT}" \
    -run "TestAuthenticatedMainnet_SustainedSoak" \
    ./internal/application/execution/... 2>&1; then
    pass "Sustained authenticated soak completed within tolerance"
else
    record_fail "Sustained soak failed or exceeded error threshold"
fi

# ── Phase 4: DryRunSubmitter stability throughout soak ───────────────
phase "Phase 4: S441 DryRunSubmitter stability throughout soak"

if env ${SOAK_ENV} go test -tags=livemainnet -v -count=1 -timeout="${TIMEOUT}" \
    -run "TestAuthenticatedMainnet_SoakDryRunStability" \
    ./internal/application/execution/... 2>&1; then
    pass "DryRunSubmitter 100% reliable throughout soak"
else
    record_fail "DryRunSubmitter stability degraded during soak"
fi

# ── Summary ──────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "${ERRORS}" -eq 0 ]; then
    echo "  S441 SMOKE: ALL PHASES PASSED"
    echo "  Authenticated mainnet proof + sustained soak complete."
    echo "  Authorization conditions C-1 and C-4 closable."
else
    echo "  S441 SMOKE: ${ERRORS} PHASE(S) FAILED"
fi
echo "═══════════════════════════════════════════════════════════"

exit "${ERRORS}"
