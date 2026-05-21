# Spot Ingest Binding Seed and Runtime Projection Closure

**Stage:** S397
**Date:** 2026-03-22
**Predecessor:** S395 (Binance Segmentation Evidence Gate)
**Gap addressed:** G3 -- "Spot ingest not seeded" (severity: medium)

---

## 1. Problem Statement

S395 closed the Binance Spot/Futures Segmentation Foundation Wave with
SUBSTANTIAL evidence. Gap G3 identified that Spot ingest bindings were not
seeded: the configctl seed script defaulted to `SOURCE=binancef` (Futures), and
no `binances` exchange adapter existed on the ingest side.

Without this closure, the Spot pipeline had:
- Execute-side adapter (S392): present and tested.
- Ingest-side adapter: absent -- no `binances` package, no Spot WebSocket URL.
- Seed path: absent -- `make seed` only produced `binancef.*` topics.
- Runtime routing: hardcoded -- WebSocket actor imported only `binancef`.

The Spot-first testnet venue execution proof cannot proceed until ingest
recognizes `binances` as a valid source at every layer.

---

## 2. What Was Delivered

### 2.1 Spot Exchange Adapter (`internal/adapters/exchanges/binances/`)

New package mirroring the `binancef` adapter structure:

| File | Purpose |
|---|---|
| `aggtrade.go` | Parses Binance Spot aggTrade WebSocket messages, normalizes to canonical `ObservationTrade` with `source=binances` |
| `websocket.go` | WebSocket client connecting to `wss://stream.binance.com:9443/ws/` (Spot endpoint) with exponential backoff reconnection |
| `aggtrade_test.go` | 9 unit tests covering parse, normalize, source identity, price preservation, stream URL |

The Spot aggTrade wire format is identical to Futures (same JSON fields). The
differences are:
- `sourceName = "binances"` (vs `"binancef"`)
- `baseWSURL = "wss://stream.binance.com:9443/ws/"` (vs `"wss://fstream.binance.com/ws/"`)

### 2.2 Source-Aware WebSocket Actor

`internal/actors/scopes/ingest/websocket_actor.go` was refactored:

**Before:** Hardcoded to `binancef.NewWSClient`, `binancef.ParseAggTrade`,
`binancef.Normalize`.

**After:** `WebSocketAdapterConfig` accepts a `Source` field. The actor routes to
the correct exchange adapter via `buildHandler()`:
- `source=binancef` -> `binancef.NewWSClient` + Futures parse/normalize
- `source=binances` -> `binances.NewWSClient` + Spot parse/normalize
- Unknown source -> actor self-poisons (fail-closed)

`ExchangeScopeActor` now passes `Source` from its config to child WebSocket
actors, so the full chain is:

```
configctl binding "binances.btcusdt"
  -> BindingWatcher parses source=binances
    -> IngestSupervisor.ensureExchangeScope(source="binances")
      -> ExchangeScopeActor(source="binances")
        -> WebSocketAdapterActor(source="binances", symbol="btcusdt")
          -> binances.NewWSClient -> wss://stream.binance.com:9443/ws/btcusdt@aggTrade
          -> binances.Normalize -> ObservationTrade{Source: "binances"}
```

### 2.3 Seed Targets

| Target | Command |
|---|---|
| `make seed-spot` | `SOURCE=binances ./scripts/seed-configctl.sh` |
| `make seed-spot-multi` | `SOURCE=binances ./scripts/seed-configctl.sh --multi-symbol` |

The existing `seed-configctl.sh` already supported `SOURCE` as an environment
variable. The new Makefile targets make this discoverable and canonical.

### 2.4 Smoke Script

`scripts/smoke-spot-ingest-binding.sh` (entrypoint: `make smoke-spot-ingest`)
validates:
1. Spot adapter unit tests pass.
2. Spot bindings seed through configctl lifecycle.
3. Active config contains `binances.*` topics.
4. Execute boots with `segment=spot` and `dry_run=true`.
5. Default config restored after validation.

---

## 3. Runtime Projection Model

After S397, the runtime projection for Spot is:

```
                   configctl
                      |
           binding: "binances.btcusdt"
                      |
              BindingWatcherActor
                      |
          activateBindingMessage{Source: "binances", Symbol: "btcusdt"}
                      |
              IngestSupervisor
                      |
          ensureExchangeScope("binances")
                      |
           ExchangeScopeActor
             /               \
   PublisherActor     WebSocketAdapterActor
   (NATS publish)     (source=binances)
                           |
                  binances.NewWSClient
                           |
            wss://stream.binance.com:9443/ws/btcusdt@aggTrade
                           |
                  binances.Normalize
                           |
                  TradeReceivedEvent{Source: "binances"}
                           |
               OBSERVATION_EVENTS stream
                           |
                    DeriveScope
                           |
               EXECUTION_EVENTS stream
                           |
                    ExecuteScope (segment=spot, dry_run=true)
```

Every layer stamps or propagates `source=binances` as the segment identity
marker. NATS subjects include the source as a routing dimension:
`execution.events.paper_order.submitted.binances.btcusdt.60`.

---

## 4. Fail-Closed Semantics Preserved

| Check | Result |
|---|---|
| Unknown source in WebSocket actor | Actor self-poisons (no silent fallback) |
| Missing segment config for `binance_spot_testnet` | Startup validation rejects |
| `dry_run=true` in execute-spot.jsonc | DryRunSubmitter wraps real adapter |
| No credentials in env | Adapter falls back to paper (fail-closed) |

---

## 5. Alignment with S395 Gap Matrix

| S395 Gap | S397 Status |
|---|---|
| G1: Concurrent multi-instance compose | Not addressed (S398 scope) |
| G2: Per-segment control gate | Not addressed (operational refinement) |
| **G3: Spot ingest not seeded** | **CLOSED** -- adapter, seed target, runtime routing, smoke |
| G4: Activation surface not queryable by segment | Not addressed (observability) |
| G5: Shared core extraction | Not addressed (future wave) |
