# Stage S197 — First Generated Family Scope Decision Report

**Stage**: S197
**Status**: COMPLETE
**Date**: 2026-03-20
**Depends on**: S193, S194, S195, S196
**Prepares**: S198 (first generated family implementation)

---

## 1. Executive Summary

S197 issues a formal decision on whether the codegen engine is ready to govern a first new analytical family. The decision is **YES — authorized under constrained conditions**. The engine may generate Artifacts A1 (consumer spec) and A2 (pipeline entry) for a single new family. All other artifacts (A3–A6, domain types, migrations, file integration) remain manual.

This decision is grounded in S196's complete validation: 12/12 golden comparisons pass across all 6 existing families with 0 structural drift. The codegen engine has proven it can reproduce the two most mechanical artifacts identically across the full complexity spectrum (evidence through execution layers, simple through compound naming, known abbreviations through multi-word identifiers).

---

## 2. Decision Summary

| Question | Answer |
|---|---|
| Is there sufficient basis for a first generated family? | **Yes** |
| What can be generated? | **A1 (consumer spec) + A2 (pipeline entry) only** |
| What remains manual? | **A3–A6, domain types, migrations, file integration** |
| How many families? | **One** |
| Which tier? | **Tier 1 only** |
| Which family? | **Not selected in S197 — deferred to S198** |

---

## 3. Evidence Base

### 3.1 — Spec Freeze (S193)

- 14-field canonical YAML schema frozen
- 6 Tier 1 artifact types defined with ownership rules
- Three-condition boundary test codified (repetitive + mechanical + spec-derivable)
- Tier 2 (read-path) explicitly deferred until Tier 1 proven

### 3.2 — Equivalence Baseline (S194)

- RSI (minimal) + Paper Order (ceiling) bracket established
- Structural vs semantic equivalence rules defined for all 6 artifact types
- Golden snapshot extraction and comparison procedure codified
- Drift classification: CRITICAL / WARNING / INFO severity model

### 3.3 — Minimal Engine (S195)

- Engine covers A1 + A2 (consumer spec + pipeline entry)
- 4/4 golden comparisons pass (2 families × 2 artifacts)
- 17 unit tests pass
- 10 derived fields computed correctly from spec
- Evidence layer exceptions handled and tested

### 3.4 — Cross-Family Validation (S196)

- Engine validated against all 6 existing families (not just 2 baselines)
- **12/12 golden comparisons PASS**
- **0 structural drift**
- **3 cosmetic drift instances** (all INFO severity):
  - D1: Comment phrasing variation
  - D2: Decorative dash length in section comments
  - D3: Evidence layer comment omission
- All cosmetic drift handled by S194 normalization
- 26 unit tests pass (17 from S195 + 9 added in S196)
- CI gate operational: `codegen-golden` job blocks merge

---

## 4. Deliverables

| # | Document | Path |
|---|---|---|
| 1 | Scope Decision | `docs/architecture/first-generated-family-scope-decision.md` |
| 2 | Generated vs Manual Boundary | `docs/architecture/generated-vs-manual-boundary-for-first-generated-family.md` |
| 3 | Risks, Success Criteria, Non-Goals | `docs/architecture/first-generated-family-risks-success-criteria-and-non-goals.md` |
| 4 | This Report | `docs/stages/stage-s197-first-generated-family-scope-decision-report.md` |

---

## 5. Key Decisions

### D1: First Generated Family Authorized

The codegen engine has sufficient evidence to govern A1 + A2 for a new family. Authorization is limited to these two artifact types.

### D2: A3–A6 Remain Manual

Mapper functions (A3), mapper tests (A4), config entries (A5), and smoke test phases (A6) are not authorized for generation. Each requires its own equivalence validation stage.

### D3: Single Family, Single Iteration

Only one family may be generated. The result must be validated before a second family is attempted.

### D4: Existing Infrastructure Required

The first generated family must target an existing layer with established NATS streams, ClickHouse tables, and registry adapters. No new infrastructure creation is authorized as part of codegen validation.

### D5: Template Freeze

Templates are frozen at S195/S196 state. No refactoring during the first generated family. Issues discovered become inputs to a future hardening stage.

---

## 6. Risks

| ID | Risk | Severity | Likelihood |
|---|---|---|---|
| R1 | Fragment integration error (manual insertion) | Medium | Medium |
| R2 | Spec authoring error (wrong values) | Medium | Low |
| R3 | Overconfidence from A1+A2 success → premature expansion | High | Medium |
| R4 | Golden snapshot tautology (new family validates against itself) | Low | Low |
| R5 | Evidence layer exception misapplication | Low | Low |

Primary mitigation: manual review, full test suite, smoke test validation, and explicit scope constraints.

---

## 7. Success Criteria

| ID | Criterion | Verification |
|---|---|---|
| SC1 | Golden comparison passes for new family | CI `codegen-golden` job |
| SC2 | Generated code compiles | `go build ./...` |
| SC3 | All unit tests pass | `go test ./...` across all modules |
| SC4 | Smoke test passes | End-to-end event flow validation |
| SC5 | No manual edits to generated code | Review: A1+A2 inserted verbatim |
| SC6 | Time savings measured | Actual vs ~65 min baseline |
| SC7 | No codegen scope creep | Only A1+A2 generated |

---

## 8. Non-Goals

- Full family generation (A1–A6)
- Codegen coverage expansion to A3–A6 or Tier 2
- Automatic file integration (marker sections)
- Multi-family generation
- New infrastructure creation
- Performance benchmarking
- Template refactoring

---

## 9. Acceptance Criteria Status

| Criterion | Status |
|---|---|
| Formal decision on first generated family exists | **MET** — Decision D1: authorized |
| Clear what can and cannot be generated | **MET** — A1+A2 yes, A3–A6 no, boundary doc delivered |
| Limits and risks of first iteration explicit | **MET** — 5 risks, 5 failure modes, 7 non-goals documented |
| Base ready for S198 gate | **MET** — candidate criteria, workflow, success criteria defined |
| Risk of premature codegen expansion reduced | **MET** — R3 explicitly identified, scope constraints codified, non-transferable authorization |

---

## 10. Preparation for S198

S198 should execute the following:

1. **Select the specific family** — apply the 6 candidate criteria from the scope decision document
2. **Author the YAML spec** — validate against S193 frozen schema
3. **Generate A1 + A2** — using the codegen engine CLI
4. **Create golden snapshots** — commit alongside spec
5. **Hand-craft A3–A6** — mapper, tests, config, smoke
6. **Integrate and validate** — full test suite + smoke
7. **Measure time** — compare against manual baseline
8. **Document findings** — friction, drift, issues for future hardening

The S198 gate should evaluate whether the generation model is ready for a second family or needs hardening first.

---

## 11. Metrics

| Metric | Value |
|---|---|
| Prior golden comparisons (S196) | 12/12 PASS |
| Structural drift instances | 0 |
| Cosmetic drift instances | 3 (all INFO) |
| Unit tests passing | 26 |
| Artifacts authorized for generation | 2 of 6 Tier 1 |
| Families authorized | 1 |
| Documents delivered | 4 |
| Implementation performed | 0 (decision-only stage) |
