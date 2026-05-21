# Minimal Retry Loop Infrastructure for Venue Failures

> Stage S319 — Architectural Record

## 1. Purpose

S316 proved that the Binance Futures testnet adapter correctly classifies errors into 8 failure classes with accurate `Retryable` flags. However, the system had no retry loop — retryable errors surfaced immediately to the caller with no recovery attempt. This document records the minimal retry infrastructure introduced to close that gap.

## 2. Design Decisions

### 2.1 Decorator Pattern over Embedded Logic

The retry loop is implemented as `RetrySubmitter`, a decorator that wraps any `VenuePort`. This keeps the adapter focused on venue-specific concerns (signing, parsing, classification) while the retry policy is orthogonal and testable in isolation.

```
caller → RetrySubmitter (retry loop) → BinanceFuturesTestnetAdapter (venue calls)
```

Both implement `ports.VenuePort`, so they are composable and the caller is unaware of the retry layer.

### 2.2 Conservative Defaults

| Parameter    | Value   | Rationale |
|-------------|---------|-----------|
| MaxAttempts | 3       | Sufficient for transient blips; prevents runaway loops |
| BaseDelay   | 100 ms  | Fast first retry for network glitches |
| MaxDelay    | 2 s     | Cap prevents long waits on testnet |
| Factor      | 2.0×    | Standard exponential backoff |
| Jitter      | ±25%    | Prevents thundering herd on concurrent retries |

### 2.3 Abort Semantics

The retry loop aborts under three conditions:

1. **Non-retryable error** — returned immediately with no retry metadata. The 8-class error taxonomy already marks authentication, client errors, parse failures, and unknown status as non-retryable.
2. **Context cancellation/deadline** — the loop checks `ctx.Err()` before each attempt and on the sleep path. This preserves the EC-3 per-request deadline invariant.
3. **MaxAttempts exhausted** — the last retryable error is returned, enriched with `retry_attempts` and `retry_exhausted` details for observability.

### 2.4 Idempotency via Deterministic Client Order ID

Retries are safe because `ClientOrderID(intent)` is deterministic — it derives a SHA-256 hash from the intent's deduplication key. Every retry sends the same `newClientOrderId` to the venue. If the first attempt actually reached the venue and was accepted, the retry will either:
- Receive a duplicate rejection (which the adapter classifies as a client error, non-retryable — loop aborts), or
- Receive the same fill response (idempotent).

No additional deduplication logic is needed at this layer.

## 3. What Is Retried

Only errors where `Problem.Retryable == true`:

| Failure Class | HTTP Status | Retryable | Retry Behavior |
|--------------|-------------|-----------|----------------|
| Rate limit    | 429         | ✓         | Backoff likely resolves |
| Venue unavailable | 503     | ✓         | Transient outage |
| Bad gateway   | 502         | ✓         | Infrastructure-level |
| Server error  | 500, 5xx    | ✓         | Transient server fault |
| Network failure | —         | ✓         | DNS/TCP/TLS transient |

Not retried (non-retryable): authentication (401/403), client error (400/422/4xx), parse failure, unknown status.

## 4. Integration Point

The `RetrySubmitter` sits between the actor layer (which enforces the safety gate) and the venue adapter:

```
Actor layer
  → SafetyGate.Check()     (kill switch + staleness)
  → RetrySubmitter.SubmitOrder()   ← NEW
    → BinanceFuturesTestnetAdapter.SubmitOrder()
```

The safety gate is checked once before the retry loop — it is NOT re-checked on each retry. This is intentional: once the gate has cleared the intent, retries for transient venue errors should not be blocked by gate read latency or transient gate failures.

## 5. Observability

When retries are exhausted, the returned `Problem` carries:
- `retry_attempts` (int) — total attempts made
- `retry_exhausted` (bool) — always `true` when retries are exhausted

These details merge with existing venue error details (`venue_http_status`, `venue_error_code`), providing a complete failure picture without requiring separate logging infrastructure.

## 6. File Map

| File | Role |
|------|------|
| `internal/application/execution/retry_submitter.go` | RetrySubmitter implementation + RetryPolicy |
| `internal/application/execution/retry_submitter_test.go` | 9 test scenarios covering all abort paths |
