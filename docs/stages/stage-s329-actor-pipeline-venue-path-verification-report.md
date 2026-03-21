# S329 — Actor Pipeline Venue Path Verification Report

> **Stage:** S329
> **Date:** 2026-03-21
> **Type:** Verification (operational proof)
> **Phase:** 31b (Production Wiring Tranche)
> **Predecessor:** S328 (Execute Supervisor Composition)
> **Successor:** S330 (NATS Integration Verification) → S331 (Production Wiring Gate)

---

## Executive Summary

S329 proves that the decorator pipeline composed in S328 participates effectively
in the real venue path. Nine verification tests (VP-01 through VP-09) exercise
the exact flow that `VenueAdapterActor.onIntent()` follows: safety gate →
composed submit → fill event construction → observability extraction.

All tests pass. Full existing suite passes (32s) with zero regressions.

**S329 verdict: COMPLETE. Composed pipeline operationally verified.**

---

## Pipeline Validated

```
VenueAdapterActor.onIntent()
  │
  ├── Track "processed" counter
  ├── Safety gate: kill switch + staleness guard  ← VP-09
  ├── Create submit context with timeout
  ├── Call a.venue.SubmitOrder()  ← composed pipeline
  │     │
  │     ├── Post200Reconciler (outermost)  ← VP-02, VP-08
  │     │     └── RetrySubmitter (middle)  ← VP-01, VP-03, VP-05, VP-06, VP-08
  │     │           ├── .WithHaltChecker   ← VP-06
  │     │           ├── .WithLogger        ← VP-01, VP-03, VP-08
  │     │           ├── .WithTracker       ← VP-05
  │     │           └── rawAdapter         ← VP-04, VP-07
  │     │
  │     └── On error: extract retry metadata → structured log  ← VP-03
  │
  ├── Construct VenueOrderFilledEvent  ← VP-01, VP-02, VP-04, VP-07, VP-08
  ├── Publish fill event (NATS)  ← deferred to S330
  └── Track filled counters  ← VP-05
```

---

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `internal/application/execution/venue_path_verification_test.go` | New: 9 venue path verification tests (VP-01 through VP-09) | ~420 lines |
| `docs/architecture/actor-pipeline-venue-path-verification.md` | New: venue path verification architecture doc | ~100 lines |
| `docs/architecture/composed-venue-path-evidence-and-operational-limitations.md` | New: evidence and limits record | ~120 lines |
| `docs/stages/stage-s329-actor-pipeline-venue-path-verification-report.md` | New: this report | ~190 lines |

---

## Verification Tests

| Test | Scenario | Proves |
|------|----------|--------|
| VP-01 | Retry success → fill event | Retry recovers transient failure; fill event has correct receipt |
| VP-02 | Post-200 recovery → fill event | Reconciler recovers body-read-failure; fill event JSON round-trips |
| VP-03 | Retry metadata in actor error log | Actor extraction of retry_attempts/retry_exhausted from Problem.Details |
| VP-04 | Fill event field preservation | All 12 critical intent fields survive composed pipeline |
| VP-05 | Tracker counters reflect pipeline | Cumulative counters: retry_attempts, success, exhausted, halted |
| VP-06 | Halt propagation through composition | Kill switch abort surfaces retry_halted through reconciler |
| VP-07 | Paper mode fill event intact | Retry-only (no reconciler); simulated fill preserved |
| VP-08 | Retry then post-200 recovery | Cross-decorator path: 503 retry → body-read-failure → recovery |
| VP-09 | Safety gate blocks before pipeline | Staleness and kill switch operate independently of decorators |

---

## Evidence Summary

| Capability | Evidence | Status |
|-----------|----------|--------|
| Retry in real path | VP-01, VP-05, VP-08: transient failures retried, counters/logs emitted | VERIFIED |
| Post-200 recovery in real path | VP-02, VP-08: body-read-failure recovered, fill event built | VERIFIED |
| Observability hooks generate signals | VP-01, VP-03, VP-05: logs and counters from pipeline events | VERIFIED |
| Submit/fill/persist behavior intact | VP-04, VP-07: 12 fields preserved, JSON serialization works | VERIFIED |
| Safety gate independent of pipeline | VP-09: staleness and kill switch block before decorators | VERIFIED |

---

## Acceptance Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Pipeline proves effective use of composed capabilities | PASS | VP-01, VP-02, VP-08 |
| Submit/fill/persist/read intact | PASS | VP-04, VP-07 |
| Observability and recovery visible in real path | PASS | VP-02, VP-03, VP-05 |
| Wiring transformed into operational evidence | PASS | All VP tests exercise actor path flow |

---

## Guard Rails Compliance

| Rail | Status |
|------|--------|
| No new order types opened | COMPLIANT |
| No multi-venue opened | COMPLIANT |
| No E2E inflation | COMPLIANT — 9 focused VP tests, no NATS/infra deps |
| No indirect asserts masking absent usage | COMPLIANT — all tests exercise composed pipeline directly |

---

## Remaining Limits

| Limit | Origin | Deferred To |
|-------|--------|-------------|
| No live NATS in VP tests | S329 scope decision | S330 (PWT-4) |
| No real HTTP venue in VP tests | S329 scope decision | S308/S314 adapter tests cover |
| Retry policy not config-driven | R-S323-3 | Post-tranche |
| Reconciliation timeout not config-driven | S322 | Post-tranche |
| No circuit breaker | Design scope | Post-tranche |
| No OpenTelemetry/tracing | Design scope | Post-tranche |

---

## Predecessor Chain

```
S306 (Venue Readiness Charter)
  → S312 (Adapter Hardening Tranche Charter)
    → S315 (Foundational Tranche Gate — PASS)
      → S321 (Venue Closure Tranche Charter)
        → S326 (Venue Progression Evidence Gate — CLOSED)
          → S327 (Production Wiring Tranche Charter — FROZEN)
            → S328 (Execute Supervisor Composition — COMPLETE)
              → S329 (Venue Path Verification — COMPLETE) ← this stage
                → S330 (NATS Integration Verification)
                  → S331 (Production Wiring Gate)
```

---

## Preparation for S330

S329 closes the application-layer verification. S330 should focus on:

1. **PWT-4 — NATS integration test:** Intent → safety gate → composed pipeline
   → fill event → NATS publish. Requires a test NATS server.
2. **Startup log verification:** Confirm the startup log emits all decorator
   state fields (`retry_submitter=true`, `post200_reconciler=true/false`,
   `retry_halt_checker=true/false`).
3. **Graceful degradation:** Control store unavailable → retry without halt
   checker. Verify the fail-open behavior in a live context.

---

## Verdict

**S329 COMPLETE. Composed pipeline operationally verified.**

The decorator pipeline composed in S328 is proven to work in the real venue
path. Retry, post-200 reconciliation, and observability hooks all participate
effectively. Submit/fill/persist behavior is intact. The tranche can proceed
to NATS integration verification (S330) and gate closure (S331).
