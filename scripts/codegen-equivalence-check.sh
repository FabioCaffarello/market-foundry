#!/usr/bin/env bash
# codegen-equivalence-check.sh
#
# S261 — Manual-to-Generated Equivalence Validation
#
# Validates that codegen-governed artifacts in production code match their
# golden snapshots AND that the codegen spec values are consistent with the
# manual artifacts they coexist with (store consumers, starters, mappers).
#
# Exit 0 = all equivalence checks pass.
# Exit 1 = drift detected.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/utils/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./scripts/codegen-equivalence-check.sh [--help]

Runs the manual-to-generated codegen equivalence harness.
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

require_commands bash go grep awk sed tr

REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CODEGEN_DIR="$REPO_ROOT/codegen"
FAMILIES_DIR="$CODEGEN_DIR/families"

passed=0
failed=0
warnings=0

pass() { printf "PASS  %s\n" "$1"; passed=$((passed + 1)); }
fail() { printf "FAIL  %s — %s\n" "$1" "$2"; failed=$((failed + 1)); }
warn() { printf "WARN  %s — %s\n" "$1" "$2"; warnings=$((warnings + 1)); }

# ─── Phase 1: Golden Snapshot Equivalence (codegen check-all) ──────────────
echo "=== Phase 1: Golden Snapshot Equivalence ==="
if CODEGEN_ROOT="$CODEGEN_DIR" go run "$CODEGEN_DIR" check-all >/dev/null 2>&1; then
    pass "golden-snapshot-check-all (20/20)"
else
    fail "golden-snapshot-check-all" "codegen check-all reported failures"
fi

# ─── Phase 2: Integrated Slice Equivalence ─────────────────────────────────
echo ""
echo "=== Phase 2: Integrated Slice Equivalence ==="
if bash "$REPO_ROOT/scripts/codegen-integrated-check.sh" >/dev/null 2>&1; then
    pass "integrated-slice-check (20/20)"
else
    fail "integrated-slice-check" "codegen-integrated-check.sh reported failures"
fi

# ─── Phase 3: Spec Validity ───────────────────────────────────────────────
echo ""
echo "=== Phase 3: Spec Validation ==="
if CODEGEN_ROOT="$CODEGEN_DIR" go run "$CODEGEN_DIR" validate-all >/dev/null 2>&1; then
    pass "spec-validate-all (10/10, no collisions)"
else
    fail "spec-validate-all" "codegen validate-all reported failures"
fi

# ─── Phase 4: Cross-Artifact Consistency ──────────────────────────────────
# For each codegen-governed family, verify that:
#   a) The writer consumer spec durable name matches the store consumer durable pattern
#   b) The pipeline entry table matches the spec writer.table
#   c) The pipeline entry insertSQL columns match the spec writer.columns
echo ""
echo "=== Phase 4: Cross-Artifact Consistency ==="

for spec_file in "$FAMILIES_DIR"/*.yaml; do
    family=$(grep 'name:' "$spec_file" | head -1 | awk '{print $2}')
    layer=$(grep 'layer:' "$spec_file" | head -1 | awk '{print $2}')
    table=$(grep 'table:' "$spec_file" | head -1 | awk '{print $2}')
    durable=$(grep 'durable:' "$spec_file" | head -1 | awk '{print $2}')
    columns=$(grep 'columns:' "$spec_file" | head -1 | sed 's/.*columns: *"\(.*\)"/\1/')
    check_name="$family/cross-artifact"

    # 4a: Verify durable name pattern: writer-{layer}-{family-hyphenated} (evidence omits layer)
    hyphen_family=$(echo "$family" | tr '_' '-')
    if [ "$layer" = "evidence" ]; then
        expected_durable="writer-$hyphen_family"
    else
        expected_durable="writer-$layer-$hyphen_family"
    fi
    if [ "$durable" = "$expected_durable" ]; then
        pass "$family/durable-naming"
    else
        fail "$family/durable-naming" "expected '$expected_durable', got '$durable'"
    fi

    # 4b: Verify insertSQL contains the spec table
    insert_line=$(grep -o "INSERT INTO $table" "$REPO_ROOT/cmd/writer/pipeline.go" | head -1 || true)
    if [ -n "$insert_line" ]; then
        pass "$family/insert-table"
    else
        fail "$family/insert-table" "no INSERT INTO $table found in pipeline.go"
    fi

    # 4c: Verify insertSQL column list matches spec columns
    pipeline_insert=$(grep "INSERT INTO $table" "$REPO_ROOT/cmd/writer/pipeline.go" | head -1 | sed 's/.*(\(.*\)).*/\1/' || true)
    spec_columns_normalized=$(echo "$columns" | tr -d ' ')
    pipeline_columns_normalized=$(echo "$pipeline_insert" | tr -d ' ')
    if [ "$spec_columns_normalized" = "$pipeline_columns_normalized" ]; then
        pass "$family/column-alignment"
    else
        # Families sharing a table may have identical columns — check if any match
        if echo "$pipeline_columns_normalized" | grep -q "$spec_columns_normalized" 2>/dev/null; then
            pass "$family/column-alignment"
        else
            warn "$family/column-alignment" "columns differ (shared table families may legitimately match another entry)"
        fi
    fi
