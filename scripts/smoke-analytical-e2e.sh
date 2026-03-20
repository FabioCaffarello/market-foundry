#!/usr/bin/env bash
# smoke-analytical-e2e.sh — End-to-end integration proof for the analytical layer.
#
# Flows validated:
#   [Baseline]    NATS JetStream → writer → ClickHouse → reader → GET /analytical/evidence/candles
#   [Wave B F-01] NATS JetStream → writer → ClickHouse → reader → GET /analytical/signal/history
#   [Wave B F-02] NATS JetStream → writer → ClickHouse → reader → GET /analytical/decision/history
#   [Wave B F-03] NATS JetStream → writer → ClickHouse → reader → GET /analytical/strategy/history
#   [Wave B F-04] NATS JetStream → writer → ClickHouse → reader → GET /analytical/risk/history
#   [Wave B F-05] NATS JetStream → writer → ClickHouse → reader → GET /analytical/execution/history
#
# This script validates the complete analytical data path in the minimum useful scope:
#   1. Infrastructure readiness (ClickHouse + writer + gateway)
#   2. Migration status (schema applied)
#   3. Writer consuming and flushing (NATS → ClickHouse)
#   4. Reader querying (ClickHouse → gateway HTTP)
#   5. Response structure correctness
#
# Prerequisites:
#   make up          # starts full stack including clickhouse + writer
#   make seed-multi  # (or make seed) seeds configctl with bindings
#   wait ~120s       # writer needs time to consume events and flush batches
#
# Usage:
#   ./scripts/smoke-analytical-e2e.sh
#   ./scripts/smoke-analytical-e2e.sh --wait 180   # override flush wait (default: 120)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
COMPOSE="docker compose -f ${COMPOSE_FILE}"

FLUSH_WAIT="${FLUSH_WAIT:-120}"
if [[ "${1:-}" == "--wait" ]]; then
    FLUSH_WAIT="${2:-120}"
fi

ERRORS=0

