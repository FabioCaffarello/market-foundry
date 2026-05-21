# Stage S62 — Risk Domain Design Report

> Date: 2026-03-18
> Status: **COMPLETE**
> Type: Design (no implementation)

---

## 1. Executive Summary

Stage S62 produced the canonical design for the `risk` domain in Market Foundry — the sixth layer in the pipeline after strategy. The design defines risk as a **domain-specific evaluation layer** that gates strategy intents through position-sizing rules and exposure limits, producing approved/modified/rejected dispositions.

The design preserves strict separation between strategy (directional intent), risk (gated intent), execution (order placement), and portfolio (position aggregation). Risk is defined as a bounded context with its own domain model, events, stream family, activation model, projection pipeline, and query surface — fully consistent with the patterns established by the five preceding domains.

---

## 2. Deliverables Produced

| # | Document                                            | Status   |
|---|-----------------------------------------------------|----------|
| 1 | `docs/architecture/risk-domain-design.md`           | Complete |
| 2 | `docs/architecture/risk-stream-families.md`         | Complete |
| 3 | `docs/architecture/risk-activation-and-ownership.md`| Complete |
| 4 | `docs/architecture/risk-query-surface-guidelines.md`| Complete |
| 5 | `docs/stages/stage-s62-risk-domain-design-report.md`| This file|

---

## 3. Key Design Decisions

### 3.1. Risk Is a Standalone Domain

Risk is not an extension of strategy. It has its own:
- Domain package: `internal/domain/risk/`
- Domain entity: `RiskAssessment` with disposition (approved/modified/rejected)
- Domain event: `RiskAssessedEvent`
- Stream: `RISK_EVENTS`
- Projection buckets: `RISK_{TYPE}_LATEST`
- Query endpoints: `GET /risk/{type}/latest`

### 3.2. Risk Lives in Derive

Risk evaluators are actors within the derive binary, receiving strategy output via local actor messages (not JetStream subscription). This follows the established pattern where signal, decision, and strategy evaluators all live in derive.

Extraction to a separate binary is explicitly deferred — only warranted if risk requires external state or independent scaling, neither of which applies in Phase 1.

### 3.3. First Family: Position Exposure

The initial risk family (`position_exposure`) is a **stateless, rule-based evaluator** that:
- Receives a strategy intent (direction + confidence).
- Calculates proposed position size against configurable limits.
- Returns approved/modified/rejected with constraints.

This family requires no external state, no position database, no portfolio aggregation. It demonstrates the risk pattern without introducing operational complexity.

### 3.4. Three Dispositions

| Disposition | Meaning                                            |
|-------------|-----------------------------------------------------|
| `approved`  | Intent accepted, constraints within limits          |
| `modified`  | Intent accepted with mandatory adjustments          |
| `rejected`  | Intent blocked (limit breach, insufficient signal)  |

This is richer than strategy's three directions (long/short/flat) because risk evaluates **acceptability**, not **direction**.

### 3.5. Domain-Owned Input Copies

Risk defines its own `StrategyInput` struct — it does not import strategy domain types. This maintains the same isolation pattern used by decision (which defines `SignalInput`) and strategy (which defines `DecisionInput`).

### 3.6. Latest-Only Projections

Phase 1 risk projections are latest-only. History projections are explicitly deferred to S65+ with clear prerequisites documented.

---

## 4. Boundary Decisions

### 4.1. Risk vs. Strategy

| Aspect          | Strategy                    | Risk                          |
|-----------------|-----------------------------|-------------------------------|
| Question        | "What should we do?"        | "Should we proceed?"          |
| Output          | Directional intent          | Gated intent with constraints |
| Statefulness    | Stateless                   | Stateless (Phase 1)           |
| Consumer        | Risk (local messages)       | Future execution layer        |

Strategy produces intent. Risk evaluates whether that intent is acceptable and under what constraints.

### 4.2. Risk vs. Execution

Risk says "yes/no/modified" to an intent. Execution places the actual order. Risk does not know about order books, exchange APIs, fill rates, or order lifecycle. Execution is a Phase 3+ concern.

### 4.3. Risk vs. Portfolio

Risk evaluates individual intents per partition key (source × symbol × timeframe). Portfolio aggregates positions across symbols and strategies. Cross-symbol correlation and aggregate exposure are portfolio concerns, not risk concerns in Phase 1.

### 4.4. Risk vs. Store

Store materializes risk projections and serves queries. Store never evaluates risk rules. The boundary is identical to all prior domains: store is read-side authority, derive is write-side authority.

### 4.5. Risk vs. Gateway

Gateway translates HTTP requests to NATS request/reply for risk queries. Gateway never interprets risk dispositions, applies risk rules, or accesses risk KV buckets directly.

---

## 5. Intentional Limits

### 5.1. What Was Explicitly Deferred

