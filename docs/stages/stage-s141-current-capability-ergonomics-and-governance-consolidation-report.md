# Stage S141 — Current Capability Ergonomics and Governance Consolidation Report

> **Date:** 2026-03-19
> **Predecessor:** S138–S140 (baseline, diagnostics, recovery semantics)
> **Successor:** S142 (recommended)

---

## 1. Executive Summary

S141 applies targeted ergonomic improvements and governance formalization to the current Market Foundry capabilities. No new features were introduced. No horizontal refactoring was performed. The focus was: reduce real operational friction, improve configuration discoverability, centralize script constants, and define clear principles for the future ClickHouse/migrations entry.

---

## 2. Ergonomic Improvements Applied

### 2.1. Script Shared Library (`scripts/utils/lib.sh`)

**Problem:** Every script (live-pipeline-activate.sh, diag-check.sh, smoke-first-slice.sh, smoke-multi-symbol.sh) reimplemented color output functions, error tracking, and JSON parsing helpers. Service port mappings were scattered across scripts as magic strings.

**Solution:** Created `scripts/utils/lib.sh` with:
- Standard logging functions (`pass`, `fail`, `info`, `warn`, `phase`)
- Error tracking (`ERRORS` counter, `record_fail`)
- Service port map (`SVC_PORTS` associative array) — single source of truth
- Environment-overridable timeouts (`HEALTH_WAIT_MAX`, `CANDLE_WAIT_MAX`, etc.)
- JSON extraction helpers (`json_field`, `json_nested`, `json_has_key`)
- Compose helper (`compose_cmd`)
- Service lists (`ALL_SERVICES`, `PIPELINE_SERVICES`)

**Impact:** Future scripts source one file instead of copy-pasting 30+ lines of boilerplate. Timeout magic numbers can be overridden via environment without editing scripts.

### 2.2. Makefile `diag` Target

**Problem:** `scripts/diag-check.sh` existed but had no Makefile target. Operators had to know the script path.

**Solution:** Added `make diag` target.

**Impact:** Diagnostic workflow is now `make diag` — consistent with `make smoke`, `make live`, etc.

### 2.3. Config Reference Document (`deploy/configs/CONFIG-REFERENCE.md`)

**Problem:** Operators had to read Go source code or individual JSONC comments to understand valid config fields, types, defaults, and ranges. No single reference existed.

**Solution:** Created `CONFIG-REFERENCE.md` co-located with config files, documenting:
- All common fields (log, http, nats) with types, defaults, and constraints
- Service port assignments
- Pipeline config (timeframes, family lists, dependency chain)
- Venue config with valid ranges
- Diagnostic endpoint summary

**Impact:** New operators can understand the config surface from one document without reading source code.

### 2.4. JSONC Config Documentation Enhancement

**Problem:** Config files had minimal comments. Service purpose, port, and dependencies were not documented in the configs themselves.

**Solution:** Enhanced all 6 service config files with:
- Service description header (purpose, port, dependencies)
- Valid value annotations inline (e.g., `// level: debug | info | warn | error`)
- Cross-references to derive.jsonc for family dependency rules
- Venue activation gate ceremony steps

**Impact:** Each config file is now self-documenting. Operators understand what they're configuring without cross-referencing other files.

---

## 3. Governance Formalized

### 3.1. Current Capability Ergonomics and Governance Document

**Deliverable:** `docs/architecture/current-capability-ergonomics-and-governance.md`

Codifies:
- Configuration validation chain (schema → range → dependency → duplicate → venue)
- Family addition governance (5-step process)
- Venue addition governance (activation gate ceremony)
- Timeframe addition governance (operational implications table)
- Config change governance (5-step no-code-change path)
- Diagnostic endpoint semantics (phase meanings, idle threshold)
- Query surface ergonomics (URL patterns, required params, error responses)
- Accepted limitations table

### 3.2. ClickHouse and Migrations Entry Principles

**Deliverable:** `docs/architecture/future-clickhouse-and-migrations-entry-principles.md`

