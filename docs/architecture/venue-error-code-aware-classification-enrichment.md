# Venue Error Code Aware Classification Enrichment

> **Stage:** S325
> **Status:** Complete
> **Predecessor:** S314 (Error Classification), S320 (Failure Path Verification)
> **Scope:** Surgical enrichment of error classification using Binance venue error codes

## 1. Problem Statement

The error classification system (S314) maps HTTP status codes to canonical problem codes with retryability semantics. This works correctly for the majority of cases, but Binance sometimes returns HTTP status codes that do not reflect the actual failure class:

| Scenario | HTTP Status | Venue Code | Before S325 | Actual Failure |
|----------|------------|-----------|-------------|----------------|
| Venue internal error | 400 | -1001 | InvalidArgument, non-retryable | Venue-side transient, retryable |
| IP-level rate limit | 418 | -1003 | InvalidArgument, non-retryable | Rate limit ban, retryable |
| Order rate limit | 400 | -1015 | InvalidArgument, non-retryable | Order rate limit, retryable |

In all three cases, the HTTP status (4xx) suggests a client error, but the venue error code reveals the actual cause is transient and retryable.

## 2. Design Decision

### 2.1 Override Model

The enrichment uses a **code-override-before-HTTP-fallback** pattern:

```
handleErrorResponse(statusCode, body)
  ├── parse venue error code from body
  ├── classifyByVenueErrorCode(statusCode, venueCode, details)
  │     ├── skip if statusCode outside 4xx range
  │     ├── skip if statusCode is 401/403/429 (already correct)
  │     ├── match on venueCode: -1001, -1003, -1015
  │     └── return override OR (nil, false) to fall through
  └── [existing HTTP-based classification switch]
```

### 2.2 Safety Constraints

1. **Auth immunity**: HTTP 401/403 are never overridden, regardless of venue code. Authentication failures are definitionally non-retryable.
2. **429 immunity**: HTTP 429 is already correctly classified as rate-limited/retryable. No override needed.
3. **5xx passthrough**: Server errors (5xx) already classify as retryable. The override only applies to the 4xx range where misclassification occurs.
4. **Unmapped codes fall through**: Any venue code not in the explicit mapping falls through to the existing HTTP-based classification. No default override.

### 2.3 Observability Enhancement

Overridden classifications include a `venue_error_class` detail field for diagnostics:

| Venue Code | venue_error_class | Meaning |
|-----------|------------------|---------|
| -1001 | `venue_internal` | Venue-side transient failure |
| -1003 | `ip_rate_limit` | IP-level rate limit/ban |
| -1015 | `order_rate_limit` | Order submission rate limit |

This field is only present when the venue code triggered an override, making it easy to filter and alert on override events.

## 3. Why These Three Codes

### -1001 (DISCONNECTED / Internal Error)

Binance returns this when their internal systems fail to process the request. Despite the HTTP 400 status, the failure is server-side and transient. Retrying after backoff is the correct behavior.

**Evidence**: Binance API documentation describes -1001 as "Internal error; unable to process your request. Please try again."

### -1003 (TOO_MANY_REQUESTS — IP Ban)

Binance uses HTTP 418 (I'm a teapot) for IP-level rate limit bans, paired with code -1003. Without this override, HTTP 418 falls into the 4xx catch-all as a non-retryable client error, when it is actually a temporary rate limit.

**Evidence**: Binance Futures API documentation explicitly lists HTTP 418 as the IP ban response.

### -1015 (TOO_MANY_ORDERS)

Binance returns this with HTTP 400 when the order submission rate exceeds the venue limit. The HTTP 400 is misleading — the order is not malformed, the rate is too high. This is semantically identical to HTTP 429 rate limiting.

**Evidence**: Binance API error code table lists -1015 as "Too many new orders."

## 4. What Was NOT Mapped (and Why)

| Venue Code | Description | Decision | Rationale |
|-----------|------------|----------|-----------|
| -1021 | Timestamp outside recvWindow | Not mapped | Configuration/clock error, not transient |
| -2010 | NEW_ORDER_REJECTED | Not mapped | Insufficient margin — genuine rejection |
| -2015 | Invalid API-key/permissions | Not mapped | Auth failure — already handled by 401/403 |
| -1100 | Illegal characters | Not mapped | Genuine validation error |
| -1121 | Invalid symbol | Not mapped | Genuine validation error |
| -4000+ | Futures-specific errors | Not mapped | Mostly parameter validation, not transient |

The mapping is deliberately minimal. Only codes where the HTTP-to-problem mapping is provably wrong are overridden. All other codes fall through to the battle-tested HTTP-based classification.

## 5. Contract Stability

### 5.1 Problem Type

No changes to `problem.Problem` struct, `ProblemCode` values, or the `problem` package API. The enrichment only changes which `ProblemCode` and `Retryable` value are assigned for specific HTTP+venueCode combinations.

### 5.2 Details Schema

One new optional field added: `venue_error_class` (string). Only present when a venue code override fires. Existing fields (`venue_http_status`, `venue_error_code`) are unchanged.

### 5.3 Behavioral Change

| Combination | Before S325 | After S325 |
|-------------|-------------|------------|
| HTTP 400 + code -1001 | InvalidArgument, non-retryable | **Unavailable, retryable** |
| HTTP 418 + code -1003 | InvalidArgument, non-retryable | **Unavailable, retryable** |
| HTTP 400 + code -1015 | InvalidArgument, non-retryable | **Unavailable, retryable** |
| All other combinations | Unchanged | Unchanged |

The behavioral change is strictly an improvement: errors that were incorrectly classified as permanent client errors are now correctly classified as transient, enabling retry recovery.
