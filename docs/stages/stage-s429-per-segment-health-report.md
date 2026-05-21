# Stage S429: Per-Segment Health and Operational Readiness Signals — Report

> Completed: 2026-03-23

## Objective

Add per-segment (Spot/Futures) health checks and minimal operational readiness signals to the unified runtime, enabling operators to distinguish segment-level health without log analysis or external dashboards.

## Scope

- Per-segment health phase computation (disabled/ready/active/degraded)
- Per-segment counters for processed, filled, rejected, and errors
- Integration into existing `/statusz` and `/diagz` HTTP endpoints
- No new infrastructure, no OTEL, no dashboards, no alerting

## What Was Delivered

### 1. SegmentHealthRegistry (`internal/shared/healthz/segment_health.go`)

New type that tracks per-segment health state:
- `Register(descriptor, tracker)` — associates a segment with a Tracker
- `Status()` — returns per-segment health in canonical order (futures, spot)
- `SegmentPhase(name)` — returns phase for a specific segment
- `IsHealthy()` — returns true when no enabled segment is degraded

### 2. Per-Segment Counters in VenueAdapterActor

`venue_adapter_actor.go` now increments segment-prefixed counters:
- `spot:processed`, `spot:filled`, `spot:rejected`, `spot:errors`
- `futures:processed`, `futures:filled`, `futures:rejected`, `futures:errors`

These coexist with existing global and symbol-level counters.

### 3. HealthServer Integration

- `WithSegments(registry)` option added to `HealthServer`
- `/statusz` response includes `"segments"` array when registry is configured
- `/diagz` response includes `"segments"` array when registry is configured
- Existing response shape unchanged — segments field is purely additive

### 4. Boot Wiring in Execute Binary

`cmd/execute/run.go` builds the segment registry from config at startup and passes it to the health server. Each enabled segment is registered with the shared `venue-adapter` tracker.

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/shared/healthz/segment_health.go` | New | SegmentHealthRegistry, SegmentStatus, SegmentDescriptor |
| `internal/shared/healthz/segment_health_test.go` | New | 9 tests for registry behavior |
| `internal/shared/healthz/healthz.go` | Modified | WithSegments option, segments in /statusz and /diagz |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Modified | Segment-prefixed counters, segmentPrefix helper |
| `internal/actors/scopes/execute/s429_segment_health_test.go` | New | 2 tests for segment prefix mapping |
| `cmd/execute/run.go` | Modified | Segment registry construction and health server wiring |
| `docs/architecture/per-segment-health-and-operational-readiness-signals.md` | New | Architecture document |
| `docs/architecture/segment-health-readiness-signals-semantics-coverage-and-limitations.md` | New | Semantics, coverage, and limitations |

## Test Evidence

### segment_health_test.go (9 tests)
- `TestSegmentHealthRegistry_EmptyReturnsNoStatus` — empty registry returns no segments and is healthy
- `TestSegmentHealthRegistry_SingleSegmentReady` — single enabled segment starts in ready phase
- `TestSegmentHealthRegistry_DisabledSegment` — disabled segment reports disabled phase
- `TestSegmentHealthRegistry_ActiveAfterProcessing` — segment transitions to active after counters increment
- `TestSegmentHealthRegistry_DegradedOnErrorsOnly` — errors-only segment is degraded, registry is unhealthy
- `TestSegmentHealthRegistry_MultiSegmentCanonicalOrder` — futures sorts before spot
- `TestSegmentHealthRegistry_MultiSegmentIndependentPhases` — segments have independent phases
- `TestSegmentHealthRegistry_SegmentPhase` — direct phase query works, unknown returns "unknown"
- `TestSegmentHealthRegistry_NilTrackerReady` — nil tracker produces ready phase with zero counters

### s429_segment_health_test.go (2 tests)
- `TestSegmentPrefix` — source-to-prefix mapping for binances, binancef, unknown, empty
- `TestSegmentPrefixConsistencyWithSettings` — prefix mapping consistent with settings.SegmentForSource

All 11 tests pass.

## Acceptance Criteria Evaluation

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Spot and Futures have distinguishable health/readiness | Met | Independent phase computation per segment in /statusz |
| Operational signals more useful and less implicit | Met | Per-segment counters and phases replace log-only visibility |
| Operability improved without scope inflation | Met | No new binaries, no OTEL, no dashboards — just JSON endpoints |
| Base ready for S430 mainnet readiness audit | Met | Segment health is queryable; gaps are documented |

## Remaining Limitations

1. **No per-segment readiness checks** — `/readyz` remains global (NATS check)
2. **No per-segment idle detection** — phase based on cumulative counters, not recency
3. **No per-segment ActivationSurface** — gate/adapter/credentials are global
4. **No per-segment control gate** — kill switch halts all segments uniformly
5. **No ingest/store segment health** — only execute binary has segment health
6. **Cumulative counters only** — no rate/windowed metrics without external tooling

## Preparation for S430

S430 (mainnet readiness audit) can now:
- Query per-segment health via `/statusz` to verify both segments are operational
- Verify segment isolation through independent phase and counter tracking
- Use `IsHealthy()` as a programmatic readiness gate
- Identify degraded segments without log analysis

Recommended focus for S430:
- Per-segment readiness checks (credential validation per segment)
- Per-segment idle detection (recency-aware phase)
- Cross-binary health aggregation (gateway probing execute segment health)
- Per-segment control gate (independent halt per segment)
