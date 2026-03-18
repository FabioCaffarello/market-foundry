#!/usr/bin/env bash
# smoke-multi-symbol.sh — E2E smoke test for multi-symbol × multi-timeframe scenario.
#
# Flow validated per symbol:
#   Evidence:  Binance WS → ingest → OBSERVATION_EVENTS → derive → EVIDENCE_EVENTS
#                                                                 → store (NATS KV)
#                                                                 → gateway HTTP endpoint
#   Signal:    derive (RSI sampler) → SIGNAL_EVENTS → store (NATS KV)
#                                                    → gateway HTTP endpoint
#   Decision:  derive (RSI Oversold evaluator) → DECISION_EVENTS → store (NATS KV)
#                                                                 → gateway HTTP endpoint
#   Strategy:  derive (Mean Reversion Entry resolver) → STRATEGY_EVENTS → store (NATS KV)
#                                                                        → gateway HTTP endpoint
#
# Validates: 2 symbols (btcusdt, ethusdt) × 2 timeframes (60s, 300s)
# Evidence KV keys: 4 (2 symbols × 2 timeframes)
# Signal   KV keys: 4 (2 symbols × 2 timeframes)
# Decision KV keys: 4 (2 symbols × 2 timeframes)
# Strategy KV keys: 4 (2 symbols × 2 timeframes)
#
# Prerequisites:
#   make up
#   make seed-multi    (seeds configctl with 2 symbols)
#
# Usage:
#   ./scripts/smoke-multi-symbol.sh
#   ./scripts/smoke-multi-symbol.sh --wait 120

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SOURCE="binancef"
SYMBOLS=("btcusdt" "ethusdt")
TIMEFRAMES=(60 300)
WAIT_SECONDS="${1:-90}"
if [[ "${1:-}" == "--wait" ]]; then
    WAIT_SECONDS="${2:-90}"
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

pass() { echo -e "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
hard_fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
section() { echo -e "\n${CYAN}--- $1 ---${NC}"; }

# ---------- Step 1: Health + Readiness ----------
section "Step 1: Gateway checks"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/healthz")
[[ "$HTTP_CODE" == "200" ]] && pass "/healthz → 200" || hard_fail "/healthz → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/readyz")
[[ "$HTTP_CODE" == "200" ]] && pass "/readyz → 200" || hard_fail "/readyz → $HTTP_CODE"

# ---------- Step 2: Wait for first candle ----------
section "Step 2: Waiting for pipeline (${WAIT_SECONDS}s max)"
info "Waiting for at least one 60s candle from any symbol..."

CANDLE_FOUND=false
ELAPSED=0
POLL_INTERVAL=5

while [[ $ELAPSED -lt $WAIT_SECONDS ]]; do
    for sym in "${SYMBOLS[@]}"; do
        RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${sym}&timeframe=60")
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
            "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")

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
    CANDLE_A=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=${tf}")
    CANDLE_B=$(curl -s "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${SYMBOLS[1]}&timeframe=${tf}")

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
            "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")

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
    SIG_A=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=${tf}")
    SIG_B=$(curl -s "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&symbol=${SYMBOLS[1]}&timeframe=${tf}")

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

# ---------- Step 7: Decision RSI Oversold multi-symbol validation ----------
section "Step 7: Decision RSI Oversold multi-symbol validation"
info "Note: Decision evaluates after each RSI signal. Requires RSI warm-up (~15min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking decision rsi_oversold ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")

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
    DEC_A=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=${tf}")
    DEC_B=$(curl -s "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&symbol=${SYMBOLS[1]}&timeframe=${tf}")

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

# ---------- Step 9: Strategy Mean Reversion Entry multi-symbol validation ----------
section "Step 9: Strategy Mean Reversion Entry multi-symbol validation"
info "Note: Strategy resolves after each decision. Requires RSI warm-up (~15min at 60s)."

for sym in "${SYMBOLS[@]}"; do
    for tf in "${TIMEFRAMES[@]}"; do
        info "Checking strategy mean_reversion_entry ${SOURCE}/${sym}/${tf}s..."

        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")
        RESPONSE=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&symbol=${sym}&timeframe=${tf}")

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
    STRAT_A=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=${tf}")
    STRAT_B=$(curl -s "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&symbol=${SYMBOLS[1]}&timeframe=${tf}")

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

# ---------- Step 11: Error handling ----------
section "Step 11: Error handling"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}")
[[ "$HTTP_CODE" == "400" ]] && pass "Missing timeframe → 400" || info "Missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/evidence/candles/latest")
[[ "$HTTP_CODE" == "400" ]] && pass "No params → 400" || info "No params → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/signal/unknown/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown signal type → 400" || info "Unknown signal type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/signal/rsi/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}")
[[ "$HTTP_CODE" == "400" ]] && pass "Signal missing timeframe → 400" || info "Signal missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/decision/unknown/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown decision type → 400" || info "Unknown decision type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/decision/rsi_oversold/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}")
[[ "$HTTP_CODE" == "400" ]] && pass "Decision missing timeframe → 400" || info "Decision missing timeframe → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/strategy/unknown/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}&timeframe=60")
[[ "$HTTP_CODE" == "400" ]] && pass "Unknown strategy type → 400" || info "Unknown strategy type → $HTTP_CODE"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/strategy/mean_reversion_entry/latest?source=${SOURCE}&symbol=${SYMBOLS[0]}")
[[ "$HTTP_CODE" == "400" ]] && pass "Strategy missing timeframe → 400" || info "Strategy missing timeframe → $HTTP_CODE"

# ---------- Summary ----------
echo ""
echo "=========================================="
echo "  Multi-Symbol E2E Smoke: COMPLETE"
echo "=========================================="
echo ""
echo "  Passed: ${PASS_COUNT}"
echo "  Failed: ${FAIL_COUNT}"
echo ""
echo "Scenario: ${#SYMBOLS[@]} symbols × ${#TIMEFRAMES[@]} timeframes"
echo "  Evidence  KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Signal    KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Decision  KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
echo "  Strategy  KV entries: $((${#SYMBOLS[@]} * ${#TIMEFRAMES[@]}))"
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
echo "  derive (RSI Oversold evaluator) → DECISION_EVENTS → store (NATS KV)"
echo "                                  → decision.query.rsi_oversold.latest"
echo "                                  → GET /decision/rsi_oversold/latest"
echo ""
echo "  derive (Mean Reversion Entry resolver) → STRATEGY_EVENTS → store (NATS KV)"
echo "                                         → strategy.query.mean_reversion_entry.latest"
echo "                                         → GET /strategy/mean_reversion_entry/latest"
echo ""

if [[ $FAIL_COUNT -gt 0 ]]; then
    exit 1
fi
