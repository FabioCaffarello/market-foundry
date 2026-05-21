# Gains, Trade-offs, and Open Debts -- Consolidated Timeline

> Single-source-of-truth ledger tracking what each project phase gained, traded, and deferred.
> Consolidates 18 phase-specific documents into one chronological reference.
> Date: 2026-03-20

---

## Still-Open Debts Summary

The following debts remain unresolved as of the stabilization gate (S210):

| ID | Debt | Origin | Current Priority |
|----|------|--------|-----------------|
| SOD-01 | Reader 10-parameter positional signature | Wave B F-05 | P1 |
| SOD-02 | Test hardcoded family counts | Wave B | P2 |
| SOD-03 | NATS consumer lag visibility | Analytical Runtime | P2 |
| SOD-04 | Schema coherence compile-time check | Wave B F-01 | P2 |
| SOD-05 | Automated baseline validation | Current Capability Consolidation | P2 |
| SOD-06 | Module graph evaluation | Stabilization | P1 |
| SOD-07 | Superseded docs not marked | Stabilization | P1 |
| SOD-08 | Per-family doc boilerplate | Stabilization | P1 |
| SOD-09 | Stage report index | Stabilization | P1 |
| SOD-10 | TC-02 scope (state persistence, WAL, cold-start) | TC-01 | Deferred |
| SOD-11 | Load testing baseline | Live Baseline | Deferred |
| SOD-12 | 4 deferred writer families | Analytical Runtime | Deferred |
| SOD-13 | Codegen Tier 2 (reader generation) | Codegen Readiness | Deferred |
| SOD-14 | clickhouse-go version alignment | Stabilization | Deferred |
| SOD-15 | Dead-letter queue / backpressure | Analytical Wave A | Deferred |
| SOD-16 | No CI integration for analytical smoke (ClickHouse in CI) | Wave B F-01 | Deferred |
| SOD-17 | Sticky degradation without auto-recovery | Analytical Wave A | Deferred |
| SOD-18 | Silent mapper fallbacks | Analytical Wave A | Low |
| SOD-19 | No pagination beyond 500 rows | Wave B F-01 | Low |
| SOD-20 | Filter case-sensitivity / no validation | Wave B F-02 | Low |
| SOD-21 | Codegen mapper generation (A3) | Codegen Readiness | Deferred |
| SOD-22 | Codegen automated fragment insertion | Generated Path | Deferred |
| SOD-23 | Live event flow proof for generated families | Generated Path | High |
| SOD-24 | Documentation entropy (440 files) | Stabilization | P1 |

---

## Phase 1: Structural Consolidation (S96-S99)

### Gains
- **Composition root legibility**: Gateway run.go reduced from 231 to 40 lines with 3 visible phases. Store supervisor dropped from 508 to ~280 lines.
- **Single-entry family addition**: One catalog entry per family; eliminates "forgot to update the second list" bugs.
- **Generic factory elimination**: `newGatewayConn[T]()`, `filterEnabled[T]()`, `familyNames[T]()` replaced ~250 lines of duplicated code.
- **Semantic precision**: Precise terminology map enforced (Scope = actor boundary, Gateway = NATS request/reply port).
- **Automated architecture enforcement**: raccoon-cli enforces 11 structural rules via AST inspection.
- **Documented growth patterns**: Step-by-step playbooks for domains, families, runtimes, and adapters.

### Trade-offs
- Documentation volume: 9 architecture documents and 4 stage reports for <15 active families.
- Abstraction ceiling risk in catalog-driven patterns at larger scale.
- Guardian tooling (raccoon-cli) maintenance cost (~550KB Rust codebase).
- Composition root rigidity (6-phase lifecycle is clean but rigid).

### Open Debts
| Debt | Status |
|------|--------|
| D1: Composition root integration tests | RESOLVED -- proven by live pipeline run (S115) and subsequent phases |
| D2: Observability gaps (no tracing, no metrics) | PARTIALLY RESOLVED -- writer/reader instrumented in Wave A; distributed tracing still deferred |
| D3: Error propagation consistency | SUPERSEDED -- error handling patterns documented and accepted as appropriate per-context |
| D4: Configuration validation completeness | RESOLVED -- S104 hardened config validation, KnownFamilies API exported |
| D5: Venue adapter expansion path | STILL OPEN -- low priority, document when next venue added |

---

## Phase 2: Platform Hardening (S101-S105)

