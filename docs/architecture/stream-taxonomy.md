# Stream Taxonomy — Market Foundry

> Canonical document. Defines the naming conventions, stream classes, and subject hierarchy for all NATS messaging.
> Approved: 2026-03-16. All current and future NATS subjects must conform to this taxonomy.

---

## Design Principles

1. **A subject name is a contract.** Changing a subject name is a breaking change.
2. **Subjects are self-describing.** Reading a subject tells you the domain, the plane, the aggregate, and the event type without consulting documentation.
3. **Streams group related subjects.** A JetStream stream captures a class of events for replay, not individual subjects.
4. **Partition keys are explicit.** When ordering matters, the subject encodes the partition key.
5. **No inherited naming.** Subject names are designed for Market Foundry — not carried from quality-service or MarketMonkey.

---

## Subject Naming Convention

```
{domain}.{plane}.{aggregate}.{verb_or_noun}[.{key}]
```

| Segment       | Purpose                          | Examples                              |
|---------------|----------------------------------|---------------------------------------|
| `domain`      | Owning domain module             | `configctl`, `observation`, `evidence`, `signal` |
| `plane`       | Communication class              | `control`, `events`, `query`          |
| `aggregate`   | Domain aggregate or entity       | `config`, `source`, `ticker`, `pipeline` |
| `verb_or_noun`| Action (control) or event name   | `create_draft`, `draft_created`, `by_scope` |
| `key`         | Optional partition/routing key   | `{sourceID}`, `{scope}`, `{pipelineID}` |

### Segment Rules

- **All lowercase**, words separated by underscores within a segment.
- **Dots separate hierarchy levels** — never use dots within a segment.
- **Domain segment matches the Go module name** under `internal/domain/`.
- **Plane is always one of three values**: `control`, `events`, `query`.
- **No versioning in subjects.** Versioning is handled by envelope metadata (`Type` field), not by subject hierarchy.

---

## Communication Planes

### control — Synchronous Command/Query

```
{domain}.control.{aggregate}.{operation}
```

Request/reply over NATS core. Used for commands (state-changing) and queries (read-only) where the caller needs an immediate response.

- **Transport**: NATS request/reply
- **Encoding**: CBOR (Envelope[T])
- **Load balancing**: Queue groups (`{domain}.control`)
- **Timeout**: Configured per service (default: 5s)

**Existing subjects:**
```
configctl.control.config.create_draft
configctl.control.config.validate_draft
configctl.control.config.validate
configctl.control.config.compile
configctl.control.config.activate
configctl.control.config.get
configctl.control.config.get_active
configctl.control.config.list
configctl.control.config.list_active_projections
configctl.control.config.list_active_bindings
```

### events — Asynchronous Domain Events

```
{domain}.events.{aggregate}.{event_name}[.{key}]
```

Published to JetStream. Used for domain events that other binaries or domains may consume asynchronously.

- **Transport**: NATS JetStream publish
- **Encoding**: CBOR (Envelope[T])
- **Deduplication**: Message ID = event metadata ID
- **Retention**: Stream-specific (see Stream Definitions below)

**Existing subjects:**
```
configctl.events.config.draft_created
configctl.events.config.validated
configctl.events.config.compiled
configctl.events.config.activated
configctl.events.config.deactivated
configctl.events.config.ingestion_runtime_changed
configctl.events.config.archived
configctl.events.config.rejected
```

### query — Synchronous Read Model Access

```
{domain}.query.{projection}.{operation}[.{key}]
```

Request/reply over NATS core. Used by the `store` binary to serve materialized projections. Separated from `control` because queries against projections have different ownership (store owns them, not the domain binary).

- **Transport**: NATS request/reply
- **Encoding**: CBOR (Envelope[T])
- **Load balancing**: Queue groups (`{domain}.query`)
- **Timeout**: Configured per service (default: 5s)

**Future subjects (Phase 3):**
```
observation.query.latest.by_source
observation.query.latest.by_ticker
evidence.query.series.by_ticker
signal.query.active.by_pipeline
signal.query.history.by_ticker
```

---

## Event Classes

Every domain event belongs to exactly one class. The class determines how the event is produced, consumed, and retained.

### Input Event

**Definition:** Raw data captured from an external source, normalized into a canonical schema.

```
observation.events.market.{event_name}.{sourceID}
```

