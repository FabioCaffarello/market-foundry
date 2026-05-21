# Family 04 Lifecycle Record -- Risk Assessments (position_exposure)

**Layer:** 5 (Evidence > Signal > Decision > Strategy > **Risk**)
**Stage range:** S178--S182
**Pattern:** Wave B v2 (9-artifact template)
**Predecessor:** Family 03 (Strategies / mean_reversion_entry)
**Role:** Pattern ceiling test

---

## Selection

### Trigger assessment (S178)

| Trigger | Status | Blocking? |
|---------|--------|-----------|
| D-4 Codegen | Activated (non-blocking) | No -- mandatory before F-06 |
| CI Smoke | Resolved (already in CI) | No |
| Friction Count | Not triggered (0 new in F-03) | No |
| JSON Column Ceiling (3>4) | Not triggered | No |
| Free-Text Column | Not triggered | No |
| Filter Scaling | Not triggered | No |
| Constructor/DI | Not triggered | No |

**Decision:** Family 04 authorized to proceed. No hardening tranche required.

### Why Risk Assessments

Risk assessments was chosen as the **pattern ceiling test**: highest column count (17 DDL), most JSON columns (4), first free-text column (`rationale`), first struct-target parser (`ParseConstraintsJSON`), established enum filter (`disposition`). All write-path artifacts pre-staged.

### What this family must prove

1. JSON column scaling to 4 columns.
2. Free-text column viability (`rationale`).
3. Mechanical repeatability (zero creative decisions).
4. Friction budget: <=2 new frictions.
5. Handler file remains under ~600 lines.

---

## Definition & Contract

### Domain: `risk.RiskAssessment`

Key fields: Type (position_exposure), Source, Symbol, Timeframe, Disposition (approved/modified/rejected), Confidence, Strategies ([]StrategyInput), Constraints (struct), Rationale (free text), Parameters (map), Metadata (map), Final (bool), Timestamp.

### Schema: migration 005 (pre-staged)

17 DDL columns. 13 domain columns in SELECT. 4 JSON columns + 1 free-text column.

### JSON columns (4 -- new record at time of implementation)

| Column | Go type | Parser |
|--------|---------|--------|
| strategies | `[]risk.StrategyInput` | `ParseStrategyInputsJSON` (new) |
| constraints | `risk.Constraints` | `ParseConstraintsJSON` (new -- first struct-target parser) |
| parameters | `map[string]string` | `ParseMetadataJSON` (reuse) |
| metadata | `map[string]string` | `ParseMetadataJSON` (reuse) |

### HTTP endpoint

```
GET /analytical/risk/history?type=...&source=...&symbol=...&timeframe=...&disposition=...&since=...&until=...&limit=...
```

Response: `{ risk_assessments: [...], source: "clickhouse", meta: { query_ms, row_count } }`

---

## Implementation

### Artifacts

| # | Artifact | Status | LOC |
|---|----------|--------|-----|
| 1 | Migration 005 | Pre-staged | -- |
| 2 | Writer mapper (`mapRiskRow`) | Pre-staged | -- |
| 3 | Pipeline entry (position_exposure) | Pre-staged | -- |
| 4 | Reader (`risk_reader.go`) | Built in S181 | 161 |
| 5 | Use case (`get_risk_history.go`) | Built in S181 | 93 |
| 6 | Contracts | Extended in S181 | +24 |
| 7 | Handler method | Extended in S181 | +98 |
| 8 | Route registration | Extended in S181 | +14 |
| 9 | Smoke test + HTTP tests | Extended in S181 | +73 |

Write-path changes: **zero** (fifth consecutive expansion).

### Key decisions

- `ParseConstraintsJSON` is the first struct-target parser -- structurally simpler than array parsers (`json.Unmarshal` into struct).
- `rationale` handled as plain string pass-through -- no JSON parsing, no special handling, simplest column type.
- `disposition` filter follows the exact passthrough pattern from `outcome` (F-02) and `direction` (F-03).
- Implementation was fully mechanical -- zero creative decisions required.

