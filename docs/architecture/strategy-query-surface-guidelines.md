# Strategy Query Surface Guidelines — Market Foundry

> Rules for exposing strategy read models via HTTP.
> Stage: S53 — Design only. Implementation deferred to S54+.
> Date: 2026-03-17

---

## 1. Four-Layer Query Chain

Strategy queries follow the same four-layer chain proven by evidence, signal, and decision:

```
HTTP Endpoint → Use Case → NATS Request/Reply → KV Bucket
     (1)          (2)            (3)              (4)
```

| Layer | Component | Responsibility |
|---|---|---|
| 1. HTTP Endpoint | Gateway route handler | Parse params, call use case, return JSON |
| 2. Use Case | `strategyclient` package | Build NATS request, send, deserialize reply |
| 3. NATS Responder | `QueryResponderActor` in store | Receive request, read KV, send reply |
| 4. KV Bucket | `STRATEGY_{TYPE}_LATEST` | Source of truth for materialized strategies |

---

## 2. Phase 1 — Latest Only

### HTTP Endpoint

```
GET /strategy/{type}/latest?source=X&symbol=Y&timeframe=Z
```

**Phase 1 types**: `mean_reversion_entry`

**Query parameters**:

| Param | Required | Description |
|---|---|---|
| `source` | Yes | Exchange identifier (e.g., `binancef`) |
| `symbol` | Yes | Trading pair, lowercase (e.g., `btcusdt`) |
| `timeframe` | Yes | Candle window in seconds (e.g., `60`) |

**Response** (200 OK):
```json
{
  "type": "mean_reversion_entry",
  "source": "binancef",
  "symbol": "btcusdt",
  "timeframe": 60,
  "direction": "long",
  "confidence": "0.85",
  "decisions": [
    {
      "type": "rsi_oversold",
      "outcome": "triggered",
      "confidence": "0.85",
      "timeframe": 60
    }
  ],
  "parameters": {
    "entry": "market",
    "target_offset": "0.02",
    "stop_offset": "0.01"
  },
  "metadata": {},
  "final": true,
  "timestamp": "2026-03-17T14:30:00Z"
}
```

**Response** (200 OK — flat direction):
```json
{
  "type": "mean_reversion_entry",
  "source": "binancef",
  "symbol": "btcusdt",
  "timeframe": 60,
  "direction": "flat",
  "confidence": "0.0",
  "decisions": [
    {
      "type": "rsi_oversold",
      "outcome": "not_triggered",
      "confidence": "0.0",
      "timeframe": 60
    }
  ],
  "parameters": {},
  "metadata": {},
  "final": true,
  "timestamp": "2026-03-17T14:30:15Z"
}
```

**Response** (404 Not Found): No strategy materialized yet for this partition.

**Response** (400 Bad Request): Missing or invalid query parameters.

---

## 3. NATS Subjects

| Operation | NATS Subject |
|---|---|
| Latest | `strategy.query.{type}.latest` |
| History (deferred) | `strategy.query.{type}.history` |

---

## 4. Envelope Types

| Direction | Envelope |
|---|---|
| Request | `strategy.query.v1.{type}_latest_request` |
| Reply | `strategy.query.v1.{type}_latest_reply` |

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
  "strategy": { ... },
  "found": true
}
```

Where `found: false` indicates no materialized strategy for the partition key.

---

## 5. Gateway Rules

The gateway MUST:
- Register strategy routes under `/strategy/{type}/{operation}`
- Parse query parameters and build typed request structs
- Forward requests to store via NATS request/reply
- Return the reply as-is in JSON format
- Return appropriate HTTP status codes (200, 400, 404, 502)

The gateway MUST NOT:
- Access strategy KV buckets directly
- Cache strategy responses
- Transform strategy data (filter, sort, enrich)
- Validate domain invariants (that's store's job)
- Interpret strategy metadata, parameters, or direction
- Cross-query strategy and decision in a single request
- Cross-query strategy and signal in a single request
- Cross-query strategy and evidence in a single request
- Add strategy-specific logic to existing decision, signal, or evidence routes

---

## 6. Use Case Layer

```
internal/application/strategyclient/
├── contracts.go                  — port interfaces
├── get_latest_strategy.go        — latest query use case
└── get_latest_strategy_test.go
```

The use case:
1. Receives typed request (source, symbol, timeframe, strategy type)
2. Builds NATS request envelope
3. Sends to `strategy.query.{type}.latest`
4. Deserializes reply
5. Returns domain-agnostic response to HTTP handler

The use case does NOT:
- Import `internal/domain/strategy`
- Access KV directly
- Cache results
- Apply business logic

---

## 7. HTTP Handler Layer

```
internal/interfaces/http/handlers/strategy.go
internal/interfaces/http/handlers/strategy_test.go
```

The handler:
1. Extracts query parameters
2. Validates required parameters are present
3. Calls use case
4. Returns JSON response with appropriate status code

---

## 8. Route Registration

```
internal/interfaces/http/routes/strategy.go
internal/interfaces/http/routes/strategy_test.go
```

Strategy routes are registered independently from decision, signal, and evidence routes. Route registration follows the same pattern as `routes/decision.go`.

---

## 9. History Queries — Deferred

History queries follow the same pattern as decision history but are deferred to S55+:

```
GET /strategy/{type}/history?source=X&symbol=Y&timeframe=Z&from=T1&to=T2
```

**Why deferred**: Latest-only is sufficient for the first slice. History adds range query parameters, pagination design, and retention policy decisions that should not block the initial pipeline proof.

---

## 10. Query Invariants

| ID | Invariant |
|---|---|
| **QI-1** | Every strategy query starts at the HTTP layer and terminates at a KV read — no shortcuts |
| **QI-2** | Gateway never reads KV directly — always via NATS request/reply to store |
| **QI-3** | Store serves strategy queries from KV buckets — not from event replay |
| **QI-4** | Query response shape matches the domain type — no ad-hoc transformations |
| **QI-5** | Strategy queries are independent of decision queries — no join, no aggregation |
| **QI-6** | Each strategy type has its own query subject — no polymorphic catch-all |
| **QI-7** | 404 means "no materialized data" — not "error" |
| **QI-8** | A `flat` direction in a 200 response is a valid result — not an error condition |
| **QI-9** | History queries (when implemented) never modify the latest bucket |

---

## References

- [strategy-domain-design.md](strategy-domain-design.md) — Domain design
- [decision-query-surface-guidelines.md](decision-query-surface-guidelines.md) — Decision query precedent
- [gateway-pattern.md](gateway-pattern.md) — Gateway rules
- [gateway-read-surface-guidelines.md](gateway-read-surface-guidelines.md) — Read surface rules
