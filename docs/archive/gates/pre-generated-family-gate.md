# Pre-Generated Family Gate — Formal Readiness Review

## Purpose

This document is the formal gate review that decides whether market-foundry can transition from manual-first family expansion to a first codegen-first iteration. The decision is based exclusively on evidence from S193–S197, not on enthusiasm for automation.

## Gate Question

> Is the codegen engine mature enough, validated enough, and bounded enough to serve as the mechanism for expanding the first new analytical family?

## Evidence Inventory

| Stage | Deliverable | Status | Evidence Strength |
|-------|------------|--------|-------------------|
| S193 | Specification freeze | COMPLETE | 14-field schema frozen, ownership rules explicit, prohibited fields documented |
| S194 | Equivalence baseline | COMPLETE | RSI–Paper Order bracket, structural vs semantic rules defined, drift severity model |
| S195 | Minimal engine | COMPLETE | 2 templates, 6 source files, 4/4 golden passes, 17 unit tests |
| S196 | Cross-family validation | COMPLETE | 12/12 golden comparisons across all 6 families, 0 structural drift, 26 unit tests |
| S197 | Scope decision | COMPLETE | A1+A2 authorized, A3–A6 explicit manual, single family, existing layer |

## Formal Assessment: Five Gate Criteria

### 1. Is the spec mature enough?

**YES — with limits.**

The 14-field YAML schema is frozen (S193). Field semantics, validation invariants, and ownership rules are explicit. The spec covers Tier 1 within-layer expansion completely for A1 (consumer spec) and A2 (pipeline entry).

**Limit**: The spec does not yet support `domain.columns` (needed for A3 mapper generation). This is correctly deferred — it is not required for A1+A2.

**Limit**: Cross-spec uniqueness validation (when >6 specs exist) is not yet in CI. Currently enforced by convention and test coverage across 6 families. Acceptable for a single new family; becomes a debt at scale.

### 2. Is the equivalence with existing families strong enough?

**YES — proven across full coverage.**

S196 validated 12/12 golden comparisons across all 6 existing families, covering all 6 layers (evidence through execution). Zero structural drift was found. Three cosmetic drift instances (comment phrasing variations) were classified INFO severity and are normalized away by the comparison pipeline.

The equivalence bracket (RSI minimal → Paper Order ceiling) established in S194 was exceeded in S196 by validating against all 6 families, including edge cases: single-word names (candle), known abbreviations (RSI, RSIOversold), multi-word compound names (mean_reversion_entry, position_exposure), and the evidence-layer naming exception.

**Limit**: Equivalence is validated for A1+A2 only. Mapper (A3) equivalence requires its own validation stage with `domain.columns` support.

### 3. Does the narrow slice prove the generation model?

**YES — for the authorized scope.**

The engine produces structurally identical output to hand-crafted code for consumer specs and pipeline entries. The generation pipeline (YAML → derived fields → template rendering → structural normalization → golden comparison) works end-to-end. CI gates on regression.

**Limit**: The engine produces fragments, not integrated files. Generated A1 code must be manually inserted into `internal/adapters/nats/{domain}_registry.go` and A2 into `cmd/writer/pipeline.go`. Marker-section file integration is not implemented.

**Limit**: The engine covers 2 of 6 Tier 1 artifacts. The remaining 4 (mapper, mapper tests, config entry, smoke phase) require manual authoring for any new family.

### 4. Is the boundary between generated and manual clear?

**YES — explicitly defined.**

S197 established the tri-condition test: an artifact is generated IF AND ONLY IF it is (1) repetitive (3+ identical implementations), (2) mechanical (zero creative decisions), and (3) spec-derivable (every value from spec, no inference).

The ownership model is three-tier: HUMAN (never generated), CODEGEN (A1+A2 for Tier 1), SPEC (family definition YAML). Permanently excluded artifact types are documented in S193.

**Limit**: The boundary is clear in documentation but enforced only by review discipline and CI golden tests. There is no automated guard preventing someone from expanding codegen scope without a formal gate.

### 5. What is the acceptable next step?

**Option A: Implement the first generated family (A1+A2 only).**

This is the recommended path. The evidence supports it:
- 12/12 golden comparisons pass
- Zero structural drift
- CI gate operational
- Scope explicitly bounded (A1+A2, single family, existing layer)
- Risk model defined with mitigations (S197)
- Success criteria are measurable (7 criteria defined)

**Option B: Amplify the narrow slice before first family.**

Not justified. The narrow slice already covers all 6 families for A1+A2. Expanding to A3 (mapper) would require `domain.columns` spec extension, which is a separate engineering effort and should not block A1+A2 validation in production.

**Option C: Harden spec/validation before any generated family.**

Partially justified — but the required hardening items are additive, not blocking:
- Cross-spec uniqueness CI check: useful at scale, not critical for family #7
- Automated scope guard: useful long-term, not critical for single controlled expansion
- File integration markers: convenience, not correctness

These items should be tracked as debts, not gate blockers.

## Gate Verdict

**PASS — Conditional Authorization**

The first codegen-generated family is authorized under the following non-negotiable conditions:

1. **Scope**: A1 (consumer spec) + A2 (pipeline entry) only. A3–A6 remain manual.
2. **Scale**: Single family. Second generated family requires its own validation gate.
3. **Infrastructure**: Must target an existing layer with existing ClickHouse table and NATS stream.
4. **Mapper**: Must use a named mapper (`mapper: "{function_name}"`), not `mapper: "generate"`.
5. **Integration**: Generated fragments inserted manually. No automatic file manipulation.
6. **Golden snapshots**: Must be created and pass CI before merge.
7. **Validation**: All 7 success criteria from S197 must be met before the family is considered complete.
8. **Templates frozen**: No template refactoring during first generated family implementation.

## What This Gate Does NOT Authorize

- Tier 2 (read-path) generation
- Mapper generation (A3) or any `domain.columns` spec extension
- Multi-family generation in a single iteration
- Automatic file integration (marker sections)
- New table DDL or migration generation
- Any expansion of the codegen engine scope without a new formal gate

## Conditions for Revoking This Authorization

The gate verdict reverts to FAIL if any of the following occur during implementation:

1. Generated A1 or A2 code requires manual editing to compile or pass tests
2. Golden comparison fails for the new family and cannot be resolved by spec correction alone
3. Scope creep: any artifact beyond A1+A2 is generated rather than hand-crafted
4. The new family requires infrastructure that does not already exist
5. Templates are modified to accommodate the new family (indicating the model is not stable)

## Preparation Checklist for Implementation

Before the first generated family can proceed:

- [ ] Select specific family using S197 candidate criteria
- [ ] Author YAML spec and validate with `codegen validate`
- [ ] Generate A1 + A2 with `codegen generate`
- [ ] Create golden snapshots and verify with `codegen compare`
- [ ] Hand-craft A3 (mapper), A4 (mapper tests), A5 (config entry), A6 (smoke phase)
- [ ] Manually integrate A1+A2 fragments into source files
- [ ] Run full test suite (`go test ./...`, `make codegen-check`)
- [ ] Run smoke test (end-to-end event flow)
- [ ] Measure time savings vs manual baseline (~65 min)
- [ ] Document findings for next hardening gate
