# First Vertical Slice — Market Foundry

> Canonical document. Defines the minimum implementable slice to prove the observation → evidence → read path.
> Designed: 2026-03-16. This document governs the scope, boundaries, and flow of the first end-to-end pipeline.

---

## 1. Objective

Prove that the Market Foundry architecture supports a complete **data capture → normalization → derivation → query** pipeline with:

- Correct layer sovereignty across two new domains (observation, evidence)
- Actor-based supervision for two new binaries (ingest, derive)
- Event-driven communication exclusively through NATS JetStream
- Configuration-driven runtime via configctl
- Read access through the existing gateway

The slice validates **architecture and flow**, not functional breadth. One exchange, one symbol, one derived artifact type.

---

## 2. Slice Definition

```
binancef websocket
       │
       ▼
┌─────────────┐   observation.events.market   ┌─────────────┐   evidence.events.candle   ┌─────────────┐
│   ingest    │ ──────────────────────────────▶│   derive    │ ──────────────────────────▶│  (JetStream) │
│  (capture)  │   .trade.binancef             │ (candle     │   .sampled.binancef       │             │
└─────────────┘                                │  sampler)   │   .btcusdt.{tf}           └──────┬──────┘
       ▲                                       └─────────────┘                                  │
       │                                              ▲                                         │
       │ configctl.events                             │ configctl.events                        │
       │ .config.ingestion_runtime_changed            │ .config.activated                       │
       │                                              │                                         ▼
┌──────┴──────┐                                       │                              ┌─────────────────┐
│  configctl  │───────────────────────────────────────┘                              │    gateway      │
│ (lifecycle) │                                                                      │  GET /evidence/ │
└─────────────┘                                                                      │  candles/latest │
                                                                                     └─────────────────┘
```

**Scope:** `binancef:btcusdt` trades → observation events → candle sampling (60s, 300s) → evidence events → gateway read endpoint.

---

## 3. Boundaries Involved

| Boundary | Domain | Role in Slice | Status |
|----------|--------|---------------|--------|
| **configctl** | configctl | Existing. Provides ingestion bindings and runtime config for ingest/derive. | Exists (Phase 0) |
| **gateway** | (translation) | Existing. Extended with evidence query routes. | Exists, extend |
| **ingest** | observation | New binary. Captures trades from binancef WebSocket, normalizes, publishes. | New (this slice) |
| **derive** | evidence | New binary. Consumes observation trades, samples candles, publishes. | New (this slice) |
| **observation** | observation | New domain. Pure types for market data events. | New (this slice) |
| **evidence** | evidence | New domain. Pure types for derived candle events. | New (this slice) |

---

## 4. Binaries and Processes

### 4.1 ingest (`cmd/ingest`)

**Sentence:** Ingest receives raw market data from external sources and publishes normalized observation events.

**Actor topology for this slice:**
```
IngestSupervisor
├── BindingWatcherActor         (JetStream consumer: configctl.events.config.ingestion_runtime_changed)
├── SourceSupervisor            (one per active source — only binancef in this slice)
│   └── WebSocketAdapterActor   (connects to binancef futures WebSocket, parses trades)
└── ObservationPublisher        (JetStream publisher: OBSERVATION_EVENTS stream)
```

**Lifecycle:**
1. Boot → connect to NATS
2. BindingWatcherActor subscribes to `configctl.events.config.ingestion_runtime_changed`
3. On `activated` event with bindings containing `binancef` → spawn SourceSupervisor
4. SourceSupervisor spawns WebSocketAdapterActor for binancef
5. WebSocketAdapterActor connects to Binance Futures WS, subscribes to `btcusdt@aggTrade`
6. Each raw trade → normalize to `ObservationTrade` → wrap in `Envelope[ObservationTrade]` → publish to `observation.events.market.trade.binancef`

**Config dependency:** Active ingestion bindings come from configctl. Ingest does NOT hardcode exchanges or symbols.

---

### 4.2 derive (`cmd/derive`)

**Sentence:** Derive consumes observation streams and produces evidence events through configured processing pipelines.

**Actor topology for this slice:**
```
DeriveSupervisor
├── PipelineWatcherActor            (JetStream consumer: configctl.events.config.activated)
├── ExchangeScopeActor              (one per exchange — only binancef in this slice)
│   └── SymbolScopeActor            (one per symbol — only btcusdt in this slice)
│       ├── CandleSamplerActor[60s]  (samples 60s candles from trades)
│       └── CandleSamplerActor[300s] (samples 300s candles from trades)
├── ObservationConsumer             (JetStream consumer: OBSERVATION_EVENTS → derive.observation)
└── EvidencePublisher               (JetStream publisher: EVIDENCE_EVENTS stream)
```

