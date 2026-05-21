# S46 — Decision Multi-Symbol Verification Report

**Status:** Complete
**Date:** 2026-03-17
**Objective:** Verify that the `decision` domain behaves correctly under a controlled multi-symbol scenario (btcusdt + ethusdt), validating activation, ownership, projections, query surface, and absence of cross-symbol bleed.

---

## 1. Executive Summary

The `decision` domain (currently RSI Oversold only) was verified for multi-symbol correctness across all layers: domain model, derive pipeline, store projection, query surface, and E2E smoke tests. **No structural issues were found.** The architecture enforces per-symbol isolation at every boundary — this stage added the missing **explicit verification** to prove it.

Key findings:
- `PartitionKey` (`{source}.{symbol}.{timeframe}`) and `DeduplicationKey` both produce unique keys per symbol — no collision possible.
- KV store keys are deterministic and symbol-scoped — no cross-symbol bleed.
- Derive spawns independent `RSIOversoldEvaluatorActor` per symbol/timeframe — decision evaluation is fully isolated.
- NATS event subjects include `{source}.{symbol}.{timeframe}` tokens — stream routing is symbol-scoped.
- The smoke test now validates decision endpoints per symbol and checks cross-symbol isolation.
- Config-driven activation (`decision_families: ["rsi_oversold"]`) works identically across multiple symbols.

---

## 2. Multi-Symbol Scenario Validated

**Symbols:** btcusdt, ethusdt
**Timeframes:** 60s, 300s
**Decision family:** RSI Oversold (DF-01)
**Source:** binancef

### Isolation Layers Verified

| Layer | Isolation Mechanism | Status |
|-------|---------------------|--------|
| Domain | `PartitionKey()` = `{source}.{symbol}.{timeframe}` | Proven (new tests) |
| Domain | `DeduplicationKey()` = `dec:{type}:{source}:{symbol}:{timeframe}:{ts}` | Proven (new tests) |
| Derive | Per-symbol evaluator actors (`decisionEvaluators map[string][]*PID`) | Verified by design review |
| Derive | `routeSignalToDecision()` fans out only to evaluators for matching symbol | Verified by design review |
| NATS subjects | `decision.events.rsi_oversold.evaluated.{source}.{symbol}.{timeframe}` | Proven (new tests) |
| Store consumer | Durable `store-decision-rsi-oversold` with filter `decision.events.rsi_oversold.evaluated.>` | Verified by registry tests |
| Store KV | Key = `{source}.{symbol}.{timeframe}` per bucket entry | Proven (existing + new tests) |
| Store projection | Monotonicity guard compares by key — symbol-scoped | Verified by design review |
| Query (gateway) | `Get(source, symbol, timeframe)` constructs key deterministically | Verified by design review |
| HTTP | `?source=...&symbol=...&timeframe=...` required params | Verified (smoke test + HTTP tests) |

### Actor Topology Under Multi-Symbol

Per source (e.g., binancef), with 2 symbols and 2 timeframes:
- 1 `DecisionPublisherActor` (shared per source)
- 4 `RSIOversoldEvaluatorActor` instances (2 symbols x 2 timeframes)
- Each evaluator receives only signals for its specific symbol (via `routeSignalToDecision`)
- Evaluator is stateless (pure function) — no warm-up state that could leak

### Activation/Config Coherence

| Config key | derive.jsonc | store.jsonc | Consistent? |
|-----------|-------------|------------|-------------|
| `decision_families` | `["rsi_oversold"]` | `["rsi_oversold"]` | Yes |
| `signal_families` | `["rsi"]` | `["rsi"]` | Yes (operational dependency) |
| `timeframes` | `[60, 300]` | — | Yes (store is timeframe-agnostic) |

Activation is independent of signal families — config declares decision families explicitly. Binding watcher dynamically discovers symbols; adding a symbol via configctl automatically spawns decision evaluators.

---

## 3. Files Changed

### Tests Added/Extended

| File | Change | Purpose |
|------|--------|---------|
| `internal/domain/decision/decision_test.go` | +2 tests | `DeduplicationKey_MultiSymbolIsolation` (3 symbols × 2 timeframes), `PartitionKey_MultiSymbolMultiTimeframe` (3 symbols × 2 timeframes) |
| `internal/adapters/nats/decision_registry_test.go` | +1 test | `SubjectRoutingMultiSymbol` — verifies unique NATS subjects per source/symbol/timeframe and consumer filter matching |

