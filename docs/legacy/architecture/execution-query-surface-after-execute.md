# Execution Query Surface After Execute

> Canonical reference for the execution query surface as it stands after the `execute` phase.
> Authority: `store` binary. All reads derive from materialized NATS KV projections.

## Overview

After S82, the execution query surface exposes three distinct layers:

| Surface | Endpoint | Semantics | KV Bucket |
|---------|----------|-----------|-----------|
| **Intent** | `GET /execution/paper_order/latest` | Latest finalized paper order from derive | `EXECUTION_PAPER_ORDER_LATEST` |
| **Result** | `GET /execution/venue_market_order/latest` | Latest finalized venue fill from execute | `EXECUTION_VENUE_MARKET_ORDER_LATEST` |
| **Status** | `GET /execution/status/latest` | Composite: intent + result + gate + propagation | Reads from both buckets + control |
| **Control** | `GET /execution/control` | Current gate state (active/halted) | `EXECUTION_CONTROL` |
| **Control** | `PUT /execution/control` | Update gate state | `EXECUTION_CONTROL` |

## Intent Surface

**Endpoint:** `GET /execution/paper_order/latest?source=...&symbol=...&timeframe=...`

Returns the latest finalized `ExecutionIntent` produced by `derive` after risk evaluation. This represents the system's *decision to execute* — not the execution itself.

**Semantics:**
- Type is always `paper_order`.
- Status is `submitted` (finalized intent, pre-venue).
- Side reflects the risk-derived direction (`buy`, `sell`, `none`).
- `Final == true` is enforced by the projection gate.
- Null response means no intent has been materialized for this partition.

**Authority chain:** derive publisher -> EXECUTION_EVENTS stream -> store consumer -> paper order projection -> KV bucket -> query responder.

## Result Surface

**Endpoint:** `GET /execution/venue_market_order/latest?source=...&symbol=...&timeframe=...`

Returns the latest finalized `ExecutionIntent` after venue execution (paper or real). This represents the *outcome* of execution — the fill.

**Semantics:**
- Type is always `venue_market_order`.
- Status reflects lifecycle completion (`filled`, `rejected`, `cancelled`, `partially_filled`).
- `Fills` array contains fill records (price, quantity, fee, simulated flag).
- `Final == true` is enforced by the projection gate.
- Null response means no fill has been materialized for this partition.

**Authority chain:** execute publisher -> EXECUTION_FILL_EVENTS stream -> store fill consumer -> fill projection -> KV bucket -> query responder.

## Status Surface (Composite)

**Endpoint:** `GET /execution/status/latest?source=...&symbol=...&timeframe=...`

Returns a composite view that unifies intent, result, and control gate into a single response. This is the primary operational endpoint for understanding execution state propagation.

**Response contract:**
```json
{
  "intent": { ... },
  "result": { ... },
  "gate": {
    "status": "active",
    "reason": "",
    "updated_at": "...",
    "updated_by": ""
  },
  "propagation": "filled"
}
```

**Propagation derivation:**
- If `result` exists -> use `result.status` (most advanced state).
- Else if `intent` exists -> use `intent.status`.
- Else -> `"none"`.

**Use cases:**
- Operational dashboard showing end-to-end flow per symbol.
- Verifying that intents propagate to fills within expected latency.
- Detecting stuck or stale executions (intent exists but result is null or lagging).

## Control Surface

**Endpoints:**
- `GET /execution/control` — current gate state.
- `PUT /execution/control` — update gate to `active` or `halted`.

**Semantics:**
- Single global gate for all execution families.
- Fail-open: missing gate defaults to `active`.
- Gate enforcement happens in `derive` publisher (pre-publish check), not in store.
- Kill switch for emergency halt of all execution intent publishing.

## What Is Latest-Only

All execution query surfaces are **latest-only**. Each KV bucket stores exactly one entry per partition key (`{source}.{symbol}.{timeframe}`). There is no history, no time-range queries, no sequence tracking.

## What Remains Out of Scope

- **Execution history**: No time-series or windowed queries. Deferred to future phase.
- **Per-fill tracking**: Individual fill events are not independently queryable. The result surface shows the aggregate fill state.
- **VenueOrderID in read model**: The `VenueOrderFilledEvent` carries a `venue_order_id`, but this is event metadata not materialized into the KV read model. Available only via stream replay.
- **Real venue execution**: All current flows are paper venue. Real venue adapters are not yet wired.
- **Cross-symbol aggregation**: No endpoint returns execution state across multiple symbols.

## Invariants

1. **Store is sole authority** for all execution read models. Gateway always queries store via NATS request/reply.
2. **Projection gates** ensure only finalized, valid, monotonically newer intents are materialized.
3. **Partition isolation**: Each symbol's execution state is independent. No cross-contamination.
4. **Graceful degradation**: If store is unavailable, gateway returns 503. If a KV bucket has no entry, null is returned (not an error).
