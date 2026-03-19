# Stage S80 — First Guarded Venue Execution Step

**Status**: COMPLETE
**Date**: 2026-03-18

## Executive Summary

S80 implements the **first guarded venue execution step**: a standalone `execute` binary
that consumes execution intents from derive, processes them through a `PaperVenueAdapter`,
and publishes fill events — all without contacting any real exchange. This closes the two
BLOCKING gaps identified by S79 (no execute binary, no VenuePort implementation) while
maintaining all safety invariants established by S74-S79.

## Pre-condition Validation

| Requirement | Evidence | Verdict |
|-------------|----------|---------|
| S79 operational validation complete | 87 unit tests pass, 16-step smoke verified | PASS |
| All 5 S74 hard blockers resolved | HB-1 through HB-5 designed (S75) and implemented (S76-S78) | PASS |
| Activation gates 1-10 passed | S76-S79 reports confirm all gates | PASS |
| Paper execution domain validated | Zero domain bugs, pipeline proven end-to-end | PASS |

**Pre-condition verdict**: S80 authorized to proceed.

## Scope Decision

**Chosen step**: Execute binary with PaperVenueAdapter (Option 1 from stage directive)

This is the minimum viable step because:
- It resolves both BLOCKING gaps without touching real venues
- Kill switch, staleness guard, and trace propagation are enforced
- Store authority is preserved (execute does not write to KV)
- No OMS, multi-venue, portfolio, or framework abstraction

## Changes Made

### New Files

| File | Purpose |
|------|---------|
| `cmd/execute/main.go` | Binary entry point |
| `cmd/execute/run.go` | Runtime wiring: engine, venue adapter, health server |
| `cmd/execute/go.mod` | Module declaration |
| `internal/actors/scopes/execute/execute_supervisor.go` | Root actor: spawns consumer + venue adapter |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Core: kill switch → staleness → VenuePort → publish fill |
| `internal/actors/scopes/execute/messages.go` | Actor message types |
| `internal/application/ports/venue.go` | VenuePort interface |
| `internal/application/execution/paper_venue_adapter.go` | PaperVenueAdapter: simulated fills |
| `internal/application/execution/paper_venue_adapter_test.go` | 5 tests for PaperVenueAdapter |
| `internal/application/execution/staleness_guard.go` | Intent age check utility |
| `internal/application/execution/staleness_guard_test.go` | 4 tests for StalenessGuard |
| `deploy/configs/execute.jsonc` | Execute service configuration |
| `docs/architecture/first-guarded-venue-execution-step.md` | Architecture document |
| `docs/architecture/execute-runtime-and-activation-model.md` | Runtime and activation model |

### Modified Files

| File | Change |
|------|--------|
| `internal/domain/execution/execution.go` | Added `StatusSent` to lifecycle, updated `ValidStatus` and `validTransitions` |
| `internal/domain/execution/events.go` | Added `VenueOrderFilledEvent` type |
| `internal/adapters/nats/execution_registry.go` | Added venue market order specs (fill stream, consumer, KV bucket, staleness constant) |
| `internal/adapters/nats/execution_publisher.go` | Added `PublishFill` method, ensured fill stream on Start |
| `internal/shared/settings/schema.go` | Registered `venue_market_order` family + dependency rule |
| `go.work` | Added `./cmd/execute` module |

## Test Results

```
ok  internal/domain/execution       — all pass (including StatusSent transition tests)
ok  internal/application/execution  — all pass (9 new tests: 5 venue adapter + 4 staleness)
ok  internal/adapters/nats          — all pass
ok  internal/actors/scopes/derive   — all pass
ok  internal/actors/scopes/store    — all pass
ok  internal/shared/settings        — all pass
```

**Total new tests**: 9 (5 PaperVenueAdapter + 4 StalenessGuard)
**All existing tests**: GREEN (no regressions)
**Build verification**: all 4 binaries compile (derive, store, gateway, execute)

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No OMS built | COMPLIANT — no order management logic |
| No multi-venue opened | COMPLIANT — single PaperVenueAdapter only |
| No portfolio opened | COMPLIANT — no position or portfolio tracking |
| No contradiction with S74-S79 | COMPLIANT — all gates respected |
| Kill switch effective | COMPLIANT — checked before every intent |
| Auditability preserved | COMPLIANT — trace IDs flow through, structured logging |

## Activation Gates (S80 Tier 3)

| Gate | Status | Evidence |
|------|--------|----------|
| G-11: Execute binary compiles | **PASS** | `go build ./cmd/execute/...` succeeds |
| G-12: PaperVenueAdapter tests | **PASS** | 5 unit tests, all green |
| G-13: Fill projection in store | **DEFERRED** | Needs store-side fill consumer + projection actor |
| G-14: Status propagation e2e | **DEFERRED** | Needs full pipeline integration |
| G-15: Kill switch halts execute | **PARTIAL** | Logic implemented, needs NATS integration test |
| G-16: Config symmetry | **PASS** | execute.jsonc validates successfully |
| G-17: Drift rules ED-6-9 | **DEFERRED** | raccoon-cli update needed |

## Limits Encountered

1. **Consumer reuses paper_order stream**: Execute consumes from `execution.events.paper_order.submitted.>` rather than a separate `venue_market_order` subject. This is correct for S80 (derive produces paper_order intents), but venue-specific subjects require derive changes in S81+.

2. **No fill projection in store**: Execute publishes fill events to `EXECUTION_FILL_EVENTS`, but store does not yet consume or project them. Fill projection requires a new consumer + projection actor in store.

3. **No NATS integration tests**: Kill switch and staleness are unit-tested but not integration-tested with live NATS. This is a known S79 gap carried into S80.

4. **Hardcoded venue adapter**: `run.go` creates `PaperVenueAdapter` directly — no config-driven adapter selection yet.

## Items Explicitly Deferred to S81+

| Item | Rationale |
|------|-----------|
| Store fill projection (G-13) | Requires new consumer + projection actor + KV bucket wiring |
| End-to-end status propagation (G-14) | Requires store integration + smoke test |
| NATS integration tests for kill switch (G-15) | Requires embedded NATS server test harness |
| Drift rules ED-6 through ED-9 (G-17) | Requires raccoon-cli Rust changes |
| Config-driven venue adapter selection | Requires venue config schema + adapter factory |
| Venue-specific derive subjects | Requires derive to produce `venue_market_order` type |
| HTTP query surface for venue orders | Requires gateway + store wiring |
| Fill history bucket | Requires separate KV bucket, not latest-only |
| Consumer redelivery behavior test | Low priority, deferred from S79 |

## Conclusion

S80 delivers the **first operational execute binary** for market-foundry. The binary is
structurally complete, tested, and guarded — but operates strictly in paper mode. The
architecture is designed to accept real venue adapters in future stages without structural
changes to the actor topology or message flow.

The action boundary has been **crossed at the smallest possible step**: simulated fills,
kill-switched, staleness-guarded, trace-propagated, and auditable. Real venue integration
remains gated behind the full 17-gate activation ceremony.
