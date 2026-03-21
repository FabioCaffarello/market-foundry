# Explainability HTTP Contracts, Payloads, and Usage Examples

> Stage: S297 — Explainability Query Surface
> Companion to: `http-explainability-query-surface-for-q1-q7.md`

## Contracts

### Request: Single Chain

```
GET /analytical/composite/chain?correlation_id=<correlation_id>
```

Required query parameter:
- `correlation_id` (string): The correlation ID that unifies all five stages of a causal chain.

### Request: Batch Chains

```
GET /analytical/composite/chains?source=<source>&symbol=<symbol>&timeframe=<timeframe>[&since=<unix_ts>&until=<unix_ts>&limit=<n>]
```

Required query parameters:
- `source` (string): Data source identifier (e.g., `binance`)
- `symbol` (string): Trading symbol (e.g., `BTCUSDT`)
- `timeframe` (integer): Timeframe in seconds (e.g., `60`)

Optional query parameters:
- `since` (int64): Unix timestamp, inclusive lower bound for time range
- `until` (int64): Unix timestamp, inclusive upper bound for time range
- `limit` (integer): Max chains to return. Default: 20. Max: 100. Range: 1–500 at HTTP layer, clamped to 100 by use case.

### Response: Composite Chain Response

```json
{
  "chains": [CompositeExecutionChain],
  "source": "clickhouse",
  "meta": {
    "total_ms": <int64>,
    "chain_count": <int>
  }
}
```

### Type: CompositeExecutionChain

```json
{
  "correlation_id": "<string>",
  "signal": <SignalWithTrace | null>,
  "decision": <DecisionWithTrace | null>,
  "strategy": <StrategyWithTrace | null>,
  "risk": <RiskWithTrace | null>,
  "execution": <ExecutionWithTrace | null>,
  "stage_count": <int 0-5>,
  "chain_complete": <bool>,
  "missing_stages": ["<stage_name>", ...]
}
```

### Type: *WithTrace (all five stages)

Each `*WithTrace` type embeds the domain struct and adds:

```json
{
  "event_id": "<string>",
  "correlation_id": "<string>",
  "causation_id": "<string>",
  "occurred_at": "<RFC3339 timestamp>"
}
```

Exception: `ExecutionWithTrace` uses `event_correlation_id` and `event_causation_id` to distinguish event-envelope metadata from domain-level fields.

### Error Responses

All errors follow the standard problem format:

```json
{
  "code": "<problem_code>",
  "message": "<human-readable message>"
}
```

| HTTP Status | Problem Code | Trigger |
|-------------|-------------|---------|
| 400 | `INVALID_ARGUMENT` | Missing required parameter, invalid integer, etc. |
| 503 | `SYS_UNAVAILABLE` | ClickHouse not configured, reader error, nil handler |

## Usage Examples

### Example 1: Q1 — Why was execution X submitted?

**Request:**
```
GET /analytical/composite/chain?correlation_id=ema-binance-btcusdt-60-1710000000
```

**Response (full chain):**
```json
{
  "chains": [
    {
      "correlation_id": "ema-binance-btcusdt-60-1710000000",
      "signal": {
        "type": "ema_crossover",
        "source": "binance",
        "symbol": "BTCUSDT",
        "timeframe": 60,
        "value": 67234.50,
        "confidence": 0.85,
        "event_id": "sig-001",
        "correlation_id": "ema-binance-btcusdt-60-1710000000",
        "causation_id": "",
        "occurred_at": "2025-03-09T12:00:00Z"
      },
      "decision": {
        "type": "ema_crossover",
        "outcome": "triggered",
        "severity": "high",
        "signals": [{"type": "ema_crossover", "value": 67234.50, "confidence": 0.85}],
        "event_id": "dec-001",
        "correlation_id": "ema-binance-btcusdt-60-1710000000",
        "causation_id": "sig-001",
        "occurred_at": "2025-03-09T12:00:01Z"
      },
      "strategy": {
        "type": "ema_crossover",
        "direction": "long",
        "entry_price": 67234.50,
        "quantity": 0.01,
        "event_id": "str-001",
        "correlation_id": "ema-binance-btcusdt-60-1710000000",
        "causation_id": "dec-001",
        "occurred_at": "2025-03-09T12:00:02Z"
      },
      "risk": {
        "type": "ema_crossover",
        "disposition": "approved",
        "constraints": {"max_position": 1.0, "daily_loss_limit": 500.0},
        "event_id": "rsk-001",
        "correlation_id": "ema-binance-btcusdt-60-1710000000",
        "causation_id": "str-001",
        "occurred_at": "2025-03-09T12:00:03Z"
      },
      "execution": {
        "type": "ema_crossover",
        "side": "buy",
        "status": "submitted",
        "quantity": 0.01,
        "price": 67234.50,
        "event_id": "exc-001",
        "event_correlation_id": "ema-binance-btcusdt-60-1710000000",
        "event_causation_id": "rsk-001",
        "occurred_at": "2025-03-09T12:00:04Z"
      },
      "stage_count": 5,
      "chain_complete": true,
      "missing_stages": []
    }
  ],
  "source": "clickhouse",
  "meta": {
    "total_ms": 5,
    "chain_count": 1
  }
}
```

