# Operational List Queries and Retrieval Ergonomics

**Stage:** S454A
**Status:** Active
**Depends on:** S453A (Historical Read Model), S413 (Lifecycle List)

## Purpose

This document defines the operational list query surfaces introduced in S454A
to reduce friction in post-session audit, troubleshooting, and lifecycle
navigation. Before S454A, retrieving execution data required knowing the full
partition key (source + symbol + timeframe + type) in advance, making ad-hoc
operational queries impractical.

## Problem Statement

| Friction | Before S454A | After S454A |
|----------|-------------|-------------|
| "Show all rejected orders" | Impossible without scanning every partition key | `GET /analytical/execution/list?status=rejected` |
| "How many fills vs rejections today?" | Fetch all rows, count manually | `GET /analytical/execution/summary?since=...` |
| "List all active lifecycle entries" | NATS-only (S413), not reachable from HTTP | `GET /execution/lifecycle/list` |
| "Show fills for btcusdt regardless of timeframe" | Separate queries per timeframe | `GET /analytical/execution/list?symbol=btcusdt&status=filled` |

## New Endpoints

### 1. Execution List (ClickHouse)

```
GET /analytical/execution/list
```

Relaxed-filter execution list query. At least one filter is required, but none
are individually mandatory.

**Parameters (all optional, at least one required):**

| Param | Type | Description |
|-------|------|-------------|
| `type` | string | Execution type (e.g., `paper_order`, `venue_market_order`) |
| `source` | string | Source identifier |
| `symbol` | string | Trading pair |
| `timeframe` | int | Timeframe in seconds |
| `side` | string | Order side (`buy`, `sell`, `none`) |
| `status` | string | Lifecycle status (`submitted`, `filled`, `rejected`, etc.) |
| `since` | int64 | Unix timestamp, inclusive lower bound |
| `until` | int64 | Unix timestamp, inclusive upper bound |
| `limit` | int | Max rows (default 50, max 500) |

**Response:**
```json
{
  "entries": [
    {
      "type": "venue_market_order",
      "source": "derive",
      "symbol": "btcusdt",
      "timeframe": 60,
      "side": "buy",
      "quantity": "0.001",
      "filled_quantity": "0.001",
      "status": "filled",
      "fills": [...],
      "correlation_id": "...",
      "causation_id": "...",
      "final": true,
      "timestamp": "2026-03-24T15:30:00Z"
    }
  ],
  "source": "clickhouse",
  "meta": {"query_ms": 12, "row_count": 1}
}
```

### 2. Execution Summary (ClickHouse)

```
GET /analytical/execution/summary
```

Returns execution counts grouped by `(type, status)` with the most recent
timestamp per group.

**Parameters (all optional, at least one required):**

| Param | Type | Description |
|-------|------|-------------|
| `source` | string | Source identifier |
| `symbol` | string | Trading pair |
| `timeframe` | int | Timeframe in seconds |
| `since` | int64 | Unix timestamp, inclusive lower bound |
| `until` | int64 | Unix timestamp, inclusive upper bound |

**Response:**
```json
{
  "entries": [
    {"type": "paper_order", "status": "submitted", "count": 42, "latest_at": "2026-03-24T15:30:00Z"},
    {"type": "venue_market_order", "status": "filled", "count": 38, "latest_at": "2026-03-24T15:29:55Z"},
    {"type": "venue_rejection", "status": "rejected", "count": 4, "latest_at": "2026-03-24T15:28:00Z"}
  ],
  "source": "clickhouse",
  "meta": {"query_ms": 5, "row_count": 3}
}
```

### 3. Lifecycle List (NATS KV, HTTP-exposed)

```
GET /execution/lifecycle/list
```

Enumerates all tracked execution lifecycle entries across the three execution
KV buckets (paper_order, venue_market_order, venue_rejection). Previously
available only via NATS request/reply (S413), now exposed through the gateway
HTTP surface.

**Parameters:** None.

**Response:**
```json
{
  "entries": [
    {
      "key": "derive.btcusdt.60",
      "source": "derive",
      "symbol": "btcusdt",
      "timeframe": 60,
      "intent_status": "submitted",
      "intent_timestamp": "2026-03-24T15:30:00Z",
      "fill_status": "filled",
      "fill_timestamp": "2026-03-24T15:30:02Z",
      "rejection_status": "",
      "rejection_timestamp": null,
      "propagation": "filled"
    }
  ],
  "total": 1
}
```

## Design Decisions

1. **Relaxed filters, not no filters.** Every list/summary query requires at
   least one filter to prevent unbounded ClickHouse scans. The `WHERE 1=1`
   pattern with optional AND clauses replaces the rigid mandatory-filter
   approach.

2. **Reuse LifecycleHistoryEntry format.** The execution list response uses
   the same entry shape as the S453A lifecycle history, ensuring consistent
   ergonomics across the historical read model.

3. **Summary is GROUP BY, not aggregation framework.** A simple
   `GROUP BY type, status` with `count()` and `max(timestamp)` satisfies the
   operational overview need without building a reporting system.

4. **Lifecycle list stays KV-backed.** The `/execution/lifecycle/list`
   endpoint reads from NATS KV (latest-only), not ClickHouse. It answers
   "what's tracked right now?" while the ClickHouse list answers "what
   happened historically?"

## Alignment with S453A

The S453A lifecycle history query (`GET /analytical/execution/lifecycle`)
remains the primary tool for reconstructing a specific order's timeline
(requires source + symbol + timeframe). S454A complements it by enabling
discovery: "what partition keys have data?" and "what statuses exist across
the system?"

The typical operational flow is now:
1. `GET /execution/lifecycle/list` — see what's tracked
2. `GET /analytical/execution/list?status=rejected` — find problems
3. `GET /analytical/execution/lifecycle?source=...&symbol=...&timeframe=...` — drill into a specific lifecycle

## Limitations

- **No cross-join or subquery composition.** Each endpoint is a single-table
  query. Complex investigative queries (e.g., "find orders where the signal
  was RSI < 30 but execution was rejected") require multiple API calls and
  client-side correlation.
- **No cursor-based pagination.** Offset/limit only. For very large result
  sets, narrow the time range instead.
- **Summary does not break down by source/symbol/timeframe.** It groups only
  by (type, status). Segment-level breakdown requires filtering first.
- **Lifecycle list is KV-scoped.** It shows only the latest state per
  partition key, not historical entries. Use ClickHouse endpoints for
  historical data.
