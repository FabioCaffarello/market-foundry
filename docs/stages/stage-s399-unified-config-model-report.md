# S399 — Unified Config Model and Segment Enablement Refactor

> Stage report | 2026-03-22 | Phase 42: Unified Segment Runtime Foundation Wave

## Objective

Eliminate the "one config active per segment" limitation by designing,
implementing, and validating a unified configuration model that governs
Spot, Futures, or both within a single config file.

## What Changed

### Config Model (`internal/shared/settings/schema.go`)

**Before (S393):** `venue.segments` was a flat `*SegmentConfig` with
`spot_enabled *bool` and `futures_enabled *bool`. These booleans had to
match the single `venue.type` scalar — one adapter per config.

**After (S399):** `venue.segments` is a `map[MarketSegment]*SegmentVenueConfig`.
Each segment entry carries `enabled bool` and `adapter VenueType`. The map
naturally supports Spot, Futures, or both in one config.

New helpers added:
- `HasUnifiedSegments()` — reports whether segments map is used
- `EnabledSegments()` — returns enabled segments in canonical order
- `IsSegmentEnabled(seg)` — checks single segment
- `AdapterForSegment(seg)` — returns adapter type for segment

### Validation (`schema.go`)

Validation rewritten with comprehensive fail-closed rules:
- Unknown segment keys rejected
- Adapter/segment mismatch rejected
- Enabled segment without adapter rejected
- Paper simulator as segment adapter rejected
- Segment-requiring `venue.type` with segments map rejected (ambiguity)
- Segments map with nothing enabled rejected
- `dry_run` applies uniformly to all segments

### Runtime (`cmd/execute/run.go`)

`buildVenueAdapter` refactored into three functions:
- `buildVenueAdapter` — entry point, dispatches to segments or legacy path
- `buildVenueAdapterFromSegments` — resolves from unified segments map
- `buildVenueAdapterByType` — builds adapter for a given type

Multi-segment detection: when both segments are enabled, currently selects
the first enabled segment and logs the multi-segment configuration.
Multi-adapter routing deferred to S400.

### Config Files (`deploy/configs/`)

- `execute.jsonc` — unified with documented segments block (commented examples)
- `execute-futures.jsonc` — migrated to unified segments format
- `execute-spot.jsonc` — migrated to unified segments format

### Tests

- `s393_segment_enablement_test.go` — rewritten (26 tests covering all valid/invalid combinations)
- `s394_segmented_compose_test.go` — updated for unified segments model (8 tests)

## Acceptance Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Spot and Futures governed by single config | Done | `execute.jsonc` segments block; `TestVenueValidateAcceptsBothSegmentsEnabled` |
| Single-config-at-a-time problem eliminated | Done | Segments map accepts both; `EnabledSegments()` returns multiple |
| Ambiguous combinations fail early | Done | 10 invalid combination tests all pass |
| Prepares multi-segment merge for S400 | Done | `buildVenueAdapterFromSegments` detects multi-segment; logs active segment |

## Residual Gaps

| ID | Gap | Owner | Target |
|----|-----|-------|--------|
| G1 | Multi-adapter routing (both adapters wired simultaneously) | S400 | Multi-segment runtime |
| G2 | Per-segment `dry_run` override (not needed yet) | Deferred | Future wave if ever |

## Guard Rails Observed

- No separate config per segment created — unified map model
- No multi-exchange opened — all segments are Binance-scoped
- No ad-hoc flags scattered — all config in `SegmentVenueConfig` struct
- No incompatibilities masked — validation rejects all ambiguous states

## Promoted Documents

- [`docs/architecture/unified-config-model-and-segment-enablement-refactor.md`](../architecture/unified-config-model-and-segment-enablement-refactor.md)
- [`docs/architecture/segment-enablement-config-valid-invalid-combinations-and-fail-closed-rules.md`](../architecture/segment-enablement-config-valid-invalid-combinations-and-fail-closed-rules.md)
