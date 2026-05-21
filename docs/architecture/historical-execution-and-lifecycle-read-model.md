# Historical Execution and Lifecycle Read Model

Stage: S453A | Wave: Operational Memory Hardening | Date: 2026-03-24

## Purpose

This document describes the historical read model for execution lifecycle introduced in S453A. The model provides a unified, chronological view of all execution events (paper_order, venue_market_order, venue_rejection) for a given source/symbol/timeframe, backed by ClickHouse.

## Problem Statement

Before S453A, execution lifecycle read surfaces were split into two categories:

1. **Latest-only (KV)** -- Three NATS KV buckets store the most recent state per partition key. Queries include `ExecutionLatestQuery`, `ExecutionStatusQuery`, `LifecycleListQuery`. These answer "what is the current state?" but cannot answer "what happened over time?"

2. **Per-type historical (ClickHouse)** -- The existing `/analytical/execution/history` endpoint queries the `executions` table but requires `type` as a mandatory parameter. To reconstruct a lifecycle timeline, a consumer must make three separate queries (paper_order, venue_market_order, venue_rejection) and merge results client-side.

Neither surface answers the core operational question: **"Show me the chronological lifecycle of all execution events for this trading context."**

## Design

### New Surface: Lifecycle History Query

A single ClickHouse-backed query that returns all execution event types for a given source/symbol/timeframe in reverse-chronological order. The `type` column is no longer a mandatory WHERE filter -- it appears in SELECT for event identification.

```
LifecycleHistoryQuery {
    Source:    string    // required
    Symbol:    string    // required
    Timeframe: int      // required, positive
    Status:    string    // optional filter
    Side:      string    // optional filter
    Limit:     int      // 1-500, default 50
    Since:     int64    // optional, unix seconds
    Until:     int64    // optional, unix seconds
}

-->

LifecycleHistoryReply {
    Entries: []LifecycleHistoryEntry
    Source:  "clickhouse"
    Meta:    QueryMeta { QueryMs, RowCount }
}

LifecycleHistoryEntry {
    Type:           string              // "paper_order" | "venue_market_order" | "venue_rejection"
    Source:         string
    Symbol:         string
    Timeframe:      int
    Side:           string
    Quantity:       string
    FilledQuantity: string
    Status:         string
    Fills:          []FillRecord
    Metadata:       map[string]string   // includes rejection_code, rejection_reason for rejections
    CorrelationID:  string
    CausationID:    string
    Final:          bool
    Timestamp:      string              // RFC3339 format
}
```

### SQL Projection

The lifecycle history query differs from the existing execution history query in its mandatory WHERE clause:

| Query | Mandatory WHERE |
|-------|----------------|
| `BuildExecutionQuery` | `type = ? AND source = ? AND symbol = ? AND timeframe = ?` |
| `BuildLifecycleHistoryQuery` | `source = ? AND symbol = ? AND timeframe = ?` |

Both share the same SELECT columns, optional filters (side, status), time range, ORDER BY, and LIMIT semantics. The `executions` table is reused -- no schema migration is required.

### HTTP Endpoint

```
GET /analytical/execution/lifecycle?source=X&symbol=Y&timeframe=Z[&side=buy][&status=filled][&since=T][&until=T][&limit=N]
```

Follows the same pattern as `/analytical/execution/history` but without the required `type` parameter.

### Gateway Wiring

The lifecycle reader is created via `newAnalyticalLifecycleReader()` in the gateway composition root. It shares the same ClickHouse `ExecutionReader` adapter but satisfies the `LifecycleHistoryReader` interface, which only declares `QueryLifecycleHistory()`.

## Implementation Layers

| Layer | File | What it does |
|-------|------|-------------|
| Contract | `internal/application/analyticalclient/contracts.go` | `LifecycleHistoryQuery`, `LifecycleHistoryEntry`, `LifecycleHistoryReply` |
| Interface | `internal/application/analyticalclient/get_lifecycle_history.go` | `LifecycleHistoryReader` interface, `GetLifecycleHistoryUseCase` |
| Adapter | `internal/adapters/clickhouse/execution_reader.go` | `QueryLifecycleHistory()`, `BuildLifecycleHistoryQuery()` |
| Handler | `internal/interfaces/http/handlers/analytical.go` | `GetLifecycleHistory()` HTTP handler |
| Route | `internal/interfaces/http/routes/analytical.go` | `/analytical/execution/lifecycle` route |
| Composition | `cmd/gateway/compose.go` | `GetLifecycleHistory` wired to `AnalyticalFamilyDeps` |
| Reader factory | `cmd/gateway/analytical_reader.go` | `newAnalyticalLifecycleReader()` |

## Relationship to Existing Surfaces

| Surface | Scope | Backing | Use case |
|---------|-------|---------|----------|
| `ExecutionLatestQuery` | Single type, latest | KV | "What is the current paper_order?" |
| `ExecutionStatusQuery` | All types, latest | KV (composite) | "What is the current composite status?" |
| `LifecycleListQuery` | All keys, latest | KV (enumeration) | "What partition keys exist and what are their states?" |
| `/analytical/execution/history` | Single type, historical | ClickHouse | "Show me all paper_orders for this context" |
| `/analytical/execution/lifecycle` (S453A) | All types, historical | ClickHouse | "Show me the full lifecycle timeline for this context" |

The new surface complements rather than replaces existing ones. KV surfaces remain authoritative for real-time operational state. The lifecycle history adds the temporal dimension.

## Consistency Model

- **Eventually consistent** -- ClickHouse receives events via the writer pipeline, which consumes from NATS JetStream. A recently published event may not yet be visible in lifecycle history.
- **Append-only** -- The `executions` table uses MergeTree with no deduplication engine. Duplicate events (e.g., from writer restart) will appear as separate rows. Consumers should use `exec_correlation_id` for grouping.
- **90-day TTL** -- Rows older than 90 days are automatically purged by ClickHouse.

## Limitations

1. **No real-time guarantee** -- ClickHouse write path has pipeline latency (typically <5s). For real-time status, use KV surfaces.
2. **No type-level ordering guarantee** -- Within the same timestamp, ordering between paper_order and venue_market_order events is not deterministic.
3. **No aggregation** -- This is a raw event timeline, not a summary. Rejection rates or fill latency statistics are not computed.
4. **Metadata-embedded rejection detail** -- Rejection code/reason are in the metadata JSON column, not top-level columns. Filtering by rejection code requires client-side processing.
5. **No cross-partition queries** -- Each query is scoped to a single source/symbol/timeframe. Listing across all symbols requires `LifecycleListQuery` first.
