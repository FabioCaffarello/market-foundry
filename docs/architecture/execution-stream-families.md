# Execution — Stream Families

> Stream family catalog for the `execution` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S69

---

## 1. What Is an Execution Family

An execution family is a named group of evaluators that share the same evaluation boundary: same event stream, same subject encoding, same producer binary (`derive`), same projection pattern (`store`), and same query surface (`gateway`). Execution families follow the same structural rules as risk families, strategy families, decision families, signal families, and evidence families.

Each execution family:
- Consumes risk assessment data via local actor message (not JetStream)
- Produces `ExecutionIntentSubmittedEvent` to a shared `EXECUTION_EVENTS` stream
- Is materialized into a family-specific KV bucket by `store`
- Is queryable via a family-specific NATS request/reply subject and HTTP endpoint

---

## 2. Family Catalog

### EF-01: Paper Order — Phase 1 (S75)

| Property | Value |
|----------|-------|
| Canonical name | `paper_order` |
| Bounded context | Simulated order intent derived from risk-approved strategy |
| Classification | Translational — maps risk disposition to order intent |
| Input risk families | `position_exposure` |
| Evaluation logic | Deterministic mapping: disposition + direction → side + quantity |
| Output | `ExecutionIntent` with side (buy/sell/none) and quantity |
| Status values | `submitted` (Phase 1 only) |
| Statefulness | Stateless — each intent is independent |
| Venue interaction | None — paper execution records intent but does not forward it |

**Evaluator contract:**

```
Evaluate(riskType, riskDisposition, riskConfidence, maxPositionPct, maxExposurePct,
         strategyDirection, strategyConfidence, timeframe, timestamp)
    → (ExecutionIntent, bool)
```

**Evaluation rules:**
- Disposition `approved` + direction `long` → side=buy, quantity=maxPositionPct
- Disposition `approved` + direction `short` → side=sell, quantity=maxPositionPct
- Disposition `modified` + direction `long` → side=buy, quantity=maxPositionPct (already capped by risk)
- Disposition `modified` + direction `short` → side=sell, quantity=maxPositionPct (already capped by risk)
- Disposition `approved`/`modified` + direction `flat` → side=none, quantity="0"
- Disposition `rejected` → side=none, quantity="0"
- Invalid disposition or direction → return false (evaluation failed)

**Parameters output:**

```json
{
  "risk_type": "position_exposure",
  "risk_disposition": "approved",
  "strategy_direction": "long",
  "strategy_confidence": "0.8500"
}
```

**Rationale**: Paper order is the minimum viable execution family. It proves the domain model, mesh flow, projection pattern, and activation model without any external dependency. The evaluation logic is deterministic — it maps risk output to order parameters without introducing new decision logic. This is intentional: execution is a translator, not a decision-maker.

### EF-02: Venue Market Order — Deferred (S77+)

| Property | Value |
|----------|-------|
| Canonical name | `venue_market_order` |
| Bounded context | Real market order submitted to a venue adapter |
| Classification | Operational — triggers real-world side effects |
| Input risk families | `position_exposure` (minimum) |
| Evaluation logic | Same mapping as paper_order + venue-specific parameters |
| Status values | `submitted`, `sent`, `filled`, `rejected`, `cancelled`, `expired` |
| Statefulness | Stateful — tracks order lifecycle |
| Venue interaction | YES — requires venue adapter binary/component |

**Deferred because:**
1. Requires venue adapter architecture (not yet designed)
2. Requires execution lifecycle state machine (not yet designed)
3. Requires kill switch mechanism (S76)
4. Requires circuit breaker (not yet designed)
5. Paper execution must be proven first

### EF-03: Venue Limit Order — Deferred (S78+)

| Property | Value |
|----------|-------|
| Canonical name | `venue_limit_order` |
| Bounded context | Limit order with price constraint submitted to venue |
| Classification | Operational |
| Additional fields | `price_limit` (string, decimal) |
| Deferred because | Requires EF-02 infrastructure + price field in domain model |

---

## 3. Stream Definition

```
Stream:      EXECUTION_EVENTS
Subjects:    execution.events.>
Storage:     FileStorage
Retention:   LimitsPolicy (72 hours)
MaxBytes:    2 GB
Dedup:       2 minute window (MsgID-based)
Discard:     DiscardOld
Replicas:    1
```

### Subject Pattern

```
execution.events.{type}.submitted.{source}.{symbol}.{timeframe}
```

Examples:
- `execution.events.paper_order.submitted.binancef.btcusdt.60`
- `execution.events.paper_order.submitted.binancef.ethusdt.300`

Future (deferred):
- `execution.events.venue_market_order.submitted.binancef.btcusdt.60`

---

## 4. KV Bucket Naming

