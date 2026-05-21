# Deferred Work Registry

> Consolidated from 14 source documents spanning S111 through S189.
> Date consolidated: 2026-03-20

---

## STILL OPEN -- Actionable Items

### Critical / Blocking

| ID | Description | Category | Origin | Trigger Condition | Current State |
|----|-------------|----------|--------|-------------------|---------------|
| TRIG-1 | Handler file at hard ceiling (615/620 lines) -- extract `parseAnalyticalParams()` | Structural | S188 (Family 05) | Family 06 would exceed 620 lines | 615 lines; extraction required before Family 06 |
| TRIG-2 | Codegen tranche scope definition -- mandatory decision | Structural | S167/S178/S189 | Family 06 boundary (deferred from F06 to dedicated stage) | Evaluated, justified, deferred to post-S189 dedicated stage |
| TRIG-3 | Reader 10-parameter positional signature approaching limit | Structural | S188 (Family 05) | Family 07 without codegen | 10 params; 11 tolerable, 12+ not |

### Deferred with Committed Triggers

| ID | Description | Category | Origin | Trigger Condition | Estimated Effort |
|----|-------------|----------|--------|-------------------|------------------|
| DEF-C1 | Codegen implementation (readers, handlers, use cases, tests) | Structural | S167/S178 | Family 07 boundary (shifted from F06 by handler extraction) | 2-3 days |
| DEF-C2 | Schema coherence compile-time verification (DDL-to-struct alignment) | Testing | S178 | ~12 analytical tables or 100+ DDL columns (currently 6 tables, ~95 columns) | 1-2 days |
| DEF-C3 | Handler file split by domain | Structural | S178 | Handler exceeds ~700 lines post-extraction (~Family 13) | 0.5-1 day; likely absorbed by codegen |
| DEF-C4 | Friction count gate (>2 new frictions per family) | Operational | S167 | Evaluated every family; Family 05 at threshold (3 frictions, same root cause) | N/A -- gate condition |

### Deferred without Committed Triggers -- Structural

| ID | Description | Severity | Origin | Revisit Condition |
|----|-------------|----------|--------|-------------------|
| D-01/TC | Per-binding timeframes (global list works at TC-01) | Low | S134 (TC-01) | TC-02 requires heterogeneous TF sets per symbol |
| D-02/TC | Per-timeframe tracker split (aggregate-only visibility) | Low | S134 (TC-01) | TC-02 adds 8+ timeframes, or per-TF stall incident |
| D-06/TC | Window state persistence / WAL (highest-impact TC-01 debt) | **High** | S134 (TC-01) | TC-02 commits to 4h+ timeframes -- **hard gate** |
| CF-08a | Actor boilerplate -- generic `SignalSamplerActor[T]` | Low | S128 (CC-02) | N=3 signal families (~2h at trigger) |
| CF-08b | Client UseCase boilerplate migration (6 domain packages) | Low | S122 (Cap-01) | New domain family addition (~1h) |
| CF-11 | NATS registry switch proliferation (4 touch points per family) | Low | S128 (CC-02) | N=3 signal families (~1-2h) |
| CF-12 | Store pipeline boilerplate (~25 lines per family) | Low | S128 (CC-02) | N=5 signal families |
| CF-13 | Per-family algorithm configuration (hardcoded periods) | Low | S128 (CC-02) | A/B testing or per-binding tuning need |
| D2/VS | Publisher actor generic extraction (5 near-identical actors) | Low | S110 (VS-01) | 6th publisher actor or cross-cutting timeout change |
| D3/VS | Query client usecase generics (unique inline validation) | Low | S110 (VS-01) | Query client exceeds 5 use case files |
| D6/VS | Route registration abstraction | Low | S110 (VS-01) | Family count exceeds 12 |
| D7/VS | Gateway wiring DRY | Low | S110 (VS-01) | Gateway count exceeds 12 |
| D7/LP | Config parameterization (NATS URLs in 6+ config files) | Low | S115 (LP) | Second deployment environment (staging/production) |
| D3/LP | Use-case pattern unification (two coexisting patterns) | Low | S115 (LP) | New domain where developer confused about which pattern |
| CF-02 | Active symbols list endpoint | Low | S122 (Cap-01) | When touching configctl routes, or N>5 symbols |
| DEF-U1 | Filter case-sensitivity (PF-3) -- no normalization or enum validation | Low | S172 | Users report confusion or case-variant values introduced |
| DEF-U2 | No pagination beyond limit=500 (D-9) | Low | S172 | Consumers need full historical scans |
| DEF-U5 | Silent mapper fallbacks (zero/empty on parse error) | Low | S172 | Analytical consumers depend on field completeness |
| DEF-U8 | Consumer/inserter naming in writer (generic names) | Low | S172 | Writer handles non-analytical events |
| DEF-U9 | Metadata validation (no schema check at write time) | Low | S172 | Metadata quality issues affect consumers |
| D-04/TC | "List available timeframes" endpoint | Low | S134 (TC-01) | External consumers query without config access |
| D-05/TC | Null response disambiguation (null = not configured vs warming up vs expired) | Low | S134 (TC-01) | Non-expert consumers exposed, or operational incident |
| D-07/TC | Gateway aggregate view / pipeline overview endpoint | Low | S134 (TC-01) | Symbol count >5 or dashboard integration |

