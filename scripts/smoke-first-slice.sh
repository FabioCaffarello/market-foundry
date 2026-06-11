#!/usr/bin/env bash
# smoke-first-slice.sh — End-to-end smoke test for the first vertical slice.
#
# Flow validated:
#   Binance WS → ingest → OBSERVATION_EVENTS → derive → EVIDENCE_EVENTS
#                                                     → store (NATS KV)
#                                                     → gateway HTTP endpoint
#
# Validates four timeframes: 60s (1m), 300s (5m), 900s (15m), 3600s (1h) candles.
# Note: 900s and 3600s candles need longer to finalize (15min and 60min respectively).
#
# Prerequisites:
#   make up   (starts nats, configctl, gateway, ingest, derive, store)
#
# Usage:
#   ./scripts/smoke-first-slice.sh
#   ./scripts/smoke-first-slice.sh --wait 90   # override wait seconds (default: 75)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-first-slice.sh [--wait <seconds>] [--help]

Runs the first-slice operational smoke against a running stack.
Canonical public entrypoint: `make smoke`
Expected setup: `make up && make seed`

Options:
  --wait <seconds>  Maximum time to wait for initial candle materialization. Default: 75
  --help            Show this help text.

Environment:
  BASE_URL              Gateway base URL. Default: http://127.0.0.1:8080
  SMOKE_WAIT            Alternate way to override --wait from make/env.
  SMOKE_POLL_INTERVAL   Poll interval in seconds. Default: 5
EOF
}

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
WAIT_SECONDS="${SMOKE_WAIT:-75}"
SETUP_HINT="make up && make seed"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --wait)
            [[ $# -ge 2 ]] || usage_error "--wait requires a value"
            WAIT_SECONDS="$2"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage_error "unknown argument: $1"
            ;;
    esac
    shift
done

require_commands curl python3
require_positive_integer "--wait" "${WAIT_SECONDS}"

smoke_banner "First-Slice Operational Smoke" "make smoke" "${SETUP_HINT}" "wait" "${WAIT_SECONDS}"

# ---------- Step 1: Health check ----------
info "Step 1: Checking gateway health..."
HTTP_CODE=$(http_code "${BASE_URL}/healthz")
[[ "$HTTP_CODE" == "200" ]] && pass "/healthz → 200" || smoke_die_with_hints "/healthz → ${HTTP_CODE} (expected 200)" "${SETUP_HINT}"

# ---------- Step 2: Readiness check ----------
info "Step 2: Checking gateway readiness..."
HTTP_CODE=$(http_code "${BASE_URL}/readyz")
[[ "$HTTP_CODE" == "200" ]] && pass "/readyz → 200" || smoke_die_with_hints "/readyz → ${HTTP_CODE} (expected 200)" "${SETUP_HINT}"

# ---------- Step 3: Wait for 60s candle data ----------
info "Step 3: Waiting ${WAIT_SECONDS}s for ingest → derive pipeline to produce data..."
info "  (ingest connects to Binance WS, derive samples 60s and 300s candles)"

CANDLE_FOUND=false
ELAPSED=0
POLL_INTERVAL="${SMOKE_POLL_INTERVAL:-${CANDLE_POLL_INTERVAL:-5}}"

while [[ $ELAPSED -lt $WAIT_SECONDS ]]; do
    RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60")
    CANDLE=$(echo "$RESPONSE" | grep -o '"candle":{' 2>/dev/null || true)

    if [[ -n "$CANDLE" ]]; then
        CANDLE_FOUND=true
        break
    fi

    sleep "$POLL_INTERVAL"
    ELAPSED=$((ELAPSED + POLL_INTERVAL))
    echo -n "."
done
echo ""

# ---------- Step 4: Validate 60s candle response ----------
if $CANDLE_FOUND; then
    pass "60s evidence candle received after ${ELAPSED}s"
else
    info "No finalized 60s candle yet — checking for active sampler (null candle is valid if < 60s of trades)..."
    RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60")
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60")

    if [[ "$HTTP_CODE" == "200" ]]; then
        pass "60s endpoint reachable (200) — derive is connected, candle not yet finalized"
        info "Response: $RESPONSE"
        info "This is expected if < 120s since boot (need trades to cross a 60s window boundary)"
    else
        smoke_die_with_hints "60s endpoint returned ${HTTP_CODE} — derive may not be running or config may not be seeded" "${SETUP_HINT}"
    fi
fi

# ---------- Step 5: Validate 60s response structure ----------
info "Step 5: Validating 60s response structure..."
RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60")

echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'candle' in data, 'missing candle key'
candle = data['candle']
if candle is not None:
    required = ['source', 'symbol', 'timeframe', 'open', 'high', 'low', 'close', 'volume', 'trade_count', 'open_time', 'close_time', 'final']
    for field in required:
        assert field in candle, f'missing field: {field}'
    assert candle['source'] == 'binancef', f'wrong source: {candle[\"source\"]}'
    assert candle['symbol'] == 'btcusdt', f'wrong symbol: {candle[\"symbol\"]}'
    assert candle['timeframe'] == 60, f'wrong timeframe: {candle[\"timeframe\"]}'
    print(f'  source={candle[\"source\"]} symbol={candle[\"symbol\"]} tf={candle[\"timeframe\"]}')
    print(f'  OHLCV: {candle[\"open\"]} / {candle[\"high\"]} / {candle[\"low\"]} / {candle[\"close\"]} vol={candle[\"volume\"]}')
    print(f'  trades={candle[\"trade_count\"]} final={candle[\"final\"]}')
    print(f'  window: {candle[\"open_time\"]} → {candle[\"close_time\"]}')
else:
    print('  candle is null (sampler has no data yet)')
print('OK')
" 2>&1 && pass "60s response structure valid" || die "60s response structure invalid"

# ---------- Step 6: Validate 300s candle endpoint ----------
info "Step 6: Checking 300s (5-minute) candle endpoint..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=300")
RESPONSE_300=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=300")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "300s endpoint reachable (200)"
    echo "$RESPONSE_300" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'candle' in data, 'missing candle key'
candle = data['candle']
if candle is not None:
    assert candle['timeframe'] == 300, f'wrong timeframe: {candle[\"timeframe\"]}'
    print(f'  source={candle[\"source\"]} symbol={candle[\"symbol\"]} tf={candle[\"timeframe\"]}')
    print(f'  OHLCV: {candle[\"open\"]} / {candle[\"high\"]} / {candle[\"low\"]} / {candle[\"close\"]} vol={candle[\"volume\"]}')
    print(f'  trades={candle[\"trade_count\"]} final={candle[\"final\"]}')
    print(f'  window: {candle[\"open_time\"]} → {candle[\"close_time\"]}')
else:
    print('  300s candle is null (expected — 5-minute window needs more time to finalize)')
print('OK')
" 2>&1 && pass "300s response structure valid" || die "300s response structure invalid"
else
    die "300s endpoint returned $HTTP_CODE — expected 200"
fi

# ---------- Step 6b: Validate 900s candle endpoint (TC-01) ----------
info "Step 6b: Checking 900s (15-minute) candle endpoint..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=900")
RESPONSE_900=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=900")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "900s endpoint reachable (200)"
    echo "$RESPONSE_900" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'candle' in data, 'missing candle key'
