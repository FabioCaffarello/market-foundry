# Stage S207: Codegen Path Stabilization or Explicit Freeze

**Date**: 2026-03-20
**Status**: Complete
**Decision**: Controlled Stabilization

---

## 1. Executive Summary

The codegen path exits this phase in a **stabilized, non-ambiguous state**. After comprehensive review of the engine, specs, templates, golden snapshots, CI gates, integrated slices, and 13 prior stage reports (S192–S204), the decision is **controlled stabilization** — not freeze, not expansion.

The codegen infrastructure is mechanically proven (14/14 golden comparisons pass, 4 CI gates operational, 26+ unit tests green) and provides governance value through drift detection and ownership markers. Freezing would waste working infrastructure during a refactoring phase where drift detection is most valuable. Expansion is premature because only 2 of 7 families are integrated, both in the same layer, and 4 of 6 Tier 1 artifacts remain manual.

---

## 2. Verification Results

| Gate | Result | Details |
|------|--------|---------|
| `make codegen-validate-all` | PASS | 7 families valid, cross-spec uniqueness OK |
| `make codegen-check` | PASS | 14/14 golden comparisons pass (7 families × 2 artifacts) |
| `make codegen-test` | PASS | All unit tests pass (0.216s) |
| `make codegen-integrated` | PASS | 4/4 integrated slices match golden snapshots |

---

## 3. Decision: Controlled Stabilization

### Why Not Freeze

- CI gates are operational and catch drift automatically during refactoring
- 4 governed code regions in 2 target files depend on marker enforcement
- Low maintenance cost (no active churn expected)
- 13 stages of infrastructure work would be wasted

### Why Not Expand

- Only 2/7 families integrated (RSI, EMA — both signal layer)
- Cross-layer viability unproven (evidence, decision, strategy, risk, execution untested)
- 4 of 6 Tier 1 artifacts remain manual (mappers, mapper tests, config, smoke tests)
- Mapper generation requires spec schema extension not yet designed
- Live event flow for generated families not proven
- Time savings modest (~15 min per family) — governance is the primary value

### What Stabilization Means

- CI gates remain active and blocking
- Existing governed regions are protected during refactoring
- New family additions via codegen are permitted under documented conditions
- Scope is capped at A1 (consumer_spec) + A2 (pipeline_entry)
- Any expansion beyond this scope requires a new architecture stage

---

## 4. Deliverables

| Deliverable | Path | Content |
|-------------|------|---------|
| Stabilization decision | `docs/architecture/codegen-path-stabilization-or-freeze-decision.md` | Formal decision with rationale, alternatives, risk assessment |
| Usage boundaries | `docs/architecture/codegen-current-usage-boundaries-and-limitations.md` | What can/cannot be used, integration map, safe operating procedures |
| Next-phase conditions | `docs/architecture/codegen-next-phase-readiness-or-freeze-conditions.md` | Freeze triggers, expansion gates, handoff checklist |
| Stage report | `docs/stages/stage-s207-codegen-path-stabilization-or-explicit-freeze-report.md` | This document |

---

## 5. Current State Inventory

### Codegen Engine

| Component | Count | Status |
|-----------|-------|--------|
| Family specs | 7 | All valid, cross-spec unique |
| Templates | 2 | consumer_spec.go.tmpl, pipeline_entry.go.tmpl |
| Golden snapshots | 14 | All pass comparison |
| Integrated slices | 4 | 2 families × 2 artifacts |
| CI gates | 4 | All operational and blocking |
| CLI commands | 5 | validate, generate, compare, validate-all, check-all |
| Unit tests | 26+ | All passing |

### Integration Coverage

| Family | Layer | Spec | Golden | Integrated | CI-Governed |
|--------|-------|------|--------|------------|-------------|
| rsi | signal | Yes | Yes | Yes (S200) | Yes |
| ema | signal | Yes | Yes | Yes (S203) | Yes |
| candle | evidence | Yes | Yes | No | Golden only |
| mean_reversion_entry | strategy | Yes | Yes | No | Golden only |
| paper_order | execution | Yes | Yes | No | Golden only |
| position_exposure | risk | Yes | Yes | No | Golden only |
| rsi_oversold | decision | Yes | Yes | No | Golden only |

---

## 6. Boundaries and Limitations

### What Is Stable

- Spec validation (per-spec + cross-spec)
- Golden snapshot generation and comparison
- Structural normalization for drift detection
- Naming derivation (PascalCase, layer-aware conventions)
- CI gate enforcement

### What Is Not Available

- Mapper generation (A3) — requires spec extension
- Config generation (A5) — JSONC tooling missing
- Automated file insertion — manual marker placement
- Cross-layer integration proof — signal layer only
- Live event flow validation — not yet tested

### What Is Explicitly Prohibited

- Expanding to new artifact types without a new stage
- Modifying the frozen S193 spec schema
- Bulk-integrating remaining 5 families without per-family validation
- Bypassing CI gates

---

## 7. Impact on Next Phase

### Positive

- CI gates continue catching drift during refactoring — prevents silent breakage
- Governed regions are explicitly marked — refactoring knows what not to touch
- Golden snapshots serve as regression baselines
- Spec validation prevents accidental collision during structural changes

### Neutral

- Codegen maintenance cost is near-zero if no templates or specs change
- 5 unintegrated families remain as-is (manual code with golden references)

### Risk

- Refactoring that changes consumer spec or pipeline entry structure may require template updates
- If refactoring changes domain types or NATS subjects, specs may need reconciliation
- Both risks are caught by CI gates before merge

---

## 8. Preparation for S208

The codegen path is ready for the next phase with:

1. **Clear status**: Stabilized for controlled use, not ambiguous
2. **Active CI gates**: 4 gates continue protecting against drift
3. **Documented boundaries**: What can/cannot be done is explicit
4. **Freeze conditions**: Automatic and discretionary triggers documented
5. **Expansion gates**: 4 formal gates must pass before scope grows
6. **No contamination**: The refactoring phase inherits a clean, bounded codegen tool

### Recommended S208 Actions

- Keep all 4 CI gates active — do not disable during refactoring
- If refactoring touches governed files, verify `make codegen-integrated` passes
- If templates need adjustment for refactored structures, update templates + all goldens
- Do not attempt to integrate remaining 5 families as part of refactoring — that is expansion, not cleanup

---

## 9. Acceptance Criteria Verification

| Criterion | Met | Evidence |
|-----------|-----|----------|
| Generated path leaves ambiguous state | Yes | Formal decision: controlled stabilization |
| Current usage clearly delimited | Yes | Boundaries document with stable/unstable/prohibited tables |
| Decision is formal and defensible | Yes | Rationale document with evidence tables and alternatives |
| Next phase not contaminated | Yes | Explicit constraints and freeze conditions documented |
| Base ready for S208 closure | Yes | Handoff checklist complete, CI gates verified |

---

## 10. Files Changed

### New Files (4)

- `docs/architecture/codegen-path-stabilization-or-freeze-decision.md`
- `docs/architecture/codegen-current-usage-boundaries-and-limitations.md`
- `docs/architecture/codegen-next-phase-readiness-or-freeze-conditions.md`
- `docs/stages/stage-s207-codegen-path-stabilization-or-explicit-freeze-report.md`

### Files Verified (no changes needed)

- `codegen/` — all specs, templates, goldens, tests pass without modification
- `.github/workflows/ci.yml` — CI gates already operational
- `Makefile` — codegen targets already present
- `scripts/codegen-integrated-check.sh` — integration verification script operational
- `codegen/integrated.yaml` — manifest current and accurate
