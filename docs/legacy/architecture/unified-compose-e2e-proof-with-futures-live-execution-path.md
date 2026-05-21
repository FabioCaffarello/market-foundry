# Unified Compose E2E Proof with Futures Live Execution Path

**Stage**: S425 (supersedes S419)
**Wave**: Phase 47 -- Futures Venue Execution Proof, Post-Simplification (S421--S426)
**Status**: Active
**Date**: 2026-03-23
**Predecessor**: S424 (unified runtime read-path auditability under real Futures responses)
**Sibling**: S408 (Spot E2E compose proof -- same pattern, different segment)

## 1. Purpose

This document records the compose-level end-to-end proof for the Futures segment on the unified runtime using the canonical surface frozen by the Runtime Simplification wave (S421). It demonstrates that the full pipeline -- from Futures exchange listening through execution to persistence and read-path -- operates correctly at the compose topology level on the consolidated, canonical config and compose surface.

S425 re-validates the S419 proof on the post-simplification canonical surface, incorporating S422/S423/S424 response shapes, explicit `ValidTransition` lifecycle assertions, and multi-cycle sustained connectivity proof.

## 2. Canonical Surface Compliance

S425 operates exclusively on the canonical surface frozen by S421:

| Artifact | Canonical Path | Purpose |
|---|---|---|
| Config | `deploy/configs/execute-venue-live.jsonc` | Both segments, `dry_run=false` |
| Compose | `docker-compose.yaml` + `docker-compose.venue-live.yaml` | Base stack + venue-live overlay |

No per-segment config variants or compose overlays are used. This is enforced by S421 non-goals NG-46, NG-47, NG-49.

## 3. Compose Topology

### 3.1 Pipeline Data Flow

```
Binance Futures testnet (wss://fstream.binance.com or testnet equivalent)
  -> ingest (binancef WebSocket adapter)
  -> NATS OBSERVATION_EVENTS
  -> derive (candle -> signal -> decision -> strategy)
  -> NATS STRATEGY_EVENTS
  -> execute (SegmentRouter -> BinanceFuturesTestnetAdapter)
  -> NATS EXECUTION_FILL_EVENTS / EXECUTION_REJECTION_EVENTS
  -> store (projection -> KV)
  -> gateway (HTTP read-path)
  -> writer -> ClickHouse (analytical persistence)
```

### 3.2 Binary Composition

| Binary | Role in Futures E2E | S425 Proof Coverage |
|---|---|---|
| ingest | Listens to Futures WebSocket (source=binancef), produces OBSERVATION_EVENTS | Compose phase 6 |
| derive | Consumes observations, produces STRATEGY_EVENTS with source=binancef | Compose phase 7 |
| execute | SegmentRouter dispatches binancef intents to BinanceFuturesTestnetAdapter | Unit tests (10) + compose phase 9 |
| store | Projects fill/rejection events into NATS KV by partition key | Compose phase 11 |
| gateway | Serves HTTP read-path queries filtered by source=binancef | Compose phase 11 |
| writer | Persists Futures data into ClickHouse (source=binancef) | Compose phase 12 |

### 3.3 Compose Overlay

`docker-compose.venue-live.yaml` (canonical) overrides the execute service to use `execute-venue-live.jsonc`:

- `dry_run: false` -- real testnet orders via both adapters
- `futures.enabled: true, adapter: binance_futures_testnet` -- live Futures adapter
- `spot.enabled: true, adapter: binance_spot_testnet` -- live Spot adapter (structural coexistence)
- Both segment credentials provided via environment variables

## 4. Segment Routing and Dispatch

### 4.1 SegmentRouter Dispatch

On the unified runtime, the SegmentRouter routes intents by their `source` field:

| Source | Segment | Adapter | Testnet URL |
|---|---|---|---|
| `binancef` | Futures | `BinanceFuturesTestnetAdapter` | `testnet.binancefuture.com` |
| `binances` | Spot | `BinanceSpotTestnetAdapter` | `testnet.binance.vision` |
| other | -- | Rejected (fail-closed) | -- |

### 4.2 Futures Fill Fidelity

Futures fills use the `avgPrice`-based response model:
- Price: `avgPrice` (top-level field)
- Fee: `cumQuote` (cumulative quote quantity)
- Single fill record per order (unlike Spot which uses `fills[]` array)

