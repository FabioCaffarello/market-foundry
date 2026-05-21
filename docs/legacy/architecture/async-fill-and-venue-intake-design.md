# Async Fill Model and Venue Intake Design

> Stage S88 — Formalizes the asynchronous fill model for real venue execution and the transition from the current paper bridge to venue-specific intake subjects.
> Date: 2026-03-19
> Classification: DESIGN — no venue real, no multi-venue.

---

## 1. Purpose

Paper fills are synchronous and instant — the adapter returns a filled receipt in the same call. Real venue fills are asynchronous and incremental — an order may take seconds to minutes to fill, may fill in parts, and may never fill at all.

This document:

1. Defines the minimum async fill model needed for the first real venue adapter.
2. Formalizes the transition from the current paper bridge to venue-specific intake subjects.
3. Specifies the event contracts and actor responsibilities for async fill handling.

---

## 2. Current Synchronous Fill Model (Paper)

### Paper Flow (Single-Call)

```
VenueAdapterActor receives intent
  → PaperVenueAdapter.SubmitOrder(intent)
    → generates venue_order_id
    → transitions: submitted → sent → accepted → filled
    → returns VenueOrderReceipt (status=filled, fills=[{simulated=true}])
  → publishes VenueOrderFilledEvent
  → ACKs consumer message
```

**Key property**: The entire lifecycle (submit → fill) completes within a single actor message handler invocation. No intermediate states are persisted.

### Why This Must Change for Real Venue

| Paper Property | Real Venue Reality |
|---------------|-------------------|
| Fill is instant | Fill may take seconds to minutes |
| Single fill event | Multiple partial fills possible |
| No intermediate states | Must persist `sent`, `accepted` states |
| SubmitOrder returns filled receipt | SubmitOrder returns accepted receipt (fill comes later) |
| No polling needed | Must poll or receive callback for fill status |

---

## 3. Async Fill Model

### 3.1 Two-Phase Execution

Real venue execution splits into two phases:

**Phase 1: Order Submission (synchronous)**
```
VenueAdapterActor receives intent
  → VenuePort.SubmitOrder(intent)
    → returns VenueOrderReceipt (status=accepted, venue_order_id)
  → publishes VenueOrderAcceptedEvent (new event type)
  → ACKs consumer message
```

**Phase 2: Fill Tracking (asynchronous)**
```
FillTrackerActor (new actor in execute binary)
  → polls VenuePort.GetOrderStatus(venue_order_id) periodically
  OR
  → receives WebSocket/callback from venue
  → on fill: publishes VenueOrderFilledEvent
  → on partial fill: publishes VenueOrderPartialFilledEvent
  → on rejection: publishes VenueOrderRejectedEvent
  → on expiry: publishes VenueOrderExpiredEvent
```

### 3.2 New Event Types (Design Only)

```go
// Phase 1: Submission acknowledgement
const EventVenueOrderAccepted events.Name = "venue_order_accepted"

type VenueOrderAcceptedEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"` // status=accepted
    VenueOrderID    string          `json:"venue_order_id"`
}

// Phase 2: Partial fill (may occur 0-N times)
const EventVenueOrderPartialFilled events.Name = "venue_order_partial_filled"

type VenueOrderPartialFilledEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"` // status=partially_filled
    VenueOrderID    string          `json:"venue_order_id"`
    Fill            FillRecord      `json:"fill"`             // this fill increment
    CumulativeFilled string         `json:"cumulative_filled"` // total filled so far
    RemainingQty     string         `json:"remaining_qty"`     // still open
}

// Phase 2: Full fill (terminal)
// VenueOrderFilledEvent already exists — reused as-is

// Phase 2: Rejection (terminal)
const EventVenueOrderRejected events.Name = "venue_order_rejected"

type VenueOrderRejectedEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"` // status=rejected
    VenueOrderID    string          `json:"venue_order_id"`
    Reason          string          `json:"reason"`
}

// Phase 2: Expiry (terminal)
const EventVenueOrderExpired events.Name = "venue_order_expired"

type VenueOrderExpiredEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"` // status=cancelled
    VenueOrderID    string          `json:"venue_order_id"`
    Reason          string          `json:"reason"`
}
```

### 3.3 VenuePort Interface Extension

The current `VenuePort` has only `SubmitOrder`. For async fills, two additional methods are needed (already designed in S75):

```go
type VenuePort interface {
    SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
    GetOrderStatus(ctx context.Context, venueOrderID string) (VenueOrderStatus, *problem.Problem)
    CancelOrder(ctx context.Context, venueOrderID string) (VenueCancelResult, *problem.Problem)
}
```

**Implementation strategy**: `GetOrderStatus` and `CancelOrder` are added to the interface only when the first real venue adapter is introduced. `PaperVenueAdapter` does not need them because fills are synchronous.

### 3.4 Fill Tracker Actor (Future)

```
FillTrackerActor (new in execute binary)
  │
  │ Responsibilities:
  │  1. Maintain in-memory set of active (non-terminal) venue_order_ids
  │  2. Periodically poll VenuePort.GetOrderStatus for each active order
  │  3. On status change: publish appropriate event
  │  4. On terminal status: remove from active set
  │  5. On timeout (configurable): publish expired event
  │
  │ Initialization:
  │  - On startup, recover active orders from EXECUTION_VENUE_MARKET_ORDER_LATEST KV
  │  - Filter: status NOT in {filled, rejected, cancelled}
  │  - Resume polling for these orders
  │
  │ Configuration:
  │  - poll_interval: 1s (default, venue-specific)
  │  - order_timeout: 300s (default, venue-specific)
  │  - max_active_orders: 100 (safety cap)
  │
  │ Health counters:
  │  - active_orders, polls, fill_events, timeouts, errors
```

**Key design decisions**:
- Polling-based, not WebSocket-based (simpler, works with all venues).
- WebSocket can be added later as an optimization for venues that support it.
- The tracker publishes events to the same `EXECUTION_FILL_EVENTS` stream.
- Store projections don't change — they already handle fill events.

---

## 4. Venue Intake Transition

### 4.1 Current Transitional Bridge

The execute binary's intake consumer subscribes to paper_order subjects:

```
Stream:  EXECUTION_EVENTS
Filter:  execution.events.paper_order.submitted.>
Durable: execute-venue-market-order-intake
```

This bridge was acceptable for proving the topology (S80) but conflates paper intent production with venue intake.

### 4.2 Why the Bridge Must Not Persist

| Concern | Risk |
|---------|------|
| Subject collision | If derive produces both paper and venue intents on EXECUTION_EVENTS, the execute consumer receives both |
| Family semantics | Paper events carry simulated fills (Final=true); venue intents should arrive without fills (Final=false) |
| Consumer isolation | A single consumer for both families prevents independent scaling and backpressure |
| Operational clarity | Operators cannot distinguish paper intent delivery from venue intent delivery in consumer stats |

### 4.3 Migration Plan (S85 5-Step, Refined)

**Step 1: Introduce VenueOrderIntentEvent** (no wire change yet)

```go
const EventVenueOrderIntentSubmitted events.Name = "venue_order_intent_submitted"

type VenueOrderIntentEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"` // Final=false, status=submitted
}
```

Key difference from `PaperOrderSubmittedEvent`: the intent arrives **without simulated fills** and with `Final=false`. The execute binary is responsible for lifecycle progression.

**Step 2: Add Venue Intent Subject**

```
Subject: execution.events.venue_market_order.submitted.{source}.{symbol}.{timeframe}
Stream:  EXECUTION_EVENTS (same stream — subject routing provides isolation)
```

**Step 3: Derive Produces Both Events** (when both families enabled)

```
derive (execution_families: ["paper_order", "venue_market_order"]):
  RiskAssessedMessage
    → PaperOrderEvaluatorActor → PaperOrderSubmittedEvent (simulated, Final=true)
    → VenueIntentEvaluatorActor → VenueOrderIntentEvent (raw, Final=false)
