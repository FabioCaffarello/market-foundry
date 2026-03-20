# Family 03 Lifecycle Record -- Strategies (mean_reversion_entry)

**Layer:** 4 (Evidence > Signal > Decision > **Strategy**)
**Stage range:** S174--S177
**Pattern:** Wave B v2 (9-artifact template)
**Predecessor:** Family 02 (Decisions / rsi_oversold)

---

## Selection

### Why Strategies

Strategies was selected as Family 03 because it was the next natural layer (layer 4 of 6) in the analytical dependency chain, maintained contiguous vertical coverage, and introduced healthy incremental complexity (3 JSON columns, a second enum-like filter) without requiring structural changes.

### Candidate comparison

| Candidate | Fit | Verdict |
|-----------|-----|---------|
| Strategies (mean_reversion_entry) | Best -- extends read path to layer 4, healthy complexity, high readiness | **Selected** |
| Risk Assessments (position_exposure) | Strong but premature -- coverage gap if added before strategies | Deferred to F-04 |
| Executions (paper_order) | Highest value but highest risk -- skips two layers, max schema complexity | Deferred to F-05 |
| EMA Crossover | Too shallow -- tests nothing new, existing infrastructure handles it | Deferred indefinitely |
| Tradeburst / Volume | Incomplete infrastructure -- require write-path work | Deferred indefinitely |

### What Strategies tests about the pattern

- 3 JSON columns (up from 2 in decisions)
- Second domain-specific enum filter (`direction`)
- `[]DecisionInput` JSON array (mirrors `[]SignalInput` from decisions)
- Schema coherence at 4 simultaneous families
- Fourth struct DI field in handler deps

---

## Definition & Contract

### Analytical contract

Given a strategy type, source, symbol, and timeframe, return historical strategy resolutions ordered newest-first, with optional direction and time-range filtering.

### Domain type: `strategy.Strategy`

Key fields: Type, Source, Symbol, Timeframe, Direction (long/short/flat), Confidence, Decisions ([]DecisionInput), Parameters (map), Metadata (map), Final (bool), Timestamp.

### JSON column inventory (3 columns -- highest at time of implementation)

| Column | Go type | Parser |
|--------|---------|--------|
| decisions | `[]DecisionInput` | `ParseDecisionInputsJSON` (new) |
| parameters | `map[string]string` | `ParseMetadataJSON` (reuse) |
| metadata | `map[string]string` | `ParseMetadataJSON` (reuse) |

### Schema: migration 004 (pre-staged)

15 DDL columns. MergeTree engine, partitioned by `toYYYYMM(timestamp)`, ordered by `(source, symbol, timeframe, type, timestamp)`, TTL 90 days.

### HTTP endpoint

```
GET /analytical/strategy/history?type=...&source=...&symbol=...&timeframe=...&direction=...&since=...&until=...&limit=...
```

Response: `{ strategies: [...], source: "clickhouse", meta: { query_ms, row_count } }`

---

## Implementation

### Artifacts

| # | Artifact | Status |
|---|----------|--------|
| 1 | Migration 004 | Pre-staged |
| 2 | Writer mapper (`mapStrategyRow`) | Pre-staged |
| 3 | Pipeline entry (mean_reversion_entry) | Pre-staged |
| 4 | Reader (`strategy_reader.go`) | Built in S176 |
| 5 | Use case (`get_strategy_history.go`) | Built in S176 |
| 6 | Contracts (query + reply) | Built in S176 |
| 7 | Handler method | Built in S176 |
| 8 | Route registration | Built in S176 |
| 9 | Smoke test + HTTP tests | Built in S176 |

Write-path changes: **zero** (fourth consecutive expansion with immutable writer).

### Key decisions

- `ParseDecisionInputsJSON` placed in `strategy_reader.go` (not shared file), following the parser-alongside-reader pattern.
- Direction filter: no validation -- invalid values return empty results (consistent with `outcome` in F-02).
- Two of three JSON columns reused `ParseMetadataJSON` -- only `[]DecisionInput` required a new parser.
- Struct DI (H-1 hardening) absorbed the fourth field with zero constructor churn.