This structural difference from Spot is intentional and proven across S416, S422, and S425.

## 5. Controls Layer

All controls verified for the Futures E2E path on canonical surface:

| ID | Control | Mechanism | Evidence |
|---|---|---|---|
| C1 | Dry-run safety | DryRunSubmitter wraps SegmentRouter | `TestS425_ComposeE2E_DryRun_CanonicalSurface` |
| C2 | Kill switch | EXECUTION_CONTROL KV gate | Inherited (S319, S380) |
| C3 | Staleness guard | StalenessGuard rejects stale intents | Inherited (S317) |
| C4 | Source guard | AllowedSources in VenueAdapterActor | `TestS425_ComposeE2E_AllowedSourcesGate_CanonicalSurface` |
| C5 | NATS consumer filter | Segment-filtered subscriptions | Inherited (S401) |
| C6 | Fail-closed routing | SegmentRouter rejects unknown sources | `TestS425_ComposeE2E_ConfigCoexistence_CanonicalSurface` |

## 6. Persistence and Read-Path

### 6.1 KV Materialization

Futures execution outcomes are projected into NATS KV using partition key `binancef.btcusdt.60`:
- Fill events: store projects filled intent with source, fills, and audit metadata
- Rejection events: store projects rejection with code, reason, and venue details

### 6.2 HTTP Read-Path

Gateway serves Futures outcomes via segment-qualified queries:
- `/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60`
- `/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60`
- `/analytical/composite/chains?source=binancef&symbol=btcusdt&timeframe=60`

### 6.3 ClickHouse Persistence

Writer persists Futures data into ClickHouse tables with `source = 'binancef'`, queryable via analytical endpoints.

## 7. Correlation Chain Integrity

The full E2E path preserves:
- `CorrelationID`: set at intent creation, survives through routing, adapter call, fill/rejection event, store projection, and read-path
- `CausationID`: propagated alongside correlation
- `Source`: `binancef` preserved through all stages
- `PartitionKey`: `binancef.btcusdt.60` derived consistently

Multi-cycle sustained connectivity (5 sequential orders) proves correlation chain uniqueness per order.

## 8. Segment Isolation

During Futures E2E execution:
- Spot adapter is NOT contacted (proven by guard server in tests)
- No cross-segment leakage in execution logs
- Both segments remain registered in SegmentRouter (structural coexistence)
- Unknown sources are rejected (fail-closed)

## 9. Smoke Script Modes

| Mode | Trigger | Execute Config | Compose Overlay | Evidence Strength |
|---|---|---|---|---|
| venue_live | Futures testnet credentials set | `execute-venue-live.jsonc` | `docker-compose.venue-live.yaml` | Full (real HTTP to testnet) |
| dry-run | No credentials | `execute-unified.jsonc` | `docker-compose.unified.yaml` | Structural (compose wiring) |

## 10. Structural Parity with S408 (Spot)

S425 follows the same structural pattern as S408, with enhancements from the post-simplification wave:

| Dimension | S408 (Spot) | S425 (Futures) |
|---|---|---|
| Source | `binances` | `binancef` |
| Adapter | `BinanceSpotTestnetAdapter` | `BinanceFuturesTestnetAdapter` |
| Fill model | `fills[]` array | `avgPrice`/`cumQuote` |
| Testnet URL | `testnet.binance.vision` | `testnet.binancefuture.com` |
| Partition key | `binances.btcusdt.60` | `binancef.btcusdt.60` |
| Tests | 8 tests | 10 tests |
| Smoke phases | 16 phases | 16 phases |
| Config surface | Canonical (shared) | Canonical (shared) |
| Lifecycle validation | Basic assertions | Explicit `ValidTransition` chain |
| Multi-cycle proof | Not included | 5-cycle sustained connectivity |

## 11. Upstream Evidence Integration

S425 builds on the evidence established by S422-S424:

| Stage | Tests | What It Proved |
|---|---|---|
| S422 | 19 | Real venue acceptance/fill, multi-cycle connectivity, ValidTransition |
| S423 | 19 | Real rejection (6 scenarios), partial fill, terminal state exhaustion |
| S424 | 16 + 20 sub | Read-path consolidation, segment parity (10/10 dimensions) |
| **S425** | **10** | **Compose-level E2E on canonical surface integrating all above** |

Total evidence base for Futures on canonical surface: **84+ tests** across 4 stages.
