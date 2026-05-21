# Stream Family Catalog

> Operational catalog of every stream family in the Market Foundry mesh.
> Each entry is a self-contained contract: name, ownership, dimensions, transport, and readiness.

---

## How to Read This Catalog

Each family entry follows a fixed schema:

| Field | Meaning |
|-------|---------|
| **Canonical name** | The family identifier used in subjects, docs, and code |
| **Bounded context** | The domain this family belongs to |
| **Classification** | Flow character: continuous, sampled, derived, lifecycle, projection |
| **Publisher owner** | The binary that produces events to this family's stream |
| **Consumer owners** | The binaries that consume from this family's stream |
| **Projection owner** | The binary that materializes read models from this family |
| **Query owner** | The binary that serves queries for this family's data |
| **Dimensions** | Partition keys in the subject encoding |
| **Stream** | JetStream stream name |
| **Retention** | MaxAge / MaxBytes |
| **Readiness** | Current, Planned, or Deferred |

---

## CF-01: configctl

| Field | Value |
|-------|-------|
| **Canonical name** | `configctl` |
| **Bounded context** | Configuration Lifecycle |
| **Classification** | Lifecycle |
| **Publisher owner** | configctl (EventRouterActor) |
| **Consumer owners** | ingest (BindingWatcherActor), derive (BindingWatcherActor) |
| **Projection owner** | None — configctl stores config state in-memory via ControlRouterActor |
| **Query owner** | configctl (ControlResponderActor) |
| **Dimensions** | None (single config authority, no market-data partitioning) |
| **Stream** | `CONFIGCTL_EVENTS` |
| **Retention** | 24h, 256 MB |
| **Readiness** | Current |

### Event Types

| Event | Subject | Envelope Type |
|-------|---------|---------------|
| Draft created | `configctl.events.config.draft_created` | `configctl.event.config.draft_created` |
| Validated | `configctl.events.config.validated` | `configctl.event.config.validated` |
| Compiled | `configctl.events.config.compiled` | `configctl.event.config.compiled` |
| Activated | `configctl.events.config.activated` | `configctl.event.config.activated` |
| Deactivated | `configctl.events.config.deactivated` | `configctl.event.config.deactivated` |
| Ingestion runtime changed | `configctl.events.config.ingestion_runtime_changed` | `configctl.event.config.ingestion_runtime_changed` |
| Archived | `configctl.events.config.archived` | `configctl.event.config.archived` |
| Rejected | `configctl.events.config.rejected` | `configctl.event.config.rejected` |

### Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|----------------|---------|
| `ingest-binding-watcher` | ingest | `configctl.events.config.ingestion_runtime_changed` | Activate/deactivate exchange bindings |
| `derive-binding-watcher` | derive | `configctl.events.config.ingestion_runtime_changed` | Activate/deactivate source scopes and samplers |

### Query Surface

| Subject | Request Type | Reply Type | Queue Group | Server |
|---------|-------------|------------|-------------|--------|
| `configctl.control.create_draft` | `configctl.command.create_draft` | `configctl.reply.create_draft` | `configctl.control` | configctl |
| `configctl.control.get_config` | `configctl.query.get_config` | `configctl.reply.get_config` | `configctl.control` | configctl |
| `configctl.control.get_active` | `configctl.query.get_active` | `configctl.reply.get_active` | `configctl.control` | configctl |
| `configctl.control.list_configs` | `configctl.query.list_configs` | `configctl.reply.list_configs` | `configctl.control` | configctl |
| `configctl.control.list_active_runtime_projections` | `configctl.query.list_active_runtime_projections` | `configctl.reply.list_active_runtime_projections` | `configctl.control` | configctl |
| `configctl.control.list_active_ingestion_bindings` | `configctl.query.list_active_ingestion_bindings` | `configctl.reply.list_active_ingestion_bindings` | `configctl.control` | configctl |
| `configctl.control.validate_draft` | `configctl.command.validate_draft` | `configctl.reply.validate_draft` | `configctl.control` | configctl |
| `configctl.control.validate_config` | `configctl.command.validate_config` | `configctl.reply.validate_config` | `configctl.control` | configctl |
| `configctl.control.compile_config` | `configctl.command.compile_config` | `configctl.reply.compile_config` | `configctl.control` | configctl |
| `configctl.control.activate_config` | `configctl.command.activate_config` | `configctl.reply.activate_config` | `configctl.control` | configctl |

