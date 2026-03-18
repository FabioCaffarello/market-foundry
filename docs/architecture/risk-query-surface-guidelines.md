# Risk Query Surface Guidelines

> Stage S62 — Approved 2026-03-18
> Status: **DESIGN ONLY — no implementation in this stage**

---

## 1. Query Architecture

Risk queries follow the same **four-layer query chain** established by evidence, signal, decision, and strategy domains.

```
Layer 1: HTTP Endpoint (gateway)
Layer 2: Use Case (riskclient package)
Layer 3: NATS Request/Reply (QueryResponderActor in store)
Layer 4: KV Bucket (RISK_{TYPE}_LATEST)
```

Each layer has a single responsibility. No layer bypasses another.

---

## 2. Phase 1 — Latest Only

### Endpoint

```
GET /risk/{type}/latest?source=X&symbol=Y&timeframe=Z
```

### Query Parameters

| Parameter   | Required | Description                     | Example    |
|-------------|----------|---------------------------------|------------|
| `source`    | Yes      | Exchange source                 | `binancef` |
| `symbol`    | Yes      | Trading pair                    | `btcusdt`  |
| `timeframe` | Yes      | Sampling window in seconds      | `60`       |

### Phase 1 Types

| Type                  | Endpoint                                    |
|-----------------------|---------------------------------------------|
| `position_exposure`   | `GET /risk/position_exposure/latest?...`    |

### Response (200 — Found)

```json
{
  "risk": {
    "type": "position_exposure",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": "60",
    "disposition": "approved",
    "confidence": "0.85",
    "strategies": [
      {
        "type": "mean_reversion_entry",
        "direction": "long",
        "confidence": "0.72"
      }
    ],
    "constraints": {
      "max_position_size": "0.01",
      "max_exposure": "0.05",
      "stop_distance": ""
    },
    "rationale": "Position size within exposure limits",
    "parameters": {
      "max_position_pct": "0.02",
      "max_portfolio_exposure_pct": "0.10"
    },
    "metadata": {},
    "final": true,
    "timestamp": 1710700000000000000
  }
}
```

### Response (200 — Not Found)

```json
{
  "risk": null
}
```

### Response (400 — Invalid Parameters)

```json
{
  "code": "invalid_parameters",
  "message": "missing required parameter: source"
}
```

---

## 3. NATS Subject Mapping

| Operation | NATS Subject                                | Status   |
|-----------|---------------------------------------------|----------|
| Latest    | `risk.query.position_exposure.latest`       | Phase 1  |
| History   | `risk.query.position_exposure.history`      | Deferred |

---

## 4. Envelope Types

| Type                                                 | Direction | Status   |
|------------------------------------------------------|-----------|----------|
| `risk.query.v1.position_exposure_latest_request`     | Request   | Phase 1  |
| `risk.query.v1.position_exposure_latest_reply`       | Reply     | Phase 1  |
| `risk.query.v1.position_exposure_history_request`    | Request   | Deferred |
| `risk.query.v1.position_exposure_history_reply`      | Reply     | Deferred |

---

## 5. Gateway Implementation Rules

### Gateway MUST

| Rule  | Description                                                                  |
|-------|------------------------------------------------------------------------------|
| GR-1  | Register routes under `/risk/{type}/{operation}`                             |
| GR-2  | Parse `source`, `symbol`, `timeframe` from query parameters                  |
| GR-3  | Forward parsed query to risk use case (riskclient package)                   |
| GR-4  | Return use case reply as JSON with appropriate HTTP status code              |
| GR-5  | Return 400 for missing or invalid query parameters                           |
| GR-6  | Return 200 with `null` entity for "not found" (no 404 for missing data)     |
| GR-7  | Use shared parameter parsing (`parseRiskKeyParams()`)                        |

### Gateway MUST NOT

