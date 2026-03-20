# Refactor Wave — Gains, Trade-offs, and Open Debts

**Date:** 2026-03-20
**Scope:** S211–S215 (Strategic Refactoring and Documentation Consolidation)
**Status:** Phase ending — conditional exit

---

## 1. Gains

### 1.1 Documentation Entropy Reduction
- **Before:** ~457 active architecture docs with no consolidation, no archive, no index.
- **After:** 240 active docs, 245 archived in 16 organized categories, 15 consolidated canonical documents, stage index (214 entries).
- **Magnitude:** 47% reduction in active docs. Archive fully preserved with no information loss.
- **Value:** Navigability improved. New contributors can find canonical references without wading through historical progression.

### 1.2 Consumer Spec Factory (H-02)
- **Before:** 18 consumer spec functions, each 12 lines of boilerplate with only 3 varying parameters.
- **After:** Single `newConsumerSpec()` factory; 18 call sites reduced to one-liners.
- **Magnitude:** ~180 lines of duplication eliminated. Adding a new family's consumer spec is now 1 line instead of 12.
- **Value:** Directly reduces family-expansion blast radius (the primary pressure point identified in S212).

### 1.3 ClickHouse Query Builder (H-03)
- **Before:** 6 analytical readers each constructed SQL queries with independent, duplicated logic.
- **After:** Centralized `BuildQuery()` in `query_builder.go` with dedicated tests.
- **Magnitude:** ~60 lines of duplication removed. All 6 readers delegate to one implementation.
- **Value:** Single point of change for query construction. Reduces risk of divergent SQL across domain readers.

### 1.4 Generic Actor Infrastructure (H-04, partial)
- **Before:** No shared infrastructure for store consumer actors. Each family had a full actor implementation (~300 lines).
- **After:** `GenericConsumerActor` and `ProjectionStats` created as reusable types.
- **Magnitude:** ~160 lines of new infrastructure. Projected savings of ~1,800 lines when per-family actors migrate.
- **Value:** Foundation laid for the largest single duplication reduction in the codebase. Migration is mechanical but was deferred to avoid blast radius in this wave.

### 1.5 Analytical/Generated Path Ownership (S214)
- **Before:** Ownership boundaries between manual code and codegen-governed code were implicit. No markers in source. Historical docs scattered across 30+ files.
- **After:** Explicit `manual:owned` annotations in 7 source files. 3 canonical documents. 3-zone model (Human/Machine/Mixed) formalized.
- **Value:** Any contributor can now determine who owns a file by looking at its markers. Codegen governance is explicit for 2 families (RSI, EMA). The boundary is documented, not guessed.

### 1.6 Architectural Census (S212)
- **Before:** No inventory of the 19-module workspace. Duplication unquantified. No prioritized refactoring map.
- **After:** Complete census (8 runtimes, 19 modules, 6 layers). 10 duplication clusters identified (~10,100 recoverable lines). Scored priority map (6 HIGH / 7 MEDIUM / 6 LOW).
- **Value:** Future refactoring decisions are now evidence-based, not intuition-based. The map survives this wave and guides future work.

### 1.7 Governance Discipline (S211)
- **Before:** No formal expansion freeze mechanism. Risk of scope creep during structural work.
- **After:** 17-item freeze matrix. Permitted/prohibited change matrix (GREEN/YELLOW/RED). 13 hard exit criteria.
- **Value:** The freeze held. Zero violations detected. This proves the governance model works.

---

## 2. Trade-offs

### 2.1 Depth vs. Breadth
- **Trade-off:** The wave executed 2.5 of 6 HIGH-priority refactoring items. It prioritized clean execution of fewer items over partial execution of all.
- **Consequence:** NATS sub-packaging (H-01), module graph simplification (H-06), and full actor migration (H-04) remain undone.
- **Assessment:** Correct trade-off. The executed items are clean and tested. Partial execution of all 6 would have risked regressions.

### 2.2 Documentation Count vs. Information Preservation
- **Trade-off:** The consolidation preserved all content (archive, not delete). This kept the count higher than aggressive deletion would have.
- **Consequence:** 240 active docs vs ≤150 target. Archive is complete but large (245 docs).
- **Assessment:** Acceptable trade-off for the consolidation wave. A short follow-up tranche can archive more aggressively now that the organization structure exists.

