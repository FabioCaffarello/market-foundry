# Controlled Capability 01 — Architectural Pressure Points

> Stage: S119 | Status: Defined | Date: 2025-03-19

## Purpose

This document maps the specific points in the architecture that CC-01 (Multi-Symbol Live Monitoring) will stress. Each pressure point identifies what is being tested, what the expected behavior is, and what friction might emerge.

The goal is not to predict failures — it is to know exactly **where to look** when something unexpected happens.

## 1. Config Activation Under Concurrent State

### What Is Pressured

The configctl lifecycle manages a single active config with multiple ingestion bindings. CC-01 activates a config with 2 bindings simultaneously (or adds a second binding incrementally).

### Expected Behavior

- Config validation passes with 2 bindings using identical family chains
- `IngestionRuntimeChangedEvent` publishes once, listing both bindings
- Ingest discovers both bindings and opens both WS connections

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| CP-1 | Incremental binding addition requires full config re-activation (draft → validate → compile → activate), not a partial update | Known | This is by design. Document if operator finds it cumbersome. |
| CP-2 | Config validation does not check for duplicate symbol bindings (same source+symbol) | Low | Would produce duplicate event streams. Validate manually. |
| CP-3 | `IngestionRuntimeChangedEvent` handling in ingest is not idempotent for already-connected bindings | Low | If triggered, ingest may open duplicate WS connections. Monitor tracker count. |

### Where to Look

- `internal/domain/configctl/` — config validation logic
- `internal/actors/scopes/ingest/` — binding watcher and WS connection management
- `configctl` logs — activation event publishing
- `ingest` logs — binding discovery and WS connection lifecycle

## 2. Concurrent WebSocket Management

### What Is Pressured

The ingest runtime opens and maintains one WS connection per binding. CC-01 requires two concurrent connections to Binance (btcusdt and ethusdt trade streams).

### Expected Behavior

- Both connections open successfully
- Both produce `TradeReceived` events independently
- If one connection drops, the other continues unaffected
- Reconnection for a dropped connection does not affect the other

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| WS-1 | Goroutine leak if WS connection management doesn't properly scope per-binding | Low | Monitor goroutine count via `runtime.NumGoroutine()` in `/statusz` if available, or `docker stats`. |
| WS-2 | Shared state between WS handlers (e.g., rate limiting, backpressure) | Low | Both connections share the same NATS publisher. Monitor for publish latency. |
| WS-3 | Binance rate limits on concurrent WS connections from same IP | Very Low | Binance allows multiple WS connections. Only relevant at 10+ symbols. |

### Where to Look

- `internal/actors/scopes/ingest/` — WS adapter actors
- ingest `/statusz` — tracker counts per WS connection
- Docker container memory/goroutine counts

## 3. Derive Throughput Doubling

### What Is Pressured

The derive runtime processes observation events through the full sampler chain (candle → RSI → rsi_oversold → mean_reversion_entry → position_exposure → paper_order). CC-01 doubles the event throughput by adding a second symbol.

### Expected Behavior

- Each sampler maintains independent state per symbol (keyed by source+symbol+timeframe)
- Event processing for ethusdt does not block or delay btcusdt processing
- All samplers produce output for both symbols independently

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| DT-1 | Sampler state isolation relies on correct KV key construction. Bug would cause cross-symbol contamination. | Very Low | Validated in unit tests and single-symbol smoke. Watch for unexpected values in ethusdt responses. |
| DT-2 | Actor mailbox backpressure under doubled load | Low | Proto.Actor handles this, but monitor for increasing idle times in `/statusz`. |
| DT-3 | RSI computation for ethusdt requires warm-up period (14 candles = 14+ minutes at 60s timeframe) | Known | This is by design. ethusdt RSI will be null for ~15 minutes. Not a bug. |

### Where to Look

- `internal/actors/scopes/derive/` — all sampler/publisher actors
- derive `/statusz` — tracker event counts and idle times
- Gateway queries for ethusdt signal/decision (expect null during warm-up)

## 4. Store Projection Write Amplification

### What Is Pressured

The store runtime materializes every domain event into KV buckets. CC-01 doubles the number of KV writes across all projection actors.

### Expected Behavior

- KV keys are correctly partitioned by symbol (e.g., `EVIDENCE_CANDLE_LATEST.binancef.ethusdt.60`)
- Write throughput doubles without errors
- KV bucket size grows linearly

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| KV-1 | NATS KV bucket size limits (default 1GB per bucket) | Very Low | Individual entries are small. Only relevant at massive scale. |
| KV-2 | Projection actors share a single NATS connection. Write contention under doubled load. | Low | NATS handles concurrent writes well. Monitor for publish errors in store logs. |
| KV-3 | KV key enumeration for "list all symbols" is not supported (point lookups only) | Known | This is a known limitation. CC-01 queries are point lookups. Symbol listing is out of scope. |

### Where to Look

