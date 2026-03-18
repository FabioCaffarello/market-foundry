# Signal Projection Pattern

> Canonical reference for how signal events are materialized into the read model.

## Overview

The signal projection pipeline follows the same structural pattern as evidence projections
(see `projection-writer-pattern.md`), with adaptations for the signal domain's
distinct stream, registry, and bucket namespace.

## Pipeline Architecture

```
SIGNAL_EVENTS stream
  -> SignalConsumer (durable JetStream, per signal family)
    -> signalReceivedMessage (actor message)
      -> SignalProjectionActor
        -> Gate 1: Final=true filter
        -> Gate 2: sig.Validate()
        -> Write: KV latest bucket (monotonicity guard)
```

## Single-Writer Invariant

Each signal family has exactly one `SignalProjectionActor` instance per deployment.
This actor is the sole writer to its KV bucket (`SIGNAL_RSI_LATEST` for RSI).

No horizontal scaling of projection writers. Scaling happens on the query side
via NATS queue groups (`signal.query`).

Multiple store instances converge via the monotonicity guard and JetStream
deduplication — the combination ensures that replayed or duplicated events
never regress the read model.

## Materialization Gates

Every signal event passes through three gates before reaching the KV store:

| Gate | Purpose | Counter on skip |
|------|---------|-----------------|
| Final gate | Only `Final=true` signals are materialized | `skipped_non_final` |
| Validate gate | Domain validation via `Signal.Validate()` | `rejected` |
| Monotonicity guard | KV read-before-write; skip if existing timestamp >= incoming | `skipped_stale` / `skipped_dedup` |

## Observability Counters

The projection actor tracks six counters (atomic int64), logged on `actor.Stopped`:

| Counter | Meaning |
|---------|---------|
| `materialized` | Signals written to KV bucket |
| `skipped_stale` | Existing signal has a strictly newer timestamp |
| `skipped_dedup` | Existing signal has the same timestamp |
| `skipped_non_final` | Non-finalized signals dropped at gate 1 |
| `rejected` | Signals that failed domain validation at gate 2 |
| `errors` | KV write failures |

## Health Tracking

Each signal family has two dedicated health trackers:

- `signal-{family}-projection` — records event on successful KV write
- `signal-{family}-consumer` — records event on successful JetStream consumption

Both are registered with the store binary's health server and visible via `/statusz`.
Idle warnings trigger after 2 minutes of inactivity.

## Bucket Ownership

| Bucket | Owner | Key Format | Storage |
|--------|-------|------------|---------|
| `SIGNAL_RSI_LATEST` | `SignalProjectionActor[rsi]` | `{source}.{symbol}.{timeframe}` | FileStorage, 64 MB |

No cross-family bucket sharing. Each signal type owns its bucket exclusively.

## Query Path

```
Gateway HTTP -> NATS request -> signal.query.rsi.latest -> QueryResponderActor
  -> SignalKVStore.Get() -> SIGNAL_RSI_LATEST bucket -> SignalLatestReply
```

The `QueryResponderActor` opens a **read-only** KV connection to the signal bucket.
It never writes — all writes go through the projection actor.

## Activation

Signal families are opt-in via `pipeline.signal_families` in store config.
When a family is absent from config, no consumer, projection, or query route is spawned.

## Known Limitations

1. **Latest-only projection** — no signal history bucket (deferred).
2. **Ack-before-projection window** — message acked before KV write; gap bounded
   to 1 signal per partition key.
3. **Single-writer assumption** — monotonicity guard relies on single writer per
   deployment; multiple writers converge but may produce redundant writes.
4. **No cross-bucket atomicity** — not applicable yet (single bucket per family).
