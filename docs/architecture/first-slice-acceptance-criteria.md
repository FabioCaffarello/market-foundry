# First Slice Acceptance Criteria — Market Foundry

> Canonical document. Defines the pass/fail criteria for the first vertical slice.
> Designed: 2026-03-16. The slice is complete only when ALL criteria below are met.

---

## 1. Architecture Criteria

### AC-1: Layer Sovereignty

- [ ] `internal/domain/observation/` has **zero imports** from application, adapters, actors, or interfaces layers.
- [ ] `internal/domain/evidence/` has **zero imports** from application, adapters, actors, or interfaces layers.
- [ ] `internal/application/ingest/` imports only from `domain/observation`, `shared/`, and standard library.
- [ ] `internal/application/derive/` imports only from `domain/observation`, `domain/evidence`, `shared/`, and standard library.
- [ ] `internal/adapters/nats/` does **not** import from actors or interfaces layers.
- [ ] `internal/actors/scopes/ingest/` does **not** import directly from domain or application layers — receives ports via injection.
- [ ] `internal/actors/scopes/derive/` does **not** import directly from domain or application layers — receives ports via injection.
- [ ] `make arch-guard` passes with zero violations.

### AC-2: Domain Module Isolation

- [ ] `observation` domain does **not** import from `evidence` domain.
- [ ] `evidence` domain does **not** import from `observation` domain.
- [ ] `configctl` domain does **not** import from `observation` or `evidence` domains.
- [ ] Cross-domain communication happens **exclusively through NATS messages**.

### AC-3: Actor Supervision

- [ ] `ingest` binary has `IngestSupervisor` as the top-level actor.
- [ ] `derive` binary has `DeriveSupervisor` as the top-level actor.
- [ ] All actors are children of a supervisor — no unsupervised goroutines.
- [ ] Actor failures are handled by supervisors (restart or escalate), not by panicking.

### AC-4: No Prohibited Carryovers

- [ ] Zero references to Kafka, quality-service, consumer, emulator, or validator in new code.
- [ ] No `pkg/nats` imports inside actors — actors receive injected ports.
- [ ] No `config.yml` or static config files — runtime config flows through configctl.
- [ ] No code copied from MarketMonkey — all patterns re-implemented natively.

---

## 2. Functional Criteria

### AC-5: Observation Capture (ingest)

- [ ] Ingest binary starts and connects to NATS.
- [ ] Ingest subscribes to `configctl.events.config.ingestion_runtime_changed` via JetStream.
- [ ] When an `IngestionRuntimeChangedEvent` with `change_type: "activated"` and a `binancef` binding is received, ingest spawns a WebSocket connection to Binance Futures.
- [ ] Raw `aggTrade` messages from Binance are normalized into `ObservationTrade` structs.
- [ ] Each trade is published to `observation.events.market.trade.binancef` as `Envelope[TradeReceivedEvent]`.
- [ ] Published messages include `Nats-Msg-Id` for deduplication: `binancef:{trade_id}`.
- [ ] Observation events are verifiable via: `nats sub "observation.events.market.trade.binancef" --count=5` returns valid envelopes.

### AC-6: Evidence Derivation (derive)

- [ ] Derive binary starts and connects to NATS.
- [ ] Derive creates a durable consumer `derive-observation` on `OBSERVATION_EVENTS` stream.
- [ ] Derive consumes observation trades and routes them to `ExchangeScopeActor → SymbolScopeActor → CandleSamplerActor`.
- [ ] CandleSamplerActor accumulates OHLCV data per timeframe window (60s, 300s).
- [ ] When a trade timestamp crosses a window boundary, the candle is finalized (`final: true`) and published.
- [ ] Finalized candles are published to `evidence.events.candle.sampled.binancef.btcusdt.{timeframe}` as `Envelope[CandleSampledEvent]`.
- [ ] Evidence events are verifiable via: `nats sub "evidence.events.candle.sampled.binancef.btcusdt.60" --count=2` returns valid candle envelopes.
- [ ] Candle `OpenTime` = `floor(first_trade_timestamp / timeframe) * timeframe` — deterministic.
- [ ] Candle `CloseTime` = `OpenTime + timeframe_duration`.

