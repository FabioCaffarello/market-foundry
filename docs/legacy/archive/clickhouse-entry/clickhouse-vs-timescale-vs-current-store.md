# ClickHouse vs TimescaleDB vs Current Store (NATS KV)

> Comparative analysis for the analytical/historical storage layer of market-foundry.

## Evaluation Context

| Dimension | Current State |
|-----------|---------------|
| Symbols | 2 (btcusdt, ethusdt) |
| Timeframes | 2 (60s, 300s) |
| Candle rate | ~8/min (4 combinations × 2 events/min) |
| History retention | 24h (NATS KV TTL) |
| Event stream retention | 72h (EVIDENCE_EVENTS) |
| Query patterns | Latest (single-key GET), History (prefix scan + range, max 100) |
| Consumers of history | HTTP API via gateway only |
| Analytics consumers | None yet |

## Candidate Comparison

### 1. Current Store: NATS KV

**What it is:** JetStream KeyValue buckets embedded in the existing NATS infrastructure.

| Aspect | Assessment |
|--------|-----------|
| Write throughput | Adequate. 8 candles/min is negligible for KV. |
| Read latency | Sub-millisecond for latest (single key GET). Adequate for history (prefix scan). |
| Retention | 24h history (TTL). No long-term archive. |
| Range queries | Key scan + filter. O(all_keys) per query. Works at current scale, does not scale. |
| Aggregation | None. Raw candles only. Client must aggregate. |
| Schema evolution | None. JSON blob per key. |
| Operational cost | Zero additional. Part of existing NATS cluster. |
| Cardinality scaling | Degrades. `Keys()` scans entire bucket. At 100 symbols × 4 timeframes × 1440 candles/day = 576K keys, prefix scan becomes expensive. |
| Multi-timeframe join | Not possible. Each timeframe is a separate key space. |
| Replay/rebuild | Must replay from EVIDENCE_EVENTS stream (72h). No projection beyond 24h. |

**Verdict:** Correct for current scale. Breaks at ~50+ symbols or when analytics queries require aggregation, joins, or retention beyond 24h.

### 2. ClickHouse

**What it is:** Column-oriented OLAP database optimized for time-series analytics, append-heavy workloads, and fast aggregation over large datasets.

| Aspect | Assessment |
|--------|-----------|
| Write throughput | Designed for millions of rows/sec in batches. 8/min is trivially handled. |
| Read latency | Analytical queries (aggregation, range scans) in milliseconds. Not optimized for single-row point lookups. |
| Retention | Unbounded. TTL policies configurable per table. Months/years of history trivially stored. |
| Range queries | Native `WHERE open_time BETWEEN ? AND ? ORDER BY open_time DESC LIMIT ?`. O(log n) with MergeTree. |
| Aggregation | Native. `SELECT toStartOfHour(open_time), min(low), max(high), ... GROUP BY 1`. Server-side. |
| Schema evolution | ALTER TABLE ADD COLUMN is online and instant. Schema is explicit. |
| Operational cost | New infrastructure. Single node sufficient for current scale. Docker image available. |
| Cardinality scaling | Excellent. Partition by month, order by (source, symbol, timeframe, open_time). 100M+ rows trivial. |
| Multi-timeframe join | Possible via SQL. 60s candles can be aggregated to 300s, 3600s, etc. server-side. |
| Replay/rebuild | Table is the archive. No need to replay from streams for historical data. |

**Strengths for market-foundry:**
- Natural fit for OHLCV time-series data
- MergeTree engine with `ORDER BY (source, symbol, timeframe, open_time)` maps 1:1 to existing key structure
- Batch insert aligns with finalized-candle-only writes (low frequency, no partial updates)
- ReplacingMergeTree provides server-side dedup on natural key (same semantics as history KV)

**Weaknesses for market-foundry today:**
- New infrastructure to operate (even as single Docker container)
- No Go-native driver in current dependency set (requires `clickhouse-go`)
- Point-lookup latency higher than NATS KV (analytical engine, not key-value store)
- Overkill for 4 key combinations and 8 writes/min

### 3. TimescaleDB

**What it is:** PostgreSQL extension for time-series data with hypertable abstraction.

| Aspect | Assessment |
|--------|-----------|
| Write throughput | Good. Thousands of rows/sec per node. |
| Read latency | Good for range queries. SQL with time-based partitioning. |
| Retention | Unbounded. Compression + retention policies. |
| Range queries | Native SQL with hypertable optimizations. |
| Aggregation | PostgreSQL SQL + continuous aggregates. Server-side. |
| Schema evolution | PostgreSQL ALTER TABLE. Familiar. |
| Operational cost | New infrastructure. Heavier than ClickHouse for OLAP workloads. |
| Cardinality scaling | Good. Hypertable chunks by time. Slower than ClickHouse for scan-heavy analytics. |
| Multi-timeframe join | Full SQL support. Continuous aggregates can pre-compute rollups. |
| Replay/rebuild | Table is the archive. Continuous aggregates rebuild automatically. |

**Strengths vs ClickHouse:**
- PostgreSQL compatibility (familiar ecosystem, tooling)
- Continuous aggregates (materialized views with auto-refresh)
- Stronger transactional guarantees (ACID)

**Weaknesses vs ClickHouse for this use case:**
- Slower scan performance on large datasets (row-oriented underneath)
- Compression less efficient than ClickHouse columnar format
- Higher memory footprint for equivalent workload
- Continuous aggregates add operational complexity
- ACID guarantees unnecessary — candle projections are idempotent and append-only

## Decision Matrix

| Criterion | Weight | NATS KV | ClickHouse | TimescaleDB |
|-----------|--------|---------|------------|-------------|
| Operational simplicity | High | ★★★★★ | ★★★☆☆ | ★★☆☆☆ |
| Current-scale adequacy | High | ★★★★★ | ★★★★★ | ★★★★★ |
| Future-scale readiness | High | ★☆☆☆☆ | ★★★★★ | ★★★★☆ |
| Analytical capability | Medium | ☆☆☆☆☆ | ★★★★★ | ★★★★☆ |
| Retention/archive | Medium | ★☆☆☆☆ | ★★★★★ | ★★★★★ |
| Boundary preservation | High | ★★★★★ | ★★★★☆ | ★★★★☆ |
| Dependency footprint | Medium | ★★★★★ | ★★★☆☆ | ★★☆☆☆ |
| OHLCV-specific fit | High | ★★☆☆☆ | ★★★★★ | ★★★☆☆ |

## Summary

**NATS KV** is the right choice today. **ClickHouse** is the right choice for the analytical layer when triggered by concrete demand (multi-symbol scale, retention > 24h, aggregation queries). **TimescaleDB** is a reasonable alternative but offers no structural advantage over ClickHouse for this workload and introduces PostgreSQL operational weight without needing ACID guarantees.
