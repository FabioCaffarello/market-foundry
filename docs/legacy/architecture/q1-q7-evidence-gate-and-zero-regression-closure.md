# Q1–Q7 Evidence Gate and Zero Regression Closure

> Formal regression verification and gate closure for the Composite Execution Observability Wave (S294–S298).
> Gate: S299 — Zero regression certification.
> Date: 2026-03-21

---

## 1. Regression Verification Scope

This document verifies zero regression across three dimensions:

1. **Query surface** — existing analytical endpoints remain functional and unbroken
2. **Composite read model** — new composition does not corrupt or interfere with domain readers
3. **Attribution** — read-side projection does not mutate or degrade write-side data

---

## 2. Query Surface Regression Assessment

### Pre-Wave Endpoints (unchanged)

The gateway already exposed analytical endpoints for individual domain families (candles, signals, decisions, strategies, risk assessments, executions) via dedicated readers. These remain untouched.

| Concern | Status | Evidence |
|---------|--------|----------|
| Existing candle/signal/decision/strategy/risk/execution readers | **NO REGRESSION** | `composite_reader.go` is a new file; existing `*_reader.go` files unmodified |
| Route registration | **NO REGRESSION** | `analytical.go` adds new routes additively; conditional registration prevents breakage if ClickHouse unavailable |
| Gateway composition | **NO REGRESSION** | `compose.go` adds new use cases without modifying existing wiring; `buildRouteDependencies` extended, not rewritten |
| Gateway build | **NO REGRESSION** | `go build ./cmd/gateway/...` succeeds cleanly |

### New Endpoints (S297–S298)

| Endpoint | Regression Risk | Mitigation | Status |
|----------|----------------|------------|--------|
| `/analytical/composite/chain` | Low — new path, no overlap | 8 handler tests | **CLEAN** |
| `/analytical/composite/chains` | Low — new path, no overlap | Handler tests cover success + error cases | **CLEAN** |
| `/analytical/composite/funnel` | Low — new path, no overlap | 3 handler tests | **CLEAN** |
| `/analytical/composite/dispositions` | Low — new path, no overlap | 3 handler tests | **CLEAN** |

### Route Isolation Verification

New composite routes are registered under `/analytical/composite/*` prefix — structurally isolated from existing `/analytical/*` domain routes. No route collision possible.

---

## 3. Composite Read Model Regression Assessment

### Adapter Layer

| Concern | Status | Evidence |
|---------|--------|----------|
| New CompositeReader vs existing domain readers | **NO REGRESSION** | CompositeReader is a new type; does not modify or replace existing SignalReader, DecisionReader, etc. |
| Shared ClickHouse client | **NO REGRESSION** | CompositeReader receives same `*clickhouse.Client` but opens no new connections; uses existing query infrastructure |
| Query interference | **NO REGRESSION** | All composite queries are SELECT-only; no writes, no DDL, no schema changes |
| Table access patterns | **NO REGRESSION** | Queries use same WHERE patterns as domain readers (source, symbol, timeframe, correlation_id) — all within MergeTree order key |

### Application Layer

| Concern | Status | Evidence |
|---------|--------|----------|
| New use cases vs existing use cases | **NO REGRESSION** | `GetCompositeChainUseCase`, `GetPipelineFunnelUseCase`, `GetDispositionBreakdownUseCase` are new types; existing use cases unmodified |
| Contract types | **NO REGRESSION** | `composite_contracts.go` defines new types; no modifications to existing domain contract types |
| Interface segregation | **NO REGRESSION** | CompositeReader satisfies both `CompositeReader` and `AggregationReader` interfaces defined in analyticalclient; existing reader interfaces untouched |

### Test Isolation

