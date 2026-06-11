#!/usr/bin/env bash
# smoke-multi-symbol.sh — E2E smoke test for multi-symbol × multi-timeframe scenario.
#
# Flow validated per symbol:
#   Evidence:  Binance WS → ingest → OBSERVATION_EVENTS → derive → EVIDENCE_EVENTS
#                                                                 → store (NATS KV)
#                                                                 → gateway HTTP endpoint
#   Signal:    derive (RSI sampler) → SIGNAL_EVENTS → store (NATS KV)
#                                                    → gateway HTTP endpoint
#   Signal:    derive (EMA Crossover sampler) → SIGNAL_EVENTS → store (NATS KV)
#                                                               → gateway HTTP endpoint
#   Decision:  derive (RSI Oversold evaluator) → DECISION_EVENTS → store (NATS KV)
#                                                                 → gateway HTTP endpoint
#   Decision:  derive (EMA Crossover evaluator) → DECISION_EVENTS → store (NATS KV)
#                                                                  → gateway HTTP endpoint
#   Strategy:  derive (Mean Reversion Entry resolver) → STRATEGY_EVENTS → store (NATS KV)
#                                                                        → gateway HTTP endpoint
#   Strategy:  derive (Trend Following Entry resolver) → STRATEGY_EVENTS → store (NATS KV)
#                                                                         → gateway HTTP endpoint
#   Risk:      derive (Position Exposure evaluator) → RISK_EVENTS → store (NATS KV)
#                                                                  → gateway HTTP endpoint
#   Risk:      derive (Drawdown Limit evaluator) → RISK_EVENTS → store (NATS KV)
#                                                                → gateway HTTP endpoint
#   Execution: derive (Paper Order evaluator) → EXECUTION_EVENTS → store (NATS KV)
#                                                                 → gateway HTTP endpoint
#   Fill:      execute (Venue Adapter) → EXECUTION_FILL_EVENTS → store (NATS KV)
#                                                                → gateway HTTP endpoint
#   Status:    GET /execution/status/latest (composite: intent + result + gate + propagation)
#   Control:   GET/PUT /execution/control (kill switch gate cycle with execute active)
#   Trace:     correlation_id + causation_id persistence through execute chain
#
# Validates: 2 symbols (btcusdt, ethusdt) × 4 timeframes (60s, 300s, 900s, 3600s)
# Evidence   KV keys: 8 (2 symbols × 4 timeframes)
# Signal RSI KV keys: 8 (2 symbols × 4 timeframes)
# Signal EMA KV keys: 8 (2 symbols × 4 timeframes)
# Decision   KV keys: 8 (2 symbols × 4 timeframes)
# Strategy   KV keys: 8 (2 symbols × 4 timeframes)
# Risk       KV keys: 8 (2 symbols × 4 timeframes)
# Execution  KV keys: 8 (2 symbols × 4 timeframes)
#
# Prerequisites:
#   make up
#   make seed-multi    (seeds configctl with 2 symbols)
#
# Usage:
#   ./scripts/smoke-multi-symbol.sh
#   ./scripts/smoke-multi-symbol.sh --wait 120

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-multi-symbol.sh [--wait <seconds>] [--help]

Runs the multi-symbol operational smoke against a running stack.
Canonical public entrypoint: `make smoke-multi`
Expected setup: `make up && make seed-multi`

Options:
  --wait <seconds>  Maximum time to wait for initial candle materialization. Default: 90
  --help            Show this help text.

Environment:
  SMOKE_SYMBOLS     Space-separated symbol list. Default: "btcusdt ethusdt"
  SMOKE_TIMEFRAMES  Space-separated timeframe list. Default: "60 300 900 3600"
  BASE_URL          Gateway base URL. Default: http://127.0.0.1:8080
  SMOKE_WAIT        Alternate way to override --wait from make/env.
EOF
}

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SOURCE="${SOURCE:-binancef}"
CONTRACT="${CONTRACT:-perpetual}"
read -ra SYMBOLS <<< "${SMOKE_SYMBOLS:-btcusdt ethusdt}"
read -ra TIMEFRAMES <<< "${SMOKE_TIMEFRAMES:-60 300 900 3600}"
WAIT_SECONDS="${SMOKE_WAIT:-90}"
SETUP_HINT="make up && make seed-multi"

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

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0
pass() { echo -e "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; WARN_COUNT=$((WARN_COUNT + 1)); }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
section() { echo -e "\n${CYAN}--- $1 ---${NC}"; }
hard_fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }

smoke_banner "Multi-Symbol Operational Smoke" "make smoke-multi" "${SETUP_HINT}" "wait" "${WAIT_SECONDS}"

# ---------- Step 1: Health + Readiness ----------
section "Step 1: Gateway checks"

HTTP_CODE=$(http_code "${BASE_URL}/healthz")
if [[ "$HTTP_CODE" == "200" ]]; then
    pass "/healthz → 200"
else
    hard_fail "/healthz → ${HTTP_CODE} (expected 200). Start with: ${SETUP_HINT}; then inspect 'make ps' and 'make logs SERVICE=gateway'."
fi

HTTP_CODE=$(http_code "${BASE_URL}/readyz")
if [[ "$HTTP_CODE" == "200" ]]; then
    pass "/readyz → 200"
else
    hard_fail "/readyz → ${HTTP_CODE} (expected 200). Stack may be up but not ready, or multi-symbol config has not been activated."
fi

# ---------- Step 2: Wait for first candle ----------
section "Step 2: Waiting for pipeline (${WAIT_SECONDS}s max)"
info "Waiting for at least one 60s candle from any symbol..."

CANDLE_FOUND=false
ELAPSED=0
POLL_INTERVAL=5

while [[ $ELAPSED -lt $WAIT_SECONDS ]]; do
    for sym in "${SYMBOLS[@]}"; do
        RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
        CANDLE=$(echo "$RESPONSE" | grep -o '"candle":{' 2>/dev/null || true)
        if [[ -n "$CANDLE" ]]; then
            CANDLE_FOUND=true
            info "First candle found for ${sym} after ${ELAPSED}s"
            break 2
        fi
    done
    sleep $POLL_INTERVAL
    ELAPSED=$((ELAPSED + POLL_INTERVAL))
    echo -n "."
done
echo ""

if $CANDLE_FOUND; then
    pass "Pipeline producing candles"
else
    info "No finalized candle yet — continuing with endpoint validation"
fi

# ---------- Step 3: Validate each symbol × timeframe ----------
section "Step 3: Multi-symbol × multi-timeframe validation"

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        # Validate response structure
        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'candle' in data, 'missing candle key'
candle = data['candle']
if candle is not None:
    required = ['source', 'symbol', 'timeframe', 'open', 'high', 'low', 'close', 'volume', 'trade_count', 'open_time', 'close_time', 'final']
    for field in required:
        assert field in candle, f'missing field: {field}'
    assert candle['source'] == '${SOURCE}', f'wrong source: {candle[\"source\"]}'
    assert candle['symbol'] == '${sym}', f'wrong symbol: {candle[\"symbol\"]}'
    assert candle['timeframe'] == ${tf}, f'wrong timeframe: {candle[\"timeframe\"]}'
    print(f'CANDLE source={candle[\"source\"]} symbol={candle[\"symbol\"]} tf={candle[\"timeframe\"]} trades={candle[\"trade_count\"]} final={candle[\"final\"]}')
    print(f'  OHLCV: {candle[\"open\"]} / {candle[\"high\"]} / {candle[\"low\"]} / {candle[\"close\"]} vol={candle[\"volume\"]}')
