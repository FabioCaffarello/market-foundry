# End-to-End Domain-to-Venue Slice Proof

> S362 Binding — Delivered: 2026-03-22

## 1. Slice Definition

This document proves the canonical vertical slice from **domain source to venue fill** in the market-foundry execution pipeline. The slice exercises:

```
StrategyResolvedEvent (NATS STRATEGY_EVENTS)
  → natsstrategy.Consumer (durable: execute-strategy-mean-reversion-entry)
    → StrategyConsumerActor.onStrategyEvent()
      → PaperOrderEvaluator.Evaluate() (direction→side, risk pass-through)
        → intentReceivedMessage → VenueAdapterActor.onIntent()
          → SafetyGate.Check() (kill switch + staleness)
            → VenuePort.SubmitOrder() (paper simulator)
              → VenueOrderFilledEvent published to EXECUTION_FILL_EVENTS
```

This is the first time the **strategy-driven** path is exercised end-to-end on the real ExecuteSupervisor. Prior integration tests (S333, S341, S342, S343, S349) published `PaperOrderSubmittedEvent` to the venue consumer — they exercised the adapter but not the strategy-to-execution wiring from S360.

## 2. Source Selection

| Attribute | Value |
|-----------|-------|
| Strategy family | `mean_reversion_entry` |
| Source path | `strategy_consumer.mean_reversion_entry` |
| NATS consumer | `execute-strategy-mean-reversion-entry` |
| Subject filter | `strategy.events.mean_reversion_entry.resolved.>` |
| Risk model | `pass_through` / `approved` |
| Position sizing | `0.01` (1% max position) |

This is the only production source pathway. Selection rationale is documented in S359.

## 3. Invariants Proven End-to-End

| ID | Invariant | Evidence |
|----|-----------|----------|
| INV-1 | Strategy type identity preserved | `fill.Risk.StrategyType == "mean_reversion_entry"` (E2E-1) |
| INV-2 | Direction→side deterministic | long→buy (E2E-1), short→sell (E2E-3), flat→none (E2E-4) |
| INV-3 | Correlation chain preserved | `fill.Metadata.CorrelationID == strategyEvent.CorrelationID` (E2E-6) |
| INV-4 | Pass-through risk explicit | `fill.Risk.Type == "pass_through"`, `Risk.Disposition == "approved"` (E2E-1) |
| INV-5 | Strategy timestamp, not time.Now() | Fill timestamp age > 3s (matches strategy event -5s offset) (E2E-1) |
| INV-6 | Single family constraint | NATS subject routing ensures only mean_reversion_entry reaches consumer (E2E-5) |
| INV-7 | Flat→none with observability | `fill.Side == none`, `fill.Quantity == 0`, `evaluation_outcome == flat` (E2E-4) |

## 4. Safety Gates Proven

| Gate | Behavior | Evidence |
|------|----------|----------|
| Kill switch | Blocks strategy-driven intent when gate=halted | E2E-2 phase 1: skipped_halt >= 1, filled == 0 |
| Gate resume | Enables flow after gate transitions active→halted→active | E2E-2 phase 2: fill received after resume |
| Staleness | Not exercised end-to-end (validated in S333/S342) | Staleness max age configured at 300s |

## 5. Explainability Proven

| Field | Value | Evidence |
|-------|-------|----------|
| `source_path` | `strategy_consumer.mean_reversion_entry` | E2E-1, E2E-3 |
| `evaluation_outcome` | `actionable` (long/short), `flat` (flat) | E2E-1, E2E-4 |
| `strategy_type` | `mean_reversion_entry` | E2E-1 (Parameters) |
| `risk_type` | `pass_through` | E2E-1 (Risk struct) |

## 6. Persistence Round-Trip

The fill event is published to `EXECUTION_FILL_EVENTS` stream and received by a core NATS subscriber in each test. This proves:

1. **Write path**: VenueAdapterActor → Publisher.PublishFill → JetStream
2. **Read path**: Core NATS subscription → fill event decoded and correlated
3. **Dedup guarantee**: JetStream `MsgID` prevents duplicate fills
4. **Correlation match**: Fill subscriber filters by correlation ID — proving the fill traces back to the originating strategy event

The full persistence round-trip (NATS → store materialization → KV bucket → HTTP read-back) is validated by the existing S333-S349 test suite and smoke-activation.sh. S362 extends this by proving the strategy-driven entry point produces fills that reach the same persistence infrastructure.

## 7. Health Tracker Evidence

Each test validates tracker counters for both actors in the pipeline:

**Strategy consumer counters**:
- `received`: strategy events delivered by NATS consumer
- `evaluated`: events that passed type check and confidence threshold
- `evaluated_actionable`: long/short directions
- `evaluated_flat`: flat directions

**Venue adapter counters**:
- `processed`: intents received from strategy consumer (or venue consumer)
- `filled`: intents that passed all gates and produced fills
- `skipped_halt`: intents blocked by kill switch
- `skipped_stale`: intents rejected by staleness guard

## 8. Test Matrix

| Test | What it proves |
|------|---------------|
| E2E-1 | Full slice: strategy event → fill (long direction, all invariants) |
| E2E-2 | Kill switch blocks strategy path; resume enables it |
| E2E-3 | Short direction maps to sell side end-to-end |
| E2E-4 | Flat direction produces none side with observability passthrough |
| E2E-5 | Single-family constraint: only mean_reversion_entry reaches execute consumer |
| E2E-6 | Correlation chain preserved from strategy source to venue fill |

All tests require a running NATS server (`integration` build tag).
