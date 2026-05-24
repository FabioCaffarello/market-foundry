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

require_commands bash curl docker git go make python3 cargo buf

pass "Required commands present: bash curl docker git go make python3 cargo buf"
info "$(go version)"
info "$(cargo --version)"
info "$(python3 --version 2>&1)"
info "$(docker --version)"
info "$(docker compose version)"
info "$(git --version)"
info "buf $(buf --version 2>&1)"

# buf version gate — proto/buf.yaml and proto/buf.gen.yaml declare
# `version: v2`, which requires buf >= 1.32.0. Foundry pins a more
# conservative minimum of 1.50.0 (well past v2 stabilization).
# See docs/DEVELOPMENT.md → "Prerequisites" → "buf" for context.
BUF_MIN_VERSION="1.50.0"
BUF_CURRENT_VERSION="$(buf --version 2>&1)"
BUF_LOWER="$(printf '%s\n%s\n' "$BUF_MIN_VERSION" "$BUF_CURRENT_VERSION" | sort -V | head -1)"
if [[ "$BUF_LOWER" != "$BUF_MIN_VERSION" ]]; then
    die "buf version $BUF_CURRENT_VERSION is below minimum $BUF_MIN_VERSION (required for buf.yaml/buf.gen.yaml schema v2). Upgrade: 'brew upgrade bufbuild/buf/buf' or download from https://github.com/bufbuild/buf/releases."
fi
pass "buf version $BUF_CURRENT_VERSION >= $BUF_MIN_VERSION"

# protoc-gen-go version gate — pinned at v1.36.8 to match the
# google.golang.org/protobuf runtime declared in internal/shared/go.mod.
# Plugin-to-runtime version pairing eliminates wire-format compatibility
# bugs. See docs/DEVELOPMENT.md → "External tooling" → "protoc-gen-go".
PROTOC_GEN_GO_REQUIRED_VERSION="v1.36.8"
PROTOC_GEN_GO_BIN="$(go env GOPATH)/bin/protoc-gen-go"
if [[ ! -x "${PROTOC_GEN_GO_BIN}" ]]; then
    # Fall back to PATH lookup before failing — contributors may have
    # installed via a non-default GOPATH.
    if ! command -v protoc-gen-go >/dev/null 2>&1; then
        die "protoc-gen-go not found. Install: 'go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_REQUIRED_VERSION}'. Required by 'make proto-gen' (ADR-0018 acceptance criterion 4)."
    fi
    PROTOC_GEN_GO_BIN="protoc-gen-go"
fi
PROTOC_GEN_GO_CURRENT_VERSION="$("${PROTOC_GEN_GO_BIN}" --version 2>&1 | awk '{print $NF}')"
if [[ "${PROTOC_GEN_GO_CURRENT_VERSION}" != "${PROTOC_GEN_GO_REQUIRED_VERSION}" ]]; then
    die "protoc-gen-go version ${PROTOC_GEN_GO_CURRENT_VERSION} != required ${PROTOC_GEN_GO_REQUIRED_VERSION}. Reinstall: 'go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_REQUIRED_VERSION}'. The pinned version matches google.golang.org/protobuf in internal/shared/go.mod; mismatches cause subtle wire-format bugs."
fi
pass "protoc-gen-go version ${PROTOC_GEN_GO_CURRENT_VERSION} (pinned)"

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
    "CLAUDE.md"
    "docs/README.md"
    "docs/ARCHITECTURE.md"
    "docs/RUNTIME.md"
    "docs/HTTP-API.md"
    "docs/DEVELOPMENT.md"
    "docs/RESUMPTION.md"
    "docs/CONTRIBUTING.md"
    "docs/GLOSSARY.md"
    "docs/domain/README.md"
    "docs/operations/README.md"
    "docs/decisions/README.md"
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

Canonical lifecycle docs:
  - docs/DEVELOPMENT.md
  - docs/RESUMPTION.md
  - docs/CONTRIBUTING.md
EOF
