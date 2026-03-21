#!/usr/bin/env bash
# codegen-integrated-check.sh
#
# Verifies that codegen-governed regions in target files match their golden
# snapshots. Reads the manifest from codegen/integrated.yaml so that new
# slices are checked automatically — no script edits needed.
#
# Usage: ./scripts/codegen-integrated-check.sh
#
# Exit codes:
#   0 - all integrated slices match their golden snapshots
#   1 - at least one slice has drifted or manifest is malformed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/codegen-integrated-check.sh [--help]

Validates integrated codegen-governed slices against their golden snapshots.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage_error "unknown argument: $1"
            ;;
    esac
    shift
done

require_commands awk diff grep sed

MANIFEST="$PROJECT_ROOT/codegen/integrated.yaml"

PASS=0
FAIL=0
CHECKED=0

# Normalize function: strip comments, collapse whitespace, trim, remove blanks.
# Mirrors codegen/compare.go normalizeForComparison logic.
normalize() {
    sed 's|//.*||' \
    | sed 's/\t/ /g' \
    | sed 's/^[[:space:]]*//' \
    | sed 's/[[:space:]]*$//' \
    | grep -v '^$'
}

# Extract region between codegen:begin and codegen:end markers.
# Uses exact substring matching (not regex) to prevent family=rsi matching
# family=rsi_oversold. Verifies the marker is followed by space/tab/EOL.
extract_region() {
    local file="$1"
    local begin_marker="$2"
    local end_marker="$3"
    awk -v begin="$begin_marker" -v end="$end_marker" '
    function exact_match(line, marker) {
        p = index(line, marker)
        if (p == 0) return 0
        rest = substr(line, p + length(marker))
        return (rest == "" || substr(rest, 1, 1) == " " || substr(rest, 1, 1) == "\t")
    }
    !found && exact_match($0, begin) { found = 1; next }
    found && exact_match($0, end)    { found = 0; next }
    found { print }
    ' "$file"
}

check_slice() {
    local family="$1"
    local artifact="$2"
    local golden="$3"
    local target="$4"
    local marker="$5"

    CHECKED=$((CHECKED + 1))

    local golden_path="$PROJECT_ROOT/$golden"
    local target_path="$PROJECT_ROOT/$target"

    if [[ ! -f "$golden_path" ]]; then
        echo "FAIL $family/$artifact: golden file not found: $golden"
        FAIL=$((FAIL + 1))
        return
    fi

    if [[ ! -f "$target_path" ]]; then
        echo "FAIL $family/$artifact: target file not found: $target"
        FAIL=$((FAIL + 1))
        return
    fi

    # Verify marker exists in target file.
    if ! grep -q "$marker" "$target_path"; then
        echo "FAIL $family/$artifact: marker not found in $target"
        echo "  expected marker: $marker"
        FAIL=$((FAIL + 1))
        return
    fi

    # Verify matching end marker exists.
    local end_marker="codegen:end ${artifact} family=${family}"
    if ! grep -q "$end_marker" "$target_path"; then
        echo "FAIL $family/$artifact: end marker not found in $target"
        echo "  expected: $end_marker"
        FAIL=$((FAIL + 1))
        return
    fi

    # Extract the marked region from target.
    local extracted
    extracted=$(extract_region "$target_path" "$marker" "$end_marker")

    if [[ -z "$extracted" ]]; then
        echo "FAIL $family/$artifact: empty region between markers in $target"
        FAIL=$((FAIL + 1))
        return
    fi

    # Normalize both golden and extracted for structural comparison.
    local golden_norm
    golden_norm=$(normalize < "$golden_path")

    local extracted_norm
    extracted_norm=$(printf '%s\n' "$extracted" | normalize)

    if [[ "$golden_norm" == "$extracted_norm" ]]; then
        echo "PASS $family/$artifact"
        PASS=$((PASS + 1))
    else
        echo "FAIL $family/$artifact: target has drifted from golden"
        echo "  golden:    $golden"
        echo "  target:    $target"
        echo "  marker:    $marker"
        echo ""
        diff <(echo "$golden_norm") <(echo "$extracted_norm") || true
        echo ""
        FAIL=$((FAIL + 1))
    fi
}

# ── Manifest-driven loop ─────────────────────────────────────────

echo "=== Codegen Integrated Slice Verification ==="
echo ""

if [[ ! -f "$MANIFEST" ]]; then
    echo "ERROR: manifest not found: $MANIFEST"
    exit 1
fi

# Parse integrated.yaml — extract family, artifact, golden, target, marker
# from each slice entry. Strips YAML list markers and indentation, then
# matches key: value pairs. Compatible with macOS and GNU awk.
parse_manifest() {
    awk '
    {
        # Trim leading whitespace and optional list marker "- ".
        gsub(/^[ \t]+/, "")
        sub(/^- /, "")
    }
    /^family:/ { sub(/^family: */, ""); family = $0 }
    /^artifact:/ { sub(/^artifact: */, ""); artifact = $0 }
    /^golden:/ { sub(/^golden: */, ""); golden = $0 }
    /^target:/ { sub(/^target: */, ""); target = $0 }
    /^marker:/ {
        sub(/^marker: */, "")
        gsub(/"/, "")
        marker = $0
        print family "|" artifact "|" golden "|" target "|" marker
    }
    ' "$MANIFEST"
}

SLICE_COUNT=0
while IFS='|' read -r family artifact golden target marker; do
    check_slice "$family" "$artifact" "$golden" "$target" "$marker"
    SLICE_COUNT=$((SLICE_COUNT + 1))
done < <(parse_manifest)

if [[ $SLICE_COUNT -eq 0 ]]; then
    echo "WARNING: no slices found in manifest. Nothing to check."
    exit 0
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ($CHECKED checked) ==="

if [[ $FAIL -gt 0 ]]; then
    echo ""
    echo "INTEGRATION DRIFT DETECTED."
    echo "To fix: regenerate from spec, compare with golden, update target."
    echo "  cd codegen && go run . check-all"
    exit 1
fi

echo "All integrated slices match golden snapshots."
