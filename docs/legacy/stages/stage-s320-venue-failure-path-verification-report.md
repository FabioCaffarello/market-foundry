# Stage S320 — Venue Failure Path Verification Report

> **Status:** Complete
> **Predecessor:** S319 (Minimal Retry Loop Infrastructure), S314 (Error Classification)
> **Scope:** Proportional verification of failure paths, classification, retry containment, and observability

## 1. Executive Summary

After S317–S319 delivered the venue adapter, persistence round-trip, and retry infrastructure, S320 verifies that the failure paths behave correctly under realistic conditions. This stage exercises 19 distinct failure scenarios (FP-01 through FP-19) covering timeout, auth failure, rate limiting, network failure, parse errors, error escalation, and containment boundaries.

All 19 tests pass. One expectation was corrected during development (FP-11: body read failure after HTTP 200 is correctly non-retryable). Zero regressions in the existing 80+ execution test suite.

**Key result**: The venue failure path is classified correctly, retry behavior is bounded, containment is clean, and observable metadata propagates through the full chain.

## 2. Failure Modes Verified

### 2.1 Summary Matrix

| Test ID | Failure Mode | Classification | Retry | Calls | Outcome |
|---------|-------------|---------------|-------|-------|---------|
| FP-01 | Context deadline mid-retry | Unavailable | Aborted | <5 | Context wins |
| FP-02 | Auth failure (401) | InvalidArgument | None | 1 | Immediate abort |
| FP-03 | 503 then 401 escalation | InvalidArgument | Abort on escalation | 2 | Non-retryable final |
| FP-04 | Rate limit recovery | Unavailable | 2 retries | 3 | Success on 3rd |
| FP-05 | Network failure recovery | Unavailable | 1 retry | 2 | Success on 2nd |
| FP-06 | Mixed retryable exhaustion | Unavailable | 3 attempts | 3 | Exhausted + metadata |
| FP-07 | HTTP 504 classification | Unavailable | Retryable | 1 | Correct classification |
| FP-08 | Non-retryable containment | InvalidArgument | None (4 codes) | 1 each | No retry metadata |
| FP-09 | Parse failure containment | Internal | None | 1 | Non-retryable |
| FP-10 | Adapter timeout vs context | Unavailable | Retryable | 1 | Adapter fires first |
| FP-11 | Body read after 200 | Internal | None | 1 | Non-retryable (finding) |
| FP-12 | Intent immutability | — | 3 attempts | 3 | No mutation |
| FP-13 | Client order ID stability | — | 3 attempts | 3 | Same ID all calls |
| FP-14 | Error details propagation | Unavailable | 2 attempts | 2 | All details preserved |
| FP-15 | HTTP 403 containment | InvalidArgument | None | 1 | Same as 401 |
| FP-16 | Timeout recovery on retry | Unavailable | 1 retry | 2 | Success on 2nd |
| FP-17 | No-action bypass | — | N/A | 0 | No venue call |
| FP-18 | Unknown status containment | Internal | None | 1 | Non-retryable |
| FP-19 | Credential redaction | InvalidArgument | None | 1 | No secrets leaked |

### 2.2 Coverage by Failure Class

| Failure Class (S314) | Tests | Integrated with Retry | Result |
|---------------------|-------|-----------------------|--------|
| Authentication (401/403) | FP-02, FP-03, FP-15, FP-19 | Yes | Correct |
| Client error (400/422) | FP-08 | Yes | Correct |
| Rate limit (429) | FP-04, FP-14 | Yes | Correct |
| Server error (5xx) | FP-06, FP-07 | Yes | Correct |
| Network failure | FP-05 | Yes | Correct |
| Timeout/deadline | FP-01, FP-10, FP-16 | Yes | Correct |
| Parse failure | FP-09, FP-11 | Yes | Correct |
| Unknown status | FP-18 | Yes | Correct |

## 3. Key Findings

### 3.1 Body Read Failure After HTTP 200 (FP-11)

**Surprise**: Initial expectation was that body read timeouts should be retryable. Actual behavior: `Internal, non-retryable`.

**Correct reasoning**: Once 200 headers are received, the venue has accepted the order. Retrying risks double execution. The non-retryable classification is the conservative, safe behavior.

**Residual**: No reconciliation mechanism for this case. Accepted as out-of-scope for testnet proof.

### 3.2 Error Escalation (FP-03)

