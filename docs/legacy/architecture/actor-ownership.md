# Actor Ownership — Market Foundry

> Canonical document. Defines the relationship between actors, streams, and projections across all binaries.
> Approved: 2026-03-16. Every actor in Market Foundry must conform to this ownership model.

---

## Ownership Principles

1. **Every stream has exactly one producer.** No stream is written to by multiple binaries.
2. **Every actor has exactly one supervisor.** No actor is spawned outside a supervision tree.
3. **Actors own their state.** No shared mutable state between actors — communication is via messages only.
4. **Supervisors own their children's lifecycle.** Spawning, restarting, and stopping is the supervisor's responsibility.
5. **Ownership is hierarchical and auditable.** Given any actor, you can trace its supervision chain to a binary's root supervisor.

---

## Ownership Notation

```
BinarySupervisor            — root actor, spawned by cmd/{binary}/run.go
├── ChildActor              — spawned by parent via SpawnChild
│   └── GrandchildActor     — spawned by ChildActor
└── AnotherChildActor
    └── ...
```

| Symbol | Meaning |
|--------|---------|
| `→ produces` | Actor publishes to a JetStream stream |
| `← consumes` | Actor subscribes to a JetStream stream |
| `⇄ serves`   | Actor handles NATS request/reply |
| `○ owns`      | Actor holds exclusive access to a resource (repo, connection) |

---

## Phase 0 — Current Ownership

### gateway (`cmd/gateway`)

```
GatewayActor
○ owns: HTTP listener, route table
○ owns: NATSRequestClient (control plane connection)
⇄ serves: HTTP → NATS translation

    Routes:
      POST /configs/draft          → configctl.control.config.create_draft
      POST /configs/validate       → configctl.control.config.validate_draft
      POST /configs/{id}/validate  → configctl.control.config.validate
      POST /configs/{id}/compile   → configctl.control.config.compile
      POST /configs/{id}/activate  → configctl.control.config.activate
      GET  /configs/{id}           → configctl.control.config.get
      GET  /configs/active         → configctl.control.config.get_active
      GET  /configs                → configctl.control.config.list
      GET  /healthz                → local (always 200)
      GET  /readyz                 → local (checks NATS + configctl)
```

**Ownership boundaries:**
- Gateway does NOT own any domain logic.
- Gateway does NOT subscribe to JetStream streams.
- Gateway does NOT maintain a repository.
- Gateway is a **stateless translator** — its only state is the HTTP listener and NATS connection.

---

### configctl (`cmd/configctl`)

```
ConfigSupervisor
○ owns: lifecycle of all child actors
│
├── EventRouterActor
│   ○ owns: DomainEventPublisher (JetStream connection)
│   → produces: CONFIGCTL_EVENTS stream
│   │   configctl.events.config.draft_created
│   │   configctl.events.config.validated
│   │   configctl.events.config.compiled
│   │   configctl.events.config.activated
│   │   configctl.events.config.deactivated
│   │   configctl.events.config.ingestion_runtime_changed
│   │   configctl.events.config.archived
│   │   configctl.events.config.rejected
│   │
│   Receives: publishDomainEventMessage
│   Returns:  publishDomainEventResult
│
├── ControlRouterActor
│   ○ owns: Repository (in-memory configctl store)
│   ○ owns: all 10 use case instances (lazy-initialized)
│   │
│   Receives: createDraftMessage, validateDraftMessage,
│             compileConfigMessage, activateConfigMessage,
│             getConfigMessage, getActiveConfigMessage,
│             listConfigsMessage, listActiveProjectionsMessage,
│             listActiveBindingsMessage, validateConfigMessage
│   Returns:  corresponding *Result messages
│   │
│   Dispatches domain events to: EventRouterActor (via engine.Request)
│
└── ControlResponderActor
    ○ owns: RequestReplyResponder (NATS queue subscription)
    ⇄ serves: configctl.control.config.* (queue group: configctl.control)
    │
    Translates: NATS messages → typed actor messages
    Forwards to: ControlRouterActor (via engine.Request with timeout)
    Returns: CBOR-encoded reply to NATS respondent
```

**Ownership boundaries:**
- ConfigSupervisor is the **sole writer** to CONFIGCTL_EVENTS stream.
- ControlRouterActor is the **sole accessor** of the config repository.
- No other binary or actor may write to configctl subjects.
- The repository is encapsulated within the actor — no external reference exists.

---

## Phase 2 — Ingest Ownership (implemented S12)

### ingest (`cmd/ingest`)

