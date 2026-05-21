# Stage S258 — Codegen Re-entry Charter and Scope Freeze Report

**Stage:** S258
**Date:** 2026-03-21
**Predecessor:** S257 (Post-Behavioral Hardening Transition Gate — PASS)
**Verdict:** PASS — charter opened and scope frozen

---

## 1. Executive Summary

Stage S258 formally opens the codegen re-entry wave following the successful closure of BEHAVIORAL-WAVE-1 at S257. The charter freezes scope around a single objective: extend codegen integration from 2 families (RSI, EMA) to all 10 existing families, proving manual→generated equivalence for `consumer_spec` and `pipeline_entry` artifacts.

The wave explicitly prohibits new artifact types, new family specs, domain logic generation, and infrastructure expansion. The boundary between codegen (mechanical mapping) and human authorship (behavioral logic, domain design) is formally documented.

---

## 2. Deliverables Produced

| # | Document | Path | Status |
|---|---|---|---|
| 1 | Charter and scope freeze | `docs/architecture/codegen-reentry-charter-and-scope-freeze.md` | DELIVERED |
| 2 | Permitted vs prohibited changes | `docs/architecture/codegen-reentry-permitted-vs-prohibited-changes.md` | DELIVERED |
| 3 | Entry, exit, and stop conditions | `docs/architecture/codegen-reentry-entry-exit-and-stop-conditions.md` | DELIVERED |
| 4 | This report | `docs/stages/stage-s258-codegen-reentry-charter-and-scope-freeze-report.md` | DELIVERED |

---

## 3. Charter Summary

### Objective

Reconcile all 10 family specs with golden snapshots and extend `integrated.yaml` from 4 entries to 20 entries (10 families × 2 artifacts), with CI enforcement.

### Scope boundaries

- **In scope:** spec reconciliation, snapshot refresh, integration expansion, marker insertion, CI gate, manual→generated equivalence
- **Out of scope:** new templates, new families, domain logic generation, actor wiring, evaluator/resolver codegen, infrastructure expansion

### Key constraint

The 47 behavioral tests from BEHAVIORAL-WAVE-1 are a hard gate. Any regression is an immediate stop condition.

---

## 4. Current State Assessment

| Metric | Current | Target |
|---|---|---|
| Family specs | 10 | 10 (no change) |
| Templates | 2 | 2 (no change) |
| Golden snapshots | 20 | 20 (refreshed) |
| `integrated.yaml` entries | 4 | 20 |
| Integration markers in target files | 2 files | 10 files |
| CI codegen enforcement | None | `codegen check-all` + `codegen validate-all` |
| Behavioral tests | 47 passing | 47 passing (maintained) |

---

## 5. Planned Stage Sequence

| Stage | Objective | Key deliverable |
|---|---|---|
| S258 | Charter and scope freeze | This report (complete) |
| S259 | Spec/template reconciliation | All 10 specs validated; all 20 snapshots refreshed |
| S260 | Integration expansion | Markers in all target files; `integrated.yaml` at 20 entries |
| S261 | CI enforcement and equivalence proof | Codegen CI gate; manual→generated verification |
| S262 | Post-codegen re-entry gate | Formal exit assessment |

---

## 6. Entry Conditions Status

| ID | Condition | Status |
|---|---|---|
| EN-1 | BEHAVIORAL-WAVE-1 closed (PASS) | MET |
| EN-2 | Zero medium-or-higher-risk debts | MET |
| EN-3 | 47 behavioral tests in CI | MET |
| EN-4 | Codegen module intact | MET |
| EN-5 | All 10 specs parseable | TO VERIFY (S259) |
| EN-6 | All 20 golden snapshots present | MET |
| EN-7 | raccoon-cli guardian operational | MET |
| EN-8 | Charter approved | MET |

---

## 7. Risk Posture

| Risk | Level | Mitigation |
|---|---|---|
| Scope creep | Medium | Prohibited changes list; escalation rules; amendment log |
| Spec drift | Medium | Early reconciliation in S259 |
| Behavioral regression | Low | Hard stop ST-1; behavioral tests as CI gate |
| Template brittleness | Low | Golden snapshot comparison catches regressions |
| Manual→generated divergence | Medium | Document as debt; do not force-fit |

---

## 8. Non-objectives (explicitly excluded)

- Generating evaluators, resolvers, or actor wrappers
- Adding new domain families
- Parameterizing template hardcoded values
- Opening new breadth or feature waves
- Building codegen into a "generate everything" tool
- Addressing deferred behavioral debts (OD-BW2, OD-BW5, OD-BW6)

---

## 9. Preparation for S259

To begin S259 (spec/template reconciliation), the following should be verified:

1. Run `codegen validate-all` — confirm all 10 specs parse and validate (EN-5).
2. Run `codegen check-all` — identify which of the 20 golden comparisons pass or fail.
3. For any failures, classify as:
   - **Spec drift** — spec values don't match current domain → fix spec
   - **Template bug** — template produces incorrect output → fix template (Tier 2)
   - **Snapshot staleness** — snapshot was generated from old template → regenerate
4. Prioritize families by layer: signal → decision → strategy → risk → execution → evidence (aligns with domain maturity order).

---

## 10. Verdict

**PASS** — The codegen re-entry charter is formally opened and scope-frozen. All entry conditions are met or scheduled for verification. The wave may proceed to S259.
