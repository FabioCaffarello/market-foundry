# Multi-Projection Pattern

> How the store service manages multiple evidence projection pipelines without collapsing into a monolith.

## Core Principle

Each evidence type owns a **self-contained projection pipeline**: its own consumer, projection actor, KV bucket(s), and health trackers. Pipelines share infrastructure (NATS connection, evidence stream, query responder) but are isolated in logic and state.

## Pipeline Anatomy

Every evidence projection follows this structure:

```
EVIDENCE_EVENTS stream
  ↓ durable consumer (type-specific filter)
[ConsumerActor]
  ↓ actor message
[ProjectionActor]
  ↓ Final gate → Validate gate → Monotonicity guard
[KV Bucket(s)]
  ↓
[QueryResponderActor]
  ↓ typed control route
[Gateway → HTTP]
```

## What Is Shared

| Shared resource | Scope | Reason |
|-----------------|-------|--------|
| `EVIDENCE_EVENTS` stream | All evidence types | Single stream with subject-based routing (`evidence.events.>`). Each consumer filters by type prefix. |
| `QueryResponderActor` | All query routes | Single responder manages all KV read connections and registers all control routes. Avoids NATS connection proliferation on the read path. |
| `EvidenceRegistry` | All NATS specs | Central registry of event specs, consumer specs, and control specs. Type-safe, no dynamic dispatch. |
| `evidence.query` queue group | All query subjects | Single horizontal scaling group for all evidence queries. |

## What Is Isolated

| Isolated resource | Per evidence type | Reason |
|-------------------|-------------------|--------|
| Durable consumer | `store-evidence`, `store-trade-burst` | Independent stream position. Replay/reset of one type doesn't affect others. |
| Projection actor | `CandleProjectionActor`, `TradeBurstProjectionActor` | Independent domain validation, write logic, and stats tracking. |
| KV bucket(s) | `CANDLE_LATEST`, `CANDLE_HISTORY`, `TRADE_BURST_LATEST` | No cross-type interference. Each bucket has its own retention and capacity. |
| Health tracker pair | `candle-projection` + `candle-consumer`, `trade-burst-projection` + `trade-burst-consumer` | `/statusz` shows health per evidence type independently. |
| Consumer actor | `EvidenceConsumerActor`, `TradeBurstConsumerActor` | Each type decodes its own event type. No shared deserialization. |

## Scaling Properties

| Dimension | Model |
|-----------|-------|
| Add evidence type | New pipeline: consumer + projection + bucket + tracker pair. Existing pipelines untouched. |
| Add symbol/timeframe | More keys in existing buckets. No structural change. |
| Horizontal query scaling | More store instances → more queue group members. All query routes scale together. |
| Independent replay | Reset one durable consumer → replay one evidence type only. |

## Supervisor Structure

The `StoreSupervisor` spawns all pipeline actors explicitly:

```
StoreSupervisor
  ├── CandleProjectionActor     (tracker: candle-projection)
  ├── EvidenceConsumerActor      (tracker: candle-consumer, filter: candle.sampled.>)
  ├── TradeBurstProjectionActor  (tracker: trade-burst-projection)
  ├── TradeBurstConsumerActor    (tracker: trade-burst-consumer, filter: tradeburst.sampled.>)
  └── QueryResponderActor        (reads all KV buckets, serves all queries)
```

Adding a new evidence type means adding 2 actors to the supervisor (consumer + projection) and 1 control route to the query responder. The supervisor logs a projection inventory on startup listing all active pipelines.

## Gateway Integration

The gateway's `Dependencies` struct holds one use case field per evidence query:

```go
type Dependencies struct {
    // ... other deps ...
    GetLatestCandle     handlersGetLatestCandleUseCase
    GetCandleHistory    handlersGetCandleHistoryUseCase
    GetLatestTradeBurst handlersGetLatestTradeBurstUseCase
}
```

Route registration is conditional — nil use cases produce no routes. The gateway stays clean because:
- Each evidence query is a separate endpoint under `/evidence/{type}/...`
- All evidence handlers share a common `parseEvidenceKeyParams` helper for source/symbol/timeframe
- The `EvidenceGateway` NATS adapter implements all query methods on a single connection

## Anti-Patterns Avoided

| Anti-pattern | Why avoided |
|-------------|-------------|
| Generic projection framework | Each type has specific validation, write semantics (history vs latest-only), and stats. A generic framework would hide these differences. |
| Shared consumer with type routing | Would couple all evidence types to one consumer's stream position. Independent consumers allow independent replay. |
| Dynamic projection registration | Projections are known at compile time. Dynamic registration adds complexity without value when the type set is small and stable. |
| Cross-type KV buckets | Mixing candle and burst data in one bucket would break prefix scans and TTL policies. |

## Limits

1. **Query responder grows linearly** — each evidence type adds a KV connection and a control route. At 10+ types, consider splitting into per-type responders.
2. **No projection discovery** — no runtime API to list active projections. The startup log is the only inventory. Acceptable while the type set is small.
3. **Shared queue group** — all evidence queries share `evidence.query`. At very high query volume, per-type queue groups may be needed for isolation.
