# Binary Boundary and Event-Flow Contract Audit

> **Stage:** S371 — Binary Boundary and Event-Flow Audit
> **Scope:** Canonical pipeline (mean_reversion_entry) across all operational binaries
> **Method:** Static code audit of registry definitions, supervisor wiring, docker-compose topology, and existing architecture docs
> **Status:** Complete

---

## 1. Binary Inventory and Ownership

The multi-binary proof involves **7 operational binaries** plus **NATS** and **ClickHouse** as infrastructure:

| Binary | Process Role | Owns (publishes) | Consumes From |
|--------|-------------|-------------------|---------------|
| **nats** | Message backbone (JetStream + KV) | N/A (infrastructure) | N/A |
| **clickhouse** | Analytical storage | N/A (infrastructure) | N/A |
| **configctl** | Configuration lifecycle authority | `CONFIGCTL_EVENTS` | — |
| **ingest** | External data boundary | `OBSERVATION_EVENTS` | `CONFIGCTL_EVENTS` |
| **derive** | Processing core | `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`, `DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS` | `OBSERVATION_EVENTS`, `CONFIGCTL_EVENTS` |
| **store** | Read model materializer (KV + query) | KV buckets (projections) | All 8 event streams |
| **execute** | Venue order executor | `EXECUTION_FILL_EVENTS` | `EXECUTION_EVENTS`, `STRATEGY_EVENTS` |
| **gateway** | HTTP API translation layer | — | KV via NATS request/reply |
| **writer** | ClickHouse analytical writer | ClickHouse rows | All 8 event streams |

### Ownership Rules (Verified)

1. **Single-writer per stream**: Each JetStream stream has exactly one publishing binary.
2. **configctl** is the sole writer to config state — all other binaries are read-only consumers.
3. **derive** owns the entire analytical pipeline from observation to execution intent.
4. **execute** owns only the venue fill stream (`EXECUTION_FILL_EVENTS`).
5. **store** owns all KV bucket writes — no other binary writes to KV.
6. **gateway** is stateless — reads only via NATS request/reply to store and configctl.

---

## 2. JetStream Stream Contracts

| Stream | Subjects | Owner | MaxAge | MaxBytes | Storage | Consumers |
|--------|----------|-------|--------|----------|---------|-----------|
| `OBSERVATION_EVENTS` | `observation.events.market.>` | ingest | 6h | 256 MB | File | derive |
| `EVIDENCE_EVENTS` | `evidence.events.>` | derive | 72h | 256 MB | File | store, writer |
| `SIGNAL_EVENTS` | `signal.events.>` | derive | 72h | 256 MB | File | store, writer |
| `DECISION_EVENTS` | `decision.events.>` | derive | 72h | 256 MB | File | store, writer |
| `STRATEGY_EVENTS` | `strategy.events.>` | derive | 72h | 256 MB | File | store, writer, execute |
| `RISK_EVENTS` | `risk.events.>` | derive | 72h | 256 MB | File | store, writer |
| `EXECUTION_EVENTS` | `execution.events.>` | derive | 72h | 256 MB | File | store, writer, execute |
| `EXECUTION_FILL_EVENTS` | `execution.fill.>` | execute | 72h | 256 MB | File | store, writer |
| `CONFIGCTL_EVENTS` | `configctl.events.config.>` | configctl | 24h | 256 MB | File | ingest, derive |

### Stream Invariants

- **SI-1**: All streams use `FileStorage` (durability across restart).
- **SI-2**: All event streams except `OBSERVATION_EVENTS` retain 72h — sufficient for replay and debug.
- **SI-3**: `OBSERVATION_EVENTS` retains only 6h — high-volume, ephemeral by design.
- **SI-4**: `CONFIGCTL_EVENTS` retains 24h — control plane, lower volume, sufficient for missed-event recovery.
- **SI-5**: No stream has overlapping subject namespaces — subject prefix isolation is clean.

---

## 3. Consumer Specifications (Canonical Pipeline)

All durable consumers share standardized configuration:
- **AckPolicy**: Explicit
- **AckWait**: 30 seconds
- **MaxDeliver**: 5

### Canonical Pipeline Consumers

