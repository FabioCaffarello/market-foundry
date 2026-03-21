# Composite Read Model — Semantics, Ordering, and Limitations

**Stage:** S296
**Status:** Delivered

---

## 1. Semantic Model

### 1.1 Unit of Composition

The atomic unit is the **CompositeExecutionChain** — one `correlation_id` resolved across 5 ClickHouse tables. Each chain represents a single causal thread from market observation (signal) through terminal action (execution).

### 1.2 Stage Presence

Each of the 5 stages is **optional**. A chain can be:

| Pattern | Stages Present | Meaning |
|---------|---------------|---------|
| Full chain | signal + decision + strategy + risk + execution | Complete flow, execution submitted |
| Risk-rejected | signal + decision + strategy + risk | Risk gate blocked execution |
| In-flight | signal + decision | Decision evaluated, strategy not yet resolved |
| Signal-only | signal | Signal emitted, no decision yet (or decision filtered) |
| Orphaned execution | execution only | Execution exists but causal chain not found (data gap) |

The `chain_complete` flag is `true` only when all 5 stages are present. The `missing_stages` array identifies which stages are absent.

### 1.3 Cardinality

The current model assumes **1:1 cardinality** at each stage within a correlation_id:
- One signal per correlation_id
- One decision per correlation_id
- One strategy per correlation_id
- One risk assessment per correlation_id
- One execution per correlation_id

When multiple events share a correlation_id within the same table (e.g., a retry or update), the composite reader returns the **most recent** event (by `timestamp DESC LIMIT 1`).

**Known limitation:** If a future architecture introduces fan-out (one signal producing multiple decisions under the same correlation_id), the 1:1 model would need extension. This is not the case today across any of the 3 proven slices.

## 2. Ordering Guarantees

### 2.1 Temporal Ordering Within a Chain

Events within a single chain are ordered by `occurred_at` (event envelope timestamp):

```
signal.occurred_at < decision.occurred_at < strategy.occurred_at < risk.occurred_at < execution.occurred_at
```

This ordering is a **strong invariant** because:
- Each stage is triggered by the previous stage's published event
- The actor layer sets `occurred_at` at publish time
- ClickHouse stores the value as written (no server-side timestamp override)

**Caveat:** Clock skew between actor goroutines is negligible (nanosecond-level within a single process). Multi-process deployment could introduce microsecond-level skew, but the causal ordering is still enforced by the CausationID DAG.

### 2.2 Ordering Across Chains (Batch Mode)

Batch results are ordered by **execution timestamp DESC** (most recent execution first). This ordering is:
- Deterministic for a given dataset
- Consistent across repeated queries (same data = same order)
- **Not** globally ordered across different symbols or timeframes

### 2.3 Consistency Model

The composite read model operates under **eventual consistency**:

1. **Write path:** Events are written to ClickHouse via the writer pipeline (NATS consumer → batch insert). Batch intervals introduce a write delay (configurable, typically 1-5 seconds).
2. **Read path:** Each of the 5 queries reads independently. There is no transactional guarantee that all 5 stages are visible simultaneously.

**Practical impact:** A chain might appear as 4/5 stages for a brief window after the execution event is written but before the signal or decision batch flushes. The `chain_complete=false` flag makes this visible.

**Mitigation:** For operational use, an incomplete chain with `missing_stages=["signal"]` and a present execution is a strong indicator of write lag, not data loss. If the chain remains incomplete after the batch interval, it indicates a genuine gap.

## 3. Consistency Between Causal Metadata and Domain Data

### 3.1 Correlation/Causation IDs

- `correlation_id` is **immutable** from signal through execution (validated in S295).
- `causation_id` forms a DAG: each event's causation_id equals its parent event's `event_id`.
- Signals have an empty `causation_id` (they are the root of the chain — S295 gap G2, by design).

### 3.2 Domain Fields

Domain fields (outcome, severity, direction, disposition, etc.) are stored as they were at write time. The composite model **does not** reconcile domain fields across stages — it presents them as-is.

Example: if a decision has `severity=high` but the risk assessment was evaluated against a different severity (due to a concurrent config change), the composite model shows both values without reconciliation. The operator must interpret the causal context.

## 4. Query Performance Characteristics

### 4.1 Single Chain (by correlation_id)

- 5 independent queries, each scanning a single table with `WHERE correlation_id = ?`
- Expected latency: < 10ms total for a single chain (5 queries × < 2ms each)
- `correlation_id` is not in the MergeTree ORDER BY key, so ClickHouse performs a data skip index scan. For tables with < 1M rows, this is sub-millisecond.

### 4.2 Batch (by symbol/timeframe/time-range)

- Initial query: GROUP BY + ORDER BY on executions table (indexed by source, symbol, timeframe)
- Enrichment: N × 5 queries where N = number of chains (default limit: 20)
- Expected latency: < 200ms for 20 chains (1 index query + 100 point lookups)

### 4.3 Scaling Considerations

| Dataset Size | Single Chain | Batch (20 chains) | Notes |
|-------------|-------------|-------------------|-------|
| < 100K rows/table | < 5ms | < 100ms | Current scale |
| 100K-1M rows/table | < 10ms | < 200ms | Adequate for months of data |
| > 1M rows/table | < 50ms | < 1s | May need correlation_id index |

If performance degrades at scale, the recommended optimization is a ClickHouse **secondary index** on `correlation_id`:

```sql
ALTER TABLE signals ADD INDEX idx_correlation_id (correlation_id) TYPE bloom_filter GRANULARITY 4;
```

This is deferred until needed — premature optimization avoided per project principles.

## 5. Limitations

### 5.1 No Real-Time Streaming

The composite model is a **pull-based read model**. It does not subscribe to live events or maintain a materialized state. Each query reads the current state of the 5 tables at query time.

### 5.2 No Cross-Symbol Composition

A chain is scoped to a single `correlation_id`. The model does not compose across symbols, timeframes, or families. Cross-symbol analysis requires multiple queries.

### 5.3 No Aggregation

The composite model returns individual chains, not aggregates. Questions like "how many executions were blocked in period T?" (Q6) require separate aggregation queries, delivered in S298.

### 5.4 Evidence-to-Signal Gap

The causal chain starts at signal. There is no link from signal back to the evidence (candle) that produced it. This is an architectural boundary documented in S295 (gap G2).

### 5.5 Fan-Out Not Modeled

The current model assumes 1:1 cardinality per correlation_id per table. If fan-out is introduced in the future (e.g., one signal producing multiple decisions), the model would need extension to support 1:N stages.

### 5.6 No Write-Side Changes

S296 is entirely read-side. It does not modify the write path, event schema, or actor message flow. All 5 tables are queried as-is.