### Architectural Notes

- configctl is **embedded** in the gateway binary, not a separate service. It runs as a supervised actor tree within the gateway process.
- The `ingestion_runtime_changed` event is the activation trigger for the entire data pipeline. Without it, ingest and derive remain idle.
- Consumer deliver policy is `DeliverLastPerSubject` for binding watchers — they only need the latest binding state, not the full history.

---

## CF-02: observation

| Field | Value |
|-------|-------|
| **Canonical name** | `observation` |
| **Bounded context** | Market Data Ingestion |
| **Classification** | Continuous |
| **Publisher owner** | ingest (PublisherActor, one per ExchangeScopeActor) |
| **Consumer owners** | derive (ConsumerActor) |
| **Projection owner** | None currently |
| **Query owner** | None currently (future: store) |
| **Dimensions** | `source` |
| **Stream** | `OBSERVATION_EVENTS` |
| **Retention** | 6h, 1 GB |
| **Readiness** | Current |

### Event Types

| Type Name | Subject Pattern | Envelope Type | Dedup Key |
|-----------|----------------|---------------|-----------|
| `observation.trade` | `observation.events.market.trade.{source}` | `observation.events.v1.trade_received` | `{source}:{trade_id}` |

### Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|----------------|---------|
| `derive-observation` | derive | `observation.events.market.trade.>` | Feed trades to sampler pipeline |

### Actor Ownership Chain

```
IngestSupervisor
└── ExchangeScopeActor (per source)
    ├── PublisherActor → OBSERVATION_EVENTS
    └── WebSocketAdapterActor[] (per symbol)
```

### Architectural Notes

- Partitioned by `source` only. Derive routes by symbol internally via SourceScopeActor.
- One WebSocket connection per symbol within each exchange scope. All symbols for one exchange share a single PublisherActor.
- No query surface today. Future `observation.query.latest.*` would be served by store if raw trade access is needed downstream.
- Source-level partitioning is a deliberate design choice — see [stream-families.md](stream-families.md) for rationale.

---

## CF-03: evidence.candle

| Field | Value |
|-------|-------|
| **Canonical name** | `evidence.candle` |
| **Bounded context** | Market Evidence |
| **Classification** | Sampled |
| **Publisher owner** | derive (EvidencePublisherActor, one per SourceScopeActor) |
| **Consumer owners** | store (EvidenceConsumerActor) |
| **Projection owner** | store (CandleProjectionActor) |
| **Query owner** | store (QueryResponderActor) |
| **Dimensions** | `source`, `symbol`, `timeframe` |
| **Stream** | `EVIDENCE_EVENTS` (shared with evidence.tradeburst) |
| **Retention** | 72h, 2 GB (shared) |
| **Readiness** | Current |

### Event Types

| Type Name | Subject Pattern | Envelope Type | Dedup Key |
|-----------|----------------|---------------|-----------|
| `candle.sampled` | `evidence.events.candle.sampled.{source}.{symbol}.{timeframe}` | `evidence.events.v1.candle_sampled` | `{source}:{symbol}:{timeframe}:{open_time_unix}` |

### Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|----------------|---------|
| `store-candle` | store | `evidence.events.candle.sampled.>` | Materialize candle projections |

### KV Projections

