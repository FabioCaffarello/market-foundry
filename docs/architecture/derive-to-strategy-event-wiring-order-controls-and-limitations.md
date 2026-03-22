# Derive-to-Strategy Event Wiring: Order, Controls, and Limitations

> S366 deliverable. Documents the ordering guarantees, control mechanisms, and explicit limitations of the canonical derive producer wiring.

## Ordering Guarantees

### Within a Single Source/Symbol/Timeframe Partition

Events flow sequentially through the actor chain:

```
Trade → Candle → Signal → Decision → Strategy → Risk → Execution
```

Within a single partition (one source + symbol + timeframe combination), the actor model guarantees:

1. **Mailbox ordering**: Each actor processes messages in FIFO order.
2. **No concurrent processing**: One message at a time per actor instance.
3. **Deterministic resolution**: Pure `Resolve()` function — same input always produces same output.

**Result**: Strategy events for a given partition are produced in the order their upstream decisions arrive.

### Across Partitions

No ordering guarantee exists across different source/symbol/timeframe combinations. Each partition runs independently through its own actor chain. NATS JetStream preserves per-subject publish order but does not guarantee cross-subject ordering.

### Timestamp Semantics

Strategy timestamps are **source-derived** (from the decision's timestamp, which traces back to the original trade). They are NOT wall-clock time. This means:

- Consumers can use timestamps for monotonicity guards.
- Timestamps reflect market time, not processing time.
- Out-of-order processing (rare) would still produce correctly-timestamped events.

## Control Mechanisms

### 1. Configuration-Driven Enablement

| Control | Scope | Mechanism |
|---|---|---|
| `pipeline.strategy_families` | Per binary instance | List of enabled strategy families in config |
| `IsStrategyFamilyEnabled(name)` | Per family | Checked at supervisor startup |
| Family not in list | Global | Actor never spawned — no events produced |

### 2. Binding-Driven Activation

| Control | Scope | Mechanism |
|---|---|---|
| `BindingWatcherActor` | Per source/symbol | Queries configctl for active bindings at startup |
| `IngestionRuntimeChangedEvent` | Per source/symbol | NATS subscription for runtime binding changes |
| `activateSamplerMessage` | Per source/symbol | Triggers `SourceScopeActor` creation on demand |

### 3. Validation Gates

| Gate | Location | Behavior on Failure |
|---|---|---|
| `Resolve()` returns false | ResolverActor | Silent return — no event produced |
| `Strategy.Validate()` fails | ResolverActor | Error logged — no event published |
| Unknown strategy type | Publisher.specForType() | Returns problem (InvalidArgument) |
| NATS publish failure | Publisher.PublishStrategy() | Error logged, health tracker records failure |
| JetStream unavailable | Publisher | Poison pill — actor shuts down |

### 4. Health Observability

| Signal | Mechanism |
|---|---|
| Publish success | `healthz.Tracker.RecordEvent()` |
| Publish failure | `healthz.Tracker.RecordError()` |
| Per-symbol counters | `published:{symbol}` |
| Per-family counters | `strategy:{type}:{direction}` |
| Health endpoint | HTTP readiness checks (NATS connectivity) |

## Explicit Limitations

### L-1: No At-Most-Once Delivery

JetStream provides at-least-once delivery. Duplicate events are possible during NATS reconnection or JetStream redelivery. Consumers must be idempotent. The deduplication key (`strat:{type}:{source}:{symbol}:{timeframe}:{unix_ts}`) provides a JetStream-level dedup window, but consumers should also implement their own idempotency (store uses monotonicity guard).

### L-2: No Cross-Partition Ordering

Events for `btcusdt` and `ethusdt` (or different timeframes) may arrive in any order at consumers. No global sequencing exists.

### L-3: No Backpressure from Execution

If execution or store consumers fall behind, strategy production continues unaffected. There is no flow control mechanism between producer and consumers.

### L-4: No Retry at Publisher Level

If a JetStream publish fails, the error is logged and the health tracker records it. The publisher does not retry. The event is lost for that publish attempt. The next decision will produce a new strategy event.

### L-5: No Multi-Decision Aggregation

Mean reversion entry uses exactly one decision input. There is no mechanism to wait for or combine multiple decisions before resolving.

### L-6: No Confidence Threshold Gate

All confidence levels produce events, including zero confidence (flat direction). Consumers are responsible for filtering by confidence if needed.

### L-7: No Per-Strategy-Type Gate

Only the global `pipeline.strategy_families` toggle exists. There is no per-symbol or per-timeframe strategy enablement.

### L-8: No Event Expiration Signal

Events persist in JetStream for 72 hours and are then silently dropped. No expiration event is produced.

### L-9: No Short Direction for RSI

The mean reversion entry resolver maps `triggered` to `long` only. There is no RSI overbought → short mapping in the current implementation.

### L-10: No Position-Size Awareness

Strategy resolution has no concept of current position or portfolio state. It resolves purely from the decision input.

### L-11: No Rate Limiting

Every decision produces a strategy event. There is no debounce or rate limiting at the producer side.

## Dependency Chain

```
Binding activation
  └─ Evidence sampler (candle)
       └─ Signal sampler (RSI)
            └─ Decision evaluator (RSI oversold)
                 └─ Strategy resolver (mean_reversion_entry) ← THIS STAGE
                      └─ Risk evaluator (position_exposure / drawdown_limit)
                           └─ Execution evaluator (paper_order)
```

Each stage depends on upstream producing events. A disabled or misconfigured upstream stage silently prevents downstream production.

## Error Propagation

| Error | Impact | Recovery |
|---|---|---|
| Invalid confidence in decision | Strategy not produced (resolver returns false) | Next valid decision resumes production |
| Strategy validation failure | Event not published (logged) | Fix resolver logic or decision input |
| NATS connection lost | Publisher actor poisoned | Actor system restarts publisher on reconnect |
| JetStream publish timeout | Event lost for this attempt | Next decision produces new event |
| Stream does not exist | Publisher startup fails | Derive binary fails preflight |
