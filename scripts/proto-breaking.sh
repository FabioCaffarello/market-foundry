#!/usr/bin/env bash
# proto-breaking.sh -- run `buf breaking` against main, treating the
# special case of "baseline proto/ is empty" as PASS.
#
# Context: on the very first introduction of proto/ in Onda H-3.a
# (Fase Wire), main does not yet contain any .proto files; buf
# reports "Module had no .proto files" with a non-zero exit. That is
# not a breaking change; there is simply nothing to compare against.
# After H-3.a merges, future runs always have a populated main
# proto/ and this special case stops triggering.
#
# Any other non-zero exit from buf (real breaking-change violation,
# configuration error, repository error) propagates unchanged.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${PROJECT_ROOT}/proto"

if out=$(buf breaking --against '../.git#branch=main,subdir=proto' 2>&1); then
    [[ -n "${out}" ]] && printf '%s\n' "${out}"
    echo "proto-breaking: PASS"
    exit 0
fi

rc=$?

if echo "${out}" | grep -q "had no .proto files"; then
    echo "proto-breaking: baseline proto/ on main is empty (initial introduction in H-3.a) — nothing to compare against. PASS."
    exit 0
fi

printf '%s\n' "${out}"
exit "${rc}"
