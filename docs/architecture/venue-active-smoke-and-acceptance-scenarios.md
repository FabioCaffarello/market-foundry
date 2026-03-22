# Venue-Active Smoke and Acceptance Scenarios

> S340 — Venue Activation Wave (S337–S342)

## Purpose

This document defines the canonical acceptance scenarios for venue activation
and the smoke procedures that validate them. It is the operational reference
for proving that activation transitions work correctly before any live
verification (S341).

## Acceptance Scenarios

### AC-1: Inactive to Active (off to on)

**Precondition:** Binary deployed with venue adapter, gate halted, credentials present.
**Action:** Operator sets gate to `active` via `PUT /execution/control`.
**Postcondition:** Effective mode transitions from `venue_halted` to `venue_live`. `IsLive()` returns true.

| Step | Adapter | Gate | Credentials | Effective | IsLive |
|------|---------|------|-------------|-----------|--------|
| Before | venue | halted | present | venue_halted | false |
| After | venue | active | present | venue_live | **true** |

**What this proves:** The gate dimension alone controls the off-to-on
transition when adapter and credentials are already in place.

### AC-2: Active to Halt (on to halt)

**Precondition:** Effective mode is `venue_live`.
**Action:** Operator sets gate to `halted` via `PUT /execution/control` with reason and identity.
**Postcondition:** Effective mode transitions to `venue_halted`. `IsLive()` returns false. Audit fields (reason, updated_by, updated_at) are preserved.

| Step | Adapter | Gate | Credentials | Effective | IsLive |
|------|---------|------|-------------|-----------|--------|
| Before | venue | active | present | venue_live | true |
| After | venue | halted | present | venue_halted | **false** |

**What this proves:** The kill-switch immediately blocks new execution.
Audit fields survive the round-trip through NATS KV.

### AC-3: Halt to Rollback (controlled return to paper)

**Precondition:** Effective mode is `venue_halted`.
**Action:** Operator halts gate, stops binary, changes config to `paper_simulator`, removes credentials, restarts binary, resumes gate.
**Postcondition:** Effective mode is `paper`. `IsLive()` returns false. `CanReachVenue()` returns false.

| Step | Adapter | Gate | Credentials | Effective | IsLive | CanReachVenue |
|------|---------|------|-------------|-----------|--------|---------------|
| Before | venue | halted | present | venue_halted | false | true |
| After | paper | active | absent | paper | false | **false** |

**What this proves:** Full rollback completely exits the venue path.
No residual venue state remains.

### AC-4: Full Lifecycle Round Trip

The canonical lifecycle sequence:

```
paper → venue_halted → venue_live → venue_halted → paper
```

Each transition uses the minimum operator action required:
1. **paper → venue_halted**: Binary restart with venue adapter config + credentials + gate halted.
2. **venue_halted → venue_live**: `PUT /execution/control {"status":"active"}`.
3. **venue_live → venue_halted**: `PUT /execution/control {"status":"halted"}`.
4. **venue_halted → paper**: Binary restart with paper config, no credentials.

### AC-5: Degraded Mode Safety

When the venue adapter is loaded but credentials are absent, the effective
mode is `venue_degraded`. This mode is **never live** — `IsLive()` returns
false. This guards against accidental activation without proper credentials.

### AC-6: Audit Field Preservation

Every gate mutation carries `reason`, `updated_by`, and `updated_at` fields.
These fields survive the NATS KV round-trip and are accessible through both
the HTTP control surface and the `ActivationSurface` domain type.

## Smoke Procedures

### Unit Test Gate

```bash
go test -count=1 -run "TestActivationAcceptance_" ./internal/domain/execution/...
```

Validates all six scenarios (AC-1 through AC-6) as pure domain-level tests.
No infrastructure required.

### HTTP Control Surface Smoke

```bash
make smoke-activation
# or: ./scripts/smoke-activation.sh
```

Validates AC-1, AC-2, and AC-3 against a live stack via HTTP.
Requires: `make up && make seed`.

| Phase | Scenario | Validation |
|-------|----------|------------|
| 1 | Readiness | Gateway + NATS + control surface reachable |
| 2 | AC-1 | halted → active via PUT, confirmed via GET |
| 3 | AC-2 | active → halted via PUT, reason preserved |
| 4 | AC-3 | halted → active (rollback), confirmed via GET |
| 5 | Unit tests | All TestActivationAcceptance_ tests pass |

### Safety Guarantees

- The smoke always restores the gate to `active` on exit (trap on EXIT).
- No real venue orders are submitted — smoke operates on the control surface only.
- Temp files in `/tmp/s340_*.json` are cleaned up at the end of the run.

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `BASE_URL` | `http://127.0.0.1:8080` | Gateway base URL for HTTP control surface |

## Prerequisites

| Requirement | How to satisfy |
|-------------|---------------|
| Running stack | `make up` |
| Seeded config | `make seed` |
| Go toolchain | Available in PATH |
| Docker Compose | Available in PATH |

## Relationship to Prior Stages

| Stage | Contribution |
|-------|-------------|
| S335 | Kill-switch control path — Phase 7 of smoke-live-stack.sh |
| S338 | Activation policy, rollout/rollback model |
| S339 | Canonical activation surface, ComputeEffectiveMode, domain types |
| **S340** | **Acceptance scenarios, smoke script, reproducible proof** |
| S341 | Controlled live verification (next) |
