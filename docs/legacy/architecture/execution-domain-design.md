# Execution — Domain Design

> Canonical design document for the `execution` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S69

---

## 1. Identity

Execution is the **7th domain layer** in Market Foundry's causal chain:

```
observation → evidence → signal → decision → strategy → risk → execution
```

WHERE `risk` evaluates whether a strategy intent is within acceptable bounds, `execution` transforms a risk-assessed intent into an actionable order intent — the first point in the chain that crosses the **action boundary**.

**One-sentence definition**: Execution receives a risk assessment (approved, modified, or rejected) and produces an `ExecutionIntent` that records the precise action the system would take, without actually performing it in the first slice.

**Examples**:

| Risk Input | Execution Output |
|-----------|-----------------|
| position_exposure: approved, direction=long, position_pct=0.018 | paper_order: side=buy, quantity=0.018, status=submitted |
| position_exposure: modified, direction=short, position_pct=0.020 (capped) | paper_order: side=sell, quantity=0.020, status=submitted |
| position_exposure: approved, direction=flat | paper_order: side=none, status=submitted (no-action recorded) |
| position_exposure: rejected | paper_order: side=none, status=submitted (rejection recorded) |

Every risk assessment produces a corresponding execution intent. The chain is never broken — even flat/rejected assessments produce a "no-action" intent for audit completeness.

---

## 2. What Execution IS

| Property | Description |
|----------|-------------|
| Input | Risk assessment (disposition, constraints, strategy provenance) via local actor message |
| Output | `ExecutionIntent` — a record of what the system would do |
| Nature | Translational — maps risk-gated strategy intent to actionable order parameters |
| Scope | Per-source, per-symbol, per-timeframe (same partitioning as all upstream domains) |
| Authority | Sole producer of `EXECUTION_EVENTS`; only domain that speaks in order-side/quantity terms |
| Activation | Opt-in via `pipeline.execution_families`; depends on risk families being enabled |
| Position | Last domain in the analytical/evaluative chain; first domain at the action boundary |
| Statefulness | Stateless in Phase 1 — each intent is independent; no memory of previous intents |
| Venue interaction | None in Phase 1 — paper execution only; intent is recorded but not forwarded |

---

## 3. What Execution IS NOT

| Anti-pattern | Why excluded |
|-------------|-------------|
| Order Management System (OMS) | Execution records intents, not orders. Order lifecycle (open, partial, filled, cancelled) is a future concern. |
| Position tracker | Execution does not track cumulative position. That is portfolio domain territory. |
| Venue router | Execution does not select venues or route orders. Venue selection is an adapter concern for future slices. |
| Fill reconciler | Execution does not match fills to orders. Fill reconciliation is a future adapter responsibility. |
| P&L calculator | Profit/loss tracking belongs to a future portfolio domain. |
| Market filter | Execution does not evaluate market conditions. That is signal/decision territory. |
| Risk evaluator | Execution consumes risk assessments; it does not produce them. |
| Strategy resolver | Execution does not decide direction. Strategy decides direction; risk gates it; execution records the intent. |
| Multi-strategy aggregator | Execution processes one risk assessment at a time. Aggregation across strategies is explicitly deferred. |
| Real-time order book consumer | Execution does not consume market microstructure data. |
| Margin manager | Margin and collateral management is future portfolio territory. |
| Circuit breaker (Phase 1) | Kill switch and circuit breaker are S76+ concerns; not part of domain logic. |

---

## 4. Domain Differentiation

### 4.1 Execution vs Strategy

| Dimension | Strategy | Execution |
|-----------|----------|-----------|
| Input | Decision outcome (triggered/not_triggered/insufficient) | Risk assessment (approved/modified/rejected) |
| Output | Strategy direction (long/short/flat) with confidence | Order intent (buy/sell/none) with quantity |
| Question | "Given this decision, what direction should we take?" | "Given this risk-gated intent, what order would we submit?" |
| Parameters | entry_price, target_price, stop_loss | quantity, price_type, side |
| Downstream | Risk | Store projection (and future venue adapters) |

### 4.2 Execution vs Risk

| Dimension | Risk | Execution |
|-----------|------|-----------|
| Input | Strategy intent (direction, confidence) | Risk assessment (disposition, constraints) |
| Output | Disposition (approved/modified/rejected) + constraints | Order intent (side, quantity, status) |
| Question | "Is this strategy intent within acceptable bounds?" | "What concrete action does this risk-approved intent produce?" |
| Parameters | max_position_pct, max_portfolio_exposure_pct | (none in Phase 1 — deterministic mapping) |
| Authority | Gates strategy intents | Records actionable intents |
| Side effects | None — pure evaluation | None in Phase 1; future: venue API calls |

