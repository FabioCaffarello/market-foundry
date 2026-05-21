# Execution Lifecycle History: Sources, Projection Semantics, and Limitations

Stage: S453A | Wave: Operational Memory Hardening | Date: 2026-03-24

## Purpose

This document catalogs the data sources, projection semantics, and known limitations of the historical execution lifecycle read model introduced in S453A.

## Data Sources

The lifecycle history read model queries a single ClickHouse table (`executions`) that is populated by three writer pipelines:

| Pipeline | NATS Stream | Event Type | Writer Mapper |
|----------|-------------|-----------|---------------|
| Paper order | `EXECUTION_EVENTS` | `PaperOrderSubmittedEvent` | `mapExecutionRow()` |
| Venue fill | `EXECUTION_FILL_EVENTS` | `VenueOrderFilledEvent` | `mapVenueFillRow()` |
| Venue rejection | `EXECUTION_REJECTION_EVENTS` | `VenueOrderRejectedEvent` | `mapVenueRejectionRow()` |

All three pipelines write to the same `executions` table with the same 20-column schema. The `type` column distinguishes the event source.

### Column Schema (executions)

| Column | Type | Source |
|--------|------|--------|
| event_id | String | Event envelope |
| occurred_at | DateTime64(3) | Event envelope |
| correlation_id | String | Event envelope (NATS causality) |
| causation_id | String | Event envelope (NATS causality) |
| type | LowCardinality(String) | Pipeline-specific (paper_order / venue_market_order / venue_rejection) |
| source | LowCardinality(String) | ExecutionIntent.Source |
| symbol | LowCardinality(String) | ExecutionIntent.Symbol |
| timeframe | UInt32 | ExecutionIntent.Timeframe |
| side | LowCardinality(String) | ExecutionIntent.Side |
| quantity | Float64 | ExecutionIntent.Quantity |
| filled_quantity | Float64 | ExecutionIntent.FilledQuantity |
| status | LowCardinality(String) | ExecutionIntent.Status |
| risk | String (JSON) | ExecutionIntent.Risk |
| fills | String (JSON) | ExecutionIntent.Fills |
| parameters | String (JSON) | ExecutionIntent.Parameters |
| metadata | String (JSON) | ExecutionIntent.Metadata |
| exec_correlation_id | String | ExecutionIntent.CorrelationID |
| exec_causation_id | String | ExecutionIntent.CausationID |
| final | Bool | ExecutionIntent.Final |
| timestamp | DateTime64(3) | ExecutionIntent.Timestamp |
| ingested_at | DateTime64(3) | Writer insertion time |

### Table Properties

- Engine: MergeTree
- Partition: `toYYYYMM(timestamp)`
- Order: `(source, symbol, timeframe, type, timestamp)`
- TTL: 90 days from `timestamp`

## Projection Semantics

### What enters the historical read model

Every event that flows through the three writer pipelines is appended to the `executions` table. This includes:

1. **Paper orders** -- Every `PaperOrderSubmittedEvent` with status=submitted, side, quantity, fills (simulated), risk metadata.
2. **Venue fills** -- Every `VenueOrderFilledEvent` with status=filled/partially_filled/accepted, actual fills, VenueOrderID in metadata.
3. **Venue rejections** -- Every `VenueOrderRejectedEvent` with status=rejected, rejection_code/rejection_reason/venue_details embedded in metadata JSON.

### What stays outside the historical read model

| Data | Why excluded | Where it lives |
|------|-------------|----------------|
| Execution control gate state | Not an execution event; operational toggle | NATS KV (`EXECUTION_CONTROL`) |
| Activation surface dimensions | Runtime adapter/credential state, not lifecycle | NATS KV (`EXECUTION_ACTIVATION_DIMENSIONS`) |
| Intermediate state transitions (submitted->sent->accepted) | Events only capture end-states; no intermediate transition events exist | Not persisted anywhere |
| KV latest-only state | Superseded by each new event; not historical | NATS KV (3 buckets) |
| Fill latency metrics | Not computed; raw timestamps available for client-side derivation | Computable from entries |
| Rejection rate aggregation | Not precomputed; derivable from filtered queries | Computable from entries |

### Lifecycle reconstruction semantics