```
IngestSupervisor
○ owns: lifecycle of all child actors
│
├── BindingWatcherActor
│   ← consumes: CONFIGCTL_EVENTS stream
│   │   Filters: configctl.events.config.ingestion_runtime_changed
│   │
│   Responsibility:
│     - Queries configctl for active bindings on startup
│     - Subscribes to IngestionRuntimeChangedEvent for dynamic updates
│     - Sends activateBindingMessage / clearBindingMessage to supervisor
│
└── ExchangeScopeActor[] (one per exchange/source, e.g., source-binancef)
    ○ owns: lifecycle of all actors for this exchange
    │
    ├── PublisherActor
    │   ○ owns: JetStream connection for observation events (per source)
    │   → produces: OBSERVATION_EVENTS stream
    │   │   observation.events.market.trade.{source}
    │   │
    │   Receives: publishTradeMessage from WebSocket adapters
    │
    └── WebSocketAdapterActor[] (one per symbol, e.g., ws-btcusdt)
        ○ owns: WebSocket connection to exchange
        │
        Responsibility:
          - Connects to exchange aggTrade stream for one symbol
          - Normalizes raw data into observation.TradeReceivedEvent
          - Forwards to PublisherActor within the same exchange scope
```

**Ownership boundaries:**
- IngestSupervisor is the **sole writer** to OBSERVATION_EVENTS stream.
- BindingWatcherActor is a **read-only consumer** of CONFIGCTL_EVENTS.
- ExchangeScopeActor owns all actors for one exchange — failure isolation by source.
- PublisherActor is scoped per exchange (one NATS connection per exchange).
- No other binary may publish observation events.

**Key design decisions:**
- **One ExchangeScopeActor per source** — if binance dies, future exchanges are unaffected.
- **Publisher per exchange scope** — each exchange gets its own NATS connection for isolation.
- **Lazy scope creation** — exchange scope created on first binding activation, not eagerly.
- **BindingWatcherActor is event-driven** — queries configctl on startup, then reacts to events.

---

## Phase 3 — Derive Ownership (implemented S12, updated S28/S31)

### derive (`cmd/derive`)

```
DeriveSupervisor
○ owns: lifecycle of all child actors
○ owns: FamilyProcessor registry (candle, tradeburst, volume)
│
├── BindingWatcherActor
│   ← consumes: CONFIGCTL_EVENTS stream
│   │   Filters: configctl.events.config.ingestion_runtime_changed
│   │
│   Responsibility:
│     - Queries configctl for active bindings on startup
│     - Subscribes to IngestionRuntimeChangedEvent for dynamic updates
│     - Sends activateSamplerMessage / clearBindingMessage to supervisor
│
├── ConsumerActor
│   ← consumes: OBSERVATION_EVENTS stream
│   │   Durable: derive-observation
│   │   Filters: observation.events.market.trade.>
│   │
│   Receives: observation events from NATS
│   Forwards: tradeReceivedMessage to supervisor for routing
│
└── SourceScopeActor[] (one per source, e.g., source-binancef)
    ○ owns: lifecycle of all actors for this source
    │
    ├── EvidencePublisherActor
    │   ○ owns: JetStream connection for evidence events (per source)
    │   → produces: EVIDENCE_EVENTS stream
    │   │   evidence.events.candle.sampled.{source}.{symbol}.{timeframe}
    │   │   evidence.events.tradeburst.sampled.{source}.{symbol}.{timeframe}
    │   │   evidence.events.volume.sampled.{source}.{symbol}.{timeframe}
    │   │
    │   Receives: publishCandleMessage, publishTradeBurstMessage, publishVolumeMessage
    │
    ├── SamplerActor[] (one per symbol × timeframe, e.g., sampler-btcusdt-60s)
    │   ○ owns: CandleSampler (pure application logic)
    │   Samples trades into OHLCV candles
    │
    ├── TradeBurstSamplerActor[] (one per symbol × timeframe)
    │   ○ owns: TradeBurstSampler (pure application logic)
    │   Samples trades into burst activity metrics
    │
    └── VolumeSamplerActor[] (one per symbol × timeframe)
        ○ owns: VolumeSampler (pure application logic)
        Samples trades into volume profiles (buy/sell volume, VWAP)
```

**Ownership boundaries:**
- DeriveSupervisor is the **sole writer** to EVIDENCE_EVENTS stream.
- SourceScopeActor owns all actors for one source — failure isolation by source.
- EvidencePublisherActor is scoped per source (one NATS connection per source).
- FamilyProcessor registry enables zero-change spawning when new evidence types are added.

