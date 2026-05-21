# Projection Families Model

> Canonical model for how the store organizes read-side projections by evidence family.

## Definition

A **projection family** is one evidence type's complete read-side pipeline in the store binary. It consists of:
1. A **durable JetStream consumer** filtering events by type-specific subject prefix
2. A **projection actor** materializing events into KV bucket(s)
3. **KV buckets** (latest, and optionally history)
4. **Query routes** in the shared QueryResponderActor
5. A **health tracker pair** (one for consumer, one for projection)

## The ProjectionPipeline Pattern

Each projection family is declared as a `ProjectionPipeline` struct in `StoreSupervisor.start()`:

```go
type ProjectionPipeline struct {
    Family         string                    // "candle", "tradeburst"
    ProjectionName string                    // actor child name
    ConsumerName   string                    // actor child name
    Buckets        []string                  // KV bucket names owned
    ConsumerSpec   adapternats.ConsumerSpec   // durable consumer config
    NewProjection  func(...) actor.Producer   // projection actor factory
    NewConsumer    func(...) actor.Producer   // consumer actor factory
}
```

### Registration Point

All projection families are registered in `StoreSupervisor.start()`. This is the **single point of truth** for which evidence types the store materializes:

```go
s.pipelines = []ProjectionPipeline{
    { Family: "candle",     ... },
    { Family: "tradeburst", ... },
}
```

### Spawning Loop

The supervisor spawns all pipelines uniformly:

```go
for _, p := range s.pipelines {
    projPID := ctx.SpawnChild(p.NewProjection(...), p.ProjectionName)
    ctx.SpawnChild(p.NewConsumer(..., projPID, ...), p.ConsumerName)
}
```

After all pipelines are spawned, the shared QueryResponderActor is spawned to serve queries for all families.

## Current Projection Families

### candle

```
EVIDENCE_EVENTS ──filter: candle.sampled.>──→ EvidenceConsumerActor (store-candle)
    │
    └──→ CandleProjectionActor
           ├──→ CANDLE_LATEST   (latest finalized candle per source/symbol/timeframe)
           └──→ CANDLE_HISTORY  (time-windowed archive, 24h TTL)
```

| Component | Value |
|-----------|-------|
| Family | `candle` |
| Consumer durable | `store-candle` |
| Projection actor | `CandleProjectionActor` |
| Buckets | `CANDLE_LATEST` (64 MB), `CANDLE_HISTORY` (256 MB, 24h TTL) |
| Query subjects | `evidence.query.candle.latest`, `evidence.query.candle.history` |
| Materialization | Final=true gate → Validate → Monotonicity guard → Write latest+history |

### tradeburst

```
EVIDENCE_EVENTS ──filter: tradeburst.sampled.>──→ TradeBurstConsumerActor (store-trade-burst)
    │
    └──→ TradeBurstProjectionActor
           └──→ TRADE_BURST_LATEST  (latest finalized burst per source/symbol/timeframe)
```

| Component | Value |
|-----------|-------|
| Family | `tradeburst` |
| Consumer durable | `store-trade-burst` |
| Projection actor | `TradeBurstProjectionActor` |
| Buckets | `TRADE_BURST_LATEST` (64 MB) |
| Query subjects | `evidence.query.tradeburst.latest` |
| Materialization | Final=true gate → Validate → Monotonicity guard → Write latest |

## Projection Family Invariants

1. **One consumer per family.** Each evidence type gets its own durable consumer with a type-specific filter subject. No shared consumer with type routing.

2. **One projection actor per family.** Each family gets its own actor that owns writes to its KV bucket(s). No generic projection framework.

3. **Single-writer per bucket.** Each KV bucket is written by exactly one projection actor. No cross-family bucket sharing.

4. **Shared query responder.** All families share one `QueryResponderActor` that registers typed routes per family. The responder opens read-only connections to all KV buckets.

5. **Independent health tracking.** Each family has its own health tracker pair (consumer + projection). Failures in one family do not mask issues in another.

6. **Final-only materialization.** Only events with `Final=true` are materialized. Interim snapshots exist only in the event stream.

7. **Monotonicity guard on latest.** Latest buckets check existing OpenTime before writing. Stale writes are silently skipped.

## What Varies Between Families

| Aspect | candle | tradeburst | Pattern |
|--------|--------|------------|---------|
| Bucket count | 2 (latest + history) | 1 (latest only) | Family-specific |
| History support | Yes (24h TTL) | No | Optional |
| Consumer actor type | `EvidenceConsumerActor` | `TradeBurstConsumerActor` | Type-specific |
| Projection actor type | `CandleProjectionActor` | `TradeBurstProjectionActor` | Type-specific |
| Query routes | 2 (latest + history) | 1 (latest) | Varies with bucket count |
| Materialization gates | Final + Validate + Monotonicity + History | Final + Validate + Monotonicity | History is optional |

## What Is Uniform Across Families

| Aspect | Value |
|--------|-------|
| Source stream | `EVIDENCE_EVENTS` |
| Consumer ack/retry | AckWait=30s, MaxDeliver=5 |
| KV storage type | FileStorage |
| Latest key format | `{source}.{symbol}.{timeframe}` |
| Query queue group | `evidence.query` |
| Health tracking | One tracker pair per family |
| Spawning pattern | projection actor first, consumer second (consumer receives projection PID) |

## Relationship to Derive Family Processors

The store's `ProjectionPipeline` mirrors derive's `FamilyProcessor`:

| Derive | Store |
|--------|-------|
| `FamilyProcessor` | `ProjectionPipeline` |
| Registered in `DeriveSupervisor.start()` | Registered in `StoreSupervisor.start()` |
| Spawned by `SourceScopeActor` per symbol/timeframe | Spawned by `StoreSupervisor` once per family |
| Each produces to shared `EVIDENCE_EVENTS` | Each consumes from shared `EVIDENCE_EVENTS` |
| One sampler actor per family × symbol × timeframe | One consumer + projection actor per family |

The derive side scales by symbol/timeframe (horizontal). The store side scales by family count (vertical). Both use declarative registration with factory functions.