| Consumer Name | Binary | Stream | Subject Filter | Purpose |
|---------------|--------|--------|----------------|---------|
| `derive-observation` | derive | `OBSERVATION_EVENTS` | `observation.events.market.trade.>` | Trade intake |
| `store-candle` | store | `EVIDENCE_EVENTS` | `evidence.events.candle.sampled.>` | Candle KV projection |
| `store-signal-rsi` | store | `SIGNAL_EVENTS` | `signal.events.rsi.generated.>` | RSI KV projection |
| `store-decision-rsi-oversold` | store | `DECISION_EVENTS` | `decision.events.rsi_oversold.evaluated.>` | Decision KV projection |
| `store-strategy-mean-reversion-entry` | store | `STRATEGY_EVENTS` | `strategy.events.mean_reversion_entry.resolved.>` | Strategy KV projection |
| `store-risk-position-exposure` | store | `RISK_EVENTS` | `risk.events.position_exposure.assessed.>` | Risk KV projection |
| `store-execution-paper-order` | store | `EXECUTION_EVENTS` | `execution.events.paper_order.submitted.>` | Execution KV projection |
| `store-execution-venue-market-order-fill` | store | `EXECUTION_FILL_EVENTS` | `execution.fill.venue_market_order.>` | Venue fill KV projection |
| `execute-strategy-mean-reversion-entry` | execute | `STRATEGY_EVENTS` | `strategy.events.mean_reversion_entry.resolved.>` | Strategy → execution wiring (S360) |
| `execute-venue-market-order-intake` | execute | `EXECUTION_EVENTS` | `execution.events.paper_order.submitted.>` | **Transitional bridge** (paper mode) |
| `writer-candle` | writer | `EVIDENCE_EVENTS` | `evidence.events.candle.sampled.>` | ClickHouse candle persistence |
| `writer-signal-rsi` | writer | `SIGNAL_EVENTS` | `signal.events.rsi.generated.>` | ClickHouse signal persistence |
| `writer-decision-rsi-oversold` | writer | `DECISION_EVENTS` | `decision.events.rsi_oversold.evaluated.>` | ClickHouse decision persistence |
| `writer-strategy-mean-reversion-entry` | writer | `STRATEGY_EVENTS` | `strategy.events.mean_reversion_entry.resolved.>` | ClickHouse strategy persistence |
| `writer-risk-position-exposure` | writer | `RISK_EVENTS` | `risk.events.position_exposure.assessed.>` | ClickHouse risk persistence |
| `writer-execution-paper-order` | writer | `EXECUTION_EVENTS` | `execution.events.paper_order.submitted.>` | ClickHouse paper execution persistence |
| `writer-execution-venue-fill` | writer | `EXECUTION_FILL_EVENTS` | `execution.fill.venue_market_order.>` | ClickHouse venue fill persistence |
| `ingest-binding-watcher` | ingest | `CONFIGCTL_EVENTS` | `configctl.events.config.ingestion_runtime_changed` | Binding activation |
| `derive-binding-watcher` | derive | `CONFIGCTL_EVENTS` | `configctl.events.config.ingestion_runtime_changed` | Binding activation |

---

## 4. Envelope and Payload Contract

All NATS messages use the canonical `Envelope[T]` wrapper:

```
Envelope[T] {
    ID              string              // UUID
    Kind            Kind                // "event" | "command" | "request" | "reply"
    Type            string              // e.g. "signal.events.v1.rsi_generated"
    Source          string              // Publishing service identifier
    Subject         string              // NATS subject path
    CorrelationID   string              // End-to-end trace
    CausationID     string              // Parent event ID
    ReplyTo         string              // For request/reply
    ContentType     string              // "application/cbor"
    Timestamp       time.Time           // UTC
    Headers         map[string]string
    Payload         T                   // Domain event
    Problem         *problem.Problem    // Error (replies only)
}
```

### Payload Contracts (Canonical Pipeline)

| Event Type | Payload | Key Fields |
|-----------|---------|------------|
| `observation.events.v1.trade_received` | `TradeReceivedEvent` | Metadata, ObservationTrade (Source, Symbol, Price, Quantity, TradeID) |
| `evidence.events.v1.candle_sampled` | `CandleSampledEvent` | Metadata, EvidenceCandle (OHLCV, Timeframe, Final) |
| `signal.events.v1.rsi_generated` | `SignalSampledEvent` | Metadata, Signal (Type, Value, Final) |
| `decision.events.v1.rsi_oversold_evaluated` | `DecisionEvaluatedEvent` | Metadata, Decision (Outcome, Severity, Confidence, SignalInputs) |
| `strategy.events.v1.mean_reversion_entry_resolved` | `StrategyResolvedEvent` | Metadata, Strategy (Direction, Confidence, Decisions[], Parameters) |
| `risk.events.v1.position_exposure_assessed` | `RiskEvaluatedEvent` | Metadata, RiskAssessment (Disposition, Constraints) |
| `execution.events.v1.paper_order_submitted` | `PaperOrderSubmittedEvent` | Metadata, ExecutionIntent (Side, Quantity, PriceType, RiskInput) |
| `execution.fill.v1.venue_market_order_filled` | `VenueOrderFilledEvent` | Metadata, ExecutionIntent, VenueOrderID |

