# Family 05 — Success Criteria, Risks, and Non-Goals

> Objective criteria for accepting Family 05 implementation, quantified risks with mitigations, and explicit non-goals to prevent scope expansion.

---

## 1. Success Criteria

### 1.1 Hard requirements (must pass)

| # | Criterion | Threshold | Measurement | Action if failed |
|---|-----------|-----------|-------------|-----------------|
| SC-1 | All 9 artifacts delivered | 9/9 | Artifact checklist | Block merge |
| SC-2 | Handler file ≤ 620 lines | 620 lines | `wc -l analytical.go` | Extract `parseAnalyticalParams()` before merge |
| SC-3 | New frictions ≤ 2 | 2 | Friction log count | If > 2: mandatory hardening before Family 06 |
| SC-4 | Creative decisions = 0 | 0 | Implementation log | If > 0: pattern review |
| SC-5 | Write path changes = 0 | 0 | `git diff cmd/writer/` | Investigate — invariant broken |
| SC-6 | Test regressions = 0 | 0 | `go test ./...` | Fix before merge |
| SC-7 | All existing analytical endpoints pass | 5/5 unchanged | Smoke test | Fix before merge |
| SC-8 | Execution endpoint returns 200 | 1 endpoint | HTTP test | Fix before merge |
| SC-9 | Missing param returns 400 | 4 required params | HTTP test | Fix before merge |
| SC-10 | Unavailable reader returns 503 | Nil reader | Unit test | Fix before merge |

### 1.2 Ceiling-test metrics (diagnostic — not blocking)

These metrics are measured post-implementation and reported in the validation findings. They do not block merge but determine the codegen/hardening scope for Family 06.

| Metric | Pre-Family-05 | Expected post | Significance |
|--------|--------------|---------------|--------------|
| Handler file size | 515 lines | 611–626 lines | Defines handler split urgency |
| Total analytical LOC | ~3,348 | ~3,943 | Defines codegen ROI |
| JSON parser count | 6 | 8 | At threshold — generic parser decision |
| Handler duplication % | ~85% | ~85% | Codegen template candidates |
| Reader duplication % | ~80% | ~80% | Codegen template candidates |
| Float64 handling friction | N/A | Expected zero | New column type validation |
| Two-filter method friction | N/A | Expected zero | Multi-filter interaction |
| Implementation time | N/A | Expected consistent with F-02–F-04 | Pattern stability |
| Smoke test line count | ~750 | ~780 | Restructuring trigger proximity |

### 1.3 9-artifact checklist

| # | Artifact | File | Verification |
|---|----------|------|-------------|
| 1 | Schema migration | `deploy/migrations/006_create_executions.sql` | Pre-staged (verify unchanged) |
| 2 | Writer mapper | `cmd/writer/mappers.go` (`mapExecutionRow`) | Pre-staged (verify unchanged) |
| 3 | Writer pipeline entry | `cmd/writer/pipeline.go` | Pre-staged (verify unchanged) |
| 4 | ClickHouse reader | `internal/adapters/clickhouse/execution_reader.go` | New file, ~142 LOC |
| 5 | Application use case + contracts | `internal/application/analyticalclient/get_execution_history.go` + `contracts.go` | New file + extension, ~151 LOC |
| 6 | HTTP handler method | `internal/interfaces/http/handlers/analytical.go` | Extended, +96–111 LOC |
| 7 | Route registration | `internal/interfaces/http/routes/analytical.go` | Extended, +12 LOC |
| 8 | Gateway wiring | `cmd/gateway/analytical_reader.go` + compose | Extended, +13 LOC |
| 9 | Tests + smoke + HTTP queries | Multiple files | ~302 LOC total |

---

## 2. Risks

### Risk 1: Handler file exceeds 620-line ceiling

- **Probability:** High (projected 611–626 lines — at boundary).
- **Impact:** Mid-implementation extraction needed, adding ~1 hour to delivery.
- **Mitigation:** Check line count after adding interface + struct + deps fields (before writing handler method body). If projected to exceed 620, extract `parseAnalyticalParams()` helper first. This extraction is mechanical, well-scoped, and explicitly authorized during Family 05.
- **Worst case:** Handler at 626 lines → extract helper → handler drops to ~470 lines (6 methods × ~30 lines each + shared code). Net improvement.

### Risk 2: Parser count at threshold (8)

