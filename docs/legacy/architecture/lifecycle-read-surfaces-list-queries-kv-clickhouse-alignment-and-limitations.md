# Lifecycle Read Surfaces: List Queries, KV, ClickHouse Alignment and Limitations

Stage: S413 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Purpose

This document describes the alignment between NATS KV and ClickHouse read surfaces for the execution lifecycle, the pragmatic treatment of list queries, and the limitations that remain after S413 consolidation.

## Storage Model: KV vs ClickHouse

| Dimension | NATS KV (Latest) | ClickHouse (History) |
|-----------|-------------------|---------------------|
| Freshness | Near real-time (event projection) | Event-driven (batch writer) |
| Cardinality | 1 entry per partition key | Unbounded (append-only) |
| Query scope | Exact key or enumerate all keys | Filtered by type/source/symbol/timeframe/time-range |
| Retention | Permanent (latest only, overwritten) | Stream retention (72h for events) + ClickHouse retention |
| Rejection audit | S407 metadata embedding in intent | S411 metadata embedding in row |
| List capability | S413 Keys() enumeration | Native SQL with filters |

## Alignment Properties

### 1. Rejection Metadata Consistency

Both KV and ClickHouse use the same metadata embedding pattern for rejection audit fields:
- `rejection_code` key in intent.Metadata
- `rejection_reason` key in intent.Metadata
- `venue_detail.*` prefixed keys for venue-specific details

This alignment was established in S407 (KV) and S411 (ClickHouse) and ensures rejection data is queryable from either surface with the same extraction logic.

### 2. Status Field Consistency

Both surfaces use the same `execution.Status` type (`submitted`, `sent`, `accepted`, `filled`, `partially_filled`, `rejected`, `cancelled`). ClickHouse stores it as a string column; KV stores it within the JSON-serialized ExecutionIntent.

### 3. Partition Key Consistency

KV uses `{source}.{symbol}.{timeframe}` as the key. ClickHouse uses `(source, symbol, timeframe)` in its ORDER BY/WHERE clause. The S413 lifecycle list query parses KV keys back into these components for the response.

## List Query Surfaces

### KV: Lifecycle List (S413)

- Route: `execution.query.lifecycle.list`
- Returns: All tracked partition keys with per-surface status and effective propagation
- Cardinality: Bounded by active source/symbol/timeframe combinations (typically 10-50 in production)
- Latency: Proportional to key count (one KV read per key per bucket, 3 buckets)
- Use case: "Show me all active lifecycle entries and their state"

### ClickHouse: Execution History

- Existing query: `QueryExecutionHistory(ctx, type, source, symbol, timeframe, side, status, since, until, limit)`
- Supports `status` filter: can query `status=rejected`, `status=filled`, etc.
- Supports time range: `since`/`until` in unix seconds
- Supports limit: default 50, max 500
- Use case: "Show me all rejected orders for BTCUSDT in the last hour"

### ClickHouse: Composite Chain

- Existing query: `QueryChainByCorrelationID` and `QueryChainsBatch`
- Reconstructs full 5-stage causal chain per correlation ID
- Use case: "Trace the full pipeline from signal to execution for this specific order"

## Pragmatic Treatment of Gaps

### Gap 1: KV is Latest-Only

KV stores only the most recent state per partition key. If a key transitions submitted -> filled -> rejected (across different orders), only the rejection is visible in KV. The history is in ClickHouse.

**Treatment:** This is by design. KV provides operational "current state" view. ClickHouse provides history. The S413 lifecycle list makes the current state enumerable without foreknowledge of keys.

### Gap 2: No Cross-Surface Join

There is no single query that joins KV current state with ClickHouse history for a given key. The lifecycle list gives current state; ClickHouse history gives the timeline.

**Treatment:** Out of scope. A cross-surface join would require the gateway to orchestrate KV + ClickHouse reads per request. This is analytics territory, not operational queryability.

### Gap 3: KV Keys May Include Stale Entries

A partition key that was written once will persist in KV indefinitely (no TTL on execution KV buckets). If a source/symbol stops trading, its last state remains visible.

**Treatment:** Accepted limitation. The lifecycle list includes timestamps so operators can identify stale entries. Adding TTL to execution KV buckets would risk losing the latest known state.

### Gap 4: No Pagination on Lifecycle List

The lifecycle list returns all entries. For production deployments with many symbols, this could grow.

**Treatment:** Accepted for now. Execution KV cardinality is bounded by the product of sources x symbols x timeframes, which is small (< 100 in current configurations). Pagination can be added if cardinality grows beyond 500.

### Gap 5: ClickHouse Rejection Queryability

Rejection-specific fields (code, reason) are embedded in the `metadata` JSON column, not in dedicated columns. Querying by rejection code requires JSON path extraction in ClickHouse SQL.

**Treatment:** Accepted. Adding dedicated columns would require a schema migration. The current embedding pattern provides queryability via `JSONExtractString(metadata, 'rejection_code')` which is sufficient for operational use.

## Invariants

1. **Propagation consistency** -- `DeriveEffectivePropagation()` is the single source of truth for effective lifecycle status. Used by both the composite status query and the lifecycle list query.
2. **Metadata embedding consistency** -- Rejection audit metadata uses the same key conventions in KV and ClickHouse.
3. **Key format consistency** -- `{source}.{symbol}.{timeframe}` is the canonical partition key across all execution KV buckets.
4. **Status type consistency** -- `execution.Status` constants are the only valid values in both KV and ClickHouse.

## Limitations Summary

| Limitation | Severity | Mitigation |
|-----------|----------|------------|
| KV latest-only (no history) | Low | ClickHouse provides history |
| No KV-to-ClickHouse cross-join | Low | Separate queries, same key format |
| Stale KV entries persist | Low | Timestamps visible in lifecycle list |
| No pagination on lifecycle list | Low | Bounded cardinality (< 100) |
| Rejection fields in JSON column | Low | JSON extraction available in ClickHouse |
| Lifecycle list is eventually consistent | Low | KV projections lag by < 1s typically |
