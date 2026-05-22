#!/usr/bin/env bash
# lib.sh — Shared script library for market-foundry automation.
#
# Source this file at the top of any script that needs standard logging,
# color output, JSON helpers, or common constants.
#
# Usage:
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "${SCRIPT_DIR}/utils/lib.sh"   # from scripts/
#   source "${SCRIPT_DIR}/lib.sh"          # from scripts/utils/

# ── Color codes ──────────────────────────────────────────────────────
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    CYAN=''
    BOLD=''
    NC=''
fi

# ── Standard logging ─────────────────────────────────────────────────
pass()  { echo -e "${GREEN}[PASS]${NC} $1"; }
fail()  { echo -e "${RED}[FAIL]${NC} $1"; }
info()  { echo -e "${YELLOW}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
phase() { echo -e "\n${CYAN}${BOLD}═══ $1 ═══${NC}"; }
die()   { fail "$1"; exit 1; }

# ── Error tracking ───────────────────────────────────────────────────
# Initialize ERRORS counter. Scripts can use record_fail() to both log
# and increment, then exit $ERRORS at the end.
ERRORS=0
record_fail() { ERRORS=$((ERRORS + 1)); fail "$1"; }

usage_error() {
    if [[ $# -gt 0 ]]; then
        printf '%s\n' "$*" >&2
    fi
    if declare -F usage >/dev/null 2>&1; then
        usage >&2
    fi
    exit 1
}

# ── Common defaults ──────────────────────────────────────────────────
# Override any of these via environment variables before sourcing.
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

# Ordered list of all Go services (startup dependency order).
ALL_SERVICES=("nats" "configctl" "gateway" "ingest" "derive" "store" "execute" "writer")

# Pipeline services (those with /statusz and /diagz).
PIPELINE_SERVICES=("configctl" "ingest" "derive" "store" "execute" "writer")

# Default timeouts (seconds).
HEALTH_WAIT_MAX="${HEALTH_WAIT_MAX:-120}"
HEALTH_POLL_INTERVAL="${HEALTH_POLL_INTERVAL:-5}"
CANDLE_WAIT_MAX="${CANDLE_WAIT_MAX:-90}"
CANDLE_POLL_INTERVAL="${CANDLE_POLL_INTERVAL:-5}"

# Default symbols and timeframes.
DEFAULT_SYMBOL="${DEFAULT_SYMBOL:-btcusdt}"
DEFAULT_SOURCE="${DEFAULT_SOURCE:-binancef}"
read -ra ALL_TIMEFRAMES <<< "${ALL_TIMEFRAMES:-60 300 900 3600}"

# ClickHouse defaults (overridable per environment).
CLICKHOUSE_PORT="${CLICKHOUSE_PORT:-9000}"
CLICKHOUSE_USER="${CLICKHOUSE_USER:-default}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-clickhouse}"
CLICKHOUSE_DATABASE="${CLICKHOUSE_DATABASE:-market_foundry}"

# ── Common validations ───────────────────────────────────────────────
require_commands() {
    local missing=()
    local cmd
    for cmd in "$@"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        die "missing required command(s): ${missing[*]}"
    fi
}

require_positive_integer() {
    local label="$1"
    local value="$2"

    if [[ ! "$value" =~ ^[0-9]+$ ]] || (( value <= 0 )); then
        die "${label} must be a positive integer (got: ${value})"
    fi
}

http_code() {
    local url="$1"
    local code
    code=$(curl -sS -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || true)
    if [[ -z "$code" || "$code" == "000000" ]]; then
        echo "000"
    else
        echo "$code"
    fi
}

smoke_banner() {
    local label="$1"
    local canonical_target="$2"
    local setup_hint="$3"
    local wait_label="$4"
    local wait_value="$5"

    phase "$label"
    info "Canonical entrypoint: ${canonical_target}"
    info "Expected setup before running: ${setup_hint}"
    info "Runtime context: BASE_URL=${BASE_URL} ${wait_label}=${wait_value}s"
}

print_smoke_diagnosis_hints() {
    local setup_hint="$1"
    cat <<EOF
Next checks:
  - Confirm stack/setup: ${setup_hint}
  - Inspect runtime state: make ps
  - Inspect gateway logs: make logs SERVICE=gateway
  - Capture a quick snapshot: make diag
EOF
}

smoke_die_with_hints() {
    local message="$1"
    local setup_hint="$2"
    fail "$message"
    echo ""
    print_smoke_diagnosis_hints "$setup_hint"
    exit 1
}

smoke_fail_summary() {
    local label="$1"
    local error_count="$2"
    local setup_hint="$3"
    fail "${label} completed with ${error_count} issue(s)"
    echo ""
    print_smoke_diagnosis_hints "$setup_hint"
}

# ── JSON helpers ─────────────────────────────────────────────────────
# json_field extracts a top-level field from a JSON string.
# Usage: value=$(echo "$json" | json_field "status")
json_field() {
    local field="$1"
    python3 -c "import sys,json; print(json.load(sys.stdin).get('${field}',''))" 2>/dev/null || echo ""
}

# json_nested extracts a nested value using a dotted path.
# Usage: value=$(echo "$json" | json_nested "candle.source")
json_nested() {
    local path="$1"
    python3 -c "
import sys,json
d=json.load(sys.stdin)
keys='${path}'.split('.')
for k in keys:
    if isinstance(d, dict):
        d=d.get(k)
    else:
        d=None
        break
print('' if d is None else d)
" 2>/dev/null || echo ""
}

# json_has_key checks if a top-level key exists and is not null.
# Usage: if echo "$json" | json_has_key "candle"; then ...
json_has_key() {
    local field="$1"
    python3 -c "
import sys,json
d=json.load(sys.stdin)
v=d.get('${field}')
print('yes' if v is not None else 'no')
" 2>/dev/null || echo "no"
}

# ── Compose helpers ──────────────────────────────────────────────────
# compose_cmd returns the docker compose command with the correct file.
# Requires PROJECT_ROOT to be set.
compose_cmd() {
    echo "docker compose -f ${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
}

# svc_port returns the internal port for a service name.
svc_port() {
    local svc="$1"
    case "$svc" in
        configctl|gateway) echo "8080" ;;
        store) echo "8081" ;;
        ingest) echo "8082" ;;
        derive) echo "8083" ;;
        execute) echo "8084" ;;
        writer) echo "8085" ;;
        *) echo "8080" ;;
    esac
}
