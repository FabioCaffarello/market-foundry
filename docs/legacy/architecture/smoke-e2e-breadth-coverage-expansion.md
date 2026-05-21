# Smoke E2E Breadth Coverage Expansion

**Stage:** S246
**Date:** 2026-03-21
**Context:** Closes D1 debt from S244 (breadth wave gate)

---

## 1. Problem Statement

The BREADTH-WAVE-1 charter (S240-S244) introduced 3 new types across decision, strategy, and risk domains:

| Domain | Existing Type (Chain A) | New Type (Chain B) | Stage |
|--------|------------------------|--------------------|-------|
| Decision | `rsi_oversold` | `ema_crossover` | S241 |
| Strategy | `mean_reversion_entry` | `trend_following_entry` | S242 |
| Risk | `position_exposure` | `drawdown_limit` | S243 |

These types were fully unit-tested and integration-tested, but the two primary smoke E2E scripts (`smoke-analytical-e2e.sh` and `smoke-multi-symbol.sh`) only validated Chain A types. This meant the breadth wave lacked operational proof at the smoke level.

## 2. Scope of Expansion

### 2.1 smoke-analytical-e2e.sh (ClickHouse analytical path)

**Before:** Validated 6 analytical families (candles, signals, rsi_oversold, mean_reversion_entry, position_exposure, paper_order).

**After:** Validates 9 analytical families, adding:
- `ema_crossover` decisions via `/analytical/decision/history?type=ema_crossover`
- `trend_following_entry` strategies via `/analytical/strategy/history?type=trend_following_entry`
- `drawdown_limit` risk assessments via `/analytical/risk/history?type=drawdown_limit`

Each new family receives the same validation depth as existing families:
1. ClickHouse row count verification
2. HTTP endpoint returns 200
3. Response JSON structure validation (source=clickhouse, meta, required fields)
4. Item count > 0 check
5. Server-Timing header presence
6. Filter validation (outcome, direction, disposition)

**Phase 7 expansion:** Added Chain B domain depth validation:
- `ema_crossover` severity/rationale propagation
- `trend_following_entry` → decision context propagation
- `drawdown_limit` → decision severity in metadata

### 2.2 smoke-multi-symbol.sh (NATS KV latest path)

**Before:** Validated Chain A types across 2 symbols x 4 timeframes.

**After:** Adds Chain B validation steps:
- Step 7a: Decision `ema_crossover` multi-symbol validation
- Step 8a: Cross-symbol `ema_crossover` decision isolation
- Step 9a: Strategy `trend_following_entry` multi-symbol validation
- Step 10a: Cross-symbol `trend_following_entry` strategy isolation
- Step 11a: Risk `drawdown_limit` multi-symbol validation
- Step 12a: Cross-symbol `drawdown_limit` risk isolation

Each new step follows the identical validation pattern as its Chain A counterpart:
- HTTP 200 check
- Response structure validation (required fields, type assertion, value domain)
- Cross-symbol isolation (collision, bleed detection)
- Error handling (missing timeframe → 400)

### 2.3 HTTP REST Client Tests (tests/http/)

Added breadth type coverage to manual REST client files:
- `decision.http`: Added `ema_crossover` latest queries (4 timeframes + ethusdt)
- `strategy.http`: Added `trend_following_entry` latest queries (4 timeframes + ethusdt)
- `risk.http`: Added `drawdown_limit` latest queries (4 timeframes + ethusdt)
- `analytical.http`: Added analytical history queries for all 3 new types (basic, limit, filter, cross-symbol)

## 3. Design Decisions

### 3.1 Additive, Not Restructuring

New validations were inserted alongside existing ones using the exact same patterns. No refactoring of existing smoke infrastructure was needed or attempted.

### 3.2 Warm-up Awareness

Chain B types have a longer warm-up requirement (EMA needs 21 candles vs RSI's 15). Smoke scripts handle this gracefully — null responses during warm-up are accepted as passing, matching the existing Chain A behavior.

### 3.3 No New Infrastructure

No new helper functions, no new scripts, no new test frameworks. The existing `validate_analytical_family()` helper and per-type validation pattern were sufficient.

## 4. What This Does NOT Cover

- **Chain B integration test** (D2 from S244): Full NATS-level chain integration test for EMA → ema_crossover → trend_following_entry → drawdown_limit remains optional.
- **Execution types for Chain B**: The execution layer (paper_order) is shared across both chains. No new execution type was introduced in the breadth wave.
- **Analytical signal EMA**: The EMA signal (`ema_crossover` in signals table) was already covered by the existing smoke-multi-symbol.sh Step 6a. No change needed.