**Key design decisions:**
- **FamilyProcessor pattern (S28)** — supervisor registers processor entries; SourceScopeActor iterates without hardcoded references.
- **Trade routing through hierarchy** — supervisor → source scope → all samplers for that symbol.
- **Publisher per source scope** — each source gets its own NATS connection for isolation.
- **Lazy scope creation** — source scope created on first binding activation.
- **Multi-family fan-out** — each trade feeds candle + tradeburst + volume samplers simultaneously.

---

## Phase 3 — Store Ownership (implemented S13, updated S29/S31)

### store (`cmd/store`)

```
StoreSupervisor
○ owns: lifecycle of all child actors
○ owns: ProjectionPipeline registry (candle, tradeburst, volume)
│
├── CandleProjectionActor
│   ○ owns: NATS KV connection (write path)
│   │   KV Buckets: CANDLE_LATEST (64 MB), CANDLE_HISTORY (256 MB, 24h TTL)
│   │
│   Receives: candleReceivedMessage from CandleConsumerActor
│   Materializes: latest and historical candles per source/symbol/timeframe
│
├── CandleConsumerActor (EvidenceConsumerActor)
│   ← consumes: EVIDENCE_EVENTS stream
│   │   Durable: store-candle
│   │   Filters: evidence.events.candle.sampled.>
│   │
│   Forwards: candleReceivedMessage to CandleProjectionActor
│
├── TradeBurstProjectionActor
│   ○ owns: NATS KV connection (write path)
│   │   KV Bucket: TRADE_BURST_LATEST (64 MB)
│   │
│   Receives: tradeBurstReceivedMessage from TradeBurstConsumerActor
│   Materializes: latest trade burst per source/symbol/timeframe
│
├── TradeBurstConsumerActor
│   ← consumes: EVIDENCE_EVENTS stream
│   │   Durable: store-trade-burst
│   │   Filters: evidence.events.tradeburst.sampled.>
│   │
│   Forwards: tradeBurstReceivedMessage to TradeBurstProjectionActor
│
├── VolumeProjectionActor
│   ○ owns: NATS KV connection (write path)
│   │   KV Bucket: VOLUME_LATEST (64 MB)
│   │
│   Receives: volumeReceivedMessage from VolumeConsumerActor
│   Materializes: latest volume profile per source/symbol/timeframe
│
├── VolumeConsumerActor
│   ← consumes: EVIDENCE_EVENTS stream
│   │   Durable: store-volume
│   │   Filters: evidence.events.volume.sampled.>
│   │
│   Forwards: volumeReceivedMessage to VolumeProjectionActor
│
└── QueryResponderActor
    ○ owns: NATS KV connections (read path) for all buckets + RequestReplyResponder
    ⇄ serves: evidence.query.candle.latest (queue group: evidence.query)
    ⇄ serves: evidence.query.candle.history
    ⇄ serves: evidence.query.tradeburst.latest
    ⇄ serves: evidence.query.volume.latest
    │
    Responsibility:
      - Receives evidence queries from gateway via NATS request/reply
      - Reads from all KV buckets (CANDLE_LATEST, CANDLE_HISTORY, TRADE_BURST_LATEST, VOLUME_LATEST)
      - Returns latest/historical evidence (no dependency on derive actors)
```

**Ownership boundaries:**
- StoreSupervisor is the **sole server** for `evidence.query.*` subjects.
- Store is a **read-only consumer** of EVIDENCE_EVENTS stream.
- Derive is now write-only: it publishes evidence events but does not serve queries.
- Persistence is via NATS KV (JetStream FileStorage) — survives restarts.
- Store never produces canonical domain events — only materializes read models.
- ProjectionPipeline registry enables zero-change spawning when new evidence types are added.

**Known limitation:** QueryResponderActor scales manually — each new evidence type adds KV store fields, route registrations, and handler methods. Manageable at 3 types; consider splitting at 5+.

---

## Cross-Binary Stream Ownership Matrix

| Stream               | Producer binary | Consumer binaries          | Phase | Status  |
|----------------------|-----------------|----------------------------|-------|---------|
| CONFIGCTL_EVENTS     | configctl       | ingest, derive             | 0     | Active  |
| OBSERVATION_EVENTS   | ingest          | derive                     | 2     | Active  |
| EVIDENCE_EVENTS      | derive          | store                      | 3     | Active  |
| SIGNAL_EVENTS        | derive (future) | store (future)             | —     | Planned |
| PROJECTION_EVENTS    | store (future)  | gateway (future)           | —     | Planned |

