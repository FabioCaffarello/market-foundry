# Vertical Slice 01 — Runtime Activation and Wiring

## Slice Identity

**Name:** `candle-to-paper-order`
**Binding:** `binancef.btcusdt.60`
**Stage:** S109 — End-to-End Implementation

---

## Runtime Port Map

| Runtime   | HTTP Port | Health Endpoints                          | Purpose                                |
|-----------|-----------|-------------------------------------------|----------------------------------------|
| configctl | :8080     | /healthz, /readyz, /statusz, /diagz       | Config lifecycle, activation           |
| gateway   | :8080     | /healthz, /readyz + all query routes      | HTTP query surface                     |
| ingest    | :8082     | /healthz, /readyz, /statusz, /diagz       | Market data capture                    |
| derive    | :8083     | /healthz, /readyz, /statusz, /diagz       | Evidence/signal/decision/strategy/risk/execution derivation |
| store     | :8081     | /healthz, /readyz, /statusz, /diagz       | Read model materialization, query serving |
| execute   | :8084     | /healthz, /readyz, /statusz, /diagz       | Venue order submission                 |

---

## Startup Dependency Chain

```
nats (JetStream enabled)
  └─► configctl (depends: nats healthy)
        └─► ingest (depends: nats + configctl healthy)
  └─► derive (depends: nats healthy)
        └─► store (depends: nats + derive healthy)
        └─► execute (depends: nats + derive healthy)
              └─► gateway (depends: nats + configctl + store healthy)
```

---

## Config Activation Flow

### Step 1: Create Draft

```
POST /configctl/drafts
Body: { YAML/JSON config document }
→ configctl creates draft, publishes DraftCreatedEvent
```

### Step 2: Validate + Compile

```
POST /configctl/drafts/:id/validate
POST /configctl/config-versions/:id/compile
→ configctl validates schema, compiles artifact
```

### Step 3: Activate

```
POST /configctl/config-versions/:id/activate
→ configctl activates config
→ publishes ActivatedEvent
→ publishes IngestionRuntimeChangedEvent
```

### Step 4: Binding Propagation

```
IngestionRuntimeChangedEvent
  → ingest.BindingWatcherActor subscribes
  → queries configctl for active ingestion bindings
  → sends activateBindingMessage to IngestSupervisor
  → IngestSupervisor spawns ExchangeScopeActor (binancef)
  → ExchangeScopeActor spawns WebSocketActor (btcusdt)
  → trades start flowing

IngestionRuntimeChangedEvent
  → derive.BindingWatcherActor subscribes
  → queries configctl for active ingestion bindings
  → sends activateSamplerMessage to DeriveSupervisor
  → DeriveSupervisor spawns SourceScopeActor (binancef)
  → samplers/evaluators/resolvers activated per family
```

---

## Event Pipeline Wiring

### Complete Chain for `binancef.btcusdt.60`

```
1. OBSERVATION
   ingest → WebSocket → Trade
   → publish to: observation.events.market.trade
   Stream: OBSERVATION_EVENTS (6h, 1GB)

2. EVIDENCE (3 families)
   derive → observation consumer → SamplerActor
   → publish to: evidence.events.candle.sampled
   → publish to: evidence.events.tradeburst.sampled
   → publish to: evidence.events.volume.sampled
   Stream: EVIDENCE_EVENTS (72h, 2GB)

3. SIGNAL
   derive → candle → RSISignalSamplerActor
   → publish to: signal.events.rsi.generated
   Stream: SIGNAL_EVENTS (72h, 2GB)

4. DECISION
   derive → signal → RSIOversoldEvaluatorActor
   → publish to: decision.events.rsi_oversold.evaluated
   Stream: DECISION_EVENTS (72h, 2GB)

5. STRATEGY
   derive → decision → MeanReversionEntryResolverActor
   → publish to: strategy.events.mean_reversion_entry.resolved
   Stream: STRATEGY_EVENTS (72h, 2GB)

6. RISK
   derive → strategy → PositionExposureEvaluatorActor
   → publish to: risk.events.position_exposure.evaluated
   Stream: RISK_EVENTS (72h, 2GB)

7. EXECUTION INTENT
   derive → risk → PaperOrderEvaluatorActor
   → checks ExecutionControlKVStore (kill switch)
   → publish to: execution.events.paper_order.submitted
   Stream: EXECUTION_EVENTS (72h, 2GB)

8. EXECUTION FILL
   execute → VenueAdapterActor → paper_simulator
   → publish to: execution.fill.venue_market_order
   Stream: EXECUTION_FILL_EVENTS (72h, 2GB)
```

