# Wave B Pattern Scalability After Family 04

## Purpose

Assess whether the Wave B expansion pattern remains sustainable after Family 04 (Risk Assessments) — the highest-complexity family expanded to date (17 DDL columns, 4 JSON columns, 1 free-text column, new parser shape).

---

## Cumulative Metrics

### Growth Trajectory

| Metric | F-01 (Candles) | F-02 (Decisions) | F-03 (Strategies) | F-04 (Risk) | Projected F-05 |
|--------|-------|-------|-------|------|------|
| Readers | 2 | 3 | 4 | 5 | 6 |
| Handler methods | 2 | 3 | 4 | 5 | 6 |
| Use cases | 2 | 3 | 4 | 5 | 6 |
| Contract structs | 4 | 6 | 8 | 10 | 12 |
| Handler file (lines) | ~210 | ~310 | ~417 | ~515 | ~595–615 |
| Smoke test (lines) | ~350 | ~480 | ~570 | ~606–750 | ~635–780 |
| Compose DI (lines) | ~6 | ~10 | ~14 | ~18 | ~22 |
| JSON column types | 1 | 2 | 3 | 4 | 4–5 |
| JSON parsers | 2 | 3 | 4 | 6 | 6–7 |
| DDL columns (max) | 16 | 14 | 15 | 17 | TBD |
| Total analytical LOC | ~1,200 | ~1,900 | ~2,600 | ~3,348 | ~3,943 |
| Total test LOC | ~700 | ~1,100 | ~1,500 | ~1,816 | ~2,523 |

### Growth Characteristics

1. **Linear, bounded increments**: Each family adds ~595 implementation + ~707 test LOC.
2. **No exponential coupling**: Zero cross-reader, cross-handler, or cross-use-case dependencies.
3. **No architectural drift**: Pattern shape identical across all 5 families.
4. **Zero correctness regressions**: No bugs from copy-paste-modify across 5 expansions.
5. **Write path immutable**: Zero changes across 5 consecutive family expansions.

---

## Pattern Strengths (Confirmed and Extended)

### S-1: Write Path Stability (STRONGEST SIGNAL)
- Zero changes across 5 family expansions.
- All 7 mappers pre-staged (including execution).
- Pipeline config, NATS consumers, inserter — all immutable since initial implementation.
- **Conclusion**: Write path architecture was correct. Multi-family service design validated.

### S-2: Struct-Based DI Scalability
- H-1 refactor (S172) eliminated constructor churn permanently.
- Adding Family 04 required only: 1 new field in `AnalyticalFamilyDeps`, 1 in `AnalyticalHandlerDeps`, 2 lines in compose.
- Zero existing code modified, zero signatures changed.
- **Conclusion**: Struct DI pattern scales to any reasonable number of families.

### S-3: Smoke Test Absorption
- `validate_analytical_family()` helper absorbs new families with ~5–8 lines of invocation code.
- Error handling validation shared across all families.
- **Conclusion**: Test infrastructure scales well, though total file size is escalating.

### S-4: Observability Parity Is Free
- Every family automatically receives: Server-Timing headers, `query_ms` metadata, source tagging, structured logging.
- Zero per-family effort.
- **Conclusion**: Observability cost per family = 0.

### S-5: ClickHouse Optionality Preserved
- All 5 families degrade gracefully when ClickHouse unavailable.
- `if chClient != nil` guard ensures operational pipeline unaffected.
- 503 responses consistent across all families.
- **Conclusion**: Optionality model is robust and proven.

### S-6: JSON Parser Reuse (NEW — Confirmed at Scale)
- `ParseMetadataJSON` reused 6 times across 4 families from 1 function.
- Parser library handles: map, slice, struct-of-slices, struct targets.
- New parser shapes absorbed mechanically.
- **Conclusion**: JSON column scaling is a solved problem.

### S-7: Schema Coherence Under Pressure (NEW — Ceiling Test)
- 17 DDL columns verified manually without tooling pressure.
- Zero coherence failures across 5 families with diverse column types.
- **Conclusion**: Manual verification remains reliable at current scale (6 tables, ~75 columns).

---

## Pattern Pressures (Updated After Family 04)

### P-1: Mechanical Duplication — MEDIUM SEVERITY, ESCALATING

| Family Count | Duplicated LOC | Manual Cost | Codegen Break-Even |
|---|---|---|---|
| 3 | ~500 | Low | Not justified |
| 4 | ~650 | Low-Medium | Not justified |
| 5 (current) | ~800 | Medium | Approaching |
| 6 | ~1,000 | High | **Justified** |
| 8 | ~1,300 | Very High | **Mandatory** |

