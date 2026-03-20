# Stabilization Wave: Gains, Tradeoffs, and Open Debts

**Stage:** S210
**Date:** 2026-03-20
**Scope:** S205 (scope freeze) → S210 (gate review)

---

## 1. What This Wave Covered

The stabilization wave (S205–S210) was a 6-stage closure sequence designed to ensure the system could safely enter a dedicated refactoring phase. It followed the longest expansion arc in the project's history (S131–S204: analytical entry → Wave A → Wave B families → codegen introduction).

| Stage | Action | Outcome |
|-------|--------|---------|
| S205 | Scope freeze + must-finish matrix | 7 MF items, 17 MD items, 12 EF items classified |
| S206 | Analytical implementation closure | Writer, reader, gateway, migrations, CI, diagnostics closed |
| S207 | Codegen path stabilization decision | Controlled stabilization (not frozen, not expanded) |
| S208 | Runtime, config, operational closure | All 8 services documented, health/diagnostics mapped, recovery semantics recorded |
| S209 | Debt registry + cleanup plan | 31 debt items classified, 440-doc entropy mapped, 4-wave refactoring plan |
| S210 | Stabilization gate review | Conditional pass — all MF items verified (1 awaiting CI confirmation) |

---

## 2. Gains

### G-1: Handler extraction completed (MF-1)
`parseAnalyticalParams()` extracted from the analytical handler. File reduced from ~620 lines (at ceiling) to 502 lines. Handler is now safely modifiable during refactoring.

### G-2: Clean build baseline established (MF-5)
All 19 Go modules build with zero errors. This is the first time a full-module build has been verified as a formal gate condition.

### G-3: Clean test baseline established (MF-6)
All unit tests pass across all 19 modules. No pre-existing failures will be confused with refactoring regressions.

### G-4: Codegen fully validated (MF-3, MF-7)
- 14/14 golden snapshots match.
- 4/4 integrated slices match.
- 7/7 specs valid with cross-family uniqueness verified.
- 4 CI gates operational and blocking.

### G-5: Binary hygiene resolved (MF-4)
Writer binary excluded from git via `.gitignore` patterns. No binary artifacts in version control.

### G-6: Comprehensive debt registry
31 items classified by priority (P0–P3) with specific blast radius and action timing. The registry is the first single-source-of-truth for all technical and architectural debt.

### G-7: Documentation entropy mapped with actionable plan
440 architecture docs analyzed. 11 redundancy clusters identified. 12-phase execution order defined. Target: reduce to 120-150 active docs with organized archive.

### G-8: Operational layer fully documented
All 8 services have documented startup validation, health endpoints, recovery semantics, and configuration. 4 smoke scripts cover distinct operational scopes.

### G-9: Codegen path in clear, governed state
S207 decision provides unambiguous boundaries: what's permitted, what's prohibited, what's required. CI gates remain active for drift detection during refactoring.

### G-10: Stabilization gate with verified evidence
First gate in the project's history where every must-finish item was independently verified (build, test, codegen) rather than assumed.

---

## 3. Tradeoffs

### T-1: CI smoke-analytical not verified on real PR
The smoke-analytical CI job is fully defined and the script runs locally, but it hasn't been triggered by a real PR on the CI platform. This means the CI safety net is unproven in its actual execution environment.

**Accepted because:** This is a verification gap, not an implementation gap. The first PR of the refactoring phase will close it.

### T-2: clickhouse-go version misalignment persists
Writer uses clickhouse-go v2.30.0; the latest is v2.43.0. This was consciously frozen during stabilization.

**Accepted because:** No operational incidents from the version gap. Upgrading during stabilization would violate scope and introduce untested dependencies.

### T-3: Codegen integration limited to 2 of 7 families
Only RSI and EMA have codegen governance markers. The remaining 5 families (candle, paper_order, etc.) use golden-only validation without integrated markers.

**Accepted because:** Full integration was explicitly not in scope (S205 EF-4 prohibits retroactive conversion). The 2-family proof is sufficient for the codegen stabilization decision.

### T-4: No load testing baseline
Performance characteristics are unknown. No baseline exists for throughput, latency, or resource consumption.

