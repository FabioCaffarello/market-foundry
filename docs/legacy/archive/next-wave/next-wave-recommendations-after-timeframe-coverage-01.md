# Next Wave Recommendations After TC-01

> Evidence-based recommendation for Market Foundry's next evolution.
> Date: 2026-03-19 | Stage: S136

---

## 1. The Four Options Evaluated

| # | Option | Description |
|---|--------|-------------|
| 1 | **TC-02**: Expand temporal matrix further | Add 4h, daily, weekly timeframes |
| 2 | **New family/capability** | Add a small new signal family, evidence type, or strategy variant |
| 3 | **Targeted hardening** | Fix specific friction points triggered by TC-01 |
| 4 | **Product wave** | Move toward concrete product-facing output |

---

## 2. Evaluation Against Evidence

### Option 1: TC-02 (More Timeframes) — NOT RECOMMENDED NOW

**Arguments for**:
- Architecture proven to scale linearly
- 4h and daily are natural next steps for intraday/swing coverage

**Arguments against**:
- TC-01 already proved the thesis. Adding 4h/daily proves the same thing again with higher risk.
- **D1 (state persistence) is a hard gate**. 4h+ timeframes with in-memory-only state means hours of data loss on crash. This requires non-trivial engineering (WAL or snapshots) before TC-02 can execute safely.
- Daily/weekly require market-session semantics and timezone design — qualitatively different work, not just config expansion.
- M7/M8 (RSI convergence at 900s/3600s) remain unverified. Expanding before validating is premature.
- Diminishing returns: 4 TFs already cover the most commonly used intraday windows.

**Verdict**: TC-02 should wait until (a) D1 is resolved, (b) M7/M8 are verified, and (c) there is product-driven demand for longer windows. Expanding for the sake of proving more scaling is refactoring by impulse.

### Option 2: New Small Family/Capability — CONDITIONALLY RECOMMENDED

**Arguments for**:
- Tests the pipeline's horizontal dimension (new family) rather than vertical (more timeframes)
- A new signal family (e.g., MACD, Bollinger) would exercise the same actor lifecycle but with different evaluation logic
- A new evidence type would test whether the sampler pattern generalizes
- Both prove something TC-01 didn't: that the architecture handles diversity of computation, not just diversity of time windows

**Arguments against**:
- Must be scoped tightly to avoid reopening horizontal abstractions
- Should not require new infrastructure (no new NATS streams, no new KV buckets beyond the pattern)

**Verdict**: Recommended **only if scoped to a single, small addition** that uses existing infrastructure patterns. Good candidates:
1. A second signal family (e.g., MACD) — tests signal evaluator generalization
2. A second decision family — tests decision resolver generalization

**Avoid**: New evidence families that require different sampler architectures (e.g., order book depth requires streaming, not windowed accumulation).

### Option 3: Targeted Hardening — NOT RECOMMENDED AS PRIMARY WAVE

**Arguments for**:
- D1 (state persistence) is a real debt
- D2 (per-TF idle detection) would improve operations

**Arguments against**:
- D1 is only triggered by TC-02 (4h+ TFs). Solving it now is premature unless TC-02 is imminent.
- D2 is low-effort but low-value at 4 TFs.
- "Hardening wave" risks becoming open-ended refactoring without product progress.
- The two P1 items (F-02, F-17) were already addressed in S135. Remaining items are P2/P3.

**Verdict**: Not a wave on its own. Individual hardening items should be folded into whatever wave comes next, as needed. D1 should be addressed as a prerequisite when TC-02 is actually planned, not speculatively.

### Option 4: Product Wave — RECOMMENDED

**Arguments for**:
- TC-01 proved the infrastructure works. The pipeline processes trades → candles → signals → decisions → strategies → risk → execution across 4 timeframes.
- The next meaningful proof is: **does this produce useful output for a real use case?**
- Moving toward product forces contact with real requirements that pure infrastructure waves cannot surface.
- The execution layer (`paper_order`) exists but has not been exercised end-to-end with real market conditions.
- Product work naturally surfaces the _right_ frictions — the ones that matter to users, not the ones that matter to architects.

**Arguments against**:
- Requires defining what "product" means for Market Foundry (paper trading? backtesting? alerting?)
- Could expose weaknesses that need hardening before being useful
- Must not become a scope explosion

**Verdict**: Recommended as the primary focus, scoped to exercising the existing pipeline end-to-end with a concrete, small product scenario.

---

## 3. Recommended Path

### Primary: Product-Oriented Wave

**Goal**: Exercise the existing pipeline from trade ingestion to execution intent with a concrete, measurable product scenario.

**Candidate scenarios** (pick one):
1. **Paper trading loop**: Execute paper orders based on mean_reversion_entry strategy, track P&L
2. **Alert/notification output**: Surface strategy signals as actionable notifications
3. **Backtest harness**: Replay historical data through the pipeline, measure strategy performance

Each of these forces the pipeline to prove value beyond "the data flows correctly."

### Secondary (if capacity exists): One New Signal Family

**Goal**: Prove the pipeline handles computational diversity, not just temporal diversity.

**Scope**: Add exactly one signal family (e.g., MACD or Bollinger Bands) using the same evaluator pattern as RSI. If it requires changing the evaluator interface or actor lifecycle, that's signal — capture the friction and stop.

### Explicitly NOT Recommended

- TC-02 (more timeframes) — wait for product demand
- State persistence (D1) — wait for TC-02 to be planned
- Horizontal abstractions (generic evaluator framework, plugin system) — no evidence of need
- Multi-symbol expansion beyond 2-3 — infrastructure scales, but product value unclear

---

## 4. Decision Framework for the Next Wave

Before starting the next wave, answer these questions:

1. **What does the wave prove that TC-01 didn't?**
   - TC-01 proved temporal scaling. The next wave should prove something different.

2. **What is the smallest scope that proves it?**
   - If the answer requires more than 2 weeks of work, scope is too large.

3. **What friction will the wave surface?**
   - Product work surfaces user-facing friction. Infrastructure work surfaces architect-facing friction. The Foundry needs more user-facing signal now.

4. **What is the exit condition?**
   - Define success criteria before starting, as TC-01 did with M1–M13.

---

## 5. Summary

| Option | Recommendation | Rationale |
|--------|---------------|-----------|
| TC-02 (more TFs) | Wait | Thesis proven; D1 hard gate; no product demand |
| New family | Conditional | Good secondary goal if scoped tightly |
| Hardening | Fold in | Not a wave; address items as prerequisites |
| **Product wave** | **Primary** | **Infrastructure proven; time to prove value** |

**The Foundry has proven it can scale. Now it needs to prove it can deliver.**
