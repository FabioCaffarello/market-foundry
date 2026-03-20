# Next Wave Recommendations After Pre-Generated Family Gate

## Context

The S198 gate reviewed codegen readiness and issued a conditional PASS for the first generated family (A1+A2 only, single family, existing layer). This document defines what should happen next, in what order, and what must not happen yet.

## Recommended Sequence

### Wave 1: First Generated Family (Immediate — Next Stage)

**Objective**: Implement one new analytical family where A1 (consumer spec) and A2 (pipeline entry) are generated from spec, and A3–A6 are hand-crafted.

**Scope**:
- Select family using S197 candidate criteria (existing layer, existing table, named mapper, complexity within validated range)
- Author YAML spec, generate A1+A2, create golden snapshots
- Hand-craft mapper (A3), mapper tests (A4), config entry (A5), smoke phase (A6)
- Manually integrate generated fragments into source files
- Validate against all 7 success criteria (S197)
- Measure time savings vs manual baseline

**Deliverables**:
- New family YAML spec in `codegen/families/`
- Golden snapshots in `codegen/golden-snapshots/{family}/`
- All Tier 1 write-path artifacts (A1–A6) integrated
- Validation report documenting findings, friction, and time savings

**Exit condition**: All 7 success criteria pass. No template modifications were required. Generated code was inserted verbatim.

### Wave 2: First Generated Family Findings Gate (After Wave 1)

**Objective**: Review findings from the first generated family and decide whether to authorize a second generated family or to harden first.

**Scope**:
- Were generated A1+A2 inserted verbatim, or were adjustments needed?
- Did the family target an existing layer successfully, or were infrastructure gaps found?
- Was time savings as expected (~15 min / ~23%)?
- Did any template or normalization issues surface?
- Is the spec authoring experience acceptable?

**Decision tree**:
- If verbatim insertion + all criteria pass → authorize second generated family
- If minor friction found → address friction, then authorize second family
- If template modification was required → pause expansion, investigate model stability
- If generated code required manual edits → revoke codegen authorization for that artifact

### Wave 3: Second Generated Family (After Wave 2 Gate)

**Objective**: Validate that the generation model is not a single-family accident.

**Scope**: Same as Wave 1 but with a second family targeting a different layer. The two generated families should together cover at least 2 distinct layers.

**Exit condition**: Both generated families pass all success criteria. No template modifications across both families.

### Wave 4: Mapper Generation Evaluation (After Wave 3)

**Objective**: Decide whether A3 (mapper function) generation justifies the engineering investment.

**Scope**:
- Design `domain.columns` spec extension
- Define column-order DDL validation strategy
- Establish mapper equivalence rules (structural + semantic, since mappers involve transform functions)
- Prototype mapper template against 2 existing families (RSI bracket)
- Golden comparison validation

**Decision tree**:
- If mapper equivalence is achievable with reasonable spec extension → authorize A3 generation
- If mapper patterns are too diverse for template expansion → keep A3 manual permanently
- If column-order DDL validation is too fragile → defer until DDL tooling matures

**This wave should NOT be rushed.** Mapper generation is the most complex Tier 1 artifact. Getting it wrong creates maintenance burden worse than manual authoring.

### Wave 5: File Integration Evaluation (After Wave 3 or Wave 4)

**Objective**: Decide whether marker-section file integration justifies its complexity.

**Scope**:
- Define marker section syntax (`// codegen:begin:{artifact}` / `// codegen:end:{artifact}`)
- Implement file reader/writer with section replacement
- CI drift detection: regenerate → diff against committed files
- Test with existing 6+ families

**Trigger**: Only if manual integration friction exceeds ~10 minutes per family or if error rate is non-zero after 2+ generated families.

## What Must NOT Happen Yet

| Action | Why Not | Earliest Trigger |
|--------|---------|------------------|
| Tier 2 (read-path) generation | Tier 1 not yet proven in production | After ≥2 generated families validated |
| Multi-family parallel generation | Single-family iteration discipline required | After ≥3 successful single-family iterations |
| Template refactoring | Templates must be stable during first family | After first generated family gate (Wave 2) |
| Spec schema extension (`domain.columns`) | Frozen spec; requires dedicated gate | Wave 4 mapper evaluation |
| Generic codegen framework | Anti-pattern per S193 | NEVER |
| Automatic CI generation (CI generates, not verifies) | Violates S193 principle | NEVER |
| Generating domain types, NATS streams, or shared infrastructure | Permanently excluded per S193 | NEVER |

## Risk Watchlist

### R1: Codegen enthusiasm exceeding evidence

The gate passed. The temptation is to accelerate. The discipline is: one family at a time, validate, gate, then decide.

**Mitigation**: Each wave has an explicit exit condition and decision tree. No wave authorizes the next without evidence.

### R2: Template instability during expansion

If the first generated family requires a template change, it means the model is not stable. This is the strongest signal to pause expansion.

**Mitigation**: Templates are frozen during Wave 1 (S197-D5). Any template change revokes the current authorization and requires a new gate.

### R3: Golden snapshot fatigue

As family count grows, golden snapshot creation and review becomes routine. Routine leads to rubber-stamping.

**Mitigation**: `TestCheckAllFamilies` in CI is the automated guard. Review discipline is the human guard. Snapshot count is manageable at current scale (~14 files for 7 families).

### R4: Manual artifact drift from generated artifacts

A3–A6 are hand-crafted. They may drift from patterns established by generated A1+A2. Over time, manual artifacts may diverge in style or convention.

**Mitigation**: This is acceptable. A3–A6 involve creative decisions (transform functions, test strategies, config structure) that inherently vary. The codegen boundary exists precisely because A1+A2 are mechanical and A3–A6 are not.

## Summary

The path forward is: implement one generated family → validate → gate → implement second → validate → gate → evaluate mapper generation. Each step produces evidence. Each gate uses that evidence. No step is pre-authorized by the previous one.
