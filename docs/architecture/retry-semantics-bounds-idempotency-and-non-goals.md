# Retry Semantics, Bounds, Idempotency, and Non-Goals

> Stage S319 — Semantic Contract

## 1. Retry Semantics

### 1.1 Retryability Is Decided by the Adapter

The `RetrySubmitter` does not classify errors — it trusts the `Problem.Retryable` flag set by the venue adapter. The S314 error classification is the single source of truth for retryability. This separation ensures that retry policy changes never affect error classification, and vice versa.

### 1.2 Retry Scope

The retry loop covers exactly one `SubmitOrder` call. It does not:
- Retry across different intents
- Retry after safety gate rejection
- Retry non-venue failures (e.g., NATS publish, ClickHouse write)

### 1.3 Attempt Counting

- Attempt 1 is the initial call (not a "retry")
- `MaxAttempts=3` means: 1 initial + 2 retries
- The `retry_attempts` detail in exhaustion reflects total attempts, not retries

## 2. Bounds

### 2.1 Time Bounds

| Bound | Value | Source |
|-------|-------|--------|
| Per-request deadline | 10 s | EC-3 invariant (adapter-enforced) |
| Max total retry time | ~7.5 s worst case | 3 × 10s deadline, but backoff is only ~0.7s total |
| Backoff budget | 100ms → 200ms → 400ms (nominal) | Exponential with ±25% jitter |

The context deadline is the hard upper bound. If the caller provides a 10s context, the retry loop cannot exceed 10s total regardless of policy settings.

### 2.2 Attempt Bounds

| Bound | Value | Rationale |
|-------|-------|-----------|
| MaxAttempts | 3 | Sufficient for transient errors; prevents runaway |
| Minimum | 1 | Policy with MaxAttempts < 1 is clamped to 1 |

### 2.3 Backoff Bounds

| Bound | Value |
|-------|-------|
| BaseDelay | 100 ms |
| MaxDelay | 2 s (cap) |
| Factor | 2.0× |
| Jitter | ±25% uniform |

The delay sequence (nominal): 100ms, 200ms, 400ms, 800ms, 1600ms, 2000ms (capped).
With 3 attempts, only 2 delays are used: ~100ms and ~200ms.

## 3. Idempotency

### 3.1 Client Order ID Invariant

`ClientOrderID(intent)` produces a deterministic 32-hex-character string from the intent's deduplication key. The retry loop passes the same `VenueOrderRequest` object on every attempt — the intent is never mutated. Therefore:

**Invariant: all retry attempts for the same intent carry the same `newClientOrderId`.**

### 3.2 Venue-Side Deduplication

Binance uses `newClientOrderId` for deduplication:
- If the first attempt was accepted, a retry with the same ID receives a duplicate order rejection (HTTP 400) — which the adapter classifies as non-retryable, aborting the loop.
- If the first attempt did not reach the venue (network error), the retry is the first the venue sees.

This means retries cannot cause duplicate execution.

### 3.3 Intent Immutability

The `RetrySubmitter` never modifies the `VenueOrderRequest` or the `ExecutionIntent` within it. Timestamp, quantity, side, symbol — all remain constant across retries. This preserves the deduplication key and partition key invariants.

## 4. Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-R1 | Generic retry framework | This is venue-path-only infrastructure, not a reusable library |
| NG-R2 | Circuit breaker | Testnet has generous limits; production circuit breaker is a future concern |
| NG-R3 | Self-imposed rate limiter | Binance testnet rate limits are high; pre-emptive limiting adds complexity without value now |
| NG-R4 | Retry queue / async retry | Retries are synchronous within the submission context; no background retry workers |
| NG-R5 | Per-error-class retry policies | All retryable errors use the same policy; differentiated backoff (e.g., longer for 429) is future work |
| NG-R6 | Retry metrics / counters | Observability is via Problem details; structured metrics (Prometheus, etc.) are deferred |
| NG-R7 | Retry for non-venue errors | NATS publish, ClickHouse write, and other infrastructure failures are not covered |
| NG-R8 | Configurable policy per adapter | Single policy is sufficient for the current single-adapter architecture |
| NG-R9 | Partial fill retry | If the venue returns a partial fill, that is a success — the retry loop does not attempt to fill the remainder |

## 5. Invariants Preserved

| Invariant | Status | Evidence |
|-----------|--------|----------|
| EC-1: Deterministic client order ID | ✓ | Same intent → same ID; test `TestRetry_PreservesDeterministicClientOrderID` |
| EC-3: Per-request context deadline | ✓ | Context checked before each attempt; deadline propagated to inner adapter |
| F-1: No bare errors escape | ✓ | RetrySubmitter returns `*problem.Problem` only |
| RF-1: Retryable flag accuracy | ✓ | RetrySubmitter trusts but does not modify the flag |
| PGR-08: Intent state preserved | ✓ | Request object is never mutated |

## 6. Failure Modes

| Scenario | Outcome |
|----------|---------|
| All 3 attempts fail with retryable error | Returns last error with `retry_attempts=3, retry_exhausted=true` |
| First attempt returns non-retryable error | Returns immediately, no retry metadata |
| Context cancelled during backoff sleep | Returns last retryable error with `retry_attempts=N` |
| Context deadline before first attempt | Returns context error wrapped as `SYS_UNAVAILABLE` |
| Inner adapter panics | Not caught — panics propagate (consistent with rest of codebase) |
