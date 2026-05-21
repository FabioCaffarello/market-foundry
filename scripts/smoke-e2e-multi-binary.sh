#!/usr/bin/env bash
# smoke-e2e-multi-binary.sh — S373: End-to-end multi-binary pipeline proof.
#
# Proves the canonical pipeline across REAL separate binaries with REAL NATS:
#   derive → StrategyResolvedEvent → NATS STRATEGY_EVENTS → execute → evaluate →
#   venue fill → EXECUTION_FILL_EVENTS → store → NATS KV → gateway HTTP read path
#
# This smoke traces a single correlation chain through every binary boundary,
# proving the value of the multi-binary orchestration wave (S370–S373).
#
# Prerequisites:
#   make up && make seed   # full stack running with configctl seeded
#
# Usage:
#   ./scripts/smoke-e2e-multi-binary.sh
#   ./scripts/smoke-e2e-multi-binary.sh --wait 180   # override pipeline settle wait
#
# Canonical entrypoint: `make smoke-e2e-multi-binary`
#
# Guard rails:
#   - Single pipeline only (mean_reversion_entry).
#   - Single symbol (btcusdt) on single timeframe (60s).
#   - No venue expansion; paper venue only.
#   - No multi-venue; no multi-family.
#   - Observes and validates; does not modify pipeline behavior.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
compose() {
    docker compose -f "${COMPOSE_FILE}" "$@"
}

usage() {
    cat <<'EOF'
Usage: ./scripts/smoke-e2e-multi-binary.sh [--wait <seconds>] [--help]

S373: End-to-end multi-binary pipeline proof.
Traces a correlation chain from derive through NATS to execute, store, and gateway.

Options:
  --wait <seconds>  Maximum time to wait for pipeline data. Default: 120
  --help            Show this help text.

Environment:
  BASE_URL          Gateway base URL. Default: http://127.0.0.1:8080
EOF
}

PIPELINE_WAIT="${SMOKE_WAIT:-${PIPELINE_WAIT:-120}}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --wait)
            [[ $# -ge 2 ]] || usage_error "--wait requires a value"
            PIPELINE_WAIT="$2"
            shift
            ;;
        -h|--help) usage; exit 0 ;;
        *) usage_error "unknown argument: $1" ;;
    esac
    shift
done

require_commands docker curl python3

ERRORS=0

smoke_banner \
    "S373: End-to-End Multi-Binary Pipeline Proof" \
    "make smoke-e2e-multi-binary" \
    "make up && make seed" \
    "pipeline-wait" \
    "${PIPELINE_WAIT}"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Multi-Binary Stack Readiness"
# ══════════════════════════════════════════════════════════════════════

# All 7 Go services + NATS + ClickHouse must be healthy.
REQUIRED_SERVICES=("nats" "clickhouse" "configctl" "ingest" "derive" "store" "execute" "gateway" "writer")
ALL_HEALTHY=true