### Gains
- **Operational contract visibility**: 10 invariants and 7 shared behavior rules documented.
- **Diagnostic surface**: Every log line carries `runtime=<name>`. `/diagz` endpoint provides combined readiness + tracker summary.
- **Error tracking accuracy**: All 17 error paths tracked via `tracker.RecordError()`.
- **Config validation hardening**: Duplicate families rejected, binding topics format-validated, family catalog exported as queryable API.
- **Governance precision**: Decision gates for all expansion types, 10 anti-patterns cataloged, ARCH_DOCS expanded to 27 entries.

### Trade-offs
- Documentation volume continued (18 total architecture documents after two waves).
- Three-level governance abstraction layer added.
- drift-detect ARCH_DOCS list grew from 8 to 27 entries.
- Actor error tracking fixes (17 RecordError additions) have no automated test coverage.

### Open Debts
| Debt | Status |
|------|--------|
| PD-1: Composition root integration tests | RESOLVED -- proven by live pipeline run (S115) |
| PD-2: Cross-registration coherence | RESOLVED -- codegen and CI gates now enforce family registration consistency |
| PD-3: Correlation ID propagation | RESOLVED -- S129 delivered HTTP correlation ID middleware |
| PD-4: Error classification taxonomy | SUPERSEDED -- not justified without alerting infrastructure |
| PD-5: Raccoon-CLI governance constant maintenance | SUPERSEDED -- ongoing maintenance, included in expansion playbook |

---

## Phase 3: Vertical Slice 01 (S107-S112)

### Gains
- **Architectural proof**: All 8 families, 25+ HTTP endpoints, full actor/adapter/infrastructure stack exercised end-to-end.
- **Bug discovery**: 7 infrastructure/wiring bugs found and fixed. Zero domain logic bugs.
- **Evidence-based refactoring**: 4 targeted refactors (correlation_id parity, projection stats normalization, dead code cleanup, generic UseCase factory).
- **Friction catalog**: Prioritized friction catalog (F01-F13) with clear severity levels.
- **Governance validation**: raccoon-cli's 950 tests validated across full codebase.

### Trade-offs
- Structural proof only, not operational proof (no live NATS/market data).
- Only 1 binding exercised (`binancef.btcusdt.60`).
- Only 4 of 13 findings applied; 9 deliberately deferred.
- Generic UseCase applied only to configctlclient; other 6 clients remain manual.

### Open Debts
| Debt | Status |
|------|--------|
| D1: Execute actor unit tests | RESOLVED -- S113 delivered SafetyGate with 22 tests |
| D2: Publisher actor generic extraction | SUPERSEDED -- pattern stable, no 6th publisher added |
| D3: Query client generics | SUPERSEDED -- pattern accepted at current scale |
| D4: Ingest actor unit tests | STILL OPEN -- low priority, no ingest behavior modification |
| D5: Configctl actor unit tests | STILL OPEN -- low priority, no configctl logic modification |
| PD-1: Composition root integration tests | RESOLVED -- proven by live run |
| PD-2: Cross-registration coherence | RESOLVED -- codegen CI gates |
| D6-D8: Route/gateway/derive boilerplate | SUPERSEDED -- acceptable at current family count |

---

## Phase 4: Live Baseline (S113-S117)

### Gains
- **Operational proof of architecture**: End-to-end with real NATS, real Binance WebSocket data, 7 services.
- **Execution safety hardening**: SafetyGate extracted, 22 new tests covering all gate combinations.
- **Quality gate signal-to-noise**: drift-detect warnings reduced from ~265 to 5 (all legitimate).
- **Explicit operational baseline**: Single-document baseline with 10 named invariants and 11-step runbook.
- **Bug discovery under real conditions**: 3 bugs found that all prior testing missed.

### Trade-offs
- 5 stages of elapsed time without feature delivery.
- Documentation volume: ~50 architecture documents and ~22 stage reports total.
- Minimal scope of operational proof (single-symbol, paper-venue, short-duration).
- Deferred item tracking fragmented across ~12 distinct items in multiple documents.

### Open Debts
| Debt | Status |
|------|--------|
| LD-1: Cross-runtime observability (correlation IDs in slog) | PARTIALLY RESOLVED -- HTTP middleware delivered; actor-level slog injection still open |
| LD-2: Endurance / soak testing | STILL OPEN -- SOD-11 |
| LD-3: Failure recovery paths | STILL OPEN -- low priority, no unplanned failure encountered |
| LD-4: Cold-start behavior | STILL OPEN -- SOD-10, deferred to TC-02 |
| LD-5: Deferred item tracking fragmentation | RESOLVED -- S209 debt registry is now single source of truth |

