# Actor Pipeline Venue Path Verification

> **Stage:** S329
> **Date:** 2026-03-21
> **Type:** Verification (operational proof)
> **Predecessor:** S328 (Execute Supervisor Composition)

---

## Purpose

This document records how the composed actor pipeline is verified at the venue
path level. S328 composed the decorator chain; S329 proves it works in the
operational path that `VenueAdapterActor.onIntent()` executes.

---

## Venue Path Under Verification

The production venue path in `VenueAdapterActor.onIntent()` follows this sequence:

```
1. Track "processed" counter
2. Safety gate: kill switch + staleness guard
3. Create submit context with configurable timeout
4. Call a.venue.SubmitOrder() ← composed pipeline
5. On error:
   a. Track error counter
   b. Extract retry metadata from Problem.Details
   c. Log structured error with retry/halt/deadline attributes
6. On success:
   a. Construct VenueOrderFilledEvent from receipt
   b. Publish fill event via NATS
   c. Track filled counters
   d. Log structured success with fills
```

The composed pipeline at step 4 is:

```
Post200Reconciler (outermost, conditional)
  → RetrySubmitter (middle, always)
    .WithHaltChecker(controlStore)
    .WithLogger(logger)
    .WithTracker(tracker)
  → rawAdapter (innermost)
```

---

## Verification Tests (VP-01 through VP-09)

Each VP test mirrors the actor's venue path, composing the pipeline exactly as
`VenueAdapterActor.start()` does and exercising the submit → fill event
construction flow.

| Test | Scenario | Path Verified |
|------|----------|---------------|
| VP-01 | Retry success → fill event | Retry recovers transient failure, fill event built from successful receipt |
| VP-02 | Post-200 recovery → fill event | Body-read-failure recovered via QueryOrder, fill event built from recovered receipt |
| VP-03 | Retry metadata in actor error log | Exhaustion metadata extracted from Problem.Details into structured log |
| VP-04 | Fill event intent field preservation | All critical fields (symbol, side, quantity, risk, correlation, fills) survive composition |
| VP-05 | Tracker counters reflect pipeline | retry_attempts, retry_success, retry_exhausted, retry_halted counters accumulate correctly |
| VP-06 | Halt propagation to actor error path | Kill switch abort surfaces retry_halted through full composition stack |
| VP-07 | Paper mode fill event intact | Retry-only composition (no reconciler) produces correct simulated fill events |
| VP-08 | Retry then post-200 recovery | Full cross-decorator path: transient retry → body-read-failure → reconciliation recovery |
| VP-09 | Safety gate blocks before pipeline | Staleness and kill switch gates verified independently of decorator chain |

---

## What VP Tests Prove Beyond SC Tests

SC-01..SC-07 (S328) proved that the decorators compose correctly in isolation.
VP tests extend this by proving:

1. **Fill event construction works with composed receipts.** The actor builds
   `VenueOrderFilledEvent` from the receipt; VP tests verify that recovered,
   retried, and direct receipts all produce valid fill events.

2. **Actor-level observability extraction works.** The actor extracts retry
   metadata from `Problem.Details` keys (`retry_attempts`, `retry_exhausted`,
   `retry_halted`, `retry_deadline_exceeded`) into structured logs. VP-03
   proves this extraction produces correct log entries.

3. **Tracker counters accumulate across pipeline events.** VP-05 runs three
   scenarios against a shared tracker, verifying that counters from different
   pipeline outcomes accumulate correctly (as they would in production).

4. **Safety gate operates independently of decorators.** VP-09 proves that
   staleness and kill switch checks happen before the decorator chain, as
   designed.

5. **JSON persistence round-trip works for recovered fill events.** VP-02
   proves that a fill event built from a reconciliation-recovered receipt
   survives JSON marshal/unmarshal (required for NATS publish and ClickHouse
   persistence).

---

## Invariants Verified

| ID | Invariant | Evidence |
|----|-----------|----------|
| EC-1 | Deterministic client order ID | VP-08 verifies correct ID passes to reconciler |
| F-1 | No bare errors / Problem type | All VP tests use Problem for error handling |
| PGR-08 | Intent immutability | VP-04 verifies all intent fields preserved |
| INV-REC-1 | No duplicate execution | VP-02, VP-08: reconciler uses query, not re-submit |
| INV-OBS-1 | Zero noise on success | VP-07: no retry counters on first-attempt success |

---

## File

Test file: `internal/application/execution/venue_path_verification_test.go`
