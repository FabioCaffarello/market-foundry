# Unified Runtime Read-Path Auditability and Segment Parity Under Real Futures Responses

Stage: S418 (consolidated in S424)
Status: Complete
Date: 2026-03-23 (S424 consolidation: 2026-03-23)

## Context

After S416 (acceptance/fill) and S417 (rejection/partial fill), the Futures segment has real venue responses flowing through the execution pipeline on the unified runtime. S407 proved Spot read-path auditability. This document consolidates the Futures read-path: how real Futures responses are queryable, how audit metadata survives the KV round-trip, and where parity with Spot holds or diverges.

**S424 consolidation**: After S422 (real acceptance/fill proof with ValidTransition assertions) and S423 (real rejection/partial fill proof with 6 error scenarios), S424 consolidates the read-path by proving that the exact response shapes produced by real Futures venue interactions flow correctly through the query surfaces and maintain full parity with the Spot segment.

## Read-Path Architecture (Unified Runtime)

### KV Buckets and Query Routes (Segment-Transparent)

| Bucket | Contents | Dedicated Route | Composite Route |
|--------|----------|-----------------|-----------------|
| `EXECUTION_PAPER_ORDER_LATEST` | Accepted intents (derive output) | `execution.query.paper_order.latest` | `execution.query.status.latest` |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Filled / partially filled intents | `execution.query.venue_market_order.latest` | `execution.query.status.latest` |
| `EXECUTION_VENUE_REJECTION_LATEST` | Rejected intents + embedded audit metadata | `execution.query.venue_rejection.latest` | `execution.query.status.latest` |

All three buckets serve both Spot and Futures via partition key isolation. No segment-specific routes or buckets exist.

### S418 Consolidation

1. **Futures rejection audit metadata**: The `RejectionProjectionActor` embeds `rejection_code`, `rejection_reason`, and `venue_detail.*` keys into the Futures intent's `Metadata` map before KV storage. This is the same mechanism proven for Spot in S407.

2. **Futures fill records**: Real Futures fills carry `Simulated=false`, price from `avgPrice`, fee from `cumQuote` (proxy), and timestamp from venue `updateTime`. These are structurally identical `FillRecord` values to Spot, with different semantic sources.

3. **Composite status enrichment**: `ExecutionStatusReply` includes `RejectionDetail` for Futures rejections, extracted from embedded metadata on read. `DeriveEffectivePropagation` applies the same timestamp-priority logic regardless of segment.

## Audit Metadata Flow

### Futures Rejection Path

```
Venue HTTP error (400, -2019 margin insufficient)
  -> BinanceFuturesTestnetAdapter.handleErrorResponse()
     -> Problem { Code, Message, Details: {venue_http_status: 400, venue_error_code: -2019} }
  -> VenueAdapterActor.publishRejection()
     -> VenueOrderRejectedEvent { RejectionCode, RejectionReason, VenueDetails, ExecutionIntent }
  -> RejectionProjectionActor.onRejection()
     -> Embeds rejection_code, rejection_reason, venue_detail.* into intent.Metadata
     -> Stores to EXECUTION_VENUE_REJECTION_LATEST KV (key: binancef.btcusdt.60)
  -> QueryResponderActor.handleExecutionVenueRejectionLatest()
     -> Reads intent from KV
     -> Extracts RejectionDetail from embedded metadata
     -> Returns ExecutionRejectionReply { ExecutionIntent, Detail }
```

### Futures Fill Path

```
Venue HTTP 200 FILLED (avgPrice, cumQuote, executedQty, updateTime)
  -> BinanceFuturesTestnetAdapter.parseOrderResponse()
     -> VenueOrderReceipt { Status=filled, Intent with Fills[]{Price=avgPrice, Fee=cumQuote, Simulated=false} }
  -> VenueAdapterActor.publishFill()
     -> VenueOrderFilledEvent { ExecutionIntent, VenueOrderID }
  -> FillProjectionActor.onFill()
     -> Stores to EXECUTION_VENUE_MARKET_ORDER_LATEST KV (key: binancef.btcusdt.60)
  -> QueryResponderActor.handleExecutionVenueMarketOrderLatest()
     -> Returns ExecutionLatestReply { ExecutionIntent }
```

### Correlation Chain

All Futures lifecycle states preserve:
- `CorrelationID`: Set by derive, carried through entire pipeline
- `CausationID`: Set to incoming event's ID at each stage
- `Source`: `binancef` (Futures segment identity)

