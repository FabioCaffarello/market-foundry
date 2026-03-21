# Next Wave Recommendations After Post-Paper Execution Gate

**Stage:** S269
**Date:** 2026-03-21
**Prerequisite:** Post-Paper Execution Gate PASSED with constraints (S264–S268)

---

## 1. Strategic Context

The Foundry has completed four consecutive waves:

1. **Breadth wave** (S241–S244) — expanded domain from signal+evidence to all 6 layers.
2. **Behavioral wave** (S249–S257) — hardened domain logic with 47 behavioral tests, severity scaling.
3. **Codegen reentry wave** (S258–S262) — reconciled specs, expanded governance to 22 artifacts.
4. **Paper execution wave** (S264–S268) — closed first operational loop from signal to paper order.

The Foundry now has a functioning domain pipeline that transforms market signals into paper orders through coherent, auditable stages. But the operational envelope — safety controls, observability, persistence — has gaps that separate "domain logic works" from "system is operationally trustworthy."

---

## 2. Options Evaluated

### Option A: Deep Paper Execution Expansion

**Description:** Expand the paper execution surface: multi-symbol, multi-timeframe, concurrent scenarios, partial fill simulation, more strategy families executing paper orders.

**Pros:**
- Increases coverage and realism of paper execution.
- Exercises pipeline under more diverse conditions.
- Natural next step for feature evolution.

**Cons:**
- Expands on an incomplete safety foundation (OD-PE1 unresolved).
- Adding breadth before closing safety gaps risks normalizing the gap.
- Multi-symbol and concurrency add complexity that may mask issues.

**Verdict:** NOT RECOMMENDED as immediate next step. Expansion should follow safety closure, not precede it.

### Option B: Venue Readiness Charter

**Description:** Open a charter to prepare for real venue integration: venue adapter contracts, order routing, real fill handling, authentication.

**Pros:**
- Moves toward the ultimate goal of live trading.
- Motivates resolution of safety and observability gaps.

**Cons:**
- Premature. SafetyGate is not proven end-to-end even in paper mode.
- Real venue requires OMS, portfolio tracking, PnL — all currently deferred.
- The gap between "paper loop works in tests" and "venue readiness" is too large for a single wave.
- Risks opening scope that is impossible to close within a disciplined charter.

**Verdict:** NOT RECOMMENDED. The prerequisite safety and observability work is not done.

### Option C: Return to Codegen/Generated Path

**Description:** Resume codegen expansion: store consumers, layer starters, mappers, config methods.

**Pros:**
- The codegen system is healthy and proven.
- Reduces manual boilerplate for future families.

**Cons:**
- Fourth infrastructure-focused decision in a row (even though paper execution was nominally "feature evolution").
- Does not address the most consequential open debts (safety, observability).
- Marginal ROI continues to decline.

**Verdict:** NOT RECOMMENDED as primary focus. Codegen improvements can proceed as side-effects.

### Option D: Bounded Operational Hardening Tranche

**Description:** A short, targeted tranche (2–4 stages) focused on closing the highest-severity debts from the paper execution wave:

1. **Wire SafetyGate into PaperOrderEvaluatorActor** — prove kill switch and staleness guard work end-to-end in the actor chain.
2. **Prove KV materialization for execution events** — paper orders must be queryable, not just published.
3. **Extend ClickHouse round-trip to execution** — prove paper execution events survive serialization and storage.
4. **Prove ControlGate runtime behavior** — demonstrate that flipping the kill switch blocks subsequent paper orders.

**Pros:**
- Directly addresses the highest-severity debt (OD-PE1).
- Converts paper execution from "proof of concept" to "operationally trustworthy in paper mode."
- Small scope — 2–4 stages with clear exit criteria.
- Does not expand features or open new surfaces.
- Prepares the foundation for either venue readiness or feature expansion.

**Cons:**
- Another round of infrastructure-adjacent work before feature delivery.
- The Foundry has now spent 4 waves + 1 tranche on infrastructure/hardening.
- Risk of scope creep if "hardening" expands to include new features.

**Verdict:** RECOMMENDED. The debts are specific, the scope is bounded, and the value is concrete.

---