| Family | Latest Bucket | History Bucket |
|--------|--------------|----------------|
| paper_order | `EXECUTION_PAPER_ORDER_LATEST` | Deferred |
| venue_market_order | `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Deferred |
| venue_limit_order | `EXECUTION_VENUE_LIMIT_ORDER_LATEST` | Deferred |

**Key format**: `{source}.{symbol}.{timeframe}`

Example: `binancef.btcusdt.60`

**Bucket properties** (Phase 1):
- Storage: FileStorage
- Max size: 64 MB
- No TTL (latest-only, overwritten by new entries)

---

## 5. Envelope Types

| Plane | Envelope Type |
|-------|--------------|
| Events | `execution.events.v1.paper_order_submitted` |
| Query request (latest) | `execution.query.v1.paper_order_latest_request` |
| Query reply (latest) | `execution.query.v1.paper_order_latest_reply` |

---

## 6. Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|-------------|--------|---------------|---------|
| `store-execution-paper-order` | store | `execution.events.paper_order.submitted.>` | Materialize paper order intents into KV |

**Consumer properties:**
- AckPolicy: Explicit
- AckWait: 30 seconds
- MaxDeliver: 5 attempts
- DeliverPolicy: DeliverAll

Future consumers:
- `store-execution-venue-market-order` (deferred)
- `venue-adapter-execution-paper-order` (deferred — future venue adapter binary)

---

## 7. Family Invariants

| ID | Invariant |
|----|-----------|
| EFI-1 | All execution families share a single `EXECUTION_EVENTS` stream. One stream, multiple subject filters. |
| EFI-2 | Each execution family has its own durable consumer in `store`. No consumer sharing across families. |
| EFI-3 | Each execution family has its own KV bucket. No bucket sharing across families. |
| EFI-4 | `ExecutionPublisherActor` is shared across families within a source scope. One publisher per source, routing by family spec. |
| EFI-5 | Execution families are independent. Enabling `paper_order` does not require or affect `venue_market_order`. |
| EFI-6 | Adding a new execution family does NOT require modifying `SourceScopeActor` spawning logic. It follows the FamilyProcessor registration pattern. |
| EFI-7 | Each family's evaluator is a pure function. No I/O, no actors, no NATS. |
| EFI-8 | Envelope types are versioned per family (v1). Version changes require new envelope types, not modifications to existing ones. |
| EFI-9 | Dedup keys include family type to prevent cross-family collision. |

---

## 8. Family Growth Pattern

To add a new execution family:

1. **Design**: Add family entry to this catalog with all properties defined.
2. **Domain**: Add evaluator contract, validation rules, and any new enum values to `internal/domain/execution/`.
3. **Application**: Create pure evaluator in `internal/application/execution/`.
4. **Adapter**: Add EventSpec and ControlSpec to execution registry. Create KV store with bucket constant.
5. **Config**: Add family name to `knownExecutionFamilies`. Add dependency to `executionDependsOnRisk`. Add `IsExecutionFamilyEnabled()` check.
6. **Derive**: Register evaluator actor in SourceScopeActor. Publisher is shared — add spec routing only.
7. **Store**: Add ProjectionPipeline entry with consumer, projection actor, and KV bucket.
8. **Gateway**: Add query spec to registry. Add HTTP handler and route (if new query pattern).
9. **Governance**: Add drift rules for new family. Update actor-ownership.md and stream-family-catalog.md.
10. **Test**: Domain validation, evaluator logic, projection gates, HTTP handler, registry contracts, KV store guards.

---

## 9. Dependency DAG Extension

```
knownExecutionFamilies = { "paper_order" }

executionDependsOnRisk = {
    "paper_order": ["position_exposure"]
}
```

Full chain:
```
candle ← rsi ← rsi_oversold ← mean_reversion_entry ← position_exposure ← paper_order
(evidence) (signal) (decision)    (strategy)             (risk)           (execution)
```

Validated at startup by `ValidatePipeline()`. Enabling `paper_order` without `position_exposure` is a hard validation error.

---

## 10. Data Flow Diagram

```
derive                                    NATS                           store                    gateway
──────                                    ────                           ─────                    ───────

RiskEvaluatorActor                                                       ExecutionConsumerActor
  │ riskAssessedMessage                                                    │ consume from
  ▼                                                                        │ EXECUTION_EVENTS
SourceScopeActor                                                           ▼
  │ routeRiskToExecution                                                 ExecutionProjectionActor
  ▼                                                                        │ final gate
PaperOrderEvaluatorActor                                                   │ validate gate
  │ PaperOrderEvaluator.Evaluate()                                         │ monotonicity guard
  │ (pure function)                                                        ▼
  │ publishExecutionMessage          ──►  EXECUTION_EVENTS  ──►         EXECUTION_PAPER_ORDER_LATEST
  ▼                                       (JetStream)                   (KV bucket)
ExecutionPublisherActor                                                    │
  │ publish to stream                                                    QueryResponderActor
                                                                           │ NATS request/reply    ◄── ExecutionGateway
                                                                           │                           │
                                                                           ▼                           ▼
                                                                         Reply                     HTTP handler
                                                                                                   GET /execution/:type/latest
```

---

## 11. References

- [execution-domain-design.md](execution-domain-design.md) — Domain model and boundary invariants
- [execution-activation-and-ownership.md](execution-activation-and-ownership.md) — Activation model and actor ownership
- [risk-stream-families.md](risk-stream-families.md) — Upstream stream family reference
- [stream-families.md](stream-families.md) — Stream taxonomy conventions
- [stream-family-catalog.md](stream-family-catalog.md) — Global stream family catalog
