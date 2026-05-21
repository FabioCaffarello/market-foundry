# Stream Mesh Model

> Canonical architectural reference for the Market Foundry stream mesh.

## Definition

The **stream mesh** is the complete topology of named, typed, ownership-bound message flows that constitute the Market Foundry runtime. It is the system's nervous system — every domain interaction, every state transition, and every query traverses the mesh.

A stream mesh is **not** a transport configuration. It is a **logical architecture** that happens to be encoded onto NATS subjects and JetStream streams. The mesh exists independently of its transport — if NATS were replaced, the mesh model would remain valid.

## Conceptual Layers

The mesh has three conceptual layers, each with distinct responsibilities:

```
┌─────────────────────────────────────────────────┐
│  Layer 3 — Query Surface                        │
│  Synchronous read access to materialized state  │
│  Pattern: request → reply                       │
├─────────────────────────────────────────────────┤
│  Layer 2 — Projection Surface                   │
│  Materialized views and KV buckets              │
│  Pattern: consume → project → store             │
├─────────────────────────────────────────────────┤
│  Layer 1 — Event Surface                        │
│  Asynchronous domain events via streams         │
│  Pattern: produce → stream → consume            │
├─────────────────────────────────────────────────┤
│  Layer 0 — Control Surface                      │
│  Commands, lifecycle, configuration              │
│  Pattern: request → handle → reply              │
└─────────────────────────────────────────────────┘
```

### Layer 0 — Control Surface

Commands and queries that govern system configuration and lifecycle. Synchronous request/reply. Does not carry market data. Owned by configctl.

### Layer 1 — Event Surface

The primary data plane. Domain events flow through JetStream streams with at-least-once delivery, ordered per partition key, and deduplication by message ID. This is where observation, evidence, and signal events live.

### Layer 2 — Projection Surface

Materialized read models derived from Layer 1 events. KV buckets hold latest and historical state. Single-writer invariant per bucket. Projections are disposable — they can be rebuilt from events.

### Layer 3 — Query Surface

Synchronous access to projections. Gateway translates HTTP requests into NATS request/reply calls served by store's query responders. Query subjects are logically separate from event subjects even when they concern the same domain.

## Mesh Dimensions

Every flow in the mesh is addressable by a combination of these dimensions:

| Dimension | Description | Examples |
|-----------|-------------|----------|
| **family** | The domain category of the flow | `observation`, `evidence`, `signal`, `configctl` |
| **surface** | Which conceptual layer | `events`, `control`, `query` |
| **aggregate** | The domain entity or type | `trade`, `candle`, `tradeburst`, `config` |
| **verb** | The action or state transition | `sampled`, `activated`, `latest`, `history` |
| **source** | The origin exchange or data provider | `binancef`, `binance`, `bybit` |
| **symbol** | The trading instrument | `btcusdt`, `ethusdt` |
| **timeframe** | The sampling window in seconds | `60`, `300`, `900` |

Not every dimension applies to every flow. Observation events are partitioned by `source` only. Evidence events are partitioned by `source`, `symbol`, and `timeframe`. Configuration events have no market-data dimensions.

## Mesh Properties

### 1. Single-Writer Streams

Every JetStream stream has exactly one producing binary. No stream accepts writes from multiple services. This is an inviolable invariant.

| Stream | Writer | Rationale |
|--------|--------|-----------|
| CONFIGCTL_EVENTS | configctl | Configuration lifecycle is a single authority |
| OBSERVATION_EVENTS | ingest | Raw market data enters through one gateway per exchange |
| EVIDENCE_EVENTS | derive | Derived facts are produced by the derivation pipeline |
| SIGNAL_EVENTS | derive (future) | Signals derive from evidence, same pipeline |
| PROJECTION_EVENTS | store (future) | Materialization notifications from the read side |

### 2. Fan-Out Consumption

Multiple consumers may read from the same stream. Each consumer has its own durable name, filter subject, and ack state. The mesh supports independent consumption rates per consumer.

### 3. Partition-Aligned Isolation

Actor trees are structured to mirror mesh partitioning. Ingest spawns one scope actor per source. Derive spawns one scope actor per source, with sampler actors per symbol/timeframe. This alignment ensures that a failure in one partition does not affect others.

### 4. Deduplication by Design

Every event carries a deterministic message ID derived from its content dimensions. JetStream enforces deduplication at the stream level. Projections add a second layer of idempotency via monotonicity guards.

### 5. Envelope Uniformity

All messages — events, commands, queries, and replies — are wrapped in `Envelope[T]` with consistent metadata: Kind, Type, Source, Subject, CorrelationID. The envelope is the unit of observability and tracing.

## Stream Lifecycle States

Streams in the mesh follow a progression:

```
Planned → Defined → Implemented → Active → (Deprecated → Removed)
```

- **Planned**: documented in architecture, no code.
- **Defined**: registry spec exists, stream/consumer constants declared.
- **Implemented**: producer and consumer actors exist, events flow in dev.
- **Active**: running in production, covered by smoke tests.
- **Deprecated**: consumers still active, producer migrating.
- **Removed**: all references deleted, stream no longer exists.

## Mesh Evolution Rules

1. **New stream families require an architecture document** before any code. The document must specify: family name, surface, aggregate, ownership, subject encoding, retention, and consumer list.

2. **New evidence types follow the evidence-read-model-guidelines checklist**. No new evidence type enters the mesh without completing all 30+ items.

3. **Subject encoding is additive**. Existing subject patterns must not change once consumers exist. New dimensions are added as suffix segments.

4. **Retention policies are per-family, not per-type**. All evidence events share EVIDENCE_EVENTS retention. Individual types do not get custom retention within a shared stream.

5. **Query subjects are never derived from event subjects**. They are independently named and independently versioned. The query surface is a separate concern from the event surface.

## Relationship to Bounded Contexts

Each stream family aligns with a bounded context, but the relationship is not 1:1:

| Stream Family | Bounded Context | Binary |
|---------------|----------------|--------|
| configctl | Configuration | configctl (embedded in all binaries) |
| observation | Market Data Ingestion | ingest |
| evidence | Market Evidence | derive (write), store (read) |
| signal | Trading Signals | derive (write, future), store (read, future) |
| projection | Read Model Notifications | store (write, future) |

The **derive** binary is the primary event producer for both evidence and signal families. This is intentional — derivation is a single pipeline concern, not split by output family. The **store** binary is the universal read-side authority.

## What the Mesh Is Not

- **Not a service mesh**: there are no proxies, sidecars, or service-to-service routing concerns. The mesh is about data flows, not service discovery.
- **Not a message bus**: the mesh has typed, versioned contracts per flow. It is not a generic pub/sub backbone.
- **Not transport configuration**: subject strings and JetStream settings are the _encoding_ of the mesh, not the mesh itself. See [mesh-vs-transport.md](mesh-vs-transport.md) for the distinction.