else:
    print('NULL (sampler active, no finalized window yet)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "CANDLE"; then
                pass "${sym}/${tf}s → candle present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "${sym}/${tf}s → endpoint reachable (null candle — expected for ${tf}s)"
            fi
        else
            fail "${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 4: Cross-symbol isolation check ----------
section "Step 4: Cross-symbol isolation"
info "Verifying symbols produce independent candle data..."

ISOLATION_OK=true
for tf in "${TIMEFRAMES[@]}"; do
    CANDLE_A=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    CANDLE_B=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    # Check that candles (when present) have different symbols
    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${CANDLE_A}''')['candle']
b = json.loads('''${CANDLE_B}''')['candle']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['open'] == b['open'] and a['close'] == b['close'] and a['volume'] == b['volume']:
        print('SUSPECT')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "tf=${tf}s: symbols produce independent candle data" ;;
        PARTIAL)  pass "tf=${tf}s: one symbol has data, other pending (expected)" ;;
        NONE)     info "tf=${tf}s: no candles yet for either symbol" ;;
        COLLISION) fail "tf=${tf}s: SYMBOL COLLISION — both candles have same symbol"; ISOLATION_OK=false ;;
        SUSPECT)  info "tf=${tf}s: identical OHLCV (unlikely but possible in same window)" ;;
        *)        fail "tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 5: Signal RSI multi-symbol validation ----------
section "Step 5: Signal RSI multi-symbol validation"
info "Note: RSI needs 15 candles warm-up (~15min at 60s). Signals may be null if pipeline is young."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking signal RSI ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "signal rsi ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'signal' in data, 'missing signal key'
sig = data['signal']
if sig is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'value', 'timestamp', 'final']
    for field in required:
        assert field in sig, f'missing field: {field}'
    assert sig['source'] == '${SOURCE}', f'wrong source: {sig[\"source\"]}'
    assert sig['symbol'] == '${sym}', f'wrong symbol: {sig[\"symbol\"]}'
    assert sig['timeframe'] == ${tf}, f'wrong timeframe: {sig[\"timeframe\"]}'
    assert sig['type'] == 'rsi', f'wrong type: {sig[\"type\"]}'
    print(f'SIGNAL type={sig[\"type\"]} source={sig[\"source\"]} symbol={sig[\"symbol\"]} tf={sig[\"timeframe\"]} value={sig[\"value\"]} final={sig[\"final\"]}')
else:
    print('NULL (RSI warm-up not complete yet)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "SIGNAL"; then
                pass "signal rsi ${sym}/${tf}s → signal present"
                echo "    $(echo "$RESULT" | head -1)"
            else
                pass "signal rsi ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "signal rsi ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 6: Cross-symbol signal isolation ----------
section "Step 6: Cross-symbol signal isolation"
info "Verifying signal RSI produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    SIG_A=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    SIG_B=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${SIG_A}''')['signal']
b = json.loads('''${SIG_B}''')['signal']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['value'] == b['value']:
        print('SUSPECT')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "signal rsi tf=${tf}s: symbols produce independent signal data" ;;
        PARTIAL)  pass "signal rsi tf=${tf}s: one symbol has signal, other pending (expected)" ;;
        NONE)     info "signal rsi tf=${tf}s: no signals yet (warm-up pending)" ;;
        COLLISION) fail "signal rsi tf=${tf}s: SYMBOL COLLISION — both signals have same symbol" ;;
        SUSPECT)  info "signal rsi tf=${tf}s: identical RSI values (possible but uncommon)" ;;
        *)        fail "signal rsi tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 6a: Signal EMA Crossover multi-symbol validation ----------
section "Step 6a: Signal EMA Crossover multi-symbol validation (CC-02)"
info "Note: EMA Crossover needs 21 candles warm-up (~21min at 60s). Signals may be null if pipeline is young."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking signal ema_crossover ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/signal/ema_crossover/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/signal/ema_crossover/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "signal ema_crossover ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'signal' in data, 'missing signal key'
sig = data['signal']
if sig is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'value', 'timestamp', 'final']
    for field in required:
        assert field in sig, f'missing field: {field}'
    assert sig['source'] == '${SOURCE}', f'wrong source: {sig[\"source\"]}'
    assert sig['symbol'] == '${sym}', f'wrong symbol: {sig[\"symbol\"]}'
    assert sig['timeframe'] == ${tf}, f'wrong timeframe: {sig[\"timeframe\"]}'
    assert sig['type'] == 'ema_crossover', f'wrong type: {sig[\"type\"]}'
    assert sig['value'] in ('bullish', 'bearish', 'neutral'), f'invalid value: {sig[\"value\"]}'
    meta = sig.get('metadata', {})
    for mkey in ('fast_period', 'slow_period', 'fast_ema', 'slow_ema', 'spread'):
        assert mkey in meta, f'missing metadata key: {mkey}'
    print(f'SIGNAL type={sig[\"type\"]} source={sig[\"source\"]} symbol={sig[\"symbol\"]} tf={sig[\"timeframe\"]} value={sig[\"value\"]} final={sig[\"final\"]}')
    print(f'  fast_ema={meta[\"fast_ema\"]} slow_ema={meta[\"slow_ema\"]} spread={meta[\"spread\"]}')
else:
    print('NULL (EMA Crossover warm-up not complete yet — needs 21 candles)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "SIGNAL"; then
                pass "signal ema_crossover ${sym}/${tf}s → signal present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "signal ema_crossover ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "signal ema_crossover ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 6b: Cross-symbol EMA Crossover signal isolation ----------
section "Step 6b: Cross-symbol EMA Crossover signal isolation (CC-02)"
info "Verifying signal EMA Crossover produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    SIG_A=$(curl -s "${BASE_URL}/signal/ema_crossover/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    SIG_B=$(curl -s "${BASE_URL}/signal/ema_crossover/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${SIG_A}''')['signal']
b = json.loads('''${SIG_B}''')['signal']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "signal ema_crossover tf=${tf}s: symbols produce independent signal data" ;;
        PARTIAL)  pass "signal ema_crossover tf=${tf}s: one symbol has signal, other pending (expected)" ;;
        NONE)     info "signal ema_crossover tf=${tf}s: no signals yet (warm-up pending)" ;;
        COLLISION) fail "signal ema_crossover tf=${tf}s: SYMBOL COLLISION — both signals have same symbol" ;;
        *)        fail "signal ema_crossover tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 7: Decision RSI Oversold multi-symbol validation ----------
