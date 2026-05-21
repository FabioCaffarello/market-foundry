# Pre-Refactor Technical Debt Registry and Cleanup Plan

**Stage:** S209
**Date:** 2026-03-20
**Status:** Registry — read-only reference for the refactoring phase.

---

## 1. Purpose

This document is the authoritative technical debt registry for market-foundry as of S209. It catalogs every known debt item, classifies it by priority and blast radius, and provides the entry plan for the refactoring phase. No cleanup is executed here — only registered and planned.

---

## 2. Debt Classification Framework

| Priority | Criteria | Action Timing |
|----------|----------|---------------|
| **P0 — Structural Blocker** | Blocks safe refactoring; creates cascade risk if untouched | Must resolve before or at refactoring phase entry |
| **P1 — High-Value Cleanup** | Reduces maintenance cost, improves navigability, or eliminates confusion | First wave of refactoring phase |
| **P2 — Moderate Cleanup** | Real improvement but no immediate risk | Second wave or opportunistic |
| **P3 — Cosmetic / Low-Value** | Nice to have; no operational or structural impact | Defer or skip |

---

## 3. Technical Debt Registry

### 3.1 Structural / Code Debt

| ID | Item | Current State | Priority | Blast Radius | Notes |
|----|------|---------------|----------|--------------|-------|
| **TD-01** | `parseAnalyticalParams()` extraction (H-5) | **RESOLVED** (S217 verified: function exists at analytical.go:90-122, file is 502 lines) | P0 (done) | Gateway handlers | Scoped since S189. S217 code inspection confirmed extraction already completed. |
| **TD-02** | Reader 10-parameter positional signature (TRIG-3) | 6 readers with 10-param constructors | P1 (formally deferred S217) | `internal/adapters/clickhouse/` | Escalation threshold reached but no new families added in this wave. Formally deferred to next expansion wave — trigger: Family 07 addition. |
| **TD-03** | Test assertion hardcoded family count (D-6) | Tests break on new family addition | P2 | `cmd/writer/`, `cmd/gateway/` | Count literals (e.g., `assert.Len(families, 7)`) scattered across test files. Replace with registry-driven counts. |
| **TD-04** | `CODEGEN_ROOT` manual env var requirement (D-7) | Codegen scripts require explicit `CODEGEN_ROOT` | P3 | Developer experience | Auto-detection via `go list -m` or relative path resolution. Low priority. |
| **TD-05** | Writer binary previously committed to git | Binary removed (S206), gitignore added | P0 (done) | Hygiene | Verified resolved in S206. Confirm removal persists. |
| **TD-06** | Backoff jitter absent in writer retry (DEF-U6) | Fixed backoff, no jitter | P3 | Writer retry | Single-instance writer makes thundering herd impossible. Only relevant at scale. |
| **TD-07** | ClickHouse client timeout hardcoded (MD-8) | No configurability | P3 | Writer/Reader | No incidents observed. Address when operational evidence demands it. |
| **TD-08** | NATS consumer lag not exposed (DEF-U3) | Health via `/statusz`/`/diagz` only | P2 | Observability | Monitoring gap. No operational incidents, but limits proactive alerting. |
| **TD-09** | Gateway lacks `/statusz`/`/diagz` | By design — stateless proxy | P3 | Gateway | Documented as intentional in S208. Revisit only if gateway gains state. |
| **TD-10** | No load testing baseline (D5) | No performance data | P2 | All services | Refactoring is structural, not performance. Establish baseline post-refactoring. |
| **TD-11** | Schema coherence compile-time verification absent (DEF-C2) | Review-enforced only | P2 | Migrations/Schema | Under 12-table/100-column threshold. Becomes P1 if schema grows. |
| **TD-12** | Gateway tracker integration absent (OD-03) | Gateway health inferred from downstream | P3 | Observability | Operational, not structural. |
| **TD-13** | Automated baseline validation absent (OD-04) | 30 success criteria are manual checks | P2 | CI/CD | Reduces human error in gate reviews. |
| **TD-14** | Generated families lack live event proof (D-1) | EMA compiles but no producer in smoke tests | P2 | Codegen | Structural proof sufficient for refactoring. Live proof needed before production. |
| **TD-15** | Only 2 of 7 families have codegen governance markers | RSI and EMA integrated; candle, paper_order, etc. remain manual | P2 | Codegen | Expanding marker coverage is a natural refactoring-phase task. |
| **TD-16** | Cross-layer codegen validation gap (D-2) | Signal layer only validated | P2 | Codegen | Codegen covers A1+A2 only; cross-layer differences are in manual artifacts. |
| **TD-17** | Mapper generation not designed (A3/D-3) | Fully manual | P3 | Codegen | Pure expansion; no dependency for refactoring. |

### 3.2 Architectural Debt