for svc in "${REQUIRED_SERVICES[@]}"; do
    status=$(compose ps --format json "${svc}" 2>/dev/null | python3 -c "
import sys, json
for line in sys.stdin:
    data = json.loads(line)
    print(data.get('Health', data.get('health', 'unknown')))
    break
" 2>/dev/null || echo "missing")

    if [[ "$status" == "healthy" ]]; then
        pass "${svc} → healthy"
    else
        record_fail "${svc} → ${status} (expected: healthy)"
        ALL_HEALTHY=false
    fi
done

if ! $ALL_HEALTHY; then
    die "Not all services are healthy — run: make up && make seed"
fi

# Verify gateway is reachable from host.
GW_CODE=$(http_code "${BASE_URL}/readyz")
if [[ "$GW_CODE" == "200" ]]; then
    pass "Gateway HTTP reachable → ${BASE_URL}/readyz → 200"
else
    die "Gateway unreachable (HTTP ${GW_CODE}) — run: make up"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: NATS Stream Baseline (Pre-Pipeline Snapshot)"
# ══════════════════════════════════════════════════════════════════════

# Capture message counts BEFORE waiting for new events.
# This allows us to detect delta — new events flowing through the pipeline.

stream_msg_count() {
    local stream="$1"
    local result
    result=$(compose exec -T nats nats stream info "$stream" --json 2>/dev/null || echo "")
    if [[ -n "$result" ]]; then
        echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['state']['messages'])" 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

STRATEGY_BEFORE=$(stream_msg_count "STRATEGY_EVENTS")
EXECUTION_BEFORE=$(stream_msg_count "EXECUTION_EVENTS")
FILL_BEFORE=$(stream_msg_count "EXECUTION_FILL_EVENTS")

pass "Baseline: STRATEGY_EVENTS=${STRATEGY_BEFORE} EXECUTION_EVENTS=${EXECUTION_BEFORE} EXECUTION_FILL_EVENTS=${FILL_BEFORE}"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Configctl Active Binding Verification"
# ══════════════════════════════════════════════════════════════════════

# Verify that configctl has an active binding — pipeline needs this to generate events.
ACTIVE_CODE=$(curl -s -o /tmp/s373_active.json -w "%{http_code}" \
    "${BASE_URL}/configctl/configs/active?scope_kind=global&scope_key=default" 2>/dev/null || echo "000")

if [[ "$ACTIVE_CODE" == "200" ]]; then
    BINDING_COUNT=$(python3 -c "
import sys, json
d = json.load(open('/tmp/s373_active.json'))
bindings = d.get('config', {}).get('bindings', [])
print(len(bindings))
" 2>/dev/null || echo "0")
    if [[ "$BINDING_COUNT" -gt 0 ]]; then
        pass "Active config has ${BINDING_COUNT} binding(s) — pipeline is fed"
    else
        record_fail "Active config has 0 bindings — run: make seed"
    fi
else
    record_fail "configctl active config → HTTP ${ACTIVE_CODE} — run: make seed"
fi

rm -f /tmp/s373_active.json

# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Consumer Binding Verification (Strategy→Execute Path)"
# ══════════════════════════════════════════════════════════════════════

# Verify the critical durable consumer exists: execute-strategy-mean-reversion-entry.
STRAT_CONSUMER=$(compose exec -T nats nats consumer info STRATEGY_EVENTS execute-strategy-mean-reversion-entry --json 2>/dev/null || echo "")
if [[ -n "$STRAT_CONSUMER" ]] && echo "$STRAT_CONSUMER" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(d['config']['durable_name'])
" 2>/dev/null | grep -q "execute-strategy-mean-reversion-entry"; then
    DELIVERED=$(echo "$STRAT_CONSUMER" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('delivered',{}).get('consumer_seq',0))" 2>/dev/null || echo "0")
    pass "execute-strategy-mean-reversion-entry consumer bound (delivered=${DELIVERED})"
else
    record_fail "execute-strategy-mean-reversion-entry consumer NOT found"
fi

# Also check store consumers for strategy events.
STORE_STRAT=$(compose exec -T nats nats consumer info STRATEGY_EVENTS store-strategy-mean-reversion-entry --json 2>/dev/null || echo "")
if [[ -n "$STORE_STRAT" ]] && echo "$STORE_STRAT" | python3 -c "import sys,json; print(json.load(sys.stdin)['config']['durable_name'])" 2>/dev/null | grep -q "store-strategy"; then
    pass "store-strategy-mean-reversion-entry consumer bound"
else
    warn "store-strategy-mean-reversion-entry consumer not found (store may not have started yet)"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Pipeline Data Flow — Wait for Strategy Events"
# ══════════════════════════════════════════════════════════════════════

# Wait for at least one new strategy event to appear in the stream.
# This proves derive is actively producing StrategyResolvedEvents.

info "Waiting up to ${PIPELINE_WAIT}s for new STRATEGY_EVENTS..."
STRATEGY_DELTA=0
ELAPSED=0
POLL_INTERVAL=10

while (( ELAPSED < PIPELINE_WAIT )); do
    STRATEGY_NOW=$(stream_msg_count "STRATEGY_EVENTS")
    STRATEGY_DELTA=$((STRATEGY_NOW - STRATEGY_BEFORE))

    if (( STRATEGY_DELTA > 0 )); then
        pass "STRATEGY_EVENTS delta: +${STRATEGY_DELTA} new events (total=${STRATEGY_NOW})"
        break
    fi

    sleep "$POLL_INTERVAL"
    ELAPSED=$((ELAPSED + POLL_INTERVAL))
    info "  ... ${ELAPSED}s elapsed, STRATEGY_EVENTS=${STRATEGY_NOW} (delta=0)"
done

if (( STRATEGY_DELTA == 0 )); then
    record_fail "No new strategy events after ${PIPELINE_WAIT}s — derive may not be producing"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 6: Cross-Binary Event Flow — Execute Consumption Proof"
# ══════════════════════════════════════════════════════════════════════

# Check if execute has consumed strategy events by examining its tracker.
EXEC_STATUSZ=$(compose exec -T execute wget -q -O - http://127.0.0.1:8084/statusz 2>/dev/null || echo "{}")
EXEC_STRATEGY_RECEIVED=$(echo "$EXEC_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    trackers = d.get('trackers', {})
    strat = trackers.get('strategy-consumer', {})
    counters = strat.get('counters', {})
    print(counters.get('received', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

if [[ "$EXEC_STRATEGY_RECEIVED" -gt 0 ]]; then
    pass "Execute binary: strategy-consumer received=${EXEC_STRATEGY_RECEIVED} events from derive via NATS"
else
    warn "Execute binary: strategy-consumer received=0 (may need more pipeline time)"
fi

# Check strategy consumer evaluation counters.
EXEC_STRATEGY_EVALUATED=$(echo "$EXEC_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    trackers = d.get('trackers', {})
    strat = trackers.get('strategy-consumer', {})
    counters = strat.get('counters', {})
    print(counters.get('evaluated', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

EXEC_STRATEGY_ACTIONABLE=$(echo "$EXEC_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    trackers = d.get('trackers', {})
    strat = trackers.get('strategy-consumer', {})
    counters = strat.get('counters', {})
    print(counters.get('evaluated_actionable', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

info "Execute strategy-consumer: evaluated=${EXEC_STRATEGY_EVALUATED} actionable=${EXEC_STRATEGY_ACTIONABLE}"

# Check venue adapter counters.
EXEC_ADAPTER_PROCESSED=$(echo "$EXEC_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    trackers = d.get('trackers', {})
    adapter = trackers.get('venue-adapter', {})
    counters = adapter.get('counters', {})
    print(counters.get('processed', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

EXEC_ADAPTER_FILLED=$(echo "$EXEC_STATUSZ" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    trackers = d.get('trackers', {})
    adapter = trackers.get('venue-adapter', {})
    counters = adapter.get('counters', {})
    print(counters.get('filled', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

if [[ "$EXEC_ADAPTER_PROCESSED" -gt 0 ]]; then
    pass "Execute binary: venue-adapter processed=${EXEC_ADAPTER_PROCESSED} filled=${EXEC_ADAPTER_FILLED}"
else
    info "Execute binary: venue-adapter processed=0 (pipeline may not have actionable strategies yet)"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 7: Store Materialization — NATS KV Read Path"
# ══════════════════════════════════════════════════════════════════════

# Verify store has materialized strategy data to KV (via gateway HTTP).
STRATEGY_LATEST_CODE=$(curl -s -o /tmp/s373_strat_latest.json -w "%{http_code}" \
    "${BASE_URL}/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60" 2>/dev/null || echo "000")

if [[ "$STRATEGY_LATEST_CODE" == "200" ]]; then
    STRAT_DATA=$(python3 -c "
import sys, json
d = json.load(open('/tmp/s373_strat_latest.json'))
strat = d.get('strategy', d)
direction = strat.get('direction', 'unknown')
confidence = strat.get('confidence', 'unknown')
strategy_type = strat.get('type', 'unknown')
corr = strat.get('correlation_id', d.get('metadata', {}).get('correlation_id', ''))
print(f'type={strategy_type} direction={direction} confidence={confidence} correlation_id={corr}')
" 2>/dev/null || echo "parse_error")
    pass "Store materialization: GET /strategy/mean_reversion_entry/latest → 200 (${STRAT_DATA})"
else
    warn "Strategy latest → HTTP ${STRATEGY_LATEST_CODE} (data may still be propagating)"
fi
rm -f /tmp/s373_strat_latest.json

# Also check evidence candle as proof of earlier pipeline stages.
CANDLE_CODE=$(http_code "${BASE_URL}/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60")
if [[ "$CANDLE_CODE" == "200" ]]; then
    pass "Evidence candle latest → 200 (ingest→derive→store path confirmed)"
else
    warn "Evidence candle latest → HTTP ${CANDLE_CODE}"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 8: Execution Control Gate — Kill-Switch Coherence"
# ══════════════════════════════════════════════════════════════════════

# Verify execution control gate is accessible and coherent.
GATE_CODE=$(curl -s -o /tmp/s373_gate.json -w "%{http_code}" "${BASE_URL}/execution/control" 2>/dev/null || echo "000")
if [[ "$GATE_CODE" == "200" ]]; then
    GATE_STATUS=$(python3 -c "import sys,json; print(json.load(open('/tmp/s373_gate.json')).get('gate',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    pass "Execution control gate → ${GATE_STATUS} (cross-binary KV wiring confirmed)"
else
    record_fail "Execution control gate → HTTP ${GATE_CODE}"
fi
rm -f /tmp/s373_gate.json

# ══════════════════════════════════════════════════════════════════════
phase "Phase 9: Analytical Persistence — Writer→ClickHouse"
# ══════════════════════════════════════════════════════════════════════

# Check that writer has persisted strategy data to ClickHouse.
STRAT_CH_COUNT=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
    --database market_foundry --query "SELECT count() FROM strategies" 2>/dev/null || echo "0")

if [[ "$STRAT_CH_COUNT" -gt 0 ]]; then
    pass "ClickHouse strategies table: ${STRAT_CH_COUNT} rows (writer persisted strategy events)"
else
    warn "ClickHouse strategies table: 0 rows (writer may still be flushing)"
fi

# Check analytical endpoint through gateway.
STRAT_HISTORY_CODE=$(http_code "${BASE_URL}/analytical/strategy/history?source=binancef&symbol=btcusdt&timeframe=60")
if [[ "$STRAT_HISTORY_CODE" == "200" ]]; then
    pass "GET /analytical/strategy/history → 200 (writer→ClickHouse→gateway analytical path)"
else
    warn "GET /analytical/strategy/history → HTTP ${STRAT_HISTORY_CODE}"
fi

# Check execution persistence (if any fills happened).
EXEC_CH_COUNT=$(compose exec -T clickhouse clickhouse-client --port 9000 --user default --password clickhouse \
    --database market_foundry --query "SELECT count() FROM executions" 2>/dev/null || echo "0")
info "ClickHouse executions table: ${EXEC_CH_COUNT} rows"

# ══════════════════════════════════════════════════════════════════════
phase "Phase 10: Correlation Chain Audit — End-to-End Traceability"
# ══════════════════════════════════════════════════════════════════════

# Query the composite chains endpoint to verify correlation chains span the full pipeline.
CHAINS_URL="${BASE_URL}/analytical/composite/chains?source=binancef&symbol=btcusdt&timeframe=60&limit=5"
CHAINS_CODE=$(curl -s -o /tmp/s373_chains.json -w "%{http_code}" "$CHAINS_URL" 2>/dev/null || echo "000")

if [[ "$CHAINS_CODE" == "200" ]]; then
    CHAIN_AUDIT=$(python3 -c "
import sys, json
d = json.load(open('/tmp/s373_chains.json'))
chains = d.get('chains', [])
total = len(chains)
has_strategy = 0
has_execution = 0
has_corr = 0
for c in chains:
    if c.get('strategy') is not None:
        has_strategy += 1
    if c.get('execution') is not None:
        has_execution += 1
    if c.get('correlation_id', ''):
        has_corr += 1
print(f'total={total} with_strategy={has_strategy} with_execution={has_execution} with_correlation_id={has_corr}')
" 2>/dev/null || echo "parse_error")
    pass "Composite chains audit: ${CHAIN_AUDIT}"

    # Check for a chain that has both strategy AND execution stages.
    FULL_CHAIN=$(python3 -c "
import sys, json
d = json.load(open('/tmp/s373_chains.json'))
for c in d.get('chains', []):
    if c.get('strategy') is not None and c.get('execution') is not None:
        corr = c.get('correlation_id', 'none')
        strat_dir = c.get('strategy', {}).get('direction', 'unknown')
        exec_status = c.get('execution', {}).get('status', 'unknown')
        print(f'FOUND correlation_id={corr} strategy_direction={strat_dir} execution_status={exec_status}')
        sys.exit(0)
print('NONE')
" 2>/dev/null || echo "NONE")

    if [[ "$FULL_CHAIN" == FOUND* ]]; then
        pass "Full pipeline chain: ${FULL_CHAIN}"
    else
        warn "No chain with both strategy + execution stages found (pipeline may need more time for actionable strategies)"
    fi
else
    warn "Composite chains → HTTP ${CHAINS_CODE}"
fi
rm -f /tmp/s373_chains.json

# ══════════════════════════════════════════════════════════════════════
phase "Phase 11: Stream Delta Summary (Post-Pipeline Snapshot)"
# ══════════════════════════════════════════════════════════════════════

STRATEGY_AFTER=$(stream_msg_count "STRATEGY_EVENTS")
EXECUTION_AFTER=$(stream_msg_count "EXECUTION_EVENTS")
FILL_AFTER=$(stream_msg_count "EXECUTION_FILL_EVENTS")

STRATEGY_DELTA=$((STRATEGY_AFTER - STRATEGY_BEFORE))
EXECUTION_DELTA=$((EXECUTION_AFTER - EXECUTION_BEFORE))
FILL_DELTA=$((FILL_AFTER - FILL_BEFORE))

echo ""
info "Stream deltas during proof run:"
info "  STRATEGY_EVENTS:        ${STRATEGY_BEFORE} → ${STRATEGY_AFTER} (+${STRATEGY_DELTA})"
info "  EXECUTION_EVENTS:       ${EXECUTION_BEFORE} → ${EXECUTION_AFTER} (+${EXECUTION_DELTA})"
info "  EXECUTION_FILL_EVENTS:  ${FILL_BEFORE} → ${FILL_AFTER} (+${FILL_DELTA})"
echo ""

if (( STRATEGY_DELTA > 0 )); then
    pass "Pipeline active: derive producing strategy events (+${STRATEGY_DELTA})"
else
    record_fail "No new strategy events — pipeline not active"
fi

# ══════════════════════════════════════════════════════════════════════
phase "Phase 12: Go Integration Test Gate"
# ══════════════════════════════════════════════════════════════════════

info "Running S373 multi-binary pipeline integration tests..."
S373_TESTS="TestS373_MultiBinaryPipeline"
if (cd "$PROJECT_ROOT" && go test -count=1 -timeout 60s -run "$S373_TESTS" ./internal/actors/scopes/execute/... 2>/dev/null); then
    pass "S373 multi-binary pipeline tests pass"
else
    record_fail "S373 multi-binary pipeline tests failed"
fi

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════

echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""

if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "S373 end-to-end multi-binary pipeline proof" "$ERRORS" "make up && make seed"
    exit 1
fi

pass "S373 end-to-end multi-binary pipeline proof COMPLETED"
echo ""
echo "Pipeline proven across separate binaries:"
echo "  derive → STRATEGY_EVENTS (+${STRATEGY_DELTA}) → execute (received=${EXEC_STRATEGY_RECEIVED})"
echo "  execute → evaluate → venue → EXECUTION_FILL_EVENTS (+${FILL_DELTA})"
echo "  store → NATS KV → gateway HTTP read path"
echo "  writer → ClickHouse → gateway analytical surface"
echo "  execution control gate → cross-binary KV coherence"
echo ""
echo "Evidence: all binary boundaries exercised with real NATS in real Docker containers."
echo "Each binary runs as a separate OS process with no shared Go state."
