# Multi-Segment Runtime Projection: Seeds, Invariants, and Limitations

> S400 architecture document. Details the seed lifecycle, runtime invariants,
> and known limitations of the multi-segment projection model.

## Seed Lifecycle

### Before S400 (Sequential)

```
make seed       -> configctl activates market-data-binancef (overwrites any prior)
make seed-spot  -> configctl activates market-data-binances (overwrites binancef)
```

One active config per scope. Switching segments required re-seeding.

### After S400 (Merged)

```
make seed-unified       -> configctl activates market-data-unified
                           bindings: binancef.btcusdt, binances.btcusdt

make seed-unified-multi -> configctl activates market-data-unified
                           bindings: binancef.btcusdt, binancef.ethusdt,
                                     binances.btcusdt, binances.ethusdt
```

One config carries bindings for all sources. The binding watcher discovers
all of them and activates ExchangeScopeActors for each source.

### Backward Compatibility

The single-source targets (`make seed`, `make seed-spot`) continue to work
unchanged. They create single-source configs that work with both the old
first-segment-only path and the new SegmentRouter path (the router simply
has one registered segment).

## Runtime Invariants

### R1: Source Identity Is Immutable

The `Source` field in an execution intent is set by the ingest layer when the
trade is first observed. It flows through derive, risk, strategy, and into
execute without modification. The SegmentRouter reads it but never rewrites it.

### R2: Segment-Adapter Affinity Is Config-Validated

Each segment maps to exactly one adapter type. The adapter type must match the
segment (e.g., `binance_spot_testnet` can only be on `spot`). This is enforced
at config validation time, before any runtime code runs.

### R3: Source-Segment Mapping Is Static

The mapping from source prefix to market segment is defined in code
(`sourceForSegment` map in `settings/schema.go`). It is not configurable at
runtime. Adding a new source requires a code change and a new adapter.

### R4: DryRunSubmitter Is Outermost

The DryRunSubmitter wraps the SegmentRouter. When active (the default), it
intercepts all intents regardless of source. This ensures that no intent can
bypass dry-run protection by being routed to a specific segment.

### R5: Kill Switch Is Segment-Agnostic

The kill switch (EXECUTION_CONTROL KV) applies to the entire execute binary,
not per-segment. When the kill switch is engaged, all segments are blocked.

### R6: Staleness Guard Is Uniform

The staleness max age is configured once in `venue.staleness_max_age` and
applies to all intents regardless of source. Per-segment staleness is not
supported.

## Evidence Matrix

| Property | Proven By | Status |
|----------|-----------|--------|
| Source→Segment mapping correct | `s400_source_segment_test.go` | Pass |
| Router dispatches futures correctly | `s400_segment_router_test.go` | Pass |
| Router dispatches spot correctly | `s400_segment_router_test.go` | Pass |
| Multi-segment isolation | `s400_segment_router_test.go` | Pass |
| Unknown source rejected | `s400_segment_router_test.go` | Pass |
| Unregistered segment rejected | `s400_segment_router_test.go` | Pass |
| Unified config validates | `s400_source_segment_test.go` | Pass |
| Round-trip source↔segment | `s400_source_segment_test.go` | Pass |
| Both segments have distinct adapters | `s400_multi_segment_test.go` | Pass |
| Execute binary builds with router | `go build ./cmd/execute/` | Pass |

## Limitations

### L1: No Per-Segment Dry-Run

`dry_run` is a global flag. You cannot dry-run Futures while live-submitting
Spot (or vice versa). This is intentional — mixed dry-run/live modes within
a single binary create operational ambiguity.

### L2: No Per-Segment Kill Switch

The kill switch is binary-wide. A segment-specific kill switch would require
per-segment KV keys and gate logic, which is not warranted at this stage.

### L3: No Per-Segment Consumer

Both segments share a single NATS consumer. Intent routing happens in the
SegmentRouter, not at the consumer level. Consumer-level isolation would
reduce blast radius but adds wiring complexity.

### L4: QueryOrder Is Sequential

When the SegmentRouter implements `VenueQueryPort`, it tries each registered
query port sequentially. This is acceptable because post-200 reconciliation
is rare, but it means query latency scales linearly with segment count.

### L5: Seed Merge Replaces, Not Appends

`make seed-unified` creates a new config that replaces the previous active
config. It does not append bindings to an existing config. This matches
configctl's activation model (one active config per scope).

### L6: Source Mapping Is Binance-Only

The `sourceForSegment` map only covers Binance sources. Adding a different
exchange (e.g., Coinbase) would require extending this map, adding new
adapter types, and new segment definitions.

## Next Steps

- **S401-S402:** Leakage hardening and compose proof with dual segments running
  concurrently under the unified config.
- **Per-segment metrics:** Add segment dimension to Prometheus counters in
  VenueAdapterActor for per-segment fill/rejection tracking.
- **Consumer isolation (future):** If blast-radius concerns arise, split the
  NATS consumer per segment with source-filtered subjects.