**Lifecycle:**
1. Boot → connect to NATS
2. PipelineWatcherActor subscribes to config events to discover active pipelines
3. ObservationConsumer starts consuming from `observation.events.market.trade.>`
4. Each trade → route to ExchangeScopeActor → SymbolScopeActor → CandleSamplerActors
5. CandleSamplerActor accumulates OHLCV data per timeframe window
6. On window close → emit `EvidenceCandle` with `final=true` → publish to `evidence.events.candle.sampled.binancef.btcusdt.{timeframe}`
7. On each trade (optional) → emit interim candle with `final=false` (realtime preview)

**Candle sampling algorithm (from MarketMonkey, re-implemented):**
- Maintain current window: open, high, low, close, volume, trade_count
- Window defined by `floor(trade.timestamp / timeframe_seconds) * timeframe_seconds`
- When trade timestamp crosses window boundary → finalize current, start new
- Finalized candle published with `final: true`; interim with `final: false`

---

### 4.3 gateway extension (`cmd/gateway`)

**New routes added to existing gateway:**

| Method | Path | Purpose | NATS Subject |
|--------|------|---------|--------------|
| GET | `/evidence/candles/latest` | Latest candle per timeframe for a symbol | `evidence.query.candle.latest` |

**Query parameters:** `source=binancef&symbol=btcusdt&timeframe=60`

**Implementation:** Gateway sends NATS request to derive's query responder, returns the response as JSON. Gateway does NOT store state — it's pure pass-through.

**Note:** For this slice, derive responds to queries directly via NATS request/reply (no store binary yet). This is a deliberate simplification — store binary comes in Phase 3.

---

## 5. End-to-End Flow

```
Step 1: Operator creates and activates a config via gateway HTTP API
        POST /configs/draft → validate → compile → activate
        → configctl emits IngestionRuntimeChangedEvent (activated, bindings=[{name:binancef, topic:btcusdt}])

Step 2: ingest reacts to IngestionRuntimeChanged
        BindingWatcherActor receives event → spawns SourceSupervisor for binancef
        → WebSocketAdapterActor connects to wss://fstream.binance.com/ws/btcusdt@aggTrade

Step 3: ingest captures and normalizes trades
        Raw aggTrade JSON → ObservationTrade{Source, Symbol, Price, Quantity, Timestamp, TradeID}
        → Envelope[ObservationTrade] published to observation.events.market.trade.binancef

Step 4: derive consumes observation trades
        ObservationConsumer → route to ExchangeScopeActor[binancef] → SymbolScopeActor[btcusdt]
        → CandleSamplerActor[60s] and CandleSamplerActor[300s] accumulate OHLCV

Step 5: derive publishes evidence candles
        On window close: Envelope[EvidenceCandle] → evidence.events.candle.sampled.binancef.btcusdt.60
        On window close: Envelope[EvidenceCandle] → evidence.events.candle.sampled.binancef.btcusdt.300

Step 6: gateway serves candle queries
        GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
        → NATS request to evidence.query.candle.latest
        → derive responds with latest candle
        → gateway returns JSON to HTTP client
```

---

## 6. New Stream Definitions

### OBSERVATION_EVENTS

```
Stream:    OBSERVATION_EVENTS
Subjects:  observation.events.market.>
Storage:   File
MaxAge:    6h
MaxBytes:  1GB
MaxMsg:    1MB
Replicas:  1
```

**Consumer groups:**
- `derive.observation` — derive binary (ordered, ack-based)

### EVIDENCE_EVENTS

```
Stream:    EVIDENCE_EVENTS
Subjects:  evidence.events.candle.>
Storage:   File
MaxAge:    72h
MaxBytes:  2GB
MaxMsg:    1MB
Replicas:  1
```

---

## 7. New NATS Subject Registry

### Observation Domain

| Subject | Plane | Type | Producer | Consumer |
|---------|-------|------|----------|----------|
| `observation.events.market.trade.binancef` | events | `observation.events.v1.trade` | ingest | derive |

### Evidence Domain

