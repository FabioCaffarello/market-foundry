# Strategy Effectiveness -- Capabilities, Questions, and Non-Goals

**Wave**: Strategy Effectiveness Measurement
**Charter stage**: S474
**Date**: 2026-03-25

---

## 1. Capabilities

Each capability is a discrete, testable deliverable. Capabilities are graded at the evidence gate (S478).

### C-SE1: Canonical Effectiveness Outcome Model

**Block**: 1 (S475)

Define the domain types and classification rules for effectiveness:

- `EffectivenessOutcome` with canonical values: `win`, `loss`, `breakeven`, `unresolved`.
- Classification rules based on realized P&L from fill data.
- Breakeven threshold (configurable tolerance around zero).
- Partial-fill handling: classification based on filled portion only.
- Cancelled-before-fill: classified as `unresolved`, not `loss`.
- Rejected orders: excluded from effectiveness (no fill, no outcome).

### C-SE2: P&L Attribution Per Decision Chain

**Block**: 1 (S475)

Attribute realized profit/loss to the originating decision chain:

- Entry cost basis from `FillRecord.CostBasis`.
- Fee impact from `FillRecord.Fee` and `FillRecord.FeeAsset`.
- Net P&L: `(exit_value - entry_value) - total_fees` for completed round-trips.
- Single-leg attribution: for fills without a paired exit within the session, P&L is `unresolved`.
- Attribution carries `CorrelationID` to link back to originating decision.
- Attribution carries decision metadata: type, severity, confidence, strategy type, direction.

### C-SE3: Effectiveness Computation From Existing Data

**Block**: 2 (S476)

Compute effectiveness without new infrastructure:

- Derive effectiveness from existing `FillRecord` data in ClickHouse.
- No new tables; effectiveness is a read-path computation.
- Deterministic: same inputs always produce same classification.
- Handles fee normalization (uses S428 fee model).

### C-SE4: Batch Effectiveness Evaluation Endpoint

**Block**: 2 (S476)

HTTP endpoint for batch effectiveness queries:

- Endpoint pattern: `GET /analytical/composite/decision/effectiveness`.
- Query parameters: `source`, `symbol`, `timeframe`, `decision_type`, `strategy_type`, `severity`, `outcome` (decision outcome filter), `effectiveness` (win/loss/breakeven filter).
- Returns list of effectiveness records with attribution metadata.
- Pagination or window support consistent with existing batch endpoints.

### C-SE5: Effectiveness Section in DecisionReviewBundle

**Block**: 2 (S476)

Extend the existing review bundle with effectiveness data:

- New `Effectiveness` section in `DecisionReviewBundle`.
- Contains: outcome classification, realized P&L, fee impact, attribution metadata.
- Present only when execution reached terminal state (filled/cancelled).
- Null/absent for in-progress or not-triggered decisions.
- Human-readable effectiveness explanation appended to existing `Explanation` field.

### C-SE6: Comparative Effectiveness Analysis

**Block**: 3 (S477)

Comparative analysis across decision cohorts:

- Aggregation by: decision type, strategy type, signal severity, timeframe, source.
- Metrics per cohort: win count, loss count, breakeven count, total P&L, average P&L, win rate.
- Endpoint pattern: `GET /analytical/composite/decision/effectiveness/summary`.
- No cross-symbol aggregation (per guard rail).
- No risk-adjusted metrics (per guard rail).

### C-SE7: Cohort Comparison Endpoint

**Block**: 3 (S477)

Side-by-side comparison of effectiveness between cohorts:

- Compare two or more cohorts by different dimension values (e.g., severity=high vs severity=low).
- Returns parallel effectiveness summaries for visual or programmatic comparison.
- Leverages C-SE6 aggregation logic.

---

## 2. Governing Questions

| ID | Question | Capability | Stage |
|----|----------|-----------|-------|
| Q-SE1 | Can the system classify each completed decision chain as win, loss, or breakeven with canonical semantics? | C-SE1 | S475 |
| Q-SE2 | Can the system attribute realized P&L (price delta, fee impact) to the originating decision and its causal inputs? | C-SE2 | S475, S476 |
| Q-SE3 | Is effectiveness computable from existing fill and fee data without new exchange connectivity? | C-SE3 | S476 |
| Q-SE4 | Can the system batch-evaluate effectiveness across a cohort of decisions (by type, timeframe, source, severity)? | C-SE4 | S476 |
| Q-SE5 | Can the system surface comparative effectiveness analysis (which decision types or strategies outperform?) | C-SE6, C-SE7 | S477 |

---

## 3. Non-Goals

Non-goals are frozen for the duration of this wave. Any item listed here MUST NOT be implemented, scoped, or designed within S475--S478.

### NG-SE1: Portfolio-Level Analytics

