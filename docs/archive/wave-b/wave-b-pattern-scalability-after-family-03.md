# Wave B Pattern Scalability After Family 03

## Purpose

Assess the scalability trajectory of the Wave B 9-artifact expansion pattern after 4 families (candles baseline + 3 Wave B expansions). Determine whether the pattern can sustain further expansion or has reached structural limits.

---

## 1. Pattern Metrics at Family 03 Completion

### Artifact Growth Per Family

| Artifact | Files per family | Lines per family | Growth model |
|----------|-----------------|------------------|-------------|
| Migration DDL | 1 | ~30 | Constant |
| Writer mapper | 0 (pre-staged) | 0 | Already complete for all 6 layers |
| Writer pipeline | 0 (pre-staged) | 0 | Already complete for all 6 layers |
| ClickHouse reader | 1 | ~138 avg | Linear, bounded |
| Use case | 1 | ~60 avg | Linear, bounded |
| Contracts (request/reply) | 0 (additive to shared file) | ~25 added | Linear, bounded |
| HTTP handler method | 0 (additive to shared file) | ~80 added | Linear, bounded |
| Route registration | 0 (additive to shared file) | ~8 added | Linear, bounded |
| Tests | 1-2 | ~100-150 avg | Linear, bounded |

**Total per family:** 3-4 new files, ~450-500 new lines, 5-7 modified files.

### Cumulative Growth

| Metric | Family 01 | Family 02 | Family 03 | Projected F-04 |
|--------|-----------|-----------|-----------|----------------|
| Readers | 2 | 3 | 4 | 5 |
| Handler methods | 2 | 3 | 4 | 5 |
| Use cases | 2 | 3 | 4 | 5 |
| Contracts structs | 4 | 6 | 8 | 10 |
| Handler file (lines) | ~210 | ~310 | ~417 | ~500 |
| Smoke test (lines) | ~350 | ~480 | ~570 | ~650 |
| Compose DI block (lines) | ~6 | ~10 | ~14 | ~18 |
| JSON column types proven | 1 | 2 | 3 | 4 |
| Migrations | 3 | 4 | 5 | 6 |

### Growth Characteristics

- **Linear, bounded increments:** Each family adds a constant-size increment (~450-500 lines).
- **No exponential growth:** No cross-family dependencies, no combinatorial explosion.
- **No architectural drift:** The pattern shape is identical across all 4 families.
- **No correctness regressions:** Zero bugs introduced by the copy-paste-modify workflow.

---

## 2. Pattern Strengths Confirmed

### S-1: Write Path Stability

The writer service (mappers, inserter, pipeline, supervisor) has required **zero changes** across all 4 family expansions. All 6 mappers were pre-staged. The write path is genuinely future-proof for the current domain model.

### S-2: Struct-Based DI Scalability

The H-1 refactor (struct-based `AnalyticalHandlerDeps` / `AnalyticalFamilyDeps`) eliminated constructor churn. Adding a new family requires:
- 1 new field in `AnalyticalFamilyDeps` (routes)
- 1 new field in `AnalyticalHandlerDeps` (handler)
- 2 lines in compose.go (reader + use case)

No existing code is modified. No signatures change.

### S-3: Smoke Test Reusability

The `validate_analytical_family()` helper function (extracted in S172 hardening) absorbs new families with a single function call (~5 lines per family). The 570-line smoke test includes 7 phases of infrastructure, migration, writer, data, read-path, error-handling, and observability validation.

### S-4: Observability Parity Is Free

Each family automatically gets identical instrumentation: `Server-Timing` headers, `query_ms` metadata, `source=clickhouse` tagging, error counters, and health integration. Zero per-family effort.

### S-5: ClickHouse Optionality Preserved

All 4 families degrade gracefully when ClickHouse is unavailable. The `if chClient != nil` guard in compose.go ensures the operational pipeline is unaffected. This invariant has held across 4 expansions without any code to maintain it.

### S-6: CI Integration Operational

The `smoke-analytical` CI job validates all 4 families in the GitHub Actions pipeline. Full E2E from NATS through writer to ClickHouse to reader to HTTP response.

---

## 3. Pattern Pressures Identified

### P-1: Mechanical Duplication (~80% per artifact)

**Severity: Medium. Trajectory: Stable.**

Each reader, handler method, and use case is ~80% identical to its siblings. At 4 families this is ~800 lines of duplicated structure. At 6 families it would be ~1200 lines.

The duplication is:
- **Predictable** — the template is well-understood.
- **Reviewable** — differences are domain-specific (column names, filters, JSON parsers).
- **Correct** — no bugs introduced by duplication across 4 families.

**Threshold:** Codegen becomes cost-effective at 6+ families. Before that, the template maintenance overhead exceeds the duplication cost.

### P-2: Handler File Size

**Severity: Low. Trajectory: Linear.**

The handler file grows by ~80-100 lines per family. At 417 lines (4 families), it's comfortable. At 6 families (~600 lines), it's acceptable. At 8+ families, it would benefit from splitting or generation.

### P-3: Schema Coherence Is Review-Enforced

**Severity: Medium. Trajectory: Stable.**

DDL ↔ mapper ↔ reader column alignment is verified by review and smoke tests, not by compile-time checks. At 4 families (60+ columns across all tables), this has produced zero coherence failures. The risk grows with table count, but the 9-artifact checklist and E2E smoke provide adequate safety nets through ~8 families.

### P-4: Filter Case-Sensitivity

**Severity: Low. Trajectory: Stable.**

Optional domain filters (`type`, `outcome`, `direction`, projected `disposition`) pass user input directly to ClickHouse WHERE clauses without case normalization. This is consistent across all families and has not caused operational issues.

### P-5: No Pagination

**Severity: Low. Trajectory: Stable.**

All queries are bounded by `limit=500`. At current data volumes, this is sufficient. Pagination becomes relevant only when analytical consumers need historical scans beyond 500 rows.

---

## 4. Scalability Projections

### Sustainable Without Changes (Families 04-05)

The current pattern can absorb 1-2 more families without any structural changes. Evidence:
- Linear growth, bounded increments.
- No correctness regressions through 4 families.
- CI validates all families E2E.
- Struct DI eliminates constructor churn.
- Smoke helper absorbs new families mechanically.

### Recommended Changes at Family 06 Boundary

| Change | Why | Effort |
|--------|-----|--------|
| Codegen for readers + handlers + use cases | ~1200 lines of duplication at 6 families | 2-3 days |
| Schema coherence compile-time check | 72+ columns across 6 tables | 1-2 days |
| Handler file split or generation | ~600+ lines in single file | 0.5-1 day |

### Not Recommended Until Family 08+

| Change | Why Not Yet |
|--------|------------|
| Generic reader abstraction | Premature — domain differences are small but real |
| Plugin/framework system | Over-engineering — 6-8 families don't justify framework cost |
| Materialized views | No aggregation queries exist yet |
| Cross-family queries | No consumer requires them |
| Pagination | Data volumes don't warrant it |

---

## 5. Conclusion

The Wave B pattern is **healthy and scalable through Family 05**. The 9-artifact expansion pattern produces predictable, bounded, correct results. No structural limits have been reached. The only activated pressure (mechanical duplication) has a well-understood resolution (codegen) with a clear activation point (Family 06).

The pattern's greatest strength is its simplicity: each family is an independent, additive unit that doesn't affect existing families. This property holds because:
- No cross-family dependencies exist.
- The write path is pre-staged and stable.
- The DI pattern is additive-only.
- The smoke test helper absorbs new families mechanically.

**The pattern does not need redesign. It needs codegen at the right time.**