## 3. Recommendation: Option D — Bounded Operational Hardening Tranche

### Rationale

The paper execution wave proved the domain loop works. But it left the operational envelope incomplete:

- **SafetyGate (OD-PE1)** is the highest-severity debt. A paper execution system where the kill switch is not enforced in the actual execution path is a test harness, not a system.
- **KV materialization (OD-PE3)** and **ClickHouse round-trip (OD-PE4)** are required for observability. Paper orders that cannot be queried or stored have no operational value.
- **ControlGate (OD-PE5)** is the runtime safety mechanism. Without end-to-end proof, the kill switch is theoretical.

These are not new features — they are the completion of what S264–S268 started. The tranche should be explicitly scoped as "closing the paper execution loop properly" rather than "starting a new wave."

### Suggested Tranche Scope

| Stage | Objective | Exit Criteria |
|-------|-----------|---------------|
| T1 | Wire SafetyGate into actor chain | Kill switch blocks paper orders in closed-loop test; staleness guard rejects old intents |
| T2 | KV materialization for execution events | Paper orders queryable from KV bucket; partition and dedup keys verified |
| T3 | ClickHouse round-trip for execution | Paper execution events survive write → read with field preservation |
| T4 (optional) | ControlGate runtime toggle | Flip kill switch mid-scenario; prove subsequent orders blocked |

### Constraints on the Tranche

- **No new domain surfaces.** No new signal families, strategies, risk models, or execution modes.
- **No venue real.** Paper mode only throughout.
- **No scope creep.** If a gap is found that is outside the 4 objectives, document it as debt — do not expand scope.
- **Hardening budget: 15%.** Up to 15% of implementation time may address secondary cleanup (S267 report, test consolidation), but 85% goes to the 4 objectives.
- **Clear exit gate.** The tranche ends with a gate review confirming SafetyGate, KV, ClickHouse, and ControlGate are proven end-to-end.

### Success Criteria

- SafetyGate.Check() called in PaperOrderEvaluatorActor before publishing.
- Kill switch (halted state) blocks paper order submission in end-to-end test.
- Staleness guard rejects old intents in end-to-end test.
- Paper orders materialize in NATS KV and are queryable by partition key.
- Paper execution events survive ClickHouse write → read round-trip.
- S267 governance gap resolved (retroactive report or folded into tranche).
- Zero regressions in existing 47 behavioral tests and 12 end-to-end scenarios.

---

## 4. After the Tranche

Once the hardening tranche closes, the Foundry will have:

- A closed operational loop with proven safety controls.
- Observable paper execution events in KV and ClickHouse.
- Runtime kill switch capability.
- Causal traceability from signal through persisted paper order.

At that point, the next strategic choice becomes:

1. **Feature evolution** — new signal families (MACD, VWAP, ATR), new decision strategies, enhanced risk models. This tests whether the infrastructure supports rapid feature delivery.
2. **Venue readiness exploration** — not a full charter, but a scoping exercise to understand the gap between paper execution and real venue integration. This informs whether venue readiness is a single wave or a multi-wave effort.
3. **Operational maturity** — multi-symbol, concurrent scenarios, performance budgets, configurable scaling. This deepens paper execution without opening venue real.

The recommendation after the tranche depends on its findings. If SafetyGate integration reveals architectural issues, the Foundry should address those before expanding. If integration is clean, feature evolution is the highest-value next step.

---

## 5. What Should NOT Happen Next

1. **Do not open venue real.** The gap between "paper loop with safety controls" and "real venue" includes OMS, portfolio, PnL, authentication, rate limiting, error handling, and regulatory compliance. This is not a single wave.
2. **Do not skip the hardening tranche.** SafetyGate integration is the difference between a proof of concept and an operational system. Skipping it to chase features normalizes the gap.
3. **Do not expand paper execution breadth before closing safety gaps.** More scenarios on an incomplete safety foundation creates more untested surface, not more safety.
4. **Do not treat the tranche as a new wave.** It should be explicitly scoped as "closing paper execution properly" — charter discipline applies, but this is completion, not initiation.
5. **Do not use the tranche to introduce new features disguised as hardening.** Every change must trace back to OD-PE1 through OD-PE5.
