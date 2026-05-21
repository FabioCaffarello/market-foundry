#!/usr/bin/env bash
#
# repository-consistency-check.sh -- lightweight documentation and repository
# consistency guard rail for the post-O20 joint `.opencode` + `docs/` topology.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat <<'EOF'
Usage: ./scripts/repository-consistency-check.sh [--help]

Runs lightweight checks for:
  - required repository entrypoints
  - primary docs area indexes and owner maps
  - local links in active support docs
  - stage report index alignment
  - bounded active support surfaces
  - orphaned active docs and competing owner entries
  - navigation consistency across docs and `.opencode`
  - historical leakage into active primary surfaces
  - `.opencode` consistency integration
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

cd "${PROJECT_ROOT}"

PASS_COUNT=0
FAIL_COUNT=0

print_result() {
    local status="$1"
    local name="$2"
    local output="$3"

    printf '[%s] %s\n' "${status}" "${name}"
    if [[ -n "${output}" ]]; then
        while IFS= read -r line; do
            [[ -z "${line}" ]] && continue
            printf '  %s\n' "${line}"
        done <<< "${output}"
    fi
}

run_check() {
    local name="$1"
    shift

    local output=""
    if output="$("$@" 2>&1)"; then
        PASS_COUNT=$((PASS_COUNT + 1))
        print_result "PASS" "${name}" "${output}"
        return 0
    fi

    FAIL_COUNT=$((FAIL_COUNT + 1))
    print_result "FAIL" "${name}" "${output}"
    return 1
}

check_required_documents() {
    local required_docs=(
        "README.md"
        "DEVELOPMENT.md"
        "docs/README.md"
        "docs/product/README.md"
        "docs/product/owners.md"
        "docs/product/system-overview.md"
        "docs/development/README.md"
        "docs/development/owners.md"
        "docs/development/workflow.md"
        "docs/development/repository-map.md"
        "docs/development/commands-and-proofs.md"
        "docs/development/stages-and-governance.md"
        "docs/operations/README.md"
        "docs/tooling/README.md"
        "docs/architecture/README.md"
        "docs/architecture/information-system-governance-and-classification.md"
        "docs/archive/README.md"
        "docs/stages/INDEX.md"
        "docs/archive/operations/README.md"
        "docs/archive/documentation/README.md"
        "cmd/README.md"
        "internal/README.md"
        "deploy/README.md"
        "scripts/README.md"
        "tests/README.md"
        "scripts/opencode-consistency-check.sh"
    )
    local missing=()
    local doc

    for doc in "${required_docs[@]}"; do
        [[ -f "${doc}" ]] || missing+=("${doc}")
    done

    if (( ${#missing[@]} > 0 )); then
        printf 'missing required repository documents:\n'
        printf '  - %s\n' "${missing[@]}"
        return 1
    fi

    printf 'required repository documents present (%d files)' "${#required_docs[@]}"
}

check_primary_doc_indexes() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

checks = [
    ("docs/product", Path("docs/product/README.md")),
    ("docs/development", Path("docs/development/README.md")),
    ("docs/tooling", Path("docs/tooling/README.md")),
]

issues = []
total = 0
for directory, index_path in checks:
    text = index_path.read_text()
    referenced = set(Path(ref).name for ref in re.findall(r"\(([^)]+\.md)\)", text))
    files = sorted(
        path.name
        for path in Path(directory).glob("*.md")
        if path.name != "README.md"
    )
    total += len(files)
    missing = [name for name in files if name not in referenced]
    if missing:
        issues.append(f"{directory}: missing from {index_path.as_posix()}")
        issues.extend(f"  - {name}" for name in missing)

if issues:
    print("primary docs missing from their area indexes:")
    for item in issues:
        print(item)
    sys.exit(1)

print(f"primary area indexes cover product/development/tooling docs ({total} docs)")
PY
}

check_architecture_entrypoints() {
    python3 - <<'PY'
from pathlib import Path
import sys

index_text = Path("docs/architecture/README.md").read_text()
required = [
    "system-vision.md",
    "system-principles.md",
    "runtime-target.md",
    "actor-ownership.md",
    "market-foundry-evolution-playbook.md",
    "stage-definition-of-done.md",
    "anti-debt-checklist.md",
    "information-system-governance-and-classification.md",
]
missing = [item for item in required if item not in index_text]
if missing:
    print("docs/architecture/README.md is missing required architecture entrypoints:")
    for item in missing:
        print(f"  - {item}")
    sys.exit(1)

print(f"docs/architecture/README.md exposes required architecture entrypoints ({len(required)} docs)")
PY
}

check_stage_index_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

index_text = Path("docs/stages/INDEX.md").read_text()
indexed = set(re.findall(r"\((stage-[^)]+-report\.md)\)", index_text))
actual = set(path.name for path in Path("docs/stages").glob("stage-*-report.md"))

missing = sorted(actual - indexed)
stale = sorted(indexed - actual)

if missing or stale:
    if missing:
        print("stage reports missing from docs/stages/INDEX.md:")
        for item in missing:
            print(f"  - {item}")
    if stale:
        print("stale docs/stages/INDEX.md references:")
        for item in stale:
            print(f"  - {item}")
    sys.exit(1)

print(f"docs/stages/INDEX.md aligned with stage report inventory ({len(actual)} reports)")
PY
}

check_support_doc_links() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

files = [
    Path("README.md"),
    Path("DEVELOPMENT.md"),
    Path("docs/README.md"),
    Path("docs/product/README.md"),
    Path("docs/product/owners.md"),
    Path("docs/product/system-overview.md"),
    Path("docs/development/README.md"),
    Path("docs/development/owners.md"),
    Path("docs/development/workflow.md"),
    Path("docs/development/repository-map.md"),
    Path("docs/development/commands-and-proofs.md"),
    Path("docs/development/stages-and-governance.md"),
    Path("docs/operations/README.md"),
    Path("docs/tooling/README.md"),
    Path("docs/architecture/README.md"),
    Path("docs/architecture/information-system-governance-and-classification.md"),
    Path("docs/archive/README.md"),
    Path("docs/archive/operations/README.md"),
    Path("docs/archive/documentation/README.md"),
    Path("docs/stages/INDEX.md"),
    Path("scripts/README.md"),
]

link_pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
broken = []

for path in files:
    text = path.read_text()
    for target in link_pattern.findall(text):
        clean = target.split("#", 1)[0]
        if not clean or "://" in clean or clean.startswith("mailto:"):
            continue
        resolved = (path.parent / clean).resolve()
        if not resolved.exists():
            broken.append(f"{path.as_posix()} -> {target}")

if broken:
    print("broken local links in active support docs:")
    for item in broken:
        print(f"  - {item}")
    sys.exit(1)

print(f"local markdown links resolved across {len(files)} support docs")
PY
}