**Accepted because:** The refactoring phase is structural, not performance-focused. A baseline established now would be invalidated by refactoring changes.

### T-5: Documentation entropy remains at 440 files
The entropy map was built but no cleanup was executed. The docs remain cluttered during this gate.

**Accepted because:** EF-6 explicitly prohibited cleanup during stabilization. The S209 plan is the prerequisite for disciplined cleanup, not a substitute for it.

### T-6: 5 families remain without live event proof via codegen
Generated families compile and subscribe but have no producer in smoke tests. Structural proof only.

**Accepted because:** S207 decision accepts structural proof as sufficient. Live proof is a post-refactoring validation concern.

---

## 4. Open Debts

### Debt carried forward to refactoring phase

| ID | Item | Priority | Registry Reference |
|----|------|----------|--------------------|
| OD-1 | Reader 10-parameter positional signature | P1 | TD-02 |
| OD-2 | Test hardcoded family counts | P2 | TD-03 |
| OD-3 | NATS consumer lag visibility | P2 | TD-08 |
| OD-4 | Schema coherence compile-time check | P2 | TD-11 |
| OD-5 | Automated baseline validation | P2 | TD-13 |
| OD-6 | Module graph evaluation | P1 | AD-01 |
| OD-7 | Superseded docs not marked | P1 | AD-03 |
| OD-8 | Per-family doc boilerplate | P1 | AD-04 |
| OD-9 | Stage report index | P1 | AD-06 |

### Debt deferred past refactoring phase

| ID | Item | Trigger for Re-evaluation |
|----|------|--------------------------|
| OD-10 | TC-02 scope (state persistence, WAL, cold-start) | Next expansion wave |
| OD-11 | Load testing baseline | Post-refactoring, pre-production |
| OD-12 | 4 deferred writer families (tradeburst, volume, ema_crossover, venue_market_order) | Demand-driven |
| OD-13 | Codegen Tier 2 (reader generation) | Post-production proof of Tier 1 |
| OD-14 | clickhouse-go version alignment | Opportunistic or incident-driven |
| OD-15 | Dead-letter queue / backpressure | Scaling evidence |
| OD-16 | CODEGEN_ROOT auto-detection | Developer experience feedback |

### Debt resolved during stabilization wave

| ID | Item | Stage | Resolution |
|----|------|-------|------------|
| RD-1 | H-5 handler extraction | S206 | `parseAnalyticalParams()` extracted |
| RD-2 | Writer binary in VCS | S206 | `.gitignore` + unstage |
| RD-3 | Gateway port missing from lib.sh | S206 | Added to SVC_PORTS |
| RD-4 | Missing compile-time assertions (4 readers) | S206 | Added for Decision, Strategy, Risk, Execution readers |
| RD-5 | Writer missing from live-pipeline health checks | S206 | Added to phases 2, 3, 5, 8 |
| RD-6 | Analytical endpoints missing from live-pipeline | S206 | Added Phase 6 validation |
| RD-7 | Codegen path unclear (freeze vs expand) | S207 | Controlled stabilization decision |
| RD-8 | Runtime/config not documented | S208 | Comprehensive closure doc |
| RD-9 | No debt registry | S209 | 31-item classified registry |
| RD-10 | No documentation cleanup plan | S209 | 12-phase execution plan |

---

## 5. Wave Health Assessment

| Dimension | Score | Notes |
|-----------|-------|-------|
| Implementation completeness | 9/10 | All tracks closed. -1 for CI verification gap. |
| Test coverage | 8/10 | All tests pass. -2 for no integration test gate and no load baseline. |
| Documentation state | 6/10 | Comprehensive but entropic. Plan exists but not yet executed. |
| Operational readiness | 8/10 | All services documented and operational. -2 for no real-CI proof. |
| Debt visibility | 10/10 | First complete, classified debt registry in project history. |
| Plan maturity | 9/10 | 4-wave plan with entry/exit gates. -1 for CI dependency in entry. |
| Overall | **8.3/10** | System is structurally stable and ready for the refactoring phase. |
