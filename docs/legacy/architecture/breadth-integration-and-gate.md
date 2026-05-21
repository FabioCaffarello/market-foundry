# Breadth Integration and Gate

**Charter:** BREADTH-WAVE-1
**Stage:** S244
**Type:** Integration proof + formal gate
**Date:** 2026-03-21
**Status:** GATE EVALUATED

---

## 1. Purpose

This document evaluates the breadth charter BREADTH-WAVE-1 against its frozen exit criteria (E1–E9). The evaluation is binary: every criterion must pass for the charter to close as succeeded.

---

## 2. Exit Criteria Matrix

| # | Criterion | Pass Condition | Evidence | Verdict |
|---|-----------|----------------|----------|---------|
| E1 | Decision breadth ≥ 2 types | Distinct family names + distinct signal sources | `rsi_oversold` (RSI numeric threshold) + `ema_crossover` (EMA categorical direction) | **PASS** |
| E2 | Strategy breadth ≥ 2 types | Distinct family names + distinct resolution logic | `mean_reversion_entry` (counter-trend, RSI) + `trend_following_entry` (pro-trend, EMA) | **PASS** |
| E3 | Risk breadth ≥ 2 types | Distinct family names + distinct risk dimensions | `position_exposure` (position size/portfolio exposure) + `drawdown_limit` (drawdown/stop-loss) | **PASS** |
| E4 | Domain validation tests | 100% of new types have domain unit tests | `decision_test.go`, `strategy_test.go`, `risk_test.go` — all exercise new type constants and validation | **PASS** |
| E5 | Application logic tests | 100% of new types have evaluator/resolver unit tests | `ema_crossover_evaluator_test.go` (20 tests), `trend_following_entry_resolver_test.go` (12 tests), `drawdown_limit_evaluator_test.go` (21 tests) | **PASS** |
| E6 | Actor integration | Fan-out, publisher, actor tests present | Actor files + test files for all 3 new types; derive_supervisor registers all; store_supervisor registers all | **PASS** |
| E7 | Chain integration | ≥ 2 distinct chain paths in integration tests | Chain A: RSI→rsi_oversold→mean_reversion_entry→position_exposure; Chain B: EMA→ema_crossover→trend_following_entry→position_exposure; 6 integration test functions | **PASS** |
| E8 | Codegen families | 3 new family YAMLs | `ema_crossover.yaml`, `trend_following_entry.yaml`, `drawdown_limit.yaml` + golden snapshots | **PASS** |
| E9 | CI green | All tests pass | `make test` passes all modules including codegen golden comparisons | **PASS** |

**Overall verdict: 9/9 PASS — Charter criteria met.**

---

## 3. Breadth Validation: Derive → Store → Read → HTTP

### 3.1 Decision Domain

| Component | rsi_oversold | ema_crossover |
|-----------|-------------|---------------|
| Domain type constant | `"rsi_oversold"` | `"ema_crossover"` |
| Evaluator | `RSIOversoldEvaluator` | `EMACrossoverEvaluator` |
| Actor | `RSIOversoldEvaluatorActor` | `EMACrossoverEvaluatorActor` |
| Derive supervisor registration | Line 165 | Line 174 |
| NATS event subject | `decision.events.rsi_oversold.evaluated.>` | `decision.events.ema_crossover.evaluated.>` |
| NATS stream | `DECISION_EVENTS` (shared) | `DECISION_EVENTS` (shared) |
| Writer pipeline entry | `writer-decision-rsi-oversold` | `writer-decision-ema-crossover` |
| Store projection | `DECISION_RSI_OVERSOLD_LATEST` KV | `DECISION_EMA_CROSSOVER_LATEST` KV |
| ClickHouse table | `decisions` (shared) | `decisions` (shared) |
| HTTP route | `GET /decision/rsi_oversold/latest` | `GET /decision/ema_crossover/latest` |
| Codegen family YAML | `rsi_oversold.yaml` | `ema_crossover.yaml` |

### 3.2 Strategy Domain

| Component | mean_reversion_entry | trend_following_entry |
|-----------|---------------------|----------------------|
| Domain type constant | `"mean_reversion_entry"` | `"trend_following_entry"` |
| Resolver | `MeanReversionEntryResolver` | `TrendFollowingEntryResolver` |
| Actor | `MeanReversionEntryResolverActor` | `TrendFollowingEntryResolverActor` |
| Derive supervisor registration | Line 188 | Line 196 |
| NATS event subject | `strategy.events.mean_reversion_entry.resolved.>` | `strategy.events.trend_following_entry.resolved.>` |
| NATS stream | `STRATEGY_EVENTS` (shared) | `STRATEGY_EVENTS` (shared) |
| Writer pipeline entry | `writer-strategy-mean-reversion-entry` | `writer-strategy-trend-following-entry` |
| Store projection | `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` KV | `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST` KV |
| ClickHouse table | `strategies` (shared) | `strategies` (shared) |
| HTTP route | `GET /strategy/mean_reversion_entry/latest` | `GET /strategy/trend_following_entry/latest` |
| Codegen family YAML | `mean_reversion_entry.yaml` (existing) | `trend_following_entry.yaml` (new) |