- Each reader: ~143 LOC, 80% identical boilerplate.
- Each handler method: ~100 LOC, 85% identical param parsing.
- Each use case: ~128 LOC, 70% identical validation.
- **Threshold**: Codegen investment (2–3 days) pays off at Family 06.

### P-2: Handler File Size — MEDIUM SEVERITY, APPROACHING THRESHOLD

| Family | Handler LOC | Zone |
|--------|-------------|------|
| F-01 | ~210 | Healthy |
| F-02 | ~310 | Healthy |
| F-03 | ~417 | Healthy |
| F-04 | ~515 | Healthy (upper) |
| F-05 (projected) | ~595–615 | **Concerning** |
| F-06 (projected) | ~695–715 | **Critical** |

- Each method adds ~80–100 lines.
- A `parseAnalyticalParams()` helper could reduce per-method overhead from ~90 to ~30 lines.
- **Threshold**: Split or extract mandatory before Family 06.

### P-3: Schema Coherence Verification — LOW SEVERITY, STABLE

- 6 tables, ~75+ DDL columns at Family 05.
- Zero coherence failures to date.
- Review-enforced verification remains reliable.
- **Threshold**: Compile-time checks at ~12 tables / 100+ columns.

### P-4: Smoke Test Size — LOW-MEDIUM SEVERITY, LINEAR

- Growing ~28–30 lines per family.
- `validate_analytical_family()` helper keeps additions mechanical.
- Total file manageable through Family 07.
- **Threshold**: Per-family file split or function extraction at Family 08+.

### P-5: JSON Parser Proliferation — LOW SEVERITY, AT LIMIT

- 6 parsers total, each ~10 lines, all identical shape.
- Family 05 may not require new parsers.
- Generic `parseJSON[T any]` viable but premature.
- **Threshold**: Generic abstraction at parser count >8.

---

## Scalability Projection

### Sustainable Without Changes (Family 05)

Family 05 (Executions) can proceed under the current pattern because:

1. Linear growth trajectory — no exponential signals.
2. Handler at ~595–615 lines — concerning but not critical.
3. Codegen duplication at ~800 lines — acceptable for one more family.
4. Zero blocking frictions from Family 04.
5. Write path, DI, observability, optionality — all stable.
6. Pre-staged artifacts exist (migration 006, mapper, pipeline config).

### Mandatory Changes at Family 06 Boundary

| Change | Effort | Why |
|--------|--------|-----|
| Codegen for readers/handlers/use cases | 2–3 days | ~1,000+ LOC duplication, template cheaper than copy-paste |
| Handler file split or param extraction | 0.5–1 day | File exceeds 600–700 lines |
| Smoke test restructuring (optional) | 0.5 day | File approaching 800 lines |

### Not Recommended Until Family 08+

| Item | Why Deferred |
|------|-------------|
| Generic reader abstraction | Premature — readers are simple and self-contained |
| Schema coherence tooling | Under 12 tables — manual verification reliable |
| Cross-family queries | No demand, no use case |
| Pagination beyond 500 | Data volumes don't warrant |
| Plugin/framework system | Over-engineering |
| Generic parser abstraction | Under 8 parsers — each trivial |

---

## What Family 04 Proved About the Pattern

### Ceiling Tests Passed

1. **4 JSON columns**: Parsed independently, no cross-dependency, no degradation.
2. **Free-text column**: Simplest column type, zero new patterns.
3. **17 DDL columns**: Highest count, verified without tooling pressure.
4. **Struct-target parser**: New shape absorbed mechanically.
5. **Zero new frictions**: Pattern health at peak after hardest family.
6. **Zero creative decisions**: Fully mechanical, strategy reader as template.

### What the Pattern Does NOT Yet Prove

1. Handler scales beyond 600 lines without refactoring.
2. Smoke test scales beyond 6 families without restructuring.
3. Codegen can replace manual expansion efficiently.
4. Cross-family queries are feasible.
5. Pattern can absorb families with non-standard schema shapes (e.g., self-referencing, time-series aggregations).

---

## Conclusion

**The Wave B pattern is healthy and scalable through Family 05, with a hard boundary at Family 06.**

The pattern has been applied 5 times with zero regressions, zero blocking frictions, and zero architectural drift. Growth is linear and bounded. The only structural pressures (duplication, handler size) have well-understood resolutions (codegen, file split) with clear activation points.

Family 05 represents the final expansion under the current manual pattern. A codegen/hardening tranche is mandatory before Family 06.
