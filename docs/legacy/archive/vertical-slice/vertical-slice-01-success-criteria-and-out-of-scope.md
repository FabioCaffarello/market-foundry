# Vertical Slice 01 — Success Criteria and Out of Scope

## Purpose

Define unambiguous, testable success criteria and explicit exclusions for vertical slice 01. This document prevents scope creep and provides a clear "done" definition.

---

## Success Criteria

### SC-1: Config-Driven Pipeline Activation

**Test:** Submit a config via `POST /configctl/configs` with binding `binancef.btcusdt`, advance it through validate → compile → activate via gateway HTTP endpoints.

**Pass condition:**
- Config reaches `activated` state
- `IngestionRuntimeChangedEvent` is published to `CONFIGCTL_EVENTS` stream
- `GET /configctl/configs/active` returns the activated config
- `GET /configctl/config-versions` lists the config with correct lifecycle state

### SC-2: Dynamic Binding Propagation

**Test:** After config activation, verify that both ingest and derive react without restart.

**Pass condition:**
- Ingest `BindingWatcherActor` receives the binding event and spawns a `binancef` exchange scope with `btcusdt` WebSocket adapter
- Derive `BindingWatcherActor` receives the binding event and spawns a `binancef` source scope with all configured family samplers
- Both runtimes log activation at INFO level with source/symbol context

### SC-3: Observation Capture

**Test:** After binding activation, verify trades flow from exchange WebSocket to NATS.

**Pass condition:**
- `TradeReceivedEvent` messages appear in `OBSERVATION_EVENTS` stream
- Events are published to `observation.events.market.trade.binancef.btcusdt`
- Ingest `/statusz` shows `evidence-publisher` tracker with event_count > 0

### SC-4: Full Derive Pipeline

**Test:** After trades are flowing, verify the complete derive chain produces events in all 6 domain streams.

**Pass condition (all within 2 candle windows = ~120 seconds for 60s timeframe):**
- `EVIDENCE_EVENTS` contains `CandleSampledEvent` with Final=true
- `SIGNAL_EVENTS` contains `SignalGeneratedEvent` for `rsi`
- `DECISION_EVENTS` contains `DecisionEvaluatedEvent` for `rsi_oversold`
- `STRATEGY_EVENTS` contains `StrategyResolvedEvent` for `mean_reversion_entry`
- `RISK_EVENTS` contains `RiskAssessedEvent` for `position_exposure`
- `EXECUTION_EVENTS` contains `PaperOrderSubmittedEvent` for `paper_order`
- Derive `/statusz` shows all 6 publisher trackers with event_count > 0

### SC-5: Execution Fill

**Test:** After paper order intent is published, verify execute runtime processes it.

**Pass condition:**
- Execute runtime consumes `PaperOrderSubmittedEvent` from `EXECUTION_EVENTS`
- Paper venue adapter produces a fill
- `VenueOrderFilledEvent` appears in `EXECUTION_FILL_EVENTS` stream
- Execute `/statusz` shows `venue-adapter` tracker with event_count > 0

### SC-6: Read Model Materialization

**Test:** After events flow through all streams, verify store projects into KV.

**Pass condition (all 8 latest buckets populated):**
- `CANDLE_LATEST` has key `binancef:btcusdt:60` with valid candle data
- `SIGNAL_RSI_LATEST` has key `binancef:btcusdt:60`
- `DECISION_RSI_OVERSOLD_LATEST` has key `binancef:btcusdt:60`
- `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` has key `binancef:btcusdt:60`
- `RISK_POSITION_EXPOSURE_LATEST` has key `binancef:btcusdt:60`
- `EXECUTION_PAPER_ORDER_LATEST` has key `binancef:btcusdt:60`
- `EXECUTION_VENUE_MARKET_ORDER_LATEST` has key `binancef:btcusdt:60`
- `CANDLE_HISTORY` has at least one key for `binancef:btcusdt:60:*`
- Store `/statusz` shows all projection trackers with event_count > 0 and error_count = 0

### SC-7: Query Surface Completeness

**Test:** After projections are populated, query all endpoints via gateway HTTP.

**Pass condition (all return 200 with valid data):**
- `GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60` → candle with Final=true
- `GET /evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60` → at least 1 candle
- `GET /signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60` → RSI value
- `GET /decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60` → outcome (triggered/not_triggered/insufficient)
- `GET /strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60` → direction
- `GET /risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60` → disposition
- `GET /execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60` → paper order
- `GET /execution/status/latest?source=binancef&symbol=btcusdt&timeframe=60` → composite status with intent + result
- `GET /execution/control` → gate status

### SC-8: Diagnostic Visibility

**Test:** After the pipeline has been running for at least 2 candle windows, verify diagnostic surfaces.

**Pass condition:**
- All 6 runtimes respond 200 on `/healthz`
- All 6 runtimes respond 200 on `/readyz`
- All 6 runtimes respond with JSON on `/statusz` containing expected trackers
- All 6 runtimes respond with JSON on `/diagz` combining readiness and tracker info
- No tracker shows `idle_warning: true` (all are actively processing)
- No tracker shows `error_count > 0` under normal operation

