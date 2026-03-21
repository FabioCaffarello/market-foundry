#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/utils/list-modules.sh [--help]

Prints module directories from go.work, one per line.
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

if [[ ! -f go.work ]]; then
  echo "go.work not found in $(pwd)" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go command not found" >&2
  exit 1
fi

# Prints module directories from go.work, one per line.
if command -v jq >/dev/null 2>&1; then
  go work edit -json | jq -r '.Use[].DiskPath'
else
  awk '
    BEGIN { in_use = 0 }
    /^[[:space:]]*use[[:space:]]*\(/ { in_use = 1; next }
    in_use && /^[[:space:]]*\)/ { in_use = 0; next }
    in_use {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
      if ($0 != "") print $0
      next
    }
    /^[[:space:]]*use[[:space:]]+\.[^[:space:]]*/ {
      line = $0
      sub(/^[[:space:]]*use[[:space:]]+/, "", line)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
      if (line != "") print line
    }
  ' go.work
fi | sort -u
