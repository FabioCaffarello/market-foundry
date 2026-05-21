# Analytical Storage Strategy

> Architectural decision record for the historical/analytical storage layer of market-foundry.

## Decision

**ClickHouse is the preferred analytical storage backend, but it does not enter the runtime now.**

The adoption is deferred until a concrete trigger is met. NATS KV remains the sole projection target for the current phase. When triggered, ClickHouse enters as a secondary projection backend behind the existing `CandleKVStore` interface — not as a replacement.

## Status

**Decided, deferred.** The architecture is locked. The implementation is gated by trigger conditions.

## Context

Market-foundry's store service materializes finalized candles into two NATS KV buckets:
- `CANDLE_LATEST` — latest candle per source/symbol/timeframe (unbounded retention, 64MB)
- `CANDLE_HISTORY` — time-indexed candles with 24h TTL (256MB)

This model was hardened in S21 with monotonicity guards, domain validation, idempotent key design, and explicit replay semantics.

The question is: when and how should the system acquire unbounded historical retention and analytical query capability?

## Rationale

### Why ClickHouse (when triggered)

1. **Data model alignment.** OHLCV candles are append-only, immutable, time-indexed, naturally ordered by (source, symbol, timeframe, open_time). This maps directly to a MergeTree table with `ORDER BY (source, symbol, timeframe, open_time)`.

2. **Idempotency alignment.** ReplacingMergeTree provides server-side dedup on the natural key — the same semantics as the current history KV key design. Replay safety carries over.

3. **Aggregation.** ClickHouse can compute server-side aggregations (hourly/daily OHLCV rollups, volume profiles) that are impossible with NATS KV prefix scans.

4. **Retention.** Months to years of candle history with partitioning by month. No TTL pressure.

5. **Boundary preservation.** ClickHouse enters behind the existing port interface (`CandleKVStore` or a new `CandleAnalyticalStore`). The gateway never talks to ClickHouse directly. The store remains the sole read-model authority.

### Why not now

1. **Scale doesn't justify it.** 2 symbols × 2 timeframes × 1 candle/min = 8 writes/min. NATS KV handles this with zero stress. Adding ClickHouse infrastructure for this volume is negative ROI.

2. **No analytics consumers.** No dashboard, no strategy engine, no reporting layer reads candle history today. Building infrastructure for consumers that don't exist is premature.

3. **Operational cost.** ClickHouse is a new container, a new Go dependency (`clickhouse-go`), new schema management, new monitoring. Each adds surface area.

4. **NATS KV is working.** The projection pipeline is clean, tested, hardened. The 24h history window serves the current operational query needs.

### Why not TimescaleDB

TimescaleDB is a reasonable alternative but offers no structural advantage for this workload:
- ACID guarantees are unnecessary (candle projections are idempotent, append-only)
- Row-oriented storage is slower for analytical scans at scale
- Continuous aggregates add operational complexity without clear need
- PostgreSQL operational weight (vacuuming, connection pooling, pg_wal) for what is fundamentally an OLAP workload

See `clickhouse-vs-timescale-vs-current-store.md` for detailed comparison.

## Trigger Conditions

ClickHouse adoption is triggered when **any one** of these conditions is met:

| Trigger | Rationale | How to detect |
|---------|-----------|---------------|
| **>10 active symbols** | `Keys()` scan degrades at O(all_keys). Prefix scan over ~14K+ keys/day becomes latency-visible. | configctl binding count |
| **Retention need >24h** | External consumer (dashboard, strategy, reporting) needs historical data beyond the NATS KV TTL. | Product/stakeholder request |
| **Aggregation query** | Consumer needs server-side aggregation (hourly OHLCV, volume profile, cross-timeframe analysis). | API requirement |
| **Event stream exhaustion** | If 72h EVIDENCE_EVENTS stream can't fully rebuild CANDLE_HISTORY (e.g., after extended outage), an archive becomes necessary. | Operational incident |

## What Stays in NATS KV

Regardless of ClickHouse adoption:

| Bucket | Stays | Reason |
|--------|-------|--------|
| `CANDLE_LATEST` | **Always** | Sub-millisecond point lookup for "current candle". ClickHouse is not optimized for this. |
| `CANDLE_HISTORY` | **As hot cache** | Recent 24h history for operational queries. ClickHouse for longer ranges. |

NATS KV is the hot tier. ClickHouse is the cold/analytical tier. They coexist, not compete.

## What Goes to ClickHouse (when adopted)

| Data | Purpose |
|------|---------|
| Finalized candles (all timeframes) | Unbounded historical archive |
| Aggregated views (hourly, daily) | Materialized views for analytical queries |
| Volume profiles (future) | New evidence types that are purely analytical |

## Boundary Rules

1. **Gateway never talks to ClickHouse.** All queries go through the store's NATS request/reply interface.
2. **Store owns the write path.** The projection actor writes to both NATS KV and ClickHouse. No other service writes to ClickHouse.
3. **ClickHouse is a secondary projection target.** It receives the same candle events as NATS KV, through the same projection actor, after the same validation gates.
4. **Query routing is store-internal.** The query responder decides whether to read from NATS KV (recent, fast) or ClickHouse (historical, analytical) based on the query parameters.
5. **No dual-write without design.** The projection actor writes to KV first, then ClickHouse. If ClickHouse is unavailable, KV writes succeed independently. ClickHouse availability does not gate the hot path.

## Risks

| Risk | Mitigation |
|------|-----------|
| Premature adoption | Trigger conditions are explicit and measurable. No adoption without evidence. |
| Dual-write complexity | ClickHouse write is fire-and-forget from the projection actor. KV write is the primary path. ClickHouse failure is logged, not fatal. |
| Schema drift | ClickHouse schema is derived from `EvidenceCandle` domain type. Schema changes require migration — same as any DB. |
| Operational overhead | Single-node ClickHouse in Docker. No cluster management. Same deployment model as NATS. |
| Query routing complexity | Binary decision: if `since` is within 24h → NATS KV. Otherwise → ClickHouse. No complex routing. |
