# Execution Read-Side Authority After Execute

## Purpose

Documents the store's authority boundaries over the execution read-side now that the `execute` binary produces fill events, and clarifies the ownership model between intent projection and fill projection.

## Store Authority Principle

The `store` binary is the sole authority for all read-side projections in market-foundry. No other binary writes to KV buckets that serve query traffic. This principle holds for execution fills:

- `execute` publishes fill events to `EXECUTION_FILL_EVENTS` (write-side, stream)
- `store` consumes those events and materializes them into `EXECUTION_VENUE_MARKET_ORDER_LATEST` (read-side, KV)
- `gateway` queries the KV bucket via the store's `QueryResponderActor` (read-only)

## Ownership Map

| KV Bucket | Sole Writer | Query Subject | Source Binary |
|-----------|------------|---------------|--------------|
| `EXECUTION_PAPER_ORDER_LATEST` | `ExecutionProjectionActor` (store) | `execution.query.paper_order.latest` | derive |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `FillProjectionActor` (store) | `execution.query.venue_market_order.latest` | execute |
| `EXECUTION_CONTROL` | `QueryResponderActor` (store) | `execution.control.{get,set}` | gateway |

## Stream vs KV Separation

```
derive → EXECUTION_EVENTS stream → store (ExecutionProjectionActor) → EXECUTION_PAPER_ORDER_LATEST KV
execute → EXECUTION_FILL_EVENTS stream → store (FillProjectionActor) → EXECUTION_VENUE_MARKET_ORDER_LATEST KV
```

Streams are append-only event logs with time-based retention (72 hours). KV buckets are latest-state projections. The store is the bridge between the two: it consumes from streams and materializes into KV.

## Gateway Responsibilities

The gateway does NOT:
- Write to any execution KV bucket
- Consume from any execution stream
- Hold execution state in memory

The gateway DOES:
- Route HTTP queries to the store via NATS request/reply
- Serve the `ExecutionLatestQuery` for both `paper_order` and `venue_market_order` types
- Serve the `ExecutionControl` get/set commands

The `ExecutionGateway` adapter uses `LatestSpecByType(execType)` to resolve the correct query subject, supporting both execution types transparently.

## Intent vs Fill: Semantic Boundary

| Aspect | Paper Order (Intent) | Venue Market Order (Fill) |
|--------|---------------------|--------------------------|
| Origin | derive evaluator | execute venue adapter |
| Status | typically `submitted` | typically `filled` or `rejected` |
| Fills array | empty | populated with fill records |
| VenueOrderID | not present | present in event (not in KV value) |
| Simulated flag | N/A | `true` in paper mode |

The intent projection captures what derive *decided*. The fill projection captures what execute *produced*. Both store the same `ExecutionIntent` struct, but at different lifecycle stages.

## Consistency Model

- **Eventual**: Fill events may arrive after the intent that triggered them. The two projections are independent and may briefly be out of sync.
- **Monotonic per partition**: Within a single `{source}.{symbol}.{timeframe}`, the KV store enforces timestamp monotonicity. No stale writes overwrite newer state.
- **No cross-bucket transactions**: There is no atomic link between the paper order bucket and the venue market order bucket. Each evolves independently.

## What This Does NOT Cover

- Fill history or journal (intentionally excluded — latest-only semantics)
- Real venue integration (paper mode only in current stage)
- Order management (no partial fill tracking, no cancel/amend workflows)
- Cross-symbol aggregation (each partition is independent)