| Item                                    | Deferred To | Rationale                                             |
|-----------------------------------------|-------------|-------------------------------------------------------|
| Risk history projections                | S65+        | Latest-only sufficient for proving the pattern        |
| Multi-strategy risk evaluation          | S65+        | Phase 1 evaluates single strategy input               |
| Portfolio-level exposure aggregation    | Portfolio   | Cross-symbol aggregation is a different bounded context|
| Real-time position tracking             | Execution   | Risk evaluates at assessment time only                |
| External risk data feeds                | S66+        | No external dependencies in Phase 1                   |
| ClickHouse risk analytics               | S67+        | Analytical storage follows operational stability      |
| Drawdown guard family (RF-02)           | S65+        | Requires execution/portfolio state                    |
| Correlation limit family (RF-03)        | S66+        | Requires multi-symbol portfolio state                 |
| Volatility scaler family (RF-04)        | S66+        | Requires evidence layer expansion                     |
| Separate risk binary                    | If needed   | Only if derive throughput becomes bottleneck          |

### 5.2. What Was Explicitly Prohibited

- Risk does not open execution or portfolio domains.
- Risk does not create contracts for domains that don't exist yet.
- Risk does not aggregate across symbols (per-partition only).
- Risk does not track positions or P&L.
- Risk does not import strategy domain types.

---

## 6. Design Questions Answered

| Question                                                   | Answer                                                     |
|------------------------------------------------------------|------------------------------------------------------------|
| What is `risk` in the Foundry?                             | Sixth domain layer; evaluates strategy intents for acceptability |
| What is NOT `risk`?                                        | Not execution, portfolio, P&L, position tracking, or data quality |
| How does `risk` depend on `strategy`?                      | Local actor messages in derive; domain-owned input copies  |
| How does `risk` differ from `strategy`?                    | Strategy = directional intent; Risk = gated intent         |
| How does `risk` differ from `execution`?                   | Risk gates; execution places orders                        |
| How does `risk` differ from `portfolio`?                   | Risk = per-partition; portfolio = cross-symbol aggregate   |
| Separate binary or processor/family?                       | Family within derive binary (same pattern as all prior)    |
| Initial family?                                            | `position_exposure` (stateless, rule-based)                |
| Activation model?                                          | Two-layer: family config + binding runtime                 |
| Who publishes?                                             | RiskPublisherActor in derive                               |
| Who projects?                                              | RiskProjectionActor in store                               |
| Who serves query?                                          | QueryResponderActor in store; gateway translates to HTTP   |
| Latest/history?                                            | Latest-only Phase 1; history deferred to S65+              |
| Coupling invariants?                                       | 10 boundary invariants (RBI-1 through RBI-10)              |
| What's deferred?                                           | 10 items with clear stage assignments (see table above)    |

---

## 7. Preparation for Next Stages

### S63 — Risk Governance Activation

**Ready to execute.** This design provides:
- 10 domain boundary invariants for raccoon-cli enforcement.
- Stream family definition for catalog registration.
- Ownership matrix for static analysis rules.
- Import prohibition graph (risk ↛ strategy domain, risk ↛ execution).

### S64 — Risk First Slice

**Blocked on S60, S61, S63.** This design provides:
- Complete domain model specification.
- Event type definition.
- Actor tree placement with conditional spawning logic.
- Store projection pattern with three materialization gates.
- Gateway endpoint specification.
- Configuration schema extension with dependency DAG.
- All implementation targets identified.

### Stage Dependency Graph

```
S60: Adapter Test Coverage Sweep  ──┐
S61: Derive Actor Test Coverage   ──┤
S62: Risk Domain Design (THIS)   ──┼──► S64: Risk First Slice ──► S65: Risk Projection Hardening
S63: Risk Governance Activation  ──┘
```

---

## 8. Confidence Assessment

| Dimension                          | Score  | Notes                                              |
|------------------------------------|--------|----------------------------------------------------|
| Domain boundary clarity            | 9/10   | 10 invariants, explicit differentiation matrix      |
| Pattern consistency                | 10/10  | Identical patterns to strategy/decision/signal      |
| First family feasibility           | 9/10   | Stateless, rule-based, no external dependencies     |
| Activation model readiness         | 10/10  | Two-layer model proven across 5 domains             |
| Query surface clarity              | 9/10   | Four-layer chain, all types/subjects specified      |
| Separation from future layers      | 9/10   | Execution and portfolio explicitly excluded          |
| Deferral honesty                   | 10/10  | 10 deferred items with stage assignments            |
| Implementation readiness           | 8/10   | Blocked only on S60/S61/S63, all design complete    |
| **Overall**                        | **9.25/10** |                                                |

---

## 9. Verdict

**Stage S62 is COMPLETE.** The risk domain is formally designed as a standalone bounded context with clear boundaries, consistent patterns, and a minimal first family that proves the evaluation model without operational complexity.

The design is ready for governance activation (S63) and first slice implementation (S64, pending hardening prerequisites).
