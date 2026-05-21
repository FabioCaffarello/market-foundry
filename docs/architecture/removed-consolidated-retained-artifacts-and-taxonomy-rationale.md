# Removed, Consolidated, and Retained Artifacts — Taxonomy Rationale

**Stage:** S418
**Date:** 2026-03-23

---

## Decision Matrix

### REMOVED

| Artifact | Type | Wave | Rationale |
|----------|------|------|-----------|
| `s394_segmented_compose_test.go` | Test | S394 | Pre-unification per-segment config validation. Subsumed by `s416_config_consolidation_test.go` which tests the canonical post-consolidation config model (paper, unified, venue-live, single-segment, mismatch, empty-map). |
| `s400_multi_segment_test.go` | Test | S400 | Multi-segment config validation and source-segment mapping. Subsumed by `s416` (config), `s401` (mapping invariants), `s408`/`s419` (E2E coexistence). No unique assertions remain. |
| `s402_unified_coexistence_test.go` | Test | S402 | Unified coexistence proof (both segments, router dispatch, DryRunSubmitter). Fully subsumed by `s408` (Spot E2E) and `s419` (Futures E2E), which test identical invariants in richer integration context. |

### CONSOLIDATED (Taxonomy)

| Before | After | Files | Rationale |
|--------|-------|-------|-----------|
| "legacy" (Type-based mode) | "standalone" | `schema.go`, `run.go`, `venue_adapter_actor.go`, `execute.jsonc`, `s401_segment_sources_test.go` | Type-based mode is the canonical `paper_simulator` config — the default for development. Labeling it "legacy" falsely implies deprecation. "Standalone" accurately describes a single-adapter mode that coexists with segments-based mode. |

### RETAINED

| Artifact | Type | Wave | Rationale |
|----------|------|------|-----------|
| `s401_segment_isolation_test.go` | Test | S401 | Structural invariant layer: source-segment injectivity, consumer filtering, NATS subject structure. Not covered by any E2E or config test. |
| `s416_config_consolidation_test.go` | Test | S416 | Canonical config validation — the authoritative test for the post-S416 config surface. |
| `s408_unified_compose_e2e_spot_test.go` | Test | S408 | Canonical Spot E2E on unified runtime. |
| `s419_unified_compose_e2e_futures_test.go` | Test | S419 | Canonical Futures E2E on unified runtime. |
| All smoke scripts | Script | Various | Each referenced by active Makefile target. No redundancy at the operational proof level. |
| All deploy configs/compose | Config | S416-S417 | Already consolidated. No transitional artifacts remain. |

---

## Taxonomy: Source/Segment Model After S418

The source/segment taxonomy is canonical and requires no structural changes. The only change was removing misleading "legacy" labels.

```
VenueConfig
├── Standalone mode (venue.type)
│   └── paper_simulator → PaperVenueAdapter
│
└── Segments-based mode (venue.segments)
    ├── spot → binance_spot_testnet → BinanceSpotTestnetAdapter
    └── futures → binance_futures_testnet → BinanceFuturesTestnetAdapter

Source ↔ Segment Mapping (canonical, bijective):
  "binances" ↔ spot
  "binancef" ↔ futures
```

### Key Invariants Preserved

1. **Fail-closed segments**: absent or nil segment entry = NOT enabled
2. **Adapter-segment compatibility**: validated at config load; mismatch rejected
3. **Source-segment bijectivity**: each source maps to exactly one segment and vice versa
4. **DryRun fail-closed**: omitted or null `dry_run` = `true`
5. **Standalone/segments mutual exclusion**: segment-requiring types must use segments map

---

## Entropy Reduction Metrics

| Metric | Before S418 | After S418 | Delta |
|--------|-------------|------------|-------|
| Test files (s3xx/s4xx) | 41 | 38 | -3 |
| "legacy" label occurrences | 8 | 0 | -8 |
| Config modes with misleading names | 1 | 0 | -1 |
| Canonical test coverage gaps | 0 | 0 | 0 (no regression) |
