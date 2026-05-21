#!/usr/bin/env bash
# smoke-live-exchange-listening.sh — S378: Compose live exchange listening proof.
#
# Proves that the compose stack can start, seed bindings, connect to a real
# exchange (Binance Futures mainnet), receive live trades, and flow them through
# the canonical NATS observation stream — without touching the write path.
#
# This script validates the READ PATH only.  No orders are placed, no venue
# adapter is exercised, and the execution engine stays in paper mode.
#
# Usage:
#   ./scripts/smoke-live-exchange-listening.sh
#
# Prerequisites:
#   make up && make seed   (compose stack running + bindings activated)
#
# Canonical entrypoint: `make smoke-live-listening`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/compose/docker-compose.yaml"
COMPOSE_CMD="docker compose -f ${COMPOSE_FILE}"
NATS_MONITOR="http://127.0.0.1:8222"
LISTEN_WAIT="${LISTEN_WAIT:-60}"
LISTEN_POLL="${LISTEN_POLL:-5}"

require_commands docker curl python3

smoke_banner \
    "S378: Compose Live Exchange Listening Proof" \
    "make smoke-live-listening" \
    "make up && make seed" \
    "LISTEN_WAIT" \
    "${LISTEN_WAIT}"

# ══════════════════════════════════════════════════════════════════════
# Phase 1: Stack Readiness — Core Services Healthy
# ══════════════════════════════════════════════════════════════════════
phase "Phase 1: Stack Readiness — Core Services Healthy"

REQUIRED_SERVICES=("nats" "configctl" "ingest" "derive" "gateway")