| Subject | Plane | Type | Producer | Consumer |
|---------|-------|------|----------|----------|
| `evidence.events.candle.sampled.binancef.btcusdt.60` | events | `evidence.events.v1.candle_sampled` | derive | (stream) |
| `evidence.events.candle.sampled.binancef.btcusdt.300` | events | `evidence.events.v1.candle_sampled` | derive | (stream) |
| `evidence.query.candle.latest` | query | `evidence.query.v1.candle_latest` | gateway | derive |

---

## 8. New Modules and Packages

### Domain Layer (zero dependencies)

| Module | Package | Purpose |
|--------|---------|---------|
| `internal/domain/observation` | `observation` | `ObservationTrade`, domain events, invariants |
| `internal/domain/evidence` | `evidence` | `EvidenceCandle`, domain events |

### Application Layer

| Module | Package | Purpose |
|--------|---------|---------|
| `internal/application/ingest` | `ingest` | Use cases: bootstrap ingest, handle binding changes |
| `internal/application/ingest/contracts` | `contracts` | Input/output contracts for ingest use cases |
| `internal/application/derive` | `derive` | Use cases: consume observations, sample candles |
| `internal/application/derive/contracts` | `contracts` | Input/output contracts for derive use cases |
| `internal/application/evidenceclient` | `evidenceclient` | Gateway-side use cases: query candles via NATS |
| `internal/application/ports` | `ports` | `EvidenceGateway` interface (NATS request/reply) |

### Adapter Layer

| Module | Package | Purpose |
|--------|---------|---------|
| `internal/adapters/nats` | `nats` | `ObservationRegistry`, `EvidenceRegistry`, publishers, consumers |
| `internal/adapters/exchanges/binancef` | `binancef` | Binance Futures WebSocket adapter (parse aggTrade) |

### Actor Layer

| Module | Package | Purpose |
|--------|---------|---------|
| `internal/actors/scopes/ingest` | `ingest` | IngestSupervisor, BindingWatcherActor, SourceSupervisor, WebSocketAdapterActor |
| `internal/actors/scopes/derive` | `derive` | DeriveSupervisor, PipelineWatcherActor, ExchangeScopeActor, SymbolScopeActor, CandleSamplerActor |

### Interface Layer

| Module | Package | Purpose |
|--------|---------|---------|
| `internal/interfaces/http/routes` | `routes` | Evidence routes registration |
| `internal/interfaces/http/handlers` | `handlers` | Evidence candle query handler |

### Cmd Layer

| Module | Package | Purpose |
|--------|---------|---------|
| `cmd/ingest` | `main` | `main.go`, `run.go`, `actors.go` — wire and start ingest |
| `cmd/derive` | `main` | `main.go`, `run.go`, `actors.go` — wire and start derive |

---

## 9. Projections

For this slice, there is **no store binary**. The derive binary maintains in-memory state for the latest candle per symbol/timeframe and responds to NATS queries directly.

| Projection | Location | Storage | Query Subject |
|------------|----------|---------|---------------|
| Latest candle (60s, btcusdt) | derive (in-memory) | Actor state | `evidence.query.candle.latest` |
| Latest candle (300s, btcusdt) | derive (in-memory) | Actor state | `evidence.query.candle.latest` |

**Simplification:** This avoids introducing the store binary prematurely. When store is introduced in Phase 3, the query subjects remain the same — only the responder changes from derive to store.

---

## 10. Smoke and Test Strategy

### 10.1 Unit Tests (per layer)

| Layer | What to test | How |
|-------|-------------|-----|
| `domain/observation` | `ObservationTrade` construction, validation | Pure Go tests, no mocks |
| `domain/evidence` | `EvidenceCandle` construction, OHLCV invariants | Pure Go tests |
| `application/derive` | Candle sampling logic (window boundary, finalization) | Table-driven tests with deterministic timestamps |
| `adapters/nats` | `ObservationRegistry`, `EvidenceRegistry` subject correctness | Contract tests against registry |
| `adapters/exchanges/binancef` | aggTrade JSON parsing → `ObservationTrade` | Table-driven with real Binance payloads |

### 10.2 Integration Tests

| Test | Scope | Dependencies |
|------|-------|-------------|
| Ingest → NATS | Publish `ObservationTrade` → verify on stream | Embedded NATS |
| Derive → NATS | Consume observation → publish evidence candle | Embedded NATS |
| Gateway → derive query | HTTP GET `/evidence/candles/latest` → NATS → derive → response | Full stack (docker compose) |

