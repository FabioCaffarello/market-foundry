# Execution Analytical Surface — Findings and Queryability

> S272 — Documents the execution analytical surface, its queryability, and the boundaries of what was proved.

## Analytical Surface

### ClickHouse Table: `executions`

| Column | Type | Source |
|--------|------|--------|
| event_id | String | envelope metadata |
| occurred_at | DateTime64 | envelope metadata |
| correlation_id | String | envelope metadata |
| causation_id | String | envelope metadata |
| type | String | domain: `paper_order` |
| source | String | domain: `derive` |
| symbol | String | domain: e.g. `btcusdt` |
| timeframe | UInt32 | domain: seconds |
| side | String | enum: `buy`, `sell`, `none` |
| quantity | Float64 | parsed from decimal string |
| filled_quantity | Float64 | parsed from decimal string |
| status | String | enum: 7 lifecycle states |
| risk | String | JSON: `RiskInput` struct |
| fills | String | JSON: `[]FillRecord` array |
| parameters | String | JSON: `map[string]string` |
| metadata | String | JSON: `map[string]string` |
| exec_correlation_id | String | domain chain trace |
| exec_causation_id | String | domain chain trace |
| final | UInt8 | boolean: terminal state flag |
| timestamp | DateTime64 | domain timestamp |

**Storage**: MergeTree, partitioned by `YYYYMM(timestamp)`, ordered by `(source, symbol, timeframe, type, timestamp)`.
**TTL**: 90 days.

### Write Path

```
PaperOrderSubmittedEvent → NATS subject execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}
  → writer consumer (durable: writer-execution-paper-order)
  → mapExecutionRow() → batch inserter → ClickHouse executions table
```

### Read Path

```
GET /analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60
  → ExecutionReader.QueryExecutionHistory()
  → BuildExecutionQuery() → parameterized SQL
  → ClickHouse query → row scan → FormatFloat/ParseRiskInputJSON/ParseFillsJSON/ParseMetadataJSON
  → []ExecutionIntent → JSON response
```

## Queryability

### HTTP Endpoint

`GET /analytical/execution/history`

| Parameter | Required | Description |
|-----------|----------|-------------|
| type | yes | Execution type (`paper_order`) |
| source | yes | Source (`derive`) |
| symbol | yes | Symbol (`btcusdt`) |
| timeframe | yes | Timeframe in seconds (`60`) |
| side | no | Filter: `buy`, `sell`, `none` |
| status | no | Filter: `submitted`, `filled`, `rejected`, etc. |
| since | no | Unix timestamp lower bound |
| until | no | Unix timestamp upper bound |
| limit | no | Max rows, default 50, max 500 |

### Response Contract

```json
{
  "executions": [
    {
      "type": "paper_order",
      "source": "derive",
      "symbol": "btcusdt",
      "timeframe": 60,
      "side": "buy",
      "quantity": "0.0192",
      "filled_quantity": "0.0192",
      "status": "filled",
      "risk": {
        "type": "position_exposure",
        "disposition": "approved",
        "confidence": "0.7500",
        "timeframe": 60,
        "strategy_type": "mean_reversion_entry",
        "decision_severity": "high"
      },
      "fills": [...],
      "parameters": {"target_offset": "0.03", "stop_offset": "0.01"},
      "metadata": {"decision_severity": "high"},
      "correlation_id": "chain-001",
      "causation_id": "risk-001",
      "final": true,
      "timestamp": "2026-03-20T10:00:00Z"
    }
  ],
  "source": "clickhouse",
  "meta": {
    "query_ms": 45,
    "row_count": 1
  }
}
```

## What Was Proved

1. **Serialization fidelity**: All 20 columns survive `mapExecutionRow` → simulated read-back via `FormatFloat`, `ParseRiskInputJSON`, `ParseFillsJSON`, `ParseMetadataJSON`.
2. **Enum completeness**: All 3 Side values and all 7 Status lifecycle values are faithfully serialized.
3. **Risk causal context**: `strategy_type` and `decision_severity` (added in S265) survive the JSON round-trip in the `risk` column.
4. **Fills fidelity**: Single and multiple `FillRecord` entries survive JSON serialization including `simulated` flag and sub-second timestamps.
5. **Quantity precision**: Float64 round-trip preserves 4-decimal precision within `1e-10` tolerance.
6. **Chain traceability**: The full `decision → strategy → risk → execution` causation chain is queryable — both envelope IDs and exec-specific IDs are preserved.
7. **Query builder correctness**: `BuildExecutionQuery` generates correct parameterized SQL with all optional filters (side, status, since, until) — already covered by `execution_reader_test.go`.
8. **Smoke E2E coverage**: `scripts/smoke-analytical-e2e.sh` Phase 5.8 validates the live NATS→writer→ClickHouse→reader→HTTP path for executions.

## What Was NOT Proved (Out of Scope)

- **Live ClickHouse integration**: Tests use write-path mappers and read-path parsers without a running ClickHouse instance. The live path is covered by `smoke-analytical-e2e.sh`.
- **Venue fill events**: Only `paper_order` (PaperOrderSubmittedEvent) was validated. Venue market order fills use a separate stream and are not part of the current analytical surface.
- **Aggregation queries**: No GROUP BY, time-bucket, or statistical queries were tested. The analytical surface provides row-level queryability only.
- **Multi-family cross-join**: Joining executions with risk_assessments/strategies/decisions by correlation_id was not tested at the SQL level.
- **Performance/latency**: Query performance under load was not measured.

## Existing Test Coverage Summary

| Test File | Scope | Tests |
|-----------|-------|-------|
| `writerpipeline/behavioral_roundtrip_test.go` | Write→read serialization (S255 + S272) | 17 scenarios |
| `clickhouse/execution_reader_test.go` | Query builder, JSON parsers | 18 tests |
| `scripts/smoke-analytical-e2e.sh` | Live E2E (NATS→CH→HTTP) | Phase 5.8 |