---

## Phase 5: Capability 01 -- Multi-Symbol (S119-S124)

### Gains
- **Config-driven horizontal scaling validated**: Adding a new symbol requires zero code changes. ethusdt added with 3 files changed (~62 lines).
- **Cross-symbol isolation validated**: Composite key design provides correct per-symbol data isolation.
- **Per-symbol diagnostic visibility**: `/statusz` shows per-symbol counter breakdowns (14 actors instrumented).
- **Automated operational checks**: Error log scanning and docker stats snapshot in activation script.
- **Zero domain logic bugs** under multi-symbol operation.

### Trade-offs
- Global kill switch (halts all symbols simultaneously).
- RSI warm-up delay (~15 minutes at 60s timeframe) -- inherent to RSI.
- 300s timeframe wait -- inherent to design.
- Manual sustained monitoring (no automated watchdog).
- Correlation ID design-only (deferred implementation).

### Open Debts
| Debt | Status |
|------|--------|
| D1: Correlation ID middleware implementation | RESOLVED -- S129 delivered HTTP middleware |
| D2: Active symbols endpoint | STILL OPEN -- low priority, workaround exists |
| D3: Client UseCase boilerplate | SUPERSEDED -- accepted at current scale |
| D4: Composition root smoke tests | RESOLVED -- proven by live runs |
| D5: Failure recovery validation | STILL OPEN -- deferred to pre-production |
| D6: Soak testing infrastructure | STILL OPEN -- SOD-11 |

---

## Phase 6: CC-02 -- Signal Family Extension (S125-S129)

### Gains
- **Proven code extensibility**: EMA crossover implemented following Playbook 1 exactly (3 new files, 7 modified, ~414 lines).
- **Domain model generality confirmed**: `signal.Signal` with string Value and map metadata handled numeric and categorical signal types.
- **Infrastructure actor reuse validated**: Publisher, projection, consumer, query actors all reused without code changes.
- **Coexistence isolation proven**: RSI code paths received zero modifications.
- **Playbook reproducibility confirmed**: Predicted vs actual file counts matched exactly.
- **HTTP correlation ID middleware delivered**: Extracted from 7 handler files.
- **Extensibility cost model established**: Measurable cost per family category.

### Trade-offs
- Actor boilerplate at N=2 (~95% identical actors, no generic yet).
- NATS registry switch dispatch (4 touch points per family).
- Hardcoded signal parameters (EMA periods 9/21, RSI period 14).
- Signal-domain-only proof (cross-domain extensibility untested).

### Open Debts
| Debt | Status |
|------|--------|
| CF-08: Generic SignalSamplerActor factory | SUPERSEDED -- not triggered at N=3, codegen path chosen instead |
| CF-11: Map-based NATS registry | SUPERSEDED -- codegen path chosen |
| CF-03: Correlation ID in actor middleware | STILL OPEN -- HTTP layer resolved, actor layer deferred |
| CF-12: Store pipeline boilerplate reduction | SUPERSEDED -- acceptable at current scale |
| CF-02: Active symbols endpoint | STILL OPEN -- low priority |
| CF-13: Per-family algorithm configuration | STILL OPEN -- deferred to A/B testing requirement |
| D4: Composition root unit tests | RESOLVED -- proven by live runs |
| D5: Failure recovery validation | STILL OPEN -- deferred to pre-production |
| D6: Soak testing infrastructure | STILL OPEN -- SOD-11 |

---

## Phase 7: TC-01 -- Timeframe Coverage (S131-S136)

### Gains
- **Config-driven timeframe scaling**: 2 to 4 timeframes with zero Go code changes.
- **Linear resource growth confirmed**: 2x actors, 2x subjects, <30% write load increase.
- **Six anticipated problems did not materialize**: NATS pressure, fan-out latency, KV contention, dedup collision, cross-TF interference, memory accumulation -- none occurred.
- **Config validation gate**: `ValidateTimeframes()` rejects invalid configs at startup.
- **Crash recovery runbook**: Documented data loss expectations per timeframe.

