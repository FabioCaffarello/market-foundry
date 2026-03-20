# Next Wave Recommendations After Capability 01

> Stage S124 — Evidence-based recommendation for the next Market Foundry wave.
> Date: 2026-03-19

---

## 1. Decision Context

CC-01 validated the architecture's horizontal scaling properties. The next wave must test something CC-01 did not: **the architecture's extensibility to new code paths**.

CC-01 deliberately avoided new code to isolate scaling from feature complexity. That was correct for a first capability. But the architecture's real value lies in how cheaply it accommodates new signal families, decision types, strategy resolvers, and risk models. The next wave must prove this.

---

## 2. The Four Options

### Option A: Ampliar CC-01 (Expand Same Capability)

**What:** Add N=3, N=5, or N=10 symbols. Validate scaling further.

**Pros:** Low risk. Exercises the same proven paths. Natural soak test.

**Cons:** Diminishing returns. N=2 already proved the architecture scales. N=10 proves the same thing 5× more slowly. Does not test extensibility. Does not deliver new value.

**Verdict:** **Not recommended.** The architecture's scaling is validated. More symbols add operational cost without architectural signal.

---

### Option B: Nova Capacidade Controlada (New Controlled Capability)

**What:** Introduce a new code path — a new signal family, decision type, or domain actor — following the controlled capability pattern (define → implement → validate → friction capture → refactor).

**Pros:** Tests the architecture's extensibility (the key unproven property). Delivers new value. Natural trigger for deferred debts (CF-03, CF-08). Exercises the full development lifecycle, not just config changes.

**Cons:** Moderate risk — new code can introduce new bugs. Requires careful capability selection.

**Verdict:** **Recommended.** This is the natural next step per S118's decision framework: "Did delivery expose architectural pain? → Minimal → Deliver next capability."

---

### Option C: Hardening Adicional (Targeted Hardening)

**What:** Address specific debts: correlation ID middleware, composition root tests, failure recovery validation.

**Pros:** Reduces known risk. Some debts (D5: failure recovery) are genuine unknowns.

**Cons:** No new value delivered. Debts are correctly scoped with natural triggers — hardening them now is premature. Risk of slipping back into architecture-as-procrastination.

**Verdict:** **Not recommended as a standalone wave.** Debts should be addressed at their natural triggers (new actor → CF-03, new runtime → D4, pre-production → D5), not as a dedicated stage.

---

### Option D: Onda de Produto (Product Wave)

**What:** Skip the controlled capability pattern and go directly to a product-oriented feature (e.g., MarketMonkey absorption, strategy dashboard, alert system).

**Pros:** Fastest path to user-visible value.

**Cons:** Market Foundry has proven config-driven scaling but not code extensibility. Jumping to product features without proving extensibility risks discovering structural problems under product deadline pressure. One more controlled capability validates the last unproven property.

**Verdict:** **Not yet.** One more controlled capability (Option B) validates code extensibility. After that, the path to product is clear.

---

## 3. Recommended Next Wave: CC-02 — New Signal Family

### 3.1 Why a New Signal Family

A new signal family (e.g., MACD, Bollinger Bands, or Moving Average Crossover) is the ideal CC-02 because it:

1. **Introduces new code** — new sampler actor, new domain type, new publisher, new projection, new query endpoint. Tests the full extensibility path.

2. **Exercises all runtimes** — the new signal feeds into existing decision → strategy → risk → execution chain. Validates that the derive/store/gateway composition handles a second signal family.

3. **Triggers deferred debts naturally:**
   - CF-03 (correlation ID middleware) — new actor is the first consumer
   - CF-08 (client boilerplate) — new domain client can use shared usecase types
   - D4 (composition root tests) — new wiring validates composition patterns

4. **Has bounded scope** — a signal family is a single actor + publisher + projection + route. The domain boundary is clear. Risk is contained.

5. **Builds on proven patterns** — RSI signal family is the template. The new family follows the same actor lifecycle, publisher pattern, projection pattern, and query route pattern.