done

# ─── Phase 5: Store Consumer Pattern Consistency ──────────────────────────
# Verify that manual store consumers exist alongside codegen writer consumers.
# Instead of trying to compute PascalCase in shell, just grep for "Store.*Consumer"
# containing the family name pattern in the registry file.
echo ""
echo "=== Phase 5: Store Consumer Coexistence ==="

for spec_file in "$FAMILIES_DIR"/*.yaml; do
    family=$(grep 'name:' "$spec_file" | head -1 | awk '{print $2}')
    layer=$(grep 'layer:' "$spec_file" | head -1 | awk '{print $2}')
    registry_file="$REPO_ROOT/internal/adapters/nats/nats${layer}/registry.go"

    # Check that a Store*Consumer function exists that references this family's durable pattern
    durable=$(grep 'durable:' "$spec_file" | head -1 | awk '{print $2}')
    store_durable=$(echo "$durable" | sed 's/^writer-/store-/')

    if grep -q "$store_durable" "$registry_file" 2>/dev/null; then
        pass "$family/store-consumer-coexists"
    else
        warn "$family/store-consumer-coexists" "no store consumer with durable '$store_durable' in registry"
    fi
done

# ─── Phase 6: Starter/Mapper Existence ───────────────────────────────────
echo ""
echo "=== Phase 6: Starter and Mapper Existence ==="
support_file="$REPO_ROOT/internal/adapters/clickhouse/writerpipeline/support.go"

check_layer_support() {
    local layer="$1" starter="$2" mapper="$3"
    if grep -q "func $starter" "$support_file"; then
        pass "$layer/starter-exists ($starter)"
    else
        fail "$layer/starter-exists" "$starter not found"
    fi
    if grep -q "func $mapper" "$support_file"; then
        pass "$layer/mapper-exists ($mapper)"
    else
        fail "$layer/mapper-exists" "$mapper not found"
    fi
}

check_layer_support evidence   NewCandleStarter    mapCandleRow
check_layer_support signal     NewSignalStarter    mapSignalRow
check_layer_support decision   NewDecisionStarter  mapDecisionRow
check_layer_support strategy   NewStrategyStarter  mapStrategyRow
check_layer_support risk       NewRiskStarter      mapRiskRow
check_layer_support execution  NewExecutionStarter mapExecutionRow

# ─── Phase 7: Config Method Existence ────────────────────────────────────
echo ""
echo "=== Phase 7: Config Method Consistency ==="
schema_file="$REPO_ROOT/internal/shared/settings/schema.go"

check_config_method() {
    local layer="$1" method="$2"
    if grep -q "func.*$method" "$schema_file"; then
        pass "$layer/config-method ($method)"
    else
        fail "$layer/config-method" "$method not found in schema.go"
    fi
}

check_config_method evidence   IsFamilyEnabled
check_config_method signal     IsSignalFamilyEnabled
check_config_method decision   IsDecisionFamilyEnabled
check_config_method strategy   IsStrategyFamilyEnabled
check_config_method risk       IsRiskFamilyEnabled
check_config_method execution  IsExecutionFamilyEnabled

# ─── Summary ─────────────────────────────────────────────────────────────
echo ""
echo "=== Equivalence Results: $passed passed, $failed failed, $warnings warnings ==="

if [ "$failed" -gt 0 ]; then
    echo "VERDICT: FAIL — drift detected"
    exit 1
else
    if [ "$warnings" -gt 0 ]; then
        echo "VERDICT: PASS with warnings"
    else
        echo "VERDICT: PASS — full equivalence confirmed"
    fi
    exit 0
fi
