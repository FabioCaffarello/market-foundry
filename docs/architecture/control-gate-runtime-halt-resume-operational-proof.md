# Control Gate Runtime Halt/Resume Operational Proof

Status: proven (S273)
Scope: execution paper pipeline — ControlGate halt/resume via real NATS KV

## Problem

The ControlGate was implemented (S78) and integrated into the actor path (S270), but the S269 gate recorded an open debt: the runtime halt/resume cycle had never been proven end-to-end against the real KV store. Existing tests used in-memory mocks (`mockGateChecker`), proving the SafetyGate decision logic but not the actual KV-backed state transition behavior.

## Objective

Prove that the ControlGate's dynamic state transitions (`active → halted → active`) propagate correctly through the real NATS KV store and that the SafetyGate — wired to the production `ControlKVStore` — blocks and resumes intent processing in response to those transitions.

## Approach

Six runtime tests in `internal/adapters/nats/natsexecution/control_gate_runtime_test.go`, each using:

- **Real `ControlKVStore`** connected to a live NATS server with JetStream
- **Real `SafetyGate`** with `ControlKVStore` as the `GateChecker` (not a mock)
- **Real `PaperVenueAdapter`** and `PaperOrderEvaluator` for intent construction
- **Real `healthz.Tracker`** for counter verification

The tests replicate the `VenueAdapterActor.onIntent()` decision path line-by-line.

## Proven Properties

| ID | Property | Evidence |
|----|----------|----------|
| CG-RT-1 | Default state (no KV key) is fail-open — intents flow | `TestControlGateRuntime_DefaultState_FailOpen_IntentFlows` |
| CG-RT-2 | `active → halted` transition blocks subsequent intents | `TestControlGateRuntime_ActiveToHalted_BlocksIntents` |
| CG-RT-3 | `halted → active` transition resumes intent flow | `TestControlGateRuntime_HaltedToActive_ResumesFlow` |
| CG-RT-4 | Full cycle `active→halted→active→halted` is repeatable | `TestControlGateRuntime_FullCycle_ActiveHaltedActiveHalted` |
| CG-RT-5 | Audit fields (reason, updated_by, updated_at) survive KV round-trip | `TestControlGateRuntime_AuditFields_SurviveRoundTrip` |
| CG-RT-6 | Sustained halt blocks all intents consistently (5/5 blocked) | `TestControlGateRuntime_MultipleIntentsDuringHalt_AllBlocked` |

## Execution Path Covered

```
ControlKVStore.Put(halted)
  → ControlKVStore.IsHalted(ctx)  [real KV read]
    → SafetyGate.Check()
      → verdict: kill_switch
        → counter: skipped_halt++
        → intent: NOT submitted

ControlKVStore.Put(active)
  → ControlKVStore.IsHalted(ctx)  [real KV read]
    → SafetyGate.Check()
      → verdict: allowed
        → PaperVenueAdapter.SubmitOrder()
          → counter: filled++
```

## Test Infrastructure

- Tests skip automatically when NATS is unreachable (`net.DialTimeout` probe)
- CI runs these against the NATS server started by the CI pipeline
- Each test creates its own `ControlKVStore` instance (shared `EXECUTION_CONTROL` bucket, `global` key)

## Invariants Upheld

- **ECI-1**: Gate state is always read from KV before every intent evaluation
- **ECI-2**: Halted state blocks all intent types (action and no-action)
- **ECI-3**: Resume restores flow without restart or redeployment
- **ECI-4**: Audit trail (reason, operator, timestamp) persists across transitions
