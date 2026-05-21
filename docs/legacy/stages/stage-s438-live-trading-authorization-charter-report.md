# Stage S438: Live Trading Authorization Wave -- Charter and Scope Freeze

> Date: 2026-03-24 | Phase: 50 (Live Trading Authorization) | Opens wave: S438-S443

## Objective

Open the Live Trading Authorization Wave by defining scope, governing questions, capabilities, non-goals, rollback criteria, stop conditions, and stage ordering. Freeze scope to prevent inflation. This stage produces no code changes -- it is a ceremony of planning and alignment.

## Context

The Mainnet Enablement Wave (S432-S437) closed with verdict **AUTHORIZED -- CONDITIONAL**. All 17 chartered capabilities were delivered at FULL rating. All three mainnet blockers (B-1, B-2, B-3) were resolved. Zero regressions were introduced. Zero real orders were placed.

The conditional authorization identified six conditions that must be satisfied before `dry_run=false` can be enabled:

| Condition | Gap | Description |
|-----------|-----|-------------|
| C-1 | RG-24 | No authenticated mainnet API call proven |
| C-2 | RG-20 | Mainnet credentials via env vars (no external secret manager) |
| C-3 | RG-22/RG-23 | ClickHouse backup manual and same-host only |
| C-4 | RG-25 | No sustained mainnet soak test |
| C-5 | -- | Kill-switch operational procedure not documented/tested |
| C-6 | -- | `dry_run=false` config rejection still active in source |

This stage transforms these conditions into a formal wave with frozen scope, explicit ordering, and ceremony-grade governance.

## Executive Summary

The Live Trading Authorization Wave is now formally open. The wave comprises 6 stages (S438-S443) organized into 5 execution blocks plus this charter. The wave's sole purpose is to prove that all prerequisites for live trading are satisfied and render a formal authorization verdict. It does NOT enable live trading.

Key decisions:

1. **Spot-first authorization.** Futures authorization requires a separate ceremony after Spot is proven.
2. **Single-symbol, minimum-size scope.** Authorization covers BTCUSDT (or operator-chosen single symbol) at minimum exchange-allowed quantity.
3. **Trade-only credential scope.** No withdrawal permissions on authorized API keys.
4. **Kill switch must be tested under operational conditions.** Runbook alone is insufficient.
5. **Authorization is necessary but not sufficient for go-live.** The operator must independently decide to enable live trading after reviewing the evidence gate.

## Deliverables

### Documentation

| Document | Description |
|----------|-------------|
| [`live-trading-authorization-wave-charter-and-scope-freeze.md`](../architecture/live-trading-authorization-wave-charter-and-scope-freeze.md) | Wave charter with scope freeze, dependency chain, success criteria, rollback criteria, and ceremony rules |
| [`live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md`](../architecture/live-trading-authorization-capabilities-questions-non-goals-and-rollback-criteria.md) | 22 governing questions, 20 chartered capabilities, 18 non-goals, 7 rollback criteria, 9 stop conditions |

### Wave Structure

| Stage | Block | Description | Resolves |
|-------|-------|-------------|----------|
| S438 | Charter | This stage -- scope freeze and wave opening | -- |
| S439 | Secret Manager | External secret manager deployment | C-2 (RG-20) |
| S440 | Backup | Automated backup with off-host replication | C-3 (RG-22, RG-23) |
| S441 | API + Soak | Authenticated mainnet API proof and sustained soak | C-1 (RG-24), C-4 (RG-25) |
| S442 | Kill-Switch | Operational runbook documentation and live test | C-5 |
| S443 | Evidence Gate | Live trading authorization verdict | C-6, final evaluation |

### Scope Summary

| Dimension | Value |
|-----------|-------|
| Governing questions | 22 |
| Chartered capabilities | 20 (across S439-S443) |
| Non-goals | 18 |
| Rollback criteria | 7 |
| Post-authorization stop conditions | 9 |
| Minimum authorized scope | 1 exchange, 1 segment (Spot), 1 symbol, minimum size, market order only |

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Wave formally opened with frozen scope | PASS | Charter document committed with 5 blocks, dependency chain, and ceremony rules |
| Objective is authorization, not go-live | PASS | Explicit in charter: "What This Wave Is NOT" section; NG-1 |
| Rollback criteria explicit | PASS | 7 wave-level rollback criteria in charter and capabilities document |
| Stop conditions explicit | PASS | 9 post-authorization stop conditions documented |
| Next stages ordered with rigor | PASS | S439-S443 ordered with dependency rationale; S439 before S441 (credential dependency) |
| Non-goals prevent scope inflation | PASS | 18 non-goals covering multi-exchange, OMS expansion, Futures authorization, etc. |
| Governing questions answerable by evidence | PASS | 22 questions with expected answers and assigned stages |
| Minimum authorized scope defined | PASS | 1 exchange, 1 segment, 1 symbol, minimum size, trade-only credentials |

## Guard Rails Verification

| Guard Rail | Status |
|------------|--------|
| No live trading authorized in this stage | PASS -- charter and NG-1 explicitly prohibit |
| No multi-exchange opened | PASS -- NG-4, Binance-only |
| No OMS expansion | PASS -- NG-5, market-order-only |
| No runtime or governance redesign | PASS -- NG-15, architecture frozen |

## Risk Assessment

This is the most consequential wave in the project's history. The risk is bounded by:

1. **No code changes in charter stage.** S438 is documentation only.
2. **dry_run=true throughout the wave.** No real orders until after S443 passes AND the operator independently enables live trading.
3. **Minimum scope limits blast radius.** Even after authorization, exposure is limited to 1 symbol at minimum size.
4. **Kill-switch provides immediate halt.** Reversibility is total at any point.
5. **14 consecutive wave passes.** Execution discipline is empirically proven.

## Next Stage

**S439: External Secret Manager Deployment.** Resolve C-2 by deploying an external secret manager and wiring mainnet adapters to retrieve credentials from it.