- **Producer**: `ingest` binary only
- **Consumers**: `derive` binary, `store` binary
- **Retention**: Short-lived (hours). Input events are transient — their value is consumed by downstream processing.
- **Ordering**: Per source ID (events from the same source must be processed in order)
- **Idempotency**: Source-assigned deduplication key (external event ID + source ID)

**Examples:**
```
observation.events.market.price_received.binance
observation.events.market.orderbook_snapshot.coinbase
observation.events.market.trade_executed.kraken
```

**What input events are NOT:** Input events are not raw wire-format data. They have been normalized by ingest into the observation domain's canonical types. The raw wire format never enters JetStream.

---

### Canonical Event

**Definition:** A domain state change within Market Foundry, produced by a domain binary as the result of processing.

```
{domain}.events.{aggregate}.{event_name}
```

- **Producer**: The owning domain binary (configctl, derive)
- **Consumers**: Any binary that needs to react to domain state changes
- **Retention**: Long-lived (days to weeks). Canonical events are the system of record for what happened.
- **Ordering**: Per aggregate (events for the same aggregate must be processed in order)
- **Idempotency**: Event metadata ID (ULID, assigned by producer)

**Examples:**
```
configctl.events.config.activated
evidence.events.record.created
evidence.events.record.enriched
signal.events.indicator.computed
signal.events.indicator.expired
```

**What canonical events are NOT:** They are not projections or read models. They represent facts about domain state transitions, not precomputed views.

---

### Projection Event

**Definition:** A notification that a materialized projection has been updated, enabling downstream invalidation or refresh.

```
{domain}.events.projection.{projection_name}.updated[.{key}]
```

- **Producer**: `store` binary only
- **Consumers**: `gateway` (for cache invalidation), other projections (for cascading updates)
- **Retention**: Ephemeral (minutes to hours). Projection events are notifications, not facts.
- **Ordering**: Not guaranteed (projections are eventually consistent)
- **Idempotency**: Projection version number (monotonically increasing)

**Examples:**
```
observation.events.projection.latest_prices.updated
evidence.events.projection.ticker_series.updated.BTCUSD
signal.events.projection.active_signals.updated
```

**What projection events are NOT:** They do not carry the projection data itself. They signal that the projection has changed — consumers query the store for current state.

---

### Lifecycle Event

**Definition:** A configctl-specific event indicating a change in runtime configuration that other binaries must react to.

```
configctl.events.config.{lifecycle_event}
```

- **Producer**: `configctl` binary only
- **Consumers**: `ingest` (binding changes), `derive` (pipeline changes), `store` (projection definitions)
- **Retention**: Long-lived (same as canonical events — config changes are system of record)
- **Ordering**: Per config set (activations/deactivations for the same config must be ordered)
- **Idempotency**: Event metadata ID

**Existing examples:**
```
configctl.events.config.activated
configctl.events.config.deactivated
configctl.events.config.ingestion_runtime_changed
```

**Note:** Lifecycle events are a subset of canonical events. They are called out separately because they serve a cross-cutting coordination role — they are the mechanism by which configctl drives runtime behavior in other binaries.

---

## Stream Definitions

Each JetStream stream groups related subjects with shared retention and storage policies.

### CONFIGCTL_EVENTS (exists)

```
Stream:    CONFIGCTL_EVENTS
Subjects:  configctl.events.config.>
Storage:   File
MaxAge:    24h
MaxBytes:  256MB
MaxMsg:    10MB
Replicas:  1 (single-node default)
```

**Purpose:** All configctl domain events. Long-lived retention for audit and replay.

---

### OBSERVATION_EVENTS (Phase 2)

```
Stream:    OBSERVATION_EVENTS
Subjects:  observation.events.market.>
Storage:   File
MaxAge:    6h
MaxBytes:  1GB
MaxMsg:    1MB
Replicas:  1
```

**Purpose:** Normalized market data from ingest. Short retention — value is consumed by derive and store, not archived.

**Consumer groups (anticipated):**
- `derive.observation` — derive binary processing
- `store.observation` — store binary projection building

---

### EVIDENCE_EVENTS (Phase 3)

```
Stream:    EVIDENCE_EVENTS
Subjects:  evidence.events.record.>
Storage:   File
MaxAge:    72h
MaxBytes:  2GB
MaxMsg:    1MB
Replicas:  1
```

**Purpose:** Structured evidence records with provenance. Medium retention for reprocessing.

---

### SIGNAL_EVENTS (Phase 3)