Given a `GET /analytical/execution/lifecycle?source=derive&symbol=btcusdt&timeframe=60&limit=100`, the response might contain:

```json
{
  "entries": [
    {"type": "venue_market_order", "status": "filled", "timestamp": "2026-03-24T15:30:00Z", ...},
    {"type": "paper_order", "status": "submitted", "timestamp": "2026-03-24T15:29:58Z", ...},
    {"type": "venue_rejection", "status": "rejected", "timestamp": "2026-03-24T14:15:02Z", ...},
    {"type": "paper_order", "status": "submitted", "timestamp": "2026-03-24T14:15:00Z", ...}
  ]
}
```

This shows two lifecycle cycles: a rejection cycle (14:15) and a fill cycle (15:29-15:30). The consumer can group by `exec_correlation_id` for precise causation tracking.

### Correlation semantics

The `exec_correlation_id` field is the primary grouping key for linking paper_order intent to its venue outcome. All events in the same lifecycle cycle share the same correlation ID.

The `correlation_id` and `causation_id` (event envelope fields) track NATS-level causality and are distinct from execution-domain correlation.

## Query Performance Characteristics

- The `executions` table is ordered by `(source, symbol, timeframe, type, timestamp)`.
- The lifecycle history query filters by `(source, symbol, timeframe)` -- matching the first three ordering columns.
- The `type` column is NOT in the WHERE clause, so ClickHouse scans all types within the partition key range.
- For typical trading volumes (10-100 events/day/symbol), query performance is sub-millisecond even without time range filters.
- The `LIMIT` clause ensures bounded response size regardless of data volume.

## Limitations

### L-1: No intermediate state transitions

The system publishes events for end-states only. A paper_order event has status=submitted; the next event for the same context might be venue_market_order with status=filled. The intermediate transitions (sent, accepted) are not persisted as separate events and cannot be reconstructed from the historical read model.

**Impact**: LOW. Intermediate states are transient (<1s) and not operationally significant for post-hoc analysis.

### L-2: Duplicate rows on writer restart

The writer pipeline uses JetStream deduplication at the consumer level, but ClickHouse does not deduplicate on insert. A writer restart during batch processing can result in duplicate rows.

**Impact**: LOW. Duplicates are identifiable by `event_id` and can be filtered client-side. The lifecycle trajectory is not affected because timestamps remain consistent.

### L-3: No real-time guarantee

ClickHouse write path has pipeline latency. Events are batched by the writer and flushed periodically. Typical latency is <5 seconds.

**Impact**: LOW. For real-time state, KV surfaces are authoritative. The historical model is for post-hoc review.

### L-4: 90-day retention

ClickHouse TTL purges rows older than 90 days. Historical reconstruction beyond this window is not possible.

**Impact**: LOW at current scale. If longer retention is needed, the TTL can be extended in the migration without schema changes.

### L-5: Metadata-embedded rejection details

Rejection code, reason, and venue details are stored inside the `metadata` JSON column, not as top-level ClickHouse columns. This means:
- Cannot use `status = 'rejected' AND rejection_code = 'insufficient_margin'` in a single SQL query
- Must filter rejection details client-side after fetching results

**Impact**: MEDIUM for rejection-focused analysis. Acceptable at current scale; a future stage could add dedicated ClickHouse columns for rejection fields if query volume warrants it.

### L-6: No cross-partition query

Each lifecycle history query is scoped to a single source/symbol/timeframe. There is no "show all lifecycle events across all symbols" query.

**Impact**: LOW. The `LifecycleListQuery` (S413, KV-backed) provides key enumeration. Consumers can iterate over known keys to build a cross-partition view.

## Trade-offs

| Decision | Alternative considered | Why this choice |
|----------|----------------------|-----------------|
| Reuse `executions` table | Create separate `lifecycle_events` table | Avoids schema migration; data already exists; no write-path changes needed |
| Query builder without type filter | UNION ALL across three separate queries | Simpler; same performance; ClickHouse handles the scan efficiently |
| LifecycleHistoryEntry with string timestamp | time.Time timestamp | RFC3339 string avoids JSON serialization timezone ambiguity; consistent with audit trail expectations |
| Return raw events, not aggregated | Pre-compute lifecycle summaries | Keeps the model minimal; aggregation is a separate concern for S454A+ |