No cross-symbol, cross-session, or account-level aggregation. Effectiveness is scoped to individual decision chains within a single symbol and session context.

### NG-SE2: Risk-Adjusted Return Metrics

No Sharpe ratio, Sortino ratio, Calmar ratio, or any metric that normalizes returns by volatility or drawdown. Raw P&L and win/loss classification only.

### NG-SE3: Real-Time Effectiveness Streaming

No write-path effectiveness computation. No streaming updates. Effectiveness is computed on read, from settled fill data.

### NG-SE4: Predictive or ML-Based Signal Scoring

No models that predict future effectiveness based on historical patterns. No feature engineering for signal quality prediction. Classification is deterministic and rule-based.

### NG-SE5: New ClickHouse Tables or Schema Changes

Effectiveness is derived from existing execution, fill, and fee data. No new tables, no schema migrations.

### NG-SE6: UI, Dashboards, or Visualization

No frontend components. HTTP endpoints only, consistent with existing gateway patterns.

### NG-SE7: OMS Expansion

No order management system changes. No new order types, no position tracking, no portfolio management.

### NG-SE8: Multi-Exchange or Multi-Venue Work

No new exchange adapters. No cross-venue effectiveness comparison. Single-venue effectiveness only.

### NG-SE9: Strategy Family Expansion

No new strategy types. No new decision evaluators. No new signal families. The wave measures existing decisions, not new ones.

### NG-SE10: Alerting or Notification

No alerts on effectiveness thresholds. No notifications when win rate drops. No integration with external notification systems.

### NG-SE11: Position Tracking or Mark-to-Market

No real-time position valuation. No mark-to-market pricing. Effectiveness uses only realized fill data, not unrealized positions.

### NG-SE12: Drawdown Analytics

No drawdown curves, peak-to-trough analysis, or recovery metrics. These belong in a performance analytics wave.

### NG-SE13: Time-Weighted or Money-Weighted Returns

No TWR, MWR, or IRR calculations. Raw P&L attribution only.

### NG-SE14: Benchmark Comparison

No comparison against market benchmarks (buy-and-hold, index). Effectiveness is absolute, not relative.

### NG-SE15: Backtesting or Historical Replay

No backtesting infrastructure. No historical replay. Effectiveness is computed from actual execution data only.

### NG-SE16: Domain Type Refactoring

No changes to existing `Decision`, `Strategy`, `RiskAssessment`, or `ExecutionIntent` types. Effectiveness is additive. New types only.

### NG-SE17: Write-Path Changes

No changes to the execution write path, order submission, fill recording, or fee normalization. Read-path extension only.

### NG-SE18: Cross-Session Attribution

No linking of decisions across sessions. Each session's effectiveness is self-contained. Cross-session strategies are a future wave concern.

### NG-SE19: Slippage Analysis

No entry-vs-expected price analysis. No slippage metrics. These require market data at decision time which is not reliably tagged (RG-2 from S473).

### NG-SE20: Capacity or Sizing Analysis

No analysis of whether decision quality degrades with position size. No capacity constraints. Size-effectiveness correlation is a future concern.

---

## 4. Capability-to-Question Traceability

| Capability | Questions Answered | Stage |
|-----------|-------------------|-------|
| C-SE1 | Q-SE1 | S475 |
| C-SE2 | Q-SE2 | S475 |
| C-SE3 | Q-SE2, Q-SE3 | S476 |
| C-SE4 | Q-SE4 | S476 |
| C-SE5 | Q-SE4 | S476 |
| C-SE6 | Q-SE5 | S477 |
| C-SE7 | Q-SE5 | S477 |

---

## 5. Assumptions

1. **Fill data is complete and correct.** Effectiveness relies on `FillRecord` data already validated by the execution pipeline. If fill data is incomplete, effectiveness is `unresolved`.
2. **Single-leg fills are common.** Many decisions will produce only entry fills without paired exits within the same session. The model must handle this gracefully with the `unresolved` classification.
3. **Fee normalization is reliable.** S428 fee normalization is trusted. Effectiveness P&L includes fee impact without re-normalizing.
4. **Decision review bundle is stable.** The `DecisionReviewBundle` structure from S471 is not being modified by other work. Effectiveness adds a section; it does not restructure existing sections.
5. **Batch endpoint patterns are established.** The existing `/decision/reviews` endpoint pattern is the template for effectiveness endpoints.

---

## 6. References

- [Wave Charter and Scope Freeze](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [S473 Evidence Gate](../stages/stage-s473-decision-quality-evidence-gate-report.md)
- [Decision Quality Capabilities and Non-Goals](decision-quality-capabilities-questions-and-non-goals.md)
- [S474 Charter Report](../stages/stage-s474-strategy-effectiveness-charter-report.md)
