# Failure Path Classification, Retry, Containment, and Observability Findings

> S320 — Detailed findings from exercising failure modes across the venue execution path.

## 1. Classification Correctness

### 1.1 Verified Classification Table

| HTTP Status | Venue Error Code | Problem Code | Retryable | Test ID |
|-------------|-----------------|-------------|-----------|---------|
| 401 | -2015 | InvalidArgument | false | FP-02, FP-08 |
| 403 | -2015 | InvalidArgument | false | FP-15, FP-08 |
| 400 | -1121 | InvalidArgument | false | FP-08 |
| 422 | -1100 | InvalidArgument | false | FP-08 |
| 429 | -1015 | Unavailable | true | FP-04, FP-14 |
| 500 | -1001 | Unavailable | true | FP-06 |
| 502 | -1001 | Unavailable | true | FP-06 |
| 503 | -1001 | Unavailable | true | FP-03, FP-04 |
| 504 | -1001 | Unavailable | true | FP-07 |
| N/A (network) | N/A | Unavailable | true | FP-05 |
| N/A (timeout) | N/A | Unavailable | true | FP-10, FP-16 |
| 200 + bad JSON | N/A | Internal | false | FP-09 |
| 200 + unknown status | N/A | Internal | false | FP-18 |
| 200 + body stall | N/A | Internal | false | FP-11 |

### 1.2 Classification Decision Points

The adapter classifies errors at three distinct points:

1. **HTTP transport layer** (line 111-113 in adapter): Network errors, DNS failures, connection refused, timeouts. All classified as `Unavailable, retryable`.

2. **HTTP status layer** (handleErrorResponse): Status-code-driven switch. 401/403 → auth, 429 → rate limit, 4xx → client error, 5xx → server error.

3. **Response parse layer** (parseOrderResponse): JSON unmarshal failures, unknown status values, body read errors. All classified as `Internal, non-retryable`.

**Finding**: Classification never uses venue-specific error codes (e.g., Binance code -1015) for routing decisions — only HTTP status. Venue codes are captured in Problem.Details for observability but don't influence retryability. This is a deliberate simplification; venue-code-aware classification would require per-venue knowledge.

## 2. Retry Behavior Findings

### 2.1 Retry Recovery Scenarios Verified

| Scenario | Attempts | Outcome | Test ID |
|----------|----------|---------|---------|
| 429 → 429 → success | 3 | Recovery | FP-04 |
| Network failure → success | 2 | Recovery | FP-05 |
| Adapter timeout → success | 2 | Recovery | FP-16 |
| 503 → 401 (escalation) | 2 | Abort on 401 | FP-03 |
| 429 → 502 → 500 (exhaustion) | 3 | Exhausted | FP-06 |
| Context expires mid-loop | <5 | Context abort | FP-01 |

### 2.2 Retry Budget Analysis

With the default policy (3 attempts, 100ms base, 2x factor, ±25% jitter):

| Attempt | Nominal Delay | Jitter Range | Cumulative |
|---------|-------------|-------------|------------|
| 1 | — | — | 0ms |
| 2 | 100ms | 75–125ms | ~100ms |
| 3 | 200ms | 150–250ms | ~300ms |

**Total backoff budget**: ~300ms (excluding per-request time).
**Per-request deadline**: 10s default (EC-3).
**Worst-case wall clock**: 3 × 10s + 300ms = ~30.3s.

**Finding**: The retry loop has no global deadline. If the caller's context allows 30+ seconds, all 3 attempts will execute even if each times out at 10s. Callers must provide their own context deadline to bound total wall-clock time.

### 2.3 Backoff Jitter Verification

