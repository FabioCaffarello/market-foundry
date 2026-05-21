# Next Wave Recommendations After Family 03 Wave B Gate

> Decision document: what comes after the third Wave B family expansion.

---

## Governing Principle

The Wave B expansion is governed by accumulated evidence, not by momentum. Each gate evaluates whether the next step should be expansion, hardening, or pause. The default is **not** to expand — the default is to ask whether expansion is justified.

---

## 1. The Three Options

### Option A: Family 04 (Risk Assessments)

Expand the read path to layer 5 of 6, adding the risk_assessments analytical family.

**Arguments for:**
- Continues the contiguous vertical coverage principle (layers 1–4 covered, layer 5 is next).
- Write path already pre-staged (migration 005, mapper, pipeline entry all exist).
- 9-artifact template proven 3 times — the expansion cost is mechanical.
- Introduces 2 new structural tests: 4 JSON columns (current ceiling is 3) and free-text column (`rationale`).
- Only 1 triggered item (D-4 codegen, non-blocking) and 0 blocking frictions.
- Risk profile is lower than Family 03 (fewer unknowns).

**Arguments against:**
- Mechanical duplication continues to accumulate (~1000 lines at 5 families).
- No hardening tranche has been executed since S172 (2 families ago).
- 2 medium-severity operational debts remain unaddressed (DEF-U3: consumer lag visibility, DEF-U4: sticky degradation).
- Adding the 5th family without codegen increases the eventual codegen effort.

**Risk profile:** Low. The pattern is proven. The new structural tests (4 JSON columns, free-text) are bounded in scope. Write path is pre-staged.

### Option B: Hardening tranche before Family 04

Pause expansion to address accumulated frictions and operational debts.

**Arguments for:**
- 2 medium-severity operational debts (DEF-U3, DEF-U4) have been carried since Wave A.
- Handler duplication is at ~320 lines (4 families × ~80 lines).
- A hardening tranche would reduce debt count before the final 2 families.

**Arguments against:**
- No triggered friction demands immediate hardening. Friction count (2 new in F-03) is within threshold.
- The medium-severity debts (consumer lag, sticky degradation) are operational, not structural — they don't block expansion.
- Codegen (the largest hardening item) is explicitly not cost-effective until Family 06.
- Hardening after Family 05 would address more accumulated duplication with a single effort.
- The prior hardening tranche (S172) was triggered by committed obligations (H-1, H-2, H-3), not by general debt accumulation.

**Risk profile:** Minimal risk, but minimal payoff. The hardening items that would be addressed are low-to-medium severity and don't block the pattern.

### Option C: Deliberate pause

Stop Wave B expansion entirely. Do not add Family 04. Redirect effort to other system areas.

**Arguments for:**
- 4 of 6 layers is substantial coverage — the most critical analytical layers (candles, signals, decisions, strategies) are available.
- Allows focus on other system priorities.
- Reduces risk of pattern fatigue.

**Arguments against:**
- The remaining 2 families (risk_assessments, executions) complete the analytical pipeline. Stopping at 4 leaves the terminal layers uncovered.
- Write path is already fully pre-staged — the infrastructure investment is partially wasted if read path coverage stops here.
- No evidence of pattern fatigue or structural limits. The pattern is sustainable through Family 05.
- Pausing now means the codegen trigger (Family 06) would never fire, leaving the evaluation as theoretical.

**Risk profile:** No execution risk, but opportunity cost — the remaining 2 families are the cheapest they will ever be to implement, given the pre-staged write path and proven pattern.

---

## 2. Assessment Against Evidence

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| Evidence supports it? | Yes — pattern proven, no blockers | Partially — no triggers demand it | Partially — no evidence against, but no evidence for stopping |
| Addresses real friction? | Indirectly (proves 4 JSON, free-text) | Directly (operational debts) | No |
| Risk of proceeding? | Low | Minimal | None |
| Opportunity cost? | Low | Medium (delays completion) | High (leaves 2 layers uncovered) |
| Precedent in prior gates? | Consistent — F-02→F-03 followed same logic | S172 was trigger-driven, not general | No precedent for pause without cause |

---

## 3. Recommendation

**Option A: Authorize Family 04 (Risk Assessments).**

The gate evidence supports continued expansion. The pattern is structurally sound, the friction count is within threshold, no triggers demand hardening, and the next family introduces bounded structural novelty (4 JSON columns, free-text column) that validates the pattern at a new ceiling.

### Why not Option B (hardening)?

The prior hardening tranches (S166, S172) were triggered by committed obligations and measurable friction. No equivalent trigger exists now. The medium-severity operational debts (consumer lag, sticky degradation) are real but do not block expansion and are not worsened by adding Family 04. Hardening for its own sake, without a triggering condition, sets a precedent that undermines the evidence-based governance model.

### Why not Option C (pause)?

No evidence supports stopping. The pattern is healthy, the remaining families are bounded in scope, and the write path is already invested. Pausing at 4 of 6 layers would leave the analytical pipeline incomplete without cause.

---

## 4. Candidate for Family 04

| Candidate | Layer | Readiness | Verdict |
|-----------|-------|-----------|---------|
| **Risk Assessments** | 5 | High — migration, mapper, pipeline pre-staged | **Selected** |
| Executions | 6 | High — pre-staged, but terminal layer (skip 1) | Deferred to Family 05 |

**Why Risk Assessments:**
- Layer 5 — contiguous with layers 1–4.
- Introduces 4 JSON columns (signal_inputs, decision_inputs, parameters, metadata) — proves JSON ceiling scales.
- Introduces free-text column (`rationale`) — new column type, bounded structural test.
- 20 DDL columns (largest schema yet) — tests pattern at higher column count.
- Dependency chain respected — risk assessments depend on decisions and strategies (both covered).

**What Family 04 must validate:**
- 4 JSON columns parsed without structural friction.
- Free-text `rationale` column round-trips correctly.
- 20-column schema coherence holds across DDL, mapper, reader, response.
- Per-family expansion cost remains at ~450–500 lines.
- No new medium-or-higher-severity frictions introduced.

**What Family 04 must NOT do:**
- Introduce cross-family queries.
- Modify existing families.
- Change the write path.
- Implement codegen (deferred to Family 06 boundary).
- Add pagination or aggregation.

---

## 5. Binding conditions

1. Family 04 follows pattern v2 (9-artifact template, struct DI, smoke helper, canonical naming).
2. >2 new frictions in Family 04 triggers mandatory hardening before Family 05.
3. Codegen remains mandatory before Family 06.
4. Family 05 (Executions) requires its own gate — this recommendation authorizes exactly one family.
5. D-4 codegen evaluation is documented as resolved; the implementation trigger is committed.

---

## 6. Stop conditions

Expansion halts immediately if any of the following occur:

1. >2 new medium-or-higher-severity frictions in Family 04.
2. Any correctness regression in existing families (candles, signals, decisions, strategies).
3. CI becomes unreliable or analytical smoke tests fail.
4. Schema coherence verification fails for any family.
5. Writer pipeline degradation correlated with read path changes.

---

## 7. What This Recommendation Does NOT Cover

- Family 05 authorization — requires its own gate after Family 04.
- Codegen timing — committed to "before Family 06" but not scheduled.
- Operational debt resolution (DEF-U3, DEF-U4) — tracked but not blocking.
- Cross-family query capability — out of Wave B scope entirely.
- Post-Wave-B architecture — what happens after all 6 layers are covered is a separate decision.
