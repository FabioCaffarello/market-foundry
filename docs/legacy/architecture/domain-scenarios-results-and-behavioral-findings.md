# Domain Scenarios: Results and Behavioral Findings

## Overview

This document records the concrete results and behavioral findings from running the S252 end-to-end scenario validation suite. All data was captured from actual test execution on 2026-03-21.

## Test Execution Summary

| Scenario | Test Function | Result | Duration |
|----------|--------------|--------|----------|
| 1. RSI → Mean Reversion → Dual Risk | `TestScenario_RSIOversold_MeanReversion_DualRisk` | PASS | 50ms |
| 2. EMA → Trend Following → Dual Risk | `TestScenario_EMACrossover_TrendFollowing_DualRisk` | PASS | 50ms |
| 3. Severity Contrast (High vs Low) | `TestScenario_SeverityContrast_HighVsLow` | PASS | 100ms |
| 4. Cross-Chain Risk Profile | `TestScenario_CrossChain_RiskProfileComparison` | PASS | 100ms |
| 5. Not-Triggered (Both Chains) | `TestScenario_NotTriggered_BothChains_FlatApproved` | PASS | 100ms |
| 6. Context Preservation | `TestScenario_ContextPreservation_RationaleEndToEnd` | PASS | 50ms |

Total: **6 scenarios, 6 passed, 0 failed.**

## Behavioral Findings

### Finding 1: Severity Produces Measurable Behavioral Divergence

The severity contrast test (Scenario 3) demonstrates that decision severity is not decorative — it produces quantitatively different risk outcomes:

```
High severity (RSI 10.0):
  decision → confidence=0.8333, severity=high
  strategy → confidence=0.8333 (×1.00), target=0.03, stop=0.01
  risk     → confidence=0.7500, position=0.0192, limit_factor=1.15

Low severity (RSI 25.0):
  decision → confidence=0.5833, severity=low
  strategy → confidence=0.4666 (×0.80), target=0.02, stop=0.02
  risk     → confidence=0.4199, position=0.0075, limit_factor=0.80
```

**Key ratios:**
- Position size: high is **2.56×** larger than low (0.0192 / 0.0075)
- Risk confidence: high is **1.79×** higher (0.7500 / 0.4199)
- Severity limit factor: high allows **1.44×** more room (1.15 / 0.80)

This confirms the S250/S251 behavioral activation is working as designed: strong signals produce more aggressive positions; weak signals are treated conservatively.

### Finding 2: Strategy Type Drives Asymmetric Risk Treatment

The cross-chain comparison (Scenario 4) proves that the same risk evaluator applies different factors based on strategy type:

| Factor | Mean Reversion (counter-trend) | Trend Following (pro-trend) |
|--------|-------------------------------|----------------------------|
| Position exposure confidence | ×0.90 | ×0.95 |
| Drawdown confidence | ×0.85 | ×0.92 |
| Drawdown stop factor | ×0.85 | ×1.15 |

Counter-trend strategies receive:
- **5.3% lower** position exposure confidence (0.90 vs 0.95)
- **7.6% lower** drawdown confidence (0.85 vs 0.92)
- **26.1% tighter** stop distance ceiling (0.85 vs 1.15)

This asymmetry reflects the domain reality: entering against the prevailing trend carries inherently higher risk than following it.

### Finding 3: Dual-Risk Assessment is Coherent

Scenarios 1 and 2 demonstrate that a single strategy resolution fans out to both risk evaluators and produces independent, valid assessments:

**Chain A (mean_reversion, high severity):**
- Position exposure: approved, confidence=0.7500, MaxPositionSize=0.0192
- Drawdown limit: approved, confidence=0.7083, StopDistance=0.0213

**Chain B (trend_following, moderate severity):**
- Position exposure: approved, confidence=0.6412, MaxPositionSize=0.0135
- Drawdown limit: approved, confidence=0.6210, StopDistance=0.0233