### Deferred without Committed Triggers -- Operational / Reliability

| ID | Description | Severity | Origin | Revisit Condition |
|----|-------------|----------|--------|-------------------|
| DEF-U3 | NATS consumer lag visibility (no metrics for writer lag) | Medium | S172 | Write volume increases (tick-level data) |
| DEF-U4 | Sticky degradation without auto-recovery (readers stay degraded after CH restart) | Medium | S172 | ClickHouse instability increases or ops requests auto-recovery |
| DEF-U6 | Backoff jitter (writer uses fixed backoff, no jitter) | Low | S172 | Writer scales to multiple instances |
| D-03/TC | Per-timeframe idle detection (uniform threshold masks high-TF stalls) | Low | S134 (TC-01) | TC-02 adds 4h+ timeframes |
| D4/LP | Long-running stability / soak test | Medium | S115 (LP) | Multi-symbol or live trading |
| D5/LP | Failure recovery validation | Medium | S115 (LP) | Before production deployment |
| D6/LP | Soak testing infrastructure | Medium | S115 (LP) | N>5 symbols or 24h continuous operation |
| D4/CC | Composition root unit tests | Low | S124 (CC) | Wiring error reaches live validation |

### Deferred without Committed Triggers -- Testing

| ID | Description | Severity | Origin | Revisit Condition |
|----|-------------|----------|--------|-------------------|
| D1/VS | Execute actor unit tests (safety-critical logic, zero tests) | Medium | S110 (VS-01) | Standalone testing initiative; extract gate logic first |
| D4/VS | Ingest actor tests (611 LOC supervisor, no tests) | Low | S110 (VS-01) | Address alongside D1/VS |
| D5/VS | Configctl actor tests (612 LOC, no routing tests) | Low | S110 (VS-01) | Address after D1/VS |
| D5/LP | Domain-specific golden-file tests (math correctness) | Low | S115 (LP) | Adding second signal family or strategy type |
| DEF-U7 | Smoke JSON content verification (PF-6) | Low | S172 | JSON parsing bugs escape unit tests |

### Deferred without Committed Triggers -- Documentation / Tooling

| ID | Description | Severity | Origin | Revisit Condition |
|----|-------------|----------|--------|-------------------|
| D1/LP | Raccoon-CLI AST parsing (heuristic parser fragility) | Low | S115 (LP) | Second false positive from heuristic parser |
| D2/LP | Stale naming cleanup in documentation (~40 old references) | Low | S115 (LP) | New doc references old names |
| D6/LP-s | Script hardening (Python one-liners, hardcoded URLs) | Low | S115 (LP) | CI/CD pipeline needs robust automation |