### AC-7: Query Path (gateway → derive)

- [ ] Derive responds to `evidence.query.candle.latest` via NATS request/reply.
- [ ] Gateway exposes `GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60`.
- [ ] With active trading, the endpoint returns a valid `EvidenceCandle` JSON response.
- [ ] Without data, the endpoint returns 404 with `problem.Problem` response.
- [ ] With invalid parameters, the endpoint returns 400 with validation issues.

### AC-8: Configuration-Driven Activation

- [ ] Ingest does NOT start capturing data until configctl publishes an `IngestionRuntimeChangedEvent`.
- [ ] When configctl publishes `change_type: "cleared"`, ingest stops the corresponding source.
- [ ] Derive reacts to config events to discover which pipelines/timeframes are active.
- [ ] The full activation flow works end-to-end: `POST /configs/draft → validate → compile → activate → ingest starts → derive processes`.

---

## 3. Contract Criteria

### AC-9: Envelope Compliance

- [ ] Every observation event is `Envelope[TradeReceivedEvent]` with `Kind: "event"`.
- [ ] Every evidence event is `Envelope[CandleSampledEvent]` with `Kind: "event"`.
- [ ] Every query request is `Envelope[CandleLatestQuery]` with `Kind: "request"`.
- [ ] Every query reply is `Envelope[CandleLatestReply]` with `Kind: "reply"`.
- [ ] All envelopes have non-empty `ID`, `Type`, `Source`, `Timestamp`.
- [ ] Evidence candle envelopes include `CausationID` pointing to the triggering observation event.

### AC-10: Subject Compliance

- [ ] All observation subjects follow `observation.events.market.{event_name}.{source}`.
- [ ] All evidence subjects follow `evidence.events.candle.{verb}.{source}.{symbol}.{timeframe}`.
- [ ] All query subjects follow `evidence.query.candle.{operation}`.
- [ ] All subject segments are lowercase.
- [ ] No dots within a segment.
- [ ] `make contract-audit` passes (if raccoon-cli contract-audit supports the new registries).

### AC-11: Type Safety

- [ ] `ObservationTrade.Price` and `Quantity` are `string`, not `float64`.
- [ ] `EvidenceCandle.Open/High/Low/Close/Volume` are `string`, not `float64`.
- [ ] `EvidenceCandle.Timeframe` is `int` (seconds), not `string` or `time.Duration`.
- [ ] All domain event structs implement `events.Event` interface (`EventName()` + `EventMetadata()`).

---

## 4. Stream Infrastructure Criteria

### AC-12: JetStream Streams

- [ ] `OBSERVATION_EVENTS` stream exists with subjects `observation.events.market.>`.
- [ ] `OBSERVATION_EVENTS` has `MaxAge: 6h`, `Storage: File`.
- [ ] `EVIDENCE_EVENTS` stream exists with subjects `evidence.events.candle.>`.
- [ ] `EVIDENCE_EVENTS` has `MaxAge: 72h`, `Storage: File`.
- [ ] Streams are created at binary startup (or via bootstrap), not manually.

### AC-13: Consumer Groups

- [ ] `derive-observation` durable consumer exists on `OBSERVATION_EVENTS`.
- [ ] Consumer uses `AckExplicit` — messages are acknowledged only after successful processing.
- [ ] Consumer tracks delivery count — after `MaxDeliver` failures, message is not retried (dead letter).

---

## 5. Test Criteria

### AC-14: Unit Tests

- [ ] `domain/observation/` has tests for `ObservationTrade` construction and validation.
- [ ] `domain/evidence/` has tests for `EvidenceCandle` construction, OHLCV invariants (high >= open, high >= close, low <= open, low <= close).
- [ ] `application/derive/` has table-driven tests for candle sampling:
  - Single trade → interim candle.
  - Multiple trades in same window → accumulated OHLCV.
  - Trade crossing window boundary → finalized candle + new window opened.
  - Edge case: exact window boundary timestamp.
