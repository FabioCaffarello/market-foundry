# Decision Query Surface Guidelines — Market Foundry

> Rules for exposing decision read models via HTTP.
> Stage: S42 — Design only. Implementation deferred to S43+.
> Approved: 2026-03-17

---

## 1. Four-Layer Query Chain

Decision queries follow the same four-layer chain proven by evidence and signal:

```
HTTP Endpoint → Use Case → NATS Request/Reply → KV Bucket
     (1)          (2)            (3)              (4)
```

| Layer | Component | Responsibility |
|---|---|---|
| 1. HTTP Endpoint | Gateway route handler | Parse params, call use case, return JSON |
| 2. Use Case | `decisionclient` package | Build NATS request, send, deserialize reply |
| 3. NATS Responder | `QueryResponderActor` in store | Receive request, read KV, send reply |
| 4. KV Bucket | `DECISION_{TYPE}_LATEST` | Source of truth for materialized decisions |

---

## 2. Phase 1 — Latest Only

### HTTP Endpoint

```
GET /decision/{type}/latest?source=X&symbol=Y&timeframe=Z
```

**Phase 1 types**: `rsi_oversold`

**Query parameters**:

| Param | Required | Description |
|---|---|---|
| `source` | Yes | Exchange identifier (e.g., `binancef`) |
| `symbol` | Yes | Trading pair, lowercase (e.g., `btcusdt`) |
| `timeframe` | Yes | Candle window in seconds (e.g., `60`) |

**Response** (200 OK):
```json
{
  "type": "rsi_oversold",
  "source": "binancef",
  "symbol": "btcusdt",
  "timeframe": 60,
  "outcome": "triggered",
  "confidence": "0.85",
  "signals": [
    {
      "type": "rsi",
      "value": "28.45",
      "timeframe": 60
    }
  ],
  "metadata": {
    "threshold": "30.0"
  },
  "final": true,
  "timestamp": "2026-03-17T14:30:00Z"
}
```

**Response** (404 Not Found): No decision materialized yet for this partition.

**Response** (400 Bad Request): Missing or invalid query parameters.

---

## 3. NATS Subjects

| Operation | NATS Subject |
|---|---|
| Latest | `decision.query.{type}.latest` |
| History (deferred) | `decision.query.{type}.history` |

---

## 4. Envelope Types

| Direction | Envelope |
|---|---|
| Request | `decision.query.v1.{type}_latest_request` |
| Reply | `decision.query.v1.{type}_latest_reply` |

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
  "decision": { ... },
  "found": true
}
```

Where `found: false` indicates no materialized decision for the partition key.

---

## 5. Gateway Rules

The gateway MUST:
- Register decision routes under `/decision/{type}/{operation}`
- Parse query parameters and build typed request structs
- Forward requests to store via NATS request/reply
- Return the reply as-is in JSON format
- Return appropriate HTTP status codes (200, 400, 404, 502)

The gateway MUST NOT:
- Access decision KV buckets directly
- Cache decision responses
- Transform decision data (filter, sort, enrich)
- Validate domain invariants (that's store's job)
- Interpret decision metadata or outcome
- Cross-query decision and signal in a single request
- Cross-query decision and evidence in a single request
- Add decision-specific logic to existing evidence or signal routes

---

## 6. Use Case Layer

```
internal/application/decisionclient/
├── contracts.go              — port interfaces
├── get_latest_decision.go    — latest query use case
└── get_latest_decision_test.go
```

The use case:
1. Receives typed request (source, symbol, timeframe, decision type)
2. Builds NATS request envelope
3. Sends to `decision.query.{type}.latest`
4. Deserializes reply
5. Returns domain-agnostic response to HTTP handler

The use case does NOT:
- Import `internal/domain/decision`
- Access KV directly
- Cache results
- Apply business logic

---

## 7. HTTP Handler Layer

```
internal/interfaces/http/handlers/decision.go
internal/interfaces/http/handlers/decision_test.go
```

The handler:
1. Extracts query parameters
2. Validates required parameters are present
3. Calls use case
4. Returns JSON response with appropriate status code

---

## 8. Route Registration

```
internal/interfaces/http/routes/decision.go
internal/interfaces/http/routes/decision_test.go
```

Decision routes are registered independently from signal and evidence routes.
Route registration follows the same pattern as `routes/signal.go`.

---

## 9. History Queries — Deferred

History queries follow the same pattern as evidence candle history but are deferred to S44+:

```
GET /decision/{type}/history?source=X&symbol=Y&timeframe=Z&from=T1&to=T2
```

**Why deferred**: Latest-only is sufficient for the first slice. History adds range query
parameters, pagination design, and retention policy decisions that should not block the
initial pipeline proof.

---

## 10. Query Invariants

| ID | Invariant |
|---|---|
| **QI-1** | Every decision query starts at the HTTP layer and terminates at a KV read — no shortcuts |
| **QI-2** | Gateway never reads KV directly — always via NATS request/reply to store |
| **QI-3** | Store serves decision queries from KV buckets — not from event replay |
| **QI-4** | Query response shape matches the domain type — no ad-hoc transformations |
| **QI-5** | Decision queries are independent of signal queries — no join, no aggregation |
| **QI-6** | Each decision type has its own query subject — no polymorphic catch-all |
| **QI-7** | 404 means "no materialized data" — not "error" |
| **QI-8** | History queries (when implemented) never modify the latest bucket |

---

## References

- [decision-domain-design.md](decision-domain-design.md) — Domain design
- [signal-query-surface-guidelines.md](signal-query-surface-guidelines.md) — Signal query precedent
- [gateway-pattern.md](gateway-pattern.md) — Gateway rules
- [gateway-read-surface-guidelines.md](gateway-read-surface-guidelines.md) — Read surface rules
