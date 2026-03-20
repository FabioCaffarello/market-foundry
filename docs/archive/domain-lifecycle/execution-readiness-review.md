# Execution Readiness Review

> Formal readiness assessment for opening an `execution` domain layer in Market Foundry.
> Date: 2026-03-18 | Stage: S68

## 1. Review Scope

This document evaluates whether Market Foundry has sufficient maturity, governance, and architectural health to open an `execution` domain in a future development cycle. Unlike previous readiness reviews (signal, decision, strategy, risk), this review carries elevated scrutiny: `execution` is the first layer that crosses the **action boundary** — it can trigger real-world financial operations with irreversible consequences.

**Guiding principle**: Market Foundry progresses by readiness, not by domain anxiety. An `execution` layer will only be recommended if every preceding layer is demonstrably solid and the gate criteria are materially stricter than any previous domain entry.

**Critical distinction**: This review assesses readiness to **design** the execution domain. It does not assess readiness to **implement** it. The readiness gate sequence (governance current → contracts defined → activation verified → pattern proven → implementation) means several intermediate stages must pass between this review and any execution code.

---

## 2. Domain Maturity Assessment

### 2.1 Observation

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | `ObservationTrade` with 8 fields, decimal strings, full validation, dedup keys |
| Application logic | STRONG | Binding parser with normalization and validation |
| Adapters | STRONG | NATS registry (stream `OBSERVATION_EVENTS`, 6h retention), publisher, consumer |
| Actors | STRONG | Ingest scope: websocket, exchange scope, publisher, binding watcher |
| Tests | MODERATE | Domain model tested (6 cases), binding tested; no publisher/consumer adapter tests, no ingest actor tests |
| HTTP surface | N/A | Observation is internal — no HTTP exposure by design |

**Rating: 7.5/10** — Production-grade pipeline. Adapter/actor test gap persists since S59.

### 2.2 Evidence

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Three types (candle, trade burst, volume), full validation, decimal precision |
| Application logic | STRONG | Pure samplers with window accumulation, Big.Float precision |
| Client layer | STRONG | 4 use cases with validation |
| Adapters | STRONG | Registry (21+ tests), gateway, publisher, consumer; KV stores |
| Actors | STRONG | Derive: sampler actors per type; Store: three-gate projection |
| Tests | MODERATE | Domain: 22+ tests; samplers: 8+ tests; registry: 21+ tests; no publisher/consumer adapter tests |
| HTTP surface | STRONG | 4 endpoints with tests |

**Rating: 8/10** — Three validated evidence types. Same adapter test gap.

