# Retry Deadline, Halt Check, and Abort Semantics

> **Stage:** S323
> **Scope:** Formal semantic contract for retry coordination hardening

## 1. Overview

This document defines the semantic invariants governing the retry coordination mechanisms introduced in S323: the global retry deadline (budget), the inter-attempt kill switch check, and the abort behavior when either triggers.

## 2. Invariants

### INV-RC-1: Deadline Budget Independence

The global deadline operates independently of the per-submit context deadline. The deadline governs the total wall-clock time of the retry sequence. The context deadline governs individual submit calls.

**Why**: A single submit call may have a 10s context timeout, but the retry sequence as a whole should not run indefinitely even if each individual call completes quickly.

### INV-RC-2: Deadline Monotonicity

The deadline is computed once at loop entry (`now + Deadline`) and never adjusted. The clock is sampled before each attempt. Once exceeded, the loop aborts even if subsequent calls might succeed.

**Why**: Prevent "just one more try" creep. A fixed budget makes retry duration predictable.

### INV-RC-3: Halt Fail-Open

When no `GateChecker` is configured or the halt check times out, the retry loop continues (fail-open). This is consistent with SafetyGate's fail-open semantics.

**Why**: Retry coordination hardening must not make the system _less_ available. A missing or slow KV store should not block retries that would otherwise succeed.

### INV-RC-4: Halt Check Positioning

The kill switch is checked after a failed attempt and before the backoff sleep, never before the first attempt. The pre-submit halt check is SafetyGate's responsibility.

**Why**: Avoid double-checking the kill switch at the same logical point. SafetyGate owns the pre-submit gate; RetrySubmitter owns the mid-retry gate.

### INV-RC-5: Mutually Exclusive Abort Metadata

Only one of `retry_exhausted`, `retry_halted`, or `retry_deadline_exceeded` is set in Problem.Details for any given abort. They never appear together.

**Why**: The consumer of the Problem needs to know _which_ termination condition fired. Overlapping flags would create ambiguity.

### INV-RC-6: Idempotency Preservation

The halt check and deadline check do not affect request content. The same `VenueOrderRequest` (with deterministic client order ID) is passed to every submit attempt, regardless of how the loop terminates.

**Why**: Client order ID stability is the foundation of retry safety (EC-1/S313). No coordination change may violate this.

### INV-RC-7: No Submission After Abort

Once the loop decides to abort (due to deadline, halt, context, or exhaustion), no further `SubmitOrder` call is made. The last error is returned with metadata.

**Why**: After the control plane signals halt, the system must not send additional orders to the venue.

## 3. Abort Taxonomy

| Abort Reason | Check Point | Metadata Key | Retryable? |
|-------------|-------------|-------------|-----------|
| Context cancelled/expired | Before each attempt, during backoff | `retry_exhausted` | Depends on last error |
| Global deadline exceeded | Before each attempt | `retry_deadline_exceeded` | Depends on last error |
| Kill switch halted | After failed attempt, before backoff | `retry_halted` | Depends on last error |
| Max attempts exhausted | After last attempt | `retry_exhausted` | Depends on last error |
| Non-retryable error | After any attempt | (none — error returned as-is) | false |

## 4. Budget Exhaustion Semantics

When the deadline is exceeded:

1. If at least one attempt has been made, the Problem from the last attempt is returned, enriched with `retry_deadline_exceeded: true` and `retry_attempts: N`.
2. If the deadline is exceeded before the first attempt (possible if `Deadline` is very small), a fresh Problem is returned with code `Unavailable` and message "retry deadline exceeded before venue submit".

The budget does not cancel in-flight submit calls. If a submit is already running when the budget expires, it will complete (or be cancelled by its own context deadline). The budget is checked only at loop iteration boundaries.

## 5. Halt Abort Semantics

When the kill switch triggers between attempts:

1. The Problem from the last failed attempt is returned, enriched with `retry_halted: true` and `retry_attempts: N`.
2. No backoff sleep occurs.
3. No further submit calls are made.

The halt check uses a 2-second timeout, consistent with SafetyGate's `gateReadTimeout`. If the KV store is unreachable, the check fails open (returns `false`), and the retry continues.

## 6. Composition

```
VenueAdapterActor.onIntent()
  → SafetyGate.Check()          [pre-submit: kill switch + staleness]
  → Post200Reconciler.SubmitOrder()
    → RetrySubmitter.SubmitOrder()
      → [deadline check]         [S323: global budget]
      → inner.SubmitOrder()
      → [halt check]             [S323: kill switch between attempts]
      → [backoff sleep]
      → (repeat)
    → [body-read-failure?]       [S322: reconciliation]
    → QueryOrder()
```

## 7. Testing Contract

| Test | Invariant Verified |
|------|-------------------|
| `TestRetry_DeadlineExceeded_AbortsLoop` | INV-RC-1, INV-RC-2, INV-RC-7 |
| `TestRetry_DeadlineZero_NoDeadlineEnforced` | INV-RC-1 (zero = disabled) |
| `TestRetry_HaltChecker_HaltsDuringRetry` | INV-RC-4, INV-RC-5, INV-RC-7 |
| `TestRetry_HaltChecker_NotHalted_RetriesNormally` | INV-RC-3 |
| `TestRetry_HaltChecker_Nil_FailOpen` | INV-RC-3 |
| `TestRetry_HaltChecker_BecomesHaltedMidLoop` | INV-RC-4 |
| `TestRetry_DeadlineAndHalt_DeadlineWinsWhenBothTrigger` | INV-RC-5 |
| `TestRetry_SuccessBeforeDeadline_NoDeadlineMetadata` | INV-RC-6 |
| `TestRetry_PreservesDeterministicClientOrderID` | INV-RC-6 (pre-existing) |

## 8. Limitations

| ID | Limitation | Risk | Rationale |
|----|-----------|------|-----------|
| L-RC-1 | Deadline does not cancel in-flight submits | Low | Per-submit context deadline handles this; adding cancellation would risk partial writes |
| L-RC-2 | Halt check uses fixed 2s timeout | Low | Consistent with SafetyGate; configurable timeout is future work |
| L-RC-3 | No jitter on halt check timing | Very Low | Halt checks are infrequent (once per retry, max 2-3 per sequence) |
| L-RC-4 | Budget does not account for backoff time already spent | Low | Budget is checked at attempt boundary; backoff time is small relative to budget |