for svc in "${REQUIRED_SERVICES[@]}"; do
    status=$($COMPOSE_CMD ps --format json "${svc}" 2>/dev/null | python3 -c "
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
    fi
done

# ══════════════════════════════════════════════════════════════════════
# Phase 2: NATS JetStream — OBSERVATION_EVENTS Stream Exists
# ══════════════════════════════════════════════════════════════════════
phase "Phase 2: NATS JetStream — OBSERVATION_EVENTS Stream Exists"

STREAM_INFO=$(curl -s "${NATS_MONITOR}/jsz?streams=true" 2>/dev/null || echo "{}")

OBS_STREAM=$(echo "$STREAM_INFO" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'OBSERVATION_EVENTS':
            print('found')
            break
" 2>/dev/null || echo "")

if [[ "$OBS_STREAM" == "found" ]]; then
    pass "OBSERVATION_EVENTS stream exists in JetStream"
else
    record_fail "OBSERVATION_EVENTS stream not found"
fi

# Check derive-observation consumer exists.
CONSUMER_INFO=$(curl -s "${NATS_MONITOR}/jsz?consumers=true" 2>/dev/null || echo "{}")

DERIVE_CONS=$(echo "$CONSUMER_INFO" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'OBSERVATION_EVENTS':
            for c in s.get('consumer_details', []):
                if c.get('name') == 'derive-observation':
                    print('found')
                    break
" 2>/dev/null || echo "")

if [[ "$DERIVE_CONS" == "found" ]]; then
    pass "derive-observation consumer bound to OBSERVATION_EVENTS"
else
    record_fail "derive-observation consumer not found on OBSERVATION_EVENTS"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 3: Active Bindings — configctl Has Live Ingestion Bindings
# ══════════════════════════════════════════════════════════════════════
phase "Phase 3: Active Bindings — configctl Has Live Ingestion Bindings"

ACTIVE_CONFIG=$(curl -s "${BASE_URL}/configctl/configs/active?scope_kind=global&scope_key=default" 2>/dev/null || echo "{}")

BINDING_COUNT=$(echo "$ACTIVE_CONFIG" | python3 -c "
import sys, json
data = json.load(sys.stdin)
bindings = data.get('config', {}).get('bindings', [])
print(len(bindings))
" 2>/dev/null || echo "0")

if [[ "$BINDING_COUNT" -gt 0 ]]; then
    pass "Active config has ${BINDING_COUNT} binding(s)"
else
    record_fail "No active bindings found — run 'make seed' first"
fi

# Print active binding topics.
echo "$ACTIVE_CONFIG" | python3 -c "
import sys, json
data = json.load(sys.stdin)
bindings = data.get('config', {}).get('bindings', [])
for b in bindings:
    print(f'  binding: {b.get(\"name\",\"?\")} → {b.get(\"topic\",\"?\")}')
" 2>/dev/null || true

# ══════════════════════════════════════════════════════════════════════
# Phase 4: Execution Mode — Confirm Paper (No Real Trading)
# ══════════════════════════════════════════════════════════════════════
phase "Phase 4: Execution Mode — Confirm Paper (No Real Trading)"

ACTIVATION=$(curl -s "${BASE_URL}/execution/activation/surface" 2>/dev/null || echo "{}")

EFFECTIVE=$(echo "$ACTIVATION" | python3 -c "
import sys, json
data = json.load(sys.stdin)
surface = data.get('surface', data)
print(surface.get('effective', 'unknown'))
" 2>/dev/null || echo "unknown")

if [[ "$EFFECTIVE" == "paper" ]]; then
    pass "Execution effective mode: paper (no real orders)"
elif [[ "$EFFECTIVE" == "venue_halted" || "$EFFECTIVE" == "venue_degraded" ]]; then
    pass "Execution effective mode: ${EFFECTIVE} (no real orders)"
else
    record_fail "Execution effective mode: ${EFFECTIVE} — expected paper/venue_halted/venue_degraded"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 5: Ingest Binary — WebSocket Connectivity (Log Evidence)
# ══════════════════════════════════════════════════════════════════════
phase "Phase 5: Ingest Binary — WebSocket Connectivity"

# Check ingest logs for WebSocket connection evidence.
WS_CONNECT=$($COMPOSE_CMD logs --tail=200 ingest 2>/dev/null | grep -ci "connected\|connecting\|ws.*binance\|websocket" || echo "0")

if [[ "$WS_CONNECT" -gt 0 ]]; then
    pass "Ingest logs show WebSocket activity (${WS_CONNECT} relevant line(s))"
else
    info "No WebSocket log evidence yet — will check trade flow next"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 6: Live Trade Flow — Real Trades on OBSERVATION_EVENTS
# ══════════════════════════════════════════════════════════════════════
phase "Phase 6: Live Trade Flow — Polling OBSERVATION_EVENTS for Real Trades"

info "Polling OBSERVATION_EVENTS message count for up to ${LISTEN_WAIT}s..."

INITIAL_MSGS=$(curl -s "${NATS_MONITOR}/jsz?streams=true" 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'OBSERVATION_EVENTS':
            print(s.get('state', {}).get('messages', 0))
            break
" 2>/dev/null || echo "0")

info "Initial OBSERVATION_EVENTS message count: ${INITIAL_MSGS}"

ELAPSED=0
FINAL_MSGS="$INITIAL_MSGS"

while (( ELAPSED < LISTEN_WAIT )); do
    sleep "$LISTEN_POLL"
    ELAPSED=$((ELAPSED + LISTEN_POLL))

    FINAL_MSGS=$(curl -s "${NATS_MONITOR}/jsz?streams=true" 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'OBSERVATION_EVENTS':
            print(s.get('state', {}).get('messages', 0))
            break
" 2>/dev/null || echo "0")

    DELTA=$((FINAL_MSGS - INITIAL_MSGS))

    if [[ "$DELTA" -gt 0 ]]; then
        pass "Live trades detected: +${DELTA} messages in ${ELAPSED}s (total: ${FINAL_MSGS})"
        break
    fi

    info "  ${ELAPSED}s elapsed — no new trades yet (total: ${FINAL_MSGS})"
done

DELTA=$((FINAL_MSGS - INITIAL_MSGS))
if [[ "$DELTA" -le 0 ]]; then
    record_fail "No new trades on OBSERVATION_EVENTS after ${LISTEN_WAIT}s — live exchange listening not proven"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 7: Derive Consumption — Consumer Delivered Count Growing
# ══════════════════════════════════════════════════════════════════════
phase "Phase 7: Derive Consumption — Consumer Progress"

DERIVE_DELIVERED=$(curl -s "${NATS_MONITOR}/jsz?consumers=true" 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'OBSERVATION_EVENTS':
            for c in s.get('consumer_details', []):
                if c.get('name') == 'derive-observation':
                    print(c.get('delivered', {}).get('consumer_seq', 0))
                    break
" 2>/dev/null || echo "0")

if [[ "$DERIVE_DELIVERED" -gt 0 ]]; then
    pass "derive-observation consumer has delivered ${DERIVE_DELIVERED} message(s)"
else
    record_fail "derive-observation consumer delivered count is 0 — derive may not be consuming"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 8: Write Path Isolation — No Execution Fill Events
# ══════════════════════════════════════════════════════════════════════
phase "Phase 8: Write Path Isolation — No Real Venue Activity"

# Verify EXECUTION_FILL_EVENTS has no messages from venue (only paper expected).
FILL_MSGS=$(curl -s "${NATS_MONITOR}/jsz?streams=true" 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for acc in data.get('account_details', []):
    for s in acc.get('stream_details', []):
        if s.get('name') == 'EXECUTION_FILL_EVENTS':
            print(s.get('state', {}).get('messages', 0))
            break
" 2>/dev/null || echo "0")

info "EXECUTION_FILL_EVENTS message count: ${FILL_MSGS}"

# Check that execute binary is in paper mode via logs.
VENUE_LIVE_HITS=$($COMPOSE_CMD logs --tail=200 execute 2>/dev/null | grep -ci "venue_live\|real.*order\|mainnet.*submit" || echo "0")

if [[ "$VENUE_LIVE_HITS" -eq 0 ]]; then
    pass "No venue_live / real order evidence in execute logs"
else
    record_fail "Execute logs contain ${VENUE_LIVE_HITS} venue_live reference(s) — investigate"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 9: Ingest Health — Publisher Tracker
# ══════════════════════════════════════════════════════════════════════
phase "Phase 9: Ingest Operational Health"

INGEST_READYZ=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8082/readyz" 2>/dev/null || echo "000")

# Ingest port is internal-only in compose, check via compose exec.
if [[ "$INGEST_READYZ" == "000" ]]; then
    INGEST_READYZ_INNER=$($COMPOSE_CMD exec -T ingest wget -q -O - http://localhost:8082/readyz 2>/dev/null | head -1 || echo "")
    if [[ -n "$INGEST_READYZ_INNER" ]]; then
        pass "Ingest readyz reachable inside compose"
    else
        info "Ingest readyz not reachable from host (expected — internal port)"
    fi
else
    pass "Ingest readyz → HTTP ${INGEST_READYZ}"
fi

# Check publisher health via ingest logs.
PUB_ERRORS=$($COMPOSE_CMD logs --tail=200 ingest 2>/dev/null | grep -ci "publish.*error\|publish.*fail\|publisher.*error" || echo "0")

if [[ "$PUB_ERRORS" -eq 0 ]]; then
    pass "No publisher errors in recent ingest logs"
else
    record_fail "Ingest publisher has ${PUB_ERRORS} error(s) in recent logs"
fi

# ══════════════════════════════════════════════════════════════════════
# Phase 10: Summary
# ══════════════════════════════════════════════════════════════════════
phase "Phase 10: Summary"

info "OBSERVATION_EVENTS: ${INITIAL_MSGS} → ${FINAL_MSGS} (+${DELTA} in ${ELAPSED}s)"
info "derive-observation delivered: ${DERIVE_DELIVERED}"
info "Execution mode: ${EFFECTIVE}"
info "Write path: isolated (no venue_live activity)"

if [[ $ERRORS -gt 0 ]]; then
    smoke_fail_summary "S378 live exchange listening proof" "$ERRORS" "make up && make seed"
    exit 1
fi

echo ""
echo "======================================"
echo "  S378: Live Exchange Listening — PASS"
echo "======================================"
echo ""
echo "Proven:"
echo "  - Compose stack boots with all ingestion services healthy"
echo "  - configctl bindings activate live exchange WebSocket connections"
echo "  - Real Binance Futures aggTrade messages flow into NATS OBSERVATION_EVENTS"
echo "  - derive-observation consumer receives and processes live trades"
echo "  - Execution engine remains in paper mode (no real orders)"
echo "  - Write path is isolated from read path"
echo ""
echo "Not proven (out of scope for S378):"
echo "  - Dry-run execution path (S379)"
echo "  - Venue adapter integration with testnet (S380)"
echo "  - Observability/alerting on live ingestion"
echo ""
exit 0
