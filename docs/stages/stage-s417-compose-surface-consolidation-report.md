# Stage S417 -- Compose Surface Consolidation Report

> Consolidated the compose and config surface from 7 compose overlays + 8 execute configs to 3 + 3 canonical artifacts. Removed 4 deprecated compose overlays, 4 deprecated execute configs, and migrated 7 smoke scripts.

---

## Objective

Reduce compose/config surface entropy by completing the removal of deprecated transitional artifacts that S416 marked for cleanup, and documenting the canonical orchestration topology.

## Scope

- Compose overlays: `deploy/compose/docker-compose.*.yaml`
- Execute configs: `deploy/configs/execute-*.jsonc`
- Smoke scripts: `scripts/smoke-*.sh` (7 scripts referencing deprecated files)
- Config reference: `deploy/configs/CONFIG-REFERENCE.md`
- Architecture docs: 2 new documents

## Findings

### Surface before S417

| Category | Total | Canonical | Deprecated |
|----------|-------|-----------|------------|
| Compose overlays | 7 | 3 | 4 |
| Execute configs | 8 | 3 | 4 (+1 base) |
| Scripts with deprecated refs | 7 | -- | -- |

### Root causes of entropy

1. **Per-segment proof artifacts**: S394 introduced `docker-compose.{spot,futures}.yaml` and `execute-{spot,futures}.jsonc` to prove single-segment boot. The unified model (S399/S400) subsumed this.
2. **Per-segment venue-live artifacts**: S405 and S419 introduced `docker-compose.unified-{spot,futures}-live.yaml` and `execute-venue-live-{spot,futures}.jsonc` for venue proofs. S416 unified these into a single `venue-live` overlay/config.
3. **Backward compatibility retention**: S416 correctly identified the redundancy and marked files deprecated, but retained them to avoid breaking smoke scripts. S417 completes the migration.

### Port 8085 anomaly

`execute-venue-live-futures.jsonc` used port 8085 while all other execute configs used 8084. This inconsistency (likely a copy error from S419) was eliminated by the removal.

## Changes Made

### Removed (8 files)

| File | Type |
|------|------|
| `deploy/compose/docker-compose.spot.yaml` | Compose overlay |
| `deploy/compose/docker-compose.futures.yaml` | Compose overlay |
| `deploy/compose/docker-compose.unified-spot-live.yaml` | Compose overlay |
| `deploy/compose/docker-compose.unified-futures-live.yaml` | Compose overlay |
| `deploy/configs/execute-spot.jsonc` | Execute config |
| `deploy/configs/execute-futures.jsonc` | Execute config |
| `deploy/configs/execute-venue-live-spot.jsonc` | Execute config |
| `deploy/configs/execute-venue-live-futures.jsonc` | Execute config |

### Updated (8 files)

| File | Change |
|------|--------|
| `scripts/smoke-segmented-compose.sh` | Rewritten to use unified overlay; proves both segments in one boot |
| `scripts/smoke-spot-ingest-binding.sh` | Migrated from spot overlay to unified overlay |
| `scripts/smoke-e2e-unified-spot.sh` | Migrated from unified-spot-live to venue-live overlay |
| `scripts/smoke-e2e-unified-futures.sh` | Migrated from unified-futures-live to venue-live overlay |
| `scripts/smoke-futures-rejection-partial-fill.sh` | Config check now validates execute-unified.jsonc |
| `scripts/smoke-spot-venue-live.sh` | Config check now validates execute-venue-live.jsonc |
| `scripts/smoke-futures-venue-live.sh` | Config check now validates execute-venue-live.jsonc |
| `deploy/configs/CONFIG-REFERENCE.md` | Removed deprecated entries from canonical config table |

### New (2 files)

| File | Purpose |
|------|---------|
| `docs/architecture/compose-surface-consolidation-and-canonical-orchestration.md` | Canonical compose topology, usage patterns, design invariants |
| `docs/architecture/compose-canonical-topology-allowed-overrides-deprecated-variants-and-limitations.md` | Classification of all artifacts, allowed overrides, removal history |

## Surface after S417

| Category | Total | Status |
|----------|-------|--------|
| Compose files | 3 | All canonical |
| Execute configs | 3 | All canonical |
| Non-execute configs | 6 | Stable (not in scope) |
| Scripts with deprecated refs | 0 | All migrated |

### Canonical compose topology

```
docker-compose.yaml                  (base: paper_simulator)
  + docker-compose.unified.yaml      (overlay: segmented dry-run)
  + docker-compose.venue-live.yaml   (overlay: real testnet)
```

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Compose surface is simpler and canonical | Met: 7 -> 3 |
| Redundant variants removed or reclassified | Met: 8 artifacts removed |
| Unified runtime orchestration is clearer | Met: 3-file model documented |
| Stage prepares removal of transitional artifacts in S418 | Met: no deprecated artifacts remain in compose/config |

## Guard Rails Adherence

| Guard rail | Status |
|------------|--------|
| No per-segment compose as norm | Met: unified model is canonical |
| No orchestration platform inflation | Met: only compose/config changes |
| No masking redundancy under new files | Met: net -6 files |
| No breaking operational scenarios | Met: all scripts migrated to canonical refs |

## Limitations

- Historical architecture docs and stage reports still reference removed filenames. These are correct as historical records and are not updated.
- Per-segment compose isolation is no longer possible at the overlay level. If needed in the future, a new overlay must be introduced (current assessment: not needed).
- The `smoke-segmented-compose.sh` script now validates unified boot rather than per-segment boot. Per-segment isolation is covered by unit tests (`TestS401_`).

## What This Stage Prepares

- S418 can focus on further runtime simplification without compose surface noise.
- The 3-file canonical model is stable and extensible for future execution modes.
- No deprecated compose/config artifacts remain to clean up.
