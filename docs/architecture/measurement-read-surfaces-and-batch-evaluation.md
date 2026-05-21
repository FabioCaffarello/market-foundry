# Measurement Read Surfaces and Batch Evaluation

**Stage**: S476
**Date**: 2026-03-25
**Wave**: Strategy Effectiveness Measurement (S474--S478)
**Predecessor**: S475 (Canonical Effectiveness Model and Attribution Semantics)

---

## 1. Purpose

This document describes the read surfaces and batch evaluation infrastructure added by S476 to make effectiveness measurement operational and repeatable. The system can now classify decision chains by outcome (win/loss/breakeven/unresolved), attribute P&L to originating decisions, and evaluate cohorts of decisions through HTTP endpoints.

---

## 2. Architecture Overview

### 2.1 Data Flow

```
ClickHouse (existing tables: executions, decisions, strategies, risk_assessments, signals)
    |
    v
CompositeReader.QueryChainsBatch() -- 5 independent queries per chain
    |
    v
GetEffectivenessUseCase -- iterates chains, calls effectiveness.Classify()
    |
    v
effectiveness.Attribution -- outcome, P&L, fees, context metadata
    |
    v
HTTP Response (JSON)
```

### 2.2 Key Principle: No New Tables

Effectiveness is computed entirely from existing `FillRecord` data in the executions table. The `effectiveness` package is a pure domain computation layer. No materialized views, no CDC, no schema changes.

### 2.3 Computation Model

Effectiveness classification is **deterministic and read-path only**:

1. Fetch composite execution chains from ClickHouse (existing `CompositeReader`).
2. For each chain with an execution stage, call `effectiveness.Classify(intent)`.
3. Rejected orders return `nil` (excluded from evaluation).
4. Non-terminal or cancelled-without-fill orders are `unresolved`.
5. Single-leg fills (entry without paired exit) are `unresolved` with cost basis recorded.
6. Paired round-trips (via `ClassifyPair`) produce `win`, `loss`, or `breakeven`.

---

## 3. Read Surfaces

### 3.1 Single-Chain Effectiveness

**Endpoint**: `GET /analytical/composite/decision/effectiveness`

**Parameters**:
| Parameter | Required | Description |
|-----------|----------|-------------|
| `correlation_id` | yes | Chain identifier |
| `symbol` | yes | S301 isolation |

**Response**: `EffectivenessReply` with 0-1 evaluations.

### 3.2 Batch Effectiveness Evaluation

**Endpoint**: `GET /analytical/composite/decision/effectiveness/batch`

**Parameters**:
| Parameter | Required | Description |
|-----------|----------|-------------|
| `source` | yes | Exchange source |
| `symbol` | yes | Trading pair |
| `timeframe` | yes | Candle interval |
| `decision_type` | no | Filter by decision evaluator type |
| `strategy_type` | no | Filter by strategy resolver type |
| `severity` | no | Filter by decision severity |
| `effectiveness` | no | Filter by outcome (win/loss/breakeven/unresolved) |
| `since` | no | Unix timestamp, inclusive |
| `until` | no | Unix timestamp, inclusive |
| `limit` | no | Default 20, max 100 |

**Response**: `EffectivenessReply` with list of `effectiveness.Attribution` records.

### 3.3 Decision Review Bundle Extension

The existing `DecisionReviewBundle` (S471) now includes an optional `effectiveness` section:

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
    "explanation": "Effectiveness unresolved: buy execution has 1 fill(s)..."
  }
}
```

**Presence rules**:
- Present when execution reached terminal state (filled/cancelled with fills).
- Absent for rejected executions (no classifiable outcome).
- Absent when no execution stage exists in the chain.

### 3.4 Explanation Enrichment

The `DecisionReviewBundle.Explanation` field now includes effectiveness information when available, appended after execution output and before consistency checks.

---

## 4. Domain Types

### 4.1 effectiveness.Outcome

```go
type Outcome string

const (
    OutcomeWin        Outcome = "win"
    OutcomeLoss       Outcome = "loss"
    OutcomeBreakeven  Outcome = "breakeven"
    OutcomeUnresolved Outcome = "unresolved"
)
```

### 4.2 effectiveness.Attribution

Links an outcome to its originating decision chain with:
- P&L fields: `RealizedPnL`, `GrossPnL`, `NetPnL`, `TotalFees`, `EntryCostBasis`
- Context: `CorrelationID`, `DecisionType`, `DecisionSeverity`, `StrategyType`, `Side`, `Symbol`, `Source`, `Timeframe`
- Execution metadata: `ExecutionStatus`, `FillCount`, `Simulated`

### 4.3 Classification Rules

| Execution Status | Fills | Classification |
|-----------------|-------|---------------|
| rejected | any | excluded (nil) |
| submitted/sent/accepted | any | unresolved |
| cancelled | 0 | unresolved |
| cancelled | > 0 | unresolved (partial) |
| partially_filled | any | unresolved (non-terminal) |
| filled | > 0, cost_basis=0 | unresolved (dry-run) |
| filled | > 0, cost_basis > 0 | unresolved (single-leg, no exit) |
| paired entry+exit | both filled | win/loss/breakeven by net P&L |

### 4.4 Breakeven Threshold

`BreakevenThreshold = 0.0001` (quote-asset units). Net P&L within this absolute tolerance is classified as breakeven.

---

## 5. Integration Points

### 5.1 Reuse of Existing Infrastructure

| Component | Reused From | Purpose |
|-----------|------------|---------|
| `CompositeReader` | S296/S298 | Chain assembly from 5 ClickHouse tables |
| `FillRecord` | S428 | Fee-normalized fill data |
| `DecisionReviewBundle` | S471 | Review surface extended with effectiveness |
| `CompositeWebHandler` | S297 | HTTP handler pattern |
| `AnalyticalFamilyDeps` | S297 | Conditional route registration |

### 5.2 Wiring

```
compose.go
  -> analyticalclient.NewGetEffectivenessUseCase(compositeReader, logger)
  -> routes.AnalyticalFamilyDeps.GetEffectiveness
  -> handlers.CompositeHandlerDeps.GetEffectiveness
  -> compositeHandler.GetEffectiveness / GetEffectivenessBatch
```

---

## 6. Limitations

1. **Single-leg attribution only.** Without paired exit fills in the same session, outcome is always `unresolved`. This is the most common case in the current pipeline.
2. **No cross-session pairing.** Entry in session A and exit in session B cannot be paired (NG-SE18).
3. **Paper/dry-run fills have zero cost basis.** These are always `unresolved`.
4. **Futures fees are "0" from venue response.** Fee impact is understated for futures segments.
5. **No mark-to-market.** Open positions have no unrealized P&L computation (NG-SE11).
6. **Batch evaluation fetches 3x limit when filters are active.** Post-filter exclusions may reduce returned count below requested limit.

---

## 7. References

- [Strategy Effectiveness Wave Charter](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [Capabilities and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
- [S474 Charter Report](../stages/stage-s474-strategy-effectiveness-charter-report.md)
