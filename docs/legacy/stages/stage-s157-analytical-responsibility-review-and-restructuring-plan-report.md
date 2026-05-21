# Stage S157 — Analytical Responsibility Review and Restructuring Plan Report

> **Status:** Complete
> **Scope:** Review analytical architecture by responsibilities; identify boundary blurs; propose minimal restructuring plan
> **Predecessor:** S156 (Wave A Analytical Readiness Review)
> **Successor:** S158 (Restructuring execution + Wave B preparation)

---

## 1. Executive Summary

S157 conducted a formal responsibility review of the analytical layer across all five domains: schema evolution (migrate), write path (writer), read path (reader), query boundaries (gateway), and observability. The review found the macro-architecture to be sound — failure isolation, lateral optionality, and independent lifecycles are well-established patterns. However, five targeted issues were identified: one boundary blur (reader placement in gateway cmd), one structural gap (distributed schema knowledge), and three S156 preconditions not yet addressed (reader instrumentation, writer config validation, integration testing).

A minimal restructuring plan of five adjustments was produced. No redesign is proposed. All adjustments reinforce existing patterns rather than introducing new abstractions.

---

## 2. Deliverables Produced

| # | Document | Purpose |
|---|----------|---------|
| 1 | `docs/architecture/analytical-responsibility-review-and-restructuring-plan.md` | Main review: responsibility map, boundary analysis, restructuring plan with prioritization |
| 2 | `docs/architecture/analytical-boundaries-writer-reader-gateway-migrate-observability.md` | Canonical boundary definitions, isolation guarantees, data flow diagrams, expansion gates |
| 3 | `docs/architecture/analytical-responsibility-anti-patterns-and-non-goals.md` | Anti-patterns catalog, non-goals, design patterns to protect |
| 4 | `docs/stages/stage-s157-analytical-responsibility-review-and-restructuring-plan-report.md` | This report |

---

## 3. Responsibility Map Summary

### Components Reviewed

| Component | Owner | Boundary Assessment |
|-----------|-------|-------------------|
| Schema evolution | `cmd/migrate` + `internal/migrate` + `deploy/migrations/` | **Clean** — standalone, forward-only, no runtime coupling |
| Write path | `cmd/writer/` (6 files + tests) | **Clean** — lateral, actor-based, failure-isolated |
| Read path | `cmd/gateway/analytical_reader.go` + `internal/application/analyticalclient/` | **Boundary blur** — implementation in composition root |
| Query boundaries | `internal/interfaces/http/{handlers,routes}/analytical.go` | **Mostly clean** — conditional wiring correct; R-02 compliant |
| Observability | `internal/shared/healthz/` + `scripts/` | **Asymmetric** — writer instrumented; reader not |
| ClickHouse adapter | `internal/adapters/clickhouse/` | **Clean** — pure data adapter, dual-use by design |

### Cross-Boundary Dependencies

| From → To | Coupling Type | Strength |
|-----------|--------------|----------|
| Writer → NATS registries | Consumer specs + durable names | Low (interface contract) |
| Writer → ClickHouse adapter | InsertBatch calls | Low (adapter pattern) |
| Writer → Domain types | Event deserialization | Low (read-only) |
| Reader → ClickHouse adapter | Query calls | Low (adapter pattern) |
| Reader → Domain types | Row scanning | Low (read-only) |
| Writer ↔ Reader | ClickHouse tables (shared data) | None (indirect via database) |
| Writer ↔ Gateway | None | None |
| Migrate ↔ Writer/Reader | DDL defines schema | Implicit (deploy-time contract) |

---

## 4. Issues Found

### 4.1 Boundary Blur: Reader in Gateway Cmd

- `analyticalCandleReader` struct with SQL query construction lives in `cmd/gateway/`.
- Should be in `internal/adapters/clickhouse/` alongside the adapter layer.
- Functional impact: none. Architectural impact: expansion friction in Wave B.

### 4.2 Distributed Schema Knowledge

- Column order defined in 3 places: DDL (migrations), mappers (writer), SELECT (reader).
- No compile-time validation; consistency relies on developer discipline.
- Current risk is low (6 stable tables). Wave B expansion multiplies coordination points.

### 4.3 Asymmetric Observability (S156 Precondition 1)

- Writer: 10+ counters per pipeline family; comprehensive logging; `/statusz` and `/diagz`.
- Reader: zero counters; no structured logging; no timing.
- Blocks confident Wave B expansion.

### 4.4 Writer Config Validation (S156 Precondition 3)

- ClickHouse addr validated; family names and batch parameters not validated.
- Typo in family name causes silent pipeline omission.

### 4.5 Integration Test Gap (S156 Precondition 2)