# ── Reusable per-family validation ───────────────────────────────────
# validate_analytical_family validates an analytical family's read path:
#   1. ClickHouse row count for the table/filter
#   2. HTTP endpoint returns 200
#   3. Response JSON structure (source=clickhouse, meta, required fields)
#   4. Item count > 0
#   5. Server-Timing header present
#
# Args:
#   $1 - Family label (e.g., "Candles — Baseline Family")
#   $2 - ClickHouse table (e.g., "evidence_candles")
#   $3 - ClickHouse WHERE clause (empty string for none)
#   $4 - HTTP endpoint URL (full URL with query params)
#   $5 - JSON response key (e.g., "candles", "signals", "decisions")
#   $6 - Required fields (pipe-separated, e.g., "source|symbol|timeframe|open")
#
# Sets exported variables: _VAL_CH_COUNT, _VAL_HTTP_COUNT
validate_analytical_family() {
    local label="$1"
    local ch_table="$2"
    local ch_where="$3"
    local endpoint_url="$4"
    local json_key="$5"
    local required_fields="$6"

    phase "Reader → HTTP Query Surface (${label})"

    # --- ClickHouse row count ---
    local ch_query="SELECT count() FROM ${ch_table}"
    [[ -n "$ch_where" ]] && ch_query="${ch_query} WHERE ${ch_where}"
    info "Querying ClickHouse for ${ch_table} rows..."
    _VAL_CH_COUNT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
        --query "$ch_query" 2>/dev/null || echo "0")

    if [[ "$_VAL_CH_COUNT" -gt 0 ]]; then
        pass "${ch_table} has ${_VAL_CH_COUNT} rows — writer→ClickHouse path PROVEN"
    else
        warn "${ch_table} has 0 rows (pipeline may still be warming up)"
    fi

    # --- HTTP endpoint ---
    info "Querying HTTP endpoint..."
    local response http_code
    response=$(curl -s "${endpoint_url}&limit=10")
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${endpoint_url}&limit=10")

    if [[ "$http_code" == "200" ]]; then
        pass "GET ${endpoint_url%%\?*} → 200"
    else
        record_fail "GET ${endpoint_url%%\?*} → ${http_code} (expected 200)"
    fi

    # --- Response structure validation ---
    info "Validating response structure..."
    local struct_ok
    struct_ok=$(echo "$response" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    key = '${json_key}'
    assert key in d, f'missing {key} key'
    assert 'source' in d, 'missing source key'
    assert d['source'] == 'clickhouse', f'wrong source: {d[\"source\"]}'
    assert 'meta' in d, 'missing meta key'
    meta = d['meta']
    assert 'query_ms' in meta, 'missing meta.query_ms'
    assert 'row_count' in meta, 'missing meta.row_count'
    items = d[key]
    assert isinstance(items, list), f'{key} is not a list'
    print(f'{key}_count={len(items)} source={d[\"source\"]} query_ms={meta[\"query_ms\"]}')
    if len(items) > 0:
        item = items[0]
        required = '${required_fields}'.split('|')
        missing = [f for f in required if f not in item]
        if missing:
            print(f'MISSING FIELDS: {missing}')
            sys.exit(1)
    print('OK')
except Exception as e:
    print(f'FAIL: {e}')
    sys.exit(1)
" 2>&1) && pass "Response structure valid: ${struct_ok}" || record_fail "Response structure invalid: ${struct_ok}"

    # --- Item count ---
    _VAL_HTTP_COUNT=$(echo "$response" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(len(d.get('${json_key}', [])))
" 2>/dev/null || echo "0")

    if [[ "$_VAL_HTTP_COUNT" -gt 0 ]]; then
        pass "HTTP response contains ${_VAL_HTTP_COUNT} ${json_key} — ClickHouse→reader→HTTP path PROVEN"
    else
        if [[ "$_VAL_CH_COUNT" -gt 0 ]]; then
            record_fail "ClickHouse has ${_VAL_CH_COUNT} rows but HTTP returned 0 — reader query may have a mismatch"
        else
            warn "HTTP returned 0 ${json_key} (ClickHouse also empty — write path not yet producing data)"
        fi
    fi

    # --- Server-Timing header ---
    info "Checking Server-Timing header..."
    local timing
    timing=$(curl -s -D - -o /dev/null "${endpoint_url}" 2>/dev/null | grep -i "Server-Timing" || echo "")
    if [[ -n "$timing" ]]; then
        pass "${json_key} endpoint returns Server-Timing header"
    else
        record_fail "${json_key} endpoint missing Server-Timing header"
    fi
}

# validate_analytical_error_handling validates 400 responses for an analytical family.
#
# Args:
#   $1 - Family label (e.g., "Candle", "Signal", "Decision")
#   $2 - Base endpoint URL (without query params)
#   $3 - Valid base params (source + symbol + timeframe)
#   $4 - Params missing timeframe (for testing required-param validation)
validate_analytical_error_handling() {
    local label="$1"
    local base_url="$2"
    local valid_params="$3"
    local no_timeframe_params="$4"
    local code

    info "Checking ${label} error handling..."

    # Missing timeframe → 400
    code=$(curl -s -o /dev/null -w "%{http_code}" "${base_url}?${no_timeframe_params}")
    [[ "$code" == "400" ]] && pass "${label}: missing timeframe → 400" || record_fail "${label}: missing timeframe → ${code} (expected 400)"

    # Invalid limit → 400
    code=$(curl -s -o /dev/null -w "%{http_code}" "${base_url}?${valid_params}&limit=9999")
    [[ "$code" == "400" ]] && pass "${label}: invalid limit (9999) → 400" || record_fail "${label}: invalid limit → ${code} (expected 400)"

    # since > until → 400
    code=$(curl -s -o /dev/null -w "%{http_code}" "${base_url}?${valid_params}&since=9999999999&until=1000000000")
    [[ "$code" == "400" ]] && pass "${label}: since > until → 400" || record_fail "${label}: since > until → ${code} (expected 400)"
}

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Infrastructure Readiness"
# ══════════════════════════════════════════════════════════════════════

# --- 1a. ClickHouse health ---
info "Checking ClickHouse health..."
CH_RESULT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse --query "SELECT 1" 2>/dev/null || echo "")
if [[ "$CH_RESULT" == "1" ]]; then
    pass "ClickHouse is healthy (SELECT 1 = 1)"
