# Composed Venue Path: Evidence and Operational Limitations

> **Stage:** S329
> **Date:** 2026-03-21
> **Type:** Evidence and limits record
> **Predecessor:** S328 (Execute Supervisor Composition)

---

## Evidence Summary

S329 transforms the S328 composition from "wired" to "operationally verified."
The following evidence proves that the composed pipeline capabilities are
effectively used in the real venue path.

---

### E1: Retry Participates in the Real Path

**Evidence:** VP-01, VP-05, VP-08

- Transient 503 failures trigger retry with exponential backoff.
- Retry success produces a valid receipt used for fill event construction.
- `retry_attempts` and `retry_success_after_retry` counters increment.
- Structured logs contain `"retry attempt failed"` and `"retry succeeded"` entries.
- On exhaustion, `retry_exhausted` counter increments and metadata surfaces in
  the Problem returned to the actor's error handling path.

### E2: Post-200 Reconciliation Participates in the Real Path

**Evidence:** VP-02, VP-08

- Body-read-failure-after-200 passes through RetrySubmitter (non-retryable).
- Post200Reconciler queries the venue using the deterministic client order ID.
- Recovered receipt is used for fill event construction with correct fills.
- Fill event survives JSON round-trip (persistence-ready).
- VP-08 proves the cross-decorator path: retry → body-read-failure → recovery.

### E3: Observability Hooks Generate Useful Signals

**Evidence:** VP-01, VP-03, VP-05, VP-07, VP-08

- Structured logs contain component tag (`retry-submitter`), attempt counts,
  and error messages for every non-terminal retry failure.
- Actor-level error log extraction (the code path at `onIntent` lines 234-242)
  correctly surfaces `retry_attempts`, `retry_exhausted`, `retry_halted`, and
  `retry_deadline_exceeded` from `Problem.Details`.
- Health tracker counters accumulate correctly across multiple pipeline invocations.
- First-attempt success produces zero retry noise (VP-07).

### E4: Submit/Fill/Persist Behavior Intact

**Evidence:** VP-04, VP-07

- All 12 critical intent fields survive the composed pipeline:
  source, symbol, timeframe, side, quantity, filled_quantity, status,
  correlation_id, causation_id, risk.type, risk.disposition, fills.
- Fill event JSON serialization produces valid payloads > 100 bytes.
- Paper mode fills preserve `Simulated=true` flag.

### E5: Safety Gate Operates Independently

**Evidence:** VP-09

- Staleness guard blocks intents older than the configured max age.
- Kill switch blocks intents regardless of freshness.
- Both gates operate before the decorator chain (no pipeline invocation on blocked intents).

---

## Operational Limitations

### L1: No Live NATS Verification

VP tests exercise the actor's venue path logic (submit → fill event construction)
but do not connect to NATS. The fill publisher and consumer are not tested in VP
scope. This is intentional: NATS integration is S330 scope (PWT-4).

**Risk:** Low. The NATS publisher is a thin wrapper already tested separately.
The fill event structure is verified via JSON round-trip in VP-02.

### L2: No Real HTTP Venue in VP Tests

VP tests use `fakeVenue` stubs, not the real `BinanceFuturesTestnetAdapter`.
This is by design: VP tests verify the actor path composition, not venue HTTP
behavior (covered by S308/S314 adapter tests and S317 round-trip tests).

**Risk:** Low. The adapter tests are comprehensive. VP tests prove the
composition wiring, not the adapter implementation.

### L3: Safety Gate Tested Independently of Composition

VP-09 tests the safety gate in isolation. In production, the gate runs inside
`onIntent()` before the composed pipeline call. The actor method itself is not
directly unit-testable without a hollywood actor engine and NATS dependencies.

**Risk:** Low. The gate and pipeline are sequential steps with no shared state
beyond the intent. Composition correctness is proven by the combination of
VP-09 (gate) and VP-01..VP-08 (pipeline).

### L4: Retry Policy Not Config-Driven

VP tests use `DefaultRetryPolicy()` or explicit test policies. The production
retry policy is hardcoded in `DefaultRetryPolicy()`. Config-driven retry
policy is deferred to post-tranche.

### L5: No Multi-Venue Routing

All VP tests exercise a single venue path. Multi-venue routing would require
a router before the decorator chain, which is out of scope.

### L6: No Distributed Tracing (OpenTelemetry)

Observability is limited to structured logs and health counters. OpenTelemetry
span instrumentation is deferred.

---

## Limits Deferred From Prior Stages (Unchanged)

| Limit | Origin | Status |
|-------|--------|--------|
| Retry policy not config-driven | R-S323-3 | Deferred to post-tranche |
| Reconciliation timeout not config-driven | S322 | Deferred to post-tranche |
| No circuit breaker | Design scope | Deferred to post-tranche |
| No OpenTelemetry/tracing | Design scope | Deferred to post-tranche |
| No multi-venue routing | Design scope | Deferred |
