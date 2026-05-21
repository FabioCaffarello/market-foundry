# S328 — Execute Supervisor Composition Report

> **Stage:** S328
> **Date:** 2026-03-21
> **Type:** Implementation (mechanical composition)
> **Phase:** 31b (Production Wiring Tranche)
> **Predecessor:** S327 (Production Wiring Tranche Charter — FROZEN)
> **Successor:** S329 (Integration Verification) or S330 (Production Wiring Gate)

---

## Executive Summary

S328 composes the RetrySubmitter, Post200Reconciler, and observability hooks into
the production venue adapter pipeline. All three PWT items originally planned across
S328 and S329 are completed in this single stage because the composition was
mechanical and the interface contracts were fully satisfied.

The decorator chain is:

```
Post200Reconciler → RetrySubmitter(+hooks) → rawAdapter
```

All 9 invariants remain preserved. Zero regressions in the full test suite.
Seven new composition tests verify the decorator interplay.

**S328 verdict: COMPLETE. PWT-1, PWT-2, PWT-3: DONE.**

---

## Items Completed

| Item | Description | Status | Evidence |
|------|-------------|--------|----------|
| PWT-1 | RetrySubmitter around adapter | DONE | `venue_adapter_actor.go:start()` lines 120-134 |
| PWT-2 | Post200Reconciler around RetrySubmitter | DONE | `venue_adapter_actor.go:start()` lines 136-139 |
| PWT-3 | WithHaltChecker + WithLogger + WithTracker | DONE | `venue_adapter_actor.go:start()` lines 127-133 |

---

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Added `VenueQuery` to config, `venue` field, decorator composition in `start()`, use `a.venue` in `onIntent()` | ~40 lines added |
| `internal/actors/scopes/execute/execute_supervisor.go` | Added `venueQuery` field, updated constructor and config wiring | ~5 lines changed |
| `cmd/execute/run.go` | `buildVenueAdapter` returns `venueAdapterResult` with both ports | ~15 lines changed |
| `internal/application/execution/supervisor_composition_test.go` | New: 7 composition tests (SC-01 through SC-07) | ~260 lines added |

### Documentation

| File | Content |
|------|---------|
| `docs/architecture/execute-supervisor-composition-of-retry-reconciler-and-observability.md` | Composition site, stack, bootstrap flow, hooks, invariants |
| `docs/architecture/venue-pipeline-decorator-order-invariants-and-limits.md` | Canonical order, invariants (INV-DO-1..4), error flow, limits |
| `docs/stages/stage-s328-execute-supervisor-composition-report.md` | This report |

---

## Composition Tests

| Test | Scenario | Verifies |
|------|----------|----------|
| SC-01 | Retry 503 → body-read-failure → reconciliation recovery | Full stack interplay |
| SC-02 | Success on first attempt | No decorator interference on happy path |
| SC-03 | Non-retryable error | Passes through both decorators unchanged |
| SC-04 | Halt checker abort | Kill switch stops retry, metadata surfaces |
| SC-05 | No query port (paper mode) | Retry-only composition works |
| SC-06 | Retry exhaustion metadata | Metadata surfaces through reconciler |
| SC-07 | Structured log component tag | Logger hook produces correct tags |

All 7 tests pass. Full existing suite (32s) passes with zero regressions.

---

## Design Decisions

### D1: Composition in `VenueAdapterActor.start()`, not in bootstrap

The control store (needed for `WithHaltChecker`) is created inside
`VenueAdapterActor.start()`. Moving it to bootstrap would require separating
the NATS connection lifecycle from the actor lifecycle, which is a design change
outside S328 scope. Composing in `start()` is the minimal-change approach.

### D2: `VenueQuery` as explicit config field, not type assertion

The S327 charter suggested checking `venue.(ports.VenueQueryPort)` via type
assertion. Instead, `VenueQuery` is passed as an explicit config field because:
- Type assertions on interfaces are fragile when the concrete type is wrapped.
- Explicit fields make the capability visible in the config struct.
- The bootstrap code knows the concrete type and can set the field directly.

### D3: All three PWT items in one stage

S327 planned PWT-1 + PWT-3 for S328 and PWT-2 for S329. Since the composition
is mechanical and all interface contracts were satisfied, splitting into two
stages would add ceremony without reducing risk. PWT-2 is included here.

---

## Invariant Verification

| ID | Invariant | Status | Evidence |
|----|-----------|--------|---------|
| EC-1 | Deterministic client order ID | Preserved | ID generation unchanged |
| EC-3 | Per-request deadline | Preserved | Each layer has own deadline |
| F-1 | No bare errors / Problem type | Preserved | All decorators use Problem |
| F-4 | Credential redaction | Preserved | Adapter internals unchanged |
| RF-1 | Retryable flag accuracy | Preserved | Classification unchanged |
| PGR-08 | Intent immutability | Preserved | SC-01..SC-07 verify |
| INV-REC-1 | No duplicate execution | Preserved | Reconciler uses GET |
| INV-RC-1 | Deadline independence | Preserved | Fresh context in reconciler |
| INV-OBS-1 | Zero noise on success | Preserved | SC-02 verifies |

---

## Gaps Closed

| Gap | Origin | Closed By |
|-----|--------|-----------|
| RetrySubmitter not in production path | S319, S326 | PWT-1 |
| Post200Reconciler not in production path | S322, S326 | PWT-2 |
| Observability hooks not wired | R-S323-3, R-S324-1 | PWT-3 |

---

## Remaining Limits

| Limit | Origin | Deferred To |
|-------|--------|-------------|
| Retry policy not config-driven | R-S323-3 | Post-tranche |
| Reconciliation timeout not config-driven | S322 | Post-tranche |
| No circuit breaker | Design scope | Post-tranche |
| No OpenTelemetry/tracing | Design scope | Post-tranche |
| PWT-4 (integration test of composed pipeline) | S327 charter | S330 |

---

## Preparation for S329 / S330

With PWT-1, PWT-2, and PWT-3 completed in S328, the tranche plan adjusts:

- **S329** is now optional (PWT-2 already done). Can be repurposed for any
  discovered friction, or skipped.
- **S330** should proceed with PWT-4: integration test of the full composed
  chain with NATS infrastructure, verifying end-to-end flow.
- **S331** is the tranche gate, verifying all exit criteria.

Recommended S330 focus:
1. Integration test exercising: intent → safety gate → composed pipeline → fill event.
2. Verify startup log contains all decorator state fields.
3. Verify graceful degradation: control store unavailable → retry without halt checker.

---

## Predecessor Chain

```
S306 (Venue Readiness Charter)
  → S312 (Adapter Hardening Tranche Charter)
    → S315 (Foundational Tranche Gate — PASS)
      → S321 (Venue Closure Tranche Charter)
        → S326 (Venue Progression Evidence Gate — CLOSED)
          → S327 (Production Wiring Tranche Charter — FROZEN)
            → S328 (Execute Supervisor Composition) ← this stage
              → S330 (Integration Verification)
                → S331 (Production Wiring Gate)
```

---

## Verdict

**S328 COMPLETE. PWT-1, PWT-2, PWT-3: DONE.**

The decorator pipeline is composed, tested, and documented. The main mechanical
gap identified by S326 is closed. The tranche can proceed to integration
verification (S330) and gate closure (S331).
