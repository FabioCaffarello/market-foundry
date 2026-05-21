# Domain Evolution Charter and Scope Freeze

**Charter:** Domain Logic Depth — Decision, Strategy, and Risk Evolution
**Stage:** S233 (charter definition) → S234+ (implementation)
**Date:** 2026-03-20
**Authorization:** S232 clean-pass gate (all 5 criteria PASS)
**Status:** OPEN — Scope frozen

---

## 1. Charter Objective

Deepen the functional value of the three downstream domains — **decision**, **strategy**, and **risk** — by evolving their domain logic, evaluators, and resolvers beyond the current single-implementation baselines, while keeping the mechanical foundation stable.

This charter is **not** about infrastructure, cleanup, or platform expansion. It is about making the existing domain pipelines produce richer, more realistic trading logic that demonstrates the system's value proposition.

---

## 2. Current Baseline

| Domain | Current Implementation | Event | Maturity |
|--------|----------------------|-------|----------|
| **Decision** | RSI oversold evaluator | `decision_evaluated` | Minimal — single signal, single outcome path |
| **Strategy** | Mean reversion entry resolver | `strategy_resolved` | Minimal — single strategy type, single direction logic |
| **Risk** | Position exposure evaluator | `risk_assessed` | Basic — single constraint model (max position, max exposure, stop distance) |

Each domain already has the full vertical slice: domain model → NATS adapter (publisher, consumer, gateway, KV store) → ClickHouse reader → projection actor → HTTP route → application use case.

The pipeline architecture is proven. The domain logic is shallow.

---

## 3. Charter Scope — What Is In

### 3.1 Decision Domain Evolution

- Add evaluators beyond RSI oversold (e.g., multi-signal decision logic, momentum-based evaluation, composite signal evaluation).
- Enrich the `Decision` model if needed to support richer outcome semantics (e.g., confidence scoring, multi-signal attribution).
- Expand decision type registry with new decision specs.
- Unit and integration tests for all new evaluators.

### 3.2 Strategy Domain Evolution

- Add resolvers beyond mean reversion (e.g., momentum entry, trend-following, breakout strategy).
- Support multi-decision input strategies (strategies informed by more than one decision type).
- Expand strategy type registry with new strategy specs.
- Unit and integration tests for all new resolvers.

### 3.3 Risk Domain Evolution

- Expand risk evaluation beyond single-position exposure (e.g., portfolio-level risk, correlation-aware sizing, drawdown constraints).
- Enrich the `Constraints` model if the current `{MaxPositionSize, MaxExposure, StopDistance}` is insufficient for new evaluators.
- Support multi-strategy input risk assessment.
- Unit and integration tests for all new evaluators.

### 3.4 Lightweight Hardening (Feature-Pulled Only)

Hardening work is **permitted only when directly required by the feature work above**:

- If a new evaluator requires a new derive actor pattern → add it with tests.
- If integration tests are needed to validate a new pipeline path → add them.
- If `make test-integration` inclusion in remote CI is needed to gate new logic → add it.
- If codegen golden snapshots must update for new registry entries → update them.
- If `quality-gate-ci` rules need adjustment for new domain patterns → adjust them.

**Rule:** Every hardening change must trace to a specific feature requirement. No speculative hardening.

---

## 4. Charter Scope — What Is Out

See `domain-evolution-permitted-vs-prohibited-changes.md` for the complete list. Summary:

- No new domain families (signal, indicator, execution are frozen for this charter).
- No new infrastructure (no new services, no new databases, no new message brokers).
- No documentation cleanup wave.
- No marketmonkey absorption.
- No operational readiness work (observability, deployment, monitoring).
- No raccoon-cli overhaul.
- No broad CI pipeline expansion beyond what features pull.

---

## 5. Priority Order

| Priority | Domain | Rationale |
|----------|--------|-----------|
| **P0** | Decision | Foundation — strategies and risk depend on richer decision output. Start here. |
| **P1** | Strategy | Second — benefits immediately from enriched decision input. |
| **P2** | Risk | Third — benefits from both enriched decisions and strategies. |

The dependency chain is: signal → **decision** → **strategy** → **risk** → execution.

Deepening decision first creates the richest input surface for strategy and risk evolution.

---

## 6. Success Criteria

The charter is successful when **all** of the following are true:

1. **Decision breadth:** At least two distinct decision evaluator types exist (current RSI oversold + at least one new type).
2. **Strategy breadth:** At least two distinct strategy resolver types exist (current mean reversion + at least one new type).
3. **Risk breadth:** At least two distinct risk evaluator types exist (current position exposure + at least one new type).
4. **Pipeline proven:** Each new evaluator/resolver produces events that flow through the full pipeline (derive → store → analytical query).
5. **Tests pass:** `make test`, `make test-integration`, `make quality-gate-ci` all green.
6. **Remote CI green:** At least one verified-green remote CI run covering the expanded logic.
7. **No regression:** Existing evaluators/resolvers continue to work unchanged.

---

## 7. Non-Success Criteria (What Does NOT Count)

- Adding domain model fields without corresponding evaluator logic.
- Adding tests without corresponding feature logic.
- Expanding infrastructure without corresponding feature pull.
- Documentation additions without code backing.

---

## 8. Hardening Budget

The charter allocates a **maximum of 20% of stage effort** to hardening activities. This means:

- In a 5-stage implementation wave (S234–S238), at most 1 stage equivalent can be pure hardening.
- Hardening that is embedded within feature stages does not count against this budget.
- If hardening threatens to exceed 20%, the charter requires a stop-and-reassess.

---

## 9. Governance

- Each implementation stage (S234+) must state which charter objective it advances.
- Each stage must end with `make quality-gate-ci` at 0 errors.
- The charter ends with a gate stage (analogous to S232) that evaluates all success criteria.
- The charter may be amended only by a formal scope-change document that explains why.

---

## 10. Relationship to S232 Authorization

This charter is the direct response to the S232 clean-pass gate authorization. It fulfills the conditions set by S232:

- ✅ Charter document written with scope, objectives, and acceptance criteria.
- ✅ Starting baseline verified (S232 gate confirms all CI green).
- ✅ `make quality-gate-ci` confirms 0 errors (84/84 checks).
- ✅ Stage numbering established: S233 (this charter), S234+ (implementation).