| Bucket | Key Format | MaxBytes | TTL | Writer Actor |
|--------|-----------|----------|-----|-------------|
| `CANDLE_LATEST` | `{source}.{symbol}.{timeframe}` | 64 MB | — | CandleProjectionActor |
| `CANDLE_HISTORY` | `{source}.{symbol}.{timeframe}.{open_time_unix}` | 256 MB | 24h | CandleProjectionActor |

### Query Surface

| Subject | Request Type | Reply Type | Queue Group | Server |
|---------|-------------|------------|-------------|--------|
| `evidence.query.candle.latest` | `evidence.query.v1.candle_latest_request` | `evidence.query.v1.candle_latest_reply` | `evidence.query` | store |
| `evidence.query.candle.history` | `evidence.query.v1.candle_history_request` | `evidence.query.v1.candle_history_reply` | `evidence.query` | store |

### HTTP Endpoints (via gateway)

| Method | Path | Query Params |
|--------|------|-------------|
| GET | `/evidence/candles/latest` | `source`, `symbol`, `timeframe` |
| GET | `/evidence/candles/history` | `source`, `symbol`, `timeframe`, `limit`, `since`, `until` |

### Actor Ownership Chain

```
DeriveSupervisor
└── SourceScopeActor (per source)
    ├── EvidencePublisherActor → EVIDENCE_EVENTS
    └── SamplerActor[] (per symbol × timeframe)

StoreSupervisor
├── EvidenceConsumerActor ← EVIDENCE_EVENTS (filter: candle.sampled.>)
├── CandleProjectionActor → CANDLE_LATEST, CANDLE_HISTORY
└── QueryResponderActor ⇄ evidence.query.candle.*
```

### Materialization Rules

- Only candles with `Final=true` are written to CANDLE_LATEST.
- Monotonicity guard: skip if existing candle has newer or equal OpenTime.
- CANDLE_HISTORY is idempotent by key design (key includes `open_time_unix`).
- Non-final (interim) candles are never materialized — they exist only in the event stream.

---

## CF-04: evidence.tradeburst

| Field | Value |
|-------|-------|
| **Canonical name** | `evidence.tradeburst` |
| **Bounded context** | Market Evidence |
| **Classification** | Sampled |
| **Publisher owner** | derive (EvidencePublisherActor, shared with candle) |
| **Consumer owners** | store (TradeBurstConsumerActor) |
| **Projection owner** | store (TradeBurstProjectionActor) |
| **Query owner** | store (QueryResponderActor) |
| **Dimensions** | `source`, `symbol`, `timeframe` |
| **Stream** | `EVIDENCE_EVENTS` (shared with evidence.candle) |
| **Retention** | 72h, 2 GB (shared) |
| **Readiness** | Current |

### Event Types

| Type Name | Subject Pattern | Envelope Type | Dedup Key |
|-----------|----------------|---------------|-----------|
| `tradeburst.sampled` | `evidence.events.tradeburst.sampled.{source}.{symbol}.{timeframe}` | `evidence.events.v1.trade_burst_sampled` | `burst:{source}:{symbol}:{timeframe}:{open_time_unix}` |

### Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|----------------|---------|
| `store-trade-burst` | store | `evidence.events.tradeburst.sampled.>` | Materialize trade burst projections |

### KV Projections

| Bucket | Key Format | MaxBytes | TTL | Writer Actor |
|--------|-----------|----------|-----|-------------|
| `TRADE_BURST_LATEST` | `{source}.{symbol}.{timeframe}` | 64 MB | — | TradeBurstProjectionActor |

### Query Surface

| Subject | Request Type | Reply Type | Queue Group | Server |
|---------|-------------|------------|-------------|--------|
| `evidence.query.tradeburst.latest` | `evidence.query.v1.trade_burst_latest_request` | `evidence.query.v1.trade_burst_latest_reply` | `evidence.query` | store |

### HTTP Endpoints (via gateway)

| Method | Path | Query Params |
|--------|------|-------------|
| GET | `/evidence/tradeburst/latest` | `source`, `symbol`, `timeframe` |

