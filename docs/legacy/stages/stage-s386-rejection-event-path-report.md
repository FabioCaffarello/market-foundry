# Stage S386 — Rejection Event Path and Write-Path Observability

**Status:** Complete
**Date:** 2026-03-22
**Predecessor:** S385 (Write-Path Integration by Execution Mode)
**Scope:** Rejection event path for venue_live mode; write-path observability closure.

---

## Executive Summary

S386 closes the primary observability gap identified by S385: venue rejections in `venue_live` mode previously existed only as `Problem` return values with structured logging, producing no downstream event. After S386, every failed venue submission emits a `VenueOrderRejectedEvent` to a dedicated NATS JetStream stream, making rejections auditable, queryable, and observable by downstream consumers.

The implementation is minimal and additive — no existing behavior was changed, no interfaces were redesigned, and the lifecycle state machine remains coherent with S383.

## Problem Statement

S385 proved 19 integration tests across all execution modes but explicitly flagged:

> "Rejections return Problem, not VenueOrderFilledEvent; no downstream event."

The venue_live rejection path was:
```
Venue → Problem → Actor logs error → return (silent drop)
```

No NATS event was published. Store projections, ClickHouse writers, and gateway queries had zero visibility into rejections.

## Solution

### Domain: VenueOrderRejectedEvent

New event type in `internal/domain/execution/events.go`:

- Implements `events.Event` interface
- Carries `ExecutionIntent` with `Status=rejected`, `Final=true`
- Includes `RejectionCode`, `RejectionReason`, `VenueDetails` for audit trail
- Event name: `venue_order_rejected`

### Transport: NATS Stream and Registry

New stream and specs in `internal/adapters/nats/natsexecution/registry.go`:

| Concern | Value |
|---|---|
| Stream | `EXECUTION_REJECTION_EVENTS` |
| Subject | `execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}` |
| Type | `execution.rejection.v1.venue_market_order_rejected` |
| Storage | FileStorage, 72h, 128 MB |

Consumer specs defined:
- `store-execution-venue-rejection` (store projection)
- `writer-execution-venue-rejection` (ClickHouse persistence)

### Publisher: PublishRejection

New method on `Publisher` in `internal/adapters/nats/natsexecution/publisher.go`:
- Ensures `EXECUTION_REJECTION_EVENTS` stream at startup
- Publishes with deduplication key: `rejection:{source}:{symbol}:{timeframe}:{ts_unix}`

### Actor: publishRejection

New method on `VenueAdapterActor` in `internal/actors/scopes/execute/venue_adapter_actor.go`:
- Called after every `prob != nil` from venue submit (both non-retryable and exhausted-retryable)
- Mutates intent to `Status=rejected`, `Final=true`
- Constructs event with full correlation chain from incoming message
- Tracks `rejected` and `rejected:{symbol}` health counters
- Logs structured `"venue order rejected"` message

### Behavior Preservation

- `Problem` return value is unchanged
- Structured error logging is unchanged (rejection event is additive)
- `VenueOrderFilledEvent` path is unchanged
- Kill switch / staleness gate blocks do NOT produce rejection events (intent never reached venue)

## Files Changed

### Code Changes

| File | Change |
|---|---|
| `internal/domain/execution/events.go` | Added `VenueOrderRejectedEvent`, `EventVenueOrderRejected` |
| `internal/adapters/nats/natsexecution/registry.go` | Added `VenueMarketOrderRejected` spec, `EXECUTION_REJECTION_EVENTS` stream, consumer specs |
| `internal/adapters/nats/natsexecution/publisher.go` | Added `PublishRejection()`, stream creation at startup |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Added `publishRejection()`, `problem` import, rejection counters in stats |

### Test Files

| File | Tests | Focus |
|---|---|---|
| `internal/domain/execution/s386_rejection_event_test.go` | 7 | Event interface, correlation, metadata, lifecycle, validation |
| `internal/actors/scopes/execute/s386_rejection_event_path_test.go` | 5 | Event construction, non-retryable/retryable, timestamp, fills |
| `internal/adapters/nats/natsexecution/s386_rejection_registry_test.go` | 7 | Registry spec, stream separation, consumer conventions |

### Documentation

| File | Purpose |
|---|---|
| `docs/architecture/rejection-event-path-and-write-path-observability.md` | Event flow, NATS contracts, backward compatibility |
| `docs/architecture/rejection-event-contract-auditability-and-lifecycle-alignment.md` | Payload contract, lifecycle alignment, audit properties |
| `docs/stages/stage-s386-rejection-event-path-report.md` | This report |

