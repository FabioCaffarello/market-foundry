# Source-to-Execution Contract: Boundaries, Invariants, and Limits

> **Stage**: S359 — Source Selection and Canonical Contract
> **Wave**: Strategy/Signal Integration (S358–S363)
> **Companion to**: [Source Selection and Canonical Integration Contract](source-selection-and-canonical-integration-contract.md)
> **Status**: Binding
> **Date**: 2026-03-22

---

## 1. Purpose

This document defines the invariants, boundary rules, and explicit limits that govern the canonical integration contract between the selected strategy source (`mean_reversion_entry`) and the execution path. Any implementation (SSI-2 through SSI-4) must satisfy these constraints.

---

## 2. Domain Boundary Rules

### 2.1 Signal → Decision Boundary

| Rule | Specification |
|------|---------------|
| Signal domain does not know about decisions | Signal types never reference decision types |
| Decision owns `SignalInput` | Decision-local struct; no import from signal domain |
| Signal value is opaque to decision | Decision evaluators interpret signal value per their own logic |
| One signal type per decision evaluator | Each evaluator consumes exactly one signal family |
| Boundary transport | Local actor fan-out within derive + NATS `SIGNAL_EVENTS` stream for materialization |

### 2.2 Decision → Strategy Boundary

| Rule | Specification |
|------|---------------|
| Decision domain does not know about strategies | Decision types never reference strategy types |
| Strategy owns `DecisionInput` | Strategy-local struct; no import from decision domain |
| Strategy may consume multiple decisions | But for the selected pair, exactly one decision feeds mean_reversion_entry |
| Severity carries forward | `DecisionInput.Severity` preserves the originating decision's semantic depth |
| Boundary transport | Local actor fan-out within derive + NATS `DECISION_EVENTS` stream for materialization |

### 2.3 Strategy → Execution Boundary (The Contract Boundary)

| Rule | Specification |
|------|---------------|
| Strategy domain does not know about execution | Strategy types never reference execution types |
| Execution does not import strategy types | `StrategyConsumerActor` reads strategy fields as primitives |
| Transport is NATS only | No local fan-out — execute scope subscribes to `STRATEGY_EVENTS` stream |
| StrategyConsumerActor owns the transformation | Maps strategy fields + defaults → `PaperOrderEvaluator` parameters |
| Risk is bypassed with explicit marker | `riskType = "pass_through"` — not omitted, not faked, explicitly marked |
| Strategy provenance preserved | All strategy fields propagated into `ExecutionIntent.Parameters` |

### 2.4 Execution → Venue Boundary (Existing, Unchanged)

| Rule | Specification |
|------|---------------|
| VenueAdapterActor owns submission | Existing pipeline: kill switch → staleness → venue adapter |
| Paper adapter is the only target | NG-7: no mainnet execution |
| Fill events published to separate stream | `EXECUTION_FILL_EVENTS` — distinct from `EXECUTION_EVENTS` |
| Correlation preserved through fill | `VenueOrderFilledEvent.CorrelationID` matches intent's `CorrelationID` |

---

## 3. Invariants

### 3.1 Contract Invariants (Must Hold at All Times)

**INV-1: Type identity preservation.**
`ExecutionIntent.Risk.StrategyType` must always equal the `strategy.Type` from the source event. The consumer must not alter, normalize, or default this value.

**INV-2: Direction-to-side determinism.**
Given the same strategy `Direction` and risk `Disposition`, the `PaperOrderEvaluator` must always produce the same `Side`. This is a pure function with no external state.

**INV-3: Causation chain integrity.**
`ExecutionIntent.CausationID` must equal the `Metadata.ID` of the consumed `StrategyResolvedEvent`. `ExecutionIntent.CorrelationID` must equal the `Metadata.CorrelationID` of the consumed event.

**INV-4: Pass-through risk is explicit.**
When risk evaluation is bypassed, `riskType` must be `"pass_through"` — never empty, never a real risk type. Any code inspecting risk provenance can trivially distinguish bypassed from evaluated.

