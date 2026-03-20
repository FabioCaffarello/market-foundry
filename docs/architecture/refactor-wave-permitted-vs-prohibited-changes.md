# Refactor Wave: Permitted vs Prohibited Changes

**Stage:** S211
**Date:** 2026-03-20
**Governing document:** `refactor-wave-charter-and-entry-freeze.md`
**Status:** ACTIVE — Enforced for the duration of the Refactor Wave.

---

## 1. Purpose

This document provides an unambiguous classification of what changes are permitted, conditionally permitted, and prohibited during the Strategic Refactoring and Documentation Consolidation Phase. When in doubt, consult this document first.

---

## 2. Permitted Changes (GREEN)

These changes are explicitly authorized and require no additional approval.

### 2.1 Documentation

| Change | Conditions |
|--------|------------|
| Consolidate redundant architecture docs into fewer files | Follow S209 entropy map cluster plan |
| Move superseded docs to `docs/archive/` | Preserve original content; do not delete without archiving |
| Delete architecture docs that are fully absorbed into consolidated versions | Only after consolidated version is committed and verified |
| Add cross-references between surviving docs | Improve navigability only; do not add new content |
| Create stage report index (by theme/phase) | AD-06 in debt registry |
| Mark superseded documents with frontmatter annotation | AD-03 in debt registry |
| Update frontmatter (date, status, predecessor/successor links) | Maintenance only |
| Create `docs/archive/` directory structure | Required for entropy reduction |
| Produce stage reports for each refactoring sub-stage | Standard project practice |

### 2.2 Code

| Change | Conditions |
|--------|------------|
| Refactor reader constructors to options pattern or builder (TD-02) | Must not change external behavior; all existing tests must pass |
| Replace hardcoded family count assertions with registry-driven (TD-03) | Must not change test coverage semantics |
| Extract shared utilities from duplicated code | Only if duplication is proven (3+ identical sites) and extraction is safe |
| Rename internal symbols for clarity | Only within a single module; cross-module renames require evaluation |
| Remove dead code (verified unreachable) | Must verify via static analysis or grep; do not guess |
| Fix compiler warnings or linter issues | Only in files already being modified for other permitted work |
| Update comments and godoc in modified files | Only in files already being changed |

### 2.3 Build / CI

| Change | Conditions |
|--------|------------|
| Fix CI job failures discovered during RW-1 verification | P0 — required for phase entry |
| Update Makefile targets to reflect renamed/moved files | Mechanical consequence of permitted refactoring |
| Fix broken test fixtures due to file moves | Mechanical consequence of permitted refactoring |

---

## 3. Conditionally Permitted Changes (YELLOW)

These changes are permitted only under specific conditions. The condition must be documented before the change is made.

### 3.1 Documentation

| Change | Condition Required |
|--------|-------------------|
| Delete a stage report | Only if content is fully absorbed into a consolidated index AND original is archived |
| Modify the S209 debt registry | Only to mark items DONE/DEFERRED/SUPERSEDED or add newly discovered items with priority |
| Create new architecture docs | Only if consolidation requires a new canonical document that replaces 3+ existing files |

### 3.2 Code

| Change | Condition Required |
|--------|-------------------|
| Evaluate module graph consolidation (AD-01) | Document findings as architecture analysis. Execute merge only if blast radius is assessed, test coverage is sufficient, and the change is reversible |
| Fix newly discovered P0 blockers | Must add to debt registry before fixing; must document justification |
| Modify test infrastructure (helpers, fixtures) | Only as consequence of permitted code changes; no speculative test refactoring |
| Modify `go.mod` / `go.sum` | Only as mechanical consequence of module refactoring (AD-01), not for version upgrades |

### 3.3 Build / CI

| Change | Condition Required |
|--------|-------------------|
| Add a new CI check | Only if it validates an existing contract (e.g., archive integrity check). Must not test new functionality |
| Modify existing CI job steps | Only to fix bugs or adapt to file moves from permitted refactoring |

---

## 4. Prohibited Changes (RED)

These changes are **not permitted under any circumstance** during the Refactor Wave. Violation requires a formal charter amendment.

### 4.1 Functional Expansion

