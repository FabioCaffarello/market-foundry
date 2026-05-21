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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/seed-configctl.sh [--multi-symbol] [--merge] [--help]

Seeds configctl through the full lifecycle: draft -> validate -> compile -> activate.
Canonical public entrypoints: `make seed`, `make seed-multi`, `make seed-unified`

Options:
  --multi-symbol  Seed the default multi-symbol configuration (btcusdt,ethusdt).
  --merge         S400: Merge bindings from multiple sources into a single config.
                  Uses SOURCES env var (default: binancef,binances).
  --help          Show this help text.

Environment:
  BASE_URL        Gateway base URL. Default: http://127.0.0.1:8080
  SOURCE          Source name used for bindings. Default: binancef
  SOURCES         Comma-separated source list for --merge mode. Default: binancef,binances
  SYMBOLS         Comma-separated symbol list. Overrides the default symbol set.
EOF
}

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SOURCE="${SOURCE:-binancef}"
CORRELATION_ID="seed-$(date +%s)"

MULTI_SYMBOL=false
MERGE_MODE=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --multi-symbol)
            MULTI_SYMBOL=true
            ;;
        --merge)
            MERGE_MODE=true
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

# ---------- Determine symbols ----------
if $MULTI_SYMBOL; then
    SYMBOLS="${SYMBOLS:-btcusdt,ethusdt}"
elif [[ -z "${SYMBOLS:-}" ]]; then
    SYMBOLS="btcusdt"
fi

IFS=',' read -ra RAW_SYMBOL_LIST <<< "$SYMBOLS"
SYMBOL_LIST=()
for sym in "${RAW_SYMBOL_LIST[@]}"; do
    sym="${sym//[[:space:]]/}"
    [[ -n "$sym" ]] || continue
    SYMBOL_LIST+=("$sym")
done

if [[ ${#SYMBOL_LIST[@]} -eq 0 ]]; then
    die "no symbols resolved from SYMBOLS=${SYMBOLS}"
fi

SYMBOLS="$(IFS=,; echo "${SYMBOL_LIST[*]}")"

# ---------- S400: Resolve source list for merge mode ----------
if $MERGE_MODE; then
    SOURCES="${SOURCES:-binancef,binances}"
    IFS=',' read -ra SOURCE_LIST <<< "$SOURCES"
    CONFIG_NAME="market-data-unified"
    CONFIG_DESC="Merged ingestion bindings for ${SOURCES}: ${SYMBOLS}"
    info "Seeding configctl (merge mode) with sources=[${SOURCES}] symbols=[${SYMBOLS}]"
else
    SOURCE_LIST=("$SOURCE")
    CONFIG_NAME="market-data-${SOURCE}"
    CONFIG_DESC="Ingestion bindings for ${SOURCE}: ${SYMBOLS}"
    info "Seeding configctl with source=${SOURCE} symbols=[${SYMBOLS}]"
fi

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/readyz" 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" != "200" ]]; then
    die "gateway not ready at ${BASE_URL}/readyz (HTTP ${HTTP_CODE})"
fi

# ---------- Build bindings JSON ----------
# S400: When --merge, generates bindings for each source × symbol combination.
BINDINGS_JSON="["
FIRST_BINDING=true
for src in "${SOURCE_LIST[@]}"; do
    src="${src//[[:space:]]/}"
    [[ -n "$src" ]] || continue
    for sym in "${SYMBOL_LIST[@]}"; do
        $FIRST_BINDING || BINDINGS_JSON+=","
        FIRST_BINDING=false
        BINDINGS_JSON+="{\"name\":\"${src}-${sym}-trades\",\"topic\":\"${src}.${sym}\"}"
    done
done
BINDINGS_JSON+="]"

# ---------- Build config content ----------
CONFIG_CONTENT=$(cat <<ENDJSON
{
  "metadata": {
    "name": "${CONFIG_NAME}",
    "description": "${CONFIG_DESC}"
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
    -d "{\"name\":\"${CONFIG_NAME}\",\"format\":\"json\",\"content\":${ESCAPED_CONTENT}}")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" != "201" && "$HTTP_CODE" != "200" ]]; then
    echo "$BODY"
    die "Create draft failed with HTTP ${HTTP_CODE}"
fi

VERSION_ID=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); cfg=d['config']; print(cfg.get('version_id') or cfg.get('id') or '')" 2>/dev/null)
if [[ -z "$VERSION_ID" ]]; then
    echo "$BODY"
    die "Could not extract version_id from response"
fi
pass "Draft created: version_id=${VERSION_ID}"

# ---------- Step 2: Validate ----------
info "Step 2: Validating config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/validate" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config validated" || die "Validate failed: HTTP ${HTTP_CODE}"

# ---------- Step 3: Compile ----------
info "Step 3: Compiling config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/compile" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}" \
    -d "{}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config compiled" || die "Compile failed: HTTP ${HTTP_CODE}"

# ---------- Step 4: Activate ----------
info "Step 4: Activating config..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/configctl/config-versions/${VERSION_ID}/activate" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-Correlation-ID: ${CORRELATION_ID}" \
    -d "{\"scope_kind\":\"global\",\"scope_key\":\"default\"}")

[[ "$HTTP_CODE" == "200" ]] && pass "Config activated" || die "Activate failed: HTTP ${HTTP_CODE}"

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
" 2>/dev/null && pass "Active config confirmed" || die "Could not confirm active config"

echo ""
echo "======================================"
echo "  Configctl Seed: COMPLETE"
echo "======================================"
echo ""
echo "Bindings activated:"
for src in "${SOURCE_LIST[@]}"; do
    src="${src//[[:space:]]/}"
    [[ -n "$src" ]] || continue
    for sym in "${SYMBOL_LIST[@]}"; do
        echo "  ${src}.${sym}"
    done
done
echo ""
echo "Services will discover bindings via:"
echo "  - ingest: binding-watcher queries + event subscription"
echo "  - derive: binding-watcher queries + event subscription"
echo ""
