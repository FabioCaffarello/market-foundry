# First Generated Family — Risks, Success Criteria, and Non-Goals

**Stage**: S197
**Status**: DEFINITIVE
**Date**: 2026-03-20
**Context**: Defines the risk model, success criteria, and explicit non-goals for the first family produced using the codegen engine.

---

## 1. Risks

### R1: Fragment Integration Error (Medium)

**Description**: Generated A1 and A2 are code fragments, not complete files. Manual insertion into registry and pipeline files introduces risk of misplacement, missing imports, or incorrect ordering.

**Likelihood**: Medium — the 6 existing families establish a clear pattern, but the manual step has no automated validation beyond compilation.

**Impact**: Low to Medium — compilation would catch syntax errors; unit tests would catch behavioral errors. But silent misplacement (e.g., wrong position in pipeline array) could cause subtle ordering issues.

**Mitigation**:
- Review generated fragments against existing patterns before insertion
- Run full test suite after integration
- Future: implement marker-section file integration to eliminate this risk

### R2: Spec Authoring Error (Low)

**Description**: The YAML spec for the new family could contain incorrect values (wrong NATS subject, wrong table name, wrong event type).

**Likelihood**: Low — spec validation catches structural errors, and golden comparison catches naming errors. But semantic errors (e.g., correct format but wrong value) are not detectable by codegen.

**Impact**: Medium — wrong NATS subject would cause silent message loss; wrong table would cause insertion failure.

**Mitigation**:
- Validate spec values against actual NATS configuration and ClickHouse schema before generating
- Smoke test validates end-to-end message flow
- Code review catches value-level errors

### R3: Overconfidence from A1+A2 Success (Medium)

**Description**: The 12/12 golden pass rate for A1+A2 may create pressure to expand codegen to A3–A6 before those artifacts are validated to the same standard.

**Likelihood**: Medium — natural tendency to extrapolate success.

**Impact**: High — premature expansion could produce incorrect mappers (column-order errors affect data integrity) or incorrect config entries (silent family disablement).

**Mitigation**:
- This document explicitly limits authorization to A1+A2
- Each artifact type requires its own equivalence validation stage before authorization
- S197 decision is non-transferable to other artifacts

### R4: Golden Snapshot Extraction Error (Low)

**Description**: For the first generated family, golden snapshots are created from generated output (not extracted from existing code, since the family is new). If the generated output has a latent bug that happens to match itself, the golden test becomes tautological.

**Likelihood**: Low — A1 and A2 templates have been validated against 6 independent hand-crafted implementations. A latent bug would have to affect only new families, not existing ones.

**Impact**: Low — even if golden comparison is tautological, compilation, unit tests, and smoke tests provide independent validation.

**Mitigation**:
- Manual review of generated A1 and A2 output before creating golden snapshots
- Cross-reference against the nearest existing family in the same layer
- Smoke test validates runtime behavior independent of golden comparison

### R5: Evidence Layer Exception Misapplication (Low)

**Description**: If the first generated family happens to be in the evidence layer, the naming exceptions (omit layer from function/consumer names) must apply correctly. If it's a non-evidence family in a layer with only one existing family, the pattern may not be fully stress-tested.

**Likelihood**: Low — all 6 layers have been validated in S196.

**Impact**: Low — naming errors cause compilation failure (function not found) which is caught immediately.

**Mitigation**: No additional mitigation needed. S196 validated all 6 layers.

---

## 2. Success Criteria

The first generated family is considered successful if and only if ALL of the following are met:

### SC1: Golden Comparison Passes

The new family's YAML spec, when processed by the codegen engine, produces A1 and A2 outputs that pass golden comparison using S194 normalization rules. CI `codegen-golden` job passes with the new family included.

### SC2: Generated Code Compiles

The generated A1 and A2 fragments, once integrated into their target files, compile without errors as part of the full `go build ./...`.

### SC3: Unit Tests Pass

All existing unit tests continue to pass. New mapper tests (A4, hand-crafted) pass. Codegen unit tests (26+) pass including the new family's golden tests.

