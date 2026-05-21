# Future Analytics Signals — Candidates for ClickHouse Persistence

> Disciplined catalog of operational and domain signals that would benefit from
> persistent time-series storage. No implementation — mapping only.
>
> Pre-condition: ClickHouse is present in Docker Compose but unused.
> Core principle: ClickHouse must remain optional; the pipeline never depends on it.

## Signal Categories

### Category 1: Tracker Telemetry (Operational)

These signals are currently available as point-in-time snapshots via `/statusz` and `/diagz`. Persisting them would enable trend analysis, regression detection, and capacity planning.

| Signal                    | Source Runtime | Current Exposure | ClickHouse Value |
|---------------------------|---------------|------------------|------------------|
| `event_count` per tracker | all           | /statusz snapshot | Throughput trends over time |
| `error_count` per tracker | all           | /statusz snapshot | Error rate trends, anomaly detection |
| `idle_seconds` per tracker| all           | /statusz snapshot | Pipeline stall history |
| `phase` per runtime       | all           | /statusz snapshot | Phase transition timeline |
| Custom counters (e.g., `filled`, `skipped_stale`) | derive, execute | /statusz snapshot | Domain-specific throughput breakdown |
| `num_goroutines`          | all           | /diagz snapshot   | Resource usage trends |

**Ingestion pattern:** Periodic scraper (e.g., every 30s) reading `/statusz` from each runtime and writing rows to a `runtime_telemetry` table.

**Schema sketch:**
```sql
CREATE TABLE runtime_telemetry (
    timestamp    DateTime64(3),
    runtime      LowCardinality(String),
    tracker      LowCardinality(String),
    event_count  UInt64,
    error_count  UInt64,
    idle_seconds UInt32,
    phase        LowCardinality(String),
    goroutines   UInt32
) ENGINE = MergeTree()
ORDER BY (runtime, tracker, timestamp)
TTL timestamp + INTERVAL 30 DAY;
```

### Category 2: Event Flow Metrics (Domain)

These signals represent the domain data flowing through NATS streams. They are currently consumed by store (for KV materialization) but not persisted as time-series.

| Signal                    | NATS Stream          | Current Consumer | ClickHouse Value |
|---------------------------|---------------------|------------------|------------------|
| Candle close events       | EVIDENCE_EVENTS     | store            | Historical candle archive, backtesting |
| Trade burst events        | EVIDENCE_EVENTS     | store            | Burst frequency analysis |
| Volume profile events     | EVIDENCE_EVENTS     | store            | Volume pattern analysis |
| RSI signal events         | (internal publish)  | store            | Signal history for strategy evaluation |
| Decision events           | (internal publish)  | store            | Decision audit trail |
| Strategy events           | (internal publish)  | store            | Strategy trigger analysis |
| Risk assessment events    | (internal publish)  | store            | Risk exposure history |
| Execution events          | (internal publish)  | store            | Trade execution log |
| Fill events               | (internal publish)  | store            | Fill quality analysis |

**Ingestion pattern:** Dedicated NATS consumer (same pattern as store service) subscribing to event streams and inserting into ClickHouse tables. The consumer would be a separate service or an optional module within store.

**Key design constraint:** The ClickHouse consumer must use its own durable consumer name (not share with store), so that store's KV materialization is unaffected.

### Category 3: Configuration Lifecycle (Audit)

| Signal                | Source              | ClickHouse Value |
|-----------------------|--------------------|------------------|
| Config draft created  | CONFIGCTL_EVENTS   | Config change audit trail |
| Config validated      | CONFIGCTL_EVENTS   | Validation history |
| Config activated      | CONFIGCTL_EVENTS   | Activation timeline |

**Ingestion pattern:** Same as Category 2 — a NATS consumer for `CONFIGCTL_EVENTS`.

### Category 4: Infrastructure Health (Operational)

| Signal                    | Source          | ClickHouse Value |
|---------------------------|----------------|------------------|
| Container memory usage    | docker stats   | Memory trend analysis |
| NATS readiness probe results | /diagz     | Connectivity reliability |
| Error log counts          | compose logs   | Error rate over time |

**Ingestion pattern:** External scraper (cron or sidecar) collecting from Docker API and health endpoints.

## Priority Ranking

| Priority | Category            | Rationale |
|----------|--------------------|-----------|
| P1       | Event Flow Metrics | Enables backtesting and strategy evaluation — direct product value |
| P2       | Tracker Telemetry  | Enables operational trend analysis and regression detection |
| P3       | Config Lifecycle   | Audit trail — important but low volume, can wait |
| P4       | Infrastructure Health | Useful but can be approximated with existing scripts |

## Pre-Conditions Before Implementation

These must be satisfied before writing any ClickHouse integration:

1. **CH-01**: Baseline is canonical and stable (satisfied by S137).
2. **CH-02**: Event schemas are stable (evidence, signal, decision, strategy, risk, execution).
3. **CH-03**: NATS consumer pattern is proven in production-like load (proven by store service).
4. **CH-04**: Retention policy is defined (30-day default suggested for telemetry, longer for domain events).
5. **CH-05**: ClickHouse schema migrations are designed (not ad-hoc DDL).
6. **CH-06**: Query use cases are defined (who queries what, how often).
7. **CH-07**: ClickHouse remains optional — pipeline health must not depend on it.

## Anti-Patterns to Avoid

- Writing to ClickHouse on the hot path (synchronous writes from derive or store).
- Using ClickHouse as a primary store (it supplements NATS KV, does not replace it).
- Creating ClickHouse tables without defined consumers (no write-only tables).
- Mixing operational telemetry and domain events in the same table.
- Adding ClickHouse as a readiness check dependency.

## Next Step

When the decision to implement ClickHouse ingestion is made, the recommended sequence is:

1. Define table schemas for P1 signals (event flow metrics).
2. Implement a ClickHouse writer service following the store consumer pattern.
3. Add a durable NATS consumer for each target stream.
4. Validate with `diag-check.sh` that ClickHouse ingestion does not affect pipeline throughput.
5. Only then expand to P2 (telemetry scraping).
