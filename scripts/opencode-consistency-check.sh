#!/usr/bin/env bash
#
# opencode-consistency-check.sh -- lightweight consistency checks for .opencode.
#
# This script protects a small set of high-signal invariants for the thin
# OpenCode navigation layer:
# - required .opencode entrypoints and profile wiring exist
# - the approved minimal file topology remains explicit and bounded
# - local references in .opencode resolve to real files, directories, or globs
# - explicit make-target references map to the real Makefile surface
# - context files remain reachable from profile/navigation entrypoints
# - each context area keeps a canonical owner-doc anchor and consistent local navigation
# - removed/prohibited repository surfaces do not reappear as live guidance
# - reports remain reports and do not become another active owner surface
# - context docs stay thin enough to avoid becoming duplicate owner docs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat <<'EOF'
Usage: ./scripts/opencode-consistency-check.sh [--help]

Runs lightweight consistency checks for the .opencode navigation layer.
Skips cleanly when .opencode/ is absent.
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

cd "${PROJECT_ROOT}"

python3 - <<'PY'
from pathlib import Path
import json
import re
import sys

root = Path(".").resolve()
opencode = root / ".opencode"

if not opencode.exists():
    print(".opencode absent; skipping OpenCode consistency checks")
    raise SystemExit(0)

issues = []
checks = []

required_files = [
    ".opencode/README.md",
    ".opencode/TARGET-TREE.md",
    ".opencode/config.json",
    ".opencode/opencode.json",
    ".opencode/agent/core/foundry-agent.md",
    ".opencode/context/navigation.md",
    ".opencode/profiles/essential/profile.json",
    ".opencode/profiles/developer/profile.json",
]

missing = [path for path in required_files if not (root / path).is_file()]
if missing:
    issues.append("missing required .opencode files:")
    issues.extend(f"  - {path}" for path in missing)
else:
    checks.append(f"required .opencode files present ({len(required_files)} files)")

allowed_root_files = {
    "README.md",
    "TARGET-TREE.md",
    "config.json",
    "opencode.json",
}
allowed_report_pattern = re.compile(r"^O\d+-report\.md$")
unexpected_root_files = sorted(
    path.name
    for path in opencode.iterdir()
    if path.is_file()
    and path.name not in allowed_root_files
    and not allowed_report_pattern.match(path.name)
)

if unexpected_root_files:
    issues.append("unexpected .opencode root files:")
    issues.extend(f"  - .opencode/{name}" for name in unexpected_root_files)
else:
    checks.append(".opencode root files stay bounded to config, navigation, and O-reports")

allowed_top_level_dirs = {"agent", "context", "profiles"}
top_level_dirs = {path.name for path in opencode.iterdir() if path.is_dir()}
unexpected_top_level_dirs = sorted(top_level_dirs - allowed_top_level_dirs)

if unexpected_top_level_dirs:
    issues.append("unexpected .opencode top-level directories:")
    issues.extend(f"  - .opencode/{name}" for name in unexpected_top_level_dirs)
else:
    checks.append(".opencode top-level directories match the approved minimal topology")

agent_root = opencode / "agent"
if agent_root.exists():
    agent_dirs = {path.name for path in agent_root.iterdir() if path.is_dir()}
    missing_agent_dirs = sorted({"core"} - agent_dirs)
    unexpected_agent_dirs = sorted(agent_dirs - {"core"})
    if missing_agent_dirs or unexpected_agent_dirs:
        if missing_agent_dirs:
            issues.append("missing required .opencode/agent subdirectories:")
            issues.extend(f"  - .opencode/agent/{name}" for name in missing_agent_dirs)
        if unexpected_agent_dirs:
            issues.append("unexpected .opencode/agent subdirectories:")
            issues.extend(f"  - .opencode/agent/{name}" for name in unexpected_agent_dirs)
    else:
        checks.append(".opencode agent surface stays bounded to the single root agent")

context_root = opencode / "context"
if context_root.exists():
    context_dirs = {path.name for path in context_root.iterdir() if path.is_dir()}
    expected_context_dirs = {"repo", "runtime", "change", "intelligence"}
    missing_context_dirs = sorted(expected_context_dirs - context_dirs)
    unexpected_context_dirs = sorted(context_dirs - expected_context_dirs)
    if missing_context_dirs or unexpected_context_dirs:
        if missing_context_dirs:
            issues.append("missing required .opencode context areas:")
            issues.extend(f"  - .opencode/context/{name}" for name in missing_context_dirs)
        if unexpected_context_dirs:
            issues.append("unexpected .opencode context areas:")
            issues.extend(f"  - .opencode/context/{name}" for name in unexpected_context_dirs)
    else:
        checks.append(".opencode context areas stay bounded to repo/runtime/change/intelligence")

