#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/utils/for-each-module.sh <command...>

Runs a command in each Go workspace module resolved from go.work, or only in
MODULE=... when that environment variable is set.
EOF
}

if [[ $# -eq 0 ]]; then
  usage >&2
  exit 1
fi

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ -n "${MODULE:-}" ]]; then
  echo ">>> ${MODULE}: $*"
  (
    cd "${MODULE}"
    "$@"
  )
  exit 0
fi

modules=()
while IFS= read -r module; do
  modules+=("$module")
done < <("$ROOT_DIR/scripts/utils/list-modules.sh")
if [[ ${#modules[@]} -eq 0 ]]; then
  echo "no modules resolved from go.work" >&2
  exit 1
fi

for module in "${modules[@]}"; do
  [[ -z "$module" ]] && continue
  echo ">>> ${module}: $*"
  (
    cd "$module"
    "$@"
  )
done
