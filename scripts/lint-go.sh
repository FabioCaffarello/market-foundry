#!/usr/bin/env bash
#
# scripts/lint-go.sh
#
# Runs golangci-lint across all workspace modules listed in go.work.
# Phase 2 environment hardening — adopted in P2.2.1.
#
# Iterates per-module because `golangci-lint run ./...` from the repo
# root fails on multi-module workspaces ("directory prefix . does not
# contain modules listed in go.work or their selected dependencies").
#
# Exit code:
#   0 — all modules lint clean
#   1 — one or more modules reported issues
#   2 — golangci-lint missing or workspace inventory empty

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "ERROR: golangci-lint not installed. See docs/DEVELOPMENT.md." >&2
    exit 2
fi

VERSION="$(golangci-lint --version 2>&1 | head -1)"
if [[ "${VERSION}" != *" version "*"2."* ]]; then
    echo "WARNING: ${VERSION}" >&2
    echo "         .golangci.yml uses v2 schema; v1.x may behave unexpectedly." >&2
fi

MODULES="$(go work edit -json | python3 -c 'import json, sys
data = json.load(sys.stdin)
for mod in data["Use"]:
    print(mod["DiskPath"])')"

if [[ -z "${MODULES}" ]]; then
    echo "ERROR: no modules listed in go.work" >&2
    exit 2
fi

TOTAL_MODULES=0
FAILED_MODULES=0
FAILED_LIST=""

for mod in ${MODULES}; do
    TOTAL_MODULES=$((TOTAL_MODULES + 1))
    printf '::: linting %s\n' "${mod}"
    if (cd "${mod}" && golangci-lint run --timeout=5m ./...); then
        :
    else
        FAILED_MODULES=$((FAILED_MODULES + 1))
        FAILED_LIST="${FAILED_LIST} ${mod}"
    fi
done

echo ""
echo "==============================================="
printf 'lint-go summary: %d modules linted\n' "${TOTAL_MODULES}"
if [[ "${FAILED_MODULES}" -eq 0 ]]; then
    echo "All modules clean."
    exit 0
fi

printf '%d module(s) with issues:\n' "${FAILED_MODULES}"
for m in ${FAILED_LIST}; do
    printf '  - %s\n' "${m}"
done
exit 1