- [ ] `adapters/exchanges/binancef/` has tests parsing real aggTrade JSON payloads into `ObservationTrade`.
- [ ] `adapters/nats/` has tests verifying `ObservationRegistry` and `EvidenceRegistry` produce correct subjects and types.

### AC-15: Integration Tests

- [ ] Publish `ObservationTrade` envelope → consume from `OBSERVATION_EVENTS` → verify payload (embedded NATS).
- [ ] Publish observation trades → derive produces candle → verify on `EVIDENCE_EVENTS` (embedded NATS).
- [ ] Gateway HTTP request → NATS → derive query responder → HTTP response (full stack).

### AC-16: Quality Gates

- [ ] `make test` passes — all unit and integration tests green.
- [ ] `make check` passes — arch-guard, contract-audit, drift-detect.
- [ ] `make verify` passes — combined test + quality gate.
- [ ] No test uses mocks for NATS — integration tests use embedded NATS server.

---

## 6. Deployment Criteria

### AC-17: Docker Compose

- [ ] `docker-compose.yaml` includes `ingest` and `derive` services.
- [ ] `ingest` depends on `nats` (healthy) and `configctl` (healthy).
- [ ] `derive` depends on `nats` (healthy).
- [ ] `make up` starts all 5 services (nats, configctl, gateway, ingest, derive).
- [ ] `make logs SERVICE=ingest` shows WebSocket connection and trade processing logs.
- [ ] `make logs SERVICE=derive` shows candle sampling and publishing logs.

### AC-18: Config Files

- [ ] `deploy/configs/ingest.jsonc` exists with NATS connection settings.
- [ ] `deploy/configs/derive.jsonc` exists with NATS connection settings.
- [ ] Config files follow the same `AppConfig` schema as existing services.

---

## 7. Observability Criteria

### AC-19: Structured Logging

- [ ] Ingest logs: `source connected`, `trade received`, `trade published`, `source disconnected` — all with structured fields (source, symbol, trade_id).
- [ ] Derive logs: `consumer started`, `trade routed`, `candle sampled`, `candle finalized` — all with structured fields (source, symbol, timeframe).
- [ ] Gateway logs: `query received`, `query replied` — with request correlation ID.
- [ ] All logs use the shared logger from `internal/shared/bootstrap`.

### AC-20: Readiness

- [ ] Ingest reports readiness only after NATS connection is established and BindingWatcherActor is subscribed.
- [ ] Derive reports readiness only after NATS connection is established and ObservationConsumer is running.

---

## 8. Negative Criteria (What Must NOT Happen)

### AC-21: Prohibitions

- [ ] No `float64` for price or volume anywhere in the pipeline.
- [ ] No hardcoded exchange list — ingest reacts to configctl events.
- [ ] No hardcoded symbol list — symbols come from ingestion bindings.
- [ ] No UPPER case in NATS subjects.
- [ ] No raw struct publishing to JetStream — always `Envelope[T]`.
- [ ] No unsupervised goroutines in any binary.
- [ ] No cross-domain imports between observation and evidence.
- [ ] No direct database calls from actors.
- [ ] No Kafka references in any file.
- [ ] No quality-service references in any file.

---

## 9. Summary Checklist

| Category | Criteria | Count |
|----------|----------|-------|
| Architecture | AC-1 to AC-4 | 4 |
| Functional | AC-5 to AC-8 | 4 |
| Contract | AC-9 to AC-11 | 3 |
| Stream Infrastructure | AC-12 to AC-13 | 2 |
| Tests | AC-14 to AC-16 | 3 |
| Deployment | AC-17 to AC-18 | 2 |
| Observability | AC-19 to AC-20 | 2 |
| Prohibitions | AC-21 | 1 |
| **Total** | | **21** |

**The slice is accepted when all 21 criteria are verified.**