### Trade-offs
- Global timeframe list (all symbols share same TFs).
- No interim candle snapshots.
- In-memory-only window state (crash loses accumulated data).
- Aggregate-only tracking (no per-TF counters).
- Two success criteria deferred (M7, M8 for 900s and 3600s RSI convergence).

### Open Debts
| Debt | Status |
|------|--------|
| D1: Window state persistence (WAL) | STILL OPEN -- SOD-10, hard gate for TC-02 |
| D2: Per-TF idle detection | STILL OPEN -- becomes relevant at 4h+ windows |
| D3: RSI convergence at long windows | STILL OPEN -- validation debt only, no code changes needed |
| D4: Per-binding timeframe config | STILL OPEN -- not needed at uniform config |
| D5: Query surface observability | RESOLVED -- S160 delivered three-layer read path instrumentation |

---

## Phase 8: Current Capability Consolidation (S137-S142)

### Gains
- **Canonical baseline definition**: 30 pass/fail success criteria across 5 validation tiers.
- **Operational phase classification**: Automatic phase computation (starting/warming/active/idle/stalled) on every service.
- **Diagnostic surface**: 4 diagnostic endpoints per service plus `diag-check.sh`.
- **Recovery semantics clarity**: Explicit survival/loss matrix for restarts.
- **Accepted limitations registry**: 5 limitations formally documented with rationale and triggers.
- **Shared script infrastructure**: `scripts/utils/lib.sh` eliminates copy-paste drift.
- **Self-documenting configuration**: CONFIG-REFERENCE.md with types, defaults, constraints.
- **ClickHouse entry governance**: 7 entry principles defined before any ClickHouse code written.
- **Persistence trigger decision matrix**: 5 pain points with explicit trigger thresholds.

### Trade-offs
- Documentation volume: 14+ architecture documents, 6 stage reports.
- Validation remains manual (30 success criteria not automated).
- No code changes to core pipeline.
- ClickHouse preparation is documentation-only.

### Open Debts
| Debt | Status |
|------|--------|
| OD-01: No state persistence for in-memory samplers | STILL OPEN -- SOD-10, hard gate for TC-02 |
| OD-02: No migration tooling | RESOLVED -- cmd/migrate implemented in S146 |
| OD-03: Gateway lacks tracker integration | RESOLVED -- analytical reader instrumented in S160 |
| OD-04: No automated baseline validation | STILL OPEN -- SOD-05 |
| OD-05: No query observability | RESOLVED -- S160 delivered three-layer instrumentation |
| OD-06: Per-timeframe idle detection | STILL OPEN -- low priority |
| OD-07: RSI convergence formal proof | STILL OPEN -- validation only |
| OD-08: Per-binding timeframe customization | STILL OPEN -- not needed |
| OD-09: Gateway aggregate view | STILL OPEN -- not needed at current cardinality |
| OD-10: Null response disambiguation | STILL OPEN -- no external consumers |

---

## Phase 9: Analytical Runtime Entry (S143-S150)

### Gains
- **ClickHouse structurally optional**: Gateway builds and runs without ClickHouse. Writer is a separate binary.
- **Migration infrastructure**: `cmd/migrate` with forward-only model, checksum validation, catalog conventions.
- **Canonical schema**: 6 tables covering all pipeline families with uniform metadata, partitioning, and TTL.
- **Write path exists**: NATS to writer to ClickHouse flow works.
- **Read path exists**: ClickHouse to gateway to HTTP flow works.
- **Operational/analytical boundary codified**: 10 optionality rules, 5 boundary rules, 8 invariants.
- **Route prefix separation**: `/evidence/*` = operational, `/analytical/*` = historical.
- **Independent failure domains**: ClickHouse down does not affect operational pipeline.

### Trade-offs
- Float64 over Decimal128 (simpler mappers, less precision).
- No deduplication (simpler MergeTree engine).
- JSON strings over native types for nested fields.
- Forward-only migrations (no automated rollback).
- Single query endpoint (candles only).
- 90-day TTL (bounded storage, no recovery of expired rows).
- Writer and reader tests skipped (implicit trade-off).
- Silent mapper fallbacks (parse errors become zero values).

