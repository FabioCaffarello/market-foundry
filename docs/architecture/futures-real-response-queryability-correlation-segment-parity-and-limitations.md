# Futures Real Response Queryability, Correlation, Segment Parity, and Limitations

Stage: S418 (consolidated in S424)
Status: Complete
Date: 2026-03-23 (S424 consolidation: 2026-03-23)

## Purpose

This document maps exactly how each Futures lifecycle state (accepted, filled, rejected, partially_filled) is queryable under real venue responses, what correlation and audit metadata is available, how segment parity with Spot holds, and what gaps remain. It is the Futures counterpart to the S407 Spot queryability document.

**S424 consolidation**: With S422 (real acceptance/fill) and S423 (real rejection/partial fill with 6 error scenarios) now complete, S424 validates that the exact metadata shapes produced by real Futures venue interactions are correctly extractable through all query surfaces and maintain full parity with the Spot segment.

## Queryability Matrix

### Per-State Query Paths

| Lifecycle State | NATS Query Subject | Reply Contract | Partition Key | Fields Available |
|-----------------|-------------------|----------------|---------------|------------------|
| Accepted (paper) | `execution.query.paper_order.latest` | `ExecutionLatestReply` | `binancef.{symbol}.{tf}` | Intent with CorrelationID, CausationID, Source=binancef |
| Filled (venue) | `execution.query.venue_market_order.latest` | `ExecutionLatestReply` | `binancef.{symbol}.{tf}` | Intent with Fills[], FilledQuantity, VenueOrderID, Simulated=false |
| Partially Filled (venue) | `execution.query.venue_market_order.latest` | `ExecutionLatestReply` | `binancef.{symbol}.{tf}` | Intent with Fills[], FilledQuantity < Quantity, Status=partially_filled |
| Rejected (venue) | `execution.query.venue_rejection.latest` | `ExecutionRejectionReply` | `binancef.{symbol}.{tf}` | Intent + RejectionDetail { Code, Reason, VenueDetails } |
| All (composite) | `execution.query.status.latest` | `ExecutionStatusReply` | `binancef.{symbol}.{tf}` | Intent + Result + Rejection + RejectionDetail + Gate + Propagation |
| Lifecycle List | `execution.query.lifecycle.list` | `LifecycleListReply` | All keys | Per-key: IntentStatus, FillStatus, RejectionStatus, Propagation |

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

These rules are segment-transparent. The same `DeriveEffectivePropagation()` function handles Spot and Futures identically.

## Correlation Chain Integrity

### End-to-End Flow (Futures)

```
Derive produces PaperOrderSubmittedEvent:
  Metadata.CorrelationID = <generated at decision point>
  ExecutionIntent.CorrelationID = Metadata.CorrelationID
  ExecutionIntent.Source = "binancef"

Execute receives via intake consumer:
  SegmentRouter dispatches to BinanceFuturesTestnetAdapter
  VenueAdapterActor preserves CorrelationID in outgoing events
  CausationID set to incoming event's Metadata.ID

Store materializes to KV:
  ExecutionIntent carries CorrelationID and CausationID through storage
  Queryable on read without loss
```

### Test Evidence

- `TestS418_CorrelationChain_FuturesRejectedIntent`: CorrelationID survives rejection metadata embedding
- `TestS418_CorrelationChain_FuturesFilledIntent`: CorrelationID survives fill path with Simulated=false
- `TestS418_CorrelationChain_FuturesPartialFillIntent`: CorrelationID survives partial fill path
- `TestS418_RejectionAuditTrail_FuturesVenueDetails`: Metadata.CorrelationID preserved from intent through rejection event

**S424 consolidation evidence:**
- `TestS424_CorrelationChain_AllFuturesLifecycleStates`: Correlation chain across all 4 lifecycle states using real venue data shapes
- `TestS424_CorrelationChain_RejectionMetadataRoundTrip`: JSON round-trip preserves S423 rejection audit metadata (marshal/unmarshal cycle)

## Segment Parity with Spot (S407)

### Parity Matrix

| Query Capability | Spot (S407) | Futures (S418) | Status |
|---|---|---|---|
| Dedicated rejection query with RejectionDetail | Proven | Proven | Parity |
| Composite status with rejection + fill + propagation | Proven | Proven | Parity |
| Rejection metadata KV round-trip | Proven | Proven | Parity |
| Partition key isolation prevents cross-segment read | Proven | Proven | Parity |
| Lifecycle list aggregation across segments | Proven | Proven | Parity |
| Fill record with Simulated=false | Proven | Proven | Parity |
| Unified runtime coexistence | Proven | Proven | Parity |

### Known Divergences (Venue-Specific, Not Architectural)

| Aspect | Spot | Futures | Architectural Impact |
|---|---|---|---|
| Fill price source | `fills[].price` (per-leg) | `avgPrice` (aggregate) | None -- same `FillRecord.Price` field |
| Fee source | `fills[].commission` (per-leg) | `cumQuote` (notional) | None -- same `FillRecord.Fee` field; different semantics |
| Rejection code `-2010` | Insufficient balance | Not applicable | None -- same `RejectionDetail.VenueDetails` structure |
| Rejection code `-2019` | Not applicable | Insufficient margin | None -- same `RejectionDetail.VenueDetails` structure |
| Timestamp source | `transactTime` | `updateTime` | None -- same `FillRecord.Timestamp` field |
| Response format | `fills[]` array present | `avgPrice`/`cumQuote` only | None -- adapter normalizes to `FillRecord` |

## Rejection Audit Detail (Futures)

### Metadata Embedding