**INV-5: Timestamp monotonicity.**
The `StrategyConsumerActor` must use the strategy event's timestamp as the `ExecutionIntent.Timestamp`. The consumer must not substitute `time.Now()` or any derived timestamp. Downstream KV materialization relies on timestamp monotonicity guards.

**INV-6: Single strategy family per consumer.**
The `StrategyConsumerActor` subscribes to exactly one strategy type's subject (`strategy.events.mean_reversion_entry.resolved.>`). It must not consume events from other strategy families.

**INV-7: Flat direction produces no execution.**
A strategy with `Direction = "flat"` must produce an `ExecutionIntent` with `Side = "none"` and `Quantity = "0"`. The intent is still published (for observability) but produces no venue submission.

**INV-8: Event acknowledgment after publication.**
The NATS message must be acknowledged only after the `ExecutionIntent` has been successfully published to `EXECUTION_EVENTS`. If publication fails, the message must be NAK'd for redelivery.

### 3.2 Operational Invariants

**INV-9: Kill switch precedence.**
The existing `VenueAdapterActor` kill switch applies to strategy-sourced intents identically to derive-sourced intents. No bypass path exists.

**INV-10: Staleness guard applies.**
The existing staleness guard in `VenueAdapterActor` applies to strategy-sourced intents. If `intent.Timestamp` is older than `StalenessMaxAge`, the intent is dropped.

**INV-11: Deduplication key uniqueness.**
`ExecutionIntent.DeduplicationKey()` produces `exec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}`. Since the consumer uses the strategy's timestamp (INV-5), and each strategy event has a unique timestamp per partition, deduplication keys are unique.

---

## 4. Boundary Separation: Signal vs Strategy vs Control vs Execution

### 4.1 Responsibility Matrix

| Concern | Signal Layer | Decision Layer | Strategy Layer | Control Layer | Execution Layer |
|---------|-------------|----------------|----------------|---------------|-----------------|
| Evidence interpretation | **Owns** | — | — | — | — |
| Condition evaluation | — | **Owns** | — | — | — |
| Directional intent | — | — | **Owns** | — | — |
| Confidence scoring | Contributes | Contributes | **Owns** (final) | — | Forwards |
| Severity assessment | — | **Owns** | Forwards | — | Forwards |
| Risk evaluation | — | — | — | — | Bypassed (pass-through) |
| Side determination | — | — | — | — | **Owns** (via evaluator) |
| Quantity sizing | — | — | — | — | **Owns** (via config) |
| Halt/resume | — | — | — | **Owns** (gate) | Respects |
| Staleness rejection | — | — | — | — | **Owns** (guard) |
| Venue submission | — | — | — | — | **Owns** (adapter) |

### 4.2 What Each Layer Must NOT Do

| Layer | Forbidden Actions |
|-------|-------------------|
| Signal | Must not evaluate conditions, determine direction, or reference downstream types |
| Decision | Must not resolve strategy, determine side, or reference strategy/execution types |
| Strategy | Must not determine execution side, quantity, or reference execution types |
| Control | Must not influence direction, confidence, or side — only halt/resume |
| Execution | Must not reinterpret signal values, re-evaluate decisions, or resolve strategies |

---

## 5. Configuration Contract

