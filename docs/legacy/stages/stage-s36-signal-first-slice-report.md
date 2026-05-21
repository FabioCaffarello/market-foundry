# Stage S36 — Signal First Slice

> **Status:** Complete
> **Date:** 2026-03-17
> **Objective:** Implement the signal domain end-to-end for RSI (first signal family), including gateway HTTP exposure.
> **Scope:** RSI only. No MACD, no history, no multi-evidence, no signal-to-signal.

---

## Executive Summary

S36 implements the **signal domain first slice** — a complete vertical path from RSI computation in derive through event streaming, store projection, and HTTP query in gateway. This is the third domain layer (`observation → evidence → signal`) with full runtime support.

All ten preconditions identified in S35 are now resolved:
- P-1 through P-8 were implemented in S36.
- P-9 (BindingWatcher) was resolved in S34.
- P-10 (evidence candle operational) has been active since S06.

---

## What Was Implemented

### Domain Layer (`internal/domain/signal/`)

- `Signal` struct with `Type`, `Source`, `Symbol`, `Timeframe`, `Value`, `Metadata`, `Final`, `Timestamp`.
- `Validate()` — field-level validation returning `problem.Problem`.
- `PartitionKey()` — `{source}.{symbol}.{timeframe}` for KV bucket keys.
- `DeduplicationKey()` — unique key for JetStream dedup.
- `SignalGeneratedEvent` — event struct with `EventName()` and `EventMetadata()`.
- Unit tests for `Signal.Validate()`, `PartitionKey()`, `DeduplicationKey()`.

### Application Layer (`internal/application/signal/`)

- `RSISampler` — Wilder's smoothed RSI with configurable period (default 14).
- `AddClose(price, timestamp)` → `(Signal, bool)` — pure, stateful, I/O-free.
- Table-driven tests covering warm-up, boundary, and steady-state scenarios.

### Application Layer (`internal/application/signalclient/`)

- `SignalLatestQuery` — request contract with `Type`, `Source`, `Symbol`, `Timeframe`.
- `SignalLatestReply` — response contract with `*signal.Signal`.
- `GetLatestSignalUseCase` — validates input, delegates to `signalGateway` interface.
- Tests for validation, happy path, nil gateway.

### Adapter Layer (`internal/adapters/nats/`)

- `SignalRegistry` — NATS subject/stream specs for RSI events and queries.
- `LatestSpecByType(signalType)` — dispatches query spec by signal type.
- `SignalPublisher` — publishes `SignalGeneratedEvent` to `SIGNAL_EVENTS` stream.
- `SignalKVStore` — persists latest signal per partition key with monotonicity guard.
- `SignalConsumer` — durable consumer for `signal.events.rsi.generated.>`.
- `SignalGateway` — NATS request/reply client implementing `ports.SignalGateway`.
- `StoreRSISignalConsumer()` — consumer spec for store RSI consumption.

### Actor Layer

- `SignalSamplerActor` (derive) — one per type × symbol × timeframe.
- `SignalPublisherActor` (derive) — one per `SourceScopeActor`, writes to `SIGNAL_EVENTS`.
- `SignalConsumerActor` (store) — consumes RSI events from `SIGNAL_EVENTS`.
- `SignalProjectionActor` (store) — projects RSI to `SIGNAL_RSI_LATEST` KV bucket.
- `QueryResponderActor` (store) — extended to serve `signal.query.rsi.latest`.

### Port Layer (`internal/application/ports/`)

- `SignalGateway` interface — `GetLatestSignal(ctx, query) (reply, problem)`.

### HTTP Layer (`internal/interfaces/http/`)

- `SignalWebHandler` — `GetLatestSignal` handler extracting `:type` path param.
- `Signal()` route registration — `GET /signal/:type/latest`.
- `SignalFamilyDeps` — dependency struct with `HasAny()` guard.
- `DefaultRoutes` extended — conditionally includes signal routes.
- Tests: handler tests (happy, unavailable, missing timeframe, null signal).
- Tests: route tests (registration, DefaultRoutes inclusion/omission).

### Gateway Wiring (`cmd/gateway/`)