### Smoke Test Extended

| File | Change | Purpose |
|------|--------|---------|
| `scripts/smoke-multi-symbol.sh` | Steps 7-8 added, Step 9 extended | Decision RSI Oversold endpoint validation per symbol, cross-symbol decision isolation check, decision error handling |

### HTTP Test Extended

| File | Change | Purpose |
|------|--------|---------|
| `tests/http/decision.http` | +4 requests | ethusdt 60s/300s queries, cross-symbol back-to-back comparison section |

---

## 4. Problems Found or Discarded

### No structural issues found

- **Cross-symbol bleed:** Not possible. PartitionKey includes symbol — KV keys are deterministic and distinct.
- **Ownership confusion:** Not possible. Each evaluator actor is named `rsi-oversold-{symbol}-{timeframe}s` and receives signals only for its symbol via `routeSignalToDecision()`.
- **State mixing:** Not possible. RSIOversoldEvaluator is a pure stateless function — no shared mutable state between symbols.
- **Config inconsistency:** Not present. `decision_families: ["rsi_oversold"]` is identically configured in both `derive.jsonc` and `store.jsonc`.
- **Activation gap:** Not present. Binding watcher discovers symbols dynamically — adding a symbol via configctl automatically spawns decision evaluators.
- **Signal dependency bleed:** Not possible. Decision evaluator receives signal data as primitives (`signalGeneratedMessage`), not as Signal structs — DBI-9 is maintained.

### Observation: Decision latency tied to RSI warm-up

Decision evaluates immediately after each RSI signal. Since RSI(14) requires 15 finalized candles (~15 minutes at 60s), the first decision also takes ~15 minutes. This is expected behavior inherited from the signal pipeline, not a defect. The smoke test accounts for this by accepting null decisions as valid (warm-up pending).

---

## 5. Impact on Readiness for S47/S49

### Decision multi-symbol readiness: PROVEN

The `decision` domain is now explicitly verified for multi-symbol operation. This removes the prerequisite for advancing toward `strategy`.

### Readiness checklist for strategy domain

| Prerequisite | Status |
|-------------|--------|
| Evidence multi-symbol proven (S17) | Done |
| Signal multi-symbol proven (S41) | Done |
| Decision domain exists and is hardened (S43-S45) | Done |
| Decision multi-symbol proven (S46) | **Done** |
| Decision activation/config coherent | **Done** |
| Decision projections isolated per symbol | **Done** |
| Decision query surface per symbol | **Done** |
| No cross-symbol bleed in decision | **Done** |

### What S46 does NOT cover (out of scope)

- Decision history (only latest-only projections verified)
- New decision families beyond RSI Oversold
- Strategy domain design or implementation
- Performance under high symbol count (>2)
- Multi-signal confluence decisions

---

## Appendix: Test Execution

```
=== RUN   TestDecision_MultiSymbolIsolation
--- PASS: TestDecision_MultiSymbolIsolation (0.00s)
=== RUN   TestDecision_TimeframeIsolation
--- PASS: TestDecision_TimeframeIsolation (0.00s)
=== RUN   TestDecision_DeduplicationKey_MultiSymbolIsolation
--- PASS: TestDecision_DeduplicationKey_MultiSymbolIsolation (0.00s)
=== RUN   TestDecision_PartitionKey_MultiSymbolMultiTimeframe
--- PASS: TestDecision_PartitionKey_MultiSymbolMultiTimeframe (0.00s)
=== RUN   TestDecisionKVStore_MultiSymbol_KeyIsolation
--- PASS: TestDecisionKVStore_MultiSymbol_KeyIsolation (0.00s)
=== RUN   TestDecisionKVStore_MultiSource_KeyIsolation
--- PASS: TestDecisionKVStore_MultiSource_KeyIsolation (0.00s)
=== RUN   TestDecisionRegistry_SubjectRoutingMultiSymbol
--- PASS: TestDecisionRegistry_SubjectRoutingMultiSymbol (0.00s)
```
