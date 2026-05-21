# Post-Squeeze Next-Wave Options Matrix

## Context

The S288–S292 squeeze vertical slice proved that the Foundry's architecture delivers complete domain features with zero new infrastructure. This document evaluates competing directions for the next macro-wave and recommends a primary path.

**Evaluation criteria** (each scored 1–5):

| Criterion | Weight | Rationale |
|-----------|--------|-----------|
| Domain Value | 5× | Does this produce independently useful trading capability? |
| Architectural Pressure | 4× | Does this stress-test and validate infrastructure in new ways? |
| Infrastructure Reuse | 4× | Does this leverage existing proven patterns? |
| Regression Risk | 3× | How likely is this to break existing proofs? (inverse: lower risk = higher score) |
| Delivery Cost | 3× | How many stages to reach meaningful closure? (inverse: lower cost = higher score) |
| Operational Readiness | 2× | Does this bring the system closer to production use? |

---

## Option A: Composite Execution Observability Platform

**Scope**: Prometheus metrics export, latency histograms, cross-binary correlation, time-series dashboards, alerting thresholds.

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 1 | No new trading capability; purely operational |
| Architectural Pressure | 3 | Tests cross-binary correlation; new dependency (Prometheus) |
| Infrastructure Reuse | 2 | Requires new infrastructure (metric export, storage, dashboards) |
| Regression Risk | 4 | Additive; unlikely to break existing functionality |
| Delivery Cost | 2 | Large scope: export layer, dashboards, alerting, cross-binary — 6+ stages |
| Operational Readiness | 5 | Directly improves debugging and operational confidence |

**Weighted Score**: 5+12+8+12+6+10 = **53/105**

**Assessment**: This would be the **fifth infrastructure wave** without delivering domain value. S292's counter-based observability is sufficient for current scale. Prometheus integration is valuable but not yet justified by operational pressure — there is no production deployment demanding it.

---

## Option B: Second Decision Family End-to-End (MACD-Based Vertical Slice)

**Scope**: MACD signal actor wiring → MACD crossover decision family → trend confirmation strategy resolver → risk/execution integration → closed-loop proof.

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 5 | New trading signal family with end-to-end execution capability |
| Architectural Pressure | 4 | Validates pattern reuse with different signal semantics (momentum vs volatility) |
| Infrastructure Reuse | 5 | Identical pattern to squeeze slice; actor wrapper + evaluator + resolver |
| Regression Risk | 4 | Additive families; existing tests unaffected |
| Delivery Cost | 4 | 4–5 stages (actor wiring, decision, strategy, risk integration, closed-loop) |
| Operational Readiness | 3 | Expands paper execution surface; validates multi-strategy coexistence |

**Weighted Score**: 25+16+20+12+12+6 = **91/105**

**Assessment**: Directly validates the S263 hypothesis that infrastructure accelerates domain delivery. Creates a second independent vertical slice that can run in parallel with the squeeze path. Momentum-based semantics (MACD) stress-test the architecture differently than volatility-regime (Bollinger). Highest domain value per stage invested.

---

## Option C: Multi-Symbol Disciplined Expansion

**Scope**: Run existing pipeline across multiple symbols (e.g., btcusdt + ethusdt). Validate JetStream isolation, ClickHouse partitioning, per-symbol configuration, parallel derive supervisors.

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 3 | No new capability; same signals on more symbols |
| Architectural Pressure | 4 | Tests JetStream subject isolation, ClickHouse partition performance |
| Infrastructure Reuse | 3 | Requires new configuration patterns (per-symbol pipeline instances) |
| Regression Risk | 3 | Could expose resource contention, JetStream subject conflicts |
| Delivery Cost | 3 | 4–6 stages (config expansion, JetStream validation, ClickHouse partitioning, smoke) |
| Operational Readiness | 4 | Essential for real trading but premature without more families |

**Weighted Score**: 15+16+12+9+9+8 = **69/105**

**Assessment**: Real-world trading requires multi-symbol, but the current single-symbol surface has only proven two vertical slices (EMA→mean_reversion and Bollinger→squeeze). Adding more symbols amplifies incomplete families without adding capability. Better to expand domain breadth first, then scale horizontally.

---

## Option D: Venue Readiness Charter

