# Stage S323 — Retry Coordination Hardening Report

> **Status:** Complete
> **Predecessor:** S319 (Retry Infrastructure), S320 (Venue Failure Path Verification), S322 (Post-200 Reconciliation)
> **Scope:** Hardening of retry coordination: global deadline, inter-attempt kill switch check, abort semantics

## 1. Executive Summary

S320 identified two medium-to-low-risk coordination gaps left in the retry path: absence of a global retry deadline (R-S320-2) and kill switch blindness during retry backoff (R-S320-3). S323 closes both gaps with minimal, backward-compatible changes to `RetrySubmitter`.

The solution adds:
- A `Deadline` field to `RetryPolicy` for global retry budget enforcement.
- A `WithHaltChecker()` method on `RetrySubmitter` for inter-attempt kill switch awareness.
- Structured abort metadata (`retry_deadline_exceeded`, `retry_halted`) for observability.
- `TestWithNowFn()` for deterministic deadline testing.
- 8 new tests covering deadline exhaustion, halt-during-retry, fail-open, mid-loop halt transition, dual-trigger precedence, and success-within-deadline.

**Key result**: R-S320-2 and R-S320-3 are closed. Retry sequences now have bounded duration and respect control plane halt signals between attempts. Zero regressions in the existing test suite.

## 2. Hardening Delivered

### 2.1 Global Retry Deadline

`RetryPolicy.Deadline` defines a wall-clock budget for the entire retry sequence. When non-zero, the loop computes an absolute deadline at entry and checks before each attempt. Default: 10 seconds.

```go
policy := RetryPolicy{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    2 * time.Second,
    Factor:      2.0,
    Deadline:    10 * time.Second,  // NEW
}
```

### 2.2 Inter-Attempt Kill Switch Check

`RetrySubmitter.WithHaltChecker()` accepts a `GateChecker` that is evaluated between retry attempts. If halted, the loop aborts with `retry_halted: true`. Nil checker = fail-open (backward-compatible).

```go
rs := NewRetrySubmitter(adapter, policy).
    WithHaltChecker(controlStore)
```

### 2.3 Abort Metadata

| Abort Reason | Metadata Key | When |
|-------------|-------------|------|
| Max attempts exhausted | `retry_exhausted: true` | All attempts consumed |
| Kill switch halted | `retry_halted: true` | GateChecker returns halted between attempts |
| Global deadline exceeded | `retry_deadline_exceeded: true` | Wall-clock budget exceeded |

All carry `retry_attempts: N`. The three keys are mutually exclusive.

## 3. Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/application/execution/retry_submitter.go` | Modified | Added `Deadline` to `RetryPolicy`; added `WithHaltChecker()`, `TestWithNowFn()`; deadline + halt checks in loop; `annotateHalted`, `annotateDeadline`, `isHalted` methods |
| `internal/application/execution/retry_submitter_test.go` | Modified | 8 new tests (S323); `dynamicGateChecker` test helper |
| `docs/architecture/retry-coordination-hardening.md` | New | Design and integration guide |
| `docs/architecture/retry-deadline-halt-check-and-abort-semantics.md` | New | Semantic contract, invariants, limitations |
| `docs/stages/stage-s323-retry-coordination-hardening-report.md` | New | This report |

## 4. Test Evidence

### 4.1 New Tests (S323)

| Test | Scenario | Outcome |
|------|----------|---------|
| `TestRetry_DeadlineExceeded_AbortsLoop` | Deadline budget exhausted mid-loop | PASS |
| `TestRetry_DeadlineZero_NoDeadlineEnforced` | Zero deadline = no budget limit | PASS |
| `TestRetry_HaltChecker_HaltsDuringRetry` | Kill switch halted → loop aborts after 1 attempt | PASS |
| `TestRetry_HaltChecker_NotHalted_RetriesNormally` | Kill switch active → retries proceed | PASS |
| `TestRetry_HaltChecker_Nil_FailOpen` | No halt checker → fail-open | PASS |
| `TestRetry_HaltChecker_BecomesHaltedMidLoop` | Halt triggered after 2nd attempt | PASS |
| `TestRetry_DeadlineAndHalt_DeadlineWinsWhenBothTrigger` | Both trigger → deadline checked first | PASS |
| `TestRetry_SuccessBeforeDeadline_NoDeadlineMetadata` | Success within budget → no abort metadata | PASS |

