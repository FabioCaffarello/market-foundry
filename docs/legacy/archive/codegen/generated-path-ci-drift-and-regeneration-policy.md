# Generated Path: CI, Drift Detection, and Regeneration Policy

> **Stage:** S199
> **Purpose:** Define how CI validates generated artifacts, detects drift, and governs regeneration
> **Prerequisite:** CI already runs `codegen-check` and `codegen-test` (validated in S196)

---

## 1. CI Integration Model

### 1.1 Principle: CI Verifies, Never Generates

Generated files are **committed to the repository**. CI does not run the codegen engine to produce output for deployment — it runs the engine to **verify** that committed output matches what the engine would produce. This ensures:

- No build-time codegen dependency
- Deterministic builds (what's committed is what's deployed)
- Generated output is always reviewable in PRs
- CI failure means drift, not missing generation step

### 1.2 Current CI Jobs

| Job | Trigger | What It Validates | Duration | Blocks Merge |
|-----|---------|-------------------|----------|-------------|
| `codegen-golden` | Push/PR to main | All golden snapshots match regenerated output | ~10s | YES |
| `codegen-test` | Push/PR to main | Codegen engine unit tests pass | ~5s | YES |
| `unit-tests` | Push/PR to main | All Go unit tests across modules | ~30s | YES |
| `smoke-analytical` | Push/PR to main (after unit-tests) | End-to-end NATS→writer→ClickHouse→reader→HTTP | ~5min | YES |

### 1.3 CI Validation Chain for Generated Families

When a PR adds a codegen-first family, CI validates in this order:

```
codegen-golden          → generated output = golden snapshots (structural)
codegen-test            → engine logic is correct
unit-tests              → generated + manual code compiles and passes tests
smoke-analytical        → family flows end-to-end through the runtime
```

All four must pass. A failure at any stage blocks merge.

---

## 2. Drift Detection

### 2.1 What Is Drift?

Drift occurs when the output that the codegen engine would produce diverges from what is committed in the repository. There are three drift scenarios:

| Scenario | Cause | Detection |
|----------|-------|-----------|
| **Spec drift** | Spec changed but golden snapshot / target file not regenerated | `codegen-golden` fails |
| **Template drift** | Template changed but golden snapshots not refreshed | `codegen-golden` fails |
| **Manual drift** | Developer hand-edited a generated fragment in a target file | `codegen-golden` passes (golden is correct), but target file diverges — detected only by review |

### 2.2 Current Detection Coverage

| Drift Type | Detected By | Automated? | Status |
|------------|-------------|-----------|--------|
| Spec ↔ Golden | `codegen-golden` (`check-all` command) | YES | Active (S196) |
| Template ↔ Golden | `codegen-golden` | YES | Active (S196) |
| Golden ↔ Target file | Not yet implemented | NO | Deferred (see 2.3) |
| Cross-spec uniqueness | Not yet implemented | NO | Deferred (trigger: >10 specs) |

### 2.3 Deferred: Golden-to-Target Drift Detection

The current CI validates that **golden snapshots match regenerated output** but does NOT validate that **target files contain the correct generated fragments**. This gap exists because:

- File integration is manual (copy-paste)
- No marker sections exist to delimit generated regions in target files
- Parsing target files to extract generated fragments is fragile without markers

**Mitigation in this phase:**
- PR review checklist requires reviewer to verify generated fragments match golden snapshots
- Compilation + unit tests + smoke tests catch most wiring errors
- Single-family iteration limits blast radius

**Planned resolution:**
- When marker-based file integration is implemented (future stage), CI can extract marked regions and compare against golden snapshots
- This closes the golden-to-target drift gap automatically

### 2.4 Structural vs Cosmetic Drift

The comparison engine (`compare.go`) performs **structural normalization** before comparison:

1. Strip single-line comments (`// ...`)
2. Normalize tabs to spaces
3. Trim whitespace per line
4. Remove empty lines

This means the following differences are **allowed** (cosmetic):
- Comment text changes
- Whitespace/indentation variations
- Blank line additions/removals

The following differences are **forbidden** (structural):
- Missing or extra fields
- Different function names
- Different SQL shapes
- Different import paths
- Different type assertions
- Different validation logic

---

## 3. Regeneration Policy

### 3.1 When to Regenerate

| Event | Regeneration Required? | Scope |
|-------|----------------------|-------|
| New family spec added | YES | New family only |
| Existing spec modified | YES | Modified family only |
| Template modified | YES (all families) | All golden snapshots + all target file fragments |
| Derivation logic changed | YES (all families) | All golden snapshots + all target file fragments |
| Manual code around fragment changed | NO | Fragment itself unchanged |
| Dependency update (go.mod) | NO | Codegen has no runtime deps on target modules |

### 3.2 Regeneration Procedure

