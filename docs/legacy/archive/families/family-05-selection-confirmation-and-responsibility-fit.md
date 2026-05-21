# Family 05 — Selection Confirmation and Responsibility Fit

> Formal confirmation of Executions (paper_order) as Family 05, with responsibility mapping, architectural fitness assessment, and explicit framing as the terminal test of the Wave B manual expansion pattern.

---

## 1. Confirmed Family

| Field | Value |
|-------|-------|
| Family | 05 |
| Domain | Executions |
| Event type | `paper_order` (PaperOrderSubmittedEvent) |
| Layer | 6 — Execution (terminal) |
| Source binary | derive |
| ClickHouse table | `executions` |
| Migration | `006_create_executions.sql` (pre-staged) |
| Writer mapper | `mapExecutionRow()` (pre-staged) |
| NATS stream | `EXECUTION_EVENTS` |

---

## 2. Why Executions

### 2.1 Architectural justification

Executions is the **only candidate** that advances the analytical read path to a new, uncovered layer. All other candidates either deepen existing layers (EMA Crossover, Tradeburst, Volume) or require infrastructure not yet built.

The analytical dependency chain is:

```
Evidence → Signals → Decisions → Strategies → Risk → Executions
   (L1)      (L2)      (L3)        (L4)       (L5)     (L6)
  Baseline    F-01      F-02        F-03       F-04     F-05
```

Family 05 completes this chain. After Family 05, the analytical surface covers every layer of the trading pipeline end-to-end. Operators can trace any event from market evidence through signal generation, decision evaluation, strategy resolution, risk assessment, to execution outcome — entirely within the analytical read path.

### 2.2 Terminal test rationale

Family 05 is explicitly positioned as the **last family expandable under the current manual pattern**. Three converging pressures make this boundary concrete:

| Pressure | Pre-Family-05 | Post-Family-05 | Family-06 threshold |
|----------|--------------|----------------|-------------------|
| Codegen necessity (T-CG) | ~800 LOC duplication | ~1,400 LOC duplication | Manual maintenance cost exceeds template cost |
| Handler file size (T-HS) | 515 lines | ~595–615 lines | >600 lines (critical threshold) |
| Total analytical LOC | ~3,348 | ~3,943 | ~4,538 (exceeds comfortable manual threshold) |

**Family 05 is the inflection point.** If it passes cleanly (zero new frictions, ≤620 handler lines, ≤2 new parsers), the manual pattern has been validated to its natural limit. If it creates friction, that friction defines exactly what the codegen/hardening tranche must address.

Either outcome is valuable. Family 05 is not just the next expansion — it is a **diagnostic instrument** for the pattern itself.

### 2.3 Responsibility fit

Executions maps cleanly to an isolated responsibility boundary:

| Responsibility | Owner | Boundary |
|---------------|-------|----------|
| Event production | `derive` binary via NATS | Writer receives, does not produce |
| Event persistence | Writer service (pipeline + inserter) | Already operational |
| Schema definition | Migration 006 | Already applied |
| Historical query | New execution reader adapter | Self-contained, no upstream dependencies |
| Query orchestration | New execution use case | Same contracts pattern as 5 predecessors |
| HTTP surface | New handler method + route | Additive to existing analytical handler |
| Gateway composition | New reader factory function | Struct DI addition |

No responsibility crosses an existing family boundary. No shared state. No new infrastructure.

---

## 3. Schema Profile

From `006_create_executions.sql`:

| Column | Type | Category | Pattern precedent |
|--------|------|----------|-------------------|
| event_id | String | Event metadata | All families |
| occurred_at | DateTime64(3) | Event metadata | All families |
| correlation_id | String | Event metadata | All families |
| causation_id | String | Event metadata | All families |
| type | LowCardinality(String) | Domain key | All families |
| source | LowCardinality(String) | Domain key | All families |
| symbol | LowCardinality(String) | Domain key | All families |
| timeframe | UInt32 | Domain key | All families |
| side | LowCardinality(String) | **Domain enum** | Pattern: outcome (F-02), direction (F-03), disposition (F-04) |
| quantity | Float64 | **Domain numeric** | **New — first Float64 in read path** |
| filled_quantity | Float64 | **Domain numeric** | **New — second Float64** |
| status | LowCardinality(String) | **Domain enum** | Pattern: outcome (F-02), direction (F-03), disposition (F-04) |
| risk | String (JSON) | JSON column | Pattern: metadata, parameters, etc. |
| fills | String (JSON) | JSON column | **New — array of fill entries** |
| parameters | String (JSON) | JSON column | Pattern: strategies (F-03), risk (F-04) |
| metadata | String (JSON) | JSON column | Pattern: all families |
| exec_correlation_id | String | Domain metadata | **New — execution-specific** |
| exec_causation_id | String | Domain metadata | **New — execution-specific** |
| final | Bool | **Domain boolean** | **New — first boolean in read path** |
| timestamp | DateTime64(3) | Domain timestamp | All families |

