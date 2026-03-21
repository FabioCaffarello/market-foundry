# Decision Context Consumption by Strategy

**Stage:** S250
**Charter:** BEHAVIORAL-WAVE-1
**Date:** 2026-03-21
**Status:** Active

---

## 1. Purpose

This document defines the formal contract for how strategy resolvers consume decision context. It serves as the reference for understanding which decision fields influence strategy behavior and how.

---

## 2. Decision Fields Available to Strategy

When a strategy resolver receives a `decisionEvaluatedMessage`, the following fields are available:

| Field | Type | Source | Example |
|-------|------|--------|---------|
| `DecisionType` | string | `decision.Type` | `"rsi_oversold"`, `"ema_crossover"` |
| `DecisionOutcome` | string | `decision.Outcome` | `"triggered"`, `"not_triggered"`, `"insufficient"` |
| `DecisionConfidence` | string | `decision.Confidence` | `"0.8500"` |
| `DecisionSeverity` | string | `decision.Severity` | `"high"`, `"moderate"`, `"low"`, `"none"` |
| `DecisionRationale` | string | `decision.Rationale` | `"RSI 20.00 below oversold threshold..."` |
| `Timeframe` | int | `decision.Timeframe` | `60` |
| `Timestamp` | time.Time | `decision.Timestamp` | — |
| `CorrelationID` | string | envelope metadata | `"chain-corr-1"` |
| `CausationID` | string | decision event ID | — |

---

## 3. Consumption Categories

### 3.1 Core Logic (Determines Direction)

| Field | Usage | Behavior |
|-------|-------|----------|
| `DecisionOutcome` | Primary branch selector | `"triggered"` → direction=long; `"not_triggered"` → direction=flat; `"insufficient"` → direction=flat + reason |

This has not changed in S250. The outcome is still the primary driver of strategy direction.

### 3.2 Behavioral Influence (S250 — NEW)

| Field | Usage | Behavior |
|-------|-------|----------|
| `DecisionSeverity` | Confidence scaling | Multiplies raw confidence by severity factor (high=1.00, moderate=0.90, low=0.80) |
| `DecisionSeverity` | Parameter adjustment | Adjusts type-specific parameters (target offsets, stop levels) by severity multipliers |
| `DecisionConfidence` | Base confidence | Provides the raw value that is then severity-scaled |

### 3.3 Traceability (Stored, Not Used for Logic)

| Field | Storage | Purpose |
|-------|---------|---------|
| `DecisionType` | `strategy.Decisions[0].Type` + `metadata["decision_type"]` | Audit trail — which decision type drove this strategy |
| `DecisionSeverity` | `strategy.Decisions[0].Severity` + `metadata["decision_severity"]` | Audit trail — severity at decision time |
| `DecisionRationale` | `strategy.Decisions[0].Rationale` + `metadata["decision_rationale"]` | Audit trail — decision's explanation |
| `DecisionConfidence` | `strategy.Decisions[0].Confidence` | Audit trail — raw confidence before scaling |
| `CorrelationID` | Strategy event envelope | End-to-end tracing |
| `CausationID` | Strategy event envelope | Direct causal link |

---

## 4. Consumption Matrix by Resolver

### 4.1 mean_reversion_entry

| Decision Field | Influences | How |
|---------------|-----------|-----|
| Outcome | Direction | `triggered` → long, else → flat |
| Confidence | Strategy.Confidence | Scaled by severity: `raw × factor` |
| Severity | Confidence scaling | high=1.00, moderate=0.90, low=0.80 |
| Severity | target_offset | high=×1.50, moderate=×1.00, low=×0.75 |
| Severity | stop_offset | high=×0.75, moderate=×1.00, low=×1.50 |
| Type | Metadata | `metadata["decision_type"]` |
| Rationale | Metadata | `metadata["decision_rationale"]` |

### 4.2 trend_following_entry

| Decision Field | Influences | How |
|---------------|-----------|-----|
| Outcome | Direction | `triggered` → long, else → flat |
| Confidence | Strategy.Confidence | Scaled by severity: `raw × factor` |
| Severity | Confidence scaling | high=1.00, moderate=0.90, low=0.80 |
| Severity | trailing_stop_pct | high=×0.75, moderate=×1.00, low=×1.50 |
| Severity | take_profit_pct | high=×1.50, moderate=×1.00, low=×0.75 |
| Type | Metadata | `metadata["decision_type"]` |
| Rationale | Metadata | `metadata["decision_rationale"]` |

