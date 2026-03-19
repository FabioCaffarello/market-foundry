# Causal Chain Guidelines

## Context

Market-foundry processes market data through a layered pipeline: `observation → evidence → signal → decision → strategy → risk`. Before the system can cross into `execution`, every event in this chain must be traceable to its origin. These guidelines define how to maintain causal integrity as new families and layers are added.

## Guidelines

### G1: Every new domain event must set both correlation_id and causation_id

When adding a new event type to the derive pipeline:

```go
meta := events.NewMetadata().
    WithCorrelationID(msg.CorrelationID).
    WithCausationID(msg.CausationID)
event := newdomain.SomeEvent{
    Metadata: meta,
    Payload:  payload,
}
```

The `CausationID` in the internal message must be the `Metadata.ID` of the event created at the previous hop.

### G2: Internal fan-out messages must carry CausationID

When an actor creates an event and fans out to the next layer via the scope actor, the fan-out message must include:
- `CorrelationID`: forwarded unchanged from the incoming message.
- `CausationID`: set to the `Metadata.ID` of the event just created at this hop.

### G3: New publishers must pass both IDs to encodeEvent

```go
data, prob := encodeEvent(spec, source, event,
    event.Metadata.CorrelationID,
    event.Metadata.CausationID,
)
```

### G4: Structured logs at each processing step must include trace fields

```go
a.logger.Info("event processed",
    "correlation_id", msg.CorrelationID,
    "causation_id", msg.CausationID,
    // ... domain fields
)
```

### G5: Projection materialization logs must include trace fields

Even though KV projections don't persist trace metadata, the materialization log line must include `correlation_id` and `causation_id` for operational traceability.

### G6: Don't invent new correlation IDs mid-chain

A correlation ID is minted once at the chain origin. Downstream actors must never generate a new correlation ID — they always forward what they received. If an actor receives an empty correlation ID, it should propagate it as empty (this signals a tracing gap, which is easier to detect than a silently invented replacement).

### G7: Causation chains must be monotonic

For a given correlation_id, the causation chain must form a directed acyclic graph. In the current linear pipeline, this means:

```
signal(id=A) → decision(id=B, causation=A) → strategy(id=C, causation=B) → risk(id=D, causation=C)
```

No cycles. No self-references.

## Checklist for new layers

When implementing a new layer (e.g., `execution`):

- [ ] Internal message struct includes `CorrelationID` and `CausationID` fields
- [ ] Actor sets `WithCorrelationID` and `WithCausationID` on event metadata
- [ ] Fan-out message sets `CausationID` to the new event's `Metadata.ID`
- [ ] Publisher calls `encodeEvent` with both IDs
- [ ] Processing log includes `correlation_id` and `causation_id`
- [ ] Projection/materialization log includes trace fields
- [ ] Tests verify that causation_id is correctly set

## What this does NOT cover

- **Distributed tracing** (OpenTelemetry, Jaeger): Not in scope. The current approach is log-based traceability.
- **Cross-service correlation**: The current system is single-process. Cross-service propagation will need NATS header-based correlation when the system is decomposed.
- **Trace sampling**: All events are traced. If volume becomes a concern, selective tracing can be added later.
