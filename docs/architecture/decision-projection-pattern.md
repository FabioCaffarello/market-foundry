# Decision Projection Pattern

> Canonical reference for how decision events are materialized into the read model.

## Overview

The decision projection pipeline follows the same structural pattern as signal projections
(see `signal-projection-pattern.md` and `projection-writer-pattern.md`), with adaptations
for the decision domain's distinct stream, registry, and bucket namespace.

## Pipeline Architecture

```
DECISION_EVENTS stream
  -> DecisionConsumer (durable JetStream, per decision family)
    -> decisionReceivedMessage (actor message)
      -> DecisionProjectionActor
        -> Gate 1: Final=true filter
        -> Gate 2: dec.Validate()
        -> Write: KV latest bucket (monotonicity guard)
```

## Single-Writer Invariant

Each decision family has exactly one `DecisionProjectionActor` instance per deployment.
This actor is the sole writer to its KV bucket (`DECISION_RSI_OVERSOLD_LATEST` for rsi_oversold).

No horizontal scaling of projection writers. Scaling happens on the query side
via NATS queue groups (`decision.query`).

Multiple store instances converge via the monotonicity guard and JetStream
deduplication — the combination ensures that replayed or duplicated events
never regress the read model.

## Materialization Gates

Every decision event passes through three gates before reaching the KV store:

| Gate | Purpose | Counter on skip |
|------|---------|-----------------|
| Final gate | Only `Final=true` decisions are materialized | `skipped_non_final` |
| Validate gate | Domain validation via `Decision.Validate()` | `rejected` |
| Monotonicity guard | KV read-before-write; skip if existing timestamp >= incoming | `skipped_stale` / `skipped_dedup` |

## Observability Counters

The projection actor tracks seven counters (atomic int64), logged on `actor.Stopped`:

| Counter | Meaning |
|---------|---------|
| `received` | Total decision events entering the projection handler |
| `materialized` | Decisions written to KV bucket |
| `skipped_stale` | Existing decision has a strictly newer timestamp |
| `skipped_dedup` | Existing decision has the same timestamp |
| `skipped_non_final` | Non-finalized decisions dropped at gate 1 |
| `rejected` | Decisions that failed domain validation at gate 2 |
| `errors` | KV write failures |

Invariant: `received = materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors`.

## Health Tracking

Each decision family has two dedicated health trackers:

- `decision-{family}-projection` — records event on successful KV write
- `decision-{family}-consumer` — records event on successful JetStream consumption

Both are registered with the store binary's health server and visible via `/statusz`.
Idle warnings trigger after 2 minutes of inactivity.

## Bucket Ownership

| Bucket | Owner | Key Format | Storage |
|--------|-------|------------|---------|
| `DECISION_RSI_OVERSOLD_LATEST` | `DecisionProjectionActor[rsi_oversold]` | `{source}.{symbol}.{timeframe}` | FileStorage, 64 MB |

No cross-family bucket sharing. Each decision type owns its bucket exclusively.

## Query Path

```
Gateway HTTP -> NATS request -> decision.query.rsi_oversold.latest -> QueryResponderActor
  -> DecisionKVStore.Get() -> DECISION_RSI_OVERSOLD_LATEST bucket -> DecisionLatestReply
```

The `QueryResponderActor` opens a **read-only** KV connection to the decision bucket.
It never writes — all writes go through the projection actor.

## Latest-Only: Intentional Design Choice

The decision projection is **latest-only by design**. There is no history bucket.
This is an intentional constraint, not a missing feature:

1. **Decisions are ephemeral evaluations** — they reflect the most recent signal state.
   Historical decisions can be re-derived from historical signals if needed.
2. **Simplicity over completeness** — a latest-only bucket is simpler to reason about,
   cheaper to operate, and easier to make replay-safe.
3. **No history until proven necessary** — opening a history bucket adds complexity
   (TTL management, key design with embedded timestamps, query surface expansion).
   This should only happen when a concrete use case demands it.

If history is needed in the future, it should follow the evidence candle history
pattern: separate bucket, embedded timestamp in key, TTL-based retention.

## Activation

Decision families are opt-in via `pipeline.decision_families` in store config.
When a family is absent from config, no consumer, projection, or query route is spawned.

## Projection Authority Rules

1. **Store owns the read model.** The `DecisionProjectionActor` is the sole authority
   for what enters the KV bucket. Derive publishes events; it does not write to KV.
2. **Gateway reads, never writes.** The `QueryResponderActor` opens a read-only KV
   connection. The gateway binary has no path to modify the projection.
3. **No cross-domain writes.** Signal and evidence projection actors never touch
   decision buckets. Each domain owns its own KV namespace.
4. **Validation is the projection's responsibility.** Even if derive publishes a
   malformed decision, the projection gate rejects it. The read model is self-protecting.

## Known Limitations

1. **Latest-only projection** — no decision history bucket (intentional, see above).
2. **Ack-before-projection window** — message acked before KV write; gap bounded
   to 1 decision per partition key.
3. **Single-writer assumption** — monotonicity guard relies on single writer per
   deployment; multiple writers converge but may produce redundant writes.
4. **Single decision family** — only `rsi_oversold` is implemented; registry has
   `LatestSpecByType()` dispatch ready for extension.
5. **No cross-type atomicity** — each decision family is projected independently.
