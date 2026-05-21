# Family 04 — Success Criteria, Risks, and Out of Scope

> Stage: S180 · Wave B · `risk_assessments`
> Status: Defined

---

## 1. Success Criteria

### 1.1 Hard Criteria (Must Pass)

| ID | Criterion | Verification |
|----|-----------|--------------|
| SC-01 | All 9 artifacts implemented following Wave B pattern v2 | Artifact checklist review |
| SC-02 | Zero write-path changes | Diff of `cmd/writer/` shows no modifications |
| SC-03 | 17-column alignment across DDL, mapper, reader, handler | Column count assertions in tests |
| SC-04 | 4 JSON columns round-trip correctly (strategies, constraints, parameters, metadata) | Unit tests with representative payloads |
| SC-05 | Free-text `rationale` column round-trips without encoding issues | Unit test with unicode, empty string, long text |
| SC-06 | `disposition` enum filter works identically to `outcome` and `direction` | Query builder test + handler test |
| SC-07 | `GET /analytical/risk/history` returns correct response shape | HTTP integration test |
| SC-08 | Server-Timing header present with `total` and `query` durations | Handler test |
| SC-09 | Graceful degradation: 503 when ClickHouse unavailable | Nil handler test |
| SC-10 | All existing analytical endpoints unaffected | Regression: existing smoke tests pass |
| SC-11 | Struct DI pattern: additive only, no constructor signature changes | Code review of deps structs |
| SC-12 | Smoke test includes `risk_assessments` family validation | `smoke-analytical-e2e.sh` updated |
| SC-13 | `go build` succeeds for gateway, writer, and migrate | CI gate |
| SC-14 | All unit tests pass (existing + new) | `go test ./...` |

### 1.2 Ceiling Test Criteria (Pattern Health)

These criteria specifically measure whether the Wave B pattern remains healthy:

| ID | Criterion | Threshold | Action if Failed |
|----|-----------|-----------|------------------|
| CT-01 | Handler file size | ≤600 lines | Confirm DEF-C3 (handler split) triggers at Family 05 |
| CT-02 | New frictions introduced | ≤2 | >2 triggers mandatory hardening before Family 05 |
| CT-03 | Implementation remains mechanical | No creative decisions required | Pattern stress documented |
| CT-04 | JSON parser proliferation | ≤6 total parser functions in reader package | Codegen pressure measured |
| CT-05 | Reader file remains self-contained | No cross-reader dependencies | Coupling check |
| CT-06 | Test count per family remains proportional | 27±5 tests for Family 04 | Over/under indicates pattern drift |

### 1.3 Observability Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| OB-01 | `QueryMeta` in response with `query_ms` and `row_count` | Response shape validation |
| OB-02 | Structured error logging on ClickHouse failures | Log review |
| OB-03 | Empty results return 200 with empty array, not 404 | Handler test |

---

## 2. Risks

### 2.1 Technical Risks

| ID | Risk | Likelihood | Impact | Mitigation |
|----|------|-----------|--------|------------|
| TR-01 | 4 JSON columns degrade reader clarity | Medium | Low | Each parser is isolated; test per parser; no shared state |
| TR-02 | `rationale` free-text contains characters that break JSON response | Low | Medium | `rationale` is a string field in the JSON response; Go's `encoding/json` handles escaping; test with unicode |
| TR-03 | Column misalignment between DDL (17 cols) and mapper/reader | Low | High | Existing column-count test in mapper; add equivalent in reader |
| TR-04 | `ParseConstraintsJSON` is a new parser shape (struct, not map/array) | Low | Low | Trivial — same `json.Unmarshal` pattern, just different target type |
| TR-05 | Handler file approaching 600-line threshold | Expected | Low | This is a measurement, not a failure — confirms DEF-C3 timing |

### 2.2 Process Risks

| ID | Risk | Likelihood | Impact | Mitigation |
|----|------|-----------|--------|------------|
| PR-01 | Scope creep: temptation to also implement executions | Low | Medium | Guard rail: one family only; executions explicitly out of scope |
| PR-02 | Over-engineering: adding validation for `disposition` enum values | Low | Low | Pattern rule: enum filters are passthrough, no validation |
| PR-03 | Premature optimization: adding indexes or materialized views | Low | Medium | Out of scope — same ClickHouse engine settings as all families |

### 2.3 Pattern Risks (Ceiling Test)

