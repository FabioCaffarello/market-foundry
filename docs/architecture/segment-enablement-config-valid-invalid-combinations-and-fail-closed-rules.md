# Segment Enablement Config: Valid/Invalid Combinations and Fail-Closed Rules

> S399 | 2026-03-22 | Canonical reference for config validation behavior

## Validation Model

The S399 unified config model supports two modes: **Type-based** (legacy, no segments)
and **Segments-based** (unified map). Validation is fail-closed: any ambiguity,
missing field, or mismatch causes startup failure with a descriptive error.

## Valid Combinations

| # | venue.type | segments | Description |
|---|-----------|----------|-------------|
| V1 | `paper_simulator` | absent | Default paper mode, no segments |
| V2 | `""` (empty) | absent | Same as V1 (backward compatible) |
| V3 | `paper_simulator` | spot enabled | Paper fallback + spot segment |
| V4 | `""` (empty) | spot enabled | Spot-only, no fallback type |
| V5 | `""` (empty) | futures enabled | Futures-only |
| V6 | `""` (empty) | both enabled | Unified Spot + Futures |
| V7 | `paper_simulator` | both enabled | Paper fallback + both segments |
| V8 | `""` (empty) | one enabled, one disabled | Single segment with explicit disable |

## Invalid Combinations

| # | venue.type | segments | Reason | Error field |
|---|-----------|----------|--------|-------------|
| I1 | `binance_futures_testnet` | absent | Segment-requiring type needs segments map | `venue.segments` |
| I2 | `binance_spot_testnet` | absent | Same as I1 | `venue.segments` |
| I3 | `binance_futures_testnet` | futures enabled | Ambiguous: type AND segments both select adapter | `venue.type` |
| I4 | `""` (empty) | spot: adapter=`""` | Enabled segment without adapter | `venue.segments.spot.adapter` |
| I5 | `""` (empty) | spot: adapter=`unknown_venue` | Unknown adapter | `venue.segments.spot.adapter` |
| I6 | `""` (empty) | spot: adapter=`binance_futures_testnet` | Adapter/segment mismatch | `venue.segments.spot.adapter` |
| I7 | `""` (empty) | spot: adapter=`paper_simulator` | Paper cannot be segment adapter | `venue.segments.spot.adapter` |
| I8 | `""` (empty) | `"options": {...}` | Unknown segment key | `venue.segments.options` |
| I9 | `""` (empty) | all disabled | Segments map present but nothing enabled | `venue.segments` |
| I10 | `paper_simulator` | absent, `dry_run=false` | Paper is inherently dry-run | `venue.dry_run` |

## Fail-Closed Rules

### Rule 1: Absent segments = no segments active

When `venue.segments` is `null`, absent, or an empty map, no market segments
are active. Only `paper_simulator` (or empty type) is allowed.

### Rule 2: Enabled segment requires adapter

A segment with `"enabled": true` and no `"adapter"` (or empty adapter) is rejected.
There is no default adapter inference.

### Rule 3: Adapter must match segment

Each adapter has an implied segment (`binance_spot_testnet` → `spot`,
`binance_futures_testnet` → `futures`). Cross-assignment is rejected.

### Rule 4: Paper simulator is not a segment adapter

`paper_simulator` cannot appear as a segment adapter. It exists only as the
`venue.type` fallback for non-segmented use.

### Rule 5: Segment-requiring type rejects segments map

If `venue.type` is set to a segment-requiring adapter (e.g., `binance_futures_testnet`)
AND a segments map is also present, validation fails. This eliminates the ambiguity
of "which one governs adapter selection?"

### Rule 6: Segments map with nothing enabled is rejected

If the segments map is present but every entry has `"enabled": false` (or `null`),
validation fails. An empty intent is not a valid config — either enable segments
or remove the map.

### Rule 7: Unknown segment keys are rejected

Only `"spot"` and `"futures"` are valid segment keys. Any other key
(e.g., `"options"`, `"perps"`) is rejected at startup.

### Rule 8: dry_run applies uniformly

`dry_run` is not per-segment. When `true` (the default), all enabled segments
are wrapped with `DryRunSubmitter`. There is no way to dry-run one segment
while live-executing another.

## Validation Implementation

Validation lives in `VenueConfig.Validate()` and `VenueConfig.validateSegmentEnablement()`
in `internal/shared/settings/schema.go`. All rules produce `problem.ValidationIssue`
entries with specific field paths for diagnostic clarity.

## Test Coverage

All valid and invalid combinations above are covered by tests in:
- `internal/shared/settings/s393_segment_enablement_test.go`
- `internal/actors/scopes/execute/s394_segmented_compose_test.go`
