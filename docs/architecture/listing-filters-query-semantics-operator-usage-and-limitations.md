# Listing Filters, Query Semantics, Operator Usage, and Limitations

**Stage:** S454A
**Status:** Active
**Companion:** [Operational List Queries and Retrieval Ergonomics](operational-list-queries-and-retrieval-ergonomics.md)

## Filter Semantics

### Execution List (`/analytical/execution/list`)

All filters are optional. At least one must be provided. Filters combine with
AND semantics (all specified conditions must match).

| Filter | Column | Behavior |
|--------|--------|----------|
| `type` | `type` | Exact match. Values: `paper_order`, `venue_market_order`, `venue_rejection` |
| `source` | `source` | Exact match. E.g., `derive` |
| `symbol` | `symbol` | Exact match. E.g., `btcusdt` |
| `timeframe` | `timeframe` | Exact match. Integer seconds. E.g., `60`, `300` |
| `side` | `side` | Exact match. Values: `buy`, `sell`, `none` |
| `status` | `status` | Exact match. Values: `submitted`, `sent`, `accepted`, `filled`, `partially_filled`, `rejected`, `cancelled` |
| `since` | `timestamp` | Inclusive lower bound. Unix seconds. |
| `until` | `timestamp` | Inclusive upper bound. Unix seconds. |
| `limit` | N/A | Max rows returned. Default: 50. Range: [1, 500]. |

**Guard:** If no filter or time range is provided, the query is rejected with
HTTP 400. This prevents unbounded full-table scans.

### Execution Summary (`/analytical/execution/summary`)

Returns counts grouped by `(type, status)`. Filters scope the aggregation.

| Filter | Column | Behavior |
|--------|--------|----------|
| `source` | `source` | Exact match |
| `symbol` | `symbol` | Exact match |
| `timeframe` | `timeframe` | Exact match |
| `since` | `timestamp` | Inclusive lower bound |
| `until` | `timestamp` | Inclusive upper bound |

**Guard:** Same as list — at least one filter required.

### Lifecycle List (`/execution/lifecycle/list`)

No filters. Returns all partition keys currently tracked in the three
execution KV buckets. This is a discovery endpoint, not a search endpoint.

## Ordering

- **List and Lifecycle History:** Results are always ordered `timestamp DESC`
  (newest first).
- **Summary:** Results are ordered `count DESC` (most frequent status first).
- **Lifecycle List (KV):** Unordered (iteration order of KV keys).

## Time Range Semantics

- `since` and `until` are unix seconds (not milliseconds).
- Both bounds are **inclusive**: `timestamp >= since AND timestamp <= until`.
- When only `since` is set, all events from that point forward are returned.
- When only `until` is set, all events up to that point are returned.
- When both are set, `since` must not exceed `until` (HTTP 400 otherwise).
- When neither is set, no time filter is applied (other filters still required).

## Operator Usage Patterns

### Post-Session Audit

```bash
# 1. What happened in the last session?
curl "$GW/analytical/execution/list?source=derive&since=1711296000&until=1711299600"

# 2. Were there any rejections?
curl "$GW/analytical/execution/list?source=derive&status=rejected&since=1711296000"

# 3. Overview: how many of each status?
curl "$GW/analytical/execution/summary?source=derive&since=1711296000"
```

### Troubleshooting a Symbol

```bash
# 1. What's the current state?
curl "$GW/execution/lifecycle/list"

# 2. All events for this symbol, any timeframe
curl "$GW/analytical/execution/list?symbol=btcusdt&limit=100"

# 3. Drill into specific lifecycle
curl "$GW/analytical/execution/lifecycle?source=derive&symbol=btcusdt&timeframe=60"
```

### System Health Check

```bash
# Quick count of all execution statuses
curl "$GW/analytical/execution/summary?since=$(date -d '1 hour ago' +%s)"
```

## Limitations

### No Wildcard or Pattern Matching
Filters are exact match only. No LIKE, regex, or IN-list support. To query
multiple symbols, make separate requests.

### No Cursor Pagination
Only offset/limit. For large datasets, narrow the time range. The 500-row
limit prevents accidental large result sets.

### No Cross-Table Queries
Each endpoint queries a single ClickHouse table (`executions`). To correlate
executions with signals, decisions, or risk assessments, use the composite
chain endpoint (`/analytical/composite/chain`) or make multiple requests.

### KV Lifecycle List Is Latest-Only
The `/execution/lifecycle/list` endpoint shows only the most recent state per
partition key. Historical entries for the same key are only available via the
ClickHouse lifecycle history endpoint.

### Summary Does Not Support Type/Side Filters
The summary groups by `(type, status)` automatically. You cannot filter the
summary to only `venue_market_order` types — the grouping shows all types
present in the data. Use the list endpoint with a type filter for
type-specific investigation.

### No Real-Time Streaming
All endpoints are request/response. There is no WebSocket or SSE push for new
executions. For real-time awareness, use the existing NATS subscription model.

## What Follows (Out of Scope)

- **Explainability surface (S455A):** Building on these list queries to provide
  "why did this order get rejected?" narratives.
- **BI/Reporting:** Dashboards, time-series aggregation, or statistical
  analysis are explicitly out of scope.
- **Query DSL:** No generic query language. Filters remain fixed and explicit.
