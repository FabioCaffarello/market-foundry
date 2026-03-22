# Store, Gateway, and Read-Path Verification for Derive-Produced Strategy Events

> S367 — Derive Integration Wave (DI-3)

## Purpose

This document verifies the complete read-path for `StrategyResolvedEvent` produced by the derive binary, from NATS publication through store materialization to gateway HTTP surfaces.

## Read-Path Architecture

```
derive binary
  └─ StrategyResolverActor → StrategyPublisherActor
       └─ natsstrategy.Publisher.PublishStrategy()
            └─ NATS JetStream: strategy.events.{family}.resolved.{source}.{symbol}.{timeframe}
                 └─ STRATEGY_EVENTS stream (FileStorage, 72h, 256 MB)

store binary
  └─ StoreSupervisor → StrategyConsumerActor (durable: store-strategy-mean-reversion-entry)
       └─ natsstrategy.Consumer → decodes CBOR → strategyReceivedMessage
            └─ StrategyProjectionActor
                 ├─ Gate 1: Final == true
                 ├─ Gate 2: strategy.Validate()
                 └─ KVStore.Put() → NATS KV: STRATEGY_MEAN_REVERSION_ENTRY_LATEST
                      └─ Monotonicity guard: Timestamp comparison

  └─ QueryResponderActor
       └─ NATS request/reply: strategy.query.mean_reversion_entry.latest
            └─ KVStore.Get() → strategy.Strategy

gateway binary
  └─ natsstrategy.Gateway (NATS RequestClient)
       └─ GetLatestStrategy() → request/reply → decode StrategyLatestReply

  └─ HTTP: GET /strategy/:type/latest?source=...&symbol=...&timeframe=...
       └─ StrategyWebHandler.GetLatestStrategy()
            └─ GetLatestStrategyUseCase.Execute()
                 └─ gateway.GetLatestStrategy()
```

## Verified Components

### 1. Publisher → NATS Stream

| Property | Value | Status |
|----------|-------|--------|
| Subject pattern | `strategy.events.{family}.resolved.{source}.{symbol}.{timeframe}` | VERIFIED |
| Encoding | CBOR via `natskit.EncodeEvent()` | VERIFIED |
| Dedup key | `strat:{type}:{source}:{symbol}:{timeframe}:{unix_ts}` | VERIFIED |
| Stream | `STRATEGY_EVENTS` (FileStorage, 72h, 256 MB) | VERIFIED |
| Correlation/Causation | Propagated in event metadata | VERIFIED |

### 2. Store Consumer → Projection Actor

| Property | Value | Status |
|----------|-------|--------|
| Consumer durable | `store-strategy-mean-reversion-entry` | VERIFIED |
| Filter subject | `strategy.events.mean_reversion_entry.resolved.>` | VERIFIED |
| AckWait | 30 seconds | VERIFIED |
| MaxDeliver | 5 | VERIFIED |
| Subject alignment | Consumer `>` wildcard matches publisher pattern | VERIFIED |

### 3. Projection Actor → KV Store

| Gate | Behavior | Status |
|------|----------|--------|
| Final gate | Skips strategies with `Final != true` | VERIFIED (test) |
| Validation gate | Rejects malformed strategies | VERIFIED (test) |
| Monotonicity guard | Skips stale (older timestamp) | VERIFIED (test) |
| Dedup guard | Skips duplicate (equal timestamp) | VERIFIED (test) |
| Stats invariant | `received == sum(outcomes)` | VERIFIED (test) |

KV Bucket: `STRATEGY_MEAN_REVERSION_ENTRY_LATEST`
- Storage: FileStorage, 64 MB max
- Key format: `{source}.{symbol}.{timeframe}`
- Value format: JSON-encoded `strategy.Strategy`

### 4. Query Responder → Gateway

| Property | Value | Status |
|----------|-------|--------|
| Query subject | `strategy.query.mean_reversion_entry.latest` | VERIFIED |
| Request type | `strategy.query.v1.mean_reversion_entry_latest_request` | VERIFIED |
| Reply type | `strategy.query.v1.mean_reversion_entry_latest_reply` | VERIFIED |
| Queue group | `strategy.query` | VERIFIED |
| Handler | `handleStrategyMeanReversionEntryLatest` | VERIFIED |

### 5. Gateway → HTTP Surface

| Property | Value | Status |
|----------|-------|--------|
| Route | `GET /strategy/:type/latest` | VERIFIED |
| Query params | `source`, `symbol`, `timeframe` | VERIFIED |
| Use case validation | type, source, symbol required; timeframe > 0 | VERIFIED (test) |
| Null strategy | Returns `{"strategy": null}` with 200 OK | VERIFIED (test) |
| Unavailable handler | Returns 503 | VERIFIED (test) |

## Field Preservation Through Read-Path

The following fields are verified to survive the full round-trip (publish → persist → read):

| Field | Preserved | Notes |
|-------|-----------|-------|
| `type` | Yes | Core identity |
| `source` | Yes | Partition key component |
| `symbol` | Yes | Partition key component |
| `timeframe` | Yes | Partition key component |
| `direction` | Yes | long/short/flat |
| `confidence` | Yes | String decimal |
| `decisions[]` | Yes | Full decision input array |
| `decisions[].severity` | Yes | Severity depth carried |
| `decisions[].rationale` | Yes | Rationale carried |
| `parameters` | Yes | Type-specific params |
| `metadata` | Yes | Domain metadata (resolver info) |
| `final` | Yes | Always true when materialized |
| `timestamp` | Yes | Used for monotonicity |

## Event Metadata Gap

**Critical finding**: Event-level metadata (`correlation_id`, `causation_id`, `occurred_at`, `id`) is NOT persisted in the KV store.

- The projection actor receives `StrategyResolvedEvent` (with `events.Metadata`)
- It passes only `msg.Event.Strategy` to `KVStore.Put()`
- The KV store serializes `strategy.Strategy` — no event metadata
- Correlation/causation ARE logged at materialization time (structured log)
- They are NOT available in the read surface (HTTP response)

**Mitigations**:
1. The analytical path (ClickHouse) preserves full event metadata when the writer pipeline is active
2. NATS JetStream retains the raw event (with metadata) for 72 hours
3. Structured logs capture correlation/causation at materialization time

## Contract Alignment Summary

| Contract | Publisher | Store Consumer | Query Responder | Gateway |
|----------|-----------|----------------|-----------------|---------|
| NATS subject | `strategy.events.mre.resolved.{key}` | `strategy.events.mre.resolved.>` | N/A | N/A |
| NATS query | N/A | N/A | `strategy.query.mre.latest` | `strategy.query.mre.latest` |
| Stream | `STRATEGY_EVENTS` | `STRATEGY_EVENTS` | N/A | N/A |
| KV bucket | N/A | `STRATEGY_MRE_LATEST` | `STRATEGY_MRE_LATEST` | N/A |
| Data format | CBOR event | CBOR → domain struct | JSON from KV | JSON via reply |

All contracts align. No mismatches found.
