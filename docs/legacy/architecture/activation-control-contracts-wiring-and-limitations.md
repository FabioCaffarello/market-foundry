# Activation Control Contracts, Wiring, and Limitations

> S339 — Venue Activation Wave

## Purpose

This document specifies the contracts, wiring paths, and known limitations of the canonical activation surface. It serves as the operator reference for how activation state flows through the system and where the boundaries lie.

## Contracts

### 1. ActivationSurface (Domain)

**Location**: `internal/domain/execution/activation.go`

```go
type ActivationSurface struct {
    Adapter     AdapterState    // "paper" | "venue"
    Gate        ControlGate     // {status, reason, updated_at, updated_by}
    Credentials CredentialState // "present" | "absent"
    Effective   EffectiveMode   // derived: "paper" | "venue_halted" | "venue_degraded" | "venue_live"
    ObservedAt  time.Time       // when the surface was computed
}
```

**Invariant**: `Effective` is always the output of `ComputeEffectiveMode(Adapter, Gate.Status, Credentials)`. It is never stored independently.

### 2. ControlGate (Domain)

**Location**: `internal/domain/execution/control.go`

```go
type ControlGate struct {
    Status    GateStatus // "active" | "halted"
    Reason    string     // operator-provided audit trail
    UpdatedAt time.Time  // UTC timestamp of last change
    UpdatedBy string     // operator identity
}
```

**Storage**: NATS KV bucket `EXECUTION_CONTROL`, key `global`.
**Default**: `active` (fail-open when key is absent or KV is unreachable).

### 3. SetExecutionControlCommand (Application)

**Location**: `internal/application/executionclient/control_contracts.go`

```go
type SetExecutionControlCommand struct {
    Status    string // "active" | "halted"
    Reason    string // optional, recommended for audit
    UpdatedBy string // optional, recommended for audit
}
```

### 4. ActivationSurfaceQuery/Reply (Application)

**Location**: `internal/application/executionclient/control_contracts.go`

```go
type ActivationSurfaceQuery struct{}

type ActivationSurfaceReply struct {
    Surface execution.ActivationSurface
}
```

## Wiring Paths

### Startup Wiring

```
AppConfig (venue.type)
    │
    ├──▶ buildVenueAdapter() → venueAdapterResult{submit, query, credentialState}
    │
    ├──▶ adapterState = paper | venue (derived from venue.type)
    │
    └──▶ NewExecuteSupervisor(..., WithActivationState(adapter, creds))
              │
              └──▶ VenueAdapterConfig{AdapterState, CredentialState}
                       │
                       └──▶ VenueAdapterActor.start()
                                │
                                ├──▶ Connect NATS KV → read ControlGate
                                └──▶ NewActivationSurface(adapter, gate, creds)
                                     → Log "activation surface resolved"
```

### Runtime Control Wiring

```
Operator
    │
    ├──▶ PUT /execution/control {status, reason, updated_by}
    │       │
    │       └──▶ Gateway HTTP handler
    │               │
    │               └──▶ NATS request → Store QueryResponder
    │                       │
    │                       └──▶ ControlKVStore.Put() → NATS KV bucket
    │
    ├──▶ [Concurrent readers]
    │       ├──▶ ExecutionPublisherActor (derive) → IsHalted() before publish
    │       ├──▶ VenueAdapterActor (execute) → SafetyGate.Check() before submit
    │       └──▶ RetrySubmitter → IsHalted() between retry attempts
    │
    └──▶ GET /execution/control
            │
            └──▶ Gateway HTTP handler → NATS request → Store → KV read → reply
```

### Safety Gate Wiring

```
VenueAdapterActor.onIntent(intent)
    │
    ├──▶ SafetyGate.Check(intent.Timestamp, now)
    │       │
    │       ├──▶ Gate 1: gateChecker.IsHalted(ctx) [2s timeout, fail-open]
    │       │       └──▶ ControlKVStore.Get() → NATS KV
    │       │
    │       └──▶ Gate 2: staleness.IsStale(ts, now) [age > maxAge]
    │
    ├──▶ [blocked] → log + counter (skipped_halt | skipped_stale)
    │
    └──▶ [allowed] → venue.SubmitOrder(ctx, request)
             │
             └──▶ Post200Reconciler → RetrySubmitter → rawAdapter
```

