# Transitional Artifact Removal and Source/Segment Taxonomy Cleanup

**Stage:** S418
**Status:** Complete
**Scope:** Post-S416/S417 consolidation — remove transitional test artifacts and clean source/segment taxonomy labels.

---

## 1. Context

After S416 (Execute Runtime Config Consolidation) and S417 (Compose Surface Consolidation), the config and compose surfaces are canonical. However, the codebase retained:

- **Transitional test files** from waves S394-S402 that validated pre-unification config structures now fully subsumed by canonical tests.
- **Misleading "legacy" labels** in code comments for the standalone Type-based config mode, which is NOT deprecated — it is the canonical mode for `paper_simulator`.

This stage removes the transitional test artifacts and corrects the taxonomy to eliminate false deprecation signals.

---

## 2. Taxonomy Cleanup: "legacy" to "standalone"

### Problem

The Type-based config mode (`venue.type = "paper_simulator"`) was labeled "legacy" in comments across `schema.go`, `run.go`, `venue_adapter_actor.go`, and `execute.jsonc`. This label is incorrect because:

1. Type-based mode is the **canonical and only valid mode** for `paper_simulator` — the default development config.
2. Calling it "legacy" implies planned removal, which would break the default workflow.
3. The two modes (standalone vs segments-based) are complementary, not transitional.

### Resolution

All "legacy" labels describing the Type-based config mode were replaced with "standalone":

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | 4 comment occurrences updated |
| `cmd/execute/run.go` | 1 comment updated |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | 1 comment updated |
| `deploy/configs/execute.jsonc` | 1 comment updated |
| `internal/shared/settings/s401_segment_sources_test.go` | Test function renamed `Legacy` to `Standalone` |

### Canonical Terminology After S418

| Term | Meaning | Example |
|------|---------|---------|
| **Standalone mode** | `venue.type` selects a single adapter; no segments map | `{"venue": {"type": "paper_simulator"}}` |
| **Segments-based mode** | `venue.segments` maps each market segment to its adapter | `{"venue": {"segments": {"spot": {...}, "futures": {...}}}}` |
| **Source** | Ingest-origin prefix string | `"binances"`, `"binancef"` |
| **MarketSegment** | Enum classifying market type | `"spot"`, `"futures"` |
| **VenueType** | Adapter identifier constant | `"paper_simulator"`, `"binance_spot_testnet"` |

---

## 3. Transitional Test Removal

### Removed Files

| File | Stage | Reason |
|------|-------|--------|
| `s394_segmented_compose_test.go` | S394 | Tested pre-S399 per-segment config structures. All config validation assertions are covered by `s416_config_consolidation_test.go`. Cross-segment adapter rejection is covered by s416 mismatch tests. |
| `s400_multi_segment_test.go` | S400 | Tested multi-segment config validation and source-segment round-trip. Fully subsumed by `s416_config_consolidation_test.go` (config), `s401_segment_isolation_test.go` (mapping), and `s408/s419` E2E tests (coexistence). |
| `s402_unified_coexistence_test.go` | S402 | Tested both-segments coexistence, router dispatch, and DryRunSubmitter wrapping. All assertions are covered by `s408_unified_compose_e2e_spot_test.go` and `s419_unified_compose_e2e_futures_test.go`, which test the same invariants in full E2E context. |

### Retained Files

| File | Stage | Reason |
|------|-------|--------|
| `s401_segment_isolation_test.go` | S401 | **Unique structural invariants** not covered elsewhere: source-to-segment mapping injectivity proof, NATS consumer subject filtering by segment, subject structure validation. Essential defense-in-depth. |
| `s416_config_consolidation_test.go` | S416 | **Canonical** post-consolidation config validation. Covers paper, unified dry-run, venue-live, single-segment, mismatch rejection, empty-map rejection. |
| `s408_unified_compose_e2e_spot_test.go` | S408 | **Canonical** Spot E2E on unified runtime. |
| `s419_unified_compose_e2e_futures_test.go` | S419 | **Canonical** Futures E2E on unified runtime. |

---

## 4. Artifacts NOT Removed (With Rationale)

### Smoke Scripts

All existing smoke scripts remain. Each is referenced by an active Makefile target and serves a distinct operational proof scope. Even scripts from earlier waves (S280, S317, S330) validate foundational invariants (restart recovery, persistence round-trip, composed pipeline) that are orthogonal to segmentation concerns.

### Configs and Compose Files

The config and compose surface was already consolidated by S416/S417. The current canonical set is:

- `execute.jsonc` — standalone paper_simulator (default)
- `execute-unified.jsonc` — segments-based dry-run (both segments)
- `execute-venue-live.jsonc` — segments-based live (both segments, dry_run=false)
- `docker-compose.yaml` — base stack
- `docker-compose.unified.yaml` — segmented overlay
- `docker-compose.venue-live.yaml` — testnet overlay

No further config/compose removals needed.

### Documentation

Architecture docs from completed waves (S390-S395, S389, S396) describe historical transitions. They are retained as evidence artifacts — archiving them is a separate documentation governance concern, not an S418 code-level cleanup.
