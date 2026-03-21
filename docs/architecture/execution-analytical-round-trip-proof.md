# Execution Analytical Round-Trip Proof

> S272 — Proves execution paper_order events survive the complete analytical serialization cycle.

## Objective

Close the analytical debt registered in S269: the execution paper round-trip through the
NATS → writer → ClickHouse → reader → HTTP analytical path had not been proved at the
serialization level. This document records the proof.

## Round-Trip Path

```
PaperOrderSubmittedEvent
  → mapExecutionRow()        (write path: domain → []any row)
  → ClickHouse INSERT        (executions table, 20-column DDL)
  → QueryExecutionHistory()  (read path: row scan → domain types)
  → HTTP JSON response       (GET /analytical/execution/history)
```

## What Was Proved

### 1. Basic Paper Order Serialization (Scenario 9)

A fully-populated `PaperOrderSubmittedEvent` survives the complete write→read cycle:
- Envelope metadata: `event_id`, `occurred_at`, `correlation_id`, `causation_id`
- Core fields: `type`, `source`, `symbol`, `timeframe`
- Side enum (`buy`), Status enum (`filled`)
- Quantity/FilledQuantity via `parseFloat` → `FormatFloat` round-trip
- Risk causal context (JSON struct) with `strategy_type` and `decision_severity`
- Fills array (JSON) with price, quantity, fee, simulated flag, timestamp
- Parameters and Metadata (JSON maps)
- Exec-specific `correlation_id` / `causation_id` (distinct from envelope)
- `final` boolean flag

### 2. Side Enum Fidelity (Scenario 10)

All three `Side` values survive: `buy`, `sell`, `none`.

### 3. Status Lifecycle Enum Fidelity (Scenario 11)

All seven `Status` values survive: `submitted`, `sent`, `accepted`, `filled`,
`partially_filled`, `rejected`, `cancelled`.

### 4. Risk Causal Context — Strategy-Type-Aware (Scenarios 12a, 12b)

The `RiskInput` JSON struct preserves:
- Counter-trend: `strategy_type=mean_reversion_entry`, `decision_severity=high`
- Pro-trend: `strategy_type=trend_following_entry`, `decision_severity=moderate`
- Confidence string with 4-decimal precision

### 5. Multiple Fills (Scenario 13)

An execution with two `FillRecord` entries survives JSON round-trip with all fields
(price, quantity, fee, simulated, timestamp) preserved per fill.

### 6. Empty Fills — Submitted Order (Scenario 14)

A submitted (non-terminal) order with no fills and `final=false` serializes correctly:
empty fills array, zero filled_quantity, false final flag.

### 7. Quantity Precision (Scenario 15)

Float64 round-trip preserves quantity values within `1e-10` tolerance for:
`0.0192`, `0.0150`, `1.5000`, `0.0001`, `100.0000`, `0.0000`.

### 8. Full Four-Stage Chain (Scenario 16)

The complete `decision → strategy → risk → execution` chain survives with:
- **Correlation ID** preserved across all four stages (`full-chain-001`)
- **Causation chain**: `signal → dec-chain-001 → strat-chain-001 → risk-chain-001 → exec-chain-001`
- **Decision severity** propagates through all stages to execution's `risk.decision_severity`
- **Strategy type** propagates through all stages to execution's `risk.strategy_type`
- **Confidence ordering**: `risk ≤ strategy ≤ decision` — execution inherits risk confidence
- **Quantity** matches risk `max_position_size`
- **Parameters** propagate from strategy through execution (`target_offset`, `stop_offset`)
- **Metadata** carries `decision_severity` and `strategy_type` from the chain

### 9. Rejected Order (Scenario 17)

A rejected execution with `side=none`, `status=rejected`, `disposition=rejected`,
zero fills, and `decision_severity=low` serializes correctly.

## Row Layout Reference

```
[0]  event_id              string
[1]  occurred_at           time.Time
[2]  correlation_id        string     (envelope)
[3]  causation_id          string     (envelope)
[4]  type                  string     ("paper_order")
[5]  source                string
[6]  symbol                string
[7]  timeframe             uint32
[8]  side                  string     (enum: buy|sell|none)
[9]  quantity              float64
[10] filled_quantity       float64
[11] status                string     (enum: 7 lifecycle states)
[12] risk                  string     (JSON: RiskInput struct)
[13] fills                 string     (JSON: []FillRecord)
[14] parameters            string     (JSON: map[string]string)
[15] metadata              string     (JSON: map[string]string)
[16] exec_correlation_id   string     (domain chain)
[17] exec_causation_id     string     (domain chain)
[18] final                 bool
[19] timestamp             time.Time
```

## Evidence

- **Test file**: `internal/adapters/clickhouse/writerpipeline/behavioral_roundtrip_test.go`
- **Tests added**: Scenarios 9–17 (9 new test functions)
- **All tests pass**: `go test ./internal/adapters/clickhouse/writerpipeline/... -count=1` — PASS
- **Zero regressions**: Existing S255 scenarios 1–8 remain green
