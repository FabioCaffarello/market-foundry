#!/usr/bin/env bash
#
# repository-consistency-check.sh -- lightweight repository consistency guard rail.
#
# This script protects a small set of high-signal repository invariants:
# - required support/documentation entrypoints exist
# - docs area entrypoints remain present
# - stage reports follow the repository naming/layout convention
# - stage reports remain indexed in docs/stages/INDEX.md
# - local links in primary support docs resolve
# - canonical docs only reference real Makefile targets
# - Makefile script wrappers point to real executable scripts
# - public script wrappers remain discoverable and self-describing
# - workflow owner docs keep the same minimal public loop and surface boundary
# - Make wrappers stay aligned with the grouped raccoon-cli taxonomy they promote
# - scripts catalog rows stay aligned with the real Make target to script mapping
# - bootstrap, Makefile, script catalog, and CLI docs stay aligned on the
#   governed support surface

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat <<'EOF'
Usage: ./scripts/repository-consistency-check.sh [--help]

Runs lightweight repository consistency checks for naming, documentation
entrypoints, stage indexing, support-doc links, workflow/Make/CLI convergence,
and Makefile script hygiene.
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
        "docs/operations/README.md"
        "docs/operations/documentary-ownership-and-canonical-navigation.md"
        "docs/operations/make-and-raccoon-cli-contract.md"
        "docs/operations/documentation-system-hardening.md"
        "docs/operations/documentation-governance-entrypoints-and-taxonomy.md"
        "docs/operations/repository-metadata-indexes-and-developer-navigation-system.md"
        "docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md"
        "docs/operations/developer-workflow-unification.md"
        "docs/operations/developer-onboarding-and-troubleshooting-guide.md"
        "docs/operations/makefile-targets-reference-and-conventions.md"
        "docs/operations/scripts-catalog-and-usage-guide.md"
        "docs/operations/smoke-and-operational-harness-governance.md"
        "docs/operations/operational-proof-entrypoints-and-ownership.md"
        "docs/operations/repository-support-surface-canonical-model.md"
        "docs/operations/repository-architecture-convergence.md"
        "docs/operations/lightweight-repository-guard-rails-and-consistency-checks.md"
        "docs/operations/repository-consistency-invariants-and-check-policy.md"
        "docs/operations/repository-policy-and-lightweight-enforcement-2.md"
        "docs/operations/repository-invariants-check-matrix-and-enforcement-policy.md"
        "docs/operations/stage-tooling-and-execution-governance-support.md"
        "docs/operations/stage-artifacts-conventions-and-support-model.md"
        "docs/operations/stage-documentation-governance-and-narrative-coherence.md"
        "docs/operations/stage-history-traceability-and-linking-model.md"
        "docs/operations/automation-support-for-waves-execution-continuity-and-repo-sustainability.md"
        "docs/operations/repository-automation-boundaries-high-value-routines-and-sustainability-rules.md"
        "docs/operations/repository-maintainability-economics-and-structural-cost-control.md"
        "docs/operations/repository-maintenance-hotspots-and-cost-reduction-principles.md"
        "docs/operations/strategic-operating-model-for-the-repository-as-a-development-platform.md"
        "docs/operations/repository-platform-governance-health-review-and-sustainability-model.md"
        "docs/operations/strategic-checkpoints-for-the-development-platform.md"
        "docs/operations/development-platform-checkpoint-triggers-scope-and-decision-model.md"
        "docs/operations/development-platform-readiness-model-for-future-foundry-waves.md"
        "docs/operations/readiness-signals-saturation-signals-and-wave-opening-rules.md"
        "docs/operations/criteria-for-opening-containing-or-rejecting-new-support-surfaces.md"
        "docs/operations/support-surface-expansion-decision-rules-and-examples.md"
        "docs/operations/continuous-prioritization-model-for-the-development-platform.md"
        "docs/operations/prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md"
        "docs/operations/long-term-documentation-and-operational-sustainability-model.md"
        "docs/operations/developer-environment-strategic-health-model.md"
        "docs/operations/repository-health-dimensions-signals-and-decision-usage.md"
        "docs/operations/repository-sustainability-review-routines-and-entropy-control.md"
        "docs/operations/periodic-review-model-for-repository-development-environment.md"
        "docs/operations/repository-review-cadence-triggers-and-follow-through-rules.md"
        "docs/operations/support-surface-sunset-consolidation-and-retirement-strategy.md"
        "docs/operations/support-surface-lifecycle-signals-and-consolidation-criteria.md"
        "docs/operations/tooling-evolution-patterns-and-repository-extension-discipline.md"
        "docs/operations/tooling-inclusion-deprecation-and-consolidation-rules.md"
        "docs/tooling/README.md"
        "docs/architecture/README.md"
        "docs/stages/INDEX.md"
        "cmd/README.md"
        "internal/README.md"
        "deploy/README.md"
        "scripts/README.md"
        "tests/README.md"
        "docs/archive/README.md"
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

