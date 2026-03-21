#!/usr/bin/env bash
# diag-check.sh — Lightweight diagnostic snapshot of a running stack.
#
# Queries /statusz and /diagz on all runtimes and prints a concise
# operational summary without starting or seeding anything.
#
# Usage:
#   ./scripts/diag-check.sh                # default: docker compose exec
#   ./scripts/diag-check.sh --local        # direct HTTP (services on host)
#
# Prerequisites:
#   A running compose stack (make up) or locally running services.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/diag-check.sh [--local] [--help]

Collects a lightweight diagnostic snapshot from the running stack.

Options:
  --local   Query services directly on the host instead of through docker compose exec.
  --help    Show this help text.
EOF
}

LOCAL=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --local)
            LOCAL=true
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
if ! $LOCAL; then
    require_commands docker
fi

fetch() {
    local svc="$1"
    local port="$2"
    local endpoint="$3"
    if $LOCAL; then
        curl -s "http://127.0.0.1:${port}${endpoint}" 2>/dev/null || echo ""
    else
        docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" exec -T "${svc}" wget -q -O - "http://127.0.0.1:${port}${endpoint}" 2>/dev/null || echo ""
    fi
}

# ---------- Readiness ----------
phase "Readiness Probes"

RUNTIME_PORTS=("configctl:8080" "ingest:8082" "derive:8083" "store:8081" "execute:8084" "writer:8085")

for svc_port in "${RUNTIME_PORTS[@]}"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    result=$(fetch "$svc" "$port" "/readyz")
    if [[ -z "$result" ]]; then
        record_fail "${svc} /readyz → unreachable"
        continue
    fi
    status=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','error'))" 2>/dev/null || echo "error")
    if [[ "$status" == "ready" ]]; then
        pass "${svc} /readyz → ready"
    else
        record_fail "${svc} /readyz → ${status}"
    fi
done

# ---------- Operational Phase ----------
phase "Operational Phase Summary"

for svc_port in "${RUNTIME_PORTS[@]}"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    statusz=$(fetch "$svc" "$port" "/statusz")
    if [[ -z "$statusz" ]]; then
        record_fail "${svc}: /statusz unreachable"
        continue
    fi
    echo "$statusz" | python3 -c "
import sys,json
d=json.load(sys.stdin)
svc='${svc}'
phase=d.get('phase','unknown')
uptime=d.get('uptime','?')
trackers=d.get('trackers',[])
active_count=sum(1 for t in trackers if t.get('event_count',0)>0)
total=len(trackers)
total_events=sum(t.get('event_count',0) for t in trackers)
total_errors=sum(t.get('error_count',0) for t in trackers)
idle_warn=sum(1 for t in trackers if t.get('idle_warning'))
line=f'{svc:>10s}  phase={phase:<10s} uptime={uptime:<12s} trackers={active_count}/{total} events={total_events} errors={total_errors}'
if idle_warn:
    line += f' idle_warnings={idle_warn}'
print(line)
" 2>/dev/null || warn "${svc}: /statusz parse error"
done

# ---------- Diagnostic Details ----------
phase "Diagnostic Details (/diagz)"

for svc_port in "${RUNTIME_PORTS[@]}"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    diagz=$(fetch "$svc" "$port" "/diagz")
    if [[ -z "$diagz" ]]; then
        record_fail "${svc}: /diagz unreachable"
        continue
    fi
    echo "$diagz" | python3 -c "
import sys,json
d=json.load(sys.stdin)
svc='${svc}'
checks=d.get('readiness_checks',[])
passed=sum(1 for c in checks if c.get('status')=='pass')
total=len(checks)
goroutines=d.get('num_goroutines','?')
go_ver=d.get('go_version','?')
phase=d.get('phase','?')
print(f'{svc:>10s}  readiness={passed}/{total} goroutines={goroutines} go={go_ver} phase={phase}')
trackers=d.get('trackers',[])
for t in trackers:
    name=t['name']
    ec=t.get('event_count',0)
    er=t.get('error_count',0)
    status=t.get('status','')
    idle=t.get('idle_seconds','')
    counters=t.get('counters',{})
    parts=[f'events={ec}', f'errors={er}']
    if status:
        parts.append(f'status={status}')
    if idle != '':
        parts.append(f'idle={idle}s')
    for k,v in sorted(counters.items()):
        parts.append(f'{k}={v}')
    print(f'             {name}: {\" \".join(parts)}')
" 2>/dev/null || warn "${svc}: /diagz parse error"
done

# ---------- Error Log Scan ----------
phase "Error Log Scan"

if ! $LOCAL; then
    ERROR_COUNT=$(docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" logs --no-log-prefix 2>/dev/null | grep -c '"level":"error"' || true)
    ERROR_COUNT="${ERROR_COUNT:-0}"
    if [[ "$ERROR_COUNT" -gt 0 ]]; then
        warn "Found ${ERROR_COUNT} error-level log entries"
        docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" logs --no-log-prefix 2>/dev/null | grep '"level":"error"' | tail -5
    else
        pass "No error-level log entries"
    fi
else
    warn "Error log scan not available in --local mode"
fi

# ---------- Summary ----------
phase "Summary"

if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}${BOLD}All diagnostics healthy.${NC}"
else
    echo -e "${RED}${BOLD}${ERRORS} issue(s) detected.${NC}"
fi

exit $ERRORS
