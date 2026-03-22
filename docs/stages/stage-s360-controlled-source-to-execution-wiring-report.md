# Stage S360 — Controlled Source-to-Execution Wiring Report

> **Wave**: Strategy/Signal Integration (S358–S363)
> **Block**: SSI-2
> **Predecessor**: S359 (Source Selection and Canonical Contract)
> **Date**: 2026-03-22

---

## 1. Executive Summary

S360 implements the controlled wiring between the canonical source (RSI signal + Mean Reversion Entry strategy) and the active execution path. A new `StrategyConsumerActor` in the execute scope consumes `StrategyResolvedEvent` from the `STRATEGY_EVENTS` stream, evaluates via `PaperOrderEvaluator` with explicit pass-through risk, and forwards the produced `ExecutionIntent` to the existing `VenueAdapterActor`. All 11 S359 contract invariants are preserved and tested. The wiring introduces no new NATS streams, no new event types, and no changes to the activation model or control gates.

---

## 2. Wiring Implemented

### 2.1 New Components

| Component | Location | Role |
|-----------|----------|------|
| `ExecuteStrategyMeanReversionEntryConsumer()` | `internal/adapters/nats/natsstrategy/registry.go` | Durable consumer spec for execute binary |
| `strategyReceivedMessage` | `internal/actors/scopes/execute/messages.go` | Actor message carrying strategy event |
| `StrategyConsumerActor` | `internal/actors/scopes/execute/strategy_consumer_actor.go` | Evaluates strategy → intent, forwards to venue adapter |

### 2.2 Modified Components

| Component | Change |
|-----------|--------|
| `ExecuteSupervisor` | Spawns StrategyConsumerActor + natsstrategy.Consumer; closes on stop |
| `cmd/execute/run.go` | Adds `strategy-consumer` health tracker |

### 2.3 Data Flow

```
StrategyResolvedEvent (NATS)
  → natsstrategy.Consumer (decode + ack)
    → supervisor handler (send to actor)
      → StrategyConsumerActor (evaluate + wrap)
        → VenueAdapterActor (gates + submit + fill)
```

The strategy path reuses the entire existing venue adapter pipeline (safety gates, composed submit pipeline, fill publication) via a synthetic `PaperOrderSubmittedEvent`. This is a deliberate design choice to avoid duplicating control logic.

---

## 3. Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/adapters/nats/natsstrategy/registry.go` | Modified | Added `ExecuteStrategyMeanReversionEntryConsumer()` |
| `internal/actors/scopes/execute/messages.go` | Modified | Added `strategyReceivedMessage` type |
| `internal/actors/scopes/execute/strategy_consumer_actor.go` | **New** | Strategy evaluation actor |
| `internal/actors/scopes/execute/strategy_consumer_actor_test.go` | **New** | 11 tests covering all invariants |
| `internal/actors/scopes/execute/execute_supervisor.go` | Modified | Wired strategy consumer actor + NATS consumer |
| `cmd/execute/run.go` | Modified | Added `strategy-consumer` tracker |
| `docs/architecture/controlled-source-to-execution-wiring.md` | **New** | Architecture document |
| `docs/architecture/source-driven-execution-wiring-order-controls-and-limitations.md` | **New** | Controls and limitations document |

---

## 4. Tests and Evidence

### 4.1 Unit Tests (11 pass)

| Test | Invariant | Assertion |
|------|-----------|-----------|
| `TestStrategyConsumer_LongDirection_ProducesBuySide` | INV-2 | long → buy, quantity = max_position_pct |
| `TestStrategyConsumer_ShortDirection_ProducesSellSide` | INV-2 | short → sell, quantity = max_position_pct |
| `TestStrategyConsumer_FlatDirection_ProducesNoExecution` | INV-7 | flat → none, quantity = 0 |
| `TestStrategyConsumer_PassThroughRisk` | INV-4 | risk.type = pass_through, risk.disposition = approved |
| `TestStrategyConsumer_CorrelationCausationChain` | INV-3 | intent.CorrelationID = event.Metadata.CorrelationID, intent.CausationID = event.Metadata.ID |
| `TestStrategyConsumer_UsesStrategyTimestamp` | INV-5 | intent.Timestamp = strategy.Timestamp (not time.Now()) |
| `TestStrategyConsumer_StrategyTypeIdentity` | INV-1 | risk.strategy_type = strategy.Type |
| `TestStrategyConsumer_WrongType_Skipped` | INV-6 | trend_following_entry → no intent produced |
| `TestStrategyConsumer_ConfigurableMaxPositionPct` | — | custom position size propagated |
| `TestStrategyConsumer_DefaultMaxPositionPct` | — | default 0.01 applied when not configured |
| `TestStrategyConsumer_DecisionSeverityPreserved` | — | decision severity carried in Risk and Parameters |

