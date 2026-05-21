# Session Explainability Surface and Cross-Surface Consistency Audit

**Stage:** S455A
**Status:** Complete
**Date:** 2026-03-24

## Purpose

This document defines the session explainability surface introduced in S455A and documents the cross-surface consistency audit between KV, ClickHouse, and gateway read surfaces.

## Explainability Surface

### Problem Statement

Before S455A, answering "what happened with this order?" required querying multiple endpoints:

1. `GET /execution/status/latest` (KV composite status — latest only)
2. `GET /analytical/execution/lifecycle` (ClickHouse history — all event types)
3. `GET /analytical/execution/summary` (ClickHouse aggregate counts)

No single surface combined current state, historical timeline, and cross-surface consistency into one operational explanation.

### Solution: `GET /analytical/execution/explain`

A new endpoint that combines:

- **KV latest state**: intent, fill, rejection status + propagation (via NATS execution gateway)
- **ClickHouse history**: most recent lifecycle events, newest-first (via ClickHouse reader)
- **Cross-surface consistency checks**: automated comparison of KV and ClickHouse state
- **Structured explanation**: human-readable lifecycle narrative

**Query parameters:** `source` (required), `symbol` (required), `timeframe` (required), `limit` (optional, default 50)

**Response structure:**

```json
{
  "source": "binance_spot",
  "symbol": "BTCUSDT",
  "timeframe": 60,
  "kv_intent_status": "submitted",
  "kv_fill_status": "filled",
  "kv_rejection_status": "",
  "kv_propagation": "filled",
  "kv_available": true,
  "history": [...],
  "ch_latest_intent_status": "submitted",
  "ch_latest_fill_status": "filled",
  "ch_latest_rejection_status": "",
  "ch_propagation": "filled",
  "ch_available": true,
  "consistency": [
    {"surface": "cross", "field": "intent_status", "status": "consistent", ...},
    {"surface": "cross", "field": "fill_status", "status": "consistent", ...},
    {"surface": "cross", "field": "propagation", "status": "consistent", ...}
  ],
  "consistent": true,
  "explanation": "Order for binance_spot.BTCUSDT.60 reached terminal state: filled. ClickHouse has 2 events: 1 paper_order, 1 venue_market_order. KV and ClickHouse are consistent.",
  "meta": {"query_ms": 12, "row_count": 2}
}
```

### Degradation Behavior

- **KV unavailable**: CH data still returned; `kv_available=false`; consistency marked as unavailable
- **CH unavailable**: KV data still returned; `ch_available=false`; consistency marked as unavailable
- **Both unavailable**: Empty response with explanation noting no data found
- **KV reader nil** (NATS not configured): Same as KV unavailable — graceful degradation

### Architecture Decisions

1. **Composition at use case layer**: The use case takes both a `LifecycleHistoryReader` (CH) and a `SessionExplainKVReader` (KV) to avoid coupling the analytical layer to NATS directly.
2. **Best-effort on both surfaces**: Neither KV nor CH failure prevents the endpoint from returning useful data.
3. **Consistency is informational**: Divergences are reported but do not change HTTP status codes. The endpoint always returns 200 with the best available data.
4. **Propagation derivation**: Both KV and CH use the same `DeriveEffectivePropagation` logic for comparable results.

## Cross-Surface Consistency Audit

### Surfaces Audited

| Surface | Technology | Semantics | Owner |
|---------|-----------|-----------|-------|
| KV Paper Order | NATS KV | Latest intent per partition | store (projection actor) |
| KV Venue Fill | NATS KV | Latest fill per partition | store (fill projection actor) |
| KV Venue Rejection | NATS KV | Latest rejection per partition | store (rejection projection actor) |
| KV Control | NATS KV | Global execution gate | store (query responder) |
| CH Executions | ClickHouse | Full history, all event types | writer pipeline |
| Gateway Operational | HTTP | Proxies KV via NATS request/reply | gateway binary |
| Gateway Analytical | HTTP | Proxies CH via reader adapters | gateway binary |

### Consistency Findings

#### Consistent

1. **Status values**: Both KV and CH use the same `execution.Status` enum values. No translation or mapping differences.
2. **Rejection metadata embedding**: Both KV (S407) and CH (S411) embed `rejection_code`, `rejection_reason`, and `venue_detail.*` into the metadata map using identical logic.
3. **Propagation derivation**: `DeriveEffectivePropagation()` is used consistently in both the KV query responder and the new explain endpoint's CH derivation.
4. **Event types**: Both surfaces recognize `paper_order`, `venue_market_order`, `venue_rejection` as the three execution event families.
5. **Partition key scheme**: Both use `{source}.{symbol}.{timeframe}` consistently.

#### Known Limitations (Not Divergences)

1. **Quantity precision**: KV stores string quantities (e.g., `"0.50"`). ClickHouse stores `Float64` (0.5). The read path uses `FormatFloat(f, 'f', -1, 64)` which strips trailing zeros. This is a representation difference, not data loss — the numeric values are identical.

2. **CorrelationID dual storage**: ClickHouse stores both `correlation_id` (event envelope) and `exec_correlation_id` (intent's own correlation ID). The reader maps `exec_correlation_id` to `ExecutionIntent.CorrelationID`. This is correct and intentional — the envelope ID is for NATS tracing, while the intent ID is the domain correlation. Both exist in the same row.

3. **Timestamp semantics**: KV stores `execution.Timestamp` (intent creation time). ClickHouse additionally has `occurred_at` (event emission), `ingested_at` (insert time), and the same `timestamp` field. The read path correctly uses `timestamp` for domain queries. The additional timestamps are infrastructure-level and not exposed.

4. **KV latest-only vs CH history**: KV only stores the most recent event per partition key per bucket. ClickHouse stores all events. This is by design — KV is for low-latency current state, CH is for historical analysis.

### Corrections Made in S455A

1. **LifecycleHistoryEntry field parity**: Added `Risk` and `Parameters` fields to `LifecycleHistoryEntry`. These were present in `ExecutionIntent` but dropped during the lifecycle entry conversion, making CH lifecycle responses less informative than KV reads.

2. **DRY conversion**: Introduced `intentToLifecycleEntry()` helper to centralize the `ExecutionIntent` → `LifecycleHistoryEntry` conversion, preventing future field parity drift.

## Limitations

- The explain endpoint does not correlate execution events with upstream pipeline stages (signal, decision, strategy, risk). Use `/analytical/composite/chain` for full causal chain queries.
- Consistency checks are point-in-time. A divergence may reflect a timing gap (event persisted to KV but not yet ingested by writer pipeline to CH, or vice versa).
- The endpoint does not modify data — it is read-only and best-effort.
- No alerting or automated consistency enforcement. Divergences are reported for operator review.
