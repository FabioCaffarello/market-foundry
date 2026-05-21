# Unified Runtime Read-Path Auditability and Segment Isolation Under Real Spot Responses

Stage: S407
Status: Complete
Date: 2026-03-23

## Context

After S405 (acceptance/fill) and S406 (rejection/partial fill), the Spot segment has real venue responses flowing through the execution pipeline. S387 introduced the rejection projection and composite status query. This document consolidates the read-path auditability and segment isolation architecture when those real responses are consumed on a unified runtime (Spot + Futures coexisting).

## Read-Path Architecture

### KV Buckets and Query Routes

| Bucket | Contents | Dedicated Route | Composite Route |
|--------|----------|-----------------|-----------------|
| `EXECUTION_PAPER_ORDER_LATEST` | Accepted intents (derive output) | `execution.query.paper_order.latest` | `execution.query.status.latest` |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Filled / partially filled intents | `execution.query.venue_market_order.latest` | `execution.query.status.latest` |
| `EXECUTION_VENUE_REJECTION_LATEST` | Rejected intents + embedded audit metadata | `execution.query.venue_rejection.latest` (S407) | `execution.query.status.latest` |

### S407 Changes

1. **Dedicated rejection query route** (`execution.query.venue_rejection.latest`): Previously rejections were only queryable via the composite status endpoint. This route returns `ExecutionRejectionReply` with both the intent and the `RejectionDetail` (code, reason, venue details).

2. **Rejection audit metadata embedding**: The `RejectionProjectionActor` now embeds `rejection_code`, `rejection_reason`, and `venue_detail.*` keys into the intent's `Metadata` map before KV storage. This preserves audit detail through the KV round-trip without changing the KV schema.

3. **Composite status enrichment**: `ExecutionStatusReply` now includes `RejectionDetail` when a rejection is present, extracted from the embedded metadata on read.

## Audit Metadata Flow

### Rejection Path (S407 Consolidated)

```
Venue HTTP error (400, -2010)
  -> BinanceSpotTestnetAdapter.handleErrorResponse()
     -> Problem { Code, Message, Details: {venue_http_status, venue_error_code} }
  -> VenueAdapterActor.publishRejection()
     -> VenueOrderRejectedEvent { RejectionCode, RejectionReason, VenueDetails, ExecutionIntent }
  -> RejectionProjectionActor.onRejection()
     -> Embeds rejection_code, rejection_reason, venue_detail.* into intent.Metadata
     -> Stores to EXECUTION_VENUE_REJECTION_LATEST KV
  -> QueryResponderActor.handleExecutionVenueRejectionLatest()
     -> Reads intent from KV
     -> Extracts RejectionDetail from embedded metadata
     -> Returns ExecutionRejectionReply { ExecutionIntent, Detail }
```

### Fill Path

```
Venue HTTP 200 FILLED
  -> BinanceSpotTestnetAdapter.parseOrderResponse()
     -> VenueOrderReceipt { Status=filled, Intent with Fills[]{Price, Qty, Fee, Simulated=false} }
  -> VenueAdapterActor.publishFill()
     -> VenueOrderFilledEvent { ExecutionIntent, VenueOrderID }
  -> FillProjectionActor.onFill()
     -> Validates, checks RC-1 correlation, RC-2 quantity boundary
     -> Stores to EXECUTION_VENUE_MARKET_ORDER_LATEST KV
  -> QueryResponderActor.handleExecutionVenueMarketOrderLatest()
     -> Returns ExecutionLatestReply { ExecutionIntent }
```

### Correlation Chain

All lifecycle states preserve:
- `CorrelationID`: Set by derive, carried through entire pipeline
- `CausationID`: Set to incoming event's ID at each stage
- `Source`: Segment identity (e.g., "binances" for Spot)

These fields are part of `ExecutionIntent` and survive KV storage.

## Segment Isolation

### Write-Time Isolation

The execute binary's intake consumer uses `ExecuteVenueIntakeConsumerForSegments(sources)` to subscribe only to NATS subjects matching registered segment sources. This prevents the venue adapter from receiving intents for unregistered segments.

### Read-Time Isolation

Segment isolation on the read-path is enforced by **partition key structure**: `{source}.{symbol}.{timeframe}`.

- Spot intent: `binances.btcusdt.60`
- Futures intent: `binancef.btcusdt.60`

These are distinct KV keys. A query for `source=binances` will never return a Futures result, and vice-versa. No additional filtering is needed.

### Unified Runtime Coexistence

On a unified runtime where both Spot and Futures adapters are registered:
- Both segments write to the **same** KV buckets with **different** source prefixes
- Reads are inherently isolated by partition key
- The `SegmentRouter` dispatches to the correct adapter based on `SegmentForSource(intent.Source)`
- No cross-segment contamination is possible at either write or read time

## Lifecycle State Coverage

| State | Source | KV Bucket | Queryable Via | Audit Detail |
|-------|--------|-----------|---------------|--------------|
| Accepted | Derive | PAPER_ORDER_LATEST | Dedicated + composite | CorrelationID, Source |
| Filled | Execute (Spot venue) | VENUE_MARKET_ORDER_LATEST | Dedicated + composite | Fills[], Simulated=false, VenueOrderID |
| Partially Filled | Execute (Spot venue) | VENUE_MARKET_ORDER_LATEST | Dedicated + composite | Fills[], FilledQuantity < Quantity |
| Rejected | Execute (Spot venue) | VENUE_REJECTION_LATEST | Dedicated (S407) + composite | RejectionCode, RejectionReason, VenueDetails |

## Limitations

1. **Latest-only semantics**: All KV buckets store only the latest intent per partition key. Historical lifecycle progression is not queryable from KV; it requires the JetStream event streams or ClickHouse (writer path).

2. **Rejection detail encoding**: Venue details are serialized as strings in the metadata map (`fmt.Sprintf("%v", v)`). Numeric venue codes become string representations. This is sufficient for audit but not for programmatic comparison without parsing.

3. **Best-effort rejection store**: If the rejection KV bucket is unavailable at startup, the dedicated rejection route is not registered and the composite status omits rejection data. The store continues to function in a degraded mode.

4. **No segment-scoped list query**: There is no way to list all Spot events across symbols/timeframes. Queries are per-partition-key only. Broad segment auditing requires the writer path (ClickHouse).
