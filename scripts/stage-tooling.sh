#!/usr/bin/env bash
#
# stage-tooling.sh -- lightweight helpers for governed stage execution.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat <<'EOF'
Usage:
  ./scripts/stage-tooling.sh help
  ./scripts/stage-tooling.sh scaffold [--stage-id C15] [--slug stage-tooling] [--title "Stage Tooling"]
  ./scripts/stage-tooling.sh check [--stage-id C15] [--slug stage-tooling] [--report docs/stages/stage-c15-...-report.md] [--require path1,path2]

Environment fallbacks:
  STAGE_ID       stage identifier (example: C15, S312)
  STAGE_SLUG     kebab-case report slug
  STAGE_TITLE    report title used by scaffold
  STAGE_REPORT   explicit report path for check
  STAGE_REQUIRE  comma-separated required artifact paths for check

This helper is intentionally lightweight:
  - scaffold creates only a stage report template
  - check validates naming, index presence, minimum section completeness,
    local links, and optional required artifacts
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

info() {
    printf '%s\n' "$*"
}

stage_id="${STAGE_ID:-}"
stage_slug="${STAGE_SLUG:-}"
stage_title="${STAGE_TITLE:-}"
stage_report="${STAGE_REPORT:-}"
stage_require="${STAGE_REQUIRE:-}"

