# Exchange Ingress Contracts and Runtime Mode Model

## Companion documents

- [Wave charter and scope freeze](exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md)
- [Execution mode semantics, fail-closed config combinations, and limits](execution-mode-semantics-fail-closed-config-combinations-and-limits.md)
- [S377 stage report](../stages/stage-s377-exchange-ingress-contracts-and-runtime-mode-report.md)

---

## 1. Purpose

This document formalizes the contracts that govern how market data enters the
system (ingress) and how execution mode is determined at runtime. It exists to
make these contracts auditable, to prevent accidental scope confusion between
read path and write path, and to prepare the compose stack for live exchange
listening in S378.

---

## 2. Boundary taxonomy

The system is partitioned into four non-overlapping concerns. Every component
and configuration knob belongs to exactly one of these.

| Concern | Definition | Owner binary | Example components |
|---------|------------|--------------|-------------------|
| **Ingestion** | Connecting to external data sources, parsing raw messages, normalizing to domain events, publishing to NATS | `ingest` | `WSClient`, `ParseAggTrade`, `Normalize`, `WebSocketAdapterActor`, publisher actor |
| **Execution** | Receiving execution intents, applying safety gates, submitting to venue adapters, publishing fill events | `execute` | `VenueAdapterActor`, `SafetyGate`, `PaperVenueAdapter`, `BinanceFuturesTestnetAdapter` |
| **Control** | Managing the runtime gate state (kill switch), querying activation surface, exposing control endpoints | `execute` (KV store), `gateway` (HTTP proxy) | `ControlGate`, `ControlKVStore`, `ActivationSurface` |
| **Activation** | Determining the effective runtime mode from three immutable+mutable dimensions | `execute` (computed at startup + runtime) | `ComputeEffectiveMode`, `ActivationDimensions` |

**Rule:** Ingestion has zero knowledge of execution configuration. Execution
has zero knowledge of ingestion topology. Control is the only runtime-mutable
surface. Activation is derived, never stored.

---

## 3. Ingress contracts

### 3.1 Exchange WebSocket contract

| Property | Value | Source |
|----------|-------|--------|
| Endpoint | `wss://fstream.binance.com/ws/{symbol}@aggTrade` | `binancef/websocket.go:14` |
| Authentication | None (public market data stream) | Binance API docs |
| Message format | JSON `aggTrade` event | `binancef/aggtrade.go:14-25` |
| Reconnect strategy | Exponential backoff: 1s → 2s → 4s → ... → 60s cap | `binancef/websocket.go:49-86` |
| Backoff reset | After 30s of stable connection | `binancef/websocket.go:65` |
| Pong timeout | 30s | `binancef/websocket.go:18` |

**Contract invariant (CI-1):** The WebSocket client connects to **mainnet**
endpoints. There is no testnet-vs-mainnet switch for market data ingestion.
Market data is always real.

### 3.2 Normalization contract

Raw exchange messages are converted to canonical domain events at the adapter
boundary. The normalization contract guarantees:

| Field | Source (Binance) | Target (canonical) | Transformation |
|-------|-----------------|-------------------|----------------|
| Price | `p` (string) | `ObservationTrade.Price` (string) | Passthrough — no float conversion |
| Quantity | `q` (string) | `ObservationTrade.Quantity` (string) | Passthrough — no float conversion |
| Trade ID | `a` (int64) | `ObservationTrade.TradeID` (string) | `formatTradeID(aggTradeID)` |
| Symbol | `s` (uppercase) | `ObservationTrade.Symbol` (lowercase) | Lowercase normalization from config |
| Timestamp | `T` (unix ms) | `ObservationTrade.Timestamp` (time.Time UTC) | `time.UnixMilli(T).UTC()` |
| Side indicator | `m` (bool) | `ObservationTrade.BuyerMaker` (bool) | Passthrough |
| Source | — | `ObservationTrade.Source` | Hardcoded `"binancef"` |

**Contract invariant (CI-2):** Numeric precision is preserved through string
passthrough. No intermediate float conversion occurs for price or quantity.

**Contract invariant (CI-3):** Validation (`trade.Validate()`) runs after
normalization. Malformed trades are rejected before NATS publish.

### 3.3 NATS publish contract

| Property | Value | Source |
|----------|-------|--------|
| Stream | `OBSERVATION_EVENTS` | `natsobservation/registry.go:18` |
| Subject | `observation.events.market.trade` | `natsobservation/registry.go:27` |
| Event type header | `observation.events.v1.trade_received` | `natsobservation/registry.go:28` |
| Deduplication | NATS `Msg-Id` = `{source}:{tradeID}` | `ObservationTrade.DeduplicationKey()` |
| Storage | File-backed JetStream | `natsobservation/registry.go:21` |
| Retention | 6 hours / 256 MB | `natsobservation/registry.go:22-23` |

