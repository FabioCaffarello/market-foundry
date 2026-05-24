---
name: inventory
description: Produce structured inventory of an area before making changes.
arguments:
  - name: area
    required: true
    description: The area to inventory (e.g., 'scripts/', 'docs/decisions/', 'tools/').
---

Produce a structured inventory of `$1` before making significant
changes. Saves output to `/tmp/inventory-<sanitized-name>.md`.

## Why

Phase 1-5 work consistently used "inventory-first" as the foundation
for fact-dense changes. Producing a structured ground-truth snapshot
before transformation prevents drift between premise and reality
(see e.g. P1A.4a runtime inventory, P4.5 Dependabot triage,
P5.0 environment audit).

## Run

```bash
AREA="$1"
SAFE_NAME=$(echo "$AREA" | tr '/' '-' | sed 's/-$//')
OUT="/tmp/inventory-${SAFE_NAME}.md"

{
    echo "# Inventory: $AREA"
    echo "Generated: $(date -u +%FT%TZ)"
    echo ""

    echo "## Files"
    echo '```'
    find "$AREA" -type f 2>/dev/null | sort
    echo '```'
    echo ""

    echo "## Sizes (LOC)"
    echo '```'
    find "$AREA" -type f -exec wc -l {} + 2>/dev/null | sort -n
    echo '```'
    echo ""

    echo "## Last modified (git)"
    echo '```'
    find "$AREA" -type f 2>/dev/null | while read -r f; do
        last=$(git log -1 --format='%ad %s' --date=short -- "$f" 2>/dev/null | cut -c 1-100)
        printf "%-60s %s\n" "$f" "$last"
    done | sort
    echo '```'
    echo ""

    echo "## Subdirectories (if any)"
    echo '```'
    find "$AREA" -maxdepth 2 -type d 2>/dev/null | sort
    echo '```'
} > "$OUT"

cat "$OUT"
```

## Use

Treat the inventory as foundation for subsequent planning. Report
key observations (counts, surprising entries, age skew) to owner
before proposing changes.
