# Next Wave Recommendations After CC-02

> Evidence-based recommendation for the next wave of market-foundry development after closing the CC-02 extensibility proof.

---

## 1. Decision Context

CC-01 proved horizontal scaling (config-driven, zero code changes).
CC-02 proved code extensibility (new signal family, predictable friction).

Together, these two capabilities establish that market-foundry:
- Scales by configuration (symbols)
- Extends by playbook (families)
- Governs debt by evidence (threshold-based triggers)

The architecture no longer needs to prove structural properties. The question shifts from "can we extend?" to "what should we build next, and why?"

---

## 2. Options Evaluated

### Option A: CC-03 — Third Signal Family (Boilerplate Bundling)

**What:** Add a third signal family (e.g., MACD, Bollinger Bands, Volume Anomaly) to trigger N=3 threshold and bundle CF-08 + CF-11 + CF-03 actor refactors.

**Pros:**
- Triggers all three converging debts naturally
- Produces generic `SignalSamplerActor`, map-based registry, actor correlation middleware
- Validates that refactored infrastructure works under real new-family pressure
- After CC-03, adding signal family N+1 costs ~240 lines (domain logic only), down from ~414

**Cons:**
- Still within signal domain — doesn't prove cross-domain extensibility
- Third signal family adds limited product value (paper trading doesn't need 3 signals)
- Risk of refactoring for refactoring's sake if the third family is chosen only to trigger N=3

**Verdict:** **Recommended only if the third family has product justification.** If chosen purely to trigger N=3, the refactors are better done as standalone hardening with synthetic test coverage.

### Option B: Cross-Domain Capability (e.g., New Decision Type)

**What:** Add a new decision type or strategy resolver that exercises the decision/strategy/risk pipeline with different logic, proving extensibility beyond signal domain.

**Pros:**
- Tests a completely different extensibility axis
- Decision/strategy/risk domains haven't been extended yet
- Proves (or disproves) that the playbook pattern generalizes across domains
- Higher product value: different decision logic = different trading behavior

**Cons:**
- Higher risk: cross-domain wiring is less tested than signal-only
- May expose friction in layers that CC-02 didn't touch
- Requires understanding of decision → strategy → risk → execution chain

**Verdict:** **Valuable as a future capability, but higher risk.** Best attempted after signal-domain friction is resolved (CC-03) so that boilerplate noise doesn't mask cross-domain friction.

### Option C: Targeted Hardening (CF-08 + CF-11 + CF-03 Without New Family)

**What:** Execute the bundled refactors (generic actor, map registry, actor correlation middleware) without introducing a new family. Validate with existing RSI + EMA crossover.

**Pros:**
- Addresses the three converging debts directly
- Lower risk than new family (no new domain logic)
- Prepares the ground for fast N=3+ additions later
- Can be scoped to a single stage (~5–7 hours)

**Cons:**
- Refactoring without a consumer: the generic actor is validated only by existing families, not by a new one using it from scratch
- Breaks the governance principle: "execute at trigger, not before"
- No new capability delivered

**Verdict:** **Not recommended as a standalone wave.** The refactors are best validated by a real new family using them. However, if the next wave is product-oriented (Option D), this hardening could precede it as a single preparatory stage.

### Option D: Product Wave — Concrete Operational Value

**What:** Shift from architectural proof to product delivery. Examples:
- Multi-timeframe signal correlation (combine 60s + 300s signals)
- Alerting/notification surface (operational value beyond query endpoints)
- MarketMonkey absorption (bring external tool into the monorepo)
- Live venue adapter activation (move beyond paper trading)
- Backtesting capability (replay historical data through pipeline)

**Pros:**
- Delivers tangible operational or product value
- Architecture has been proven sufficiently — further proofs have diminishing returns
- Shifts the project from "can we build?" to "what should we build?"
- Naturally triggers deferred debts if the product feature touches the right layers

**Cons:**
- Product features under boilerplate friction (N=2 without generic actor) may be slower than necessary
- If the product feature requires a third signal family, CC-03 refactors become prerequisite friction
- Risk of structural problems if product feature pressure bypasses governance

**Verdict:** **Strongest strategic option, with a qualification.** If the product feature requires a third signal family, do CC-03 first (Option A). If it operates in a different domain or extends existing families, proceed directly.

---

## 3. Recommendation

### Primary Path: Option D (Product Wave) with Optional Option C Preparation

**Rationale:**

1. **Diminishing returns on architectural proof.** CC-01 proved scaling. CC-02 proved extensibility. A third proof (CC-03) within the same domain adds confidence but not capability. The architecture is ready for product work.

2. **Product pressure is the best test.** Real product requirements expose friction that controlled capabilities cannot anticipate. Building something the operator actually needs validates the architecture under authentic pressure.

3. **Governance works.** The threshold-based trigger model has been accurate across two capability waves. Trust it: when N=3 arrives naturally (because a product feature needs a third family), the bundled refactors will be justified and validated by real use.

4. **The hardening option (C) is available as a single preparatory stage.** If the chosen product feature clearly benefits from generic actor infrastructure, execute CF-08 + CF-11 + CF-03 actor as a single ~5–7 hour stage before the product wave. This is pragmatic, not impulsive.

### Decision Tree

```
Is the next product feature a third signal family?
├── Yes → Execute CC-03 (Option A) as first stage, then product wave
│         CC-03 triggers N=3 refactors naturally
│
└── No → Does the product feature benefit from generic actor infra?
    ├── Yes → Execute hardening (Option C) as prep stage, then product wave
    │         ~5-7 hours, validated by existing families
    │
    └── No → Proceed directly to product wave (Option D)
              Deferred debts remain deferred until their triggers fire
```

### Candidate Product Features (Priority Order)

| Feature | Strategic Value | Architectural Risk | Dependencies |
|---------|---------------|-------------------|-------------|
| **Multi-timeframe correlation** | High — core trading capability; combines existing 60s + 300s signals | Low — within signal domain; exercises existing query surface | None |
| **Backtesting/replay** | High — validates strategy without live market; strong operator demand | Medium — new data path (historical → pipeline); may need new runtime | May need new NATS stream pattern |
| **Live venue adapter** | High — moves beyond paper trading; real execution | High — real money; safety model must be production-grade | D5 (failure recovery), kill switch refinement |
| **Alerting/notification** | Medium — operational visibility beyond query endpoints | Low — read-only consumer of existing events | None |
| **MarketMonkey absorption** | Medium — consolidates external tool into monorepo | Medium — integration complexity; different codebase conventions | Governance alignment |

---

## 4. Anti-Patterns to Avoid

1. **CC-03 for refactoring's sake.** Don't add a third signal family just to trigger N=3. If the family has no product justification, the refactors are better done standalone.

2. **Horizontal refactoring wave.** Don't batch all deferred debts (CF-02, CF-12, D4, D5, D6) into a "cleanup sprint." Each has a specific trigger for a reason.

3. **Premature production hardening.** D5 (failure recovery) and D6 (soak testing) are pre-production requirements. The Foundry is in paper-trading mode. Don't optimize for production resilience before production is on the roadmap.

4. **Over-abstracting from two examples.** Two signal families provide a pattern, not a universal law. The generic actor should be designed at N=3, not speculatively generalized from N=2.

5. **Ignoring the product question.** The architecture exists to serve a product. After two capability waves, the highest-value question is "what should the Foundry do?" not "how clean is the Foundry's code?"

---

## 5. Success Criteria for the Next Wave

Regardless of which option is chosen, the next wave should:

1. **Deliver measurable value** — operational capability, not just structural proof
2. **Naturally trigger at most 1–2 deferred debts** — evidence that governance works
3. **Produce a friction capture** — continue the evidence loop
4. **Complete within 5–6 stages** — bounded scope, clear exit
5. **Leave the Foundry closer to product** — not just closer to "clean"

---

## 6. Closing Position

Market-foundry has earned the right to build product. Two controlled capability waves have established that the architecture scales, extends, and governs friction by evidence. The next wave should be driven by what the operator needs, not by what the codebase could theoretically improve.

The deferred debts are real, scoped, and trigger-gated. They will be resolved when — and only when — evidence demands it. This is not neglect; it is discipline.
