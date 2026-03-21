# Breadth Wave Smoke Coverage — Before and After

**Stage:** S246
**Date:** 2026-03-21

---

## 1. smoke-analytical-e2e.sh (ClickHouse Path)

### Phase 5: Per-Family Analytical Read Path Validation

| Family | Type | Before S246 | After S246 |
|--------|------|:-----------:|:----------:|
| Candles (Baseline) | — | Covered | Covered |
| Signals/RSI (Wave B F-01) | `rsi` | Covered | Covered |
| Decisions (Wave B F-02) | `rsi_oversold` | Covered | Covered |
| **Decisions (Breadth S241)** | **`ema_crossover`** | **Not covered** | **Covered** |
| Strategies (Wave B F-03) | `mean_reversion_entry` | Covered | Covered |
| **Strategies (Breadth S242)** | **`trend_following_entry`** | **Not covered** | **Covered** |
| Risk (Wave B F-04) | `position_exposure` | Covered | Covered |
| **Risk (Breadth S243)** | **`drawdown_limit`** | **Not covered** | **Covered** |
| Executions (Wave B F-05) | `paper_order` | Covered | Covered |

### Phase 5: Filter Validation

| Type | Filter | Before S246 | After S246 |
|------|--------|:-----------:|:----------:|
| `rsi_oversold` | `outcome=triggered` | Covered | Covered |
| **`ema_crossover`** | **`outcome=triggered`** | **Not covered** | **Covered** |
| `mean_reversion_entry` | `direction=long` | Covered | Covered |
| **`trend_following_entry`** | **`direction=long`** | **Not covered** | **Covered** |
| `position_exposure` | `disposition=approved` | Covered | Covered |
| **`drawdown_limit`** | **`disposition=approved`** | **Not covered** | **Covered** |

### Phase 7: Domain Depth Validation

| Chain | Depth Check | Before S246 | After S246 |
|-------|-------------|:-----------:|:----------:|
| Chain A | `rsi_oversold` severity/rationale | Covered | Covered |
| Chain A | `mean_reversion_entry` → decision context | Covered | Covered |
| Chain A | `position_exposure` → decision severity in metadata | Covered | Covered |
| **Chain B** | **`ema_crossover` severity/rationale** | **Not covered** | **Covered** |
| **Chain B** | **`trend_following_entry` → decision context** | **Not covered** | **Covered** |
| **Chain B** | **`drawdown_limit` → decision severity in metadata** | **Not covered** | **Covered** |

---

## 2. smoke-multi-symbol.sh (NATS KV Path)

### Multi-Symbol x Multi-Timeframe Validation

| Step | Domain | Type | Before S246 | After S246 |
|------|--------|------|:-----------:|:----------:|
| 5 | Signal | `rsi` | Covered | Covered |
| 6a | Signal | `ema_crossover` | Covered | Covered |
| 7 | Decision | `rsi_oversold` | Covered | Covered |
| **7a** | **Decision** | **`ema_crossover`** | **Not covered** | **Covered** |
| 8 | Decision | `rsi_oversold` isolation | Covered | Covered |
| **8a** | **Decision** | **`ema_crossover` isolation** | **Not covered** | **Covered** |
| 9 | Strategy | `mean_reversion_entry` | Covered | Covered |
| **9a** | **Strategy** | **`trend_following_entry`** | **Not covered** | **Covered** |
| 10 | Strategy | `mean_reversion_entry` isolation | Covered | Covered |
| **10a** | **Strategy** | **`trend_following_entry` isolation** | **Not covered** | **Covered** |
| 11 | Risk | `position_exposure` | Covered | Covered |
| **11a** | **Risk** | **`drawdown_limit`** | **Not covered** | **Covered** |
| 12 | Risk | `position_exposure` isolation | Covered | Covered |
| **12a** | **Risk** | **`drawdown_limit` isolation** | **Not covered** | **Covered** |
| 13-21 | Execution, Fill, Status, Control, Trace | — | Covered | Covered |

### Error Handling Validation (Step 22)

| Domain | Type | Check | Before S246 | After S246 |
|--------|------|-------|:-----------:|:----------:|
| Decision | `rsi_oversold` | missing timeframe → 400 | Covered | Covered |
| **Decision** | **`ema_crossover`** | **missing timeframe → 400** | **Not covered** | **Covered** |
| Strategy | `mean_reversion_entry` | missing timeframe → 400 | Covered | Covered |
| **Strategy** | **`trend_following_entry`** | **missing timeframe → 400** | **Not covered** | **Covered** |
| Risk | `position_exposure` | missing timeframe → 400 | Covered | Covered |
| **Risk** | **`drawdown_limit`** | **missing timeframe → 400** | **Not covered** | **Covered** |

---

## 3. HTTP REST Client Tests (tests/http/)

| File | Type | Before S246 | After S246 |
|------|------|:-----------:|:----------:|
| `decision.http` | `rsi_oversold` (4tf + ethusdt + errors) | Covered | Covered |
| `decision.http` | **`ema_crossover`** (4tf + ethusdt) | **Not covered** | **Covered** |
| `strategy.http` | `mean_reversion_entry` (4tf + ethusdt + errors) | Covered | Covered |
| `strategy.http` | **`trend_following_entry`** (4tf + ethusdt) | **Not covered** | **Covered** |
| `risk.http` | `position_exposure` (4tf + ethusdt) | Covered | Covered |
| `risk.http` | **`drawdown_limit`** (4tf + ethusdt) | **Not covered** | **Covered** |
| `analytical.http` | Baseline + Wave B types | Covered | Covered |
| `analytical.http` | **`ema_crossover` history** | **Not covered** | **Covered** |
| `analytical.http` | **`trend_following_entry` history** | **Not covered** | **Covered** |
| `analytical.http` | **`drawdown_limit` history** | **Not covered** | **Covered** |

---

## 4. Coverage Summary

| Metric | Before S246 | After S246 | Delta |
|--------|:-----------:|:----------:|:-----:|
| Analytical E2E families | 6 | 9 | +3 |
| Analytical filter checks | 3 | 6 | +3 |
| Domain depth chain checks | 3 (Chain A) | 6 (Chain A + B) | +3 |
| Multi-symbol type validations | 5 types | 8 types | +3 |
| Cross-symbol isolation checks | 5 types | 8 types | +3 |
| Error handling checks (multi-symbol) | 12 | 15 | +3 |
| HTTP REST client test cases | ~65 | ~85 | +20 |
| **D1 debt status** | **Open (Medium)** | **Closed** | — |
