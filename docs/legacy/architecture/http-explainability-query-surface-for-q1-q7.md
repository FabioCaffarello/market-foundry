# HTTP Explainability Query Surface for Q1–Q7

> Stage: S297 — Explainability Query Surface
> Status: Complete
> Predecessor: S296 (Composite Execution Read Model)

## Purpose

This document defines the HTTP query surface that exposes the composite execution read model (S296) as operationally consumable endpoints. The surface is designed to answer—or contribute to answering—the seven governing questions (Q1–Q7) defined in the S294 wave charter.

## Governing Questions Coverage

| Question | Description | Coverage | Endpoint |
|----------|-------------|----------|----------|
| Q1 | Why was execution X submitted? What signal chain produced it? | **Full** | `GET /analytical/composite/chain?correlation_id=...` |
| Q2 | Why was execution X rejected or modified by risk? What constraint triggered? | **Partial** | `GET /analytical/composite/chain` — risk stage shows disposition + constraints; structured attribution deferred to S298 |
| Q3 | Which signals contributed to decision D? With what values? | **Full** | `GET /analytical/composite/chain` — decision.signals + signal stage with values |
| Q4 | What was the confidence/severity flow from signal through execution? | **Full** | `GET /analytical/composite/chain` — each stage carries confidence; decision carries severity |
| Q5 | Why did symbol S stop receiving executions? Where did the pipeline break? | **Partial** | `GET /analytical/composite/chains?symbol=...` — `missing_stages` reveals breakpoints |
| Q6 | How many executions were blocked vs approved in period T? Why? | **Deferred** | Requires aggregation — planned for S298 |
| Q7 | What is the conversion rate at each pipeline stage for family F? | **Deferred** | Requires aggregation — planned for S298 |

## Endpoint Design

### 1. Single Chain Lookup

```
GET /analytical/composite/chain?correlation_id=<id>
```

**Purpose:** Reconstruct the full causal chain for one execution. This is the primary explainability endpoint — it answers "what happened and why" for a specific correlation_id.

**Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `correlation_id` | Yes | The correlation ID that unifies the causal chain |

**Response:** A `compositeChainResponse` containing 0 or 1 chains. Returns 0 chains if the correlation_id has no events in any table.

### 2. Batch Chain Lookup

```
GET /analytical/composite/chains?source=<s>&symbol=<s>&timeframe=<n>[&since=<ts>&until=<ts>&limit=<n>]
```

**Purpose:** List recent composite chains for a given source/symbol/timeframe. Starts from executions table, walks backward through the causal chain. This endpoint enables answering "what happened recently for this symbol" and "where are pipelines breaking."

**Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `source` | Yes (use case validates) | Data source, e.g. `binance` |
| `symbol` | Yes (use case validates) | Trading symbol, e.g. `BTCUSDT` |
| `timeframe` | Yes | Timeframe in seconds |
| `since` | No | Unix timestamp, inclusive lower bound |
| `until` | No | Unix timestamp, inclusive upper bound |
| `limit` | No | Max chains returned (default: 20, max: 100; handler validates 1–500 range, use case clamps to 100) |

**Response:** A `compositeChainResponse` containing 0–N chains ordered by execution timestamp DESC.

## Response Shape

Both endpoints return the same response structure:

```json
{
  "chains": [
    {
      "correlation_id": "abc-123",
      "signal": { /* SignalWithTrace */ },
      "decision": { /* DecisionWithTrace */ },
      "strategy": { /* StrategyWithTrace */ },
      "risk": { /* RiskWithTrace */ },
      "execution": { /* ExecutionWithTrace */ },
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

Each stage is optional — null/omitted when the stage has no event for this correlation_id. The `missing_stages` array names which of the five stages are absent.

## Architectural Alignment

- **Additive:** These endpoints are added under the existing `/analytical/` prefix alongside per-family history endpoints. They do not modify or overlap with operational routes.
- **Thin handler:** Each handler method is ~30 lines — parse params, call use case, format response. Zero domain logic in the HTTP layer.
- **Graceful degradation:** If ClickHouse is not configured, composite endpoints return 503 (same pattern as other analytical endpoints).
- **Conditional registration:** Routes are only registered when the composite use case is wired (non-nil in `AnalyticalFamilyDeps`).
- **Server-Timing header:** Both endpoints emit `Server-Timing` with total and query durations for operational visibility.

## What This Surface Does NOT Do

- No aggregation or counting (Q6, Q7) — deferred to S298
- No structured attribution of risk rejections — deferred to S298
- No streaming or websocket endpoints
- No UI or dashboard
- No write-side changes
- No cross-symbol composition