- **Probability:** Certain (two new parsers: ParseRiskInputJSON, ParseFillsJSON).
- **Impact:** Low — 8 is at the healthy threshold, not exceeding it.
- **Mitigation:** Document parser count in validation findings. Flag for codegen tranche: generic `parseJSON[T]` as candidate for Family 06+ template.
- **Worst case:** If execution schema requires a 9th parser (unexpected), generic parser evaluation becomes immediate.

### Risk 3: Float64 precision in round-trip

- **Probability:** Low.
- **Impact:** Low — cosmetic precision difference in quantity/filled_quantity representation.
- **Mitigation:** `FormatFloat` already handles float64 → string conversion for confidence fields in decisions, strategies, risk. Same function, same precision behavior. Known and accepted (documented since Family 02).
- **Worst case:** Quantity `0.001` stored as float64 reads back as `0.001000000000000000066613...`. FormatFloat formats to `%g` representation, which truncates trailing precision noise.

### Risk 4: Two-filter WHERE clause interaction

- **Probability:** Very low.
- **Impact:** Low — both filters are independent additive WHERE clauses.
- **Mitigation:** `side` and `status` are independent LowCardinality columns. Adding both WHERE clauses is the same as adding one — ClickHouse evaluates them independently. No logical interaction is possible.
- **Worst case:** Both filters applied simultaneously return empty results when data exists but doesn't match the combined criteria. Expected behavior — not a bug.

### Risk 5: Reader signature width (10 parameters)

- **Probability:** Certain (by design — 10 params).
- **Impact:** Low — functional, testable, consistent with pattern.
- **Mitigation:** This is the widest reader signature in the system. Acceptable for Family 05 as a one-off. If Family 06+ requires wider signatures, codegen should generate readers from schema definitions rather than handwritten signatures.
- **Worst case:** Reader signature is unwieldy but functional. Codegen eliminates the issue for future families.

### Risk 6: Fills array parsing surprise

- **Probability:** Low — FillRecord is a typed struct with 5 fields.
- **Impact:** Low — follows ParseStrategyInputsJSON pattern (slice of structs).
- **Mitigation:** FillRecord has typed fields (`Price string`, `Quantity string`, `Fee string`, `Simulated bool`, `Timestamp time.Time`). json.Unmarshal handles this transparently. Empty fills array (most common case for paper orders) deserializes to `[]FillRecord{}`.
- **Worst case:** FillRecord has a field that json.Unmarshal handles differently than expected (e.g., time.Time format mismatch). Fix in parser fallback — same pattern as all other parsers.

### Risk 7: Codegen scope creep post-Family-05

- **Probability:** Medium.
- **Impact:** Medium — codegen tranche could expand to tests, smoke, migrations.
- **Mitigation:** Scope codegen to the three highest-duplication artifacts: readers (~80% identical), handlers (~85% identical), use cases (~70% identical). Tests and smoke are better served by parameterized helpers. Migrations are too varied for templates.
- **Note:** This is a post-Family-05 risk, not an implementation risk.

---

## 3. Non-Goals

### 3.1 Explicitly out of scope for Family 05

| Non-goal | Rationale |
|----------|-----------|
| **Implement `venue_market_order` support** | Only `paper_order` is in scope. Venue fills represent a separate event type with different source binary (execute, not derive) and potentially different schema needs. Requires its own gate. |
| **Cross-family queries** | No execution-to-risk joins, no pipeline trace queries, no aggregate views spanning multiple tables. Each family is a self-contained read path. |
| **Aggregation or analytics** | Raw row-level data only. No SUM, AVG, GROUP BY, window functions, or materialized views. |
| **Pagination beyond limit=500** | Hard cap at 500 rows, consistent with all 5 existing families. Cursor-based pagination is a separate feature. |
| **Filter value validation** | `side` and `status` values are not validated against known enums. Invalid values return empty results. Consistent with `outcome`, `direction`, `disposition` in prior families. |
| **Write-path changes** | Writer mapper, pipeline, NATS consumer are pre-staged and MUST NOT be modified. |
| **Codegen implementation** | Codegen is mandatory *before Family 06*, not *during Family 05*. |
| **Handler refactoring** | Unless the 620-line hard ceiling is breached. `parseAnalyticalParams()` extraction is allowed only as a triggered response, not proactive. |
| **Smoke test restructuring** | Linear growth is acceptable through Family 05. Restructuring (per-family files, separate functions) is recommended at Family 07+ but not required. |
| **CI smoke integration** | CI runs unit tests. Adding compose-based integration is infrastructure work beyond Family 05 scope. |
| **NATS consumer lag visibility** | Write-path observability improvement deferred (DEF-U3). |
| **Sticky degradation auto-recovery** | ClickHouse reconnection logic deferred (DEF-U4). |
| **Generic `parseJSON[T]` parser** | Parser count reaches 8 (threshold) but does not exceed it. Generic parser recommended if exceeded at Family 06. |
| **Family 06 preparation** | Family 05 is the terminal manual expansion. Post-implementation obligations (codegen tranche, handler split, pattern assessment) are defined but not implemented during Family 05. |

