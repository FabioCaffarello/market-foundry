# Next Wave Recommendations Timeline

> Consolidated from 20 source documents spanning S86 through S210. Each gate's recommendations are annotated with their outcome: **EXECUTED**, **DEFERRED**, or **SUPERSEDED**.
>
> Original files archived to `docs/archive/next-wave/`.

---

## Pre-Pipeline Era

### S86 — Next Frontier Entry Prerequisites

**Context:** Post-paper-integrated execution (S74-S85). Pipeline: derive, execute, store, gateway operational in paper mode.

| Recommendation | Status |
|---|---|
| Operational Hardening Gate (Docker Compose, NATS integration tests, observability, fill reconciliation, CI) | **EXECUTED** — addressed across S87-S105 |
| Real Venue Design Gate (transitional bridge, async fill model, credential infra) | **DEFERRED** — venue expansion remains future work |
| Activation Gate Ceremony (17 formal gates) | **DEFERRED** — not yet triggered |

---

### S86 — Next Phase Readiness (Post-Sanitization)

**Context:** Quality-service identity fully removed, market-foundry identity established.

| Recommendation | Status |
|---|---|
| Absorb MarketMonkey patterns | **DEFERRED** — repeatedly deferred in favor of proving Foundry's own patterns first |
| Implement new domains (observation, evidence) | **EXECUTED** — implemented through vertical slice and capability waves |
| Evolve raccoon-cli for market-foundry | **SUPERSEDED** — CLI evolved into architecture guardian role |
| Evolve infrastructure as needed | **EXECUTED** — NATS, ClickHouse, migrations infrastructure all added |

---

## Structural Consolidation Era (S96-S105)

### S99 — Next Technical Wave Recommendations

**Context:** Post-structural consolidation (S96-S99). Foundation patterns established.

| Recommendation | Status |
|---|---|
| **Priority 1:** Vertical slice completion (candle-to-paper-order) | **EXECUTED** — completed in S107-S111 |
| **Priority 2:** Operational confidence layer (structured logging, diagnostics) | **EXECUTED** — observability surfaces built in S102-S103, expanded through Wave A |
| **Priority 3:** MarketMonkey absorption | **DEFERRED** — consistently deferred |
| Event schema formalization | **DEFERRED** — single producer per event type, still not triggered |
| OpenTelemetry / distributed tracing | **DEFERRED** — structured logging remains sufficient |

### S105 — Next Platform Wave Recommendations

**Context:** Post-hardening wave (S101-S105). Platform structurally complete but never run end-to-end.

| Recommendation | Status |
|---|---|
| **Priority 1:** Vertical slice execution | **EXECUTED** — S107-S111 |
| **Priority 2:** Operational confidence (correlation IDs, composition root tests) | **PARTIALLY EXECUTED** — diagnostics built; correlation IDs deferred multiple times |
| **Priority 3:** MarketMonkey absorption | **DEFERRED** |
| Additional infrastructure hardening | **DEFERRED** — correctly, per recommendation |
| Raccoon-CLI new analyzers | **DEFERRED** |
| Documentation consolidation | **EXECUTED** — now underway in refactoring phase (S210+) |

---

## Vertical Slice and Live Baseline Era (S107-S117)

### S111 — After Vertical Slice 01

**Context:** First end-to-end pipeline proven structurally. Zero domain logic bugs. Architecture operationally unproven.

| Recommendation | Status |
|---|---|
| **Phase 1:** Close operational gap (live pipeline run, execute actor tests) | **EXECUTED** — live baseline established in S113-S117 |
| **Phase 2:** Controlled product evolution | **SUPERSEDED** — capability waves (CC-01, CC-02) chosen instead of product evolution |
| Multi-symbol live monitoring | **EXECUTED** — CC-01 proved multi-symbol scaling |
| Strategy performance dashboard | **DEFERRED** |
| ClickHouse integration | **EXECUTED** — implemented in S143-S149 |
| MarketMonkey absorption | **DEFERRED** |
| Full E2E test automation | **PARTIALLY EXECUTED** — smoke tests and CI established |