### Complexity profile

| Metric | Family 04 (Risk) | Family 05 (Executions) | Delta |
|--------|------------------|----------------------|-------|
| DDL columns | 17 | 20 | +3 |
| JSON columns | 4 | 4 | 0 |
| Enum-like filters | 1 (disposition) | 2 (side, status) | +1 |
| Float64 columns | 0 | 2 | +2 (new type) |
| Boolean columns | 0 | 1 | +1 (new type) |
| Free-text columns | 1 (rationale) | 0 | -1 |
| Extra correlation IDs | 0 | 2 | +2 |

### What's genuinely new

1. **Float64 columns** — `quantity` and `filled_quantity`. First non-string, non-integer numeric type in the read path. `FormatFloat` already exists (used for `confidence` in decisions, strategies, risk) so the scan/format pattern is proven; the novelty is in having two Float64 columns in one family.

2. **Boolean column** — `final`. First boolean in the read path. Trivial scan — Go `bool` maps directly to ClickHouse `Bool`. No parser needed.

3. **Fills JSON array** — `fills` stores execution fill entries. Structure depends on domain definition but is expected to be `[]map[string]interface{}` or a typed struct array. Requires one new parser (`ParseFillsJSON` or similar). Pushes parser count to 7.

4. **Two optional enum filters** — `side` and `status` as query parameters. First family with two optional filters in one handler method. Pattern proven individually — additive WHERE clauses, no interaction risk.

5. **Execution-specific correlation IDs** — `exec_correlation_id` and `exec_causation_id`. Direct string scan, no special handling. Column count increases but complexity does not.

---

## 4. Responsibility Map (9-Artifact Template)

| # | Artifact | Path | Status | Estimated LOC |
|---|----------|------|--------|---------------|
| 1 | Schema migration | `deploy/migrations/006_create_executions.sql` | **Pre-staged** | 39 (exists) |
| 2 | Writer mapper | `cmd/writer/mappers.go` (`mapExecutionRow`) | **Pre-staged** | ~35 (exists) |
| 3 | Writer pipeline entry | `cmd/writer/pipeline.go` | **Pre-staged** | ~5 (exists) |
| 4 | ClickHouse reader | `internal/adapters/clickhouse/execution_reader.go` | **To build** | ~143 |
| 5 | Application use case | `internal/application/analyticalclient/get_execution_history.go` | **To build** | ~128 |
| 6 | Contracts | `internal/application/analyticalclient/contracts.go` | **To extend** | ~30 |
| 7 | HTTP handler method | `internal/interfaces/http/handlers/analytical.go` | **To extend** | ~80–100 |
| 8 | Route registration | `internal/interfaces/http/routes/analytical.go` | **To extend** | ~5 |
| 9 | Gateway wiring | `cmd/gateway/analytical_reader.go` | **To extend** | ~8 |

### Test artifacts

| Test | Path | Estimated LOC |
|------|------|---------------|
| Reader unit tests | `internal/adapters/clickhouse/execution_reader_test.go` | ~95 |
| Use case unit tests | `internal/application/analyticalclient/get_execution_history_test.go` | ~90 |
| Handler unit tests | `internal/interfaces/http/handlers/analytical_test.go` | ~85 |
| Smoke test extension | `scripts/smoke-analytical-e2e.sh` | ~30 |
| HTTP test queries | `tests/http/analytical.http` | ~15 |

### Documentation artifacts

| Document | Purpose |
|----------|---------|
| Coherence table | DDL → mapper → reader column alignment |
| Endpoint spec | `GET /analytical/execution/history` query params and response |
| Known limits | What the family does and does not cover |
| Friction log | New frictions discovered during implementation |

---

## 5. Endpoint Specification (Projected)

```
GET /analytical/execution/history
    ?type=paper_order              (required — LowCardinality)
    &source=derive                 (required)
    &symbol=BTCUSDT                (required)
    &timeframe=60                  (required)
    &side=BUY                      (optional — enum filter)
    &status=filled                 (optional — enum filter)
    &since=2026-03-01T00:00:00Z    (optional — time range)
    &until=2026-03-20T00:00:00Z    (optional — time range)
    &limit=50                      (optional — default 50, max 500)
```

Response shape follows the established contract:

```json
{
  "source": "analytical/clickhouse",
  "data": [ ... ],
  "meta": {
    "query_ms": 12,
    "row_count": 3
  }
}
```

---

## 6. Success Criteria

### Hard requirements

| Criterion | Threshold | Action if exceeded |
|-----------|-----------|-------------------|
| Handler file size | ≤620 lines | Immediate `parseAnalyticalParams()` extraction |
| New frictions | ≤2 | Mandatory hardening before Family 06 |
| JSON parser count | ≤8 | Generic `parseJSON[T]` evaluation |
| Creative decisions | 0 | Pattern review |
| Write path changes | 0 | Investigate; immutability invariant broken |
| Test regressions | 0 | Immediate fix before merge |

