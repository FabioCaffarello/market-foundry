# Wave B Family 02 Lifecycle Record -- Decisions (RSI Oversold)

**Family:** Decisions (RSI Oversold)
**Wave B Iteration:** 02
**Stages:** S168-S170
**Status:** Complete -- proven end-to-end

---

## Definition

**Selected:** Decisions (RSI Oversold) -- the second controlled Wave B expansion. Last expansion before the mandatory hardening tranche (Family 03).

**Selection rationale:**
1. Write path already active -- `mapDecisionRow()` exists, pipeline consumes `decision.events.rsi_oversold.evaluated`. Zero write-path changes required.
2. Schema ready -- migration `003_create_decisions.sql` already applied.
3. Controlled complexity increase -- 10 domain columns vs 8 for Signals. Two JSON columns (`signals` array + `metadata` map) vs one. Minimum increase to prove JSON extensibility.
4. Dependency chain proven -- decisions depend on signals which depend on evidence, already validated.
5. Explicitly authorized by S167 CONDITIONAL PASS verdict.

**Complexity delta from Family 01:**

| Dimension | Signals (F-01) | Decisions (F-02) | Delta |
|---|---|---|---|
| Domain columns | 8 | 10 | +2 (outcome, confidence) |
| JSON columns | 1 (metadata) | 2 (signals, metadata) | +1 (array type) |
| Enum-like columns | 0 | 1 (outcome) | +1 |
| Family-specific query param | 0 | 1 (outcome filter) | +1 |

**What this iteration must prove:**
1. JSON array deserialization works in the read path (`signals` as `[]SignalInput`).
2. Two JSON columns don't break the pattern.
3. Outcome filtering works as a domain-specific query parameter.
4. Constructor accumulation manageable at 3 use cases.
5. The 9-artifact pattern scales without surprise.

---

## Schema & Gateway

### Column Mapping: DDL -> Writer -> Reader

| Column | DDL Type | Writer | Reader | Aligned |
|---|---|---|---|---|
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 -> int | YES |
| outcome | LowCardinality(String) | string(Outcome) | string -> decision.Outcome | YES |
| confidence | Float64 | parseFloat -> float64 | float64 -> FormatFloat -> string | YES |
| signals | String | marshalJSON([]SignalInput) | ParseSignalInputsJSON -> []SignalInput | YES |
| metadata | String | marshalJSON(map[string]string) | ParseMetadataJSON -> map[string]string | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |

Event metadata (event_id, occurred_at, correlation_id, causation_id, ingested_at) are write-only.

**Schema coherence: 15/15 columns verified (10 domain + 5 write-only).**

### Data Flow

```
NATS JetStream (decision.events.rsi_oversold.evaluated)
  -> writerConsumer -> mapDecisionRow() -> INSERT INTO decisions (batch)
  -> decisions MergeTree table (partitioned by toYYYYMM, TTL 90 days)
  -> DecisionReader.QueryDecisionHistory() (parameterized SELECT, 10 domain columns)
  -> GetDecisionHistoryUseCase (validation, timing)
  -> GET /analytical/decision/history -> 200 JSON + Server-Timing
```

### Endpoint Specification

```
GET /analytical/decision/history
  Required: type, source, symbol, timeframe
  Optional: outcome (triggered|not_triggered|insufficient), limit (1-500, default 50), since, until
  Response: { decisions: [...], source: "clickhouse", meta: { query_ms, row_count } }
  Headers: Server-Timing: total;dur=N, query;dur=M
  Errors: 400 (invalid params), 503 (ClickHouse unavailable)
```

### Gateway Composition

DecisionReader wired only when ClickHouse available. Optionality preserved (R-02 compliance). No changes to non-analytical composition.

---

## Implementation

### 9-Artifact Pattern