### S117 — After Live Baseline

**Context:** Full event chain runs with real market data. 7 runtimes operational. Endurance/resilience unproven.

| Recommendation | Status |
|---|---|
| **Primary:** New capability on the proven mesh (multi-symbol first) | **EXECUTED** — CC-01 (multi-symbol) delivered |
| Inject correlation IDs into slog (~15 files) | **DEFERRED** — repeatedly deferred |
| Document cold-start behavior | **PARTIALLY EXECUTED** — documented in consolidation wave |
| Soak test infrastructure | **DEFERRED** |
| ClickHouse write path | **EXECUTED** — writer service built in S148 |
| MarketMonkey absorption | **DEFERRED** |

---

## Capability Wave Era (CC-01, CC-02, TC-01)

### S124 — After Capability 01 (CC-01)

**Context:** CC-01 validated horizontal scaling (multi-symbol, config-driven). Code extensibility unproven.

| Recommendation | Status |
|---|---|
| **CC-02:** New signal family (EMA Crossover recommended) | **EXECUTED** — EMA Crossover implemented as CC-02 |
| Address CF-03 (correlation ID), CF-08 (client boilerplate) during CC-02 | **PARTIALLY EXECUTED** — CF-08 addressed; CF-03 deferred again |
| Do not add more symbols (N>2) | **FOLLOWED** |
| Do not absorb MarketMonkey yet | **FOLLOWED** |

### S130 — After CC-02

**Context:** CC-01 proved scaling, CC-02 proved extensibility. Architecture no longer needs structural proof.

| Recommendation | Status |
|---|---|
| **Primary:** Product wave (multi-timeframe correlation, backtesting, venue, alerting) | **SUPERSEDED** — TC-01 (timeframe coverage) chosen instead |
| Optional hardening prep (CF-08 + CF-11 + CF-03 bundled refactors) | **DEFERRED** — N=3 trigger not reached |
| CC-03 (third signal family) only if product-justified | **DEFERRED** — not product-justified |
| Do not over-abstract from two examples | **FOLLOWED** |

### S136 — After Timeframe Coverage 01 (TC-01)

**Context:** 4 timeframes proven (60s, 180s, 300s, 900s). Temporal scaling validated. State persistence (D1) identified as hard gate for longer timeframes.

| Recommendation | Status |
|---|---|
| **Primary:** Product-oriented wave | **SUPERSEDED** — consolidation wave chosen instead |
| **Secondary:** One new signal family (if capacity) | **DEFERRED** |
| TC-02 (more timeframes) — wait for D1 (state persistence) | **DEFERRED** — D1 still unresolved |
| State persistence (WAL/snapshots) | **DEFERRED** — listed as post-refactoring candidate |

---

## Consolidation and Analytical Era (S137-S162)

### S142 — After Current Capability Consolidation

**Context:** Consolidation wave closed gap between "works" and "defined, observable, governable." ClickHouse preparation documented but unimplemented.

| Recommendation | Status |
|---|---|
| **Primary:** Phased ClickHouse/Migrations preparation | **EXECUTED** — S143-S149 delivered migrations, schema, writer, query surface |
| Phase 1: Migration infrastructure (`cmd/migrate`) | **EXECUTED** — S146 |
| Phase 2: Core schema design (6 tables) | **EXECUTED** — S147 |
| Phase 3: Writer service | **EXECUTED** — S148 |
| Phase 4: Query surface extension | **EXECUTED** — S149 |
| Phase 5: Cold-start bootstrap | **DEFERRED** — high complexity, optionality boundary risk |
| New pipeline families | **DEFERRED** — correctly per recommendation |

### S150 — After Analytical Runtime Entry

**Context:** Analytical skeleton viable (migrations, writer, reader, gateway). Lacks operational confidence (tests, failure handling, observability).