### 4.2 Existing Tests (Regression Check)

All 9 pre-existing retry tests pass without modification:

| Test | Outcome |
|------|---------|
| `TestRetry_SuccessOnFirstAttempt` | PASS |
| `TestRetry_SuccessOnSecondAttempt` | PASS |
| `TestRetry_ExhaustsMaxAttempts` | PASS |
| `TestRetry_NonRetryableError_NoRetry` | PASS |
| `TestRetry_ContextCancelled_AbortsLoop` | PASS |
| `TestRetry_PreservesDeterministicClientOrderID` | PASS |
| `TestRetry_PolicyMaxAttemptsZero_DefaultsToOne` | PASS |
| `TestRetry_BackoffIncreases` | PASS |
| `TestRetry_SuccessOnThirdAttempt_MatchesRealScenario` | PASS |

Full execution package test suite: **17 tests, all pass**, zero regressions.

## 5. Invariants Preserved

| Invariant | Source | Verification |
|-----------|--------|-------------|
| INV-RC-1: Deadline independence from context | S323 | Deadline + DeadlineZero tests |
| INV-RC-2: Deadline monotonicity | S323 | Computed once at loop entry |
| INV-RC-3: Halt fail-open | S323 | Nil + NotHalted tests |
| INV-RC-4: Halt check after attempt, before backoff | S323 | HaltsDuringRetry (1 call only) |
| INV-RC-5: Mutually exclusive abort metadata | S323 | DeadlineAndHalt test |
| INV-RC-6: Idempotency preservation | S313/S323 | PreservesDeterministicClientOrderID |
| INV-RC-7: No submission after abort | S323 | All halt/deadline tests verify call count |
| EC-1: Deterministic client order ID | S313 | Unchanged |
| EC-3: Per-request deadline | S308 | Unchanged |

## 6. R-S320 Gap Closure

| Gap | Before S323 | After S323 |
|-----|------------|-----------|
| R-S320-2: No global retry deadline | Unbounded wall-clock possible | `Deadline` field; default 10s |
| R-S320-3: Kill switch blind during retry | Checked only before first attempt | Checked between every attempt via `WithHaltChecker` |

**Verdict**: R-S320-2 and R-S320-3 are **closed**.

## 7. Residual Gaps

| ID | Gap | Risk Level | Note |
|----|-----|-----------|------|
| R-S323-1 | Deadline does not cancel in-flight submits | Low | Per-submit context deadline handles this |
| R-S323-2 | Halt check timeout is fixed at 2s | Low | Consistent with SafetyGate; configurable later |
| R-S323-3 | Production wiring of `WithHaltChecker` in actor pipeline | Medium | Actor-layer integration is next step |
| R-S323-4 | No circuit breaker for repeated venue failures | Low | Separate concern; not in scope for coordination hardening |

## 8. Preparation for S324

With R-S320-2 and R-S320-3 closed, the retry coordination layer is hardened. The venue execution path now has:
- Complete error classification (S314)
- Bounded retry with idempotency (S319)
- Global deadline + halt coordination (S323)
- Verified failure paths (S320)
- Post-200 reconciliation (S322)

Recommended next directions:
1. **Actor-layer wiring** — Wire `RetrySubmitter.WithHaltChecker(controlStore)` and `Post200Reconciler` into the production `VenueAdapterActor` pipeline (R-S323-3).
2. **Evidence gate closure** — Aggregate all venue-readiness evidence across S308–S323.
3. **Operational observability** — Structured metrics for halt-during-retry and deadline-exceeded events.