command="${1:-help}"
if [[ $# -gt 0 ]]; then
    shift
fi

while [[ $# -gt 0 ]]; do
    case "$1" in
        --stage-id)
            [[ $# -ge 2 ]] || die "--stage-id requires a value"
            stage_id="$2"
            shift 2
            ;;
        --slug)
            [[ $# -ge 2 ]] || die "--slug requires a value"
            stage_slug="$2"
            shift 2
            ;;
        --title)
            [[ $# -ge 2 ]] || die "--title requires a value"
            stage_title="$2"
            shift 2
            ;;
        --report)
            [[ $# -ge 2 ]] || die "--report requires a value"
            stage_report="$2"
            shift 2
            ;;
        --require)
            [[ $# -ge 2 ]] || die "--require requires a value"
            stage_require="$2"
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            die "unknown argument: $1"
            ;;
    esac
done

normalize_stage_id() {
    local raw="$1"
    raw="${raw// /}"
    printf '%s' "${raw}" | tr '[:upper:]' '[:lower:]'
}

resolve_report_path() {
    local id_lower="$1"
    local slug="$2"

    if [[ -n "${stage_report}" ]]; then
        printf '%s' "${stage_report}"
        return 0
    fi

    if [[ -n "${slug}" ]]; then
        printf 'docs/stages/stage-%s-%s-report.md' "${id_lower}" "${slug}"
        return 0
    fi

    local matches=()
    while IFS= read -r match; do
        matches+=("${match}")
    done < <(find docs/stages -maxdepth 1 -type f -name "stage-${id_lower}-*-report.md" | sort)

    if [[ "${#matches[@]}" -eq 1 ]]; then
        printf '%s' "${matches[0]}"
        return 0
    fi

    if [[ "${#matches[@]}" -eq 0 ]]; then
        die "could not infer report path for stage ${stage_id}; pass --slug or --report"
    fi

    die "multiple reports matched stage ${stage_id}; pass --report to disambiguate"
}

run_scaffold() {
    [[ -n "${stage_id}" ]] || die "scaffold requires STAGE_ID or --stage-id"
    [[ -n "${stage_slug}" ]] || die "scaffold requires STAGE_SLUG or --slug"
    [[ -n "${stage_title}" ]] || die "scaffold requires STAGE_TITLE or --title"

    local id_lower
    id_lower="$(normalize_stage_id "${stage_id}")"
    local report_path
    report_path="$(resolve_report_path "${id_lower}" "${stage_slug}")"

    [[ ! -e "${PROJECT_ROOT}/${report_path}" ]] || die "${report_path} already exists"

    mkdir -p "${PROJECT_ROOT}/docs/stages"
    cat > "${PROJECT_ROOT}/${report_path}" <<EOF
# Stage ${stage_id^^} Report: ${stage_title}

## Summary

Stage ${stage_id^^} strengthens the repository support surface for disciplined
stage execution without changing domain semantics.

## Objective

State the single operational/governance objective of this stage in one sentence.

## Scope Boundaries

### In scope

- stage-support tooling, report ergonomics, or validation helpers
- supporting operations/docs updates required to keep the workflow coherent

### Out of scope

- functional domain behavior
- heavy stage-management frameworks
- broad policy rewrites without direct execution value

### Not changed

- stage semantics defined by the Opus
- domain runtime ownership and architecture layers

## Changes Applied

- list tooling or docs changes actually delivered

## Artifacts Added Or Updated

| Artifact | Purpose |
|---|---|
| \`${report_path}\` | Stage completion record |

## Validation

- \`make repo-consistency-check\`
- \`make stage-check STAGE_ID=${stage_id^^} STAGE_SLUG=${stage_slug}\`

## Limits And Deferred Follow-Ups

- record what stayed manual, intentionally lightweight, or deferred

## Preparation For Next Stage

- record the next useful tightening step
EOF

    info "created ${report_path}"
    info "next steps:"
    info "  1. add the new report to docs/stages/INDEX.md"
    info "  2. update any canonical operations/architecture doc promoted by the stage"
    info "  3. run make stage-check STAGE_ID=${stage_id^^} STAGE_SLUG=${stage_slug}"
}

run_check() {
    [[ -n "${stage_id}" || -n "${stage_report}" ]] || die "check requires STAGE_ID or STAGE_REPORT"

    local id_lower=""
    if [[ -n "${stage_id}" ]]; then
        id_lower="$(normalize_stage_id "${stage_id}")"
    fi

    local report_path
    report_path="$(resolve_report_path "${id_lower}" "${stage_slug}")"

    cd "${PROJECT_ROOT}"

    python3 - "${report_path}" "${stage_require}" <<'PY'
from pathlib import Path
import re
import sys

report_path = Path(sys.argv[1])
required_raw = sys.argv[2].strip()
required_paths = [item.strip() for item in required_raw.split(",") if item.strip()]

errors = []

if not report_path.is_file():
    errors.append(f"missing report: {report_path.as_posix()}")
else:
    text = report_path.read_text()
    lines = text.splitlines()

    if not re.fullmatch(r"stage-[a-z0-9-]+-report\.md", report_path.name):
        errors.append(f"invalid report filename: {report_path.name}")

    if not lines or not lines[0].startswith("# "):
        errors.append("report missing top-level title")

    h2s = re.findall(r"^## ", text, re.M)
    if len(h2s) < 4:
        errors.append(f"report expected at least 4 level-2 sections, found {len(h2s)}")

    prefix = r"^## (?:\d+\.\s+)?"
    section_patterns = {
        "summary": prefix + r"(Summary|Executive Summary)\b",
        "changes": prefix + r"(Changes Applied|Delivered Changes|Improvements Applied|Main Changes|Tooling Changes)\b",
        "validation": prefix + r"Validation\b",
        "next-stage": prefix + r"(Preparation For Next Stage|Preparation For C\d+|Preparation For S\d+|Next Stage|Recommended Preparation|Outcome|Final Operating Model)\b",
    }
    for label, pattern in section_patterns.items():
        if not re.search(pattern, text, re.M):
            errors.append(f"report missing expected section family: {label}")

    scope_signals = [
        r"^## (?:\d+\.\s+)?Scope Boundaries\b",
        r"^### (?:\d+\.\s+)?In scope\b",
        r"^### (?:\d+\.\s+)?Out of scope\b",
        r"^### (?:\d+\.\s+)?Not changed\b",
        r"\bout of scope\b",
        r"\bnot changed\b",
    ]
    if not any(re.search(pattern, text, re.M | re.I) for pattern in scope_signals):
        errors.append("report missing explicit scope-boundary signals")

    index_text = Path("docs/stages/INDEX.md").read_text()
    if report_path.name not in index_text:
        errors.append("report not indexed in docs/stages/INDEX.md")

    link_pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
    for target in link_pattern.findall(text):
        if target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        target = target.split("#", 1)[0]
        if not target:
            continue
        resolved = (report_path.parent / target).resolve()
        if not resolved.exists():
            errors.append(f"broken local link in report: {target}")

for rel in required_paths:
    if not Path(rel).exists():
        errors.append(f"missing required artifact: {rel}")

if errors:
    print("stage-check: FAIL")
    for item in errors:
        print(f"  - {item}")
    sys.exit(1)

print("stage-check: PASS")
print(f"  - report: {report_path.as_posix()}")
if required_paths:
    print(f"  - required artifacts present: {len(required_paths)}")
else:
    print("  - no extra required artifacts declared")
PY
}

case "${command}" in
    --help|-h)
        usage
        ;;
    help)
        usage
        ;;
    scaffold)
        run_scaffold
        ;;
    check)
        run_check
        ;;
    *)
        die "unknown command: ${command}"
        ;;
esac