check_active_surface_shape() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

allowed = {
    "docs/product": {"README.md", "owners.md", "system-overview.md"},
    "docs/development": {
        "README.md",
        "owners.md",
        "workflow.md",
        "repository-map.md",
        "commands-and-proofs.md",
        "stages-and-governance.md",
    },
    "docs/operations": {"README.md"},
}

issues = []
for directory, expected in allowed.items():
    actual = {path.name for path in Path(directory).glob("*.md")}
    extra = sorted(actual - expected)
    missing = sorted(expected - actual)
    if missing:
        issues.append(f"{directory}: missing required active docs")
        issues.extend(f"  - {name}" for name in missing)
    if extra:
        issues.append(f"{directory}: unexpected active docs")
        issues.extend(f"  - {name}" for name in extra)

stage_entries = sorted(
    path.name for path in Path("docs/stages").glob("*.md")
    if path.name != "INDEX.md" and not re.fullmatch(r"stage-[a-z0-9-]+-report\.md", path.name)
)
if stage_entries:
    issues.append("docs/stages contains non-report markdown files:")
    issues.extend(f"  - {name}" for name in stage_entries)

if issues:
    print("active surface topology drift detected:")
    for item in issues:
        print(item)
    sys.exit(1)

print("active support surfaces stay bounded to the approved lightweight topology")
PY
}

check_owner_maps_and_orphans() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

table_row = re.compile(r"^\| ([^|]+) \| \[([^\]]+)\]\(([^)]+)\) \|", re.M)
issues = []
covered = []

for owner_map in [Path("docs/development/owners.md"), Path("docs/product/owners.md")]:
    text = owner_map.read_text()
    rows = table_row.findall(text)
    if not rows:
        issues.append(f"{owner_map.as_posix()}: owner map table not found")
        continue

    seen_subjects = {}
    seen_owner_paths = {}
    for subject, _label, rel in rows:
        subject_key = subject.strip().lower()
        if subject_key in seen_subjects:
            issues.append(
                f"{owner_map.as_posix()}: competing owner entries for subject `{subject.strip()}`"
            )
        else:
            seen_subjects[subject_key] = rel

        owner_path = Path(rel.split("#", 1)[0]).name
        if owner_path in seen_owner_paths:
            issues.append(
                f"{owner_map.as_posix()}: owner doc `{owner_path}` appears more than once"
            )
        else:
            seen_owner_paths[owner_path] = subject
            covered.append(owner_path)

readme_coverage = {
    "docs/product/README.md": {Path(ref).name for ref in re.findall(r"\(([^)]+\.md)\)", Path("docs/product/README.md").read_text())},
    "docs/development/README.md": {Path(ref).name for ref in re.findall(r"\(([^)]+\.md)\)", Path("docs/development/README.md").read_text())},
}