### SC-9: Graceful Lifecycle

**Test:** Start all runtimes, let the pipeline process at least 2 candle windows, then shut down cleanly.

**Pass condition:**
- All runtimes start without error logs at WARN or ERROR level
- All runtimes reach ready state (200 on `/readyz`) within 30 seconds
- SIGTERM triggers graceful shutdown on all runtimes
- No goroutine leaks, no panic, no orphan consumers
- Restart produces clean startup (no stale lock conflicts)

### SC-10: Envelope Integrity

**Test:** Verify envelope contracts are maintained across the full chain.

**Pass condition:**
- All published events have unique, non-empty `ID`
- All events have correct `Type` matching registry definitions
- All events have `Source` matching the publishing runtime
- Request/reply pairs maintain `CorrelationID` → `CausationID` chain
- CBOR encoding round-trips produce identical payloads

---

## Out of Scope

### OS-1: Multi-Symbol / Multi-Source

The slice validates a single binding (`binancef.btcusdt`). Multi-symbol and multi-source scenarios are deferred to a future stress/breadth validation stage. Rationale: adding more bindings tests the same architectural path; the proof value is in the path, not the volume.

### OS-2: Multi-Timeframe

The slice uses a single timeframe (60s). Multi-timeframe behavior (e.g., 60s + 300s candles for the same symbol) is deferred. Rationale: timeframe multiplexing is an orthogonal concern that does not affect the core event routing proof.

### OS-3: Production Venue Connectivity

The slice uses `paper_simulator` only. Connecting to real exchanges (even testnets) for order submission is deferred. Rationale: venue adapter wiring is execution-domain specific and already exercised by the paper simulator.

### OS-4: ClickHouse Projections

The slice uses only NATS KV for read models. ClickHouse-based projections and analytical queries are deferred. Rationale: NATS KV proves the read/write separation pattern; ClickHouse adds an infrastructure dependency that is not needed for architecture validation.

### OS-5: Horizontal Scaling

All runtimes run as single instances. Consumer group behavior, partition-aware processing, and multi-instance coordination are deferred. Rationale: scaling is an operational concern, not an architectural wiring concern.

### OS-6: Schema Evolution

No event schema version migrations are tested. The slice operates on v1 schemas only. Rationale: schema evolution requires a versioning policy that is premature before the first slice proves basic interoperability.

### OS-7: Authentication and Multi-Tenancy

No auth, no RBAC, no tenant isolation. The gateway serves unauthenticated requests. Rationale: security boundaries are a feature layer concern, not an architecture wiring concern.

### OS-8: Automated E2E Test Suite

The slice is validated manually (or via `make smoke`). A comprehensive automated E2E test suite is a future deliverable. The slice may reveal what such a suite needs to cover, which is part of its discovery value.

### OS-9: Performance and Latency Benchmarks

No latency SLAs, no throughput benchmarks. The slice proves correctness, not performance. Rationale: meaningful benchmarks require baseline data from a working pipeline — which is exactly what this slice produces.

### OS-10: Additional Evidence Families

`tradeburst` and `volume` families are not included. They follow the same candle pattern and do not exercise additional architectural concerns. Rationale: including them would increase the surface area without proving new patterns.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| RSI sampler requires candle history that doesn't exist on first window | Medium | Pipeline stalls at signal layer | RSI sampler must handle insufficient data gracefully (emit `insufficient` outcome) |
| Paper venue adapter is a pass-through stub without meaningful logic | Low | Execution layer proves nothing | Verify fills contain realistic field population (price, quantity, side) |
| WebSocket disconnects during candle window | Medium | Incomplete candle data | Verify reconnection logic and candle finalization with partial data |
| NATS KV bucket creation races between runtimes | Low | Projection write fails on startup | Ensure bucket creation is idempotent (create-or-bind pattern) |
| Store query responder not wired for all 8 query types | Medium | Query surface incomplete | Verify all 8 query handlers are registered in store supervisor |

---

## Why This Slice Is the Best Architecture Probe

1. **Exercises every layer**: observation → evidence → signal → decision → strategy → risk → execution → fill. No domain is skipped.
2. **Exercises every runtime**: All 6 binaries participate. No runtime is idle.
3. **Exercises both communication patterns**: JetStream pub/sub (events) and request/reply (queries).
4. **Exercises config-driven activation**: The pipeline cannot start without a valid activated config.
5. **Exercises read/write separation**: Derive writes, store materializes, gateway reads.
6. **Exercises diagnostic surfaces**: Every runtime's health, activity, and diagnostics endpoints are verified.
7. **Minimal surface area**: One binding, one timeframe, one source. Maximum architectural coverage with minimum variables.
8. **Already implemented**: All actors, publishers, consumers, projections, and query handlers exist. The slice is a validation exercise, not a development project.
9. **Exposes real friction**: If any wiring is broken, misconfigured, or mismatched, the slice will surface it immediately — because every link in the chain must work for the final query to return data.
10. **Produces artifacts for the next phase**: A working slice produces baseline data (latency, throughput, error rates) that informs the next wave of decisions.