FP-10 (from S319's TestRetry_BackoffIncreases) verified that:
- Delays increase exponentially: ~100ms → ~200ms → ~400ms
- Jitter range is ±25% (0.75x to 1.25x)
- The MaxDelay cap (2s) is respected

### 2.4 Idempotency Through Retry

FP-13 verified that the same `newClientOrderId` is sent in every HTTP request across retry attempts. The client order ID is derived from `SHA-256(intent.DeduplicationKey())`, which is deterministic. No time-varying or random components are introduced by the retry loop.

**Finding**: Venue-side deduplication depends on the venue honoring `newClientOrderId`. For Binance, duplicate client order IDs on the same symbol are rejected (HTTP 400, code -2022). This means a retry after a lost response will get a 400, which is classified as non-retryable — effectively preventing double execution but also preventing fill recovery. This is the correct conservative behavior.

## 3. Containment Findings

### 3.1 Non-Retryable Error Containment

FP-08 verified that for ALL non-retryable HTTP status codes (400, 401, 403, 422):
- Exactly 1 HTTP call is made
- `retry_attempts` is NOT attached to Problem.Details
- `retry_exhausted` is NOT attached to Problem.Details
- `Retryable` flag is false

**Finding**: The retry submitter's containment boundary is clean. Non-retryable errors pass through without any retry metadata contamination.

### 3.2 Parse Failure Containment

FP-09 verified that malformed JSON from a 200 response:
- Is classified as `Internal`
- Does NOT trigger retry
- Produces exactly 1 HTTP call

**Finding**: Parse failures are contained at the adapter layer. The retry submitter never sees a retryable flag on parse errors, so the containment is structural (not just by convention).

### 3.3 Body Read Failure Containment (Key Finding)

FP-11 verified that when HTTP 200 headers are received but the body read fails:
- Error is classified as `Internal, non-retryable`
- This is the correct behavior: the venue has already accepted the order

**Residual gap**: No reconciliation mechanism exists. If the body read fails after 200, the system knows the order was accepted but does not know the fill details (price, quantity, fee). Recovery would require:
1. Query venue order status by client order ID
2. Populate fill details from the status response
3. Resume the persistence pipeline with the recovered fill

This is out of scope for S320 but identified as a production readiness item.

### 3.4 Error Escalation Containment

FP-03 verified that a transition from retryable (503) to non-retryable (401) during retries:
- Stops the retry loop immediately
- Returns the non-retryable error (not the original retryable one)
- Does NOT attach retry metadata (because the final error is non-retryable)

**Finding**: The retry submitter correctly respects error escalation. It does not treat the retry loop as all-or-nothing — each attempt's error is independently classified.

## 4. Observability Findings

### 4.1 Error Detail Propagation

FP-14 verified that after retry exhaustion, the final Problem carries:
- `venue_http_status`: from the last attempt's response
- `venue_error_code`: from the last attempt's Binance error body
- `retry_attempts`: total number of attempts made
- `retry_exhausted`: true

**Finding**: Details from the adapter layer survive through the retry submitter's annotation. The `annotate()` method uses `WithDetail()` which copies (not replaces) existing details.

### 4.2 Credential Redaction

FP-19 verified that error messages from the full adapter→retry path never contain:
- API key values
- API secret values

This holds for auth failures (401), where the error message uses the venue's error message, not the credentials themselves.

### 4.3 Observable Signals by Failure Class

| Failure Class | Problem.Code | Problem.Retryable | Details Keys |
|--------------|-------------|-------------------|-------------|
| Auth | InvalidArgument | false | venue_http_status, venue_error_code |
| Client error | InvalidArgument | false | venue_http_status, venue_error_code |
| Rate limit | Unavailable | true | venue_http_status, venue_error_code, retry_* |
| Server error | Unavailable | true | venue_http_status, venue_error_code, retry_* |
| Network | Unavailable | true | retry_* |
| Timeout | Unavailable | true | retry_* |
| Parse failure | Internal | false | (none — stdlib error wrapped) |
| Unknown status | Internal | false | (none — format string only) |

## 5. Surprises and Corrections

### 5.1 FP-11 Expectation Correction

Initial hypothesis: body read failure after 200 should be retryable (like a network error).
Actual behavior: non-retryable.
Corrected understanding: once 200 is received, the venue accepted the order. Retrying risks double execution. The non-retryable classification is the safer, correct behavior.

### 5.2 No Surprises in Classification

All 8 failure classes from S314 behaved exactly as documented. The classification switch in `handleErrorResponse` correctly routes every HTTP status code. No misclassification was found.

### 5.3 Retry Loop Termination

The retry loop has three clean termination paths:
1. Success → return receipt
2. Non-retryable error → immediate return (no metadata)
3. MaxAttempts exhausted or context expired → return with retry metadata

No deadlock, infinite loop, or ambiguous termination was observed in any test.

## 6. Residual Gaps (Identified, Not Addressed)

| Gap | Risk | Mitigation Path |
|-----|------|-----------------|
| No body-read-failure reconciliation | Order accepted but fill unknown | Add order status polling by client order ID |
| No global retry deadline | Caller must provide context deadline | Add optional global timeout to RetryPolicy |
| Kill switch not checked during retry backoff | Kill switch change ignored mid-retry | Check IsHalted before each retry attempt |
| No per-error-class retry policies | Same backoff for rate limit and server error | Differentiate: rate limit could use Retry-After header |
| Venue error codes not used for classification | Potential misclassification edge cases | Map Binance codes to classes for higher fidelity |
| No structured retry metrics | Retry behavior not visible in dashboards | Emit retry_attempt/retry_exhausted as metrics |