| ID | Item | Current State | Priority | Notes |
|----|------|---------------|----------|-------|
| **AD-01** | Go module count in monorepo | **PARTIALLY RESOLVED** (S220: 19→17 modules; `internal/migrate` absorbed into `cmd/migrate/migrate`, `internal/adapters/repositories` absorbed into `internal/application/configctl/memoryrepo`) | P1 (partial) | H-06 completed in S220. 2 modules absorbed (zero new dependency edges). Remaining 17 modules justified by external dependency isolation or size. Further simplification not recommended without evidence of friction. |
| **AD-02** | Deferred items scattered across 15+ "next-wave" docs | **RESOLVED** (S215 consolidated into `deferred-work-registry.md` + `gains-tradeoffs-and-open-debts-timeline.md`) | P0 (done) | Addressed by S215 consolidation wave. |
| **AD-03** | Superseded documents not marked | **RESOLVED** (S215: 245 docs archived into 16 categories, superseded docs moved to `docs/archive/`) | P1 (done) | Addressed by S215 consolidation wave. |
| **AD-04** | Per-family doc boilerplate (~8-12 docs per family) | **RESOLVED** (S215: family lifecycle docs consolidated into 4 records + archive) | P1 (done) | Addressed by S215 consolidation wave. |
| **AD-05** | 440 architecture docs with high entropy | **PARTIALLY RESOLVED** (S215: 457→243 active docs, 47% reduction; target ≤150 not met) | P0 (partial) | S215 consolidation reduced entropy significantly. Remaining 93-doc gap is the sole FAIL exit criterion (XC-1). |
| **AD-06** | Stage reports (205 files) accumulate without summary index | **RESOLVED** (S215: INDEX.md created with 214 entries across 17 phases) | P1 (done) | Addressed by S215 consolidation wave. |
| **AD-07** | TC-01 deferred items (D-01 through D-06) | All deferred with TC-02 gate | P2 | State persistence (D-06) is TC-02 hard gate. Not in refactoring scope. |
| **AD-08** | Cold-start bootstrap / state persistence WAL | Deferred past Wave A | P2 | Architectural decision needed before implementation. Not blocking refactoring. |
| **AD-09** | 4 deferred writer families | tradeburst, volume, ema_crossover, venue_market_order | P3 | Acknowledged but explicitly not in current scope. |

### 3.3 CI/Build Debt

| ID | Item | Current State | Priority | Notes |
|----|------|---------------|----------|-------|
| **CI-01** | CI smoke-analytical job not verified end-to-end | Job defined in ci.yml, untested on real PR | P0 | MF-2 from S205. CI is the safety net for refactoring. |
| **CI-02** | Codegen integrated check needs full-chain verification | Script exists, needs all 7 families verified | P0 | MF-3 from S205. Drift during refactoring would be invisible without this. |
| **CI-03** | All 13 Go modules build verification | Presumed working, not gated | P0 | MF-5 from S205. Entry condition for refactoring. |
| **CI-04** | All unit tests pass verification | Presumed passing, not gated | P0 | MF-6 from S205. Entry condition for refactoring. |
| **CI-05** | Codegen cross-spec validation gate verification | Implemented in S201, needs gate run | P0 | MF-7 from S205. Collision prevention during refactoring. |

---

## 4. S205 Must-Finish Items: Closure Status

These items from the S205 stabilization matrix are **prerequisites for the refactoring phase**:

| S205 ID | Item | Status as of S209 | Action Required |
|---------|------|-------------------|-----------------|
| MF-1 | H-5 `parseAnalyticalParams()` extraction | **NOT DONE** — remains at line limit | Must complete before refactoring entry |
| MF-2 | CI smoke-analytical verification | **NOT VERIFIED** | Must verify before refactoring entry |
| MF-3 | Codegen integrated check on all 7 families | **NOT VERIFIED** | Must verify before refactoring entry |
| MF-4 | Writer binary removal | **DONE** (S206) | Confirm persistence |
| MF-5 | All modules build cleanly | **NOT VERIFIED** | Must verify before refactoring entry |
| MF-6 | All unit tests pass | **NOT VERIFIED** | Must verify before refactoring entry |
| MF-7 | Codegen cross-spec validation | **NOT VERIFIED** | Must verify before refactoring entry |

**Gate rule:** The refactoring phase MUST NOT begin until all MF items are verified. This is a hard prerequisite, not a recommendation.

---

## 5. Cleanup Plan Structure

### Phase R1: Entry Gate (pre-refactoring)

1. Close all P0 items (MF-1 through MF-7).
2. Verify clean build + test baseline across all 13 modules.
3. Tag the repository at the stabilization exit point.
4. Create a `REFACTORING-PHASE.md` at repo root with scope, rules, and exit criteria.

### Phase R2: High-Value Structural Cleanup

1. Execute documentation entropy plan (see companion document).
2. Address TD-01 (handler extraction).
3. Address TD-02 (reader signature refactoring).
4. Address AD-01 (evaluate module consolidation).
5. Address AD-04 (family doc consolidation).

### Phase R3: Moderate Cleanup

1. Address TD-03 (test hardcoded counts).
2. Address TD-08 (NATS consumer lag visibility).
3. Address TD-10 (load testing baseline).
4. Address AD-06 (stage report index).

### Phase R4: Verification and Exit Gate

1. All modules build clean.
2. All tests pass.
3. CI gates verified.
4. Documentation entropy reduced to target (see companion document).
5. No new P0 items introduced.

---

## 6. What This Registry Does NOT Cover

- **Feature work**: No new families, no new domains, no new services.
- **Performance optimization**: Load testing baseline is registered but execution is P2.
- **External integrations**: Venue adapter expansion, real exchange connectivity.
- **TC-02 scope**: State persistence, WAL, cold-start bootstrap — these are post-refactoring.

---

## 7. Registry Maintenance Rules

1. This document is the **single source of truth** for technical debt during the refactoring phase.
2. New debt discovered during refactoring must be added here with a priority classification.
3. Completed items must be marked with stage number and date.
4. Priority can be re-evaluated but changes must be justified in the Notes column.
5. Items cannot be deleted — only marked as DONE, DEFERRED, or SUPERSEDED.
