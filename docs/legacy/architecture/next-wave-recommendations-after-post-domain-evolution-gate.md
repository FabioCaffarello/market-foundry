# Next-Wave Recommendations After Post-Domain-Evolution Gate

**Gate:** S238
**Date:** 2026-03-20
**Context:** Charter S233–S237 delivered domain depth; breadth criteria deferred.

---

## 1. Strategic Options

### Option A: Domain Breadth Charter (Feature Evolution)

**Objective:** Achieve the original ≥2 evaluator types per domain, now on the richer semantic foundation.

**Scope:**
- Decision evaluator #2 (e.g., volume spike, MACD crossover, or bollinger breach)
- Strategy resolver #2 (e.g., momentum entry, breakout entry)
- Risk evaluator #2 (e.g., volatility-adjusted sizing, drawdown limiter)
- Pipeline proof for each new family through derive → store → analytical

**Pros:**
- Directly closes the S233 charter's unmet breadth criteria.
- The depth work (severity, rationale, traceability) gives new evaluators richer semantics from day one.
- Validates that the patterns generalize beyond the RSI/MeanReversion/PositionExposure trio.
- Codegen infrastructure already supports multi-family; new families prove the codegen path.

**Cons:**
- Does not address infrastructure debts (CI, performance, production readiness).
- New families may surface model tensions that require domain model amendments.

**Estimated scope:** 4–5 stages (charter + 3 families + integration proof).

### Option B: Codegen and Generated-Path Evolution

**Objective:** Prove that codegen can generate full family pipelines, reducing the cost of breadth expansion.

**Scope:**
- Codegen template expansion for decision/strategy/risk families.
- Generated pipeline entry, writer mappers, reader queries, and HTTP handlers.
- Golden snapshot validation for generated code.
- At least one fully codegen-driven family as proof.

**Pros:**
- Reduces marginal cost of each new family dramatically.
- Forces pattern standardization across families.
- Directly enables faster breadth expansion afterward.

**Cons:**
- Meta-work: generates infrastructure for future features, not features themselves.
- Risk of over-engineering codegen before the patterns are proven with ≥2 families.
- Codegen template complexity may introduce its own debt.

**Estimated scope:** 3–4 stages.

### Option C: Short Hardening Tranche

**Objective:** Close specific test and CI gaps before expanding further.

**Scope:**
- Strategy domain test parity (multi-symbol isolation, decision input preservation).
- Inter-actor chain integration test (decision→strategy→risk in single harness).
- Correlation/causation ID propagation verification.
- Risk evaluator boundary condition tests.
- Position sizing confidence scaling justification.

**Pros:**
- Low scope, high confidence gain.
- Directly addresses the test coverage gaps identified in the gate review.
- Makes the foundation more trustworthy before adding breadth.

**Cons:**
- Does not add functional value.
- Delays breadth achievement further.
- Some gaps are low severity and may not justify dedicated stages.

**Estimated scope:** 1–2 stages.

---

## 2. Recommendation

**Option A (Domain Breadth) with a preparatory mini-tranche from Option C.**

### Rationale

1. **Breadth is the primary unmet objective.** The charter pivoted to depth, which was correct, but breadth remains the original goal and the most impactful next step.

2. **Depth makes breadth cheaper.** Severity, rationale, and traceability patterns are established. A second evaluator family inherits them.

3. **A short hardening pass first (1 stage) closes the highest-priority test gaps** — specifically strategy test parity and the inter-actor chain test. This ensures the foundation is solid before adding new families.

4. **Codegen evolution (Option B) is premature.** With only one family per domain, the patterns aren't proven enough to generalize into codegen templates. After ≥2 families exist, the commonalities will be clearer.

### Proposed Sequence

| Stage | Focus | Type |
|-------|-------|------|
| S239 | Test coverage hardening (strategy parity + actor chain test) | Hardening |
| S240 | Breadth charter definition and scope freeze | Governance |
| S241 | Decision evaluator #2 (new signal type) | Feature |
| S242 | Strategy resolver #2 (new resolution logic) | Feature |
| S243 | Risk evaluator #2 (new risk model) | Feature |
| S244 | Breadth pipeline proof + integration | Integration |
| S245 | Breadth gate evaluation | Governance |

### Guard Rails for Next Wave

1. **Charter amendment discipline.** If the scope pivots, document the amendment formally before proceeding.
2. **Breadth criteria are non-negotiable.** The next charter must deliver ≥2 types per domain or explicitly amend the criteria with justification.
3. **Hardening budget remains ≤20%.** Feature stages must focus on features.
4. **No severity-dependent logic until data exists.** Severity remains observability-only until evidence justifies activation.
5. **No infrastructure expansion.** CI, deployment, and monitoring improvements are separate charter concerns.

---

## 3. What NOT to Do Next

1. **Do not open another depth charter.** The current depth is sufficient. Adding severity levels, more metadata, or rationale formats would be over-engineering without breadth to validate the patterns.

2. **Do not invest in codegen before ≥2 families exist.** Template generalization without proven patterns leads to premature abstraction.

3. **Do not attempt production readiness.** The system needs feature breadth before deployment hardening makes sense.

4. **Do not merge documentation cleanup into feature work.** Documentation entropy is a real debt but requires its own bounded charter.

5. **Do not start the next charter without closing this gate.** S238 must be formally complete before S239 begins.

---

## 4. Decision Required

The next action is a formal decision:

- **Accept** the recommendation (hardening mini-tranche → breadth charter).
- **Modify** the sequence (e.g., skip hardening, reorder families, change scope).
- **Defer** and pursue a different direction entirely.

This decision should be recorded in the S238 stage report.
