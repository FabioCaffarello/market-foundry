# Derive Producer: Boundaries, Invariants, Lifecycle, and Limits

> **Stage**: S365 — Producer Spec and Derive Ownership (DI-1)
> **Wave**: Derive Integration Wave (S364–S369)
> **Companion to**: [StrategyResolvedEvent Producer Spec and Derive Ownership Model](strategy-resolved-event-producer-spec-and-derive-ownership-model.md)
> **Status**: Binding
> **Date**: 2026-03-22

---

## 1. Purpose

This document defines the boundaries between derive, strategy, execution, and
store in the context of `StrategyResolvedEvent` production. It specifies the
event lifecycle from creation to consumption, the invariants that must hold
at every boundary crossing, and the explicit limits of the current model.

---

## 2. Boundary Model

### 2.1 Four-Domain Boundary Map

```
┌──────────────────────────────────────────────────────────────────────┐
│                         DERIVE BINARY                                │
│                                                                      │
│  ┌─────────────────┐    ┌──────────────────────┐    ┌─────────────┐ │
│  │ Decision Domain  │───▶│  Strategy Domain      │───▶│ Publisher   │ │
│  │ (evaluator)      │    │  (resolver)           │    │ (NATS)     │ │
│  │                  │    │                        │    │            │ │
│  │ Produces:        │    │ Produces:              │    │ Publishes: │ │
│  │ decisionEval-    │    │ Strategy struct        │    │ Strategy-  │ │
│  │ uatedMessage     │    │ StrategyResolvedEvent  │    │ Resolved-  │ │
│  │ (primitive data) │    │ (domain type)          │    │ Event      │ │
│  └─────────────────┘    └──────────────────────┘    └──────┬──────┘ │
│                                                             │        │
└─────────────────────────────────────────────────────────────┼────────┘
                                                              │
                                    NATS JetStream            │
                                    STRATEGY_EVENTS           │
                                                              │
                    ┌─────────────────────────────────────────┼────────┐
                    │                                         │        │
     ┌──────────────▼──────────┐        ┌────────────────────▼──────┐ │
     │     STORE BINARY        │        │     EXECUTE BINARY        │ │
     │                         │        │                           │ │
     │  StrategyConsumerActor  │        │  StrategyConsumerActor    │ │
     │  StrategyProjection-    │        │  PaperOrderEvaluator      │ │
     │    Actor                │        │  VenueAdapterActor        │ │
     │  KV Bucket write        │        │  ExecutionIntent          │ │
     │                         │        │                           │ │
     │  Durable: store-        │        │  Durable: execute-        │ │
     │    strategy-mean-       │        │    strategy-mean-         │ │
     │    reversion-entry      │        │    reversion-entry        │ │
     └─────────────────────────┘        └───────────────────────────┘ │
                                                                      │
                    ┌─────────────────────────────────────────────────┘
                    │
     ┌──────────────▼──────────┐
     │     GATEWAY BINARY      │
     │                         │
     │  HTTP → NATS translator │
     │  No domain logic        │
     │  No KV access           │
     └─────────────────────────┘
```

### 2.2 Boundary Rules

| Boundary | Transport | Rule |
|---|---|---|
| Decision → Strategy (within derive) | Actor message (`decisionEvaluatedMessage`) | Primitive data only (DBI-9). No `decision.Decision` struct crosses. |
| Strategy resolver → Publisher (within derive) | Actor message (`publishStrategyMessage`) | Domain event (`StrategyResolvedEvent`) crosses — same binary, same scope. |
| Derive → NATS | JetStream publish | Serialized JSON. Subject format enforced by registry. Dedup key set as MsgID. |
| NATS → Store | JetStream consume (durable) | Store deserializes, validates, projects. Independent consumer from execute. |
| NATS → Execute | JetStream consume (durable) | Execute deserializes, validates, evaluates. Independent consumer from store. |
| Store → Gateway | NATS request/reply | Gateway is a stateless HTTP↔NATS translator. No KV access. |

