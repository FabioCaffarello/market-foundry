#!/usr/bin/env bash
# ci-wait-ready.sh — Reusable readiness polling for CI and local pre-smoke setup.
#
# Polls infrastructure services (ClickHouse, gateway) until they respond healthy
# or a configurable timeout expires. Designed to replace inline readiness loops
# duplicated across CI workflows and smoke scripts.
#
# Prerequisites:
#   Stack must be starting (make up already issued).
#
# Usage:
#   ./scripts/ci-wait-ready.sh
#   ./scripts/ci-wait-ready.sh --timeout 180
#   ./scripts/ci-wait-ready.sh --help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/ci-wait-ready.sh [--timeout <seconds>] [--skip-clickhouse] [--help]

Polls infrastructure readiness (ClickHouse + gateway) with structured output.
Exits 0 on success, 1 on timeout.

Options:
  --timeout <seconds>   Maximum wait time per service. Default: 120
  --skip-clickhouse     Skip ClickHouse readiness check (useful for non-analytical smokes).
  --help                Show this help text.

Environment:
  BASE_URL              Gateway base URL. Default: http://127.0.0.1:8080
  CI_READY_TIMEOUT      Alternate way to set --timeout from env.
EOF
}

TIMEOUT="${CI_READY_TIMEOUT:-120}"
SKIP_CLICKHOUSE=false
COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --timeout)
            [[ $# -ge 2 ]] || { echo "error: --timeout requires a value" >&2; exit 1; }
            TIMEOUT="$2"
            shift
            ;;
        --skip-clickhouse)
            SKIP_CLICKHOUSE=true
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "error: unknown argument: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
    shift
done

require_positive_integer "--timeout" "${TIMEOUT}"

ERRORS=0

phase "Infrastructure Readiness Check"
info "Timeout per service: ${TIMEOUT}s"
info "Gateway: ${BASE_URL}"
info "ClickHouse: $([ "$SKIP_CLICKHOUSE" = true ] && echo "skipped" || echo "enabled")"

# ── ClickHouse ─────────────────────────────────────────────────────
if [[ "$SKIP_CLICKHOUSE" = false ]]; then
    info "Waiting for ClickHouse..."
    ELAPSED=0
    CH_READY=false
    while [[ $ELAPSED -lt $TIMEOUT ]]; do
        if docker compose -f "${COMPOSE_FILE}" exec -T clickhouse \
            clickhouse-client --port 9000 --user default --password clickhouse \
            --query "SELECT 1" 2>/dev/null | grep -q 1; then
            CH_READY=true
            break
        fi
        sleep 5
        ELAPSED=$((ELAPSED + 5))
    done

    if [[ "$CH_READY" = true ]]; then
        pass "ClickHouse ready after ${ELAPSED}s"
    else
        record_fail "ClickHouse not ready after ${TIMEOUT}s"
    fi
fi

# ── Gateway ────────────────────────────────────────────────────────
info "Waiting for gateway readiness..."
ELAPSED=0
GW_READY=false
while [[ $ELAPSED -lt $TIMEOUT ]]; do
    if curl -fsS "${BASE_URL}/readyz" >/dev/null 2>&1; then
        GW_READY=true
        break
    fi
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [[ "$GW_READY" = true ]]; then
    pass "Gateway ready after ${ELAPSED}s"
else
    record_fail "Gateway not ready after ${TIMEOUT}s"
fi

# ── Summary ────────────────────────────────────────────────────────
if [[ $ERRORS -gt 0 ]]; then
    fail "Infrastructure readiness check failed ($ERRORS issue(s))"
    info "Next steps: make ps && make logs SERVICE=gateway && make diag"
    exit 1
fi

pass "All infrastructure services ready"
