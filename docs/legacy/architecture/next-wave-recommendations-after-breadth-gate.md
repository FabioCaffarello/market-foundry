# Next Wave Recommendations After Breadth Gate

**Charter:** BREADTH-WAVE-1
**Gate Stage:** S244
**Date:** 2026-03-21

---

## 1. Recommendation

**Execute a short hardening tranche (1–2 stages) before opening the next feature wave.**

This is option (3) from the stage brief. The rationale is evidence-based:

---

## 2. Why Not Option (1): Continue in Feature Evolution

Feature evolution (adding depth to existing types or new features like execution improvements) would be premature because:

- **D1 (smoke test gap)** means new breadth types have not been proven at the E2E deployed-pipeline layer. Adding more features on an unvalidated foundation compounds risk.
- **D3 (remote CI)** — the breadth wave accumulated significant staged and unstaged changes across S241–S244. These must be committed, pushed, and verified green in remote CI before any forward motion.
- The breadth wave was disciplined; the next wave should start on clean ground.

---

## 3. Why Not Option (2): Codegen/Generated Path

The codegen framework is functional and useful as a consistency validator, but:

- Making codegen generative (producing compilable production code) is a substantial infrastructure investment that would consume multiple stages.
- The current pattern — manual implementation validated by golden snapshot comparison — is adequate for the current scale (6 families per domain-layer, 2 types per analytical domain).
- Codegen evolution becomes worthwhile when the number of types per domain exceeds 3–4, creating enough repetition to justify the generator investment.

**Recommendation:** Defer codegen evolution to a dedicated infrastructure charter when type count growth justifies it.

---

## 4. Hardening Tranche: Proposed Scope

### Stage H1: CI Closure and Smoke Test Extension

**Primary deliverable:** Remote CI green with all breadth wave changes committed.

Tasks:
1. Stage and commit all S241–S244 changes (currently mix of staged, unstaged, and untracked)
2. Push to remote and verify CI passes
3. Extend `scripts/smoke-analytical-e2e.sh` to cover:
   - `ema_crossover` decision type
   - `trend_following_entry` strategy type
   - `drawdown_limit` risk type
4. Verify smoke test passes in CI

**Exit criterion:** Remote CI green including extended smoke test.

### Stage H2 (Optional): Integration Test Hardening

**Primary deliverable:** Chain B integration test with drawdown_limit risk.

Tasks:
1. Add integration test: EMA→ema_crossover→trend_following_entry→drawdown_limit (full Chain B with breadth risk)
2. Verify N×1 fan-out: both risk evaluators receive strategy results from both strategy resolvers
3. Confirm no cross-type interference in fan-out routing

**Exit criterion:** Integration tests cover both risk types in chain context.

---

## 5. After Hardening: Next Charter Options

Once the hardening tranche closes with clean CI, three charter directions are viable:

### Option A: Depth Wave

**Objective:** Enrich existing types with more sophisticated logic.

Examples:
- RSI oversold: multi-timeframe confirmation, divergence detection
- EMA crossover: signal strength weighting, cooldown periods
- Position exposure: correlation-adjusted position sizing
- Drawdown limit: trailing drawdown tracking, max loss windows

**When appropriate:** When the analytical pipeline needs more sophisticated evaluation logic before adding more types.

### Option B: Additional Breadth (Wave 2)

**Objective:** Add a third type per domain.

Examples:
- Decision: `volume_spike` (volume-based triggering)
- Strategy: `breakout_entry` (range breakout resolution)
- Risk: `volatility_regime` (VIX/volatility-based risk gating)

**When appropriate:** When the pipeline needs to demonstrate it handles 3+ types per domain cleanly.

### Option C: Execution Domain Evolution

**Objective:** Evolve the execution layer beyond paper_order.

Examples:
- Second execution type (e.g., `limit_order`)
- Execution feedback loop (fill confirmation → risk update)
- Multi-venue routing

**When appropriate:** When the derive→store→read pipeline is mature enough that the execution layer becomes the constraint.

---

## 6. Decision Framework

| Signal | Recommended Direction |
|--------|-----------------------|
| Pipeline will run in production soon | Option A (depth) — sophisticate the logic |
| Pipeline needs to prove scalability | Option B (breadth wave 2) — stress the N-type pattern |
| Execution layer is the bottleneck | Option C — evolve execution |
| None of the above | Continue hardening until a clear signal emerges |

---

## 7. Anti-Patterns to Avoid

1. **Do not open a new feature charter without closing the hardening tranche.** The breadth wave left concrete debts (D1, D3) that must be resolved first.
2. **Do not combine depth and breadth in a single charter.** The current wave succeeded because it had a singular breadth objective.
3. **Do not retroactively extend the breadth charter.** BREADTH-WAVE-1 is closed. Any new work requires a new charter.
4. **Do not skip smoke test extension.** The smoke test gap is the most significant open debt and directly affects deployment confidence.