approved_context_files = {
    "navigation.md",
    "repo/navigation.md",
    "repo/repository-shape.md",
    "repo/development-workflow.md",
    "repo/architecture-boundaries.md",
    "repo/tooling-contracts.md",
    "repo/documentation-topology.md",
    "runtime/navigation.md",
    "runtime/services-topology.md",
    "runtime/configs-compose-streams.md",
    "runtime/smoke-and-live-flows.md",
    "runtime/troubleshooting-paths.md",
    "change/navigation.md",
    "change/impact-analysis.md",
    "change/tdd-and-validation.md",
    "change/stage-execution.md",
    "change/safe-change-rules.md",
    "intelligence/navigation.md",
    "intelligence/raccoon-cli-usage.md",
    "intelligence/make-target-map.md",
    "intelligence/repo-guardrails.md",
    "intelligence/code-intelligence-paths.md",
}
actual_context_files = {
    path.relative_to(context_root).as_posix()
    for path in context_root.rglob("*.md")
}
missing_context_files = sorted(approved_context_files - actual_context_files)
unexpected_context_files = sorted(actual_context_files - approved_context_files)
if missing_context_files or unexpected_context_files:
    if missing_context_files:
        issues.append("missing approved .opencode context files:")
        issues.extend(f"  - .opencode/context/{name}" for name in missing_context_files)
    if unexpected_context_files:
        issues.append("unexpected .opencode context files:")
        issues.extend(f"  - .opencode/context/{name}" for name in unexpected_context_files)
else:
    checks.append(".opencode context file topology matches the approved minimal set")


def load_json(path: Path):
    return json.loads(path.read_text())


profile_files = []
profile_includes = set()

try:
    config = load_json(opencode / "config.json")
    opencode_json = load_json(opencode / "opencode.json")
except Exception as exc:
    issues.append(f"failed to parse .opencode JSON metadata: {exc}")
    config = {}
    opencode_json = {}
else:
    if config.get("agent") != "foundry-agent":
        issues.append(".opencode/config.json: agent must remain 'foundry-agent'")
    if opencode_json.get("contextRoot") != "context":
        issues.append(".opencode/opencode.json: contextRoot must remain 'context'")

    profile_files = [Path(value) for value in opencode_json.get("profiles", {}).values()]
    if not profile_files:
        issues.append(".opencode/opencode.json: expected at least one profile registration")
    for rel in profile_files:
        path = opencode / rel
        if not path.is_file():
            issues.append(f".opencode/opencode.json: missing profile {rel.as_posix()}")
            continue
        try:
            profile = load_json(path)
        except Exception as exc:
            issues.append(f"{path.relative_to(root).as_posix()}: failed to parse JSON: {exc}")
            continue
        if profile.get("agent") != "foundry-agent":
            issues.append(f"{path.relative_to(root).as_posix()}: agent must remain foundry-agent")
        for include in profile.get("includes", []):
            include_path = opencode / include
            if not include_path.is_file():
                issues.append(f"{path.relative_to(root).as_posix()}: missing include {include}")
            else:
                profile_includes.add(include_path.resolve())
    checks.append(f"opencode profile wiring resolved ({len(profile_files)} profiles)")

agent_names = {}
for rel in [
    ".opencode/agent/core/foundry-agent.md",
]:
    path = root / rel
    if not path.is_file():
        continue
    match = re.search(r"^name:\s*([^\n]+)$", path.read_text(), re.M)
    if match:
        agent_names[match.group(1).strip()] = rel

if "FoundryAgent" not in agent_names:
    issues.append("missing FoundryAgent frontmatter registration in .opencode/agent/core/foundry-agent.md")
else:
    checks.append("OpenCode agent wiring resolves to the Foundry agent")

