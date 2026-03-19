# Vertical Slice 01 — Contracts, Events, and Read Models

## Purpose

Enumerate every contract, event, stream, subject, KV bucket, and query surface the vertical slice must exercise. This document serves as a checklist: if any item listed here does not work end-to-end, the slice has exposed a real architectural issue.

---

## 1. Configuration Contracts

### Config Lifecycle (configctl)

| Operation | Subject | Request Type | Reply Type |
|-----------|---------|-------------|------------|
| Create draft | `configctl.control.create_draft` | `configctl.command.create_draft` | `configctl.reply.create_draft` |
| Validate draft | `configctl.control.validate_draft` | `configctl.command.validate_draft` | `configctl.reply.validate_draft` |
| Validate config | `configctl.control.validate_config` | `configctl.command.validate_config` | `configctl.reply.validate_config` |
| Compile config | `configctl.control.compile_config` | `configctl.command.compile_config` | `configctl.reply.compile_config` |
| Activate config | `configctl.control.activate_config` | `configctl.command.activate_config` | `configctl.reply.activate_config` |
| Get active | `configctl.control.get_active` | `configctl.query.get_active` | `configctl.reply.get_active` |
| Get config | `configctl.control.get_config` | `configctl.query.get_config` | `configctl.reply.get_config` |
| List configs | `configctl.control.list_configs` | `configctl.query.list_configs` | `configctl.reply.list_configs` |
| List bindings | `configctl.control.list_active_ingestion_bindings` | `configctl.query.list_active_ingestion_bindings` | `configctl.reply.list_active_ingestion_bindings` |
| List projections | `configctl.control.list_active_runtime_projections` | `configctl.query.list_active_runtime_projections` | `configctl.reply.list_active_runtime_projections` |

### Config Events (published by configctl)

| Event | Subject | Stream |
|-------|---------|--------|
| `DraftCreatedEvent` | `configctl.events.config.draft_created.{id}` | `CONFIGCTL_EVENTS` |
| `ConfigValidatedEvent` | `configctl.events.config.validated.{id}` | `CONFIGCTL_EVENTS` |
| `ConfigCompiledEvent` | `configctl.events.config.compiled.{id}` | `CONFIGCTL_EVENTS` |
| `ConfigActivatedEvent` | `configctl.events.config.activated.{id}` | `CONFIGCTL_EVENTS` |
| `IngestionRuntimeChangedEvent` | `configctl.events.config.ingestion_runtime_changed` | `CONFIGCTL_EVENTS` |

**Slice requirement:** The full lifecycle must be exercisable via gateway HTTP endpoints. The `IngestionRuntimeChangedEvent` must reach both ingest and derive binding watchers.

---

## 2. Domain Events

### Event Flow (published by the indicated runtime)

| # | Event | Published By | Subject Pattern | Stream |
|---|-------|-------------|-----------------|--------|
| 1 | `TradeReceivedEvent` | ingest | `observation.events.market.trade.{source}.{symbol}` | `OBSERVATION_EVENTS` |
| 2 | `CandleSampledEvent` | derive | `evidence.events.candle.sampled.{source}.{symbol}.{timeframe}` | `EVIDENCE_EVENTS` |
| 3 | `SignalGeneratedEvent` | derive | `signal.events.rsi.generated.{source}.{symbol}.{timeframe}` | `SIGNAL_EVENTS` |
| 4 | `DecisionEvaluatedEvent` | derive | `decision.events.rsi_oversold.evaluated.{source}.{symbol}.{timeframe}` | `DECISION_EVENTS` |
| 5 | `StrategyResolvedEvent` | derive | `strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}` | `STRATEGY_EVENTS` |
| 6 | `RiskAssessedEvent` | derive | `risk.events.position_exposure.assessed.{source}.{symbol}.{timeframe}` | `RISK_EVENTS` |
| 7 | `PaperOrderSubmittedEvent` | derive | `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}` | `EXECUTION_EVENTS` |
| 8 | `VenueOrderFilledEvent` | execute | `execution.fill.venue_market_order.{source}.{symbol}.{timeframe}` | `EXECUTION_FILL_EVENTS` |

### Concrete Subjects for Slice Binding (`binancef.btcusdt.60`)

