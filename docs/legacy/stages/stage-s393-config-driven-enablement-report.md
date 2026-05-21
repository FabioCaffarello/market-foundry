# Stage S393 -- Config-Driven Enablement for Binance Spot and Futures

**Wave:** Binance Spot/Futures Segmentation Foundation (S390--S395)
**Block:** B3 -- Config-Driven Segment Enablement
**Depends on:** S391 (Venue Model Refactor), S392 (Adapter Boundary Split)
**Enables:** S394 (Compose-Level Segmented Listening + Dry-Run Proof)

---

## 1. Executive Summary

S393 implements explicit, config-driven enablement for Binance Spot and Futures market segments. After S391 defined the four-dimension venue model and S392 defined the adapter boundary split, this stage adds the configuration infrastructure that governs which segments a deployment may target.

The core deliverable is a `SegmentConfig` struct with `spot_enabled` and `futures_enabled` boolean pointers, validated at startup with **fail-closed semantics**: absent or null values mean disabled. A venue type that implies a segment (e.g., `binance_futures_testnet` implies `futures`) will fail config validation unless that segment is explicitly enabled.

All existing controls (dry_run, kill switch, staleness guard, activation surface) are preserved and orthogonal to segment enablement.

---

## 2. Deliverables

| # | Artifact | Path | Status |
|---|----------|------|--------|
| D1 | SegmentConfig and VenueType.Segment() implementation | `internal/shared/settings/schema.go` | Complete |
| D2 | Segment enablement validation in VenueConfig.Validate() | `internal/shared/settings/schema.go` | Complete |
| D3 | Registration of `binance_spot_testnet` VenueType | `internal/shared/settings/schema.go` | Complete |
| D4 | Execute binary wiring for spot testnet | `cmd/execute/run.go` | Complete |
| D5 | Config example with segments documentation | `deploy/configs/execute.jsonc` | Complete |
| D6 | Architecture: config-driven enablement | `docs/architecture/config-driven-enablement-for-binance-spot-and-futures.md` | Complete |
| D7 | Architecture: config examples and fail-closed catalog | `docs/architecture/segmented-config-examples-fail-closed-behavior-and-limitations.md` | Complete |
| D8 | Tests: 25 test cases covering all segment enablement paths | `internal/shared/settings/s393_segment_enablement_test.go` | Complete |
| D9 | Stage report (this document) | `docs/stages/stage-s393-config-driven-enablement-report.md` | Complete |

---

## 3. Implementation Summary

### 3.1 Config Model Additions

**MarketSegment type:**
- `MarketSegmentSpot = "spot"`
- `MarketSegmentFutures = "futures"`

**VenueType extensions:**
- `VenueTypeBinanceSpotTestnet = "binance_spot_testnet"` registered in `knownVenueTypes`
- `VenueType.Segment()` maps each type to its implied segment
- `VenueType.RequiresSegmentConfig()` reports whether a type needs segment config

**SegmentConfig struct:**
```go
type SegmentConfig struct {
    SpotEnabled    *bool `json:"spot_enabled,omitempty"`
    FuturesEnabled *bool `json:"futures_enabled,omitempty"`
}
```

Fail-closed: `nil` pointer = disabled. Only `*field == true` enables.

### 3.2 Validation Rules

1. Segment-requiring venue types fail if `segments` block is absent.
2. Segment-requiring venue types fail if their segment is not `true`.
3. `paper_simulator` with any segment enabled is rejected (no segment applicable).
4. `paper_simulator` with empty segments block (all nil) is accepted.
5. dry_run validation is preserved and independent.

### 3.3 Execute Binary

- `binance_spot_testnet` case added to `buildVenueAdapter` switch.
- Returns explicit error indicating adapter implementation is pending (S392).
- Config validation ensures this path is only reachable with `segments.spot_enabled=true`.

---

## 4. Test Evidence

25 test cases in `s393_segment_enablement_test.go`:

| Category | Tests | Coverage |
|----------|-------|----------|
| VenueType.Segment() | 3 | Each venue type returns correct segment |
| VenueType.RequiresSegmentConfig() | 1 | Paper=false, Futures/Spot=true |
| SegmentConfig nil/absent | 2 | nil pointer and empty struct both disabled |
| SegmentConfig explicit values | 3 | true, false, mixed |
| Valid venue configs | 5 | Paper without segments, Futures/Spot with enabled, both enabled, dry_run |
| Invalid venue configs | 6 | Missing segments, disabled segment, wrong segment, paper with segments |
| dry_run preservation | 3 | dry_run=false on paper rejected, on futures accepted, default=true |

All tests pass. All existing tests unaffected (55 total in settings package).

---

## 5. Files Changed

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added MarketSegment, SegmentConfig, VenueType.Segment(), VenueType.RequiresSegmentConfig(), segment validation |
| `internal/shared/settings/s393_segment_enablement_test.go` | New: 25 test cases |
| `cmd/execute/run.go` | Added `binance_spot_testnet` case in buildVenueAdapter |
| `deploy/configs/execute.jsonc` | Added segments config documentation |
| `docs/architecture/config-driven-enablement-for-binance-spot-and-futures.md` | New: canonical enablement model |
| `docs/architecture/segmented-config-examples-fail-closed-behavior-and-limitations.md` | New: examples and fail-closed catalog |
| `docs/stages/stage-s393-config-driven-enablement-report.md` | New: this report |

---

## 6. Residual Limitations

1. **Spot adapter not implemented:** `binance_spot_testnet` is config-registered but has no adapter code. S392 defines the design; implementation is a separate step.
2. **Single venue type per binary:** Each execute instance targets one VenueType. Multi-segment routing is not supported.
3. **Mainnet not registered:** `binance_futures_mainnet` and `binance_spot_mainnet` are not in the code registry. Requires activation ceremony.
4. **No multi-exchange:** SegmentConfig is Binance-specific. Other exchanges would need a different config structure.
5. **No runtime switching:** Segment enablement is startup-time. Changes require restart.

---

## 7. Preparation for S394

S394 (Compose-Level Segmented Listening + Dry-Run Proof) can now rely on:

- **Config-validated segment identity:** The executing binary knows exactly which segment it targets, validated at startup.
- **Fail-closed guarantees:** No ambiguous segment state can reach the pipeline.
- **VenueType.Segment() API:** Downstream code can query the active segment without parsing strings.
- **Both segments registered:** Compose files can declare Spot and Futures instances with different configs, each validated independently.

Recommended S394 focus:
1. Create compose profiles for Futures-only and Spot+Futures deployments.
2. Wire NATS subject prefixes to segment identity via `VenueType.Segment()`.
3. Prove end-to-end listening + dry-run with segmented config.
4. Validate that a Spot config instance does not consume Futures intents (and vice versa).
