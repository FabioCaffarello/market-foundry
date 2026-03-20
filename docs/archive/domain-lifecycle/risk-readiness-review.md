# Risk Readiness Review

> Formal readiness assessment for opening a `risk` domain layer in Market Foundry.
> Date: 2026-03-18 | Stage: S59

## 1. Review Scope

This document evaluates whether Market Foundry has sufficient maturity, governance, and architectural health to open a `risk` domain in the next development cycle. The review examines every layer from observation through strategy, plus cross-cutting concerns (projection authority, query surfaces, config dependencies, CLI governance, activation model).

**Guiding principle**: Market Foundry progresses by readiness, not by domain anxiety. A `risk` layer will only be recommended if the foundation beneath it is demonstrably solid.

---

## 2. Domain Maturity Assessment

### 2.1 Observation

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | `ObservationTrade` with 8 fields, decimal strings, full validation, dedup keys |
| Application logic | STRONG | Binding parser with normalization and validation |
| Adapters | STRONG | NATS registry (stream `OBSERVATION_EVENTS`, 6h retention), publisher, consumer |
| Actors | STRONG | Ingest scope: websocket, exchange scope, publisher, binding watcher |
| Tests | MODERATE | Domain model tested (6 cases), binding tested; **no publisher/consumer adapter tests, no ingest actor tests** |
| HTTP surface | N/A | Observation is internal — no HTTP exposure by design |

**Rating: 7.5/10** — Production-grade pipeline but adapter and actor test coverage is a gap.

### 2.2 Evidence

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Three types (candle, trade burst, volume), each with full validation, decimal precision, events |
| Application logic | STRONG | Pure samplers (candle, trade burst, volume) with window-based accumulation, Big.Float precision |
| Client layer | STRONG | 4 use cases (latest candle, candle history, latest trade burst, latest volume) |
| Adapters | STRONG | Registry (21+ tests), gateway, publisher, consumer; KV stores for candle/tradeburst/volume |
| Actors | STRONG | Derive: sampler actors per type; Store: projection actors with three-gate pattern (final → validate → monotonicity) |
| Tests | MODERATE | Domain: 22+ tests; samplers: 8+ tests; registry: 21+ tests; KV stores: tested; **no publisher/consumer adapter tests, no derive actor tests** |
| HTTP surface | STRONG | 4 endpoints (candle latest/history, tradeburst latest, volume latest) with tests |

**Rating: 8/10** — Three independently validated evidence types with comprehensive coverage. Adapter test gap is consistent across domains.

