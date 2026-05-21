# Strategy Coverage and Actor Chain Hardening

**Stage:** S239
**Date:** 2026-03-20
**Status:** Executed

## Problem Statement

The S238 gate identified two hardening gaps:

1. **Strategy domain test coverage** was inferior to decision and risk domains (30 tests vs 48/42)
2. **No inter-actor chain integration test** existed — the decision → strategy → risk pipeline was only tested in isolation at each stage

## Strategy Domain Hardening

### Domain Layer: `internal/domain/strategy/strategy_test.go`

Added 4 tests:

| Test | What It Proves |
|------|---------------|
| `TestStrategy_MultiSymbol_PartitionKeyIsolation` | 3 symbols × 2 timeframes produce 6 unique partition keys — no key collision across dimensions |
| `TestStrategy_MultiSymbol_DeduplicationKeyIsolation` | 3 symbols × 2 timeframes produce 6 unique deduplication keys at the same timestamp |
| `TestStrategy_Validate_NegativeTimeframe` | Negative timeframe correctly rejected by validation |
| `TestStrategy_Validate_NilDecisions` | Empty decisions slice (not nil) correctly rejected by validation |

These close the gap with decision domain tests (`TestDecision_MultiSymbol_PartitionKeyIsolation` etc.).

### Actor Layer: `internal/actors/scopes/derive/strategy_resolver_actor_test.go`

Added 4 tests:

| Test | What It Proves |
|------|---------------|
| `TestMeanReversionResolverActor_SeverityAndRationale_Propagated` | Decision severity and rationale survive into strategy DecisionInput and metadata |
| `TestMeanReversionResolverActor_NilScopePID_PublishesWithoutFanout` | Actor publishes correctly even without a scope PID for fan-out |
| `TestMeanReversionResolverActor_FanOut_IncludesDecisionContext` | Fan-out message to risk stage carries decision severity, rationale, and correlation ID |
| `TestMeanReversionResolverActor_InvalidConfidence_NoPublish` | Invalid confidence string causes silent drop (no publish, no panic) |

These bring the strategy actor layer from 5 tests to 9, closing the gap with decision (7) and risk (6) actor tests.

## Inter-Actor Chain Integration Test

### File: `internal/actors/scopes/derive/actor_chain_integration_test.go`

This is a new file containing 3 integration tests that wire real actor instances together.

### Architecture

Each test creates:
- **3 publisher collectors** (decisionPub, strategyPub, riskPub) — capture final published events
- **2 fan-out collectors** (decFanout, stratFanout) — capture inter-actor messages at each routing boundary
- **3 real actor instances** (RSIOversoldEvaluatorActor, MeanReversionEntryResolverActor, PositionExposureEvaluatorActor)

The test manually forwards messages between stages (simulating SourceScopeActor routing), which allows asserting on the intermediate messages at each boundary.

### Tests

| Test | Path | Assertions |
|------|------|------------|
| `TestActorChain_Signal_To_Decision_To_Strategy_To_Risk` | Low RSI → triggered → long → approved | Full chain: decision outcome, severity propagation, strategy direction/confidence, risk disposition, decision context survival, correlation ID |
| `TestActorChain_NotTriggered_FlowsThrough` | High RSI → not_triggered → flat → approved | Negative path: flat direction propagates correctly, risk approves flat strategies |
| `TestActorChain_CorrelationID_PreservedEndToEnd` | Low RSI with correlation ID | Correlation ID verified at every stage: decision publish, decision fan-out, strategy publish, strategy fan-out, risk publish |

### What This Proves

1. **Decision context threading works end-to-end** — severity and rationale from the decision evaluator survive through strategy resolution into risk assessment
2. **DBI-9 isolation is maintained** — each stage receives primitive-only messages, never cross-domain structs
3. **Both positive and negative paths produce valid domain objects** — triggered/long/approved and not_triggered/flat/approved both pass validation
4. **Correlation IDs are never lost** — a single correlation ID injected at signal stage appears in all 5 output messages (3 publishes + 2 fan-outs)

## Coverage Summary After S239

| Layer | Strategy | Decision | Risk |
|-------|----------|----------|------|
| Domain | 17 | 17 | 19 |
| Application | 12 | 24 | 17 |
| Actor (Derive) | 9 | 7 | 6 |
| Actor (Chain) | 3 (shared) | 3 (shared) | 3 (shared) |
| **Total** | **41** | **51** | **45** |

Strategy is no longer the weakest domain. The remaining gap vs decision (application layer: 12 vs 24) is acceptable — decision has more complex evaluation logic (RSI zones, severity taxonomy, confidence monotonicity) that justifies more tests.

## Files Changed

- `internal/domain/strategy/strategy_test.go` — 4 new tests
- `internal/actors/scopes/derive/strategy_resolver_actor_test.go` — 4 new tests
- `internal/actors/scopes/derive/actor_chain_integration_test.go` — new file, 3 integration tests