### 4.3 Execution vs Portfolio (Future)

| Dimension | Execution | Portfolio (Future) |
|-----------|-----------|-------------------|
| Input | Single risk assessment | Execution intents + fill confirmations |
| Output | Single order intent | Cumulative position, P&L, exposure |
| Question | "What order would we submit now?" | "What is our aggregate state across all positions?" |
| Statefulness | Stateless (per-event) | Stateful (cumulative) |
| Time horizon | Instantaneous | Continuous |

### 4.4 Execution vs Venue Adapter (Future)

| Dimension | Execution Domain | Venue Adapter (Future) |
|-----------|-----------------|----------------------|
| Layer | Domain/Application | Adapter |
| Concern | What order to submit | How to submit it |
| Knowledge | Order parameters (side, quantity) | API credentials, rate limits, protocols |
| Failure modes | Logic errors (wrong mapping) | Network errors, API errors, rate limiting |
| Placement | derive binary | Separate binary or derive adapter |

---

## 5. Domain Boundary Invariants

| ID | Invariant |
|----|-----------|
| EBI-1 | Execution domain types MUST NOT import from risk, strategy, decision, signal, evidence, or observation domains. All upstream data arrives as primitive fields in local actor messages. |
| EBI-2 | Execution evaluators MUST be pure functions — no I/O, no actor references, no NATS operations. Testable with table-driven unit tests on synthetic data. |
| EBI-3 | Risk assessment data arrives via `riskAssessedMessage` (local actor message with primitive types per DBI-9), NOT via JetStream consumption. Execution actors MUST NOT subscribe to `RISK_EVENTS`. |
| EBI-4 | Only the `derive` binary publishes to `EXECUTION_EVENTS`. Single-writer invariant. |
| EBI-5 | Only the `store` binary materializes execution projections into KV buckets. Single-writer on each bucket. |
| EBI-6 | The `gateway` binary MUST NOT access execution KV buckets directly. All reads via NATS request/reply to `store`. |
| EBI-7 | Phase 1 execution MUST NOT interact with any external venue API. Paper execution only. |
| EBI-8 | `ExecutionIntent` MUST NOT embed `risk.RiskAssessment`. It uses `RiskInput` — a domain-owned struct with primitive copies of risk fields. |
| EBI-9 | Execution MUST NOT aggregate across multiple risk assessments, symbols, or timeframes. One risk assessment → one execution intent. |
| EBI-10 | Execution MUST NOT track cumulative position, P&L, or portfolio state. These are future domain concerns. |

---

## 6. Dependency on Risk

Execution depends on `risk` the same way strategy depends on `decision` and risk depends on `strategy`: through **local actor messages with primitive types**.

### Separation Mechanisms

1. **No import**: `internal/domain/execution/` never imports `internal/domain/risk/`.
2. **Domain-owned input type**: `RiskInput` is defined in the execution domain and copies only the fields execution needs.
3. **Local message**: `riskAssessedMessage` carries primitive data from `SourceScopeActor` to execution evaluators.
4. **No stream subscription**: Execution actors do NOT subscribe to `RISK_EVENTS`. They receive risk data from the derive-internal fan-out.
5. **Independent projection**: Execution projections are independent KV buckets; they do not join with risk projections.
6. **Independent query**: `GET /execution/:type/latest` does not correlate with `GET /risk/:type/latest`.

### Dependency Chain

```
observation → evidence (candle) → signal (rsi) → decision (rsi_oversold) → strategy (mean_reversion_entry) → risk (position_exposure) → execution (paper_order)
```

Execution is the terminal node. Nothing downstream consumes execution events — in Phase 1. Future portfolio or venue adapter layers would consume from `EXECUTION_EVENTS`.

---

## 7. Binary Placement

Execution evaluators live in the `derive` binary, inside `SourceScopeActor`, following the established fan-out pattern:

