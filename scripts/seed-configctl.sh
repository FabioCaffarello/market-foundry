#!/usr/bin/env bash
# seed-configctl.sh — Seed configctl with ingestion bindings via the full lifecycle.
#
# Creates a config document with the specified bindings, then runs the complete
# lifecycle: draft → validate → compile → activate.
#
# Usage:
#   ./scripts/seed-configctl.sh                          # default: btcusdt only
#   ./scripts/seed-configctl.sh --multi-symbol            # btcusdt + ethusdt
#   SYMBOLS="btcusdt,ethusdt,solusdt" ./scripts/seed-configctl.sh  # custom list
#
# Prerequisites:
#   make up   (nats + configctl + gateway must be running)

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SOURCE="${SOURCE:-binancef}"
CORRELATION_ID="seed-$(date +%s)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

# ---------- Determine symbols ----------
if [[ "${1:-}" == "--multi-symbol" ]]; then
    SYMBOLS="${SYMBOLS:-btcusdt,ethusdt}"
elif [[ -z "${SYMBOLS:-}" ]]; then
    SYMBOLS="btcusdt"
fi

IFS=',' read -ra SYMBOL_LIST <<< "$SYMBOLS"
info "Seeding configctl with source=${SOURCE} symbols=[${SYMBOLS}]"

# ---------- Build bindings JSON ----------
BINDINGS_JSON="["
for i in "${!SYMBOL_LIST[@]}"; do
    sym="${SYMBOL_LIST[$i]}"
    [[ $i -gt 0 ]] && BINDINGS_JSON+=","
    BINDINGS_JSON+="{\"name\":\"${sym}-trades\",\"topic\":\"${SOURCE}.${sym}\"}"
done
BINDINGS_JSON+="]"

# ---------- Build config content ----------
CONFIG_CONTENT=$(cat <<ENDJSON
{
  "metadata": {
    "name": "market-data-${SOURCE}",
    "description": "Ingestion bindings for ${SOURCE}: ${SYMBOLS}"
  },
  "bindings": ${BINDINGS_JSON},
  "fields": [
    {"name": "price", "type": "string", "required": true},
    {"name": "quantity", "type": "string", "required": true}
  ],
  "rules": [
    {"name": "price_required", "field": "price", "operator": "required", "severity": "error"},
    {"name": "quantity_required", "field": "quantity", "operator": "required", "severity": "error"}
  ]
}
ENDJSON
)

# Escape the content for JSON embedding.
ESCAPED_CONTENT=$(echo "$CONFIG_CONTENT" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read().strip()))")

# ---------- Step 1: Create draft ----------
info "Step 1: Creating config draft..."
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "${BASE_URL}/configctl/configs" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}" \
    -d "{\"name\":\"market-data-${SOURCE}\",\"format\":\"json\",\"content\":${ESCAPED_CONTENT}}")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" != "201" && "$HTTP_CODE" != "200" ]]; then
    echo "$BODY"
    fail "Create draft failed with HTTP ${HTTP_CODE}"
fi

VERSION_ID=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); cfg=d['config']; print(cfg.get('version_id') or cfg.get('id') or '')" 2>/dev/null)
if [[ -z "$VERSION_ID" ]]; then
    echo "$BODY"
    fail "Could not extract version_id from response"
fi
pass "Draft created: version_id=${VERSION_ID}"

# ---------- Step 2: Validate ----------
info "Step 2: Validating config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/validate" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config validated" || fail "Validate failed: HTTP ${HTTP_CODE}"

# ---------- Step 3: Compile ----------
info "Step 3: Compiling config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/compile" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}" \
    -d "{}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config compiled" || fail "Compile failed: HTTP ${HTTP_CODE}"

# ---------- Step 4: Activate ----------
info "Step 4: Activating config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/activate" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}" \
    -d "{\"scope_kind\":\"global\",\"scope_key\":\"default\"}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config activated" || fail "Activate failed: HTTP ${HTTP_CODE}"

# ---------- Step 5: Confirm ----------
info "Step 5: Confirming active config..."
RESPONSE=$(curl -s "${BASE_URL}/configctl/configs/active?scope_kind=global&scope_key=default")

echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
config = data.get('config', {})
bindings = config.get('bindings', [])
print(f'  Active config: {config.get(\"name\", \"unknown\")}')
print(f'  Version: {config.get(\"version\", \"?\")}')
print(f'  Bindings: {len(bindings)}')
for b in bindings:
    print(f'    - {b.get(\"name\", \"?\")} → {b.get(\"topic\", \"?\")}')
" 2>/dev/null && pass "Active config confirmed" || fail "Could not confirm active config"

echo ""
echo "======================================"
echo "  Configctl Seed: COMPLETE"
echo "======================================"
echo ""
echo "Bindings activated:"
for sym in "${SYMBOL_LIST[@]}"; do
    echo "  ${SOURCE}.${sym}"
done
echo ""
echo "Services will discover bindings via:"
echo "  - ingest: binding-watcher queries + event subscription"
echo "  - derive: binding-watcher queries + event subscription"
echo ""