| ID | Risk | What It Would Mean | Action |
|----|------|--------------------|--------|
| PC-01 | >2 new frictions in this family | Pattern is degrading under JSON/column load | Mandatory hardening pause before Family 05 |
| PC-02 | Handler file exceeds 600 lines | Monolith handler confirmed unsustainable | DEF-C3 must execute before Family 05 |
| PC-03 | Implementation requires non-mechanical decisions | Pattern has hit its expressive limit | Document and evaluate codegen acceleration |
| PC-04 | Test maintenance cost noticeably higher than Family 03 | Scaling cost is super-linear | Friction gate triggered |

---

## 3. Out of Scope

### 3.1 Explicit Non-Goals

| Item | Reason |
|------|--------|
| Family 05 (executions) definition or implementation | One family at a time; requires its own gate |
| Cross-family queries | Deliberate architectural non-goal |
| Response pagination beyond `limit=500` | No demand; deferred |
| `disposition` enum validation in handler | Pattern rule: enum filters are passthrough |
| JSON content verification in smoke tests | Unit tests provide coverage |
| Materialized views or secondary indexes | Same ClickHouse engine settings as all families |
| Codegen implementation | Deferred to DEF-C1 trigger (mandatory before Family 06) |
| Handler file split | Measured but not executed unless threshold exceeded |
| Write-path changes | Hard constraint of Wave B pattern |
| `rationale` full-text search | ClickHouse not configured for FTS; out of analytical scope |
| Alerting on reader errors | Operational gap acknowledged (DEF-NC) but not addressed here |
| NATS consumer lag monitoring | Existing operational gap; not in scope for Family 04 |
| Backfill or historical data migration | Analytical layer is append-only, forward-looking |
| Gateway config changes | ClickHouse connection already configured |
| Writer config changes | Pipeline already pre-staged |

### 3.2 Deferred Decisions

| Decision | Deferred Until | Notes |
|----------|---------------|-------|
| Whether to implement Family 05 | Post-Family-04 gate (S183+) | Depends on ceiling test results |
| Codegen for readers/handlers | DEF-C1 trigger (before Family 06) | May accelerate after ceiling test findings |
| Handler file restructuring | DEF-C3 trigger (file > 600 lines) | Measured during Family 04 |
| Wave B continuation vs. pause | Post-Family-04 readiness review | Central question of this iteration |

---

## 4. Friction Budget

Family 04 operates under a **strict friction budget of 2 new frictions**.

### 4.1 What Counts as a Friction

A friction is any implementation obstacle that:
- Requires a workaround or deviation from the established pattern
- Introduces a new category of technical debt
- Forces a decision not covered by the existing 9-artifact template
- Reveals a scaling limit not previously observed

### 4.2 What Does NOT Count as a Friction

- Expected mechanical effort (writing the reader, handler, tests)
- Handler file size approaching threshold (this is a measurement, not a friction)
- Adding a new JSON parser function (follows established pattern)
- `rationale` being a plain string scan (simpler than JSON, not a friction)

### 4.3 Friction Response Protocol

| Friction Count | Response |
|----------------|----------|
| 0 | Pattern confirmed healthy; proceed to Family 05 gate |
| 1 | Document and carry forward; pattern still viable |
| 2 | Document with care; pattern at limit; Family 05 gate must evaluate |
| >2 | **Stop.** Mandatory hardening before Family 05. Wave B may need pattern revision |

---

## 5. How Family 04 Evaluates the Pattern Ceiling

### 5.1 Measurement Points

After Family 04 implementation, the following measurements determine pattern health:

| Measurement | Healthy | Concerning | Critical |
|-------------|---------|------------|----------|
| New frictions | 0-1 | 2 | >2 |
| Handler file size | <550 lines | 550-600 lines | >600 lines |
| Reader file size | <150 lines | 150-180 lines | >180 lines |
| Implementation time vs Family 03 | Similar | 1.5x | >2x |
| Test count vs Family 03 | ±5 | ±10 | Pattern drift |
| JSON parser count (total) | ≤6 | 7-8 | >8 |
| Creative decisions required | 0 | 1 | >1 |

### 5.2 Interpretation

- **All healthy:** Pattern confirmed for at least 2 more families. Codegen still deferred to DEF-C1 trigger.
- **Mix of healthy and concerning:** Pattern viable but showing age. Family 05 proceeds with caution; codegen evaluation moves up.
- **Any critical:** Pattern ceiling reached. Mandatory hardening or codegen before Family 05.

### 5.3 Gate Output

Family 04's completion produces a clear signal for the Wave B continuation decision:

> "The Wave B manual expansion pattern {can / cannot} sustain further families without structural intervention."

This is the primary deliverable of Family 04 as a ceiling test.