| Test Suite | Pre-Wave | Post-Wave | Delta | Status |
|------------|----------|-----------|-------|--------|
| `internal/adapters/clickhouse/...` | existing tests | existing + 4 new composite unit + 6 integration | additive only | **PASS** |
| `internal/application/analyticalclient/...` | existing tests | existing + 21 new composite tests | additive only | **PASS** |
| `internal/interfaces/http/handlers/...` | existing tests | existing + 15 new composite tests | additive only | **PASS** |

All test suites pass: `go test ./internal/application/analyticalclient/... ./internal/interfaces/http/handlers/... ./internal/adapters/clickhouse/... -count=1 -short` — **4 packages, ALL OK**.

---

## 4. Attribution Regression Assessment

### Write-Side Integrity

| Concern | Status | Evidence |
|---------|--------|----------|
| Risk domain schema | **NO MODIFICATION** | `internal/domain/risk/risk.go` — `RiskAssessment`, `Constraints`, `StrategyInput` types unchanged |
| Risk actor behavior | **NO MODIFICATION** | No changes to risk assessment actors or write paths |
| ClickHouse schema | **NO MODIFICATION** | No new migrations; no ALTER TABLE; existing table schemas intact |
| Event contracts | **NO MODIFICATION** | NATS event envelopes, subject patterns, codec — all unchanged |

### Attribution as Pure Projection

The `computeAttribution()` function in `get_composite_chain.go` is a **pure read-side projection**:
- Input: `CompositeExecutionChain` (already queried)
- Output: `RiskAttribution` struct populated from existing Risk stage fields
- Side effects: **NONE** — no writes, no mutations, no state changes
- Failure mode: if risk stage absent, attribution is nil (graceful)

This design ensures attribution cannot cause regression by construction.

---

## 5. raccoon-cli Regression Assessment

| Concern | Status | Evidence |
|---------|--------|----------|
| Contract audit checks | **NO REGRESSION** | `contracts.rs` changes are additive (new checks, not modified existing) |
| CLI integration tests | **97 tests, ALL PASS** | `cli_integration.rs` and `validation_matrix.rs` — comprehensive coverage |
| Exit code contract | **PRESERVED** | 0=pass, 1=fail, 2=runtime error — verified by 8 dedicated tests |
| JSON output schema | **PRESERVED** | 18 schema contract tests pass |

---

## 6. Regression Verdict

| Dimension | Verdict | Confidence |
|-----------|---------|------------|
| Query surface | **ZERO REGRESSION** | High — additive routes only, no overlap with existing paths |
| Composite read model | **ZERO REGRESSION** | High — new types and files, existing readers untouched |
| Attribution | **ZERO REGRESSION** | High — pure projection, no write-side changes |
| Domain schemas | **ZERO REGRESSION** | High — no migrations, no schema changes |
| Build & tests | **ZERO REGRESSION** | High — gateway builds, all test suites pass |
| raccoon-cli | **ZERO REGRESSION** | High — 97 integration tests pass, contract preserved |

### Overall Regression Status: **ZERO RELEVANT REGRESSION DETECTED**

---

## 7. Gate Evidence Checklist

| # | Gate Criterion | Met? |
|---|----------------|------|
| 1 | Q1–Q7 audited with concrete evidence | YES — see `q1-q7-answerability-evidence-matrix-and-residual-gaps.md` |
| 2 | Each question has identified endpoint(s) | YES — 4 endpoints cover all 7 questions |
| 3 | Residual gaps are explicit and bounded | YES — GAP-Q2-A (per-constraint trigger), GAP-Q5-A (pre-execution chain discovery) |
| 4 | Zero regression on query surface | YES — additive changes only |
| 5 | Zero regression on composite read model | YES — new types, existing readers untouched |
| 6 | Zero regression on write-side / attribution | YES — pure read-side projection |
| 7 | All test suites pass | YES — 36+ composite tests, 97 CLI tests, gateway builds |
| 8 | Wave scope respected (read-side only, no NG violations) | YES — no write-side changes, no schema mutations |

**All 8 gate criteria met.**