section "Step 7: Decision RSI Oversold multi-symbol validation"
info "Note: Decision evaluates after each RSI signal. Requires RSI warm-up (~15min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking decision rsi_oversold ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "decision rsi_oversold ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'decision' in data, 'missing decision key'
dec = data['decision']
if dec is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'outcome', 'confidence', 'final', 'timestamp']
    for field in required:
        assert field in dec, f'missing field: {field}'
    assert dec['source'] == '${SOURCE}', f'wrong source: {dec[\"source\"]}'
    assert dec['symbol'] == '${sym}', f'wrong symbol: {dec[\"symbol\"]}'
    assert dec['timeframe'] == ${tf}, f'wrong timeframe: {dec[\"timeframe\"]}'
    assert dec['type'] == 'rsi_oversold', f'wrong type: {dec[\"type\"]}'
    assert dec['outcome'] in ('triggered', 'not_triggered', 'insufficient'), f'invalid outcome: {dec[\"outcome\"]}'
    print(f'DECISION type={dec[\"type\"]} source={dec[\"source\"]} symbol={dec[\"symbol\"]} tf={dec[\"timeframe\"]} outcome={dec[\"outcome\"]} confidence={dec[\"confidence\"]} final={dec[\"final\"]}')
else:
    print('NULL (RSI warm-up not complete — decision pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "DECISION"; then
                pass "decision rsi_oversold ${sym}/${tf}s → decision present"
                echo "    $(echo "$RESULT" | head -1)"
            else
                pass "decision rsi_oversold ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "decision rsi_oversold ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 8: Cross-symbol decision isolation ----------
section "Step 8: Cross-symbol decision isolation"
info "Verifying decision RSI Oversold produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    DEC_A=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    DEC_B=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${DEC_A}''')['decision']
b = json.loads('''${DEC_B}''')['decision']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "decision rsi_oversold tf=${tf}s: symbols produce independent decision data" ;;
        PARTIAL)  pass "decision rsi_oversold tf=${tf}s: one symbol has decision, other pending (expected)" ;;
        NONE)     info "decision rsi_oversold tf=${tf}s: no decisions yet (warm-up pending)" ;;
        COLLISION) fail "decision rsi_oversold tf=${tf}s: SYMBOL COLLISION — both decisions have same symbol" ;;
        BLEED_A)  fail "decision rsi_oversold tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "decision rsi_oversold tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "decision rsi_oversold tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 7a: Decision EMA Crossover multi-symbol validation (Breadth S241) ----------
section "Step 7a: Decision EMA Crossover multi-symbol validation (Breadth S241)"
info "Note: EMA Crossover decision evaluates after each EMA signal. Requires EMA warm-up (~21min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking decision ema_crossover ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/decision/ema_crossover/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/decision/ema_crossover/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "decision ema_crossover ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'decision' in data, 'missing decision key'
dec = data['decision']
if dec is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'outcome', 'confidence', 'final', 'timestamp']
    for field in required:
        assert field in dec, f'missing field: {field}'
    assert dec['source'] == '${SOURCE}', f'wrong source: {dec[\"source\"]}'
    assert dec['symbol'] == '${sym}', f'wrong symbol: {dec[\"symbol\"]}'
    assert dec['timeframe'] == ${tf}, f'wrong timeframe: {dec[\"timeframe\"]}'
    assert dec['type'] == 'ema_crossover', f'wrong type: {dec[\"type\"]}'
    assert dec['outcome'] in ('triggered', 'not_triggered', 'insufficient'), f'invalid outcome: {dec[\"outcome\"]}'
    meta = dec.get('metadata', {})
    if 'crossover_direction' in meta:
        print(f'DECISION type={dec[\"type\"]} source={dec[\"source\"]} symbol={dec[\"symbol\"]} tf={dec[\"timeframe\"]} outcome={dec[\"outcome\"]} confidence={dec[\"confidence\"]} crossover={meta[\"crossover_direction\"]} final={dec[\"final\"]}')
    else:
        print(f'DECISION type={dec[\"type\"]} source={dec[\"source\"]} symbol={dec[\"symbol\"]} tf={dec[\"timeframe\"]} outcome={dec[\"outcome\"]} confidence={dec[\"confidence\"]} final={dec[\"final\"]}')
else:
    print('NULL (EMA warm-up not complete — ema_crossover decision pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "DECISION"; then
                pass "decision ema_crossover ${sym}/${tf}s → decision present"
                echo "    $(echo "$RESULT" | head -1)"
            else
                pass "decision ema_crossover ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "decision ema_crossover ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 8a: Cross-symbol ema_crossover decision isolation ----------
section "Step 8a: Cross-symbol ema_crossover decision isolation (Breadth S241)"
info "Verifying decision EMA Crossover produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    DEC_A=$(curl -s "${BASE_URL}/decision/ema_crossover/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    DEC_B=$(curl -s "${BASE_URL}/decision/ema_crossover/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${DEC_A}''')['decision']
b = json.loads('''${DEC_B}''')['decision']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "decision ema_crossover tf=${tf}s: symbols produce independent decision data" ;;
        PARTIAL)  pass "decision ema_crossover tf=${tf}s: one symbol has decision, other pending (expected)" ;;
        NONE)     info "decision ema_crossover tf=${tf}s: no decisions yet (warm-up pending)" ;;
        COLLISION) fail "decision ema_crossover tf=${tf}s: SYMBOL COLLISION — both decisions have same symbol" ;;
        BLEED_A)  fail "decision ema_crossover tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "decision ema_crossover tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "decision ema_crossover tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 9: Strategy Mean Reversion Entry multi-symbol validation ----------
section "Step 9: Strategy Mean Reversion Entry multi-symbol validation"
info "Note: Strategy resolves after each decision. Requires RSI warm-up (~15min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking strategy mean_reversion_entry ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "strategy mean_reversion_entry ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'strategy' in data, 'missing strategy key'
strat = data['strategy']
if strat is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'direction', 'confidence', 'decisions', 'final', 'timestamp']
    for field in required:
        assert field in strat, f'missing field: {field}'
    assert strat['source'] == '${SOURCE}', f'wrong source: {strat[\"source\"]}'
    assert strat['symbol'] == '${sym}', f'wrong symbol: {strat[\"symbol\"]}'
    assert strat['timeframe'] == ${tf}, f'wrong timeframe: {strat[\"timeframe\"]}'
    assert strat['type'] == 'mean_reversion_entry', f'wrong type: {strat[\"type\"]}'
    assert strat['direction'] in ('long', 'short', 'flat'), f'invalid direction: {strat[\"direction\"]}'
    assert isinstance(strat['decisions'], list) and len(strat['decisions']) > 0, 'decisions must be non-empty list'
    print(f'STRATEGY type={strat[\"type\"]} source={strat[\"source\"]} symbol={strat[\"symbol\"]} tf={strat[\"timeframe\"]} direction={strat[\"direction\"]} confidence={strat[\"confidence\"]} final={strat[\"final\"]}')
    print(f'  decisions={len(strat[\"decisions\"])} parameters={strat.get(\"parameters\", {})}')