| # | Artifact | File | Status |
|---|---|---|---|
| 1 | Migration DDL | `deploy/migrations/003_create_decisions.sql` | PRE-EXISTING |
| 2 | Writer mapper | `cmd/writer/mappers.go` (mapDecisionRow) | PRE-EXISTING |
| 3 | Writer pipeline entry | `cmd/writer/pipeline.go` (rsi_oversold) | PRE-EXISTING |
| 4 | Reader adapter | `internal/adapters/clickhouse/decision_reader.go` | NEW |
| 5 | Use case + contracts | `internal/application/analyticalclient/get_decision_history.go`, `contracts.go` | NEW |
| 6 | HTTP handler + route | `internal/interfaces/http/handlers/analytical.go`, `routes/analytical.go` | EXTENDED |
| 7 | Gateway composition | `cmd/gateway/analytical_reader.go`, `compose.go` | EXTENDED |
| 8 | Integration test | `tests/http/analytical.http` | EXTENDED |
| 9 | Smoke test section | `scripts/smoke-analytical-e2e.sh` (Phase 5c) | EXTENDED |

### JSON Deserialization

- **signals column** (`[]SignalInput`): First JSON array in the read path. `ParseSignalInputsJSON()` returns empty slice on invalid JSON. `json.Unmarshal` handles arrays and maps identically -- no friction observed.
- **metadata column** (`map[string]string`): Reuses existing `ParseMetadataJSON()` from signal reader. No changes needed.
- **confidence column** (Float64 -> string): Round-trip via `parseFloat()` / `FormatFloat()` preserves reasonable precision; may alter representation (e.g., "0.80" -> "0.8").

### Known Limits

- Outcome filtering is case-sensitive and unvalidated (lowercase by convention).
- No aggregation -- raw decision events only.
- No signal drill-down -- `signals` array returned as-is, no cross-family joins.
- Constructor now takes 4 positional args (candle, signal, decision, logger) -- H-1 hardening committed for Family 03.

---

## Validation

### Unit Tests: 30+ decision-related tests -- all passing

| Package | Tests |
|---|---|
| `internal/adapters/clickhouse` | 12 decision reader tests |
| `internal/application/analyticalclient` | 11 decision use case tests |
| `internal/interfaces/http/handlers` | 7 decision handler tests |
| `cmd/writer` | mapper tests |

### JSON Payload Coherence

| Column | Write -> Store -> Read | Verified |
|---|---|---|
| signals | `marshalJSON([]SignalInput)` -> JSON string -> `ParseSignalInputsJSON()` -> `[]SignalInput` | YES |
| metadata | `marshalJSON(map[string]string)` -> JSON string -> `ParseMetadataJSON()` -> `map[string]string` | YES |

Fallbacks verified: empty/nil -> writer produces `"[]"` or `"{}"` -> reader returns empty type.

### Integration Smoke

`scripts/smoke-analytical-e2e.sh` Phase 5c (12 checks):
- ClickHouse row count for `decisions WHERE type='rsi_oversold'`
- HTTP 200 with correct JSON structure, all 10 domain fields
- `signals` field is valid JSON array, `metadata` is valid JSON object
- Server-Timing header present
- 400 for missing params, invalid limit, since > until
- Outcome filter returns subset

### Boundary Verification

- Operational pipeline (NATS KV) unchanged
- Candle and signal families: zero changes
- ClickHouse optionality preserved (503 when unavailable)
- Writer pipeline isolation -- rsi_oversold independent of candle/signal pipelines
- No cross-family queries

---

## Runtime & Operability

### Activation Rules

Decision endpoint activates when: ClickHouse configured and reachable, `decisions` table exists (migration 003 applied), rsi_oversold pipeline enabled in writer. If any condition unmet, endpoint returns 503. Gateway remains healthy -- ClickHouse not in readiness check.

### Diagnostic Commands

```bash
# Writer consuming decisions?
curl -s http://127.0.0.1:8085/statusz  # check decision tracker

# ClickHouse data?
SELECT count() FROM decisions WHERE type = 'rsi_oversold';
SELECT outcome, count() FROM decisions WHERE type = 'rsi_oversold' GROUP BY outcome;

# Endpoint responding?
curl -s "http://127.0.0.1:8080/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&limit=5"
```

### Failure Modes

