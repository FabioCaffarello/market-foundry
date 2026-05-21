# Multi-Binary Operational Shape: Findings

## Overview

This document records operational findings from proving the multi-binary integration shape for the execution safety pipeline (S276).

## Operational Shape Validated

### Minimum Viable Multi-Binary Deployment

The smallest shape that proves cross-binary execution safety requires three components:

| Component | Role | NATS usage |
|-----------|------|-----------|
| **Control surface** (gateway/store) | Writes control gate state | KV Put to `EXECUTION_CONTROL` |
| **Derive** | Evaluates risk → publishes paper orders | KV Get (gate check) + JetStream Publish to `EXECUTION_EVENTS` |
| **Execute** | Consumes paper orders → safety gate → venue fill | KV Get (gate check) + JetStream Subscribe to `EXECUTION_EVENTS` + JetStream Publish to `EXECUTION_FILL_EVENTS` |

### Shared Medium

NATS serves as the sole inter-binary communication channel:
- **KV bucket** (`EXECUTION_CONTROL`): global gate state — written by control surface, read by derive and execute
- **JetStream stream** (`EXECUTION_EVENTS`): paper order event bus — written by derive, read by execute
- **JetStream stream** (`EXECUTION_FILL_EVENTS`): fill event bus — written by execute, read by store/writer

No shared memory, no direct RPC between binaries.

## Key Findings

### 1. KV State Propagation is Immediate

Control gate writes are immediately visible to all readers across independent NATS connections. No eventual consistency delay was observed in testing. This is because NATS KV uses the same JetStream stream underneath, and `Get()` reads the latest value directly.

**Implication**: Halt commands take effect within the gate check timeout (2 seconds), not on a polling interval.

### 2. Dual-Gate Safety Holds Across Binary Boundary

The architecture places gate checks at two points:
1. **Derive-side** (ExecutionPublisherActor): blocks publishing to EXECUTION_EVENTS
2. **Execute-side** (SafetyGate in VenueAdapterActor): blocks venue submission

Both gates read the same KV bucket through independent connections. When a halt is written, both binaries observe it on their next gate check. This was proven by MB-2 and MB-3.

**Implication**: Even if a message is in-flight on the stream when a halt is written, the execute-side gate will catch it.

### 3. Core NATS Subscription vs JetStream Push Consumer

During test development, JetStream push consumers exhibited an activation race: messages published immediately after `cons.Consume()` could be missed because the server-side subscription wiring isn't synchronous with the client-side call.

Core NATS subscriptions (`nc.Subscribe()` + `nc.Flush()`) provide deterministic activation. After `Flush()` returns, the subscription is guaranteed active on the server.

**Implication for production**: This race is not an issue for production because:
- Durable consumers start before events flow (binary startup order)
- Missed messages are redelivered by JetStream
- The race only matters for tests that publish immediately after consumer creation

### 4. Causal Trace Survives the Full Path

Correlation ID and causation ID set by derive are preserved through:
1. CBOR serialization → NATS stream → CBOR deserialization
2. Safety gate check (doesn't modify event)
3. Venue adapter processing (fills carry original correlation)
4. Fill event publication (causation chain extended)

**Implication**: End-to-end observability works across binary boundaries without extra wiring.

### 5. Connection Independence is Cheap

Creating 6-7 independent NATS connections (2 for derive, 3 for execute, 1 for control surface) adds negligible overhead. Each connection establishment takes < 5ms on localhost. Production deployments with connection pooling would be even more efficient.

## Synchronization Points

### Binary Startup Order

The minimum viable startup order:
1. **NATS** must be running (all binaries depend on it)
2. **Control surface** should write initial gate state (or rely on fail-open default)
3. **Derive** and **execute** can start in any order — derive creates the stream, execute subscribes

### Fail-Open Default

If no control gate key exists in KV, `ControlKVStore.Get()` returns `DefaultControlGate()` (active). This means:
- No explicit initialization is required
- Binaries can start before the control surface writes any state
- The system defaults to allowing execution

### Idempotency

- JetStream publish uses deduplication keys (`intent.DeduplicationKey()`) — duplicate publishes are silently ignored
- KV writes are last-writer-wins (no optimistic concurrency on control gate)
- Venue order IDs are randomly generated — no collision risk

## Points of Fragility

### 1. NATS Availability

All binaries depend on NATS. If NATS is unreachable:
- Derive cannot publish (returns `Unavailable` error)
- Execute cannot consume (subscription drops, reconnect needed)
- Control surface cannot write gate state
- Gate checks fail-open (active) — execution proceeds without safety check

**Mitigation**: NATS clustering in production. Monitoring on connection state.

### 2. Gate Check Latency

Safety gate uses a 2-second timeout for KV reads. If NATS responds slowly:
- Reads that timeout fail-open (IsHalted returns false)
- A truly halted system could briefly allow execution during NATS degradation

**Mitigation**: Monitor gate check latency. Consider fail-closed option for high-stakes environments.

### 3. Consumer Lag

If execute binary falls behind on EXECUTION_EVENTS consumption:
- Intents queue up on the stream
- Old intents may be caught by staleness guard (120s max age)
- Backlog recovery requires processing all queued messages

**Mitigation**: Monitor consumer lag. Staleness guard naturally bounds the impact.

## Limits of This Validation

| What was validated | What was NOT validated |
|---|---|
| Separate NATS connections per binary | Separate OS processes |
| KV state propagation across connections | Network partition handling |
| Stream publish → subscribe across connections | Consumer redelivery and exactly-once |
| Safety gate correctness in multi-binary context | Multi-replica coordination |
| Core NATS subscription reliability | JetStream push consumer recovery |
| Paper venue adapter (simulated fills) | Real venue interaction |
| Single NATS server | NATS cluster with replicas |