```
Stream:    SIGNAL_EVENTS
Subjects:  signal.events.indicator.>
Storage:   File
MaxAge:    72h
MaxBytes:  1GB
MaxMsg:    512KB
Replicas:  1
```

**Purpose:** Computed signal indicators. Medium retention for backtesting and audit.

---

### PROJECTION_EVENTS (Phase 3)

```
Stream:    PROJECTION_EVENTS
Subjects:  *.events.projection.>
Storage:   Memory
MaxAge:    1h
MaxBytes:  128MB
MaxMsg:    64KB
Replicas:  1
```

**Purpose:** Projection update notifications. Ephemeral, memory-backed. Not a system of record.

---

## Subject Key Patterns

When subjects include partition keys, these conventions apply:

| Key type       | Format          | Example                           | Purpose                    |
|----------------|-----------------|-----------------------------------|----------------------------|
| Source ID      | `{sourceID}`    | `observation.events.market.price_received.binance` | Order by data source |
| Ticker         | `{ticker}`      | `evidence.events.projection.ticker_series.updated.BTCUSD` | Route by instrument |
| Scope          | `{kind}.{key}`  | `configctl.control.config.get_active.global.default` | Filter by activation scope |
| Pipeline ID    | `{pipelineID}`  | `signal.events.indicator.computed.macd_15m` | Route by processing pipeline |

**Rules:**
- Keys use lowercase alphanumeric characters and underscores only.
- Keys must not contain dots (dots are hierarchy separators).
- Key segments are always the **last** segment(s) in a subject.
- Wildcard subscriptions use `>` to match all keys: `observation.events.market.price_received.>`

---

## Envelope Type Convention

The `Type` field in the envelope identifies the message schema. It follows a separate convention from subject names:

```
{domain}.{plane}.{version}.{name}
```

**Examples:**
```
configctl.control.v1.create_draft_command
configctl.control.v1.create_draft_reply
configctl.events.v1.draft_created
observation.events.v1.price_received
evidence.events.v1.record_created
signal.events.v1.indicator_computed
```

**Version segment (`v1`)** enables schema evolution without changing subjects. A consumer can handle multiple versions by inspecting the Type field.

---

## Migration Notes (Current → Canonical)

The existing configctl subjects pre-date this taxonomy. The following renames align them:

| Current subject                                     | Canonical subject                                    |
|-----------------------------------------------------|------------------------------------------------------|
| `configctl.control.create_draft`                    | `configctl.control.config.create_draft`              |
| `configctl.control.validate_draft`                  | `configctl.control.config.validate_draft`            |
| `configctl.control.validate_config`                 | `configctl.control.config.validate`                  |
| `configctl.control.compile_config`                  | `configctl.control.config.compile`                   |
| `configctl.control.activate_config`                 | `configctl.control.config.activate`                  |
| `configctl.control.get_config`                      | `configctl.control.config.get`                       |
| `configctl.control.get_active`                      | `configctl.control.config.get_active`                |
| `configctl.control.list_configs`                    | `configctl.control.config.list`                      |
| `configctl.control.list_active_runtime_projections` | `configctl.control.config.list_active_projections`   |
| `configctl.control.list_active_ingestion_bindings`  | `configctl.control.config.list_active_bindings`      |

**Impact:** These renames affect `configctl_registry.go` and `configctl_gateway.go`. The migration is mechanical — no logic changes required. This should be executed as a standalone commit before Phase 2 begins.

---

## Summary Table

| Class            | Subject pattern                                    | Transport       | Producer    | Retention |
|------------------|----------------------------------------------------|-----------------|-------------|-----------|
| Control command  | `{domain}.control.{agg}.{operation}`               | Request/reply   | gateway     | None      |
| Control reply    | (NATS inbox)                                       | Request/reply   | domain bin  | None      |
| Input event      | `observation.events.market.{event}.{source}`       | JetStream       | ingest      | Hours     |
| Canonical event  | `{domain}.events.{agg}.{event}`                    | JetStream       | domain bin  | Days      |
| Lifecycle event  | `configctl.events.config.{event}`                  | JetStream       | configctl   | Days      |
| Projection event | `{domain}.events.projection.{name}.updated[.key]`  | JetStream       | store       | Minutes   |
| Query request    | `{domain}.query.{projection}.{operation}[.key]`    | Request/reply   | gateway     | None      |
| Query reply      | (NATS inbox)                                       | Request/reply   | store       | None      |
