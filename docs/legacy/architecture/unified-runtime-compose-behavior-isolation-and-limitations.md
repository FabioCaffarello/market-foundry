# Unified Runtime Compose Behavior, Isolation, and Limitations

This document records the observed runtime behavior when Spot and Futures
coexist in the unified compose stack (S402), focusing on isolation guarantees,
behavioral boundaries, and known limitations.

## Observed Runtime Behavior

### Boot Sequence

When execute starts with `execute-unified.jsonc`:

1. Config parsed; `HasUnifiedSegments()` returns true.
2. `buildVenueAdapterFromSegments()` iterates enabled segments in canonical order.
3. For each segment, `buildVenueAdapterByType()` creates the adapter and loads credentials.
4. All adapters registered in `SegmentRouter` keyed by segment.
5. `DryRunSubmitter` wraps the router (outermost decorator).
6. Startup log emits: `type=multi_segment`, `enabled_segments=spot,futures`, `segment_count=2`, `dry_run=true`.
7. Execute supervisor spawns VenueAdapterActor with the wrapped router.
8. NATS consumer subscribes with filter subjects for both `binances` and `binancef`.

### Intent Routing

When an execution intent arrives via NATS:

1. Consumer delivers intent to VenueAdapterActor.
2. Gate 0: AllowedSources check — rejects if source not in set.
3. Gate 1+2: Kill switch + staleness check.
4. DryRunSubmitter intercepts the call (when `dry_run=true`), generates a synthetic fill.
5. If dry_run were false, SegmentRouter maps `intent.Source` -> segment -> adapter.
6. Fill or rejection event published to NATS.

### Credential Handling

Each segment loads its own credentials from environment variables:

| Segment | Key env var | Secret env var |
|---|---|---|
| Spot | `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` | `MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET` |
| Futures | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET` |

If any segment has credentials, the activation surface reports `credentials=present`.
In dry-run mode, credentials are loaded but never used for real HTTP calls.

## Isolation Guarantees

### Source-to-Segment Mapping

The mapping is hardcoded, injective, and round-trip consistent:

| Source | Segment | Adapter |
|---|---|---|
| `binances` | spot | `binance_spot_testnet` |
| `binancef` | futures | `binance_futures_testnet` |

Unknown sources (e.g., `kraken`, `bybit`, `""`) return empty segment and are rejected.

### NATS Subject Partitioning

Each execution event subject includes the source as a token:

```
execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}
```

The consumer uses `FilterSubjects` to subscribe only to enabled segment sources.
In unified mode, two filter subjects are registered:
- `execution.events.paper_order.submitted.binances.>`
- `execution.events.paper_order.submitted.binancef.>`

### VenueAdapterActor Source Guard

The actor maintains an `AllowedSources` set built from `EnabledSegmentSources()`.
Intents with sources outside this set are rejected before reaching the SegmentRouter.

### Config Validation Boundaries

The config parser enforces:
- Adapter must match segment (spot adapter in spot segment, futures in futures).
- Both `type` and `segments` cannot be set simultaneously.
- At least one segment must be enabled.
- `paper_simulator` cannot be a segment adapter.

## Compose-Level Behavior

### Service Topology (Unchanged)

The unified compose overlay only changes the execute service configuration.
All other services (nats, configctl, gateway, ingest, derive, store, writer)
run identically to the base compose.

### Boot Order

No change from base compose. Execute depends on nats (healthy) and derive (healthy).

### Health Check

Execute exposes `/readyz` on port 8084. The health check verifies NATS connectivity
and consumer health. Both segment adapters being built is a precondition for the
supervisor to start — a segment build failure is fatal and prevents boot.

## Limitations

### Per-Segment Control

| Control | Unified behavior | Per-segment override |
|---|---|---|
| `dry_run` | Binary-wide | Not supported |
| Kill switch | Binary-wide (execution_control KV) | Not supported |
| Staleness guard | Binary-wide (`staleness_max_age`) | Not supported |
| Submit timeout | Binary-wide | Not supported |

**Rationale**: Per-segment overrides add config complexity without proven need.
The current model is simpler, auditable, and sufficient for the testnet scope.

### Exchange Support

Only Binance is supported. The source-to-segment mapping is Binance-specific.
Multi-exchange support (e.g., Kraken Spot + Binance Futures) would require
extending the mapping and adapter registry.

### QueryOrder

`SegmentRouter.QueryOrder` iterates registered query ports sequentially because
the query path does not carry a source field. This is acceptable because
QueryOrder is only used for post-200 reconciliation, which is rare.

### Mainnet

No mainnet adapters exist. All adapters target testnet endpoints:
- Spot: `testnet.binance.vision`
- Futures: `testnet.binancefuture.com`

### Observability

Metrics (processed, filled, rejected) are not segmented — they aggregate across
all segments. Per-segment counters would require labeling by source in the
VenueAdapterActor, which is a future enhancement.

## Configuration Matrix

| Config file | Spot | Futures | Dry-run | Router | Use case |
|---|---|---|---|---|---|
| `execute.jsonc` | - | - | true | Paper simulator | Default development |
| `execute-spot.jsonc` | enabled | disabled | true | Spot adapter | Spot-only testing |
| `execute-futures.jsonc` | disabled | enabled | true | Futures adapter | Futures-only testing |
| `execute-unified.jsonc` | enabled | enabled | true | SegmentRouter (both) | Coexistence proof |

## Canonical Entrypoints

| Action | Command |
|---|---|
| Boot unified stack | `docker compose -f docker-compose.yaml -f docker-compose.unified.yaml up -d` |
| Seed unified bindings | `make seed-unified` |
| Run coexistence proof | `make smoke-unified-coexistence` |
| Check logs | `make logs SERVICE=execute` |
