# Analytical Execution Queryability — Findings

> S277 — Operational findings on the execution analytical surface after live round-trip proof.

## Summary

The execution analytical surface (`executions` table → `ExecutionReader` → `GetExecutionHistoryUseCase` → `GET /analytical/execution/history`) is **fully queryable with production-grade fidelity**. The S277 live proof confirms that every field written by the writer pipeline is recoverable through the read path with no silent data loss or type corruption.

## Queryability Surface

### Supported Filters

| Filter | Type | Behavior | Proven |
|--------|------|----------|--------|
| `type` | Mandatory | Exact match (e.g., `paper_order`) | LAE-1 |
| `source` | Mandatory | Exact match (e.g., `binancef`) | LAE-1 |
| `symbol` | Mandatory | Exact match (e.g., `btcusdt`) | LAE-8 |
| `timeframe` | Mandatory | Exact match as UInt32 | LAE-8 |
| `side` | Optional | Exact match (`buy`, `sell`, `none`) | LAE-3 |
| `status` | Optional | Exact match (7 lifecycle values) | LAE-3 |
| `since` | Optional | `timestamp >= ?` (unix seconds → DateTime64) | LAE-4 |
| `until` | Optional | `timestamp <= ?` (unix seconds → DateTime64) | LAE-4 |
| `limit` | Required | 1–500, default 50 | LAE-1 |

### Ordering

Results are always returned **newest-first** (`ORDER BY timestamp DESC`). This matches the most common analytical query pattern: "show me the most recent executions."

### JSON Column Fidelity

| Column | Domain Type | Serialization | Round-Trip Proven |
|--------|-------------|---------------|-------------------|
| `risk` | `RiskInput` | JSON string with strategy_type, decision_severity | LAE-5 |
| `fills` | `[]FillRecord` | JSON array with price, quantity, fee, simulated, timestamp | LAE-6 |
| `parameters` | `map[string]string` | JSON object | LAE-7 |
| `metadata` | `map[string]string` | JSON object | LAE-7 |

All JSON columns use deterministic `json.Marshal` / `json.Unmarshal`. Empty maps serialize to `"{}"` and empty slices to `"[]"`. Nil maps serialize to `"{}"` (via `marshalJSON` fallback).

## Numeric Precision

| Field | Write Type | Storage Type | Read Type | Precision |
|-------|-----------|--------------|-----------|-----------|
| `quantity` | `float64` (from `parseFloat(string)`) | `Float64` | `float64` → `FormatFloat` | No loss for values with ≤15 significant digits |
| `filled_quantity` | `float64` | `Float64` | `float64` → `FormatFloat` | Same |
| `timeframe` | `uint32` | `UInt32` | `uint32` → `int` | Exact |

`FormatFloat` uses `strconv.FormatFloat(f, 'f', -1, 64)`, which preserves all significant digits without trailing zeros.

## Timestamp Semantics

| Aspect | Detail |
|--------|--------|
| Storage precision | DateTime64(3) — millisecond resolution |
| Filter precision | Unix seconds (since/until converted via `time.Unix(s, 0)`) |
| Ordering | By `timestamp` column, DESC |
| Partition key | `toYYYYMM(timestamp)` — monthly partitions |
| TTL | 90 days from `timestamp` |

**Finding**: The since/until filters operate at **second** granularity while the storage has **millisecond** precision. This means sub-second filtering is not possible via the current query API. This is acceptable for the current use case (execution events are not sub-second frequency).

## Causal Traceability

The execution analytical surface preserves the **four-stage causal chain**:

```
event_id       ← unique event identifier
correlation_id ← event-level correlation (from events.Metadata)
causation_id   ← event-level causation (from events.Metadata)
exec_correlation_id ← execution-level correlation (from ExecutionIntent)
exec_causation_id   ← execution-level causation (from ExecutionIntent)
```

This dual-layer tracing enables:
- **Event-level**: trace which event produced this row (`correlation_id`, `causation_id`).
- **Execution-level**: trace which risk assessment or strategy produced this intent (`exec_correlation_id`, `exec_causation_id`).

Both layers survive the full round-trip (proven by LAE-9).

## Performance Characteristics

| Aspect | Observation |
|--------|-------------|
| Index strategy | `ORDER BY (source, symbol, timeframe, type, timestamp)` — optimal for the mandatory filter combination |
| Partition pruning | Queries with `since`/`until` benefit from monthly partitioning |
| LowCardinality | `type`, `source`, `symbol`, `side`, `status` — efficient compression |
| JSON columns | Not indexed; full-text scan within row; acceptable for audit-depth queries |

## Gaps and Limitations

### Not Queryable Today

1. **Aggregation queries** (COUNT, AVG, GROUP BY) — the reader returns raw rows only.
2. **Full-text search on JSON columns** — risk, fills, parameters, metadata are opaque strings at the query level.
3. **Sub-second time filtering** — since/until use unix seconds despite DateTime64(3) storage.
4. **Cross-family joins** — no query joins execution with decision, strategy, or risk rows.

### Operational Limits

1. **Eventual consistency** — writer batches with `flush_interval=5s`; real-time queries may miss recent events by up to 5 seconds.
2. **No deduplication at storage level** — MergeTree does not enforce uniqueness on `event_id`. Duplicate protection relies on NATS JetStream deduplication and consumer idempotency.
3. **TTL-based retention** — rows older than 90 days are deleted by ClickHouse background merge. No archive path exists.
4. **Single-writer assumption** — no concurrent writer conflict resolution; operational safety depends on running a single writer instance.

## Recommendations

1. The current queryability surface is **sufficient for operational observability** of execution paper orders.
2. Aggregation and cross-family queries should wait for explicit functional requirements, not be built speculatively.
3. Sub-second filtering can be added later by accepting `int64` millisecond timestamps if needed.
4. The `ingested_at` column provides a secondary audit timestamp but is not queryable via the current API.