### Open Debts
| Debt | Status |
|------|--------|
| Writer unit tests | RESOLVED -- S152 delivered 25 mapper + 10 inserter + 8 reader tests |
| Reader unit tests | RESOLVED -- S152 delivered reader query builder tests |
| Writer pipeline recovery | RESOLVED -- S154 delivered per-family restart with backoff |
| INSERT retry | RESOLVED -- S153 delivered exponential backoff with configurable retries |
| Buffer overflow metrics | RESOLVED -- S153 delivered overflow counters |
| Mapper error visibility | RESOLVED -- S153 delivered WARN logging with context |
| Cold-start bootstrap | STILL OPEN -- SOD-10 |
| Event schema versioning | STILL OPEN -- deferred, acceptable at single-developer scale |
| Cross-table correlation queries | STILL OPEN -- deferred, requires multi-table query builder |

---

## Phase 10: Analytical Wave A -- Hardening (S151-S156)

### Gains
- **Test foundation**: 25 mapper tests, 10 inserter tests, 8 reader tests with explicit coverage boundaries.
- **Failure handling alignment**: Buffer retention during retry (critical data-loss bug fixed), exponential backoff, overflow counters.
- **Pipeline recovery**: Per-family restart with degraded state. Unaffected families continue while failed families restart.
- **Observability**: Buffer depth gauge, flush counter, flush duration, events received counter, degraded trackers array, diagnostic script, runbook.
- **Scope freeze discipline**: Zero new tables/endpoints/families/infrastructure added.

### Trade-offs
- Unit tests over integration tests (no NATS-to-ClickHouse E2E).
- Sticky degradation over auto-recovery.
- Pull-only observability (no Prometheus/Grafana).
- Mapper fallbacks over rejection (invalid data becomes zero values).
- Actor-blocking retry over async retry.
- No consumer-inserter backpressure.

### Open Debts
| Debt | Status |
|------|--------|
| Reader path zero instrumentation | RESOLVED -- S160 delivered three-layer read path instrumentation |
| No integration test (NATS to HTTP) | RESOLVED -- S159 delivered 7-phase automated E2E validation |
| Writer config validation absent | RESOLVED -- S161 delivered fail-fast startup validation |
| Backoff has no jitter | STILL OPEN -- SOD-17 related, trivial fix |
| Consumer/supervisor message handling untested | STILL OPEN -- low priority |
| ClickHouse client timeout not configurable | STILL OPEN -- low priority |
| No NATS consumer lag visibility | STILL OPEN -- SOD-03 |

---

## Phase 11: Pre-Wave-B Hardening (S157-S161)

### Gains
- **Responsibility clarity**: Complete responsibility map for all 6 analytical components.
- **Boundary hardening**: Reader extracted to adapter layer, compile-time interface assertions, writer config validation.
- **Integration proof**: 7-phase automated E2E validation (NATS to ClickHouse to HTTP) via `make smoke-analytical`.
- **Read path observability**: Three-layer instrumentation with Server-Timing headers.
- **Startup robustness**: Fail-fast 3-phase startup (config validation to connections to run).

### Trade-offs
- Script-based integration proof (bash), not Go integration tests.
- Structured logging, not metrics (no Prometheus).
- Reviewer-enforced schema coherence, not compile-time.
- Sticky degradation, not auto-recovery.
- Candle-only read path (5 of 6 tables are write-only).
- Per-family counters, not per-request metrics.

### Open Debts
| Debt | Status |
|------|--------|
| D1: CI integration of smoke-analytical | STILL OPEN -- SOD-16 |
| D2: No backoff jitter | STILL OPEN -- trivial fix deferred |
| D3: No NATS consumer lag visibility | STILL OPEN -- SOD-03 |
| D4: No connection pool monitoring | STILL OPEN -- low priority |
| D5: No load testing | STILL OPEN -- SOD-11 |
| D6: Non-candle families untested E2E | RESOLVED -- Wave B expanded all 6 families with smoke coverage |
| D7: No chaos testing | STILL OPEN -- deferred to post-Wave-B |
| D8: Gateway readiness excludes analytical | BY DESIGN -- intentional (R-02) |

---

## Phase 12: Wave B Iteration 01 -- Signal Family (S163-S167)

### Gains
- **9-artifact expansion unit validated**: Schema to writer to reader to gateway to tests to smoke to gate -- functioning, no shortcuts.
- **Schema coherence testable without ClickHouse**: Unit tests assert row length and column count at compile time.
- **Write path was future-proof**: Writer already consumed RSI signals. Entire expansion was read-path only.
- **Observability parity automatic**: Per-pipeline health tracking and Server-Timing headers for every family.
- **CI gates merges**: GitHub Actions runs unit tests and smoke-analytical on every push/PR.
- **Optionality invariant preserved**: Analytical layer remains entirely optional.