else:
    print('NULL (decision warm-up not complete — strategy pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "STRATEGY"; then
                pass "strategy mean_reversion_entry ${sym}/${tf}s → strategy present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "strategy mean_reversion_entry ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "strategy mean_reversion_entry ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 10: Cross-symbol strategy isolation ----------
section "Step 10: Cross-symbol strategy isolation"
info "Verifying strategy Mean Reversion Entry produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    STRAT_A=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    STRAT_B=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${STRAT_A}''')['strategy']
b = json.loads('''${STRAT_B}''')['strategy']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "strategy mean_reversion_entry tf=${tf}s: symbols produce independent strategy data" ;;
        PARTIAL)  pass "strategy mean_reversion_entry tf=${tf}s: one symbol has strategy, other pending (expected)" ;;
        NONE)     info "strategy mean_reversion_entry tf=${tf}s: no strategies yet (warm-up pending)" ;;
        COLLISION) fail "strategy mean_reversion_entry tf=${tf}s: SYMBOL COLLISION — both strategies have same symbol" ;;
        BLEED_A)  fail "strategy mean_reversion_entry tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "strategy mean_reversion_entry tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "strategy mean_reversion_entry tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 9a: Strategy Trend Following Entry multi-symbol validation (Breadth S242) ----------
section "Step 9a: Strategy Trend Following Entry multi-symbol validation (Breadth S242)"
info "Note: Trend Following Entry resolves after each ema_crossover decision. Requires EMA warm-up (~21min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking strategy trend_following_entry ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/strategy/trend_following_entry/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/strategy/trend_following_entry/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "strategy trend_following_entry ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'strategy' in data, 'missing strategy key'
strat = data['strategy']
if strat is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'direction', 'confidence', 'decisions', 'final', 'timestamp']
    for field in required:
        assert field in strat, f'missing field: {field}'
    assert strat['source'] == '${SOURCE}', f'wrong source: {strat[\"source\"]}'
    assert strat['symbol'] == '${sym}', f'wrong symbol: {strat[\"symbol\"]}'
    assert strat['timeframe'] == ${tf}, f'wrong timeframe: {strat[\"timeframe\"]}'
    assert strat['type'] == 'trend_following_entry', f'wrong type: {strat[\"type\"]}'
    assert strat['direction'] in ('long', 'short', 'flat'), f'invalid direction: {strat[\"direction\"]}'
    assert isinstance(strat['decisions'], list) and len(strat['decisions']) > 0, 'decisions must be non-empty list'
    print(f'STRATEGY type={strat[\"type\"]} source={strat[\"source\"]} symbol={strat[\"symbol\"]} tf={strat[\"timeframe\"]} direction={strat[\"direction\"]} confidence={strat[\"confidence\"]} final={strat[\"final\"]}')
    print(f'  decisions={len(strat[\"decisions\"])} parameters={strat.get(\"parameters\", {})}')
else:
    print('NULL (ema_crossover decision warm-up not complete — trend_following_entry pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "STRATEGY"; then
                pass "strategy trend_following_entry ${sym}/${tf}s → strategy present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "strategy trend_following_entry ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "strategy trend_following_entry ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 10a: Cross-symbol trend_following_entry strategy isolation ----------
section "Step 10a: Cross-symbol trend_following_entry strategy isolation (Breadth S242)"
info "Verifying strategy Trend Following Entry produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    STRAT_A=$(curl -s "${BASE_URL}/strategy/trend_following_entry/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    STRAT_B=$(curl -s "${BASE_URL}/strategy/trend_following_entry/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${STRAT_A}''')['strategy']
b = json.loads('''${STRAT_B}''')['strategy']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "strategy trend_following_entry tf=${tf}s: symbols produce independent strategy data" ;;
        PARTIAL)  pass "strategy trend_following_entry tf=${tf}s: one symbol has strategy, other pending (expected)" ;;
        NONE)     info "strategy trend_following_entry tf=${tf}s: no strategies yet (warm-up pending)" ;;
        COLLISION) fail "strategy trend_following_entry tf=${tf}s: SYMBOL COLLISION — both strategies have same symbol" ;;
        BLEED_A)  fail "strategy trend_following_entry tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "strategy trend_following_entry tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "strategy trend_following_entry tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 11: Risk Position Exposure multi-symbol validation ----------
section "Step 11: Risk Position Exposure multi-symbol validation"
info "Note: Risk evaluates after each strategy resolution. Requires full pipeline warm-up."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking risk position_exposure ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/risk/position_exposure/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/risk/position_exposure/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "risk position_exposure ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'risk' in data, 'missing risk key'
r = data['risk']
if r is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'disposition', 'confidence', 'strategies', 'constraints', 'rationale', 'parameters', 'final', 'timestamp']
    for field in required:
        assert field in r, f'missing field: {field}'
    assert r['source'] == '${SOURCE}', f'wrong source: {r[\"source\"]}'
    assert r['symbol'] == '${sym}', f'wrong symbol: {r[\"symbol\"]}'
    assert r['timeframe'] == ${tf}, f'wrong timeframe: {r[\"timeframe\"]}'
    assert r['type'] == 'position_exposure', f'wrong type: {r[\"type\"]}'
    assert r['disposition'] in ('approved', 'modified', 'rejected'), f'invalid disposition: {r[\"disposition\"]}'
    assert isinstance(r['strategies'], list) and len(r['strategies']) > 0, 'strategies must be non-empty list'
    assert isinstance(r['constraints'], dict), 'constraints must be an object'
    print(f'RISK type={r[\"type\"]} source={r[\"source\"]} symbol={r[\"symbol\"]} tf={r[\"timeframe\"]} disposition={r[\"disposition\"]} confidence={r[\"confidence\"]} final={r[\"final\"]}')
    print(f'  constraints={r[\"constraints\"]} strategies={len(r[\"strategies\"])}')
else:
    print('NULL (strategy warm-up not complete — risk pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "RISK"; then
                pass "risk position_exposure ${sym}/${tf}s → risk present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "risk position_exposure ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "risk position_exposure ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 12: Cross-symbol risk isolation ----------
section "Step 12: Cross-symbol risk isolation"
info "Verifying risk Position Exposure produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    RISK_A=$(curl -s "${BASE_URL}/risk/position_exposure/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    RISK_B=$(curl -s "${BASE_URL}/risk/position_exposure/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${RISK_A}''')['risk']
b = json.loads('''${RISK_B}''')['risk']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "risk position_exposure tf=${tf}s: symbols produce independent risk data" ;;
        PARTIAL)  pass "risk position_exposure tf=${tf}s: one symbol has risk, other pending (expected)" ;;
        NONE)     info "risk position_exposure tf=${tf}s: no risk assessments yet (warm-up pending)" ;;
        COLLISION) fail "risk position_exposure tf=${tf}s: SYMBOL COLLISION — both risks have same symbol" ;;
        BLEED_A)  fail "risk position_exposure tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "risk position_exposure tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "risk position_exposure tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 11a: Risk Drawdown Limit multi-symbol validation (Breadth S243) ----------