### 3.3 Risk Domain

| Component | position_exposure | drawdown_limit |
|-----------|------------------|----------------|
| Domain type constant | `"position_exposure"` | `"drawdown_limit"` |
| Evaluator | `PositionExposureEvaluator` | `DrawdownLimitEvaluator` |
| Actor | `PositionExposureEvaluatorActor` | `DrawdownLimitEvaluatorActor` |
| Derive supervisor registration | Line 209 | Line 218 |
| NATS event subject | `risk.events.position_exposure.assessed.>` | `risk.events.drawdown_limit.assessed.>` |
| NATS stream | `RISK_EVENTS` (shared) | `RISK_EVENTS` (shared) |
| Writer pipeline entry | `writer-risk-position-exposure` | `writer-risk-drawdown-limit` |
| Store projection | `RISK_POSITION_EXPOSURE_LATEST` KV | `RISK_DRAWDOWN_LIMIT_LATEST` KV |
| ClickHouse table | `risk_assessments` (shared) | `risk_assessments` (shared) |
| HTTP route | `GET /risk/position_exposure/latest` | `GET /risk/drawdown_limit/latest` |
| Codegen family YAML | `position_exposure.yaml` (existing) | `drawdown_limit.yaml` (new) |

---

## 4. Chain Integration Evidence

### 4.1 Chain A (Original — RSI path)

```
RSI signal → RSIOversoldEvaluatorActor → MeanReversionEntryResolverActor → PositionExposureEvaluatorActor
```

**Test:** `TestActorChain_Signal_To_Decision_To_Strategy_To_Risk`
- RSI=28.50 → triggered (severity=low) → long (confidence from decision) → approved (constraints set)
- Correlation ID preserved end-to-end

### 4.2 Chain B (New — EMA path)

```
EMA signal → EMACrossoverEvaluatorActor → TrendFollowingEntryResolverActor → PositionExposureEvaluatorActor
```

**Test:** `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk`
- Bullish EMA → triggered (severity=moderate) → long (trailing_stop_pct=0.03) → approved
- Decision severity and rationale propagated through entire chain

### 4.3 Additional Chain Variants Tested

- Not-triggered path (RSI=75): `TestActorChain_NotTriggered_FlowsThrough`
- EMA bearish path: `TestActorChain_EMACrossover_Bearish_NotTriggered`
- Correlation ID preservation: `TestActorChain_CorrelationID_PreservedEndToEnd`

---

## 5. Amendment Log Review

**Amendments filed during charter: 0**

The charter's Amendments Log in `breadth-charter-and-scope-freeze.md` records: _"No amendments recorded. Charter is in its original frozen state."_

No stop conditions were triggered. No scope changes were needed. The charter executed as planned.

---

## 6. Governance Compliance

| Rule | Description | Compliance |
|------|-------------|------------|
| R1 | Pre-execution documentation | Charter frozen before S241 began | **COMPLIANT** |
| R2 | Exit criteria explicitly tracked | All E1–E9 evaluated above | **COMPLIANT** |
| R3 | Mid-charter gate | S242 delivered strategy breadth; decision and strategy domains confirmed ≥2 types | **COMPLIANT** |
| R4 | No retroactive modification | Original charter text unchanged | **COMPLIANT** |
| R5 | Post-hoc amendments flagged | No deviations discovered | **COMPLIANT** |

---

## 7. Gate Decision

**BREADTH-WAVE-1 charter: PASSED**

All nine exit criteria are met. Zero amendments were required. The charter delivered exactly what it committed to: ≥2 evaluator/resolver types per domain in Decision, Strategy, and Risk, with full pipeline integration.

---

## 8. Open Items Deferred to Next Charter

These items were observed during the breadth wave but are explicitly out of scope:

1. **Smoke test coverage gap:** The E2E smoke test (`scripts/smoke-analytical-e2e.sh`) validates only Wave A types (rsi_oversold, mean_reversion_entry, position_exposure). New breadth types are proven in unit + integration tests but not in the full smoke pipeline.

2. **drawdown_limit → Chain B risk:** Integration tests route Chain B through `position_exposure` risk, not `drawdown_limit`. The drawdown_limit evaluator is proven in isolation but not yet in a full chain integration test.

3. **No decision-to-strategy mapping configuration:** The mapping of which decision types feed which strategy resolvers is implicit in the derive supervisor registration, not driven by configuration.

4. **Codegen is descriptive, not generative:** Family YAMLs and golden snapshots validate naming conventions but do not generate production code. The codegen engine produces reference artifacts only.