## Test Evidence

**19 S386 tests — all pass.**

### Domain Tests (7)

| Test | Validates |
|---|---|
| `TestS386_RejectedEvent_ImplementsEventInterface` | Event interface compliance |
| `TestS386_RejectedEvent_PreservesCorrelationChain` | Metadata and intent-level correlation |
| `TestS386_RejectedEvent_CarriesRejectionMetadata` | RejectionCode, RejectionReason, VenueDetails |
| `TestS386_RejectedEvent_IntentIsTerminalAndFinal` | Status=rejected, Final=true, IsTerminal() |
| `TestS386_RejectedEvent_LifecycleTransitionValid` | submitted→rejected valid, rejected→* invalid |
| `TestS386_RejectedEvent_IntentValidatesWithRejectedStatus` | Validate() passes with rejected status |
| `TestS386_RejectedEvent_EventNameIsDistinctFromFill` | Event names don't collide |

### Actor Integration Tests (5)

| Test | Validates |
|---|---|
| `TestS386_RejectionEventConstruction_FromNonRetryableProblem` | Full event construction from Problem, field preservation |
| `TestS386_RejectionEventConstruction_FromExhaustedRetryable` | Exhausted retryable produces rejection event |
| `TestS386_RejectionEvent_LifecycleTransitionFromSubmitted` | submitted→rejected valid, rejected terminal |
| `TestS386_RejectionEvent_PreservesOriginalIntentTimestamp` | Intent timestamp survives, event OccurredAt is separate |
| `TestS386_RejectionEvent_NoFillsOnRejection` | Zero fills, empty FilledQuantity |

### Registry Tests (7)

| Test | Validates |
|---|---|
| `TestS386_Registry_RejectionEventSpecExists` | Spec fields populated |
| `TestS386_Registry_RejectionSubjectFollowsConvention` | Subject pattern correctness |
| `TestS386_Registry_RejectionTypeFollowsConvention` | Type string correctness |
| `TestS386_Registry_RejectionStreamIsSeparateFromFills` | Separate stream from fills |
| `TestS386_Registry_RejectionStreamSubjects` | Stream subject wildcard |
| `TestS386_StoreRejectionConsumer_FollowsConventions` | Store consumer durable/subject/type |
| `TestS386_WriterRejectionConsumer_FollowsConventions` | Writer consumer durable/subject |

### Regression

All existing tests in affected packages pass (S383, S384, S385 tests unaffected).

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Rejections in venue_live produce auditable event | **Met** — `VenueOrderRejectedEvent` published to NATS |
| Primary write-path observability gap closed | **Met** — every submit outcome now produces an event |
| Lifecycle coherent with S383 | **Met** — submitted→rejected valid, rejected terminal, Final=true |
| Solution small and without OMS scope inflation | **Met** — 4 code files changed, 3 test files added, no new interfaces |

## Guard Rails Verification

| Guard Rail | Status |
|---|---|
| No cancel-order API | **Respected** |
| No async protocol for `sent` | **Respected** |
| No broad dashboards | **Respected** |
| No ExecutionIntent redesign | **Respected** — two fields mutated (Status, Final), no structural change |
| No scope inflation beyond rejection path | **Respected** |

## Limitations and Deferred Work

1. **Rejection projection actor not wired**: Consumer specs exist but no store actor consumes rejection events yet. The projection actor wiring is a future stage concern.
2. **No rejection KV bucket**: A `EXECUTION_VENUE_REJECTION_LATEST` KV bucket is not yet created.
3. **No ClickHouse schema for rejections**: The writer consumer spec exists but the ClickHouse table schema for rejections is deferred.
4. **Gate-blocked intents are silent**: Kill switch and staleness blocks produce log entries but no NATS events. This is by design — these intents never reached the venue.

## Recommended Preparation for S387

Three options, ordered by risk closure value:

1. **OMS Foundation — Read-Path Queries (recommended)**: Wire rejection and fill projections into queryable read models. The gateway can then expose order status including rejections.

2. **Cancel-Order Path**: Implement `accepted → cancelled` transition via exchange cancel API. This would complete the third terminal state path.

3. **Rejection Projection Wiring**: Wire `store-execution-venue-rejection` consumer to materialize rejections into a NATS KV bucket, making them queryable by the gateway.