## File Map

| File | Role | Changed in S339 |
|------|------|----------------|
| `internal/domain/execution/activation.go` | Canonical activation surface domain model | **New** |
| `internal/domain/execution/activation_test.go` | Exhaustive truth table tests | **New** |
| `internal/domain/execution/control.go` | Gate domain model | Unchanged |
| `internal/application/executionclient/control_contracts.go` | Query/command contracts | **Extended** |
| `cmd/execute/run.go` | Binary startup with activation logging | **Modified** |
| `internal/actors/scopes/execute/execute_supervisor.go` | Supervisor with activation state | **Modified** |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Adapter with surface logging | **Modified** |
| `internal/application/execution/safety_gate.go` | Pre-submit safety checks | Unchanged |
| `internal/adapters/nats/natsexecution/control_kv_store.go` | KV persistence | Unchanged |
| `internal/interfaces/http/handlers/execution_control.go` | HTTP control surface | Unchanged |

## Limitations

### L-1: Gate State Is Unknown at Binary Startup

The activation surface at startup logs the effective mode assuming `gate=active` because NATS KV is not yet connected. The true gate state is resolved only after `VenueAdapterActor.start()` connects to KV. Between these two points, the logged effective mode may be optimistic.

**Mitigation**: Two distinct log lines clearly indicate when gate is assumed vs. resolved.

### L-2: No Drain Semantics

When the gate transitions from `active` to `halted`, intents that have already passed checkpoint 1 (derive publisher) but not yet reached checkpoint 2 (execute adapter) will still be processed. There is a window of ~seconds where orders may execute after halt.

**Mitigation**: Acceptable for current scope. Future enhancement could add JetStream consumer pause.

### L-3: Single Global Gate

The execution control gate is global — it halts ALL execution families and ALL symbols simultaneously. There is no per-symbol, per-family, or per-source gate.

**Mitigation**: Sufficient for current activation scope. Per-symbol gating is a future enhancement if needed.

### L-4: Fail-Open on KV Unavailability

Both checkpoints default to `active` when the NATS KV store is unreachable. This means a KV outage during halted state allows execution to proceed.

**Mitigation**: Documented, intentional design — availability is prioritized over safety. Operators should ensure KV is healthy before activating venue execution.

### L-5: No Composite HTTP Endpoint for Full Surface

The activation surface is currently logged at startup and can be queried via the existing `/execution/control` endpoint for the gate dimension. A dedicated `/activation/surface` endpoint that returns the full `ActivationSurface` (including adapter and credential state) is deferred — it requires the execute binary to expose its startup state through the gateway, which is not wired in this stage.

**Mitigation**: Startup logs provide the full surface. Gate state is queryable at runtime.

### L-6: No Automatic Rollback

If the system enters an undesirable state (e.g., excessive errors in venue_live mode), there is no automatic transition back to `halted`. The operator must manually issue `PUT /execution/control {status: "halted"}`.

**Mitigation**: Acceptable for current activation scope. Automated circuit-breaker is a future enhancement.

## Operator Checklist

Before transitioning to `venue_live`:

1. Verify `venue.type=binance_futures_testnet` in config
2. Verify credentials are set in environment (`MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY`, `_API_SECRET`)
3. Verify gate is `halted` before starting the binary
4. Start binary — confirm "activation surface resolved" log shows `effective=venue_halted`
5. Activate gate: `PUT /execution/control {status: "active", reason: "...", updated_by: "..."}`
6. Confirm "activation surface resolved" or health counter updates show `venue_live` behavior
7. Monitor fills, errors, and gate state
8. If any issue: `PUT /execution/control {status: "halted", reason: "...", updated_by: "..."}`
