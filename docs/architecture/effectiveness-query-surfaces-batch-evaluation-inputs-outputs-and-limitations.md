# Effectiveness Query Surfaces -- Batch Evaluation Inputs, Outputs, and Limitations

**Stage**: S476
**Date**: 2026-03-25
**Wave**: Strategy Effectiveness Measurement (S474--S478)

---

## 1. Purpose

This document specifies the exact inputs, outputs, semantics, and limitations of the effectiveness query surfaces introduced by S476. It serves as a reference contract for consumers of the effectiveness endpoints and the `DecisionReviewBundle` effectiveness section.

---

## 2. Query Surfaces

### 2.1 Single-Chain Effectiveness

**Path**: `GET /analytical/composite/decision/effectiveness`

#### Inputs

| Parameter | Type | Required | Semantics |
|-----------|------|----------|-----------|
| `correlation_id` | string | yes | CorrelationID of the decision chain |
| `symbol` | string | yes | Symbol for S301 cross-symbol isolation |

#### Outputs

```json
{
  "evaluations": [
    {
      "outcome": "unresolved",
      "realized_pnl": 0,
      "total_fees": 0.5,
      "gross_pnl": 5000.0,
      "net_pnl": 4999.5,
      "entry_cost_basis": 5000.0,
      "fill_count": 1,
      "correlation_id": "corr-001",
      "decision_type": "ema_crossover",
      "decision_severity": "high",
      "strategy_type": "trend_following",
      "side": "buy",
      "symbol": "BTCUSDT",
      "source": "binance_spot",
      "timeframe": 60,
      "execution_status": "filled",
      "simulated": false
    }
  ],
  "source": "clickhouse",
  "meta": {
    "total_ms": 42,
    "evaluation_count": 1,
    "chains_scanned": 1,
    "excluded": 0
  }
}
```

#### Semantics

- Returns 0 evaluations if: chain has no execution, execution is rejected, or chain not found.
- Returns 1 evaluation for any other terminal or non-terminal execution status.
- The `excluded` counter tracks rejected orders that were scanned but produced no evaluation.

---

### 2.2 Batch Effectiveness Evaluation

**Path**: `GET /analytical/composite/decision/effectiveness/batch`

#### Inputs

| Parameter | Type | Required | Default | Semantics |
|-----------|------|----------|---------|-----------|
| `source` | string | yes | -- | Exchange source identifier |
| `symbol` | string | yes | -- | Trading pair |
| `timeframe` | int | yes | -- | Candle interval in seconds |
| `decision_type` | string | no | -- | Filter by decision evaluator type (e.g., `ema_crossover`) |
| `strategy_type` | string | no | -- | Filter by strategy resolver type (e.g., `trend_following`) |
| `severity` | string | no | -- | Filter by decision severity (`none`, `low`, `moderate`, `high`) |
| `effectiveness` | string | no | -- | Filter by outcome (`win`, `loss`, `breakeven`, `unresolved`) |
| `since` | int64 | no | 0 | Unix timestamp, inclusive lower bound |
| `until` | int64 | no | 0 | Unix timestamp, inclusive upper bound |
| `limit` | int | no | 20 | Max results returned (clamped to 100) |

#### Outputs

Same structure as single-chain response, but `evaluations` array may contain 0..N items.

#### Semantics

- Chains are fetched from ClickHouse in descending timestamp order (most recent first).
- Post-filters (`decision_type`, `strategy_type`, `severity`, `effectiveness`) are applied after chain fetch and classification.
- When post-filters are active, the system fetches `min(limit * 3, 100)` chains to compensate for filter exclusions.
- `chains_scanned` reflects total chains fetched from ClickHouse before filtering.
- `excluded` reflects rejected orders that produced no classification.
- `evaluation_count` reflects final count after all filters.

---

### 2.3 Decision Review Bundle Extension

**Path**: Existing `GET /analytical/composite/decision/review` and `GET /analytical/composite/decision/reviews`

#### Added Field

```json
{
  "effectiveness": {
    "outcome": "unresolved",
    "realized_pnl": 0,
    "gross_pnl": 5000.0,
    "net_pnl": 4999.5,
    "total_fees": 0.5,
    "entry_cost_basis": 5000.0,
    "fill_count": 1,
    "simulated": false,
    "explanation": "Effectiveness unresolved: buy execution has 1 fill(s) with cost_basis=5000.000000, fees=0.500000 but no paired exit within session scope."
  }
}
```

#### Presence Rules

| Condition | `effectiveness` field |
|-----------|----------------------|
| No execution stage | `null` / absent |
| Rejected execution | `null` / absent |
| Submitted/sent/accepted execution | Present, outcome=`unresolved` |
| Cancelled with no fills | Present, outcome=`unresolved` |
| Filled with cost_basis=0 (dry-run) | Present, outcome=`unresolved` |
| Filled with fills (single-leg) | Present, outcome=`unresolved` |
| Not-triggered decision (no execution) | `null` / absent |

---

## 3. P&L Computation Rules

### 3.1 Single-Leg (Most Common Case)

For a single buy or sell fill without a paired opposite-side fill:

- `gross_pnl` = total cost basis from fills
- `net_pnl` = gross_pnl - total fees
- `outcome` = `unresolved` (always, since no exit price is available)

### 3.2 Paired Round-Trip (via `ClassifyPair`)

For a matched entry/exit pair:

