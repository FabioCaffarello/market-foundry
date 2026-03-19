# Execution Projection Failure Semantics

## Purpose

This document defines the precise failure semantics of the execution read-side projection, including what happens at each gate, how errors propagate, and what operators should expect when things go wrong.

## Projection Pipeline Overview

```
NATS JetStream
    │
    ▼
ExecutionConsumer (decode + ack)
    │  actor.Send (fire-and-forget)
    ▼
ExecutionProjectionActor
    │  Gate 1: Final check
    │  Gate 2: Domain validation
    │  Gate 3: KV monotonicity guard
    ▼
ExecutionKVStore (NATS KV)
```

## Gate Semantics

### Gate 1: Final Check

**Condition:** `intent.Final == false`

| Aspect | Behavior |
|--------|----------|
| Action | Skip silently |
| Stat | `skippedNonFinal` incremented |
| Log | None (high frequency, intentional) |
| Recoverable | N/A — not an error |

Non-final intents are intermediate evaluations. The projection only materializes final intents to maintain read-model consistency.

### Gate 2: Domain Validation

**Condition:** `intent.Validate()` returns non-nil problem

| Aspect | Behavior |
|--------|----------|
| Action | Skip with warning |
| Stat | `rejected` incremented |
| Log | WARN with error message, type, source, symbol, timeframe |
| Recoverable | No — structural data issue, redelivery won't help |

Validation rejects intents with missing required fields (type, source, symbol, side, status, quantity, risk). A rejected intent indicates a bug in the upstream evaluator or publisher.

### Gate 3: KV Monotonicity Guard

**Condition:** `store.Put()` returns non-nil problem

| Aspect | Behavior |
|--------|----------|
| Action | Drop event, log error |
| Stat | `errors` incremented |
| Health | `tracker.RecordError()` called |
| Log | ERROR with message, code, type, source, symbol, timeframe, side, status, correlation_id |
| Recoverable | Depends on cause — transient KV unavailability will self-heal on next event |

### Gate 3: Monotonicity Outcomes (no error)

| Result | Behavior | Stat |
|--------|----------|------|
| `PutWritten` | New or newer intent materialized | `materialized` |
| `PutSkippedStale` | Existing intent has newer timestamp | `skippedStale` |
| `PutSkippedDuplicate` | Same timestamp already exists | `skippedDedup` |

## KV Store Error Semantics

### `ExecutionKVStore.Put`

```
nil receiver/store  → (PutWritten, Unavailable)  — store not initialized
Get existing fails  → proceed (treat as first write)
Timestamp stale     → (PutSkippedStale, nil)
Timestamp duplicate → (PutSkippedDuplicate, nil)
Marshal failure     → (PutWritten, Internal)      — structural bug
KV put failure      → (PutWritten, Unavailable)   — transient infra error
Success             → (PutWritten, nil)
```

**Contract:** Callers MUST check `prob != nil` before interpreting `PutResult`. When `prob != nil`, the `PutResult` value is undefined and must not be used for branching.

### `ExecutionKVStore.Get`

```
nil receiver/store  → (nil, Unavailable)
Key not found       → (nil, nil)            — not an error
KV get failure      → (nil, Unavailable)    — transient infra error
Unmarshal failure   → (nil, Internal)       — corrupted KV data
Validation failure  → (nil, Internal)       — corrupted/incomplete KV entry
Success             → (*intent, nil)
```

Post-read validation detects KV data corruption. A validation failure on Get means the KV entry was written in a corrupted state — this is a non-recoverable error that requires manual investigation.

## Consumer-Projection Contract

The consumer and projection communicate via fire-and-forget actor messages. This has specific implications:

1. **Consumer acks before projection completes.** The consumer calls `handler()` which does `actor.Send()` (non-blocking), then acks the NATS message.

2. **Projection errors don't cause redelivery.** If the projection fails on KV put, the consumer has already acked. The event will not be redelivered.

3. **Latest-only semantics provide natural recovery.** Since the projection stores only the latest intent per partition key, the next successful write for the same key restores correctness. This makes the fire-and-forget contract acceptable for latest-only projections.

4. **Stats invariant detects dropped events.** On actor stop, `checkStatsInvariant()` verifies that `received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors`. A mismatch indicates a code-level bug in outcome tracking.

## Error Propagation Summary

| Layer | Error Source | Propagation | Recovery |
|-------|-------------|-------------|----------|
| Consumer | Decode failure | terminateOrNak → NATS | Term (permanent) or NAK (retry) |
| Consumer | Ack failure | Logged, NATS redelivers on timeout | Automatic via AckWait |
| Projection | Validation | Absorbed, stat counted | None needed |
| Projection | KV put | Absorbed, stat counted, tracker error | Self-heals on next event |
| KV Store | Marshal | Propagated as Internal problem | None — structural bug |
| KV Store | NATS KV | Propagated as Unavailable problem | Transient, self-heals |

## Operational Diagnostics

### Detecting Projection Failures

1. **`/statusz` endpoint**: Check `error_count` for execution projection and consumer trackers
2. **Structured logs**: Filter for `actor=execution-projection` at ERROR level
3. **Stats on shutdown**: Final stats log shows all outcome counters; invariant violation logged at ERROR

### Common Failure Scenarios

| Symptom | Likely Cause | Action |
|---------|-------------|--------|
| `error_count` rising on projection tracker | KV store unavailable | Check NATS KV health |
| `rejected` count rising | Upstream evaluator producing invalid intents | Check derive logs |
| Stats invariant violated on stop | Bug in projection outcome tracking | Code investigation required |
| `idle_warning` on execution trackers | No events flowing through pipeline | Check upstream derive pipeline |

## Limitations

1. **No application-level retry on KV put.** Transient KV failures cause the event to be dropped from the projection. Recovery depends on the next event for the same partition key.

2. **No dead-letter queue for projection failures.** Events that fail KV put are counted but not stored for retry.

3. **No backpressure from projection to consumer.** The consumer cannot slow down based on projection failure rate.

These limitations are acceptable for a latest-only projection with low-frequency execution events. They should be revisited if execution volume increases significantly or if history preservation is required.
