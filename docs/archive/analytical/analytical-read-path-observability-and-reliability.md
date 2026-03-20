# Analytical Read Path — Observability and Reliability

> Status: active | Introduced: S160 | Scope: read path instrumentation

## Purpose

This document defines the observability and reliability model for the analytical read path — the chain from HTTP request through the gateway, use case, and ClickHouse adapter that serves historical candle queries.

Before S160, the read path was functionally correct but operationally opaque: queries succeeded or failed with no timing, row counting, or structured diagnostic signals. This document records what is now observable and what remains outside instrumentation scope.

## Instrumentation Layers

The read path is instrumented at three layers, each with distinct responsibilities:

### Layer 1 — ClickHouse Adapter (`candle_reader.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Query completed | DEBUG | `source`, `symbol`, `timeframe`, `rows`, `elapsed_ms` |
| Query failed | ERROR | `source`, `symbol`, `timeframe`, `elapsed_ms`, `error` |
| Scan failed | ERROR | `source`, `symbol`, `timeframe`, `error` |
| Row iteration failed | ERROR | `source`, `symbol`, `timeframe`, `error` |

The adapter measures wall-clock time from query dispatch to row iteration completion. This captures ClickHouse round-trip latency plus row scanning overhead.

### Layer 2 — Use Case (`get_candle_history.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Query completed | INFO | `source`, `symbol`, `timeframe`, `rows`, `query_ms` |
| Query failed | WARN | `source`, `symbol`, `timeframe`, `elapsed_ms`, `error` |

The use case measures the reader call duration and populates `QueryMeta` in the reply:
- `query_ms`: milliseconds spent in the reader adapter
- `row_count`: number of rows returned

These meta fields flow through the HTTP response, making every successful analytical response self-describing.

### Layer 3 — HTTP Handler (`analytical.go`)

| Signal | Level | Fields |
|--------|-------|--------|
| Request failed | WARN | `source`, `symbol`, `timeframe`, `total_ms`, `problem` |

The handler adds a `Server-Timing` header to successful responses:
```
Server-Timing: total;dur=15, query;dur=12
```

This enables timing analysis from curl, browser devtools, or any HTTP client without parsing the response body.

## Response Enrichment

Every successful analytical response now includes a `meta` object:

```json
{
  "candles": [...],
  "source": "clickhouse",
  "meta": {
    "query_ms": 12,
    "row_count": 50
  }
}
```

This makes the read path self-documenting: an operator can tell at a glance whether a query was fast or slow, and whether it returned the expected number of rows.

## Failure Visibility

| Failure Mode | HTTP Status | Log Level | Signal |
|---|---|---|---|
| ClickHouse not configured | 404 (route absent) | INFO at startup | "clickhouse not configured" |
| ClickHouse unreachable | 503 | WARN | "analytical query failed" with error |
| Query timeout | 503 | ERROR+WARN | adapter logs elapsed_ms, use case logs error |
| Scan/type mismatch | 503 | ERROR | "scan failed" with column details |
| Validation failure | 400 | — | client-side error, no server log |

## What Is NOT Observable

The following remain outside instrumentation scope after S160:

1. **Query plan analysis** — no EXPLAIN or query profiling; requires ClickHouse-side tooling.
2. **Connection pool state** — the adapter does not expose pool metrics; degradation surfaces as increased `query_ms`.
3. **Per-column type mismatches** — scan errors report the row but not which column failed.
4. **Concurrent query load** — no request counter or concurrency gauge; would require middleware.
5. **Historical trend analysis** — logs are emitted per-request with no aggregation; external log tooling needed.
6. **Write path correlation** — no shared trace ID between writer inserts and reader queries.

## Design Constraints

- **No readiness impact**: ClickHouse is not part of the gateway readiness check (R-02). Analytical failures do not degrade operational endpoints.
- **No heavy observability**: No metrics libraries, no tracing SDKs, no external dependencies. All signals use `log/slog` and HTTP headers.
- **No response shape changes for errors**: Error responses remain RFC 7807 problem objects. Meta is only present on 200 responses.
- **Additive only**: All changes are additive to the existing read path. No existing behavior or response contracts were modified for operational endpoints.

## Operational Guidance

See [analytical-read-path-runbook-and-signal-interpretation.md](analytical-read-path-runbook-and-signal-interpretation.md) for scenario-based playbooks.
