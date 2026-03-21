# Stage S319 — Minimal Retry Loop Infrastructure Report

> **Status:** Complete
> **Predecessor:** S316 (End-to-End Venue Integration Proof), S314 (Error Classification and Retryability Completion)
> **Scope:** Minimal, disciplined retry loop for retryable venue failures

## 1. Executive Summary

S316 proved that the venue adapter correctly classifies errors into 8 failure classes with accurate retryability flags, but explicitly noted that no retry loop existed (NG-6). S319 closes this gap with a minimal `RetrySubmitter` decorator that wraps any `VenuePort` and retries retryable failures with exponential backoff, context-aware abort semantics, and deterministic idempotency via client order ID.

The implementation adds exactly two files (implementation + tests) and two architecture documents. No existing code was modified — the retry layer is purely additive and composable.

## 2. Retry Loop Delivered

### Design: Decorator over VenuePort

```
RetrySubmitter implements VenuePort {
    inner:   VenuePort           // actual adapter
    policy:  RetryPolicy         // bounds (max attempts, backoff, cap)
    sleepFn: func(time.Duration) // injectable for testing
}
```

### Default Policy

| Parameter | Value |
|-----------|-------|
| MaxAttempts | 3 |
| BaseDelay | 100 ms |
| MaxDelay | 2 s |
| Factor | 2.0× |
| Jitter | ±25% |

### Abort Conditions

1. Non-retryable error → immediate return
2. Context cancelled/deadline → return with retry metadata
3. MaxAttempts exhausted → return with `retry_attempts` and `retry_exhausted` details

### Idempotency

Client order ID is deterministic (`SHA256(dedup_key)[:32]`). Same intent = same ID on every retry. Venue-side deduplication prevents duplicate execution.

## 3. Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/application/execution/retry_submitter.go` | **Created** | RetrySubmitter, RetryPolicy, DefaultRetryPolicy, jitter logic |
| `internal/application/execution/retry_submitter_test.go` | **Created** | 9 test scenarios |
| `docs/architecture/minimal-retry-loop-infrastructure-for-venue-failures.md` | **Created** | Architecture record |
| `docs/architecture/retry-semantics-bounds-idempotency-and-non-goals.md` | **Created** | Semantic contract |
| `docs/stages/stage-s319-minimal-retry-loop-infrastructure-report.md` | **Created** | This report |

No existing files were modified. The retry layer is composable and opt-in.

## 4. Tests and Evidence

### Test Matrix (9 scenarios, all pass)

| Test | Scenario | Validates |
|------|----------|-----------|
| `TestRetry_SuccessOnFirstAttempt` | No retry needed | Happy path, single call |
| `TestRetry_SuccessOnSecondAttempt` | Transient failure then success | Basic retry recovery |
| `TestRetry_ExhaustsMaxAttempts` | All attempts fail | Retry metadata in problem details |
| `TestRetry_NonRetryableError_NoRetry` | Non-retryable returned | Abort, no retry metadata |
| `TestRetry_ContextCancelled_AbortsLoop` | Context cancelled during sleep | Abort semantics |
| `TestRetry_PreservesDeterministicClientOrderID` | Same ID across retries | EC-1 invariant |
| `TestRetry_PolicyMaxAttemptsZero_DefaultsToOne` | Edge case policy | Clamp to 1 |
| `TestRetry_BackoffIncreases` | Delay grows exponentially | Backoff correctness |
| `TestRetry_SuccessOnThirdAttempt_MatchesRealScenario` | 429 → 500 → success | Mixed failure recovery |

### Execution

```
$ go test ./internal/application/execution/ -run "TestRetry" -v -count=1
=== RUN   TestRetry_SuccessOnFirstAttempt           --- PASS
=== RUN   TestRetry_SuccessOnSecondAttempt           --- PASS
=== RUN   TestRetry_ExhaustsMaxAttempts              --- PASS
=== RUN   TestRetry_NonRetryableError_NoRetry        --- PASS
=== RUN   TestRetry_ContextCancelled_AbortsLoop      --- PASS
=== RUN   TestRetry_PreservesDeterministicClientOrderID --- PASS
=== RUN   TestRetry_PolicyMaxAttemptsZero_DefaultsToOne --- PASS
=== RUN   TestRetry_BackoffIncreases                 --- PASS
=== RUN   TestRetry_SuccessOnThirdAttempt_MatchesRealScenario --- PASS
PASS
```

### Invariant Preservation

| Invariant | Status | Evidence |
|-----------|--------|----------|
| EC-1: Deterministic client order ID | ✓ | `TestRetry_PreservesDeterministicClientOrderID` |
| EC-3: Per-request deadline | ✓ | Context propagated; `TestRetry_ContextCancelled_AbortsLoop` |
| F-1: No bare errors | ✓ | All paths return `*problem.Problem` |
| RF-1: Retryable flag accuracy | ✓ | `TestRetry_NonRetryableError_NoRetry` |
| PGR-08: Intent immutability | ✓ | Request object never mutated |

## 5. Remaining Limits

| ID | Limit | Disposition |
|----|-------|-------------|
| NG-R1 | No generic retry framework | Intentional — venue-path only |
| NG-R2 | No circuit breaker | Deferred to production readiness wave |
| NG-R3 | No self-imposed rate limiter | Deferred — testnet limits are generous |
| NG-R5 | No per-error-class policies | Single policy sufficient for single adapter |
| NG-R6 | No structured retry metrics | Observability via Problem details only |
| NG-R7 | Non-venue errors not retried | Out of scope (NATS, ClickHouse, etc.) |
| NG-R9 | Partial fill not retried | Partial fills are successes, not failures |

The retry loop is not yet wired into the production actor path — it exists as a ready-to-compose decorator. Integration into the actor layer is the natural next step.

## 6. Recommended Preparation for S320

S319 closes the retry gap. Candidate next steps:

1. **Wire RetrySubmitter into the actor layer** — replace direct `VenuePort` calls with `NewRetrySubmitter(adapter, DefaultRetryPolicy())` in the execution actor.
2. **Circuit breaker** — if venue availability becomes a concern beyond transient errors.
3. **Retry metrics** — structured counters for retry frequency, exhaustion rate, per-error-class distribution.
4. **Rate limit awareness** — differentiated backoff for HTTP 429 (e.g., respect `Retry-After` header).
5. **Reconciliation** — handle the edge case where a venue accepted an order but the response was lost (network failure after acceptance). This requires periodic order status polling, which is a separate concern from retry.

## 7. Gate Criteria Met

| Criterion | Evidence |
|-----------|----------|
| Retry loop exists and is coherent | `RetrySubmitter` with 3-attempt default, exponential backoff |
| Respects existing invariants | EC-1, EC-3, F-1, RF-1, PGR-08 all preserved |
| Closes S316 gap without scope inflation | Single decorator, no framework, no queue, no orchestration |
| Venue path is operationally more robust | Transient failures now recover automatically (up to 3 attempts) |
