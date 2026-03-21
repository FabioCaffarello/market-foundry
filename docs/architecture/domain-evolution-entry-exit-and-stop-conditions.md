# Domain Evolution — Entry, Exit, and Stop Conditions

**Charter:** Domain Logic Depth — Decision, Strategy, and Risk Evolution
**Date:** 2026-03-20
**Reference:** `domain-evolution-charter-and-scope-freeze.md`

---

## 1. Entry Conditions

The charter may begin implementation (S234) only when **all** of the following are satisfied:

| # | Condition | Status | Evidence |
|---|-----------|--------|----------|
| 1 | S232 clean-pass gate PASS | ✅ Met | S232 report: 5/5 criteria PASS |
| 2 | Charter document frozen | ✅ Met | `domain-evolution-charter-and-scope-freeze.md` |
| 3 | Permitted/prohibited scope defined | ✅ Met | `domain-evolution-permitted-vs-prohibited-changes.md` |
| 4 | `make quality-gate-ci` at 0 errors | ✅ Met | 84/84 checks, 0 errors (S232 evidence) |
| 5 | Remote CI green on current baseline | ✅ Met | Run `23365571775`, all 3 jobs green |
| 6 | Release tag on green commit | ✅ Met | `v0.1.0-s231` on `edb3010` |

**Verdict: All entry conditions are satisfied. Implementation may begin at S234.**

---

## 2. Exit Conditions (Charter Completion)

The charter is **complete** when **all** of the following are true:

| # | Condition | Measurement |
|---|-----------|-------------|
| 1 | Decision breadth achieved | ≥2 distinct decision evaluator types with tests |
| 2 | Strategy breadth achieved | ≥2 distinct strategy resolver types with tests |
| 3 | Risk breadth achieved | ≥2 distinct risk evaluator types with tests |
| 4 | Full pipeline proof | Each new evaluator/resolver produces events through derive → store → analytical |
| 5 | All CI green | `make test`, `make test-integration`, `make quality-gate-ci` all pass |
| 6 | Remote CI green | At least one verified-green remote CI run post-implementation |
| 7 | No regressions | Existing evaluators/resolvers pass unchanged |
| 8 | Gate stage completed | A formal gate stage (analogous to S232) evaluates all criteria |
| 9 | Hardening budget respected | ≤20% of stage effort spent on non-feature hardening |

**Partial exit is not permitted.** All exit conditions must be met for charter closure.

---

## 3. Stop Conditions

A **stop condition** triggers an immediate pause in implementation. Work resumes only after the condition is resolved or the charter is formally amended.

### 3.1 Hard Stops (Implementation Halts)

| # | Condition | Trigger | Resolution |
|---|-----------|---------|------------|
| 1 | Scope violation | A prohibited change is merged or proposed | Revert or justify via charter amendment |
| 2 | CI regression | `make quality-gate-ci` reports >0 errors | Fix before next stage |
| 3 | Remote CI failure | Remote CI run fails on charter work | Fix before next stage |
| 4 | Hardening overrun | Hardening effort exceeds 20% budget | Stop and reassess charter priorities |
| 5 | Domain model break | Existing evaluator/resolver tests fail due to charter changes | Fix or revert before proceeding |
| 6 | Architectural drift | Changes require modifications outside decision/strategy/risk domains that are not feature-pulled | Stop and evaluate if charter scope needs amendment |

### 3.2 Soft Stops (Pause and Reassess)

| # | Condition | Trigger | Resolution |
|---|-----------|---------|------------|
| 1 | Complexity escalation | A single evaluator/resolver requires >2 stages to implement | Simplify scope or split into phases |
| 2 | Dependency discovery | Feature work reveals a required upstream change (signal/indicator) | Document the dependency; defer if possible; amend charter if not |
| 3 | Test coverage gap | New logic has no clear integration test path | Design the test path before proceeding |
| 4 | Model tension | New evaluator requires domain model changes that conflict with existing evaluators | Design the resolution before implementing |

### 3.3 Information Stops (Document and Continue)

| # | Condition | Trigger | Action |
|---|-----------|---------|--------|
| 1 | Technical debt discovered | Implementation reveals pre-existing issues unrelated to charter | Log in stage report; do not fix unless feature-pulled |
| 2 | Optimization opportunity | A performance improvement is identified but not required | Log for future charter; do not implement |
| 3 | Documentation gap | Missing docs for existing patterns are noticed | Log for future charter; do not start doc cleanup |

---

## 4. Stage Cadence

### Expected Stage Sequence

| Stage | Focus | Domain | Type |
|-------|-------|--------|------|
| S233 | Charter definition and scope freeze | All | Governance |
| S234 | Decision evaluator #2 | Decision | Feature |
| S235 | Strategy resolver #2 | Strategy | Feature |
| S236 | Risk evaluator #2 | Risk | Feature |
| S237 | Pipeline integration proof + CI hardening | All | Integration |
| S238 | Gate evaluation | All | Governance |

This is a **suggested** sequence. Actual stages may be reordered or consolidated based on implementation findings, provided the charter scope is respected.

### Per-Stage Requirements

Every implementation stage (S234–S237) must:

1. State which charter objective it advances.
2. Map all changes to the permitted/prohibited categories.
3. End with `make quality-gate-ci` at 0 errors.
4. Produce a stage report documenting what was done and why.
5. Flag any gray-zone changes with justification.

---

## 5. Charter Amendment Process

If the charter scope must change (e.g., a hard stop reveals that upstream changes are unavoidable):

1. Document the reason for the amendment in a dedicated architecture document.
2. Explicitly state what changes in scope (additions and removals).
3. Re-evaluate the success criteria and hardening budget.
4. Record the amendment in the next stage report.
5. Do not retroactively modify existing charter documents — append amendments.

---

## 6. S234 Preparation Checklist

Before S234 begins implementation:

- [x] Charter document frozen (`domain-evolution-charter-and-scope-freeze.md`)
- [x] Permitted/prohibited changes defined (`domain-evolution-permitted-vs-prohibited-changes.md`)
- [x] Entry/exit/stop conditions defined (this document)
- [x] Entry conditions all satisfied
- [ ] Current decision evaluator reviewed (understand RSI oversold evaluator in detail)
- [ ] Target decision evaluator #2 designed (type, signals, outcome logic)
- [ ] Derive actor pattern for new evaluator mapped
- [ ] Test strategy for new evaluator defined
