# Signal Query Surface Guidelines

> Rules and principles for how the gateway exposes signal read models via HTTP.
> Produced in Stage S35. This is a design document — no implementation is included.
> Companion to: `signal-domain-design.md`, `signal-activation-and-ownership.md`, `gateway-read-surface-guidelines.md`, `query-contracts-by-family.md`.

---

## 1. Purpose

This document defines **how signal data becomes queryable** — the full chain from HTTP endpoint to KV bucket, the contracts involved, the naming rules, and the invariants that prevent the gateway from absorbing domain logic.

Signal queries follow the same structural pattern as evidence queries. Where differences exist, they are called out explicitly.

---

## 2. Query Chain

Every signal query follows the same four-layer chain as evidence:

```
HTTP Endpoint → Use Case → NATS Request/Reply → KV Bucket
   (gateway)    (application)    (signal.query.*)    (store)
```

The gateway owns the HTTP layer. The store owns the KV layer. The NATS request/reply subjects are the contract boundary between them.

---

## 3. Relationship Between Gateway, Store, and Signal Projections

### 3.1 Store Is the Projection Authority

Store materializes signal events into KV read models (`SIGNAL_MACD_LATEST`, `SIGNAL_RSI_LATEST`). Store is the **sole reader** of these buckets for query purposes. No other binary accesses signal KV buckets.

### 3.2 Store Serves Signal Queries

`QueryResponderActor` in store is extended to serve `signal.query.*` subjects alongside `evidence.query.*` subjects. Signal queries use a **separate queue group** (`signal.query`) from evidence queries (`evidence.query`), enabling independent scaling.

### 3.3 Gateway Translates HTTP to NATS

Gateway exposes signal endpoints under `/signal/{type}/{operation}`. Each request is translated to a NATS request/reply call to the store. Gateway holds no signal state, no KV connections, no domain logic. It is identical in role to evidence query translation.

### 3.4 Derive Has No Query Role

Derive produces signal events but does not serve queries. It has no involvement in the query surface.

---

## 4. First-Slice Queries (Latest Only)

Phase 1 exposes **latest-only** queries for each signal family. This mirrors the evidence progression (candle started latest-only; history was added in S19/S20).

### 4.1 MACD Latest

| Layer | Value |
|-------|-------|
| **HTTP** | `GET /signal/macd/latest?source=X&symbol=Y&timeframe=Z` |
| **Use Case** | `GetLatestMACDUseCase` |
| **NATS Subject** | `signal.query.macd.latest` |
| **Request Type** | `signal.query.v1.macd_latest_request` |
| **Reply Type** | `signal.query.v1.macd_latest_reply` |
| **Queue Group** | `signal.query` |
| **Server** | store → `QueryResponderActor` |
| **KV Bucket** | `SIGNAL_MACD_LATEST` |
| **Key** | `{source}.{symbol}.{timeframe}` |

**Request contract:**
```go
MACDLatestQuery {
    Source    string  // required
    Symbol   string  // required
    Timeframe int    // required, positive integer (seconds)
}
```

**Reply contract:**
```go
MACDLatestReply {
    Signal *Signal  // nil if not found; includes Value, Metadata, Timestamp, Final
}
```

### 4.2 RSI Latest

| Layer | Value |
|-------|-------|
| **HTTP** | `GET /signal/rsi/latest?source=X&symbol=Y&timeframe=Z` |
| **Use Case** | `GetLatestRSIUseCase` |
| **NATS Subject** | `signal.query.rsi.latest` |
| **Request Type** | `signal.query.v1.rsi_latest_request` |
| **Reply Type** | `signal.query.v1.rsi_latest_reply` |
| **Queue Group** | `signal.query` |
| **Server** | store → `QueryResponderActor` |
| **KV Bucket** | `SIGNAL_RSI_LATEST` |
| **Key** | `{source}.{symbol}.{timeframe}` |

**Request contract:**
```go
RSILatestQuery {
    Source    string  // required
    Symbol   string  // required
    Timeframe int    // required, positive integer (seconds)
}
```

**Reply contract:**
```go
RSILatestReply {
    Signal *Signal  // nil if not found; includes Value, Metadata, Timestamp, Final
}
```

---

## 5. Latest vs History

### 5.1 Latest (First Slice — S36)

Latest queries return the most recent finalized signal for a given partition key (`source/symbol/timeframe`). One KV entry per key, overwritten on each finalized signal. This is the only query surface in Phase 1.

### 5.2 History (Deferred — S37+)

Signal history queries (`/signal/{type}/history`) are **explicitly deferred**. They will be introduced when:
- A concrete consumer (likely the decision domain) requires historical signal lookback, OR
- Backtesting/replay scenarios require signal time-series access.

When history is added, it will follow the same pattern as evidence history:
- Separate KV bucket per type: `SIGNAL_MACD_HISTORY`, `SIGNAL_RSI_HISTORY`.
- Key format: `{source}.{symbol}.{timeframe}.{timestamp_unix}`.
- Query params: `limit`, `since`, `until` — same semantics as `evidence/candles/history`.
- NATS subject: `signal.query.{type}.history`.