| Rule  | Prohibition                                                                  |
|-------|------------------------------------------------------------------------------|
| GN-1  | Access risk KV buckets directly                                              |
| GN-2  | Cache risk query responses                                                   |
| GN-3  | Transform, filter, or enrich risk data                                       |
| GN-4  | Validate domain invariants (disposition values, confidence ranges)            |
| GN-5  | Interpret risk metadata or constraints                                       |
| GN-6  | Cross-query between risk and other domains (no join queries)                 |
| GN-7  | Add domain logic to risk routes (aggregation, thresholds, alerts)            |
| GN-8  | Produce domain events from query handlers                                    |

---

## 6. Use Case Layer (riskclient)

### Contracts

```go
// internal/application/riskclient/contracts.go

type GetLatestRiskQuery struct {
    Type      string
    Source    string
    Symbol    string
    Timeframe string
}

type GetLatestRiskReply struct {
    Risk *risk.RiskAssessment
}
```

### Implementation

```go
// internal/application/riskclient/get_latest_risk.go

type GetLatestRisk struct {
    gateway ports.RiskGateway
}

func (uc *GetLatestRisk) Execute(ctx context.Context, query GetLatestRiskQuery) (GetLatestRiskReply, *problem.Problem) {
    return uc.gateway.GetLatestRisk(ctx, query)
}
```

The use case is a thin delegation to the port. No domain logic.

---

## 7. Store Query Responder

QueryResponderActor registers the `risk.query.position_exposure.latest` subject and handles requests by:

1. Decoding the NATS request envelope.
2. Constructing the KV key from query parameters: `{source}.{symbol}.{timeframe}`.
3. Reading from `RISK_POSITION_EXPOSURE_LATEST` bucket.
4. Encoding the risk assessment (or null) into the reply envelope.
5. Publishing the reply.

No domain logic in the responder — it is a KV-to-NATS translator.

---

## 8. Query Invariants

| ID    | Invariant                                                                        |
|-------|----------------------------------------------------------------------------------|
| QI-1  | Each risk type has its own query subject — no shared query handlers               |
| QI-2  | Gateway is stateless — no caching, no aggregation, no enrichment                 |
| QI-3  | KV bucket is the single source of truth for query responses                      |
| QI-4  | Query responses reflect the latest materialized projection, not real-time events |
| QI-5  | Risk queries are independent of strategy/decision/signal queries                 |
| QI-6  | No cross-domain joins in query layer                                             |
| QI-7  | Query parameter validation happens at gateway level only                         |
| QI-8  | Domain validation happens at projection level only                               |
| QI-9  | History queries are deferred — no "latest N" or "since T" in Phase 1             |

---

## 9. Adding a New Risk Query Type

When a new risk family is implemented (e.g., `drawdown_guard`), the query surface extends by:

1. **riskclient**: New `GetLatestDrawdownGuardQuery`/`Reply` types (or reuse generic if identical shape).
2. **Adapter**: New NATS gateway method for `risk.query.drawdown_guard.latest`.
3. **Handler**: New HTTP handler method in `internal/interfaces/http/handlers/risk.go`.
4. **Route**: New route registration in `internal/interfaces/http/routes/risk.go`.
5. **Store**: QueryResponderActor registers `risk.query.drawdown_guard.latest` subject.
6. **Test**: Route test, handler test, use case test.

No changes to core gateway, readiness, actor lifecycle, or webserver.

---

## 10. Phase 2 — History (Deferred)

When history queries are needed (S65+), the following extends:

### Endpoint

```
GET /risk/{type}/history?source=X&symbol=Y&timeframe=Z&limit=N[&since=T&until=T]
```

### Additional Parameters

| Parameter | Required | Description                      | Default |
|-----------|----------|----------------------------------|---------|
| `limit`   | No       | Max results to return            | 100     |
| `since`   | No       | Unix nanos lower bound           | —       |
| `until`   | No       | Unix nanos upper bound           | —       |

### Requirements Before History

- History KV bucket created (`RISK_POSITION_EXPOSURE_HISTORY`).
- Projection actor extended to write both latest and history.
- Query responder extended with range query support.
- Gateway route and handler extended.
- Retention policy reviewed for history bucket size.
