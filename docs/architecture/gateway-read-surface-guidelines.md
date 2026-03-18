# Gateway Read Surface Guidelines

> Rules and principles for how the gateway exposes evidence read models via HTTP.

## Gateway Role

The gateway is a **stateless HTTP-to-NATS translator**. It does not hold domain logic, does not access KV buckets directly, and does not maintain read-side state. All evidence data flows through NATS request/reply to the store binary.

## Principles

### 1. Gateway exposes, store owns

The gateway decides which HTTP endpoints exist. The store decides what data they return. The gateway must never bypass the store's query surface to access projections directly.

### 2. Evidence routes are optional

Evidence endpoints are conditionally registered. If the evidence gateway (NATS client to store) is unavailable at startup, the gateway operates without evidence routes. Configctl routes remain available. This is graceful degradation, not failure.

### 3. One handler per query operation

Each evidence query operation gets its own HTTP handler method. Handlers are thin wrappers: parse parameters, call use case, format response. No business logic in handlers.

### 4. Shared parameter parsing

All evidence queries share the same partition key (`source`, `symbol`, `timeframe`). The shared `parseEvidenceKeyParams()` function extracts and validates these. Family-specific parameters (like `limit`, `since`, `until` for history) are parsed in the handler.

### 5. Family-grouped dependencies

Evidence use cases are grouped in `EvidenceFamilyDeps` struct, not scattered as flat fields. This makes it clear which use cases belong to which family and where new families plug in.

### 6. Readiness is configctl-gated, not evidence-gated

The gateway is ready when configctl is reachable. Evidence availability is probed but does not block readiness. This prevents a store outage from marking the gateway as unready.

## HTTP URL Convention

Evidence endpoints follow this pattern:

```
GET /evidence/{type}/{operation}?source=X&symbol=Y&timeframe=Z[&extra_params]
```

| Segment | Description | Examples |
|---------|-------------|----------|
| `/evidence` | Family prefix — all evidence queries live here | — |
| `/{type}` | Evidence type (plural for collections, singular for entities) | `candles`, `tradeburst`, `volume` |
| `/{operation}` | Query operation | `latest`, `history` |

### Current Endpoints

| Endpoint | Family | Operation | Extra Params |
|----------|--------|-----------|-------------|
| `GET /evidence/candles/latest` | candle | latest | — |
| `GET /evidence/candles/history` | candle | history | `limit`, `since`, `until` |
| `GET /evidence/tradeburst/latest` | tradeburst | latest | — |

### Naming Rules

- Evidence type in URL uses **plural form for collections** (`candles`, not `candle`) when the type name is countable. Use **singular** when the type is a mass noun or summary (`tradeburst`, not `tradebursts`).
- Operations are always lowercase: `latest`, `history`.
- No nested resources: `/evidence/candles/latest` not `/evidence/candles/{id}/latest`.
- Query parameters for partition key: `source`, `symbol`, `timeframe` (never in path).

## Response Format

### Success Response

```json
HTTP 200
{
  "candle": { ... }        // single entity (latest)
  "candles": [ ... ]       // array (history)
  "trade_burst": { ... }   // single entity (latest)
}
```

- Latest queries return a single entity wrapper. The entity is `null` if not found (still 200).
- History queries return an array. Empty array `[]` if no results (still 200).
- No pagination metadata — limit and range are in the request.

### Error Response

```json
HTTP 4xx/5xx
{
  "code": "INVALID_ARGUMENT",
  "message": "timeframe must be a valid integer"
}
```

Uses the `problem.Problem` type for consistent error formatting across all endpoints.

## What the Gateway Must NOT Do

1. **Access KV buckets directly.** All reads go through NATS request/reply to store.
2. **Cache evidence data.** The gateway is stateless. If caching is needed, it enters via PROJECTION_EVENTS (future).
3. **Transform evidence data.** The gateway returns what the store provides. No aggregation, no filtering beyond what the use case supports.
4. **Produce domain events.** The gateway is read-only for evidence. Write operations go only to configctl.
5. **Validate domain invariants.** The gateway validates HTTP input (parameter format). Domain validation (e.g., "timeframe must be a configured value") belongs to the use case layer.
6. **Grow into an API gateway.** No auth middleware, no rate limiting, no request routing beyond its own endpoints. These concerns belong to infrastructure, not the application gateway.

## Adding Endpoints for a New Evidence Family

Follow this checklist when exposing a new evidence type:

1. **Add use case field** to `EvidenceFamilyDeps` in `routes/core.go`
2. **Add handler method** to `EvidenceWebHandler` in `handlers/evidence.go`
3. **Add route block** in `Evidence()` function in `routes/evidence.go`
4. **Add use case creation** in `cmd/gateway/run.go` (inside the `if evGateway != nil` block)
5. **Add route test** in `routes/evidence_test.go`

No changes needed to: core.go's `DefaultRoutes()`, the readiness checker, the gateway actor, or the webserver.
