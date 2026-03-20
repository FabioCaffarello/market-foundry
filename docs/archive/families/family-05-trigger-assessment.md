# Family 05 Trigger Assessment

## Purpose

Formal trigger assessment after Family 04, evaluating whether the analytical layer can continue expanding to Family 05 (Executions) or whether accumulated friction demands a new hardening tranche first.

## Context

Family 04 (Risk Assessments) was designed as the **ceiling test** for the Wave B pattern — highest column count (17 DDL), most JSON columns (4), first free-text column (`rationale`), and a new parser shape (`ParseConstraintsJSON` struct-target). S182 confirmed all ceiling tests passed mechanically with zero new frictions and zero creative decisions.

This assessment determines whether that success extends the pattern's runway to Family 05 or whether the pattern has reached its manual-expansion limit.

---

## Trigger Evaluation Matrix

### T-CG: Codegen Necessity

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **ACTIVATED — non-blocking for Family 05, mandatory before Family 06** |
| Duplication | ~800 lines across 5 readers (~143 LOC each, 80% identical), 5 handlers (~100 LOC each, 85% identical), 5 use cases (~128 LOC each, 70% identical) |
| Growth per family | ~595 LOC implementation + ~707 LOC tests |
| Current total | ~3,348 LOC analytical layer |
| Projected at Family 05 | ~3,943 LOC |
| Projected at Family 06 | ~4,538 LOC — exceeds manual maintenance threshold |
| Threshold | Template maintenance cost < duplication cost at ≤5 families. Equation inverts at 6+. |
| **Assessment** | Family 05 is the last family where manual copy-paste-modify remains cheaper than codegen. **Family 06 requires codegen.** |

### T-HS: Handler File Size

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **APPROACHING THRESHOLD — non-blocking for Family 05** |
| Current | 515 lines (5 methods) |
| Projected Family 05 | ~595–615 lines |
| Healthy threshold | <550 lines |
| Concerning threshold | 550–600 lines |
| Critical threshold | >600 lines |
| **Assessment** | Family 05 pushes the handler into "concerning" territory but likely stays under 600. DEF-C3 (handler split) becomes **mandatory before Family 06**. A `parseAnalyticalParams()` helper extraction could reduce per-method duplication from ~90 to ~30 lines, buying runway, but is not required for Family 05. |

### T-SM: Smoke Test Scalability

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **ESCALATING — non-blocking** |
| Current | ~606–750 lines (depends on measurement point) |
| Growth per family | ~28–30 lines (validation call + filter checks) |
| Projected Family 05 | ~635–780 lines |
| Mitigation | `validate_analytical_family()` helper absorbs additions mechanically |
| **Assessment** | Smoke test growth is linear and manageable. The helper function keeps per-family additions to ~5–8 lines of new validation invocations. Restructuring (per-family functions, separate files) recommended at Family 07+ but not required for Family 05. |

### T-CI: CI Integration

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **RESOLVED** |
| Evidence | CI workflow includes `smoke-analytical` job validating all families E2E |
| History | PF-4 carried forward through 4 families; resolved at S166/S172, documentation lagged |
| **Assessment** | No trigger. CI is operational and absorbs new families automatically through smoke test extension. |

### T-SC: Schema Coherence

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **NOT TRIGGERED** |
| Current state | 6 migrations (+ metadata), 5 active families, ~75 DDL columns total |
| Verification | Review-enforced, smoke-tested, unit-tested per column |
| Family 04 result | 17/17 columns verified DDL → mapper → reader alignment, zero coherence failures |
| Threshold | Compile-time checks recommended at ~12 tables / 100+ columns |
| **Assessment** | At 6 tables (post-Family 05), still well under the threshold. Manual verification remains reliable. Schema coherence tooling deferred to Family 08+. |