**Invariant:** Every stream has exactly one producer binary. This is non-negotiable.

**Note:** SIGNAL_EVENTS and PROJECTION_EVENTS are naming reservations only. No code may reference these streams until their respective domains are formally approved for entry. See `signal-readiness-review.md` for prerequisites.

---

## Control Plane Ownership Matrix

| Subject pattern                        | Server binary | Client binary | Phase   | Status  |
|----------------------------------------|---------------|---------------|---------|---------|
| `configctl.control.config.*`           | configctl     | gateway       | 0       | Active  |
| `evidence.query.candle.latest`         | store         | gateway       | 3 (S13) | Active  |
| `evidence.query.candle.history`        | store         | gateway       | 3 (S20) | Active  |
| `evidence.query.tradeburst.latest`     | store         | gateway       | 3 (S24) | Active  |
| `evidence.query.volume.latest`         | store         | gateway       | 3 (S31) | Active  |
| `signal.query.*`                       | store         | gateway       | future  | Planned |

**Invariant:** Every control/query subject has exactly one server binary.

---

## Actor ↔ Stream ↔ Projection Relationship

```
Actor produces events → JetStream stream stores events → Actor consumes events → Projection built

Concrete example (Phase 3 steady state):

IngestSupervisor
  └── ObservationPublisher
        → OBSERVATION_EVENTS stream
              ← DeriveSupervisor.PipelineActor.ObservationConsumerActor
                    → EvidenceBuilderActor
                          → EVIDENCE_EVENTS stream
                                ← StoreSupervisor.ProjectionBuilderActor
                                      ○ builds: ticker_series projection
                                      → PROJECTION_EVENTS (notification)
                                            ← GatewayActor (cache invalidation)
```

**The flow is always:**
1. **Producer actor** publishes to a JetStream stream
2. **Stream** persists the event for configured retention
3. **Consumer actor** in another binary reads from the stream
4. Consumer actor processes and may produce to a different stream
5. **Projection actor** in store reads from streams and builds read models
6. **Query actor** in store serves projections via request/reply

No actor both produces to and consumes from the same stream. Feedback loops are prohibited.

---

## Supervision Strategy

All supervisors follow the same restart policy:

| Event                    | Response                                    |
|--------------------------|---------------------------------------------|
| Child actor panic        | Restart child, log error, increment counter |
| Child restart threshold  | Stop child, escalate to parent supervisor   |
| NATS disconnect          | Buffer messages locally, reconnect          |
| NATS reconnect           | Resume from last acknowledged position      |
| Graceful shutdown signal | Stop children in reverse spawn order        |

**Restart budget:** Configurable per supervisor. Default: 5 restarts per 60 seconds before escalation.

---

## Naming Convention for Actors

```
{Domain}{Role}Actor
```

| Component | Convention          | Examples                                   |
|-----------|---------------------|--------------------------------------------|
| Domain    | PascalCase domain   | Config, Observation, Evidence, Signal      |
| Role      | Describes function  | Supervisor, Router, Responder, Publisher, Consumer, Builder, Watcher |

**Examples:**
- `ConfigSupervisor` (not `ConfigctlSupervisorActor`)
- `ObservationPublisher` (not `MarketDataJetStreamPublisher`)
- `EvidenceBuilderActor` (not `EvidenceProcessorTransformerActor`)

Keep names **short and role-descriptive**. The domain prefix tells you *what*, the role suffix tells you *how*.

---

## Open Decisions

These aspects of actor ownership require validation during implementation:

| ID   | Decision                                          | Options                                       | Impact    | Status |
|------|---------------------------------------------------|-----------------------------------------------|-----------|--------|
| AO-1 | Should derive have separate binaries for evidence and signal? | Single `derive` binary vs `evidence` + `signal` binaries | Binary count, deployment complexity | Open |
| AO-2 | ~~Should store use persistent storage or event replay?~~ **Resolved S13: NATS KV (FileStorage)** | Persistent store via NATS KV | Startup time, storage requirements | Resolved |
| AO-3 | How should pipeline-specific actor trees be versioned during config changes? | Graceful drain + respawn vs hot-swap | Processing continuity, complexity | Open |
| AO-4 | ~~Should ObservationPublisher be per-source or centralized?~~ **Resolved S12: per-source** | One publisher per exchange/source scope | Throughput, connection count | Resolved |
| AO-5 | Should gateway subscribe to PROJECTION_EVENTS for cache invalidation? | Push invalidation vs TTL-based cache vs no cache | Latency, complexity | Open |