The `RejectionProjectionActor` embeds the following keys for Futures rejections:

| Metadata Key | Source | Example Value |
|-------------|--------|---------------|
| `rejection_code` | `VenueOrderRejectedEvent.RejectionCode` | `VAL_INVALID_ARGUMENT` |
| `rejection_reason` | `VenueOrderRejectedEvent.RejectionReason` | `Margin is insufficient.` |
| `venue_detail.venue_http_status` | `VenueOrderRejectedEvent.VenueDetails` | `400` |
| `venue_detail.venue_error_code` | `VenueOrderRejectedEvent.VenueDetails` | `-2019` |

### Futures-Specific Rejection Codes (S417)

| HTTP | Venue Code | Message | Classification |
|------|-----------|---------|----------------|
| 400 | -2019 | Margin is insufficient | InvalidArgument, non-retryable |
| 400 | -2010 | Insufficient balance | InvalidArgument, non-retryable |
| 400 | -1013 | LOT_SIZE violation | InvalidArgument, non-retryable |
| 401 | -2015 | Auth failure | InvalidArgument, non-retryable |
| 429 | -- | Rate limit | Unavailable, retryable |
| 400 | -1001 | Venue internal | Unavailable, retryable (override) |
| 400 | -1015 | Order rate limit | Unavailable, retryable (override) |
| 503 | -- | Server error | Unavailable, retryable |
| 200 | -- | status=REJECTED | StatusRejected (response parsing) |
| 200 | -- | status=EXPIRED | StatusRejected (response parsing) |

All rejection codes produce the same `RejectionDetail` structure on the read-path, regardless of the original HTTP response pattern.

## Segment Isolation Evidence

### Partition Key Structure

| Segment | Source Prefix | Example Key |
|---------|--------------|-------------|
| Spot | `binances` | `binances.btcusdt.60` |
| Futures | `binancef` | `binancef.btcusdt.60` |

A KV Get for `binancef.btcusdt.60` is physically incapable of returning data written with key `binances.btcusdt.60`.

### Write-Side Isolation

`ExecuteVenueIntakeConsumerForSegments(sources)` creates subject filters:
- Futures-only: subscribes to `execution.events.paper_order.submitted.binancef.>`
- Unified: subscribes to both `...binances.>` and `...binancef.>`

The `SegmentRouter.SubmitOrder()` routes by `SegmentForSource(intent.Source)`:
- `binancef` -> `MarketSegmentFutures` -> `BinanceFuturesTestnetAdapter`
- `binances` -> `MarketSegmentSpot` -> `BinanceSpotTestnetAdapter`

Fail-closed: unknown source returns Problem without contacting any venue.

### Test Evidence

- `TestS418_PartitionKey_FuturesSegmentIsolation`: Spot and Futures keys are distinct
- `TestS418_PartitionKey_FuturesRejectionIsolated`: Futures and Spot rejection keys differ
- `TestS418_UnifiedRuntime_FuturesFillDoesNotContactSpot`: Spot adapter not called for Futures fill
- `TestS418_RejectionAuditTrail_FuturesVenueDetails`: Spot adapter not called for Futures rejection

**S424 consolidation evidence:**
- `TestS424_SegmentParity_PartitionKeyIsolation`: Same symbol/timeframe produces distinct keys across segments
- `TestS424_SegmentParity_PropagationIdentical`: Propagation logic identical for 4 scenarios (fill, rejection, mixed newer/older)
- `TestS424_SegmentParity_RejectionDetailStructure`: Same extraction contract, expected venue-specific error code divergences
- `TestS424_SegmentParity_FillRecordStructuralEquivalence`: Fill records structurally identical despite fee semantic divergence
- `TestS424_LifecycleList_ConsolidatedMixedSegments`: Mixed Spot/Futures lifecycle list with different propagation states
- `TestS424_FeeSemantics_FuturesCumQuoteAuditTrail`: cumQuote fee proxy preserved and distinguishable by source

## Limitations

1. **Latest-only KV semantics**: Each partition key holds only the most recent intent. A fill overwrites a previous partial fill for the same key. Historical progression requires JetStream streams or ClickHouse.

2. **No cross-key queries**: Cannot list "all Futures rejections" or "all fills for binancef" from KV. Each query targets a specific `{source}.{symbol}.{timeframe}` triple. The `LifecycleList` query (S413) enumerates all partition keys but does not filter by segment.

3. **Venue detail string encoding**: Numeric values from `VenueDetails` are stored as strings in the metadata map. A `venue_error_code` of `-2019` becomes the string `"-2019"`. Consumers must parse for programmatic comparison.

4. **Best-effort rejection store**: If `EXECUTION_VENUE_REJECTION_LATEST` bucket is unavailable, rejection queries degrade silently. Other query routes continue to function.

5. **No explicit segment ACL**: Any caller can query any source prefix. Access control is not enforced at the query layer.

6. **Partial fill lifecycle gap**: A `partially_filled` intent in KV is a snapshot. No reconciliation loop exists for partial fills. This is identical to the Spot limitation documented in S407.

7. **Fee semantic divergence**: Spot fee is per-fill commission in a quote asset; Futures fee is cumQuote (total notional value). Both use the same `FillRecord.Fee` field. Consumers interpreting fee values must account for the source segment. This is a venue-level semantic difference, not an architectural gap.

8. **No ClickHouse segment-filtered queries yet**: The writer pipeline persists Futures events to ClickHouse with `source=binancef`, but no pre-built segment-filtered analytical queries exist. Ad-hoc SQL filtering by `source` column is available.
