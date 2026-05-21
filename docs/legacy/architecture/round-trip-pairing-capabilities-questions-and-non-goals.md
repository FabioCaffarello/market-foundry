# Round-Trip Pairing -- Capabilities, Questions, and Non-Goals

**Wave**: Round-Trip Pairing
**Charter stage**: S479
**Date**: 2026-03-26

---

## 1. Capabilities

Each capability is a discrete, testable deliverable. Capabilities are graded at the evidence gate (S483).

### C-RT1: Canonical Round-Trip Model

**Block**: 1 (S480)

Define the domain types and matching rules for round-trip trades:

- `RoundTrip` type with entry leg, exit leg, matching metadata, lifecycle state.
- `Leg` type representing one side of a trade: direction (entry/exit), execution intent reference, fill data, fee data, timestamp.
- Matching rules: same symbol, same segment, opposite side (buy entry pairs with sell exit for longs; sell entry pairs with buy exit for shorts), temporal ordering (entry timestamp before exit timestamp).
- Pairing state: `paired` (both legs matched), `unmatched_entry` (entry without exit), `unmatched_exit` (exit without entry -- rare but possible with data gaps).
- Reason codes for unmatched legs: `no_exit_found`, `quantity_mismatch_remainder`, `session_boundary`, `rejected_leg`, `cancelled_leg`.

### C-RT2: FIFO Leg-Matching Strategy

**Block**: 1 (S480)

Implement the default matching algorithm:

- FIFO: earliest unmatched entry pairs with earliest available exit for the same symbol/segment.
- Partial-fill handling: when entry quantity exceeds exit quantity (or vice versa), match the minimum available quantity and leave the remainder as a separate unmatched leg.
- Proportional P&L: paired P&L computed on the matched quantity. Remainder carries forward as unmatched.
- One-to-one matching: each fill participates in at most one round-trip. No double-counting.
- Deterministic: same input data always produces same pairing result.

### C-RT3: Pairing Read Model

**Block**: 2 (S481)

Query existing execution data and produce paired round-trips:

- Read existing execution intents and fills from ClickHouse via CompositeReader.
- Group by symbol + segment + correlation-ID scope.
- Apply FIFO matching to produce `RoundTrip` instances.
- Wire each paired round-trip through `ClassifyPair()` to get win/loss/breakeven outcome and P&L attribution.
- Produce paired effectiveness attributions with the same `Attribution` struct used by S476.
- Track matching statistics: total entries, total exits, paired count, unmatched count, resolved rate.

### C-RT4: Paired Batch Effectiveness Integration

**Block**: 2 (S481)

Integrate paired outcomes into the existing effectiveness evaluation pipeline:

- Extend batch evaluation to include paired results alongside single-leg results.
- Paired attributions carry the same context fields (correlation_id, decision_type, strategy_type, severity, source).
- Resolved rate metric: `paired_resolved / total_chains` available in batch response.
- No change to existing single-leg effectiveness endpoints (additive only).

### C-RT5: Round-Trip Review Endpoint

**Block**: 3 (S482)

HTTP endpoint for reviewing paired outcomes:

- Endpoint pattern: `GET /analytical/composite/decision/effectiveness/pairs`.
- Returns list of round-trip records with entry leg, exit leg, outcome, P&L, matching metadata.
- Query parameters consistent with existing effectiveness endpoints: `source`, `symbol`, `timeframe`, `outcome`.
- Includes unmatched legs with reason codes.

### C-RT6: Outcome Reconciliation Surface

**Block**: 3 (S482)

Surface for inspecting and reconciling unresolved outcomes:

- List unmatched legs with structured reason codes.
- Reconciliation summary: matched count, unmatched count, resolved rate, unmatched reasons distribution.
- Extension of `DecisionReviewBundle` with optional pairing section when a round-trip is found for the reviewed decision.
- Pairing section includes: entry leg summary, exit leg summary, match quality indicator, round-trip P&L.

---

## 2. Governing Questions

| ID | Question | Capability | Stage |
|----|----------|-----------|-------|
| Q-RT1 | Can the system identify and pair entry/exit legs of a round-trip trade from existing execution data with canonical matching rules? | C-RT1, C-RT2 | S480 |
| Q-RT2 | Does automated pairing increase the resolved rate (reduce `unresolved` outcomes) compared to the pre-wave single-leg baseline? | C-RT3, C-RT4 | S481 |
| Q-RT3 | Are paired round-trip outcomes correctly classified (win/loss/breakeven) with accurate P&L attribution including fees? | C-RT2, C-RT3 | S480, S481 |
| Q-RT4 | Can the system surface paired outcomes through HTTP for review, and flag unmatched/unresolved legs with clear reasons? | C-RT5, C-RT6 | S482 |
| Q-RT5 | Is round-trip pairing computable from existing data without new exchange connectivity, new ClickHouse tables, or OMS expansion? | C-RT3 | S481 |

---

## 3. Non-Goals

Non-goals are frozen for the duration of this wave. Any item listed here MUST NOT be implemented, scoped, or designed within S480--S483.