Defines 7 core principles:
- P-01: ClickHouse remains optional
- P-02: Migrations are code, not ad-hoc DDL
- P-03: One migration tool, one catalog
- P-04: Schema follows events
- P-05: Writer is a consumer, not a producer
- P-06: Migrations are idempotent
- P-07: No dual-write complexity

Plus: entry sequence (5 ordered steps), pre-conditions checklist (6 items), anti-patterns (7 items), decision authority process.

### 3.3. Migration Catalog Organization Guidelines

**Deliverable:** `docs/architecture/future-migration-catalog-organization-guidelines.md`

Defines:
- Directory layout (`deploy/migrations/`)
- Naming convention (`{NNN}_{action}_{target}.sql`)
- Reserved number ranges
- Required file header structure
- Versioning rules (append-only, immutable after apply, checksum verification)
- Metadata table schema (`_migrations`)
- Lifecycle (create, apply, roll back)
- Retention policy guidelines by table category
- Relationship to `cmd/migrate`

---

## 4. Files Changed

### New Files

| File | Type | Purpose |
|------|------|---------|
| `scripts/utils/lib.sh` | Script | Shared library for all automation scripts |
| `deploy/configs/CONFIG-REFERENCE.md` | Doc | Single-page config reference |
| `docs/architecture/current-capability-ergonomics-and-governance.md` | Doc | Ergonomic and governance codification |
| `docs/architecture/future-clickhouse-and-migrations-entry-principles.md` | Doc | ClickHouse entry principles |
| `docs/architecture/future-migration-catalog-organization-guidelines.md` | Doc | Migration catalog conventions |
| `docs/stages/stage-s141-*.md` | Doc | This report |

### Modified Files

| File | Change |
|------|--------|
| `Makefile` | Added `diag` target and `.PHONY` entry |
| `deploy/configs/gateway.jsonc` | Enhanced inline documentation |
| `deploy/configs/derive.jsonc` | Enhanced inline documentation with family reference |
| `deploy/configs/store.jsonc` | Enhanced inline documentation with cross-reference |
| `deploy/configs/execute.jsonc` | Enhanced inline documentation with venue ceremony |
| `deploy/configs/ingest.jsonc` | Enhanced inline documentation |
| `deploy/configs/configctl.jsonc` | Enhanced inline documentation |

---

## 5. Principles Defined for Future Work

| Principle | Scope | Key Constraint |
|-----------|-------|----------------|
| ClickHouse optional | Runtime | Pipeline must work without CH |
| Migrations as code | Schema | No ad-hoc DDL |
| Single catalog | Organization | `deploy/migrations/`, flat |
| Schema follows events | Design | Event structure = source of truth |
| Idempotent migrations | Safety | Must be re-runnable |
| Writer as consumer | Architecture | Same pattern as store |
| No dual-write | Coordination | Parallel consumers, not coordinated writes |

---

## 6. Limits Maintained

| Guard Rail | Status |
|------------|--------|
| No new features opened | Maintained |
| No horizontal refactoring | Maintained |
| No ClickHouse implementation | Maintained |
| No `cmd/migrate` created | Maintained (principles only) |
| No governance bureaucracy without operational gain | Maintained |

---

## 7. Preparation for S142

S142 should consider:

1. **Script migration to shared library** — Existing scripts (live-pipeline-activate.sh, diag-check.sh, smoke-first-slice.sh) can be incrementally refactored to source `lib.sh` and replace inline boilerplate. This was not done in S141 to keep the blast radius small.

2. **ClickHouse schema design** — If trigger conditions from the S140 trigger matrix are met, S142 could begin Phase A (schema design) per the entry principles. This produces SQL files with zero runtime impact.

3. **OpenAPI spec** — The query surface is stable enough to warrant an OpenAPI spec. This would be a new governance artifact, not a code change.

4. **Config validation dry-run** — A `make config-check` target that validates all JSONC configs without starting services. This is a small tooling improvement with clear payoff.

5. **Raccoon-cli config audit rule** — Add a raccoon-cli check that verifies store's family lists are a superset of derive's. Currently this is a convention; it could be an automated gate.