**For a single family (spec change):**
```bash
# 1. Regenerate golden snapshots
cd codegen
go run . generate families/{family}.yaml consumer_spec > golden-snapshots/{family}/consumer_spec.go.golden
go run . generate families/{family}.yaml pipeline_entry > golden-snapshots/{family}/pipeline_entry.go.golden

# 2. Verify all families still pass
go run . check-all

# 3. Update target file fragments (if the family is already integrated)
go run . generate families/{family}.yaml consumer_spec   # copy to registry file
go run . generate families/{family}.yaml pipeline_entry  # copy to pipeline.go

# 4. Run full validation
cd .. && make codegen-check && make codegen-test && make test
```

**For all families (template or derivation change):**
```bash
# 1. Regenerate ALL golden snapshots
cd codegen
for spec in families/*.yaml; do
  family=$(basename "$spec" .yaml)
  go run . generate "$spec" consumer_spec > "golden-snapshots/$family/consumer_spec.go.golden"
  go run . generate "$spec" pipeline_entry > "golden-snapshots/$family/pipeline_entry.go.golden"
done

# 2. Verify
go run . check-all

# 3. Update ALL target file fragments for codegen-first families
# (manual families do not get updated — they remain hand-crafted)

# 4. Full validation
cd .. && make codegen-check && make codegen-test && make test
```

### 3.3 Regeneration Rules

1. **Never partially regenerate.** If a template changes, ALL golden snapshots must be refreshed — not just the ones that look different.
2. **Golden snapshots and target file fragments must be updated in the same PR.** Split updates create drift windows.
3. **Regeneration does not bypass review.** Every regenerated artifact goes through standard PR review.
4. **Manual families are never regenerated.** The 6 existing families are golden references, not codegen targets. Their golden snapshots reflect what the engine *should* produce, not what should be deployed.
5. **Regeneration is idempotent.** Running the engine twice with the same spec and template produces identical output. If it doesn't, that's a bug.

---

## 4. CI Failure Response

### 4.1 `codegen-golden` Failure

**Meaning:** Regenerated output does not match committed golden snapshots.

**Diagnosis:**
```bash
cd codegen && go run . check-all
# Shows which families and artifacts have diffs
```

**Common causes:**
1. Spec changed but golden not refreshed → regenerate golden snapshots
2. Template changed but goldens not refreshed → regenerate all golden snapshots
3. Derivation logic changed → regenerate all golden snapshots
4. Merge conflict in golden file → resolve conflict, regenerate

**Resolution:** Always regenerate from source (spec + template), never hand-edit golden files.

### 4.2 `codegen-test` Failure

**Meaning:** Codegen engine unit tests fail.

**Diagnosis:**
```bash
cd codegen && go test ./... -v
```

**Common causes:**
1. Derivation logic change broke naming conventions → fix logic or update tests
2. Spec validation rule change → update validation tests
3. Comparison logic change → update comparison tests

### 4.3 `unit-tests` Failure (After Adding Generated Family)

**Meaning:** Generated fragment doesn't compile or wiring is wrong.

**Diagnosis:**
```bash
make test 2>&1 | head -50
```

**Common causes:**
1. Fragment inserted in wrong location → move to correct position
2. Import missing in target file → add required import
3. Mapper function referenced in spec doesn't exist → author mapper (A3)
4. Domain type referenced doesn't exist → author domain type

**If generated fragment itself is wrong** (function name, types, SQL): this is a revocation trigger. The fix must come from spec/template/derivation, not manual edit.

### 4.4 `smoke-analytical` Failure (After Adding Generated Family)

**Meaning:** Family doesn't flow end-to-end through the runtime.

**Diagnosis:**
```bash
make logs-writer  # check writer startup and consumption
make logs-gateway # check gateway reader initialization
```

**Common causes:**
1. Config not updated (A5) → add config entry to writer.jsonc
2. Smoke test not updated (A6) → add family phase to smoke script
3. Migration missing → add ClickHouse DDL
4. NATS stream not configured → verify stream exists

---

## 5. Policy Summary

| Policy | Rule |
|--------|------|
| **CI generates?** | No — CI verifies committed output |
| **Golden snapshots committed?** | Yes — always in repo |
| **Generated output committed?** | Yes — in target files |
| **Manual edit of golden?** | Never — always regenerate |
| **Manual edit of generated fragment?** | Never — revocation trigger |
| **Template change scope?** | Requires new stage; all goldens refreshed |
| **Spec change scope?** | Affected family only; golden + target refreshed |
| **Drift detection gap?** | Golden-to-target not yet automated (review-based) |
| **Regeneration idempotency?** | Required — non-idempotent output is a bug |
| **Partial regeneration?** | Forbidden for template/derivation changes |

---

## 6. Future CI Enhancements (Not in This Phase)

| Enhancement | Trigger | Priority |
|-------------|---------|----------|
| Golden-to-target comparison via markers | Marker section implementation | HIGH |
| Cross-spec uniqueness validation | Spec count > 10 | MEDIUM |
| Automated regeneration helper (`make codegen-regen`) | Frequent regeneration cycles | LOW |
| Tier 2 golden coverage | Tier 2 authorization | NOT SCHEDULED |

These are documented for future reference. None are authorized or required for S199/S200.
