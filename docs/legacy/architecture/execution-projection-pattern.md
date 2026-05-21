# Execution Projection Pattern

## Purpose

This document defines the projection ownership, authority, and query path for the `execution` domain in Market Foundry. It makes explicit the invariants that govern how execution intents are materialized, read, and served.

## Projection Authority

Each execution family (e.g., `paper_order`) has exactly **one projection actor** that is the sole writer for its KV bucket.

| Family        | Projection Actor                     | Bucket                           | Semantics    |
|---------------|--------------------------------------|----------------------------------|--------------|
| `paper_order` | `execution-paper-order-projection`   | `EXECUTION_PAPER_ORDER_LATEST`   | latest-only  |

**Invariant**: No other actor, process, or tool may write to a projection's bucket. The projection actor is the single source of truth for materialized state.

## Semantics: Latest-Only

The execution projection uses **latest-only** semantics:

- Each key in the KV bucket represents a unique `{source}.{symbol}.{timeframe}` partition.
- Only the most recent finalized execution intent is stored per partition.
- There is no history bucket for execution — this is an intentional design choice.
- History may be introduced in a future stage if operationally justified, but must not be assumed.

## Three-Gate Pipeline

Every execution event passes through three gates before materialization:

### Gate 1: Final Guard

```
if !intent.Final → skip (skippedNonFinal++)
```

Only finalized execution intents are materialized. Non-final intents are discarded at the projection level.

### Gate 2: Domain Validation

```
if intent.Validate() fails → reject (rejected++)
```

The full domain validation runs before any KV write. Invalid intents are rejected with a warning log.

### Gate 3: Monotonicity Guard

```
if existing.Timestamp > intent.Timestamp → skip stale (skippedStale++)
if existing.Timestamp == intent.Timestamp → skip duplicate (skippedDedup++)
```

The KV adapter enforces timestamp-based monotonicity. This prevents out-of-order or duplicate writes at the storage layer.

## Stats Invariant

The projection actor tracks an accounting invariant:

```
received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors
```

This invariant is checked on actor shutdown. A violation is logged at ERROR level — it indicates a logic bug in the gate pipeline.

## Query Path

```
HTTP client
  → Gateway (GET /execution/:type/latest?source=...&symbol=...&timeframe=...)
    → ExecutionGateway (NATS request/reply)
      → QueryResponderActor (store)
        → ExecutionKVStore.Get()
          → NATS KV bucket read
            → Post-read validation
```

**Key properties**:
- The query path never touches derive actors or the event stream.
- Query reads are served from the materialized KV bucket only.
- Post-read validation catches corrupted or incomplete data in KV.
- Not-found is represented as `nil` intent in the reply, not as an error.

## Ownership Boundaries

| Concern              | Owner                         |
|----------------------|-------------------------------|
| Event production     | Derive: `PaperOrderEvaluatorActor` + `ExecutionPublisherActor` |
| Event consumption    | Store: `ExecutionConsumerActor` |
| Materialization      | Store: `ExecutionProjectionActor` (sole writer) |
| Query serving        | Store: `QueryResponderActor` → `ExecutionKVStore.Get()` |
| HTTP routing         | Gateway: `ExecutionWebHandler` |

## Limitations

- No history bucket exists — only latest state is queryable.
- No cross-family aggregation (each execution type is independent).
- Projection rebuilds require stream replay (no snapshot mechanism yet).
