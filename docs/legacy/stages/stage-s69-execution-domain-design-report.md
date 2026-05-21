# Stage S69 — Execution Domain Design Report

## Executive Summary

S69 produced the formal domain design for `execution` — Market Foundry's 7th domain layer and the first to cross the action boundary. The design defines `execution` as a narrow, stateless translator that converts risk-approved strategy intents into recorded order intents (paper-only in Phase 1), without touching real venue APIs, tracking positions, or managing order lifecycle.

The first execution family is `paper_order`: a deterministic mapping from risk disposition + strategy direction to order side + quantity. No new decision logic. No venue interaction. No state. The domain model, stream families, activation model, projection/query pipeline, and boundary invariants follow the established canonical patterns proven across 6 prior domains.

---

## Structural Objective

Design the `execution` domain with explicit boundaries, contracts, stream families, activation model, and projection/query implications — preparing a first-slice implementation (S75) without improvisation.

---

## Pre-Condition Verification

S68 (Execution Readiness Review) concluded:

> **CONDITIONALLY READY — with elevated prerequisites.**
> "This review assesses readiness to **design** the execution domain. It does not assess readiness to **implement** it."

The 4 hard blockers (HB-1 through HB-4) prevent **code**, not **design documentation**. This stage produces design only. Pre-condition satisfied.

---

## Key Design Decisions

### D-1: Execution is a Translator, Not a Decision-Maker

Execution does not introduce new evaluation logic. It maps:
- Risk disposition (approved/modified/rejected) → action/no-action
- Strategy direction (long/short/flat) → order side (buy/sell/none)
- Risk constraints (max_position_pct) → order quantity

The evaluation is deterministic. Execution answers "what concrete action does this risk-approved intent produce?" — not "should we act?" (that is risk's job) or "in which direction?" (that is strategy's job).

### D-2: Paper-Only First Slice

The first execution family is `paper_order`. It records the intent but does not forward it to any venue. Rationale:
1. Proves domain model, mesh flow, projection pattern, and activation model without external dependencies
2. Avoids venue adapter failure modes that would obscure domain logic validation
3. Follows the MarketMonkey principle: prove the mesh first, then add operational adapters
4. Venue adapter architecture requires separate design (S77+)

### D-3: Every Risk Assessment Produces an Execution Intent

The causal chain is never broken. Flat directions and rejected dispositions produce "no-action" intents (side=none, quantity="0"). This ensures:
- Audit trail is complete (every risk assessment has a corresponding execution record)
- Chain integrity is verifiable (no silent drops)
- Projection stats account for all events (received == sum of outcomes)

### D-4: Side, Not Direction

Execution uses `buy`/`sell`/`none` (order-side terminology), not `long`/`short`/`flat` (strategy-direction terminology). This is a deliberate semantic boundary:
- Strategy speaks in market positions (long/short/flat)
- Execution speaks in order actions (buy/sell/none)
- The mapping is explicit: long→buy, short→sell, flat→none

### D-5: No OrderID, No Lifecycle

ExecutionIntent has no OrderID because it is not an order. It has no lifecycle state machine because Phase 1 has only "submitted" status. Future families (venue_market_order) will introduce lifecycle tracking. This prevents premature abstraction in the domain model.

### D-6: Binary Placement in Derive

Execution evaluators live in the `derive` binary within `SourceScopeActor`, following the established fan-out pattern. Rationale:
- Pure function evaluation (no external state needed)
- Same partitioning (per-source/symbol/timeframe)
- Same activation model (family + binding)
- No justification for a separate binary in Phase 1

### D-7: RiskInput, Not RiskAssessment

ExecutionIntent contains `RiskInput` — a domain-owned struct with primitive copies of risk fields. No import from `internal/domain/risk/`. This follows the StrategyInput pattern in the risk domain, maintaining domain module isolation (Principle 2).

---

## Domain Model Summary

```
ExecutionIntent
├── Type       string            // "paper_order"
├── Source     string            // "binancef"
├── Symbol     string            // "btcusdt"
├── Timeframe  int               // 60
├── Side       Side              // buy, sell, none
├── Quantity   string            // "0.0180" (decimal string)
├── Status     Status            // submitted (Phase 1 only)
├── Risk       RiskInput         // domain-owned risk reference
├── Parameters map[string]string // evaluation context
├── Metadata   map[string]string // extensible
├── Final      bool              // finalization flag
└── Timestamp  time.Time         // intent creation time

RiskInput
├── Type        string  // "position_exposure"
├── Disposition string  // "approved", "modified", "rejected"
├── Confidence  string  // decimal string
└── Timeframe   int     // seconds

Side: buy | sell | none
Status: submitted
```

---

## Stream Topology

| Resource | Type | Owner | Consumers |
|----------|------|-------|-----------|
| `EXECUTION_EVENTS` | JetStream stream | derive (ExecutionPublisherActor) | store (ExecutionConsumerActor) |
| `EXECUTION_PAPER_ORDER_LATEST` | KV bucket | store (ExecutionProjectionActor) | store (QueryResponderActor) |
| `execution.query.paper_order.latest` | NATS req/reply | store (QueryResponderActor) | gateway (ExecutionGateway) |
| `GET /execution/:type/latest` | HTTP endpoint | gateway (HTTP handler) | External consumers |

Subject pattern: `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`

---

## Boundary Invariants (10)

