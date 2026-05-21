# Rejection Event Path and Write-Path Observability

> S386 — Closes the primary observability gap in the venue_live write-path identified by S385.

## Purpose

This document defines the rejection event path for the `venue_live` execution mode. Prior to S386, venue rejections were returned as `Problem` values to the actor layer and logged, but never published as auditable events. Downstream consumers (store projections, ClickHouse writers, gateway queries) had no visibility into rejections.

S386 introduces `VenueOrderRejectedEvent` as the canonical audit trail for venue rejections, making the write-path fully observable for both successful and failed order submissions.

## Gap Closed

| Before S386 | After S386 |
|---|---|
| Venue rejection returns `Problem` to actor | `Problem` still returned (behavior preserved) |
| Actor logs error and returns early | Actor logs error **and** publishes `VenueOrderRejectedEvent` |
| No downstream event for rejections | Rejection event published to `EXECUTION_REJECTION_EVENTS` stream |
| Rejection not queryable from store/gateway | Store and writer consumers can subscribe and materialize |
| No rejection counter in health stats | `rejected` counter tracked per symbol |

## Rejection Event Flow

```
Venue (e.g. Binance Futures Testnet)
    │ returns HTTP 4xx / error response
    ▼
BinanceFuturesTestnetAdapter.SubmitOrder()
    │ returns (empty receipt, Problem{Code: VAL_INVALID_ARGUMENT, Retryable: false})
    ▼
RetrySubmitter.SubmitOrder()
    │ passes through (non-retryable) or exhausts retries (retryable)
    ▼
VenueAdapterActor.onIntent()
    │ logs error (existing behavior, preserved)
    │ calls publishRejection() ← NEW (S386)
    ▼
VenueOrderRejectedEvent constructed:
  - Intent.Status = rejected
  - Intent.Final = true
  - RejectionCode = Problem.Code
  - RejectionReason = Problem.Message
  - VenueDetails = Problem.Details
  - Metadata.CorrelationID = incoming event's CorrelationID
  - Metadata.CausationID = incoming event's ID
    ▼
Publisher.PublishRejection()
    │ NATS JetStream publish
    ▼
Stream: EXECUTION_REJECTION_EVENTS
  Subject: execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}
  Type: execution.rejection.v1.venue_market_order_rejected
    ▼
Consumers: store-execution-venue-rejection, writer-execution-venue-rejection
```

## What Produces a Rejection Event

Every `Problem` returned by the venue submit pipeline produces a rejection event, regardless of whether the original problem was:

1. **Non-retryable** (true venue rejection): e.g. insufficient margin (HTTP 400, code -2019), invalid parameters, authentication failure.
2. **Exhausted retryable**: After `RetrySubmitter` exhausts all retry attempts, the final `Problem` is surfaced to the actor. This also produces a rejection event because the intent is terminal — no further processing will occur.

The distinction between these two cases is preserved in `VenueDetails`:
- Non-retryable: `venue_http_status`, `venue_error_code` present
- Exhausted retryable: `retry_attempts`, `retry_exhausted` present

## What Does NOT Produce a Rejection Event

- **Gate blocks** (kill switch, staleness): These prevent venue submission entirely. No rejection event because the intent never reached the venue.
- **Successful submissions**: These produce `VenueOrderFilledEvent` as before.
- **Publisher failures**: If the rejection event fails to publish, the error is logged but no secondary event is emitted (no infinite recursion).

## NATS Subject and Stream Contract

| Concern | Value |
|---|---|
| Stream | `EXECUTION_REJECTION_EVENTS` |
| Subject pattern | `execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}` |
| Event type | `execution.rejection.v1.venue_market_order_rejected` |
| Storage | FileStorage |
| Max age | 72 hours |
| Max bytes | 128 MB |
| Deduplication key | `rejection:{source}:{symbol}:{timeframe}:{timestamp_unix}` |

### Consumer Specs

| Consumer | Durable Name | Purpose |
|---|---|---|
| Store projection | `store-execution-venue-rejection` | Materialize to KV for queryability |
| Writer persistence | `writer-execution-venue-rejection` | Persist to ClickHouse for analytics |

## Observability Counters

The venue adapter actor tracks rejection events via health counters:

- `rejected` — total rejection events published
- `rejected:{symbol}` — per-symbol rejection count

These complement existing counters: `processed`, `filled`, `skipped_stale`, `skipped_halt`, `errors`.

## Backward Compatibility

- `VenueOrderFilledEvent` is unchanged; existing fill path is not affected.
- The `Problem` return value from `SubmitOrder()` is unchanged.
- Structured error logging is unchanged (rejection event publication is additive).
- The `EXECUTION_FILL_EVENTS` stream is unchanged.
- No existing consumers are modified.

## Limitations

1. **No rejection projection actor yet**: Consumer specs are defined but the store/writer actors do not yet consume rejection events. This is a wiring task for a future stage.
2. **No rejection KV bucket**: A `EXECUTION_VENUE_REJECTION_LATEST` KV bucket is not yet created. Rejection materialization will be addressed when the projection actor is wired.
3. **Single-event-per-intent**: Each failed submission produces exactly one rejection event. If the retry submitter makes 3 attempts before giving up, only the final failure produces the event.
