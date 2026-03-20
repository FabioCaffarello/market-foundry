# Stage S168 — Decisions Family Expansion Definition Report

**Status:** COMPLETE
**Stage:** S168
**Predecessor:** S167 (CONDITIONAL PASS)
**Successor:** S169 (Decisions family implementation)
**Date:** 2026-03-19

---

## 1. Executive Summary

Stage S168 formally defines the second Wave B iteration: **Decisions (RSI Oversold)**. This is the last expansion iteration before the mandatory hardening tranche (Family 03).

The Decisions family was selected because it introduces controlled complexity — two JSON columns (`signals` and `metadata`) versus one for Signals, plus a categorical `outcome` filter — without exceeding the pattern's proven capacity. The write path is already active. The schema already exists. The iteration is strictly additive.

All deliverables are complete. The base is ready for disciplined implementation in S169.

---

## 2. Deliverables Produced

| # | Document | Purpose |
|---|----------|---------|
| 1 | `docs/architecture/wave-b-family-02-decisions-definition.md` | Family definition, rationale, complexity analysis, constraints, pre-commitments |
| 2 | `docs/architecture/wave-b-family-02-decisions-schema-writer-reader-gateway-scope.md` | Full technical scope: schema mapping, 9 artifacts, endpoint spec, file manifest |
| 3 | `docs/architecture/wave-b-family-02-decisions-success-criteria-and-non-goals.md` | 30 success criteria, 13 non-goals, gate review protocol, stop conditions |
| 4 | `docs/stages/stage-s168-decisions-family-expansion-definition-report.md` | This report |

---

## 3. Family Selection Summary

### Why Decisions (RSI Oversold)

- Write path already active (`mapDecisionRow()`, consumer running, pipeline registered)
- Schema already applied (migration 003)
- Minimal complexity delta from Signals: +2 domain columns, +1 JSON column, +1 enum-like column
- Explicitly authorized by S167 CONDITIONAL PASS
- Tests JSON array deserialization (`[]SignalInput`) — not yet proven by any family
- Tests family-specific query parameter (`outcome` filter) — first non-shared parameter

### Why Not Other Families

- **Strategy:** 3 JSON columns — too large a complexity jump
- **Risk Assessment:** 5 JSON columns + rationale string — exceeds controlled expansion
- **Execution:** Most complex family with dual correlation/causation IDs — wrong candidate for iteration 02

---

## 4. Complexity Delta

| Dimension | Signals (F01) | Decisions (F02) | Delta |
|-----------|---------------|-----------------|-------|
| Domain columns | 8 | 10 | +2 |
| JSON columns | 1 | 2 | +1 |
| Enum-like columns | 1 | 2 | +1 |
| SELECT columns | 8 | 10 | +2 |
| Family-specific query params | 0 | 1 (`outcome`) | +1 |

---

## 5. What the Iteration Must Prove

1. **JSON array deserialization** — `signals` column stores `[]SignalInput`, not `map[string]string`. This is structurally different from anything proven so far.
2. **Two JSON columns in one family** — writer writes two `marshalJSON()` calls; reader must parse both independently with appropriate fallbacks.
3. **Family-specific query parameter** — `outcome` is the first query parameter that exists only for one family. This tests whether the handler pattern accommodates per-family filtering without architectural changes.
4. **Constructor at 3 use cases** — confirms whether the current argument-list constructor is still viable or whether struct-based DI (H-1) must be accelerated.
5. **Pattern repeatability** — 9-artifact template works again without structural deviation.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| JSON array parse failure on malformed `signals` data | Low | Low | Silent fallback to empty slice, unit test coverage |
| Constructor with 3 use cases becomes unwieldy | Medium | Low | Tolerate for F02; H-1 committed for F03 |
| `outcome` filter introduces query builder divergence | Low | Medium | Single optional WHERE clause; does not restructure builder |
| New frictions exceed S167's 2-friction threshold | Low | High | Stop condition enforced; expansion pauses if exceeded |
| `confidence` type mismatch (Float64 in CH, string in Go) | Low | Medium | Explicit conversion in reader with unit test |