### Permanently Accepted Trade-offs (No Action Planned)

| ID | Description | Rationale |
|----|-------------|-----------|
| D8/VS | Derive-configctl dependency model | Correct eventual-consistency behavior by design |
| CF-06 | No sustained automated watchdog | Manual monitoring sufficient at N=2 symbols |
| CF-07 | Global kill switch (not per-symbol) | Paper-only; halting both is safe |
| CF-09 | RSI warm-up delay (~15 min) | Mathematical requirement (14 candles) |
| CF-10 | 300s timeframe wait | Inherent to timeframe design |
| F-03/TC | Integer-only timeframe representation | Unambiguous, machine-friendly |
| F-06/TC | Log verbosity scaling | Linear by design; slog supports filtering |
| F-09/TC | HTTP test file duplication | Independently executable; documentation value |
| F-10/TC | Actor count growth | Linear by design; Hollywood handles thousands |
| F-11/TC | KV key cardinality | NATS KV handles millions |
| F-12/TC | NATS subject cardinality | NATS handles millions |
| F-14/TC | Signal warmup latency at high TFs | Physics constraint |
| F-16/TC | Smoke test wait time assumptions | Three-tier validation correctly separates concerns |
| F-18/TC | Configctl has no timeframe concept | Correct separation of concerns |

---

## Resolved Items

| ID | Description | Resolution | Resolved At |
|----|-------------|-----------|-------------|
| R1/VS | Signal publisher missing correlation_id | Added to error log | S111 |
| R2/VS | Projection actor stats normalization (5 actors) | Added received counter + checkStatsInvariant | S111 |
| R3/VS | Raccoon-CLI dead code cleanup (26 warnings) | Zero warnings after cleanup | S111 |
| R4/VS | Generic UseCase factory for configctlclient (10 files) | `CommandUseCase[Cmd,Reply]` + `GatewayUseCase[In,Out]` | S111 |
| R1/LP | Drift-detect false positive suppression (~260 warnings) | Split scan by directory | S116 |
| R2/LP | Stale variable name in runtime test | `validatorRecord` -> `projectionRecord` | S116 |
| R3/LP | AGENTS.md prohibited patterns clarification | Clarified old binary names | S116 |
| R4/LP | Raccoon-CLI test fixture modernization | Updated to market-foundry/ingest | S116 |
| R1/Cap | Per-symbol tracker counters (CF-01) | `Counter("key:"+symbol)` across 13 actors | S123 |
| R2/Cap | Automated error-level log scanning (CF-04) | `grep -c` in live-pipeline-activate.sh | S123 |
| R3/Cap | Automated memory usage snapshot (CF-05) | `docker stats --no-stream` in activation script | S123 |
| R1/CC2 | HTTP correlation ID middleware (CF-03 partial) | `webserver.CorrelationID` middleware, removed 12 manual extractions | S129 |
| R-01/TC | Timeframe config validation (`ValidateTimeframes`) | Rejects dupes, <10s, >86400s | S135 |
| R-02/TC | Post-crash recovery expectations per timeframe | Documentation in validation procedure | S135 |
| D-1 | `parseEvidenceKeyParams` naming | Renamed to `parseAnalyticalKeyParams` | S172 (H-3) |
| D-2 | Struct-based DI for analytical handlers | `AnalyticalHandlerDeps` / `AnalyticalFamilyDeps` | S172 (H-1) |
| D-3 | Smoke test extraction | `validate_analytical_family()` helper | S172 (H-2) |
| PF-4 | No CI for analytical smoke test | `smoke-analytical` job in ci.yml | S166/S172 |
| D-4 | Codegen evaluation | Evaluated at S178; justified, deferred to Family 06+ | S178 |
| CT-ceiling | Family 04 ceiling test | Passed at S182 | S182 |
| F05-coverage | Full vertical coverage L1-L6 | Confirmed at S187 | S187 |

