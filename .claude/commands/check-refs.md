---
name: check-refs
description: Find all active references to a path before deletion or rename.
arguments:
  - name: path
    required: true
    description: The path (file or directory) to check references for.
---

Find all active references to `$1` (the path argument) across source
code, configuration, documentation, Makefile, and CI workflows.

## Why

Deletion or rename without a ref-check is the leading cause of broken
validation/orientation infrastructure. Phase 1-4 surfaced multiple
instances where stale-infrastructure-post-restructure caused
verification or onboarding to silently fail — most recently the
P4.0 `backups/` near-miss (a recommendation to `rm -rf backups/`
would have destroyed 96 files of operational data and broken 6
Makefile targets; the pre-audit ref-check stopped the action).

## Search

```bash
PATH_TO_CHECK="$1"
echo "=== References to '$PATH_TO_CHECK' ==="
echo ""

echo "## In source code"
grep -rln "$PATH_TO_CHECK" \
    --include="*.go" \
    --include="*.rs" \
    --include="*.py" \
    --include="*.sh" \
    --include="*.bash" \
    . 2>/dev/null | head -20
echo ""

echo "## In config files"
grep -rln "$PATH_TO_CHECK" \
    --include="*.yml" \
    --include="*.yaml" \
    --include="*.toml" \
    --include="*.json" \
    --include="*.jsonc" \
    . 2>/dev/null | head -20
echo ""

echo "## In docs"
grep -rln "$PATH_TO_CHECK" \
    --include="*.md" \
    docs/ README.md AGENTS.md CLAUDE.md 2>/dev/null | head -20
echo ""

echo "## In Makefile"
grep -n "$PATH_TO_CHECK" Makefile 2>/dev/null | head -10
echo ""

echo "## In .github/workflows/"
grep -rln "$PATH_TO_CHECK" .github/ 2>/dev/null | head -10
```

## Interpretation

For each reference found, classify it as:

- **Active**: load-bearing. Removal would break validation, build, or
  orientation. Must be updated as part of the deletion/rename.
- **Historical**: narrative reference in docs. Can be left or updated
  to reflect new reality.
- **Dead**: already obsolete (e.g., references something else removed
  earlier). Safe to remove with the path.

Report classification to owner before proceeding. Pause if any
"active" references need decisions on how to update.
