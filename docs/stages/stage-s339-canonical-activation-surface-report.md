# Stage S339 — Canonical Activation Surface and Runtime Controls

> Venue Activation Wave · Completed 2026-03-22

## Executive Summary

S339 transforms venue activation from an implicit composition of scattered state into an explicit, canonical surface with a deterministic truth table. The `ActivationSurface` domain type composes three independent dimensions (adapter, gate, credentials) into a derived effective mode. The execute binary now logs the full activation surface at startup, and the venue adapter actor logs the resolved surface after NATS KV connect. All existing runtime controls (kill switch, staleness guard, dual-checkpoint pattern) are preserved and wired through the canonical model. The activation surface is testable in isolation via an exhaustive 8-row truth table.

## Surface and Controls Delivered

### Canonical ActivationSurface

A new domain type `ActivationSurface` in `internal/domain/execution/activation.go` composes:

| Dimension | Type | Source | Mutability |
|-----------|------|--------|-----------|
| Adapter | `AdapterState` (`paper` / `venue`) | `venue.type` config | Immutable per process |
| Gate | `ControlGate` (`active` / `halted`) | NATS KV `EXECUTION_CONTROL/global` | Runtime-mutable |
| Credentials | `CredentialState` (`present` / `absent`) | Env vars at startup | Immutable per process |

Derived effective mode via `ComputeEffectiveMode()`:

| Adapter | Gate | Credentials | Effective | Live? |
|---------|------|-------------|-----------|-------|
| paper | * | * | `paper` | No |
| venue | halted | * | `venue_halted` | No |
| venue | active | absent | `venue_degraded` | No |
| venue | active | present | **`venue_live`** | **Yes** |

### Runtime Controls (Unchanged, Now Canonical)

- **Enable**: `PUT /execution/control {"status":"active"}`
- **Halt**: `PUT /execution/control {"status":"halted"}`
- **Query**: `GET /execution/control`
- **Dual checkpoint**: derive publisher (CP-1) + execute adapter (CP-2)
- **Safety gate**: kill switch → staleness → submit timeout

### Startup Activation Logging

Two canonical log lines provide full activation audit trail:

1. `"activation surface at startup"` — logged by `cmd/execute/run.go` before NATS connect, shows adapter + credentials + effective-without-gate
2. `"activation surface resolved"` — logged by `VenueAdapterActor.start()` after KV connect, shows full 3D surface including actual gate state

## Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/domain/execution/activation.go` | Canonical activation surface domain model with truth table |
| `internal/domain/execution/activation_test.go` | Exhaustive truth table tests (8 rows + edge cases) |
| `docs/architecture/canonical-activation-surface-and-runtime-controls.md` | Architecture: surface model, controls, invariants |
| `docs/architecture/activation-control-contracts-wiring-and-limitations.md` | Contracts, wiring paths, operator checklist, limitations |

### Modified Files

| File | Change |
|------|--------|
| `cmd/execute/run.go` | Added credential state to `venueAdapterResult`, startup activation surface logging, `WithActivationState` option to supervisor |
| `internal/actors/scopes/execute/execute_supervisor.go` | Added `adapterState`/`credentialState` fields, `SupervisorOption` pattern with `WithActivationState` |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Added `AdapterState`/`CredentialState` to config, resolved activation surface logging after KV connect |
| `internal/application/executionclient/control_contracts.go` | Added `ActivationSurfaceQuery`/`ActivationSurfaceReply` contracts |

## Tests and Evidence

### Unit Tests (All GREEN)

- `TestComputeEffectiveMode_PaperAlwaysPaper` — paper adapter ignores gate and credentials
- `TestComputeEffectiveMode_VenueHalted` — halted gate blocks regardless of credentials
- `TestComputeEffectiveMode_VenueActivePresentIsLive` — only combination that is live
- `TestComputeEffectiveMode_VenueActiveAbsentIsDegraded` — missing credentials = degraded
- `TestComputeEffectiveMode_ExhaustiveTruthTable` — all 8 rows verified
- `TestNewActivationSurface_ComputesEffective` — surface composition
- `TestNewActivationSurface_PaperNotLive` — IsLive/CanReachVenue predicates
- `TestNewActivationSurface_ObservedAtIsSet` — timestamp bounds

### Existing Tests (All GREEN, Backward Compatible)

- `internal/domain/execution` — 13 tests pass (5 new + 8 existing)
- `internal/application/execution` — all tests pass (SafetyGate, StalenessGuard, RetrySubmitter, etc.)
- `NewExecuteSupervisor` backward compatible via variadic `opts ...SupervisorOption`

### Compilation Evidence

- All 4 binaries compile: execute, gateway, derive, store
- `go vet` clean on all modified packages

## Limits Remaining

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|------------|
| L-1 | Gate unknown at binary startup | Low | Two-phase logging (assumed → resolved) |
| L-2 | No drain semantics on halt | Medium | In-flight intents may complete; acceptable for current scope |
| L-3 | Single global gate (no per-symbol) | Low | Sufficient for activation wave |
| L-4 | Fail-open on KV unavailability | Medium | Documented, intentional; operators ensure KV health |
| L-5 | No `/activation/surface` HTTP endpoint | Low | Startup logs + `/execution/control` cover current needs |
| L-6 | No automatic rollback on error rates | Medium | Manual operator halt; circuit-breaker is future work |

## Preparation for S340

S340 should consider:

1. **Smoke verification with canonical surface**: Use the activation surface model to drive smoke test assertions — verify that the binary reports `venue_halted` before gate activation and `venue_live` after.

2. **Activation surface HTTP endpoint**: Wire `ActivationSurfaceQuery`/`ActivationSurfaceReply` through the gateway to expose the full 3D surface via HTTP. This requires the execute binary to report its startup dimensions to the store binary.

3. **Controlled single-order test**: Execute the Phase 1 rollout (gate activated briefly, one fill, immediate re-halt) with the canonical surface model providing structured evidence of state transitions.

4. **Drain semantics investigation**: Evaluate whether JetStream consumer pause can provide deterministic drain behavior on halt, closing limitation L-2.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Canonical activation surface exists | `ActivationSurface` type with `ComputeEffectiveMode` |
| Runtime controls are clear and coherent | Kill switch + staleness + dual checkpoint, all documented |
| Activation does not depend on implicit composition | Three dimensions explicitly composed, truth table verified |
| Prepares smoke and verification | Startup logs, contracts, and operator checklist ready |
