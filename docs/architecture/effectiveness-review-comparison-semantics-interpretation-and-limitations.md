# Effectiveness Review: Comparison Semantics, Interpretation, and Limitations

**Stage**: S477
**Wave**: Strategy Effectiveness Measurement (S474--S478)
**Date**: 2026-03-25

---

## 1. Purpose

This document explains how to correctly interpret effectiveness review results and comparative analysis outputs, what the numbers mean, what they do not mean, and where the system's measurement boundaries lie.

---

## 2. Interpreting CohortSummary

### 2.1 Win Rate

`win_rate = win_count / resolved`

- Ratio 0.0 to 1.0, NOT a percentage. Multiply by 100 for display if needed.
- Computed only over **resolved** chains (win + loss + breakeven). Unresolved chains are excluded.
- When `resolved = 0`, `win_rate = 0`. This does not mean "0% win rate" -- it means "no classifiable data."
- A high unresolved count alongside a non-zero win_rate means the win_rate is based on a small subsample. Treat with caution.

### 2.2 Average P&L

`avg_pnl = total_pnl / resolved`

- Computed over resolved chains only.
- Includes breakeven chains (near-zero P&L).
- When `resolved = 0`, `avg_pnl = 0`. This is a sentinel, not a measurement.
- Does not account for capital deployed. A $1 avg P&L on $100 trades is different from $1 on $10,000 trades.

### 2.3 Total Fees

`total_fees = sum(total_fees) over all evaluated chains`

- Includes fees from unresolved chains (capital deployed but not yet returned).
- Futures fees may be zero due to the S428 limitation.
- Fee normalization is best-effort; cross-asset fee comparison requires external context.

### 2.4 Unresolved Dominance

In the current pipeline scope, most chains will be `unresolved`:

- Single-leg fills without a paired exit within session scope are always unresolved.
- Paper/dry-run fills with zero cost basis are always unresolved.
- The `resolved` count will be low until the system processes round-trip pairs.

**Implication**: Comparative analysis between cohorts is meaningful only when both cohorts have a sufficient `resolved` count. Comparing two cohorts where `resolved < 5` is noise, not signal.

---

## 3. Comparison Semantics

### 3.1 What Comparison Answers

When using `group_by=decision_type`:

> "Within this source/symbol/timeframe partition, which decision evaluator type has historically produced better P&L outcomes?"

When using `group_by=strategy_type`:

> "Which strategy resolver type has historically produced better P&L outcomes?"

When using `group_by=severity`:

> "Do higher-severity decisions correlate with better outcomes?"

When using `group_by=source`:

> "Are outcomes different across exchange sources?"

### 3.2 What Comparison Does NOT Answer

- **Causal attribution**: A higher win_rate for decision_type A vs B does not mean A is a better algorithm. Market conditions, timing, and selection bias all confound the comparison.
- **Statistical significance**: No p-values, confidence intervals, or hypothesis tests are computed. Small sample sizes can produce misleading differences.
- **Risk-adjusted performance**: Win rate and avg P&L say nothing about drawdown, Sharpe ratio, or tail risk.
- **Cross-symbol comparison**: Each query is scoped to one symbol. Comparing effectiveness across symbols requires separate queries and external reasoning.
- **Cross-session comparison**: Attribution is scoped to the current session's fill data. Multi-session strategies are not tracked.
- **Forward-looking prediction**: Past effectiveness does not predict future effectiveness. Market regimes change.

### 3.3 Minimum Sample Guidance

| Resolved count | Interpretation confidence |
|---------------|--------------------------|
| 0 | No data. No conclusion possible. |
| 1--4 | Anecdotal. Do not act on this. |
| 5--19 | Directional hint. Very low confidence. |
| 20--49 | Moderate signal. Useful for hypothesis generation, not confirmation. |
| 50+ | Reasonable sample for operational decisions, with caveats above. |

These thresholds are heuristic, not statistically derived. The system does not enforce them.

---

## 4. Limitations

### 4.1 Structural Limitations

1. **Single-leg fill dominance**: Until the pipeline processes round-trip pairs, most chains are `unresolved`. The summary endpoint will return high `unresolved_count` and low `resolved` counts.

2. **No paired matching endpoint**: `ClassifyPair()` exists in the domain model but is not exposed as an HTTP endpoint. Round-trip P&L requires programmatic use or future endpoint work.

3. **Session-scoped attribution**: Effectiveness is computed from fills within one session. Strategies that span multiple sessions are not attributed correctly.

4. **Futures fees are zero**: S428 limitation means futures fee impact is understated in `total_fees` and `avg_pnl`.

5. **Post-filter may reduce sample size**: The `limit` parameter controls chains scanned, not evaluations returned. Pre-aggregation filters can further reduce the sample.

### 4.2 Comparison Limitations

1. **No normalized comparison**: Cohorts may have different sample sizes, time ranges, or market conditions. The system does not normalize for these.

2. **No statistical tests**: Differences between cohorts may be due to chance. No significance testing is performed.

3. **No temporal decomposition**: The summary aggregates over the full time range. There is no breakdown by time period (hourly, daily) within a single query.

4. **No survivorship correction**: If a strategy type was disabled mid-session, its cohort only contains data from when it was active.

5. **Empty dimension values**: If a chain's decision_type or strategy_type is empty (e.g., enrichment failed), it appears in a cohort with key `"(unknown)"`. This is a data quality signal, not an error.

### 4.3 What This Surface Does NOT Do

- No dashboards or visualizations (NG-SE6).
- No alerting on effectiveness thresholds (NG-SE10).
- No ML-based scoring or prediction (NG-SE4).
- No cross-symbol or portfolio-level aggregation (NG-SE1).
- No risk-adjusted metrics (NG-SE2).
- No backtesting or historical replay (NG-SE15).

---

## 5. Correct Usage Patterns

### 5.1 Review a Single Decision

```
GET /analytical/composite/decision/review?correlation_id=X&symbol=Y
```

Returns the full evidence bundle including effectiveness section with outcome, P&L, fees, and explanation.

### 5.2 Review a Cohort

```
GET /analytical/composite/decision/effectiveness/summary?source=binance&symbol=btcusdt&timeframe=60
```

Returns aggregated win/loss/breakeven counts, P&L, and win_rate for all evaluated chains.

### 5.3 Compare Decision Types

```
GET /analytical/composite/decision/effectiveness/summary?source=binance&symbol=btcusdt&timeframe=60&group_by=decision_type
```

Returns one CohortSummary per decision evaluator type, sorted by evaluated count descending.

### 5.4 Filter Then Compare

```
GET /analytical/composite/decision/effectiveness/summary?source=binance&symbol=btcusdt&timeframe=60&severity=high&group_by=strategy_type
```

First filters to high-severity decisions, then groups by strategy type.

---

## 6. References

- [Decision Effectiveness Review and Comparative Analysis](decision-effectiveness-review-and-comparative-analysis.md)
- [Effectiveness Query Surfaces, Inputs, Outputs, and Limitations](effectiveness-query-surfaces-batch-evaluation-inputs-outputs-and-limitations.md)
- [Capabilities, Questions, and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