---

## 7. Constraints and Guard Rails

### Active Constraints

- **C-1:** One family per iteration — Decisions only
- **C-7:** No horizontal redesign
- **C-9:** Additive only — zero changes to candle or signal artifacts
- **S167-1:** Follow pattern v2 exactly (9 artifacts, CI gate, 5-point review)
- **S167-2:** Must pass its own gate before Family 03
- **S167-3:** Pause if >2 new frictions

### Guard Rails Enforced

- No more than one family opened
- No more than one endpoint (`/analytical/decision/history`)
- No anticipation of Family 03
- No horizontal redesign
- Clear out-of-scope documentation (13 non-goals enumerated)

---

## 8. Hardening Pre-commitments (Unchanged)

These remain committed for Family 03 and are NOT resolved in this iteration:

| ID | Commitment | Trigger |
|----|------------|---------|
| H-1 | Handler constructor → struct-based DI (`AnalyticalHandlerDeps`) | Family 03 entry |
| H-2 | Smoke test → extract `validate_analytical_family()` | Family 03 entry |
| H-3 | Rename `parseEvidenceKeyParams` → `parseAnalyticalKeyParams` | Family 03 entry |

---

## 9. Preparation for S169 (Implementation)

### 9.1 Pre-conditions Met

- [x] S167 gate passed (CONDITIONAL PASS)
- [x] Family formally defined with rationale
- [x] Schema exists (migration 003 applied)
- [x] Write path operational (mapper + consumer active)
- [x] 9-artifact checklist defined with file paths
- [x] Success criteria enumerated (30 criteria across 6 categories)
- [x] Non-goals explicitly documented (13 items)
- [x] Risks assessed with mitigations
- [x] Stop conditions defined

### 9.2 Implementation Sequence for S169

S169 must follow the left-to-right dependency chain:

1. **Contracts** — Add `DecisionHistoryQuery` and `DecisionHistoryReply` to `contracts.go`
2. **Reader** — Create `decision_reader.go` with `QueryDecisionHistory()` and `BuildDecisionQuery()`
3. **Reader tests** — Create `decision_reader_test.go` with 12 test cases
4. **Use case** — Create `get_decision_history.go` with validation and delegation
5. **Use case tests** — Create `get_decision_history_test.go` with 12 test cases
6. **Handler** — Extend `analytical.go` with `GetDecisionHistory()` method
7. **Handler tests** — Extend `analytical_test.go` with 6 test cases
8. **Routes** — Extend `analytical.go` routes with decision endpoint
9. **Composition** — Wire decision reader and use case in `compose.go`
10. **Integration test** — Extend `analytical.http` with decision requests
11. **Smoke test** — Extend `smoke-analytical-e2e.sh` with Phase 5c
12. **Documentation** — Schema coherence table, endpoint spec, known limits, friction log

### 9.3 Recommended Verification Before Starting S169

```bash
# Verify decisions table exists and has data
docker exec clickhouse clickhouse-client --query "SELECT count() FROM decisions"

# Verify writer is consuming decision events
curl -s localhost:8080/statusz | jq '.decision_rsi_oversold'

# Verify existing tests pass
go test ./internal/adapters/clickhouse/... ./internal/application/analyticalclient/...

# Verify CI is green
gh run list --limit 1
```

---

## 10. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Second family clearly defined | PASS — Decisions (RSI Oversold) with full rationale |
| Scope remains minimal and controlled | PASS — 9 artifacts, one endpoint, additive only |
| JSON payload has explicit rationale | PASS — Section 4 of definition doc covers both JSON columns |
| Success and non-success criteria are objective | PASS — 30 verifiable criteria, 13 enumerated non-goals |
| Base ready for S169 implementation | PASS — Pre-conditions met, sequence defined, verification commands provided |
| Guard rails enforced | PASS — One family, one endpoint, no Family 03 anticipation, no redesign |

**Stage S168 verdict: COMPLETE. S169 may proceed.**
