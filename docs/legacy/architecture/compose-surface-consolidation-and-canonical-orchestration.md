# Compose Surface Consolidation and Canonical Orchestration

> S417: Defines the canonical compose and config topology for the unified runtime, documents the consolidation from 7 compose + 8 execute configs to 3 + 3.

---

## Context

After S416 marked 4 compose overlays and 4 execute configs as deprecated (retained for backward compatibility), the compose/config surface had accumulated transitional entropy:

- 7 compose files (3 canonical + 4 deprecated)
- 8 execute configs (3 canonical + 4 deprecated + 1 non-execute base)
- 7 smoke scripts referencing deprecated artifacts

This stage completes the removal that S416 prepared.

## Canonical Compose Topology

The canonical compose surface is exactly **3 files**:

| File | Role | Execute Config | dry_run |
|------|------|----------------|---------|
| `docker-compose.yaml` | Base stack | `execute.jsonc` | true |
| `docker-compose.unified.yaml` | Segmented dry-run overlay | `execute-unified.jsonc` | true |
| `docker-compose.venue-live.yaml` | Real testnet overlay | `execute-venue-live.jsonc` | false |

### Usage patterns

```bash
# Development / pipeline validation (paper_simulator, no segments)
docker compose -f deploy/compose/docker-compose.yaml up -d

# Segmented dry-run (both Spot + Futures segments, dry_run=true)
docker compose -f deploy/compose/docker-compose.yaml \
               -f deploy/compose/docker-compose.unified.yaml up -d

# Real testnet execution (both segments, dry_run=false)
docker compose -f deploy/compose/docker-compose.yaml \
               -f deploy/compose/docker-compose.venue-live.yaml up -d
```

### Overlay mechanics

Both overlays follow the same pattern: they override **only** the `execute` service, swapping:
- The config file mount (command + volume)
- Environment variables for testnet credentials

All other services (nats, clickhouse, configctl, ingest, derive, store, gateway, writer) are unchanged across all three modes.

## Canonical Config Topology

The canonical execute config surface is exactly **3 files**:

| File | Segments | dry_run | Port | Adapters |
|------|----------|---------|------|----------|
| `execute.jsonc` | None (paper_simulator) | true | 8084 | paper_simulator |
| `execute-unified.jsonc` | spot + futures | true | 8084 | binance_spot_testnet, binance_futures_testnet |
| `execute-venue-live.jsonc` | spot + futures | false | 8084 | binance_spot_testnet, binance_futures_testnet |

All three configs share identical `log`, `http`, `nats`, and `pipeline` blocks. The only variation is the `venue` block.

### Single-segment execution

To run only one segment (e.g., spot without futures), modify `execute-unified.jsonc` or `execute-venue-live.jsonc` by setting the unwanted segment's `enabled` to `false`. No separate per-segment config file is needed.

## Artifacts Removed (S417)

### Compose overlays (4 removed)

| File | Origin | Reason for removal |
|------|--------|--------------------|
| `docker-compose.spot.yaml` | S394 | Subsumed by `docker-compose.unified.yaml` |
| `docker-compose.futures.yaml` | S394 | Subsumed by `docker-compose.unified.yaml` |
| `docker-compose.unified-spot-live.yaml` | S408 | Identical to `docker-compose.venue-live.yaml` |
| `docker-compose.unified-futures-live.yaml` | S419 | Identical to `docker-compose.venue-live.yaml` |

### Execute configs (4 removed)

| File | Origin | Reason for removal |
|------|--------|--------------------|
| `execute-spot.jsonc` | S394 | Subset of `execute-unified.jsonc` |
| `execute-futures.jsonc` | S394 | Subset of `execute-unified.jsonc` |
| `execute-venue-live-spot.jsonc` | S405 | Identical to `execute-venue-live.jsonc` |
| `execute-venue-live-futures.jsonc` | S419 | Identical to `execute-venue-live.jsonc` (also had port 8085 inconsistency) |

### Scripts updated (7 migrated)

| Script | Previous reference | New reference |
|--------|--------------------|---------------|
| `smoke-segmented-compose.sh` | `docker-compose.{spot,futures}.yaml` | `docker-compose.unified.yaml` |
| `smoke-spot-ingest-binding.sh` | `docker-compose.spot.yaml` | `docker-compose.unified.yaml` |
| `smoke-e2e-unified-spot.sh` | `docker-compose.unified-spot-live.yaml` | `docker-compose.venue-live.yaml` |
| `smoke-e2e-unified-futures.sh` | `docker-compose.unified-futures-live.yaml` | `docker-compose.venue-live.yaml` |
| `smoke-futures-rejection-partial-fill.sh` | `execute-futures.jsonc` | `execute-unified.jsonc` |
| `smoke-spot-venue-live.sh` | `execute-venue-live-spot.jsonc` | `execute-venue-live.jsonc` |
| `smoke-futures-venue-live.sh` | `execute-venue-live-futures.jsonc` | `execute-venue-live.jsonc` |

## Design Invariants

1. **No per-segment compose overlay**: the unified model handles both segments in one config.
2. **Overlay only touches execute**: no overlay should redefine infrastructure services.
3. **Three modes, three files**: paper / segmented dry-run / venue-live. No fourth mode.
4. **Port consistency**: all execute configs use port 8084.
5. **Credential pass-through**: overlays use `${VAR:-}` syntax; dummy values are set by smoke scripts.

## Limitations

- Docs in `docs/architecture/` that reference deleted files (e.g., S405, S408, S416, S419 reports) retain historical references. These are correct as historical artifacts.
- The `smoke-segmented-compose.sh` script now proves unified boot rather than per-segment boot. Per-segment isolation is covered by unit tests (`TestS401_`).