No infrastructure (buckets, subjects, routes) for history is created in S36. The door is open but the room is empty.

---

## 6. Naming and Contract Conventions

### 6.1 HTTP URL Pattern

```
GET /signal/{type}/{operation}?source=X&symbol=Y&timeframe=Z[&extra_params]
```

| Segment | Description | Examples |
|---------|-------------|----------|
| `/signal` | Domain prefix — all signal queries live here, separate from `/evidence` | — |
| `/{type}` | Signal type identifier (always singular, lowercase) | `macd`, `rsi` |
| `/{operation}` | Query operation | `latest`, `history` (deferred) |

### 6.2 Naming Rules

- Signal types in URL use **singular lowercase**: `macd`, `rsi` — not `macds`, not `MACD`.
- Operations are always lowercase: `latest`, `history`.
- No nested resources: `/signal/macd/latest`, not `/signal/macd/{id}/latest`.
- Partition key always in query parameters: `source`, `symbol`, `timeframe` (never in path).
- Signal URL prefix is `/signal`, not `/signals`. Consistent with the domain name.

### 6.3 NATS Subject Pattern

```
signal.query.{type}.{operation}
```

Examples:
- `signal.query.macd.latest`
- `signal.query.rsi.latest`
- `signal.query.macd.history` (deferred)

### 6.4 Envelope Type Pattern

```
signal.query.v1.{type}_{operation}_{request|reply}
```

Examples:
- `signal.query.v1.macd_latest_request`
- `signal.query.v1.rsi_latest_reply`

### 6.5 KV Bucket Pattern

```
SIGNAL_{TYPE}_LATEST     (Phase 1)
SIGNAL_{TYPE}_HISTORY    (deferred)
```

Examples:
- `SIGNAL_MACD_LATEST`
- `SIGNAL_RSI_LATEST`

---

## 7. Common Query Parameters

All signal queries share the same three required parameters as evidence queries:

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `source` | string | Required, non-empty (use case validates) | Exchange identifier (e.g., `binancef`) |
| `symbol` | string | Required, non-empty (use case validates) | Trading pair (e.g., `btcusdt`) |
| `timeframe` | int | Required, positive integer (handler validates) | Window duration in seconds (e.g., `300`) |

These mirror the KV key format `{source}.{symbol}.{timeframe}` and the evidence partition key dimensions.

**Shared parameter parsing**: Signal handlers reuse the same `parseSignalKeyParams()` function (or a generalized `parsePartitionKeyParams()`), following the pattern established by `parseEvidenceKeyParams()` in evidence handlers.

---

## 8. Response Format

### 8.1 Success Response (Latest)

```json
HTTP 200
{
  "signal": {
    "type": "macd",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 300,
    "value": "0.00123",
    "metadata": {
      "signal_line": "0.00098",
      "histogram": "0.00025"
    },
    "final": true,
    "timestamp": "2026-03-17T12:00:00Z"
  }
}
```

- Latest queries return a `"signal"` wrapper with the full `Signal` struct.
- If not found, `"signal": null` (still HTTP 200). This matches evidence behavior.
- `metadata` is type-specific and opaque to the gateway. Clients parse by signal type.

### 8.2 Success Response (History — Deferred)

When history is added:
```json
HTTP 200
{
  "signals": [ ... ]
}
```

- Array wrapper, empty `[]` if no results (still HTTP 200).
- Newest-first ordering, matching evidence history convention.

### 8.3 Error Response

```json
HTTP 4xx/5xx
{
  "code": "INVALID_ARGUMENT",
  "message": "timeframe must be a valid integer"
}
```

Uses `problem.Problem` type — same as evidence error responses.

---

## 9. What the Gateway Must NOT Do

These rules are identical to evidence (see `gateway-read-surface-guidelines.md`) and apply equally to signal:

1. **Access signal KV buckets directly.** All reads go through NATS request/reply to store.
2. **Cache signal data.** The gateway is stateless.
3. **Transform signal data.** The gateway returns what the store provides. No aggregation, no filtering, no metadata interpretation.
4. **Produce signal events.** The gateway is read-only for signal. Signal production belongs to derive.
5. **Validate signal domain invariants.** The gateway validates HTTP input (parameter format). Domain validation belongs to the use case layer.
6. **Interpret signal metadata.** The gateway does not parse `metadata` fields. It passes them through as-is. Signal-type-specific interpretation belongs to clients.
7. **Cross-query signal and evidence.** The gateway does not join signal and evidence data. If a client needs both, it makes two requests. Aggregation is a client concern, not a gateway concern.

---

## 10. What the Gateway CAN Do

1. **Register signal routes conditionally.** If the signal gateway (NATS client to store for signal queries) is unavailable at startup, the gateway operates without signal routes. Same graceful degradation as evidence.
2. **Parse and validate HTTP parameters.** `source`, `symbol`, `timeframe` validation at the system boundary.
3. **Return signal data as-is from store.** No transformation, no enrichment.
4. **Use separate dependency grouping.** Signal use cases are grouped in a `SignalFamilyDeps` struct (or equivalent), separate from `EvidenceFamilyDeps`. This maintains clear family separation.

