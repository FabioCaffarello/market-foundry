# Live Fill Round-Trip: Ordering, Correlation, and Limitations

> S334 — Ordering guarantees, correlation invariants, and known limitations
> for the fill event round-trip through the live stack.

## Ordering Guarantees

### Within a single symbol

1. **NATS JetStream ordering**: Events within EXECUTION_FILL_EVENTS are ordered
   per subject (source.symbol.timeframe). JetStream delivers in sequence within
   a single durable consumer.

2. **Writer flush ordering**: The writer pipeline uses batch inserts with
   configurable flush intervals. Within a batch, row ordering follows NATS
   delivery order. Across batches, ClickHouse MergeTree may reorder during
   merges — but the `timestamp` column preserves causal ordering.

3. **Composite reader ordering**: Uses `ORDER BY timestamp DESC LIMIT 1` for
   single-chain lookup and `GROUP BY correlation_id ORDER BY max(timestamp) DESC`
   for batch queries. Both are deterministic and favor the latest event.

### Cross-symbol

- No ordering guarantee across symbols. Each symbol's events are independent.
- The composite reader enforces symbol isolation (S301): `WHERE symbol = ?`.

### Paper order vs venue fill

- Paper order is published first (derive → EXECUTION_EVENTS).
- Venue fill is published second (execute → EXECUTION_FILL_EVENTS).
- The venue fill always has a later timestamp.
- The composite reader's `ORDER BY timestamp DESC LIMIT 1` returns the venue
  fill when both exist for the same correlation_id.

## Correlation Invariants

| Invariant | Description | Enforced by |
|-----------|-------------|-------------|
| CI-1 | CorrelationID is immutable across the entire chain | Domain event constructors |
| CI-2 | CausationID links each event to its parent | Actor publish logic |
| CI-3 | Metadata.CorrelationID = ExecutionIntent.CorrelationID | Publisher + venue adapter actor |
| CI-4 | Composite reader queries by Metadata.CorrelationID | queryExecutionByCorrelation WHERE clause |
| CI-5 | ExecutionIntent.CorrelationID stored in exec_correlation_id column | mapVenueFillRow column 16 |

## Consistency Model

- **Eventual consistency**: The fill event may take up to writer flush interval
  (default: 5s) + ClickHouse merge delay to appear in the composite surface.
- **No strong consistency guarantee**: A composite chain query immediately after
  fill publication may return the paper order (not yet replaced by venue fill).
- **Read-your-writes**: Not guaranteed. The writer and reader are independent
  processes connecting to potentially different ClickHouse replicas (single-node
  in current deployment).

## Known Limitations

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|------------|
| L-S334-1 | Continuous live round-trip not observed for extended periods (>24h) | Medium | Structural + behavioral tests prove correctness; smoke script validates stack-level path |
| L-S334-2 | Testnet fills are always atomic (no partial fills observed) | Low | Domain model supports `partially_filled` status; no real data to test with |
| L-S334-3 | Commission uses cumQuote proxy (not real commission endpoint) | Low | Documented in S316; real fee reported but sourced from cumQuote |
| L-S334-4 | Single venue (Binance Futures testnet) | Out of scope | Design supports multi-venue via venue adapter abstraction |
| L-S334-5 | No WebSocket/async fill notification | Out of scope | REST polling in venue adapter; async path deferred |
| L-S334-6 | Writer flush latency visible in composite surface | Low | Acceptable for analytical reads; not a real-time feed |
| L-S334-7 | ClickHouse has no native dedup on executions table | Low | NATS JetStream dedup + writer idempotency prevent duplicates |

## Reconciliation Gates (FillProjectionActor)

The store binary's FillProjectionActor enforces reconciliation before
materializing fills to the KV bucket:

| Gate | Description | Action on failure |
|------|-------------|-------------------|
| RC-1 | Fill-to-intent correlation (orphan detection) | Log + count orphaned, skip materialization |
| RC-2 | Cumulative filled quantity ≤ requested quantity | Log + count overflow, skip materialization |
| RC-4 | Non-final intents skipped | Skip materialization |
| Monotonicity | Timestamp-based dedup via KV adapter | Skip stale/duplicate |

These gates apply only to the KV path. The writer pipeline does **not** enforce
reconciliation gates — it writes all fills to ClickHouse unconditionally for
audit completeness.

## What S334 Does NOT Cover

- Real-time fill streaming (WebSocket/SSE)
- Multi-venue fan-out
- Fill aggregation or P&L computation
- Dashboard or visualization layer
- Alerting on fill events
