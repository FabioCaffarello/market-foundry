# Execute/Runtime Config Consolidation

> S416: Canonical model for the execute/runtime configuration surface.

---

## Problem Statement

After the segmentation wave (S394-S403) and the venue execution proofs (S405-S420), the execute config surface accumulated 6 config files and 5 compose overlays. Many of these were per-segment or per-proof artifacts that duplicated the unified model with minor variations (a disabled segment, a different port).

This created:
- **Redundancy**: `execute-venue-live-spot.jsonc` and `execute-venue-live-futures.jsonc` had identical venue blocks.
- **Entropy**: per-segment configs (`execute-spot.jsonc`, `execute-futures.jsonc`) duplicated what the unified config already supported via segment enablement.
- **Stale documentation**: `CONFIG-REFERENCE.md` did not document segments, dry_run, or the unified model.

## Canonical Config Model

After consolidation, the execute config surface has **three canonical configs** and **four deprecated artifacts**:

### Canonical Configs

| Config | Purpose | dry_run | Segments |
|--------|---------|---------|----------|
| `execute.jsonc` | Paper simulator base (development, testing) | true (default) | None |
| `execute-unified.jsonc` | Segmented execution (Spot + Futures) | true | Both |
| `execute-venue-live.jsonc` | Real testnet execution | false | Both |

### Deprecated Artifacts (retained for backward compatibility)

| Config | Superseded By | Origin |
|--------|--------------|--------|
| `execute-spot.jsonc` | `execute-unified.jsonc` (disable futures) | S394 |
| `execute-futures.jsonc` | `execute-unified.jsonc` (disable spot) | S394 |
| `execute-venue-live-spot.jsonc` | `execute-venue-live.jsonc` | S405/S408 |
| `execute-venue-live-futures.jsonc` | `execute-venue-live.jsonc` | S416/S419 |

### Compose Overlays

| Overlay | Purpose | Status |
|---------|---------|--------|
| `docker-compose.yaml` | Base stack | Canonical |
| `docker-compose.unified.yaml` | Segmented execution (dry-run) | Canonical |
| `docker-compose.venue-live.yaml` | Real testnet execution | Canonical |
| `docker-compose.spot.yaml` | Spot-only segment | Deprecated |
| `docker-compose.futures.yaml` | Futures-only segment | Deprecated |
| `docker-compose.unified-spot-live.yaml` | Spot venue-live | Deprecated |
| `docker-compose.unified-futures-live.yaml` | Futures venue-live | Deprecated |

## Design Decisions

### Single-segment execution via enablement, not separate configs

The unified schema already supports running a single segment by omitting or disabling the other segment in the segments map. A separate per-segment config file adds no capability -- it only adds surface area to maintain.

### Unified venue-live config

The former `execute-venue-live-spot.jsonc` and `execute-venue-live-futures.jsonc` differed only in port and comments. Both enabled both segments with `dry_run: false`. Which segment is actually exercised is determined by the intents flowing through the pipeline, not by the config.

### Fail-closed semantics preserved

All fail-closed invariants from S379/S399 remain enforced:
- Omitted `dry_run` defaults to `true`
- `dry_run=false` with `paper_simulator` is rejected
- Enabled segment without adapter is rejected
- Adapter/segment mismatch is rejected
- Empty segments map is rejected

## Validation

The `s416_config_consolidation_test.go` file validates all canonical config shapes and fail-closed invariants as unit tests.
