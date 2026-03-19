# Controlled Capability 01 — Multi-Symbol Live Monitoring

> Stage: S119 | Status: Defined | Date: 2025-03-19

## 1. Objective

Activate and sustain the full pipeline chain for **two or more symbols simultaneously** (btcusdt + ethusdt) under controlled live operation, using exclusively the existing config-driven activation mechanism and zero new application code.

This is the first capability delivery on the proven architecture. It validates that the mesh scales horizontally as designed, under real concurrent load.

## 2. Capability Identity

| Field | Value |
|-------|-------|
| **Name** | Multi-Symbol Live Monitoring |
| **Type** | Operational capability (config-only) |
| **Risk** | Low — smoke-tested via `make smoke-multi` |
| **New code** | Zero application code. Config + operational tooling only. |
| **Primary value** | Proves horizontal scaling of the full event pipeline |

## 3. Why This Capability, Why Now

### 3.1 Strategic Rationale

The S118 readiness review concluded that the architecture is proven under controlled, single-symbol conditions. The next insight does not come from more architecture work — it comes from **using the architecture to deliver capability**.

Multi-symbol monitoring is the ideal first capability because:

1. **Lowest possible risk.** The pipeline, actors, projections, and query surfaces already handle symbol as a first-class dimension. `make smoke-multi` has exercised this path. No new domain logic is needed.

2. **Highest architectural signal.** Running two full pipeline chains concurrently stresses every runtime simultaneously — ingest (two WS connections), derive (double event throughput), store (double projection writes), execute (two independent paper order streams), gateway (queries across symbols).

3. **Natural soak testing.** An operator who activates multi-symbol monitoring will naturally run it for hours, creating the sustained operation that S118 identified as the key unproven dimension — without requiring dedicated soak infrastructure.

4. **Config-driven activation proof.** The entire activation happens through the configctl lifecycle (draft → validate → compile → activate). This proves the config system handles multiple concurrent bindings, which is the fundamental scaling mechanism of the Foundry.

5. **Immediate operational value.** Monitoring multiple markets simultaneously is the most basic capability a trading system should have.

### 3.2 Alternatives Evaluated and Rejected

| Alternative | Risk | Why Rejected |
|-------------|------|--------------|
| New signal family (e.g., MACD) | Medium | Requires new domain code, actor wiring, tests. Premature before proving horizontal scaling. |
| Candle history query enrichment | Low-Med | Useful but exercises only store+gateway, not the full mesh. |
| Strategy performance tracking | Medium | New domain concept. Better after multi-symbol proves the base. |
| Live venue adapter (testnet) | High | Requires external dependency (Binance testnet API), credentials, new safety gates. Explicitly deferred. |
| MarketMonkey absorption | High | Importing external code before proving Foundry delivers on its own patterns creates coupling before validation. |

## 4. Runtimes and Domains Exercised

### 4.1 Runtime Activation Map

| Runtime | What Changes | Pressure Type |
|---------|-------------|---------------|
| **configctl** | Receives second binding (ethusdt). Publishes `IngestionRuntimeChangedEvent` for new binding. | Config lifecycle under concurrent state. |
| **ingest** | Opens second WebSocket connection (Binance ethusdt). Publishes to OBSERVATION_EVENTS with new symbol key. | Concurrent WS management, doubled event production. |
| **derive** | All samplers (candle, rsi, etc.) process events for both symbols independently. | Doubled throughput, independent state per symbol. |
| **store** | All projection actors materialize KV entries for both symbols. | Doubled KV write load, key-space expansion. |
| **execute** | Paper venue adapter processes orders for both symbols. | Concurrent order evaluation across symbols. |
| **gateway** | Query surfaces serve data for both symbols via existing `?symbol=` parameter. | No change — already parameterized. |

### 4.2 Domain Coverage

All 8 domains are exercised. No new domains are introduced.

| Domain | Impact |
|--------|--------|
| observation | Second symbol's trade stream |
| evidence | Candle/tradeburst/volume for ethusdt |
| signal | RSI computation for ethusdt |
| decision | rsi_oversold evaluation for ethusdt |
| strategy | mean_reversion_entry for ethusdt |
| risk | position_exposure for ethusdt |
| execution | paper_order for ethusdt |
| configctl | Second ingestion binding management |

### 4.3 NATS Topology

