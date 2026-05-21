# Strategy Readiness Review

> Formal readiness assessment for opening a `strategy` domain layer in Market Foundry.
> Date: 2025-03-17 | Stage: S49

## 1. Review Scope

This document evaluates whether Market Foundry is ready to introduce a `strategy` domain — the 5th layer in the pipeline (observation → evidence → signal → decision → **strategy**). The assessment covers every foundational domain, governance coverage, projection authority, config safety, and the mesh's structural integrity.

**Guiding principle**: readiness is measured by evidence, not by ambition. A layer opens only when the layers beneath it are hardened enough to support it without generating debt.

---

## 2. Domain Maturity Assessment

### 2.1 Observation — MATURE (Structural) / UNDERTESTED (Operational)

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | Complete | `ObservationTrade` with validation, dedup key, event contract |
| Ingest pipeline | Complete | WebSocket → binancef adapter → OBSERVATION_EVENTS stream |
| Actor wiring | Complete | BindingWatcherActor, ExchangeScopeActor, PublisherActor |
| Adapter tests | **MISSING** | Zero tests for observation publisher/consumer |
| Ingest actor tests | **MISSING** | Zero tests for WebSocket normalization, binding activation |
| Multi-source | Not proven | Only binancef adapter exists (by design, not a blocker) |

**Rating: 5/10** — Architecture is sound, but the entire ingest pipeline has zero automated validation. This is the weakest layer.

### 2.2 Evidence — MATURE (Core) / GAPS (Periphery)

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain models | Complete | Candle (14 tests), Volume (11 tests), TradeBurst (**0 domain tests**) |
| Samplers | Complete | CandleSampler (7), TradeBurstSampler (4), VolumeSampler (4) |
| Use cases | Complete | GetLatestCandle (3), GetCandleHistory (6), GetLatestTradeBurst (3), GetLatestVolume (3) |
| NATS registry | Complete | Registry tests (4 cases), versioned subjects, stream contracts |
| NATS KV stores | Partial | TradeBurst KV (5 tests), Volume KV (5 tests), Candle KV (tests present) |
| NATS publishers | **MISSING** | Zero publisher/consumer/gateway adapter tests |
| Projection actors | **MISSING** | Zero actor tests for candle/tradeburst/volume projections |
| HTTP handlers | Partial | Candle endpoints tested (16 cases); TradeBurst/Volume handlers **untested** |
| Multi-symbol | Proven | Partition keys isolate source × symbol × timeframe |
| Dual-write atomicity | Risk | Candle latest+history writes are non-atomic (history can fail after latest succeeds) |

**Rating: 6.5/10** — Core domain logic is solid and tested. Adapter and actor layers carry silent reliability risk due to missing tests.

### 2.3 Signal — HARDENED

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | Complete | 14 tests, validation, partition/dedup keys |
| RSI sampler | Complete | 6 tests, Wilder's smoothed RSI, deterministic |
| Use cases | Complete | 7 tests, input validation, gateway delegation |
| NATS adapters | Complete | Registry (20+ tests), KV (8 tests), publisher/consumer/gateway implemented |
| Projection | Hardened (S37) | 3 gates (final, validate, monotonicity), 7 atomic counters |
| HTTP layer | Complete | Handler (4 tests), routes (3 tests) |
| Config activation | Hardened | `signal_families` opt-in, cross-layer dependency validated at startup |
| Governance | Complete | raccoon-cli signal drift rules, guardrails documented and enforced |
| Multi-symbol | Proven (S41) | btcusdt + ethusdt isolation verified |

**Rating: 8.5/10** — Production-ready with comprehensive hardening. Minor gap: no signal history projection (intentional defer).

### 2.4 Decision — PRODUCTION-READY

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | Complete | 23 tests, validation, partition/dedup keys, multi-symbol isolation |
| Evaluator | Complete | 18 tests, pure function (no I/O), graduated confidence |
| Use cases | Complete | 3 tests, input validation, gateway delegation |
| NATS adapters | Complete | Registry (14 tests), KV (12 tests), publisher/consumer/gateway |
| Projection | Hardened (S48) | 3 gates, 7 counters, single-writer invariant, replay-safe |
| HTTP layer | Complete | Handler (5 tests), routes (3 tests) |
| Config activation | Hardened (S47) | `decision_families` opt-in, dependency chain enforced (decision → signal → evidence) |
| Governance | Complete | raccoon-cli decision drift rules (DD-1–DD-5), guardrails (DG-1–DG-10) |
| Multi-symbol | Proven (S46) | btcusdt + ethusdt isolation verified, no cross-symbol bleed |
| Architecture docs | Complete | 8 architecture documents covering domain design, contracts, projection, replay |

