# Multi-Symbol Deterministic Scenario Pack

> Stage S302 — Phase 29: Multi-Symbol Operational Scaling Wave

## Purpose

Define a small, robust pack of deterministic multi-symbol scenarios that prove the composite read model behaves correctly and consistently when multiple symbols are active simultaneously.

This is NOT combinatorial explosion. This is targeted, representative coverage.

## Symbols in Scope

| Symbol   | Role in Scenarios                              |
|----------|-----------------------------------------------|
| btcusdt  | Primary — always approved, full chain          |
| ethusdt  | Secondary — varies (approved, rejected)        |
| solusdt  | Tertiary — varies (approved, modified, short)  |

Source: `binancef` (futures). Timeframe: `60` (1 minute).

## Scenario Inventory

### SC1: Simultaneous Approved Chains — Different Characteristics

**What it proves:** Three symbols produce independent full chains with different signal types, directions, and constraint profiles.

| Symbol   | Signal     | Decision      | Strategy        | Risk     | Execution |
|----------|-----------|---------------|-----------------|----------|-----------|
| btcusdt  | rsi=28.5  | rsi_oversold/high | long/mean_rev   | approved | buy 0.1   |
| ethusdt  | macd=0.0025 | macd_cross/moderate | long/trend_fol  | approved | buy 1.5   |
| solusdt  | boll=1.85 | squeeze/low   | short/squeeze   | approved | sell 10.0 |

**Assertions:**
- All 3 chains are `chain_complete=true`, `stage_count=5`
- Every stage within a chain belongs to the correct symbol
- Attribution per symbol matches its specific constraints and rationale
- Signal types, strategy directions, and execution sides are symbol-specific

### SC2: Mixed Dispositions Across Symbols

**What it proves:** Different risk outcomes coexist across symbols without contamination.

| Symbol   | Risk Disposition | Execution | Stage Count |
|----------|-----------------|-----------|-------------|
| btcusdt  | approved         | buy 0.1   | 5 (complete) |
| ethusdt  | rejected         | none      | 4 (missing execution) |
| solusdt  | modified         | sell 5.0  | 5 (complete) |

**Assertions:**
- ethusdt chain has `chain_complete=false`, `missing_stages=["execution"]`
- ethusdt attribution shows `disposition=rejected`, `rationale="drawdown limit exceeded"`
- solusdt attribution shows `disposition=modified`, `rationale="position size reduced"`
- btcusdt is unaffected by other symbols' dispositions

### SC3: Concurrent Batch Queries — Correct Counts Per Symbol

**What it proves:** Batch queries return correct chain counts per symbol when data from multiple symbols exists.

| Symbol   | Chain Count | Expected Batch Result |
|----------|-------------|----------------------|
| btcusdt  | 3           | 3 chains             |
| ethusdt  | 2           | 2 chains             |
| solusdt  | 1           | 1 chain              |

**Assertions:**
- Each batch query returns exactly the expected count
- `meta.chain_count` matches actual chain count
- All chains in each batch belong to the queried symbol only
- No cross-symbol chains leak into any batch result

### SC4: Attribution Diversity Per Symbol

**What it proves:** The attribution projection (S298) produces correct, symbol-specific values across all three disposition types.

| Symbol   | Disposition | Severity | Direction | Max Position |
|----------|-------------|----------|-----------|-------------|
| btcusdt  | approved    | high     | long      | 0.10        |
| ethusdt  | rejected    | moderate | short     | 0.05        |
| solusdt  | modified    | low      | long      | 0.03        |

**Assertions:**
- `attribution.disposition` matches per symbol
- `attribution.rationale` is symbol-specific
- `attribution.strategy_context[0].decision_severity` matches
- `attribution.active_constraints.max_position_size` matches

## Integration Test Scenarios (requireclickhouse)

Four additional integration tests exercise the same scenarios against a live ClickHouse instance:

| Test ID          | Description                                    |
|------------------|-----------------------------------------------|
| S302-SC1-INT     | Three approved chains with causal chain integrity |
| S302-SC2-INT     | Mixed dispositions (approved/rejected/modified) |
| S302-SC3-INT     | Funnel and disposition aggregate independence  |
| S302-SC4-INT     | Batch count per symbol with ordering proof     |

## HTTP Handler Test Scenarios

Three handler-level tests validate the HTTP surface:

| Test ID          | Description                                    |
|------------------|-----------------------------------------------|
| S302-HTTP-1      | Sequential symbol chain queries (3 dispositions) |
| S302-HTTP-2      | Funnel results per symbol (different counts)   |
| S302-HTTP-3      | Disposition breakdown per symbol               |

## Design Decisions

1. **3 symbols, not 10.** Three symbols cover the key isolation boundaries without combinatorial noise.
2. **Deterministic fixtures, not random data.** Every value is hardcoded and auditable.
3. **One file per test layer.** Unit tests and integration tests are separate, following existing patterns.
4. **Reuse existing helpers.** `insertCompositeFixtureForSymbol` from S301 is reused; new helpers added only for rejected and modified fixtures.
5. **No schema changes.** All scenarios work with the existing 5-table schema.

## Limitations

- Integration tests require ClickHouse (`requireclickhouse` build tag).
- Scenarios use paper execution mode only (no real venue).
- Time-based ordering is tested but not sub-millisecond precision.
- Fan-out scenarios (1 signal -> N decisions across symbols) are out of scope.
