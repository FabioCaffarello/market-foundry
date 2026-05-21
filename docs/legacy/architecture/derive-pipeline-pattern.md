# Derive Pipeline Pattern — Market Foundry

> Canonical document. Defines the Consume-Transform-Publish pattern implemented by the derive binary.
> Approved: 2026-03-16. All current and future derive pipelines must conform to this pattern.

---

## Pattern: Consume -> Transform -> Publish

The derive binary implements a single canonical pattern: it consumes events from a source stream, applies a pure transformation, and publishes the result to a target stream. Every pipeline within derive is a concrete instance of this pattern.

The current implementation transforms observation trade events into candle evidence events via the CandleSampler. Future pipelines (signal computation, enrichment, aggregation) follow the same structure.

---

## Canonical Pipeline Structure

```
BindingWatcherActor
  |
  v  (activation / deactivation)
DeriveSupervisor
  |
  v  (routes by partition key = source)
SourceScopeActor [per source]
  |-- ConsumerActor     reads from source stream
  |-- SamplerActor      pure transform (CandleSampler)
  |-- PublisherActor     writes to target stream
```

### 1. Consumer Actor

Reads from the source JetStream stream using a durable, ack-explicit consumer. Messages are delivered to the scope's transform actor. Acknowledgment happens only after the transformed result has been successfully published downstream.

### 2. Supervisor

Routes incoming activation events to the correct scope actor based on the partition key (source). Creates scope actors on demand, tears them down on deactivation. The supervisor is the only actor that knows the full set of active scopes.

### 3. Scope Actor (per partition)

One scope actor exists per partition key (e.g., per exchange source). The scope isolates the failure domain: if processing for one source fails, other sources continue unaffected. Each scope owns its own consumer, transform, and publisher actors as children.

### 4. Transform Actor

Executes pure business logic with no I/O, no NATS access, and no actor framework dependencies. The transform function is a plain Go function that accepts domain input and returns domain output.

Current implementation: `CandleSampler` in `internal/application/derive/sampler.go`. It accumulates trade observations and emits candle evidence events at timeframe boundaries.

### 5. Publisher Actor

Writes transformed events to the target JetStream stream. Each scope has its own publisher, isolating NATS connections per partition. The publisher sets a message ID derived from the event metadata to enable JetStream deduplication.

---

## Key Properties

**Transform logic is pure and testable.** The CandleSampler is a plain struct with no infrastructure dependencies. It can be tested with table-driven tests using synthetic trade data. No NATS, no actors, no mocks required.

**Scope actors provide failure isolation.** A panic or persistent error in the Binance scope does not affect the Coinbase scope. Supervisors restart failed scopes independently.

**Publisher per scope isolates connections.** Each scope maintains its own NATS publisher. A slow or backpressured target stream for one scope does not block publishing for others.

**Deduplication prevents duplicate evidence on replay.** Every published event carries a deterministic message ID. If a consumer replays messages (after restart or redelivery), the target stream's deduplication window rejects duplicates automatically.

---

## Dynamic Activation

The `BindingWatcherActor` drives pipeline activation and deactivation at runtime:

1. On startup, it queries configctl for the current set of active bindings.
2. It subscribes to configctl lifecycle events for ongoing changes.
3. When a binding is activated, the watcher instructs the supervisor to create the corresponding scope.
4. When a binding is deactivated, the watcher instructs the supervisor to tear down the scope.

This means no pipeline runs unless configctl has an active configuration for it. There is no static wiring.

---

## Reusability

The Consume-Transform-Publish pattern is not specific to candle sampling. Future pipelines follow the same structure with different transform functions:

| Pipeline             | Source stream         | Transform             | Target stream         |
|----------------------|-----------------------|-----------------------|-----------------------|
| Candle sampling      | Observation trades    | CandleSampler         | Evidence candles      |
| Signal computation   | Evidence candles      | SignalEvaluator       | Signal indicators     |
| Enrichment           | Evidence candles      | Enricher              | Evidence enriched     |

The scope actor, consumer, and publisher are structurally identical across pipelines. Only the transform function changes.

---

## Anti-Patterns

### Coupling transform logic to actor lifecycle

The transform function must not import the actor framework. It receives domain types and returns domain types. The actor that hosts the transform is responsible for message deserialization, invoking the function, and forwarding the result to the publisher. The transform itself knows nothing about actors, messages, or supervision.

### Sharing publishers across scopes

Each scope must own its own publisher. Sharing a publisher across scopes creates contention, eliminates failure isolation, and makes backpressure from one scope affect all others. The cost of an additional NATS connection per scope is negligible compared to the operational risk of shared state.

### Processing trades for unknown symbols

If a trade arrives for a symbol that has no active binding, the scope drops it silently. This is intentional. The derive binary processes only what configctl has activated. Logging unknown symbols at a low level is acceptable for observability, but the system must not error, block, or accumulate state for unrecognized input.

---

## Related Documents

- [System Principles](system-principles.md) -- Foundational architectural rules
- [Stream Taxonomy](stream-taxonomy.md) -- NATS subject and stream definitions
- [Actor Ownership](actor-ownership.md) -- Actor hierarchy and supervision model