```
derive-supervisor
└── source-{source} (SourceScopeActor)
    ├── publisher (EvidencePublisherActor)
    ├── signal-publisher (SignalPublisherActor)
    ├── decision-publisher (DecisionPublisherActor)
    ├── strategy-publisher (StrategyPublisherActor)
    ├── risk-publisher (RiskPublisherActor)
    ├── execution-publisher (ExecutionPublisherActor)          ← NEW
    ├── sampler-SYMBOL-60s (CandleSamplerActor)
    ├── signal-rsi-SYMBOL-60s (RSISignalSamplerActor)
    ├── decision-rsi-oversold-SYMBOL-60s (RSIOversoldEvaluatorActor)
    ├── strategy-mean-reversion-entry-SYMBOL-60s (MeanReversionEntryResolverActor)
    ├── risk-position-exposure-SYMBOL-60s (PositionExposureEvaluatorActor)
    └── execution-paper-order-SYMBOL-60s (PaperOrderEvaluatorActor)  ← NEW
```

### Rationale

| Factor | Decision |
|--------|----------|
| Same fan-out pattern | Execution receives risk output via SourceScopeActor, identical to how risk receives strategy output |
| Pure evaluation | PaperOrderEvaluator is a pure function, same as PositionExposureEvaluator |
| No external state | Phase 1 paper execution needs no venue connection, database, or external state |
| Partition alignment | Per-source/symbol/timeframe partitioning matches all upstream domains |
| Single publisher | One ExecutionPublisherActor per source scope, same as all other publishers |

### Alternatives Considered

| Alternative | Rejection Rationale |
|-------------|---------------------|
| Separate `execute` binary | Unjustified complexity for Phase 1 paper execution. No external state or venue interaction. Consider for live execution only. |
| Store-side execution | Violates derive-owns-writes principle. Store materializes; it does not produce domain events. |
| Gateway-side execution | Violates gateway-is-stateless principle. Gateway translates HTTP; it does not process domain logic. |

---

## 8. Domain Model

### 8.1 Core Types

```go
package execution

type Side string

const (
    SideBuy  Side = "buy"
    SideSell Side = "sell"
    SideNone Side = "none"
)

type Status string

const (
    StatusSubmitted Status = "submitted"
)

type RiskInput struct {
    Type        string // risk assessment type (e.g., "position_exposure")
    Disposition string // risk disposition (approved/modified/rejected)
    Confidence  string // risk confidence (decimal string)
    Timeframe   int    // risk assessment timeframe in seconds
}

type ExecutionIntent struct {
    Type       string            // execution family (e.g., "paper_order")
    Source     string            // data source (e.g., "binancef")
    Symbol     string            // trading pair (e.g., "btcusdt")
    Timeframe  int               // period in seconds
    Side       Side              // buy, sell, or none
    Quantity   string            // position size as decimal string (from risk constraints)
    Status     Status            // current status (submitted for Phase 1)
    Risk       RiskInput         // risk assessment that produced this intent
    Parameters map[string]string // execution-specific parameters
    Metadata   map[string]string // extensible metadata
    Final      bool              // marks finalized intent
    Timestamp  time.Time         // intent creation time
}
```

### 8.2 Validation Rules

- `Type` MUST be non-empty and known (e.g., "paper_order")
- `Source` MUST be non-empty
- `Symbol` MUST be non-empty
- `Timeframe` MUST be > 0
- `Side` MUST be one of: "buy", "sell", "none"
- `Quantity` MUST be non-empty (may be "0" for no-action intents)
- `Status` MUST be one of: "submitted" (Phase 1 only)
- `Risk.Type` MUST be non-empty
- `Risk.Disposition` MUST be non-empty
- `Timestamp` MUST be non-zero

### 8.3 Key Design Choices

| Field | Choice | Rationale |
|-------|--------|-----------|
| `Side` (not `Direction`) | Orders use buy/sell terminology, not long/short. Strategy speaks in direction; execution speaks in order side. |
| `Quantity` (string) | Decimal precision preserved as string, consistent with all other domains. |
| `Status` (single value) | Phase 1 has only "submitted." Future statuses (filled, cancelled, expired) are explicitly deferred. |
| `RiskInput` (not `RiskAssessment`) | Domain-owned copy of risk fields. No cross-domain import. Follows StrategyInput pattern in risk domain. |
| `Side=none` for flat/rejected | Every risk assessment produces an intent. No-action intents have side=none and quantity="0". Chain is never broken. |
| No `Price` field | Paper orders have no price semantics. Price fields are deferred to venue adapter slices. |
| No `OrderID` | Execution intents are not orders. They have no lifecycle. Partition/dedup keys provide identity. |

### 8.4 Partition and Deduplication Keys

