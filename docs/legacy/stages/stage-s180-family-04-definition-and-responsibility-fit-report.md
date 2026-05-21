# Stage S180 — Family 04 Definition and Responsibility Fit Report

> Wave B · Analytical Layer Expansion
> Status: **Complete**
> Outcome: Family 04 (`risk_assessments`) formally defined as pattern ceiling test

---

## Executive Summary

Stage S180 formally defines Family 04 of the Wave B analytical expansion. `risk_assessments` is selected as the fourth family — not as routine expansion, but as a deliberate **ceiling test** of the current manual pattern.

Risk assessments push every dimension of the pattern incrementally: 17 DDL columns (highest), 4 JSON columns (new record), a free-text `rationale` column (first in the analytical layer), and the `disposition` enum filter. The write path requires zero changes — migration, mapper, and pipeline are pre-staged. The read path requires ~6 new artifacts following the 9-artifact template exactly.

The central question this family answers: **does the Wave B pattern still scale healthily, or has it reached its ceiling?**

---

## Family 04 Selection Rationale

### Why `risk_assessments`

| Factor | Value |
|--------|-------|
| Analytical layer | 5 of 6 (Risk) — natural sequence after Strategies |
| DDL columns | 17 — highest count, tests column alignment discipline |
| JSON columns | 4 — tests JSON parser scaling (`strategies`, `constraints`, `parameters`, `metadata`) |
| Free-text column | `rationale` — new column type never tested in analytical layer |
| Enum filter | `disposition` (approved/modified/rejected) — follows established pattern |
| Pre-staged artifacts | 3/9 already exist (migration, mapper, pipeline) |
| Domain fit | Risk follows strategies in the decision chain |

### Why Not Alternatives

- **Executions** (layer 6): Simpler payload, lower ceiling-test value. Better as Family 05 if pattern holds.
- **Multiple families**: Violates guard rail — one family at a time, each with its own gate.

---

## Deliverables Produced

| # | Document | Purpose |
|---|----------|---------|
| 1 | `docs/architecture/family-04-definition-and-responsibility-fit.md` | Family definition, responsibility map by track, architectural fit |
| 2 | `docs/architecture/family-04-schema-writer-reader-gateway-scope.md` | Detailed technical scope: DDL, mapper, reader, use case, handler, routes, gateway wiring |
| 3 | `docs/architecture/family-04-success-criteria-risks-and-out-of-scope.md` | 14 hard criteria, 6 ceiling-test criteria, 5 technical risks, explicit non-goals |
| 4 | This report | Stage completion record |

---

## Analytical Contract Summary

### Endpoint

```
GET /analytical/risk/history
    ?type=position_exposure
    &source=binance
    &symbol=BTCUSD
    &timeframe=60
    [&disposition=approved]
    [&since=1710000000]
    [&until=1710100000]
    [&limit=100]
```

### Response

```json
{
  "risk_assessments": [{
    "type": "position_exposure",
    "source": "binance",
    "symbol": "BTCUSD",
    "timeframe": 60,
    "disposition": "approved",
    "confidence": "0.82",
    "strategies": [{"type": "...", "direction": "...", "confidence": "...", "timeframe": 60}],
    "constraints": {"max_position_size": "0.1", "max_exposure": "1000.00"},
    "rationale": "Position within exposure limits",
    "parameters": {},
    "metadata": {},
    "final": true,
    "timestamp": "2026-03-20T12:00:00.000Z"
  }],
  "source": "clickhouse",
  "meta": {"query_ms": 12, "row_count": 1}
}
```

---

## Responsibility Distribution

| Track | Pre-Staged | New in S181 |
|-------|-----------|-------------|
| Schema / Migrate | Migration DDL | — |
| Writer | Mapper + pipeline + tests | — |
| Reader (Adapter) | — | `risk_reader.go` + 2 new JSON parsers + tests |
| Application | — | Contracts + use case + tests |
| HTTP | — | Handler method + route + tests |
| Gateway | — | Reader factory + composition wiring |
| Smoke/Integration | — | Smoke entry + HTTP test queries |

**Ratio:** 3/9 artifacts pre-staged, 6/9 new in S181.

---

## Ceiling Test Framework

Family 04 is explicitly a ceiling test. After implementation, these measurements determine pattern health:

| Measurement | Healthy | Concerning | Critical |
|-------------|---------|------------|----------|
| New frictions | 0-1 | 2 | >2 → mandatory hardening |
| Handler file size | <550 lines | 550-600 | >600 → DEF-C3 triggers |
| Reader file size | <150 lines | 150-180 | >180 → pattern stress |
| Creative decisions | 0 | 1 | >1 → pattern limit |
| JSON parser count (total) | ≤6 | 7-8 | >8 → codegen pressure |

**Gate output:** A clear signal — "the pattern can / cannot sustain further families without structural intervention."

---

## Risk Summary

| Category | Count | Highest Severity |
|----------|-------|-----------------|
| Technical risks | 5 | TR-03 (column misalignment) — mitigated by tests |
| Process risks | 3 | PR-01 (scope creep to executions) — guarded |
| Pattern risks | 4 | PC-01 (>2 frictions) — stop condition defined |

No risks require pre-mitigation before S181. All are addressed through the existing test and review pattern.

---

## Out of Scope (Explicit)

- Family 05 (executions): requires its own gate
- Cross-family queries: deliberate non-goal
- Codegen implementation: deferred to DEF-C1 trigger
- Handler file split: measured but not executed
- Write-path changes: hard constraint
- `rationale` full-text search: not in analytical scope
- Alerting, NATS lag monitoring, backfill: operational gaps, not Family 04 scope

---

## Guard Rails Compliance

| Guard Rail | Status |
|-----------|--------|
| No more than one family | Compliant — only `risk_assessments` |
| No more than one endpoint | Compliant — only `/analytical/risk/history` |
| No Family 05 anticipation | Compliant — explicitly deferred |
| No horizontal redesign | Compliant — additive changes only |
| Out-of-scope documented | Compliant — 15 explicit non-goals |

---

## Preparation for S181

S181 implements Family 04. The following is ready:

**Pre-staged (no action needed):**
- `deploy/migrations/005_create_risk_assessments.sql`
- `mapRiskRow()` in `cmd/writer/mappers.go` with tests
- Pipeline entry in `cmd/writer/pipeline.go`

**Implementation order for S181:**
1. Reader adapter (`risk_reader.go` + tests)
2. Contracts (extend `contracts.go`)
3. Use case (`get_risk_history.go` + tests)
4. Handler method (extend `analytical.go` + tests)
5. Routes (extend routes `analytical.go`)
6. Gateway wiring (extend `compose.go` + `analytical_reader.go`)
7. Smoke test (extend `smoke-analytical-e2e.sh`)
8. HTTP test queries (extend `analytical.http`)
9. Validation and friction capture

**Estimated new artifact count:** ~6 files, ~500 lines of implementation, ~400 lines of tests.

---

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Family 04 clearly defined and justified | Done — `risk_assessments` with ceiling-test rationale |
| Scope small and controlled | Done — single endpoint, single family, 9-artifact template |
| Responsibilities per track explicit | Done — responsibility map in definition document |
| Success criteria objective | Done — 14 hard + 6 ceiling-test + 3 observability criteria |
| Risks documented | Done — 12 risks across 3 categories |
| Ready for disciplined S181 implementation | Done — pre-staged artifacts identified, implementation order defined |
