# Stage S268: Full Closed-Loop Scenario Validation — Report

## Status: COMPLETE

## Objective

Validate complete closed-loop scenarios from signal intelligence through decision, strategy, risk, and paper execution — proving the Market Foundry domain pipeline produces coherent, observable, operationally meaningful paper execution as a single integrated unit.

## Context

- S265 aligned the risk → execution boundary contract.
- S266 proved paper order generation through the actor chain.
- S267 proved fill/status/control behavior.
- S268 proves the **first real closed loop** of the Foundry — not isolated stages, but the complete operational narrative.

## Scenarios Validated

| ID | Scenario | Signal | Chain | Outcome |
|----|----------|--------|-------|---------|
| CL-A | Mean Reversion Full Observability | RSI 10 (high severity) | rsi_oversold → mean_reversion → dual risk → paper buy | PASS |
| CL-B | Trend Following Full Observability | EMA bullish (moderate) | ema_crossover → trend_following → dual risk → paper buy | PASS |
| CL-C | Severity Contrast at Every Stage | RSI 10 vs RSI 25 | Same chain, different severities | PASS |
| CL-D | No-Signal Suppression | RSI 75 (not triggered) | Full chain → no-action paper order | PASS |
| CL-E | Cross-Chain Behavioral Distinction | RSI 10 vs EMA bullish | Mean reversion vs trend following | PASS |

**All 5 scenarios pass. Zero production code changes required.**

## Key Evidence

### Severity actively shapes execution quantities
```
High severity (RSI 10):  qty=0.0192 (decision=high → strategy_conf=0.8333 → risk_pos=0.0192)
Low severity  (RSI 25):  qty=0.0075 (decision=low  → strategy_conf=0.4666 → risk_pos=0.0075)
Ratio: 2.56×
```

### Dual risk fan-out produces independent orders
```
Position exposure: qty=0.0192 (conservative, confidence factor=0.90)
Drawdown limit:    qty=0.0575 (wider tolerance)
```

### Cross-chain behavioral distinction
```
Mean reversion:   qty=0.0192 (strategy=mean_reversion_entry, params: target_offset, stop_offset)
Trend following:  qty=0.0135 (strategy=trend_following_entry, params: trailing_stop_pct, take_profit_pct)
```

### No-signal suppression closes correctly
```
RSI 75 → not_triggered → flat → approved (trivial) → side=none, qty=0, fills=0
```

### Causal trace preserved across all stages
- CorrelationID survives 4 stage boundaries in all scenarios
- CausationID set at each stage transition
- Domain validation passes for all intermediate and final outputs

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `internal/actors/scopes/derive/closed_loop_end_to_end_test.go` | **New** — 5 closed-loop scenarios with full intermediate observability | ~650 |
| `docs/architecture/full-closed-loop-scenario-validation.md` | **New** — scenario design, principles, and validation plan | — |
| `docs/architecture/closed-loop-scenarios-results-and-operational-findings.md` | **New** — results, operational findings, before/after assessment | — |

## What This Proves

1. **The Foundry has a working closed loop.** Signal intelligence transforms into paper execution through a coherent, observable pipeline.
2. **Every intermediate stage is auditable.** Not just the final output — decision, strategy, risk, and execution each produce typed domain events.
3. **Severity is behavioral, not decorative.** It actively shapes confidence, parameters, constraints, and quantities at every stage.
4. **Negative paths close correctly.** Non-triggered signals produce auditable no-action events, not silent drops.
5. **Strategy families produce distinct operational profiles.** Counter-trend and pro-trend chains differ at every stage.

## Limitations Documented

- No venue adapter (paper fills only)
- No SafetyGate integration (kill switch/staleness not proven end-to-end)
- No KV materialization or ClickHouse round-trip
- Single symbol (btcusdt@60s)
- No concurrent scenarios
- Static signal values (not computed from candle data)

## Before/After

| Aspect | Before S268 | After S268 |
|--------|-------------|------------|
| Closed loop proof | No | Yes — 5 scenarios validated |
| Intermediate observability | Unit tests only | Cross-stage behavioral proof |
| Operational exit | Domain enrichment "without exit" | Signal → paper execution proven |
| Gate readiness | Not ready | Ready for S269 gate evaluation |

## Acceptance Criteria Status

- [x] Closed-loop scenarios exist and are validated
- [x] Foundry proves first complete loop to paper execution
- [x] Wave has operational exit (not just domain enrichment)
- [x] Base is ready for S269 gate evaluation
- [x] Scenarios are small, useful, and auditable

## Guard Rails Respected

- [x] No venue real opened
- [x] No large scenario matrix (5 focused scenarios)
- [x] No production readiness inflation
- [x] No gaps masked with artificial scenarios
- [x] Limitations clearly documented

## Preparation for S269

The S269 gate evaluation should assess:
1. **Completeness**: Do the 5 closed-loop scenarios cover the operational surface? (Yes — both families, both risk types, action + no-action, severity contrast, cross-chain)
2. **Remaining gaps**: SafetyGate, KV materialization, ClickHouse round-trip, multi-symbol
3. **Production readiness delta**: What remains between current state and production paper trading?
4. **Next wave scope**: Whether to extend closed-loop depth or expand breadth to new families