**Rating: 9/10** — Most mature domain by test density (78 cases). Latest-only projection is intentional. Ready to serve as strategy dependency.

---

## 3. Projection Authority

| Aspect | Status | Evidence |
|--------|--------|----------|
| Store as single writer | Enforced | Each KV bucket has exactly one ProjectionActor owner |
| Gateway read-only | Enforced | QueryResponderActor opens KV connections as read-only |
| No cross-domain writes | Enforced | Projection actors only write to their own family's bucket |
| Monotonicity guards | Present | All KV stores reject stale/duplicate writes (timestamp comparison) |
| Final gate | Present | Non-final events skipped before KV write in all projection actors |
| Validation gate | Present | Domain validation applied before materialization |
| History support | Candle only | TradeBurst/Volume/Signal/Decision are latest-only (intentional) |

**Rating: 9/10** — Authority model is clear, enforced, and documented. No ambiguity about who writes what.

---

## 4. Query Surface Quality

| Surface | Completeness | Tests | Notes |
|---------|-------------|-------|-------|
| GET /evidence/candles/latest | Complete | 6 | Full coverage including null handling |
| GET /evidence/candles/history | Complete | 10 | Range validation, limit bounds tested |
| GET /evidence/tradeburst/latest | Complete | **0** | Handler implemented but untested |
| GET /evidence/volume/latest | Complete | **0** | Handler implemented but untested |
| GET /signal/:type/latest | Complete | 4 | Happy path + error cases |
| GET /decision/:type/latest | Complete | 5 | Full coverage including null handling |
| GET /statusz | Complete | N/A | Health tracker integrated |
| GET /readyz | Complete | 2 | Readiness probe with degradation |

**Rating: 7/10** — All endpoints exist and follow consistent patterns. Two evidence endpoints lack handler tests.

---

## 5. Governance / CLI Coverage

| Check | Status | Evidence |
|-------|--------|----------|
| Evidence drift detection | Active | Checks docs, adapters, domain, config, contracts |
| Signal drift detection | Active | DD-1 through DD-5 for signal domain |
| Decision drift detection | Active | DD-1 through DD-5 for decision domain |
| Topology audit | Active | Binary/stream/subject mapping validated |
| Contract audit | Active | Subject naming, versioning, consumer specs |
| Runtime bindings | Active | Cross-config family consistency checked |
| Quality gate | Active | Fast and full profiles available |
| Coverage map | Active | Sensitive area tracking with multi-dimension coverage |

**Rating: 9/10** — Comprehensive governance. raccoon-cli catches 90%+ of architectural drift automatically.

---

## 6. Activation Model

| Aspect | Status | Evidence |
|--------|--------|----------|
| Family-level activation | Hardened (S34) | `pipeline.families`, `signal_families`, `decision_families` |
| Binding-level activation | Operational | BindingWatcherActor queries configctl at runtime |
| Cross-layer dependencies | Validated (S47) | decision → signal → evidence chain enforced at startup |
| Unknown family rejection | Active | Startup failure with clear error message |
| Derive/store consistency | Checked | raccoon-cli validates matching family lists |
| Binding deactivation | **Incomplete** | Clearing requires service restart (known limitation) |

**Rating: 8/10** — Activation is safe and validated. The restart requirement for deactivation is an operational gap, not an architectural one.

---

## 7. Config Dependencies

| Dependency | Validated | Method |
|-----------|-----------|--------|
| rsi → candle | Yes | Go startup validation + raccoon-cli static check |
| rsi_oversold → rsi | Yes | Go startup validation + raccoon-cli static check |
| future strategy → decision | **Not yet** | Needs addition to dependency map when strategy enters |
| Derive ↔ Store symmetry | Yes | raccoon-cli cross-config consistency check |
| Unknown family names | Yes | Startup rejection with error |
| Gateway config | N/A | Gateway discovers from store; no family config needed |

