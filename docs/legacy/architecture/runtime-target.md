# Runtime Target — Market Foundry

> Canonical document. Defines the minimal target topology of processes, their responsibilities, and phasing.
> Approved: 2026-03-16. This document governs which binaries exist and why.

---

## Design Constraints

1. Each binary has **one clear responsibility** expressible in a single sentence.
2. Binaries communicate **exclusively through NATS** (request/reply or JetStream).
3. No binary imports another binary's domain or application logic.
4. The topology must be **deployable locally** with `docker compose up`.
5. New binaries are added only when an existing binary cannot absorb the responsibility without violating its single-sentence purpose.

---

## Target Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                        Market Foundry                           │
│                                                                 │
│  ┌───────────┐   ┌────────────┐   ┌────────────┐               │
│  │  gateway   │   │ configctl  │   │  ingest    │  Phase 2      │
│  │  (HTTP)    │   │ (lifecycle)│   │  (capture) │               │
│  └─────┬─────┘   └─────┬──────┘   └─────┬──────┘               │
│        │               │                │                       │
│        │   ┌────────────┐   ┌────────────┐                      │
│        │   │  derive    │   │   store    │  Phase 3              │
│        │   │ (process)  │   │  (project) │                      │
│        │   └─────┬──────┘   └─────┬──────┘                      │
│        │         │                │                              │
│  ══════╪═════════╪════════════════╪══════════════════════════    │
│        │         │                │                              │
│  ┌─────▼─────────▼────────────────▼──────┐                      │
│  │          NATS + JetStream             │                      │
│  │          (message backbone)           │                      │
│  └───────────────────────────────────────┘                      │
│                                                                 │
│  ┌───────────────────────────────────────┐                      │
│  │  raccoon-cli (offline enforcement)    │                      │
│  └───────────────────────────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Binary Definitions

### gateway

| Attribute       | Value                                                        |
|-----------------|--------------------------------------------------------------|
| **Purpose**     | Expose Market Foundry capabilities over HTTP                 |
| **Sentence**    | Gateway translates HTTP requests into NATS operations and returns results |
| **Binary**      | `cmd/gateway`                                                |
| **Phase**       | 0 (exists today)                                             |
| **Owns**        | HTTP routes, readiness probes, request correlation           |
| **Does NOT own**| Domain logic, repositories, event publishing                 |

Gateway is a **thin translation layer**. It holds client-side use cases that encode requests, send them over NATS, and decode replies. It never executes domain logic directly.

As new domains are added, gateway acquires new route groups — but each route group delegates to the domain's control plane subjects. Gateway grows in surface area, not in depth.

**Actor topology:**
```
GatewayActor
└── HTTP listener (routes registered at startup)
```

---

### configctl

| Attribute       | Value                                                        |
|-----------------|--------------------------------------------------------------|
| **Purpose**     | Manage configuration document lifecycle                      |
| **Sentence**    | Configctl owns the full lifecycle of configuration documents from draft to activation |
| **Current name**| `cmd/configctl`                                              |
| **Target name** | `cmd/configctl` (unchanged)                                  |
| **Phase**       | 0 (exists today)                                             |
| **Owns**        | Config domain, repository, event publishing, control plane   |
| **Does NOT own**| HTTP exposure, market data, domain modules                   |

Configctl is the **control plane authority** for runtime configuration. It is the only binary that writes to the config repository and publishes config lifecycle events.

Other binaries consume configctl events (e.g., `IngestionRuntimeChanged`) to discover their runtime configuration — they never query the repository directly.

**Actor topology:**
```
ConfigSupervisor
├── EventRouterActor      (JetStream publisher)
├── ControlRouterActor    (use case dispatcher)
└── ControlResponderActor (NATS queue subscriber)
```

---

### ingest

| Attribute       | Value                                                        |
|-----------------|--------------------------------------------------------------|
| **Purpose**     | Capture and normalize external market data into canonical events |
| **Sentence**    | Ingest receives raw market data from external sources and publishes normalized observation events |
| **Target name** | `cmd/ingest`                                                 |
| **Phase**       | 2 (MarketMonkey absorption)                                  |
| **Owns**        | External connections, normalization, observation event publishing |
| **Does NOT own**| Signal derivation, strategy, storage, HTTP exposure          |

Ingest is the **boundary between the outside world and Market Foundry's internal event streams**. It subscribes to configctl's `IngestionRuntimeChanged` events to discover which data sources and bindings are active.

Raw data enters through protocol-specific adapters (WebSocket, REST polling, FIX — adapter choice per source). Ingest normalizes raw data into canonical observation events and publishes them to JetStream.

Ingest does NOT interpret, score, or derive signals from the data. It captures and normalizes — nothing more.

**Actor topology (anticipated):**
```
IngestSupervisor
├── BindingWatcherActor     (subscribes to IngestionRuntimeChanged)
├── SourceSupervisor[]      (one per active data source)
│   └── SourceAdapterActor  (protocol-specific capture)
└── ObservationPublisher    (JetStream: canonical observation events)
```

---

### derive

| Attribute       | Value                                                        |
|-----------------|--------------------------------------------------------------|
| **Purpose**     | Transform observation events into evidence and signals       |
| **Sentence**    | Derive consumes observation streams and produces evidence and signal events through configured processing pipelines |
| **Target name** | `cmd/derive`                                                 |
| **Phase**       | 3 (after ingest is operational)                              |
| **Owns**        | Evidence construction, signal computation, pipeline orchestration |
| **Does NOT own**| Raw data capture, strategy, execution, persistent storage    |

Derive is the **processing core**. It subscribes to observation event streams and applies domain-specific transformations to produce evidence (structured observations with provenance) and signals (actionable indicators derived from evidence).

