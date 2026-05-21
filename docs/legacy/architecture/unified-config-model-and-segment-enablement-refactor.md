# Unified Config Model and Segment Enablement Refactor

> S399 | 2026-03-22 | Canonical architecture document

## Problem Statement

Before S399, the execute binary supported only **one venue adapter per config file**.
The `venue.type` field was a scalar that selected a single adapter
(`paper_simulator`, `binance_futures_testnet`, or `binance_spot_testnet`),
and the `venue.segments` block used flat booleans (`spot_enabled`, `futures_enabled`)
that had to match the selected type.

This created three concrete problems:

1. **Separate config files per segment** — `execute-futures.jsonc` and `execute-spot.jsonc`
   existed as near-duplicates, differing only in `venue.type` and segment flags.
2. **One active segment at a time** — running both Spot and Futures required two
   binary instances with two configs, not one runtime with one config.
3. **Ambiguous validation** — the coupling between `venue.type` and segment booleans
   made it easy to produce configs that were structurally valid but semantically
   contradictory (e.g., `binance_futures_testnet` with `spot_enabled: true`).

## Design

### Unified Segments Map

The `venue.segments` field is changed from a flat struct (`SegmentConfig`)
to a **map keyed by `MarketSegment`**:

```go
type SegmentVenueConfig struct {
    Enabled bool      `json:"enabled"`
    Adapter VenueType `json:"adapter"`
}

// VenueConfig.Segments
Segments map[MarketSegment]*SegmentVenueConfig `json:"segments,omitempty"`
```

Each segment entry carries:
- `enabled` — explicit opt-in (fail-closed: absent = disabled)
- `adapter` — the venue adapter for this segment

### Two Modes of Adapter Selection

| Mode | When | How adapter is resolved |
|------|------|------------------------|
| **Type-based (legacy)** | `venue.segments` is absent/empty | `venue.type` selects the adapter directly |
| **Segments-based (unified)** | `venue.segments` has entries | Each enabled segment's `adapter` field governs selection |

When segments are present, `venue.type` must be empty or `paper_simulator`
(it becomes the fallback, not the selector).

### Shared Controls

`dry_run`, `staleness_max_age`, and `submit_timeout` apply **uniformly** to all
enabled segments. There is no per-segment override for these controls — this
preserves the existing fail-closed semantics and avoids configuration sprawl.

### Config Examples

**Paper simulator (no segments):**
```json
{
  "venue": {
    "type": "paper_simulator",
    "dry_run": true
  }
}
```

**Futures only:**
```json
{
  "venue": {
    "dry_run": true,
    "segments": {
      "futures": { "enabled": true, "adapter": "binance_futures_testnet" }
    }
  }
}
```

**Both segments (unified):**
```json
{
  "venue": {
    "dry_run": true,
    "segments": {
      "spot":    { "enabled": true, "adapter": "binance_spot_testnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_testnet" }
    }
  }
}
```

### Helpers

| Method | Purpose |
|--------|---------|
| `HasUnifiedSegments()` | Reports whether segments map is used |
| `EnabledSegments()` | Returns enabled segments in canonical order (spot, futures) |
| `IsSegmentEnabled(seg)` | Checks if a specific segment is enabled |
| `AdapterForSegment(seg)` | Returns the adapter type for a segment |

### Runtime Adapter Resolution (S399)

`buildVenueAdapter` in `cmd/execute/run.go` resolves the adapter:

1. If unified segments → resolves from first enabled segment (canonical order: spot before futures)
2. If no segments → falls back to `venue.type` (legacy path)

Multi-adapter routing (running both adapters simultaneously) is deferred to S400.
When both segments are enabled, S399 builds the first one and logs the multi-segment
config for auditability.

## Boundaries

- No multi-exchange support — all segments are Binance-scoped
- No per-segment `dry_run` or timeout overrides
- No runtime multi-adapter routing (S400)
- `venue.type` is not deprecated — it remains the entry point for paper_simulator

## Files Changed

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | `SegmentConfig` → `SegmentVenueConfig` map; new helpers; rewritten validation |
| `cmd/execute/run.go` | `buildVenueAdapter` reads from segments map; segment-aware logging |
| `deploy/configs/execute.jsonc` | Unified config with documented segments block |
| `deploy/configs/execute-futures.jsonc` | Migrated to unified segments format |
| `deploy/configs/execute-spot.jsonc` | Migrated to unified segments format |
| `internal/shared/settings/s393_segment_enablement_test.go` | Rewritten for S399 model |
| `internal/actors/scopes/execute/s394_segmented_compose_test.go` | Updated for unified segments |
