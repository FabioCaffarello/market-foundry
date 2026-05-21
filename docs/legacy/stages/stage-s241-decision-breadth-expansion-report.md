# Stage S241 — Decision Breadth Expansion Report

**Charter:** BREADTH-WAVE-1
**Stage:** S241
**Previous:** S240 (Breadth Charter and Scope Freeze)
**Date:** 2026-03-21

---

## 1. Executive Summary

S241 delivered the second decision evaluator/type (`ema_crossover`) to the market-foundry decision domain, achieving the breadth target defined in S240. The implementation reuses 100% of existing infrastructure (domain struct, NATS stream, ClickHouse table, HTTP handlers, writer pipeline) while adding only the type-specific routing (NATS subjects, KV buckets, durable consumers). All existing tests continue to pass. Two new integration tests validate the EMA crossover actor chain.

---

## 2. Breadth Applied

### Before S241
| Domain | Types | Families |
|--------|-------|----------|
| Decision | 1 (`rsi_oversold`) | 1 |

### After S241
| Domain | Types | Families |
|--------|-------|----------|
| Decision | 2 (`rsi_oversold`, `ema_crossover`) | 2 |

### Breadth Measurement

| Metric | Before | After | Target | Status |
|--------|--------|-------|--------|--------|
| Decision evaluator types | 1 | 2 | >= 2 | PASS |
| Distinct signal sources consumed | 1 (RSI) | 2 (RSI + EMA) | >= 2 | PASS |
| Decision family YAMLs | 1 | 2 | >= 2 | PASS |
| Actor chain integration tests | 3 | 5 | >= 2 | PASS |

---

## 3. Files Changed

### Added (4 files)
| File | Purpose |
|------|---------|
| `internal/application/decision/ema_crossover_evaluator.go` | Pure evaluation logic |
| `internal/application/decision/ema_crossover_evaluator_test.go` | 20+ unit test cases |
| `internal/actors/scopes/derive/ema_crossover_decision_evaluator_actor.go` | Actor wrapper |
| `codegen/families/ema_crossover.yaml` | Codegen family spec |

### Modified (7 files)
| File | Change |
|------|--------|
| `internal/adapters/nats/natsdecision/registry.go` | +2 event specs, +2 control specs, +2 consumer specs |
| `internal/adapters/nats/natsdecision/publisher.go` | +1 switch case in `specForType` |
| `internal/adapters/nats/natsdecision/kv_store.go` | +1 bucket constant |
| `internal/actors/scopes/derive/derive_supervisor.go` | +1 `DecisionFamilyProcessor` entry |
| `internal/actors/scopes/store/store_supervisor.go` | +1 decision projection pipeline |
| `cmd/writer/pipeline.go` | +1 writer pipeline entry |
| `internal/actors/scopes/derive/actor_chain_integration_test.go` | +2 integration tests |

### Documentation (3 files)
| File | Purpose |
|------|---------|
| `docs/architecture/decision-breadth-expansion.md` | Architecture changes and decisions |
| `docs/architecture/decision-type-02-semantics-rationale-and-boundaries.md` | Type 02 semantics |
| `docs/stages/stage-s241-decision-breadth-expansion-report.md` | This report |

---

## 4. Semantic and Operational Gains

### 4.1 Validated Abstractions
- **`Decision` struct** proven to host fundamentally different evaluation models (numeric threshold vs. categorical classification)
- **`SignalInput`** works for both continuous (RSI float) and categorical (EMA direction string) signal types
- **Actor fan-out** routing correctly delivers signals to all registered decision evaluators per symbol
- **Shared infrastructure** (stream, table, HTTP, projections) handles multi-type decisions without modification

### 4.2 New Analytical Chain Foundation
The EMA crossover decision establishes the first half of Chain B:
```
candle → ema_signal → ema_crossover → [trend_following_entry] → [drawdown_limit] → paper_order
```
S242 will complete the next segment by adding the `trend_following_entry` strategy resolver.

### 4.3 Operational Observability
All existing observability channels (NATS events, KV state, ClickHouse, HTTP, actor logs) automatically support the new type through parameterized routing.

---

## 5. Limits and Trade-offs

### 5.1 Fixed Severity/Confidence
The EMA crossover evaluator assigns baseline severity (`moderate`) and confidence (`0.75`) because the actor chain's `signalGeneratedMessage` contract does not carry signal metadata (EMA values, spread). This is a deliberate breadth-over-depth trade-off.

**Impact:** The decision correctly identifies crossover direction but cannot graduate severity by crossover magnitude.
**Mitigation path:** Future depth stage can widen `signalGeneratedMessage` to carry signal metadata, enabling proportional severity/confidence.

### 5.2 Bullish-Only Triggering
Only bullish crossovers trigger. Bearish crossovers produce `not_triggered`. This aligns with S242's `trend_following_entry` strategy (which enters long on bullish signals) but limits the type to one direction.

**Impact:** Short-entry strategies would need a separate `ema_crossover_bearish` type or a directional parameter.
**Mitigation path:** Future breadth/depth can add directional variants without modifying this evaluator.

### 5.3 No Cross-Evaluator Correlation
The two decision types (`rsi_oversold`, `ema_crossover`) evaluate independently. There is no composite decision that combines both signals.

**Impact:** Strategy resolvers see individual decisions, not correlated multi-signal assessments.
**Mitigation path:** A future composite evaluator could consume multiple decision outputs, but this is explicitly out of breadth scope.

---

## 6. Test Evidence

```
ok  internal/application/decision    — all RSI + EMA evaluator tests pass
ok  internal/actors/scopes/derive    — 5 actor chain integration tests pass
ok  internal/actors/scopes/store     — projection tests pass
ok  cmd/writer                       — writer pipeline tests pass
ok  internal/domain/decision         — domain validation tests pass
```

---

## 7. Preparation for S242

S242 target: Add `trend_following_entry` strategy resolver.

### What S241 Provides
- `decisionEvaluatedMessage` with `DecisionType: "ema_crossover"` already flows through the actor chain
- Strategy resolvers receive all decision fan-outs; a new resolver just needs to filter by `DecisionType`
- The `DecisionInput` struct in the strategy domain already carries `Type`, `Outcome`, `Confidence`, `Severity`, `Rationale`

### What S242 Needs to Do
1. Create `internal/application/strategy/trend_following_entry_resolver.go`
2. Create the actor wrapper
3. Register NATS specs for `trend_following_entry` in the strategy registry
4. Register the strategy processor in the derive supervisor
5. Add writer and store pipeline entries
6. Add integration tests proving the `ema_crossover → trend_following_entry` chain

---

## 8. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Decision has >= 2 evaluators/types | PASS (rsi_oversold + ema_crossover) |
| New breadth is explainable and observable | PASS (rationale, logs, KV, CH, HTTP) |
| End-to-end trail remains coherent | PASS (fan-out, publisher, projections) |
| Base ready for S242 strategy expansion | PASS (decision messages flow to strategy layer) |
| Breadth without scope explosion | PASS (4 new files, 7 modified, no new infra) |

---

## 9. Guard Rail Compliance

| Guard Rail | Compliance |
|------------|------------|
| No new analytical family opened | PASS — reuses `decisions` table and `DECISION_EVENTS` stream |
| No rule inflation | PASS — one evaluator, three branches (bullish/bearish/neutral) |
| Explainability preserved | PASS — rationale generated for every outcome |
| No infrastructure wave | PASS — all infrastructure reused, only routing config added |
| Limits documented | PASS — see Section 5 |
