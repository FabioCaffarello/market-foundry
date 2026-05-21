# Behavioral Wave — Entry, Exit, Amendment, and Stop Conditions

**Charter:** BEHAVIORAL-WAVE-1
**Stage:** S249
**Date:** 2026-03-21
**Status:** Active

---

## 1. Purpose

This document codifies the formal conditions governing the BEHAVIORAL-WAVE-1 charter lifecycle. It defines when the charter can begin implementation, when it is complete, how it can be amended, and under what conditions it must be suspended.

---

## 2. Entry Conditions

The charter **may not begin implementation** (S250) until all of the following are true:

| # | Entry Condition | Status | Evidence |
|---|----------------|--------|----------|
| EC1 | BREADTH-WAVE-1 charter formally PASSED | DONE | S244 gate: 9/9 criteria met, 0 amendments |
| EC2 | Post-breadth hardening gate PASSED or CONDITIONAL PASS | DONE | S248 gate: CONDITIONAL PASS |
| EC3 | OD1 closed (S246–S247 committed, remote CI green) | **PENDING** | Must be completed before S250 begins |
| EC4 | This charter document (S249) formally accepted | **THIS STAGE** | Acceptance = S249 report filed |
| EC5 | Existing test pyramid fully green (unit, actor, integration, codegen) | DONE | S248 verification: all PASS |
| EC6 | No blocking architectural debts from prior waves | DONE | OD2/OD3 are non-blocking hygiene items |

### 2.1 Entry Gate Verdict

**Status: CONDITIONAL — converts to READY upon EC3 closure (OD1).**

EC3 is a mechanical step (commit + push + CI green). No architectural work is required. S250 implementation should begin immediately after EC3 closure.

---

## 3. Exit Criteria

The charter is **complete** when all of the following are true:

| # | Exit Criterion | Verification |
|---|---------------|-------------|
| EX1 | Multi-decision strategy input works for ≥1 strategy type | Integration test showing 2 decision types feeding one strategy resolver |
| EX2 | Multi-evaluator risk gating works for ≥1 strategy proposal | Integration test showing 2 risk evaluators gating one strategy proposal |
| EX3 | Scenario 1 (single-chain baseline) passes | Existing integration tests remain green — no new test required |
| EX4 | Scenario 2 (multi-decision input) passes | Dedicated integration test with evidence |
| EX5 | Scenario 3 (multi-evaluator risk) passes | Dedicated integration test with evidence |
| EX6 | Scenario 4 (cross-chain end-to-end) passes | Dedicated integration test with evidence |
| EX7 | Correlation tracing works end-to-end | Scenario test validates correlation ID propagation across ≥3 domain boundaries |
| EX8 | Behavioral routing is configuration-driven | Demonstration that routing changes via configctl, not code changes |
| EX9 | No new streams, tables, or binaries introduced | Architecture review confirmation |
| EX10 | `make test` and `make test-integration` pass | Local test run evidence |
| EX11 | Remote CI green for ≥1 commit in this charter | CI run ID + URL |
| EX12 | No regression in existing functionality | Existing test suite passes unchanged |

### 3.1 Partial Exit (If Amendment Required)

If the charter must close early (e.g., due to stop condition), a partial exit is acceptable if:

- EX1 OR EX2 is met (at least one tier delivered)
- EX3 is met (no regression)
- EX10 and EX11 are met (CI green)
- All delivered work is integration-tested

A partial exit must be documented with explicit reasoning for which criteria were not met and why.

---

## 4. Mid-Charter Gate

A mandatory mid-charter gate occurs after S251 (Tier 2 delivery):

### 4.1 Gate Questions

| # | Question | Expected Answer |
|---|----------|----------------|
| G1 | Is Tier 1 (decision→strategy multi-input) delivered? | Yes |
| G2 | Is Tier 2 (strategy→risk multi-gate) delivered? | Yes |
| G3 | Were any amendments filed? | Ideally 0 |
| G4 | Has hardening stayed within 20% budget? | Yes |
| G5 | Are existing tests still passing? | Yes |
| G6 | Is the implementation sequence on track? | Yes |

