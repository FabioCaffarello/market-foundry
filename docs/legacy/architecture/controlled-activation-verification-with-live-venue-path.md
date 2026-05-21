# Controlled Activation Verification with Live Venue Path

> S341 — Proves that the activation surface controls real event flow through the live actor pipeline.

## Context

S337–S340 established:

- Three-dimensional activation model: adapter (paper/venue), gate (active/halted), credentials (present/absent)
- Canonical acceptance scenarios (AC-1 through AC-6) validated at the domain level
- HTTP control surface smoke proving gate transitions via `/execution/control`
- Live consumer flow tests (LF-1 through LF-4) proving NATS → actor → venue → fill pipeline

**Gap at S340 exit**: acceptance tests validate the domain model in isolation; live flow tests validate a single gate state. No test proves that gate transitions during a running supervisor actually control event flow through the real actor pipeline.

S341 closes this gap.

## Verification Strategy

The controlled activation verification exercises the **full runtime path**:

```
NATS JetStream → Durable Consumer → Hollywood Actor → VenueAdapterActor
    → SafetyGate (kill-switch + staleness) → Venue Submit → Fill Publish
```

Gate transitions are applied **at runtime** via NATS KV writes while the supervisor is running, and their effect on event flow is observed through fill events and health tracker counters.

## Verification Scenarios

### CAV-1: Halted Gate Blocks Live Path (Precondition)

| Step | Action | Expected |
|------|--------|----------|
| 1 | Start supervisor with gate=halted | Actor pipeline active |
| 2 | Publish PaperOrderSubmittedEvent | Event reaches actor (processed counter) |
| 3 | Observe outcome | skipped_halt incremented, filled=0 |

**Proves**: halted gate blocks execution in the real actor path, not just domain model.

### CAV-2: Gate Open Enables Live Flow

| Step | Action | Expected |
|------|--------|----------|
| 1 | Start supervisor with gate=active | Actor pipeline active |
| 2 | Publish PaperOrderSubmittedEvent | Event flows through full pipeline |
| 3 | Observe outcome | VenueOrderFilledEvent received, filled counter incremented |

**Proves**: active gate allows real event flow through the full actor pipeline.

### CAV-3: Gate Halt Blocks After Enable (Runtime Transition)

| Step | Action | Expected |
|------|--------|----------|
| 1 | Start with gate=active | Flow enabled |
| 2 | Publish event, receive fill | Baseline: flow working |
| 3 | Set gate=halted via NATS KV | Runtime halt |
| 4 | Publish another event | Event blocked |
| 5 | Observe outcome | filled count unchanged, skipped_halt incremented |

**Proves**: runtime gate transition from active→halted immediately blocks subsequent events in the real pipeline.

### CAV-4: Full Activation Lifecycle

| Phase | Gate State | Event Outcome | Evidence |
|-------|-----------|---------------|----------|
| 1 — Deploy halted | halted | blocked | skipped_halt >= 1, filled=0 |
| 2 — Operator enables | active | fill received | filled >= 1 |
| 3 — Operator halts | halted | blocked | filled unchanged, skipped_halt increased |

**Proves**: the complete halted → enabled → halted lifecycle on a single running supervisor, using runtime NATS KV gate transitions.

### CAV-5: Audit Fields Observable Through Live Path

| Step | Action | Expected |
|------|--------|----------|
| 1 | Write gate with explicit reason, updated_by, updated_at | NATS KV stores all fields |
| 2 | Read gate back from NATS KV | All audit fields preserved |
| 3 | Construct ActivationSurface from retrieved gate | Correct effective mode, audit fields accessible |

**Proves**: audit trail round-trips through the NATS KV store and composes correctly into the activation surface.

## Test Infrastructure

All CAV tests reuse the S333 test harness:

- `s333NatsURL()`: real NATS connection (localhost:4222 or NATS_URL env)
- `s333BuildEvent()`: PaperOrderSubmittedEvent construction with unique dedup keys
- `s333FillSubscriber`: core NATS subscription for fill events (immediately active)
- `s341SetGate()`: NATS KV gate write with audit fields
- `s341SpawnSupervisor()`: real ExecuteSupervisor with cleanup registration
- `s341WaitCounter()`: polls health tracker counter to target value

## Integration with Existing Smoke

The `smoke-activation.sh` script (Phase 6) runs the S341 integration tests after the S340 acceptance phases. This provides a single `make smoke-activation` entry point that validates both domain-level acceptance and live-path verification.

## File Map

| File | Change |
|------|--------|
| `internal/actors/scopes/execute/controlled_activation_verification_test.go` | New: 5 integration tests (CAV-1 through CAV-5) |
| `scripts/smoke-activation.sh` | Modified: Phase 6 added for S341 integration gate |
| `Makefile` | Modified: smoke-activation target description updated |

## Relationship to Prior Stages

| Stage | What It Proved | S341 Builds On |
|-------|----------------|----------------|
| S333 | NATS → actor pipeline works (single gate state) | Reuses test harness, adds gate transitions |
| S339 | ActivationSurface domain model | Validates surface controls real flow |
| S340 | Acceptance scenarios (domain-only) | Proves scenarios on real actor path |
| S341 | Gate transitions control real event flow | — |