| Failure | Symptom | Recovery |
|---|---|---|
| ClickHouse down | 503 on endpoint | Restart ClickHouse; writer retries |
| Writer consumer degraded | Stale data | Check writer logs, restart writer |
| decisions table missing | 503 | Run `make migrate-up` |
| JSON deserialization failure | signals returns `[]` | Check writer marshalJSON output |
| Confidence parse error | confidence returns "0" | Check writer parseFloat logs |

### Performance

- ORDER BY key `(source, symbol, timeframe, type, timestamp)` aligns with query patterns.
- `outcome` filter on LowCardinality column -- ClickHouse optimizes efficiently.
- Batch/flush/retry shared with all writer pipelines.

---

## Findings

### What Worked
- F-1: JSON array deserialization adds no structural friction -- `json.Unmarshal` handles arrays and maps identically.
- F-2: Two JSON columns do not compound complexity -- each parsed independently.
- F-3: Domain-specific optional filter (outcome) integrates cleanly -- one WHERE clause, one passthrough per layer.
- F-4: Write path required zero changes (second time confirmed).
- F-5: Observability parity achieved mechanically (third time).
- F-6: Error handling contracts remain consistent across 3 families.
- F-7: Confidence float64 round-trip works (cosmetic precision changes only).

### Pattern Frictions

| ID | Friction | Status |
|---|---|---|
| PF-1 | Constructor with 4 positional args (confirmed, escalated) | H-1: struct-based DI for Family 03 |
| PF-2 | `parseEvidenceKeyParams()` naming residue (3 consumers) | H-3: rename for Family 03 |
| PF-3 | Smoke test at ~200 lines for 3 phases | H-2: extract `validate_analytical_family()` for Family 03 |
| PF-4 | Outcome filter case-sensitive and unvalidated | Accepted (empty results, no risk) |
| PF-5 | No CI integration for analytical smoke test | Carried forward (high impact) |
| PF-6 | No pagination beyond limit=500 | Deferred (sufficient for current usage) |

### Pre-Committed Hardening for Family 03

| ID | Item | Origin |
|---|---|---|
| H-1 | Refactor NewAnalyticalWebHandler to struct-based DI | F-01 PF-2, confirmed F-02 PF-1 |
| H-2 | Parameterize smoke test with validate_analytical_family() | F-01 PF-5, approaching threshold |
| H-3 | Rename parseEvidenceKeyParams() to parseAnalyticalKeyParams() | F-01 PF-1, confirmed F-02 PF-2 |
| H-4 | Review naming consistency (consumer/inserter labels) | S169 |

### Success Criteria Met

- SC-1 through SC-6: Schema coherence verified (10/10 domain columns, JSON round-trips, confidence/outcome types).
- RP-1 through RP-6: Read path correctness (ordering, outcome filter, time range, limit defaults).
- AL-1 through AL-5: Application layer (validation, error mapping, QueryMeta).
- HS-1 through HS-6: HTTP surface (200/400/503, Server-Timing, outcome param).
- IN-1 through IN-6: Integration (smoke phases pass, CI passes).
- BP-1 through BP-6: Boundary preservation (zero writer/candle/signal changes).

### What the Pattern Proves After 2 Expansions
1. JSON array and JSON map columns follow the same pattern.
2. Domain-specific optional filters integrate without structural changes.
3. Write path remains stable across expansions.
4. Observability parity is mechanical.
5. The 9-artifact pattern produces consistent, predictable results.

### What the Pattern Does NOT Yet Prove
1. That H-1 (struct-based DI) resolves constructor accumulation cleanly.
2. That the smoke test scales beyond 3 families.
3. That CI integration for smoke tests is achievable.
4. That the pattern works for fundamentally different data shapes.

---

*Consolidated from: wave-b-family-02-decisions-definition.md, wave-b-family-02-decisions-end-to-end-validation.md, wave-b-family-02-decisions-implementation-notes.md, wave-b-family-02-decisions-runtime-and-operability-notes.md, wave-b-family-02-decisions-schema-writer-reader-gateway-scope.md, wave-b-family-02-decisions-success-criteria-and-non-goals.md, wave-b-family-02-decisions-validation-findings-and-pattern-frictions.md*