---

## Store Materialization

### JetStream Consumers (Durable)

| Consumer                                 | Source Stream            | Family                  |
|------------------------------------------|--------------------------|-------------------------|
| derive-observation                       | OBSERVATION_EVENTS       | market trade            |
| store-candle                             | EVIDENCE_EVENTS          | candle                  |
| store-tradeburst                         | EVIDENCE_EVENTS          | tradeburst              |
| store-volume                             | EVIDENCE_EVENTS          | volume                  |
| store-signal-rsi                         | SIGNAL_EVENTS            | rsi                     |
| store-decision-rsi-oversold              | DECISION_EVENTS          | rsi_oversold            |
| store-strategy-mean-reversion-entry      | STRATEGY_EVENTS          | mean_reversion_entry    |
| store-risk-position-exposure             | RISK_EVENTS              | position_exposure       |
| store-execution-paper-order              | EXECUTION_EVENTS         | paper_order             |
| execute-venue-market-order-intake        | EXECUTION_EVENTS         | paper_order (bridge)    |
| store-execution-venue-market-order-fill  | EXECUTION_FILL_EVENTS    | venue_market_order      |

### KV Read Models (Buckets)

| Bucket                                  | Domain     | Family               | Key Pattern                           |
|-----------------------------------------|------------|-----------------------|---------------------------------------|
| CANDLE_LATEST                           | evidence   | candle                | {source}.{symbol}.{timeframe}         |
| CANDLE_HISTORY                          | evidence   | candle                | {source}.{symbol}.{timeframe}         |
| TRADEBURST_LATEST                       | evidence   | tradeburst            | {source}.{symbol}.{timeframe}         |
| VOLUME_LATEST                           | evidence   | volume                | {source}.{symbol}.{timeframe}         |
| SIGNAL_RSI_LATEST                       | signal     | rsi                   | {source}.{symbol}.{timeframe}         |
| DECISION_RSI_OVERSOLD_LATEST            | decision   | rsi_oversold          | {source}.{symbol}.{timeframe}         |
| STRATEGY_MEAN_REVERSION_ENTRY_LATEST    | strategy   | mean_reversion_entry  | {source}.{symbol}.{timeframe}         |
| RISK_POSITION_EXPOSURE_LATEST           | risk       | position_exposure     | {source}.{symbol}.{timeframe}         |
| EXECUTION_PAPER_ORDER_LATEST            | execution  | paper_order           | {source}.{symbol}.{timeframe}         |
| EXECUTION_VENUE_MARKET_ORDER_LATEST     | execution  | venue_market_order    | {source}.{symbol}.{timeframe}         |

---

## Query Surface Wiring

### Gateway → Store (via NATS Request/Reply)

| HTTP Endpoint                                            | NATS Subject                                  | Store Responder          |
|----------------------------------------------------------|-----------------------------------------------|--------------------------|
| GET /evidence/candle/latest?...                          | evidence.query.candle.latest                  | CandleQueryResponder     |
| GET /evidence/candle/history?...                         | evidence.query.candle.history                 | CandleQueryResponder     |
| GET /evidence/tradeburst/latest?...                      | evidence.query.tradeburst.latest              | TradeBurstQueryResponder |
| GET /evidence/volume/latest?...                          | evidence.query.volume.latest                  | VolumeQueryResponder     |
| GET /signal/rsi/latest?...                               | signal.query.rsi.latest                       | SignalQueryResponder     |
| GET /decision/rsi_oversold/latest?...                    | decision.query.rsi_oversold.latest            | DecisionQueryResponder   |
| GET /strategy/mean_reversion_entry/latest?...            | strategy.query.mean_reversion_entry.latest    | StrategyQueryResponder   |
| GET /risk/position_exposure/latest?...                   | risk.query.position_exposure.latest           | RiskQueryResponder       |
| GET /execution/:type/latest?...                          | execution.query.{type}.latest                 | ExecutionQueryResponder  |
| GET /execution/status/latest?...                         | execution.query.status.latest                 | ExecutionQueryResponder  |
| GET /execution/control                                   | execution.control.get                         | ControlGateResponder     |
| PUT /execution/control                                   | execution.control.set                         | ControlGateResponder     |

