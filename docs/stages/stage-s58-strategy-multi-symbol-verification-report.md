# Stage S58 — Strategy Multi-Symbol Verification Report

**Status:** Complete
**Date:** 2026-03-18
**Scope:** Validate `strategy` domain behavior under controlled multi-symbol scenario

---

## 1. Executive Summary

Stage S58 validates that the `strategy` domain (family: `mean_reversion_entry`) behaves correctly under a multi-symbol scenario (btcusdt, ethusdt × 60s, 300s). The verification confirms:

- **Partition key isolation**: Each symbol×timeframe combination produces a unique KV key (`{source}.{symbol}.{timeframe}`), preventing cross-symbol bleed.
- **Deduplication key isolation**: Deduplication keys include the symbol, ensuring JetStream deduplication operates per-symbol.
- **Projection materialization**: The `StrategyProjectionActor` correctly materializes strategies for multiple symbols independently, with accurate stats tracking.
- **Query surface isolation**: HTTP handler returns the correct symbol in each response, with no cross-contamination.
- **Smoke test coverage**: `smoke-multi-symbol.sh` now validates strategy endpoints (Steps 9-10) with the same rigor applied to evidence, signal, and decision.

No structural issues, ownership confusion, or cross-symbol bleed were found. The `strategy` domain is multi-symbol ready.

---

## 2. Multi-Symbol Scenario Validated

| Dimension | Values |
|-----------|--------|
| **Source** | `binancef` |
| **Symbols** | `btcusdt`, `ethusdt` |
| **Timeframes** | 60s, 300s |
| **Strategy family** | `mean_reversion_entry` |
| **KV entries expected** | 4 (2 symbols × 2 timeframes) |

### Dependency Chain (per symbol)
```
Evidence (candle) → Signal (RSI) → Decision (RSI Oversold) → Strategy (Mean Reversion Entry)
```

Each layer independently processes per-symbol data. Strategy inherits the multi-symbol behavior proven in S17 (evidence), S41 (signal), and S46 (decision).

### Partition Key Format
```
binancef.btcusdt.60
binancef.btcusdt.300
binancef.ethusdt.60
binancef.ethusdt.300
```

All 4 keys are unique — no collision or overlap possible by construction.

---

## 3. Files Changed

### New Tests Added

| File | Test | Purpose |
|------|------|---------|
| `internal/actors/scopes/store/strategy_projection_actor_test.go` | `TestStrategyProjection_MultiSymbol_IndependentMaterialization` | 2 symbols × 2 timeframes → 4 independent materializations, stats, tracker |
| `internal/actors/scopes/store/strategy_projection_actor_test.go` | `TestStrategyProjection_MultiSymbol_NoBleed_PartitionKeys` | 3 symbols × 2 timeframes → 6 unique partition keys, collision detection |
| `internal/actors/scopes/store/strategy_projection_actor_test.go` | `TestStrategyProjection_MultiSymbol_DeduplicationKeys` | 2 symbols at same timestamp → unique dedup keys |
| `internal/interfaces/http/handlers/strategy_test.go` | `TestStrategyWebHandler_GetLatestStrategy_MultiSymbol_NoBleed` | Queries for btcusdt/ethusdt verify correct symbol+direction in response |

### Smoke Test Updated

| File | Change |
|------|--------|
| `scripts/smoke-multi-symbol.sh` | Added **Step 9**: Strategy multi-symbol validation (2 symbols × 2 timeframes, structure + field assertions) |
| `scripts/smoke-multi-symbol.sh` | Added **Step 10**: Cross-symbol strategy isolation (COLLISION, BLEED_A, BLEED_B detection) |
| `scripts/smoke-multi-symbol.sh` | Added **Step 11**: Strategy error handling (unknown type → 400, missing timeframe → 400) |
| `scripts/smoke-multi-symbol.sh` | Updated header, summary, and flow diagram to include strategy |

### HTTP Test File Updated

| File | Change |
|------|--------|
| `tests/http/strategy.http` | Added all 4 symbol×timeframe combinations + error cases (unknown type, missing timeframe, missing all params) |

---

## 4. Problems Found or Discarded

### Found: None

- **Cross-symbol bleed**: Not present. Partition keys are structurally unique by construction (`{source}.{symbol}.{timeframe}`).
- **Ownership confusion**: Not present. Strategy events include source/symbol/timeframe in subject wildcard (`strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}`), ensuring per-symbol routing.
- **Deduplication collision**: Not present. Dedup keys include symbol (`strat:mean_reversion_entry:{source}:{symbol}:{timeframe}:{timestamp}`).
- **Projection race**: Not applicable. Store consumes from a single durable consumer and dispatches to a single projection actor — no concurrent write paths for the same KV bucket.
- **Config asymmetry**: Not present. Both `derive.jsonc` and `store.jsonc` declare `strategy_families: ["mean_reversion_entry"]`.

### Pre-existing Issue (Not S58 Scope)

- `TestCompileUseCaseBuildsDefaultArtifactMetadata` in `internal/application/configctl` fails. This is unrelated to strategy and predates S58.

---

## 5. Impact on Readiness for S59

### Strengthened Confidence

| Aspect | Status | Notes |
|--------|--------|-------|
| Multi-symbol activation | Proven | Symbols activated via configctl binding → derive binding-watcher → per-symbol actor tree |
| Multi-symbol projection | Proven | Independent KV entries per symbol×timeframe |
| Multi-symbol query surface | Proven | HTTP endpoint returns correct symbol with no bleed |
| Config coherence | Proven | derive/store configs symmetric; activation is opt-in |
| Cross-symbol isolation | Proven | Partition keys, dedup keys, and NATS subjects are structurally unique |

### Prerequisites for Risk Domain (S59+)

The `strategy` domain now satisfies the multi-symbol verification gate:

1. **Strategy produces correct per-symbol data** — partition keys isolate state.
2. **Strategy projections are latest-only and per-symbol** — no history, no aggregation across symbols.
3. **Smoke tests cover strategy in multi-symbol scenario** — regression protection in place.
4. **No workarounds or manual patches** — all isolation is structural (by key format, subject pattern, and actor-per-symbol topology).

The `risk` domain can safely consume strategy data knowing that each symbol's strategy resolution is independent and correctly scoped.

---

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No new strategy families opened | Compliant — `mean_reversion_entry` only |
| No history expansion | Compliant — latest-only projections |
| No `risk` implementation | Compliant — S58 is verification-only |
| No manual workarounds | Compliant — all isolation is structural |
| Limitations documented | No limitations found |
