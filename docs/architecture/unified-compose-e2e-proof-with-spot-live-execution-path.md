# Unified Compose E2E Proof with Spot Live Execution Path

**Stage**: S408
**Status**: Complete
**Date**: 2026-03-23
**Predecessor**: S407 (unified runtime read-path, auditability, and segment isolation)

## Purpose

This document proves the compose-level end-to-end pipeline for the Spot segment on the unified runtime. It connects:

```
Spot ingest (binances WS) -> OBSERVATION_EVENTS -> derive pipeline ->
STRATEGY_EVENTS -> execute (SegmentRouter -> BinanceSpotTestnetAdapter) ->
lifecycle outcome (fill/reject) -> EXECUTION_FILL_EVENTS ->
store (projection -> KV) -> gateway (HTTP read-path) -> audit trail
```

This is the capstone proof that the Spot segment works end-to-end at the compose level, on a unified runtime that structurally preserves Futures coexistence.

## Compose Topology

### Service Stack

| Service | Role in Spot E2E | Port |
|---|---|---|
| nats | Event bus: OBSERVATION, STRATEGY, EXECUTION streams + KV buckets | 4222 |
| configctl | Config lifecycle: Spot bindings (source=binances) activated | 8080 |
| ingest | Spot WebSocket adapter: connects to Binance Spot via `binances.NewWSClient` | 8082 |
| derive | Pipeline: candle -> signal -> decision -> strategy (source-agnostic) | 8083 |
| execute | SegmentRouter -> BinanceSpotTestnetAdapter (venue_live) or DryRunSubmitter | 8084 |
| store | Projection: fill/rejection events -> KV materialization | 8081 |
| gateway | HTTP read-path: evidence, strategy, execution query surfaces | 8080 |
| writer | ClickHouse persistence: candles, strategies, executions | 8085 |
| clickhouse | Analytical storage | 8123/9000 |

### Compose Overlays

| Overlay | Config | dry_run | Purpose |
|---|---|---|---|
| `docker-compose.yaml` (base) | `execute.jsonc` | true | Default paper mode |
| `docker-compose.unified.yaml` | `execute-unified.jsonc` | true | Both segments, dry-run |
| `docker-compose.unified-spot-live.yaml` | `execute-venue-live-spot.jsonc` | **false** | Both segments, Spot venue_live |

The S408 proof uses `docker-compose.unified-spot-live.yaml` when credentials are available, falling back to `docker-compose.unified.yaml` (dry-run) otherwise.

## Pipeline Data Flow

### Phase 1: Spot Ingest

```
Binance Spot testnet WebSocket (wss://testnet.binance.vision/ws/btcusdt@aggTrade)
  -> ingest: WebSocketAdapterActor (source=binances)
    -> binances.ParseAggTrade -> binances.Normalize
    -> PublisherActor -> NATS OBSERVATION_EVENTS
```

The ingest binary routes to the correct exchange adapter based on the `source` field in the binding config. Source `binances` triggers `binances.NewWSClient` which connects to the Binance Spot testnet WebSocket.

### Phase 2: Derive Pipeline

```
OBSERVATION_EVENTS (source=binances, symbol=btcusdt)
  -> derive: candle aggregation (60s, 300s, 900s, 3600s)
    -> signal: RSI computation
    -> decision: RSI oversold evaluation
    -> strategy: mean_reversion_entry resolution
  -> STRATEGY_EVENTS
```

The derive pipeline is source-agnostic. Spot data (source=binances) flows through the same pipeline families as Futures data.

### Phase 3: Execute (Spot Segment)

```
STRATEGY_EVENTS
  -> execute: StrategyConsumerActor -> evaluation -> ExecutionIntent (source=binances)
    -> VenueAdapterActor:
      Gate 0: AllowedSources check (binances in allowed set)
      Gate 1: Kill switch check (EXECUTION_CONTROL KV)
      Gate 2: Staleness guard (intent.Timestamp within max_age)
      Gate 3: SubmitOrder:
        When dry_run=false:
          SegmentRouter -> SegmentForSource("binances") -> MarketSegmentSpot
            -> BinanceSpotTestnetAdapter -> POST testnet.binance.vision/api/v3/order
            <- VenueOrderReceipt (Status=filled/rejected, Fills=[{Simulated: false}])
        When dry_run=true:
          DryRunSubmitter intercepts -> synthetic fill (Simulated: true)
    -> PublishFill/PublishRejection -> EXECUTION_FILL_EVENTS / EXECUTION_REJECTION_EVENTS
```

