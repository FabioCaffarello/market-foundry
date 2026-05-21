# Compose Canonical Topology, Allowed Overrides, Deprecated Variants, and Limitations

> S417: Classification of compose and config artifacts into canonical, allowed, and removed categories.

---

## Canonical Artifacts

These are the **only** compose and execute config files that should exist going forward.

### Compose files (3)

| File | Classification | Purpose |
|------|---------------|---------|
| `deploy/compose/docker-compose.yaml` | **Base** | Full stack: nats, clickhouse, configctl, ingest, derive, store, execute, gateway, writer |
| `deploy/compose/docker-compose.unified.yaml` | **Overlay** | Swaps execute to unified segmented dry-run mode |
| `deploy/compose/docker-compose.venue-live.yaml` | **Overlay** | Swaps execute to real testnet mode (dry_run=false) |

### Execute configs (3)

| File | Classification | Venue mode |
|------|---------------|-----------|
| `deploy/configs/execute.jsonc` | **Base** | paper_simulator, no segments |
| `deploy/configs/execute-unified.jsonc` | **Segmented** | Both segments, dry_run=true |
| `deploy/configs/execute-venue-live.jsonc` | **Venue live** | Both segments, dry_run=false |

### Non-execute configs (stable, not part of this consolidation)

| File | Service |
|------|---------|
| `deploy/configs/configctl.jsonc` | configctl |
| `deploy/configs/ingest.jsonc` | ingest |
| `deploy/configs/derive.jsonc` | derive |
| `deploy/configs/store.jsonc` | store |
| `deploy/configs/gateway.jsonc` | gateway |
| `deploy/configs/writer.jsonc` | writer |

## Allowed Overrides

The only allowed compose override pattern is:

1. Base (`docker-compose.yaml`) always present
2. At most **one** overlay (`unified.yaml` or `venue-live.yaml`)
3. The overlay overrides **only** the `execute` service

No stacking of multiple overlays is supported or needed.

### When to create a new overlay

A new compose overlay should only be created when:

- A genuinely new execution mode is introduced (not a variant of an existing one)
- The mode requires different service wiring (not just a config change)
- The criteria in `docs/development/stages-and-governance.md` are met

### When NOT to create a new overlay

- For single-segment runs: disable the unwanted segment in the config file
- For different pipeline families: change `execute.jsonc` pipeline block
- For log level changes: change `execute.jsonc` log block
- For per-test config needs: use Go test fixtures, not compose overlays

## Deprecated Variants (Removed in S417)

The following artifacts were removed. They were marked deprecated in S416 and retained only for backward compatibility. S417 completed the migration of all consumers.

### Compose overlays removed

| File | Introduced | Deprecated | Removed | Replacement |
|------|-----------|------------|---------|-------------|
| `docker-compose.spot.yaml` | S394 | S416 | S417 | `docker-compose.unified.yaml` |
| `docker-compose.futures.yaml` | S394 | S416 | S417 | `docker-compose.unified.yaml` |
| `docker-compose.unified-spot-live.yaml` | S408 | S416 | S417 | `docker-compose.venue-live.yaml` |
| `docker-compose.unified-futures-live.yaml` | S419 | S416 | S417 | `docker-compose.venue-live.yaml` |

### Execute configs removed

| File | Introduced | Deprecated | Removed | Replacement |
|------|-----------|------------|---------|-------------|
| `execute-spot.jsonc` | S394 | S416 | S417 | `execute-unified.jsonc` |
| `execute-futures.jsonc` | S394 | S416 | S417 | `execute-unified.jsonc` |
| `execute-venue-live-spot.jsonc` | S405 | S416 | S417 | `execute-venue-live.jsonc` |
| `execute-venue-live-futures.jsonc` | S419 | S416 | S417 | `execute-venue-live.jsonc` |

## Limitations

1. **Historical docs not updated**: Architecture and stage reports from S394-S419 reference the removed files by name. These are historical records and are correct as-is.
2. **No per-segment compose isolation**: If a future requirement demands running only one segment at the compose level (not just config level), a new overlay would need to be introduced. Current assessment: not needed.
3. **Credential pass-through**: Both overlays pass all four credential env vars (spot key/secret + futures key/secret) even when only one segment is exercised. This is intentional -- the unified config expects both, and dummy values are safe under dry-run.
4. **Port 8085 anomaly resolved**: `execute-venue-live-futures.jsonc` used port 8085 while all other configs used 8084. The consolidated `execute-venue-live.jsonc` uses 8084. This inconsistency no longer exists.
