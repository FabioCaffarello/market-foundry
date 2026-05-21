# Per-Segment Health and Operational Readiness Signals

> S429 — Per-segment health checks and minimal operational readiness signals for the unified multi-segment runtime.

## Context

After S428 (fee normalization), the unified runtime supports Spot and Futures segments with full lifecycle coverage. However, all health and readiness signals were deployment-wide — there was no way to distinguish whether Spot or Futures was healthy, active, or degraded independently.

S429 adds minimal, per-segment health awareness without inflating into an observability platform.

## Design Principles

1. **Minimal useful signals** — expose the smallest set of per-segment data that enables an operator to answer "is Spot healthy? is Futures healthy?" without opening dashboards or grep'ing logs.
2. **No new infrastructure** — reuse existing `Tracker` and `HealthServer` primitives; no OTEL, no Jaeger, no new HTTP servers.
3. **Segment-prefixed counters** — the existing `Tracker.Counter(name)` mechanism is extended by convention: `<segment>:processed`, `<segment>:filled`, `<segment>:rejected`, `<segment>:errors`.
4. **Backward compatible** — existing global counters (`processed`, `filled`, `rejected`) remain unchanged. Per-segment counters are additive.

## Architecture

### SegmentHealthRegistry

New type in `internal/shared/healthz/segment_health.go`:

```
SegmentHealthRegistry
  ├── Register(descriptor, tracker)
  ├── Status() → []SegmentStatus
  ├── SegmentPhase(name) → string
  └── IsHealthy() → bool
```

Each registered segment produces a `SegmentStatus`:

| Field     | Type   | Description |
|-----------|--------|-------------|
| segment   | string | "spot" or "futures" |
| enabled   | bool   | Whether the segment is configured as enabled |
| adapter   | string | Adapter type name (e.g., "binance_spot_testnet") |
| phase     | string | "disabled", "ready", "active", "degraded" |
| processed | int64  | Count of intents processed for this segment |
| filled    | int64  | Count of fills for this segment |
| rejected  | int64  | Count of rejections for this segment |
| errors    | int64  | Count of errors for this segment |

### Phase Computation

Per-segment phase is derived from the segment's counter state:

| Condition | Phase |
|-----------|-------|
| Segment not enabled | `disabled` |
| No tracker or no counters | `ready` |
| Only errors, no processed | `degraded` |
| Processed > 0 | `active` |

### HTTP Endpoint Integration

The `HealthServer` accepts a `WithSegments(registry)` option. When set:

- **`/statusz`** — includes a `"segments"` array in the JSON response alongside existing trackers
- **`/diagz`** — includes a `"segments"` array in the diagnostic dump

Existing response shapes are preserved — the `segments` field is additive.

### Counter Wiring

`VenueAdapterActor.onIntent()` now increments segment-prefixed counters alongside existing global counters:

```
Intent received (source=binances):
  processed       +1   (existing)
  processed:BTCUSDT +1 (existing)
  spot:processed   +1  (S429)

Fill published:
  filled          +1   (existing)
  filled:BTCUSDT  +1   (existing)
  spot:filled     +1   (S429)

Rejection published:
  rejected        +1   (existing)
  rejected:BTCUSDT +1  (existing)
  spot:rejected   +1   (S429)

Submit error:
  (error count)   +1   (existing)
  spot:errors     +1   (S429)
```

### Boot Wiring

In `cmd/execute/run.go`, the segment registry is built from config at startup:

```go
segmentRegistry := healthz.NewSegmentHealthRegistry()
for _, seg := range enabledSegs {
    segmentRegistry.Register(healthz.SegmentDescriptor{
        Name:    string(seg),
        Enabled: true,
        Adapter: string(config.Venue.AdapterForSegment(seg)),
    }, trackers["venue-adapter"])
}
```

The registry is passed to the health server via `healthz.WithSegments(segmentRegistry)`.

## Example /statusz Response

```json
{
  "status": "ok",
  "phase": "active",
  "runtime": "execute",
  "uptime": "5m30s",
  "trackers": [...],
  "segments": [
    {
      "segment": "futures",
      "enabled": true,
      "adapter": "binance_futures_testnet",
      "phase": "active",
      "processed": 42,
      "filled": 38,
      "rejected": 2,
      "errors": 2
    },
    {
      "segment": "spot",
      "enabled": true,
      "adapter": "binance_spot_testnet",
      "phase": "active",
      "processed": 67,
      "filled": 65,
      "rejected": 1,
      "errors": 1
    }
  ]
}
```

## File Map

| File | Change |
|------|--------|
| `internal/shared/healthz/segment_health.go` | New: SegmentHealthRegistry, SegmentStatus, SegmentDescriptor |
| `internal/shared/healthz/healthz.go` | Modified: WithSegments option, segments in /statusz and /diagz |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Modified: segment-prefixed counter increments |
| `cmd/execute/run.go` | Modified: segment registry construction and health server wiring |
| `internal/shared/healthz/segment_health_test.go` | New: 9 tests for registry behavior |
| `internal/actors/scopes/execute/s429_segment_health_test.go` | New: 2 tests for segment prefix mapping |
