# S20: Historical Range Query — Stage Report

## Summary

Extends the S19 candle history query with time-range semantics (`since`/`until` unix timestamps). This closes the minimal historical query cycle: callers can now ask "give me candles for btcusdt/60s between time A and time B" through the canonical gateway → store path.

No new endpoints, no new NATS subjects, no new actors. The existing `GET /evidence/candles/history` gains two optional parameters.

## Query Contract

```
GET /evidence/candles/history
  ?source=binancef
  &symbol=btcusdt
  &timeframe=60
  &limit=20          # optional, default 10, max 100
  &since=1710000000  # optional, unix seconds, inclusive lower bound
  &until=1710003600  # optional, unix seconds, inclusive upper bound
```

**Semantics:**
- `since`/`until` filter candles by `OpenTime` (inclusive on both ends)
- Either, both, or neither may be provided
- `since > until` is a validation error (400)
- Results always sorted newest-first (descending `OpenTime`)
- `limit` applied after range filtering

**Response:**
```json
{
  "candles": [
    { "source": "binancef", "symbol": "btcusdt", "timeframe": 60, "open": "...", ... },
    ...
  ]
}
```

## What Changed

### Contract Layer — `evidenceclient/contracts.go`
- Added `Since int64` and `Until int64` fields to `CandleHistoryQuery` (0 = unset, `omitempty` on wire)

### Use Case — `evidenceclient/get_candle_history.go`
- Validates `since >= 0`, `until >= 0`, `since <= until` when both are set
- Existing limit clamping unchanged

### Store Adapter — `nats/candle_kv_store.go`
- `GetHistory` signature extended with `since, until int64`
- Key-level filtering: extracts unix timestamp from key suffix, applies range bounds before fetching values
- No extra deserialization — filtering happens on key metadata only

### Query Responder — `store/query_responder_actor.go`
- Passes `query.Since` and `query.Until` through to store

### HTTP Handler — `handlers/evidence.go`
- Parses `since` and `until` query parameters (int64)
- Returns 400 for non-numeric values

### Gateway Wiring
- No changes — `since`/`until` flow through the existing NATS request/reply envelope transparently (CBOR-encoded in `CandleHistoryQuery`)

## Boundary Preservation

| Boundary | Status |
|----------|--------|
| Gateway → Store | Unchanged. Gateway sends `CandleHistoryQuery` via NATS request/reply. Never touches KV directly. |
| Store owns read model | Unchanged. `CandleKVStore.GetHistory` is the sole reader. Range filtering lives in the store adapter. |
| Single query path | Preserved. No new endpoints or NATS subjects. `evidence.query.candle.history` serves all history queries. |
| Latest vs History | Clean separation. Latest = single key GET. History = prefix scan with optional range + limit. |

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Extend existing contract, not new endpoint | One query path for history. Range is a filter refinement, not a different query type. |
| Unix seconds, not RFC3339 | Keys already store unix timestamps. No parsing overhead. Unambiguous timezone (UTC). |
| Key-level filtering (no deserialization) | Parse timestamp from key suffix to accept/reject before fetching value bytes. O(keys) string ops, not O(keys) JSON unmarshal. |
| Inclusive bounds `[since, until]` | Natural for candle alignment. `since=T` includes the candle starting at T. |
| 0 = unset (not pointer) | Avoids nullable fields in CBOR/JSON. Unix timestamp 0 (1970-01-01) is never a valid candle time. |

## Intentional Limitations

1. **Full key scan** — `Keys()` lists all keys in the bucket, then filters by prefix + range. Adequate for 24h/256MB retention, but would not scale to millions of keys. ClickHouse migration will replace this.
2. **No cursor/pagination** — `limit` caps results but there is no `offset` or continuation token. Sufficient for "last N in window" but not for scrolling through large ranges.
3. **No aggregation** — returns raw candles, no server-side OHLCV aggregation across timeframes.
4. **24h retention ceiling** — `CANDLE_HISTORY` bucket TTL is 24h. Queries for older data return empty results silently.

## Files Modified

| File | Change |
|------|--------|
| `internal/application/evidenceclient/contracts.go` | Added `Since`/`Until` to `CandleHistoryQuery` |
| `internal/application/evidenceclient/get_candle_history.go` | Range validation |
| `internal/application/evidenceclient/get_candle_history_test.go` | Range validation tests, passthrough tests |
| `internal/adapters/nats/candle_kv_store.go` | `GetHistory` gains `since`/`until` key filtering |
| `internal/actors/scopes/store/query_responder_actor.go` | Pass range params to store |
| `internal/interfaces/http/handlers/evidence.go` | Parse `since`/`until` query params |
| `internal/interfaces/http/handlers/evidence_test.go` | Range handler tests |
| `tests/http/evidence.http` | Range query smoke examples |
| `docs/stages/stage-s20-historical-range-query-report.md` | This report |

## Verification

1. `go build ./...` — all packages compile
2. `go test ./...` — all tests pass (15 new/updated test cases)
3. `GET /evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60` — still works (no range = all recent)
4. `GET /evidence/candles/history?...&since=T1&until=T2` — returns only candles in window
5. `GET /evidence/candles/history?...&since=T2&until=T1` — returns 400

## S21 Hardening Points

1. **Prefix-scoped key listing** — if NATS KV gains native prefix filtering on `Keys()`, switch to it to avoid full bucket scan.
2. **Cursor-based pagination** — add `after` cursor token for scrolling through large result sets.
3. **Response metadata** — include `count`, `has_more`, `oldest`/`newest` timestamps in reply for client-side pagination awareness.
4. **ClickHouse migration** — when time-series storage moves to ClickHouse, the store adapter swaps from KV scan to SQL `WHERE open_time BETWEEN ? AND ? ORDER BY open_time DESC LIMIT ?`. The contract and boundary stay identical.
5. **Rate limiting** — range queries over the full 24h window can return up to 1440 60s candles (capped at 100 by limit). Consider whether tighter default limits or server-side cost accounting are needed.