Processing pipelines are configured through configctl — derive subscribes to config changes to discover which pipelines are active and what transformations to apply.

**Actor topology (anticipated):**
```
DeriveSupervisor
├── PipelineWatcherActor     (subscribes to config changes)
├── PipelineSupervisor[]     (one per active pipeline)
│   ├── EvidenceBuilderActor (observation → evidence)
│   └── SignalComputeActor   (evidence → signal)
└── DerivePublisher          (JetStream: evidence + signal events)
```

---

### store

| Attribute       | Value                                                        |
|-----------------|--------------------------------------------------------------|
| **Purpose**     | Maintain queryable projections from event streams            |
| **Sentence**    | Store consumes domain events and builds read-optimized projections for query access |
| **Target name** | `cmd/store`                                                  |
| **Phase**       | 3 (after derive is operational)                              |
| **Owns**        | Projection building, query serving, materialized views       |
| **Does NOT own**| Event production, domain logic, external data capture        |

Store is the **read model authority**. It subscribes to JetStream event streams (observations, evidence, signals) and builds materialized projections optimized for query access.

Store exposes its projections through NATS request/reply (consumed by gateway for HTTP exposure). It is a **CQRS read side** — it never produces domain events, only consumes them.

**Actor topology (anticipated):**
```
StoreSupervisor
├── ProjectionWatcherActor    (subscribes to config for projection definitions)
├── ProjectionBuilderActor[]  (one per active projection)
│   └── StreamConsumerActor   (JetStream consumer for source events)
└── QueryResponderActor       (NATS request/reply for projection queries)
```

---

## Phase Map

| Phase | Binary     | Status       | Trigger                                  |
|-------|------------|--------------|------------------------------------------|
| 0     | gateway    | **Exists**   | Operational as `cmd/gateway`             |
| 0     | configctl  | **Exists**   | Already operational                      |
| 0     | nats       | **Exists**   | Infrastructure, always present           |
| 2     | ingest     | **Exists**   | Operational (S12)                        |
| 3     | derive     | **Exists**   | Operational (S12)                        |
| 3     | store      | **Exists**   | Operational (S13)                        |

**Phase 1** (current) produces no new binaries — it defines the canonical vision and taxonomy that Phase 2 builds upon.

---

## What This Topology Does NOT Include

| Excluded element      | Reason                                                           |
|-----------------------|------------------------------------------------------------------|
| `consumer` binary     | Legacy Kafka→NATS bridge; ingest replaces this role natively     |
| `emulator` binary     | Synthetic data generation is a test concern, not a runtime binary|
| `validator` binary    | Quality-domain artifact; validation is per-domain, not centralized|
| `scheduler` binary    | Premature; scheduling is a strategy/execution concern (Phase 4+) |
| `portfolio` binary    | Premature; requires execution domain operational first           |
| `admin` binary        | Gateway handles all administrative HTTP exposure                 |

---

## Deployment Topology (Docker Compose Target)

### Phase 0 (current)

```yaml
services:
  nats:       # Message backbone
  configctl:  # Config lifecycle
  gateway:    # HTTP API
```

### Phase 2 (after MarketMonkey absorption)

```yaml
services:
  nats:       # Message backbone
  configctl:  # Config lifecycle
  gateway:    # HTTP API
  ingest:     # Market data capture
```

### Phase 3 (processing + projections)

```yaml
services:
  nats:       # Message backbone
  configctl:  # Config lifecycle
  gateway:    # HTTP API
  ingest:     # Market data capture
  derive:     # Evidence + signal processing
  store:      # Queryable projections
```

---

## Inter-Binary Communication Matrix

| From → To     | Mechanism               | Purpose                              |
|----------------|------------------------|--------------------------------------|
| gateway → configctl | NATS request/reply  | Config CRUD operations               |
| gateway → store    | NATS request/reply   | Projection queries                   |
| configctl → *      | JetStream publish    | Config lifecycle events              |
| ingest → *         | JetStream publish    | Observation events                   |
| ingest ← configctl | JetStream subscribe  | Ingestion binding discovery          |
| derive ← ingest    | JetStream subscribe  | Observation event consumption        |
| derive → *         | JetStream publish    | Evidence + signal events             |
| derive ← configctl | JetStream subscribe  | Pipeline config discovery            |
| store ← *          | JetStream subscribe  | All domain events for projection     |
| store → gateway    | NATS request/reply   | Projection query responses           |

**Rule:** Arrows indicate message direction. No binary calls another binary's internal APIs. All communication flows through NATS subjects defined in the stream taxonomy.

---

## Scaling Model

Each binary scales independently:

- **gateway**: Horizontal (stateless HTTP translation)
- **configctl**: Single-writer preferred (repository consistency); read replicas via NATS queue groups
- **ingest**: Horizontal per data source (one SourceAdapter per external feed)
- **derive**: Horizontal per pipeline (one PipelineSupervisor per processing chain)
- **store**: Horizontal per projection type (one ProjectionBuilder per materialized view)

NATS queue groups ensure load balancing for request/reply patterns. JetStream consumer groups ensure ordered processing for event streams.

---

## Invariants

1. **Five is the ceiling.** No more than five domain binaries (gateway, configctl, ingest, derive, store) unless a new binary can justify its existence with a single sentence that no existing binary can absorb.
2. **Gateway is the only HTTP surface.** No other binary exposes HTTP endpoints (health checks excluded via NATS-based liveness).
3. **Configctl is the only config authority.** No binary maintains its own configuration lifecycle outside configctl.
4. **Every binary has a supervisor root.** The top-level actor is always a supervisor that owns all child actors.
5. **No binary-to-binary RPC chains.** A request from gateway may fan out to one domain binary — never to a chain of binaries synchronously.