### Tests: 47+ risk-specific tests (26 adapter + 13 use case + 8 handler) -- all passing. Total analytical: 245.

---

## Validation

### End-to-end proof

- 17/17 DDL columns verified aligned across DDL > writer > reader.
- 4 JSON columns round-tripped (strategies array, constraints struct, parameters map, metadata map).
- Free-text `rationale` round-tripped without encoding issues.
- Struct-target parser (`ParseConstraintsJSON`) verified for valid, empty, and malformed inputs.
- Disposition filter verified for all values (approved, modified, rejected, nonexistent).

### Ceiling test measurements

| Measurement | Value | Status |
|-------------|-------|--------|
| Handler file | 515 lines | Healthy (<550) |
| Reader file | 161 lines | Concerning (150--180 range) |
| New frictions | 0 | Healthy |
| Creative decisions | 0 | Healthy |
| JSON parser count | 6 total | Healthy (at limit) |

**Gate output:** The Wave B manual expansion pattern can sustain at least one more family without structural intervention.

---

## Runtime & Operability

### Activation

- ClickHouse optionality preserved: risk reader only created if ClickHouse client available.
- Writer pipeline activates via `IsRiskFamilyEnabled("position_exposure")`.
- Migration 005 applied by `cmd/migrate` during startup.

### Observability

Identical instrumentation to all prior families. Writer-side observability pre-staged (tracker in `/statusz`, `/diagz`).

### Boundary review

All component boundaries verified strict: reader > use case (interface), use case > handler (private interface), handler > route (struct DI), route > gateway (adapter boundary). No cross-reader dependencies. Import graph clean with no circular dependencies.

### Graceful degradation

ClickHouse not configured: 503. ClickHouse down: 503. Empty results: 200 with `[]`. Missing params: 400.

---

## Findings & Frictions

### Positive findings

- Four JSON columns add no structural friction -- pattern handles 4 as easily as 1, 2, or 3.
- Struct-target parser (`ParseConstraintsJSON`) is simpler than array parsers.
- Free-text column (`rationale`) is the simplest column type -- no new patterns needed.
- `ParseMetadataJSON` now reused 6 times across 4 families.
- Disposition filter integrates identically to outcome and direction.
- 17 DDL columns verified without tooling pressure.

### Frictions

| ID | Friction | Severity | Status |
|----|----------|----------|--------|
| PF-1 | Handler method duplication ~90 lines per family (5th method) | Medium | Escalating -- extract at F-05 if confirmed |
| PF-2 | Smoke test at ~750 lines | Medium | Carried |
| PF-3 | Disposition filter case-sensitive and unvalidated | Low | Accepted |
| PF-4 | No CI integration for analytical smoke test | High | Carried (4th time) |
| PF-5 | No pagination beyond limit=500 | Low | Deferred |
| PF-6 | Smoke test doesn't verify JSON column contents | Low | Accepted |
| PF-7 | 6 parser functions with identical shape -- extraction candidate | Low | Accept; codegen would eliminate |

---

## Success Criteria & Blockers

### Success criteria (all passed)

- All 9 artifacts implemented following Wave B v2 pattern.
- Zero write-path changes. 17-column alignment verified. 4 JSON columns round-trip correctly.
- Free-text `rationale` round-trips without encoding issues.
- `disposition` filter works identically to prior enum filters.
- Graceful degradation (503) when ClickHouse unavailable.
- All existing analytical endpoints unaffected. Struct DI additive only.
- Handler <=600 lines. <=2 new frictions. Zero creative decisions.

### Ceiling test verdict

Pattern confirmed healthy at 4 JSON columns, 17 DDL columns, free-text column, and struct-target parser. Manual pattern viable for at least one more family. Codegen justified but non-blocking before Family 05.

### Deferred decisions

- Family 05 implementation: depends on F-04 gate results (passed).
- Codegen: mandatory before Family 06.
- Handler restructuring: triggered if handler exceeds 600 lines.
