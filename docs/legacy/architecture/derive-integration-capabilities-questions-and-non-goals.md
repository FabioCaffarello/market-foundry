# Derive Integration — Capabilities, Questions, and Non-Goals

> Companion to the [Derive Integration Wave Charter](derive-integration-wave-charter-and-scope-freeze.md).
> Defines the capabilities under assessment, governing questions, evaluation
> criteria, and explicit non-goals for the wave.
>
> Date: 2026-03-22.

---

## 1. Context

"Derive integration" means proving that the derive binary is a correct,
contract-compliant producer of `StrategyResolvedEvent` and that the full
analytical-to-execution pipeline — from signal ingestion through strategy
resolution to paper order execution — works as a connected system.

The Strategy/Signal Integration Wave (S358–S363) proved the consumer side.
This wave closes the producer side for **one** signal+strategy pair
(RSI + mean_reversion_entry) under **paper** execution.

---

## 2. Capabilities Under Assessment

| # | Capability | Current State | Target State |
|---|---|---|---|
| DC-1 | Derive producer contract compliance | Resolver and publisher exist; never audited against S359 invariants | Field-level compliance verified for all 11 invariants |
| DC-2 | Derive publisher correctness | Publisher publishes to STRATEGY_EVENTS; subject and dedup format untested | Unit tests verify subject format, dedup key, payload shape |
| DC-3 | Store materialization of derive output | Store projection exists; never exercised with derive-produced events | Integration test proves materialization with monotonicity guard |
| DC-4 | Gateway read path for derive output | Gateway query routes exist; never exercised with derive-produced state | HTTP endpoint returns derive-produced strategy state |
| DC-5 | Analytical-to-execution correlation chain | CorrelationID proven in execute scope (S362); unproven across derive→execute | Full correlation chain verified: signal → derive → execute |
| DC-6 | End-to-end pipeline proof | Synthetic events proven (S362); real derive pipeline never connected to execute | Full pipeline test: signal → derive → strategy → execute → fill |

---

## 3. Governing Questions

### DI-1: Producer Spec and Derive Ownership

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| DIQ-1 | Does the derive strategy resolver produce `StrategyResolvedEvent` payloads that satisfy all 11 S359 contract invariants? | DI-1 | Audit matrix + code review |
| DIQ-2 | Is there a documented field-level compliance mapping between derive resolver output and the S359 contract? | DI-1 | Document |

### DI-2: Canonical Derive Producer Wiring

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| DIQ-3 | Do unit tests prove each of the 11 S359 invariants holds on the derive producer side? | DI-2 | Unit tests |
| DIQ-4 | Does the derive strategy publisher produce NATS messages with correct subject format, deduplication key, and payload shape? | DI-2 | Unit tests |

### DI-3: Store/Gateway/Read-Path Verification

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| DIQ-5 | Does the store projection correctly materialize derive-produced `StrategyResolvedEvent` into KV buckets? | DI-3 | Integration test |
| DIQ-6 | Does the gateway HTTP endpoint return derive-produced strategy state with correct field mapping? | DI-3 | Integration test |

### DI-4: Analytical-to-Execution End-to-End Proof

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| DIQ-7 | Does a single end-to-end test demonstrate the full connected pipeline: signal → derive → strategy event → execute → fill? | DI-4 | Integration test (NATS + derive + execute) |
| DIQ-8 | Does the correlation chain propagate correctly from signal origin through derive resolution to execution fill? | DI-4 | Integration test assertion |

---

## 4. Evaluation Criteria

Each capability and question is classified using the standard scale:

| Rating | Meaning |
|---|---|
| **FULL** | Capability demonstrated with complete evidence; all related questions answered YES |
| **SUBSTANTIAL** | Most evidence present; minor gaps documented but non-blocking |
| **PARTIAL** | Capability demonstrated in limited scope; material gaps remain |
| **PENDING** | Not demonstrated in this wave (out-of-scope per charter) |

---

## 5. Non-Goals

