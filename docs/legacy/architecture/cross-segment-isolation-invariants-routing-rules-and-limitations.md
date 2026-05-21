# Cross-Segment Isolation Invariants, Routing Rules, and Limitations

> Architecture document | S401 | 2026-03-22

## Invariants

These properties are enforced by code and validated by tests. Violations of
any invariant indicate a regression.

### INV-1: Source-Segment Mapping is Bijective

Every known source prefix maps to exactly one segment, and every segment maps
to exactly one source prefix. The mapping is static and locked:

| Source | Segment |
|--------|---------|
| `binancef` | `futures` |
| `binances` | `spot` |

**Enforced by:** `sourceForSegment` map in `settings/schema.go`
**Tested by:** `TestS401_AllKnownSegmentsHaveSourceMapping`, `TestS401_SourceToSegmentMappingIsInjective`

### INV-2: Unknown Sources are Rejected (Fail-Closed)

Any source not in the mapping returns empty segment, which causes:
1. Consumer filter: never subscribed (if segment-scoped)
2. Source guard: rejected at VenueAdapterActor (if AllowedSources set)
3. SegmentRouter: rejected with `InvalidArgument` Problem

**Tested by:** `TestS401_UnknownSourceReturnsEmptySegment`, `TestSegmentRouterRejectsUnknownSource`

### INV-3: Spot Intent Never Reaches Futures Adapter

A `binances` intent can only route to the spot adapter. The SegmentRouter
maps `binances` -> `spot` -> `adapters[spot]`. There is no code path that
maps `binances` to `futures`.

**Tested by:** `TestS401_SpotSourceNeverMapToFutures`, `TestSegmentRouterMultiSegmentIsolation`

### INV-4: Futures Intent Never Reaches Spot Adapter

Symmetric to INV-3. `binancef` -> `futures` -> `adapters[futures]`.

**Tested by:** `TestS401_FuturesSourceNeverMapToSpot`, `TestSegmentRouterMultiSegmentIsolation`

### INV-5: Consumer Subscribes Only to Enabled Segments

When unified segments are configured, the intake consumer's `FilterSubjects`
contains only subjects matching enabled segment sources.

**Tested by:** `TestS401_SpotOnlyConsumerExcludesFutures`, `TestS401_FuturesOnlyConsumerExcludesSpot`, `TestS401_UnifiedConsumerIncludesBothSegments`

### INV-6: Source Embedded in Every NATS Subject

All execution subjects follow the pattern `{base}.{source}.{symbol}.{timeframe}`.
This ensures every message carries segment identity in its subject path.

**Tested by:** `TestS401_PublishSubjectsContainSourcePrefix`

### INV-7: Adapter-Segment Compatibility Validated at Config Time

A futures adapter cannot be placed on a spot segment slot (and vice versa).
This is validated by `VenueConfig.Validate()` at startup.

**Tested by:** `TestVenueValidateRejectsAdapterSegmentMismatch` (S393)

## Routing Rules

### Rule R1: Segment Resolution

```
source := intent.Source           // e.g. "binancef"
segment := SegmentForSource(source) // e.g. "futures"
adapter := router.adapters[segment] // per-segment adapter
```

If any step yields empty/nil, the intent is rejected with a structured Problem.

### Rule R2: Consumer Subject Construction

```
For each enabled source S:
  filter := "execution.events.paper_order.submitted." + S + ".>"
```

If no segments are configured (legacy mode), the consumer uses the wildcard
`execution.events.paper_order.submitted.>`.

### Rule R3: Publish Subject Construction

```
subject := base + "." + intent.Source + "." + intent.Symbol + "." + intent.Timeframe
```

Source is always taken from the intent — never overridden or defaulted.

## Limitations

### L1: Single-Exchange Architecture

The source-segment mapping is hardcoded for Binance (binancef/binances).
Adding a second exchange (e.g., Bybit) requires extending the mapping.
This is intentionally out of scope — S401 guard rails explicitly forbid
opening multi-exchange.

### L2: Consumer Durable Name is Shared Across Segment Configurations

The durable consumer name `execute-venue-market-order-intake` is the same
regardless of which segments are enabled. Changing the segment configuration
on an existing deployment updates the consumer's filter subjects in place
(via NATS `CreateOrUpdateConsumer`). This is correct but means the consumer
may briefly receive old-filter messages during the transition.

### L3: QueryOrder Does Not Carry Source

`SegmentRouter.QueryOrder` iterates all registered query ports because the
query interface does not carry a source field. This is acceptable because
QueryOrder is only used for post-200 reconciliation (rare path). Adding
source to the query interface would require API changes across the domain.

### L4: Store Receives All Segments

The store binary's consumers use wildcard subscriptions (`>.`) and receive
events from all segments. This is correct — the store materializes all
segments into the same KV buckets using composite keys that include source.
Segment-scoped store consumers are not needed because the store has no
routing logic — it simply persists all events it receives.