| Recommendation | Status |
|---|---|
| **Wave A:** Analytical runtime hardening (tests, failure handling, observability) | **EXECUTED** — S151-S156 |
| **Wave B:** Controlled schema/query expansion | **EXECUTED** — S163+ (Wave B family expansion) |
| **Wave C:** Cold-start bootstrap | **DEFERRED** — high complexity |
| Writer mapper/inserter/supervisor tests | **EXECUTED** |
| Pipeline recovery with backoff | **EXECUTED** |
| Buffer overflow and mapper error visibility | **EXECUTED** |

### S156 — After Analytical Wave A

**Context:** Wave A hardening complete. Three preconditions identified for Wave B entry.

| Recommendation | Status |
|---|---|
| P1: Reader path minimum instrumentation | **EXECUTED** — S158 |
| P2: One end-to-end integration test | **EXECUTED** — S159 |
| P3: Writer config validation at startup | **EXECUTED** — S161 |
| Wave B: One new writer pipeline family + one new query endpoint | **EXECUTED** — signals family added first |
| Backoff jitter | **EXECUTED** |
| Dead-letter queue, push alerting, auto-recovery | **DEFERRED** |
| Materialized views, schema versioning | **DEFERRED** |

---

## Wave B Family Expansion Era (S162-S190)

### S162 — After Pre-Wave-B Gate

**Context:** All three preconditions satisfied. Analytical layer cleared for controlled expansion.

| Recommendation | Status |
|---|---|
| One new read-path adapter (signals suggested) | **EXECUTED** — signals family (S163-S165) |
| Extend smoke-analytical-e2e.sh | **EXECUTED** |
| Wire smoke-analytical into CI | **EXECUTED** |
| Backoff jitter + configurable query timeout | **EXECUTED** |
| Do not add all 5 remaining families simultaneously | **FOLLOWED** |

### S167 — After Wave B Iteration 01

**Context:** First expansion (signals) validated pattern. Second family recommendation issued.

| Recommendation | Status |
|---|---|
| Second family: Decisions (RSI Oversold) | **EXECUTED** — S168-S170 |
| Mandatory hardening at Family 3 (struct DI, smoke parameterization, helper renaming) | **EXECUTED** — S172 |
| Codegen evaluation at Family 4 | **EXECUTED** — evaluation triggered, scoped at S192 |
| Do not add Prometheus/OpenTelemetry | **FOLLOWED** |
| Do not add dead-letter queue | **FOLLOWED** |

### S173 — After Post-Hardening Wave B Gate

**Context:** Mandatory hardening tranche (H-1, H-2, H-3) complete. Pattern cheaper to execute.

| Recommendation | Status |
|---|---|
| Authorize Family 03 (Strategies) | **EXECUTED** — S174-S177 |
| D-4 codegen evaluation at Family 03 gate | **EXECUTED** |
| PF-5 CI smoke assessment | **EXECUTED** |
| Additional hardening before F-03 | **NOT CHOSEN** — correctly per evidence |

### S179 — After Family 03 Wave B Gate

**Context:** Three families covered (candles, signals, decisions, strategies). Pattern proven 3 times.

| Recommendation | Status |
|---|---|
| Authorize Family 04 (Risk Assessments) | **EXECUTED** — S180-S182 |
| Codegen mandatory before Family 06 | **EXECUTED** — codegen scoped at S192-S195 |
| >2 frictions in F-04 triggers mandatory hardening | **NOT TRIGGERED** — pattern remained healthy |
| Do not introduce cross-family queries | **FOLLOWED** |

### S183 — After Family 05 / Pre-Family-06 Gate (S190)

**Context:** 5 families complete. Full vertical coverage. Within-layer variants next.

| Recommendation | Status |
|---|---|
| Family 06 conditional authorization | **EXECUTED** — trigger assessment at S191, codegen tranche path chosen |
| Scope codegen tranche formally | **EXECUTED** — S192-S195 |
| Defer codegen implementation to post-F06 | **SUPERSEDED** — codegen path chosen over another manual family |
| Do not pre-authorize Family 07 | **FOLLOWED** |