### 2.3 What Each Domain Must NOT Do

| Domain | Forbidden Actions |
|---|---|
| **Derive (strategy producer)** | Must NOT determine execution side or quantity. Must NOT write to KV. Must NOT import execution domain types. Must NOT bypass validation before publish. |
| **Strategy domain** | Must NOT import decision, signal, evidence, or observation domains (SBI-1). Must NOT feed back into decision (SBI-7). Must NOT consume signals directly (SBI-10). |
| **Execute (strategy consumer)** | Must NOT re-resolve strategies. Must NOT alter strategy type or confidence. Must NOT write to STRATEGY_EVENTS. |
| **Store (strategy projector)** | Must NOT produce strategy events. Must NOT alter strategy fields during projection. Must NOT serve queries without monotonicity gate. |
| **Gateway** | Must NOT access KV directly. Must NOT cache, transform, or interpret strategy metadata. Must NOT cross-query strategy with other domains in a single request. |

---

## 3. Event Lifecycle

### 3.1 Lifecycle Phases

```
Phase 1: RESOLUTION
  │  MeanReversionEntryResolver.Resolve()
  │  Input: decision primitives (type, outcome, confidence, severity, rationale, timeframe, timestamp)
  │  Output: Strategy struct (or false if no resolution)
  │  Properties: pure function, deterministic, no I/O
  │
Phase 2: VALIDATION
  │  Strategy.Validate()
  │  Gates: type not empty, source not empty, symbol not empty, timeframe > 0,
  │         direction ∈ {long, short, flat}, confidence valid, timestamp not zero,
  │         at least one DecisionInput
  │  On failure: error logged, no event produced
  │
Phase 3: EVENT CONSTRUCTION
  │  Metadata = events.NewMetadata()
  │    .WithCorrelationID(msg.CorrelationID)
  │    .WithCausationID(msg.CausationID)
  │  Event = StrategyResolvedEvent{Metadata, Strategy}
  │  Properties: Metadata.ID is fresh UUID, Metadata.OccurredAt is time.Now().UTC()
  │
Phase 4: INTRA-BINARY DISPATCH
  │  Actor message: publishStrategyMessage{Event}
  │  Delivered to StrategyPublisherActor via actor PID send
  │  Also: strategyResolvedMessage sent to ScopePID for risk fan-out
  │
Phase 5: PUBLICATION
  │  natsstrategy.Publisher.PublishStrategy()
  │  Subject: strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}
  │  MsgID: strat:mean_reversion_entry:{source}:{symbol}:{timeframe}:{timestamp_unix}
  │  JetStream synchronous publish with 5s timeout
  │  On failure: health tracker error + error log (no retry at publisher level)
  │
Phase 6: STREAM PERSISTENCE
  │  STRATEGY_EVENTS JetStream stream
  │  Retention: 72 hours, file-backed
  │  Max bytes: 256 MB
  │  Dedup window: JetStream default (2 minutes)
  │  Multiple independent consumers subscribe
  │
Phase 7a: STORE CONSUMPTION
  │  StoreMeanReversionEntryStrategyConsumer (durable)
  │  Deserialize → Validate → Final gate → Monotonicity guard → KV write
  │  KV bucket: STRATEGY_MEAN_REVERSION_ENTRY_LATEST
  │  Key: {source}.{symbol}.{timeframe}
  │
Phase 7b: EXECUTE CONSUMPTION (parallel with 7a)
  │  ExecuteStrategyMeanReversionEntryConsumer (durable)
  │  Deserialize → Validate → Type check → PaperOrderEvaluator → ExecutionIntent
  │  Direction→Side mapping → Execution publication
  │
Phase 8: QUERY
    Gateway HTTP → NATS request → Store QueryResponderActor → KV read → reply
```

### 3.2 Lifecycle Invariants

