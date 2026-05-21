# Multi-Binary Execution Safety Integration Proof

## Purpose

This document records the proof that control, execution safety, and paper order flow behave correctly when derive, execute, and control surface components run as distinct binaries communicating exclusively through NATS.

## Context

Prior stages proved all control/safety behavior within a single process:
- **S271**: KV round-trip (store-level persistence)
- **S273**: Control gate runtime (safety gate + venue adapter in-process)
- **S275**: Control plane full path (publisher gate check + stream observation in-process)

The remaining gap was: do these properties hold when components use **separate NATS connections** — the way real binaries would operate?

## Multi-Binary Shape

Three independent "binaries", each with its own NATS connections:

```
┌──────────────┐       ┌─────────────────────────────┐       ┌──────────────────┐
│ Control      │       │                             │       │ Execute Binary   │
│ Surface      │       │     NATS (JetStream + KV)   │       │                  │
│              │       │                             │       │ ControlKVStore   │
│ ControlKV    │──Put──▶ EXECUTION_CONTROL KV bucket ◀──Get──│ SafetyGate       │
│ Store (own   │       │                             │       │ PaperVenueAdapter│
│ connection)  │       │                             │       │ Fill Publisher   │
└──────────────┘       │                             │       │ (own connections)│
                       │ EXECUTION_EVENTS stream     │       └────────▲─────────┘
┌──────────────┐       │                             │                │
│ Derive Binary│       │                             │                │
│              │       │                             │                │
│ ControlKV    │──Get──▶                             │     subscribe  │
│ Store (own   │       │  execution.events.paper_    │───────────────▶│
│ connection)  │       │  order.submitted.>          │
│ Publisher    │──Pub──▶                             │
│ (own conn)   │       │                             │
└──────────────┘       └─────────────────────────────┘
```

### Binary Isolation Guarantees

| Property | How enforced |
|----------|-------------|
| No shared Go references | Each binary creates its own `ControlKVStore`, `Publisher`, subscriptions |
| Independent NATS connections | `natsclient.Connect(url)` called separately per binary |
| Communication only via NATS | KV bucket and JetStream subjects are the sole shared medium |
| Matches production topology | derive, execute, gateway each have independent connections to NATS |

## Proven Properties

### MB-1: Normal Flow

**Derive publishes → NATS stream → Execute consumes and fills.**

- Derive: evaluates risk → produces ExecutionIntent → gate check (active) → publishes to EXECUTION_EVENTS
- Execute: receives PaperOrderSubmittedEvent via subscription → safety gate check (active) → PaperVenueAdapter fills → VenueOrderFilledEvent produced
- Verified: fill status, side, venue order ID, correlation ID preserved across binary boundary

### MB-2: Halt Propagates Across Binaries

**Control surface halts → derive blocked → execute also reports halted.**

- Control surface: writes `GateHalted` to EXECUTION_CONTROL KV
- Derive: `ControlKVStore.IsHalted()` returns true → publish blocked
- Execute: `SafetyGate.Check()` via its own `ControlKVStore` → `kill_switch` verdict
- Verified: both binaries observe the same KV state through independent connections

### MB-3: Cross-Binary Safety

**Derive publishes while active → gate halts → execute-side safety gate blocks → derive also blocked on next attempt.**

- Phase 1: Active gate → derive publishes → execute fills (happy path)
- Phase 2: Halt → derive blocked → execute safety gate independently confirms kill_switch
- Verified: dual-gate safety (derive-side + execute-side) operates coherently across binary boundary

### MB-4: Resume Propagates

**Halt → resume → both sides allow again.**

- Phase 1: Halted → derive publish blocked
- Phase 2: Resume → derive publishes → execute fills
- Verified: gate state transitions propagate immediately to both binaries via KV reads

### MB-5: Full Active→Halt→Resume Cycle

**Three-phase cycle observed coherently across binary boundary.**

- Phase 1 (Active): derive publishes, execute fills
- Phase 2 (Halted): derive blocked
- Phase 3 (Resume): derive publishes, execute fills
- Verified: cumulative counters (published=2, halted=1, filled=2) correct across cycle

### MB-6: KV Materialization Round-Trip Across Boundary

**Causal trace, fill record, and control state survive the full cross-binary path.**

- Correlation ID from derive → stream → execute fill preserved
- Source, symbol, type, venue order ID all correct
- Fill record exists with `simulated=true`
- A completely separate KV store instance (4th NATS connection) reads the same control state
- Proves KV is the shared medium, not any Go-level reference

## Test Infrastructure

### Location

`internal/adapters/nats/natsexecution/multi_binary_integration_test.go`

### Simulated Binary Types

| Type | Implementation | NATS connections |
|------|---------------|------------------|
| `deriveBinary` | Publisher + ControlKVStore | 2 independent connections |
| `executeBinary` | ControlKVStore + SafetyGate + PaperVenueAdapter + Publisher (fills) + core NATS subscription | 3 independent connections |
| `controlSurface` | ControlKVStore (write path) | 1 independent connection |

### Why Core NATS Subscription

The execute binary uses `nc.Subscribe()` (core NATS) instead of a JetStream push consumer because:
- Core subscriptions become active immediately after `nc.Flush()`
- JetStream push consumers have an inherent activation race between `cons.Consume()` and the server's internal subscription wiring
- The test needs deterministic message delivery, not at-least-once semantics
- The binary boundary proof is about NATS as IPC, not consumer durability

### NATS Requirement

Tests require a running NATS server (localhost:4222 or `NATS_URL`). Skipped automatically when unreachable.

## What This Does Not Prove

- Actual OS process isolation (tests run in a single Go test process)
- Network partition behavior (tests use localhost NATS)
- JetStream consumer durability/redelivery across binary restarts
- Multi-replica coordination (single NATS server, not clustered)
- Real venue interaction (paper venue adapter only)

These are acknowledged gaps for future stages.
