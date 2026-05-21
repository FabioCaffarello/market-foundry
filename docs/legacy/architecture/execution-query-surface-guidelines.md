# Execution — Query Surface Guidelines

> Query surface specification for the `execution` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S69

---

## 1. Four-Layer Query Chain

```
(1) HTTP Endpoint     →  (2) Use Case          →  (3) NATS Request/Reply  →  (4) KV Bucket
    (gateway)              (executionclient)         (QueryResponderActor)      (EXECUTION_*_LATEST)
```

| Layer | Component | Responsibility |
|-------|-----------|----------------|
| 1 | HTTP handler (gateway) | Parse HTTP request, validate parameters, call use case, format JSON response |
| 2 | Use case (executionclient) | Validate query contract, delegate to gateway port, return typed result |
| 3 | NATS responder (store) | Decode request envelope, read KV bucket, encode reply envelope |
| 4 | KV bucket (store) | Store latest execution intent per partition key |

---

## 2. Phase 1 — Latest Only

### HTTP Endpoint

```
GET /execution/{type}/latest?source={source}&symbol={symbol}&timeframe={timeframe}
```

Phase 1 type: `paper_order`

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `type` | YES (path) | Execution family type (e.g., "paper_order") |
| `source` | YES (query) | Data source identifier (e.g., "binancef") |
| `symbol` | YES (query) | Trading pair (e.g., "btcusdt") |
| `timeframe` | YES (query) | Period in seconds (e.g., "60") |

### Response: 200 OK — Intent Found

```json
{
  "execution_intent": {
    "type": "paper_order",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "side": "buy",
    "quantity": "0.0180",
    "status": "submitted",
    "risk": {
      "type": "position_exposure",
      "disposition": "approved",
      "confidence": "0.8075",
      "timeframe": 60
    },
    "parameters": {
      "risk_type": "position_exposure",
      "risk_disposition": "approved",
      "strategy_direction": "long",
      "strategy_confidence": "0.8500"
    },
    "metadata": {},
    "final": true,
    "timestamp": "2026-03-18T14:30:00Z"
  }
}
```

### Response: 200 OK — No-Action Intent (Flat/Rejected)

```json
{
  "execution_intent": {
    "type": "paper_order",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "side": "none",
    "quantity": "0",
    "status": "submitted",
    "risk": {
      "type": "position_exposure",
      "disposition": "rejected",
      "confidence": "0.0000",
      "timeframe": 60
    },
    "parameters": {
      "risk_type": "position_exposure",
      "risk_disposition": "rejected",
      "strategy_direction": "long",
      "strategy_confidence": "0.8500"
    },
    "metadata": {},
    "final": true,
    "timestamp": "2026-03-18T14:30:00Z"
  }
}
```

### Response: 200 OK — No Intent Yet

```json
{
  "execution_intent": null
}
```

The `execution_intent` field is always present. It is `null` when no intent has been materialized for the given partition.

### Response: 400 Bad Request

```json
{
  "type": "validation_failed",
  "title": "invalid query parameters",
  "detail": "timeframe is required and must be a positive integer"
}
```

### Response: 503 Service Unavailable

```json
{
  "type": "service_unavailable",
  "title": "execution query service unavailable",
  "detail": "NATS connection to store is not available"
}
```

---

## 3. NATS Subjects

| Operation | NATS Subject |
|-----------|-------------|
| Latest query | `execution.query.paper_order.latest` |
| History query | `execution.query.paper_order.history` (deferred) |

Queue group: `execution.query`

---

## 4. Envelope Types

| Direction | Envelope Type |
|-----------|--------------|
| Request (gateway → store) | `execution.query.v1.paper_order_latest_request` |
| Reply (store → gateway) | `execution.query.v1.paper_order_latest_reply` |

### Request Payload

```json
{
  "source": "binancef",
  "symbol": "btcusdt",
  "timeframe": 60
}
```

### Reply Payload

```json
{
  "found": true,
  "execution_intent": {
    "type": "paper_order",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "side": "buy",
    "quantity": "0.0180",
    "status": "submitted",
    "risk": { ... },
    "parameters": { ... },
    "metadata": {},
    "final": true,
    "timestamp": "2026-03-18T14:30:00Z"
  }
}
```

When no intent exists: `{ "found": false, "execution_intent": null }`

---

## 5. Gateway Rules

### MUST

1. Register execution routes conditionally (only if `ExecutionGateway` is available)
2. Parse `type` path parameter and validate against known types
3. Parse and validate `source`, `symbol`, `timeframe` query parameters
4. Forward validated query to `GetLatestExecutionUseCase`
5. Propagate `X-Correlation-ID` header to use case layer

### MUST NOT

