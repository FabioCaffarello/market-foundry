# Family 04 — Definition and Responsibility Fit

> Stage: S180 · Wave B · Analytical Layer Expansion
> Status: Defined
> Family: `risk_assessments` (Risk Layer)

---

## 1. Family 04 Selection: `risk_assessments`

### 1.1 Why `risk_assessments` Is the Right Choice Now

`risk_assessments` is chosen as Family 04 because it is the strongest available candidate for testing the ceiling of the current Wave B pattern. The selection is deliberate — not expansion for its own sake, but a controlled stress test.

**Arguments in favor:**

| Factor | Detail |
|--------|--------|
| Highest column count | 17 DDL columns (vs 15 for strategies, 14 for decisions) |
| Most JSON columns | 4 (`strategies`, `constraints`, `parameters`, `metadata`) — new record |
| First free-text column | `rationale` introduces a column type never tested in the analytical layer |
| Established enum filter | `disposition` (approved/modified/rejected) follows the pattern from `outcome` and `direction` |
| Pre-staged artifacts | Migration (`005`), mapper (`mapRiskRow`), pipeline config, NATS consumer — all exist |
| Clear domain semantics | Risk is layer 5 of 6, naturally follows strategies in the decision chain |
| Incremental complexity | Pushes every dimension slightly without requiring redesign |

**Arguments against alternatives:**

- `executions` (layer 6) has simpler payload but depends semantically on risk assessments existing first.
- Skipping to executions would leave a gap in the analytical chain and miss the opportunity to stress-test JSON column scaling.

### 1.2 What This Family Must Prove

Family 04 is explicitly positioned as a **pattern ceiling test**. The central question:

> Does the Wave B pattern (9-artifact template, struct DI, manual reader/handler/use-case) still scale healthily at 4 JSON columns, 17 DDL columns, and a free-text field?

Specific proof points:

1. **JSON column scaling**: 4 JSON columns in a single reader must parse without degrading test clarity or maintenance cost.
2. **Free-text viability**: The `rationale` column must round-trip through writer → ClickHouse → reader → HTTP without encoding issues, truncation, or special-case handling.
3. **Mechanical repeatability**: Implementation must remain mechanical — no creative decisions, no structural changes to the pattern.
4. **Friction budget**: ≤2 new frictions allowed. More than 2 triggers mandatory hardening before Family 05.
5. **Handler file size**: The analytical handler file must remain under ~600 lines. Exceeding this confirms DEF-C3 (handler split) must happen before Family 06.

---

## 2. Responsibility Map by Track

### 2.1 Schema / Migrate

| Responsibility | Status | Notes |
|---------------|--------|-------|
| DDL definition | Done | `005_create_risk_assessments.sql` pre-staged |
| Column alignment | Verify | 17 columns must match mapper, reader, and handler exactly |
| Partition strategy | Done | `toYYYYMM(timestamp)` — consistent with all families |
| TTL policy | Done | 90-day retention — consistent |
| ORDER BY key | Done | `(source, symbol, timeframe, type, timestamp)` — consistent |
| Migration runner | Done | `cmd/migrate` applies this migration automatically |

**No schema changes required.** The migration is pre-staged and validated.

### 2.2 Writer

| Responsibility | Status | Notes |
|---------------|--------|-------|
| Mapper function | Done | `mapRiskRow` in `cmd/writer/mappers.go` |
| Mapper tests | Done | Column count + domain field tests exist |
| Pipeline entry | Done | `position_exposure` family configured in `pipeline.go` |
| NATS consumer | Done | `WriterPositionExposureRiskConsumer()` registered |
| JSON marshaling | Done | 4 columns use `marshalJSON()` helper |
| `rationale` handling | Done | Direct string pass-through (no JSON encoding) |

**Zero write-path changes required.** This is a hard constraint of the Wave B pattern.

### 2.3 Reader (Adapter)

| Responsibility | Status | Scope |
|---------------|--------|-------|
| `RiskReader` struct | New | ~138 lines, follows `StrategyReader` pattern |
| `QueryRiskHistory()` method | New | SELECT 13 domain columns + optional `disposition` filter |
| `BuildRiskQuery()` function | New | Exported for testing |
| `ParseStrategyInputsJSON()` | New | Parser for `[]risk.StrategyInput` array |
| `ParseConstraintsJSON()` | New | Parser for `risk.Constraints` struct |
| `ParseMetadataJSON()` | Reuse | Already exists — for `parameters` and `metadata` columns |
| `FormatFloat()` | Reuse | Already exists — for `confidence` column |
| Row scanning | New | 13 domain fields scanned in DDL column order |

**Key decision:** `rationale` is scanned as a plain `string` — no JSON parsing, no special handling. This is the simplest possible treatment and matches the writer's direct pass-through.

### 2.4 Gateway (Composition)

| Responsibility | Status | Scope |
|---------------|--------|-------|
| `newAnalyticalRiskReader()` | New | Factory function in `analytical_reader.go` |
| Wire `GetRiskHistory` | New | Add to `buildRouteDependencies` in `compose.go` |
| ClickHouse optionality | Existing | Risk reader only created if ClickHouse client available |
| Graceful degradation | Existing | 503 if handler nil — pattern already established |

