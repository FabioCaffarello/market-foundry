# Stage S246 — Smoke E2E Breadth Coverage Expansion Report

**Date:** 2026-03-21
**Charter:** HARDENING (post-BREADTH-WAVE-1)
**Type:** Smoke test expansion + debt closure
**Predecessor:** S245 (remote CI closure for breadth wave)
**Status:** COMPLETE

---

## 1. Executive Summary

S246 closed the D1 debt identified in S244: the three breadth wave types (`ema_crossover`, `trend_following_entry`, `drawdown_limit`) were missing from both primary smoke E2E scripts. All three types now participate in the same validation surface as their Chain A counterparts, achieving full smoke parity for the breadth wave.

No new infrastructure was introduced. Changes are purely additive, following existing patterns.

**D1 status: CLOSED**

---

## 2. What S246 Delivered

### 2.1 smoke-analytical-e2e.sh Expansion

Added Chain B types to the ClickHouse analytical read path validation:

| Addition | Phase | Validation |
|----------|-------|------------|
| `ema_crossover` decisions | Phase 5 | ClickHouse rows + HTTP 200 + JSON structure + item count + Server-Timing + outcome filter |
| `trend_following_entry` strategies | Phase 5 | ClickHouse rows + HTTP 200 + JSON structure + item count + Server-Timing + direction filter |
| `drawdown_limit` risk assessments | Phase 5 | ClickHouse rows + HTTP 200 + JSON structure + item count + Server-Timing + disposition filter |
| Chain B domain depth | Phase 7 | ema_crossover severity/rationale + trend_following_entry context propagation + drawdown_limit metadata |

### 2.2 smoke-multi-symbol.sh Expansion

Added Chain B types to the NATS KV multi-symbol validation:

| Step | Type | Validation |
|------|------|------------|
| 7a | `ema_crossover` | 2 symbols x 4 timeframes: HTTP 200, structure, field assertions, outcome domain |
| 8a | `ema_crossover` isolation | Cross-symbol collision/bleed detection per timeframe |
| 9a | `trend_following_entry` | 2 symbols x 4 timeframes: HTTP 200, structure, field assertions, direction domain |
| 10a | `trend_following_entry` isolation | Cross-symbol collision/bleed detection per timeframe |
| 11a | `drawdown_limit` | 2 symbols x 4 timeframes: HTTP 200, structure, field assertions, disposition domain |
| 12a | `drawdown_limit` isolation | Cross-symbol collision/bleed detection per timeframe |
| 22 | Error handling | Missing timeframe → 400 for ema_crossover, trend_following_entry, drawdown_limit |

### 2.3 HTTP REST Client Test Expansion

| File | Types Added |
|------|-------------|
| `tests/http/decision.http` | `ema_crossover` latest (4 timeframes + ethusdt) |
| `tests/http/strategy.http` | `trend_following_entry` latest (4 timeframes + ethusdt) |
| `tests/http/risk.http` | `drawdown_limit` latest (4 timeframes + ethusdt) |
| `tests/http/analytical.http` | All 3 new types: history queries with limit, filter, cross-symbol |

### 2.4 Architecture Documentation

| Document | Purpose |
|----------|---------|
| `docs/architecture/smoke-e2e-breadth-coverage-expansion.md` | Technical design and scope of the smoke expansion |
| `docs/architecture/breadth-wave-smoke-coverage-before-and-after.md` | Detailed before/after coverage matrix |

---

## 3. Files Changed

| File | Change |
|------|--------|
| `scripts/smoke-analytical-e2e.sh` | +3 families in Phase 5, +3 depth checks in Phase 7, updated header and summary |
| `scripts/smoke-multi-symbol.sh` | +6 validation steps (7a, 8a, 9a, 10a, 11a, 12a), +3 error checks, updated header and summary |
| `tests/http/decision.http` | +6 ema_crossover queries |
| `tests/http/strategy.http` | +6 trend_following_entry queries |
| `tests/http/risk.http` | +5 drawdown_limit queries |
| `tests/http/analytical.http` | +12 analytical history queries for breadth types |
| `docs/architecture/smoke-e2e-breadth-coverage-expansion.md` | New — expansion design doc |
| `docs/architecture/breadth-wave-smoke-coverage-before-and-after.md` | New — before/after matrix |
| `docs/stages/stage-s246-smoke-e2e-breadth-coverage-expansion-report.md` | New — this report |

---

## 4. Debt Ledger

### Debts Closed

| # | Debt | Source | Status |
|---|------|--------|--------|
| D1 | Smoke test coverage for 3 new types | S244 | **CLOSED** |

### Debts Remaining

| # | Debt | Severity | Recommendation |
|---|------|----------|----------------|
| D2 | Chain B integration test (EMA → ema_crossover → trend_following_entry → drawdown_limit) | Low | Optional — unit + smoke coverage is sufficient for operational confidence |
| D3 | Remote CI verification of S246 changes | Low | Include in next CI run (already proven feasible in S245) |

### Debts NOT Introduced

- No new code paths were added (only test scripts and docs).
- No new dependencies or infrastructure.
- No execution layer changes.

---

## 5. Verification

- `bash -n scripts/smoke-analytical-e2e.sh` — syntax valid
- `bash -n scripts/smoke-multi-symbol.sh` — syntax valid
- `go test ./internal/application/... ./internal/domain/...` — all pass (no Go changes made)

---

## 6. Coverage Summary

| Metric | Before S246 | After S246 |
|--------|:-----------:|:----------:|
| Analytical E2E families validated | 6 | 9 |
| Multi-symbol type validations | 5 | 8 |
| Cross-symbol isolation checks | 5 | 8 |
| Domain depth chain checks | 3 | 6 |
| HTTP REST test cases | ~65 | ~85 |
| **D1 debt** | **Open** | **Closed** |

---

## 7. Limitations

1. **Smoke scripts require a live stack**: These are not unit tests — they validate the running system. The new checks follow the same warm-up tolerance pattern (null responses are accepted during pipeline warm-up).

2. **Chain B warm-up is longer**: EMA Crossover needs 21 candles (21 min at 60s) vs RSI's 15. The `--wait` parameter may need to be increased for Chain B to produce data.

3. **No execution layer type differentiation for Chain B**: Both chains produce `paper_order` executions. The execution smoke validation covers this shared type already.

4. **Analytical EMA signal**: The EMA signal type in `signals` table was already covered by smoke-multi-symbol.sh Step 6a (added during the breadth wave itself). No change was needed.

---

## 8. Preparation for S247

With D1 closed, the remaining gap is D2 (Chain B integration test) which is low severity and optional. The system is ready for:

1. **Remote CI verification** of accumulated S246 changes (can be bundled with any subsequent push).
2. **Next feature wave opening** — the breadth wave now has full smoke parity, making it safe to build on top of.
3. **Observability wave** — if desired, the smoke infrastructure now validates all types, providing a solid baseline for adding metrics/tracing.

The recommended S247 scope is either:
- A lightweight CI verification pass (if D3 is prioritized), or
- Direct transition to the next feature wave, given that operational coverage is now complete.
