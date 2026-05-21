# Spot Real Response Queryability, Correlation, Segment Isolation, and Limitations

Stage: S407
Status: Complete
Date: 2026-03-23

## Purpose

This document maps exactly how each Spot lifecycle state (accepted, filled, rejected, partially_filled) is queryable, what correlation and audit metadata is available, how segment isolation holds, and what gaps remain.

## Queryability Matrix

### Per-State Query Paths

| Lifecycle State | NATS Query Subject | Reply Contract | Partition Key | Fields Available |
|-----------------|-------------------|----------------|---------------|------------------|
| Accepted (paper) | `execution.query.paper_order.latest` | `ExecutionLatestReply` | `binances.{symbol}.{tf}` | Intent with CorrelationID, CausationID, Source, Status=accepted |
| Filled (venue) | `execution.query.venue_market_order.latest` | `ExecutionLatestReply` | `binances.{symbol}.{tf}` | Intent with Fills[], FilledQuantity, VenueOrderID, Simulated=false |
| Partially Filled (venue) | `execution.query.venue_market_order.latest` | `ExecutionLatestReply` | `binances.{symbol}.{tf}` | Intent with Fills[], FilledQuantity < Quantity, Status=partially_filled |
| Rejected (venue) | `execution.query.venue_rejection.latest` | `ExecutionRejectionReply` | `binances.{symbol}.{tf}` | Intent + RejectionDetail { Code, Reason, VenueDetails } |
| All (composite) | `execution.query.status.latest` | `ExecutionStatusReply` | `binances.{symbol}.{tf}` | Intent + Result + Rejection + RejectionDetail + Gate + Propagation |

### Composite Status Propagation Rules

```
if result AND rejection exist:
    newer timestamp wins -> propagation = that status
else if result exists:
    propagation = result.status
else if rejection exists:
    propagation = rejection.status
else if intent exists:
    propagation = intent.status
else:
    propagation = "none"
```

## Correlation Chain Integrity

### End-to-End Flow

```
Derive produces PaperOrderSubmittedEvent:
  Metadata.CorrelationID = <generated at decision point>
  Metadata.CausationID = <risk assessment event ID>
  ExecutionIntent.CorrelationID = Metadata.CorrelationID
  ExecutionIntent.CausationID = Metadata.CausationID

Execute receives via intake consumer:
  VenueAdapterActor preserves CorrelationID in outgoing events
  CausationID set to incoming event's Metadata.ID

Store materializes to KV:
  ExecutionIntent carries CorrelationID and CausationID through storage
  Queryable on read without loss
```

### Verified by Tests

- `TestS407_CorrelationChain_PreservedInRejectedIntent`: CorrelationID and CausationID survive rejection metadata embedding
- `TestS407_RejectionAuditTrail_SpotVenueDetails`: Correlation chain preserved from intent through rejection event construction
- `TestS407_FillReadPath_SpotRealFillCarriesSegmentAndAudit`: Correlation chain preserved through fill path

## Segment Isolation Evidence

### Partition Key Structure

Source is embedded in every partition key: `{source}.{symbol}.{timeframe}`.

| Segment | Source Prefix | Example Key |
|---------|--------------|-------------|
| Spot | `binances` | `binances.btcusdt.60` |
| Futures | `binancef` | `binancef.btcusdt.60` |

A KV Get for `binances.btcusdt.60` is physically incapable of returning data written with key `binancef.btcusdt.60`.

### Write-Side Isolation

`ExecuteVenueIntakeConsumerForSegments(sources)` creates subject filters:
- Spot-only: subscribes to `execution.events.paper_order.submitted.binances.>`
- Unified: subscribes to both `...binances.>` and `...binancef.>`

The `SegmentRouter.SubmitOrder()` routes by `SegmentForSource(intent.Source)`:
- `binances` -> `MarketSegmentSpot` -> `BinanceSpotTestnetAdapter`
- `binancef` -> `MarketSegmentFutures` -> `BinanceFuturesTestnetAdapter`

Fail-closed: unknown source returns Problem without contacting any venue.

### Read-Side Isolation

All queries include `Source` in the request. The KV store uses `fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)` as the lookup key. No wildcard reads exist.

### Test Evidence

- `TestS407_PartitionKey_SegmentIsolation`: Spot and Futures partition keys are distinct
- `TestS407_UnifiedRuntime_SpotFillDoesNotContactFutures`: Futures adapter not called for Spot fill on unified runtime
- `TestS407_RejectionAuditTrail_SpotVenueDetails`: Futures adapter not called for Spot rejection

## Rejection Audit Detail (S407)

### Problem

Before S407, the rejection KV bucket stored only `ExecutionIntent` with `Status=rejected`. The rejection code, reason, and venue-specific details (`venue_http_status`, `venue_error_code`) from the `VenueOrderRejectedEvent` were lost at the KV boundary.

### Solution

The `RejectionProjectionActor` now embeds audit fields into the intent's `Metadata` map before KV storage:

| Metadata Key | Source | Example Value |
|-------------|--------|---------------|
| `rejection_code` | `VenueOrderRejectedEvent.RejectionCode` | `VAL_INVALID_ARGUMENT` |
| `rejection_reason` | `VenueOrderRejectedEvent.RejectionReason` | `Account has insufficient balance...` |
| `venue_detail.venue_http_status` | `VenueOrderRejectedEvent.VenueDetails` | `400` |
| `venue_detail.venue_error_code` | `VenueOrderRejectedEvent.VenueDetails` | `-2010` |

On read, `extractRejectionDetail()` reconstructs `RejectionDetail` from these keys.

### Contracts

```go
type RejectionDetail struct {
    RejectionCode   string         `json:"rejection_code"`
    RejectionReason string         `json:"rejection_reason"`
    VenueDetails    map[string]any `json:"venue_details,omitempty"`
}

type ExecutionRejectionReply struct {
    ExecutionIntent *ExecutionIntent `json:"execution_intent"`
    Detail          *RejectionDetail `json:"detail,omitempty"`
}
```

## Limitations

1. **Latest-only KV semantics**: Each partition key holds only the most recent intent. A fill overwrites a previous partial fill for the same key. Historical progression requires JetStream streams or ClickHouse.

2. **No cross-key queries**: Cannot list "all Spot rejections" or "all fills for binances" from KV. Each query targets a specific `{source}.{symbol}.{timeframe}` triple.

3. **Venue detail string encoding**: Numeric values from `VenueDetails` are stored as strings in the metadata map. A `venue_http_status` of `400` becomes the string `"400"`. Consumers must parse if needed.

4. **Best-effort rejection store**: If `EXECUTION_VENUE_REJECTION_LATEST` bucket is unavailable, the dedicated rejection query route is not registered and composite status omits rejection data. Other query routes continue to function.

5. **No explicit segment ACL**: Any caller can query any source prefix. Access control is not enforced at the query layer. This is appropriate for the current single-operator deployment model.

6. **Partial fill lifecycle gap**: A `partially_filled` intent in KV is a snapshot. The transition to `filled` or `cancelled` depends on subsequent venue updates. There is no built-in polling or reconciliation loop for partial fills in the current architecture.