### 2.3 Signal

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Signal struct with 7 fields, decimal value, partition/dedup keys; 9 tests |
| Application logic | STRONG | RSI sampler (Wilder's smoothed, period=14), streaming |
| Client layer | STRONG | `GetLatestSignalUseCase` with validation |
| Adapters | STRONG | Registry with type routing, gateway, KV store with monotonicity |
| Actors | STRONG | Derive: signal sampler + publisher; Store: projection + query responder |
| Tests | MODERATE | Domain: 9 tests; RSI: tested; KV store: guard tests; no adapter tests |
| HTTP surface | STRONG | `GET /signal/:type/latest` with type routing |

**Rating: 8/10** — Complete, extensible by type.

### 2.4 Decision

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | 10 fields, outcome enum (triggered/not_triggered/insufficient), 14 tests |
| Application logic | STRONG | RSI oversold evaluator (threshold=30.0), pure, tested |
| Client layer | STRONG | `GetLatestDecisionUseCase`, 5 tests |
| Adapters | STRONG | Registry, gateway, KV store with monotonicity |
| Actors | STRONG | Derive: evaluator + publisher; Store: projection + query responder |
| Tests | MODERATE | Domain: 14 tests; evaluator: tested; projection: tested; no adapter tests |
| HTTP surface | STRONG | `GET /decision/:type/latest` with type routing |

**Rating: 8.5/10** — Most mature mid-chain domain.

### 2.5 Strategy

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Direction enum (long/short/flat), DecisionInput provenance, 14 tests |
| Application logic | STRONG | MeanReversionEntryResolver (pure), 8 tests; use case, 5 tests |
| Adapters | STRONG | Registry (8 tests), publisher, consumer, gateway, KV store (8 tests) |
| Actors | STRONG | Derive: resolver + publisher; Store: projection (21 tests, multi-symbol) |
| Tests | STRONG (projection) / MODERATE (adapters) | Projection: 21 tests including multi-symbol isolation |
| HTTP surface | STRONG | `GET /strategy/:type/latest`, handler (10 tests), routes (3 tests) |
| Documentation | EXCEPTIONAL | Domain design (2400+ lines), readiness review, activation/ownership |

**Rating: 8.5/10** — Production-ready. Best-documented domain.

### 2.6 Risk

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | RiskAssessment with disposition enum (approved/modified/rejected), Constraints, StrategyInput provenance, 22 tests |
| Application logic | STRONG | PositionExposureEvaluator (pure, no I/O), confidence scaling, position sizing, 19 tests |
| Client layer | STRONG | `GetLatestRiskUseCase` with validation, 4 tests |
| Adapters | STRONG | Registry (8 tests), publisher, consumer, KV store (6+ tests), gateway |
| Actors | STRONG | Derive: evaluator + publisher; Store: projection (11 tests, multi-symbol, stats invariant) |
| Tests | STRONG | 75+ tests total across domain, application, adapters, projection, HTTP |
| HTTP surface | STRONG | `GET /risk/:type/latest`, handler (5 tests) |
| Multi-symbol | VERIFIED | S66 multi-symbol verification: partition isolation, dedup isolation, no cross-bleed |
| Traceability | HARDENED | S67 causation_id propagation, structured log trace context |

**Rating: 8.5/10** — Complete first risk family with multi-symbol verification and traceability hardening.

---

## 3. Projection Authority

| Aspect | Status | Evidence |
|--------|--------|----------|
| Single-writer invariant | ENFORCED | Each KV bucket has exactly one writer actor; documented in stream-ownership-matrix |
| Store as sole authority | ENFORCED | Store binary owns all projection pipelines; gateway reads via NATS request/reply |
| Gateway statelessness | ENFORCED | Gateway has no pipeline config, discovers routes from deps at startup |
| Monotonicity guards | ENFORCED | All KV stores reject stale/duplicate writes via timestamp comparison |
| Three-gate projection | ENFORCED | Final gate → validation gate → monotonicity guard in all 5 projection actor types |
| Replay idempotency | ENFORCED | Dedup keys include type + coordinates + timestamp; JetStream MsgID set on publish |
| Deduplication consistency | ENFORCED | Pattern consistent across evidence, signal, decision, strategy, risk |

**Rating: 9/10** — Projection authority model is sound, proven across 5 domains.

**Execution-specific concern**: An `execution` domain would introduce the first projection where **downstream systems may take real-world action** based on materialized state. The projection authority model must be extended with additional safeguards:
- Execution state must never be stale without detection
- Execution projections need explicit staleness windows (not just "latest wins")
- The single-writer invariant becomes safety-critical, not just correctness-critical

---

## 4. Query Surface Quality

| Surface | Endpoint | Tests | Conditional | Notes |
|---------|----------|-------|-------------|-------|
| Evidence candle latest | GET /evidence/candles/latest | YES | YES | Full handler + route tests |
| Evidence candle history | GET /evidence/candles/history | YES | YES | Range query |
| Evidence tradeburst latest | GET /evidence/tradeburst/latest | YES | YES | |
| Evidence volume latest | GET /evidence/volume/latest | YES | YES | |
| Signal latest | GET /signal/:type/latest | YES | YES | Type-routed |
| Decision latest | GET /decision/:type/latest | YES | YES | Type-routed |
| Strategy latest | GET /strategy/:type/latest | YES | YES | Type-routed, 10 handler tests |
| Risk latest | GET /risk/:type/latest | YES | YES | Type-routed, 5 handler tests |
| Healthz | GET /healthz | YES | NO | Always registered |
| Readyz | GET /readyz | YES | NO | Readiness checks |

**Rating: 9/10** — All surfaces conditional, tested, consistent.

**Execution-specific concern**: An execution query surface would need to expose not just "latest state" but potentially "current active orders," "execution status," and "position state." These require richer query semantics than the current latest-only pattern supports.

---

## 5. Governance and CLI Coverage

| Check | Scope | Status |
|-------|-------|--------|
| Architecture layer boundaries | 11 rules | ENFORCED |
| Stream registry consistency | All 6 domains | ENFORCED |
| Config ↔ Compose alignment | All services | ENFORCED |
| Naming identity | No residual "server" refs | ENFORCED |
| Signal drift rules (SD-1..SD-5) | Docs, adapters, domain, config, contracts | ENFORCED |
| Decision drift rules (DD-1..DD-5) | Same scope | ENFORCED |
| Strategy drift rules (STD-1..STD-5) | Same scope | ENFORCED |
| Risk drift rules (RD-1..RD-5) | Inferred from pattern | NOT VERIFIED |
| Config symmetry (derive ↔ store) | Pipeline family arrays | ENFORCED |
| Quality gate profiles | fast, ci, deep | ENFORCED |
| Causal chain guidelines | Documented | DOCUMENTED (S67) |
| Stage definition of done | 13-point checklist | ENFORCED |

**Gaps in governance:**

| Gap | Severity | Impact on execution |
|-----|----------|---------------------|
| Risk-specific drift rules not explicitly verified | LOW | Must be verified before execution design begins |
| Cannot verify resolver/evaluator purity at CLI level | LOW | Mitigated by code review and pure function pattern |
| Cannot enforce KV bucket configuration consistency | LOW | Mitigated by infra review |
| Cannot enforce activation dependency chain at CLI level | MEDIUM | Strategy can theoretically be enabled without decision; mitigated by startup validation |
| No execution-specific governance rules exist | EXPECTED | Must be created as part of execution domain design |
| No adapter/consumer test enforcement rule | MEDIUM | Gap grows with each domain; execution makes it riskier |

**Rating: 8.5/10** — Governance is mature and mechanically extensible.

---

## 6. Activation Model

| Layer | Mechanism | Status |
|-------|-----------|--------|
| Family activation (structural) | `pipeline.*_families` config keys | ENFORCED — requires restart |
| Binding activation (runtime) | BindingWatcherActor + configctl events | ENFORCED — dynamic per symbol |
| Dependency validation | `ValidatePipeline()` checks transitive deps | ENFORCED — blocks binary startup on invalid chain |
| Known family registry | Closed set per layer, rejects unknown names | ENFORCED — typo protection |

**Current validated dependency chain:**
```
observation → evidence (candle) → signal (rsi) → decision (rsi_oversold) → strategy (mean_reversion_entry) → risk (position_exposure)
```

**For execution, the chain extends:**
```
... → risk (position_exposure) → execution (???)
```

**Execution-specific concern**: The activation model currently supports **opt-in by config restart**. An execution layer that controls real orders may need:
- Explicit activation ceremony (not just config change)
- Kill switch mechanism (disable execution without full restart)
- Separate activation for paper trading vs. live trading
- Activation audit trail

**Rating: 9/10** — Two-layer activation with transitive validation is production-grade for observation→risk. Execution introduces new activation requirements.

---

## 7. Config Dependency Safety

| Rule | Status | Evidence |
|------|--------|----------|
| Signal depends on evidence | ENFORCED | `signalDependsOnEvidence["rsi"] = ["candle"]` |
| Decision depends on signal | ENFORCED | `decisionDependsOnSignal["rsi_oversold"] = ["rsi"]` |
| Strategy depends on decision | ENFORCED | `strategyDependsOnDecision["mean_reversion_entry"] = ["rsi_oversold"]` |
| Risk depends on strategy | ENFORCED | `riskDependsOnStrategy["position_exposure"] = ["mean_reversion_entry"]` |
| Unknown family rejection | ENFORCED | Closed sets per layer |
| Derive ↔ Store symmetry | ENFORCED by CLI | Drift-detect checks pipeline arrays match |

**For execution**: A new `executionDependsOnRisk` map must be added, plus `knownExecutionFamilies` closed set. The pattern is mechanical and proven through 5 layers.

**Rating: 9/10**

---

## 8. Mesh Integrity

| Stream | Owner | Consumers | Status |
|--------|-------|-----------|--------|
| CONFIGCTL_EVENTS | configctl | ingest, derive | HEALTHY |
| OBSERVATION_EVENTS | ingest | derive | HEALTHY |
| EVIDENCE_EVENTS | derive | store | HEALTHY |
| SIGNAL_EVENTS | derive | store | HEALTHY |
| DECISION_EVENTS | derive | store | HEALTHY |
| STRATEGY_EVENTS | derive | store | HEALTHY |
| RISK_EVENTS | derive | store | HEALTHY |

**Single-writer verified**: Each stream has exactly one publisher binary. Store consumes and materializes. Gateway reads from store KV buckets via request/reply.

A future `EXECUTION_EVENTS` stream would follow the same pattern — but with a critical difference: **execution events may need to flow to an external venue adapter**, not just to store. This breaks the current unidirectional flow: `configctl → ingest → derive → store → gateway`.

**Rating: 9/10** — Mesh is clean and mechanically extensible, but execution introduces flow topology questions.

---

## 9. Traceability and Auditability

| Aspect | Status | Evidence |
|--------|--------|----------|
| CorrelationID propagation | ENFORCED | Chain-wide trace ID from observation through risk |
| CausationID propagation | ENFORCED (S67) | Parent-child linking across all derive actors |
| Structured log trace context | ENFORCED (S67) | Both IDs in all derive and store actor logs |
| NATS envelope trace fields | ENFORCED (S67) | Both IDs in transport envelope |
| Log-based trace reconstruction | OPERATIONAL | Filter by correlation_id to reconstruct full chain |
| KV projection trace metadata | NOT PERSISTED | Trace available in JetStream and logs, not in KV |
| Automated traceability verification | NOT IMPLEMENTED | Manual/visual verification only |
| Causal chain guidelines | DOCUMENTED | Guidelines for maintaining integrity in new layers |

**Execution-specific concern**: For `execution`, auditability is not a "nice to have" — it is a regulatory and operational necessity:
- Every order placed must be traceable to the signal/decision/strategy/risk chain that produced it
- Trace metadata must be persisted (not just in logs) for post-trade analysis
- Automated verification of the causal chain becomes a hard requirement, not a gap to acknowledge

**Rating: 8/10** — Traceability is structural and operational. Two gaps (KV persistence, automated verification) become blockers for execution.

---

## 10. Cross-Cutting Test Gap Analysis

The most significant gap across all domains remains the **absence of adapter-level and actor-level unit tests** for publishers, consumers, and derive actors:

| Component | Test Status | Affected Domains |
|-----------|-------------|------------------|
| NATS publisher adapters | NO TESTS | all 6 domains |
| NATS consumer adapters | NO TESTS | all 6 domains |
| Ingest actor scope | NO TESTS | observation |
| Derive actor scope | NO TESTS | evidence, signal, decision, strategy, risk |
| Store projection actors | TESTED | all domains (11-21 tests per projection) |
| Domain models | TESTED | all domains (9-22 tests each) |
| Application logic | TESTED | all domains (pure functions, table-driven) |
| HTTP handlers | TESTED | all query surfaces |
| NATS registries | TESTED | all domains |
| KV stores | TESTED | guard tests + functional |

**Impact on execution readiness**: This gap has been carried since S59. For execution, untested adapter code represents a higher risk — a message encoding bug in a risk publisher is a data quality issue; a message encoding bug in an execution publisher could result in malformed orders.

**Recommendation**: Adapter test coverage should be addressed before execution domain design begins. This gap has been documented as acceptable for observation→risk but is not acceptable for execution.

---

## 11. Readiness Verdict

### Is Market Foundry ready to open an `execution` layer?

**Verdict: CONDITIONALLY READY — with elevated prerequisites**

The architectural foundation is sound. Six domains (observation, evidence, signal, decision, strategy, risk) are implemented and operational. Governance is mature. Traceability is structural. The mesh is clean. The projection authority model is proven.

However, `execution` is qualitatively different from all previous domains:

1. **It crosses the action boundary** — every previous domain is observational or analytical. Execution produces real-world side effects.
2. **Errors are not self-healing** — a stale risk projection is corrected by the next event. A stale execution state may result in duplicate orders.
3. **Auditability is mandatory, not optional** — post-trade analysis requires persisted trace metadata, not just log-based reconstruction.
4. **The unidirectional flow model may need extension** — execution may need to communicate with external venue adapters, introducing a feedback path.

### Prerequisites Before Execution Domain Design (S69+)

| # | Prerequisite | Severity | Resolution Path |
|---|-------------|----------|-----------------|
| P-1 | Adapter test coverage sweep | HIGH | Dedicated stage: publisher/consumer tests across all 6 domains |
| P-2 | Risk drift rules verified via raccoon-cli | MEDIUM | Run drift-detect, add rules if missing |
| P-3 | Automated traceability verification test | HIGH | Integration test asserting full causation chain with running NATS |
| P-4 | KV projection trace metadata persistence decision | HIGH | Design decision: persist in KV, separate audit bucket, or JetStream replay |
| P-5 | Execution domain boundary definition | HIGH | What execution means in Market Foundry (order lifecycle? position tracking? venue routing?) |
| P-6 | Venue adapter architecture decision | HIGH | Does execution touch real venue APIs? Paper-only first? Simulated venue? |
| P-7 | Kill switch / circuit breaker design | HIGH | How to halt execution without full restart |
| P-8 | Execution activation ceremony design | MEDIUM | Elevated activation beyond config restart |

### Score Summary

| Dimension | Rating | Weight | Weighted |
|-----------|--------|--------|----------|
| Observation maturity | 7.5/10 | 0.08 | 0.60 |
| Evidence maturity | 8.0/10 | 0.10 | 0.80 |
| Signal maturity | 8.0/10 | 0.10 | 0.80 |
| Decision maturity | 8.5/10 | 0.10 | 0.85 |
| Strategy maturity | 8.5/10 | 0.12 | 1.02 |
| Risk maturity | 8.5/10 | 0.15 | 1.28 |
| Projection authority | 9.0/10 | 0.08 | 0.72 |
| Query surfaces | 9.0/10 | 0.04 | 0.36 |
| Governance/CLI | 8.5/10 | 0.06 | 0.51 |
| Activation model | 9.0/10 | 0.05 | 0.45 |
| Config dependencies | 9.0/10 | 0.04 | 0.36 |
| Traceability/auditability | 8.0/10 | 0.08 | 0.64 |

**Overall: 8.39/10 — CONDITIONALLY READY (elevated gate)**

The score is comparable to risk readiness (8.38), but the bar for execution is higher. A score of 8.39 against a risk-level gate would yield "ready." Against an execution-level gate (target: 9.0), it yields "conditionally ready with mandatory prerequisites."

---

## 12. Smallest Acceptable Execution Design

If and when execution is opened, the minimum viable design would be:

1. **Domain model**: `ExecutionIntent` — not an order, not a trade. An intent to act, derived from risk-approved strategy with full provenance chain.
2. **Single family**: One execution evaluator (e.g., `paper_execution`) consuming risk-approved output. Paper-only. No real venue adapter.
3. **Stream**: `EXECUTION_EVENTS` with same envelope/dedup patterns.
4. **Projection**: Latest-only KV bucket for execution intent state, single-writer projection actor.
5. **Query surface**: `GET /execution/:type/latest` — same conditional pattern.
6. **Config**: `pipeline.execution_families`, `executionDependsOnRisk` dependency map.
7. **Governance**: Execution drift rules (ED-1..ED-5), execution guardrails, raccoon-cli integration.
8. **Kill switch**: Configuration-driven execution halt that does not require restart.
9. **Audit trail**: Execution events must persist trace metadata (correlation_id, causation_id) in the materialized projection, not just logs.

**Explicitly excluded from first slice:**
- Real venue adapter (no exchange API calls)
- Order management (no order lifecycle state machine)
- Position tracking (no portfolio state)
- Fill reconciliation (no matching engine interaction)
- Multi-strategy aggregation (one risk → one execution intent)

### First Slice Question: Should It Touch a Real Venue Adapter?

**Answer: NO.**

The first execution slice must be paper-only. Rationale:
1. The domain model for execution must be proven before any venue interaction.
2. Venue adapters introduce external failure modes (rate limits, network partitions, API changes) that would obscure domain logic bugs.
3. Paper execution allows full end-to-end chain verification without financial risk.
4. The adapter pattern is proven in ingest (WebSocket → NATS); venue adapters can follow the same pattern in a subsequent stage.
5. A paper execution first slice proves the mesh flow, projection pattern, and activation model — which is the actual structural question.

---

## 13. Recommended Next Stages

Based on this review:

| Stage | Title | Type | Objective | Blocks Execution? |
|-------|-------|------|-----------|-------------------|
| S69 | Adapter Test Coverage Sweep | Hardening | Publisher/consumer adapter tests across all 6 domains | YES (P-1) |
| S70 | Risk Governance Verification | Governance | Verify risk drift rules pass, add if missing | YES (P-2) |
| S71 | Automated Traceability Verification | Hardening | Integration test asserting full causation chain | YES (P-3) |
| S72 | Trace Metadata Persistence Design | Design | Decide: KV persistence, audit bucket, or stream replay | YES (P-4) |
| S73 | Execution Domain Boundary Design | Design | Define execution semantics, intent model, lifecycle | YES (P-5) |
| S74 | Execution Governance Activation | Governance | Drift rules, guardrails, known families, raccoon-cli | Depends on S73 |
| S75 | Execution First Slice (Paper) | Implementation | Paper execution intent, end-to-end, no venue adapter | Depends on S73+S74 |
| S76 | Execution Kill Switch | Hardening | Configuration-driven execution halt mechanism | Depends on S75 |

**Critical path**: S69 → S70 → S71 → S72 → S73 → S74 → S75 → S76

**Parallelizable**: S69+S70 can run concurrently. S71+S72 can run concurrently after S69+S70. S73 can start after S71+S72 complete.