### 10.3 End-to-End Smoke Test

```bash
# 1. Start full stack
make up

# 2. Create and activate a config with binancef binding
curl -X POST http://localhost:8080/configs/draft -d '...'
# ... validate, compile, activate

# 3. Wait for ingest to connect (observe logs)
make logs SERVICE=ingest

# 4. Verify observation events are flowing
nats sub "observation.events.market.trade.binancef" --count=5

# 5. Verify evidence candles are being produced
nats sub "evidence.events.candle.sampled.binancef.btcusdt.60" --count=2

# 6. Query via HTTP
curl http://localhost:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
```

### 10.4 Quality Gate

```bash
make check       # arch-guard, contract-audit — layer violations = build failure
make test        # all unit + integration tests
make verify      # tests + quality gate combined
```

---

## 11. Docker Compose Topology (this slice)

```yaml
services:
  nats:       # Message backbone (exists)
  configctl:  # Config lifecycle (exists)
  gateway:    # HTTP API (exists, renamed from server)
  ingest:     # NEW — market data capture
  derive:     # NEW — candle sampling
```

New service definitions follow the existing pattern: build from Dockerfile, pass `-config` flag, depend on nats (healthy) and configctl (healthy).

---

## 12. What Is Explicitly OUT of Scope

| Item | Reason | When |
|------|--------|------|
| Volume profile | Not needed to prove pipeline | Phase 3a |
| Orderbook / heatmap | Complex state — separate slice | Phase 3b |
| Multiple exchanges | Horizontal expansion, not architecture validation | Phase 3c |
| Store binary | Persistence is separate concern | Phase 3d |
| WebSocket endpoints | Client protocol is Phase 4 | Phase 4 |
| Signal domain | Upper domain, not foundation | Phase 3+ |
| Strategy / risk / execution | Way beyond foundation | Future |
| Book events (snapshot/delta) | Only trades for this slice | Phase 2 extension |
| Multiple symbols | One symbol proves the pipeline | Phase 2 extension |
| Finalized vs sampled subject split | Use single `sampled` with `final` flag in payload | Phase 2d |
| Auth / rate limiting | Gateway middleware, orthogonal | Separate track |
| Prometheus metrics | Observability aspect, not architecture | Phase 4 |
| Subject rename migration | Existing configctl subjects work — rename is separate commit | Pre-Phase 2 cleanup |

---

## 13. Implementation Order

The slice is implemented layer-by-layer, inner to outer, validating at each step:

```
Step 1: domain/observation + domain/evidence
        → make check (arch-guard validates zero infra imports)

Step 2: adapters/nats (observation + evidence registries, publisher, consumer)
        → make check (contract-audit validates subjects)

Step 3: adapters/exchanges/binancef (WebSocket adapter)
        → unit tests with recorded payloads

Step 4: application/ingest + application/derive
        → unit tests for candle sampling logic

Step 5: actors/scopes/ingest + actors/scopes/derive
        → integration tests with embedded NATS

Step 6: cmd/ingest + cmd/derive (wiring)
        → docker compose up, smoke test

Step 7: gateway extension (evidence routes + handlers)
        → end-to-end HTTP test

Step 8: docker compose update + configs
        → full stack smoke test
```

Each step produces a commit. Each commit must pass `make check`.

---

## 14. Risks and Permitted Simplifications

| Risk | Impact | Mitigation | Simplification Allowed |
|------|--------|------------|----------------------|
| Binance WS rate limits during development | Ingest can't connect | Use recorded trade replay mode for tests | Mock WebSocket adapter in tests |
| CBOR envelope overhead at high throughput | Latency | Benchmark in Step 5; envelope is non-negotiable | Accept overhead; optimize later if measured |
| Candle window accuracy depends on clock | Off-by-one windows | Use trade timestamp (not system clock) for windowing | Single-node clock is fine |
| derive holding in-memory candles = state loss on restart | Lost interim candles | Acceptable — candles rebuild from stream replay | No persistence required for this slice |
| No store binary = no historical query | Can only query latest | Acceptable — proves flow, not storage | Return only latest candle, not series |
| Single exchange adapter = no abstraction pressure | May miss adapter interface | Accept — abstract when second exchange added | Binancef adapter can be concrete, not generic |
| Config schema for ingestion bindings may not match slice needs | Binding format doesn't express exchange+symbols | Extend configctl binding schema if needed | Minimal extension only |