| ID | Invariant |
|----|-----------|
| EBI-1 | No cross-domain imports (risk, strategy, decision, signal, evidence, observation) |
| EBI-2 | Execution evaluators are pure functions (no I/O, no actors, no NATS) |
| EBI-3 | Risk data arrives via local actor messages with primitive types, not JetStream |
| EBI-4 | Only derive publishes to EXECUTION_EVENTS (single-writer) |
| EBI-5 | Only store materializes execution projections (single-writer per bucket) |
| EBI-6 | Gateway never accesses execution KV directly (NATS request/reply only) |
| EBI-7 | Phase 1: no external venue API interaction |
| EBI-8 | RiskInput is domain-owned, not an import of risk.RiskAssessment |
| EBI-9 | No aggregation across risk assessments, symbols, or timeframes |
| EBI-10 | No cumulative position, P&L, or portfolio state |

---

## Intentional Limits

### What This Design Explicitly Does NOT Cover

| Excluded Concern | Why | When |
|-----------------|-----|------|
| Venue adapter architecture | External failure modes; separate design needed | S77+ |
| Order lifecycle state machine | Requires venue interaction for fill/cancel states | S77+ |
| Fill processing and reconciliation | No venue means no fills | S77+ |
| Execution history projection | Requires S72 trace metadata persistence decision | S77+ |
| Multi-strategy aggregation | One risk → one intent; aggregation is portfolio domain | Future |
| Price fields (limit, stop) | Paper orders have no price semantics | S77+ |
| Rate limiting | No venue means no rate limit concern | S76+ |
| Kill switch | S76 design concern, not domain design | S76 |
| Position tracking | Portfolio domain responsibility | Future |
| Circuit breaker | Requires venue adapter to protect | Future |

### What This Design Explicitly Defers

| Item | Rationale |
|------|-----------|
| `venue_market_order` family (EF-02) | Requires venue adapter, lifecycle, kill switch |
| `venue_limit_order` family (EF-03) | Requires EF-02 plus price fields |
| Execution history bucket | Requires S72 design decision |
| Execution-specific smoke test | Implementation concern (S75) |
| raccoon-cli execution drift rules | Governance concern (S70+) |

---

## Preparation for Next Stages

### S70: Risk Governance Drift Verification

Before execution implementation, verify:
- Risk drift rules exist and pass in raccoon-cli
- Add execution drift rules (ED-1..ED-5)
- Update actor-ownership.md with execution actors
- Update stream-family-catalog.md with EXECUTION_EVENTS

### S71-S72: Traceability Hardening

From S68 prerequisites:
- S71: Automated traceability verification test (integration test with NATS)
- S72: Trace metadata persistence design (KV, audit bucket, or stream replay)

### S75: Implementation (First Slice)

Implementation scope, fully determined by this design:

| Component | Deliverable |
|-----------|-------------|
| Domain | `internal/domain/execution/` — ExecutionIntent, RiskInput, Side, Status, events, validation, tests |
| Application | `internal/application/execution/` — PaperOrderEvaluator (pure function), tests |
| Client | `internal/application/executionclient/` — contracts, use case, tests |
| Adapter | `internal/adapters/nats/` — ExecutionPublisher, ExecutionConsumer, ExecutionKVStore, ExecutionRegistry, ExecutionGateway |
| Derive actors | `internal/actors/scopes/derive/` — PaperOrderEvaluatorActor, ExecutionPublisherActor, messages |
| Store actors | `internal/actors/scopes/store/` — ExecutionConsumerActor, ExecutionProjectionActor |
| HTTP | `internal/interfaces/http/` — handlers, routes |
| Config | `internal/shared/settings/` — knownExecutionFamilies, executionDependsOnRisk, PipelineConfig extension |
| Binaries | `cmd/derive/`, `cmd/store/`, `cmd/gateway/` — execution wiring |

---

## Deliverables Produced

| Document | Location | Purpose |
|----------|----------|---------|
| Execution Domain Design | `docs/architecture/execution-domain-design.md` | Domain model, boundaries, invariants |
| Execution Stream Families | `docs/architecture/execution-stream-families.md` | Stream catalog, families, growth pattern |
| Execution Activation and Ownership | `docs/architecture/execution-activation-and-ownership.md` | Activation model, actor trees, data flow |
| Execution Query Surface Guidelines | `docs/architecture/execution-query-surface-guidelines.md` | Query chain, endpoints, envelope types |
| Stage Report (this document) | `docs/stages/stage-s69-execution-domain-design-report.md` | Summary and decisions |

---

## Files Changed

### Documentation (new)
- `docs/architecture/execution-domain-design.md` — 19 sections, ~500 lines
- `docs/architecture/execution-stream-families.md` — 11 sections, ~250 lines
- `docs/architecture/execution-activation-and-ownership.md` — 13 sections, ~300 lines
- `docs/architecture/execution-query-surface-guidelines.md` — 12 sections, ~250 lines

### No Code Changes

S69 is a design stage. No code was modified. No tests were added. No governance rules were changed. The stage produced documentation only, consistent with the playbook principle: "contracts defined → activation verified → pattern proven → implementation."

---

## Stage Closure Checklist

- [x] Single structural capability declared (execution domain design)
- [x] No code changes (design stage)
- [x] No governance debt introduced
- [x] Boundaries with risk, store, gateway, and future adapters are explicit
- [x] Stream families and ownership are clear
- [x] Activation model follows established patterns
- [x] Domain model is narrow and conservative
- [x] Deferred work is documented with rationale
- [x] No execution code implemented
- [x] No portfolio domain opened
- [x] No excessive contracts for non-immediate futures
- [x] Execution has clear owner in each binary
- [x] First slice scope is deterministic from design
- [x] Pre-condition (S68 readiness for design) verified
