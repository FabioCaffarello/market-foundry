# Stage S266: Controlled Paper Order Generation — Report

**Stage:** S266
**Title:** Controlled Paper Order Generation
**Verdict:** PASS
**Date:** 2026-03-21

## Objective

Activate the generation of controlled paper orders by connecting the domain intelligence pipeline (decision → strategy → risk) to the paper execution path, producing observable and auditable execution intents.

## What Was Done

### Discovery

The actor chain from risk to execution was already fully wired:
- Risk evaluator actors (`PositionExposureEvaluatorActor`, `DrawdownLimitEvaluatorActor`) send `riskAssessedMessage` to `ScopePID` when set
- `PaperOrderEvaluatorActor` handles `riskAssessedMessage`, evaluates, simulates fills, and publishes
- No production code changes were needed — the pipeline already generates paper orders when properly wired

The S266 gap was **end-to-end proof**: no test exercised the full signal → decision → strategy → risk → paper order path.

### Implementation

**New test file:** `internal/actors/scopes/derive/paper_order_end_to_end_test.go`

7 end-to-end behavioral scenarios:

| Scenario | Description | Validates |
|----------|-------------|-----------|
| S266-1 | RSI oversold → mean reversion → position exposure → paper buy | Full chain produces filled paper buy order |
| S266-2 | EMA crossover → trend following → position exposure → paper buy | Pro-trend chain produces paper buy with moderate severity |
| S266-3 | RSI 75.0 → not triggered → flat → no-action | Non-triggered signals produce SideNone, no fills |
| S266-4 | Dual risk fan-out → two independent paper orders | Both position_exposure and drawdown_limit produce orders |
| S266-5 | High vs low severity → different quantities | Severity contrast produces observably different order sizes |
| S266-6 | Mean reversion vs trend following cross-chain | Different strategy families produce distinct risk profiles |
| S266-7 | Partition and dedup key validation | KV materialization keys are correct |

### Architecture Documents

- `docs/architecture/controlled-paper-order-generation.md` — generation path, observability, severity influence
- `docs/architecture/paper-order-generation-guardrails-and-boundaries.md` — guard rails, boundaries, failure modes

## Test Results

```
=== RUN   TestPaperOrder_FullChain_RSIOversold_Buy
--- PASS: TestPaperOrder_FullChain_RSIOversold_Buy (0.05s)
=== RUN   TestPaperOrder_FullChain_EMACrossover_Buy
--- PASS: TestPaperOrder_FullChain_EMACrossover_Buy (0.05s)
=== RUN   TestPaperOrder_NotTriggered_FlatStrategy_NoAction
--- PASS: TestPaperOrder_NotTriggered_FlatStrategy_NoAction (0.05s)
=== RUN   TestPaperOrder_DualRiskFanout_TwoIndependentOrders
--- PASS: TestPaperOrder_DualRiskFanout_TwoIndependentOrders (0.05s)
=== RUN   TestPaperOrder_SeverityContrast_HighVsLow_DifferentQuantities
--- PASS: TestPaperOrder_SeverityContrast_HighVsLow_DifferentQuantities (0.10s)
=== RUN   TestPaperOrder_CrossChain_MeanReversionVsTrendFollowing
--- PASS: TestPaperOrder_CrossChain_MeanReversionVsTrendFollowing (0.10s)
=== RUN   TestPaperOrder_PartitionAndDedupKeys
--- PASS: TestPaperOrder_PartitionAndDedupKeys (0.05s)
```

All existing tests continue to pass. Build clean across all 8 binaries.

## Observed Quantities

From test execution:

| Scenario | Severity | Strategy | Risk Type | Quantity |
|----------|----------|----------|-----------|----------|
| S266-1 | high | mean_reversion_entry | position_exposure | 0.0192 |
| S266-2 | moderate | trend_following_entry | position_exposure | 0.0135 |
| S266-4 (exposure) | high | mean_reversion_entry | position_exposure | 0.0192 |
| S266-4 (drawdown) | high | mean_reversion_entry | drawdown_limit | 0.0575 |
| S266-5 high | high | mean_reversion_entry | position_exposure | 0.0192 |
| S266-5 low | low | mean_reversion_entry | position_exposure | 0.0075 |

Severity contrast is clear: high severity (0.0192) produces 2.56× the quantity of low severity (0.0075).

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/paper_order_end_to_end_test.go` | **NEW** — 7 end-to-end paper order scenarios |
| `docs/architecture/controlled-paper-order-generation.md` | **NEW** — generation path documentation |
| `docs/architecture/paper-order-generation-guardrails-and-boundaries.md` | **NEW** — guard rails and boundaries |
| `docs/stages/stage-s266-controlled-paper-order-generation-report.md` | **NEW** — this report |

**Production code changes: 0** — the pipeline was already wired; S266 proved it end-to-end.

## Guard Rails Verified

- Paper mode only: all intents are `type: "paper_order"`, all fills are `Simulated: true`
- Risk-gated: rejected dispositions produce `SideNone`
- Flat strategies produce `SideNone`
- Causal trace preserved: CorrelationID flows from signal through to paper order
- Per-symbol isolation: partition keys are `source.symbol.timeframe`
- Domain validation: all intents pass `Validate()` before publishing
- No real venue interaction: no external calls in any test or production path

## Gains

1. **End-to-end proof**: Domain intelligence now provably generates paper orders
2. **Severity influence visible**: High/moderate/low severity produces different order sizes
3. **Dual risk coverage**: Both risk evaluators independently produce execution intents
4. **Cross-chain distinction**: Mean reversion and trend following produce semantically distinct orders
5. **Full traceability**: Every paper order traces back to its originating signal

## Debts

1. **SafetyGate integration**: Kill switch and staleness guard are not yet enforced in the actor chain (deferred to S267)
2. **Round-trip fills**: Paper orders are filled inline by `PaperFillSimulator`; venue-side round-trip with status propagation is deferred to S267
3. **KV materialization**: Paper orders are published as events but not yet proven to materialize in KV store (deferred to S267)

## Trade-Offs

| Decision | Rationale |
|----------|-----------|
| No production code changes | Pipeline was already wired; proof-by-test is sufficient |
| Dual execution evaluators per risk type | Preserves per-risk-type observability; aggregation is out of scope |
| SafetyGate deferred to S267 | Generation proof is the priority; safety enforcement is next |
| Inline fill simulation | Matches existing derive-side design; venue round-trip is execute-side concern |

## Preparation for S267

S267 should focus on:
1. **SafetyGate integration**: Wire `SafetyGate.Check()` into `PaperOrderEvaluatorActor` before publishing
2. **Staleness enforcement**: Apply `StalenessGuard` in the actor chain
3. **Round-trip fill proof**: Prove that `PaperOrderSubmittedEvent` → `PaperVenueAdapter` → `VenueOrderFilledEvent` closes the loop
4. **KV materialization**: Prove that paper orders materialize in NATS KV and are queryable via `/execution/status/latest`
5. **ControlGate runtime test**: Prove that flipping the kill switch at runtime blocks subsequent paper orders
