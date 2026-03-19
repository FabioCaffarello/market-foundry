# Execution Failure Recovery Model

## Purpose

This document defines the failure model for the `execution` domain's runtime paths. It classifies failures by recoverability, documents the expected system behavior under each failure mode, and establishes the recovery discipline that operators and future stages can rely on.

## Scope

Covers the execution publish path (derive → NATS JetStream) and the execution projection path (NATS JetStream → KV store). Does not cover venue integration, OMS, or external system failures — those remain out of scope until venue integration is activated.

## Failure Classification

### Recoverable (Transient)

| Failure | Location | Recovery Behavior |
|---------|----------|-------------------|
| NATS publish timeout | `ExecutionPublisherActor` | Single retry after 500ms backoff. If both attempts fail, error is logged, tracker records error, event is dropped. |
| NATS connection blip | `ExecutionPublisher.PublishExecution` | Returns `Unavailable` problem → triggers retry in actor. |
| KV store put timeout | `ExecutionKVStore.Put` | Returns `Unavailable` problem → logged at ERROR, error count incremented, event dropped from projection. Consumer has already acked. |
| Consumer decode transient | `ExecutionConsumer.onMessage` | Returns non-InvalidArgument problem → NAK, NATS redelivers (up to MaxDeliver=5). |

### Non-Recoverable (Terminal)

| Failure | Location | Recovery Behavior |
|---------|----------|-------------------|
| Unknown execution type | `ExecutionPublisher.PublishExecution` | Returns `InvalidArgument` → no retry, error logged immediately. |
| Encoding failure | `encodeEvent` | Returns `Internal` problem → no retry, structural bug. |
| Consumer decode corruption | `ExecutionConsumer.onMessage` | Returns `InvalidArgument` → message terminated (Term), permanently removed from consumer. |
| KV marshal failure | `ExecutionKVStore.Put` | Returns `Internal` problem → structural bug, no retry. |
| Domain validation failure | `ExecutionProjectionActor.onExecution` | Intent rejected at Gate 2, counted as `rejected`, not retried. |

### Out of Scope (Not Yet Modeled)

| Failure | Why |
|---------|-----|
| NATS cluster partition | Handled by NATS clustering; no application-level recovery. |
| KV bucket corruption | Requires manual intervention; no auto-recovery. |
| Actor system crash | Hollywood actor engine handles restart; no custom supervision. |
| Venue/exchange errors | No venue integration exists yet. |

## Retry Discipline

### Publisher Path (derive → JetStream)

- **Max attempts:** 2 (initial + 1 retry)
- **Backoff:** 500ms fixed delay between attempts
- **Retry condition:** Only `Unavailable` errors (transient NATS failures)
- **Non-retryable:** `InvalidArgument`, `Internal` errors fail immediately
- **On exhaustion:** Error logged at ERROR level with full context (type, source, symbol, timeframe, correlation_id), tracker records error

### Consumer Path (JetStream → projection actor)

- **NATS-managed retry:** AckWait=30s, MaxDeliver=5
- **Decode errors:** InvalidArgument → Term (permanent), other → NAK (retry)
- **Max delivery exhaustion:** Logged at ERROR when NumDelivered reaches MaxDeliver
- **Post-decode:** Consumer acks immediately after dispatching to projection actor via actor message. Projection failures are NOT fed back to the consumer.

### Projection Path (actor → KV store)

- **No application-level retry** on KV Put failures. The projection is a single-writer and errors at this layer indicate infrastructure-level problems (NATS KV unavailable) that retry is unlikely to resolve within the same message handler.
- **Error tracking:** KV failures increment `errors` stat and call `tracker.RecordError()` for health surface visibility.
- **Rationale:** Retrying KV puts inside the projection would block the actor mailbox, creating backpressure on the consumer. The single-retry approach is reserved for the publish path where the cost of dropping is highest.

## Known Gap: Consumer-Projection Decoupling

The execution consumer acks the NATS message before the projection completes its KV write. If the projection encounters a KV error after the consumer has acked, the event is lost — NATS will not redeliver it.

**Why this is acceptable today:**
1. KV errors at this layer are rare (infrastructure failures)
2. The projection is latest-only — the next successful write for the same partition key will restore correctness
3. Adding consumer-projection backpressure would require significant architectural changes (request/response between actors) and is deferred to a future stage

**Mitigation:**
- `errors` stat count in projection detects data loss
- `tracker.RecordError()` makes the failure visible on `/statusz`
- Stats invariant check on actor stop verifies received == sum of all outcomes

## Health and Observability

### Publisher Actor
- `published` counter: successful publishes
- `errors` counter: failed publishes (after retry exhaustion)
- `tracker.RecordEvent()`: on success
- `tracker.RecordError()`: on failure
- Stats logged on actor stop

### Projection Actor
- `received`, `materialized`, `skippedStale`, `skippedDedup`, `skippedNonFinal`, `rejected`, `errors` counters
- `checkStatsInvariant()` on stop: verifies received == sum of outcomes
- `tracker.RecordEvent()`: on successful materialization
- `tracker.RecordError()`: on KV put failure

### Consumer
- `delivered`, `redelivered`, `terminated`, `nakked` counters
- Max delivery exhaustion warning logged at ERROR level
- `tracker.RecordEvent()`: on each decoded event (consumer-side)

### `/statusz` Endpoint
- `event_count`: successful event processing
- `error_count`: errors encountered (new in S76)
- `idle_warning`: component has been idle beyond threshold
- Error counts visible per tracker for targeted diagnostics
