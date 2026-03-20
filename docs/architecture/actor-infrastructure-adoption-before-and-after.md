# Actor Infrastructure Adoption: Before and After

> Comparing the store scope's actor topology before and after the S219 migration.

## Metrics Summary

| Metric | Before | After | Delta |
|---|---|---|---|
| Consumer actor files | 9 | 1 (`generic_consumer_actor.go`) | -8 files |
| Consumer Config types | 9 | 1 (`GenericConsumerConfig`) | -8 types |
| Consumer Actor types | 9 | 1 (`GenericConsumerActor`) | -8 types |
| Consumer constructors | 9 | 1 (`NewGenericConsumerActor`) | -8 functions |
| Consumer Receive methods | 9 | 1 | -8 methods |
| Consumer start methods | 9 | 1 | -8 methods |
| Lines of consumer actor code | ~720 | ~100 + ~110 closure lines | ~-510 lines |
| Projection actor files | 9 | 9 | 0 (unchanged) |
| Total store actor files | 22 | 14 | -8 files |

## File Inventory

### Deleted files (consumer actors)
```
internal/actors/scopes/store/evidence_consumer_actor.go      (93 lines)
internal/actors/scopes/store/trade_burst_consumer_actor.go    (93 lines)
internal/actors/scopes/store/volume_consumer_actor.go         (91 lines)
internal/actors/scopes/store/signal_consumer_actor.go         (91 lines)
internal/actors/scopes/store/decision_consumer_actor.go       (91 lines)
internal/actors/scopes/store/strategy_consumer_actor.go       (91 lines)
internal/actors/scopes/store/risk_consumer_actor.go           (91 lines)
internal/actors/scopes/store/execution_consumer_actor.go      (84 lines)
internal/actors/scopes/store/fill_consumer_actor.go           (84 lines)
```

### Retained files (unchanged)
```
internal/actors/scopes/store/generic_consumer_actor.go        (infrastructure)
internal/actors/scopes/store/store_supervisor.go              (updated: closures replace constructor calls)
internal/actors/scopes/store/messages.go                      (unchanged)
internal/actors/scopes/store/projection_store.go              (unchanged)
internal/actors/scopes/store/candle_projection_actor.go       (unchanged)
internal/actors/scopes/store/trade_burst_projection_actor.go  (unchanged)
internal/actors/scopes/store/volume_projection_actor.go       (unchanged)
internal/actors/scopes/store/signal_projection_actor.go       (unchanged)
internal/actors/scopes/store/decision_projection_actor.go     (unchanged)
internal/actors/scopes/store/strategy_projection_actor.go     (unchanged)
internal/actors/scopes/store/risk_projection_actor.go         (unchanged)
internal/actors/scopes/store/execution_projection_actor.go    (unchanged)
internal/actors/scopes/store/fill_projection_actor.go         (unchanged)
internal/actors/scopes/store/query_responder_actor.go         (unchanged)
```

## Architecture Topology

### Before: per-domain consumer actor types
```
StoreSupervisor
  ├── declarePipelines()
  │     ├── NewEvidenceConsumerActor(EvidenceConsumerConfig{...})
  │     ├── NewTradeBurstConsumerActor(TradeBurstConsumerConfig{...})
  │     ├── NewVolumeConsumerActor(VolumeConsumerConfig{...})
  │     ├── NewSignalConsumerActor(SignalConsumerConfig{...})
  │     ├── NewDecisionConsumerActor(DecisionConsumerConfig{...})
  │     ├── NewStrategyConsumerActor(StrategyConsumerConfig{...})
  │     ├── NewRiskConsumerActor(RiskConsumerConfig{...})
  │     ├── NewExecutionConsumerActor(ExecutionConsumerConfig{...})
  │     └── NewFillConsumerActor(FillConsumerConfig{...})
  │
  │   9 types × (Config + Actor + Constructor + Receive + start) = 45 definitions
  │
  └── [9 projection actors — each domain-specific]
```

### After: single generic consumer with closure-captured variance
```
StoreSupervisor
  ├── declarePipelines()
  │     ├── startConsumer("candle",      fn → CandleConsumer + candleReceivedMessage)
  │     ├── startConsumer("tradeburst",  fn → TradeBurstConsumer + tradeBurstReceivedMessage)
  │     ├── startConsumer("volume",      fn → VolumeConsumer + volumeReceivedMessage)
  │     ├── startConsumer("rsi",         fn → SignalConsumer + signalReceivedMessage)
  │     ├── startConsumer("ema_crossover", fn → SignalConsumer + signalReceivedMessage)
  │     ├── startConsumer("rsi_oversold", fn → DecisionConsumer + decisionReceivedMessage)
  │     ├── startConsumer("mean_reversion_entry", fn → StrategyConsumer + strategyReceivedMessage)
  │     ├── startConsumer("position_exposure", fn → RiskConsumer + riskReceivedMessage)
  │     ├── startConsumer("paper_order", fn → ExecutionConsumer + executionReceivedMessage)
  │     └── startConsumer("venue_market_order", fn → FillConsumer + fillReceivedMessage)
  │
  │   1 type × (Config + Actor + Constructor + Receive + start) = 5 definitions
  │   10 closures × ~8 lines each = ~80 lines of declaration
  │
  └── [9 projection actors — each domain-specific, unchanged]
```

## Key Design Decisions

1. **Closure capture over interface dispatch**: The `ConsumerStartFn` closure captures the registry and message routing at declaration time. This avoids the need for a shared consumer interface while preserving type safety within each closure.

2. **`startConsumer` helper**: A local helper in `declarePipelines()` that wires `ConsumerStartFn` into `NewGenericConsumerActor`, reducing the per-pipeline boilerplate to a single function call.

3. **Projection actors remain domain-specific**: Their domain logic (validation gates, dual-bucket writes, log field selection) differs enough that generalization would increase complexity without proportional reduction in code.

## Adding a New Pipeline

Before this migration, adding a new pipeline required:
1. Creating a new `*ConsumerConfig` struct
2. Creating a new `*ConsumerActor` struct
3. Implementing `NewXxxConsumerActor()`, `Receive()`, and `start()`
4. Adding the pipeline entry in `declarePipelines()`

After this migration, adding a new pipeline requires:
1. Adding a pipeline entry in `declarePipelines()` with a `startConsumer` closure
2. That's it — the closure is ~8 lines
