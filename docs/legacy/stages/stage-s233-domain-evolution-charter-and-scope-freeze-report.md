# Stage S233 — Domain Evolution Charter and Scope Freeze Report

**Date:** 2026-03-20
**Objective:** Define and freeze the charter for the next wave of evolution, focused on decision, strategy, and risk domain deepening
**Verdict:** **COMPLETE — Charter open, scope frozen, ready for S234**

---

## 1. Executive Summary

S233 formalizes the next charter of evolution for market-foundry. Following the S232 clean-pass gate authorization, this stage defines what the system will build next, what it will not touch, and under what conditions work proceeds or stops.

**Charter:** Domain Logic Depth — Decision, Strategy, and Risk Evolution.

The charter focuses exclusively on deepening the functional value of three downstream domains — decision, strategy, and risk — by adding new evaluators and resolvers. Infrastructure expansion, documentation cleanup, and new domain families are explicitly frozen.

**Key decisions:**
- Scope frozen to decision/strategy/risk domain logic evolution only.
- Priority order: Decision (P0) → Strategy (P1) → Risk (P2).
- Hardening capped at 20% of effort, subordinated to feature pull.
- Six entry conditions all satisfied; implementation authorized for S234.

---

## 2. Charter Summary

### Central Objective

Move the decision, strategy, and risk domains from single-implementation baselines to multi-evaluator/multi-resolver depth, proving that the pipeline architecture supports real trading logic diversity.

### Current State → Target State

| Domain | Current | Target |
|--------|---------|--------|
| Decision | 1 evaluator (RSI oversold) | ≥2 evaluators (multi-signal capable) |
| Strategy | 1 resolver (mean reversion) | ≥2 resolvers (multi-decision capable) |
| Risk | 1 evaluator (position exposure) | ≥2 evaluators (multi-constraint capable) |

### Priority Chain

Decision first (P0) because strategies and risk assessments depend on richer decision output. The dependency chain signal → **decision** → **strategy** → **risk** → execution dictates the build order.

---

## 3. Scope Boundaries

### In Scope

1. New evaluator/resolver implementations in decision, strategy, risk.
2. Domain model enrichments pulled by concrete feature needs.
3. Registry expansions for new types.
4. Adapter-layer adjustments (NATS, ClickHouse, HTTP) pulled by domain changes.
5. Tests (unit + integration) for all new logic.
6. Lightweight CI hardening (e.g., `make test-integration` in remote CI) when feature-pulled.

### Out of Scope (Frozen)

1. New domain families (signal, indicator, execution changes).
2. New infrastructure (services, databases, brokers).
3. Documentation cleanup waves.
4. marketmonkey absorption.
5. Operational readiness (observability, deployment, monitoring).
6. raccoon-cli comprehensive audit.
7. Code style or formatting sweeps.
8. Package restructuring.

### Hardening Rule

Every hardening change must trace to a specific feature requirement. Speculative hardening is prohibited. Total hardening effort capped at 20% of stage effort.

---

## 4. Entry, Exit, and Stop Conditions

### Entry (All Satisfied)

| Condition | Status |
|-----------|--------|
| S232 clean-pass gate | ✅ PASS |
| Charter frozen | ✅ This stage |
| Scope defined | ✅ Permitted/prohibited document |
| `quality-gate-ci` at 0 errors | ✅ 84/84 |
| Remote CI green | ✅ Run 23365571775 |
| Release tag on green commit | ✅ v0.1.0-s231 |

### Exit (All Required for Charter Closure)

1. ≥2 decision evaluator types.
2. ≥2 strategy resolver types.
3. ≥2 risk evaluator types.
4. Full pipeline proof for each new implementation.
5. All CI green (local + remote).
6. No regressions in existing implementations.
7. Formal gate stage completed.
8. Hardening budget ≤20%.

### Stop Conditions

- **Hard stop:** Scope violation, CI regression, hardening overrun, domain model break.
- **Soft stop:** Complexity escalation, upstream dependency discovered, model tension.
- **Info stop:** Tech debt noticed, optimization opportunity, doc gap — log and continue.

---

## 5. Suggested Stage Sequence

| Stage | Focus | Type |
|-------|-------|------|
| S233 | Charter and scope freeze | Governance ✅ |
| S234 | Decision evaluator #2 | Feature |
| S235 | Strategy resolver #2 | Feature |
| S236 | Risk evaluator #2 | Feature |
| S237 | Pipeline integration proof + CI hardening | Integration |
| S238 | Gate evaluation | Governance |

---

## 6. Non-Objectives and Guard Rails

This charter explicitly does **not** aim to:

- Achieve production readiness.
- Complete feature coverage for all trading scenarios.
- Replace or absorb external systems.
- Expand the analytical layer beyond the three charter domains.
- Clean up accumulated documentation entropy.

Guard rails enforced:

- No new domain families opened.
- No new infrastructure wave.
- No documentation cleanup reopened.
- No charter/implementation mixing in this stage.
- Every stage must map changes to permitted categories.

---

## 7. S234 Preparation

The following preparation is recommended before S234 begins:

1. **Review the decision evaluator pattern.** Understand the RSI oversold evaluator's actor, event flow, registry entry, and test structure in detail.
2. **Design decision evaluator #2.** Choose the evaluator type (e.g., momentum crossover, composite multi-signal). Define input signals, outcome logic, and confidence model.
3. **Map the derive actor pattern.** Confirm the new evaluator follows the existing derive actor template.
4. **Define the test strategy.** Unit tests for evaluator logic; integration test for the full derive → store → analytical pipeline.

---

## 8. Artifacts Produced

| Artifact | Path |
|----------|------|
| Charter and scope freeze | `docs/architecture/domain-evolution-charter-and-scope-freeze.md` |
| Permitted vs. prohibited changes | `docs/architecture/domain-evolution-permitted-vs-prohibited-changes.md` |
| Entry, exit, and stop conditions | `docs/architecture/domain-evolution-entry-exit-and-stop-conditions.md` |
| This report | `docs/stages/stage-s233-domain-evolution-charter-and-scope-freeze-report.md` |

---

## 9. Stage Closure

S233 is a governance stage. No code was changed. The charter is formally open, the scope is frozen, and all entry conditions for S234 are satisfied.

**Next action:** Begin S234 — Decision evaluator #2 implementation.
