# Compose Proof with Live Listening and Dry-Run on Segmented Binance Paths

Stage: S394
Status: proven
Previous: S391 (venue model refactor), S392 (adapter boundary split), S393 (config-driven enablement)

## Summary

This document records the compose-level proof that the Foundry's segmented Binance architecture (Spot and Futures) operates correctly at runtime with live exchange listening and dry-run execution protection.

## Architecture Under Proof

After S391-S393, the segmented architecture exists in configuration and code:

```
VenueType decomposition:
  binance_futures_testnet → Exchange=binance, Segment=futures, Environment=testnet
  binance_spot_testnet    → Exchange=binance, Segment=spot,    Environment=testnet
```

Each segment has:
- Its own adapter (`BinanceFuturesTestnetAdapter`, `BinanceSpotTestnetAdapter`)
- Its own compose config (`execute-futures.jsonc`, `execute-spot.jsonc`)
- Its own compose override (`docker-compose.futures.yaml`, `docker-compose.spot.yaml`)
- Its own credential namespace (`MF_VENUE_BINANCE_FUTURES_TESTNET_*`, `MF_VENUE_BINANCE_SPOT_TESTNET_*`)
- Config-driven enablement via `segments.futures_enabled` / `segments.spot_enabled`
- Fail-closed defaults at every layer

## Compose-Level Deployment Model

The segmented architecture uses compose overrides rather than separate stacks:

```
# Default (paper mode — no segmentation):
docker compose -f docker-compose.yaml up -d

# Futures segment (dry-run):
docker compose -f docker-compose.yaml -f docker-compose.futures.yaml up -d

# Spot segment (dry-run):
docker compose -f docker-compose.yaml -f docker-compose.spot.yaml up -d
```

Each override replaces only the execute service's config and credentials. All other services (nats, ingest, derive, store, gateway, writer) remain unchanged — they process events regardless of source segment.

## Pipeline Flow per Segment

### Futures Path
```
Binance Futures mainnet (wss://fstream.binance.com)
  → ingest (source: binancef)
  → OBSERVATION_EVENTS
  → derive → signal → decision → strategy → risk
  → EXECUTION_EVENTS (paper_order.submitted.binancef.*)
  → execute (binance_futures_testnet + dry_run=true)
  → DryRunSubmitter intercepts → dryrun-{hex} fill
  → EXECUTION_FILL_EVENTS
  → store → gateway → writer
```

### Spot Path
```
Binance Spot mainnet (wss://stream.binance.com)
  → ingest (source: binances)
  → OBSERVATION_EVENTS
  → derive → signal → decision → strategy → risk
  → EXECUTION_EVENTS (paper_order.submitted.binances.*)
  → execute (binance_spot_testnet + dry_run=true)
  → DryRunSubmitter intercepts → dryrun-{hex} fill
  → EXECUTION_FILL_EVENTS
  → store → gateway → writer
```

## Segment Isolation Guarantees

| Property | Mechanism | Verified |
|----------|-----------|----------|
| Config-level isolation | VenueConfig.Validate() rejects cross-segment enablement | S394 structural tests |
| Credential namespace isolation | MF_VENUE_{TYPE}_{KEY} per segment | LoadCredentials convention |
| NATS source value isolation | `binancef` vs `binances` in subject hierarchy | S391 convention |
| Adapter selection isolation | Factory switch in buildVenueAdapter selects by VenueType | S394 wiring |
| Dry-run isolation | DryRunSubmitter never delegates to inner adapter | S379 bomb-adapter test |

## Safety Layers (Active in This Proof)

| Layer | State | Effect |
|-------|-------|--------|
| `venue.dry_run=true` | Active (default) | DryRunSubmitter intercepts all venue calls |
| `venue.type` | Segment-specific | Selects correct adapter at startup |
| `segments.*_enabled` | Explicit true | Config validation passes |
| Credentials | Dummy (smoke) | Never used because dry-run intercepts |
| Kill switch | Active (KV) | Runtime halt if needed |
| Staleness guard | 120s | Rejects stale intents |

## Startup Logging (Auditability)

Execute logs at startup with segment identity:

```
venue adapter selected  type=binance_futures_testnet  segment=futures  dry_run=true
activation surface at startup  adapter=venue  credentials=present  dry_run=true
```

```
venue adapter selected  type=binance_spot_testnet  segment=spot  dry_run=true
activation surface at startup  adapter=venue  credentials=present  dry_run=true
```

## Limitations

1. **Single execute instance per compose stack**: each override targets one segment. Multi-segment requires either two execute instances (separate ports) or sequential testing.
2. **Ingest source binding**: ingest must be configured with the correct exchange source for the target segment. The current seed config targets Futures (binancef).
3. **Spot ingest not yet seeded**: live listening for Spot requires a separate seed config with binances source bindings.
4. **Mainnet not activated**: only testnet adapters are implemented.
5. **No multi-exchange**: architecture is Binance-only.
6. **Control gate is global**: kill switch affects all segments equally.

## Files Changed

| File | Change |
|------|--------|
| `internal/application/execution/binance_spot_testnet_adapter.go` | New: Spot adapter implementation |
| `internal/application/execution/binance_spot_testnet_adapter_test.go` | New: Spot adapter tests |
| `cmd/execute/run.go` | Wire spot adapter in factory, add segment logging |
| `deploy/configs/execute-futures.jsonc` | New: Futures segmented config |
| `deploy/configs/execute-spot.jsonc` | New: Spot segmented config |
| `deploy/configs/execute.jsonc` | Update comments |
| `deploy/compose/docker-compose.futures.yaml` | New: Futures compose override |
| `deploy/compose/docker-compose.spot.yaml` | New: Spot compose override |
| `internal/actors/scopes/execute/s394_segmented_compose_test.go` | New: Structural tests |
| `scripts/smoke-segmented-compose.sh` | New: Smoke script |
| `Makefile` | Add smoke-segmented-compose target |