check_active_doc_indexes() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

checks = [
    ("docs/operations", Path("docs/operations/README.md")),
    ("docs/tooling", Path("docs/tooling/README.md")),
]

issues = []
count = 0
for directory, index_path in checks:
    index_text = index_path.read_text()
    referenced = set(Path(ref).name for ref in re.findall(r"\(([^)]+\.md)\)", index_text))
    files = sorted(
        path.name
        for path in Path(directory).glob("*.md")
        if path.name != "README.md"
    )
    count += len(files)
    missing = [name for name in files if name not in referenced]
    if missing:
        issues.append(f"{directory}: missing from {index_path.as_posix()}")
        issues.extend(f"  - {name}" for name in missing)

if issues:
    print("active support docs missing from canonical area indexes:")
    for item in issues:
        print(item)
    sys.exit(1)

print(f"active operations/tooling docs are indexed in their canonical area READMEs ({count} docs)")
PY
}

check_health_model_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "model": Path("docs/operations/developer-environment-strategic-health-model.md"),
    "signals": Path("docs/operations/repository-health-dimensions-signals-and-decision-usage.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "repository-health-dimensions-signals-and-decision-usage.md" not in text["model"]:
    issues.append("strategic health model does not link to the signals/decision document")
if "developer-environment-strategic-health-model.md" not in text["signals"]:
    issues.append("signals/decision document does not link back to the strategic health model")

if issues:
    print("health-model canonical docs are not cross-linked:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("health-model canonical docs cross-link correctly")
PY
}

check_support_surface_lifecycle_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "strategy": Path("docs/operations/support-surface-sunset-consolidation-and-retirement-strategy.md"),
    "criteria": Path("docs/operations/support-surface-lifecycle-signals-and-consolidation-criteria.md"),
    "cadence": Path("docs/operations/repository-review-cadence-triggers-and-follow-through-rules.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "support-surface-lifecycle-signals-and-consolidation-criteria.md" not in text["strategy"]:
    issues.append("strategy doc does not link to the lifecycle-signals/criteria doc")
if "support-surface-sunset-consolidation-and-retirement-strategy.md" not in text["criteria"]:
    issues.append("criteria doc does not link back to the sunset/consolidation strategy doc")
if "repository-review-cadence-triggers-and-follow-through-rules.md" not in text["strategy"]:
    issues.append("strategy doc does not link to the cadence/trigger rules")

if issues:
    print("support-surface lifecycle docs are not cross-linked:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("support-surface lifecycle docs cross-link correctly")
PY
}

check_repository_platform_model_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "strategic": Path("docs/operations/strategic-operating-model-for-the-repository-as-a-development-platform.md"),
    "applied": Path("docs/operations/repository-platform-governance-health-review-and-sustainability-model.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "repository-platform-governance-health-review-and-sustainability-model.md" not in text["strategic"]:
    issues.append("strategic repository-platform model does not link to the applied governance model")
if "strategic-operating-model-for-the-repository-as-a-development-platform.md" not in text["applied"]:
    issues.append("applied repository-platform model does not link back to the strategic operating model")

if issues:
    print("repository-platform model docs are not cross-linked:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("repository-platform model docs cross-link correctly")
PY
}

check_prioritization_model_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "model": Path("docs/operations/continuous-prioritization-model-for-the-development-platform.md"),
    "examples": Path("docs/operations/prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md" not in text["model"]:
    issues.append("continuous prioritization model does not link to the criteria/examples document")
if "continuous-prioritization-model-for-the-development-platform.md" not in text["examples"]:
    issues.append("criteria/examples document does not link back to the continuous prioritization model")

if issues:
    print("prioritization-model docs are not cross-linked:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("prioritization-model docs cross-link correctly")
PY
}

check_strategic_checkpoint_model_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "model": Path("docs/operations/strategic-checkpoints-for-the-development-platform.md"),
    "companion": Path("docs/operations/development-platform-checkpoint-triggers-scope-and-decision-model.md"),
    "applied": Path("docs/operations/repository-platform-governance-health-review-and-sustainability-model.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "development-platform-checkpoint-triggers-scope-and-decision-model.md" not in text["model"]:
    issues.append("strategic checkpoint model does not link to the companion trigger/scope/decision doc")
if "strategic-checkpoints-for-the-development-platform.md" not in text["companion"]:
    issues.append("checkpoint companion doc does not link back to the strategic checkpoint model")
if "strategic-checkpoints-for-the-development-platform.md" not in text["applied"]:
    issues.append("applied governance model does not link to the strategic checkpoint model")

if issues:
    print("strategic checkpoint docs are not cross-linked correctly:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("strategic checkpoint docs cross-link correctly")
PY
}

check_support_surface_expansion_model_cross_links() {
    python3 - <<'PY'
from pathlib import Path
import sys

paths = {
    "criteria": Path("docs/operations/criteria-for-opening-containing-or-rejecting-new-support-surfaces.md"),
    "examples": Path("docs/operations/support-surface-expansion-decision-rules-and-examples.md"),
    "discipline": Path("docs/operations/tooling-evolution-patterns-and-repository-extension-discipline.md"),
}

text = {name: path.read_text() for name, path in paths.items()}
issues = []

if "support-surface-expansion-decision-rules-and-examples.md" not in text["criteria"]:
    issues.append("criteria doc does not link to the examples/decision-rules doc")
if "criteria-for-opening-containing-or-rejecting-new-support-surfaces.md" not in text["examples"]:
    issues.append("examples/decision-rules doc does not link back to the criteria doc")
if "criteria-for-opening-containing-or-rejecting-new-support-surfaces.md" not in text["discipline"]:
    issues.append("extension-discipline doc does not link to the C31 criteria doc")

if issues:
    print("support-surface expansion model docs are not cross-linked correctly:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("support-surface expansion model docs cross-link correctly")
PY
}

check_docs_area_entrypoints() {
    python3 - <<'PY'
from pathlib import Path
import sys

expected = {
    "docs": "README.md",
    "docs/operations": "README.md",
    "docs/tooling": "README.md",
    "docs/architecture": "README.md",
    "docs/archive": "README.md",
    "docs/stages": "INDEX.md",
}

missing = []
for directory, entrypoint in expected.items():
    path = Path(directory) / entrypoint
    if not path.is_file():
        missing.append(path.as_posix())

if missing:
    print("missing documentation area entrypoints:")
    for item in missing:
        print(f"  - {item}")
    sys.exit(1)

print(f"documentation area entrypoints present ({len(expected)} directories)")
PY
}

check_stage_report_naming() {
    local invalid=()
    local count=0
    local path basename

    while IFS= read -r -d '' path; do
        basename="$(basename "${path}")"
        count=$((count + 1))
        if [[ ! "${basename}" =~ ^stage-[a-z0-9-]+-report\.md$ ]]; then
            invalid+=("${basename}")
        fi
    done < <(find docs/stages -maxdepth 1 -type f -name '*.md' ! -name 'INDEX.md' -print0 | sort -z)

    if (( count == 0 )); then
        printf 'no stage reports found under docs/stages\n'
        return 1
    fi

    if (( ${#invalid[@]} > 0 )); then
        printf 'invalid stage report filenames:\n'
        printf '  - %s\n' "${invalid[@]}"
        return 1
    fi

    printf 'stage report filenames aligned with stage-*-report.md convention (%d files)' "${count}"
}

check_stage_report_shape() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

bad = []
for path in sorted(Path("docs/stages").glob("stage-*-report.md")):
    text = path.read_text()
    lines = text.splitlines()
    if not lines or not lines[0].startswith("# "):
        bad.append(f"{path.name}: missing level-1 title")
        continue
    h2_count = len(re.findall(r"^## ", text, re.M))
    if h2_count < 2:
        bad.append(f"{path.name}: expected at least 2 level-2 sections, found {h2_count}")

if bad:
    print("stage reports missing minimum document shape:")
    for item in bad:
        print(f"  - {item}")
    sys.exit(1)

count = len(list(Path("docs/stages").glob("stage-*-report.md")))
print(f"stage reports have a title and at least two level-2 sections ({count} files)")
PY
}

check_stage_index_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

index_text = Path("docs/stages/INDEX.md").read_text()
referenced = set(re.findall(r"\((stage-[^)]+-report\.md)\)", index_text))
actual = set(path.name for path in Path("docs/stages").glob("stage-*-report.md"))

missing = sorted(actual - referenced)
stale = sorted(referenced - actual)

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

root = Path(".").resolve()
files = [
    Path("README.md"),
    Path("DEVELOPMENT.md"),
    Path("docs/README.md"),
    Path("docs/architecture/README.md"),
    Path("docs/archive/README.md"),
    Path("docs/stages/INDEX.md"),
]
files.extend(sorted(Path("docs/operations").glob("*.md")))
files.extend(sorted(Path("docs/tooling").glob("*.md")))

pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
broken = []

for path in files:
    text = path.read_text()
    for target in pattern.findall(text):
        if target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        target = target.split("#", 1)[0]
        if not target:
            continue
        resolved = (path.parent / target).resolve()
        if not resolved.exists():
            broken.append(f"{path.as_posix()}: {target}")

if broken:
    print("broken local links in primary support docs:")
    for item in broken:
        print(f"  - {item}")
    sys.exit(1)

print(f"local markdown links resolved across {len(files)} primary support docs")
PY
}

check_primary_doc_make_targets() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
targets = set(re.findall(r"^([A-Za-z0-9_.-]+):(?:\s|$)", makefile, re.M))
docs = [
    Path("README.md"),
    Path("DEVELOPMENT.md"),
    Path("docs/operations/README.md"),
    Path("docs/operations/makefile-targets-reference-and-conventions.md"),
]

pattern = re.compile(r"`make\s+([A-Za-z0-9_.-]+)")
unknown = []

for path in docs:
    text = path.read_text()
    for target in sorted(set(pattern.findall(text))):
        if target not in targets:
            unknown.append(f"{path.as_posix()}: make {target}")

if unknown:
    print("canonical docs reference unknown Makefile targets:")
    for item in unknown:
        print(f"  - {item}")
    sys.exit(1)

print(f"canonical docs reference only real Makefile targets ({len(docs)} docs)")
PY
}

check_makefile_script_wrappers() {
    python3 - <<'PY'
from pathlib import Path
import re
import stat
import sys

makefile = Path("Makefile").read_text()
scripts = sorted(set(re.findall(r"\./(scripts/[^\s\\]+?\.sh)", makefile)))
issues = []

for rel in scripts:
    path = Path(rel)
    if not path.exists():
        issues.append(f"{rel}: missing")
        continue
    if not (path.stat().st_mode & stat.S_IXUSR):
        issues.append(f"{rel}: not executable")

if issues:
    print("Makefile script wrappers are inconsistent:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print(f"Makefile script wrappers resolved to executable files ({len(scripts)} scripts)")
PY
}

check_public_scripts_are_self_describing() {
    python3 - <<'PY'
from pathlib import Path
import subprocess
import sys

scripts = [
    "scripts/bootstrap-check.sh",
    "scripts/seed-configctl.sh",
    "scripts/diag-check.sh",
    "scripts/live-pipeline-activate.sh",
    "scripts/smoke-first-slice.sh",
    "scripts/smoke-multi-symbol.sh",
    "scripts/smoke-analytical-e2e.sh",
    "scripts/smoke-os-process-operational.sh",
    "scripts/smoke-restart-recovery.sh",
    "scripts/ci-wait-ready.sh",
    "scripts/codegen-integrated-check.sh",
    "scripts/codegen-equivalence-check.sh",
    "scripts/repository-consistency-check.sh",
    "scripts/stage-tooling.sh",
    "scripts/utils/for-each-module.sh",
    "scripts/utils/list-modules.sh",
]

issues = []
for rel in scripts:
    path = Path(rel)
    if not path.is_file():
        issues.append(f"{rel}: missing")
        continue

    with path.open() as fh:
        first_line = fh.readline().rstrip("\n")
    if first_line != "#!/usr/bin/env bash":
        issues.append(f"{rel}: expected bash shebang")
        continue

    proc = subprocess.run([str(path), "--help"], capture_output=True, text=True)
    if proc.returncode != 0:
        issues.append(f"{rel}: --help exited {proc.returncode}")

if issues:
    print("public script entrypoints are not self-describing:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print(f"public script entrypoints expose bash shebang + --help ({len(scripts)} scripts)")
PY
}

check_bootstrap_entrypoints_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

text = Path("scripts/bootstrap-check.sh").read_text()
match = re.search(r"required_paths=\(\n(?P<body>.*?)\n\)", text, re.S)
if not match:
    print("scripts/bootstrap-check.sh: could not locate required_paths block")
    sys.exit(1)

entries = set(re.findall(r'"([^"]+)"', match.group("body")))
required = {
    "Makefile",
    "README.md",
    "DEVELOPMENT.md",
    "docs/README.md",
    "docs/operations/README.md",
    "docs/operations/documentary-ownership-and-canonical-navigation.md",
    "docs/operations/documentation-system-hardening.md",
    "docs/operations/documentation-governance-entrypoints-and-taxonomy.md",
    "docs/operations/repository-policy-and-lightweight-enforcement-2.md",
    "docs/operations/repository-invariants-check-matrix-and-enforcement-policy.md",
    "docs/operations/developer-workflow-unification.md",
    "docs/operations/developer-onboarding-and-troubleshooting-guide.md",
    "docs/tooling/README.md",
    "docs/architecture/README.md",
    "docs/stages/INDEX.md",
    "deploy/compose/docker-compose.yaml",
    "deploy/envs/local.env",
    "tools/raccoon-cli/Cargo.toml",
    "go.work",
}

missing = sorted(required - entries)
if missing:
    print("bootstrap required_paths missing governed entrypoints:")
    for item in missing:
        print(f"  - {item}")
    sys.exit(1)

print(f"bootstrap required_paths include governed entrypoints ({len(required)} paths)")
PY
}

check_makefile_script_catalog_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
catalog = Path("docs/operations/scripts-catalog-and-usage-guide.md").read_text()

scripts = sorted(set(re.findall(r"\./(scripts/[^\s\\]+?\.sh)", makefile)))
missing = [rel for rel in scripts if rel not in catalog]

if missing:
    print("Makefile script wrappers missing from scripts catalog:")
    for item in missing:
        print(f"  - {item}")
    sys.exit(1)

print(f"scripts catalog covers all Makefile script wrappers ({len(scripts)} scripts)")
PY
}

check_workflow_owner_loop_alignment() {
    python3 - <<'PY'
from pathlib import Path
import sys

docs = {
    "DEVELOPMENT.md": [
        "`make bootstrap`",
        "`make check`",
        "`make tdd`",
        "`make verify`",
        "`make smoke*`",
        "`raccoon-cli`",
        "`scripts/*.sh`",
    ],
    "docs/operations/developer-workflow-unification.md": [
        "`make bootstrap`",
        "`make check`",
        "`make tdd`",
        "`make verify`",
        "`make smoke*`",
        "`raccoon-cli`",
        "`scripts/*.sh`",
    ],
    "docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md": [
        "`make bootstrap`",
        "`make check`",
        "`make tdd`",
        "`make verify`",
        "`make smoke*`",
        "direct `raccoon-cli`",
        "`scripts/*.sh`",
    ],
    "docs/operations/make-and-raccoon-cli-contract.md": [
        "`make check`",
        "`make tdd`",
        "`make verify`",
        "`make smoke*`",
        "`raccoon-cli`",
        "`scripts/*.sh`",
    ],
}

issues = []
for rel, tokens in docs.items():
    text = Path(rel).read_text()
    for token in tokens:
        if token not in text:
            issues.append(f"{rel}: missing {token}")

if issues:
    print("workflow owner docs drifted from the minimal public loop/boundary contract:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print(f"workflow owner docs preserve the minimal public loop and boundary ({len(docs)} docs)")
PY
}

check_makefile_raccoon_wrapper_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
doc = Path("docs/operations/makefile-targets-reference-and-conventions.md").read_text()
refs = Path("tools/raccoon-cli/src/command_refs.rs").read_text()

consts = dict(re.findall(r'pub\(crate\) const ([A-Z_]+): &str = "([^"]+)";', refs))
required_consts = {
    "CHECK_ARCH": "arch-guard",
    "CHECK_DRIFT": "drift-detect",
    "INSPECT_COVERAGE": "coverage-map",
    "CHANGE_TDD": "tdd",
    "CHANGE_BRIEFING": "briefing",
    "CHANGE_RECOMMEND": "recommend",
}

issues = []
for const_name, target in required_consts.items():
    command = consts.get(const_name)
    if command is None:
        issues.append(f"tools/raccoon-cli/src/command_refs.rs: missing {const_name}")
        continue
    suffix = command.removeprefix("raccoon-cli ")
    expected_recipe = f"$(RACCOON_BIN) --project-root . {suffix}"
    block_match = re.search(rf"(?ms)^{re.escape(target)}:.*?\n((?:\t.*\n)+)", makefile)
    if not block_match:
        issues.append(f"Makefile: could not locate recipe for `{target}`")
        continue
    if expected_recipe not in block_match.group(1):
        issues.append(f"Makefile: `{target}` no longer wraps `{command}`")
    if f"`{command}`" not in doc:
        issues.append(f"docs/operations/makefile-targets-reference-and-conventions.md: missing `{command}`")

quality_gate_expectations = {
    "quality-gate": "$(RACCOON_BIN) --project-root . check gate",
    "quality-gate-ci": "$(RACCOON_BIN) --project-root . check gate --profile ci --json",
    "quality-gate-deep": "$(RACCOON_BIN) --project-root . check gate --profile deep",
}
for target, expected_recipe in quality_gate_expectations.items():
    block_match = re.search(rf"(?ms)^{re.escape(target)}:.*?\n((?:\t.*\n)+)", makefile)
    if not block_match:
        issues.append(f"Makefile: could not locate recipe for `{target}`")
        continue
    if expected_recipe not in block_match.group(1):
        issues.append(f"Makefile: `{target}` no longer wraps the grouped `check gate` taxonomy")

if "`raccoon-cli check gate`" not in doc:
    issues.append("docs/operations/makefile-targets-reference-and-conventions.md: missing `raccoon-cli check gate`")

if issues:
    print("Makefile to raccoon-cli wrapper contract drift detected:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("Makefile raccoon wrappers stay aligned with grouped command refs and public docs")
PY
}

check_scripts_catalog_contract_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
catalog = Path("docs/operations/scripts-catalog-and-usage-guide.md").read_text()

targets = set(re.findall(r"^([A-Za-z0-9_.-]+):(?:\s|$)", makefile, re.M))
wrapper_pairs = set(
    (
        target,
        script,
    )
    for target, script in re.findall(
        r"(?ms)^([A-Za-z0-9_.-]+):.*?\n(?:\t.*\n)*?\t@\./(scripts/[^\s\\]+?\.sh)",
        makefile,
    )
    if not target.startswith(".")
)

documented_pairs = set()
documented_targets = set()
documented_scripts = set()

for line in catalog.splitlines():
    if not line.startswith("|"):
        continue
    cols = [part.strip() for part in line.split("|")[1:-1]]
    if len(cols) < 3:
        continue
    canonical_cell = cols[1]
    script_cell = cols[2]
    row_targets = set(re.findall(r"make ([A-Za-z0-9_.-]+)", canonical_cell))
    row_scripts = set(re.findall(r"(scripts/[A-Za-z0-9_./-]+\.sh)", script_cell))
    if row_targets:
        documented_targets |= row_targets
    if row_scripts:
        documented_scripts |= row_scripts
    for target in row_targets:
        for script in row_scripts:
            documented_pairs.add((target, script))

issues = []
for target, script in sorted(wrapper_pairs):
    if (target, script) not in documented_pairs:
        issues.append(
            f"docs/operations/scripts-catalog-and-usage-guide.md: missing row for `make {target}` -> `{script}`"
        )

for target in sorted(documented_targets):
    if target not in targets:
        issues.append(
            f"docs/operations/scripts-catalog-and-usage-guide.md: references nonexistent `make {target}`"
        )

for script in sorted(documented_scripts):
    path = Path(script)
    if not path.exists():
        issues.append(f"docs/operations/scripts-catalog-and-usage-guide.md: references missing `{script}`")
    elif not path.is_file() or not path.stat().st_mode & 0o111:
        issues.append(f"docs/operations/scripts-catalog-and-usage-guide.md: `{script}` is not executable")

if issues:
    print("scripts catalog drift detected against the real Make/script contract:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("scripts catalog preserves the real Make target to script mapping")
PY
}

check_canonical_smoke_taxonomy_alignment() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
targets = sorted(
    target
    for target in set(re.findall(r"^([A-Za-z0-9_.-]+):(?:\s|$)", makefile, re.M))
    if target == "smoke" or target.startswith("smoke-")
)

required = [target for target in targets if target != "smoke-help"]
docs = {
    "docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md": required,
    "docs/operations/makefile-targets-reference-and-conventions.md": required,
    "docs/operations/operational-proof-entrypoints-and-ownership.md": required,
    "docs/operations/smoke-and-operational-harness-governance.md": required,
}

issues = []
for rel, expected_targets in docs.items():
    text = Path(rel).read_text()
    for target in expected_targets:
        token = f"`make {target}`"
        if token not in text:
            issues.append(f"{rel}: missing {token}")

if issues:
    print("canonical smoke-taxonomy docs are not aligned with Makefile:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print(f"canonical smoke-taxonomy docs aligned with Makefile ({len(required)} targets across {len(docs)} docs)")
PY
}

check_direct_smoke_script_claims() {
    python3 - <<'PY'
from pathlib import Path
import re
import sys

makefile = Path("Makefile").read_text()
targets = set(re.findall(r"^([A-Za-z0-9_.-]+):(?:\s|$)", makefile, re.M))
scripts = sorted(Path("scripts").glob("smoke*.sh"))
issues = []

for path in scripts:
    text = path.read_text()
    for target in sorted(set(re.findall(r"`make\s+([A-Za-z0-9_.-]+)`", text))):
        if target not in targets:
            issues.append(f"{path.as_posix()}: claims nonexistent `make {target}`")

if issues:
    print("smoke scripts claim nonexistent Makefile entrypoints:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print(f"smoke scripts only claim real Makefile entrypoints ({len(scripts)} scripts)")
PY
}

check_cli_governance_surface() {
    python3 - <<'PY'
from pathlib import Path
import sys

cli = Path("tools/raccoon-cli/src/cli/mod.rs").read_text()
docs = Path("docs/operations/raccoon-cli-command-reference.md").read_text()
overview = Path("docs/tooling/cli-overview.md").read_text()

issues = []

cli_tokens = [
    'about = "Repository support CLI for market-foundry"',
    "not a product control plane",
    "Canonical taxonomy:",
    "check    repository guard rails and audits",
    "inspect  read-only structural and contract analysis",
    "change   impact mapping and validation guidance",
    "legacy   fragile or deprecated helper flows",
    "Prefer `make smoke*` for runtime proof.",
    'name = "runtime-smoke"',
    "Prefer Makefile operational flows instead:",
]
for token in cli_tokens:
    if token not in cli:
        issues.append(f"tools/raccoon-cli/src/cli/mod.rs: missing `{token}`")

doc_tokens = [
    "raccoon-cli check <subcommand>",
    "raccoon-cli inspect <subcommand>",
    "raccoon-cli change <subcommand> [TARGET...]",
    "raccoon-cli legacy runtime-smoke",
    "The preferred model is grouped usage.",
]
for token in doc_tokens:
    if token not in docs:
        issues.append(f"docs/operations/raccoon-cli-command-reference.md: missing `{token}`")

overview_tokens = [
    "| `check` | Repository guard rails and audits |",
    "| `inspect` | Read-only structural and contract analysis |",
    "| `change` | Impact mapping and validation guidance |",
    "| `legacy` | Deprecated or fragile helpers |",
]
for token in overview_tokens:
    if token not in overview:
        issues.append(f"docs/tooling/cli-overview.md: missing `{token}`")

if issues:
    print("CLI governance surface drift detected:")
    for item in issues:
        print(f"  - {item}")
    sys.exit(1)

print("CLI governance surface preserved across source and canonical docs")
PY
}

echo "=== repository-consistency-check ==="

overall_status=0
run_check "required-documents" check_required_documents || overall_status=1
run_check "docs-area-entrypoints" check_docs_area_entrypoints || overall_status=1
run_check "active-doc-indexes" check_active_doc_indexes || overall_status=1
run_check "health-model-cross-links" check_health_model_cross_links || overall_status=1
run_check "support-surface-lifecycle-cross-links" check_support_surface_lifecycle_cross_links || overall_status=1
run_check "repository-platform-model-cross-links" check_repository_platform_model_cross_links || overall_status=1
run_check "prioritization-model-cross-links" check_prioritization_model_cross_links || overall_status=1
run_check "strategic-checkpoint-model-cross-links" check_strategic_checkpoint_model_cross_links || overall_status=1
run_check "support-surface-expansion-model-cross-links" check_support_surface_expansion_model_cross_links || overall_status=1
run_check "stage-report-naming" check_stage_report_naming || overall_status=1
run_check "stage-report-shape" check_stage_report_shape || overall_status=1
run_check "stage-index-alignment" check_stage_index_alignment || overall_status=1
run_check "support-doc-links" check_support_doc_links || overall_status=1
run_check "primary-doc-make-targets" check_primary_doc_make_targets || overall_status=1
run_check "makefile-script-wrappers" check_makefile_script_wrappers || overall_status=1
run_check "public-scripts-self-describing" check_public_scripts_are_self_describing || overall_status=1
run_check "bootstrap-entrypoints-alignment" check_bootstrap_entrypoints_alignment || overall_status=1
run_check "makefile-script-catalog-alignment" check_makefile_script_catalog_alignment || overall_status=1
run_check "workflow-owner-loop-alignment" check_workflow_owner_loop_alignment || overall_status=1
run_check "makefile-raccoon-wrapper-alignment" check_makefile_raccoon_wrapper_alignment || overall_status=1
run_check "scripts-catalog-contract-alignment" check_scripts_catalog_contract_alignment || overall_status=1
run_check "canonical-smoke-taxonomy-alignment" check_canonical_smoke_taxonomy_alignment || overall_status=1
run_check "direct-smoke-script-claims" check_direct_smoke_script_claims || overall_status=1
run_check "cli-governance-surface" check_cli_governance_surface || overall_status=1

TOTAL_COUNT=$((PASS_COUNT + FAIL_COUNT))

if (( overall_status == 0 )); then
    printf 'verdict: PASS (%d checks)\n' "${TOTAL_COUNT}"
    exit 0
fi

printf 'verdict: FAIL (%d passed, %d failed)\n' "${PASS_COUNT}" "${FAIL_COUNT}"
exit 1