The following items are explicitly excluded from this wave. Each non-goal
includes a rationale.

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-1 | **Batch strategy family onboarding** — integrating trend_following_entry, squeeze_breakout_entry, or any other family | Depth over breadth: prove contract compliance with one family, others follow the proven pattern mechanically |
| NG-2 | **Multiple signal families** — integrating more than RSI | Canonical pair (RSI + mean_reversion_entry) selected in S359; additional signals are breadth expansion |
| NG-3 | **Multi-venue execution** — routing derive-driven orders to multiple venues | Venue expansion is a separate wave; paper adapter is sufficient proof |
| NG-4 | **OMS (Order Management System)** — order lifecycle tracking, fills reconciliation, position management | OMS is a distinct domain; this wave proves derive → intent, not intent → position |
| NG-5 | **Portfolio risk management** — portfolio-level exposure, correlation analysis, drawdown limits | Requires OMS and risk domain; out of scope |
| NG-6 | **Mainnet execution** — any real-money order submission | All proof uses paper adapter; mainnet is an operational readiness gate |
| NG-7 | **Dashboard or UI** — Grafana dashboards, web UI, or any visualization layer | Prometheus metrics are exposed; consumption is infrastructure, not domain |
| NG-8 | **Derive runtime redesign** — refactoring actor hierarchy, changing supervisor topology, reworking family processor pattern | Derive actors work correctly; this wave proves contract compliance, not architecture improvement |
| NG-9 | **Multi-binary orchestration (Docker Compose)** — containerized multi-service deployment | Tests compose in-process; Docker Compose is a separate wave |
| NG-10 | **Push alerting** — Alertmanager, PagerDuty, or any notification system | Deferred from prior waves; remains infrastructure concern |
| NG-11 | **Log aggregation** — centralized log collection (ELK, Loki, etc.) | Deferred from prior waves; remains infrastructure concern |
| NG-12 | **Strategy parameter optimization** — backtesting, parameter sweeps, ML-based tuning | Optimization requires data infrastructure not yet built |
| NG-13 | **Historical replay** — replaying historical market data through the pipeline | Replay requires ingest backfill and time-simulation capabilities |
| NG-14 | **Risk domain implementation** — formal risk evaluation between strategy and execution | Risk is critical but separate; pass-through risk (INV-4) is by design for this wave |
| NG-15 | **New domain types** — creating new domains (portfolio, allocation, etc.) | Wiring existing domains, not creating new ones |

---

## 6. Relationship to S359 Contract

This wave's primary obligation is proving that derive's producer output
satisfies the contract established in S359. The contract defines:

| Contract element | S359 reference | This wave's responsibility |
|---|---|---|
| 11 binding invariants (INV-1 through INV-11) | Source-selection-and-canonical-integration-contract §4 | Verify derive output satisfies each invariant |
| Field-level mapping (StrategyResolvedEvent → ExecutionIntent) | Source-selection-and-canonical-integration-contract §3 | Verify derive produces all required fields |
| NATS consumer spec (durable, subject, stream) | Source-selection-and-canonical-integration-contract §5 | Verify derive publishes to matching subject |
| 5 domain boundary specifications | Source-selection-and-canonical-integration-contract §6 | Verify derive respects domain boundaries |

The S360 consumer spec (StrategyConsumerActor) is **not modified** in this
wave. The consumer is proven correct. This wave proves the producer is
correct, then connects them.

---

## 7. Relationship to Existing Derive Implementation

The derive binary already has the following components implemented:

| Component | File | Status |
|---|---|---|
| MeanReversionEntryResolverActor | `internal/actors/scopes/derive/strategy_resolver_actor.go` | EXISTS — needs contract audit |
| TrendFollowingEntryResolverActor | `internal/actors/scopes/derive/trend_following_entry_resolver_actor.go` | EXISTS — out of scope (NG-1) |
| SqueezeBreakoutEntryResolverActor | `internal/actors/scopes/derive/squeeze_breakout_entry_resolver_actor.go` | EXISTS — out of scope (NG-1) |
| StrategyPublisherActor | `internal/actors/scopes/derive/strategy_publisher_actor.go` | EXISTS — needs publisher audit |
| NATS strategy publisher | `internal/adapters/nats/natsstrategy/publisher.go` | EXISTS — needs subject/dedup audit |
| NATS strategy registry | `internal/adapters/nats/natsstrategy/registry.go` | EXISTS — 3 families registered |
| Actor messages | `internal/actors/scopes/derive/messages.go` | EXISTS — publishStrategyMessage, strategyResolvedMessage |

The wave's implementation risk is **low** because all building blocks exist.
The work is primarily **auditing, testing, and proving composition**, not
building new capabilities.

---

## 8. Relationship to Prior Residual Gaps

The S363 evidence matrix cataloged 14 items (2 wave-scoped, 12 deferred).
This wave's relationship to the key items:

| Gap | Source | This wave |
|---|---|---|
| DG-11: No derive-side strategy event production | S363 deferred | **PRIMARY TARGET** — this wave's core objective |
| DG-12: No multi-binary orchestration test | S363 deferred | **Not addressed** (NG-9) — separate wave |
| WG-1: Gateway SourcePathConfigProvider not wired | S363 wave-scoped | **Potentially addressed** in DI-3 if gateway verification requires it |
| WG-2: Confidence threshold not exercised in E2E | S363 wave-scoped | **Not primary target** — may be exercised incidentally in DI-4 |

---

## References

- [Derive Integration Wave Charter](derive-integration-wave-charter-and-scope-freeze.md)
- [Source Selection and Canonical Integration Contract (S359)](source-selection-and-canonical-integration-contract.md)
- [Controlled Source-to-Execution Wiring (S360)](controlled-source-to-execution-wiring.md)
- [Strategy/Signal Integration Evidence Matrix](strategy-signal-integration-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Derive Pipeline Pattern](derive-pipeline-pattern.md)
- [Derive Family Processor Pattern](derive-family-processor-pattern.md)
