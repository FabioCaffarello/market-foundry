# First Guarded Real-Venue Step

> **Stage:** S90
> **Date:** 2026-03-19
> **Gate Authority:** S89 post-hardening action boundary gate (GO CONDICIONAL)
> **Option Selected:** Option 2 — Activation/secrets/wiring-only

---

## 1. Scope Decision

The S89 gate concluded with **GO CONDICIONAL**, identifying three hard blockers that must be resolved before activation. The S90 stage selected **Option 2: Activation/secrets/wiring-only** — implementing the infrastructure prerequisites without triggering any real venue execution.

This is NOT a first real-venue implementation. It is the wiring layer that makes a future first real-venue step possible and safe.

## 2. What Was Implemented

### HB-S89-1: Credential Infrastructure (RESOLVED)

| Component | Path | Purpose |
|-----------|------|---------|
| `LoadCredentials()` | `internal/application/execution/credentials.go` | Fail-fast credential loading from environment variables |
| `CredentialSet` | `internal/application/execution/credentials.go` | Immutable credential holder with security invariants |
| Unit tests | `internal/application/execution/credentials_test.go` | 5 tests: all present, missing/fail-fast, no keys, has-key, nil safety |
| env_file template | `deploy/configs/execute.env.example` | Placeholder for real credentials |
| .gitignore | `.gitignore` | `*.env` pattern added |
| Run.go integration | `cmd/execute/run.go` | Future venue types call `LoadCredentials()` with fail-fast |

**Security invariants enforced:**
- Credential values are never logged, printed, or included in error messages.
- LoadCredentials fails fast on missing required keys.
- `.env` files are git-ignored.
- Convention: `MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME}`.

### HB-S89-2: Reconciliation Invariant Enforcement (RESOLVED)

| Invariant | Enforcement | Location |
|-----------|-------------|----------|
| RC-1: Fill-to-intent correlation | FillProjectionActor checks intent KV before materializing | `fill_projection_actor.go` |
| RC-2: Quantity boundary | FillProjectionActor rejects fills where `filled > requested` | `fill_projection_actor.go` |
| RC-4: Orphan fill handling | Orphan fills logged at WARN, counted, and rejected | `fill_projection_actor.go` |

**Implementation details:**
- FillProjectionActor gains `intentStore` (read-only access to `EXECUTION_PAPER_ORDER_LATEST` KV).
- RC-1 gate runs after domain validation but before monotonicity guard.
- RC-2 gate parses quantity strings to float64 for comparison.
- New stats counters: `orphaned` (RC-4), `overflowed` (RC-2).
- Stats invariant updated: `received == sum(materialized + skipped* + rejected + orphaned + overflowed + errors)`.
- Intent store unavailability degrades gracefully (RC-1/RC-2 disabled, logged as WARN).

### HB-S89-3: Integration Test Infrastructure (PARTIALLY RESOLVED)

| Component | Path | Purpose |
|-----------|------|---------|
| `make test-integration` | `Makefile` | Target for running integration tests with build tag |

**Note:** The Makefile target is in place. The embedded NATS test harness with 8 scenarios requires a separate implementation step (test infrastructure with `nats-server` embedded). The infrastructure scaffolding (build tag isolation, CI target) is ready.

### PRE-A5: Venue Submit Timeout (RESOLVED)

| Component | Path | Change |
|-----------|------|--------|
| VenueAdapterConfig | `venue_adapter_actor.go` | Added `SubmitTimeout` field |
| VenueAdapterActor.onIntent | `venue_adapter_actor.go` | `context.Background()` → `context.WithTimeout()` |
| VenueConfig | `schema.go` | Added `submit_timeout` field with validation (1s–60s) |
| execute.jsonc | `deploy/configs/execute.jsonc` | Added `"submit_timeout": "10s"` |
| ExecuteSupervisor | `execute_supervisor.go` | Reads `SubmitTimeoutDuration()` from config |

### PRE-A6: Staleness MaxAge Configurable (RESOLVED)

| Component | Path | Change |
|-----------|------|--------|
| VenueConfig | `schema.go` | Added `staleness_max_age` field with validation (30s–600s) |
| `StalenessMaxAgeDuration()` | `schema.go` | Config-driven duration with 120s default |
| execute.jsonc | `deploy/configs/execute.jsonc` | Added `"staleness_max_age": "120s"` |
| ExecuteSupervisor | `execute_supervisor.go` | Uses `StalenessMaxAgeDuration()` instead of hardcoded constant |

## 3. What Was NOT Implemented

| Item | Reason | Target |
|------|--------|--------|
| Real venue adapter | No activation gate ceremony executed yet | S91 |
| Embedded NATS test harness | Requires nats-server embedded dependency | S91 |
| FillTrackerActor | Async polling — not needed until real venue | S91+ |
| Background reconciliation actor | RC-5 stuck detection — PRE-O3 | S95 |
| Prometheus /metrics | PRE-O1 — not required for guarded phase | S95 |
| CI pipeline automation | PRE-O2 — not required for guarded phase | S95 |
| Consumer-projection coupling fix | PRE-O4 — structural risk, not a blocker | S95 |
| Transitional bridge migration | Requires venue-specific intent events | S91+ |

## 4. Reversibility

All changes are additive and backward-compatible:

- Credential loading only activates for non-`paper_simulator` venue types.
- RC-1/RC-2 gates degrade gracefully when intent store is unavailable.
- New config fields have defaults that match previous behavior (`120s`, `10s`).
- New stats counters (`orphaned`, `overflowed`) are additive — existing counters unchanged.
- `make test-integration` is a new target that doesn't affect existing `make test`.

Rolling back to paper-only mode requires zero code changes — just keep `venue.type: "paper_simulator"`.

## 5. Remaining Path to Real Venue

```
S90 (this stage) ── credential infra, RC enforcement, config improvements
     │
     ▼
S91 ── real venue adapter + embedded NATS test harness + venue arch doc + drift rules
     │
     ▼
S92 ── activation gate ceremony (AG-1..AG-17)
     │
     ▼
S93 ── shadow phase (24h real venue)
```