### Trade-offs
- Mechanical duplication accepted (~80% identical code across families).
- Manual schema coherence verification (review-enforced, not compile-time).
- Monolithic smoke test (linear growth per family).
- Sticky degradation accepted.
- Silent mapper fallbacks accepted.
- No backoff jitter accepted.

### Open Debts
| Debt | Status |
|------|--------|
| D-1: parseEvidenceKeyParams naming | RESOLVED -- S172 renamed to parseQueryKeyParams |
| D-2: Handler constructor argument accumulation | RESOLVED -- S172 switched to struct DI |
| D-3: Smoke test linear growth | RESOLVED -- S172 extracted validate_analytical_family() |
| D-4: Codegen evaluation | RESOLVED -- S178 evaluated, threshold set at Family 06 |
| D-5 to D-12: Various operational debts | See SOD list above; most STILL OPEN at low severity |

---

## Phase 13: Wave B Family 02 + Hardening (S168-S173)

### Gains
- **Struct DI eliminates constructor churn**: `AnalyticalHandlerDeps` struct -- adding Family N+1 requires one field, zero signature changes.
- **Smoke extraction**: `validate_analytical_family()` reduces family addition from ~80 lines to ~7 lines (91% reduction).
- **Helper naming corrected**: `parseQueryKeyParams` replaces `parseEvidenceKeyParams`.
- **9-artifact pattern proven across two distinct data shapes**: 12 columns vs 15 columns, different JSON structures.
- **Formal process governs expansion**: Pattern v2 with hardening thresholds, checklist, 5-point gate review.
- **Optionality invariant holds after 3 tables**.

### Trade-offs
- Manual duplication accepted over premature codegen (at 3 families).
- Review-enforced schema coherence (at 9 tables).
- Sticky degradation accepted.
- No CI smoke integration (requires Docker-in-Docker).
- Silent mapper fallbacks accepted.

### Open Debts
| Debt | Status |
|------|--------|
| D-4: Codegen at Family 4 | RESOLVED -- evaluated in S178, threshold set at Family 06 |
| PF-4: Outcome filter case-sensitive | STILL OPEN -- SOD-20 |
| PF-5: No CI integration for analytical smoke | STILL OPEN -- SOD-16 |
| D-5 to D-12: Various operational debts | Carried forward, see SOD list |

---

## Phase 14: Wave B Family 03 -- Strategies (S174-S179)

### Gains
- **Four-layer analytical coverage**: Candles, Signals, Decisions, Strategies.
- **Three JSON columns proven**: Validates higher-column families are structurally feasible.
- **Direction filter as mechanical addition**: Domain-specific optional filters are a proven mechanical step.
- **Struct DI validated under real expansion pressure**: Purely additive, zero signature changes.
- **Write path immutability at four families**: Zero modifications across 4 expansions.
- **D-4 codegen trigger evaluated and resolved**: Justified at Family 06, explicitly non-blocking before then.

### Trade-offs
- Mechanical duplication tolerated (~800 lines across 4 readers/handlers/use cases).
- Review-enforced schema coherence (at 4 tables, manageable).
- No filter validation (invalid values return empty results).
- No pagination beyond 500 rows.
- Handler file as monolith (417 lines, split recommended at ~600).

### Open Debts
| Debt | Status |
|------|--------|
| DEF-C1: Codegen implementation | RESOLVED -- codegen engine built and validated in S192-S196 |
| DEF-C2: Schema coherence compile-time check | STILL OPEN -- SOD-04 |
| DEF-C3: Handler file split | RESOLVED -- S206 extracted parseAnalyticalParams(), file at 502 lines |
| DEF-U1 to DEF-U9: Various uncommitted debts | Mostly STILL OPEN at low severity, see SOD list |

---

## Phase 15: Wave B Family 05 -- Executions (post-S183-S190)

### Gains
- **Full vertical analytical coverage**: L1 (Evidence) through L6 (Executions). All pipeline layers have read-path endpoints.
- **Zero creative decisions across 5 family expansions**: Pattern is mechanically reproducible and codegen-ready.
- **Zero write-path changes across 6 expansions**.
- **Linear cost growth**: ~780 LOC per family at ~45 minutes effort.
- **JSON complexity solved**: 1, 2, 3, and 4 JSON columns with struct/slice/map/mixed targets.
- **New type classes absorbed**: Float64 and Boolean without pattern modification.
- **Dual optional filters compose cleanly**.

