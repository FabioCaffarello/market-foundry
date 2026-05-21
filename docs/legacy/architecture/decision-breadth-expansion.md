# Decision Breadth Expansion

**Stage:** S241
**Charter:** BREADTH-WAVE-1
**Domain:** Decision
**Date:** 2026-03-21

---

## 1. Purpose

Expand the decision domain from one evaluator/type (`rsi_oversold`) to two (`rsi_oversold` + `ema_crossover`), achieving the breadth target defined in S240's charter without introducing a generic rule engine or combinatorial explosion.

---

## 2. What Changed

### 2.1 New Decision Type: `ema_crossover`

A second decision evaluator that consumes EMA crossover signals and produces structured decisions with outcome, severity, confidence, rationale, and metadata — sharing the same `Decision` domain struct as `rsi_oversold`.

### 2.2 Files Added

| File | Purpose |
|------|---------|
| `internal/application/decision/ema_crossover_evaluator.go` | Pure application logic for EMA crossover evaluation |
| `internal/application/decision/ema_crossover_evaluator_test.go` | Comprehensive unit tests (20+ test cases) |
| `internal/actors/scopes/derive/ema_crossover_decision_evaluator_actor.go` | Actor wrapper for the evaluator |
| `codegen/families/ema_crossover.yaml` | Codegen family definition |

### 2.3 Files Modified

| File | Change |
|------|--------|
| `internal/adapters/nats/natsdecision/registry.go` | Added `EMACrossoverEvaluated`, `EMACrossoverLatest` specs, consumer specs |
| `internal/adapters/nats/natsdecision/publisher.go` | Extended `specForType` switch for `ema_crossover` |
| `internal/adapters/nats/natsdecision/kv_store.go` | Added `EMACrossoverLatestBucket` constant |
| `internal/actors/scopes/derive/derive_supervisor.go` | Registered `ema_crossover` in `DecisionFamilyProcessor` list |
| `internal/actors/scopes/store/store_supervisor.go` | Added EMA crossover projection pipeline |
| `cmd/writer/pipeline.go` | Added EMA crossover writer pipeline entry |
| `internal/actors/scopes/derive/actor_chain_integration_test.go` | Added 2 integration tests for EMA crossover actor |

---

## 3. Architecture Decisions

### 3.1 Categorical Input, Not Numeric

The EMA crossover signal produces categorical values (`"bullish"`, `"bearish"`, `"neutral"`) unlike RSI which produces a continuous numeric value. The evaluator handles this by switching on the categorical value rather than parsing a float. This validates that the `SignalInput` abstraction supports both numeric and categorical signal types.

### 3.2 Fixed Severity and Confidence

Because the `signalGeneratedMessage` contract (DBI-9) passes only primitive signal data without signal metadata (fast_ema, slow_ema, spread), the EMA crossover evaluator assigns baseline severity (`moderate` for triggered) and confidence (`0.75` for directional, `0.50` for neutral). This is a deliberate breadth-over-depth trade-off: carrying signal metadata through the actor chain would require widening the message contract, which is out of scope for this stage.

### 3.3 Shared Infrastructure

The new decision type reuses 100% of the existing infrastructure:
- Same `Decision` domain struct
- Same `DecisionEvaluatedEvent` event type
- Same `DECISION_EVENTS` NATS stream
- Same `DecisionProjectionActor` for KV materialization
- Same `DecisionReader` for ClickHouse analytical queries
- Same HTTP handler (`GET /decision/:type/latest`)
- Same writer pipeline mapper (`mapDecisionRow`)

The only new infrastructure is the per-type routing: separate NATS subjects, separate KV buckets, separate durable consumers.

### 3.4 Opt-In Activation

The EMA crossover decision family is activated via `pipeline.decision_families` in the pipeline config, following the same opt-in pattern as all other families. No changes to existing deployments unless explicitly enabled.

---

## 4. What Was NOT Done

- **No signal contract widening.** `signalGeneratedMessage` was not extended to carry signal metadata. This limits severity/confidence granularity but keeps the message contract stable.
- **No new analytical family.** The EMA crossover decision writes to the same `decisions` ClickHouse table using the same schema.
- **No generic decision engine.** Each evaluator remains a focused, single-purpose application service. No rule DSL, no pluggable evaluation framework.
- **No depth enrichment of existing types.** `rsi_oversold` was not modified.
- **No strategy/risk changes.** The EMA crossover decision fans out to strategy resolvers via the existing `decisionEvaluatedMessage`, but no new strategy resolver consumes it yet (that's S242).

---

## 5. Breadth Verification

| Metric | Before S241 | After S241 | Target |
|--------|------------|------------|--------|
| Decision evaluator types | 1 | 2 | >= 2 |
| Signal sources consumed | 1 (RSI) | 2 (RSI + EMA) | >= 2 |
| Decision family YAMLs | 1 | 2 | >= 2 |
| Integration test paths | 3 | 5 | >= 2 |

---

## 6. Preparation for S242

The EMA crossover decision produces `decisionEvaluatedMessage` with:
- `DecisionType: "ema_crossover"`
- `DecisionOutcome: "triggered" / "not_triggered"`
- `DecisionConfidence: "0.7500" / "0.5000"`
- `DecisionSeverity: "moderate" / "none"`
- `DecisionRationale: "EMA crossover bullish: ..."`

S242 will add a `trend_following_entry` strategy resolver that consumes these messages, creating the second complete analytical chain: `candle → ema_signal → ema_crossover → trend_following_entry → ...`.