```go
func (e ExecutionIntent) PartitionKey() string {
    return fmt.Sprintf("%s.%s.%d", e.Source, e.Symbol, e.Timeframe)
}

func (e ExecutionIntent) DeduplicationKey() string {
    return fmt.Sprintf("exec:%s:%s:%s:%d:%d", e.Type, e.Source, e.Symbol, e.Timeframe, e.Timestamp.UnixMilli())
}
```

- `PartitionKey` isolates KV entries per source/symbol/timeframe (consistent with all domains)
- `DeduplicationKey` prevents duplicate JetStream publishes (consistent with all domains)

---

## 9. Event Contract

### 9.1 Stream

| Property | Value |
|----------|-------|
| Stream name | `EXECUTION_EVENTS` |
| Subjects | `execution.events.>` |
| Storage | FileStorage |
| Retention | 72 hours (LimitsPolicy) |
| Max bytes | 2 GB |
| Dedup window | 2 minutes |
| Discard policy | DiscardOld |

### 9.2 Subject Pattern

```
execution.events.{type}.submitted.{source}.{symbol}.{timeframe}
```

Example: `execution.events.paper_order.submitted.binancef.btcusdt.60`

### 9.3 Event Type

| Event | Emitter | Trigger |
|-------|---------|---------|
| `ExecutionIntentSubmittedEvent` | PaperOrderEvaluatorActor (via ExecutionPublisherActor) | Every risk assessment received |

```go
type ExecutionIntentSubmittedEvent struct {
    Metadata        events.Metadata
    ExecutionIntent ExecutionIntent
}

func (e ExecutionIntentSubmittedEvent) EventName() string {
    return "execution_intent_submitted"
}
```

### 9.4 Envelope

| Plane | Envelope Type |
|-------|--------------|
| Events | `execution.events.v1.paper_order_submitted` |
| Query request | `execution.query.v1.paper_order_latest_request` |
| Query reply | `execution.query.v1.paper_order_latest_reply` |

---

## 10. Activation Model

### Layer 1: Family Activation (Structural)

```jsonc
// derive.jsonc / store.jsonc
{
  "pipeline": {
    "execution_families": ["paper_order"]  // opt-in; empty = disabled
  }
}
```

| Property | Value |
|----------|-------|
| Key | `pipeline.execution_families` |
| Semantics | List of enabled execution families; empty = none enabled |
| Change | Requires binary restart |
| Validation | `executionDependsOnRisk["paper_order"] = ["position_exposure"]` |
| Default | `[]` (disabled) |

### Layer 2: Binding Activation (Runtime)

| Trigger | Effect |
|---------|--------|
| `IngestionRuntimeChanged` (new binding) | SourceScopeActor spawns execution evaluator actors for new symbol/timeframe combinations |
| `IngestionRuntimeChanged` (removed binding) | Execution evaluator actors stop for removed combinations |

### Dependency Validation

```
executionDependsOnRisk["paper_order"] = ["position_exposure"]
```

At binary startup, `ValidatePipeline()` rejects configurations where `paper_order` is enabled without `position_exposure`.

### No Auto-Activation Principle

Execution is NEVER activated by default. It requires explicit opt-in in both derive and store configurations. This is consistent with all post-evidence domains (signal, decision, strategy, risk).

---

## 11. Publication, Projection, Query

### Who Publishes

`ExecutionPublisherActor` in `derive` binary. One publisher per source scope. Receives `publishExecutionMessage` from evaluator actors. Publishes `ExecutionIntentSubmittedEvent` to `EXECUTION_EVENTS` stream.

### Who Projects

`ExecutionProjectionActor` in `store` binary. Single-writer to `EXECUTION_PAPER_ORDER_LATEST` KV bucket. Applies three-gate pattern (final → validate → monotonicity).

### Who Serves Query

`QueryResponderActor` in `store` binary. Responds to `execution.query.paper_order.latest` NATS subject. Reads from KV bucket. Returns `ExecutionLatestReply`.

### Pipeline Flow

```
derive                          store                         gateway
──────                          ─────                         ───────
RiskEvaluatorActor              ExecutionConsumerActor         HTTP handler
  │ riskAssessedMessage           │ consume from                │ GET /execution/:type/latest
  ▼                               │ EXECUTION_EVENTS            │
SourceScopeActor                  ▼                             ▼
  │ routeRiskToExecution        ExecutionProjectionActor       ExecutionGateway
  ▼                               │ three-gate                  │ NATS request
PaperOrderEvaluatorActor          │ materialize                 │ to store
  │ evaluate (pure)               ▼                             ▼
  │ publishExecutionMessage     EXECUTION_PAPER_ORDER_LATEST  QueryResponderActor
  ▼                             (KV bucket)                    │ read KV
ExecutionPublisherActor                                        │ return reply
  │ publish to EXECUTION_EVENTS
```

