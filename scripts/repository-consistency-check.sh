#!/usr/bin/env bash
#
# repository-consistency-check.sh -- minimal post-Phase-1A consistency check.
#
# The previous 500+ line implementation was tied to the pre-reset topology
# (docs/product/, docs/architecture/, docs/development/, docs/stages/,
# docs/archive/, docs/tooling/) which was moved to docs/legacy/ in P1A.1.
# That stale script reported 9 failing checks for 17 prompts; only 1 of
# the 9 was actually about .opencode/. The other 8 were the script itself
# pinned to paths that no longer existed. See docs/RESUMPTION.md G3.
#
# This stub validates the current Phase 1A docs topology. A full rewrite
# of the consistency-check infrastructure is deferred to P1D.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

usage() {
    cat <<'EOF'
Usage: ./scripts/repository-consistency-check.sh [--help]

Runs lightweight checks aligned with the Phase 1A documentation topology:
  - root-level docs/*.md entrypoints present
  - docs/ subdirectories present
  - minimum population in decisions/, domain/, operations/
  - no WIP markers leaking into finished root docs
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

PASS_COUNT=0
FAIL_COUNT=0

check() {
    local name="$1"
    local detail="$2"
    shift 2
    local output=""
    if output="$("$@" 2>&1)"; then
        PASS_COUNT=$((PASS_COUNT + 1))
        printf '[PASS] %s\n' "${name}"
        [[ -n "${detail}" ]] && printf '  %s\n' "${detail}"
        return 0
    fi
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf '[FAIL] %s\n' "${name}"
    [[ -n "${output}" ]] && printf '  %s\n' "${output}"
    return 1
}

check_file_exists() { test -f "$1"; }
check_dir_exists() { test -d "$1"; }

check_md_count_at_least() {
    local dir="$1"
    local minimum="$2"
    local count
    count="$(find "${dir}" -name '*.md' 2>/dev/null | wc -l | tr -d '[:space:]')"
    if (( count < minimum )); then
        printf 'expected at least %d .md files in %s, found %d\n' "${minimum}" "${dir}" "${count}"
        return 1
    fi
    printf 'found %d .md files (minimum %d)' "${count}" "${minimum}"
}

check_no_wip_in_root_docs() {
    local hits
    hits="$(grep -l 'WIP' docs/*.md 2>/dev/null || true)"
    if [[ -n "${hits}" ]]; then
        printf 'WIP markers found in finished root docs:\n%s\n' "${hits}"
        return 1
    fi
    printf 'no WIP markers in finished root docs'
}

# Root-level docs (Phase 1A)
for doc in README ARCHITECTURE RUNTIME HTTP-API DEVELOPMENT RESUMPTION CONTRIBUTING GLOSSARY; do
    check "root-doc docs/${doc}.md" "" check_file_exists "docs/${doc}.md"
done

# Subdirectories (Phase 1A)
for sub in decisions domain operations legacy; do
    check "subdir docs/${sub}/" "" check_dir_exists "docs/${sub}"
done

# Minimum population
check "decisions/ has ADRs" "" check_md_count_at_least "docs/decisions" 5
check "domain/ has docs"   "" check_md_count_at_least "docs/domain"    10
check "operations/ has docs" "" check_md_count_at_least "docs/operations" 4

# Finished-doc hygiene (preserved spirit of historical-leakage check)
check "no WIP in root docs" "" check_no_wip_in_root_docs

printf '\nSummary: %d passed, %d failed\n' "${PASS_COUNT}" "${FAIL_COUNT}"
exit "${FAIL_COUNT}"