---

## 5. Boundary Rules

### 5.1 What Strategy Resolvers Must NOT Do

- Import from the decision domain package
- Reference decision domain types (`decision.Outcome`, `decision.Severity`)
- Call decision evaluator functions
- Modify or recompute decision fields
- Assume specific decision types (resolvers work with any type string)

### 5.2 What Strategy Resolvers May Do

- Read decision field values as primitive strings
- Apply severity-based scaling using their own per-type maps
- Store decision context in `DecisionInput` and `Metadata`
- Build rationale strings that reference decision context
- Default to neutral behavior (×1.00) for unknown severity values

### 5.3 Data Flow Direction

```
Decision Domain → [primitive strings via actor message] → Strategy Application Layer
```

The strategy application layer owns:
- The `DecisionInput` struct (strategy domain type, not decision domain type)
- The scaling maps and multiplier tables
- The rationale construction logic
- The metadata population logic

The decision domain owns:
- The severity values themselves
- The confidence computation
- The rationale text

There is no shared code between the two domains.

---

## 6. Backward Compatibility

| Scenario | Behavior |
|----------|----------|
| Decision with unknown severity | Strategy applies ×1.00 scaling (identical to pre-S250) |
| Decision with empty severity | Strategy applies ×1.00 scaling (identical to pre-S250) |
| Decision with `"none"` severity | Strategy applies ×1.00 scaling (identical to pre-S250) |
| Decision with no rationale | `metadata["decision_rationale"]` is not set |
| New decision type added in the future | Resolver works unchanged — uses severity/outcome strings generically |

---

## 7. Impact on Downstream Consumers

### 7.1 Risk Evaluators

Risk evaluators receive `DecisionSeverity` and `DecisionRationale` via the `strategyResolvedMessage`. These values are **unchanged** — they still originate from the decision domain and pass through the strategy layer. The only difference is that the strategy's own confidence is now severity-scaled.

**Impact:** Risk evaluators that use `StrategyConfidence` for their own scaling (e.g., `position_exposure` applies ×0.95) will now scale a severity-adjusted value. This is correct behavior — the risk layer should respect the strategy's confidence, which now includes severity context.

### 7.2 Store / Writer / Gateway

Store projections, writer inserts, and gateway reads consume the `Strategy` struct. The struct shape is unchanged — no new fields, no removed fields. The only differences are:

- `Confidence` values may differ from decision confidence
- `Parameters` values may differ from base defaults
- `Metadata` has additional keys (`decision_type`, `decision_severity`, `rationale`)

No schema changes, no projection changes, no HTTP handler changes required.

### 7.3 ClickHouse

The `strategies` table stores `confidence` and `parameters` as-is (string/map fields). The values changed but the schema did not. Existing queries will see different values but will not break.

---

## 8. Testing Contract

The behavioral contract is verified by:

| Test | What It Proves |
|------|---------------|
| `TestMeanReversionEntryResolver_SeverityScalesConfidence` | Each severity level produces the expected scaled confidence |
| `TestMeanReversionEntryResolver_SeverityAdjustsParameters` | Each severity level produces the expected adjusted parameters |
| `TestTrendFollowingEntryResolver_SeverityScalesConfidence` | Same for trend following |
| `TestTrendFollowingEntryResolver_SeverityAdjustsParameters` | Same for trend following |
| `TestMeanReversionEntryResolver_DecisionInputPreservesRawConfidence` | DecisionInput carries raw confidence, not scaled |
| `TestTrendFollowingEntryResolver_DecisionInputPreservesRawConfidence` | Same for trend following |
| `TestScaleConfidence` | Scaling function works correctly for all inputs |
| `TestAdjustParam` | Parameter adjustment function works correctly |
| `TestActorChain_Signal_To_Decision_To_Strategy_To_Risk` | Full chain produces severity-aware strategy output |
| `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk` | Chain B produces severity-aware strategy output |