### Test summary: 33 tests (14 adapter + 12 use case + 7 handler) -- all passing.

---

## Validation

### End-to-end proof

Full data path validated: NATS > writer > ClickHouse > reader > HTTP endpoint.

- 16/16 DDL columns verified aligned across DDL > writer > reader.
- 3 JSON columns round-tripped successfully (decisions array, parameters map, metadata map).
- Fallback behavior verified for empty/nil/invalid JSON inputs.
- Direction filter verified for all values (long, short, flat, empty, nonexistent).
- H-1 struct-based DI proven in first real use post-hardening.

### Boundary verification

- Zero changes to operational pipeline, candle/signal/decision read/write paths.
- ClickHouse optionality preserved (503 when unavailable).
- Writer pipeline isolation maintained.
- No cross-family queries.

---

## Runtime & Operability

### Observability

Identical instrumentation to all prior families: wall-clock timing in adapter, `QueryMeta` in use case, `Server-Timing` header in handler, structured logging at every layer.

### Failure modes

| Symptom | Cause | Recovery |
|---------|-------|----------|
| 503 | ClickHouse not configured/down | Check `clickhouse.addr` in config |
| Empty results | Writer pipeline disabled or events not flushed | Check `IsStrategyFamilyEnabled("mean_reversion_entry")` |
| Empty JSON fields | Serialization mismatch | Check writer `marshalJSON` output |

### Known limits

No pagination beyond 500 rows. TTL 90 days. No direction validation. No confidence filtering. No decision drill-down. No cross-family queries. JSON parsed client-side (no ClickHouse JSON functions).

---

## Findings & Frictions

### Positive findings

- Three JSON columns added zero structural friction -- pattern handles 3 as easily as 1 or 2.
- `ParseMetadataJSON` reused for 2 of 3 JSON columns -- JSON parsing scales through reuse.
- Direction filter integrates identically to outcome filter -- domain-specific filters are mechanical.
- Write path required zero changes for the fourth consecutive time.
- Observability parity achieved mechanically. Error handling contracts consistent across 4 families.

### Frictions

| ID | Friction | Severity | Status |
|----|----------|----------|--------|
| PF-1 | Handler method duplication ~80 lines per family (4th copy) | Medium | Carried -- accept until F-05+ |
| PF-2 | Smoke test approaching ~700 lines | Medium | Carried -- helper absorbs growth |
| PF-3 | Direction filter case-sensitive and unvalidated | Low | Accepted -- consistent with pattern |
| PF-4 | No CI integration for analytical smoke test | High | Carried (3rd time flagged) |
| PF-5 | No pagination beyond limit=500 | Low | Deferred |
| PF-6 | Smoke test doesn't verify JSON column contents | Low | Accepted |

---

## Success Criteria & Blockers

### Pre-expansion blockers (all cleared in S172 hardening tranche)

1. **H-1 Struct-Based DI** -- `NewAnalyticalWebHandler` accepts `AnalyticalHandlerDeps` struct. Cleared.
2. **H-2 Smoke Test Function Extraction** -- `validate_analytical_family()` reusable function exists. Cleared.
3. **H-3 Helper Renaming** -- `parseEvidenceKeyParams` renamed to `parseAnalyticalKeyParams`. Cleared.

### Success criteria (all passed at S177 gate)

- Schema coherence: 11/11 domain columns aligned DDL > writer > reader.
- Read path: direction filter, limit defaults/caps, empty results, time range -- all working.
- Application layer: all validation rules (required fields, bounds, ranges) enforced.
- HTTP surface: 200/400/503 codes correct, Server-Timing header present.
- Integration: smoke test exercises endpoint, all prior families unaffected.
- Boundary preservation: zero changes to writer, migration, or existing families.

### Pattern verdict after Family 03

Three JSON columns proven. Struct DI scalable. Write path immutable. Pattern ready for Family 04 evaluation. No blocking friction.