| ID | Invariant | Phase(s) |
|---|---|---|
| **LI-1** | Strategy is validated before any publication | 2 |
| **LI-2** | Metadata.ID is unique per event (UUID) | 3 |
| **LI-3** | CorrelationID is never generated by strategy — always propagated | 3 |
| **LI-4** | CausationID points to the upstream decision event | 3 |
| **LI-5** | Publication is synchronous — publisher waits for JetStream ack | 5 |
| **LI-6** | Dedup key is deterministic from strategy content | 5 |
| **LI-7** | Store and execute consume independently (different durable names) | 7a, 7b |
| **LI-8** | Store monotonicity guard rejects timestamp regression | 7a |
| **LI-9** | Event is immutable after Phase 3 — no field mutation in phases 4–8 | 3–8 |

---

## 4. Producer-Side Invariants

### 4.1 Structural Invariants

| ID | Invariant | Enforcement |
|---|---|---|
| **PI-1** | `Strategy.Type` is always `"mean_reversion_entry"` for this resolver | Hardcoded in `MeanReversionEntryResolver.Resolve()` return |
| **PI-2** | `Strategy.Direction` is one of `{long, short, flat}` | Enforced by `Resolve()` logic + `Validate()` |
| **PI-3** | `Strategy.Confidence` is a valid decimal string in [0.0, 1.0] | Set by `ScaleConfidence()` or hardcoded `"0.0000"` |
| **PI-4** | `Strategy.Decisions` has exactly one entry | Set by `Resolve()` — single decision input for mean reversion |
| **PI-5** | `Strategy.Final` is always `true` | Hardcoded in `Resolve()` return |
| **PI-6** | `Strategy.Timestamp` is the decision timestamp, not `time.Now()` | Passed through from `decisionEvaluatedMessage.Timestamp` |

### 4.2 Behavioral Invariants

| ID | Invariant | Enforcement |
|---|---|---|
| **BI-1** | Resolution is deterministic: same input → same output | `Resolve()` is a pure function with no external state |
| **BI-2** | Validation failure never produces an event | `onDecisionEvaluated()` returns early on validation failure |
| **BI-3** | Unknown decision outcome never produces an event | `Resolve()` returns `false` for unrecognized outcomes |
| **BI-4** | Severity scaling is bounded | Scaling factors are in static maps; worst case = 0.80× (never negative, never > 1.0×) |
| **BI-5** | Flat direction always has zero confidence | `"0.0000"` for not_triggered and insufficient |
| **BI-6** | Event metadata is constructed once and never mutated | `events.NewMetadata()` call in `onDecisionEvaluated()` |

### 4.3 Transport Invariants

| ID | Invariant | Enforcement |
|---|---|---|
| **TI-1** | Subject matches registry-defined pattern | `Publisher.specForType()` resolves from `Registry` |
| **TI-2** | Dedup key matches `Strategy.DeduplicationKey()` | Called in `PublishStrategy()` → `jetstream.WithMsgID()` |
| **TI-3** | Envelope type matches registry-defined type | Passed through `natskit.EncodeEvent()` with `spec.Type` |
| **TI-4** | CorrelationID and CausationID are passed to `EncodeEvent()` | Explicit parameters in `PublishStrategy()` |
| **TI-5** | Stream is created/updated before first publish | `Publisher.Start()` calls `js.CreateOrUpdateStream()` |

---

## 5. Cross-Domain Responsibility Matrix

| Concern | Derive (Producer) | Execute (Consumer) | Store (Projector) | Gateway (Query) |
|---|---|---|---|---|
| Strategy resolution | **OWNS** | — | — | — |
| Direction determination | **OWNS** | — | — | — |
| Confidence scoring | **OWNS** | Forwards | Forwards | Forwards |
| Severity interpretation | **OWNS** (scaling) | Forwards | Forwards | Forwards |
| NATS publication | **OWNS** | — | — | — |
| Side determination | — | **OWNS** (via evaluator) | — | — |
| Quantity sizing | — | **OWNS** (via config) | — | — |
| Risk pass-through | — | **OWNS** (explicit marker) | — | — |
| Kill switch | — | **OWNS** (gate) | — | — |
| Staleness guard | — | **OWNS** (guard) | — | — |
| KV materialization | — | — | **OWNS** | — |
| Monotonicity guard | — | — | **OWNS** | — |
| HTTP translation | — | — | — | **OWNS** |
| Dedup key (STRATEGY_EVENTS) | **OWNS** | — | — | — |
| Dedup key (EXECUTION_EVENTS) | — | **OWNS** | — | — |
| Correlation chain start | Propagates | Propagates | Propagates | — |