### Encoding Contract

- **Codec**: CBOR (`github.com/fxamacker/cbor/v2`)
- **Deduplication**: JetStream `MsgID` set to deterministic key (e.g., `strat:mean_reversion_entry:{source}:{symbol}:{timeframe}:{unix_ts}`)
- **Content-Type**: `application/cbor`

---

## 5. Correlation and Causation Chain

The canonical pipeline preserves a full correlation chain across binary boundaries:

```
[ingest binary]
  TradeReceivedEvent
    .Metadata.ID = UUID_1
    .Metadata.CorrelationID = CORR_1 (originated or propagated)
        │
        ▼ (NATS: OBSERVATION_EVENTS)
[derive binary]
  CandleSampledEvent
    .Metadata.CorrelationID = CORR_1 (preserved)
    .Metadata.CausationID = UUID_1 (parent)
    .Metadata.ID = UUID_2
        │
  RSI SignalSampledEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_2
    .Metadata.ID = UUID_3
        │
  RSIOversold DecisionEvaluatedEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_3
    .Metadata.ID = UUID_4
        │
  MeanReversionEntry StrategyResolvedEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_4
    .Metadata.ID = UUID_5
        │
  PositionExposure RiskEvaluatedEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_5
    .Metadata.ID = UUID_6
        │
  PaperOrderSubmittedEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_6
    .Metadata.ID = UUID_7
        │
        ▼ (NATS: EXECUTION_EVENTS)
[execute binary]
  VenueOrderFilledEvent
    .Metadata.CorrelationID = CORR_1
    .Metadata.CausationID = UUID_7
    .Metadata.ID = UUID_8
```

### Chain Invariants

- **CI-1**: CorrelationID is immutable once set — never regenerated mid-pipeline.
- **CI-2**: CausationID always points to the immediately preceding event's Metadata.ID.
- **CI-3**: Chain spans 3 binary boundaries: ingest → derive, derive → execute, derive → store.
- **CI-4**: Store and writer receive the same events with the same correlation/causation — no fan-out mutation.

---

## 6. KV Bucket Contracts (Store Binary)

| Bucket | Key Pattern | Value Type | Source Stream | Monotonicity |
|--------|------------|------------|---------------|-------------|
| `EVIDENCE_CANDLE_LATEST` | `{source}.{symbol}.{timeframe}` | `EvidenceCandle` (JSON) | `EVIDENCE_EVENTS` | Timestamp |
| `SIGNAL_RSI_LATEST` | `{source}.{symbol}.{timeframe}` | `Signal` (JSON) | `SIGNAL_EVENTS` | Timestamp |
| `DECISION_RSI_OVERSOLD_LATEST` | `{source}.{symbol}.{timeframe}` | `Decision` (JSON) | `DECISION_EVENTS` | Timestamp |
| `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | `{source}.{symbol}.{timeframe}` | `Strategy` (JSON) | `STRATEGY_EVENTS` | Timestamp |
| `RISK_POSITION_EXPOSURE_LATEST` | `{source}.{symbol}.{timeframe}` | `RiskAssessment` (JSON) | `RISK_EVENTS` | Timestamp |
| `EXECUTION_PAPER_ORDER_LATEST` | `{source}.{symbol}.{timeframe}` | `ExecutionIntent` (JSON) | `EXECUTION_EVENTS` | Timestamp |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `{source}.{symbol}.{timeframe}` | `ExecutionIntent` (JSON) | `EXECUTION_FILL_EVENTS` | Timestamp |

### KV Invariants

- **KV-1**: Store binary is the sole writer to all KV buckets — no other binary writes.
- **KV-2**: Monotonicity guard rejects stale writes (timestamp < persisted).
- **KV-3**: `Final == true` gate on strategy projections — non-final events are dropped.
- **KV-4**: Validation gate (`strategy.Validate()`) applied before KV write.
- **KV-5**: Event-level metadata (correlation_id, causation_id) is **NOT** persisted in KV — documented gap, mitigated by ClickHouse and NATS stream retention.

---

## 7. Request/Reply Contracts (Gateway ↔ Store)

| Subject | Request Type | Reply Type | Served By | Used By |
|---------|-------------|-----------|-----------|---------|
| `strategy.query.mean_reversion_entry.latest` | Query{source, symbol, timeframe} | Strategy | store | gateway |
| `evidence.query.candle.latest` | Query{source, symbol, timeframe} | EvidenceCandle | store | gateway |
| `signal.query.rsi.latest` | Query{source, symbol, timeframe} | Signal | store | gateway |
| `execution.query.paper_order.latest` | Query{source, symbol, timeframe} | ExecutionIntent | store | gateway |
| `execution.query.venue_market_order.latest` | Query{source, symbol, timeframe} | ExecutionIntent | store | gateway |
| `execution.query.status.latest` | Query{source, symbol, timeframe} | CompositeStatus | store + execute | gateway |
| `execution.control.get` | GetRequest | GateState | execute | gateway |
| `execution.control.set` | SetRequest{state} | GateState | execute | gateway |
| `execution.activation.surface` | SurfaceRequest | ActivationSurface | execute | gateway |
| `configctl.control.*` | various | various | configctl | gateway |

### Request/Reply Invariants

- **RR-1**: All use QueueGroup for load-balanced consumption.
- **RR-2**: All use CBOR `Envelope[T]` encoding with CorrelationID.
- **RR-3**: Timeout is configurable per client (default varies by binary).
- **RR-4**: Error responses use `Problem` in Envelope — no raw error strings.

---

## 8. Docker Compose Dependency Graph

```
nats ─────────────────────────────────────┐
  │                                        │
  ├─── configctl ────────────┐             │
  │       │                  │             │
  │       ├─── ingest        │             │
  │       │                  │             │
  │       ├─── gateway ──── store ── derive│
  │       │                               │
  │       └────────────────────────────────┘
  │
  ├─── derive
  │       │
  │       ├─── store
  │       └─── execute
  │
  └─── clickhouse
          │
          └─── writer