### Trade-offs
- Manual artisanship at scale (tolerable at 5 families, waste at 10+).
- Handler monolith (501 lines after H-5 extraction).
- Positional reader parameters (10 args, error-prone at 12+).
- No CI integration for analytical smoke.
- No pagination beyond 500 rows.
- Test duplication (~430 LOC per family).

### Open Debts
| Debt | Status |
|------|--------|
| Codegen tranche scoping | RESOLVED -- S192-S196 delivered codegen engine |
| Reader query-object pattern | STILL OPEN -- SOD-01 |
| Generic JSON parser parseJSON[T] | STILL OPEN -- deferred to codegen scope |
| CI analytical smoke test | STILL OPEN -- SOD-16 |
| Schema coherence tooling | STILL OPEN -- SOD-04 |

---

## Phase 16: Codegen Readiness (S192-S198)

### Gains
- **Proven structural equivalence**: 12/12 golden comparisons pass across all 6 existing families.
- **Deterministic naming convention enforcement**: 10 derived fields per spec, all correct.
- **CI gate prevents regression**: `codegen-golden` and `codegen-test` CI jobs on every push/PR.
- **Specification-driven expansion model**: 14-field YAML spec instead of copy-paste.
- **~15 min saved per family** (~23% reduction from ~65 min manual baseline).

### Trade-offs
- Fragment generation, not file generation (manual insertion ~5 min per family).
- 2 of 6 Tier 1 artifacts covered (A1 consumer spec, A2 pipeline entry).
- No automated scope guard (review discipline only).
- Golden snapshot maintenance grows linearly.
- Normalization pipeline calibrated against 6 families only.

### Open Debts
| Debt | Status |
|------|--------|
| D1: Mapper generation (A3) | STILL OPEN -- SOD-21, requires domain.columns spec extension |
| D2: File integration with marker sections | STILL OPEN -- SOD-22 |
| D3: CI drift detection job | STILL OPEN -- deferred to after D2 |
| D4: Cross-spec uniqueness validation in CI | RESOLVED -- S203 delivered validate-all with cross-family uniqueness |
| D5-D7: Config/smoke/Tier 2 generation | STILL OPEN -- low priority, deferred |

---

## Phase 17: Generated Path (S199-S204)

### Gains
- **Naming correctness enforcement**: 7 families x 10 derived fields = 70 derivations, all correct.
- **Cross-spec uniqueness validation**: `codegen validate-all` enforces global uniqueness.
- **Governance auditability**: Spec to golden to marker to target to integrated check chain.
- **Deterministic reproducibility**: Same spec always produces same output.
- **Structural equivalence proof**: 14/14 golden comparisons (extended from 12/12 with EMA).
- **CI regression gate**: `codegen-golden` + `codegen-integrated` block merge on drift.

### Trade-offs
- Manual insertion accepted (developers copy fragments from golden snapshots).
- Fragment generation only (not complete files).
- 2/6 Tier 1 artifacts generated.
- Frozen schema and templates (no evolution during generated path operation).
- Golden snapshot maintenance scales linearly.
- Structural-only activation proof for EMA (no live event flow).

### Open Debts
| Debt | Status |
|------|--------|
| D-1: Live event flow proof | STILL OPEN -- SOD-23 |
| D-2: Cross-layer validation | STILL OPEN -- only signal layer validated |
| D-3: Mapper generation feasibility | STILL OPEN -- SOD-21 |
| D-4: Automated fragment insertion | STILL OPEN -- SOD-22 |
| D-5: Config registration automation | STILL OPEN -- deferred |
| D-10 to D-13: Tier 2, template evolution, schema extension, retroactive conversion | DEFERRED by design |

---

## Phase 18: Stabilization Wave (S205-S210)

### Gains
- **Handler extraction completed**: `parseAnalyticalParams()` extracted, handler reduced to 502 lines.
- **Clean build baseline**: All 19 Go modules build with zero errors.
- **Clean test baseline**: All unit tests pass across all 19 modules.
- **Codegen fully validated**: 14/14 golden snapshots, 4/4 integrated slices, 7/7 specs valid.
- **Binary hygiene**: Writer binary excluded from git.
- **Comprehensive debt registry**: 31 items classified by priority (P0-P3).
- **Documentation entropy mapped**: 440 docs analyzed, 11 redundancy clusters, 12-phase cleanup plan.
- **Operational layer fully documented**: All 8 services with startup, health, recovery, config docs.
- **Codegen path in governed state**: Clear boundaries for what is permitted, prohibited, and required.
- **First verified gate**: Every must-finish item independently verified.

