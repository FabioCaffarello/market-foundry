# Next Wave Recommendations After Post-Hardening Wave B Gate

> **Purpose:** Objective recommendation for what should follow the S173 post-hardening gate review. This is a decision document, not a celebration.

---

## 1. The Three Options

### Option A: Authorize Family 03

Expand the Wave B pattern with a third analytical family, following pattern v2 and the full checklist.

### Option B: Additional Hardening Before Family 03

Execute further structural improvements before allowing expansion.

### Option C: Deliberate Pause — No Expansion Now

Stop analytical family expansion. Focus elsewhere. Return to Wave B when external pressure demands it.

---

## 2. Assessment Against Each Option

### Option A: Family 03

**Arguments for:**
- The pattern is proven across two data shapes with controlled complexity delta
- All three mandatory hardening items are verified in code
- Friction threshold not exceeded (2 new frictions from Family 02, both low/known)
- Structural expansion cost has decreased: handler 1 field + smoke ~7 lines
- Gate criteria met cleanly — no waivers, no partial passes
- Remaining NATS subjects carry analytical value (strategies, risk assessments, executions)

**Arguments against:**
- PF-5 (no CI smoke integration) is high severity and unresolved — each family adds integration surface that CI cannot validate
- D-6 (consumer lag) and D-7 (sticky degradation) are invisible failure modes that scale with pipeline count
- Manual process is cheaper but still manual — no codegen until Family 4
- Three families may already cover the analytical surface area needed in the near term

**Risk profile:** Low structural risk, moderate operational risk (CI gap, invisible failures).

### Option B: Additional Hardening

**Arguments for:**
- PF-5 could be addressed: CI smoke integration would catch integration regressions
- D-6 and D-7 could be mitigated: consumer lag visibility and/or auto-recovery
- More hardening makes Family 03 execution even cheaper and safer

**Arguments against:**
- PF-5 is an infrastructure problem (Docker-in-Docker), not a pattern problem — hardening the pattern won't fix it
- D-6 and D-7 are runtime concerns, not expansion concerns — they exist equally at 2 families or 5
- S172 already executed the mandatory tranche; additional hardening was not committed
- Risk of over-engineering: optimizing a process that works acceptably for diminishing returns

**Risk profile:** Low risk but low value. The hardening that had structural payoff has already been done.

### Option C: Deliberate Pause

**Arguments for:**
- Three families (candles, signals, decisions) may provide sufficient analytical coverage for current needs
- Pausing allows focus on other market-foundry priorities
- No expansion pressure from external stakeholders

**Arguments against:**
- The remaining families (strategies, risk assessments, executions) are straightforward and carry the same mechanical pattern
- Pattern knowledge is fresh; pausing means re-learning expansion mechanics later
- No technical reason to stop — the gate passed cleanly

**Risk profile:** No risk, but potential opportunity cost if analytical coverage is needed soon.

---

## 3. Recommendation

**Option A: Authorize Family 03, with one binding condition.**

The gate evidence supports continued expansion. The pattern is structurally sound, the hardening reduced artisanship measurably, and the expansion cost is now mechanical rather than architectural. Pausing would be defensible but not evidence-driven — there is no signal that the pattern is failing or that the system needs rest.

### Binding Condition

Family 03 must produce a gate review before Family 04 is considered. At that gate:
- D-4 (codegen evaluation) must be formally assessed — is codegen worth the investment, or is manual expansion still acceptable?
- PF-5 (CI smoke) must be formally assessed — is the gap growing, stable, or shrinking?
- If Family 03 introduces >1 new friction not already tracked, Family 04 pauses for investigation

### Why Not Option B

The three hardening items that had structural payoff have been executed. The remaining open debts (consumer lag, sticky degradation, CI smoke) are real but they are infrastructure and runtime concerns, not pattern concerns. Hardening the pattern further won't reduce their risk. They should be addressed when their risk becomes operational, not as a gate condition for expansion.

### Why Not Option C

There is no technical or process signal indicating the expansion should stop. The gate passed cleanly. The pattern is cheaper to execute now than it was before hardening. The remaining families are mechanically similar to what has already been delivered. Stopping now would be caution without evidence.

---

## 4. Candidate for Family 03

The next family should be selected based on the same criteria as Families 01 and 02:

1. NATS subject exists and is actively published
2. Event structure is stable (no pending schema changes)
3. Write path already exists (mapper pre-implemented in writer)
4. Complexity delta is controlled and understood

Likely candidates from the remaining analytical subjects:

| Family | NATS Subject | Complexity Delta | Notes |
|--------|-------------|------------------|-------|
| Strategies | strategies.* | Low — similar to signals | Strategy metadata, entry/exit conditions |
| Risk Assessments | risk_assessments.* | Low-Medium — may include nested risk factors | Position sizing, risk metrics |
| Executions | executions.* | Medium — may include order details, fill data | Trade execution records, fill prices |

The family definition stage should assess which candidate provides the most analytical value with the smallest complexity delta, consistent with the Wave B selection criteria.

---

## 5. What This Recommendation Does NOT Cover

- Which specific family to implement (requires definition stage)
- CI smoke integration (infrastructure concern, not pattern concern)
- Consumer lag or auto-recovery (runtime concern, not expansion concern)
- Codegen implementation (evaluation committed at Family 4, not Family 3)
- Any changes to the operational pipeline
- Any horizontal refactoring of existing families
