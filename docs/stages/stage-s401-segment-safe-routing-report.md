# S401 — Segment-Safe Routing and Leakage Hardening

> Stage report | 2026-03-22 | Phase 42: Unified Segment Runtime Foundation Wave

## Objective

Harden routing and isolation between Spot and Futures segments in the
multi-segment runtime introduced by S400. Reduce or close the cross-segment
intent leakage risk identified in S397, without adding new functional capability.

## What Changed

### NATS Consumer Subject Filtering (`internal/adapters/nats/natsexecution/`)

**ConsumerSpec** (`natskit/types.go`): Added `FilterSubjects []string` field to
support multi-subject consumer filtering. When set, replaces the single
`FilterSubject` in the NATS `ConsumerConfig`.

**Consumer** (`natsexecution/consumer.go`): Updated `Start()` to use
`FilterSubjects` when available, falling back to single `FilterSubject`.

**Registry** (`natsexecution/registry.go`): New factory
`ExecuteVenueIntakeConsumerForSegments(sources)` builds a consumer spec with
segment-scoped filter subjects:
- `[]string{"binances"}` -> subscribes only to `execution.events.paper_order.submitted.binances.>`
- `[]string{"binancef"}` -> subscribes only to `execution.events.paper_order.submitted.binancef.>`
- `[]string{"binances", "binancef"}` -> subscribes to both
- `nil` -> wildcard fallback (legacy compatibility)

### VenueAdapterActor Source Guard (`internal/actors/scopes/execute/`)

**VenueAdapterConfig**: Added `AllowedSources map[string]bool` field.

**VenueAdapterActor.onIntent**: New Gate 0 — source guard. Before kill switch and
staleness checks, validates `intent.Source` is in `AllowedSources`. Rejected
intents are logged and counted (`rejected_source` counter). Defense-in-depth
against the SegmentRouter's own source validation.

### Execute Supervisor Wiring (`internal/actors/scopes/execute/execute_supervisor.go`)

Updated `start()` to:
1. Build `allowedSources` map from `EnabledSegmentSources()`
2. Pass it to `VenueAdapterConfig.AllowedSources`
3. Pass `enabledSources` to `ExecuteVenueIntakeConsumerForSegments()` instead of
   the old wildcard `ExecuteVenueMarketOrderIntakeConsumer()`

### Settings (`internal/shared/settings/schema.go`)

New method `VenueConfig.EnabledSegmentSources() []string`: returns the canonical
source prefixes for all enabled segments. Used by the execute supervisor to
build both consumer filters and the source guard.

## Tests

| File | Count | What |
|------|-------|------|
| `s401_segment_sources_test.go` | 5 | EnabledSegmentSources: both, spot-only, futures-only, legacy, disabled exclusion |
| `s401_segment_consumer_test.go` | 6 | Consumer spec: single source, dual source, empty fallback, durable name, source prefix, isolation |
| `s401_segment_isolation_test.go` | 10 | Isolation invariants: mapping completeness, injectivity, cross-segment rejection, consumer partitioning, source consistency, subject construction |

All 21 new tests pass. Full workspace test suite passes (no regressions).

## Evidence Matrix

| Criterion | Evidence | Result |
|-----------|----------|--------|
| Spot-only consumer excludes futures | `TestS401_SpotOnlyConsumerExcludesFutures` | Pass |
| Futures-only consumer excludes spot | `TestS401_FuturesOnlyConsumerExcludesSpot` | Pass |
| Unified consumer includes both segments | `TestS401_UnifiedConsumerIncludesBothSegments` | Pass |
| Source-segment mapping is bijective | `TestS401_AllKnownSegmentsHaveSourceMapping` | Pass |
| Unknown sources return empty segment | `TestS401_UnknownSourceReturnsEmptySegment` | Pass |
| Spot source never maps to futures | `TestS401_SpotSourceNeverMapToFutures` | Pass |
| Futures source never maps to spot | `TestS401_FuturesSourceNeverMapToSpot` | Pass |
| EnabledSegmentSources matches EnabledSegments | `TestS401_EnabledSegmentSourcesMatchEnabledSegments` | Pass |
| Execute binary builds | `go build cmd/execute/...` | Pass |
| No test regressions | `go test internal/...` | Pass |

## Defense Layers Added

| Layer | Where | What |
|-------|-------|------|
| L1 | NATS subscription | Consumer filter subjects scoped to enabled segment sources |
| L2 | VenueAdapterActor | AllowedSources gate before kill switch and staleness |

These join existing layers: config validation (L0), SegmentRouter dispatch (L3),
producer-side stamping (L4), subject partitioning (L5), composite KV keys (L6).

## Architecture Documents

- `docs/architecture/segment-safe-routing-and-leakage-hardening.md`
- `docs/architecture/cross-segment-isolation-invariants-routing-rules-and-limitations.md`

## Residual Risk

| Risk | Severity | Status |
|------|----------|--------|
| Cross-segment intent leakage (S397 risk) | Low -> Very Low | Hardened with 2 new defense layers |
| Source string spoofing via NATS | Very Low | Requires auth bypass; not addressed (out of scope) |
| New source prefix without mapping update | Low | Fail-closed: SegmentRouter rejects unknown sources |
| Consumer filter change on live deployment | Low | NATS handles via CreateOrUpdateConsumer |

## Scope Compliance

| Guard Rail | Status |
|------------|--------|
| No multi-exchange | Respected — hardcoded Binance mapping only |
| No runtime mesh inflation | Respected — minimal changes to existing structures |
| No leakage masking | Respected — all vectors assessed transparently |
| No adapter redesign | Respected — only added filtering and guards |

## Preparation for S402

The segment isolation is now hardened at subscription, actor, and router levels.
S402 (compose-level unified proof) can proceed with confidence that:
1. Spot-only, Futures-only, and unified configs all filter correctly
2. Cross-segment intent flow is blocked at multiple layers
3. The defense model is documented and tested
