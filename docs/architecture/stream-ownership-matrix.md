# Stream Ownership Matrix

> Consolidated cross-cutting view of who writes, reads, projects, and queries every flow in the mesh.

---

## Event Stream Ownership

| Stream | Family | Writer Binary | Writer Actor | Consumer Binaries | Consumer Actors (Durable) | Phase | Status |
|--------|--------|--------------|-------------|-------------------|--------------------------|-------|--------|
| `CONFIGCTL_EVENTS` | configctl | gateway (configctl) | EventRouterActor | ingest, derive | BindingWatcherActor (`ingest-binding-watcher`), BindingWatcherActor (`derive-binding-watcher`) | 0 | Active |
| `OBSERVATION_EVENTS` | observation | ingest | PublisherActor (per source) | derive | ConsumerActor (`derive-observation`) | 2 | Active |
| `EVIDENCE_EVENTS` | evidence | derive | EvidencePublisherActor (per source) | store | EvidenceConsumerActor (`store-candle`), TradeBurstConsumerActor (`store-trade-burst`), VolumeConsumerActor (`store-volume`) | 3 | Active |
| `SIGNAL_EVENTS` | signal | derive | (future) | store | (future) | — | Planned |
| `PROJECTION_EVENTS` | projection | store | (future) | gateway | (future) | — | Planned |

### Single-Writer Invariant

Every row has exactly one Writer Binary. This is non-negotiable. Verification:

| Stream | Writer Count | Compliant |
|--------|-------------|-----------|
| CONFIGCTL_EVENTS | 1 (configctl) | Yes |
| OBSERVATION_EVENTS | 1 (ingest) | Yes |
| EVIDENCE_EVENTS | 1 (derive) | Yes |

---

## Projection Ownership

| KV Bucket | Family | Type | Writer Binary | Writer Actor | Read Binaries | Read Actor | Status |
|-----------|--------|------|--------------|-------------|---------------|------------|--------|
| `CANDLE_LATEST` | evidence | candle | store | CandleProjectionActor | store | QueryResponderActor | Active |
| `CANDLE_HISTORY` | evidence | candle | store | CandleProjectionActor | store | QueryResponderActor | Active |
| `TRADE_BURST_LATEST` | evidence | tradeburst | store | TradeBurstProjectionActor | store | QueryResponderActor | Active |
| `VOLUME_LATEST` | evidence | volume | store | VolumeProjectionActor | store | QueryResponderActor | Active |

### Single-Writer Invariant (Projection)

Every KV bucket has exactly one Writer Actor. No bucket is written to by multiple actors or binaries.

| Bucket | Writer Count | Compliant |
|--------|-------------|-----------|
| CANDLE_LATEST | 1 (CandleProjectionActor) | Yes |
| CANDLE_HISTORY | 1 (CandleProjectionActor) | Yes |
| TRADE_BURST_LATEST | 1 (TradeBurstProjectionActor) | Yes |
| VOLUME_LATEST | 1 (VolumeProjectionActor) | Yes |

---

## Query Surface Ownership

| Subject | Family | Type | Server Binary | Server Actor | Client Binary | Queue Group | Status |
|---------|--------|------|--------------|-------------|---------------|-------------|--------|
| `configctl.control.create_draft` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.get_config` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.get_active` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.list_configs` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.list_active_runtime_projections` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.list_active_ingestion_bindings` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.validate_draft` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.validate_config` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.compile_config` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `configctl.control.activate_config` | configctl | — | gateway (configctl) | ControlResponderActor | gateway | `configctl.control` | Active |
| `evidence.query.candle.latest` | evidence | candle | store | QueryResponderActor | gateway | `evidence.query` | Active |
| `evidence.query.candle.history` | evidence | candle | store | QueryResponderActor | gateway | `evidence.query` | Active |
| `evidence.query.tradeburst.latest` | evidence | tradeburst | store | QueryResponderActor | gateway | `evidence.query` | Active |
| `evidence.query.volume.latest` | evidence | volume | store | QueryResponderActor | gateway | `evidence.query` | Active |

### Single-Server Invariant

Every query subject has exactly one Server Binary. No subject is served by multiple binaries.

---

## Binary Role Summary

| Binary | Writes To | Reads From | Projects To | Serves Queries |
|--------|-----------|------------|-------------|----------------|
| **gateway** | — | — | — | HTTP → NATS translation only |
| **configctl** (in gateway) | CONFIGCTL_EVENTS | — | — | `configctl.control.*` |
| **ingest** | OBSERVATION_EVENTS | CONFIGCTL_EVENTS | — | — |
| **derive** | EVIDENCE_EVENTS | OBSERVATION_EVENTS, CONFIGCTL_EVENTS | — | — |
| **store** | — | EVIDENCE_EVENTS | CANDLE_LATEST, CANDLE_HISTORY, TRADE_BURST_LATEST, VOLUME_LATEST | `evidence.query.*` |

### Ownership Characteristics

- **gateway**: Stateless translator. No streams, no projections, no domain logic.
- **configctl**: Single authority for configuration. Write-only to CONFIGCTL_EVENTS. Serves all config control queries.
- **ingest**: Write-only to OBSERVATION_EVENTS. Read-only consumer of configctl bindings. No queries, no projections.
- **derive**: Write-only to EVIDENCE_EVENTS. Read-only consumer of observation and configctl. No queries served (write-only pipeline).
- **store**: Read-only consumer of EVIDENCE_EVENTS. Write-only to KV buckets. Sole server for all evidence queries. Never produces domain events.

### Data Flow Direction

```
configctl ──events──→ ingest ──events──→ derive ──events──→ store ──queries──→ gateway
    │                                       │                 │
    └──events──→ derive                     │                 └──KV──→ (read by self)
                                            │
                                            └──events──→ store
```

There are no feedback loops. No binary both produces to and consumes from the same stream. The flow is strictly unidirectional: configctl → ingest → derive → store → gateway.

---

## Consumer Cursor Summary

| Durable Name | Binary | Stream | Filter Subject | Deliver Policy | Purpose |
|-------------|--------|--------|----------------|----------------|---------|
| `ingest-binding-watcher` | ingest | CONFIGCTL_EVENTS | `configctl.events.config.ingestion_runtime_changed` | DeliverLastPerSubject | Latest binding state |
| `derive-binding-watcher` | derive | CONFIGCTL_EVENTS | `configctl.events.config.ingestion_runtime_changed` | DeliverLastPerSubject | Latest binding state |
| `derive-observation` | derive | OBSERVATION_EVENTS | `observation.events.market.trade.>` | DeliverAll | All trades for sampling |
| `store-candle` | store | EVIDENCE_EVENTS | `evidence.events.candle.sampled.>` | DeliverAll | Candle projection |
| `store-trade-burst` | store | EVIDENCE_EVENTS | `evidence.events.tradeburst.sampled.>` | DeliverAll | Trade burst projection |
| `store-volume` | store | EVIDENCE_EVENTS | `evidence.events.volume.sampled.>` | DeliverAll | Volume projection |

All consumers use AckWait=30s, MaxDeliver=5 unless otherwise specified.

---

## Ownership Verification Checklist

Use this checklist when adding new flows or verifying the matrix:

- [ ] Every JetStream stream has exactly one Writer Binary
- [ ] Every KV bucket has exactly one Writer Actor
- [ ] Every query subject has exactly one Server Binary
- [ ] Every consumer has a unique Durable Name
- [ ] No binary both writes to and reads from the same stream
- [ ] No cross-family event dependencies within a single processing step
- [ ] Store is the sole projection authority for all evidence types
- [ ] Gateway accesses data only through query surfaces, never KV directly