**Scope**: Move from paper execution to real venue integration. Define OMS, order routing, portfolio management, fill reconciliation, error recovery.

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 5 | Enables actual trading; transforms from simulation to production |
| Architectural Pressure | 5 | Tests every layer under real constraints (latency, failures, partial fills) |
| Infrastructure Reuse | 1 | Requires entirely new infrastructure (OMS, venue adapter, portfolio) |
| Regression Risk | 2 | High blast radius; changes execution semantics across all families |
| Delivery Cost | 1 | 10+ stages minimum; compliance, risk management, venue APIs |
| Operational Readiness | 5 | Directly production-enabling |

**Weighted Score**: 25+20+4+6+3+10 = **68/105**

**Assessment**: Premature. SafetyGate proven only in isolation (S270). Paper execution barely validated (S266–S268). Scaling factors not calibrated against real data. The Foundry has proven exactly two complete paths through paper mode — opening venue readiness now would normalize an incomplete paper surface. Requires at least 3–4 more proven vertical slices and multi-symbol validation before this charter is responsible.

---

## Option E: Signal Actor Wiring Completion (Breadth-Only)

**Scope**: Create actor wrappers for MACD, VWAP, ATR signal families. Register in derive supervisor. Add writer pipeline entries. No new decision/strategy families.

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 2 | Signals without decisions/strategies have limited standalone value |
| Architectural Pressure | 2 | Repeats proven pattern (identical to Bollinger wiring in S288) |
| Infrastructure Reuse | 5 | Exact same pattern; copy-adapt-register |
| Regression Risk | 5 | Pure additive; cannot break existing functionality |
| Delivery Cost | 5 | 1–2 stages maximum |
| Operational Readiness | 2 | Signals alone don't produce execution intents |

**Weighted Score**: 10+8+20+15+15+4 = **72/105**

**Assessment**: Low cost but low value. Signals without decisions are inert data in the pipeline. This is a necessary prerequisite for Option B but insufficient as a standalone wave. Better folded into the first stage of a vertical slice wave.

---

## Option F: Codegen Framework Expansion

**Scope**: Extend codegen to generate actor wrappers, decision evaluators, strategy resolvers (not just consumer_spec and pipeline_entry).

| Criterion | Score | Justification |
|-----------|-------|---------------|
| Domain Value | 1 | No domain capability; purely developer tooling |
| Architectural Pressure | 3 | Tests codegen's ability to handle more complex artifacts |
| Infrastructure Reuse | 3 | Builds on existing codegen; but requires new templates and validation |
| Regression Risk | 3 | Could invalidate existing generated artifacts if templates change |
| Delivery Cost | 3 | 3–5 stages (template design, generation, validation, migration) |
| Operational Readiness | 1 | Zero operational impact |

**Weighted Score**: 5+12+12+9+9+2 = **49/105**

**Assessment**: S263 explicitly stated codegen expansion should be a side-effect, not a primary objective. The squeeze slice proved that manual actor wrappers are small (50–100 lines) and follow a copy-adapt pattern. The ROI of codegen expansion is low until the Foundry has 8+ families per layer, which is several waves away.

---

## Comparative Matrix

| Option | Domain | Arch Pressure | Reuse | Risk (inv) | Cost (inv) | Ops Ready | **Total** |
|--------|--------|---------------|-------|------------|------------|-----------|-----------|
| **A: Observability** | 5 | 12 | 8 | 12 | 6 | 10 | **53** |
| **B: MACD Vertical Slice** | 25 | 16 | 20 | 12 | 12 | 6 | **91** |
| **C: Multi-Symbol** | 15 | 16 | 12 | 9 | 9 | 8 | **69** |
| **D: Venue Readiness** | 25 | 20 | 4 | 6 | 3 | 10 | **68** |
| **E: Signal Wiring Only** | 10 | 8 | 20 | 15 | 15 | 4 | **72** |
| **F: Codegen Expansion** | 5 | 12 | 12 | 9 | 9 | 2 | **49** |

---

## Recommendation

### Primary Direction: Option B — MACD-Based Vertical Slice

**Score: 91/105 (highest by 19 points)**

**Why this wins:**

1. **Maximum domain value per stage**: Each stage delivers a new independently useful component, culminating in a second complete signal-to-execution path.