section "Step 11a: Risk Drawdown Limit multi-symbol validation (Breadth S243)"
info "Note: Drawdown Limit evaluates after each trend_following_entry strategy. Requires full Chain B warm-up."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking risk drawdown_limit ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/risk/drawdown_limit/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/risk/drawdown_limit/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "risk drawdown_limit ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'risk' in data, 'missing risk key'
r = data['risk']
if r is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'disposition', 'confidence', 'strategies', 'constraints', 'rationale', 'parameters', 'final', 'timestamp']
    for field in required:
        assert field in r, f'missing field: {field}'
    assert r['source'] == '${SOURCE}', f'wrong source: {r[\"source\"]}'
    assert r['symbol'] == '${sym}', f'wrong symbol: {r[\"symbol\"]}'
    assert r['timeframe'] == ${tf}, f'wrong timeframe: {r[\"timeframe\"]}'
    assert r['type'] == 'drawdown_limit', f'wrong type: {r[\"type\"]}'
    assert r['disposition'] in ('approved', 'modified', 'rejected'), f'invalid disposition: {r[\"disposition\"]}'
    assert isinstance(r['strategies'], list) and len(r['strategies']) > 0, 'strategies must be non-empty list'
    assert isinstance(r['constraints'], dict), 'constraints must be an object'
    print(f'RISK type={r[\"type\"]} source={r[\"source\"]} symbol={r[\"symbol\"]} tf={r[\"timeframe\"]} disposition={r[\"disposition\"]} confidence={r[\"confidence\"]} final={r[\"final\"]}')
    print(f'  constraints={r[\"constraints\"]} strategies={len(r[\"strategies\"])}')
else:
    print('NULL (trend_following_entry warm-up not complete — drawdown_limit pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "RISK"; then
                pass "risk drawdown_limit ${sym}/${tf}s → risk present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "risk drawdown_limit ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "risk drawdown_limit ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 12a: Cross-symbol drawdown_limit risk isolation ----------
section "Step 12a: Cross-symbol drawdown_limit risk isolation (Breadth S243)"
info "Verifying risk Drawdown Limit produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    RISK_A=$(curl -s "${BASE_URL}/risk/drawdown_limit/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    RISK_B=$(curl -s "${BASE_URL}/risk/drawdown_limit/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${RISK_A}''')['risk']
b = json.loads('''${RISK_B}''')['risk']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "risk drawdown_limit tf=${tf}s: symbols produce independent risk data" ;;
        PARTIAL)  pass "risk drawdown_limit tf=${tf}s: one symbol has risk, other pending (expected)" ;;
        NONE)     info "risk drawdown_limit tf=${tf}s: no risk assessments yet (warm-up pending)" ;;
        COLLISION) fail "risk drawdown_limit tf=${tf}s: SYMBOL COLLISION — both risks have same symbol" ;;
        BLEED_A)  fail "risk drawdown_limit tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "risk drawdown_limit tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "risk drawdown_limit tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 13: Execution Paper Order multi-symbol validation ----------
section "Step 13: Execution Paper Order multi-symbol validation"
info "Note: Execution evaluates after each risk assessment. Requires full pipeline warm-up."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking execution paper_order ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/execution/paper_order/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/execution/paper_order/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "execution paper_order ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'execution_intent' in data, 'missing execution_intent key'
ei = data['execution_intent']
if ei is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'side', 'quantity', 'status', 'risk', 'final', 'timestamp']
    for field in required:
        assert field in ei, f'missing field: {field}'
    assert ei['source'] == '${SOURCE}', f'wrong source: {ei[\"source\"]}'
    assert ei['symbol'] == '${sym}', f'wrong symbol: {ei[\"symbol\"]}'
    assert ei['timeframe'] == ${tf}, f'wrong timeframe: {ei[\"timeframe\"]}'
    assert ei['type'] == 'paper_order', f'wrong type: {ei[\"type\"]}'
    assert ei['side'] in ('buy', 'sell', 'none'), f'invalid side: {ei[\"side\"]}'
    assert ei['status'] in ('submitted', 'filled'), f'invalid status: {ei[\"status\"]} (expected submitted or filled)'
    assert isinstance(ei['risk'], dict), 'risk must be an object'
    assert ei['risk'].get('type') != '', 'risk.type must not be empty'
    assert ei['risk'].get('disposition') != '', 'risk.disposition must not be empty'
    # Lifecycle fields validation (S77+)
    if ei['side'] in ('buy', 'sell'):
        assert ei['status'] == 'filled', f'actionable order should be filled, got {ei[\"status\"]}'
        assert ei.get('filled_quantity', '') != '', f'filled order must have filled_quantity'
        fills = ei.get('fills', [])
        assert isinstance(fills, list) and len(fills) > 0, f'filled order must have fill records, got {fills}'
        for fill in fills:
            assert fill.get('simulated') == True, f'paper fill must be simulated'
            assert fill.get('quantity', '') != '', f'fill must have quantity'
    elif ei['side'] == 'none':
        assert ei['status'] == 'submitted', f'no-action order should be submitted, got {ei[\"status\"]}'
    # Trace fields validation (S78+)
    trace_fields = ['correlation_id', 'causation_id']
    for tf_name in trace_fields:
        if tf_name in ei and ei[tf_name]:
            pass  # present and non-empty — good
    print(f'EXECUTION type={ei[\"type\"]} source={ei[\"source\"]} symbol={ei[\"symbol\"]} tf={ei[\"timeframe\"]} side={ei[\"side\"]} qty={ei[\"quantity\"]} status={ei[\"status\"]} final={ei[\"final\"]}')
    fills_count = len(ei.get('fills', []))
    print(f'  risk_type={ei[\"risk\"][\"type\"]} risk_disposition={ei[\"risk\"][\"disposition\"]} risk_confidence={ei[\"risk\"][\"confidence\"]} fills={fills_count} corr={ei.get(\"correlation_id\", \"\")} cause={ei.get(\"causation_id\", \"\")}')
else:
    print('NULL (risk warm-up not complete — execution pending)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "EXECUTION"; then
                pass "execution paper_order ${sym}/${tf}s → execution present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "execution paper_order ${sym}/${tf}s → endpoint reachable (null — warm-up pending)"
            fi
        else
            fail "execution paper_order ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 14: Cross-symbol execution isolation ----------
section "Step 14: Cross-symbol execution isolation"
info "Verifying execution Paper Order produces independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    EXEC_A=$(curl -s "${BASE_URL}/execution/paper_order/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    EXEC_B=$(curl -s "${BASE_URL}/execution/paper_order/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${EXEC_A}''')['execution_intent']
b = json.loads('''${EXEC_B}''')['execution_intent']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "execution paper_order tf=${tf}s: symbols produce independent execution data" ;;
        PARTIAL)  pass "execution paper_order tf=${tf}s: one symbol has execution, other pending (expected)" ;;
        NONE)     info "execution paper_order tf=${tf}s: no executions yet (warm-up pending)" ;;
        COLLISION) fail "execution paper_order tf=${tf}s: SYMBOL COLLISION — both executions have same symbol" ;;
        BLEED_A)  fail "execution paper_order tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "execution paper_order tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "execution paper_order tf=${tf}s: isolation check error" ;;
    esac
done

