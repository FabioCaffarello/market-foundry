# Next Wave Recommendations — After Post-Generated Family Gate (S204)

## Context

S204 concluded with a **CONDITIONAL PASS**: the generated path works for same-layer, same-infrastructure families. The first codegen-first family (EMA) confirmed mechanical correctness, governance enforcement, and infrastructure reuse. It did NOT confirm cross-layer viability, mapper generation, batch efficiency, or live event flow.

## Recommendation

**Option 2: One more same-layer codegen-first family, then hardening gate.**

The generated path should prove **repeatability** before expanding scope. A single family (EMA) on a single layer (signal) with full infrastructure reuse is not sufficient evidence that the model works generically. A second same-layer family would confirm that the process is repeatable without being trivially identical to EMA.

Expanding to a cross-layer family or authorizing mapper generation at this point would skip validation steps and risk overextending based on one data point.

## Decision Tree

```
                  S204 gate: CONDITIONAL PASS
                              │
                ┌─────────────┴──────────────┐
                │                            │
          Repeatability                 Premature
          confirmed?                    expansion
                │                       (NOT AUTHORIZED)
                │
     ┌──────────┴──────────┐
     │                     │
  Option A              Option B
  Second same-layer     Cross-layer
  family (RECOMMENDED)  family
     │                  (DEFERRED —
     │                   needs own gate)
     │
     ▼
  Hardening gate
  (mandatory before
   cross-layer)
```

## Recommended Sequence

### Wave 1: Second Codegen-First Family (Same Layer)

**Objective**: Prove the generated path is repeatable, not a one-off success.

**Candidate selection criteria**:
- Must be on an existing layer with full infrastructure reuse
- Should NOT be signal layer (if possible) to test at least one different layer
- If signal layer is the only option with full reuse, it still has value for repeatability
- Must use existing mapper, table, NATS stream
- Must follow same A1+A2-only scope

**Candidate evaluation**:

| Layer | Has existing table? | Has existing mapper? | Has existing consumer factory? | Full reuse? |
|-------|-------|--------|---------|------|
| Evidence | Yes (evidence_candles) | Yes (mapCandleRow) | Yes (NewCandleConsumer) | Yes — but evidence layer has naming exception |
| Signal | Yes (signals) | Yes (mapSignalRow) | Yes (NewSignalConsumer) | Yes — RSI + EMA already here |
| Decision | Yes (decisions) | Yes (mapDecisionRow) | Yes (NewDecisionConsumer) | Yes |
| Strategy | Yes (strategies) | Yes (mapStrategyRow) | Yes (NewStrategyConsumer) | Yes |
| Risk | Yes (risk_assessments) | Yes (mapRiskRow) | Yes (NewRiskConsumer) | Yes |
| Execution | Yes (executions) | Yes (mapExecutionRow) | Yes (NewExecutionConsumer) | Yes |

**Recommendation**: Prefer a non-signal layer (decision, strategy, risk, or execution) to test cross-layer naming derivation. Evidence layer's naming exception adds risk that should be tested but not as the second iteration.

**Deliverables**: Spec, golden snapshots, integrated fragments, validation, frictions delta report.

**Success criteria**:
1. All S203 success criteria met again
2. Time delta documented (vs EMA iteration)
3. Frictions catalogued and compared to EMA frictions
4. Zero regressions across all existing families and tests

### Wave 2: Repeatability Hardening Gate

**Objective**: After 2 codegen-first families, assess whether the pattern is stable enough for controlled expansion.

**Questions to answer**:
1. Did the second family introduce any new frictions not seen with EMA?
2. Is manual insertion still acceptable or has toil become dominant?
3. Are cross-spec uniqueness checks scaling (now 8+ specs)?
4. Is the CI chain stable under growing golden snapshot count?
5. Is the developer experience acceptable (CODEGEN_ROOT, copy-paste, marker format)?

**Possible outcomes**:
- **Expand to cross-layer**: If both same-layer families succeeded cleanly and the developer experience is acceptable
- **Harden first**: If frictions compounded or new failure modes appeared
- **Pause**: If the cost/benefit ratio did not improve with repetition

### Wave 3: Cross-Layer Validation (Conditional)

**Prerequisite**: Wave 2 gate PASS.

**Objective**: Test whether the generated path works for a family on a layer that is structurally different from signal.

**Key risks**:
- Evidence layer naming exception (omits layer from function names)
- Possible template adjustments needed for different layers
- Different mapper signatures across layers

**If template changes are needed**: This triggers a template evolution ceremony — a new authorization stage, re-validation of all golden snapshots, and CI regression proof.

### Wave 4: Live Activation Proof (Required Before Wave 5)

**Prerequisite**: At least one generated family must demonstrate live event flow, not just structural compilation.

**Options**:
1. Synthetic NATS event injection during smoke test
2. Extend derive service to emit events for a generated family
3. Accept structural proof as sufficient (NOT RECOMMENDED for the long term)

**Debt**: D-1 (live event flow proof) must be resolved before a third codegen-first family is authorized.

### Wave 5: Mapper Generation Feasibility (Conditional)

**Prerequisite**: ≥2 validated codegen-first families (A1+A2). Cross-layer validation complete.

**Objective**: Design `domain.columns` spec extension and assess whether mapper generation (A3) is feasible and cost-effective.

**This is NOT authorized now.** It requires:
1. Schema evolution ceremony (extending from 14 to N fields)
2. New template design and validation
3. Equivalence baseline for mappers across existing families
4. Golden snapshot creation and CI integration

## What Should NOT Happen Next

1. **Do not batch-generate multiple families.** One at a time, with explicit validation.
2. **Do not expand to mapper generation (A3).** The evidence base covers A1+A2 only.
3. **Do not retroactively convert manual families.** The 6 original families are permanently manual golden references.
4. **Do not modify templates.** Frozen until a template evolution ceremony is formally authorized.
5. **Do not extend the spec schema.** 14 fields are sufficient for A1+A2; extension requires its own stage.
6. **Do not automate fragment insertion.** Manual insertion is acceptable for ≤4 governed families.
7. **Do not skip the repeatability gate.** A second same-layer family is required before cross-layer or other expansion.

## Timeline Expectation

| Wave | Dependency | Expected effort |
|------|-----------|----------------|
| Wave 1: Second family | None — can start immediately | 1 stage |
| Wave 2: Repeatability gate | Wave 1 complete | 1 stage |
| Wave 3: Cross-layer | Wave 2 PASS | 1-2 stages |
| Wave 4: Live activation | Independent of Wave 3 | 1 stage |
| Wave 5: Mapper feasibility | Waves 2+3 complete | 2-3 stages |

## Summary

The generated path earned the right to continue — but only one step at a time. The next step is a second same-layer family to prove repeatability, followed by a mandatory hardening gate. Cross-layer expansion, mapper generation, and batch operations are all explicitly deferred until their prerequisites are met. The discipline that governed S199–S203 must continue: evidence before expansion, not enthusiasm.
