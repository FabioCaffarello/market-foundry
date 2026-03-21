# Venue Pipeline: Decorator Order, Invariants, and Limits

> **Stage:** S328
> **Date:** 2026-03-21
> **Scope:** Documents the canonical decorator order and its constraints
> **Predecessor:** S327 (Production Wiring Tranche Charter)

---

## Canonical Decorator Order

The venue submit pipeline uses a fixed decorator chain. The order is not
arbitrary — it encodes semantic constraints:

```
[3] Post200Reconciler    — outermost
[2] RetrySubmitter       — middle
[1] rawAdapter           — innermost (BinanceFuturesTestnetAdapter)
```

### Order Invariants

**INV-DO-1: Retry wraps the raw adapter, not the reconciler.**
RetrySubmitter retries the raw adapter's responses directly. body-read-failure-after-200
is non-retryable (`Retryable: false`), so it passes through the retry loop without
triggering additional attempts. If the reconciler were inside the retry loop, the
recovery query would execute on every retry iteration — incorrect and wasteful.

**INV-DO-2: Reconciler wraps the retry layer, not the raw adapter.**
The reconciler sees the final result after all retry attempts. This ensures that
transient failures are retried first, and only genuine body-read-failure-after-200
(a non-retryable condition) triggers reconciliation.

**INV-DO-3: Safety gate is outside the decorator chain.**
The safety gate (kill switch + staleness) runs in `onIntent()` before calling
`a.venue.SubmitOrder()`. It is not a decorator — it's a pre-submit guard that
decides whether to call the pipeline at all. This separation ensures the gate
evaluation is not affected by retry or reconciliation logic.

**INV-DO-4: Halt checker operates at two levels independently.**
The safety gate's kill switch check (Gate 1) blocks new intents. The
RetrySubmitter's `WithHaltChecker` checks the kill switch between retry attempts.
These are independent checks at different scopes: one guards entry, the other
guards continuation.

---

## Decorator Contract

All decorators implement `ports.VenuePort`:

```go
SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
```

This uniform interface enables arbitrary composition. Each decorator:
- Delegates to its inner `VenuePort`
- May inspect the returned `Problem` for specific markers
- May modify the `Problem` (add metadata) but never the `VenueOrderRequest`
- Never modifies the `ExecutionIntent` on the request side (PGR-08)

---

## Error Flow Through the Stack

| Error Type | Retryable? | RetrySubmitter Action | Post200Reconciler Action |
|------------|------------|----------------------|-------------------------|
| Transient (503, 502, 429) | Yes | Retries up to MaxAttempts | Passes through (not body-read) |
| Client error (401, 4xx) | No | Returns immediately | Passes through (not body-read) |
| Venue error code override (-1001, -1003, -1015) | Yes | Retries | Passes through |
| body-read-failure-after-200 | No | Returns immediately (no retry) | Intercepts, queries venue |
| Context cancelled | No | Aborts loop | Passes through |
| Kill switch halt | No | Aborts loop with `retry_halted` | Passes through |
| Deadline exceeded | No | Aborts loop with `retry_deadline_exceeded` | Passes through |

---

## Metadata Enrichment Path

```
rawAdapter returns Problem
  → RetrySubmitter adds: retry_attempts, retry_exhausted/halted/deadline_exceeded
    → Post200Reconciler adds: reconciliation_attempted, reconciliation_failed, reconciliation_error
      → VenueAdapterActor.onIntent() logs all metadata fields
```

Metadata is additive: each layer appends to `Problem.Details` without removing
upstream fields. The venue adapter actor's error logging extracts retry metadata
keys for structured observability.

---

## Limits

### What this composition does NOT cover

1. **Multi-venue routing.** The pipeline is single-venue. Multi-venue would
   require a router before the decorator chain, not a new decorator.

2. **Order management (OMS).** The Post200Reconciler handles exactly one case:
   body-read-failure-after-200. It is not a general-purpose order recovery system.

3. **Retry policy tuning.** `DefaultRetryPolicy()` is used. Policy parameters
   are not config-driven. Tuning is deferred (R-S323-3).

4. **Reconciliation timeout tuning.** The query timeout uses the default (10s).
   It is not config-driven.

5. **Circuit breaker.** There is no circuit breaker in the chain. The kill switch
   serves as a manual circuit breaker. Automatic circuit breaking is not in scope.

6. **Middleware observability (OpenTelemetry, tracing).** Observability is via
   structured logs and health counters. Distributed tracing integration is deferred.

7. **Paper adapter reconciliation.** The paper adapter does not implement
   `VenueQueryPort`, so the reconciler is inactive for paper mode. This is correct:
   paper mode has no body-read-failure scenario.

### Composition constraints for future decorators

Any new decorator added to this chain must:

1. Implement `ports.VenuePort`.
2. Not modify `VenueOrderRequest.Intent` (PGR-08).
3. Use `Problem.Details` for metadata enrichment, not new error types.
4. Be nil-safe for optional hooks.
5. Be inserted at the correct position relative to retry and reconciliation
   semantics — document the rationale.
6. Not change the kill switch / staleness gate semantics.
