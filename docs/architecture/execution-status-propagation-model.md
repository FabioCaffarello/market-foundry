# Execution Status Propagation Model

> Defines how execution status propagates through the system and how it becomes queryable.

## Propagation Chain

```
derive                    execute                   store
  |                         |                         |
  | PaperOrderSubmittedEvent|                         |
  |--- EXECUTION_EVENTS --->|                         |
  |                         |--- (intake) ----------->|  paper_order projection
  |                         |                         |  -> EXECUTION_PAPER_ORDER_LATEST
  |                         |                         |
  |                         | VenueOrderFilledEvent   |
  |                         |--- EXECUTION_FILL_EVENTS|  fill projection
  |                         |                         |  -> EXECUTION_VENUE_MARKET_ORDER_LATEST
  |                         |                         |
                         gateway                      |
                            |                         |
                            |  execution.query.*      |
                            |<--- request/reply ----->|
                            |                         |
                         HTTP API
```

## Status Lifecycle

An execution intent progresses through the following statuses:

| Status | Surface | Meaning |
|--------|---------|---------|
| `submitted` | Intent | Derive produced a paper order. Pre-venue. |
| `sent` | (transient) | Order sent to venue. Not materialized as latest-only. |
| `accepted` | (transient) | Venue acknowledged receipt. Not materialized as latest-only. |
| `filled` | Result | Venue completed the order. Terminal. |
| `partially_filled` | Result | Venue partially filled. May transition to `filled` or `cancelled`. |
| `rejected` | Result | Venue rejected the order. Terminal. |
| `cancelled` | Result | Order was cancelled. Terminal. |

### What Gets Materialized

Only **final** intents (with `Final == true`) are materialized into KV buckets. Transient states (`sent`, `accepted`) are event-sourced but not projected into the latest read model. This is by design — the read model shows the last known stable state, not in-flight transitions.

## Propagation Status Derivation

The composite status endpoint (`GET /execution/status/latest`) derives an effective `propagation` field:

```
if result exists:
    propagation = result.status    # filled, rejected, cancelled, partially_filled
else if intent exists:
    propagation = intent.status    # submitted (always, since only final intents are projected)
else:
    propagation = "none"           # no execution activity for this partition
```

### Interpreting Propagation

| Propagation | Meaning | Action |
|-------------|---------|--------|
| `none` | No execution data for this source/symbol/timeframe | Normal for unconfigured symbols |
| `submitted` | Intent exists but no fill yet | Expected transient state; if persistent, check execute binary |
| `filled` | Full lifecycle complete | Normal terminal state |
| `partially_filled` | Venue partially filled | Awaiting remaining fills or cancellation |
| `rejected` | Venue rejected the order | Check venue logs for rejection reason |
| `cancelled` | Order was cancelled | Investigate if unexpected |

### Detecting Stale Propagation

When `propagation == "submitted"` and the intent timestamp is older than `DefaultStalenessMaxAge` (120 seconds), the execution pipeline may be stuck. Causes:

1. `execute` binary is down or lagging.
2. Control gate is `halted` (check `gate.status` in the status response).
3. NATS stream consumer is behind.

The status endpoint provides all three data points (intent, result, gate) in one call, enabling this diagnosis without multiple queries.

## Control Gate Effect on Propagation

The control gate (`active` / `halted`) affects the **derive publisher**, not the store or gateway. When halted:

- Derive stops publishing new `PaperOrderSubmittedEvent` messages.
- Existing events in streams continue to be consumed and projected.
- The status endpoint still returns the last known state.
- The gate status is included in the composite response for visibility.

This means halting the gate does **not** immediately change the `propagation` field — it prevents new intents from entering the pipeline.

## Data Flow Guarantees

1. **Monotonicity**: KV stores enforce timestamp-based ordering. A newer intent always overwrites an older one; stale writes are skipped.
2. **Deduplication**: JetStream `MsgId` ensures each event is processed at most once per consumer.
3. **Finality gate**: Only `Final == true` intents reach the projection. Non-final events are dropped silently.
4. **Domain validation**: Each projection validates the intent against domain rules before writing.

## Latency Expectations

Under normal conditions:
- Derive -> store (paper_order): ~50-200ms (stream publish + consumer dispatch + KV write).
- Derive -> execute -> store (venue_market_order): ~100-500ms (additional venue round-trip + fill publish).

These are not SLA-bound numbers — they are operational expectations for paper venue mode.

## Limitations

- **No causal ordering between surfaces**: The intent and result surfaces are independently materialized. There is no guarantee that querying both returns causally consistent data. A result may appear before the corresponding intent is visible (race condition between two independent consumers).
- **No event sequence tracking**: The read model does not expose which event sequence produced the current state.
- **No cross-symbol correlation**: Each partition is independent. There is no global execution state view.