Both assessments:
- Pass domain validation (`Validate()` returns nil)
- Carry distinct strategy-type-aware factors
- Preserve decision severity end-to-end
- Record strategy type in metadata

### Finding 4: Decision Context Survives the Full Pipeline

Scenario 6 traces a specific rationale string through 6 checkpoints:

```
Original: "RSI 15.0000 below oversold threshold 30.0 (distance 50.0%); severity moderate"

Preserved at:
  ✓ decisionEvaluatedMessage.DecisionRationale
  ✓ Strategy.Decisions[0].Rationale
  ✓ Strategy.Metadata["decision_rationale"]
  ✓ strategyResolvedMessage.DecisionRationale
  ✓ RiskAssessment.Strategies[0].DecisionRationale
  ✓ RiskAssessment.Metadata["decision_rationale"]
```

This means any observer at any pipeline stage can trace back to the original decision's reasoning — critical for debugging and compliance.

### Finding 5: Non-Triggered Paths Are Clean

Both chains handle non-triggered decisions without errors:
- Decision: `not_triggered`, severity=`none`
- Strategy: direction=`flat`, confidence=`0.0000`
- Risk: disposition=`approved`, confidence=`1.0000`

Flat strategies bypass all sizing and constraint logic, producing safe defaults. No edge cases were encountered.

### Finding 6: Correlation IDs Survive Dual-Risk Fan-Out

Correlation IDs are preserved not just through the linear chain but also through the fan-out to multiple risk evaluators. Both position_exposure and drawdown_limit assessments carry the original correlation ID from the signal injection point.

## Before/After Behavioral Richness

### Before S252

| Capability | Status |
|-----------|--------|
| Individual chain wiring | Validated (actor_chain_integration_test.go) |
| Severity-aware strategy resolution | Validated per-type (unit tests) |
| Strategy-type-aware risk assessment | Validated per-type (unit tests) |
| Dual-risk fan-out | **Not validated** |
| Severity contrast (same chain, different inputs) | **Not validated** |
| Cross-chain risk profile comparison | **Not validated** |
| Context preservation end-to-end | **Not validated** |
| Quantitative behavioral divergence | **Not validated** |

### After S252

| Capability | Status |
|-----------|--------|
| All above | Validated |
| Dual-risk fan-out | **Validated** — Scenarios 1, 2 |
| Severity contrast | **Validated** — Scenario 3 (2.56× position ratio) |
| Cross-chain comparison | **Validated** — Scenario 4 (asymmetric factors) |
| Context preservation | **Validated** — Scenario 6 (6 checkpoints) |
| Quantitative divergence | **Validated** — Scenarios 3, 4 (precise ratios) |

## Observed Simplifications

1. **2-decimal formatting edge case:** `stop_offset = 0.01 × 0.75 = 0.0075` formats to `"0.01"` via `FormatParam("%.2f")`. This is a known limitation of 2-decimal display — the underlying float64 arithmetic is correct. A future stage could increase parameter precision if needed.

2. **EMA crossover fixed severity:** The `EMACrossoverEvaluator` always returns `SeverityModerate` for bullish crossovers. There is no graduated severity based on crossover distance/speed. This is adequate for the current evaluator design but limits severity contrast testing on Chain B.

3. **No execution stage:** Scenarios end at risk assessment. The paper_order execution evaluator is not included — it would require additional wiring and is outside the scope of behavioral chain validation.

## Recommendations

1. **S253 scope:** Use these scenarios as CI regression anchors. Any behavioral change to evaluators or resolvers will be caught by the quantitative assertions in Scenarios 3 and 4.

2. **Future enrichment:** If additional decision types are added (e.g., MACD, Bollinger), create corresponding scenario tests following the same dual-risk pattern.

3. **Parameter precision:** Consider increasing `FormatParam` to 4 decimal places for parameters where sub-1% differences are significant. This is cosmetic, not functional.