### Actor Ownership Chain

```
DeriveSupervisor
└── SourceScopeActor (per source)
    ├── EvidencePublisherActor → EVIDENCE_EVENTS
    └── TradeBurstSamplerActor[] (per symbol × timeframe)

StoreSupervisor
├── TradeBurstConsumerActor ← EVIDENCE_EVENTS (filter: tradeburst.sampled.>)
├── TradeBurstProjectionActor → TRADE_BURST_LATEST
└── QueryResponderActor ⇄ evidence.query.tradeburst.*
```

### Intentional Limitations

- No history bucket. Latest-only projection for now.
- Burst threshold (2.0x) is hardcoded, not configurable.
- Single-window baseline — no rolling average.
- No burst-specific query filter — clients filter via the `burst` boolean field.

---

## CF-05: signal (Planned)

| Field | Value |
|-------|-------|
| **Canonical name** | `signal` |
| **Bounded context** | Trading Signals |
| **Classification** | Derived |
| **Publisher owner** | derive (future SignalPublisherActor) |
| **Consumer owners** | store (future) |
| **Projection owner** | store (future) |
| **Query owner** | store (future) |
| **Dimensions** | `source`, `symbol`, `timeframe` (possibly `strategy`) |
| **Stream** | `SIGNAL_EVENTS` |
| **Retention** | 72h (estimated), TBD MaxBytes |
| **Readiness** | Planned — blocked by config-driven activation prerequisite (S25) |

### Anticipated Event Types

| Type Name | Subject Pattern | Notes |
|-----------|----------------|-------|
| Signal generated | `signal.events.{type}.generated.{source}.{symbol}.{timeframe}` | When signal conditions are met |
| Signal expired | `signal.events.{type}.expired.{source}.{symbol}.{timeframe}` | When signal conditions lapse |

### Design Constraints

- Signals derive from evidence, never from raw observation.
- Signal and evidence share the derive binary but produce to separate streams.
- May introduce a `strategy` dimension if multiple signal models coexist for the same symbol/timeframe.
- No signal implementation until the activation mechanism from configctl is proven.

---

## CF-06: projection (Planned)

| Field | Value |
|-------|-------|
| **Canonical name** | `projection` |
| **Bounded context** | Read Model Notifications |
| **Classification** | Projection |
| **Publisher owner** | store (future) |
| **Consumer owners** | gateway (future, for cache invalidation) |
| **Projection owner** | N/A — this family IS the projection notification |
| **Query owner** | N/A |
| **Dimensions** | `family`, `type` |
| **Stream** | `PROJECTION_EVENTS` |
| **Retention** | 1-2h (estimated), TBD MaxBytes |
| **Readiness** | Planned — low priority, gateway is stateless today |

### Anticipated Event Types

| Type Name | Subject Pattern | Notes |
|-----------|----------------|-------|
| Projection materialized | `projection.events.{family}.{type}.materialized` | Notify that a KV bucket was updated |

### Design Notes

- This is the only family where store is the writer.
- Useful when gateway or external consumers need push-based update notifications.
- Not needed while gateway remains stateless. Becomes relevant with caching or SSE/WebSocket push.

---

## CF-07: evidence.volume (implemented S31)

| Field | Value |
|-------|-------|
| **Canonical name** | `evidence.volume` |
| **Bounded context** | Market Evidence |
| **Classification** | Sampled |
| **Publisher owner** | derive (EvidencePublisherActor, shared) |
| **Consumer owners** | store (VolumeConsumerActor) |
| **Projection owner** | store (VolumeProjectionActor) |
| **Query owner** | store (QueryResponderActor, shared) |
| **Dimensions** | `source`, `symbol`, `timeframe` |
| **Stream** | `EVIDENCE_EVENTS` (shared) |
| **Retention** | 72h, 2 GB (shared) |
| **Readiness** | Current |

### Event Types