---

## 6. Activation and Lifecycle

### 6.1 Producer Activation

| Step | Trigger | Effect |
|---|---|---|
| 1 | `pipeline.strategy_families` includes `"mean_reversion_entry"` in derive config | Family structurally enabled |
| 2 | `BindingWatcherActor` detects active binding (source+symbol+timeframe) | `SourceScopeActor` spawned |
| 3 | `SourceScopeActor` starts | Spawns `MeanReversionEntryResolverActor` + `StrategyPublisherActor` as children |
| 4 | `DecisionEvaluatorActor` produces decision | `decisionEvaluatedMessage` sent to resolver |
| 5 | Resolver resolves + validates | `publishStrategyMessage` sent to publisher |
| 6 | Publisher publishes to NATS | Event available on STRATEGY_EVENTS |

### 6.2 Producer Deactivation

| Step | Trigger | Effect |
|---|---|---|
| 1 | Binding removed from configctl | `BindingWatcherActor` signals deactivation |
| 2 | `SourceScopeActor` stops | All child actors (resolver, publisher) stopped |
| 3 | Publisher closes NATS connection | No further events for this partition |

### 6.3 Dependency Chain

```
pipeline.families         → evidence samplers
pipeline.signal_families  → signal samplers      ← depends on evidence
pipeline.decision_families → decision evaluators  ← depends on signals
pipeline.strategy_families → strategy resolvers   ← depends on decisions
```

**Operator responsibility**: each layer is independently configurable. Activating
`mean_reversion_entry` without `rsi_oversold` in `decision_families` means the
resolver never receives input and remains idle. No implicit activation chains.

---

## 7. Explicit Limits

### 7.1 What This Model Guarantees

| Guarantee | Scope |
|---|---|
| Single canonical producer per family | Only `MeanReversionEntryResolverActor` produces `mean_reversion_entry` events |
| Deterministic resolution | Same decision input → same strategy output (pure function) |
| Correlation chain preserved | CorrelationID propagates unaltered from signal through strategy |
| Causation chain established | Each hop sets CausationID to upstream event ID |
| Dedup prevents duplicates | JetStream MsgID-based dedup in STRATEGY_EVENTS |
| Validation before publication | Invalid strategies are never published |
| Independent consumers | Store and execute have separate durable consumers, independent offsets |
| Event immutability | Published event is never mutated after construction |

### 7.2 What This Model Does NOT Guarantee

| Limitation | Rationale | Resolution Path |
|---|---|---|
| **No at-most-once delivery** | JetStream provides at-least-once; consumer idempotency required | Consumers use dedup keys + monotonicity guards |
| **No cross-partition ordering** | Events for different source/symbol/timeframe may arrive out of order | Each partition is independent; ordering is per-partition |
| **No backpressure from execution** | Execution failures do not halt strategy production | By design — strategy is purely analytical |
| **No retry at publisher level** | Publish failure logs error but does not retry | JetStream ack ensures persistence; next resolution will produce a new event |
| **No multi-decision aggregation** | Mean reversion uses exactly one decision | Multi-decision strategies need aggregation logic (deferred) |
| **No confidence threshold gate** | All confidence levels produce events (including 0.0000 for flat) | SSI-3 deferred: configurable minimum confidence |
| **No per-strategy-type gate** | Only global kill switch exists | SSI-3 deferred: strategy-type-specific gates |
| **No event expiration** | Events persist for 72h in stream, no expiration event produced | `strategy_expired` event name reserved but not implemented |
| **No short direction for RSI** | Current resolver only maps triggered→long, not overbought→short | RSI oversold decision triggers long; overbought is a separate decision family |
| **No position-size awareness** | Strategy has no concept of position or portfolio | Position sizing is execution/risk concern |
| **No rate limiting** | Every decision produces a strategy (no debounce/cooldown) | Deferred to future hardening |