**Contract invariant (CI-4):** Trade deduplication prevents duplicate
processing on WebSocket reconnect. NATS JetStream enforces this via the
`Msg-Id` header using `DeduplicationKey()`.

### 3.4 Ingress actor hierarchy

```
IngestSupervisor
├── BindingWatcherActor          ← watches configctl for active bindings
└── ExchangeScopeActor[source]   ← one per exchange source (e.g., "binancef")
    ├── PublisherActor            ← NATS publisher (shared per source)
    └── WebSocketAdapterActor[symbol]  ← one per symbol
        └── WSClient.Run(ctx)    ← raw WebSocket read loop
```

**Contract invariant (CI-5):** Binding activation/deactivation is driven by
configctl, not by execution configuration. The ingest binary does not read
`venue.type`, credentials, or gate state.

### 3.5 Ingress event payload

```json
{
  "metadata": {
    "id": "<uuid>",
    "type": "observation.events.v1.trade_received",
    "occurred_at": "2026-03-22T14:30:00.123Z"
  },
  "trade": {
    "source": "binancef",
    "symbol": "btcusdt",
    "price": "87234.50",
    "quantity": "0.015",
    "trade_id": "4829173625",
    "buyer_maker": true,
    "timestamp": "2026-03-22T14:30:00.100Z"
  }
}
```

---

## 4. Runtime mode model

### 4.1 Three-dimensional activation surface

The effective runtime mode is determined by exactly three independent
dimensions. Two are immutable per process lifetime; one is runtime-mutable.

| Dimension | Type | Mutability | Authority | Values |
|-----------|------|-----------|-----------|--------|
| **AdapterState** | `execution.AdapterState` | Immutable (startup) | `venue.type` in config file | `paper`, `venue` |
| **CredentialState** | `execution.CredentialState` | Immutable (startup) | Environment variables (`MF_VENUE_*`) | `present`, `absent` |
| **GateStatus** | `execution.GateStatus` | Mutable (runtime) | NATS KV bucket `EXECUTION_CONTROL` | `active`, `halted` |

### 4.2 Canonical truth table

| AdapterState | GateStatus | CredentialState | EffectiveMode | Real orders? |
|-------------|-----------|-----------------|---------------|-------------|
| `paper` | `*` | `*` | `paper` | NO |
| `venue` | `halted` | `*` | `venue_halted` | NO |
| `venue` | `active` | `absent` | `venue_degraded` | NO |
| `venue` | `active` | `present` | **`venue_live`** | **YES** |

Source: `internal/domain/execution/activation.go:66-87`

**Contract invariant (CI-6):** `EffectiveMode` is always **computed**, never
stored. It is derived on every query from the current values of the three
dimensions. There is no cached or persisted mode value.

**Contract invariant (CI-7):** Only `venue_live` produces real venue orders.
All other modes are safe — zero venue interaction occurs.

### 4.3 Configuration-to-adapter mapping

| `venue.type` config value | AdapterState | Adapter implementation | CredentialState |
|--------------------------|-------------|----------------------|-----------------|
| `""` (empty/absent) | `paper` | `PaperVenueAdapter` | `absent` |
| `"paper_simulator"` | `paper` | `PaperVenueAdapter` | `absent` |
| `"binance_futures_testnet"` | `venue` | `BinanceFuturesTestnetAdapter` | depends on env vars |

Source: `cmd/execute/run.go` — `buildVenueAdapter()`

**Contract invariant (CI-8):** Empty or absent `venue.type` defaults to
`paper_simulator`. The system is paper-by-default.

### 4.4 Gate default and control

| Property | Value |
|----------|-------|
| Default gate status | `active` |
| Gate key | `"global"` (single gate for all execution families) |
| Set gate | `PUT /execution/control` → gateway → NATS KV |
| Read gate | NATS KV watch from execute binary |
| Gate scope | Global — applies to all execution families in the deployment |

Source: `internal/domain/execution/control.go:33-39`

**Contract invariant (CI-9):** The gate defaults to `active`. This is
intentional: in paper mode (the default), an active gate is harmless because
`AdapterState = paper` dominates. In venue mode, the gate provides a runtime
kill switch.

### 4.5 Activation surface observability

| Endpoint | Method | Response |
|----------|--------|----------|
| `/execution/activation/surface` | GET | `ActivationSurface` JSON |

Returns the current composite of all three dimensions plus the computed
`EffectiveMode`. This endpoint is the single source of truth for "what is this
deployment doing right now?"

---

## 5. Read path vs write path separation

### 5.1 Read path (ingestion → derivation)

```
Exchange WebSocket
  → binancef.ParseAggTrade()
  → binancef.Normalize()
  → NATS: observation.events.market.trade
  → derive consumer
  → candle → signal → decision → strategy → risk → execution intent
  → NATS: execution.events.paper_order.submitted
```

**Ownership:** `ingest` binary (ingestion) + `derive` binary (derivation).