### Trade-offs
- CI smoke-analytical not verified on real PR (verification gap, not implementation gap).
- clickhouse-go version misalignment persists (v2.30.0 vs v2.43.0).
- Codegen limited to 2 of 7 families.
- No load testing baseline.
- Documentation entropy remains at 440 files (plan exists, not executed).
- 5 families remain without live event proof via codegen.

### Open Debts
| Debt | Status |
|------|--------|
| OD-1 to OD-9: Carried to refactoring phase | STILL OPEN -- see SOD list |
| OD-10 to OD-16: Deferred past refactoring | STILL OPEN -- various triggers |
| RD-1 to RD-10: Resolved during stabilization | RESOLVED |

---

## Debt Resolution Summary

### Fully Resolved Debts (across all phases)
1. Composition root integration tests -- proven by live pipeline runs
2. Config validation hardening -- KnownFamilies API, format validation
3. Execute actor unit tests -- SafetyGate with 22 tests
4. HTTP correlation ID middleware -- extracted from 7 handlers
5. Migration tooling -- cmd/migrate implemented
6. Writer unit tests -- 25 mapper + 10 inserter tests
7. Reader unit tests -- query builder tests
8. Writer pipeline recovery -- per-family restart with backoff
9. INSERT retry -- exponential backoff with configurable retries
10. Buffer overflow metrics -- overflow counters
11. Mapper error visibility -- WARN logging with context
12. Reader path instrumentation -- three-layer with Server-Timing
13. Integration test -- 7-phase automated E2E validation
14. Writer config validation -- fail-fast startup
15. Handler extraction -- parseAnalyticalParams()
16. Struct DI -- AnalyticalHandlerDeps
17. Smoke test extraction -- validate_analytical_family()
18. Codegen engine -- built and validated
19. Cross-spec uniqueness -- validate-all in CI
20. Deferred item tracking fragmentation -- S209 debt registry

### Superseded Debts (no longer relevant)
1. Error propagation consistency -- accepted as appropriate per-context
2. Error classification taxonomy -- no alerting infrastructure
3. Raccoon-CLI governance constants -- ongoing maintenance, not debt
4. Publisher actor generic extraction -- pattern stable
5. Query client generics -- accepted at current scale
6. Route/gateway/derive boilerplate -- acceptable at current family count

---

## Source Documents

This timeline consolidates the following 18 documents (archived to `docs/archive/gains-tradeoffs/`):

1. `structural-gains-tradeoffs-and-open-debts.md` (S96-S99)
2. `platform-gains-tradeoffs-and-open-debts.md` (S101-S105)
3. `vertical-slice-01-gains-tradeoffs-and-open-debts.md` (S107-S112)
4. `live-baseline-gains-tradeoffs-and-open-debts.md` (S113-S117)
5. `capability-01-gains-tradeoffs-and-open-debts.md` (S119-S124)
6. `cc-02-gains-tradeoffs-and-open-debts.md` (S125-S129)
7. `timeframe-coverage-01-gains-tradeoffs-and-open-debts.md` (S131-S136)
8. `current-capability-consolidation-gains-tradeoffs-and-open-debts.md` (S137-S142)
9. `analytical-runtime-gains-tradeoffs-and-open-debts.md` (S143-S150)
10. `analytical-wave-a-gains-tradeoffs-and-open-debts.md` (S151-S156)
11. `pre-wave-b-analytical-gains-tradeoffs-and-open-debts.md` (S157-S161)
12. `wave-b-iteration-01-gains-tradeoffs-and-open-debts.md` (S163-S167)
13. `wave-b-after-family-02-and-hardening-gains-tradeoffs-and-open-debts.md` (S168-S173)
14. `wave-b-after-family-03-gains-tradeoffs-and-open-debts.md` (S174-S179)
15. `wave-b-after-family-05-gains-tradeoffs-and-open-debts.md` (post-S183-S190)
16. `codegen-readiness-gains-tradeoffs-and-open-debts.md` (S192-S198)
17. `generated-path-gains-tradeoffs-and-open-debts.md` (S199-S204)
18. `stabilization-wave-gains-tradeoffs-and-open-debts.md` (S205-S210)