- No automated test covers NATS → writer → ClickHouse → reader → HTTP.
- Manual tests exist (`tests/http/analytical.http`) but are not CI-ready.

---

## 5. Restructuring Plan

Five targeted adjustments, ordered by dependency:

| # | Adjustment | Type | Effort | S156 Alignment |
|---|-----------|------|--------|----------------|
| 1 | Extract reader to `internal/adapters/clickhouse/candle_reader.go` | File move | Low | Boundary clarity |
| 2 | Writer config validation at startup | Code addition | Low | Precondition 3 |
| 3 | Reader instrumentation (logging + tracker counters) | Code addition | Medium | Precondition 1 |
| 4 | Schema contract documentation | Documentation | Low | Boundary clarity |
| 5 | Integration test skeleton | Test addition | Medium | Precondition 2 |

**Execution order:** 1 → 2 → 3 → 4 → 5

---

## 6. Anti-Patterns Cataloged

| ID | Anti-Pattern | Severity | Correction |
|----|-------------|----------|------------|
| AP-01 | Reader implementation in composition root | Medium | Extract to adapter layer |
| AP-02 | Distributed schema knowledge without contract | Medium | Document 3-point coordination rule |
| AP-03 | Asymmetric observability | High | Add reader instrumentation |
| AP-04 | Config validation gap for writer | Medium | Add startup validation |
| AP-05 | No automated integration test | Medium | Create test skeleton |

---

## 7. Design Patterns Reinforced

| ID | Pattern | Status |
|----|---------|--------|
| DP-01 | Lateral optionality | Validated — writer/reader absence has zero operational impact |
| DP-02 | Independent durable consumers | Validated — writer-*/store-* fully isolated |
| DP-03 | Graceful degradation | Validated — 503 on CH unavailable; readiness unaffected |
| DP-04 | Composition root purity | Needs restoration (AP-01) |
| DP-05 | Mechanical transformation | Validated — mappers are purely positional |
| DP-06 | Forward-only migration | Validated — no rollback paths |
| DP-07 | Fail-fast configuration | Needs extension (AP-04) |
| DP-08 | Actor-based lifecycle | Validated — supervisor + restart budget + backoff |

---

## 8. Non-Goals Documented

Explicitly out of scope for the analytical layer at this stage:
- Wave B expansion (new families, readers, endpoints)
- Materialized views
- Dead-letter queue
- Auto-recovery from degraded state
- Distributed tracing / OpenTelemetry
- Shared column-order constants (compile-time schema safety)
- Query-time deduplication
- Dynamic family registration
- Backfill mechanism
- In-code schema versioning

---

## 9. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Analytical architecture reviewed by responsibilities | **Met** — all 6 components mapped |
| Ambiguities and couplings made explicit | **Met** — 5 issues cataloged with severity |
| Restructuring plan is small and clear | **Met** — 5 targeted adjustments, no redesign |
| Canonical design patterns reinforced | **Met** — 8 patterns validated, 2 need restoration |
| Base ready for boundary hardening | **Met** — clear execution path to S158 |

---

## 10. Guard Rails Compliance

| Guard Rail | Status |
|-----------|--------|
| No functional expansion of analytical layer | **Compliant** |
| No Wave B implementation | **Compliant** |
| No broad redesign | **Compliant** — 5 targeted adjustments only |
| No refactors without responsibility basis | **Compliant** — each adjustment traces to a specific anti-pattern |
| Out-of-scope items documented | **Compliant** — 10 non-goals explicit |

---

## 11. Preparation for S158

S158 should execute the restructuring plan in order:

1. **Extract reader** — move `analyticalCandleReader` to adapter layer; update imports.
2. **Writer config validation** — add family and batch parameter validation at startup.
3. **Reader instrumentation** — add logging, timing, and tracker counters to read path.
4. **Schema contract** — produce `analytical-schema-contract-and-coordination-rules.md`.
5. **Integration test** — create minimal automated test for full analytical data path.

After S158 completes, all three S156 preconditions will be satisfied, and the analytical layer will have clean boundaries, symmetric observability, and validated configuration — ready for Wave B family expansion.

---

## 12. Open Debts Inherited from S156

These remain active and are NOT addressed in S157:

**Must (addressed in S158 restructuring):**
- Reader path instrumentation
- Integration test
- Writer config validation

**Should (deferred beyond S158):**
- Jitter in exponential backoff
- Consumer lag monitoring
- Configurable timeouts
- Auto-recovery from degraded
- Insert-level deduplication
- Cold-start bootstrap

**Can defer (no timeline):**
- Push alerts
- Per-family config override
- Schema versioning
- Concurrent migration
- Materialized views
- Dead-letter queue