### Ceiling-test metrics (diagnostic, not blocking)

These metrics do not block Family 05 but their values define the codegen/hardening scope for Family 06:

| Metric | Pre-Family-05 | Measure post-Family-05 | Significance |
|--------|--------------|----------------------|--------------|
| Total analytical LOC | ~3,348 | Expected ~3,943 | Defines codegen ROI |
| Handler duplication % | ~85% | Expected ~85% | Defines extraction targets |
| Reader duplication % | ~80% | Expected ~80% | Defines template candidates |
| Smoke test growth | ~750 lines | Expected ~780 lines | Restructuring trigger proximity |
| Float64 handling friction | N/A | Measure | New column type validation |
| Two-filter handler method | N/A | Measure | Interaction complexity |

---

## 7. What Family 05 Must NOT Do

1. **Implement codegen.** Codegen is mandatory *before Family 06*, not *during Family 05*.
2. **Refactor the handler.** Unless the 620-line hard ceiling is breached.
3. **Refactor smoke tests.** Linear growth is acceptable through Family 05.
4. **Open Family 06.** Family 06 requires codegen tranche as a prerequisite.
5. **Change the write path.** Writer must remain immutable for the 6th consecutive expansion.
6. **Add venue_market_order support.** Only paper_order is in scope. Venue fills are a separate future candidate.
7. **Implement cross-family queries.** Execution-to-risk joins or pipeline trace queries are out of scope.

---

## 8. Signals This Family Must Produce

Family 05's primary function beyond coverage completion is to **generate diagnostic signals** for the pattern's terminal assessment:

### Signal 1: Handler file size at boundary

- **Expected:** 595–615 lines.
- **If ≤600:** Manual pattern has headroom. Codegen is optimization, not survival.
- **If 600–620:** Manual pattern at limit. Codegen prevents regression at Family 06.
- **If >620:** Manual pattern has hit ceiling. Immediate extraction needed; codegen is urgent.

### Signal 2: Friction count

- **Expected:** 0–1 new frictions (cosmetic).
- **If 0:** Pattern is robust through 6 layers. Codegen is about efficiency, not correctness.
- **If 1–2:** Pattern is strained but functional. Codegen addresses specific pain points.
- **If >2:** Pattern is degrading. Mandatory hardening before any further expansion.

### Signal 3: Float64 handling

- **Expected:** Reuses `FormatFloat`, zero friction.
- **If clean:** Float64 is a non-concern. Financial precision handled by existing patterns.
- **If friction:** Float64 requires special formatting or precision handling. Document for codegen template.

### Signal 4: Two-filter handler method

- **Expected:** Additive WHERE clauses, no interaction.
- **If clean:** Multi-filter methods are mechanical. No special handling for Family 06+.
- **If friction:** Filter combination creates unexpected complexity. Document for codegen template.

### Signal 5: Parser count trajectory

- **Expected:** 7 parsers (one new for fills array).
- **If ≤7:** Well within manageable range. Generic parser deferred.
- **If 8:** At concerning threshold. Generic parser becomes recommended.

### Signal 6: Implementation time

- **Expected:** Consistent with Families 02–04 (mechanically repeatable).
- **If consistent:** Pattern is stable and predictable. Codegen ROI is about LOC reduction, not time reduction.
- **If slower:** Additional complexity from Float64/fills/two-filters is slowing the pattern. Document for codegen design.

---

## 9. Post-Family-05 Obligations

Upon Family 05 completion and validation, the following become **mandatory**:

1. **Codegen tranche definition** — Scope, effort, and implementation plan for generating readers, handlers, and use cases from templates. This is the hard gate for Family 06.

2. **Handler file split decision** — Based on actual post-Family-05 line count, decide between:
   - `parseAnalyticalParams()` extraction (~1 hour)
   - Handler file split by domain (~2 hours)
   - Codegen-based generation (absorbs both)

3. **Pattern terminal assessment** — Formal document evaluating whether the manual expansion pattern succeeded, what it proved, and what the codegen tranche must address.

4. **Family 06 gate prerequisites** — Define what must be true before Family 06 can be considered (codegen operational, handler refactored, CI smoke restructured if needed).

---

## 10. Conclusion

Executions (paper_order) is confirmed as Family 05. The selection is architecturally defensible on every dimension: it is the only candidate that advances vertical coverage, provides maximum ceiling-test value, has complete pre-staging, and occupies the terminal position in the analytical dependency chain.

More importantly, Family 05 is positioned as a **diagnostic instrument**. Its implementation will produce concrete signals — handler size, friction count, Float64 handling, multi-filter complexity, parser trajectory — that determine the scope and urgency of the codegen/hardening tranche required before Family 06.

The base is ready for S186 to freeze the contract and implementation scope.
