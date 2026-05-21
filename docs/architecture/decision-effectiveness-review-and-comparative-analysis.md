# Decision Effectiveness Review and Comparative Analysis

**Stage**: S477
**Wave**: Strategy Effectiveness Measurement (S474--S478)
**Date**: 2026-03-25

---

## 1. Purpose

This document defines the effectiveness review surface and comparative analysis capability added by S477. It answers **Q-SE5: Can the system surface comparative effectiveness analysis (which decision types or strategies outperform?)**

The system can now:

1. Aggregate batch effectiveness evaluations into cohort summaries.
2. Compare cohorts side-by-side by grouping on decision_type, strategy_type, severity, or source.
3. Answer "was this decision good?" with win/loss/breakeven classification, P&L attribution, and cohort-relative context.

---

## 2. Surfaces

### 2.1 Effectiveness Summary (New -- S477)

**Endpoint**: `GET /analytical/composite/decision/effectiveness/summary`

**Required parameters**: `source`, `symbol`, `timeframe`.

**Optional parameters**:
- `group_by`: `decision_type`, `strategy_type`, `severity`, or `source`. When omitted, returns a single cohort with key `"all"`.
- `decision_type`, `strategy_type`, `severity`: Pre-aggregation filters.
- `since`, `until`: Unix seconds time range.
- `limit`: Max chains to scan (default 100, max 300).

**Response shape**:

```json
{
  "cohorts": [
    {
      "key": "rsi_oversold",
      "win_count": 5,
      "loss_count": 3,
      "breakeven_count": 1,
      "unresolved_count": 12,
      "evaluated": 21,
      "resolved": 9,
      "total_pnl": 42.50,
      "avg_pnl": 4.72,
      "total_fees": 8.25,
      "win_rate": 0.555
    }
  ],
  "source": "clickhouse",
  "meta": {
    "total_ms": 45,
    "evaluation_count": 21,
    "chains_scanned": 30,
    "excluded": 2
  }
}
```

### 2.2 Existing Surfaces (S476 -- Unchanged)

- `GET /analytical/composite/decision/effectiveness` -- single-chain lookup.
- `GET /analytical/composite/decision/effectiveness/batch` -- individual evaluations.
- `GET /analytical/composite/decision/review` -- single review bundle with effectiveness section.
- `GET /analytical/composite/decision/reviews` -- batch review bundles.

---

## 3. Comparative Analysis Design

### 3.1 Grouping Dimensions

| Dimension | Semantics | Example question |
|-----------|-----------|-----------------|
| `decision_type` | Which decision evaluator produced better outcomes? | "Is rsi_oversold more effective than ema_crossover?" |
| `strategy_type` | Which strategy resolver produces better outcomes? | "Does mean_reversion outperform trend_following?" |
| `severity` | Do higher-severity decisions produce better outcomes? | "Are high-severity decisions worth the risk?" |
| `source` | Are outcomes different across exchange sources? | "Does binance produce different effectiveness than the paper venue?" |

### 3.2 Aggregation Pipeline

1. Fetch chains via `CompositeReader.QueryChainsBatch()` (same S296 infrastructure).
2. Classify each chain via `effectiveness.Classify()` (same S475 domain model).
3. Enrich attribution from composite chain decision/strategy stages.
4. Apply pre-aggregation filters (decision_type, strategy_type, severity).
5. Group attributions by dimension value (or "all" when ungrouped).
6. Compute per-group CohortSummary.
7. Sort cohorts by evaluated count descending.

### 3.3 CohortSummary Semantics

| Field | Computation | Notes |
|-------|-------------|-------|
| `win_count` | Count of `outcome=win` | |
| `loss_count` | Count of `outcome=loss` | |
| `breakeven_count` | Count of `outcome=breakeven` | |
| `unresolved_count` | Count of `outcome=unresolved` | Dominates in single-leg-fill regime |
| `evaluated` | Sum of all outcome counts | |
| `resolved` | `win + loss + breakeven` | Excludes unresolved |
| `total_pnl` | Sum of `net_pnl` over resolved chains | 0 when all unresolved |
| `avg_pnl` | `total_pnl / resolved` | 0 when resolved=0 |
| `total_fees` | Sum of `total_fees` over all evaluated | Includes unresolved |
| `win_rate` | `win_count / resolved` | Ratio 0.0--1.0, NOT percentage. 0 when resolved=0 |

---

## 4. Architecture Integration

### 4.1 Data Flow

```
ClickHouse (existing)
    |
    v
CompositeReader.QueryChainsBatch() [S296]
    |
    v
effectiveness.Classify() [S475]
    |
    v
enrichFromChain() [S476]
    |
    v
aggregateCohort() / aggregateByDimension() [S477 -- NEW]
    |
    v
EffectivenessSummaryReply
```

### 4.2 Code Location

| Component | File |
|-----------|------|
| Contracts | `internal/application/analyticalclient/effectiveness_contracts.go` |
| Use case | `internal/application/analyticalclient/get_effectiveness_summary.go` |
| Handler | `internal/interfaces/http/handlers/composite.go` |
| Routes | `internal/interfaces/http/routes/analytical.go` |
| Composition | `cmd/gateway/compose.go` |

### 4.3 No New Infrastructure

- No new ClickHouse tables or schema changes.
- No new NATS subjects.
- No write-path changes.
- Reuses existing CompositeReader, effectiveness.Classify, enrichFromChain.

---

## 5. Guard Rails

| Guard Rail | Status |
|-----------|--------|
| No new exchange connectivity | OBSERVED |
| No new ClickHouse tables | OBSERVED |
| No portfolio analytics | OBSERVED -- single symbol/source/timeframe partition |
| No risk-adjusted metrics | OBSERVED -- raw P&L and win/loss only |
| No real-time streaming | OBSERVED |
| No UI or dashboard work | OBSERVED |
| No ML or predictive scoring | OBSERVED |
| Additive only | OBSERVED -- zero changes to existing behavior |

---

## 6. References

- [Effectiveness Query Surfaces, Inputs, Outputs, and Limitations](effectiveness-query-surfaces-batch-evaluation-inputs-outputs-and-limitations.md)
- [Measurement Read Surfaces and Batch Evaluation](measurement-read-surfaces-and-batch-evaluation.md)
- [Strategy Effectiveness Wave Charter](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