1. Access execution KV buckets directly
2. Cache execution intents
3. Transform or enrich execution data before returning
4. Validate execution domain rules (side, quantity, status semantics)
5. Interpret execution intent meaning (e.g., filter by side)
6. Join execution data with risk, strategy, or other domain data
7. Produce execution events
8. Make venue API calls
9. Track position or P&L state
10. Apply business logic to execution intents
11. Filter, aggregate, or sort execution intents

---

## 6. Use Case Layer

### Directory Structure

```
internal/application/executionclient/
├── contracts.go           // ExecutionLatestQuery, ExecutionLatestReply
├── get_latest_execution.go     // GetLatestExecutionUseCase
└── get_latest_execution_test.go
```

### Use Case Process

1. Receive `ExecutionLatestQuery` with type, source, symbol, timeframe
2. Validate all fields are present and timeframe > 0
3. Delegate to `ExecutionGateway` port (NATS request/reply)
4. Return `ExecutionLatestReply` with execution intent (nullable)
5. Return `problem.Problem` on validation or gateway errors

### MUST NOT

1. Import execution domain types directly (use client-owned contracts)
2. Apply domain logic (validation beyond parameter presence)
3. Access NATS or KV directly (delegate to gateway port)

---

## 7. HTTP Handler Layer

### Directory Structure

```
internal/interfaces/http/handlers/execution.go
internal/interfaces/http/handlers/execution_test.go
```

### Handler Process

1. Extract `type` from path parameter
2. Extract `source`, `symbol`, `timeframe` from query parameters
3. Extract `X-Correlation-ID` from request header
4. Call `GetLatestExecutionUseCase` with constructed query
5. Return 200 with JSON body (execution_intent field always present, may be null)
6. Return 400 on validation errors
7. Return 503 on service unavailable

---

## 8. Route Registration

### Directory Structure

```
internal/interfaces/http/routes/execution.go
```

### Registration Pattern

Execution routes are registered independently from risk, strategy, decision, signal, and evidence routes. Routes are conditionally registered only if `ExecutionFamilyDeps.HasAny()` returns true (at least one execution use case is available).

```go
type ExecutionFamilyDeps struct {
    GetLatestExecution handlersGetLatestExecutionUseCase
}

func (d ExecutionFamilyDeps) HasAny() bool {
    return d.GetLatestExecution != nil
}
```

---

## 9. History Queries — Deferred

```
GET /execution/{type}/history?source={source}&symbol={symbol}&timeframe={timeframe}&limit={limit}[&since={since}&until={until}]
```

**Deferred because:**
1. History projection requires trace metadata persistence decision (S72)
2. Phase 1 paper execution only needs latest state
3. History semantics for execution intents require lifecycle tracking (submitted → filled → cancelled)
4. Audit trail for execution is better served by a purpose-built audit mechanism than a history bucket

---

## 10. Store Query Responder

### Handler Registration

When `ExecutionRegistry` is enabled, `QueryResponderActor` creates:
- `ExecutionKVStore` connection to `EXECUTION_PAPER_ORDER_LATEST` bucket
- NATS request/reply handler on `execution.query.paper_order.latest` subject

### Handler Process

1. Decode NATS request envelope
2. Extract `source`, `symbol`, `timeframe` from payload
3. Construct KV key: `{source}.{symbol}.{timeframe}`
4. Read from `EXECUTION_PAPER_ORDER_LATEST` bucket
5. Encode reply with `found` boolean and execution intent (nullable)
6. Return encoded reply envelope

### MUST NOT

1. Apply domain logic (filtering, transformation)
2. Join with other domain projections
3. Validate execution domain rules
4. Cache results

---

## 11. Query Invariants

| ID | Invariant |
|----|-----------|
| QI-1 | Each execution type has its own NATS query subject. No subject sharing across types. |
| QI-2 | Gateway is a stateless translator. It parses, forwards, and formats — nothing else. |
| QI-3 | KV bucket is the single source of truth for materialized execution intents. |
| QI-4 | Execution queries are independent of risk, strategy, decision, signal, and evidence queries. No cross-domain joins. |
| QI-5 | Query response always includes `execution_intent` field (never omitted; null when not found). |
| QI-6 | Validation of domain rules happens at projection level, not at query level. |
| QI-7 | `timeframe` parameter is required and must be a positive integer. |
| QI-8 | Unknown `type` path parameter returns 400 (not 404). |
| QI-9 | Execution query availability does not affect gateway readiness. Graceful degradation. |

---

## 12. References

- [execution-domain-design.md](execution-domain-design.md) — Domain model and boundary invariants
- [execution-stream-families.md](execution-stream-families.md) — Stream family catalog
- [execution-activation-and-ownership.md](execution-activation-and-ownership.md) — Activation model and actor ownership
- [risk-query-surface-guidelines.md](risk-query-surface-guidelines.md) — Upstream query surface reference
- [strategy-query-surface-guidelines.md](strategy-query-surface-guidelines.md) — Pattern reference
