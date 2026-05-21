# Next Wave Recommendations — After Post-Behavioral-Wave Gate

**Charter:** BEHAVIORAL-WAVE-1 (S249–S253)
**Gate:** S254
**Date:** 2026-03-21

---

## 1. Options Evaluated

The gate review identified four possible next directions:

1. **Continue deepening domain behavior** — extend behavioral composition within current domains
2. **Return to codegen/generated path** — resume the codegen pipeline and raccoon-cli evolution
3. **Open a new functional evolution line** — e.g., execution domain, multi-symbol, or observability
4. **Execute a short hardening tranche** — close medium-risk debts before moving forward

---

## 2. Assessment by Option

### Option 1: Continue Deepening Domain Behavior

**What it means:** Add more behavioral nuance — graduated EMA severity, multi-decision fusion (RSI + EMA combined), cross-strategy correlation, configurable scaling factors.

**Pros:**
- Builds directly on the behavioral foundation just delivered
- Would make the behavioral model richer and more production-realistic
- Natural continuation of the charter's trajectory

**Cons:**
- Risk of diminishing returns — the core behavioral chain already works
- Graduated severity and multi-decision fusion are partially breadth-adjacent (new behavioral modes, not just composition)
- The current behavioral model has not been validated against real market data; deepening it without validation may be premature optimization
- No operational feedback yet to guide which behavioral parameters need refinement

**Verdict:** Not recommended as immediate next step. The behavioral model needs to prove itself in a more realistic environment before further refinement.

### Option 2: Return to Codegen/Generated Path

**What it means:** Resume raccoon-cli development, advance the code generation pipeline, close the codegen→domain gap.

**Pros:**
- raccoon-cli is the intended architecture guardian and velocity multiplier
- Codegen path was paused for the behavioral wave; returning maintains strategic balance
- Generated code quality and coverage improvements compound over time

**Cons:**
- The behavioral wave left one medium-risk debt (full-stack smoke) that should be closed first
- Jumping directly to codegen without consolidating behavioral gains risks the new behavioral code being under-protected when codegen modifies adjacent code

**Verdict:** Viable as second step, after a short hardening tranche. The codegen path should not be blocked by behavioral debt, but a clean handoff is warranted.

### Option 3: Open a New Functional Evolution Line

**What it means:** Start work on the execution domain, multi-symbol support, observability infrastructure, or a new analytical family.

**Pros:**
- Execution domain is the natural next layer in the trading pipeline
- Multi-symbol support would unlock production-realistic scenarios

**Cons:**
- Opening a new functional line without closing existing debts creates compounding risk
- The execution domain requires significant design work (new actors, new streams, new domain model)
- Multi-symbol and observability are infrastructure-heavy — the opposite of the behavioral wave's discipline
- The system has not been proven end-to-end through the existing layers; adding another layer increases the surface without validating the foundation

**Verdict:** Not recommended at this time. The foundation must be consolidated before vertical expansion.

### Option 4: Execute a Short Hardening Tranche

**What it means:** A focused tranche (2–3 stages) to close medium-risk debts and strengthen the behavioral foundation before the next wave.

**Pros:**
- Closes OD-BW1 (full-stack behavioral smoke) — the only medium-risk debt
- Strengthens boundary-case coverage without opening new scope
- Provides a clean, validated foundation for whichever direction comes next
- Maintains the charter discipline established in the behavioral wave
- Low risk, bounded effort, high confidence of success

**Cons:**
- Delays forward progress by 2–3 stages
- Hardening work is less exciting than new features
- Risk of hardening expanding beyond scope if not tightly chartered

**Verdict:** Recommended as immediate next step, with a tight charter and clear scope boundary.

---

## 3. Recommendation

**Recommended path: Option 4 → Option 2**

Execute a short hardening tranche (2–3 stages), then return to the codegen/generated path.

### Hardening Tranche Scope (Suggested)

The tranche should be chartered with explicit scope freeze, similar to BEHAVIORAL-WAVE-1:

**In scope:**
1. Full-stack behavioral smoke test — validate behavioral properties through NATS serialization, ClickHouse write, and HTTP read-back (closes OD-BW1)
2. Severity boundary hardening — add defensive normalization and edge-case tests (closes OD-BW4)
3. Risk evaluator boundary coverage — test rejection/modification threshold behavior (partially closes OD-BW3)

**Out of scope:**
- Configurable scaling factors (OD-BW2) — defer until operational need
- Performance budgets (OD-BW5) — defer until scale concern
- Configctl activation (OD-BW6) — defer with OD-BW2
- Execution layer (OD-BW7) — future charter
- New behavioral features, new analytical types, new infrastructure

### Post-Tranche Direction

After the hardening tranche, the codegen/generated path (Option 2) is the recommended next wave. Reasons:

1. raccoon-cli as architecture guardian needs to catch up with the behavioral changes
2. Codegen improvements compound and accelerate future waves
3. The behavioral model is validated and hardened; codegen can safely operate adjacent to it
4. This alternation (behavior → hardening → codegen) maintains strategic balance

---

## 4. What This Recommendation Is NOT

- It is not a recommendation to open the next wave immediately
- It is not a recommendation to celebrate the behavioral wave as complete
- It is not a recommendation to defer all debts indefinitely
- It is not a recommendation to harden everything before moving forward — only the medium-risk debt warrants immediate action

---

## 5. Decision Required

The next step requires a chartering decision:

1. **Accept Option 4 → 2:** Charter a short hardening tranche, then return to codegen
2. **Accept Option 2 directly:** Skip hardening, proceed directly to codegen (accepts medium-risk debt)
3. **Accept Option 1:** Continue behavioral deepening (requires strong justification)
4. **Accept Option 3:** Open new functional line (requires strong justification and risk acceptance)
5. **Defer:** No wave opened; review again after operational feedback

This gate does not prescribe the decision — it provides the evidence for it.
