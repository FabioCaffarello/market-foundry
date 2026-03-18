# Evidence Query Model Consolidation

> Defines the canonical query model for all evidence types in market-foundry. Consolidates naming, contracts, subjects, and HTTP paths into a coherent taxonomy.

## Query Model Overview

Every evidence type exposes queries through a three-layer stack:

```
HTTP (gateway)  →  NATS request/reply (store)  →  KV bucket (projection)
```

Each layer uses consistent naming derived from the evidence type name.

## Naming Taxonomy

### Evidence Type Names

| Type | NATS segment | Go prefix | HTTP segment | KV bucket prefix |
|------|-------------|-----------|-------------|-----------------|
| Candle | `candle` | `Candle` | `candles` | `CANDLE_` |
| Trade Burst | `tradeburst` | `TradeBurst` | `tradeburst` | `TRADE_BURST_` |

**Rules:**
- NATS subjects: single lowercase word per type segment (`candle`, `tradeburst`)
- Go types: PascalCase compound (`CandleLatestQuery`, `TradeBurstLatestQuery`)
- NATS type strings: snake_case (`candle_sampled`, `trade_burst_sampled`)
- HTTP paths: lowercase, plural for candle (`candles`), compound for trade burst (`tradeburst`)
- KV buckets: SCREAMING_SNAKE_CASE (`CANDLE_LATEST`, `TRADE_BURST_LATEST`)

### Consumer Durable Names

| Type | Durable name | Pattern |
|------|-------------|---------|
| Candle | `store-candle` | `store-{type}` |
| Trade Burst | `store-trade-burst` | `store-{type-hyphenated}` |

**Rule:** All durable names use hyphen-separated words. No camelCase in NATS durables.

## Current Query Inventory

### Candle Queries

| Query | HTTP | NATS Subject | Contract |
|-------|------|-------------|----------|
| Latest | `GET /evidence/candles/latest` | `evidence.query.candle.latest` | `CandleLatestQuery → CandleLatestReply` |
| History | `GET /evidence/candles/history` | `evidence.query.candle.history` | `CandleHistoryQuery → CandleHistoryReply` |

**Parameters (Latest):** `source`, `symbol`, `timeframe`
**Parameters (History):** `source`, `symbol`, `timeframe`, `limit`, `since`, `until`

### Trade Burst Queries

| Query | HTTP | NATS Subject | Contract |
|-------|------|-------------|----------|
| Latest | `GET /evidence/tradeburst/latest` | `evidence.query.tradeburst.latest` | `TradeBurstLatestQuery → TradeBurstLatestReply` |

**Parameters:** `source`, `symbol`, `timeframe`

## Query Contract Structure

All evidence queries share a common key: `(source, symbol, timeframe)`. This is enforced by:
- `parseEvidenceKeyParams(r)` in the HTTP handler layer
- Validation in each use case (`source required`, `symbol required`, `timeframe > 0`)

### Latest Query Pattern

```go
// Request
type {Type}LatestQuery struct {
    Source    string `json:"source"`
    Symbol   string `json:"symbol"`
    Timeframe int   `json:"timeframe"`
}

// Reply
type {Type}LatestReply struct {
    {Type} *evidence.Evidence{Type} `json:"{json_key},omitempty"`
}
```

Reply uses pointer + `omitempty` so the response includes `null` when no data exists yet.

### History Query Pattern (candle only, currently)

```go
type CandleHistoryQuery struct {
    Source    string `json:"source"`
    Symbol   string `json:"symbol"`
    Timeframe int   `json:"timeframe"`
    Limit     int   `json:"limit"`
    Since     int64  `json:"since,omitempty"`
    Until     int64  `json:"until,omitempty"`
}

type CandleHistoryReply struct {
    Candles []evidence.EvidenceCandle `json:"candles"`
}
```

History replies use a slice (empty `[]`, never null) with no `omitempty`.

## Asymmetries (Intentional)

| Asymmetry | Reason | Resolution path |
|-----------|--------|----------------|
| Candle has history; Trade Burst does not | Trade burst was introduced as latest-only proof of pattern. History follows when needed. | Add `TRADE_BURST_HISTORY` bucket following candle history pattern. |
| HTTP plural (`candles`) vs compound (`tradeburst`) | Candle is a natural English plural. "Trade bursts" would require a hyphen or look awkward. | Acceptable divergence. Both are stable. |
| Candle consumer named `EvidenceConsumerActor` | Legacy name from when candles were the only evidence. Type alias kept for backwards compatibility. | Rename to `CandleConsumerActor` when justified by a larger refactor. |

## Consolidation Rules Applied (S25)

| Rule | Before | After |
|------|--------|-------|
| Consumer durable names use hyphens | `store-evidence`, `store-tradeburst` | `store-candle`, `store-trade-burst` |
| Actor logger matches spawn name | Logger: `evidence-consumer`, Spawn: `candle-consumer` | Both: `candle-consumer` |
| Registry function matches type | `StoreEvidenceConsumer()` | `StoreCandleConsumer()` |
| Consumer spec tests verify naming | No tests for durable names | Tests verify hyphen-separated format |