for directory, owner_map, readme in [
    ("docs/product", Path("docs/product/owners.md"), Path("docs/product/README.md")),
    ("docs/development", Path("docs/development/owners.md"), Path("docs/development/README.md")),
]:
    owner_text = owner_map.read_text()
    owner_refs = {Path(rel).name for *_rest, rel in table_row.findall(owner_text)}
    readme_refs = readme_coverage[readme.as_posix()]
    for path in sorted(Path(directory).glob("*.md")):
        if path.name == "README.md":
            continue
        if path.name not in owner_refs and path.name not in readme_refs:
            issues.append(f"{path.as_posix()}: orphaned active doc (not reachable from README or owner map)")

if issues:
    print("owner-map competition or orphaned active docs detected:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("owner maps stay unique and active docs are reachable from README/owner entrypoints")
PY
}

check_navigation_consistency() {
    python3 - <<'PY'
from pathlib import Path
import sys

checks = {
    "docs/README.md": [
        "product/README.md",
        "development/README.md",
        "tooling/README.md",
        "architecture/README.md",
        "stages/INDEX.md",
        "archive/README.md",
    ],
    "docs/operations/README.md": [
        "../development/workflow.md",
        "../development/repository-map.md",
        "../development/commands-and-proofs.md",
        "../development/stages-and-governance.md",
        "../tooling/README.md",
        "../archive/operations/README.md",
    ],
    "docs/development/stages-and-governance.md": [
        "../stages/INDEX.md",
        "../architecture/stage-definition-of-done.md",
        "../architecture/market-foundry-evolution-playbook.md",
        "../architecture/information-system-governance-and-classification.md",
    ],
    ".opencode/context/repo/documentation-topology.md": [
        "docs/development/owners.md",
        "docs/product/owners.md",
        "docs/tooling/README.md",
        "docs/architecture/README.md",
        "docs/architecture/information-system-governance-and-classification.md",
        "docs/stages/INDEX.md",
        "docs/archive/README.md",
    ],
}

issues = []
for rel, tokens in checks.items():
    text = Path(rel).read_text()
    for token in tokens:
        if token not in text:
            issues.append(f"{rel}: missing navigation anchor `{token}`")

if issues:
    print("navigation inconsistency detected across docs and .opencode:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("docs and .opencode navigation anchors stay aligned")
PY
}

check_historical_leakage() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

active_dirs = [Path("docs/product"), Path("docs/development"), Path("docs/operations")]
historical_name = re.compile(
    r"(report|charter|gate|closure|reconciliation|timeline|wave|evidence|archive)",
    re.I,
)
issues = []

for directory in active_dirs:
    for path in sorted(directory.glob("*.md")):
        if path.name == "README.md":
            continue
        if historical_name.search(path.stem):
            issues.append(f"{path.as_posix()}: historical filename pattern in active primary surface")

if issues:
    print("historical evidence is leaking into active primary surfaces:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("active primary surfaces stay free of stage/history-shaped filenames")
PY
}

check_make_docs_output() {
    local output
    output="$(make docs)"
    local required=(
        "docs/product/README.md"
        "docs/product/owners.md"
        "docs/development/README.md"
        "docs/development/owners.md"
        "docs/development/workflow.md"
        "docs/development/repository-map.md"
        "docs/development/commands-and-proofs.md"
        "docs/development/stages-and-governance.md"
        "docs/tooling/README.md"
        "docs/architecture/README.md"
        "docs/stages/INDEX.md"
        "docs/archive/README.md"
    )
    local missing=()
    local item
    for item in "${required[@]}"; do
        grep -Fq "${item}" <<< "${output}" || missing+=("${item}")
    done
    if (( ${#missing[@]} > 0 )); then
        printf 'make docs output missing required entrypoints:\n'
        printf '  - %s\n' "${missing[@]}"
        return 1
    fi
    printf 'make docs prints the curated active entrypoints'
}

check_opencode_integration() {
    ./scripts/opencode-consistency-check.sh
}

overall_status=0
run_check "required-documents" check_required_documents || overall_status=1
run_check "primary-doc-indexes" check_primary_doc_indexes || overall_status=1
run_check "architecture-entrypoints" check_architecture_entrypoints || overall_status=1
run_check "stage-index-alignment" check_stage_index_alignment || overall_status=1
run_check "support-doc-links" check_support_doc_links || overall_status=1
run_check "active-surface-shape" check_active_surface_shape || overall_status=1
run_check "owner-maps-and-orphans" check_owner_maps_and_orphans || overall_status=1
run_check "navigation-consistency" check_navigation_consistency || overall_status=1
run_check "historical-leakage" check_historical_leakage || overall_status=1
run_check "make-docs-output" check_make_docs_output || overall_status=1
run_check "opencode-integration" check_opencode_integration || overall_status=1

printf '\nSummary: %d passed, %d failed\n' "${PASS_COUNT}" "${FAIL_COUNT}"
exit "${overall_status}"