```
observation.events.market.trade.binancef.btcusdt
evidence.events.candle.sampled.binancef.btcusdt.60
signal.events.rsi.generated.binancef.btcusdt.60
decision.events.rsi_oversold.evaluated.binancef.btcusdt.60
strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60
risk.events.position_exposure.assessed.binancef.btcusdt.60
execution.events.paper_order.submitted.binancef.btcusdt.60
execution.fill.venue_market_order.binancef.btcusdt.60
```

---

## 3. JetStream Streams

| Stream | Subject Filter | Max Age | Max Bytes | Consumed By |
|--------|---------------|---------|-----------|-------------|
| `OBSERVATION_EVENTS` | `observation.events.market.>` | 6h | 1GB | derive |
| `EVIDENCE_EVENTS` | `evidence.events.>` | 72h | 2GB | store |
| `SIGNAL_EVENTS` | `signal.events.>` | 72h | 2GB | store |
| `DECISION_EVENTS` | `decision.events.>` | 72h | 2GB | store |
| `STRATEGY_EVENTS` | `strategy.events.>` | 72h | 2GB | store |
| `RISK_EVENTS` | `risk.events.>` | 72h | 2GB | store |
| `EXECUTION_EVENTS` | `execution.events.>` | 72h | 2GB | store, execute |
| `EXECUTION_FILL_EVENTS` | `execution.fill.>` | 72h | 2GB | store |
| `CONFIGCTL_EVENTS` | `configctl.events.config.>` | 24h | 256MB | ingest, derive |

---

## 4. Durable Consumers

| Consumer Name | Stream | Filter Subject | Runtime |
|---------------|--------|---------------|---------|
| `derive-observation` | `OBSERVATION_EVENTS` | `observation.events.market.trade.>` | derive |
| `store-candle` | `EVIDENCE_EVENTS` | `evidence.events.candle.sampled.>` | store |
| `store-signal-rsi` | `SIGNAL_EVENTS` | `signal.events.rsi.generated.>` | store |
| `store-decision-rsi-oversold` | `DECISION_EVENTS` | `decision.events.rsi_oversold.evaluated.>` | store |
| `store-strategy-mean-reversion-entry` | `STRATEGY_EVENTS` | `strategy.events.mean_reversion_entry.resolved.>` | store |
| `store-risk-position-exposure` | `RISK_EVENTS` | `risk.events.position_exposure.assessed.>` | store |
| `store-execution-paper-order` | `EXECUTION_EVENTS` | `execution.events.paper_order.submitted.>` | store |
| `store-execution-venue-market-order-fill` | `EXECUTION_FILL_EVENTS` | `execution.fill.venue_market_order.>` | store |
| `execute-venue-market-order-intake` | `EXECUTION_EVENTS` | `execution.events.paper_order.submitted.>` | execute |
| `ingest-binding-watcher` | `CONFIGCTL_EVENTS` | `configctl.events.config.ingestion_runtime_changed` | ingest |
| `derive-binding-watcher` | `CONFIGCTL_EVENTS` | `configctl.events.config.ingestion_runtime_changed` | derive |

---

## 5. Read Models (NATS KV Buckets)

### Latest Buckets (one entry per partition key)

| Bucket | Key Format | Projected From | Validation |
|--------|-----------|----------------|------------|
| `CANDLE_LATEST` | `binancef:btcusdt:60` | `CandleSampledEvent` (Final=true only) | Monotonicity guard on OpenTime |
| `SIGNAL_RSI_LATEST` | `binancef:btcusdt:60` | `SignalGeneratedEvent` | — |
| `DECISION_RSI_OVERSOLD_LATEST` | `binancef:btcusdt:60` | `DecisionEvaluatedEvent` | — |
| `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | `binancef:btcusdt:60` | `StrategyResolvedEvent` | — |
| `RISK_POSITION_EXPOSURE_LATEST` | `binancef:btcusdt:60` | `RiskAssessedEvent` | — |
| `EXECUTION_PAPER_ORDER_LATEST` | `binancef:btcusdt:60` | `PaperOrderSubmittedEvent` | — |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `binancef:btcusdt:60` | `VenueOrderFilledEvent` | — |
| `EXECUTION_CONTROL` | `global` | `SetExecutionControlCommand` | — |

### History Buckets

| Bucket | Key Format | Projected From |
|--------|-----------|----------------|
| `CANDLE_HISTORY` | `binancef:btcusdt:60:{open_time}` | `CandleSampledEvent` (Final=true only) |

---

## 6. Query Surfaces (Gateway HTTP → Store NATS)

### Evidence Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /evidence/candles/latest` | `store.query.evidence.candle_latest` | source, symbol, timeframe |
| `GET /evidence/candles/history` | `store.query.evidence.candle_history` | source, symbol, timeframe, since?, until?, limit? |

