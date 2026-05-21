# Stage S67 — End-to-End Traceability Hardening Report

## Executive Summary

S67 hardened the causal traceability chain between `decision`, `strategy`, and `risk` layers. The work activated `causation_id` propagation (which was structurally present but never set), added trace context to all structured logs in the derive and store layers, and documented the invariants that must hold before crossing into `execution`.

## Improvements Applied

### 1. CausationID propagation in derive chain

**Before**: Every event in the chain set only `correlation_id`. The `causation_id` field existed in `events.Metadata` and `envelope.Envelope` but was never populated for domain events.

**After**: Each derive actor now sets `causation_id` to the `Metadata.ID` of the event at the previous hop:

| Actor | Event Created | CausationID Source |
|---|---|---|
| RSISignalSamplerActor | SignalGeneratedEvent | (origin — not set) |
| RSIOversoldEvaluatorActor | DecisionEvaluatedEvent | SignalGeneratedEvent.Metadata.ID |
| MeanReversionEntryResolverActor | StrategyResolvedEvent | DecisionEvaluatedEvent.Metadata.ID |
| PositionExposureEvaluatorActor | RiskAssessedEvent | StrategyResolvedEvent.Metadata.ID |

### 2. Internal message CausationID fields

Added `CausationID` field to all inter-actor fan-out messages:
- `signalGeneratedMessage`
- `decisionEvaluatedMessage`
- `strategyResolvedMessage`

These fields carry the predecessor event's `Metadata.ID` so the next actor can set it as `CausationID` on the new event.

### 3. NATS envelope causation_id

Updated `encodeEvent` to accept and propagate `causationID` into the transport envelope alongside `correlationID`. All publishers (evidence, signal, decision, strategy, risk, observation, jetstream) updated to pass both IDs.

### 4. Structured log trace context

Added `correlation_id` and `causation_id` to structured log output at:
- Signal generation (signal sampler actor)
- Decision evaluation (decision evaluator actor)
- Strategy resolution (strategy resolver actor)
- Risk assessment (risk evaluator actor)
- Decision materialization (decision projection actor)
- Strategy materialization (strategy projection actor)
- Risk materialization (risk projection actor)
- Publish error logs (decision, strategy, risk publisher actors)

## Files Changed

### Runtime (derive actors)
- `internal/actors/scopes/derive/messages.go` — added CausationID to fan-out messages
- `internal/actors/scopes/derive/signal_sampler_actor.go` — set CausationID on fan-out, log correlation_id
- `internal/actors/scopes/derive/decision_evaluator_actor.go` — set CausationID on metadata and fan-out, log trace context
- `internal/actors/scopes/derive/strategy_resolver_actor.go` — set CausationID on metadata and fan-out, log trace context
- `internal/actors/scopes/derive/risk_evaluator_actor.go` — set CausationID on metadata, log trace context

### Runtime (publisher actors)
- `internal/actors/scopes/derive/decision_publisher_actor.go` — log correlation_id on errors
- `internal/actors/scopes/derive/strategy_publisher_actor.go` — log correlation_id on errors
- `internal/actors/scopes/derive/risk_publisher_actor.go` — log correlation_id on errors

### Runtime (store actors)
- `internal/actors/scopes/store/decision_projection_actor.go` — log trace context on materialization
- `internal/actors/scopes/store/strategy_projection_actor.go` — log trace context on materialization
- `internal/actors/scopes/store/risk_projection_actor.go` — log trace context on materialization

### Transport (NATS adapter)
- `internal/adapters/nats/codec.go` — `encodeEvent` now accepts causationID parameter
- `internal/adapters/nats/decision_publisher.go` — pass CausationID to encodeEvent
- `internal/adapters/nats/strategy_publisher.go` — pass CausationID to encodeEvent
- `internal/adapters/nats/risk_publisher.go` — pass CausationID to encodeEvent
- `internal/adapters/nats/signal_publisher.go` — pass CausationID to encodeEvent
- `internal/adapters/nats/evidence_publisher.go` — pass CausationID to encodeEvent (3 methods)
- `internal/adapters/nats/observation_publisher.go` — pass CausationID to encodeEvent
- `internal/adapters/nats/jetstream_publisher.go` — pass CausationID to encodeEvent

### Tests
- `internal/adapters/nats/codec_roundtrip_test.go` — updated all encodeEvent calls
- `internal/adapters/nats/consumer_dispatch_test.go` — updated all encodeEvent calls

### Documentation
- `docs/architecture/end-to-end-traceability.md` — traceability invariants and propagation rules
- `docs/architecture/causal-chain-guidelines.md` — guidelines for maintaining causal integrity

## Remaining Limitations

1. **KV projections don't persist trace metadata**: The latest-state KV buckets store only the domain model. Trace metadata is available in the JetStream stream and in logs, but not in the materialized projection. This is acceptable for latest-only projections but will need attention if analytical storage is adopted.

2. **No automated traceability verification test**: Trace integrity is verified by visual log inspection or manual smoke tests. An automated integration test that asserts the full causation chain would be valuable but requires a running NATS server.

3. **Observation → Evidence correlation origin**: The correlation_id is currently minted at the observation layer and propagated through the candle sampler. The exact minting point depends on the ingest adapter. This document focuses on the decision→strategy→risk segment.

4. **Cross-service correlation not yet needed**: The system runs as a single process. When it is decomposed into separate services, NATS header-based correlation propagation will be needed.

5. **No trace sampling**: All events carry trace metadata. If event volume becomes a concern, selective tracing can be added without structural changes.

## Impact on S68 Readiness

This stage directly reduces structural risk for the execution boundary:

- **Causal chain is now explicit**: Any event reaching the execution layer will carry both `correlation_id` (full chain) and `causation_id` (direct predecessor), enabling audit and rollback analysis.
- **Log-based trace reconstruction is operational**: With JSON logging enabled, the full processing chain for any market event can be reconstructed by filtering on `correlation_id`.
- **Guidelines are documented**: The checklist in `causal-chain-guidelines.md` ensures that new layers (including execution) maintain trace integrity from day one.
- **No runtime overhead added**: The changes are metadata propagation (fields that already existed in structs) and log field additions. No new allocations, goroutines, or network calls.
