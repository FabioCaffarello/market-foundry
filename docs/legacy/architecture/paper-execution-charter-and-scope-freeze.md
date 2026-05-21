# Paper Execution Charter and Scope Freeze

Stage: S264
Status: Open — Scope Frozen
Predecessor: S263 (Post-Codegen Reentry Gate)
Date: 2026-03-21

---

## 1. Charter Statement

**The Paper Execution wave exists to prove that the decision → strategy → risk → execution chain can complete a full operational loop in paper mode, under strong guard rails, without opening venue real, OMS, portfolio, or multi-venue scope.**

This charter formalizes the transition from domain-breadth and codegen governance work (S241–S263) into feature-evolution territory. The system already possesses all five paper execution components (`PaperOrderEvaluator`, `PaperFillSimulator`, `PaperVenueAdapter`, `SafetyGate`, `StalenessGuard`), behavioral activation across all domain boundaries, and codegen governance for execution artifacts. What remains is proving the loop closes end-to-end under realistic operational conditions — not expanding infrastructure or opening new domain surfaces.

## 2. Strategic Rationale

### Why now

| Factor | Evidence |
|--------|----------|
| Domain maturity | Decision, strategy, risk fully behavioral-activated (S249–S257) |
| Execution components present | All five paper execution components implemented and unit-tested |
| Codegen governance stable | 22 artifacts governed, zero drift (S263 gate PASS) |
| Actor wiring exists | `PaperOrderEvaluatorActor` + `ExecutionPublisherActor` in derive scope |
| Projection exists | `ExecutionProjectionActor` in store scope with monotonicity enforcement |
| Natural next step | S263 gate recommended "pivot to feature evolution" |

### Why not alternatives

| Alternative | Rejection reason |
|-------------|-----------------|
| Open venue real | Premature — paper loop not yet proven end-to-end under operational conditions |
| New breadth wave | Breadth already covers 10 families across 5 domains; diminishing returns |
| OMS / portfolio | Requires proven paper loop as prerequisite; opening now creates ungrounded complexity |
| Multi-venue | Venue abstraction exists but real venue integration is a distinct, later concern |
| More codegen coverage | 14% codegen coverage is by design; domain logic stays manual |

## 3. Scope Definition

### Tier 1 — Required (must complete for wave success)

| ID | Scope item | Rationale |
|----|-----------|-----------|
| T1-1 | End-to-end paper loop proof: decision → strategy → risk → execution intent → simulated fill | Core objective — proves the chain closes |
| T1-2 | SafetyGate integration proof under operational conditions | Guard rails must be proven active, not just unit-tested |
| T1-3 | StalenessGuard rejection proof with aged intents | Stale intent rejection is a safety-critical path |
| T1-4 | Kill switch (ControlGate) halt proof | Emergency stop must demonstrably halt execution |
| T1-5 | Behavioral context preservation across full chain | CorrelationID, CausationID, severity must survive decision → fill |
| T1-6 | Execution projection round-trip proof (event → KV materialization) | Write model must prove data reaches read model |

### Tier 2 — Permitted (may complete if time allows, not required)

| ID | Scope item | Condition |
|----|-----------|-----------|
| T2-1 | Dual-chain paper loop (both Chain A and Chain B) | Only after T1-1 through T1-6 pass for at least one chain |
| T2-2 | Paper loop latency characterization | Only observational; no performance optimization work |
| T2-3 | Error path documentation for paper execution failures | Documentation only; no new error handling infrastructure |

### Out of Scope

| Item | Rationale |
|------|-----------|
| Real venue integration (Binance, testnet, any exchange) | Paper mode only; venue real is a future wave |
| OMS (Order Management System) | Not a prerequisite for paper loop proof |
| Portfolio tracking or PnL calculation | Downstream of proven loop; separate concern |
| Multi-venue routing or venue selection | Single paper venue only |
| New signal families or risk evaluators | Breadth is frozen; use existing families |
| Execution retry logic or recovery | Paper fills are instant and deterministic |
| New NATS streams or JetStream infrastructure | Existing streams sufficient |
| ClickHouse schema changes for execution | Existing writer pipeline and schema sufficient |
| Real money, real orders, real positions | Explicitly prohibited |
| Performance optimization | Observation permitted; optimization prohibited |

## 4. Domain Interaction Model

### Current state (post-S263)

```
Decision (rsi_oversold / ema_crossover)
  │ severity + confidence
  ▼
Strategy (mean_reversion_entry / trend_following_entry)
  │ scaled confidence + parameters
  ▼
Risk (position_exposure / drawdown_limit)
  │ disposition + scaled limits
  ▼
Execution [components exist, wired in actors, but loop not proven end-to-end]
  ├── PaperOrderEvaluator  → ExecutionIntent
  ├── PaperFillSimulator    → Filled intent
  ├── SafetyGate            → SafetyVerdict
  ├── StalenessGuard        → Age validation
  └── PaperVenueAdapter     → Simulated venue receipt
```

### Target state (post-wave)