The retry loop correctly handles error class transitions. A 503 (retryable) followed by a 401 (non-retryable) stops immediately. The final error reflects the actual failure, not the original transient error.

### 3.3 Observable Metadata Chain

Error details flow cleanly through adapter → retry submitter → annotate:
- Venue details (`venue_http_status`, `venue_error_code`) survive from adapter
- Retry metadata (`retry_attempts`, `retry_exhausted`) added by retry submitter
- Both coexist in the final Problem.Details without collision

### 3.4 Adapter Timeout vs Context Deadline (FP-10)

The per-request HTTP client timeout and the caller's context deadline operate independently. The shorter one wins. This was verified with a 200ms adapter timeout and 5s context — the adapter timeout fired in ~200ms.

## 4. Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/application/execution/failure_path_verification_test.go` | New | 19 integrated failure path tests |
| `docs/architecture/venue-failure-path-verification-and-containment.md` | New | Failure path verification model |
| `docs/architecture/failure-path-classification-retry-containment-and-observability-findings.md` | New | Detailed findings and gaps |
| `docs/stages/stage-s320-venue-failure-path-verification-report.md` | New | This report |

## 5. Test Evidence

```
=== RUN   TestFP01 ... PASS (0.57s)
=== RUN   TestFP02 ... PASS (0.00s)
=== RUN   TestFP03 ... PASS (0.00s)
=== RUN   TestFP04 ... PASS (0.00s)
=== RUN   TestFP05 ... PASS (0.00s)
=== RUN   TestFP06 ... PASS (0.00s)
=== RUN   TestFP07 ... PASS (0.00s)
=== RUN   TestFP08 ... PASS (0.00s)  [4 sub-tests: 400, 401, 403, 422]
=== RUN   TestFP09 ... PASS (0.00s)
=== RUN   TestFP10 ... PASS (2.00s)
=== RUN   TestFP11 ... PASS (3.00s)
=== RUN   TestFP12 ... PASS (0.00s)
=== RUN   TestFP13 ... PASS (0.01s)
=== RUN   TestFP14 ... PASS (0.00s)
=== RUN   TestFP15 ... PASS (0.00s)
=== RUN   TestFP16 ... PASS (2.00s)
=== RUN   TestFP17 ... PASS (0.00s)
=== RUN   TestFP18 ... PASS (0.00s)
=== RUN   TestFP19 ... PASS (0.00s)
PASS — 19/19 tests, 0 regressions in existing suite
```

## 6. Invariants Preserved

| Invariant | Source | Verification |
|-----------|--------|-------------|
| EC-1: Deterministic client order ID | S313 | FP-13: same ID across 3 retry attempts |
| EC-3: Per-request deadline | S308 | FP-10: adapter timeout fires correctly |
| PGR-08: Intent immutability | S310 | FP-12: no mutation across retries |
| F-1: No bare errors | S308 | All 19 tests return Problem |
| RF-1: Retryable flag accuracy | S314 | All 8 classes verified |
| VA-1.13: Credential redaction | S314 | FP-19: no secrets in errors |

## 7. Residual Gaps

| ID | Gap | Risk Level | Mitigation Path |
|----|-----|-----------|-----------------|
| R-S320-1 | No reconciliation for body-read-failure-after-200 | Medium | Order status polling by client order ID |
| R-S320-2 | No global retry deadline in RetryPolicy | Low | Callers provide context deadline |
| R-S320-3 | Kill switch not checked during retry backoff | Low | Add IsHalted check before each retry |
| R-S320-4 | Venue error codes unused for classification | Low | Map codes for higher-fidelity routing |
| R-S320-5 | No structured retry metrics | Low | Emit metrics from retry submitter |
| R-S320-6 | No per-error-class differentiated retry policies | Low | Rate limit could use Retry-After |

## 8. Preparation for S321

The failure path is now verified and documented. Recommended next directions:

1. **Retry integration into actor layer** — Wire `RetrySubmitter` into the venue adapter actor so retry happens transparently in the execution pipeline.
2. **Reconciliation mechanism** — Add order status polling for the R-S320-1 gap (body-read-failure-after-200).
3. **Operational observability** — Add structured retry metrics/logging for production visibility.
4. **Kill switch coordination** — Check kill switch status between retry attempts.

Each option is self-contained and does not require the others. Priority depends on whether the next focus is operational confidence (options 1, 3) or correctness completeness (options 2, 4).