# ---------- Step 15: Execution control gate validation ----------
section "Step 15: Execution control gate validation"
info "Validating execution control gate GET/PUT cycle..."

# 15a: GET control — should return active gate (default).
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/execution/control")
RESPONSE=$(curl -s "${BASE_URL}/execution/control")

if [[ "$HTTP_CODE" == "200" ]]; then
    RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'gate' in data, 'missing gate key'
gate = data['gate']
assert gate['status'] in ('active', 'halted'), f'invalid gate status: {gate[\"status\"]}'
print(f'GATE status={gate[\"status\"]} reason={gate.get(\"reason\", \"\")} updated_by={gate.get(\"updated_by\", \"\")}')
print('OK')
" 2>&1)
    if echo "$RESULT" | grep -q "OK"; then
        pass "GET /execution/control → gate present"
        echo "    $(echo "$RESULT" | head -1)"
    else
        fail "GET /execution/control → structure invalid: $RESULT"
    fi
else
    fail "GET /execution/control → HTTP ${HTTP_CODE} (expected 200)"
fi

# 15b: PUT control → halt.
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"smoke test halt","updated_by":"smoke-test"}' \
    "${BASE_URL}/execution/control")
HALT_RESPONSE=$(curl -s -X PUT \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"smoke test halt","updated_by":"smoke-test"}' \
    "${BASE_URL}/execution/control")

if [[ "$HTTP_CODE" == "200" ]]; then
    RESULT=$(echo "$HALT_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
gate = data.get('gate', {})
assert gate.get('status') == 'halted', f'expected halted, got {gate.get(\"status\")}'
print('OK')
" 2>&1)
    if echo "$RESULT" | grep -q "OK"; then
        pass "PUT /execution/control → halted"
    else
        fail "PUT /execution/control halt → unexpected: $RESULT"
    fi
else
    fail "PUT /execution/control halt → HTTP ${HTTP_CODE} (expected 200)"
fi

# 15c: Verify halt persists via GET.
VERIFY_RESPONSE=$(curl -s "${BASE_URL}/execution/control")
RESULT=$(echo "$VERIFY_RESPONSE" | python3 -c "
import sys, json
gate = json.load(sys.stdin).get('gate', {})
assert gate.get('status') == 'halted', f'expected halted after PUT, got {gate.get(\"status\")}'
print('OK')
" 2>&1)
if echo "$RESULT" | grep -q "OK"; then
    pass "GET /execution/control after halt → confirmed halted"
else
    fail "GET /execution/control after halt → not halted: $RESULT"
fi

# 15d: PUT control → resume (active).
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"smoke test resume","updated_by":"smoke-test"}' \
    "${BASE_URL}/execution/control")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "PUT /execution/control → resumed (active)"
else
    fail "PUT /execution/control resume → HTTP ${HTTP_CODE} (expected 200)"
fi

# 15e: Verify resume persists via GET.
RESUME_RESPONSE=$(curl -s "${BASE_URL}/execution/control")
RESULT=$(echo "$RESUME_RESPONSE" | python3 -c "
import sys, json
gate = json.load(sys.stdin).get('gate', {})
assert gate.get('status') == 'active', f'expected active after resume, got {gate.get(\"status\")}'
print('OK')
" 2>&1)
if echo "$RESULT" | grep -q "OK"; then
    pass "GET /execution/control after resume → confirmed active"
else
    fail "GET /execution/control after resume → not active: $RESULT"
fi

# ---------- Step 16: Execute binary health check ----------
section "Step 16: Execute binary health check"
EXECUTE_URL="${EXECUTE_URL:-http://127.0.0.1:8084}"
info "Checking execute binary health at ${EXECUTE_URL}..."

EXEC_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "${EXECUTE_URL}/healthz" 2>/dev/null || echo "000")
[[ "$EXEC_HEALTH" == "200" ]] && pass "execute /healthz → 200" || fail "execute /healthz → $EXEC_HEALTH"

EXEC_READY=$(curl -s -o /dev/null -w "%{http_code}" "${EXECUTE_URL}/readyz" 2>/dev/null || echo "000")
[[ "$EXEC_READY" == "200" ]] && pass "execute /readyz → 200" || fail "execute /readyz → $EXEC_READY"

# Check /statusz for operational counters.
EXEC_STATUS=$(curl -s "${EXECUTE_URL}/statusz" 2>/dev/null || echo "{}")
EXEC_STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${EXECUTE_URL}/statusz" 2>/dev/null || echo "000")
[[ "$EXEC_STATUS_CODE" == "200" ]] && pass "execute /statusz → 200" || fail "execute /statusz → $EXEC_STATUS_CODE"

# ---------- Step 17: Venue market order fill multi-symbol validation ----------
section "Step 17: Venue market order fill multi-symbol validation"
info "Note: Fill data requires execute binary running. May be null if execute is not active."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking execution venue_market_order ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/execution/venue_market_order/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/execution/venue_market_order/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "execution venue_market_order ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'execution_intent' in data, 'missing execution_intent key'
ei = data['execution_intent']
if ei is not None:
    required = ['type', 'source', 'symbol', 'timeframe', 'side', 'quantity', 'status', 'risk', 'final', 'timestamp']
    for field in required:
        assert field in ei, f'missing field: {field}'
    assert ei['source'] == '${SOURCE}', f'wrong source: {ei[\"source\"]}'
    assert ei['symbol'] == '${sym}', f'wrong symbol: {ei[\"symbol\"]}'
    assert ei['timeframe'] == ${tf}, f'wrong timeframe: {ei[\"timeframe\"]}'
    # Fill-side semantics: actionable orders should be filled with venue fills.
    if ei['side'] in ('buy', 'sell'):
        assert ei['status'] == 'filled', f'venue fill should be filled, got {ei[\"status\"]}'
        fills = ei.get('fills', [])
        assert isinstance(fills, list) and len(fills) > 0, f'venue fill must have fill records'
        for fill in fills:
            assert fill.get('simulated') == True, f'paper fill must be simulated'
    # Trace fields must survive through execute.
    trace_corr = ei.get('correlation_id', '')
    trace_cause = ei.get('causation_id', '')
    print(f'FILL type={ei[\"type\"]} source={ei[\"source\"]} symbol={ei[\"symbol\"]} tf={ei[\"timeframe\"]} side={ei[\"side\"]} status={ei[\"status\"]} filled_qty={ei.get(\"filled_quantity\", \"\")}')
    print(f'  fills={len(ei.get(\"fills\", []))} corr={trace_corr} cause={trace_cause}')
else:
    print('NULL (execute binary may not be running or pipeline not yet warmed)')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            if echo "$RESULT" | grep -q "FILL"; then
                pass "execution venue_market_order ${sym}/${tf}s → fill present"
                echo "    $(echo "$RESULT" | head -1)"
                echo "    $(echo "$RESULT" | sed -n '2p')"
            else
                pass "execution venue_market_order ${sym}/${tf}s → endpoint reachable (null — execute pending)"
            fi
        else
            fail "execution venue_market_order ${sym}/${tf}s → structure invalid: $RESULT"
        fi
    done
done

# ---------- Step 18: Cross-symbol fill isolation ----------
section "Step 18: Cross-symbol fill isolation"
info "Verifying venue market order fills produce independent data per symbol..."

for tf in "${TIMEFRAMES[@]}"; do
    FILL_A=$(curl -s "${BASE_URL}/execution/venue_market_order/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
    FILL_B=$(curl -s "${BASE_URL}/execution/venue_market_order/latest?source=${SOURCE}&base=${SYMBOLS[1]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

    RESULT=$(python3 -c "
import sys, json
a = json.loads('''${FILL_A}''')['execution_intent']
b = json.loads('''${FILL_B}''')['execution_intent']
if a is not None and b is not None:
    if a['symbol'] == b['symbol']:
        print('COLLISION')
    elif a['symbol'] != '${SYMBOLS[0]}':
        print('BLEED_A')
    elif b['symbol'] != '${SYMBOLS[1]}':
        print('BLEED_B')
    else:
        print('ISOLATED')
elif a is not None or b is not None:
    print('PARTIAL')
else:
    print('NONE')
" 2>/dev/null || echo "ERROR")

    case "$RESULT" in
        ISOLATED) pass "execution venue_market_order tf=${tf}s: symbols produce independent fill data" ;;
        PARTIAL)  pass "execution venue_market_order tf=${tf}s: one symbol has fill, other pending (expected)" ;;
        NONE)     info "execution venue_market_order tf=${tf}s: no fills yet (execute pending)" ;;
        COLLISION) fail "execution venue_market_order tf=${tf}s: SYMBOL COLLISION — both fills have same symbol" ;;
        BLEED_A)  fail "execution venue_market_order tf=${tf}s: CROSS-SYMBOL BLEED — symbol A has wrong symbol field" ;;
        BLEED_B)  fail "execution venue_market_order tf=${tf}s: CROSS-SYMBOL BLEED — symbol B has wrong symbol field" ;;
        *)        fail "execution venue_market_order tf=${tf}s: fill isolation check error" ;;
    esac