```

### Dependency Invariants

- **DI-1**: `nats` is the root dependency — all Go binaries depend on it.
- **DI-2**: `configctl` depends only on `nats` — it is the first Go binary to start.
- **DI-3**: `ingest` depends on `nats` + `configctl` — needs config to bind exchanges.
- **DI-4**: `derive` depends only on `nats` — does NOT depend on `configctl` for startup (binding watcher connects asynchronously).
- **DI-5**: `store` depends on `nats` + `derive` — needs derive healthy to ensure streams exist.
- **DI-6**: `execute` depends on `nats` + `derive` — same rationale as store.
- **DI-7**: `gateway` depends on `nats` + `configctl` + `store` — needs all query backends ready.
- **DI-8**: `writer` depends on `nats` + `clickhouse` — isolated analytical path.

### Port Allocation

| Binary | HTTP Port | Purpose |
|--------|-----------|---------|
| nats | 4222, 8222 | Client, monitoring |
| configctl | 8080 (internal) | Health |
| gateway | 8080 (external) | Public API |
| store | 8081 | Health |
| ingest | 8082 | Health |
| derive | 8083 | Health |
| execute | 8084 | Health |
| writer | 8085 | Health |
| clickhouse | 8123, 9000 | HTTP, native |

---

## 9. Health Contract Across Binaries

All Go binaries implement the standard health server:

| Endpoint | Semantics | Contract |
|----------|-----------|---------|
| `/healthz` | Liveness | Always 200 (no business logic) |
| `/readyz` | Readiness | 200 if all `ReadinessCheck` pass; 503 otherwise |
| `/statusz` | Activity | Tracker metrics (event counts, errors, idle) |

### Readiness Checks per Binary

| Binary | Readiness Checks |
|--------|-----------------|
| configctl | NATS connectivity |
| ingest | NATS connectivity |
| derive | NATS connectivity |
| store | NATS connectivity |
| execute | NATS connectivity |
| writer | NATS connectivity |
| gateway | configctl reachable, evidence store reachable (via HTTP integration) |

---

## 10. Audit Findings Summary

### Boundaries: Clean

All binary boundaries are enforced through:
1. **NATS subjects** — no direct function calls between binaries.
2. **Stream ownership** — single-writer per stream.
3. **KV ownership** — store is sole writer to all buckets.
4. **Layer sovereignty** — enforced by `raccoon-cli arch-guard`.
5. **Domain isolation** — domains do not import each other.

### Contracts: Consistent

1. All consumers use standardized AckWait/MaxDeliver configuration.
2. All events use `Envelope[T]` with CBOR encoding.
3. All streams use FileStorage with documented retention.
4. Correlation/causation chain is fully propagated.

### Gaps Identified

1. **Event metadata not in KV** — documented, mitigated by ClickHouse + stream retention.
2. **Transitional bridge in execute** — paper mode intake subscribes to paper_order subjects instead of venue-specific subjects.
3. **derive → configctl dependency is soft** — binding watcher is async; derive starts without configctl and waits for events. This is correct but compose does not enforce it.