### T-JP: JSON Parser Count

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **AT LIMIT — non-blocking** |
| Current | 6 parsers (ParseMetadataJSON ×6 reuse, ParseSignalInputsJSON, ParseDecisionInputsJSON, ParseStrategyInputsJSON, ParseConstraintsJSON, FormatFloat) |
| Healthy threshold | ≤6 |
| Concerning threshold | 7–8 |
| Family 05 (Executions) | May not require new parsers — execution schema likely reuses existing patterns (metadata, float formatting) |
| **Assessment** | Parser count at healthy upper limit. If Family 05 requires 0–1 new parsers, remains manageable. Generic `parseJSON[T any]` deferred unless count exceeds 8. |

### T-FC: Friction Count

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **NOT TRIGGERED** |
| Family 04 new frictions | 0 (zero) |
| Threshold | >2 new frictions per family expansion |
| Cumulative active frictions | 7 (PF-1 through PF-7), none high-severity |
| **Assessment** | Family 04 produced zero new frictions. Pattern health at its peak. Friction accumulation remains linear and cosmetic. |

### T-MR: Mapper/Reader/Gateway Complexity

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **NOT TRIGGERED** |
| Reader growth | 143 LOC average per reader, linear, self-contained |
| Mapper growth | ~35 LOC per family, pre-staged, zero write-path changes (5th consecutive) |
| Gateway composition | ~8 LOC per family, struct DI additive only |
| Cross-reader dependencies | Zero |
| **Assessment** | All three layers scale linearly with zero coupling. No architectural pressure. |

### T-GE: Governance and Expansion Control

| Aspect | Evidence | Verdict |
|--------|----------|---------|
| Status | **HEALTHY** |
| Gate enforcement | S179 gate authorized exactly one family (Family 04), required new gate for Family 05 |
| Pattern version | v2 (9-artifact template, struct DI, smoke helper, canonical naming) |
| Trigger assessment cadence | Every family since S167 |
| **Assessment** | Governance model is working. Each family requires explicit authorization. No automatic expansion. |

---

## Consolidated Trigger Summary

| Trigger | Status | Blocks Family 05? | Action Required |
|---------|--------|-------------------|-----------------|
| T-CG: Codegen | Activated (non-blocking) | **No** | Mandatory before Family 06 |
| T-HS: Handler size | Approaching threshold | **No** | Split mandatory before Family 06 |
| T-SM: Smoke growth | Escalating | **No** | Restructure at Family 07+ |
| T-CI: CI integration | Resolved | **No** | None |
| T-SC: Schema coherence | Not triggered | **No** | Tooling at 12+ tables |
| T-JP: JSON parsers | At limit | **No** | Generic parser at count >8 |
| T-FC: Friction count | Not triggered | **No** | None |
| T-MR: Mapper/reader/gateway | Not triggered | **No** | None |
| T-GE: Governance | Healthy | **No** | Continue per-family gates |

---

## Verdict

**Family 05 (Executions) may proceed, subject to a formal gate.**

No trigger blocks Family 05. However, Family 05 is the **last family that can be expanded under the current manual pattern**. Three triggers converge at the Family 06 boundary:

1. **Codegen becomes mandatory** — duplication at ~4,500+ LOC makes template generation cheaper than copy-paste.
2. **Handler split becomes mandatory** — file at ~595–615 lines post-Family 05, exceeds 600-line threshold at Family 06.
3. **Smoke restructuring becomes advisable** — not mandatory but recommended before the 7th family.

### Binding Conditions for Family 05

1. Must follow Wave B v2 pattern (9-artifact template).
2. All existing constraints (C-1 through C-9) must be satisfied.
3. >2 new frictions in Family 05 → mandatory hardening before Family 06.
4. Handler file must remain ≤620 lines (hard ceiling).
5. Codegen tranche becomes mandatory before Family 06 begins.
6. Family 06 requires a new gate with codegen as prerequisite.

### What This Assessment Does NOT Authorize

- Implementation of Family 05.
- Codegen implementation during Family 05.
- Handler refactoring during Family 05.
- Automatic continuation to Family 06.
- Any architectural changes to the analytical layer.