- `internal/actors/scopes/store/` — all projection actors
- store `/statusz` — tracker event counts and error counts
- NATS monitoring (if enabled) — KV bucket stats

## 5. Execute Actor Concurrent Order Processing

### What Is Pressured

The execute runtime's paper venue adapter processes execution intent events for both symbols. The safety model (kill switch, staleness guard, submit timeout) must work independently per symbol.

### Expected Behavior

- Paper orders for btcusdt and ethusdt are evaluated independently
- Safety gates apply per-event, not per-symbol (no shared state between symbols)
- Fill events are published with correct symbol attribution

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| EX-1 | SafetyGate staleness check uses wall-clock comparison. If ethusdt events arrive slightly delayed, staleness may trigger incorrectly. | Low | The staleness window (120s default) is generous. Monitor for unexpected staleness rejections. |
| EX-2 | Paper simulator has no per-symbol position tracking. Each order is independent. | Known | By design for paper_order family. Real position tracking would be a separate capability. |

### Where to Look

- `internal/application/execution/safety_gate.go` — safety gate logic
- `internal/actors/scopes/execute/venue_adapter_actor.go` — order processing
- execute `/statusz` — tracker event counts

## 6. Gateway Query Surface Under Multi-Symbol Load

### What Is Pressured

The gateway serves HTTP queries for all domains. CC-01 does not change the gateway code, but operators will query both symbols, doubling query volume.

### Expected Behavior

- All endpoints correctly return data for both symbols via `?symbol=` parameter
- Response times remain stable under doubled query load
- Missing data (ethusdt during warm-up) returns structured null, not errors

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| GW-1 | No endpoint to list all available symbols. Operator must know which symbols are active. | Known | This is a known gap. Not blocking for CC-01 but likely to surface as operator friction. |
| GW-2 | No cross-symbol aggregation endpoint (e.g., "show RSI for all active symbols") | Known | Out of scope. Point queries only. |

### Where to Look

- Gateway response bodies for `?symbol=ethusdt` queries
- Gateway response times (manual or log-based)

## 7. Cross-Runtime Debugging

### What Is Pressured

With two symbols flowing through 6 runtimes simultaneously, debugging any issue requires correlating events across runtimes and symbols.

### Expected Behavior

- Structured logs include `source`, `symbol`, `timeframe` fields for filtering
- Events can be traced by timestamp correlation

### Potential Friction

| # | Friction | Likelihood | Signal |
|---|---------|------------|--------|
| DB-1 | No correlation ID in slog context. Cross-runtime event tracing requires manual timestamp matching across 6 services × 2 symbols. | **High** | This was identified as F1 in S118. CC-01 will almost certainly confirm it. |
| DB-2 | Log volume doubles. Finding relevant entries becomes harder without structured filtering tools. | Medium | `docker compose logs --no-log-prefix` with grep may become unwieldy. |

### Where to Look

- All runtime logs during any debugging session
- Operator experience report during CC-01 validation

## 8. Pressure Point Summary

| # | Pressure Point | Severity | Expected Outcome |
|---|---------------|----------|-----------------|
| CP-1..3 | Config activation | Low | Works as designed |
| WS-1..3 | Concurrent WS | Low | Works as designed |
| DT-1..3 | Derive throughput | Low-Med | Works, with warm-up period |
| KV-1..3 | Store projections | Low | Works as designed |
| EX-1..2 | Execute concurrent | Low | Works as designed |
| GW-1..2 | Gateway queries | Known | Works, with known gaps |
| DB-1..2 | Cross-runtime debug | **High** | Will confirm F1 friction |

### Expected Outcome Distribution

- **Will just work:** CP, WS, KV, EX (config, ingest, store, execute)
- **Will work with known limitations:** DT (warm-up period), GW (no symbol listing)
- **Will confirm known friction:** DB (correlation ID gap)
- **May reveal new friction:** DT-2 (actor mailbox under load), WS-1 (goroutine management)

## 9. Monitoring Checklist During CC-01 Execution

Use this checklist during the live multi-symbol operation to systematically verify each pressure point:

```
[ ] CP: Config activates with 2 bindings. No errors in configctl logs.
[ ] WS: Two WS trackers visible in ingest /statusz. Both event_count > 0.
[ ] DT: derive /statusz shows activity for both symbols (tracker event counts).
[ ] DT: ethusdt RSI is null for first ~15 minutes (warm-up). Not a bug.
[ ] KV: store /statusz shows doubled tracker activity. Zero error_count.
[ ] EX: execute /statusz shows activity. No staleness rejections in logs.
[ ] GW: All 12 query endpoints (6 domains × 2 symbols) return valid data.
[ ] DB: Note any debugging difficulty encountered. Record correlation ID pain.
[ ] S:  Docker stats at 10-min mark. Record memory per container.
[ ] S:  Docker stats at 30-min mark. Compare to 10-min baseline.
[ ] S:  No error-level logs from domain logic over 30 minutes.
```