**Configuration dependencies:** configctl bindings (which symbols to watch),
pipeline families (which domain layers to activate). No venue or credential
configuration involved.

### 5.2 Write path (execution → venue)

```
NATS: execution.events.paper_order.submitted
  → execute consumer (intake)
  → SafetyGate.Check() [kill switch + staleness]
  → VenuePort.SubmitOrder() [paper or real adapter]
  → NATS: execution.fill.venue_market_order
```

**Ownership:** `execute` binary exclusively.

**Configuration dependencies:** `venue.type`, `MF_VENUE_*` env vars, gate
state in NATS KV.

### 5.3 Independence invariant

**Contract invariant (CI-10):** The read path operates correctly regardless
of write path configuration. Changing `venue.type`, adding or removing
credentials, or halting the gate has zero effect on ingestion or derivation.

**Contract invariant (CI-11):** The write path consumes from the same NATS
subjects regardless of whether the data originated from live exchange or
simulated sources. The `OBSERVATION_EVENTS` stream carries both live and
simulated data through the same subject hierarchy.

---

## 6. NATS subject and stream topology

### 6.1 Streams

| Stream | Subjects | Owner | Consumers |
|--------|----------|-------|-----------|
| `OBSERVATION_EVENTS` | `observation.events.market.>` | `ingest` (publisher) | `derive` |
| `EXECUTION_EVENTS` | `execution.events.>` | `derive` (publisher) | `execute`, `store`, `writer` |
| `EXECUTION_FILL_EVENTS` | `execution.fill.>` | `execute` (publisher) | `store`, `writer` |

### 6.2 KV buckets

| Bucket | Key | Owner | Readers |
|--------|-----|-------|---------|
| `EXECUTION_CONTROL` | `global` | `execute` (via gateway HTTP) | `execute` (safety gate) |
| `EXECUTION_PAPER_ORDER_LATEST` | `{source}:{symbol}:{timeframe}` | `store` | `gateway` |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `{source}:{symbol}:{timeframe}` | `store` | `gateway` |

---

## 7. Safety gate composition

The execute binary composes a multi-layer safety pipeline before any venue
interaction:

```
1. Kill switch check      → SafetyGate.gateChecker.IsHalted()
2. Staleness guard        → SafetyGate.staleness.IsStale()
3. Venue adapter dispatch → VenuePort.SubmitOrder()
4. Post-200 reconciler    → Post200Reconciler wraps RetrySubmitter
5. Fill publish           → NATS EXECUTION_FILL_EVENTS
```

**Contract invariant (CI-12):** If the gate checker returns `halted` or is
unreachable (timeout after 2s), the intent is rejected with reason
`"kill_switch"`. If the intent timestamp exceeds `staleness_max_age` (default
120s), the intent is rejected with reason `"stale"`.

---

## 8. Staleness guard interaction with live data

With live exchange data, execution intents carry timestamps close to `now`.
The staleness guard's 120s default window is sufficient for live operation:

| Scenario | Intent age | Staleness verdict |
|----------|-----------|-------------------|
| Live trade → 60s candle → derive pipeline | ~60-65s | PASS (< 120s) |
| Live trade → 60s candle → derive pipeline + backlog | ~90-110s | PASS (< 120s) |
| Disconnection → reconnect → stale backlog | > 120s | REJECT (stale) |

**Assessment:** The 120s default is appropriate for live operation with 60s
timeframes. No tuning is required for S378.

---

## 9. Contract summary table

| ID | Contract | Scope | Enforcement |
|----|----------|-------|-------------|
| CI-1 | WebSocket connects to mainnet (no testnet switch for market data) | Ingestion | Hardcoded URL constant |
| CI-2 | Numeric precision preserved through string passthrough | Ingestion | No float conversion in normalization |
| CI-3 | Malformed trades rejected before NATS publish | Ingestion | `trade.Validate()` after normalization |
| CI-4 | Trade deduplication via NATS Msg-Id | Ingestion | `DeduplicationKey()` |
| CI-5 | Ingest binary does not read venue config or credentials | Ingestion | No venue dependency in ingest code |
| CI-6 | EffectiveMode is computed, never stored | Activation | `ComputeEffectiveMode()` on every query |
| CI-7 | Only venue_live produces real orders | Execution | Truth table in `ComputeEffectiveMode()` |
| CI-8 | Empty venue.type defaults to paper_simulator | Execution | `buildVenueAdapter()` default case |
| CI-9 | Gate defaults to active (safe in paper mode) | Control | `DefaultControlGate()` |
| CI-10 | Read path independent of write path config | Architecture | No shared config between ingest/derive and execute |
| CI-11 | Live and simulated data share NATS subjects | Architecture | Single subject hierarchy |
| CI-12 | Safety gate rejects on halted gate or stale intent | Execution | `SafetyGate.Check()` |
