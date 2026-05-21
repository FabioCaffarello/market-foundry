# Strategy Projection Pattern

> Canonical reference for how strategy events are materialized into the read model.

## Overview

The strategy projection pipeline follows the same structural pattern as signal and decision
projections (see `signal-projection-pattern.md`, `decision-projection-pattern.md`, and
`projection-writer-pattern.md`), with adaptations for the strategy domain's distinct
stream, registry, and bucket namespace.

## Pipeline Architecture

```
STRATEGY_EVENTS stream
  -> StrategyConsumer (durable JetStream, per strategy family)
    -> strategyReceivedMessage (actor message)
      -> StrategyProjectionActor
        -> Gate 1: Final=true filter
        -> Gate 2: strat.Validate()
        -> Write: KV latest bucket (monotonicity guard)
```

## Single-Writer Invariant

Each strategy family has exactly one `StrategyProjectionActor` instance per deployment.
This actor is the sole writer to its KV bucket (`STRATEGY_MEAN_REVERSION_ENTRY_LATEST`
for mean_reversion_entry).

No horizontal scaling of projection writers. Scaling happens on the query side
via NATS queue groups (`strategy.query`).

Multiple store instances converge via the monotonicity guard and JetStream
deduplication — the combination ensures that replayed or duplicated events
never regress the read model.

## Materialization Gates

Every strategy event passes through three gates before reaching the KV store:

| Gate | Purpose | Counter on skip |
|------|---------|-----------------|
| Final gate | Only `Final=true` strategies are materialized | `skipped_non_final` |
| Validate gate | Domain validation via `Strategy.Validate()` | `rejected` |
| Monotonicity guard | KV read-before-write; skip if existing timestamp >= incoming | `skipped_stale` / `skipped_dedup` |

## Observability Counters

The projection actor tracks seven counters (atomic int64), logged on `actor.Stopped`:

| Counter | Meaning |
|---------|---------|
| `received` | Total strategy events entering the projection handler |
| `materialized` | Strategies written to KV bucket |
| `skipped_stale` | Existing strategy has a strictly newer timestamp |
| `skipped_dedup` | Existing strategy has the same timestamp |
| `skipped_non_final` | Non-finalized strategies dropped at gate 1 |
| `rejected` | Strategies that failed domain validation at gate 2 |
| `errors` | KV write failures |

Invariant: `received = materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors`.

## Health Tracking

Each strategy family has two dedicated health trackers:

- `strategy-{family}-projection` — records event on successful KV write
- `strategy-{family}-consumer` — records event on successful JetStream consumption

Both are registered with the store binary's health server and visible via `/statusz`.
Idle warnings trigger after 2 minutes of inactivity.

## Bucket Ownership

| Bucket | Owner | Key Format | Storage |
|--------|-------|------------|---------|
| `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | `StrategyProjectionActor[mean_reversion_entry]` | `{source}.{symbol}.{timeframe}` | FileStorage, 64 MB |

No cross-family bucket sharing. Each strategy type owns its bucket exclusively.

## Query Path

```
Gateway HTTP -> NATS request -> strategy.query.mean_reversion_entry.latest -> QueryResponderActor
  -> StrategyKVStore.Get() -> STRATEGY_MEAN_REVERSION_ENTRY_LATEST bucket -> StrategyLatestReply
```

The `QueryResponderActor` opens a **read-only** KV connection to the strategy bucket.
It never writes — all writes go through the projection actor.

## Latest-Only: Intentional Design Choice

The strategy projection is **latest-only by design**. There is no history bucket.
This is an intentional constraint, not a missing feature:

1. **Strategies are ephemeral resolutions** — they reflect the most recent decision state.
   Historical strategies can be re-derived from historical decisions if needed.
2. **Simplicity over completeness** — a latest-only bucket is simpler to reason about,
   cheaper to operate, and easier to make replay-safe.
3. **No history until proven necessary** — opening a history bucket adds complexity
   (TTL management, key design with embedded timestamps, query surface expansion).
   This should only happen when a concrete use case demands it.
4. **Strategy is pre-action, not archival** — the value of a strategy is its current
   directional intent. Once a newer strategy replaces it, the old one has no operational
   relevance. Archival belongs to downstream layers (risk, execution, portfolio).

If history is needed in the future, it should follow the evidence candle history
pattern: separate bucket, embedded timestamp in key, TTL-based retention.

## Activation

Strategy families are opt-in via `pipeline.strategy_families` in store config.
When a family is absent from config, no consumer, projection, or query route is spawned.

Strategy activation requires its dependency chain to be satisfied:
- `mean_reversion_entry` requires `rsi_oversold` in `pipeline.decision_families`
- This is enforced at config validation time by `ValidatePipeline()`

## Projection Authority Rules

1. **Store owns the read model.** The `StrategyProjectionActor` is the sole authority
   for what enters the KV bucket. Derive publishes events; it does not write to KV.
2. **Gateway reads, never writes.** The `QueryResponderActor` opens a read-only KV
   connection. The gateway binary has no path to modify the projection.
3. **No cross-domain writes.** Decision, signal, and evidence projection actors never
   touch strategy buckets. Each domain owns its own KV namespace.
4. **Validation is the projection's responsibility.** Even if derive publishes a
   malformed strategy, the projection gate rejects it. The read model is self-protecting.
5. **Direction enum is enforced at projection.** Only `long`, `short`, `flat` are
   accepted. Any other value is rejected by `Strategy.Validate()`.

## Strategy-Specific Validation

Beyond the structural fields common to all stream families, strategies enforce:

- **Direction enum validation**: must be one of `long`, `short`, `flat`
- **Confidence field**: must not be empty (decimal string)
- **Decisions array**: must contain at least one `DecisionInput`
- **Parameters and Metadata**: optional maps, not validated at projection level

## Known Limitations

1. **Latest-only projection** — no strategy history bucket (intentional, see above).
2. **Ack-before-projection window** — message acked before KV write; gap bounded
   to 1 strategy per partition key.
3. **Single-writer assumption** — monotonicity guard relies on single writer per
   deployment; multiple writers converge but may produce redundant writes.
4. **Single strategy family** — only `mean_reversion_entry` is implemented; registry
   has `LatestSpecByType()` dispatch ready for extension.
5. **No cross-type atomicity** — each strategy family is projected independently.
6. **No strategy-to-risk feedback** — strategy is a terminal analytical layer;
   it does not feed back into any upstream domain.