### 3.2 Why these non-goals are correct

1. **Venue fills are a different event type.** `VenueOrderFilledEvent` comes from the `execute` binary (not `derive`), uses a different NATS stream (`EXECUTION_FILL_EVENTS`), and may have different schema requirements. Treating it as part of Family 05 would expand scope beyond one family.

2. **Cross-family queries change the architecture.** Joining executions with risk assessments requires either a ClickHouse view or application-layer aggregation — both are architectural decisions, not mechanical pattern applications.

3. **Codegen during Family 05 defeats the diagnostic purpose.** Family 05 exists to measure the manual pattern's ceiling. Implementing codegen during the measurement invalidates the measurement.

4. **Filter validation creates maintenance burden.** Validating `side` against `{"buy", "sell", "none"}` and `status` against 7 known values would require keeping the handler in sync with domain enum definitions. The current approach (pass-through, empty results on invalid values) is simpler and consistent across all families.

---

## 4. Trade-offs

### 4.1 Accepted trade-offs

| Trade-off | What we accept | What we gain |
|-----------|---------------|-------------|
| Handler at 611–626 lines (near ceiling) | File is large but still navigable | Pattern consistency — no mid-family extraction required unless ceiling breached |
| Parser count at 8 (at threshold) | More parser functions than ideal | Each parser is ~10 lines, grep-safe, no abstraction overhead |
| Reader signature at 10 parameters | Wide function signature | Direct, explicit, no options-struct indirection |
| Two unvalidated optional filters | Invalid filter values return empty results | Simpler implementation, consistent with all prior families |
| No venue_market_order coverage | Layer 6 partially covered (paper orders only) | Clear, bounded scope — venue fills are a separate decision |

### 4.2 Why these trade-offs are acceptable for the terminal test

Family 05's primary purpose is diagnostic — it measures the manual pattern's ceiling. The trade-offs above are **features, not bugs** for this purpose:

- Handler at ceiling → tells us whether extraction is needed before codegen
- Parser count at threshold → tells us whether generic parser is needed
- Wide reader signature → tells us whether codegen should use schema-driven signatures
- Two unvalidated filters → tells us whether multi-filter handlers create unexpected complexity
- No venue coverage → keeps scope bounded so the measurement is clean

---

## 5. Post-Family-05 Obligations

These items become mandatory after Family 05 is validated:

| Obligation | Trigger | Estimated effort |
|-----------|---------|-----------------|
| Codegen tranche definition | Family 05 validated | Scope: readers, handlers, use cases |
| Handler split decision | Actual line count measured | 0.5–1 day (may be absorbed by codegen) |
| Pattern terminal assessment | Family 05 validation findings documented | Document: what the manual pattern proved, what it didn't, what codegen must address |
| Family 06 gate prerequisites | Codegen tranche complete | Define what must be true before Family 06 |

---

## 6. Verification Protocol

### 6.1 Pre-merge checklist

```
[ ] All 9 artifacts present and correct
[ ] Handler file ≤ 620 lines (or extraction applied)
[ ] All existing tests pass: go test ./...
[ ] New reader tests pass (query builder + parsers)
[ ] New use case tests pass (validation + execution)
[ ] New handler tests pass (HTTP contract)
[ ] Smoke test extended and passing
[ ] HTTP test queries added and verified
[ ] Column coherence verified: DDL (20) → mapper (20) → reader SELECT (16) → scan (16) → domain (16)
[ ] Writer mapper unchanged (git diff confirms)
[ ] No new creative decisions documented
[ ] Friction log updated (expected: 0–2 new frictions)
[ ] Ceiling-test metrics recorded
```

### 6.2 Post-merge verification

```
[ ] Endpoint accessible: GET /analytical/execution/history
[ ] Response shape matches contract (executions array, source, meta)
[ ] Optional filters work (side, status)
[ ] Time range filtering works (since, until)
[ ] Server-Timing header present
[ ] Error codes correct (400 for invalid params, 503 for unavailable)
[ ] All 5 prior endpoints unchanged (regression check)
```