### Signal Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /signal/rsi/latest` | `store.query.signal.rsi_latest` | source, symbol, timeframe |

### Decision Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /decision/rsi_oversold/latest` | `store.query.decision.rsi_oversold_latest` | source, symbol, timeframe |

### Strategy Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /strategy/mean_reversion_entry/latest` | `store.query.strategy.mean_reversion_entry_latest` | source, symbol, timeframe |

### Risk Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /risk/position_exposure/latest` | `store.query.risk.position_exposure_latest` | source, symbol, timeframe |

### Execution Queries

| HTTP Endpoint | NATS Subject | Query Params |
|--------------|-------------|--------------|
| `GET /execution/paper_order/latest` | `store.query.execution.paper_order_latest` | source, symbol, timeframe |
| `GET /execution/venue_market_order/latest` | `store.query.execution.venue_market_order_latest` | source, symbol, timeframe |
| `GET /execution/status/latest` | `store.query.execution.status_latest` | source, symbol, timeframe |
| `GET /execution/control` | `store.query.execution.control_get` | — |
| `PUT /execution/control` | `store.query.execution.control_set` | status, reason, updated_by |

### Configctl Queries

| HTTP Endpoint | NATS Subject |
|--------------|-------------|
| `POST /configctl/configs` | `configctl.control.create_draft` |
| `GET /configctl/config-versions` | `configctl.control.list_configs` |
| `GET /configctl/config-versions/:id` | `configctl.control.get_config` |
| `GET /configctl/configs/active` | `configctl.control.get_active` |
| `POST /configctl/configs/validate` | `configctl.control.validate_draft` |
| `POST /configctl/config-versions/:id/validate` | `configctl.control.validate_config` |
| `POST /configctl/config-versions/:id/compile` | `configctl.control.compile_config` |
| `POST /configctl/config-versions/:id/activate` | `configctl.control.activate_config` |

---

## 7. Diagnostic Surfaces

### Health Endpoints (all 6 runtimes)

| Endpoint | Purpose | Expected Behavior |
|----------|---------|-------------------|
| `GET /healthz` | Liveness | Always 200 |
| `GET /readyz` | Readiness | 200 when NATS connected and dependencies available |
| `GET /statusz` | Activity | JSON with tracker stats, event counts, idle warnings |
| `GET /diagz` | Combined | Readiness checks + tracker overview |

### Key Trackers to Validate

| Runtime | Tracker | Expected |
|---------|---------|----------|
| ingest | `evidence-publisher` | event_count > 0 after first trade |
| derive | `evidence-publisher`, `signal-publisher`, `decision-publisher`, `strategy-publisher`, `risk-publisher`, `execution-publisher` | All event_count > 0 after first candle finalization |
| store | `candle-projection`, `signal-rsi-projection`, `decision-rsi-oversold-projection`, `strategy-mean-reversion-entry-projection`, `risk-position-exposure-projection`, `execution-paper-order-projection`, `execution-venue-market-order-projection` | All event_count > 0 after first projection write |
| execute | `venue-adapter` | event_count > 0 after first intent processed |

---

## 8. Envelope Contract

All messages use `Envelope[T]` with CBOR encoding.

### Required Envelope Fields

| Field | Requirement |
|-------|-------------|
| `ID` | Unique per message (16-byte hex or timestamp-based) |
| `Kind` | `event` for domain events, `request`/`reply` for queries, `command` for mutations |
| `Type` | Matches registry spec (e.g., `evidence.events.v1.candle_sampled`) |
| `Source` | Runtime name (e.g., `derive`, `ingest`, `store`) |
| `Subject` | NATS subject for routing |
| `ContentType` | `application/cbor` |
| `Timestamp` | UTC |
| `CorrelationID` | Propagated through request/reply chains |
| `CausationID` | Links reply to triggering request |

### Slice Validation

The slice must verify that:
- Every published event has a valid, unique `ID`
- `CorrelationID` flows from gateway request through store reply
- `CausationID` in replies points to the request `ID`
- `Type` strings match registry definitions exactly
- CBOR encoding/decoding round-trips without data loss