---

## Codegen Era (S192-S204)

### S198 — After Pre-Generated Family Gate

**Context:** Codegen engine built (S195). A1+A2 artifacts generated from YAML specs. Conditional pass for first generated family.

| Recommendation | Status |
|---|---|
| Wave 1: First generated family (A1+A2 only, single family) | **EXECUTED** — EMA family generated (S200-S203) |
| Wave 2: Findings gate | **EXECUTED** — S204 |
| Wave 3: Second generated family (different layer) | **EXECUTED** — mean_reversion_entry (S203 scope) |
| Wave 4: Mapper generation feasibility | **DEFERRED** — requires stable cross-layer evidence |
| Wave 5: File integration evaluation (marker sections) | **DEFERRED** |
| Do not batch-generate multiple families | **FOLLOWED** |
| Do not modify templates during first family | **FOLLOWED** |

### S204 — After Post-Generated Family Gate

**Context:** First codegen-first family (EMA) confirmed mechanical correctness. Conditional pass.

| Recommendation | Status |
|---|---|
| Second same-layer codegen-first family for repeatability | **EXECUTED** — S203 delivered second generated family |
| Repeatability hardening gate | **EXECUTED** — S204 gate |
| Cross-layer validation | **DEFERRED** — requires its own gate |
| Live activation proof (D-1) | **DEFERRED** |
| Mapper generation feasibility | **DEFERRED** |
| Do not retroactively convert manual families | **FOLLOWED** — 6 originals remain permanent golden references |

---

## Stabilization and Refactoring Era (S205-S210+)

### S210 — After Pre-Refactor Stabilization Gate

**Context:** Stabilization scope frozen. Must-finish matrix defined. 440 architecture docs identified for consolidation.

| Recommendation | Status |
|---|---|
| **Step 1:** Push and verify CI (MF-2) | **IN PROGRESS** |
| **Step 2:** Tag repository (`stabilization-exit-s210`) | **IN PROGRESS** |
| **Step 3-4:** Documentation cleanup (~440 to 120-150 docs) | **IN PROGRESS** |
| **Step 5:** Code debt cleanup (TD-02, AD-01, TD-03) | **PENDING** |
| **Step 6:** Verification and exit gate | **PENDING** |
| No new analytical families during refactoring | **ACTIVE FREEZE** |
| No codegen template modifications | **ACTIVE FREEZE** |
| No new services or domain entities | **ACTIVE FREEZE** |

**Post-refactoring candidates (not yet authorized):**
1. Production readiness assessment
2. Codegen expansion evaluation
3. TC-02 planning (state persistence, WAL, cold-start bootstrap)
4. Venue expansion (real exchange connectivity)

---

## Recurring Themes Across All Gates

### Consistently Deferred Items

| Item | Times Deferred | Current Status |
|---|---|---|
| MarketMonkey absorption | 7+ | Still deferred; no trigger has fired |
| Correlation ID middleware | 5+ | Still deferred; structured logging sufficient |
| OpenTelemetry / distributed tracing | 5+ | Still deferred; log-based debugging sufficient |
| Event schema formalization | 4+ | Still deferred; single producer per event type |
| State persistence / WAL | 3+ | Deferred; blocks TC-02 |
| Cold-start bootstrap | 3+ | Deferred; high complexity, optionality risk |
| Soak test infrastructure | 3+ | Deferred; no operational need yet |
| Real venue implementation | 2+ | Deferred; requires design gates |

### Governance Principles That Held

1. **Evidence before expansion** — every gate used evidence, not momentum, to decide next steps.
2. **One family at a time** — Wave B never batch-expanded; each family earned individual authorization.
3. **Hardening at triggers, not on schedule** — hardening tranches executed only when committed obligations or measurable friction demanded them.
4. **Debts resolve at natural triggers** — deferred items were reconsidered when their triggers fired, not before.
5. **Product pressure is the best test** — repeatedly recommended but ultimately superseded by analytical infrastructure needs.