| Prohibited Change | Rationale |
|-------------------|-----------|
| Add new analytical families (EF-1) | Phase is structural, not functional |
| Add new ClickHouse tables or columns (EF-9) | Schema is frozen |
| Add new NATS streams or subjects (EF-8) | Infrastructure is frozen |
| Add new HTTP endpoints or routes (RF-2) | No new API surface |
| Add new domain entities or events (RF-1) | No new business logic |
| Add new `cmd/*` services (EF-12) | No new services |
| Add new writer pipeline families (EF-11) | Write path is frozen |

### 4.2 Codegen Expansion

| Prohibited Change | Rationale |
|-------------------|-----------|
| Modify codegen templates (EF-2) | Templates frozen per S193 |
| Extend codegen spec schema (EF-3) | 14-field schema is sufficient |
| Convert manual families to generated (EF-4) | Manual families are golden references |
| Authorize Tier 2 codegen (EF-5) | Not designed, not validated |
| Integrate additional families beyond RSI + EMA | Expansion, not refactoring |

### 4.3 Infrastructure / Dependencies

| Prohibited Change | Rationale |
|-------------------|-----------|
| Upgrade clickhouse-go or other dependencies (RF-3) | Version changes introduce untested paths |
| Modify Docker Compose service definitions (add/remove services) | Infrastructure is frozen |
| Add new external service dependencies | No new infrastructure |
| Modify ClickHouse configuration files | Schema and config are frozen |

### 4.4 Behavioral / Structural

| Prohibited Change | Rationale |
|-------------------|-----------|
| Performance optimization work (RF-4) | Structural changes must complete first |
| New architecture decisions (RF-7) | Phase executes existing decisions |
| Mass doc deletion without archive (RF-5) | Content preservation is mandatory |
| Module boundary changes without evaluation (RF-6) | Requires documented assessment first |
| CI job definition changes beyond bug fixes (RF-8) | CI is the safety net |
| Changes to startup validation logic | Operational layer is closed (S208) |
| Changes to health/diagnostics endpoints | Operational layer is closed (S208) |
| Changes to recovery/reconnection semantics | Operational layer is closed (S208) |

---

## 5. Edge Case Resolution

### "This refactoring requires a small behavioral change"
If a refactoring (e.g., reader constructor change) requires adjusting internal behavior, it is permitted **only if**:
1. External behavior (HTTP responses, NATS messages, ClickHouse writes) is unchanged.
2. All existing tests pass without modification to assertions.
3. The change is documented in the stage report.

If any of these conditions cannot be met, the refactoring must be re-scoped or deferred.

### "I found a bug while refactoring"
- **If the bug is in code being actively refactored:** fix it, add a test, document in stage report.
- **If the bug is in unrelated code:** add to debt registry as a new item with priority. Do not fix unless it is P0 and blocks refactoring work.

### "This doc consolidation reveals a design inconsistency"
Document the inconsistency in the stage report. Do not resolve it during the documentation wave. Design resolution is out of scope.

### "A dependency has a security vulnerability"
This is the one potential exception to RF-3. If a CVE is published with severity HIGH or CRITICAL for a direct dependency during the phase:
1. Document the CVE and affected dependency.
2. Assess whether the vulnerability is exploitable in the project's context.
3. If exploitable: request charter amendment for a targeted version bump only.
4. If not exploitable: register in debt registry and defer.

---

## 6. Quick Reference Matrix

| Category | GREEN | YELLOW | RED |
|----------|-------|--------|-----|
| Consolidate docs | Yes | — | — |
| Archive docs | Yes | — | — |
| Delete docs without archive | — | — | **Prohibited** |
| Create new canonical doc | — | If replaces 3+ files | — |
| Refactor reader signatures | Yes | — | — |
| Fix hardcoded test counts | Yes | — | — |
| Evaluate module consolidation | — | Document first | — |
| Execute module merge | — | After evaluation | — |
| Add analytical family | — | — | **Prohibited** |
| Add HTTP endpoint | — | — | **Prohibited** |
| Upgrade dependency | — | — | **Prohibited** (except CVE) |
| Modify codegen template | — | — | **Prohibited** |
| Add ClickHouse table | — | — | **Prohibited** |
| Fix CI bug | Yes | — | — |
| Modify CI job structure | — | Only for file moves | — |
| Add new CI job | — | Only if validates existing contract | — |
| Performance tuning | — | — | **Prohibited** |
| New architecture decisions | — | — | **Prohibited** |