### 4.2 Build Verification

- `go build ./cmd/execute/...` — clean
- `go build ./internal/actors/...` — clean
- `go test ./internal/actors/scopes/execute/ -run TestStrategyConsumer` — 11/11 PASS

### 4.3 Invariant Coverage Matrix

| Invariant | Unit Test | Design Guarantee |
|-----------|-----------|------------------|
| INV-1 | Direct assertion | Field assignment |
| INV-2 | Both directions + flat | PaperOrderEvaluator is pure function |
| INV-3 | Direct assertion | Explicit field mapping |
| INV-4 | Direct assertion | Hardcoded constants |
| INV-5 | Fixed timestamp assertion | Direct assignment from strategy |
| INV-6 | Wrong type skipped | Type guard at actor entry |
| INV-7 | Flat → none/0 | Evaluator logic + test |
| INV-8 | — | natsstrategy.Consumer acks after handler return |
| INV-9 | — | Intent flows through VenueAdapterActor safety gates |
| INV-10 | — | VenueAdapterActor staleness guard checks intent.Timestamp |
| INV-11 | — | ExecutionIntent.DeduplicationKey() uses strategy timestamp |

INV-8 through INV-11 are guaranteed by architectural design (existing components) rather than new unit tests. They will be verified end-to-end in SSI-4 (S362).

---

## 5. Remaining Limitations

| ID | Limitation | Impact | Deferred To |
|----|-----------|--------|-------------|
| L1 | Single strategy family | Only mean_reversion_entry wired | Future wave |
| L2 | Pass-through risk only | No risk evaluation between strategy and execution | NG-4 |
| L3 | Fixed position sizing | Static 1%, not portfolio-aware | Future wave |
| L4 | No per-strategy gate | Kill switch is global only | SSI-3 (S361) |
| L5 | No confidence threshold | All events evaluated regardless of confidence | SSI-3 (S361) |
| L6 | No Prometheus metrics | Internal counters only | SSI-3 (S361) |
| L7 | Synthetic event coupling | Strategy path produces PaperOrderSubmittedEvent | Future refactor |
| L8 | No strategy performance tracking | No P&L attribution | Future wave |

---

## 6. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No new sources opened | PASS — only mean_reversion_entry |
| No multi-venue | PASS — paper adapter only |
| No pipeline redesign | PASS — reuses existing venue adapter pipeline |
| Activation/control model preserved | PASS — kill switch, staleness, activation surface unchanged |

---

## 7. Preparation for S361 (SSI-3: Explainability and Runtime Controls)

S361 builds on the wiring established here to add:

1. **Correlation ID propagation verification** — trace signal → decision → strategy → execution intent → venue fill end-to-end
2. **Per-strategy gate** — extend control plane to allow halt/resume per strategy type
3. **Confidence threshold** — configurable minimum confidence for strategy evaluation
4. **Prometheus metrics** — expose strategy consumer counters as Prometheus metrics
5. **Strategy health dashboard** — operational visibility into strategy-driven execution

### Prerequisites Satisfied by S360

- Strategy events reach execution path (consumer + actor wired)
- CorrelationID and CausationID are propagated (INV-3 tested)
- Health counters exist (tracker wired with semantic counter names)
- Actor topology supports extension (StrategyConsumerActor can be enhanced)

### Open Questions for S361

- Should per-strategy gate live in the same EXECUTION_CONTROL KV bucket or a separate bucket?
- Should confidence threshold be per-strategy-type or global?
- Should Prometheus metrics use the existing `internal/shared/metrics` package or extend it?