---

## 11. Invariants That Prevent Improper Coupling

| # | Invariant | Enforcement | Violation Example |
|---|-----------|-------------|-------------------|
| I-1 | Gateway never reads signal KV directly | No KV connection in gateway binary for signal buckets | Gateway importing `nats.KeyValue` for signal reads |
| I-2 | Gateway has no signal domain imports | `internal/interfaces/http/` has no imports from `internal/domain/signal/` | Handler importing signal domain types directly |
| I-3 | Signal and evidence queries are independent | Separate NATS queue groups, separate dependency structs, separate route groups | Shared handler or shared use case between signal and evidence |
| I-4 | Store is sole signal query server | Only `QueryResponderActor` subscribes to `signal.query.*` | Gateway subscribing to signal KV watch or creating its own responder |
| I-5 | Signal query contracts are versioned | All request/reply types use `signal.query.v1.*` envelope | Unversioned subjects or breaking changes without version bump |
| I-6 | Signal metadata is opaque to gateway | Gateway does not parse, validate, or transform `metadata` fields | Gateway applying signal-type-specific logic to metadata |
| I-7 | No cross-domain joins in gateway | Gateway does not combine signal + evidence in a single response | Endpoint that returns "candle with MACD" |
| I-8 | Signal routes do not block readiness | Signal availability does not affect gateway readiness probe | Readiness check depending on signal query success |

---

## 12. Adding a Query for a New Signal Family

Follow this checklist when exposing a new signal type (mirrors evidence checklist from `evidence-read-model-guidelines.md`):

1. **Contract** — `{Type}LatestQuery` / `{Type}LatestReply` in `signalclient/contracts.go`
2. **Use case** — `GetLatest{Type}UseCase` in `signalclient/get_latest_{type}.go`
3. **Port method** — `GetLatest{Type}()` on `SignalGateway` interface
4. **NATS adapter** — `GetLatest{Type}()` on `nats.SignalGateway`
5. **Registry spec** — `Signal{Type}Latest ControlSpec` in `SignalRegistry` (or extended `EvidenceRegistry`)
6. **HTTP handler** — `GetLatest{Type}()` on `SignalWebHandler`
7. **Route** — `GET /signal/{type}/latest` in `routes/signal.go`
8. **Deps field** — `GetLatest{Type}` in `SignalFamilyDeps`
9. **Wiring** — use case creation in `cmd/gateway/run.go`

The pattern is fully additive — no existing query contract changes.

---

## 13. Ownership Summary

| Concern | Owner | File/Location |
|---------|-------|---------------|
| Signal HTTP endpoint definition | gateway | `routes/signal.go` (NEW) |
| Signal parameter parsing | gateway | `handlers/signal.go` (NEW) |
| Signal use case (client-side) | application | `signalclient/` (NEW) |
| Signal NATS request encoding | adapters | `nats/signal_gateway.go` (NEW) |
| Signal query serving | store | `QueryResponderActor` (extended) |
| Signal KV read access | store | `QueryResponderActor` (extended) |
| Signal KV write access | store | `SignalProjectionActor[]` |

---

## 14. What Is Deferred

| Topic | Target | Rationale |
|-------|--------|-----------|
| Signal history queries (`/signal/{type}/history`) | S37+ | Start with latest-only. Add when a concrete consumer needs history. |
| History KV buckets (`SIGNAL_{TYPE}_HISTORY`) | S37+ | No bucket creation without a query that reads it. |
| Aggregated signal endpoints (e.g., "all signals for symbol") | Indefinite | Premature API surface. Clients query per-type. |
| Signal-to-evidence cross-query | Indefinite | No joins in gateway. Clients compose. |
| Pagination for signal history | S37+ | Enters with history, follows evidence pagination pattern. |
| Signal WebSocket/streaming | Indefinite | HTTP request/reply first. Streaming is a separate architectural decision. |
| Raccoon-CLI signal query drift rules | S36 | Enters alongside signal query implementation. |
| `SignalRegistry` NATS specs | S36 | Registry specs are implementation, not design. |

---

## 15. Relationship to Existing Documents

| Document | Relationship |
|----------|-------------|
| `signal-domain-design.md` | Defines signal types, events, invariants. This document defines how those types become queryable. |
| `signal-activation-and-ownership.md` | Defines who writes signal events and projections. This document defines who reads them and how. |
| `signal-stream-families.md` | Defines signal stream taxonomy. This document defines query subjects, not event subjects. |
| `gateway-read-surface-guidelines.md` | Defines evidence gateway rules. This document extends those rules to signal with the same principles. |
| `query-contracts-by-family.md` | Defines evidence query contracts. Signal contracts follow identical structure under `signal.query.*` namespace. |
| `evidence-read-model-guidelines.md` | Defines evidence projection checklist. Signal projections follow the same checklist adapted for `internal/domain/signal/`. |