- `newSignalGateway()` — factory matching `newEvidenceGateway()` pattern.
- `Run()` — initializes signal gateway, creates use case, injects into `Dependencies.Signal`.

---

## Design Decisions

### 1. Single parameterized route (`/signal/:type/latest`)

Instead of per-type routes (`/signal/rsi/latest`, `/signal/macd/latest`), a single route with `:type` path param dispatches via `SignalRegistry.LatestSpecByType()`. This avoids route proliferation as new signal types are added.

### 2. `Type` field added to `SignalLatestQuery`

The query contract includes the signal type so the gateway adapter can resolve the correct NATS subject. This keeps the HTTP handler stateless — it extracts the type from the path and passes it through.

### 3. Gateway degrades gracefully

Signal gateway initialization is optional. If NATS is unavailable or store is down, `getLatestSignalUseCase` is nil, `SignalFamilyDeps.HasAny()` returns false, and no signal routes are registered. This matches the evidence gateway pattern exactly.

---

## Precondition Resolution

| # | Precondition | Status |
|---|-------------|--------|
| P-1 | `pipeline.signal_families` in settings | **Done** (S36) |
| P-2 | `SIGNAL_EVENTS` stream | **Done** (S36) |
| P-3 | `IsFamilyEnabled` for signal | **Done** (S36) |
| P-4 | SignalFamilyProcessor in derive | **Done** (S36) |
| P-5 | Signal projection in store | **Done** (S36) |
| P-6 | Signal KV buckets | **Done** (S36) |
| P-7 | Signal query subjects | **Done** (S36) |
| P-8 | Signal HTTP routes in gateway | **Done** (S36) |
| P-9 | BindingWatcher wired | **Active** (S34) |
| P-10 | Evidence candle operational | **Active** (S06) |

---

## Test Coverage

| Package | Tests | Status |
|---------|-------|--------|
| `internal/domain/signal` | Validate, PartitionKey, DeduplicationKey | Pass |
| `internal/application/signal` | RSI warm-up, boundary, steady-state | Pass |
| `internal/application/signalclient` | Validation, happy path, nil gateway | Pass |
| `internal/interfaces/http/handlers` | Signal handler: happy, unavailable, missing param, null | Pass |
| `internal/interfaces/http/routes` | Signal route registration, DefaultRoutes with/without signal | Pass |

---

## Intentional Limits

1. **RSI only.** MACD sampler is not implemented. Registry has RSI specs only.
2. **Latest-only projection.** No signal history.
3. **No multi-evidence signals.** RSI consumes candle evidence only.
4. **No signal-to-signal.** Flat dependency graph.
5. **No WebSocket/streaming.** HTTP request/reply only.
6. **No raccoon-cli drift rules.** Signal contract governance deferred.
7. **No MACD registry entries.** `LatestSpecByType("macd")` returns false.

---

## Deferred to S37

- **MACD sampler** — implement `MACDSampler`, register in `SignalRegistry`, add `LatestSpecByType("macd")`, add `SIGNAL_MACD_LATEST` bucket.
- **Signal history projections** — if a consumer demands historical lookback.
- **Per-type domain structs** — if Metadata parsing proves error-prone.
- **Raccoon-CLI signal drift rules** — contract validation for signal domain.
- **Signal expiration events** — `signal_expired` lifecycle.

## Deferred to S38+

- Multi-evidence signals (candle + volume → momentum).
- Signal-to-signal composition.
- Decision domain design.

---

## Source Documents

| Document | Path | Purpose |
|----------|------|---------|
| Signal First Slice | `docs/architecture/signal-first-slice.md` | Precondition tracker and slice summary |
| Signal Family SF-01 Contracts | `docs/architecture/signal-family-01-contracts.md` | RSI event, query, HTTP, KV contracts |
| Signal Domain Design | `docs/architecture/signal-domain-design.md` | Core design (S35) |
| Signal Stream Families | `docs/architecture/signal-stream-families.md` | Family catalog (S35) |
| Signal Activation and Ownership | `docs/architecture/signal-activation-and-ownership.md` | Activation flow (S35) |
| Signal Query Surface Guidelines | `docs/architecture/signal-query-surface-guidelines.md` | Query chain (S35) |
