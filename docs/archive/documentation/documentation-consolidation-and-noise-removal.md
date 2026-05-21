# Documentation Consolidation and Noise Removal (S215)

> Executed: 2026-03-20
> Scope: Full documentation entropy reduction per S209 cleanup plan

## Objective

Reduce documentation entropy in `docs/architecture/` from ~457 files to a navigable, canonical set by applying archive, consolidate, and reorganize actions defined in the S209 entropy map and debt registry.

## Approach

1. **Archive before delete** — no content lost; all originals preserved in `docs/archive/` with subdirectory organization
2. **Consolidate by cluster** — multiple docs covering the same topic merged into single authoritative documents
3. **Preserve active references** — documents still referenced by code, CI, or active processes remain untouched
4. **One cluster at a time** — each cluster processed independently to limit blast radius

## Results Summary

| Metric | Before | After | Reduction |
|--------|--------|-------|-----------|
| Architecture docs | 457 | 237 | 48% |
| Files archived | — | 245 | — |
| Consolidated docs created | — | 15 | Replaced ~120 originals |
| Archive categories | 0 | 16 | — |
| Stage reports | 214 | 214 + INDEX.md | Index added |

## Consolidation Clusters Executed

### 1. Next-Wave Recommendations (18 files → 1)
- **Source:** 18 "next-wave-recommendations-after-*" files spanning all project gates
- **Target:** `next-wave-recommendations-timeline.md` — chronological timeline with EXECUTED/DEFERRED/SUPERSEDED annotations
- **Archive:** `docs/archive/next-wave/`
- **Value:** Eliminated most scattered forward-looking recommendations; single source for deferred items

### 2. Gains/Tradeoffs/Open Debts (18 files → 1)
- **Source:** 18 "*-gains-tradeoffs-and-open-debts.md" files across all phases
- **Target:** `gains-tradeoffs-and-open-debts-timeline.md` — phase-by-phase gains, tradeoffs, and debt tracking with resolution status
- **Archive:** `docs/archive/gains-tradeoffs/`
- **Value:** Primary source of scattered deferred work now centralized

### 3. Deferred/Triggered Refactors (14 files → 1)
- **Source:** 14 "refactors-*", "triggered-*", "evidence-driven-*" files
- **Target:** `deferred-work-registry.md` — categorized registry with status tracking (DONE/OPEN/SUPERSEDED)
- **Archive:** `docs/archive/deferred-work/`
- **Value:** Most operationally impactful consolidation; single source of truth for all deferred work

### 4. Family Lifecycle Records (34 files → 4)
- **Source:** 34 family-specific docs across families 03–06
- **Target:** `family-{03,04,05,06}-lifecycle-record.md` — standard structure (Selection → Definition → Implementation → Validation → Findings)
- **Archive:** `docs/archive/families/`
- **Value:** Reduced per-family boilerplate from 8-11 docs to 1

### 5. Wave B Lifecycle Records (20 files → 2 + archive)
- **Source:** 12 Wave B family-specific docs + 8 pattern/iteration docs
- **Target:** `wave-b-family-{01,02}-lifecycle-record.md`
- **Kept:** `wave-b-family-expansion-pattern-v2.md`, `wave-b-family-checklist-*`, `wave-b-iteration-constraints-*`
- **Archive:** `docs/archive/wave-b/`

### 6. Analytical Infrastructure (23 files → 4)
- **Source:** 23 analytical boundary, runtime, observability, and scope docs
- **Target:**
  - `analytical-boundary-and-responsibility-model.md`
  - `analytical-runtime-lifecycle-and-recovery.md`
  - `analytical-observability-and-runbook.md`
  - `analytical-scope-and-planning-summary.md`
- **Kept:** `analytical-implementation-closure.md`, `analytical-writer-correctness-*`, `analytical-storage-strategy.md`, `analytical-generated-path-consolidation.md`, `analytical-vs-generated-ownership-*`
- **Archive:** `docs/archive/analytical/`

### 7. Codegen (30 files → 3 + archive)
- **Source:** 12 codegen spec/validation/boundary docs + 18 generated-path lifecycle docs
- **Target:**
  - `codegen-specification-and-schema.md`
  - `codegen-validation-and-ci-strategy.md`
  - `codegen-boundaries-and-governance.md`
- **Kept:** `codegen-path-stabilization-or-freeze-decision.md`, `codegen-current-usage-*`, `codegen-tranche-scoping.md`
- **Archive:** `docs/archive/codegen/`

### 8. Gate/Readiness Reviews (30 files → archive)
- **Source:** 30 "post-*-readiness-review", "pre-*-gate", gate-adjacent docs
- **Archive:** `docs/archive/gates/`
- **Value:** Historical audit trail preserved, no longer cluttering active docs

### 9. Superseded Documents (5 files → archive)
- **Source:** v1→v2 transitions, duplicate docs
- **Archive:** `docs/archive/superseded/`

### 10. Historical Phase Docs (various → archive)
- **Vertical Slice** (6 files) → `docs/archive/vertical-slice/`
- **Live Pipeline** (3 files) → `docs/archive/live-pipeline/`
- **Capability-01** (8 files) → `docs/archive/capability/`
- **CC-02** (7 files) → `docs/archive/cc-02/`
- **Timeframe** (8 files) → `docs/archive/timeframe/`
- **ClickHouse Entry** (7 files) → `docs/archive/clickhouse-entry/`
- **Domain Lifecycle** (14 files) → `docs/archive/domain-lifecycle/`

### 11. Stage Report Index (new)
- **Created:** `docs/stages/INDEX.md` — thematic grouping of all 214 stage reports across 17 phases
- **Value:** Makes audit trail navigable without modifying any stage reports

## What Was NOT Touched

- **Stage reports** — all 214 preserved as-is (immutable audit trail)
- **Domain design docs** — signal, decision, strategy, risk, execution domain designs remain in place
- **Active reference docs** — implementation notes, definitions, and docs referenced by code/CI
- **Recent S211-S214 work** — refactor wave docs, census, strategic refactor, analytical consolidation
- **Code and configuration** — zero code changes

## Guard Rails Applied

- All originals preserved in `docs/archive/` (git history also available)
- Each cluster processed independently
- No deletions — only moves and consolidations
- Active references verified before archiving
- Recent work (S211+) explicitly excluded from consolidation
