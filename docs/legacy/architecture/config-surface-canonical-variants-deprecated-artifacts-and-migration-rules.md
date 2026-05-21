# Config Surface: Canonical Variants, Deprecated Artifacts, and Migration Rules

> S416: Classification of every execute/runtime config artifact with explicit migration paths.

---

## Classification Legend

| Status | Meaning |
|--------|---------|
| **Canonical** | Actively maintained, used by default compose and documented workflows |
| **Deprecated** | Superseded by a canonical config; retained only for backward compatibility with existing smoke scripts |
| **Remove** | Scheduled for deletion in a future cleanup stage |

---

## Config File Classification

### Canonical

#### `execute.jsonc`
- **Role**: Paper simulator base config (no segments, dry_run=true by default).
- **Used by**: `docker-compose.yaml` (base stack).
- **When to use**: Development, testing, pipeline validation without venue adapters.

#### `execute-unified.jsonc`
- **Role**: Unified segmented config (Spot + Futures, dry_run=true).
- **Used by**: `docker-compose.unified.yaml`.
- **When to use**: Segmented execution with dry-run. To run a single segment, disable the unwanted segment in this config rather than using a per-segment config.

#### `execute-venue-live.jsonc`
- **Role**: Real testnet execution (both segments, dry_run=false).
- **Used by**: `docker-compose.venue-live.yaml`.
- **When to use**: Venue proofs, testnet submission, integration testing with real APIs.

### Deprecated

#### `execute-spot.jsonc`
- **Origin**: S394 (segmented compose proof).
- **Superseded by**: `execute-unified.jsonc` with futures segment omitted/disabled.
- **Migration**: Replace references with `execute-unified.jsonc`; disable futures segment if needed.

#### `execute-futures.jsonc`
- **Origin**: S394 (segmented compose proof).
- **Superseded by**: `execute-unified.jsonc` with spot segment omitted/disabled.
- **Migration**: Same as above.

#### `execute-venue-live-spot.jsonc`
- **Origin**: S405 (Spot venue acceptance/fill proof).
- **Superseded by**: `execute-venue-live.jsonc` (identical venue block).
- **Migration**: Replace references with `execute-venue-live.jsonc`.

#### `execute-venue-live-futures.jsonc`
- **Origin**: S416/S419 (Futures venue execution proof).
- **Superseded by**: `execute-venue-live.jsonc` (identical venue block).
- **Migration**: Replace references with `execute-venue-live.jsonc`.

---

## Compose Overlay Classification

### Canonical

| File | Config | Purpose |
|------|--------|---------|
| `docker-compose.yaml` | `execute.jsonc` | Base stack (paper simulator) |
| `docker-compose.unified.yaml` | `execute-unified.jsonc` | Segmented dry-run |
| `docker-compose.venue-live.yaml` | `execute-venue-live.jsonc` | Real testnet execution |

### Deprecated

| File | Superseded By | Origin |
|------|--------------|--------|
| `docker-compose.spot.yaml` | `docker-compose.unified.yaml` | S394 |
| `docker-compose.futures.yaml` | `docker-compose.unified.yaml` | S394 |
| `docker-compose.unified-spot-live.yaml` | `docker-compose.venue-live.yaml` | S408 |
| `docker-compose.unified-futures-live.yaml` | `docker-compose.venue-live.yaml` | S419 |

---

## Migration Rules

### For smoke scripts referencing deprecated configs

Existing smoke scripts that reference deprecated configs continue to work. No immediate migration is required. When scripts are updated for other reasons, they should migrate to canonical configs.

### For new development

1. **Never create new per-segment config files.** Use `execute-unified.jsonc` and toggle segment enablement.
2. **Never create new per-segment compose overlays.** Use `docker-compose.unified.yaml` or `docker-compose.venue-live.yaml`.
3. **All new venue execution modes** (e.g., mainnet, new exchanges) should extend the unified config model, not create parallel config trees.

### Removal timeline

Deprecated artifacts will be removed when all referencing smoke scripts have been migrated. This is expected to happen as part of a future simplification stage (S421+).

---

## Invariant Summary

These invariants are enforced by validation at startup (fail-closed):

1. `dry_run` defaults to `true` when omitted or null.
2. `dry_run=false` with `paper_simulator` (or no segments) is rejected.
3. Enabled segment without adapter is rejected.
4. Adapter/segment mismatch (e.g., futures adapter on spot segment) is rejected.
5. `paper_simulator` as segment adapter is rejected.
6. Segment-requiring `type` with segments map is rejected (ambiguity).
7. Segments map with zero enabled segments is rejected.