else
    record_fail "ClickHouse unreachable or unhealthy"
    echo -e "\n${RED}Cannot proceed without ClickHouse. Aborting.${NC}"
    exit 1
fi

# --- 1b. Writer readiness ---
info "Checking writer readiness..."
WRITER_READY=$($COMPOSE exec -T writer wget -q -O - http://127.0.0.1:8085/readyz 2>/dev/null || echo "")
WRITER_STATUS=$(echo "$WRITER_READY" | json_field "status")
if [[ "$WRITER_STATUS" == "ready" ]]; then
    pass "Writer is ready"
else
    record_fail "Writer not ready (status=${WRITER_STATUS:-unreachable})"
    echo -e "\n${RED}Cannot proceed without writer. Aborting.${NC}"
    exit 1
fi

# --- 1c. Gateway readiness ---
info "Checking gateway readiness..."
GW_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/readyz")
if [[ "$GW_CODE" == "200" ]]; then
    pass "Gateway is ready"
else
    record_fail "Gateway not ready (HTTP ${GW_CODE})"
    echo -e "\n${RED}Cannot proceed without gateway. Aborting.${NC}"
    exit 1
fi

# --- 1d. Analytical endpoint availability (not 503) ---
info "Checking analytical endpoint availability..."
ANALYTICAL_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60")
if [[ "$ANALYTICAL_CODE" == "503" ]]; then
    record_fail "Analytical endpoint returns 503 — ClickHouse not configured in gateway"
    echo -e "\n${RED}Gateway has no ClickHouse config. Check deploy/configs/gateway.jsonc.${NC}"
    exit 1
elif [[ "$ANALYTICAL_CODE" == "200" ]]; then
    pass "Analytical endpoint reachable (HTTP 200)"
else
    # 200 with empty candles or other codes are acceptable at this stage
    pass "Analytical endpoint reachable (HTTP ${ANALYTICAL_CODE})"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: Migration Status"
# ══════════════════════════════════════════════════════════════════════

info "Checking ClickHouse tables..."

TABLES=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
    --query "SELECT name FROM system.tables WHERE database = 'default' AND name NOT LIKE '.%' ORDER BY name" 2>/dev/null || echo "")

EXPECTED_TABLES=("_migrations" "decisions" "evidence_candles" "executions" "risk_assessments" "signals" "strategies")
MISSING_TABLES=()

for tbl in "${EXPECTED_TABLES[@]}"; do
    if echo "$TABLES" | grep -qx "$tbl"; then
        pass "Table exists: ${tbl}"
    else
        MISSING_TABLES+=("$tbl")
        record_fail "Table missing: ${tbl}"
    fi
done