### 2.3 Signal

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Signal struct with 7 fields, decimal value, metadata map, partition/dedup keys; 9 tests |
| Application logic | STRONG | RSI sampler (Wilder's smoothed, period=14), warm-up phase, streaming calculation |
| Client layer | STRONG | `GetLatestSignalUseCase` with validation |
| Adapters | STRONG | Registry with type routing (`LatestSpecByType`), gateway, KV store with monotonicity guard |
| Actors | STRONG | Derive: signal sampler + publisher; Store: projection (final gate, validation, stats) + query responder |
| Tests | MODERATE | Domain: 9 tests; RSI: tested; KV store: guard tests; projection: tested; **no publisher/consumer adapter tests, no derive actor tests** |
| HTTP surface | STRONG | `GET /signal/:type/latest` with type routing |

**Rating: 8/10** — Complete signal implementation, extensible by type. Same adapter test gap.

### 2.4 Decision

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | 10 fields with outcome enum (triggered/not_triggered/insufficient), confidence decimal, signal provenance; 14 tests |
| Application logic | STRONG | RSI oversold evaluator (threshold=30.0), pure logic, no I/O; tested |
| Client layer | STRONG | `GetLatestDecisionUseCase` with validation; 5 tests |
| Adapters | STRONG | Registry, gateway, KV store with monotonicity; guard tests |
| Actors | STRONG | Derive: evaluator + publisher; Store: projection + query responder |
| Tests | MODERATE | Domain: 14 tests; evaluator: tested; KV: guard tests; projection: tested; **no publisher/consumer adapter tests, no derive actor tests** |
| HTTP surface | STRONG | `GET /decision/:type/latest` with type routing |

**Rating: 8.5/10** — Most mature downstream domain. Enum safety adds correctness guarantees.

### 2.5 Strategy

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | STRONG | Strategy with direction enum (long/short/flat), DecisionInput provenance, confidence, parameters/metadata; 14 tests |
| Application logic | STRONG | MeanReversionEntryResolver (pure, no I/O), 8 tests; `GetLatestStrategyUseCase`, 5 tests |
| Adapters | STRONG | Registry (8 tests), publisher, consumer, gateway, KV store (8 tests) with monotonicity |
| Actors | STRONG | Derive: resolver + publisher; Store: projection (21 tests, three-gate, multi-symbol verified) + consumer |
| Tests | STRONG (projection) / MODERATE (adapters) | Projection: 21 tests including multi-symbol isolation; **no publisher/consumer adapter tests, no derive actor tests** |
| HTTP surface | STRONG | `GET /strategy/:type/latest`, handler (10 tests), routes (3 tests) |
| Documentation | EXCEPTIONAL | Domain design (2400+ lines), readiness review, activation/ownership, first-slice contracts |
| Config integration | COMPLETE | `pipeline.strategy_families` in derive and store configs; dependency chain validated |

**Rating: 8.5/10** — Production-ready implementation with the best documentation of any domain. Same adapter test gap.

---

## 3. Projection Authority

| Aspect | Status | Evidence |
|--------|--------|----------|
| Single-writer invariant | ENFORCED | Each KV bucket has exactly one writer actor; documented in stream-ownership-matrix.md |
| Store as sole authority | ENFORCED | Store binary owns all projection pipelines; gateway reads via NATS request/reply |
| Gateway statelessness | ENFORCED | Gateway has no pipeline config, discovers routes from deps at startup |
| Monotonicity guards | ENFORCED | All KV stores (signal, decision, strategy) reject stale/duplicate writes via timestamp comparison |
| Three-gate projection | ENFORCED | Final gate → validation gate → monotonicity guard in all projection actors |
| Replay idempotency | ENFORCED | Dedup keys include type + coordinates + timestamp; JetStream MsgID set on publish |

**Rating: 9/10** — Projection authority model is sound and consistently applied.

---

## 4. Query Surface Quality

| Surface | Endpoint | Tests | Conditional | Notes |
|---------|----------|-------|-------------|-------|
| Evidence candle latest | GET /evidence/candles/latest | YES | YES | Full handler + route tests |
| Evidence candle history | GET /evidence/candles/history | YES | YES | Range query with limit/since/until |
| Evidence tradeburst latest | GET /evidence/tradeburst/latest | YES | YES | |
| Evidence volume latest | GET /evidence/volume/latest | YES | YES | |
| Signal latest | GET /signal/:type/latest | YES | YES | Type-routed |
| Decision latest | GET /decision/:type/latest | YES | YES | Type-routed |
| Strategy latest | GET /strategy/:type/latest | YES | YES | Type-routed, 10 handler tests |
| Healthz | GET /healthz | YES | NO | Always registered |
| Readyz | GET /readyz | YES | NO | Readiness checks |

**Rating: 9/10** — All surfaces are conditional, tested, and follow consistent patterns.

---

## 5. Governance and CLI Coverage

| Check | Scope | Status |
|-------|-------|--------|
| Architecture layer boundaries | 11 rules | ENFORCED |
| Stream registry consistency | All domains | ENFORCED |
| Config ↔ Compose alignment | All services | ENFORCED |
| Naming identity | No residual "server" refs | ENFORCED |
| Signal drift rules (SD-1..SD-5) | Docs, adapters, domain, config, contracts | ENFORCED |
| Decision drift rules (DD-1..DD-5) | Same scope as signal | ENFORCED |
| Strategy drift rules (STD-1..STD-5) | Same scope as signal/decision | ENFORCED |
| Config symmetry (derive ↔ store) | Pipeline family arrays | ENFORCED |
| Quality gate profiles | fast, ci, deep | ENFORCED |

**Gaps in governance:**

| Gap | Severity | Impact on risk |
|-----|----------|---------------|
| Cannot verify resolver/evaluator purity (no I/O) | LOW | Risk resolvers could silently introduce I/O; mitigated by code review |
| Cannot verify KV bucket configuration (retention, max size) | LOW | Bucket misconfiguration possible; mitigated by infra review |
| Cannot enforce activation dependency chain at CLI level | MEDIUM | Strategy can be enabled without decision in theory; mitigated by config validation at startup |
| No risk-specific drift rules exist yet | EXPECTED | Must be created as part of risk domain design |

**Rating: 8.5/10** — Governance is mature and mechanically extensible.

---

## 6. Activation Model

| Layer | Mechanism | Status |
|-------|-----------|--------|
| Family activation (structural) | `pipeline.*_families` config keys | ENFORCED — requires restart |
| Binding activation (runtime) | BindingWatcherActor + configctl events | ENFORCED — dynamic per symbol |
| Dependency validation | `ValidatePipeline()` checks transitive deps | ENFORCED — blocks binary startup on invalid chain |
| Known family registry | Closed set per layer, rejects unknown names | ENFORCED — typo protection |

**Dependency chain (validated at startup):**
```
observation → evidence (candle) → signal (rsi) → decision (rsi_oversold) → strategy (mean_reversion_entry)
```

A future `risk` domain would extend this chain:
```
... → strategy (mean_reversion_entry) → risk (???)
```

**Rating: 9/10** — Two-layer activation with transitive dependency validation is production-grade.

---

## 7. Config Dependency Safety

| Rule | Status | Evidence |
|------|--------|----------|
| Signal depends on evidence | ENFORCED | `signalDependsOnEvidence["rsi"] = ["candle"]` |
| Decision depends on signal | ENFORCED | `decisionDependsOnSignal["rsi_oversold"] = ["rsi"]` |
| Strategy depends on decision | ENFORCED | `strategyDependsOnDecision["mean_reversion_entry"] = ["rsi_oversold"]` |
| Unknown family rejection | ENFORCED | `knownStrategyFamilies` closed set |
| Derive ↔ Store symmetry | ENFORCED by CLI | Drift-detect checks pipeline arrays match |

**For risk**: A new `riskDependsOnStrategy` map must be added, plus `knownRiskFamilies` closed set. The pattern is mechanical and proven.

**Rating: 9/10** — Config dependency model is the strongest cross-cutting safeguard.

---

## 8. Mesh Integrity

| Stream | Owner | Consumers | Status |
|--------|-------|-----------|--------|
| OBSERVATION_EVENTS | ingest | derive | HEALTHY |
| EVIDENCE_EVENTS | derive | store | HEALTHY |
| SIGNAL_EVENTS | derive | store | HEALTHY |
| DECISION_EVENTS | derive | store | HEALTHY |
| STRATEGY_EVENTS | derive | store | HEALTHY |

**Single-writer verified**: Each stream has exactly one publisher binary. Store consumes and materializes. Gateway reads from store KV buckets via request/reply.

A `RISK_EVENTS` stream would follow the identical pattern.

**Rating: 9/10** — Mesh is clean, protected by raccoon-cli, and mechanically extensible.

---

## 9. Cross-Cutting Test Gap Analysis

The most significant gap across all domains is the **absence of adapter-level and actor-level unit tests** for publishers, consumers, and derive actors:

| Component | Test Status | Affected Domains |
|-----------|-------------|------------------|
| NATS publisher adapters | NO TESTS | observation, evidence, signal, decision, strategy |
| NATS consumer adapters | NO TESTS | observation, evidence, signal, decision, strategy |
| Ingest actor scope | NO TESTS | observation |
| Derive actor scope | NO TESTS | evidence, signal, decision, strategy |

**What IS tested:**
- NATS registries (contract tests)
- NATS KV stores (guard tests + some functional)
- Store projection actors (functional tests, up to 21 cases for strategy)
- HTTP handlers and routes
- Domain models and application logic

**Impact assessment**: The untested components are infrastructure plumbing that follows repetitive patterns. The risk is moderate — bugs would manifest as message loss or encoding errors, detectable in integration but not caught pre-deploy. This gap exists uniformly across ALL domains and is **not specific to strategy**.

---

## 10. Readiness Verdict

### Is Market Foundry ready to open a `risk` layer?

**Verdict: CONDITIONALLY READY — with prerequisites**

The architecture, governance, and domain implementations are mature enough to support a new layer. However, three conditions must be met before `risk` design begins:

1. **Strategy multi-symbol proof must be verified in a live environment** (smoke tests exist but runtime proof is needed)
2. **Adapter test debt must be acknowledged and planned** (not blocking, but the gap grows with each new domain)
3. **Strategy governance drift rules must be verified as passing** in the current codebase via `raccoon-cli drift-detect`

### Score Summary

| Dimension | Rating | Weight | Weighted |
|-----------|--------|--------|----------|
| Observation maturity | 7.5/10 | 0.10 | 0.75 |
| Evidence maturity | 8.0/10 | 0.15 | 1.20 |
| Signal maturity | 8.0/10 | 0.15 | 1.20 |
| Decision maturity | 8.5/10 | 0.15 | 1.28 |
| Strategy maturity | 8.5/10 | 0.15 | 1.28 |
| Projection authority | 9.0/10 | 0.08 | 0.72 |
| Query surfaces | 9.0/10 | 0.05 | 0.45 |
| Governance/CLI | 8.5/10 | 0.07 | 0.60 |
| Activation model | 9.0/10 | 0.05 | 0.45 |
| Config dependencies | 9.0/10 | 0.05 | 0.45 |

**Overall: 8.38/10 — CONDITIONALLY READY**

---

## 11. Smallest Acceptable Risk Design

If and when `risk` is opened, the minimum viable design would be:

1. **Domain model**: `Risk` struct with type, source, symbol, timeframe, strategy reference, risk score (decimal), risk level enum (low/medium/high/critical), metadata, final, timestamp
2. **Single family**: One risk evaluator (e.g., `position_risk` or `exposure_risk`) consuming strategy output
3. **Stream**: `RISK_EVENTS` with same retention/dedup patterns
4. **Projection**: Latest-only KV bucket, single-writer projection actor
5. **Query surface**: `GET /risk/:type/latest`
6. **Config**: `pipeline.risk_families`, `riskDependsOnStrategy` dependency map
7. **Governance**: Risk drift rules (RD-1..RD-5), risk guardrails, raccoon-cli integration
8. **No**: No portfolio aggregation, no position tracking, no execution, no external risk feeds

This mirrors exactly how signal, decision, and strategy were introduced — one family, one type, latest-only, full governance.

---

## 12. Recommended Next Stages

Based on this review:

| Stage | Title | Type | Objective |
|-------|-------|------|-----------|
| S60 | Adapter Test Coverage Sweep (Round 2) | Hardening | Add publisher/consumer adapter tests across all 5 domains |
| S61 | Derive Actor Test Coverage | Hardening | Add unit tests for derive scope actors (sampler, evaluator, resolver chains) |
| S62 | Risk Domain Design | Design | Produce risk-domain-design.md following strategy-domain-design.md pattern |
| S63 | Risk Governance Activation | Governance | Add risk drift rules, guardrails, known families to raccoon-cli |
| S64 | Risk First Slice | Implementation | Implement single risk family end-to-end |
| S65 | Risk Projection Hardening | Hardening | Three-gate projection, multi-symbol verification |