### 3.2 Candidate Signal Families

| Family | Complexity | Dependencies | Suitability |
|--------|-----------|-------------|-------------|
| **MACD** (Moving Average Convergence Divergence) | Medium — requires EMA computation over candle history | Needs candle history access (already available in KV) | **Good** — moderate complexity, proven data source |
| **Moving Average Crossover** | Low — two SMAs compared | Needs candle history access | **Best for CC-02** — simplest new signal, clear implementation path |
| **Bollinger Bands** | Medium — SMA + standard deviation | Needs candle history access | Good — slightly more complex than MA Crossover |
| **Volume Anomaly** | Low-Medium — compare current volume burst to historical average | Needs trade burst history | Acceptable — different data source, tests evidence → signal path |

**Recommendation:** Moving Average Crossover as CC-02. It is the simplest signal family that exercises the full extensibility path. MACD or Bollinger Bands can follow as CC-03 once the pattern is proven.

### 3.3 Expected CC-02 Deliverables

Following the controlled capability pattern (S119–S123):

1. **CC-02 Definition** — scope, success criteria, pressure points, exclusions
2. **CC-02 Implementation** — new signal sampler actor, publisher, projection, query route
3. **CC-02 Live Validation** — both signal families running concurrently for 2 symbols
4. **CC-02 Friction Capture** — evidence of what the new code path exposed
5. **CC-02 Surgical Refactors** — fixes for frictions that emerged

### 3.4 Debts to Address During CC-02

| Debt | When During CC-02 | Why |
|------|-------------------|-----|
| CF-03 (correlation ID middleware) | Implementation stage | New actor is first consumer; validates the S123 design |
| CF-08 (client boilerplate) | Implementation stage | New signal client uses shared usecase types |
| CF-02 (active symbols endpoint) | Implementation stage (if touching configctl routes) | Natural opportunity |

---

## 4. What NOT to Do Next

| Anti-pattern | Why to Avoid |
|-------------|-------------|
| **Add more symbols (N>2)** | Diminishing returns. Architecture's scaling is proven. |
| **Standalone hardening stage** | Debts have natural triggers. Hardening without a capability driver is architecture-as-procrastination. |
| **Absorb MarketMonkey now** | Prove code extensibility first. Absorption under unproven extensibility creates two risks at once. |
| **Build soak test infrastructure** | No consumer at N=2. Build when N>5 or 24-hour operation is a goal. |
| **Unify use-case patterns** | CF-08 is ~180 lines of correct duplication. Migrate when a new family makes it natural, not as a standalone task. |
| **Redesign correlation ID system** | Design is ready. Implement on first consumer (CC-02 new actor), not before. |

---

## 5. Success Criteria for CC-02

CC-02 succeeds if:

1. A new signal family is defined, implemented, validated, and friction-captured following the CC-01 pattern
2. The new signal feeds into the existing decision → strategy → risk → execution chain
3. Both signal families (RSI + new) operate concurrently for 2 symbols without interference
4. At least one deferred debt (CF-03 or CF-08) is resolved as a natural trigger
5. The friction capture produces evidence for or against further extensibility

After CC-02, the path to product features (MarketMonkey absorption, strategy dashboards, alert systems) should be clear and unblocked.

---

## 6. Timeline Expectation

CC-02 follows the same 5-stage pattern as CC-01:

| Stage | Expected Scope |
|-------|---------------|
| S125: CC-02 Definition | Scope, criteria, pressure points (~1 session) |
| S126: CC-02 Implementation | New signal actor + publisher + projection + route (~2-3 sessions) |
| S127: CC-02 Live Validation | Both families × 2 symbols running live (~1 session) |
| S128: CC-02 Friction Capture | Evidence-based friction analysis (~1 session) |
| S129: CC-02 Surgical Refactors | Targeted fixes from S128 evidence (~1 session) |

**Total:** ~6-8 sessions. After S129, the architecture's extensibility is proven and the platform is ready for product waves.