The `StrategyConsumerActor` requires these configuration values:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_position_pct` | Decimal string | `"0.01"` | Position size cap when risk is bypassed (1%) |
| `staleness_max_age` | Duration | `120s` | Maximum age of strategy event before rejection (existing VenueAdapterActor config) |

These are the only configurable parameters. The consumer must not accept configuration for risk type, risk disposition, or direction mapping — these are invariants, not configuration.

---

## 6. Error Handling Contract

### 6.1 Deserialization Failure

If the NATS message cannot be deserialized into a `StrategyResolvedEvent`, the message is terminated (acknowledged without processing). The error is logged with the raw message subject and sequence number.

### 6.2 Validation Failure

If `strategy.Validate()` returns a problem, the message is terminated. The error is logged with the strategy type and partition key.

### 6.3 Wrong Strategy Type

If `strategy.Type != "mean_reversion_entry"`, the message is terminated. This is a defensive guard — the NATS subject filter should prevent this, but the consumer validates anyway.

### 6.4 Evaluator Failure

If `PaperOrderEvaluator.Evaluate()` returns `false` (no intent produced), the message is acknowledged. This is not an error — it means the strategy direction/disposition combination produced no actionable intent (e.g., flat direction).

### 6.5 Publication Failure

If publishing the `ExecutionIntent` to NATS fails, the message is NAK'd for redelivery (up to `MaxDeliver = 5`). After max deliveries, the message is dead-lettered by JetStream.

---

## 7. Observability Contract

### 7.1 Required Log Fields

Every strategy event processed must log:

| Field | Value |
|-------|-------|
| `strategy_type` | `mean_reversion_entry` |
| `source` | Strategy source (e.g., `binancef`) |
| `symbol` | Strategy symbol (e.g., `btcusdt`) |
| `timeframe` | Strategy timeframe (e.g., `60`) |
| `direction` | Strategy direction (`long`, `short`, `flat`) |
| `confidence` | Strategy confidence |
| `outcome` | `"intent_produced"`, `"no_action"`, `"validation_failed"`, `"deserialization_failed"` |
| `side` | Resulting execution side (if intent produced) |
| `correlation_id` | From event metadata |

### 7.2 Health Tracker Integration

The `StrategyConsumerActor` must integrate with the existing health tracker pattern:
- Increment received counter on message arrival
- Increment processed counter on successful intent production
- Increment error counter on failures
- No new counter families — use existing `health.Tracker` contract

### 7.3 Prometheus Metrics (Deferred to SSI-3)

Strategy-specific Prometheus metrics (accepted/rejected/below-threshold counters) are explicitly deferred to SSI-3. The consumer implementation in SSI-2 uses health tracker only.

---

## 8. Testing Contract

### 8.1 Unit Test Requirements (SSI-2)

| Test | Assertion |
|------|-----------|
| Long direction produces buy side | `strategy.Direction = "long"` → `intent.Side = "buy"` |
| Short direction produces sell side | `strategy.Direction = "short"` → `intent.Side = "sell"` |
| Flat direction produces none side | `strategy.Direction = "flat"` → `intent.Side = "none"`, `intent.Quantity = "0"` |
| Strategy provenance preserved | `intent.Parameters["strategy_type"] = "mean_reversion_entry"` |
| Decision severity forwarded | `intent.Risk.DecisionSeverity = strategy.Decisions[0].Severity` |
| Risk type is pass-through | `intent.Risk.Type = "pass_through"` |
| Correlation ID propagated | `intent.CorrelationID = event.Metadata.CorrelationID` |
| Causation ID set | `intent.CausationID = event.Metadata.ID` |
| Timestamp preserved | `intent.Timestamp = strategy.Timestamp` |
| Validation failure handled | Invalid strategy → message terminated, error logged |

### 8.2 Integration Test Requirements (SSI-4)

| Test | Assertion |
|------|-----------|
| End-to-end: signal → execution | RSI signal → decision → strategy → consumer → intent → paper fill |
| Kill switch blocks execution | Gate halted → intent reaches VenueAdapterActor but is not submitted |
| Staleness rejects old intents | Strategy timestamp > 120s ago → intent dropped |
| Provenance chain intact | CorrelationID matches across all domain events in chain |

---

## 9. Migration Path

### 9.1 When Risk Integration Is Added (Future Wave)

The `StrategyConsumerActor` will be extended to:
1. Query `RISK_EVENTS` or `RISK_LATEST` KV for the current risk assessment
2. Replace pass-through defaults with real risk values
3. `riskType` changes from `"pass_through"` to the actual risk family type

The `PaperOrderEvaluator` interface does not change. The `ExecutionIntent` structure does not change. Only the input values to `Evaluate()` change.

### 9.2 When Additional Strategy Families Are Added (Future Wave)

Each new strategy family gets its own `StrategyConsumerActor` instance with its own NATS consumer spec. The pattern established by this contract is replicated per-family. No shared consumer, no multi-type routing.

### 9.3 When Per-Strategy Gates Are Added (SSI-3)

The `StrategyConsumerActor` will check a strategy-type-specific KV gate before calling `PaperOrderEvaluator`. This is an additive change — the contract fields do not change.

---

## 10. Explicit Limits

### 10.1 What This Contract Guarantees

- One signal family (RSI) feeds one strategy family (mean_reversion_entry)
- Strategy events are consumed by exactly one actor in execute scope
- ExecutionIntents carry full strategy provenance
- Causation and correlation chains are preserved
- Risk is explicitly bypassed with auditable markers
- Kill switch and staleness guards remain active
- Paper adapter is the only execution target

### 10.2 What This Contract Does NOT Guarantee

- **No real risk evaluation** — all intents are auto-approved
- **No position management** — quantity is a fixed config value, not portfolio-aware
- **No multi-timeframe correlation** — single timeframe per partition
- **No strategy performance tracking** — no P&L, no win rate, no drawdown
- **No backpressure from execution to strategy** — execution failures do not halt strategy production
- **No guaranteed delivery semantics beyond JetStream** — at-least-once delivery, consumer idempotency is the actor's responsibility
- **No ordering guarantee across partitions** — events for different source/symbol/timeframe combinations may arrive out of order relative to each other

### 10.3 Known Simplifications

| Simplification | Justification | Resolution Path |
|----------------|---------------|-----------------|
| Risk pass-through | NG-4: Risk domain changes excluded | Future wave adds risk query before evaluation |
| Fixed position size | No risk constraints available | Risk integration provides dynamic sizing |
| Single decision severity | `Decisions[0].Severity` used even if multiple decisions exist | Mean reversion entry uses exactly one decision; multi-decision strategies need aggregation logic |
| No confidence threshold | All confidence levels produce intents | SSI-3 adds configurable minimum confidence |
| No per-strategy gate | Only global kill switch exists | SSI-3 adds strategy-type-specific gates |

---

## 11. Compliance with Wave Constraints

| Constraint | Status |
|------------|--------|
| FC-1: No new blocks after S358 | Compliant — this is SSI-1 deliverable |
| FC-2: Block scope frozen | Compliant — contract matches SSI-1 specification |
| FC-4: Only selected signal+strategy pair | Compliant — RSI + mean_reversion_entry only |
| FC-5: Paper adapter only | Compliant — venue submission uses paper adapter |
| FC-7: Domain types may extend but not restructure | Compliant — no domain type changes |
| NG-1: No multiple signal families | Compliant — RSI only |
| NG-2: No multiple strategy families | Compliant — mean_reversion_entry only |
| NG-4: No risk domain changes | Compliant — pass-through defaults |
| NG-11: No new domain types | Compliant — uses existing types only |

---

## 12. Preparation for S360 (SSI-2 Wiring)

The following items are ready for implementation in the next stage:

1. **Consumer spec defined** — `execute-strategy-mean-reversion-entry` on `strategy.events.mean_reversion_entry.resolved.>`
2. **Field mapping defined** — every parameter to `PaperOrderEvaluator.Evaluate()` has a documented source
3. **Error handling defined** — every failure mode has a specified behavior
4. **Test cases defined** — unit and integration test assertions are enumerated
5. **Invariants defined** — implementation can be validated against 11 invariants
6. **Configuration defined** — only `max_position_pct` is configurable; everything else is invariant or derived

The implementation scope for S360 is:
- Create `StrategyConsumerActor` in `internal/actors/scopes/execute/`
- Wire into execute binary topology (`cmd/execute/run.go`)
- Add consumer spec to `natsexecution` registry or create new `natsstrategy` consumer reference
- Unit tests per Section 8.1
- Validate against invariants per Section 3
