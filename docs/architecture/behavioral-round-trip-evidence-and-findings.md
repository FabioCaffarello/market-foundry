# Behavioral Round-Trip Evidence and Findings

**Stage:** S255
**Date:** 2026-03-21

## Evidence Summary

This document records the concrete evidence produced by the S255 behavioral
full-stack smoke closure. It catalogs what was tested, what was found, and
what implications arise for future work.

## Test Inventory

### Round-Trip Serialization Tests (17 tests, 0 failures)

```
TestBehavioralRoundTrip_DecisionSeverity_High              PASS
TestBehavioralRoundTrip_DecisionSeverity_Low                PASS
TestBehavioralRoundTrip_DecisionSeverity_AllEnumValues
  /none                                                     PASS
  /low                                                      PASS
  /moderate                                                 PASS
  /high                                                     PASS
TestBehavioralRoundTrip_Strategy_SeverityScaledConfidence   PASS
TestBehavioralRoundTrip_Strategy_LowSeverity_ReducedConfidence PASS
TestBehavioralRoundTrip_Risk_PositionExposure_CounterTrend  PASS
TestBehavioralRoundTrip_Risk_DrawdownLimit_ProTrend         PASS
TestBehavioralRoundTrip_SeverityContrast_HighVsLow          PASS
TestBehavioralRoundTrip_CrossChain_RiskProfileDivergence    PASS
TestBehavioralRoundTrip_NotTriggered_CleanFlow              PASS
TestBehavioralRoundTrip_ConfidencePrecision
  /0.8333                                                   PASS
  /0.6666                                                   PASS
  /0.7650                                                   PASS
  /0.8280                                                   PASS
  /0.4666                                                   PASS
  /0.9500                                                   PASS
  /1.0000                                                   PASS
  /0.0000                                                   PASS
TestBehavioralRoundTrip_FullChain_HighSeverity_MeanReversion PASS
```

### Smoke Analytical Behavioral Phase (6 checks)

| Check | Gate | Expected Outcome |
|-------|------|-----------------|
| Decision severity enum fidelity | PASS/WARN | All values in {none,low,moderate,high} |
| Strategy confidence ≤ decision confidence | PASS/WARN | Severity scaling preserves ordering |
| Risk behavioral metadata present | PASS/WARN | strategy_type + confidence_factor in metadata |
| Risk constraints non-empty (approved) | PASS/WARN | max_position_size or stop_distance set |
| Dual-risk fan-out | PASS/WARN | Both evaluators have rows in ClickHouse |
| Chain B behavioral metadata | PASS/WARN | trend_following → drawdown_limit stop_distance |

## Key Findings

### Finding 1: Float64 Precision Is Lossless for Behavioral Values

All confidence values used by the behavioral wave (0.0000 through 1.0000, with
up to 4 decimal places) survive the `string → parseFloat → float64 → FormatFloat → string`
round-trip with zero precision loss (delta < 1e-10).

**Implication:** No precision guards are needed for the current value range.
If future work introduces sub-1e-10 precision requirements, this should be revisited.

### Finding 2: JSON Nested Structure Preservation Is Complete

The behavioral chain uses nested JSON structures:
- `strategy.decisions[]` carries `severity` and `rationale` from decisions
- `risk.strategies[]` carries `decision_severity` and `decision_rationale` from strategy

Both survive `marshalJSON → ClickHouse String column → ParseDecisionInputsJSON / ParseStrategyInputsJSON`
with full field fidelity. The `omitempty` tag on `DecisionSeverity` and `DecisionRationale`
in `risk.StrategyInput` correctly preserves values when present and omits them when empty.

**Implication:** Backward compatibility is maintained — pre-behavioral-wave data
(without severity/rationale) deserializes cleanly with zero-value fields.

### Finding 3: Severity Enum Is String-Typed End-to-End

The severity field uses `LowCardinality(String)` in ClickHouse, which means:
- No integer encoding/decoding risk
- The enum values are stored as literal strings: "none", "low", "moderate", "high"
- The reader casts back to `decision.Severity(string)` directly

**Implication:** Adding new severity levels in the future requires no schema migration,
only application-layer changes.

### Finding 4: Behavioral Metadata Is Fully Recoverable

Risk assessments carry behavioral metadata in `map[string]string`:
- `strategy_type`: "mean_reversion_entry" or "trend_following_entry"
- `confidence_factor`: "0.90" or "0.95"
- `severity_limit_factor`: "1.15", "1.00", or "0.80"
- `decision_severity`: "high", "moderate", "low", or "none"

All survive the round-trip. This metadata enables post-hoc behavioral analysis
via ClickHouse queries without requiring application-layer decoding.

### Finding 5: Confidence Ordering Is Invariant

For triggered chains, the behavioral wave guarantees:
```
risk_confidence ≤ strategy_confidence ≤ decision_confidence
```

This ordering is preserved through serialization because:
1. Confidence values are computed as multiplicative products (factor ≤ 1.0)
2. Float64 multiplication preserves ordering
3. FormatFloat uses natural precision, avoiding rounding artifacts

The round-trip test explicitly asserts this invariant.

### Finding 6: Correlation/Causation Chain Survives Intact

The full chain test verifies:
- `correlation_id` is identical across decision, strategy, and risk rows
- `causation_id` forms a chain: signal → decision → strategy

This enables full behavioral trace reconstruction from ClickHouse analytical data.

## Behavioral Coverage Matrix

| Behavioral Property | In-Process (S252) | Round-Trip (S255) | Full-Stack (S255) |
|---------------------|:-----------------:|:-----------------:|:-----------------:|
| Severity enum fidelity | YES | YES | YES |
| Severity → confidence scaling | YES | YES | YES |
| Severity → parameter adjustment | YES | YES | — |
| Strategy-type → risk confidence | YES | YES | YES |
| Strategy-type → stop distance | YES | YES | YES |
| Dual-risk fan-out | YES | — | YES |
| Context preservation (rationale) | YES | YES | YES |
| Correlation/causation chain | YES | YES | — |
| Confidence ordering invariant | YES | YES | YES |
| Not-triggered clean flow | YES | YES | — |
| Constraints non-zero (approved) | YES | YES | YES |
| Behavioral metadata in risk | — | YES | YES |

## What Remains Outside Scope

| Item | Reason | Risk |
|------|--------|------|
| CBOR envelope encoding isolation | Covered by integration tests + smoke pipeline | Low |
| Actual ClickHouse protocol encoding | Covered by smoke-analytical against live CH | Low |
| HTTP JSON response field ordering | Go's json.Marshal is deterministic for structs | None |
| Performance under behavioral load | Defer to OD-BW5 (performance budgets) | Low |
| Configctl-driven severity thresholds | Defer to OD-BW6 (configuration wave) | Low |