### KV Buckets

| Bucket | Key Format | Purpose |
|--------|-----------|---------|
| `EXECUTION_PAPER_ORDER_LATEST` | `{source}.{symbol}.{timeframe}` | Latest execution intent per partition |

---

## 12. Latest vs History

| Aspect | Phase 1 (S75) | Deferred |
|--------|---------------|----------|
| Latest projection | YES — single KV entry per partition | — |
| History projection | NO | Future: execution history bucket for audit trail |
| Lifecycle tracking | NO — status is always "submitted" | Future: submitted → filled → cancelled state machine |
| Fill recording | NO | Future: fill events from venue adapter |
| Audit persistence | Trace metadata in logs + JetStream | Future: persisted in projection or audit bucket (S72 design decision) |

**Rationale for latest-only**: The first execution slice proves the domain model, mesh flow, and projection pattern. History and lifecycle are second-slice concerns that require the trace metadata persistence decision (S72) to be resolved first.

---

## 13. Projection Gates

| Gate | Purpose |
|------|---------|
| **Final gate** | Skip execution intents where `Final=false`. Only finalized intents are materialized. |
| **Validation gate** | Reject intents that fail `Validate()`. Invalid intents are counted but not stored. |
| **Monotonicity guard** | Skip intents where `Timestamp` is older than the existing entry. Prevents stale overwrites. |

Stats counters (7, consistent with all projection actors):
- `received`, `materialized`, `skippedStale`, `skippedDedup`, `skippedNonFinal`, `rejected`, `errors`

Stats invariant: `received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors`

---

## 14. Query Surface

### HTTP Endpoint

```
GET /execution/{type}/latest?source={source}&symbol={symbol}&timeframe={timeframe}
```

Phase 1 type: `paper_order`

### NATS Subjects

| Operation | NATS Subject |
|-----------|-------------|
| Latest query | `execution.query.paper_order.latest` |
| History query | `execution.query.paper_order.history` (deferred) |

### Gateway Rules

**MUST:**
- Register routes conditionally (only if ExecutionGateway is available)
- Parse and validate query parameters (source, symbol, timeframe)
- Forward to use case layer
- Return well-formed JSON response
- Propagate X-Correlation-ID header

**MUST NOT:**
- Access execution KV buckets directly
- Cache execution intents
- Transform or enrich execution data
- Validate execution domain rules
- Interpret execution intent semantics (side, quantity)
- Join execution data with risk/strategy data
- Produce execution events
- Make venue API calls
- Track position or P&L state
- Apply business logic to execution intents
- Filter intents by side or status

---

## 15. Risk Consumption Pattern

Execution receives risk assessment data via the same local-message fan-out pattern used by all upstream domains.

```
PositionExposureEvaluatorActor
  │ produces riskAssessedMessage (primitive types: symbol, type, disposition, confidence, constraints, timeframe, timestamp, correlationID, causationID)
  ▼
SourceScopeActor.routeRiskToExecution()
  │ fans out to all execution evaluators for this symbol
  ▼
PaperOrderEvaluatorActor
  │ receives riskAssessedMessage
  │ calls PaperOrderEvaluator.Evaluate() (pure function)
  │ produces ExecutionIntentSubmittedEvent
  ▼
ExecutionPublisherActor
```

This is **NOT a domain import**. The `riskAssessedMessage` carries:
- `Symbol` (string)
- `RiskType` (string)
- `RiskDisposition` (string)
- `RiskConfidence` (string)
- `MaxPositionPct` (string)
- `MaxExposurePct` (string)
- `StrategyDirection` (string)
- `StrategyConfidence` (string)
- `Timeframe` (int)
- `Timestamp` (time.Time)
- `CorrelationID` (string)
- `CausationID` (string)

All primitive types. No risk domain imports.

---

## 16. Operational Invariants

