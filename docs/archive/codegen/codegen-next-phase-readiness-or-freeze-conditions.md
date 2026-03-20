# Codegen: Next-Phase Readiness and Freeze Conditions

**Stage**: S207
**Date**: 2026-03-20
**Status**: Stabilized — conditions below govern next-phase behavior

---

## 1. Next-Phase Context

The next phase focuses on refactoring, design pattern improvements, and documentation cleanup. The codegen path enters this phase as a stabilized, narrow tool — not as the primary development method.

---

## 2. What Carries Forward

### Active During Next Phase

| Component | Role in Next Phase |
|-----------|-------------------|
| CI gates (4 jobs) | Continue blocking — prevent drift during refactoring |
| Golden snapshots (14 files) | Reference artifacts — validate template correctness |
| Integrated slices (4 governed regions) | Protected — refactoring must not break governed regions |
| Spec validation | Active — ensures spec integrity during any structural changes |
| `integrated.yaml` manifest | Active — tracks governed regions |

### Passive During Next Phase

| Component | Status |
|-----------|--------|
| Template files | Unchanged unless refactoring requires adjustment |
| Family specs | Frozen — no new families expected during refactoring |
| Codegen CLI | Available but not primary tool during refactoring |

---

## 3. Conditions That Would Trigger a Freeze

The codegen path MUST be explicitly frozen if any of the following occur:

### Automatic Freeze Triggers

1. **CI gate failure that cannot be resolved within the refactoring scope**
   If a template or golden snapshot breaks due to refactoring and the fix requires codegen schema changes, freeze codegen until a dedicated stage addresses it.

2. **Governance marker corruption**
   If refactoring inadvertently removes or corrupts codegen:begin/end markers in governed files and restoration is non-trivial, freeze integration until markers are restored.

3. **Spec schema incompatibility**
   If refactoring changes domain types, NATS subjects, or table schemas in ways that invalidate the S193 spec schema, freeze codegen until specs are reconciled.

4. **Template output divergence**
   If refactoring changes the expected structure of consumer specs or pipeline entries in ways that make templates produce incorrect code, freeze codegen until templates are updated.

### Discretionary Freeze Triggers

5. **Codegen maintenance becomes a distraction**
   If keeping golden snapshots and integrated slices aligned with refactoring changes consumes more than 30 minutes per refactoring PR, consider temporary freeze.

6. **Architectural pivot**
   If the refactoring phase changes the writer pipeline architecture fundamentally (e.g., removing the actor model, changing NATS consumer patterns), freeze codegen until the new architecture stabilizes.

---

## 4. Conditions For Expanding Codegen in a Future Phase

Codegen scope expansion (beyond A1+A2) requires ALL of the following:

### Gate 1: Integration Proof

- [ ] At least 3 families integrated with governance markers (currently 2)
- [ ] At least 2 different layers represented in integrated families (currently 1: signal)
- [ ] All integrated families pass live event flow test

### Gate 2: Tooling Maturity

- [ ] Automated file insertion (codegen integrate command) implemented
- [ ] Golden snapshot regeneration automated (codegen regen-all command)
- [ ] Manifest auto-discovery or validation implemented

### Gate 3: Artifact Extension

- [ ] `domain.columns` spec extension designed and reviewed
- [ ] Mapper template (A3) implemented with golden snapshots
- [ ] Mapper test template (A4) implemented with golden snapshots
- [ ] All existing families' mapper goldens validated against manual mappers

### Gate 4: Governance

- [ ] New architecture stage opened for expansion
- [ ] S193 spec schema formally unfrozen with documented changes
- [ ] Updated cross-spec validation covers new fields
- [ ] CI gates extended for new artifact types

---

## 5. Readiness Matrix for Next Phase

| Aspect | Ready | Notes |
|--------|-------|-------|
| CI gates operational | Yes | All 4 gates pass, blocking |
| Golden snapshots current | Yes | 14/14 pass |
| Integrated slices valid | Yes | 4/4 pass |
| Spec schema frozen | Yes | S193 freeze active |
| Template correctness | Yes | All families × all artifacts match |
| Cross-spec uniqueness | Yes | 7 families, no collisions |
| Maintenance cost acceptable | Yes | No active churn expected during refactoring |
| Documentation complete | Yes | S207 deliverables cover boundaries and procedures |

---

## 6. What the Next Phase Should NOT Do

1. **Do not integrate remaining 5 families** unless a specific refactoring task requires it and per-family validation is performed.

2. **Do not modify templates** unless refactoring changes the structure of consumer specs or pipeline entries.

3. **Do not add new artifact types** — this requires a dedicated stage with spec extension.

4. **Do not remove CI gates** — even if codegen seems dormant, gates protect against accidental drift.

5. **Do not treat codegen as a refactoring tool** — codegen generates boilerplate, it does not refactor existing code.

6. **Do not bulk-regenerate** without per-family validation of the output.

---

## 7. Handoff Checklist for S208

Before closing S207, the following must be true:

- [x] All 4 CI gates pass (validate-all, check, test, integrated)
- [x] Decision documented (stabilize, not freeze)
- [x] Usage boundaries documented
- [x] Freeze conditions documented
- [x] Expansion gates defined
- [x] Next-phase constraints explicit
- [x] No ambiguity about codegen path status