### NG-RT1: OMS Expansion

No new order types, no order amendment, no order cancellation logic, no position tracking, no portfolio management. Pairing reads existing execution data; it does not change how orders are managed.

### NG-RT2: Position or Risk Engine

No ongoing position tracking. No risk exposure calculation. No margin requirements. Pairing determines historical trade outcomes, not current position state.

### NG-RT3: Portfolio-Level P&L Aggregation

No cross-symbol P&L totals. No account-level profit tracking. Each round-trip is per-symbol, per-segment. Portfolio analytics is a separate, future wave.

### NG-RT4: Real-Time Pairing or Streaming

No write-path pairing. No streaming pair events. Pairing is computed on read from settled data.

### NG-RT5: Multi-Exchange or Cross-Venue Pairing

No pairing across exchanges. No cross-venue trade matching. Single-venue (Binance) only.

### NG-RT6: Cross-Session Pairing

No pairing of entries from one session with exits from a different session beyond what correlation-ID scope provides. Cross-session lifecycle tracking is a separate wave.

### NG-RT7: Advanced Matching Strategies

No LIFO, HIFO (highest-in-first-out), or specific-lot matching. FIFO only for this wave. Alternative strategies are a future extension.

### NG-RT8: UI, Dashboards, or Visualization

No frontend components. HTTP endpoints only, consistent with existing gateway patterns.

### NG-RT9: Risk-Adjusted Return Metrics

No Sharpe ratio, Sortino ratio, or volatility-normalized returns. Raw P&L and win/loss classification from paired round-trips only.

### NG-RT10: New ClickHouse Tables or Schema Changes

Pairing is derived from existing execution and fill data. No new tables, no schema migrations.

### NG-RT11: Write-Path Changes

No changes to order submission, fill recording, fee normalization, or execution lifecycle events. Read-path extension only.

### NG-RT12: Slippage or Market Impact Analysis

No entry-vs-expected price analysis. No market impact metrics. Pairing uses fill prices only.

### NG-RT13: Strategy Family Expansion

No new strategy types. No new decision evaluators. No new signal families. The wave pairs existing executions, not new ones.

### NG-RT14: ML or Predictive Scoring

No models that predict pair outcomes. No feature engineering. Classification is deterministic and rule-based via existing `ClassifyPair()`.

### NG-RT15: Advanced Derivatives Handling

No funding rate integration. No liquidation pairing. No perpetual futures mark price. Standard spot and futures fill pairing only.

### NG-RT16: Alerting or Notification

No alerts on pairing outcomes, resolved rate thresholds, or P&L triggers. No integration with external notification systems.

### NG-RT17: Benchmark Comparison

No comparison of paired outcomes against market benchmarks (buy-and-hold, index). Outcomes are absolute.

### NG-RT18: Statistical Significance Testing

No p-values, confidence intervals, or significance tests on paired cohort comparisons. Raw counts and rates only.

---

## 4. Capability-to-Question Traceability

| Capability | Questions Answered | Stage |
|-----------|-------------------|-------|
| C-RT1 | Q-RT1 | S480 |
| C-RT2 | Q-RT1, Q-RT3 | S480 |
| C-RT3 | Q-RT2, Q-RT3, Q-RT5 | S481 |
| C-RT4 | Q-RT2 | S481 |
| C-RT5 | Q-RT4 | S482 |
| C-RT6 | Q-RT4 | S482 |

---

## 5. Assumptions

1. **Execution data contains both entries and exits.** At least some sessions have both buy and sell fills for the same symbol, making pairing possible. If no exits exist at all, the pairing model is still valid but the resolved rate improvement will be zero -- this is documented, not a failure.
2. **Fill data is complete and correct.** Pairing relies on `FillRecord` data already validated by the execution pipeline. Incomplete fills produce unmatched legs with reason codes.
3. **FIFO is sufficient for the current trading model.** The system executes simple directional strategies (trend-following, squeeze breakout). FIFO matching aligns with the temporal execution order. Complex multi-leg strategies requiring specific-lot matching are not in scope.
4. **CorrelationID provides sufficient scope for matching.** Entries and exits within the same correlation chain can be paired. Entries without a correlated exit are unmatched.
5. **`ClassifyPair()` is stable and tested.** The domain function from S476 handles all P&L computation. This wave provides the inputs; it does not change the classification logic.
6. **Existing effectiveness endpoints remain unchanged.** Pairing is additive. Existing single-leg effectiveness continues to work. Paired effectiveness is a new surface alongside the existing one.

---

## 6. References

- [Wave Charter and Scope Freeze](round-trip-pairing-wave-charter-and-scope-freeze.md)
- [S478 Evidence Gate Report](../stages/stage-s478-strategy-effectiveness-evidence-gate-report.md)
- [Effectiveness Evidence Matrix and Residual Gaps](strategy-effectiveness-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Effectiveness Capabilities and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
- [S479 Charter Report](../stages/stage-s479-round-trip-pairing-charter-report.md)
