# Mesh vs. Transport

> Separating the logical stream mesh from its NATS encoding.

## The Distinction

The **stream mesh** is a logical architecture: named flows, ownership rules, partitioning dimensions, delivery guarantees, and family boundaries. It describes _what_ flows _where_ and _who_ owns it.

The **transport encoding** is how that logical architecture is realized on NATS: subject strings, JetStream stream configurations, KV bucket names, consumer specs, and envelope formats. It describes _how_ the mesh is implemented.

This distinction matters because:

1. **The mesh is stable; the transport evolves.** Adding a NATS cluster, changing retention, or migrating to a different subject hierarchy does not change what the mesh represents.

2. **Architecture discussions use mesh vocabulary.** When we say "evidence events flow from derive to store", we are speaking about the mesh. When we say "`evidence.events.candle.sampled.binancef.btcusdt.60`", we are speaking about transport.

3. **Tests can validate mesh invariants without transport.** A unit test that verifies "derive produces exactly one candle per window" is testing the mesh. An integration test that verifies "the NATS message ID prevents duplicate delivery" is testing transport.

4. **raccoon-cli audits both layers independently.** Contract audits validate mesh-level rules (ownership, naming). Drift detection validates transport-level rules (registry specs match runtime).

## Mesh Layer

### Vocabulary

| Mesh Term | Meaning |
|-----------|---------|
| **Family** | A named group of related flows sharing a domain boundary (e.g., `evidence`) |
| **Surface** | The communication pattern: events, control, query, projection |
| **Aggregate** | The domain entity within a family (e.g., `candle`, `tradeburst`) |
| **Partition key** | The dimensions that scope a flow (e.g., source, symbol, timeframe) |
| **Writer** | The single binary authorized to produce events for a stream |
| **Consumer** | A binary that reads from a stream with its own cursor |
| **Projection** | A materialized view derived from stream events |

### Mesh-Level Rules

These rules are transport-agnostic:

1. Every stream has exactly one writer.
2. Every event has a deterministic deduplication identity.
3. Families are isolated — no cross-family event dependencies within a single processing step.
4. Projections are disposable and rebuildable from events.
5. Query access never bypasses the projection layer.
6. New families require architectural approval.

## Transport Layer

### NATS Subject Encoding

The mesh is encoded into NATS subjects using this canonical pattern:

```
{family}.{surface}.{aggregate}.{verb}[.{partition_segments}]
```

**Examples:**

| Mesh Flow | NATS Subject |
|-----------|-------------|
| Observation trade event from binancef | `observation.events.market.trade.binancef` |
| Evidence candle sampled from binancef/btcusdt/60s | `evidence.events.candle.sampled.binancef.btcusdt.60` |
| Evidence candle latest query | `evidence.query.candle.latest` |
| Config activated event | `configctl.events.config.activated` |
| Config get-active query | `configctl.control.get_active` |

### Segment Mapping

| Segment Position | Mesh Dimension | Notes |
|-----------------|----------------|-------|
| 1 | family | `observation`, `evidence`, `signal`, `configctl` |
| 2 | surface | `events`, `control`, `query` |
| 3 | aggregate | `market`, `candle`, `tradeburst`, `config` |
| 4 | verb | `trade`, `sampled`, `latest`, `activated` |
| 5+ | partition key | `{source}`, `{source}.{symbol}`, `{source}.{symbol}.{timeframe}` |

### JetStream Stream Configuration

Each event-surface family maps to one JetStream stream:

| Stream Name | Subject Filter | Storage | MaxAge | MaxBytes |
|-------------|---------------|---------|--------|----------|
| `CONFIGCTL_EVENTS` | `configctl.events.config.>` | File | 24h | 256 MB |
| `OBSERVATION_EVENTS` | `observation.events.market.>` | File | 6h | 1 GB |
| `EVIDENCE_EVENTS` | `evidence.events.>` | File | 72h | 2 GB |
| `SIGNAL_EVENTS` (planned) | `signal.events.>` | File | 72h | TBD |
| `PROJECTION_EVENTS` (planned) | `projection.events.>` | File | 2h | TBD |

### KV Bucket Mapping

Projections are materialized into NATS KV buckets:

| Bucket Name | Family | Type | Key Format | Storage | MaxBytes | TTL |
|-------------|--------|------|------------|---------|----------|-----|
| `CANDLE_LATEST` | evidence | candle | `{source}.{symbol}.{timeframe}` | File | 64 MB | — |
| `CANDLE_HISTORY` | evidence | candle | `{source}.{symbol}.{timeframe}.{open_time_unix}` | File | 256 MB | 24h |
| `TRADE_BURST_LATEST` | evidence | tradeburst | `{source}.{symbol}.{timeframe}` | File | 64 MB | — |

### Consumer Specifications

| Durable Name | Stream | Filter Subject | Ack Wait | Max Deliver |
|-------------|--------|----------------|----------|-------------|
| `derive-observation` | OBSERVATION_EVENTS | `observation.events.market.trade.>` | 30s | 5 |
| `ingest-binding-watcher` | CONFIGCTL_EVENTS | `configctl.events.config.ingestion_runtime_changed` | 30s | 5 |
| `derive-binding-watcher` | CONFIGCTL_EVENTS | `configctl.events.config.ingestion_runtime_changed` | 30s | 5 |
| `store-candle` | EVIDENCE_EVENTS | `evidence.events.candle.sampled.>` | 30s | 5 |
| `store-trade-burst` | EVIDENCE_EVENTS | `evidence.events.tradeburst.sampled.>` | 30s | 5 |

### Envelope Type Encoding

Event types are encoded as versioned strings:

```
{family}.{surface}.{version}.{name}
```

| Mesh Event | Envelope Type |
|------------|--------------|
| Trade received | `observation.events.v1.trade_received` |
| Candle sampled | `evidence.events.v1.candle_sampled` |
| Trade burst sampled | `evidence.events.v1.trade_burst_sampled` |
| Config activated | `configctl.event.config.activated` |

### Queue Groups

Request/reply responders use queue groups for load distribution:

| Queue Group | Surface | Responder Binary |
|-------------|---------|-----------------|
| `configctl.control` | control | configctl |
| `evidence.query` | query | store |

## Mapping Discipline

### Transport Changes That Do NOT Affect the Mesh

- Changing stream retention from 72h to 168h
- Adding a new KV bucket for an existing evidence type
- Changing CBOR to Protobuf encoding
- Modifying consumer ack wait or max deliver
- Adding stream mirrors or replicas
- Changing storage from File to Memory

### Mesh Changes That REQUIRE Transport Updates

- Adding a new stream family (new JetStream stream)
- Adding a new evidence type (new subject pattern, consumer, KV bucket)
- Changing partition dimensions (subject segment changes)
- Splitting a stream (e.g., separating candle and signal events)
- Adding a new query surface (new request/reply subjects)

### Decision Framework

When making a change, ask:

1. **Does this change who writes or reads?** → Mesh change.
2. **Does this change what flows or its meaning?** → Mesh change.
3. **Does this change how it's delivered or stored?** → Transport change.
4. **Does this change the subject encoding?** → Both layers.

Mesh changes require architecture review. Transport changes require registry updates and raccoon-cli validation.
