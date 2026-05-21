# Execution Family Separation After Paper Step

> Stage S85 — Defines the architectural boundary between the paper execution family and the venue execution family, documenting the current transitional bridge and the migration path for clean venue-specific routing.

## Context

Stage S80 introduced the execute binary with the deliberate minimal step: reuse the paper_order event flow for venue adapter intake. This decision was correct for proving the topology but created a cross-family coupling that must be explicitly documented and bounded to prevent drift.

## Family Definitions

### Paper Family (`paper_order`)

| Attribute | Value |
|-----------|-------|
| **Owner** | derive binary |
| **Event** | `PaperOrderSubmittedEvent` |
| **Stream** | `EXECUTION_EVENTS` |
| **Subject** | `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}` |
| **KV Bucket** | `EXECUTION_PAPER_ORDER_LATEST` |
| **Authority** | store binary (`ExecutionProjectionActor`) |
| **Consumer (store)** | `store-execution-paper-order` |
| **Semantics** | Simulated intent evaluation — no venue interaction. Intents arrive Final=true with simulated fills. |

### Venue Family (`venue_market_order`)

| Attribute | Value |
|-----------|-------|
| **Owner** | execute binary |
| **Event** | `VenueOrderFilledEvent` |
| **Stream** | `EXECUTION_FILL_EVENTS` |
| **Subject** | `execution.fill.venue_market_order.{source}.{symbol}.{timeframe}` |
| **KV Bucket** | `EXECUTION_VENUE_MARKET_ORDER_LATEST` |
| **Authority** | store binary (`FillProjectionActor`) |
| **Consumer (store)** | `store-execution-venue-market-order-fill` |
| **Semantics** | Venue-submitted order results. Carries VenueOrderID and real (or simulated) fill records. |

## Transitional Bridge

### Current State (Paper Mode)

The execute binary's venue intake consumer (`execute-venue-market-order-intake`) subscribes to **paper_order subjects**:

```
Filter: execution.events.paper_order.submitted.>
Stream: EXECUTION_EVENTS
Event:  PaperOrderSubmittedEvent
```

This works because:
1. Derive only produces `PaperOrderSubmittedEvent` (no venue-specific intent event exists yet).
2. The paper_order event carries a complete `ExecutionIntent` that the venue adapter can process.
3. In paper mode, `PaperVenueAdapter` simulates fills regardless of the event origin.

### Why This Is Acceptable

- The execute binary treats the paper_order event as an **intent signal**, not as a paper-specific instruction.
- The venue adapter applies its own gates (kill switch, staleness) before processing — it does not blindly trust paper fills.
- The fill output is always on the venue family's own stream (`EXECUTION_FILL_EVENTS`) with its own event type (`VenueOrderFilledEvent`).

### Why This Must Not Persist

When real venue execution is introduced:
- Derive must produce a **venue-specific intent event** (e.g., `VenueOrderIntentEvent`) on a venue-specific subject.
- The execute binary's intake consumer must migrate to the venue-specific subject filter.
- Paper and venue intent events must flow through distinct subjects to prevent the execute binary from processing paper-only intents.

## Migration Path

### Step 1: Introduce Venue Intent Event (Future Stage)

```go
// New event type for venue family intake.
const EventVenueOrderIntentSubmitted events.Name = "venue_order_intent_submitted"

type VenueOrderIntentEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"`
}
```

### Step 2: Add Venue Intent Subject

```
Subject: execution.events.venue_market_order.submitted.{source}.{symbol}.{timeframe}
Stream:  EXECUTION_EVENTS (same stream, different subject prefix)
```

### Step 3: Migrate Intake Consumer

```go
// Before (transitional bridge):
Subject: "execution.events.paper_order.submitted.>"

// After (venue-specific):
Subject: "execution.events.venue_market_order.submitted.>"
```

### Step 4: Update Derive

Derive produces both events when both families are enabled:
- `PaperOrderSubmittedEvent` on `execution.events.paper_order.submitted.>`
- `VenueOrderIntentEvent` on `execution.events.venue_market_order.submitted.>`

### Step 5: Remove Bridge

Once the migration is complete, the execute binary no longer subscribes to paper_order subjects. The bridge is fully removed.

## Invariants

1. **Paper family events stay on paper subjects.** No new consumer should subscribe to paper_order subjects for venue-specific processing beyond the documented transitional bridge.
2. **Venue family fill events stay on venue subjects.** The store's fill consumer only reads from `EXECUTION_FILL_EVENTS`.
3. **Cross-family queries are the only shared surface.** `StatusLatest` reads both KV buckets; `ControlGet/ControlSet` manages the global gate.
4. **Each family has its own KV bucket.** No bucket mixing: paper intents in `EXECUTION_PAPER_ORDER_LATEST`, venue fills in `EXECUTION_VENUE_MARKET_ORDER_LATEST`.

## Compatibility

- All existing consumers, subjects, and KV buckets remain unchanged.
- The transitional bridge is documented in code (registry, supervisor, messages) with `TRANSITIONAL BRIDGE` annotations.
- No breaking changes to the paper execution flow.
- No new NATS infrastructure required until venue intent events are introduced.