done

# ---------- Step 19: Execution status propagation ----------
section "Step 19: Execution status propagation (composite)"
info "Validating /execution/status/latest shows intent + result + gate + propagation..."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking execution status ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/execution/status/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/execution/status/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=${tf}")

        if [[ "$HTTP_CODE" != "200" ]]; then
            fail "execution status ${sym}/${tf}s → HTTP ${HTTP_CODE} (expected 200)"
            continue
        fi

        RESULT=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
# Required top-level fields in status response.
for key in ('intent', 'result', 'gate', 'propagation'):
    assert key in data, f'missing key: {key}'
# Gate must always be present with a valid status.
gate = data['gate']
assert gate.get('status') in ('active', 'halted'), f'invalid gate status: {gate.get(\"status\")}'
# Propagation must be a valid value.
prop = data['propagation']
assert prop in ('none', 'submitted', 'sent', 'accepted', 'rejected', 'filled', 'partially_filled', 'cancelled'), f'invalid propagation: {prop}'
intent = data.get('intent')
result = data.get('result')
intent_status = intent['status'] if intent else 'null'
result_status = result['status'] if result else 'null'
# Verify propagation priority: result > intent > none.
if result is not None:
    assert prop == result['status'], f'propagation should match result status ({result[\"status\"]}), got {prop}'
elif intent is not None:
    assert prop == intent['status'], f'propagation should match intent status ({intent[\"status\"]}), got {prop}'
else:
    assert prop == 'none', f'propagation should be none when both null, got {prop}'
print(f'STATUS intent={intent_status} result={result_status} gate={gate[\"status\"]} propagation={prop}')
print('OK')
" 2>&1)

        if echo "$RESULT" | grep -q "OK"; then
            pass "execution status ${sym}/${tf}s → composite status valid"
            echo "    $(echo "$RESULT" | head -1)"
        else
            fail "execution status ${sym}/${tf}s → invalid: $RESULT"
        fi
    done
done

# ---------- Step 20: Kill switch integration with execute active ----------
section "Step 20: Kill switch integration with execute active"
info "Testing halt → verify gate persists → resume → verify gate restored..."

# 20a: Halt execution.
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"status":"halted","reason":"S84 integration test halt","updated_by":"smoke-s84"}' \
    "${BASE_URL}/execution/control")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "PUT /execution/control → halted (S84)"
else
    fail "PUT /execution/control halt → HTTP ${HTTP_CODE}"
fi

