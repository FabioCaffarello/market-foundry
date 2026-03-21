#!/usr/bin/env bash
# bootstrap-check.sh -- validate local prerequisites and repository entrypoints.
#
# Intended to answer a simple onboarding question:
# "Is this machine ready to use the canonical market-foundry workflow?"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/bootstrap-check.sh [--help]

Validates the local prerequisites and canonical repository entrypoints for the
official developer workflow.
Canonical public entrypoint: `make bootstrap`

Checks:
  - required host commands
  - Docker daemon and docker compose availability
  - compose configuration renderability
  - presence of canonical repository entrypoints
  - presence of the local env file used by migration/runtime flows
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

cd "${PROJECT_ROOT}"

phase "Host Tooling"

require_commands bash curl docker git go make python3 cargo

pass "Required commands present: bash curl docker git go make python3 cargo"
info "$(go version)"
info "$(cargo --version)"
info "$(python3 --version 2>&1)"
info "$(docker --version)"
info "$(docker compose version)"
info "$(git --version)"

phase "Docker Availability"

if docker info >/dev/null 2>&1; then
    pass "Docker daemon reachable"
else
    die "docker daemon is not reachable"
fi

phase "Repository Entrypoints"

required_paths=(
    "Makefile"
    "README.md"
    "DEVELOPMENT.md"
    "docs/README.md"
    "docs/operations/README.md"
    "docs/operations/documentation-system-hardening.md"
    "docs/operations/documentation-governance-entrypoints-and-taxonomy.md"
    "docs/operations/repository-policy-and-lightweight-enforcement-2.md"
    "docs/operations/repository-invariants-check-matrix-and-enforcement-policy.md"
    "docs/operations/developer-workflow-unification.md"
    "docs/operations/developer-onboarding-and-troubleshooting-guide.md"
    "docs/tooling/README.md"
    "docs/architecture/README.md"
    "docs/stages/INDEX.md"
    "deploy/compose/docker-compose.yaml"
    "deploy/envs/local.env"
    "tools/raccoon-cli/Cargo.toml"
    "go.work"
)

missing=()
for path in "${required_paths[@]}"; do
    [[ -e "${path}" ]] || missing+=("${path}")
done

if [[ ${#missing[@]} -gt 0 ]]; then
    printf 'Missing required repository entrypoints:\n'
    printf '  - %s\n' "${missing[@]}"
    exit 1
fi

pass "Canonical repository entrypoints present"

phase "Compose Configuration"

if docker compose -f "${PROJECT_ROOT}/deploy/compose/docker-compose.yaml" config >/dev/null; then
    pass "docker compose configuration renders cleanly"
else
    die "docker compose configuration is invalid"
fi

phase "Next Steps"

cat <<'EOF'
Official path after bootstrap:
  1. make live                 # fastest single-symbol bring-up
  2. make smoke                # canonical baseline operational proof
  3. make check                # pre-change guard rail
  4. make tdd                  # impact-driven validation guidance
  5. make verify               # post-change validation

Controlled manual path:
  1. make up
  2. make seed   (or make seed-multi)
  3. make smoke  (or the narrowest relevant make smoke*)

Troubleshooting first steps:
  - make diag
  - make ps
  - make logs SERVICE=gateway
EOF
