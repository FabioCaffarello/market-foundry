# Controlled Source-to-Execution Wiring

> S360 — Strategy Signal Integration Wave (SSI-2)

## Purpose

This document describes how the canonical source (RSI signal + Mean Reversion Entry strategy) is wired to the active execution path. The wiring connects the strategy domain's output to the execution domain's intake without breaking control gates, activation semantics, or auditability.

## Wiring Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│  derive binary                                                       │
│                                                                      │
│  SignalSampler → SignalEvent → DecisionEvaluator → DecisionEvent    │
│  → StrategyResolver → StrategyResolvedEvent                         │
│                          │                                           │
│                          ▼                                           │
│              NATS: STRATEGY_EVENTS stream                            │
│              Subject: strategy.events.mean_reversion_entry.resolved.>│
└──────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│  execute binary                                                      │
│                                                                      │
│  ┌─────────────────────────────────┐                                │
│  │ natsstrategy.Consumer           │                                │
│  │ Durable: execute-strategy-      │                                │
│  │   mean-reversion-entry          │                                │
│  │ Decodes StrategyResolvedEvent   │                                │
│  └────────────┬────────────────────┘                                │
│               │ handler closure (supervisor)                         │
│               ▼                                                      │
│  ┌─────────────────────────────────┐                                │
│  │ StrategyConsumerActor           │                                │
│  │ • Validates strategy type       │                                │
│  │ • Evaluates via PaperOrder-     │                                │
│  │   Evaluator (pass-through risk) │                                │
│  │ • Preserves correlation chain   │                                │
│  │ • Produces synthetic            │                                │
│  │   PaperOrderSubmittedEvent      │                                │
│  └────────────┬────────────────────┘                                │
│               │ intentReceivedMessage                                │
│               ▼                                                      │
│  ┌─────────────────────────────────┐                                │
│  │ VenueAdapterActor (existing)    │                                │
│  │ • Kill switch gate              │                                │
│  │ • Staleness guard (120s)        │                                │
│  │ • Composed submit pipeline      │                                │
│  │   (Retry + Post200Reconciler)   │                                │
│  │ • Fill event publication        │                                │
│  └─────────────────────────────────┘                                │
└──────────────────────────────────────────────────────────────────────┘
```

## Integration Points

### 1. NATS Consumer Spec

| Property | Value |
|----------|-------|
| Durable | `execute-strategy-mean-reversion-entry` |
| Stream | `STRATEGY_EVENTS` |
| Subject | `strategy.events.mean_reversion_entry.resolved.>` |
| Type | `strategy.events.v1.mean_reversion_entry_resolved` |
| AckWait | 30s |
| MaxDeliver | 5 |

Defined in `internal/adapters/nats/natsstrategy/registry.go` as `ExecuteStrategyMeanReversionEntryConsumer()`.

### 2. Strategy-to-Intent Evaluation

The `StrategyConsumerActor` transforms a `StrategyResolvedEvent` into an `ExecutionIntent` via `PaperOrderEvaluator`:

| ExecutionIntent Field | Source | Value |
|----------------------|--------|-------|
| Type | Default | `"paper_order"` |
| Source | Strategy | `strategy.Source` |
| Symbol | Strategy | `strategy.Symbol` |
| Timeframe | Strategy | `strategy.Timeframe` |
| Side | Computed | Direction mapping (long→buy, short→sell, flat→none) |
| Quantity | Config | `MaxPositionPct` (default: `"0.01"`) |
| Risk.Type | Default | `"pass_through"` (INV-4) |
| Risk.Disposition | Default | `"approved"` (INV-4) |
| Risk.Confidence | Strategy | `strategy.Confidence` |
| Risk.StrategyType | Strategy | `strategy.Type` (INV-1) |
| Risk.DecisionSeverity | Strategy | `strategy.Decisions[0].Severity` |
| Risk.Timeframe | Strategy | `strategy.Timeframe` |
| CorrelationID | Event | `event.Metadata.CorrelationID` (INV-3) |
| CausationID | Event | `event.Metadata.ID` (INV-3) |
| Timestamp | Strategy | `strategy.Timestamp` (INV-5, not `time.Now()`) |

### 3. Actor Topology

```
ExecuteSupervisor
├── VenueAdapterActor (existing — receives intents from both paths)
│   ├── natsexecution.Consumer (paper_order intake — transitional bridge)
│   └── Safety gates + composed submit pipeline
├── StrategyConsumerActor (S360 — evaluates strategy events)
│   └── Forwards intentReceivedMessage → VenueAdapterActor
└── natsstrategy.Consumer (strategy event intake — S360)
```

### 4. Health Tracking

| Tracker | Counters |
|---------|----------|
| `strategy-consumer` | `received`, `evaluated`, `evaluated_flat`, `evaluated_actionable`, `skipped_wrong_type`, errors |

Exposed via the existing health server at the execute binary's HTTP addr.

## Invariants Preserved

| ID | Invariant | How Preserved |
|----|-----------|---------------|
| INV-1 | Strategy type identity | `Risk.StrategyType == strategy.Type` — direct assignment |
| INV-2 | Deterministic direction-to-side | PaperOrderEvaluator is a pure function |
| INV-3 | Correlation/causation chain | `intent.CorrelationID = event.Metadata.CorrelationID`, `intent.CausationID = event.Metadata.ID` |
| INV-4 | Explicit pass-through risk | `riskType = "pass_through"`, `riskDisposition = "approved"` — never empty |
| INV-5 | Strategy timestamp used | `ts = strategy.Timestamp` — not `time.Now()` |
| INV-6 | Single strategy family | Actor validates `strategy.Type == "mean_reversion_entry"` and skips others |
| INV-7 | Flat produces no execution | Side=none, Quantity=0 (intent still forwarded for observability) |
| INV-8 | Ack after successful publication | natsstrategy.Consumer acks only after handler returns |
| INV-9 | Kill switch applies | Intent flows through VenueAdapterActor safety gates unchanged |
| INV-10 | Staleness guard applies | VenueAdapterActor checks intent.Timestamp age (120s max) |
| INV-11 | Deduplication keys unique | `ExecutionIntent.DeduplicationKey()` uses strategy timestamp |

## Design Decisions

### Synthetic PaperOrderSubmittedEvent

The strategy consumer actor wraps its produced `ExecutionIntent` in a `PaperOrderSubmittedEvent` before sending to the venue adapter. This reuses the entire existing safety gate + submit pipeline without code duplication. The event is "synthetic" in that it was not published by derive's paper order path, but carries identical semantics.

**Trade-off**: This creates a coupling to the paper_order event type. When a dedicated strategy-driven event type is introduced (future SSI stage), this bridge will be replaced.

### Actor vs. Inline Evaluation

The strategy evaluation logic lives in a dedicated `StrategyConsumerActor` rather than inline in the supervisor's handler closure. This provides:
- Testability (actor can be spawned independently with mock PIDs)
- Health tracking isolation (separate counter namespace)
- Clear lifecycle management (stats on stop)

### No New NATS Streams

The wiring does not introduce new NATS streams or event types. Strategy events are consumed from the existing `STRATEGY_EVENTS` stream. Execution intents flow through the existing venue adapter pipeline. Fill events are published to the existing `EXECUTION_FILL_EVENTS` stream.

## Files Changed

| File | Change |
|------|--------|
| `internal/adapters/nats/natsstrategy/registry.go` | Added `ExecuteStrategyMeanReversionEntryConsumer()` |
| `internal/actors/scopes/execute/messages.go` | Added `strategyReceivedMessage` |
| `internal/actors/scopes/execute/strategy_consumer_actor.go` | New: strategy evaluation actor |
| `internal/actors/scopes/execute/strategy_consumer_actor_test.go` | New: 11 invariant-covering tests |
| `internal/actors/scopes/execute/execute_supervisor.go` | Wired strategy consumer actor + natsstrategy.Consumer |
| `cmd/execute/run.go` | Added `strategy-consumer` health tracker |