all_md = sorted(opencode.rglob("*.md"))
all_context = {path.resolve() for path in opencode.joinpath("context").rglob("*.md")}
path_refs = {}
broken_refs = []
token_pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)|`([^`]+)`")
make_pattern = re.compile(r"\bmake\s+([A-Za-z0-9_.-]+)\b")
root_files = {"AGENTS.md", "Makefile", "README.md", "DEVELOPMENT.md"}
repo_prefixes = ("docs/", "cmd/", "internal/", "deploy/", "scripts/", "tests/", "tools/", ".opencode/")
repo_suffixes = (".md", ".json", ".sh", ".yaml", ".yml")
nonexistent_prohibited_tokens = {".context/", ".tmp/"}

for path in all_md:
    text = path.read_text()
    for match in token_pattern.finditer(text):
        token = match.group(1) or match.group(2)
        if not token:
            continue
        if token.startswith(("http://", "https://", "mailto:", "#")):
            continue
        if token.startswith("make ") or token.startswith("raccoon-cli "):
            continue
        if " " in token and not token.startswith(("./", "../")):
            continue
        looks_like_path = (
            token in root_files
            or token.startswith(("./", "../"))
            or token.startswith(repo_prefixes)
            or token.endswith(repo_suffixes)
            or token.endswith("/")
            or "/" in token
        )
        if not looks_like_path:
            continue

        path_refs.setdefault(path.resolve(), set()).add(token)
        target = token.split("#", 1)[0]
        if not target:
            continue
        if target in nonexistent_prohibited_tokens:
            continue
        base = root if (target in root_files or target.startswith(repo_prefixes)) else path.parent
        if "*" in target:
            if not list(base.glob(target)):
                broken_refs.append(f"{path.relative_to(root).as_posix()}: {token}")
            continue

        resolved = (base / target).resolve()
        if not resolved.exists():
            broken_refs.append(f"{path.relative_to(root).as_posix()}: {token}")

if broken_refs:
    issues.append("broken .opencode local references:")
    issues.extend(f"  - {item}" for item in sorted(broken_refs))
else:
    checks.append(f".opencode local references resolved across {len(all_md)} markdown files")

makefile_targets = set(
    re.findall(r"^([A-Za-z0-9_.-]+):(?:\s|$)", (root / "Makefile").read_text(), re.M)
)
unknown_targets = []
for path in all_md:
    for target in sorted(set(make_pattern.findall(path.read_text()))):
        if target not in makefile_targets:
            unknown_targets.append(f"{path.relative_to(root).as_posix()}: make {target}")

if unknown_targets:
    issues.append("unknown Makefile targets referenced from .opencode:")
    issues.extend(f"  - {item}" for item in unknown_targets)
else:
    checks.append(".opencode make-target references match the real Makefile surface")

owner_anchor_rules = {
    "context/navigation.md": [
        "AGENTS.md",
        "Makefile",
        "README.md",
        "DEVELOPMENT.md",
        "docs/operations/documentary-ownership-and-canonical-navigation.md",
        "docs/architecture/README.md",
    ],
    "context/repo/navigation.md": [
        "AGENTS.md",
        "Makefile",
        "README.md",
        "DEVELOPMENT.md",
        "docs/operations/documentary-ownership-and-canonical-navigation.md",
    ],
    "context/runtime/navigation.md": [
        "README.md",
        "DEVELOPMENT.md",
        "deploy/README.md",
        "docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md",
        "docs/operations/operational-proof-entrypoints-and-ownership.md",
    ],
    "context/change/navigation.md": [
        "AGENTS.md",
        "DEVELOPMENT.md",
        "docs/architecture/stage-definition-of-done.md",
        "docs/architecture/anti-debt-checklist.md",
        "docs/architecture/opus-guidance-rules.md",
    ],
    "context/intelligence/navigation.md": [
        "docs/operations/make-and-raccoon-cli-contract.md",
        "docs/tooling/cli-overview.md",
        "docs/operations/raccoon-cli-command-reference.md",
        "tools/raccoon-cli/README.md",
    ],
}
owner_anchor_issues = []
for rel, required_tokens in owner_anchor_rules.items():
    text = (opencode / rel).read_text()
    for token in required_tokens:
        if token not in text:
            owner_anchor_issues.append(f".opencode/{rel}: missing canonical owner anchor `{token}`")

if owner_anchor_issues:
    issues.append("missing canonical owner-doc anchors in .opencode navigation:")
    issues.extend(f"  - {item}" for item in owner_anchor_issues)
else:
    checks.append(".opencode navigation keeps canonical owner-doc anchors explicit")

navigation_expectations = {
    "repo": {
        "navigation": opencode / "context/repo/navigation.md",
        "members": [
            "repository-shape.md",
            "development-workflow.md",
            "architecture-boundaries.md",
            "tooling-contracts.md",
            "documentation-topology.md",
        ],
    },
    "runtime": {
        "navigation": opencode / "context/runtime/navigation.md",
        "members": [
            "services-topology.md",
            "configs-compose-streams.md",
            "smoke-and-live-flows.md",
            "troubleshooting-paths.md",
        ],
    },
    "change": {
        "navigation": opencode / "context/change/navigation.md",
        "members": [
            "impact-analysis.md",
            "tdd-and-validation.md",
            "stage-execution.md",
            "safe-change-rules.md",
        ],
    },
    "intelligence": {
        "navigation": opencode / "context/intelligence/navigation.md",
        "members": [
            "raccoon-cli-usage.md",
            "make-target-map.md",
            "repo-guardrails.md",
            "code-intelligence-paths.md",
        ],
    },
}

navigation_issues = []
for area, config_ in navigation_expectations.items():
    text = config_["navigation"].read_text()
    for member in config_["members"]:
        if member not in text:
            navigation_issues.append(
                f"{config_['navigation'].relative_to(root).as_posix()}: missing route to {member}"
            )

if navigation_issues:
    issues.append("inconsistent .opencode area navigation:")
    issues.extend(f"  - {item}" for item in navigation_issues)
else:
    checks.append(".opencode area navigation covers every approved local context file")

legacy_terms = ("quality-service", "Kafka", "validator", "consumer", "emulator", ".context/")
legacy_allow_markers = ("do not", "Do not", "removed", "Removed", "prohibited", "Prohibited", "reintroduce", "avoid", "Avoid")
legacy_hits = []

for path in sorted(opencode.rglob("*")):
    if not path.is_file() or path.suffix not in {".md", ".json"}:
        continue
    lines = path.read_text().splitlines()
    for lineno, line in enumerate(lines, start=1):
        window = " ".join(lines[max(0, lineno - 5):lineno]).lower()
        for term in legacy_terms:
            if term not in line:
                continue
            if term == "validator" and "validation" in line:
                continue
            if term == "consumer" and "consumption" in line:
                continue
            if any(marker.lower() in window for marker in legacy_allow_markers):
                continue
            legacy_hits.append(f"{path.relative_to(root).as_posix()}:{lineno}: {term}")

if legacy_hits:
    issues.append("legacy or removed surfaces referenced without explicit prohibition context:")
    issues.extend(f"  - {item}" for item in legacy_hits)
else:
    checks.append(".opencode does not promote removed or prohibited repository surfaces")

report_paths = sorted(opencode.glob("O*-report.md"))
report_issues = []
for path in report_paths:
    text = path.read_text()
    for token in (
        "Canonical owner docs",
        "Owner surfaces",
        "Owner docs",
        "Subject Ownership Map",
        "Start Here",
    ):
        if token in text:
            report_issues.append(
                f"{path.relative_to(root).as_posix()}: report is accumulating active owner-surface language (`{token}`)"
            )

if report_issues:
    issues.append(".opencode reports are drifting into active owner-doc territory:")
    issues.extend(f"  - {item}" for item in report_issues)
else:
    checks.append(".opencode reports remain reports instead of a parallel owner index")

reachable = set(profile_includes)
queue = list(profile_includes)

while queue:
    current = queue.pop()
    for token in path_refs.get(current, set()):
        target = token.split("#", 1)[0]
        if not target or "*" in target:
            continue
        resolved = (Path(current).parent / target).resolve()
        if resolved in all_context and resolved not in reachable:
            reachable.add(resolved)
            queue.append(resolved)

orphans = sorted(path.relative_to(root).as_posix() for path in (all_context - reachable))
if orphans:
    issues.append("orphaned .opencode context files (not reachable from profiles/navigation):")
    issues.extend(f"  - {item}" for item in orphans)
else:
    checks.append(f".opencode context files are reachable from profile/navigation entrypoints ({len(all_context)} files)")

oversized = []
for path in sorted(opencode.joinpath("context").rglob("*.md")):
    nonempty_lines = [line for line in path.read_text().splitlines() if line.strip()]
    if len(nonempty_lines) > 40:
        oversized.append(f"{path.relative_to(root).as_posix()}: {len(nonempty_lines)} non-empty lines")

if oversized:
    issues.append("oversized .opencode context docs risk duplicating owner docs:")
    issues.extend(f"  - {item}" for item in oversized)
else:
    checks.append("context docs remain thin enough to stay navigational")

if issues:
    for issue in issues:
        print(issue)
    raise SystemExit(1)

for check in checks:
    print(check)
PY