| ID | Invariant |
|----|-----------|
| OI-1 | One `ExecutionPublisherActor` per source scope. No shared publishers across scopes. |
| OI-2 | One `PaperOrderEvaluatorActor` per symbol/timeframe combination. Partition-aligned isolation. |
| OI-3 | Execution evaluator failures do NOT affect risk, strategy, or upstream actors. Failure is scoped. |
| OI-4 | Execution projection stats invariant MUST hold at shutdown. Received == sum of all outcomes. |
| OI-5 | Execution health tracker records every publish and materialization event. |
| OI-6 | ExecutionPublisherActor logs errors with full trace context (correlation_id, causation_id). |
| OI-7 | Execution projection logs materialization with side, quantity, status, trace context. |
| OI-8 | Every risk assessment produces exactly one execution intent. The chain is never silently dropped. |
| OI-9 | Paper execution never calls external APIs, opens network connections, or writes to non-NATS storage. |

---

## 17. What Is Explicitly NOT (Yet)

| Future Domain/Capability | Relationship to Execution | When |
|--------------------------|--------------------------|------|
| Venue Adapter | Consumes execution intents and submits real orders | After paper execution is proven |
| Fill Processing | Records fills from venue, updates execution state | After venue adapter |
| Portfolio | Aggregates execution + fills into position and P&L state | After execution lifecycle is complete |
| Multi-Risk Aggregation | Combines multiple risk assessments before execution | After additional risk families exist |
| Kill Switch | Halts execution processing without restart | S76 (design only) |
| Circuit Breaker | Auto-halts execution on anomaly detection | After venue adapter |
| Execution History | Historical record of all execution intents | After S72 trace persistence decision |

---

## 18. Deferred Work

### Deferred Topics

| Topic | Rationale | Target |
|-------|-----------|--------|
| Venue adapter integration | Phase 1 is paper-only; venue introduces external failure modes | S77+ |
| Execution lifecycle state machine | Phase 1 has single status ("submitted"); lifecycle requires fills | S77+ |
| Fill events and reconciliation | No venue means no fills | S77+ |
| Execution history projection | Requires S72 trace metadata persistence decision | S77+ |
| Multi-strategy aggregation | Single risk assessment → single intent; aggregation is portfolio territory | S77+ |
| Price fields (limit, stop) | Paper orders have no price semantics | S77+ |
| Rate limiting | No venue means no rate limit concern | S76+ |
| Kill switch | Design in S76; implementation after first slice is proven | S76 |
| Position tracking | Portfolio domain, not execution | Future |
| Portfolio aggregation | Future domain beyond execution | Future |

### S70 Scope (Governance)

Per S68 readiness review, the following must be resolved before or alongside execution implementation:
- Verify risk drift rules pass via raccoon-cli
- Add execution drift rules (ED-1..ED-5)
- Add execution guardrails preventing premature implementation details
- Update actor-ownership.md with execution actors
- Update stream-family-catalog.md with execution families

### S75 Scope (Implementation)

Implementation of the first execution slice (paper_order family):
- Domain model: `ExecutionIntent`, `RiskInput`, `Side`, `Status`
- Application: `PaperOrderEvaluator` (pure function)
- Actor: `PaperOrderEvaluatorActor`, `ExecutionPublisherActor`
- Adapter: `ExecutionPublisher`, `ExecutionConsumer`, `ExecutionKVStore`, `ExecutionRegistry`, `ExecutionGateway`
- Store: `ExecutionConsumerActor`, `ExecutionProjectionActor`
- Gateway: HTTP handler, route, use case
- Config: `execution_families` in PipelineConfig, dependency validation
- Tests: Domain, application, projection, HTTP, registry, KV store

---

## 19. References

- [risk-domain-design.md](risk-domain-design.md) — Immediate upstream domain design
- [strategy-domain-design.md](strategy-domain-design.md) — Pattern reference for domain design documents
- [execution-readiness-review.md](execution-readiness-review.md) — S68 readiness assessment
- [execution-entry-prerequisites.md](execution-entry-prerequisites.md) — Mandatory conditions before code
- [execution-risks-and-blockers.md](execution-risks-and-blockers.md) — Risk catalog
- [execution-stream-families.md](execution-stream-families.md) — Stream family catalog for execution
- [execution-activation-and-ownership.md](execution-activation-and-ownership.md) — Activation model and actor ownership
- [execution-query-surface-guidelines.md](execution-query-surface-guidelines.md) — Query surface specification
- [market-foundry-evolution-playbook.md](market-foundry-evolution-playbook.md) — Evolution rulebook
- [system-principles.md](system-principles.md) — Foundational principles
- [end-to-end-traceability.md](end-to-end-traceability.md) — Traceability invariants
- [causal-chain-guidelines.md](causal-chain-guidelines.md) — Guidelines for new domain layers