### Phase 4: Store and Read-Path

```
EXECUTION_FILL_EVENTS / EXECUTION_REJECTION_EVENTS
  -> store:
    Fill projection: intent updated with fill data -> KV (partition: binances.btcusdt.60)
    Rejection projection: audit metadata embedded -> KV (partition: binances.btcusdt.60)
  -> gateway HTTP:
    GET /execution/status/latest?source=binances&symbol=btcusdt&timeframe=60
    GET /execution/venue_rejection/latest?source=binances&...
    GET /evidence/candles/latest?source=binances&symbol=btcusdt&timeframe=60
    GET /strategy/mean_reversion_entry/latest?source=binances&symbol=btcusdt&timeframe=60
```

### Phase 5: Analytical Persistence

```
NATS streams -> writer:
  candles -> ClickHouse candles (WHERE source = 'binances')
  strategies -> ClickHouse strategies (WHERE source = 'binances')
  executions -> ClickHouse executions
```

## Segment Routing on Unified Runtime

The `SegmentRouter` maps intent sources to adapters:

| Intent Source | MarketSegment | Adapter | Compose Config |
|---|---|---|---|
| `binances` | `spot` | `BinanceSpotTestnetAdapter` | `spot.enabled: true, adapter: binance_spot_testnet` |
| `binancef` | `futures` | `BinanceFuturesTestnetAdapter` | `futures.enabled: true, adapter: binance_futures_testnet` |
| other | - | **rejected** (fail-closed) | - |

Defense-in-depth layers:
1. **AllowedSources gate** (VenueAdapterActor): rejects intents from sources not in the enabled set
2. **SegmentRouter source mapping**: `SegmentForSource()` maps source to segment
3. **SegmentRouter adapter dispatch**: routes to the segment's registered adapter
4. **NATS consumer filter**: subscribes only to subjects matching enabled segment sources

## Controls

### Dry-Run Safety

When `dry_run=true` (compose default):
- `DryRunSubmitter` wraps the entire `SegmentRouter` as the outermost decorator
- All intents (both Spot and Futures) are intercepted before reaching any real adapter
- Fills carry `Simulated=true` and `dryrun-` prefixed VenueOrderIDs
- Zero real HTTP calls to venue testnet

When `dry_run=false` (venue_live mode):
- `DryRunSubmitter` is NOT composed
- `SegmentRouter` dispatches directly to real adapters
- Real HTTP calls reach `testnet.binance.vision` (Spot) and `testnet.binancefuture.com` (Futures)
- Fills carry `Simulated=false` and venue-assigned order IDs

### Kill Switch

The `EXECUTION_CONTROL` KV bucket provides a runtime kill switch:
- `gate.status=active`: execution proceeds normally
- `gate.status=halted`: all intents blocked regardless of dry_run or segment

### Staleness Guard

Intents older than `staleness_max_age` (120s default) are silently dropped, preventing stale strategy signals from reaching the venue.

## Evidence Summary

| Evidence | Source | Status |
|---|---|---|
| 9 unit tests (S408) | `s408_unified_compose_e2e_spot_test.go` | PASS |
| All prior tests (S405, S406, S407) | execution, actor test suites | PASS, zero regressions |
| Smoke script (16 phases) | `scripts/smoke-e2e-unified-spot.sh` | Structural |
| Compose overlay | `docker-compose.unified-spot-live.yaml` | Ready |
| Config | `execute-venue-live-spot.jsonc` | Validated (S405) |

## Limitations

1. **No Futures E2E in this proof**: Futures segment is structurally present but not exercised at compose level in S408. This is by design (Spot-first scope).
2. **Strategy dependency**: The E2E flow depends on live market data producing actionable strategy signals. Low-volatility periods may produce zero execution intents during the smoke window.
3. **Testnet-only**: All venue connectivity targets Binance testnet. No mainnet paths are exercised.
4. **Single symbol**: The proof exercises `btcusdt` only. Multi-symbol Spot E2E is not in scope.
5. **No soak or benchmark**: This is a point-in-time proof, not a sustained performance test.