**Reading the answer to Q1:** The execution was submitted because an EMA crossover signal (confidence 0.85) triggered a high-severity decision, which produced a long strategy at 67234.50, was approved by risk, and resulted in a buy order.

### Example 2: Q2 — Why was execution X rejected by risk?

**Request:**
```
GET /analytical/composite/chain?correlation_id=squeeze-binance-ethusdt-300-1710050000
```

**Response (risk-rejected chain):**
```json
{
  "chains": [
    {
      "correlation_id": "squeeze-binance-ethusdt-300-1710050000",
      "signal": { "type": "squeeze_breakout", "confidence": 0.72, "..." : "..." },
      "decision": { "outcome": "triggered", "severity": "medium", "..." : "..." },
      "strategy": { "direction": "long", "quantity": 0.5, "..." : "..." },
      "risk": {
        "type": "squeeze_breakout",
        "disposition": "rejected",
        "constraints": {"max_position": 1.0, "daily_loss_limit": 500.0},
        "event_id": "rsk-050",
        "causation_id": "str-050",
        "occurred_at": "2025-03-09T14:00:03Z"
      },
      "execution": null,
      "stage_count": 4,
      "chain_complete": false,
      "missing_stages": ["execution"]
    }
  ],
  "source": "clickhouse",
  "meta": { "total_ms": 4, "chain_count": 1 }
}
```

**Reading the answer to Q2:** The execution was rejected at the risk gate. The risk stage shows `disposition: "rejected"` and the `constraints` field reveals which limits were active. The chain has 4 stages — execution is missing because risk blocked it.

> **Limitation:** The current surface shows that risk rejected and what constraints were configured, but does not yet provide structured attribution of *which specific constraint* caused the rejection. Structured attribution is planned for S298.

### Example 3: Q5 — Why did symbol S stop receiving executions?

**Request:**
```
GET /analytical/composite/chains?source=binance&symbol=ETHUSDT&timeframe=300&limit=10
```

**Response (mixed chains showing pipeline breaks):**
```json
{
  "chains": [
    {
      "correlation_id": "trend-binance-ethusdt-300-1710060000",
      "signal": { "type": "trend_strength", "..." : "..." },
      "decision": { "outcome": "skipped", "..." : "..." },
      "strategy": null,
      "risk": null,
      "execution": null,
      "stage_count": 2,
      "chain_complete": false,
      "missing_stages": ["strategy", "risk", "execution"]
    },
    {
      "correlation_id": "squeeze-binance-ethusdt-300-1710050000",
      "signal": { "..." : "..." },
      "decision": { "..." : "..." },
      "strategy": { "..." : "..." },
      "risk": { "disposition": "rejected", "..." : "..." },
      "execution": null,
      "stage_count": 4,
      "chain_complete": false,
      "missing_stages": ["execution"]
    }
  ],
  "source": "clickhouse",
  "meta": { "total_ms": 42, "chain_count": 2 }
}
```

**Reading the answer to Q5:** Recent chains for ETHUSDT show two patterns of pipeline breakage: (1) decisions are being skipped (chain stops at decision with `outcome: "skipped"`), and (2) risk is rejecting strategies. No complete chains reached execution.

> **Note:** Batch mode starts from the executions table, so chains that never produced an execution will NOT appear here. This endpoint shows execution-rooted chains. For chains that broke before execution, you would need to query individual stages via the per-family analytical history endpoints.

## Response Headers

Both endpoints emit a `Server-Timing` header:

```
Server-Timing: total;dur=5, query;dur=3
```

- `total`: Wall-clock time for the full HTTP request/response cycle
- `query`: Wall-clock time for the composite read model queries (from use case meta)

## Limits and Caveats

1. **Eventual consistency:** The five tables are queried independently. A very recent event may not yet be visible in all tables. Typical propagation delay is sub-second.

2. **1:1 cardinality assumption:** Each stage returns at most one event per correlation_id (the most recent). If a correlation_id has multiple events in one stage, only the latest is returned.

3. **Batch starts from executions:** The batch endpoint discovers correlation_ids from the executions table. Chains that never reached execution are not discoverable via batch — use the single-chain endpoint with a known correlation_id instead.

4. **No aggregation:** These endpoints return individual chains, not counts or percentages. Aggregation (Q6, Q7) is planned for S298.

5. **No cross-symbol queries:** Each batch query is scoped to one source/symbol/timeframe triple.

6. **Performance:** Single chain: <10ms typical. Batch of 20 chains: <200ms typical.