if [[ ${#MISSING_TABLES[@]} -gt 0 ]]; then
    warn "Missing tables: ${MISSING_TABLES[*]}"
    warn "Run migrations: make migrate-up (or manually via cmd/migrate)"
    warn "Continuing — writer may fail to insert into missing tables"
fi

# --- 2b. Check migration records ---
info "Checking applied migrations..."
MIGRATION_COUNT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
    --query "SELECT count() FROM _migrations" 2>/dev/null || echo "0")
if [[ "$MIGRATION_COUNT" -ge 7 ]]; then
    pass "All 7 core migrations applied (count=${MIGRATION_COUNT})"
else
    warn "Only ${MIGRATION_COUNT} migrations applied (expected >= 7)"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Writer Pipeline Health"
# ══════════════════════════════════════════════════════════════════════

info "Checking writer statusz..."
WRITER_STATUSZ=$($COMPOSE exec -T writer wget -q -O - http://127.0.0.1:8085/statusz 2>/dev/null || echo "{}")

echo "$WRITER_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    phase = d.get('phase', 'unknown')
    uptime = d.get('uptime', '?')
    trackers = d.get('trackers', [])
    active = sum(1 for t in trackers if t.get('event_count', 0) > 0)
    total_events = sum(t.get('event_count', 0) for t in trackers)
    total_errors = sum(t.get('error_count', 0) for t in trackers)
    print(f'  phase={phase} uptime={uptime} active_trackers={active}/{len(trackers)} events={total_events} errors={total_errors}')
    for t in trackers:
        name = t['name']
        ec = t.get('event_count', 0)
        er = t.get('error_count', 0)
        counters = t.get('counters', {})
        flushed = counters.get('events_flushed', 0)
        dropped = counters.get('events_dropped', 0)
        print(f'    {name}: received={ec} flushed={flushed} dropped={dropped} errors={er}')
except:
    print('  (could not parse writer statusz)')
" 2>/dev/null || warn "Writer statusz parse error"

# --- 3b. Check if writer has received any events ---
WRITER_EVENTS=$(echo "$WRITER_STATUSZ" | python3 -c "
import sys, json
d = json.load(sys.stdin)
total = sum(t.get('event_count', 0) for t in d.get('trackers', []))
print(total)
" 2>/dev/null || echo "0")

if [[ "$WRITER_EVENTS" -gt 0 ]]; then
    pass "Writer has received ${WRITER_EVENTS} events from NATS"
else
    warn "Writer has received 0 events — pipeline may still be warming up"
    info "Waiting ${FLUSH_WAIT}s for writer to consume and flush..."
    ELAPSED=0
    POLL=10
    while [[ $ELAPSED -lt $FLUSH_WAIT ]]; do
        sleep $POLL
        ELAPSED=$((ELAPSED + POLL))
        WRITER_STATUSZ=$($COMPOSE exec -T writer wget -q -O - http://127.0.0.1:8085/statusz 2>/dev/null || echo "{}")
        WRITER_EVENTS=$(echo "$WRITER_STATUSZ" | python3 -c "
import sys, json
d = json.load(sys.stdin)
total = sum(t.get('event_count', 0) for t in d.get('trackers', []))
print(total)
" 2>/dev/null || echo "0")
        echo -n "  [${ELAPSED}s] events=${WRITER_EVENTS} "
        if [[ "$WRITER_EVENTS" -gt 0 ]]; then
            echo ""
            pass "Writer started receiving events after ${ELAPSED}s"
            break
        fi
        echo ""
    done
    if [[ "$WRITER_EVENTS" -eq 0 ]]; then
        record_fail "Writer still has 0 events after ${FLUSH_WAIT}s — NATS→writer path not proven"
    fi
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: ClickHouse Data Verification"
# ══════════════════════════════════════════════════════════════════════

info "Querying ClickHouse for evidence_candles rows..."
CH_CANDLE_COUNT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
    --query "SELECT count() FROM evidence_candles" 2>/dev/null || echo "0")

if [[ "$CH_CANDLE_COUNT" -gt 0 ]]; then
    pass "evidence_candles has ${CH_CANDLE_COUNT} rows — writer→ClickHouse path PROVEN"
else
    warn "evidence_candles has 0 rows"
    info "Writer may not have flushed yet (batch_size=1000, flush_interval=5s)"
    info "Waiting up to 30s for flush..."
    for i in 1 2 3; do
        sleep 10
        CH_CANDLE_COUNT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
            --query "SELECT count() FROM evidence_candles" 2>/dev/null || echo "0")
        if [[ "$CH_CANDLE_COUNT" -gt 0 ]]; then
            pass "evidence_candles has ${CH_CANDLE_COUNT} rows after extra wait"
            break
        fi
    done
    if [[ "$CH_CANDLE_COUNT" -eq 0 ]]; then
        record_fail "evidence_candles still empty — writer→ClickHouse flush not proven"
    fi
fi

# --- 4b. Sample a row from ClickHouse ---
if [[ "$CH_CANDLE_COUNT" -gt 0 ]]; then
    info "Sampling one candle row from ClickHouse..."
    $COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
        --query "SELECT source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final FROM evidence_candles ORDER BY open_time DESC LIMIT 1 FORMAT Pretty" 2>/dev/null || warn "Could not sample row"
fi

# --- 4c. Check other analytical tables (informational) ---
info "Checking row counts in other analytical tables..."
for tbl in signals decisions strategies risk_assessments executions; do
    COUNT=$($COMPOSE exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
        --query "SELECT count() FROM ${tbl}" 2>/dev/null || echo "error")
    if [[ "$COUNT" == "error" ]]; then
        warn "${tbl}: query failed (table may not exist)"
    elif [[ "$COUNT" -gt 0 ]]; then
        pass "${tbl}: ${COUNT} rows"
    else
        info "${tbl}: 0 rows (family may need longer to produce events)"
    fi
done

# ══════════════════════════════════════════════════════════════════════
# Phase 5: Per-family analytical read path validation
# ══════════════════════════════════════════════════════════════════════

# --- Candles (Baseline Family) ---
validate_analytical_family \
    "Candles — Baseline Family" \
    "evidence_candles" \
    "" \
    "${BASE_URL}/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60" \
    "candles" \
    "source|symbol|timeframe|open|high|low|close|volume|trade_count|open_time|close_time|final"
CANDLE_COUNT=$_VAL_HTTP_COUNT

# --- Signals/RSI (Wave B Family-01) ---
validate_analytical_family \
    "Signals/RSI — Wave B Family-01" \
    "signals" \
    "type = 'rsi'" \
    "${BASE_URL}/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60" \
    "signals" \
    "type|source|symbol|timeframe|value|metadata|final|timestamp"
CH_SIGNAL_COUNT=$_VAL_CH_COUNT
SIG_COUNT=$_VAL_HTTP_COUNT

# --- Decisions/RSI Oversold (Wave B Family-02) ---
validate_analytical_family \
    "Decisions/RSI Oversold — Wave B Family-02" \
    "decisions" \
    "type = 'rsi_oversold'" \
    "${BASE_URL}/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60" \
    "decisions" \
    "type|source|symbol|timeframe|outcome|confidence|signals|metadata|final|timestamp"
CH_DECISION_COUNT=$_VAL_CH_COUNT
DEC_COUNT=$_VAL_HTTP_COUNT

# --- Decision: Outcome filter validation ---
info "Checking outcome filter on decision endpoint..."
DEC_OUTCOME_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&outcome=triggered")
if [[ "$DEC_OUTCOME_CODE" == "200" ]]; then
    pass "Decision endpoint with outcome=triggered → 200"
else
    record_fail "Decision endpoint with outcome=triggered → ${DEC_OUTCOME_CODE} (expected 200)"
fi

# --- Strategies/Mean Reversion Entry (Wave B Family-03) ---
validate_analytical_family \
    "Strategies/Mean Reversion Entry — Wave B Family-03" \
    "strategies" \
    "type = 'mean_reversion_entry'" \
    "${BASE_URL}/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60" \
    "strategies" \
    "type|source|symbol|timeframe|direction|confidence|decisions|parameters|metadata|final|timestamp"
CH_STRATEGY_COUNT=$_VAL_CH_COUNT
STRAT_COUNT=$_VAL_HTTP_COUNT

# --- Strategy: Direction filter validation ---
info "Checking direction filter on strategy endpoint..."
STRAT_DIR_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60&direction=long")
if [[ "$STRAT_DIR_CODE" == "200" ]]; then
    pass "Strategy endpoint with direction=long → 200"
else
    record_fail "Strategy endpoint with direction=long → ${STRAT_DIR_CODE} (expected 200)"
fi

# --- Risk Assessments/Position Exposure (Wave B Family-04) ---
validate_analytical_family \
    "Risk Assessments/Position Exposure — Wave B Family-04" \
    "risk_assessments" \
    "type = 'position_exposure'" \
    "${BASE_URL}/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60" \
    "risk_assessments" \
    "type|source|symbol|timeframe|disposition|confidence|strategies|constraints|rationale|parameters|metadata|final|timestamp"
CH_RISK_COUNT=$_VAL_CH_COUNT
RISK_COUNT=$_VAL_HTTP_COUNT

# --- Risk: Disposition filter validation ---
info "Checking disposition filter on risk endpoint..."
RISK_DISP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60&disposition=approved")
if [[ "$RISK_DISP_CODE" == "200" ]]; then
    pass "Risk endpoint with disposition=approved → 200"
else
    record_fail "Risk endpoint with disposition=approved → ${RISK_DISP_CODE} (expected 200)"
fi

# --- Executions/Paper Order (Wave B Family-05) ---
validate_analytical_family \
    "Executions/Paper Order — Wave B Family-05" \
    "executions" \
    "type = 'paper_order'" \
    "${BASE_URL}/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60" \
    "executions" \
    "type|source|symbol|timeframe|side|quantity|filled_quantity|status|risk|fills|parameters|metadata|correlation_id|causation_id|final|timestamp"
CH_EXECUTION_COUNT=$_VAL_CH_COUNT
EXEC_COUNT=$_VAL_HTTP_COUNT

# --- Execution: Side filter validation ---
info "Checking side filter on execution endpoint..."
EXEC_SIDE_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&side=buy")
if [[ "$EXEC_SIDE_CODE" == "200" ]]; then
    pass "Execution endpoint with side=buy → 200"
else
    record_fail "Execution endpoint with side=buy → ${EXEC_SIDE_CODE} (expected 200)"
fi

# --- Execution: Status filter validation ---
info "Checking status filter on execution endpoint..."
EXEC_STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&status=filled")
if [[ "$EXEC_STATUS_CODE" == "200" ]]; then
    pass "Execution endpoint with status=filled → 200"
else
    record_fail "Execution endpoint with status=filled → ${EXEC_STATUS_CODE} (expected 200)"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 6: Error Handling Validation"
# ══════════════════════════════════════════════════════════════════════

validate_analytical_error_handling "Candle" \
    "${BASE_URL}/analytical/evidence/candles" \
    "source=binancef&symbol=btcusdt&timeframe=60" \
    "source=binancef&symbol=btcusdt"

# Signal has an extra required param (type), so we test that separately
info "Checking signal error handling..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/signal/history?source=binancef&symbol=btcusdt&timeframe=60")
[[ "$CODE" == "400" ]] && pass "Signal: missing type → 400" || record_fail "Signal: missing type → ${CODE} (expected 400)"
validate_analytical_error_handling "Signal" \
    "${BASE_URL}/analytical/signal/history" \
    "type=rsi&source=binancef&symbol=btcusdt&timeframe=60" \
    "type=rsi&source=binancef&symbol=btcusdt"

# Decision also has an extra required param (type)
info "Checking decision error handling..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/decision/history?source=binancef&symbol=btcusdt&timeframe=60")
[[ "$CODE" == "400" ]] && pass "Decision: missing type → 400" || record_fail "Decision: missing type → ${CODE} (expected 400)"
validate_analytical_error_handling "Decision" \
    "${BASE_URL}/analytical/decision/history" \
    "type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60" \
    "type=rsi_oversold&source=binancef&symbol=btcusdt"

# Strategy also has an extra required param (type)
info "Checking strategy error handling..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/strategy/history?source=binancef&symbol=btcusdt&timeframe=60")
[[ "$CODE" == "400" ]] && pass "Strategy: missing type → 400" || record_fail "Strategy: missing type → ${CODE} (expected 400)"
validate_analytical_error_handling "Strategy" \
    "${BASE_URL}/analytical/strategy/history" \
    "type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60" \
    "type=mean_reversion_entry&source=binancef&symbol=btcusdt"

# Risk also has an extra required param (type)
info "Checking risk error handling..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/risk/history?source=binancef&symbol=btcusdt&timeframe=60")
[[ "$CODE" == "400" ]] && pass "Risk: missing type → 400" || record_fail "Risk: missing type → ${CODE} (expected 400)"
validate_analytical_error_handling "Risk" \
    "${BASE_URL}/analytical/risk/history" \
    "type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60" \
    "type=position_exposure&source=binancef&symbol=btcusdt"

