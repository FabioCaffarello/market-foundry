# Live Analytical Execution Proof

> S277 — Proves the complete, live, reproducible analytical round-trip for execution paper orders.

## Objective

Demonstrate that a `PaperOrderSubmittedEvent` traverses the full analytical path and survives with field-level coherence:

```
execution event → mapExecutionRow → InsertBatch → ClickHouse (executions table)
→ QueryExecutionHistory → []ExecutionIntent → domain assertion
```

This proof closes the gap between serialization-only round-trip tests (S272) and full operational infrastructure smoke tests (`smoke-analytical-e2e.sh`), by providing a **deterministic, self-contained, live ClickHouse test** that exercises real persistence and real query execution.

## Proven Path

### Write Side

| Step | Component | Evidence |
|------|-----------|----------|
| 1 | `PaperOrderSubmittedEvent` fixture | Deterministic test data with all 20 columns populated |
| 2 | Row construction (mapExecutionRow layout) | Column order matches DDL `006_create_executions.sql` |
| 3 | `Client.InsertBatch` | Real ClickHouse native protocol batch insert |
| 4 | `executions` table | MergeTree engine, partition by `toYYYYMM(timestamp)` |

### Read Side

| Step | Component | Evidence |
|------|-----------|----------|
| 5 | `BuildExecutionQuery` | Parameterized SELECT with mandatory + optional filters |
| 6 | `Client.Query` | Real ClickHouse native protocol query |
| 7 | Row scanning (16 columns) | Scan into typed locals, parse JSON fields |
| 8 | `[]ExecutionIntent` | Domain objects with Risk, Fills, Parameters, Metadata |

## Scenarios Proven (LAE-1 through LAE-9)

### LAE-1: Basic Queryability
A paper order event written via `InsertBatch` is immediately queryable via `QueryExecutionHistory`. Proves that the write path produces data the read path can consume.

### LAE-2: All 16 Query Columns Survive Round-Trip
Every column in the SELECT clause (`type`, `source`, `symbol`, `timeframe`, `side`, `quantity`, `filled_quantity`, `status`, `risk`, `fills`, `parameters`, `metadata`, `exec_correlation_id`, `exec_causation_id`, `final`, `timestamp`) is asserted at the domain level after the round-trip.

### LAE-3: Side and Status Filters
- `side=buy` returns only buy orders.
- `status=partially_filled` returns only partially filled orders.
- Combined `side=buy AND status=filled` narrows correctly.

### LAE-4: Time-Range Filters
- `since` (inclusive) filters out events before the specified timestamp.
- `until` (inclusive) filters out events after the specified timestamp.
- Boundary precision: DateTime64(3) with second-level filter granularity.

### LAE-5: RiskInput JSON Fidelity
The full `RiskInput` struct survives including `strategy_type` and `decision_severity` (S265 additions). Proves that the behavioral causal chain is preserved in the analytical store.

### LAE-6: FillRecord Array Fidelity
Single and multi-fill arrays survive with `price`, `quantity`, `fee`, `simulated`, and `timestamp` fields intact.

### LAE-7: Parameters and Metadata Maps
`map[string]string` fields survive JSON serialization and deserialization with key/value fidelity.

### LAE-8: Multi-Symbol Partition Isolation
Events for `btcusdt/60` and `ethusdt/300` are independently queryable. Cross-partition queries return zero results correctly (no data leakage).

### LAE-9: Full Field-Level Coherence
Every field of the emitted fixture is compared against the queried result with appropriate tolerances:
- String fields: exact match
- Float fields: 1e-10 tolerance
- Timestamp: ≤1 second drift (DateTime64(3) precision)
- Boolean: exact match

## Test Infrastructure

| Aspect | Detail |
|--------|--------|
| Test file | `internal/adapters/clickhouse/live_execution_analytical_test.go` |
| Gate | `CLICKHOUSE_DSN` env var; skipped when not set |
| Table lifecycle | Created before test, dropped after test |
| CI behavior | Skipped in unit-test CI; available in integration CI with ClickHouse service |
| Determinism | Fixed timestamps, deterministic event IDs |

## Relationship to Prior Proofs

| Stage | What It Proved | What S277 Adds |
|-------|---------------|----------------|
| S255 | Serialization round-trip (mapXxxRow → ParseXxxJSON) | **Real ClickHouse persistence** |
| S272 | 26 sub-tests on mapper/parser fidelity | **Live query execution with filters** |
| smoke-analytical-e2e.sh | Full stack with docker compose | **Deterministic, self-contained, no docker required** |

## Limitations

1. **No NATS consumer in the loop**: The test inserts rows directly via `InsertBatch`, bypassing the NATS consumer. The consumer path is proven separately by `smoke-analytical-e2e.sh`.
2. **Single-node ClickHouse**: The test targets a single ClickHouse instance, not a cluster.
3. **No concurrent writers**: The test does not prove behavior under concurrent inserts.
4. **No aggregation queries**: `QueryExecutionHistory` is a simple filter+scan; GROUP BY / aggregation queries are not exercised.
5. **Ingested-at not asserted**: The `ingested_at` column (DEFAULT now64(3)) is not in the SELECT; its value is not verified.