### Gateway → Configctl (via NATS Request/Reply)

| HTTP Endpoint                                            | NATS Subject                                          |
|----------------------------------------------------------|-------------------------------------------------------|
| POST /configctl/drafts                                   | configctl.control.create_draft                        |
| GET  /configctl/config-versions/:id                      | configctl.control.get_config                          |
| GET  /configctl/config-versions/active                   | configctl.control.get_active                          |
| GET  /configctl/config-versions                          | configctl.control.list_configs                        |
| POST /configctl/drafts/:id/validate                      | configctl.control.validate_draft                      |
| POST /configctl/config-versions/:id/validate             | configctl.control.validate_config                     |
| POST /configctl/config-versions/:id/compile              | configctl.control.compile_config                      |
| POST /configctl/config-versions/:id/activate             | configctl.control.activate_config                     |
| GET  /configctl/runtime-projections                      | configctl.control.list_active_runtime_projections     |
| GET  /configctl/ingestion-bindings                       | configctl.control.list_active_ingestion_bindings      |

---

## Execution Control Gate

The execution pipeline includes a control gate (kill switch) that can halt order submission:

- **KV Store:** `EXECUTION_CONTROL` — singleton key storing gate status
- **Default state:** `active` — orders flow through
- **Halted state:** `halted` — derive's ExecutionPublisherActor skips publishing, logs halted count
- **Query:** `GET /execution/control` returns current gate status
- **Update:** `PUT /execution/control` sets gate status with reason and updater identity

---

## Health Tracking

Each runtime registers health trackers for its pipeline components:

| Runtime | Trackers |
|---------|----------|
| ingest  | observation-publisher |
| derive  | evidence-publisher |
| store   | candle-consumer, candle-projection, tradeburst-consumer, tradeburst-projection, volume-consumer, volume-projection, signal-rsi-consumer, signal-rsi-projection, decision-rsi-oversold-consumer, decision-rsi-oversold-projection, strategy-mean-reversion-entry-consumer, strategy-mean-reversion-entry-projection, risk-position-exposure-consumer, risk-position-exposure-projection, execution-paper-order-consumer, execution-paper-order-projection, execution-venue-market-order-fill-consumer, execution-venue-market-order-fill-projection |
| execute | venue-adapter |

The `/statusz` endpoint reports per-tracker: event count, error count, idle duration, and idle warnings.
The `/diagz` endpoint returns machine-readable diagnostic summary.

---

## Activation Sequence for Slice Validation

1. Start all services: `docker compose up -d`
2. Wait for all healthchecks to pass
3. Create config draft with binding `binancef.btcusdt.60`:
   ```
   POST http://localhost:8080/configctl/drafts
   ```
4. Validate and compile the config
5. Activate the config:
   ```
   POST http://localhost:8080/configctl/config-versions/{id}/activate
   ```
6. Ingest starts capturing trades from Binance Futures WebSocket
7. Derive processes the pipeline (candle → RSI → decision → strategy → risk → paper_order)
8. Store materializes all KV read models
9. Execute processes paper orders through paper simulator
10. Query all endpoints to verify data flow:
    ```
    GET http://localhost:8080/evidence/candle/latest?source=binancef&symbol=btcusdt&timeframe=60
    GET http://localhost:8080/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60
    GET http://localhost:8080/execution/status/latest?source=binancef&symbol=btcusdt&timeframe=60
    ```
