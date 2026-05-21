# End-to-End Traceability

## Purpose

This document defines the traceability invariants for the causal chain between `decision`, `strategy`, and `risk` layers in market-foundry. These rules exist to ensure that every event can be traced back to its origin before the system crosses the execution boundary.

## Core Concepts

### Correlation ID

A stable identifier that groups all events belonging to the same logical flow. It is minted once at the origin of a processing chain (typically when a candle is finalized from raw trades) and propagated unchanged through every downstream event.

- **Minted at**: Evidence sampler (candle finalization) — inherited from the observation trade event.
- **Propagated through**: signal → decision → strategy → risk.
- **Semantics**: "all events with the same correlation_id belong to the same market data processing chain."

### Causation ID

A pointer from a child event to the specific parent event that directly caused it. Unlike correlation_id (which is shared across an entire chain), causation_id links exactly two events in a parent→child relationship.

- **Set by**: Each derive actor when creating a new event.
- **Value**: The `Metadata.ID` of the event that triggered this event's creation.
- **Semantics**: "this event was directly caused by the event with this ID."

### Event Metadata ID

Each event receives a unique random hex ID at creation (`events.NewMetadata().ID`). This ID serves as the identity for causation linking.

## Propagation Rules

### Rule 1: Correlation ID is immutable within a chain

Once minted, the correlation_id must not be modified or replaced at any hop. Every actor in the derive chain receives and forwards the same correlation_id.

### Rule 2: Causation ID links consecutive layers

| Producer Layer | Event | CausationID Source |
|---|---|---|
| Signal sampler | `SignalGeneratedEvent` | (not set — chain origin) |
| Decision evaluator | `DecisionEvaluatedEvent` | `SignalGeneratedEvent.Metadata.ID` |
| Strategy resolver | `StrategyResolvedEvent` | `DecisionEvaluatedEvent.Metadata.ID` |
| Risk evaluator | `RiskAssessedEvent` | `StrategyResolvedEvent.Metadata.ID` |

### Rule 3: Both IDs travel in the NATS envelope

The `encodeEvent` function writes both `correlation_id` and `causation_id` into the envelope. Consumers decode the full envelope and can access both IDs via `event.Metadata`.

### Rule 4: Internal messages carry both IDs

Actor-to-actor messages (`signalGeneratedMessage`, `decisionEvaluatedMessage`, `strategyResolvedMessage`) carry both `CorrelationID` and `CausationID` fields to enable proper metadata construction at each hop.

### Rule 5: Structured logs include trace context

Every processing log in the derive and store layers includes `correlation_id` and `causation_id` fields. This enables log-based trace reconstruction without external tooling.

## Transport Layer

```
Envelope[T]
├── ID              (unique envelope ID)
├── Kind            ("event")
├── Type            (e.g., "decision.events.v1.rsi_oversold_evaluated")
├── Source          (e.g., "derive.decision-publisher.binancef")
├── CorrelationID   (chain-wide trace ID)
├── CausationID     (parent event ID)
├── Timestamp
├── ContentType     ("application/cbor")
└── Payload: T
    ├── Metadata
    │   ├── ID              (event identity)
    │   ├── OccurredAt
    │   ├── CorrelationID   (matches envelope)
    │   └── CausationID     (matches envelope)
    └── [Domain payload]
```

## Materialization (KV Store)

KV projections currently store only the domain model (e.g., `Decision`, `Strategy`, `RiskAssessment`) without trace metadata. This is acceptable for latest-state projections where the JetStream stream retains the full event with metadata.

**Future consideration**: If an analytical store (e.g., ClickHouse) is adopted, trace metadata should be persisted alongside the domain data for query-time correlation.

## Verification

To verify end-to-end traceability:

1. Enable JSON logging (`LOG_FORMAT=json`).
2. Run a smoke test that generates events through the full chain.
3. Filter logs by a single `correlation_id` — all hops from signal through risk must appear.
4. Verify `causation_id` chains: each event's `causation_id` matches the `event_id` logged at the previous hop.