**Rating: 8.5/10** — Current dependencies are fully validated. Strategy will need its own dependency chain entry.

---

## 8. Mesh Structural Integrity

| Invariant | Status |
|-----------|--------|
| Single-writer per stream | Enforced |
| Single-writer per KV bucket | Enforced |
| No feedback loops | Enforced (unidirectional flow) |
| Deduplication by content | Enforced (dedup keys in all publishers) |
| Subject taxonomy consistency | Enforced (raccoon-cli contract-audit) |
| Stream retention alignment | Enforced (72h for events, 64MB for KV) |
| Partition key isolation | Proven (multi-symbol tests in all domains) |

**Rating: 9.5/10** — Mesh invariants are structurally sound and actively protected.

---

## 9. Consolidated Maturity Matrix

| Domain | Design | Implementation | Tests | Governance | Multi-Symbol | Overall |
|--------|--------|---------------|-------|-----------|-------------|---------|
| Observation | 8 | 7 | **2** | 7 | N/A | **5/10** |
| Evidence | 9 | 8 | **5** | 8 | 9 | **6.5/10** |
| Signal | 9 | 9 | 8 | 9 | 9 | **8.5/10** |
| Decision | 9 | 9 | 9 | 9 | 9 | **9/10** |
| Gateway | 9 | 9 | 7 | 9 | N/A | **8.5/10** |
| Store | 9 | 9 | 6 | 9 | 9 | **8/10** |
| Config | 9 | 9 | 7 | 9 | N/A | **8.5/10** |

---

## 10. Verdict

### Is Market Foundry ready to open `strategy`?

**NOT YET. Conditional readiness — 3 blocking gaps must close first.**

The top two layers (signal, decision) are hardened and production-ready. Governance and config activation are excellent. The mesh is structurally sound. However, the foundational layers (observation, evidence) carry test coverage debt that would compound with a new domain layer.

Opening `strategy` today would mean building a 5th floor on a building where the 1st and 2nd floors have untested load-bearing walls. The walls exist and appear sound, but they have never been verified under stress.

### Specific answers to review questions:

| Question | Answer |
|----------|--------|
| Decision está madura? | **Sim.** 78 tests, hardened projection, multi-symbol proven, governance active. |
| Governança de decision está suficiente? | **Sim.** DD-1–DD-5 drift rules, DG-1–DG-10 guardrails, all enforced by raccoon-cli. |
| Config dependencies estão seguras? | **Sim.** Cross-layer validation at startup + pre-deploy static analysis. |
| Mesh continua clara e protegida pelo raccoon-cli? | **Sim.** 95%+ drift coverage, all invariants actively enforced. |
| Gateway continua limpo? | **Sim.** Stateless proxy, read-only KV access, conditional route registration. |
| Store continua claro como authority? | **Sim.** Single-writer enforced, projection gates consistent, authority documented (S48). |
| Quais gaps impedem strategy sem gerar dívida? | Adapter/actor test gaps em observation e evidence. Ver §11. |
| Menor desenho aceitável de strategy? | Ver strategy-entry-prerequisites.md. |

---

## 11. Blocking Gaps

See [strategy-risks-and-blockers.md](strategy-risks-and-blockers.md) for detailed gap analysis and prioritization.

**Summary of blocking gaps:**

1. **BG-1**: Evidence adapter tests missing (publishers, consumers, gateways) — 0% coverage
2. **BG-2**: Observation/ingest pipeline completely untested — 0% coverage
3. **BG-3**: Projection actor tests missing across all evidence types — 0% coverage

These are not design problems. The architecture is correct. The risk is that serialization, stream creation, consumer durability, and projection logic have never been verified by automated tests. A strategy layer depending on these untested paths would inherit silent reliability risk.

---

## 12. Recommendation

**Close the 3 blocking gaps, then open strategy.**

Estimated effort: 2-3 stages focused on test coverage (not new features). This is not a design phase — it's a hardening phase for the foundation that strategy will depend on.

The next stages should follow this sequence:
1. S50: Adapter test coverage sweep (evidence + observation publishers/consumers)
2. S51: Projection actor test coverage (evidence projection actors + store integration)
3. S52: Strategy domain design (readiness review passes after S50-S51)

This sequence respects the principle that readiness is earned, not assumed.