```

**Important**: These are two distinct actors with distinct responsibilities. The paper evaluator simulates fills. The venue intent evaluator produces a raw intent for the venue adapter.

**Step 4: Migrate Execute Intake Consumer**

```
Before: Filter: execution.events.paper_order.submitted.>
After:  Filter: execution.events.venue_market_order.submitted.>
```

The durable consumer name changes to reflect the new subject filter.

**Step 5: Remove Bridge**

Once migration is complete, the execute binary no longer subscribes to paper_order subjects. The old durable consumer is deleted.

### 4.4 Backward Compatibility

| Dimension | Impact |
|-----------|--------|
| Store paper projection | Unchanged — still consumes from paper_order subjects |
| Store fill projection | Unchanged — still consumes from venue_market_order fill subjects |
| Gateway queries | Unchanged — reads both KV buckets |
| Derive paper evaluation | Unchanged — still produces paper_order events |
| Smoke tests | Must be extended to validate new subject after migration |
| Drift rules | Must be extended to check new subject and consumer |

---

## 5. Stream Subject Taxonomy (Post-Migration)

```
EXECUTION_EVENTS stream:
  execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}          → store (paper projection)
  execution.events.venue_market_order.submitted.{source}.{symbol}.{timeframe}   → execute (venue intake)

EXECUTION_FILL_EVENTS stream:
  execution.fill.venue_market_order.{source}.{symbol}.{timeframe}               → store (fill projection)
```

### Consumer Mapping (Post-Migration)

| Consumer | Stream | Filter | Binary |
|----------|--------|--------|--------|
| store-execution-paper-order | EXECUTION_EVENTS | paper_order.submitted.> | store |
| execute-venue-intake | EXECUTION_EVENTS | venue_market_order.submitted.> | execute |
| store-execution-venue-market-order-fill | EXECUTION_FILL_EVENTS | venue_market_order.> | store |

---

## 6. Paper vs. Venue Intent Comparison

| Field | Paper Intent (current) | Venue Intent (future) |
|-------|----------------------|----------------------|
| Type | `paper_order` | `venue_market_order` |
| Final | `true` (pre-filled) | `false` (awaiting venue) |
| Status | `filled` | `submitted` |
| Fills | `[{simulated: true, ...}]` | `[]` (empty) |
| FilledQuantity | Copies Quantity | `"0"` |
| VenueOrderID | N/A (in event) | N/A (assigned by execute) |

This difference is critical: paper intents arrive **complete** (the whole lifecycle is done in derive). Venue intents arrive **initial** (lifecycle progression happens in execute).

---

## 7. Timing and Rate Considerations

### Fill Polling Budget

For the first real venue (single symbol, single timeframe):

```
poll_interval: 1s
max concurrent orders: 1 (controlled by staleness guard — new intent supersedes old)
API calls per minute: ≤ 60
```

Most venues allow 1200 requests/minute. A single-symbol, single-timeframe setup uses < 5% of rate budget.

### Backpressure

If venue fills arrive faster than store can project:
- JetStream buffering absorbs bursts (stream retention: 72h, 2GB)
- Store projection operates at KV write speed (< 1ms per write)
- No practical backpressure concern for single-venue, single-symbol

---

## 8. Gaps Closed by This Design

| Gap | Before S88 | After S88 |
|-----|-----------|-----------|
| Async fill model | Not designed | Two-phase (submit + track) with event contracts |
| Venue intake transition | 5-step outline in S85 | Refined with event types, subject taxonomy, consumer mapping |
| Paper vs. venue intent semantics | Implicit | Explicit (Final, Status, Fills differences documented) |
| Fill tracker actor | Not designed | Polling-based design with recovery, timeout, and health counters |
| Rate/timing budget | Not considered | Single-venue budget calculated |

---

## 9. What Remains Deferred

| Item | Reason | Earliest Stage |
|------|--------|---------------|
| VenueOrderIntentEvent implementation | Requires activation gate ceremony | S89+ |
| FillTrackerActor implementation | Requires async venue adapter | S90+ |
| WebSocket fill notification | Optimization, not required for first venue | S91+ |
| Multi-venue rate limiting | Single venue first | S92+ |
| VenuePort.GetOrderStatus implementation | No real venue adapter yet | S90+ |
| VenuePort.CancelOrder implementation | No real venue adapter yet | S90+ |