## Segment Parity Assessment

### Where Parity Holds

| Capability | Spot (S407) | Futures (S418) | Parity |
|---|---|---|---|
| Rejection audit metadata embedding | Proven | Proven | Full |
| Rejection metadata KV round-trip | Proven | Proven | Full |
| RejectionDetail extraction from metadata | Proven | Proven | Full |
| Composite status propagation derivation | Proven | Proven | Full |
| Partition key segment isolation | Proven | Proven | Full |
| Fill record Simulated=false | Proven | Proven | Full |
| Correlation chain preservation | Proven | Proven | Full |
| LifecycleEntry field population | Proven | Proven | Full |
| LifecycleListReply mixed-segment aggregation | Proven | Proven | Full |
| Unified runtime coexistence | Spot-only doesn't contact Futures | Futures-only doesn't contact Spot | Full |

### Where Segments Diverge (Expected)

| Aspect | Spot | Futures | Impact on Read-Path |
|---|---|---|---|
| Source prefix | `binances` | `binancef` | Partition keys are distinct |
| Fill price source | `fills[].price` | `avgPrice` | Same `FillRecord.Price` field |
| Fee source | `fills[].commission` | `cumQuote` | Same `FillRecord.Fee` field |
| Timestamp source | `transactTime` | `updateTime` | Same `FillRecord.Timestamp` field |
| Rejection codes | `-2010` (balance) | `-2019` (margin) | Same `RejectionDetail` structure |

These divergences are venue-specific and do not affect the read-path architecture. The same contracts, query routes, and KV buckets serve both segments transparently.

## Lifecycle State Coverage (Futures)

| State | Source | KV Bucket | Queryable Via | Audit Detail |
|-------|--------|-----------|---------------|--------------|
| Accepted | Derive | PAPER_ORDER_LATEST | Dedicated + composite | CorrelationID, Source=binancef |
| Filled | Execute (Futures venue) | VENUE_MARKET_ORDER_LATEST | Dedicated + composite | Fills[], Simulated=false, VenueOrderID |
| Partially Filled | Execute (Futures venue) | VENUE_MARKET_ORDER_LATEST | Dedicated + composite | Fills[], FilledQuantity < Quantity |
| Rejected | Execute (Futures venue) | VENUE_REJECTION_LATEST | Dedicated + composite | RejectionCode, RejectionReason, VenueDetails |

## Test Evidence

### Application-Level (s418_futures_read_path_audit_test.go)

| Test | What It Proves |
|------|----------------|
| `TestS418_RejectionDetail_FuturesExtractFromMetadata` | Rejection audit metadata extractable from Futures intent |
| `TestS418_RejectionDetail_FuturesNilWhenFilled` | No false-positive rejection detail for filled intents |
| `TestS418_Propagation_FuturesRejectionNewerThanFill` | Timestamp-priority propagation for Futures |
| `TestS418_Propagation_FuturesFillNewerThanRejection` | Reverse timestamp priority |
| `TestS418_Propagation_FuturesPartiallyFilled` | Partial fill propagation for Futures |
| `TestS418_Propagation_FuturesIntentOnly` | Intent-only propagation for Futures |
| `TestS418_Propagation_FuturesNone` | None propagation when no surfaces exist |
| `TestS418_PartitionKey_FuturesSegmentIsolation` | Spot and Futures keys are distinct |
| `TestS418_PartitionKey_FuturesRejectionIsolated` | Rejection keys are segment-isolated |
| `TestS418_CorrelationChain_FuturesRejectedIntent` | Correlation survives rejection metadata |
| `TestS418_CorrelationChain_FuturesFilledIntent` | Correlation survives fill path |
| `TestS418_CorrelationChain_FuturesPartialFillIntent` | Correlation survives partial fill |
| `TestS418_LifecycleEntry_FuturesFieldPopulation` | LifecycleEntry correct for Futures fill |
| `TestS418_LifecycleEntry_FuturesRejection` | LifecycleEntry correct for Futures rejection |
| `TestS418_SegmentParity_PropagationSymmetry` | Propagation identical for Spot and Futures |
| `TestS418_SegmentParity_RejectionDetailExtraction` | Detail extraction identical across segments |
| `TestS418_SegmentParity_FillRecordFormat` | Fill records structurally identical |
| `TestS418_LifecycleList_MixedSegmentAggregation` | Both segments aggregate correctly |

### Actor-Level (s418_futures_read_path_audit_test.go in execute scope)