```
Decision (rsi_oversold / ema_crossover)
  │ severity + confidence + correlation_id
  ▼
Strategy (mean_reversion_entry / trend_following_entry)
  │ scaled confidence + causation_id
  ▼
Risk (position_exposure / drawdown_limit)
  │ disposition + causation_id
  ▼
Execution [loop proven closed, guard rails proven active]
  ├── SafetyGate ──── ControlGate (kill switch): PROVEN halt
  │                └── StalenessGuard: PROVEN rejection of aged intents
  ├── PaperOrderEvaluator → ExecutionIntent: PROVEN translation
  ├── PaperFillSimulator  → Filled intent: PROVEN instant fill
  ├── PaperVenueAdapter   → Venue receipt: PROVEN simulated submission
  └── ExecutionProjection → KV materialization: PROVEN round-trip
```

### Preserved invariants

- All domain boundaries remain behavioral-activated (severity scaling, confidence propagation)
- Codegen governance for execution artifacts remains intact (zero drift tolerance)
- Actor isolation model unchanged (derive scope → store scope separation)
- NATS stream topology unchanged (`EXECUTION_EVENTS`, `EXECUTION_FILL_EVENTS`)
- KV bucket semantics unchanged (latest-only, monotonicity by timestamp)

## 5. Minimum Viable Scenarios

| # | Scenario | Proves |
|---|----------|--------|
| 1 | High-severity decision triggers paper order with full fill | Chain closes: decision → strategy → risk → execution intent → simulated fill |
| 2 | SafetyGate blocks stale intent (age > maxAge) | Staleness guard rails are operationally active |
| 3 | Kill switch (ControlGate) halts execution mid-flow | Emergency stop works at execution boundary |
| 4 | Low-severity decision produces reduced position via paper loop | Behavioral scaling (severity → confidence → position) survives into execution |
| 5 | Correlation/causation IDs survive from decision through fill | Causal trace integrity across all four domains |
| 6 | Execution event materializes to KV projection | Write model → read model round-trip for execution data |
| 7 | No-action risk disposition produces no execution intent | Negative path: system correctly does nothing when risk says no |

## 6. Success Criteria

| ID | Criterion | Verification |
|----|-----------|-------------|
| SC-1 | All 7 minimum viable scenarios pass | Automated test suite, CI green |
| SC-2 | Paper mode only — no real venue calls in any test or code path | Code review + grep for real venue invocations |
| SC-3 | Guard rails proven active (not just present) | Scenarios 2, 3, 7 pass with explicit assertions |
| SC-4 | Behavioral context preserved across full chain | Scenario 5 asserts correlation_id, causation_id, severity at fill |
| SC-5 | Zero regression in existing test suite | CI green on all existing tests |
| SC-6 | Zero codegen drift introduced | `codegen-equivalence-check.sh` passes |
| SC-7 | Execution projection proves materialization | Scenario 6 verifies KV state after event publish |

## 7. Non-Success Criteria

- Real venue order placement is NOT a success criterion
- Performance benchmarks are NOT required
- Multi-chain coverage (both Chain A and Chain B) is NOT required (Tier 2)
- ClickHouse persistence of execution data is NOT required (writer pipeline already proven in S255)
- New error handling infrastructure is NOT required

## 8. Hardening Budget

Up to 15% of wave effort may be spent on hardening existing execution test infrastructure (fixtures, helpers, test utilities). No new test frameworks or libraries may be introduced. Hardening must serve the minimum viable scenarios directly.

## 9. Planned Stage Sequence

| Stage | Deliverable | Depends on |
|-------|------------|------------|
| S264 | Charter and scope freeze (this document) | S263 gate PASS |
| S265 | Paper execution boundary alignment and wiring validation | S264 charter open |
| S266 | End-to-end paper loop scenario implementation | S265 boundary alignment |
| S267 | Guard rail activation proof (safety gate, kill switch, staleness) | S266 loop scenarios |
| S268 | Post-paper-execution gate | S266 + S267 complete |

### Sequencing rationale

S265 validates that existing wiring is correct and complete before S266 attempts to close the loop. S267 proves guard rails separately to keep safety concerns isolated from happy-path scenarios. S268 gates the wave before any subsequent work can begin.

## 10. Governance Framework

### Amendment rules

- Tier 1 scope items may NOT be removed without charter re-vote
- Tier 2 scope items may be added or removed by consensus
- Out of Scope items may NOT be moved into scope without a new charter
- New scenarios may be added to Tier 2 if they serve an existing Tier 1 item

### Stop conditions threshold

- Any real venue call detected in code or tests → immediate stop
- Any new NATS stream or JetStream infrastructure created → immediate stop
- Any new domain surface opened (new signal family, new risk evaluator) → immediate stop
- CI regression exceeding 2 tests → pause and assess
- Codegen equivalence drift detected → pause and assess

### Behavioral test gates

All behavioral tests from S249–S257 must remain green throughout the wave. Any behavioral regression is a hard stop.

## 11. Amendments Log

| Date | Amendment | Justification |
|------|-----------|---------------|
| — | (none) | — |