2. **Validates pattern reuse with different semantics**: MACD (momentum/trend) is semantically distinct from Bollinger (volatility regime). If the same architecture handles both without modification, the infrastructure investment is validated.

3. **Highest infrastructure reuse**: Every actor wrapper, evaluator, resolver, and integration test follows the proven squeeze pattern. Delivery should be measurably faster than the squeeze slice — if it isn't, that's a signal worth investigating.

4. **Low regression risk**: All new families are additive. Existing squeeze and EMA paths are unaffected.

5. **Completes the S283 charter intent**: S283 delivered MACD, VWAP, ATR at application level but the charter's true goal was "validate that infrastructure translates into low-cost domain expansion." Only a complete vertical slice proves this.

**Proposed stage sequence** (4–5 stages):

| Stage | Scope | Exit Criteria |
|-------|-------|---------------|
| S294 | MACD signal actor wiring + bollinger_squeeze pipeline gap closure | MACD signal flows from candle to NATS; bollinger_squeeze decisions persist to ClickHouse |
| S295 | MACD crossover decision family (evaluator + actor + tests) | Decision evaluator consuming MACD signals; behavioral tests passing |
| S296 | MACD trend confirmation strategy resolver | Strategy resolver with severity-aware scaling; closed-loop unit tests |
| S297 | Risk integration + full closed-loop MACD scenario | 4+ E2E scenarios proving triggered/suppressed/severity contrast/context |
| S298 | Post-MACD-slice gate + velocity comparison | Honest comparison of squeeze vs MACD delivery velocity and cost |

### Secondary Direction: Signal Actor Wiring for VWAP and ATR

**Folded into the primary wave** — during S294, wire VWAP and ATR signal actors alongside MACD. This is a low-cost, high-leverage addition (each actor wrapper is ~80 lines following an identical pattern) that completes the S283 charter's signal layer intent.

These signal families remain "inert" (no decision/strategy consumers) until future vertical slices activate them. This is acceptable — signals flowing into ClickHouse have analytical value even without execution paths.

---

## What Explicitly NOT to Open Now

| Direction | Reason | When to Revisit |
|-----------|--------|-----------------|
| **Composite Observability Platform** | Fifth infrastructure wave; S292 counters sufficient; no production deployment demanding dashboards | After 3+ vertical slices are running in parallel |
| **Multi-Symbol Expansion** | Amplifies incomplete families without adding capability; JetStream isolation is a concern but not yet a blocker | After MACD and at least one more vertical slice are complete |
| **Venue Readiness** | Paper execution barely validated; scaling factors uncalibrated; SafetyGate proven only in isolation | After multi-symbol + multi-family validation + explicit compliance charter |
| **Codegen Framework Expansion** | Manual actor wrappers are 50–100 lines; ROI of codegen expansion low until 8+ families per layer | When copy-adapt pattern becomes measurably burdensome (not yet) |
| **Short-Side Strategy Resolution** | Long-only is sufficient for current validation phase; short-side requires separate risk semantics | After long-side is proven across 3+ strategies |
| **Parallel Feature Fronts** | Violates single-front discipline established in S258 and reaffirmed in every subsequent gate | Never without explicit charter amendment |

---

## Risk Assessment for Recommended Path

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| MACD semantics require architecture changes | Low | Medium | MACD is a standard momentum indicator; signal→decision flow is generic |
| Decision family for MACD requires new patterns | Low | Low | RSI oversold and Bollinger squeeze decisions already prove two patterns |
| Strategy resolver needs new risk integration | Very Low | Low | risk_scaling.go is strategy-type-agnostic |
| Multi-strategy coexistence causes conflicts | Low | Medium | Strategies resolve independently; paper execution is strategy-agnostic |
| Delivery velocity doesn't improve over squeeze | Medium | Low | Would be valuable signal about infrastructure costs; not a failure |

---

## Success Criteria for the Recommended Wave

1. **MACD vertical slice closed in ≤5 stages** (squeeze took 5 stages; equal or faster validates infrastructure ROI).
2. **Zero regressions** in existing squeeze and EMA paths.
3. **Pattern identical** — if any architecture change is required, it must be documented as a finding.
4. **Velocity measurement** — explicit comparison of squeeze vs MACD delivery cost in the closing gate.
5. **CI green** throughout — every intermediate stage passes all existing tests.
