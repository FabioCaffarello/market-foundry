# Execution Integrated Operational Validation Matrix (S84)

**Status**: Active
**Date**: 2026-03-18
**Scope**: Post-execute integrated validation — derive → execute → store → gateway

## Context

This matrix supersedes the S79 operational validation matrix by extending coverage to the full integrated mesh including the `execute` binary. S79 validated domain model + derive-side pipeline. S84 validates the complete chain: derive publishes execution intent → execute consumes, applies kill switch + staleness guard, submits to venue adapter, publishes fill → store projects fill → gateway exposes composite status.

## Unit Test Coverage

### Domain Model (30 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Validation: required fields, side values, status values | Structural | PASS |
| Lifecycle transitions: 7 valid, invalid blocked | State machine | PASS |
| Terminal states: filled, rejected, cancelled | State machine | PASS |
| Fill records: simulated, quantity, fee | Domain | PASS |
| Partition key: source.symbol.timeframe | Key isolation | PASS |
| Dedup key: exec:type:source:symbol:timeframe:ts | Idempotency | PASS |
| Multi-symbol isolation: 3 symbols × 2 timeframes | Isolation | PASS |

### Control Gate (13 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Validation: active/halted status | Structural | PASS |
| IsHalted: active=false, halted=true | Behavior | PASS |
| DefaultControlGate: active, fail-open | Default | PASS |
| Halt/resume cycles | Lifecycle | PASS |
| Invalid status rejected | Validation | PASS |

### Paper Order Evaluator (8 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Approved + long → SideBuy | Decision | PASS |
| Approved + short → SideSell | Decision | PASS |
| Rejected → SideNone, qty=0 | Decision | PASS |
| Modified → preserves direction | Decision | PASS |
| Flat direction → SideNone | Decision | PASS |

### Paper Fill Simulator (7 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Buy order → filled, 1 fill record | Fill | PASS |
| Sell order → filled, 1 fill record | Fill | PASS |
| No-action → stays submitted, 0 fills | Fill | PASS |
| Fill marked simulated | Fill | PASS |

### Staleness Guard (5 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Fresh (30s old) → not stale | Guard | PASS |
| Stale (3min old, 2min max) → stale | Guard | PASS |
| Exact boundary → not stale | Guard | PASS |
| Future timestamp → not stale | Guard | PASS |
| Integration with venue adapter | S84 | PASS |

### Execution Projection — Store (21 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Final gate: skips non-final | Gate | PASS |
| Validation gate: rejects malformed | Gate | PASS |
| Put written → materialized | Projection | PASS |
| Put skipped stale → skippedStale | Monotonicity | PASS |
| Put skipped duplicate → skippedDedup | Idempotency | PASS |
| Put error → errors + tracker | Error | PASS |
| Stats invariant: received = sum(outcomes) | Invariant | PASS |
| Multi-symbol: independent materialization | Isolation | PASS |

### Fill Projection — Store (14 tests — ALL PASS)

| Test | Category | Status |
|------|----------|--------|
| Final gate: skips non-final | Gate | PASS |
| Validation gate: rejects malformed | Gate | PASS |
| Put written → materialized | Projection | PASS |
| Put skipped stale → skippedStale | Monotonicity | PASS |
| Put skipped duplicate → skippedDedup | Idempotency | PASS |
| Put error → errors + tracker | Error | PASS |
| Stats invariant: received = sum(outcomes) | Invariant | PASS |
| Multi-symbol: independent materialization | Isolation | PASS |
| Venue order ID does not affect gating | Passthrough | PASS |

## Pipeline Integration Tests (8 tests — ALL PASS)

### Pre-S84 (3 tests)

| Test | Chain | Status |
|------|-------|--------|
| EvaluateSimulateEmit_BuyOrder | risk → eval → sim → event | PASS |
| EvaluateSimulateEmit_RejectedRisk_NoFill | risk → eval → sim (no-action) | PASS |
| MultiSymbol_FullIsolation | 3 symbols × 2 timeframes isolation | PASS |

### S84 New (5 tests)

| Test | Chain | Status |
|------|-------|--------|
| VenueAdapter_FullChain_DeriveToFill | eval → sim → venue → fill event + trace | PASS |
| VenueAdapter_NoAction_NoFillRecord | eval → sim → venue (no-action) | PASS |
| StalenessGuard_Integration | guard + venue pipeline | PASS |
| StatusPropagation_IntentAndResult | DeriveEffectivePropagation (4 combos) | PASS |
| MultiSymbol_FillIsolation | 3 sym × 2 tf through venue + trace | PASS |

## E2E Smoke Test Steps (22 steps)

### Pre-S84 (Steps 1-15)

| Step | Validation | Category |
|------|-----------|----------|
| 1 | Gateway healthz/readyz | Health |
| 2 | Wait for first candle | Pipeline warmup |
| 3-4 | Evidence candle multi-symbol + isolation | Evidence |
| 5-6 | Signal RSI multi-symbol + isolation | Signal |
| 7-8 | Decision RSI Oversold multi-symbol + isolation | Decision |
| 9-10 | Strategy Mean Reversion Entry multi-symbol + isolation | Strategy |
| 11-12 | Risk Position Exposure multi-symbol + isolation | Risk |
| 13-14 | Execution Paper Order multi-symbol + isolation | Execution |
| 15 | Execution control gate GET/PUT cycle | Control |

### S84 New (Steps 16-22)

| Step | Validation | Category |
|------|-----------|----------|
| 16 | Execute binary healthz/readyz | Execute health |
| 17 | Venue market order fill multi-symbol validation | Fill materialization |
| 18 | Cross-symbol fill isolation | Fill isolation |
| 19 | Execution status propagation (composite: intent + result + gate + propagation) | Status propagation |
| 20 | Kill switch integration with execute active (halt → verify via status → resume → verify) | Kill switch integration |
| 21 | Trace persistence through execute chain (correlation_id + causation_id in fills) | Trace persistence |
| 22 | Error handling (missing params, unknown types) | Error surface |

## Coverage by Validation Dimension

| Dimension | Unit | Integration | Smoke | Status |
|-----------|------|-------------|-------|--------|
| Domain model (validation, lifecycle, keys) | 30 | 3 | 4 | COMPLETE |
| Control gate (halt/resume, fail-open) | 13 | — | 5 | COMPLETE |
| Paper order evaluator | 8 | 5 | 4 | COMPLETE |
| Fill simulator | 7 | 2 | — | COMPLETE |
| Staleness guard | 5 | 1 | — | COMPLETE |
| Venue adapter (submit, fill, trace) | — | 3 | 4 | COMPLETE |
| Execution projection (store) | 21 | — | 4 | COMPLETE |
| Fill projection (store) | 14 | — | 4 | COMPLETE |
| Status propagation (composite) | — | 4 | 4 | COMPLETE |
| Kill switch integration | — | — | 4 | COMPLETE |
| Trace persistence (derive → execute) | — | 1 | 2 | COMPLETE |
| Multi-symbol isolation | 6 | 2 | 6 | COMPLETE |
| Query surface (HTTP endpoints) | — | — | 22 | COMPLETE |
| Error handling (param validation) | — | — | 12 | COMPLETE |

## Totals

| Metric | Count |
|--------|-------|
| Unit tests | 105+ |
| Integration tests | 8 |
| E2E smoke steps | 22 |
| Validation dimensions | 14 |
| Dimensions fully covered | 14/14 |
