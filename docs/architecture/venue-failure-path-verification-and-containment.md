# Venue Failure Path Verification and Containment

> S320 — Verification of failure path classification, retry behavior, containment semantics, and observable effects across the venue execution path.

## 1. Purpose

This document records the verified failure path behaviors for the venue execution path after S317–S319 delivered the minimal retry infrastructure. The goal is proportional verification — proving that the most operationally valuable failure modes behave correctly without inflating into a full SRE program.

## 2. Failure Mode Taxonomy (Verified)

### 2.1 Non-Retryable Failure Paths (Abort Immediately)

| ID | Failure Mode | HTTP Code(s) | Problem Code | Retry Behavior | Containment |
|----|-------------|-------------|-------------|----------------|-------------|
| FP-02 | Auth failure | 401, 403 | InvalidArgument | No retry, 1 call only | Immediate abort, no retry metadata |
| FP-03 | Auth escalation | 503 then 401 | InvalidArgument | Aborts on escalation | Stops at non-retryable boundary |
| FP-08 | Client error | 400, 422 | InvalidArgument | No retry | No retry metadata attached |
| FP-09 | Parse failure | 200 + bad JSON | Internal | No retry | Non-retryable, adapter layer |
| FP-11 | Body read failure | 200 + stall | Internal | No retry | Non-retryable after acceptance |
| FP-18 | Unknown status | 200 + bad status | Internal | No retry | Non-retryable, domain mapping |

### 2.2 Retryable Failure Paths (Subject to Retry Loop)

| ID | Failure Mode | HTTP Code(s) | Problem Code | Retry Behavior | Recovery |
|----|-------------|-------------|-------------|----------------|----------|
| FP-04 | Rate limit | 429 | Unavailable | Retries with backoff | Recovers when limit lifts |
| FP-05 | Network failure | N/A (conn refused) | Unavailable | Retries with backoff | Recovers when network restores |
| FP-06 | Mixed retryable | 429, 502, 500 | Unavailable | Exhausts all attempts | Returns last error + metadata |
| FP-07 | Gateway timeout | 504 | Unavailable | Retries with backoff | Standard retry path |
| FP-16 | Adapter timeout | N/A (deadline) | Unavailable | Retries with backoff | Recovers when venue responds |

### 2.3 Containment Boundaries

| Boundary | Verified By | Behavior |
|----------|-----------|----------|
| Non-retryable → no retry metadata | FP-08 | retry_attempts/retry_exhausted never attached |
| Non-retryable → single call | FP-02, FP-09, FP-15, FP-18 | Exactly 1 HTTP call made |
| Retryable → exhaustion metadata | FP-06, FP-14 | retry_attempts and retry_exhausted present |
| Context deadline → early abort | FP-01 | Retry loop aborts before MaxAttempts |
| Error escalation → abort | FP-03 | Retryable→non-retryable transition stops loop |
| Intent immutability | FP-12 | No mutation across retry attempts |
| Client order ID stability | FP-13 | Same ID in every HTTP request across retries |
| Credential redaction | FP-19 | API key/secret never in error messages |
| No-action bypass | FP-17 | SideNone skips venue entirely |

## 3. Key Findings

### 3.1 Body Read Failure After HTTP 200 (FP-11)

**Finding**: When the venue returns HTTP 200 headers but the body read fails (e.g., due to timeout during body streaming), the error is classified as `Internal` and `non-retryable`.

**Why this is correct**: The venue has already accepted the order. The 200 status means the order was processed. Retrying would send the same client order ID, but we cannot guarantee the venue handles this idempotently in all cases. The safer behavior is to surface the error without retrying.

**Residual risk**: No reconciliation mechanism exists to determine the actual order outcome when body read fails. This is acceptable for the current scope (testnet, single-order) but would need reconciliation via order status polling for production.

### 3.2 Error Escalation Containment (FP-03)

**Finding**: When a transient error (503) is followed by a non-retryable error (401) on retry, the retry loop correctly aborts. The final error is the auth error (InvalidArgument, non-retryable), and no retry metadata is attached.

**Why this matters**: Venue credentials could rotate or be revoked between retries. The retry loop must not mask a permanent error by continuing to retry.

### 3.3 Context Deadline vs Retry Budget (FP-01)

**Finding**: When the caller's context deadline is shorter than the total retry budget (MaxAttempts * (request time + backoff)), the context deadline wins. The retry loop checks `ctx.Err()` before each attempt and during backoff sleep.

**Operational implication**: The caller controls the total wall-clock budget. The retry policy controls per-attempt behavior within that budget.

### 3.4 Observable Metadata Propagation (FP-14)

**Finding**: When retries are exhausted, the final Problem carries both venue-specific details (`venue_http_status`, `venue_error_code`) from the last attempt AND retry metadata (`retry_attempts`, `retry_exhausted`). No details are lost during the retry→annotate chain.

## 4. Containment Model

```
Intent arrives
    │
    ├── Side == None? ──→ synthetic receipt (no venue call)
    │
    ├── Safety Gate ──→ kill switch / staleness check (actor layer)
    │
    └── VenuePort.SubmitOrder
         │
         ├── HTTP error?
         │    ├── 401/403 ──→ InvalidArgument, non-retryable → ABORT
         │    ├── 400/422 ──→ InvalidArgument, non-retryable → ABORT
         │    ├── 429     ──→ Unavailable, retryable → RETRY
         │    ├── 5xx     ──→ Unavailable, retryable → RETRY
         │    └── network ──→ Unavailable, retryable → RETRY
         │
         ├── Parse error? ──→ Internal, non-retryable → ABORT
         │
         ├── Unknown status? ──→ Internal, non-retryable → ABORT
         │
         └── Success ──→ receipt with fills
                │
                └── Body read failure after 200? ──→ Internal, non-retryable → ABORT
```

## 5. Invariants Verified

| Invariant | Source | Status |
|-----------|--------|--------|
| EC-1: Deterministic client order ID | S313 | Verified (FP-13) |
| EC-3: Per-request deadline enforcement | S308 | Verified (FP-10) |
| PGR-08: Intent immutability | S310 | Verified (FP-12) |
| F-1: No bare errors | S308 | Verified (all tests use Problem) |
| RF-1: Retryable flag accuracy | S314 | Verified (FP-08, all classification tests) |
| VA-1.13: No credential leakage | S314 | Verified (FP-19) |

## 6. Non-Goals (Explicitly Out of Scope)

- Circuit breaker pattern
- Per-error-class differentiated retry policies
- Structured retry metrics/counters
- Async/queue-based retry
- Reconciliation for accepted-but-response-lost orders
- Kill switch liveness during retry backoff
- Multi-venue error normalization