### 7.3 Known Simplifications

| Simplification | Justification | Impact |
|---|---|---|
| Single DecisionInput per strategy | Mean reversion uses one decision (RSI oversold) | Multi-decision strategies need new resolver pattern |
| Severity scaling is static map | No dynamic adjustment of scaling factors | Sufficient for first proof; optimization is NG-12 |
| `Metadata.OccurredAt` vs `Strategy.Timestamp` | OccurredAt is event creation time; Timestamp is decision-sourced | No conflict — each serves different purpose |
| Publisher creates stream on Start() | Stream creation is idempotent (CreateOrUpdate) | Safe for multi-scope startup |

---

## 8. Preparation for S366

S366 (DI-2: Canonical Derive Producer Wiring) should:

1. **Add unit tests for the resolver** — one test per invariant (PI-1 through PI-6, BI-1 through BI-6), verifying that:
   - `triggered` outcome → `Direction=long`, confidence scaled by severity
   - `not_triggered` outcome → `Direction=flat`, confidence `"0.0000"`
   - `insufficient` outcome → `Direction=flat`, confidence `"0.0000"`, metadata has `reason`
   - Unknown outcome → no strategy produced
   - All fields present in `Decisions[0]`
   - `Final=true` always
   - `Type="mean_reversion_entry"` always
   - `Timestamp` matches input timestamp

2. **Add unit tests for the publisher** — verifying:
   - Subject format: `strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}`
   - Dedup key format: `strat:mean_reversion_entry:{source}:{symbol}:{timeframe}:{ts_unix}`
   - Envelope type: `strategy.events.v1.mean_reversion_entry_resolved`
   - CorrelationID and CausationID passed through to `EncodeEvent()`

3. **Add unit tests for the resolver actor** — verifying:
   - `Metadata.CorrelationID` propagated from `decisionEvaluatedMessage`
   - `Metadata.CausationID` propagated from `decisionEvaluatedMessage`
   - Validation failure prevents event dispatch
   - `publishStrategyMessage` sent to publisher PID
   - `strategyResolvedMessage` sent to scope PID for risk fan-out

4. **Fix any contract violations found** — current audit shows zero blocking mismatches, but unit tests may reveal edge cases.

---

## 9. Non-Goals for This Document

| Non-Goal | Reason |
|---|---|
| Consumer-side spec | Already proven in S358–S363 |
| Store materialization verification | DI-3 (S367) |
| Gateway query verification | DI-3 (S367) |
| End-to-end proof | DI-4 (S368) |
| Multiple strategy families | NG-1 |
| Risk domain integration | NG-14 |
| Derive runtime redesign | NG-8 |
| Docker Compose orchestration | NG-9 |

---

## References

- [StrategyResolvedEvent Producer Spec and Derive Ownership Model](strategy-resolved-event-producer-spec-and-derive-ownership-model.md)
- [Source Selection and Canonical Integration Contract (S359)](source-selection-and-canonical-integration-contract.md)
- [Source-to-Execution Contract: Boundaries, Invariants, and Limits (S359)](source-to-execution-contract-boundaries-invariants-and-limits.md)
- [Strategy Domain Design (S53)](strategy-domain-design.md)
- [Strategy Activation and Ownership (S53)](strategy-activation-and-ownership.md)
- [Derive Pipeline Pattern](derive-pipeline-pattern.md)
- [Derive Integration Wave Charter (S364)](derive-integration-wave-charter-and-scope-freeze.md)
