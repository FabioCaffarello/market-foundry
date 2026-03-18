# Query Contracts by Family

> Maps every evidence query contract to its projection family, showing the full chain from HTTP endpoint to NATS subject to KV bucket.

## Contract Chain

Every evidence query follows a four-layer chain:

```
HTTP Endpoint → Use Case → NATS Request/Reply → KV Bucket
   (gateway)    (application)    (evidence.query.*)    (store)
```

The gateway owns the HTTP layer. The store owns the KV layer. The NATS request/reply subjects are the contract boundary between them.

## Candle Family

### Latest Candle

| Layer | Value |
|-------|-------|
| **HTTP** | `GET /evidence/candles/latest?source=X&symbol=Y&timeframe=Z` |
| **Use Case** | `GetLatestCandleUseCase` |
| **NATS Subject** | `evidence.query.candle.latest` |
| **Request Type** | `evidence.query.v1.candle_latest_request` |
| **Reply Type** | `evidence.query.v1.candle_latest_reply` |
| **Queue Group** | `evidence.query` |
| **Server** | store → `QueryResponderActor` |
| **KV Bucket** | `CANDLE_LATEST` |
| **Key** | `{source}.{symbol}.{timeframe}` |

**Request contract:**
```go
CandleLatestQuery {
    Source    string  // required
    Symbol   string  // required
    Timeframe int    // required, positive integer (seconds)
}
```

**Reply contract:**
```go
CandleLatestReply {
    Candle *EvidenceCandle  // nil if not found
}
```

### Candle History

| Layer | Value |
|-------|-------|
| **HTTP** | `GET /evidence/candles/history?source=X&symbol=Y&timeframe=Z&limit=N&since=T1&until=T2` |
| **Use Case** | `GetCandleHistoryUseCase` |
| **NATS Subject** | `evidence.query.candle.history` |
| **Request Type** | `evidence.query.v1.candle_history_request` |
| **Reply Type** | `evidence.query.v1.candle_history_reply` |
| **Queue Group** | `evidence.query` |
| **Server** | store → `QueryResponderActor` |
| **KV Bucket** | `CANDLE_HISTORY` |
| **Key** | `{source}.{symbol}.{timeframe}.{open_time_unix}` (prefix scan) |

**Request contract:**
```go
CandleHistoryQuery {
    Source    string  // required
    Symbol   string  // required
    Timeframe int    // required, positive integer (seconds)
    Limit     int    // optional, default 10, max 100
    Since     int64  // optional, unix seconds, inclusive lower bound (0 = unset)
    Until     int64  // optional, unix seconds, inclusive upper bound (0 = unset)
}
```

**Reply contract:**
```go
CandleHistoryReply {
    Candles []EvidenceCandle  // newest-first, empty array if none
}
```

## TradeBurst Family

### Latest Trade Burst

| Layer | Value |
|-------|-------|
| **HTTP** | `GET /evidence/tradeburst/latest?source=X&symbol=Y&timeframe=Z` |
| **Use Case** | `GetLatestTradeBurstUseCase` |
| **NATS Subject** | `evidence.query.tradeburst.latest` |
| **Request Type** | `evidence.query.v1.trade_burst_latest_request` |
| **Reply Type** | `evidence.query.v1.trade_burst_latest_reply` |
| **Queue Group** | `evidence.query` |
| **Server** | store → `QueryResponderActor` |
| **KV Bucket** | `TRADE_BURST_LATEST` |
| **Key** | `{source}.{symbol}.{timeframe}` |

**Request contract:**
```go
TradeBurstLatestQuery {
    Source    string  // required
    Symbol   string  // required
    Timeframe int    // required, positive integer (seconds)
}
```

**Reply contract:**
```go
TradeBurstLatestReply {
    TradeBurst *EvidenceTradeBurst  // nil if not found
}
```

## Common Query Parameters

All evidence queries share three required parameters:

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `source` | string | Required, non-empty (use case validates) | Exchange identifier (e.g., `binancef`) |
| `symbol` | string | Required, non-empty (use case validates) | Trading pair (e.g., `btcusdt`) |
| `timeframe` | int | Required, positive integer (handler validates) | Window duration in seconds (e.g., `60`) |

These are the partition key dimensions of the evidence family. They mirror the KV key format `{source}.{symbol}.{timeframe}`.

## Ownership Summary

| Concern | Owner | Rationale |
|---------|-------|-----------|
| HTTP endpoint definition | gateway (routes/evidence.go) | Gateway owns the read surface |
| Parameter parsing/validation | gateway (handlers/evidence.go) | Input validation at system boundary |
| Business rule validation | application (evidenceclient/*) | Domain rules (limit bounds, range checks) |
| NATS request encoding | adapters (nats/evidence_gateway.go) | Transport encoding is adapter concern |
| Query serving | store (QueryResponderActor) | Store is the projection authority |
| KV read access | store (QueryResponderActor) | Single reader, no direct KV access from gateway |
| KV write access | store (projection actors) | Single writer per bucket |

## Adding a Query for a New Evidence Family

When a new evidence type is added (e.g., `volume`), the query chain requires:

1. **Contract** — `VolumeLatestQuery` / `VolumeLatestReply` in `evidenceclient/contracts.go`
2. **Use case** — `GetLatestVolumeUseCase` in `evidenceclient/get_latest_volume.go`
3. **Port method** — `GetLatestVolume()` on `EvidenceGateway` interface
4. **NATS adapter** — `GetLatestVolume()` on `nats.EvidenceGateway`
5. **Registry spec** — `VolumeLatest ControlSpec` in `EvidenceRegistry`
6. **HTTP handler** — `GetLatestVolume()` on `EvidenceWebHandler`
7. **Route** — `GET /evidence/volume/latest` in `routes/evidence.go`
8. **Deps field** — `GetLatestVolume` in `EvidenceFamilyDeps`
9. **Wiring** — use case creation in `cmd/gateway/run.go`

The pattern is fully additive — no existing query contract changes.