- **Long (buy entry, sell exit)**: `gross_pnl` = exit_cost_basis - entry_cost_basis
- **Short (sell entry, buy exit)**: `gross_pnl` = entry_cost_basis - exit_cost_basis
- `net_pnl` = gross_pnl - (entry_fees + exit_fees)
- `outcome`:
  - `|net_pnl| <= 0.0001` -> `breakeven`
  - `net_pnl > 0` -> `win`
  - `net_pnl < 0` -> `loss`

### 3.3 Fee Handling

- Fees are sourced from `FillRecord.Fee` (S428 normalized).
- Spot fills: actual commission from venue.
- Futures fills: `Fee = "0"` (not available from RESULT response).
- Paper/dry-run fills: `Fee = "0"`, `CostBasis = "0"`.

---

## 4. Enrichment from Composite Chain

When the effectiveness use case classifies a chain, it enriches the attribution with upstream context:

1. If `DecisionType` is empty on the execution's `RiskInput`, it falls back to `chain.Decision.Type`.
2. If `DecisionSeverity` is empty, it falls back to `chain.Decision.Severity`.
3. If `StrategyType` is empty, it falls back to `chain.Strategy.Type`.

This ensures that effectiveness records carry full context even when the execution's `RiskInput` has incomplete metadata.

---

## 5. Limitations

### 5.1 Structural Limitations

| Limitation | Impact | Mitigation |
|-----------|--------|------------|
| Single-leg fills always `unresolved` | Most effectiveness evaluations will be `unresolved` in current pipeline | Document; paired evaluation available via `ClassifyPair` when exit data exists |
| No cross-session pairing | Multi-session strategies cannot have complete round-trip P&L | NG-SE18; future wave scope |
| Futures fees are zero | Fee impact understated for futures segment | Known S428 limitation; document |
| Dry-run fills have zero cost basis | All dry-run evaluations are `unresolved` | By design; dry-run has no real P&L |

### 5.2 Query Limitations

| Limitation | Impact | Mitigation |
|-----------|--------|------------|
| Post-filter may return fewer than `limit` results | Consumer must handle `evaluation_count < limit` | Over-fetch 3x when filters active |
| No aggregation in batch endpoint | Consumer receives individual evaluations, not summaries | S477 will add cohort aggregation |
| No pagination cursor | Large result sets require time-range windowing | Use `since`/`until` for windowed queries |

### 5.3 Coverage Gaps

| Gap | Status | Resolution |
|-----|--------|------------|
| Comparative analysis (which type outperforms?) | Not in S476 | S477 scope |
| Cohort aggregation (win rate, average P&L) | Not in S476 | S477 scope |
| Paired round-trip matching from ClickHouse | Not automated | `ClassifyPair` available for manual/programmatic use |

---

## 6. Test Coverage

### 6.1 Domain Tests (15 tests)

| Test | Validates |
|------|----------|
| `TestValidOutcome` | Outcome type validation |
| `TestClassify_RejectedReturnsNil` | Rejected order exclusion |
| `TestClassify_NonTerminalIsUnresolved` | Non-terminal status handling |
| `TestClassify_CancelledNoFillsIsUnresolved` | Cancelled-before-fill rule |
| `TestClassify_FilledSingleLegIsUnresolved` | Single-leg classification |
| `TestClassify_ZeroCostBasisIsUnresolved` | Dry-run/paper classification |
| `TestClassify_PartiallyFilledIsUnresolved` | Partial fill handling |
| `TestClassify_AttributionCarriesContext` | Context metadata propagation |
| `TestClassifyPair_WinRoundTrip` | Long win round-trip P&L |
| `TestClassifyPair_LossRoundTrip` | Long loss round-trip P&L |
| `TestClassifyPair_ShortWin` | Short win round-trip P&L |
| `TestClassifyPair_Breakeven` | Breakeven threshold |
| `TestClassifyPair_RejectedReturnsNil` | Rejected pair exclusion |
| `TestExplain_AllOutcomes` | Human-readable explanations |
| `TestClassify_MultipleFillsAggregated` | Multi-fill aggregation |

### 6.2 Use Case / Integration Tests (15 tests)

| Test | Validates |
|------|----------|
| `TestGetEffectiveness_Single_FilledChain` | Single-chain effectiveness query |
| `TestGetEffectiveness_Single_RejectedExcluded` | Rejected exclusion in single mode |
| `TestGetEffectiveness_Single_NoExecution` | No execution stage handling |
| `TestGetEffectiveness_Single_MissingSymbol` | S301 validation |
| `TestGetEffectiveness_Batch_Success` | Batch evaluation |
| `TestGetEffectiveness_Batch_SeverityFilter` | Severity post-filter |
| `TestGetEffectiveness_Batch_StrategyTypeFilter` | Strategy type post-filter |
| `TestGetEffectiveness_Batch_ValidationErrors` | Input validation (3 sub-tests) |
| `TestGetEffectiveness_NilUseCase` | Nil use case degradation |
| `TestGetEffectiveness_Batch_MixedRejectedAndFilled` | Mixed batch handling |
| `TestGetDecisionReview_EffectivenessSection_FilledExecution` | Review bundle extension |
| `TestGetDecisionReview_EffectivenessSection_NoExecution` | Absent effectiveness when no execution |
| `TestGetDecisionReview_EffectivenessSection_RejectedExecution` | Absent effectiveness for rejected |
| `TestGetDecisionReview_ExplanationIncludesEffectiveness` | Explanation enrichment |

---

## 7. References

- [Measurement Read Surfaces and Batch Evaluation](measurement-read-surfaces-and-batch-evaluation.md)
- [Strategy Effectiveness Wave Charter](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [Capabilities and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
