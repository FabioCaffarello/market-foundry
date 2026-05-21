# Config-Driven Enablement for Binance Spot and Futures

**Stage:** S393
**Date:** 2026-03-22
**Wave:** Binance Spot/Futures Segmentation Foundation (S390--S395)
**Depends on:** S391 (Venue Model Refactor), S392 (Adapter Boundary Split)
**Authority:** This document defines the canonical segment enablement config model. Changes require a new stage.

---

## 1. Problem Statement

After S391 decomposed the venue model into four orthogonal dimensions (exchange, market segment, environment, execution mode) and S392 defined the adapter boundary split for Spot and Futures, there is no mechanism to **explicitly enable or disable** individual market segments by configuration.

Without config-driven enablement:
- A deployment could accidentally target a segment that was not intended.
- There is no fail-closed default -- an absent config allows any venue type through.
- Operators cannot declaratively state which segments a given deployment supports.

### 1.1 Design Goals

| Goal | Description |
|------|-------------|
| Explicit enablement | Each segment requires `*_enabled: true` in config |
| Fail-closed | Absent or null segment config blocks all segment-requiring venue types |
| Config validation at startup | Invalid combinations are caught before any order processing |
| Preserve dry_run | Segment enablement is orthogonal to dry_run -- both must pass |
| No ad-hoc flags | Segment enablement lives in a structured `segments` block |

---

## 2. Config Model

### 2.1 SegmentConfig

Added to `VenueConfig` as an optional `segments` field:

```go
type SegmentConfig struct {
    SpotEnabled    *bool `json:"spot_enabled,omitempty"`
    FuturesEnabled *bool `json:"futures_enabled,omitempty"`
}
```

**Fail-closed semantics:**
- `nil` SegmentConfig pointer: all segments disabled.
- `nil` field pointer within SegmentConfig: that segment is disabled.
- Only `*field == true` enables a segment.

### 2.2 VenueType.Segment()

Each VenueType now exposes its implied market segment:

```go
VenueTypeBinanceFuturesTestnet.Segment() == MarketSegmentFutures
VenueTypeBinanceSpotTestnet.Segment()    == MarketSegmentSpot
VenueTypePaperSimulator.Segment()        == ""  // no segment
```

### 2.3 Validation Rules

At startup, `VenueConfig.Validate()` enforces:

1. **Segment-requiring types need segments config:** If `VenueType.Segment() != ""`, the `segments` block must be present.
2. **Matching segment must be enabled:** The VenueType's implied segment must be explicitly `true`.
3. **Paper has no segments:** `paper_simulator` with any segment enabled is rejected.
4. **dry_run preserved:** Segment validation is additive to existing dry_run validation.

### 2.4 VenueType Registry

S393 registers `binance_spot_testnet` as a known VenueType:

```go
VenueTypeBinanceSpotTestnet VenueType = "binance_spot_testnet"
```

This makes it available for config selection once the S392 adapter implementation is complete.

---

## 3. Config Flow at Startup

```
Config Load
    |
    v
VenueConfig.Validate()
    |-- Is venue type known?
    |-- Duration fields valid?
    |-- dry_run consistent with venue type? (S379)
    |-- Segment enablement consistent? (S393)
    |       |-- VenueType.RequiresSegmentConfig()?
    |       |       |-- Yes: segments block present?
    |       |       |       |-- segment enabled?
    |       |       |       |       |-- Yes: pass
    |       |       |       |       `-- No: REJECT
    |       |       |       `-- No: REJECT
    |       |       `-- No: pass (paper_simulator)
    |       `-- Paper with segments enabled? REJECT
    |
    v
buildVenueAdapter()  // only reaches if validation passes
```

---

## 4. Interaction with Existing Controls

| Control | Source | Relationship to S393 |
|---------|--------|---------------------|
| dry_run | S379 | Orthogonal: segment must be enabled AND dry_run must be appropriate |
| kill switch | S344 | Runtime gate, independent of config-time segment enablement |
| staleness guard | S328 | Runtime check, independent of config-time segment enablement |
| activation surface | S339 | Reflects adapter state; segment enablement is a precondition |

---

## 5. Invariants

1. No order can reach a venue adapter whose segment is not config-enabled.
2. Config validation fails fast at startup -- no runtime ambiguity.
3. `paper_simulator` never requires segment config and rejects it when segments are enabled.
4. Both segments can be enabled simultaneously (for future multi-segment deployments).
5. Absent segment config is equivalent to all segments disabled (fail-closed).

---

## 6. Limitations

- **Single venue type per binary:** Each execute instance targets one VenueType. Multi-segment within a single binary is not supported.
- **Spot adapter not yet implemented:** `binance_spot_testnet` is registered in config but `buildVenueAdapter` returns an error until S392 adapter code lands.
- **No mainnet segments:** `binance_futures_mainnet` and `binance_spot_mainnet` are defined in S391 docs but not registered in code. This is intentional -- mainnet activation requires a separate ceremony.
- **No multi-exchange:** Segment config is Binance-specific. Expanding to other exchanges requires a new config model.
