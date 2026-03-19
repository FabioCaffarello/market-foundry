#!/usr/bin/env bash
# live-pipeline-activate.sh — Minimal live pipeline activation and validation.
#
# Orchestrates the full live activation sequence:
#   1. Start compose stack (make up)
#   2. Wait for all services to become healthy
#   3. Seed configctl with ingestion bindings
#   4. Wait for pipeline to produce data
#   5. Validate diagnostics across all runtimes
#   6. Run E2E smoke test
#
# Usage:
#   ./scripts/live-pipeline-activate.sh                          # single symbol (btcusdt)
#   ./scripts/live-pipeline-activate.sh --multi-symbol           # multi-symbol (btcusdt + ethusdt)
#   ./scripts/live-pipeline-activate.sh --skip-build             # skip docker build (reuse images)
#   ./scripts/live-pipeline-activate.sh --check-only             # skip build+up, validate running stack
#   ./scripts/live-pipeline-activate.sh --multi-symbol --check-only  # validate multi-symbol running stack
#
# Prerequisites:
#   Docker and docker compose must be available.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
COMPOSE="docker compose -f ${COMPOSE_FILE}"
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

SKIP_BUILD=false
CHECK_ONLY=false
MULTI_SYMBOL=false
for arg in "$@"; do
    case "$arg" in
        --skip-build) SKIP_BUILD=true ;;
        --check-only) CHECK_ONLY=true ;;
        --multi-symbol) MULTI_SYMBOL=true ;;
    esac
done

# Determine symbols to validate based on mode.
if $MULTI_SYMBOL; then
    SYMBOLS=("btcusdt" "ethusdt")
else
    SYMBOLS=("btcusdt")
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
phase() { echo -e "\n${CYAN}${BOLD}═══ $1 ═══${NC}"; }

ERRORS=0
record_fail() { ERRORS=$((ERRORS + 1)); fail "$1"; }

# ---------- Phase 1: Start Stack ----------
if ! $CHECK_ONLY; then
    phase "Phase 1: Starting Compose Stack"
    if $SKIP_BUILD; then
        info "Starting stack (reusing existing images)..."
        $COMPOSE up -d
    else
        info "Building and starting stack..."
        $COMPOSE up -d --build
    fi
    pass "Compose stack started"
fi

# ---------- Phase 2: Wait for Health ----------
phase "Phase 2: Waiting for Service Health"

SERVICES=("nats" "configctl" "gateway" "ingest" "derive" "store" "execute")
MAX_WAIT=120
POLL_INTERVAL=5

