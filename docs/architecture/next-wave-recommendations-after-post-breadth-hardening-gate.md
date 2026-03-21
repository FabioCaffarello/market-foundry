# Next Wave Recommendations — After Post-Breadth Hardening Gate

**Stage:** S248
**Date:** 2026-03-21
**Scope:** Objective recommendation for what comes after the breadth hardening gate, based on evidence from S245–S247.

---

## 1. Gate Status

The post-breadth hardening gate issued a **CONDITIONAL PASS**. All three explicit debts (D1, D2, D3) are resolved. The single remaining condition is mechanical: commit S246–S247 to main and obtain remote CI green.

---

## 2. Decision Framework

The question is: what is the next acceptable step?

| Option | Description                                        | Preconditions Met? |
|--------|----------------------------------------------------|--------------------|
| A      | Open a new feature wave                            | Conditional        |
| B      | Execute a short residual correction                | Not required       |
| C      | Pause until a specific blocker is resolved         | No blockers exist  |

---

## 3. Assessment

### Option A — Open a New Feature Wave

**Precondition:** S246–S247 committed to main with remote CI green.

Once this mechanical step is complete, the breadth wave is formally closed with full evidence. The codebase is in a clean state:
- All tests pass (unit, actor, integration, codegen golden).
- Smoke scripts cover all types symmetrically.
- Remote CI has validated the core breadth delivery.
- Zero production code changes during hardening.
- No architectural debt blocking forward progress.

**Verdict:** Acceptable, contingent on OD1 closure (commit + CI green).

### Option B — Execute a Short Residual Correction

The only candidates for residual correction are OD2 (migration linting) and OD3 (CI cache overhead). Neither is blocking:
- OD2 prevents a future class of error but has no current manifestation.
- OD3 adds ~30s to CI runs but does not affect correctness.

**Verdict:** Not required before the next wave. These can be addressed as tooling improvements within or alongside the next charter.

### Option C — Pause Until a Specific Blocker is Resolved

There are no architectural, operational, or correctness blockers. The codebase is clean. The test pyramid is symmetric. The CI pipeline is proven.

**Verdict:** No basis for pausing.

---

## 4. Recommendation

**Execute Option A with a single prerequisite:**

1. **Immediate (S248 closure):** Commit S246–S247 implementation to main, push, and verify remote CI green. This closes OD1 and converts the gate from CONDITIONAL PASS to PASS.

2. **Next charter:** The breadth wave is formally closed. The codebase can sustain a new feature wave. The specific scope of the next wave is a product/architecture decision, not a hardening concern.

**What the next wave should NOT do:**
- Assume breadth types are battle-tested in production. They are test-proven, not production-proven. The next wave should include observability or monitoring if it depends on breadth type correctness under real load.
- Skip CI validation. S245 proved that local and remote can diverge. Every stage should close with CI green.

**What the next wave CAN safely assume:**
- The pipeline pattern is stable and repeatable. Adding a new type follows the same codegen + actor + smoke pattern.
- Both chains (A and B) are integration-tested end-to-end.
- Risk domain symmetry holds. Both `position_exposure` and `drawdown_limit` are operationally equivalent in coverage.
- Smoke infrastructure scales. Adding a new type to smoke is additive, not structural.

---

## 5. Optional Improvements (Non-blocking)

These can be included in the next wave's charter or handled as standalone micro-stages:

| Improvement              | Priority | Effort | Rationale                                |
|--------------------------|----------|--------|------------------------------------------|
| Migration statement lint | Low      | Small  | Prevents repeat of S245 defect class     |
| CI cache optimization    | Low      | Small  | ~30s savings per CI run                  |
| Node.js action updates   | Low      | Small  | Deadline: June 2026                      |

None of these should gate the next wave. They are hygiene items.

---

## 6. Summary

The breadth wave hardening is complete. The recommendation is:

1. Close OD1 (commit + CI) → gate becomes PASS.
2. Open the next feature wave with confidence.
3. Carry OD2/OD3 as optional hygiene items, not blockers.

The breadth is no longer "delivered but not yet hardened." After OD1 closure, it is **delivered, hardened, and CI-proven**.