| Test | What It Proves |
|------|----------------|
| `TestS418_RejectionAuditTrail_FuturesVenueDetails` | Futures rejection carries full venue audit trail; Spot not contacted |
| `TestS418_RejectionMetadataEmbedding_FuturesRoundTrip` | Futures rejection metadata survives JSON round-trip |
| `TestS418_FillReadPath_FuturesRealFillCarriesSegmentAndAudit` | Futures fill carries segment identity and correlation |
| `TestS418_UnifiedRuntime_FuturesFillDoesNotContactSpot` | Segment isolation on unified runtime |

## S424 Consolidation: Real Venue Evidence Bridge

S424 adds a consolidation test suite (`s424_futures_read_path_consolidation_test.go`) that bridges S422/S423 write-path evidence with read-path extraction:

### New Test Evidence (S424)

| Test | What It Proves |
|------|----------------|
| `TestS424_RejectionDetail_RealFuturesMarginInsufficient` | Read-path extraction using exact S423 margin rejection metadata |
| `TestS424_RejectionDetail_AllFuturesRejectionScenarios` | All 6 S423 error scenarios produce extractable audit detail |
| `TestS424_CompositeStatus_FuturesFilledWithIntent` | Composite status with S422 fill data (avgPrice, cumQuote) |
| `TestS424_CompositeStatus_FuturesRejectedWithAuditDetail` | Composite status carries RejectionDetail for S423 rejections |
| `TestS424_CompositeStatus_FuturesPartialFill` | Composite status for structural partial fill |
| `TestS424_CompositeStatus_FuturesMixedFillAndRejection_TimestampPriority` | Timestamp-priority propagation under mixed real outcomes |
| `TestS424_CorrelationChain_AllFuturesLifecycleStates` | Correlation chain across all 4 lifecycle states |
| `TestS424_CorrelationChain_RejectionMetadataRoundTrip` | JSON round-trip preserves S423 rejection audit metadata |
| `TestS424_SegmentParity_PropagationIdentical` | Propagation identical for Spot and Futures (4 scenarios) |
| `TestS424_SegmentParity_RejectionDetailStructure` | Same extraction contract, expected venue-specific divergences |
| `TestS424_SegmentParity_FillRecordStructuralEquivalence` | Fill records structurally identical across segments |
| `TestS424_SegmentParity_PartitionKeyIsolation` | Partition keys prevent cross-segment reads |
| `TestS424_LifecycleList_ConsolidatedMixedSegments` | Mixed Spot/Futures lifecycle list with different propagations |
| `TestS424_FeeSemantics_FuturesCumQuoteAuditTrail` | cumQuote fee proxy preserved and distinguishable by source |

### Consolidated Evidence Chain

```
S422 (write-path: acceptance/fill)
  + S423 (write-path: rejection/partial fill)
  + S418 (read-path: query surfaces, projection actors, KV round-trip)
  = S424 (consolidated proof: real venue shapes -> query extraction -> parity)
```

## Limitations

1. **Latest-only semantics**: KV buckets store only the latest intent per partition key. Historical lifecycle progression requires JetStream streams or ClickHouse.

2. **Rejection detail string encoding**: Numeric venue codes are stored as strings in the metadata map (`venue_http_status: "400"`, `venue_error_code: "-2019"`). Consumers must parse for programmatic comparison.

3. **No segment-scoped list query**: Cannot list "all Futures rejections" from KV. Each query targets a specific `{source}.{symbol}.{timeframe}` triple. The `LifecycleList` query (S413) enumerates all partition keys but does not filter by segment.

4. **Best-effort rejection store**: If `EXECUTION_VENUE_REJECTION_LATEST` bucket is unavailable, rejection queries degrade silently.

5. **Partial fill lifecycle gap**: A `partially_filled` intent in KV is a snapshot. No reconciliation loop exists for partial fills. Partial fills are not observed on Futures testnet (market orders fill instantly); structural proof only (S423).

6. **Fee semantics differ by segment**: Spot fee is per-fill commission; Futures fee is cumQuote (total notional). Both are stored in the same `FillRecord.Fee` field. Consumers must interpret based on source. S424 confirms this is an expected venue-level divergence, not an architectural gap.

7. **No explicit segment ACL**: Any caller can query any source prefix. Access control is not enforced at the query layer.

8. **No ClickHouse segment-filtered queries**: Events are written to ClickHouse with `source=binancef`, but no pre-built segment-filtered analytical queries exist. Ad-hoc SQL filtering by `source` column is available.
