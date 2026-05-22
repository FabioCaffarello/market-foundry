# ADR 0003: ClickHouse for analytical storage

## Status

Accepted.

## Context

market-foundry needed an analytical store for:
- Long-term retention of domain events (evidence, signal, decision,
  strategy, risk, execution).
- Time-range queries with aggregations.
- Explainability reads ("what happened during this session/round-trip").
- Roundtrip reconciliation and effectiveness classification.

Operational reads (latest-value queries per partition) are served by
NATS KV. The need was for the **historical/analytical** path.

The candidates:

- **PostgreSQL with time partitioning** — general-purpose RDBMS.
- **TimescaleDB** — PostgreSQL extension for time-series.
- **ClickHouse** — columnar analytical database.
- **InfluxDB** — purpose-built time-series database.

System requirements:

- Write-heavy: every signal/decision/strategy/risk computation is
  potentially a write.
- Read patterns: aggregations over time ranges, joins between
  domains (decisions joined to their effectiveness, etc.).
- Schema evolution: domain types grow; new event types appear.

## Decision

**ClickHouse is the analytical store.** Reads at the `/analytical/*`
HTTP endpoints go through writer's read adapter against ClickHouse
tables. Migrations are forward-only and tracked in a `_migrations`
metadata table.

NATS KV serves operational latest-value reads; ClickHouse serves
historical and analytical reads. The two are deliberately separate
stores with different access patterns.

## Consequences

### Positive

- **Excellent write performance**: ClickHouse handles the volume of
  per-partition event writes with low overhead.
- **Time-range query performance**: PREWHERE + ORDER BY (TimeStamp)
  natural pattern, fast scans.
- **Aggregation built-in**: ClickHouse functions (uniq, quantile,
  sumIf, etc.) make analytical queries concise.
- **Columnar storage**: efficient for the workload pattern
  (write many narrow events, read a few columns over a range).
- **Self-contained**: no external dependencies beyond the binary.

### Negative

- **Eventual consistency with NATS streams**: writer consumes from
  JetStream and writes to ClickHouse, so there's a delay between
  an event landing in NATS and being queryable in ClickHouse.
  Analytical reads are not real-time.
- **Joins are limited**: ClickHouse doesn't excel at multi-table
  joins. Composite analytical reads (decision + effectiveness +
  pairing) require careful query design or pre-joined materialized
  views.
- **Schema migrations are append-only**: forward-only migrations
  mean schema changes accumulate; no easy "drop and recreate"
  if a column becomes wrong.
- **Operator unfamiliarity**: ClickHouse is less commonly known than
  PostgreSQL; on-call diagnostics require ClickHouse-specific knowledge.

## Alternatives considered

**PostgreSQL with time partitioning**: rejected for performance.
PostgreSQL's row-based storage is inefficient for the read pattern
(scan many narrow events to extract a few columns).

**TimescaleDB**: closer to fit but inherits PostgreSQL's row-storage
limitation. Hypertables help but ClickHouse outperforms for the
specific workload.

**InfluxDB**: rejected for query expressiveness. Flux/InfluxQL are
less flexible than SQL for the kinds of analytical queries the
gateway needs to serve.

**Use NATS KV for everything**: rejected because KV cannot serve
time-range queries efficiently. KV is for operational latest-value
reads; analytical needs are different.

## References

- `internal/adapters/clickhouse/` — adapter implementation
- `deploy/migrations/` — 8 forward-only SQL migrations
- [`../RUNTIME.md`](../RUNTIME.md) → "ClickHouse migrations"
- [`../operations/backups.md`](../operations/backups.md) — backup strategy
- ClickHouse docs: https://clickhouse.com/docs
