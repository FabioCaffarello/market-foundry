# Control Plane Full-Path Proof

Status: proven (S275)
Scope: execution control plane — full path from control surface to stream-observable behavior

## Problem

Prior stages proved individual segments of the control plane:

- **S271** proved KV round-trip: `ControlKVStore.Put() → Get()` with field fidelity
- **S273** proved runtime halt/resume at the **venue adapter** level: `ControlKVStore → SafetyGate → VenueAdapterActor`
- **S266/S268** proved the actor chain: signal → decision → strategy → risk → execution

The gap: no test proved the **derive-side gate path** end-to-end:

```
control surface write → KV persistence → ExecutionPublisherActor gate check → NATS stream presence/absence
```

The ExecutionPublisherActor has its own independent gate check (lines 106-122 of `execution_publisher_actor.go`) that reads from the same `EXECUTION_CONTROL` KV bucket. This path was untested at the full-path level — only the venue adapter path was proven.

## Objective

Prove that the complete control plane path works as a coherent unit:

1. A control surface write (KV Put) propagates immediately to the gate check
2. The gate check in the publisher path blocks/allows NATS stream publication
3. Stream-level observation confirms the gate decision (messages present or absent)
4. Both gate check points (derive publisher and venue adapter) observe the same state

## Approach

Five integration tests in `internal/adapters/nats/natsexecution/control_plane_full_path_test.go`, each using:

- **Real `ControlKVStore`** connected to a live NATS server with JetStream
- **Real `Publisher`** publishing to the EXECUTION_EVENTS stream
- **Core NATS subscriber** observing the stream for published messages
- **CBOR envelope decoding** to verify message identity (correlation ID)
- **Gate check replication** mirroring ExecutionPublisherActor.publishWithRetry() exactly

## Proven Properties

| ID | Property | Evidence |
|----|----------|----------|
| CP-FP-1 | Active gate → intent published, message observable on EXECUTION_EVENTS stream | `TestControlPlane_FullPath_Active_PublishesToStream` |
| CP-FP-2 | Halted gate → intent blocked, no message on stream | `TestControlPlane_FullPath_Halted_BlocksStreamPublish` |
| CP-FP-3 | Active→Halted→Resume cycle → stream observability matches gate state at each phase | `TestControlPlane_FullPath_ActiveHaltedResume_Cycle` |
| CP-FP-4 | Dual checkpoint: derive publisher path AND venue adapter path observe same KV state | `TestControlPlane_FullPath_DualCheckpoint_PublisherAndVenue` |
| CP-FP-5 | Control surface writes propagate immediately (10 rapid alternations, zero stale reads) | `TestControlPlane_FullPath_ImmediatePropagation` |

## Full Path Topology

```
                    ┌─────────────────────────────┐
                    │   Control Surface (KV Put)   │
                    │   ControlKVStore.Put(gate)   │
                    └──────────────┬──────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │  EXECUTION_CONTROL KV Bucket │
                    │     key: "global"            │
                    │     val: {status, reason,    │
                    │           updated_at,        │
                    │           updated_by}        │
                    └──────┬──────────────┬───────┘
                           │              │
              ┌────────────▼────┐   ┌─────▼──────────────┐
              │ Derive Path     │   │ Execute Path       │
              │ (publisher)     │   │ (venue adapter)    │
              │                 │   │                    │
              │ ControlKVStore  │   │ ControlKVStore     │
              │ .IsHalted(ctx)  │   │ → SafetyGate       │
              │                 │   │   .Check()         │
              └────────┬───────┘   └─────┬──────────────┘
                       │                  │
              ┌────────▼───────┐   ┌─────▼──────────────┐
              │ if active:     │   │ if allowed:        │
              │   Publisher    │   │   VenueAdapter     │
              │   .Publish()   │   │   .SubmitOrder()   │
              │ if halted:     │   │ if kill_switch:    │
              │   drop + count │   │   skip + count     │
              └────────┬───────┘   └─────┬──────────────┘
                       │                  │
              ┌────────▼───────┐   ┌─────▼──────────────┐
              │ EXECUTION_     │   │ EXECUTION_FILL_    │
              │ EVENTS stream  │   │ EVENTS stream      │
              │ (paper_order)  │   │ (venue_order)      │
              └────────────────┘   └────────────────────┘
```

## Gate Check Points

The control plane has **two independent gate check points**, both reading from the same KV source:

1. **Derive Publisher Actor** (`execution_publisher_actor.go:106-122`):
   - Reads `ControlKVStore.IsHalted(ctx)` before every publish
   - Blocks with `halted` counter increment, drops message
   - Fail-open if store is nil

2. **Venue Adapter Actor** (via `SafetyGate.Check()`):
   - Reads `GateChecker.IsHalted(ctx)` as first gate
   - Blocks with `kill_switch` reason
   - Fail-open if checker is nil or times out

CP-FP-4 proves these two paths observe identical state from the same KV source.

## Relationship to Prior Proofs

| Stage | What Was Proven | Gap This Closes |
|-------|----------------|-----------------|
| S271 | KV Put/Get round-trip, field fidelity | Persistence layer |
| S273 | SafetyGate halt/resume with real KV (venue path) | Execute-side runtime |
| **S275** | **Publisher gate + NATS stream observation (derive path)** | **Full-path derive-side** |
| **S275** | **Dual checkpoint consistency** | **Cross-path coherence** |
