# NATS Consumer to Actor Live Flow

> S333 — LSI-1: Proven live path from NATS JetStream durable consumer through
> Hollywood actor system to venue adapter fill publication.

## 1. Canonical Flow Path

```
                        NATS JetStream
                    ┌─────────────────────┐
Derive binary       │  EXECUTION_EVENTS   │       Execute binary
publishes ────────► │  (paper_order       │ ◄──── durable consumer
PaperOrder          │   .submitted.>)     │       "execute-venue-
SubmittedEvent      └─────────────────────┘        market-order-intake"
                                                          │
                                                          ▼
                                                  Consumer.onMessage()
                                                  ├─ CBOR decode
                                                  ├─ handler callback
                                                  │   ├─ consumerTracker.RecordEvent()
                                                  │   └─ actor.Send(adapterPID, intentReceivedMessage)
                                                  └─ msg.Ack()
                                                          │
                                                          ▼
                                                  VenueAdapterActor.Receive()
                                                  └─ onIntent(msg)
                                                     ├─ tracker.Counter("processed").Add(1)
                                                     ├─ SafetyGate.Check()
                                                     │   ├─ Gate 1: ControlKVStore.IsHalted()
                                                     │   └─ Gate 2: StalenessGuard.IsStale()
                                                     ├─ venue.SubmitOrder() [decorated pipeline]
                                                     │   └─ Post200Reconciler → RetrySubmitter → rawAdapter
                                                     ├─ fillPublisher.PublishFill()
                                                     │   └─ EXECUTION_FILL_EVENTS stream
                                                     └─ tracker.Counter("filled").Add(1)
```

## 2. Components and Ownership

| Component | Owner | File |
|-----------|-------|------|
| `ExecuteSupervisor` | execute binary | `internal/actors/scopes/execute/execute_supervisor.go` |
| `VenueAdapterActor` | execute binary | `internal/actors/scopes/execute/venue_adapter_actor.go` |
| `Consumer` (durable) | natsexecution adapter | `internal/adapters/nats/natsexecution/consumer.go` |
| `Publisher` (fill) | natsexecution adapter | `internal/adapters/nats/natsexecution/publisher.go` |
| `ControlKVStore` | natsexecution adapter | `internal/adapters/nats/natsexecution/control_kv_store.go` |
| `SafetyGate` | application/execution | `internal/application/execution/safety_gate.go` |
| `PaperVenueAdapter` | application/execution | `internal/application/execution/paper_venue_adapter.go` |

## 3. Consumer Spec

| Field | Value |
|-------|-------|
| Durable name | `execute-venue-market-order-intake` |
| Stream | `EXECUTION_EVENTS` |
| Filter subject | `execution.events.paper_order.submitted.>` |
| AckWait | 30s |
| MaxDeliver | 5 |
| AckPolicy | Explicit |

**Transitional bridge:** The intake consumer subscribes to `paper_order` subjects
because derive only produces `PaperOrderSubmittedEvent`. When venue-specific intent
subjects are introduced, this consumer will migrate to venue-specific subjects.

## 4. Correlation/Causation Chain

```
PaperOrderSubmittedEvent (from derive)
  Metadata.ID            = <source-event-id>
  Metadata.CorrelationID = <trace-id>         ← immutable across chain
  Metadata.CausationID   = <parent-event-id>

         ▼ (consumer delivers to actor)

VenueOrderFilledEvent (from execute)
  Metadata.ID            = <fill-event-id>     ← new unique ID
  Metadata.CorrelationID = <trace-id>          ← preserved from source
  Metadata.CausationID   = <source-event-id>   ← links to source event
```

## 5. Safety Gates

| Gate | Component | Behavior |
|------|-----------|----------|
| Kill switch | `ControlKVStore.IsHalted()` | Blocks all intents when gate = halted |
| Staleness | `StalenessGuard.IsStale()` | Blocks intents older than `staleness_max_age` |
| Submit timeout | `context.WithTimeout()` | Cancels venue call after `submit_timeout` |
| Fail-open | `ControlKVStore` | Missing KV key defaults to active (no accidental halt) |

## 6. Deduplication

| Layer | Key Pattern | Purpose |
|-------|-------------|---------|
| Execution events | `exec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}` | Prevents duplicate intent publication |
| Fill events | `fill:{venue_order_id}:{timestamp_unix}` | Prevents duplicate fill publication |
| JetStream | `WithMsgID(dedupKey)` | Server-side dedup within window |

## 7. Health Tracking

| Tracker | Counter | Meaning |
|---------|---------|---------|
| venue-consumer | EventCount | Events delivered from NATS to actor |
| venue-adapter | processed | Intents that entered onIntent |
| venue-adapter | filled | Intents that completed venue submit + fill publication |
| venue-adapter | skipped_halt | Intents blocked by kill switch |
| venue-adapter | skipped_stale | Intents blocked by staleness guard |
| venue-adapter | ErrorCount | Venue submit or fill publication failures |

## 8. Evidence (S333 Tests)

| Test | What it proves |
|------|---------------|
| LF-1 | Full NATS → actor → fill round-trip with real `ExecuteSupervisor` |
| LF-2 | Durable consumer resumes after supervisor restart |
| LF-3 | Kill switch blocks real actor path while consumer still delivers |
| LF-4 | Multiple events processed sequentially through real pipeline |

All tests use `//go:build integration` and require a live NATS server.

**Test file:** `internal/actors/scopes/execute/live_consumer_flow_test.go`