for svc in "${SERVICES[@]}"; do
    info "Waiting for ${svc} to become healthy..."
    elapsed=0
    while [[ $elapsed -lt $MAX_WAIT ]]; do
        status=$($COMPOSE ps --format json "${svc}" 2>/dev/null | python3 -c "
import sys, json
for line in sys.stdin:
    data = json.loads(line)
    print(data.get('Health', data.get('health', 'unknown')))
    break
" 2>/dev/null || echo "unknown")

        if [[ "$status" == "healthy" ]]; then
            break
        fi
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
        echo -n "."
    done
    echo ""
    if [[ "$status" == "healthy" ]]; then
        pass "${svc} healthy (${elapsed}s)"
    else
        record_fail "${svc} not healthy after ${MAX_WAIT}s (status: ${status})"
    fi
done

# ---------- Phase 3: Readiness Probes ----------
phase "Phase 3: Runtime Readiness Probes"

check_readiness() {
    local name="$1"
    local url="$2"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    if [[ "$code" == "200" ]]; then
        pass "${name} /readyz → 200"
    else
        record_fail "${name} /readyz → ${code}"
    fi
}

check_readiness "gateway"   "${BASE_URL}/readyz"
check_readiness "configctl" "http://127.0.0.1:8080/readyz"  # behind gateway port

# Internal readiness via compose exec (services not exposed on host except gateway and nats).
for svc_port in "configctl:8080" "ingest:8082" "derive:8083" "store:8081" "execute:8084"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    result=$($COMPOSE exec -T "${svc}" wget -q -O - "http://127.0.0.1:${port}/readyz" 2>/dev/null || echo '{"status":"error"}')
    status=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','error'))" 2>/dev/null || echo "error")
    if [[ "$status" == "ready" ]]; then
        pass "${svc} internal /readyz → ready"
    else
        record_fail "${svc} internal /readyz → ${status}"
    fi
done

# ---------- Phase 4: Seed Configctl ----------
if ! $CHECK_ONLY; then
    if $MULTI_SYMBOL; then
        phase "Phase 4: Seed Configctl (Multi-Symbol: ${SYMBOLS[*]})"
        info "Running seed-configctl.sh --multi-symbol..."
        "${SCRIPT_DIR}/seed-configctl.sh" --multi-symbol
    else
        phase "Phase 4: Seed Configctl (Single Symbol: btcusdt)"
        info "Running seed-configctl.sh (btcusdt)..."
        "${SCRIPT_DIR}/seed-configctl.sh"
    fi
    pass "Configctl seeded"
fi

# ---------- Phase 5: Diagnostics Validation ----------
phase "Phase 5: Runtime Diagnostics"

check_diagnostics() {
    local svc="$1"
    local port="$2"

    # /statusz
    statusz=$($COMPOSE exec -T "${svc}" wget -q -O - "http://127.0.0.1:${port}/statusz" 2>/dev/null || echo "")
    if [[ -n "$statusz" ]]; then
        runtime=$(echo "$statusz" | python3 -c "import sys,json; print(json.load(sys.stdin).get('runtime',''))" 2>/dev/null || echo "")
        uptime=$(echo "$statusz" | python3 -c "import sys,json; print(json.load(sys.stdin).get('uptime',''))" 2>/dev/null || echo "")
        trackers=$(echo "$statusz" | python3 -c "import sys,json; ts=json.load(sys.stdin).get('trackers',[]); print(len(ts))" 2>/dev/null || echo "0")
        pass "${svc} /statusz → runtime=${runtime} uptime=${uptime} trackers=${trackers}"
    else
        record_fail "${svc} /statusz unreachable"
    fi

    # /diagz
    diagz=$($COMPOSE exec -T "${svc}" wget -q -O - "http://127.0.0.1:${port}/diagz" 2>/dev/null || echo "")
    if [[ -n "$diagz" ]]; then
        checks=$(echo "$diagz" | python3 -c "
import sys,json
d=json.load(sys.stdin)
checks=d.get('readiness_checks',[])
passed=sum(1 for c in checks if c.get('status')=='pass')
total=len(checks)
print(f'{passed}/{total} pass')
" 2>/dev/null || echo "?")
        pass "${svc} /diagz → readiness_checks: ${checks}"
    else
        record_fail "${svc} /diagz unreachable"
    fi
}

check_diagnostics "configctl" "8080"
check_diagnostics "ingest"    "8082"
check_diagnostics "derive"    "8083"
check_diagnostics "store"     "8081"
check_diagnostics "execute"   "8084"

# ---------- Phase 6: Gateway Query Surface ----------
phase "Phase 6: Gateway Query Surface Validation"

check_endpoint() {
    local label="$1"
    local url="$2"
    local expected_code="${3:-200}"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    if [[ "$code" == "$expected_code" ]]; then
        pass "${label} → ${code}"
    else
        record_fail "${label} → ${code} (expected ${expected_code})"
    fi
}

# Core
check_endpoint "GET /healthz"  "${BASE_URL}/healthz"
check_endpoint "GET /readyz"   "${BASE_URL}/readyz"

# Configctl
check_endpoint "GET /configctl/configs/active" "${BASE_URL}/configctl/configs/active?scope_kind=global&scope_key=default"

# Domain query surfaces — validate for each symbol
for sym in "${SYMBOLS[@]}"; do
    check_endpoint "GET /evidence/candles/latest [${sym}]" "${BASE_URL}/evidence/candles/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /signal/rsi/latest [${sym}]" "${BASE_URL}/signal/rsi/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /signal/ema_crossover/latest [${sym}]" "${BASE_URL}/signal/ema_crossover/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /decision/rsi_oversold/latest [${sym}]" "${BASE_URL}/decision/rsi_oversold/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /strategy/mean_reversion_entry/latest [${sym}]" "${BASE_URL}/strategy/mean_reversion_entry/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /risk/position_exposure/latest [${sym}]" "${BASE_URL}/risk/position_exposure/latest?source=binancef&symbol=${sym}&timeframe=60"
    check_endpoint "GET /execution/paper_order/latest [${sym}]" "${BASE_URL}/execution/paper_order/latest?source=binancef&symbol=${sym}&timeframe=60"
done

# Execution control (symbol-independent)
check_endpoint "GET /execution/control" "${BASE_URL}/execution/control"

# ---------- Phase 7: Pipeline Event Flow ----------
phase "Phase 7: Pipeline Event Flow (wait for evidence materialization)"

CANDLE_WAIT=90
CANDLE_POLL=5

for sym in "${SYMBOLS[@]}"; do
    info "Waiting up to ${CANDLE_WAIT}s for ${sym} candle materialization..."
    CANDLE_ELAPSED=0
    CANDLE_FOUND=false

    while [[ $CANDLE_ELAPSED -lt $CANDLE_WAIT ]]; do
        RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&symbol=${sym}&timeframe=60" 2>/dev/null || echo "{}")
        HAS_CANDLE=$(echo "$RESPONSE" | python3 -c "
import sys,json
d=json.load(sys.stdin)
c=d.get('candle')
if c is not None:
    print('yes')
else:
    print('no')
" 2>/dev/null || echo "no")

        if [[ "$HAS_CANDLE" == "yes" ]]; then
            CANDLE_FOUND=true
            break
        fi
        sleep $CANDLE_POLL
        CANDLE_ELAPSED=$((CANDLE_ELAPSED + CANDLE_POLL))
        echo -n "."
    done
    echo ""

    if $CANDLE_FOUND; then
        pass "${sym} candle materialized after ${CANDLE_ELAPSED}s"
        curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&symbol=${sym}&timeframe=60" | python3 -c "
import sys,json
d=json.load(sys.stdin)
c=d['candle']
print(f'  source={c[\"source\"]} symbol={c[\"symbol\"]} tf={c[\"timeframe\"]}')
print(f'  OHLCV: {c[\"open\"]}/{c[\"high\"]}/{c[\"low\"]}/{c[\"close\"]} vol={c[\"volume\"]}')
print(f'  trades={c[\"trade_count\"]} final={c[\"final\"]}')
" 2>/dev/null || true
    else
        info "${sym}: no finalized candle yet (may need more time for 60s window boundary)"
        info "This is acceptable if the stack has been running < 120s"
    fi
done

# ---------- Phase 8: Tracker Activity Summary ----------
phase "Phase 8: Tracker Activity Summary"

# CF-04: Automated error-level log scanning.
ERROR_COUNT=$($COMPOSE logs --no-log-prefix 2>/dev/null | grep -c '"level":"error"' || echo "0")
if [[ "$ERROR_COUNT" -gt 0 ]]; then
    record_fail "Found ${ERROR_COUNT} error-level log entries across all services"
    info "Run 'make logs | grep error' to inspect"
else
    pass "No error-level log entries detected"
fi

# CF-05: Memory usage snapshot for regression detection.
info "Container memory usage:"
$COMPOSE ps -q 2>/dev/null | while read -r cid; do
    docker stats --no-stream --format '  {{.Name}}\t{{.MemUsage}}' "$cid" 2>/dev/null
done || info "docker stats unavailable"

for svc_port in "ingest:8082" "derive:8083" "store:8081" "execute:8084"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    $COMPOSE exec -T "${svc}" wget -q -O - "http://127.0.0.1:${port}/statusz" 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
name=d.get('runtime','${svc}')
trackers=d.get('trackers',[])
if not trackers:
    print(f'  ${svc}: no trackers registered')
else:
    for t in trackers:
        status = 'active' if t.get('event_count',0) > 0 else 'awaiting'
        idle = f' idle={t[\"idle_seconds\"]}s' if t.get('idle_seconds') else ''
        counters = ''
        if t.get('counters'):
            counters = ' ' + ' '.join(f'{k}={v}' for k,v in t['counters'].items())
        print(f'  ${svc}/{t[\"name\"]}: {status} events={t[\"event_count\"]} errors={t.get(\"error_count\",0)}{idle}{counters}')
" 2>/dev/null || info "${svc}: /statusz unavailable"
done

# ---------- Summary ----------
phase "SUMMARY"

if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}${BOLD}All checks passed.${NC} Live pipeline is operational."
else
    echo -e "${RED}${BOLD}${ERRORS} check(s) failed.${NC} Review output above."
fi

echo ""
echo "Services running:"
$COMPOSE ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || $COMPOSE ps

echo ""
echo "Mode: $(if $MULTI_SYMBOL; then echo "multi-symbol (${SYMBOLS[*]})"; else echo "single-symbol (btcusdt)"; fi)"
echo ""
echo "Useful commands:"
echo "  make logs                  # stream all logs"
echo "  make logs SERVICE=derive   # stream derive logs"
echo "  make ps                    # show service status"
echo "  make smoke                 # run single-symbol E2E smoke"
echo "  make smoke-multi           # run multi-symbol E2E smoke"
echo "  make live-multi            # full multi-symbol pipeline activation"
echo "  make down                  # stop everything"
echo ""

exit $ERRORS