No new streams or consumers are needed. Events are partitioned by subject key (source.symbol.timeframe). The existing 9 streams and 11 durable consumers handle multi-symbol by design.

## 5. Activation Flow

### 5.1 Config Activation Sequence

```
1. POST /configctl/configs/draft
   → Create draft with ethusdt binding added to existing btcusdt config

2. POST /configctl/configs/{id}/validate
   → Validate pipeline dependencies (all families already enabled)

3. POST /configctl/configs/{id}/compile
   → Compile binding set

4. POST /configctl/configs/{id}/activate
   → Activate → publishes IngestionRuntimeChangedEvent
   → ingest discovers new binding
   → ingest opens second WS connection
   → events flow through full chain for ethusdt
```

### 5.2 Operational Procedure

```bash
# Option A: Use existing multi-symbol seed
make seed-multi          # seeds btcusdt + ethusdt bindings

# Option B: Incremental activation (preferred for validation)
make seed                # seed btcusdt first
# ... validate single-symbol flow ...
make seed-multi          # activate ethusdt on top of existing btcusdt
# ... validate both symbols flowing ...
```

## 6. Query Surfaces

All existing gateway endpoints support multi-symbol queries via the `?symbol=` parameter. No new endpoints are needed.

### 6.1 Validation Queries

```
GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /evidence/candles/latest?source=binancef&symbol=ethusdt&timeframe=60

GET /signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /signal/rsi/latest?source=binancef&symbol=ethusdt&timeframe=60

GET /decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /decision/rsi_oversold/latest?source=binancef&symbol=ethusdt&timeframe=60

GET /strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /strategy/mean_reversion_entry/latest?source=binancef&symbol=ethusdt&timeframe=60

GET /risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /risk/position_exposure/latest?source=binancef&symbol=ethusdt&timeframe=60

GET /execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60
GET /execution/paper_order/latest?source=binancef&symbol=ethusdt&timeframe=60
```

### 6.2 Diagnostic Surfaces

```
GET /healthz            → all runtimes healthy
GET /readyz             → all runtimes ready
GET /statusz            → tracker counts show activity for both symbols
GET /diagz              → readiness checks all pass
```

## 7. Diagnostics and Observability

### 7.1 What Exists Today

- `/statusz` tracker counts per runtime (aggregate, not per-symbol)
- `/diagz` readiness checks (binary pass/fail)
- Structured logs with `source`, `symbol`, `timeframe` fields

### 7.2 What This Capability Will Reveal

- Whether tracker counts accurately reflect doubled throughput
- Whether logs are sufficient to debug cross-symbol issues without correlation IDs
- Whether `/statusz` needs per-symbol breakdown (suspected: yes)
- Whether resource usage (memory, goroutines) scales linearly

## 8. Implementation Scope

### 8.1 Required Changes (Minimal)

| Item | Type | Effort |
|------|------|--------|
| Update `seed-configctl.sh` to support incremental symbol addition | Script | Small |
| Update `live-pipeline-activate.sh` to validate multi-symbol flow | Script | Small |
| Add multi-symbol validation to smoke test scenarios | Test | Small |
| Document multi-symbol activation procedure | Doc | Small |

### 8.2 Explicitly Not Required

- No new actors, samplers, or projections
- No new NATS streams or consumers
- No new gateway endpoints
- No new domain models
- No changes to the actor engine or supervisor hierarchy
- No changes to config validation or compilation logic

## 9. Expected Gains

| Gain | Category |
|------|----------|
| Proof that pipeline scales horizontally via config | Architecture |
| Sustained multi-symbol operation baseline | Operations |
| Natural soak testing without dedicated infrastructure | Operations |
| Confidence to add more symbols (3, 5, 10...) | Scaling |
| First capability delivered on the proven mesh | Strategic |
| Evidence for next capability prioritization | Strategic |

## 10. Relationship to Deferred Items

This capability may trigger deferred items from S116:

| Deferred Item | Trigger Condition | Likelihood |
|---------------|-------------------|------------|
| Soak test infrastructure | Multi-symbol sustained operation | **High** — this capability IS the trigger |
| Cross-runtime correlation tracing | Debugging multi-symbol event flow | **Medium** — depends on whether issues arise |
| Config parameterization | If operator wants different families per symbol | **Low** — out of scope for CC-01 |

If triggers fire, they are addressed in S120, not in this stage.