---

## Lineage -- Source Document Map

Each source document and the items it introduced or carried forward:

| Source Document | Stage | Items Introduced | Items Carried Forward |
|----------------|-------|------------------|-----------------------|
| `refactors-deferred-after-vertical-slice-01.md` | S111 | D1-D8 (VS-01 series) | -- |
| `evidence-driven-refactors-after-vertical-slice-01.md` | S111 | R1-R4 (VS-01 executed refactors) | -- |
| `bounded-pain-refactors-after-live-pipeline.md` | S116 | R1-R4 (LP executed refactors) | -- |
| `refactors-deferred-after-live-pipeline.md` | S116 | D1-D7 (LP series) | -- |
| `evidence-driven-surgical-refactors-after-capability-01.md` | S123 | R1-R3 + D1 design (Cap-01) | -- |
| `refactors-deferred-after-capability-01.md` | S123 | CF-02, CF-08, CF-03 impl | CF-06/07/09/10 accepted |
| `cc-02-triggered-vs-deferred-refactor-matrix.md` | S128 | CF-11, CF-12, CF-13 (new) | CF-03, CF-08, CF-02, D4-D6 from Cap-01 |
| `triggered-refactors-after-cc-02.md` | S129 | R1/CC2 (HTTP middleware) | -- |
| `refactors-still-deferred-after-cc-02.md` | S129 | -- | CF-03 actor, CF-08, CF-11, CF-12, CF-02, CF-13, D4-D6 |
| `triggered-refactors-after-timeframe-coverage-01.md` | S135 | R-01/TC, R-02/TC | -- |
| `refactors-still-deferred-after-timeframe-coverage-01.md` | S135 | D-01 through D-07 (TC series) | F-03/06/09/10/11/12/14/16/18 accepted |
| `triggered-vs-deferred-items-before-family-04.md` | S178 | TRIG-1/2, DEF-C1-C4, DEF-U1-U9 | Resolved: D-1 through D-4, PF-4 |
| `triggered-vs-deferred-items-before-family-05.md` | S183 | TRIG-2 (handler size) | All DEF-C and DEF-U items from S178 |
| `triggered-vs-deferred-hardening-items-after-family-05.md` | S189 | TRIG-1 (handler ceiling critical), PF-7 | All prior deferred items; codegen trigger shifted to F07 |

---

## Debt Trajectory

| Checkpoint | Active Items | High Severity | Blocking |
|------------|-------------|---------------|----------|
| Post-VS-01 (S111) | 8 | 1 (D1 execute tests) | 0 |
| Post-LP (S116) | 15 | 1 | 0 |
| Post-Cap-01 (S123) | 12 | 0 | 0 |
| Post-CC-02 (S129) | 9 | 0 | 0 |
| Post-TC-01 (S135) | 7 deferred + accepted | 1 (D-06 state persistence) | 0 |
| Pre-hardening (S166) | 14 | 1 (CI) | 0 |
| Post-hardening (S172) | 11 | 0 | 0 |
| Pre-Family 04 (S178) | 15 | 0 | 0 |
| Pre-Family 05 (S183) | 16 | 0 | 0 |
| **Post-Family 05 (S189)** | **16** | **1 (handler ceiling)** | **1** |

---

## Key Convergence Points

1. **N=3 signal families (CC-03)**: CF-08a (generic actor), CF-11 (NATS registry), CF-03 actor-layer correlation ID -- estimated ~5-7 hours total.
2. **Family 06 gate**: TRIG-1 handler extraction (mandatory blocker), TRIG-2 codegen scope decision.
3. **Family 07 boundary**: DEF-C1 codegen implementation becomes mandatory (without it, reader params and handler size hit hard limits).
4. **TC-02 gate**: D-06/TC window state persistence is a hard gate; D-02/TC + D-03/TC per-TF diagnostics recommended.
5. **Pre-production gate**: D5/LP failure recovery validation, D6/LP soak testing, D4/LP stability validation.