### 4.2 Gate Outcomes

| Outcome | Condition | Action |
|---------|-----------|--------|
| **PASS** | G1–G6 all YES | Proceed to S252 (Tier 3) |
| **CONDITIONAL PASS** | G1 and G2 YES, minor issues in G3–G6 | Proceed with documented conditions |
| **AMEND** | G1 OR G2 not met, but recoverable | File amendment, adjust S252 scope |
| **STOP** | G1 AND G2 not met, or fundamental blocker | Suspend charter, review root cause |

---

## 5. Amendment Rules

### 5.1 When an Amendment Is Required

An amendment **must** be filed before execution if:

| Trigger | Example |
|---------|---------|
| A behavioral tier is deferred or dropped | "Tier 2 deferred to next charter" |
| A minimum viable scenario is changed or removed | "Scenario 4 replaced with simpler version" |
| Breadth or depth work exceeds 20% of a stage | "50% of S251 spent on actor refactoring" |
| Implementation order changes | "S252 before S251" |
| A new stream, table, or binary is proposed | "Need a COMPOSITE_RISK_EVENTS stream" |
| An exit criterion is modified | "EX7 removed — correlation tracing deferred" |

### 5.2 Amendment Process

1. **Document** — Create a dated amendment record with:
   - Original charter commitment being changed
   - Proposed new scope
   - Rationale for the change
   - Impact on exit criteria (which criteria change, what replaces them)
   - Impact on hardening budget

2. **Append** — Add the amendment to the charter's Amendments Log (Section 11 of the charter document). The original charter text is immutable.

3. **Evaluate** — If the amendment changes ≥2 exit criteria, a stop-and-reassess is triggered.

### 5.3 Post-Hoc Amendments

If a deviation is discovered after execution:

1. Acknowledge the governance deviation in the stage report
2. File a post-hoc amendment with explanation
3. Document corrective action
4. Flag in the gate report as a governance finding

---

## 6. Stop Conditions

The charter must be **immediately suspended** if any of the following occur:

| # | Stop Condition | Severity | Action |
|---|---------------|----------|--------|
| SC1 | Two consecutive stages fail to deliver their primary behavioral deliverable | Critical | Suspend and root-cause analysis |
| SC2 | Mid-charter gate reveals ≥2 tiers undelivered | Critical | Suspend and charter re-evaluation |
| SC3 | Hardening exceeds 20% budget in any single stage | Serious | Stop and reassess scope |
| SC4 | A blocking architectural issue requires redesign affecting >1 domain | Serious | Stop, document issue, evaluate recovery |
| SC5 | Behavioral work inadvertently introduces new breadth (new types) | Serious | Revert breadth, file governance deviation |
| SC6 | New infrastructure (streams, tables, binaries) created without amendment | Serious | Revert, file post-hoc amendment |
| SC7 | Existing test suite regression not resolved within the same stage | Moderate | Stop until regression is fixed |

### 6.1 Stop Resolution Process

1. **Document** the stop condition trigger with evidence
2. **Evaluate** whether the charter can continue (recoverable) or must close (terminal)
3. **If recoverable:** File amendment, adjust remaining stages, resume with explicit conditions
4. **If terminal:** Close charter with partial exit (Section 3.1), document lessons learned

---

## 7. Relationship to Prior Governance

This document complements (does not replace):

- **Charter Amendment Rules (S239)** — The five amendment rules apply verbatim to this charter
- **Evolution Playbook** — All golden rules and readiness gates apply
- **breadth-charter-and-scope-freeze.md** — The breadth charter is CLOSED. This behavioral charter is its successor, not an extension.

### 7.1 Governance Chain

```
S239 (amendment rules) → S240 (breadth charter) → S244 (breadth gate: PASS) → S248 (hardening gate: CONDITIONAL PASS) → S249 (this behavioral charter)
```

The behavioral charter inherits the governance framework but defines its own scope, criteria, and conditions. It is a new charter, not an amendment to the breadth charter.