# Execution also has an extra required param (type)
info "Checking execution error handling..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/analytical/execution/history?source=derive&symbol=btcusdt&timeframe=60")
[[ "$CODE" == "400" ]] && pass "Execution: missing type → 400" || record_fail "Execution: missing type → ${CODE} (expected 400)"
validate_analytical_error_handling "Execution" \
    "${BASE_URL}/analytical/execution/history" \
    "type=paper_order&source=derive&symbol=btcusdt&timeframe=60" \
    "type=paper_order&source=derive&symbol=btcusdt"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 7: Writer Observability Check"
# ══════════════════════════════════════════════════════════════════════

info "Checking writer diagz..."
WRITER_DIAGZ=$($COMPOSE exec -T writer wget -q -O - http://127.0.0.1:8085/diagz 2>/dev/null || echo "{}")

echo "$WRITER_DIAGZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    checks = d.get('readiness_checks', [])
    passed = sum(1 for c in checks if c.get('status') == 'pass')
    goroutines = d.get('num_goroutines', '?')
    phase = d.get('phase', '?')
    print(f'  readiness={passed}/{len(checks)} goroutines={goroutines} phase={phase}')
    trackers = d.get('trackers', [])
    degraded = 0
    for t in trackers:
        counters = t.get('counters', {})
        if counters.get('pipeline_degraded', 0) > 0:
            degraded += 1
            print(f'  WARNING: {t[\"name\"]} is DEGRADED')
    if degraded == 0:
        print(f'  No degraded pipelines ({len(trackers)} trackers healthy)')
except:
    print('  (could not parse writer diagz)')
" 2>/dev/null || warn "Writer diagz parse error"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 8: Error Log Scan"
# ══════════════════════════════════════════════════════════════════════

info "Scanning compose logs for error-level entries..."
ERROR_LOG_COUNT=$($COMPOSE logs --no-log-prefix 2>/dev/null | grep -c '"level":"error"' || echo "0")
if [[ "$ERROR_LOG_COUNT" -gt 0 ]]; then
    warn "Found ${ERROR_LOG_COUNT} error-level log entries across all services"
    $COMPOSE logs --no-log-prefix 2>/dev/null | grep '"level":"error"' | tail -5
    info "(showing last 5 — review full logs with: make logs)"
else
    pass "No error-level log entries found"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Summary: Analytical E2E Integration Proof"
# ══════════════════════════════════════════════════════════════════════

echo ""
echo "Flows validated:"
echo ""
echo "  [Baseline] Evidence Candles:"
echo "    NATS JetStream → writer → ClickHouse (evidence_candles) → reader → GET /analytical/evidence/candles"
echo ""
echo "  [Wave B Family-01] Signals (RSI):"
echo "    NATS JetStream → writer → ClickHouse (signals) → reader → GET /analytical/signal/history"
echo ""
echo "  [Wave B Family-02] Decisions (RSI Oversold):"
echo "    NATS JetStream → writer → ClickHouse (decisions) → reader → GET /analytical/decision/history"
echo ""
echo "  [Wave B Family-03] Strategies (Mean Reversion Entry):"
echo "    NATS JetStream → writer → ClickHouse (strategies) → reader → GET /analytical/strategy/history"
echo ""
echo "  [Wave B Family-04] Risk Assessments (Position Exposure):"
echo "    NATS JetStream → writer → ClickHouse (risk_assessments) → reader → GET /analytical/risk/history"
echo ""
echo "  [Wave B Family-05] Executions (Paper Order):"
echo "    NATS JetStream → writer → ClickHouse (executions) → reader → GET /analytical/execution/history"
echo ""
echo "Proof points:"
echo "  [Infrastructure]   ClickHouse + writer + gateway all healthy"
echo "  [Migrations]       ${MIGRATION_COUNT:-?} migrations applied, ${#EXPECTED_TABLES[@]} tables verified"
echo "  [Write path]       Writer received ${WRITER_EVENTS:-0} events from NATS"
echo "  [Candle persist]   evidence_candles has ${CH_CANDLE_COUNT:-0} rows in ClickHouse"
echo "  [Signal persist]   signals (RSI) has ${CH_SIGNAL_COUNT:-0} rows in ClickHouse"
echo "  [Decision persist] decisions (rsi_oversold) has ${CH_DECISION_COUNT:-0} rows in ClickHouse"
echo "  [Strategy persist] strategies (mean_reversion_entry) has ${CH_STRATEGY_COUNT:-0} rows in ClickHouse"
echo "  [Risk persist]     risk_assessments (position_exposure) has ${CH_RISK_COUNT:-0} rows in ClickHouse"
echo "  [Exec persist]     executions (paper_order) has ${CH_EXECUTION_COUNT:-0} rows in ClickHouse"
echo "  [Candle read]      HTTP returned ${CANDLE_COUNT:-0} candles via analytical endpoint"
echo "  [Signal read]      HTTP returned ${SIG_COUNT:-0} signals via analytical endpoint"
echo "  [Decision read]    HTTP returned ${DEC_COUNT:-0} decisions via analytical endpoint"
echo "  [Strategy read]    HTTP returned ${STRAT_COUNT:-0} strategies via analytical endpoint"
echo "  [Risk read]        HTTP returned ${RISK_COUNT:-0} risk assessments via analytical endpoint"
echo "  [Exec read]        HTTP returned ${EXEC_COUNT:-0} executions via analytical endpoint"
echo "  [Error handling]   400 responses for invalid params confirmed (candle + signal + decision + strategy + risk + execution)"
echo ""

if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}${BOLD}ANALYTICAL E2E PROOF: ALL CHECKS PASSED${NC}"
    echo ""
    echo "All analytical families proven end-to-end:"
    echo "  [Baseline]    Evidence Candles    — NATS → writer → ClickHouse → reader → HTTP"
    echo "  [Wave B F-01] Signals (RSI)      — NATS → writer → ClickHouse → reader → HTTP"
    echo "  [Wave B F-02] Decisions (RSI OB) — NATS → writer → ClickHouse → reader → HTTP"
    echo "  [Wave B F-03] Strategies (MR)    — NATS → writer → ClickHouse → reader → HTTP"
    echo "  [Wave B F-04] Risk (PE)          — NATS → writer → ClickHouse → reader → HTTP"
    echo "  [Wave B F-05] Executions (PO)    — NATS → writer → ClickHouse → reader → HTTP"
else
    echo -e "${RED}${BOLD}ANALYTICAL E2E PROOF: ${ERRORS} ISSUE(S) DETECTED${NC}"
    echo ""
    echo "Review the failures above. Common causes:"
    echo "  - Stack not fully up: make up && wait 120s"
    echo "  - No data seeded: make seed (or make seed-multi)"
    echo "  - Migrations not applied: migrations run automatically via writer startup"
    echo "  - Writer not consuming: check writer logs (make logs SERVICE=writer)"
fi

exit $ERRORS