| Type Name | Subject Pattern | Envelope Type | Dedup Key |
|-----------|----------------|---------------|-----------|
| `volume.sampled` | `evidence.events.volume.sampled.{source}.{symbol}.{timeframe}` | `evidence.events.v1.volume_sampled` | `vol:{source}:{symbol}:{timeframe}:{open_time_unix}` |

### Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|----------------|---------|
| `store-volume` | store | `evidence.events.volume.sampled.>` | Materialize volume projections |

### KV Projections

| Bucket | Key Format | MaxBytes | TTL | Writer Actor |
|--------|-----------|----------|-----|-------------|
| `VOLUME_LATEST` | `{source}.{symbol}.{timeframe}` | 64 MB | — | VolumeProjectionActor |

### Query Surface

| Subject | Request Type | Reply Type | Queue Group | Server |
|---------|-------------|------------|-------------|--------|
| `evidence.query.volume.latest` | `evidence.query.v1.volume_latest_request` | `evidence.query.v1.volume_latest_reply` | `evidence.query` | store |

### HTTP Endpoints (via gateway)

| Method | Path | Query Params |
|--------|------|-------------|
| GET | `/evidence/volume/latest` | `source`, `symbol`, `timeframe` |

### Actor Ownership Chain

```
DeriveSupervisor
└── SourceScopeActor (per source)
    ├── EvidencePublisherActor → EVIDENCE_EVENTS
    └── VolumeSamplerActor[] (per symbol × timeframe)

StoreSupervisor
├── VolumeConsumerActor ← EVIDENCE_EVENTS (filter: volume.sampled.>)
├── VolumeProjectionActor → VOLUME_LATEST
└── QueryResponderActor ⇄ evidence.query.volume.*
```

### Domain Fields

- `BuyVolume` — notional buy volume (price × qty for buyer-is-maker trades)
- `SellVolume` — notional sell volume
- `TotalVolume` — BuyVolume + SellVolume
- `VWAP` — volume-weighted average price
- `TradeCount` — number of trades in the window

---

## CF-08: evidence.stats (Planned)

| Field | Value |
|-------|-------|
| **Canonical name** | `evidence.stats` |
| **Bounded context** | Market Evidence |
| **Classification** | Sampled |
| **Publisher owner** | derive (EvidencePublisherActor, shared) |
| **Consumer owners** | store |
| **Projection owner** | store |
| **Query owner** | store (QueryResponderActor, shared) |
| **Dimensions** | `source`, `symbol`, `timeframe` |
| **Stream** | `EVIDENCE_EVENTS` (shared) |
| **Retention** | 72h, 2 GB (shared) |
| **Readiness** | Planned |

### Rationale

Statistical summary per window: trade count distribution, price volatility (standard deviation), min/max spread, tick frequency. Complements candle (price) and trade burst (activity) with distributional evidence.

### Subject Pattern

`evidence.events.stats.sampled.{source}.{symbol}.{timeframe}`

### Why Planned, Not Deferred

- Same justification as volume — pure evidence derivation, no new infrastructure.
- Directly useful for signal readiness (provides volatility and spread metrics).

---

## Deferred Families

The following families are naming reservations only. They have no architecture documents, no stream specs, and no planned implementation timeline. They are listed to prevent naming collisions and to document the conceptual domain progression inherited from Market Raccoon.

| Family | Classification | Bounded Context | Blocked By |
|--------|---------------|-----------------|------------|
| `decision` | Derived | Strategy/Decision | Signal domain must be operational first |
| `risk` | Derived | Risk Management | Decision domain must be operational first |
| `execution` | Lifecycle | Order Execution | Risk domain must be operational first |
| `portfolio` | Lifecycle | Portfolio State | Execution domain must be operational first |

**Rule:** No code, no registry entry, no subject pattern may reference these families until a dedicated architecture document is approved. The domain progression (observation → evidence → signal → decision → risk → execution → portfolio) is sequential and gated.
