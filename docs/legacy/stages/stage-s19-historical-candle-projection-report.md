# S19: Historical Candle Projection — Stage Report

## Summary

Added a historical candle projection alongside the existing latest-candle projection. The store now writes every finalized candle to a second NATS KV bucket (`CANDLE_HISTORY`) with time-indexed keys, enabling "last N candles" queries through the full gateway → store → NATS KV read path.

## What Changed

### NATS Adapter — `candle_kv_store.go`
- Added `CANDLE_HISTORY` bucket (24h TTL, 256MB max, file storage).
- `PutHistory(ctx, candle)` writes candle at key `{source}.{symbol}.{tf}.{open_time_unix}`.
- `GetHistory(ctx, source, symbol, timeframe, limit)` lists keys by prefix, sorts descending, fetches up to `limit` candles.
- Both buckets opened on `Start()` using the same NATS connection.

### Evidence Registry — `evidence_registry.go`
- Added `CandleHistory` control spec: subject `evidence.query.candle.history`, queue group `evidence.query`.

### Evidence Client — `evidenceclient/`
- `CandleHistoryQuery` and `CandleHistoryReply` contracts in `contracts.go`.
- `GetCandleHistoryUseCase` in `get_candle_history.go` with validation (source, symbol, timeframe required; limit clamped to [1, 100], default 10).
- Full test coverage in `get_candle_history_test.go`.

### Evidence Port — `ports/evidence.go`
- Added `GetCandleHistory` to `EvidenceGateway` interface.

### NATS Evidence Gateway — `evidence_gateway.go`
- `GetCandleHistory` implementation following the same encode/request/decode pattern as `GetLatestCandle`.

### Store Actors
- `CandleProjectionActor`: `onCandle()` now writes to both `CANDLE_LATEST` and `CANDLE_HISTORY`.
- `QueryResponderActor`: registered second control route for `CandleHistory`, delegates to `store.GetHistory()`.

### HTTP Layer
- `handlers/evidence.go`: added `GetCandleHistory` handler with limit validation [1, 100].
- `routes/evidence.go`: added `GET /evidence/candles/history` route. Evidence function now accepts both use cases and registers routes conditionally.
- `routes/core.go`: added `GetCandleHistory` to `Dependencies`.

### Gateway Wiring
- `cmd/gateway/run.go`: creates `GetCandleHistoryUseCase` from the existing `evGateway` and wires it into `Dependencies`.

### Docs & Tests
- Updated `docs/architecture/read-model-authority.md` with history projection details.
- Added history examples to `tests/http/evidence.http`.
- All test files updated for new constructor signatures and new test cases.

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Same consumer, two projections | Avoids duplicate event processing. Single `CandleProjectionActor` writes both buckets. |
| Unix timestamp in key | Naturally sortable, dedup-safe (same open_time overwrites same key). |
| 24h TTL + 256MB | Proportional for NATS KV. Prepares for ClickHouse migration later. |
| Limit [1, 100], default 10 | Prevents unbounded reads. Most common use case is "last few candles". |
| Descending order | Newest first — most useful for UI/dashboard consumers. |

## Verification

1. `go build ./...` — all packages compile
2. `go test ./...` — all tests pass
3. `GET /evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&limit=5` — returns candle history array
4. `GET /evidence/candles/latest` — still works unchanged
