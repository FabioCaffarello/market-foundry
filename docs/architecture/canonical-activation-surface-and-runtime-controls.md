# Canonical Activation Surface and Runtime Controls

> S339 — Venue Activation Wave

## Purpose

This document defines the canonical activation surface for the market-foundry venue path. The activation surface is the single authoritative model that composes the three independent activation dimensions into a derived effective mode, replacing implicit reasoning about scattered state with explicit, queryable, testable control.

## Three-Dimensional Activation Model

Venue activation is determined by exactly three independent dimensions:

| Dimension | Authority | Mutability | Values |
|-----------|-----------|------------|--------|
| **Adapter** | `venue.type` in AppConfig | Immutable per process | `paper`, `venue` |
| **Gate** | NATS KV `EXECUTION_CONTROL/global` | Runtime-mutable via HTTP | `active`, `halted` |
| **Credential** | Environment variables at startup | Immutable per process | `present`, `absent` |

No other dimension affects whether the system produces real venue orders. Any future dimension must be added to this model explicitly.

## Effective Mode Truth Table

The effective mode is **always derived, never stored**. It is computed from the three inputs at observation time:

| Adapter | Gate | Credentials | Effective Mode | Orders Reach Venue? |
|---------|------|-------------|----------------|---------------------|
| paper | * | * | `paper` | No |
| venue | halted | * | `venue_halted` | No |
| venue | active | absent | `venue_degraded` | No |
| venue | active | present | **`venue_live`** | **Yes** |

Only `venue_live` produces real orders. All other modes are safe.

## ActivationSurface Type

```go
type ActivationSurface struct {
    Adapter     AdapterState    `json:"adapter"`
    Gate        ControlGate     `json:"gate"`
    Credentials CredentialState `json:"credentials"`
    Effective   EffectiveMode   `json:"effective"`
    ObservedAt  time.Time       `json:"observed_at"`
}
```

**Domain location**: `internal/domain/execution/activation.go`

The surface is computed via `NewActivationSurface(adapter, gate, creds)` which calls `ComputeEffectiveMode` internally.

## Runtime Controls

### Enable Execution (Activate Gate)

```
PUT /execution/control
{ "status": "active", "reason": "s339-single-order-test", "updated_by": "operator" }
```

Transitions the gate from halted to active. Combined with venue adapter + credentials, this enables real execution.

### Halt Execution (Kill Switch)

```
PUT /execution/control
{ "status": "halted", "reason": "risk-limit-breach", "updated_by": "oncall" }
```

Immediately blocks new intents at both checkpoints (derive publisher + execute adapter). In-flight intents may complete.

### Query State

```
GET /execution/control
```

Returns the current gate state with audit fields (reason, updated_by, updated_at).

## Checkpoint Architecture

The kill switch is enforced at two independent points:

```
Derive Binary                    Execute Binary
┌─────────────────┐             ┌──────────────────────┐
│ ExecutionPublisher│             │ VenueAdapterActor    │
│   ┌────────────┐ │             │   ┌────────────────┐ │
│   │ Gate Check │ │             │   │ SafetyGate     │ │
│   │ (CP-1)     │ │             │   │  Gate 1: Kill  │ │
│   └────────────┘ │             │   │  Gate 2: Stale │ │
│        │         │             │   └────────────────┘ │
│   ┌────────────┐ │             │         │            │
│   │ Publish    │ │     NATS    │   ┌────────────────┐ │
│   │ Intent     │─┼────────────┼──▶│ SubmitOrder    │ │
│   └────────────┘ │             │   └────────────────┘ │
└─────────────────┘             └──────────────────────┘
```

Both checkpoints read the same NATS KV key (`EXECUTION_CONTROL/global`). Both are fail-open: if KV is unreachable, execution proceeds.

## Startup Logging

At binary startup, the execute binary logs two canonical activation lines:

1. **Startup surface** (before NATS KV connect): logs adapter state, credential state, and the effective mode assuming gate=active.
2. **Resolved surface** (after NATS KV connect in VenueAdapterActor): logs the full three-dimensional surface with the actual gate state from KV.

These two lines are the canonical evidence for activation state auditing.

## Design Invariants

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| S339-1 | Effective mode is always derived, never stored | `ComputeEffectiveMode` is a pure function |
| S339-2 | Only `venue_live` produces real orders | Truth table enforced by SafetyGate + adapter selection |
| S339-3 | Gate changes are auditable | ControlGate carries reason, updated_by, updated_at |
| S339-4 | Adapter/credential state is immutable per process | Set at startup, no hot-reload |
| S339-5 | Both checkpoints read the same KV key | Single `EXECUTION_CONTROL/global` key |
| S339-6 | Fail-open semantics on KV unavailability | Documented, intentional, consistent |

## Non-Goals

- Hot-reload of venue adapter or credentials at runtime.
- Per-symbol or per-family gate isolation.
- Automatic gate transitions based on error rates or metrics.
- Gradual percentage-based rollout.
- Drain semantics on halt (in-flight intents may complete).