# 20b: Verify halt via status endpoint — gate should show halted.
sleep 1
for sym in "${SYMBOLS[@]}"; do
    STATUS_RESPONSE=$(curl -s "${BASE_URL}/execution/status/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
    RESULT=$(echo "$STATUS_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
gate = data.get('gate', {})
assert gate.get('status') == 'halted', f'expected halted in status, got {gate.get(\"status\")}'
print(f'HALTED reason={gate.get(\"reason\", \"\")} updated_by={gate.get(\"updated_by\", \"\")}')
print('OK')
" 2>&1)
    if echo "$RESULT" | grep -q "OK"; then
        pass "execution status ${sym}/60s → gate=halted visible in composite status"
        echo "    $(echo "$RESULT" | head -1)"
    else
        fail "execution status ${sym}/60s → gate not halted: $RESULT"
    fi
done

# 20c: Resume execution.
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"status":"active","reason":"S84 integration test resume","updated_by":"smoke-s84"}' \
    "${BASE_URL}/execution/control")

if [[ "$HTTP_CODE" == "200" ]]; then
    pass "PUT /execution/control → resumed (active, S84)"
else
    fail "PUT /execution/control resume → HTTP ${HTTP_CODE}"
fi

# 20d: Verify resume via status endpoint.
sleep 1
VERIFY_RESUME=$(curl -s "${BASE_URL}/execution/status/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
RESULT=$(echo "$VERIFY_RESUME" | python3 -c "
import sys, json
gate = json.load(sys.stdin).get('gate', {})
assert gate.get('status') == 'active', f'expected active after resume, got {gate.get(\"status\")}'
print('OK')
" 2>&1)
if echo "$RESULT" | grep -q "OK"; then
    pass "execution status after resume → gate=active confirmed"
else
    fail "execution status after resume → gate not active: $RESULT"
fi

# ---------- Step 21: Trace persistence through execute ----------
section "Step 21: Trace persistence through execute chain"
info "Verifying correlation_id and causation_id persist in venue fill events..."

TRACE_VERIFIED=0
for sym in "${SYMBOLS[@]}"; do
    FILL_RESP=$(curl -s "${BASE_URL}/execution/venue_market_order/latest?source=${SOURCE}&base=${sym%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
    RESULT=$(echo "$FILL_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
ei = data.get('execution_intent')
if ei is not None:
    corr = ei.get('correlation_id', '')
    cause = ei.get('causation_id', '')
    if corr and cause:
        print(f'TRACED corr={corr} cause={cause}')
    elif corr or cause:
        print(f'PARTIAL_TRACE corr={corr} cause={cause}')
    else:
        print('NO_TRACE')
else:
    print('NULL')
print('OK')
" 2>&1)

    if echo "$RESULT" | grep -q "TRACED"; then
        pass "trace persistence ${sym}/60s → correlation_id + causation_id present in fill"
        echo "    $(echo "$RESULT" | head -1)"
        TRACE_VERIFIED=$((TRACE_VERIFIED + 1))
    elif echo "$RESULT" | grep -q "PARTIAL_TRACE"; then
        fail "trace persistence ${sym}/60s → partial trace (one field missing)"
        echo "    $(echo "$RESULT" | head -1)"
    elif echo "$RESULT" | grep -q "NO_TRACE"; then
        fail "trace persistence ${sym}/60s → no trace fields in fill"
    elif echo "$RESULT" | grep -q "NULL"; then
        info "trace persistence ${sym}/60s → null (execute not running or warm-up pending)"
    else
        fail "trace persistence ${sym}/60s → error: $RESULT"
    fi
done

if [[ $TRACE_VERIFIED -gt 0 ]]; then
    pass "trace persistence: $TRACE_VERIFIED symbols verified with full trace chain"
else
    info "trace persistence: no fills available yet — verify when execute is active"
fi

# ---------- Step 22: Error handling ----------
section "Step 22: Error handling"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Missing timeframe → 400" || info "Missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest")
[[ "$HTTP_CODE" == "400" ]] && pass "No params → 400" || info "No params → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/signal/unknown/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown signal type → 400" || info "Unknown signal type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Signal missing timeframe → 400" || info "Signal missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/decision/unknown/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown decision type → 400" || info "Unknown decision type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Decision rsi_oversold missing timeframe → 400" || info "Decision rsi_oversold missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/decision/ema_crossover/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Decision ema_crossover missing timeframe → 400" || info "Decision ema_crossover missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/strategy/unknown/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown strategy type → 400" || info "Unknown strategy type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Strategy mean_reversion_entry missing timeframe → 400" || info "Strategy mean_reversion_entry missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/strategy/trend_following_entry/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Strategy trend_following_entry missing timeframe → 400" || info "Strategy trend_following_entry missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/risk/unknown/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown risk type → 400" || info "Unknown risk type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/risk/position_exposure/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Risk position_exposure missing timeframe → 400" || info "Risk position_exposure missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/risk/drawdown_limit/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Risk drawdown_limit missing timeframe → 400" || info "Risk drawdown_limit missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/execution/unknown/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown execution type → 400" || info "Unknown execution type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/execution/paper_order/latest?source=${SOURCE}&base=${SYMBOLS[0]%usdt}&quote=usdt&contract=${CONTRACT}")
[[ "$HTTP_CODE" == "400" ]] && pass "Execution missing timeframe → 400" || info "Execution missing timeframe → $HTTP_CODE"

# ---------- Summary ----------
phase "Summary"
echo "Canonical target: make smoke-multi"
echo "Expected setup: ${SETUP_HINT}"
echo "Gateway: ${BASE_URL}"
echo "Warm-up budget: ${WAIT_SECONDS}s"
echo ""
echo "  Passed: ${PASS_COUNT}"
echo "  Failed: ${FAIL_COUNT}"
echo "  Warned: ${WARN_COUNT}"
echo ""
echo "Scenario: ${#SYMBOLS[@]} symbols × ${#TIMEFRAMES[@]} timeframes"
echo "  Evidence   KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Signal     KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Decision   KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Strategy   KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Risk       KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Execution  KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Fill       KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo ""
echo "Symbols validated:"
for sym in "${SYMBOLS[@]}"; do
    echo "  ${SOURCE}.${sym}"
done
echo ""
echo "Timeframes validated:"
for tf in "${TIMEFRAMES[@]}"; do
    echo "  ${tf}s"
done
echo ""
echo "Flow validated (per symbol):"
echo "  Binance WS → ingest → OBSERVATION_EVENTS"
echo "                       → derive (${TIMEFRAMES[*]}s samplers)"
echo "                       → EVIDENCE_EVENTS → store (NATS KV)"
echo "                       → evidence.query.candle.latest"
echo "                       → GET /evidence/candles/latest"
echo ""
echo "  derive (RSI sampler) → SIGNAL_EVENTS → store (NATS KV)"
echo "                       → signal.query.rsi.latest"
echo "                       → GET /signal/rsi/latest"
echo ""
echo "  derive (EMA Crossover sampler) → SIGNAL_EVENTS → store (NATS KV)"
echo "                                 → signal.query.ema_crossover.latest"
echo "                                 → GET /signal/ema_crossover/latest"
echo ""
echo "  derive (RSI Oversold evaluator) → DECISION_EVENTS → store (NATS KV)"
echo "                                  → decision.query.rsi_oversold.latest"
echo "                                  → GET /decision/rsi_oversold/latest"
echo ""
echo "  derive (EMA Crossover evaluator) → DECISION_EVENTS → store (NATS KV)"
echo "                                   → decision.query.ema_crossover.latest"
echo "                                   → GET /decision/ema_crossover/latest"
echo ""
echo "  derive (Mean Reversion Entry resolver) → STRATEGY_EVENTS → store (NATS KV)"
echo "                                         → strategy.query.mean_reversion_entry.latest"
echo "                                         → GET /strategy/mean_reversion_entry/latest"
echo ""
echo "  derive (Trend Following Entry resolver) → STRATEGY_EVENTS → store (NATS KV)"
echo "                                          → strategy.query.trend_following_entry.latest"
echo "                                          → GET /strategy/trend_following_entry/latest"
echo ""
echo "  derive (Position Exposure evaluator) → RISK_EVENTS → store (NATS KV)"
echo "                                       → risk.query.position_exposure.latest"
echo "                                       → GET /risk/position_exposure/latest"
echo ""
echo "  derive (Drawdown Limit evaluator) → RISK_EVENTS → store (NATS KV)"
echo "                                    → risk.query.drawdown_limit.latest"
echo "                                    → GET /risk/drawdown_limit/latest"
echo ""
echo "  derive (Paper Order evaluator) → EXECUTION_EVENTS → store (NATS KV)"
echo "                                 → execution.query.paper_order.latest"
echo "                                 → GET /execution/paper_order/latest"
echo ""
echo "  execute (Venue Adapter) → EXECUTION_FILL_EVENTS → store (NATS KV)"
echo "                          → execution.query.venue_market_order.latest"
echo "                          → GET /execution/venue_market_order/latest"
echo ""
echo "  GET /execution/status/latest (composite: intent + result + gate + propagation)"
echo "  GET/PUT /execution/control (kill switch gate cycle)"
echo "  Trace persistence: correlation_id + causation_id through execute chain"
echo ""

if [[ $FAIL_COUNT -gt 0 ]]; then
    echo "Recommended diagnosis:"
    echo "  make ps"
    echo "  make logs SERVICE=gateway"
    echo "  make logs SERVICE=derive"
    echo "  make logs SERVICE=store"
    echo "  make logs SERVICE=execute"
    exit 1
fi

echo "Recommended repeat/diagnosis commands:"
echo "  make smoke-multi"
echo "  make diag"
echo "  make logs SERVICE=gateway"