### 2.3 Ownership Annotations vs. Template Updates
- **Trade-off:** S214 added `manual:owned` markers to source files but did not update codegen templates to use the new factory patterns (consumer spec, query builder).
- **Consequence:** Pre-existing drift in RSI/EMA golden snapshots. Code uses factory; goldens show expanded literal. Both produce identical runtime values.
- **Assessment:** Correct trade-off within the freeze rules. Template changes are prohibited under EF-2/EF-3. The drift is cosmetic and documented.

### 2.4 Local Verification vs. CI Verification
- **Trade-off:** All build/test/codegen verification was done locally. No real push to verify CI pipeline.
- **Consequence:** EC-7 remains pending. We have high confidence but not proof.
- **Assessment:** Unavoidable within the session scope. Must be closed in the exit tranche.

---

## 3. Open Debts

### 3.1 Structural Debts (from S212 map, not yet addressed)

| ID | Item | Priority | Lines Recoverable | Status |
|----|------|----------|-------------------|--------|
| H-01 | NATS adapter sub-packaging | HIGH | ~2,000 (complexity, not lines) | NOT STARTED |
| H-04 | Per-family actor migration to generic | HIGH | ~1,800 | INFRASTRUCTURE READY, MIGRATION DEFERRED |
| H-06 | Module graph simplification (19→~10) | HIGH | Structural | NOT STARTED |
| M-01 | Analytical use case consolidation | MEDIUM | ~400 | NOT STARTED |
| M-02 | HTTP handler extraction + registration | MEDIUM | ~200 | NOT STARTED |
| M-03 | Writer pipeline consolidation | MEDIUM | ~150 | NOT STARTED |
| M-04 | KV store generic extraction | MEDIUM | ~600 | NOT STARTED |
| M-05 | Consumer/publisher generic extraction | MEDIUM | ~800 | NOT STARTED |
| M-06 | Gateway compose cleanup | MEDIUM | ~100 | NOT STARTED |
| M-07 | Settings schema split | MEDIUM | Structural | NOT STARTED |

### 3.2 Documentation Debts

| Item | Current | Target | Gap |
|------|---------|--------|-----|
| Active architecture docs | 240 | ≤150 | ~90 docs to archive or consolidate |
| Cross-document references | Unknown | All valid | Archived docs may have broken inbound refs |
| Domain subdirectory organization | Flat | `domains/{signal,...}/` | Optional, deferred |
| Deep content merge | Curated consolidation | Maximum conciseness | Not attempted |

### 3.3 Process Debts

| Item | Status |
|------|--------|
| MF-1: Handler extraction (`parseAnalyticalParams()`) | NOT DONE — P0 blocker |
| EC-7: CI verification on real push | PENDING |
| EC-8/XC-11: Repository tagging | PENDING |
| XC-13: Debt registry final update | PARTIAL |
| Golden snapshot drift (RSI/EMA) | DOCUMENTED, NOT FIXED |

### 3.4 Accepted Permanent Trade-offs (unchanged from prior phases)

These are not debts — they are design decisions that will not change:
- Eventual consistency between operational and analytical paths
- Paper-only trading (no real venue execution)
- NATS subject cardinality (one per family × timeframe × symbol)
- Per-family actor isolation (intentional, not redundancy)
- Codegen scope limited to Tier 1 artifacts (consumer specs + pipeline entries)

---

## 4. Debt Disposition Summary

| Category | Count | Action |
|----------|-------|--------|
| Must close in exit tranche | 5 | MF-1, EC-7, XC-1 doc count, XC-11 tag, XC-13 registry |
| Deferred to next expansion wave | 10 | H-01, H-04 migration, H-06, M-01–M-07 |
| Accepted permanent trade-offs | 5 | No action required |
| Documented exceptions | 2 | Golden snapshot drift, domain subdirectory |

---

## 5. Net Assessment

The Foundry exited the S211–S215 wave in a measurably better state:
- **47% less documentation noise**
- **~240 fewer lines of code duplication** (with infrastructure for ~1,800 more)
- **Explicit ownership boundaries** where none existed
- **Complete architectural census** for evidence-based decisions
- **Zero governance violations** — the freeze model works

The wave did not achieve perfection. 3.5 of 6 HIGH items remain open. Documentation exceeds target. CI is unverified on push. These are honest gaps, not failures — the wave correctly prioritized clean execution over rushed coverage.

The remaining gaps are closeable in one short tranche.