### 2.5 Application Layer (Use Case + Contracts)

| Responsibility | Status | Scope |
|---------------|--------|-------|
| `RiskHistoryQuery` contract | New | Add to `contracts.go` |
| `RiskHistoryReply` contract | New | Add to `contracts.go` |
| `RiskReader` interface | New | Add to `contracts.go` |
| `GetRiskHistoryUseCase` | New | ~60 lines, follows exact `GetStrategyHistoryUseCase` pattern |
| Validation rules | New | Same rules: required fields, bounds, range, limit cap |
| `disposition` filter | New | Passthrough — no validation, empty = no filter |
| `QueryMeta` timing | Existing | Reuse `QueryMeta` struct |

### 2.6 HTTP Layer (Handler + Routes)

| Responsibility | Status | Scope |
|---------------|--------|-------|
| `GetRiskHistory` handler method | New | ~80 lines in `analytical.go` |
| `getRiskHistory` interface | New | Add to handler deps |
| Route registration | New | `/analytical/risk/history` GET |
| `HasAny()` update | New | Add `GetRiskHistory` check |
| Parameter parsing | Reuse | `parseAnalyticalKeyParams()` for source/symbol/timeframe |
| `disposition` query param | New | Optional filter — empty if not set |
| Server-Timing header | Existing | Same pattern as all families |

### 2.7 Observability & Operability

| Responsibility | Status | Scope |
|---------------|--------|-------|
| Structured logging | Existing | Reader/use-case log errors via `slog.Logger` |
| Server-Timing header | Existing | Exposes `total` and `query` duration per request |
| `QueryMeta` in response | Existing | `query_ms` and `row_count` in every response |
| Smoke test entry | New | Add `risk_assessments` to `validate_analytical_family()` |
| `.http` test file | New | Add `tests/http/analytical-risk.http` or extend `analytical.http` |

---

## 3. Architectural Fit Assessment

### 3.1 Pattern Compliance

Family 04 follows the **9-artifact template** without deviation:

| # | Artifact | Source |
|---|----------|--------|
| 1 | Schema migration | `005_create_risk_assessments.sql` (pre-staged) |
| 2 | Writer mapper | `mapRiskRow` (pre-staged) |
| 3 | Pipeline config | `pipeline.go` (pre-staged) |
| 4 | ClickHouse reader | `risk_reader.go` (new) |
| 5 | Application use case | `get_risk_history.go` (new) |
| 6 | Contract types | `contracts.go` (extend) |
| 7 | HTTP handler method | `analytical.go` (extend) |
| 8 | Route registration | `analytical.go` routes (extend) |
| 9 | Integration test | Smoke + unit tests (new) |

### 3.2 What Changes vs What Stays

| Dimension | Changes | Stays |
|-----------|---------|-------|
| Write path | Nothing | Mapper + pipeline pre-staged |
| DI pattern | Add one field to deps structs | Struct-based, additive |
| Query builder | New function, same shape | SQL template identical |
| JSON parsers | 2 new (StrategyInputs, Constraints) | ParseMetadataJSON reused |
| Route pattern | One new conditional route | Registration pattern identical |
| Error handling | Same problem.Problem approach | No new error types |
| HTTP contract | Same query-param style | No body parsing needed |

### 3.3 Pressure Points This Family Will Expose

| Pressure Point | Expected Impact | Measurement |
|---------------|-----------------|-------------|
| Handler file growth | ~500→~580 lines | Under 600-line threshold |
| Reader JSON parsing complexity | 4 parsers in one reader | Code clarity subjective review |
| Test count growth | +27 tests expected | Total analytical tests ~130+ |
| Contracts file growth | +3 types added | File stays manageable |
| Compose function growth | +5 lines | No structural change |

---

## 4. Domain Context

### 4.1 Risk Assessment in the Pipeline

```
Evidence (Candles) → Signals → Decisions → Strategies → Risk Assessments → Executions
     F-01              F-01      F-02         F-03            F-04            (future)
```

Risk assessments evaluate strategy outputs against risk constraints before execution. Each assessment:
- References contributing strategies (`strategies` JSON array)
- Applies position/exposure constraints (`constraints` JSON struct)
- Produces a disposition: `approved`, `modified`, or `rejected`
- Records rationale as free text
- Carries type-specific parameters and arbitrary metadata

### 4.2 Event Source

- **NATS subject:** `risk.events.position_exposure.assessed.>`
- **Event type:** `risk.events.v1.position_exposure_assessed`
- **Domain type:** `risk.RiskAssessedEvent` containing `risk.RiskAssessment`

### 4.3 Analytical Query Use Cases

The `/analytical/risk/history` endpoint serves:
- Reviewing risk disposition patterns over time
- Auditing which strategies were approved/modified/rejected
- Understanding constraint application patterns
- Investigating rationale for specific dispositions
- Correlating risk decisions with downstream executions (future)
