# Stage S416 -- Execute/Runtime Config Consolidation Report

> Consolidated the execute/runtime configuration surface from 6 configs + 5 compose overlays to 3 canonical configs + 3 canonical overlays, with 4 deprecated artifacts explicitly marked and migration paths documented.

---

## Objective

Reduce config surface entropy by consolidating redundant and transitional artifacts into a canonical model, without breaking existing smoke scripts or fail-closed invariants.

## Scope

- Config files: `deploy/configs/execute-*.jsonc`
- Compose overlays: `deploy/compose/docker-compose.*.yaml`
- Schema: `internal/shared/settings/schema.go` (no changes needed -- already canonical)
- Tests: `internal/shared/settings/s416_config_consolidation_test.go`
- Documentation: `deploy/configs/CONFIG-REFERENCE.md`, architecture docs

## Findings

### Redundancy identified

1. **`execute-venue-live-spot.jsonc` and `execute-venue-live-futures.jsonc`** had identical venue blocks (both segments enabled, `dry_run: false`). Only the port and comments differed.
2. **`execute-spot.jsonc` and `execute-futures.jsonc`** were subsets of `execute-unified.jsonc`. The unified schema already supports single-segment execution by disabling the unwanted segment.
3. **`docker-compose.unified-spot-live.yaml` and `docker-compose.unified-futures-live.yaml`** were functionally identical overlays pointing to configs that were themselves identical.

### Schema already canonical

The `VenueConfig` struct, validation logic, and fail-closed semantics required no changes. The S399/S400 unified model was already architecturally sound -- the entropy was purely at the config artifact layer.

## Changes Made

### New artifacts (canonical)

| File | Purpose |
|------|---------|
| `deploy/configs/execute-venue-live.jsonc` | Consolidated venue-live config (both segments, dry_run=false) |
| `deploy/compose/docker-compose.venue-live.yaml` | Consolidated venue-live compose overlay |
| `internal/shared/settings/s416_config_consolidation_test.go` | Invariant tests for canonical config shapes |
| `docs/architecture/execute-runtime-config-consolidation.md` | Architecture doc: consolidated model |
| `docs/architecture/config-surface-canonical-variants-deprecated-artifacts-and-migration-rules.md` | Classification and migration rules |

### Updated artifacts

| File | Change |
|------|--------|
| `deploy/configs/execute.jsonc` | Updated header: marked as CANONICAL paper base |
| `deploy/configs/execute-unified.jsonc` | Updated header: marked as CANONICAL segmented config |
| `deploy/configs/execute-spot.jsonc` | Marked DEPRECATED with migration guidance |
| `deploy/configs/execute-futures.jsonc` | Marked DEPRECATED with migration guidance |
| `deploy/configs/execute-venue-live-spot.jsonc` | Marked DEPRECATED, points to execute-venue-live.jsonc |
| `deploy/configs/execute-venue-live-futures.jsonc` | Marked DEPRECATED, points to execute-venue-live.jsonc |
| `deploy/compose/docker-compose.spot.yaml` | Marked DEPRECATED |
| `deploy/compose/docker-compose.futures.yaml` | Marked DEPRECATED |
| `deploy/compose/docker-compose.unified-spot-live.yaml` | Marked DEPRECATED |
| `deploy/compose/docker-compose.unified-futures-live.yaml` | Marked DEPRECATED |
| `deploy/configs/CONFIG-REFERENCE.md` | Updated: venue section now documents segments, dry_run, canonical file table |

## Before / After

### Config surface

| Metric | Before | After |
|--------|--------|-------|
| Total execute configs | 6 | 3 canonical + 4 deprecated |
| Total compose overlays | 5 | 3 canonical + 4 deprecated |
| Redundant venue blocks | 2 (spot-live = futures-live) | 0 |
| Per-segment configs | 2 (execute-spot, execute-futures) | 0 canonical |
| CONFIG-REFERENCE.md coverage | Missing segments, dry_run | Complete |

### Canonical model

```
execute.jsonc                    -- paper_simulator, no segments (dev/test)
execute-unified.jsonc            -- both segments, dry_run=true (segmented dry-run)
execute-venue-live.jsonc         -- both segments, dry_run=false (real testnet)

docker-compose.yaml              -- base stack (paper)
docker-compose.unified.yaml      -- segmented dry-run overlay
docker-compose.venue-live.yaml   -- real testnet overlay
```

## Test Evidence

All 18 S416 consolidation invariant tests pass:

- `TestCanonicalPaperConfigIsValid`
- `TestCanonicalUnifiedDryRunConfigIsValid`
- `TestCanonicalVenueLiveConfigIsValid`
- `TestSingleSegmentDisablementIsValid`
- `TestDryRunFalseWithPaperSimulatorIsRejected`
- `TestSegmentsWithSegmentRequiringTypeIsRejected`
- `TestAdapterSegmentMismatchIsRejected`
- `TestEmptySegmentsMapIsRejected`
- `TestEnabledSegmentSourcesReturnsCanonicalPrefixes`

Plus all pre-existing S393/S399/S400/S401 tests continue to pass.

## Guard Rails Compliance

| Guard rail | Status |
|-----------|--------|
| No new per-segment config as standard | Compliant -- deprecated, not created |
| No config platform inflation | Compliant -- only execute/runtime scope |
| No hiding legacy under new names | Compliant -- deprecated artifacts explicitly marked |
| No broken invariants | Compliant -- all fail-closed tests pass |

## Residual Items

1. **Smoke script migration**: Existing smoke scripts still reference deprecated configs. These should be migrated in a future cleanup stage.
2. **Deprecated artifact removal**: Deprecated configs and compose overlays should be deleted once all referencing scripts are migrated (target: S421+).

## Prepares

- **S417**: Compose consolidation can now build on the canonical 3-overlay model instead of navigating 7 overlays.