### SC4: Smoke Test Passes

The hand-crafted smoke test phase (A6) for the new family validates end-to-end: publish event → NATS → writer consumer → ClickHouse insert → query returns row.

### SC5: No Manual Edits to Generated Code

The A1 and A2 fragments are inserted verbatim into source files. No manual adjustments to the generated code are required. If adjustments are needed, this constitutes a template bug that must be fixed in the template (not in the generated output).

### SC6: Time Savings Measured

Actual time spent on the new family is measured and compared against the ~65 min baseline for a fully manual family. Expected savings: ~15 min (23%). Any significant deviation (positive or negative) is documented for future reference.

### SC7: No Codegen Scope Creep

The generation stays within A1+A2. No additional artifacts are generated, even if they seem "easy" or "obvious." A3–A6 remain manual. This criterion ensures the first iteration validates the authorized boundary, not a wider one.

---

## 3. Non-Goals

### NG1: Full Family Generation

The first generated family is NOT fully generated. 8+ artifacts remain manual. This is by design — the goal is to validate the A1+A2 generation model in a production-like context, not to eliminate all manual work.

### NG2: Codegen Coverage Expansion

S197 does NOT authorize expanding codegen to A3–A6 or Tier 2. Each artifact type requires its own validation stage. The first generated family is a proof point for the current slice, not a trigger for expansion.

### NG3: Automatic File Integration

Generated code is inserted manually. Marker-section automation is a future enhancement, not a prerequisite for the first generated family.

### NG4: Multi-Family Generation

Only one family is generated. Batch generation of multiple families is not authorized until the single-family workflow is validated.

### NG5: New Infrastructure

The first generated family must not require new NATS streams, new ClickHouse tables (unless via committed migration), or new adapter packages. It must fit within existing infrastructure.

### NG6: Performance Benchmarking

The first generated family is not a performance test. Writer throughput, ClickHouse insert latency, and query performance are not in scope for this decision.

### NG7: Template Refactoring

Templates must not be refactored or "improved" during the first generated family. They are frozen at S195/S196 state. Any template issues discovered become inputs to a future hardening stage.

---

## 4. Failure Modes and Response Protocol

### FM1: Golden Comparison Fails for New Family

**Response**: Do NOT force-fix the golden snapshot. Investigate whether the template has a family-specific bug or the spec has incorrect values. Fix the root cause (template or spec), re-generate, re-compare.

### FM2: Generated Code Requires Manual Edits

**Response**: This is a template bug. Document the required edit, fix the template, verify the fix doesn't break existing 6 families (golden regression), then re-generate. The generated family must use the fixed template output verbatim.

### FM3: Smoke Test Fails

**Response**: Investigate whether the failure is in generated code (A1/A2), manual code (A3–A6), or infrastructure. If A1/A2 are correct but A3+ have bugs, codegen is not at fault — fix the manual artifacts. If A1/A2 contributed to the failure, treat as FM2.

### FM4: Integration Conflicts

**Response**: If inserting A1/A2 fragments into existing files causes conflicts (e.g., import collisions, ordering issues), document the friction and add it as input for the marker-section file integration feature.

### FM5: Time Savings Not Realized

**Response**: If the new family takes equal or more time than a fully manual family, document why. Possible causes: spec authoring friction, golden snapshot creation overhead, integration complexity. These become inputs for deciding whether to continue with codegen or invest in hardening first.

---

## 5. Open Questions for S198

1. **Which family?** — S197 does not select the specific family. S198 must choose a candidate that meets all criteria in Section 3 of the scope decision document.
2. **New family or second instance in existing layer?** — Adding a second signal family (e.g., EMA) exercises the "same layer, different family" pattern. Adding a family in a layer with only one existing family (e.g., a second evidence family) exercises layer infrastructure reuse.
3. **Who measures time?** — SC6 requires timing. The implementer should track time spent on each artifact category.
4. **What if the family needs a new table?** — If the best candidate requires a new migration, that's acceptable (migrations are manual and already proven), but increases scope. Prefer candidates that write to existing tables.