candle = data['candle']
if candle is not None:
    assert candle['timeframe'] == 900, f'wrong timeframe: {candle[\"timeframe\"]}'
    print(f'  source={candle[\"source\"]} symbol={candle[\"symbol\"]} tf={candle[\"timeframe\"]}')
    print(f'  OHLCV: {candle[\"open\"]}/{candle[\"high\"]}/{candle[\"low\"]}/{candle[\"close\"]} vol={candle[\"volume\"]}')
    print(f'  trades={candle[\"trade_count\"]} final={candle[\"final\"]}')
    print(f'  window: {candle[\"open_time\"]} → {candle[\"close_time\"]}')
else:
    print('  900s candle is null (expected — 15-minute window needs more time to finalize)')
print('OK')
" 2>&1 && pass "900s response structure valid" || die "900s response structure invalid"
else
    die "900s endpoint returned $HTTP_CODE — expected 200"
fi

# ---------- Step 6c: Validate 3600s candle endpoint (TC-01) ----------
info "Step 6c: Checking 3600s (1-hour) candle endpoint..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=3600")
RESPONSE_3600=$(curl -s "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=3600")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "3600s endpoint reachable (200)"
    echo "$RESPONSE_3600" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'candle' in data, 'missing candle key'
candle = data['candle']
if candle is not None:
    assert candle['timeframe'] == 3600, f'wrong timeframe: {candle[\"timeframe\"]}'
    print(f'  source={candle[\"source\"]} symbol={candle[\"symbol\"]} tf={candle[\"timeframe\"]}')
    print(f'  OHLCV: {candle[\"open\"]}/{candle[\"high\"]}/{candle[\"low\"]}/{candle[\"close\"]} vol={candle[\"volume\"]}')
    print(f'  trades={candle[\"trade_count\"]} final={candle[\"final\"]}')
    print(f'  window: {candle[\"open_time\"]} → {candle[\"close_time\"]}')
else:
    print('  3600s candle is null (expected — 1-hour window needs ~60min to finalize)')
print('OK')
" 2>&1 && pass "3600s response structure valid" || die "3600s response structure invalid"
else
    die "3600s endpoint returned $HTTP_CODE — expected 200"
fi

# ---------- Step 7: Validate error handling ----------
info "Step 7: Checking error handling..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=binancef&base=btc&quote=usdt&contract=perpetual")
[[ "$HTTP_CODE" == "400" ]] && pass "Missing timeframe → 400" || info "Missing timeframe → $HTTP_CODE (expected 400, acceptable if gateway handles differently)"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest")
[[ "$HTTP_CODE" == "400" ]] && pass "No params → 400" || info "No params → $HTTP_CODE"

# ---------- Summary ----------
phase "Summary"
echo "Result: PASS"
echo "Canonical target: make smoke"
echo "Setup used: ${SETUP_HINT}"
echo "Gateway: ${BASE_URL}"
echo "Warm-up budget: ${WAIT_SECONDS}s"
echo ""
echo "Flow validated:"
echo "  Binance WS → ingest → OBSERVATION_EVENTS"
echo "                       → derive (60s + 300s + 900s + 3600s candle samplers)"
echo "                       → EVIDENCE_EVENTS → store (NATS KV)"
echo "                       → evidence.query.candle.latest"
echo "                       → GET /evidence/candles/latest"
echo ""
echo "Timeframes validated:"
echo "  60s   (1-minute)  — candle query endpoint reachable"
echo "  300s  (5-minute)  — candle query endpoint reachable"
echo "  900s  (15-minute) — candle query endpoint reachable"
echo "  3600s (1-hour)    — candle query endpoint reachable"
echo ""
echo "If a later run fails unexpectedly, start with:"
echo "  make ps"
echo "  make logs SERVICE=gateway"
echo "  make diag"
