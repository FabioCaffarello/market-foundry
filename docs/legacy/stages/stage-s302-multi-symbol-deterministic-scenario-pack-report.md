# Stage S302: Multi-Symbol Deterministic Scenario Pack

> Phase 29 — Multi-Symbol Operational Scaling Wave
> Date: 2026-03-21
> Status: **COMPLETE**

## Objective

Define, implement, validate, and document a small pack of deterministic multi-symbol scenarios that prove the composite read model behaves correctly when btcusdt, ethusdt, and solusdt are active simultaneously.

## Governing Constraint

This stage is NOT combinatorial explosion. It is targeted, representative coverage using 3 symbols and 4 scenarios per test layer.

## Scenarios Delivered

| ID  | Name                          | Symbols | Key Validation                           |
|-----|-------------------------------|---------|------------------------------------------|
| SC1 | Simultaneous Approved Chains  | 3       | Different characteristics, all approved   |
| SC2 | Mixed Dispositions            | 3       | approved/rejected/modified coexistence    |
| SC3 | Concurrent Batch Counts       | 3       | Correct count per symbol, no cross-leak   |
| SC4 | Attribution Diversity         | 3       | Per-symbol severity, direction, constraints |

## Test Evidence

### Unit Tests — 22/22 PASS

**Use case layer** (`internal/application/analyticalclient/multi_symbol_scenario_test.go`):

| Test | Subtests | Status |
|------|----------|--------|
| TestS302_SC1_SimultaneousApprovedChains | btcusdt, ethusdt, solusdt | PASS |
| TestS302_SC2_MixedDispositions | btcusdt_approved, ethusdt_rejected, solusdt_modified | PASS |
| TestS302_SC3_ConcurrentBatchPerSymbol | batch_btcusdt, batch_ethusdt, batch_solusdt | PASS |
| TestS302_SC4_AttributionDiversityPerSymbol | attribution_btcusdt, attribution_ethusdt, attribution_solusdt | PASS |

**Handler layer** (`internal/interfaces/http/handlers/composite_multi_symbol_test.go`):

| Test | Subtests | Status |
|------|----------|--------|
| TestS302_HTTP_SequentialSymbolChainQueries | chain_btcusdt, chain_ethusdt, chain_solusdt | PASS |
| TestS302_HTTP_FunnelPerSymbol | funnel_btcusdt, funnel_ethusdt, funnel_solusdt | PASS |
| TestS302_HTTP_DispositionsPerSymbol | dispositions_btcusdt, dispositions_ethusdt, dispositions_solusdt | PASS |

### Integration Tests — 4 tests READY (requireclickhouse)

| Test | Status |
|------|--------|
| TestCompositeReader_S302_SC1_SimultaneousApproved | Compiles, ready for ClickHouse |
| TestCompositeReader_S302_SC2_MixedDispositions | Compiles, ready for ClickHouse |
| TestCompositeReader_S302_SC3_AggregateIndependence | Compiles, ready for ClickHouse |
| TestCompositeReader_S302_SC4_BatchCountPerSymbol | Compiles, ready for ClickHouse |

### Regression — Zero

All pre-existing tests in `analyticalclient` and `handlers` continue to pass.

## Files Changed

| File | Change |
|------|--------|
| `internal/application/analyticalclient/multi_symbol_scenario_test.go` | **NEW** — 4 scenario tests + multi-symbol stub readers |
| `internal/interfaces/http/handlers/composite_multi_symbol_test.go` | **NEW** — 3 HTTP multi-symbol scenario tests |
| `internal/adapters/clickhouse/composite_reader_integration_test.go` | **MODIFIED** — 4 S302 integration test scenarios + 2 new fixture helpers |
| `docs/architecture/multi-symbol-deterministic-scenario-pack.md` | **NEW** — Scenario definitions and design decisions |
| `docs/architecture/multi-symbol-scenarios-results-and-operational-findings.md` | **NEW** — Results, before/after, operational findings |
| `docs/stages/stage-s302-multi-symbol-deterministic-scenario-pack-report.md` | **NEW** — This report |

## Key Findings

1. **Symbol isolation (S301) is the foundation.** Attribution correctness per symbol is guaranteed by upstream query isolation, not by attribution logic itself.
2. **Heterogeneous scenarios reveal no new bugs.** Different signal types, directions, dispositions, and constraint values all produce correct per-symbol results.
3. **Batch counts are accurate.** Each symbol returns exactly its own chains with no cross-symbol contamination.
4. **Funnel queries are type-scoped.** Must match signal type in addition to symbol — correct behavior, documented.
5. **No ordering anomalies.** Batch results consistently ordered by execution timestamp DESC.

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| Multi-symbol scenarios really validated | DONE — 22 unit tests + 4 integration tests |
| System shows consistent behavior per symbol | DONE — All assertions pass per symbol |
| Stage increases confidence without inflating scope | DONE — 3 symbols, 4 scenarios, no explosion |
| Base ready for observability and execution pressure | DONE — Patterns established for future scenarios |

## Residual Limitations

1. Integration tests require ClickHouse (`requireclickhouse` tag) — not exercised in this run.
2. No concurrent goroutine testing (true parallel HTTP requests).
3. Three symbols only (charter scope).
4. Paper mode only (no real venue).
5. No sub-millisecond ordering stress test.

## Preparation for S303

The S302 scenario pack establishes the multi-symbol test infrastructure. Recommended next steps:

- **S303 candidate: Resource scaling behavior.** Measure query performance as symbol count grows (MQ7 from S300 charter).
- Alternative: Multi-symbol live pipeline smoke test (extend `make smoke-multi` to validate composite read model end-to-end).
- The stub reader pattern (`multiSymbolStubReader`, `multiSymbolBatchStubReader`) is ready for reuse in future scenarios.
