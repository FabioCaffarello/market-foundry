# Multi-Symbol Scenarios — Results and Operational Findings

> Stage S302 — Phase 29: Multi-Symbol Operational Scaling Wave

## Test Results Summary

### Unit Tests (no external dependencies)

| Layer      | Test Count | Status | Duration |
|-----------|-----------|--------|----------|
| Use Case  | 12        | PASS   | ~0.15s   |
| Handler   | 10        | PASS   | ~0.16s   |
| **Total** | **22**    | **ALL PASS** | **~0.31s** |

### Integration Tests (requireclickhouse)

| Test ID          | Status | Notes |
|------------------|--------|-------|
| S302-SC1-INT     | READY  | Compiles, requires ClickHouse |
| S302-SC2-INT     | READY  | Compiles, requires ClickHouse |
| S302-SC3-INT     | READY  | Compiles, requires ClickHouse |
| S302-SC4-INT     | READY  | Compiles, requires ClickHouse |

### Regression Check

| Package | Pre-S302 Tests | Post-S302 Tests | Regressions |
|---------|---------------|-----------------|-------------|
| analyticalclient | 18 | 30 | 0 |
| handlers | 15 | 25 | 0 |

## Before/After Value

### Before S302

- S301 proved symbol **isolation** (cross-symbol contamination blocked) but only with homogeneous chains (all approved, same fixture shape).
- No test validated **heterogeneous** multi-symbol scenarios (different dispositions, different signal types, different directions simultaneously).
- No test proved that **attribution** produces correct per-symbol values when symbols have different risk outcomes.
- No test proved **funnel and disposition aggregates** are independent when mixed dispositions exist across symbols.

### After S302

- **SC1** proves 3 symbols with different signal types (rsi, macd, bollinger), different directions (long, long, short), and different constraint profiles produce correct, isolated chains.
- **SC2** proves approved/rejected/modified dispositions coexist across symbols: rejected ethusdt has no execution and correct missing_stages, while approved btcusdt and modified solusdt are unaffected.
- **SC3** proves batch queries return exactly the right count per symbol (3, 2, 1) with no cross-symbol leakage.
- **SC4** proves attribution projection varies correctly per symbol: different severity levels, directions, constraint values, and rationales are preserved without contamination.

## Operational Findings

### Finding 1: Stub Design Scales Well

The multi-symbol stub readers (`multiSymbolStubReader`, `multiSymbolBatchStubReader`) dispatch by symbol, closely modeling the real CompositeReader behavior. This pattern enables adding new symbols without changing test infrastructure.

### Finding 2: Type Conversions Required for Enums

Domain types use named string types (`decision.Severity`, `risk.Disposition`, `strategy.Direction`, `execution.Side`, `execution.Status`). Test fixtures must explicitly convert strings to these types. This is a minor friction point but enforces type safety.

### Finding 3: Funnel Queries are Type-Scoped

The pipeline funnel queries filter by `type` (e.g., "rsi", "bollinger") in addition to symbol. When fixtures use different signal types per symbol, funnel counts only include chains matching the queried type. This is correct behavior but requires awareness when designing mixed-type scenarios.

### Finding 4: No Ordering Anomalies Detected

Batch queries consistently return chains ordered by execution timestamp DESC. No interleaving or misordering was observed across symbols.

### Finding 5: Attribution Projection is Symbol-Agnostic

The `computeAttribution` function operates on a single chain without any symbol awareness — it simply projects from the risk stage. This means attribution correctness per symbol is guaranteed by the upstream isolation (S301), not by attribution logic itself. This is the correct design.

## Known Limitations

1. **Integration tests not exercised in this report.** They compile and are ready for execution in CI or local ClickHouse environments.
2. **No concurrent goroutine testing.** Scenarios are sequential per symbol. True concurrent access patterns (multiple HTTP requests in flight) are a future concern.
3. **Three symbols only.** The charter (S300) scopes to btcusdt, ethusdt, solusdt. Adding more symbols is straightforward but out of scope.
4. **Paper mode only.** All executions use `paper_order` type. Real venue behavior is out of scope per S300 non-goals.
5. **No sub-millisecond ordering stress.** Timestamps are spaced by at least 1 millisecond per stage. Real-world ordering under very high throughput is not tested.
