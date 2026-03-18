# Read-Model Authority: Store Service

> Canonical reference for read-side ownership in market-foundry's CQRS architecture.

## Role

Store is the read-side authority. It consumes domain events, materializes read-optimized projections, and serves queries. It has no write-side responsibilities and produces no canonical domain events.

## Responsibilities

1. **Consume domain events** from JetStream streams (currently `EVIDENCE_EVENTS`).
2. **Materialize projections** into read-optimized storage (currently NATS KV buckets).
3. **Serve queries** via NATS request/reply, consumed by the gateway service.

## Non-Responsibilities

Store does **not**:

- Produce canonical domain events.
- Execute domain logic (sampling, aggregation, validation).
- Own any write-side concern. All upstream event production belongs to ingest and derive.

## Projection Inventory

The store manages multiple projection pipelines. Each pipeline has its own consumer, projection actor, and KV bucket(s).

| Evidence Type | Consumer | Projection Actor | Bucket(s) | Query Subject(s) |
|---------------|----------|-----------------|-----------|-------------------|
| Candle | `store-evidence` | `CandleProjectionActor` | `CANDLE_LATEST`, `CANDLE_HISTORY` | `evidence.query.candle.latest`, `evidence.query.candle.history` |
| Trade Burst | `store-trade-burst` | `TradeBurstProjectionActor` | `TRADE_BURST_LATEST` | `evidence.query.tradeburst.latest` |

### CANDLE_LATEST KV Bucket

- **Key format:** `{source}.{symbol}.{timeframe}`
- **Materialization rule:** only candles with `Final=true` are written. Monotonicity guard prevents regression.
- **Writer:** `CandleProjectionActor`.

### CANDLE_HISTORY KV Bucket

- **Key format:** `{source}.{symbol}.{timeframe}.{open_time_unix}`
- **Retention:** 24h TTL, 256MB max.
- **Query semantics:** prefix scan + optional time-range filtering + descending sort by open_time.
- **Writer:** `CandleProjectionActor` (same actor writes both candle buckets).

### TRADE_BURST_LATEST KV Bucket

- **Key format:** `{source}.{symbol}.{timeframe}`
- **Materialization rule:** only bursts with `Final=true` are written. Monotonicity guard prevents regression.
- **Writer:** `TradeBurstProjectionActor`.

## Query Serving Pattern

`QueryResponderActor` implements all evidence query routes. It opens read-only connections to all KV buckets and registers typed control routes for each query subject.

| Query | Subject | Queue group | Data source |
|-------|---------|-------------|-------------|
| Latest candle | `evidence.query.candle.latest` | `evidence.query` | `CANDLE_LATEST` KV |
| Candle history | `evidence.query.candle.history` | `evidence.query` | `CANDLE_HISTORY` KV |
| Latest trade burst | `evidence.query.tradeburst.latest` | `evidence.query` | `TRADE_BURST_LATEST` KV |

The gateway sends a NATS request to the query subject. Store receives the request through the queue group, reads the projection from KV, and returns the reply. The gateway never accesses KV directly.

## Health Tracking

Each projection pipeline has its own pair of health trackers:

| Tracker | Component |
|---------|-----------|
| `candle-projection` | CandleProjectionActor materialization events |
| `candle-consumer` | Candle evidence consumer message processing |
| `trade-burst-projection` | TradeBurstProjectionActor materialization events |
| `trade-burst-consumer` | Trade burst consumer message processing |

All trackers surface on `/statusz` with independent idle detection. This allows operators to see health per evidence type.

## Ownership Boundaries

| Boundary | Rule |
|---|---|
| `evidence.query.*` subjects | Store is the sole server. No other service subscribes as a responder. |
| `EVIDENCE_EVENTS` stream | Store is a read-only consumer. It never publishes to this stream. |
| Derive service | Writes evidence events but does not serve queries. |
| Gateway service | Has no direct access to KV buckets. All reads go through store's query interface. |

## Adding Future Projections

See `evidence-read-model-guidelines.md` for the complete checklist. The structural pattern:

1. **Consumer** — durable JetStream consumer filtering by evidence-type-specific subject prefix.
2. **Projection actor** — applies Final gate + Validate gate + monotonicity guard, writes to KV.
3. **Health trackers** — one pair per projection type, registered on `/statusz`.
4. **Query route** — registered in `QueryResponderActor` via typed control route.

Each projection pipeline is independent. Adding a new evidence type does not modify existing pipelines.
