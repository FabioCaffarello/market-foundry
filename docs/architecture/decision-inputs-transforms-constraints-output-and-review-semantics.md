# Decision Inputs, Transforms, Constraints, Output, and Review Semantics

**Stage:** S471
**Status:** Implemented
**Date:** 2026-03-25

## Purpose

This document defines the semantic model behind the decision review surface: what each section of a `DecisionReviewBundle` contains, how the data flows between pipeline stages, and what an operator should understand when reviewing a decision.

## Pipeline Stages as Review Sections

The market-foundry pipeline follows a strict causal chain:

```
Signal -> Decision -> Strategy -> Risk -> Execution
```

The decision review maps these stages to five review sections:

### 1. Inputs (Signal -> Decision boundary)

**What it shows:** The signal evidence that the decision evaluator consumed.

**Data source:** `Decision.Signals []SignalInput` — this is the decision's own record of what it saw. Each `SignalInput` carries:
- `type` — signal family (e.g., "rsi", "ema", "bollinger")
- `value` — the signal's primary value at evaluation time
- `timeframe` — the evidence window in seconds
- `event_id` — (S470) causal reference to the originating signal event

**Semantics:**
- The decision evaluator receives signals and evaluates them against thresholds.
- The `SignalInput` records are a decision-owned snapshot — they represent what the decision saw, not necessarily the latest signal value.
- If the chain's signal event is available, its `event_id` and `occurred_at` provide the exact signal origin.

**What it does NOT show:**
- Raw candle/evidence data that the signal was computed from (use `/analytical/evidence/candles` for that).
- Signal metadata (period, avg_gain, etc.) — available via `/analytical/signal/history` drill-down.

### 2. Transform (Decision evaluation)

**What it shows:** The decision evaluation itself — the core of the review.

**Data source:** `Decision` domain entity.

| Field | Semantics |
|---|---|
| `type` | Decision family (e.g., "rsi_oversold", "ema_crossover", "bollinger_squeeze") |
| `outcome` | Categorical result: `triggered`, `not_triggered`, `insufficient` |
| `severity` | Signal strength: `none`, `low`, `moderate`, `high` |
| `confidence` | Decimal string (0.0-1.0) — evaluator's confidence in the outcome |
| `rationale` | Human-readable explanation of why this outcome was produced |
| `final` | Whether this is a finalized evaluation (true) or interim (false) |
| `metadata` | Type-specific fields (e.g., threshold values, computed distances) |

**Semantics:**
- `outcome=triggered` means the signal conditions met the evaluator's criteria.
- `severity` is only meaningful when `outcome=triggered` — it grades how extreme the signal was. For `not_triggered`, severity is always `none`.
- `confidence` reflects the evaluator's certainty. Different evaluator families compute this differently (see actor implementations).
- `rationale` is the primary explainability field — it should answer "why this outcome?"

**What it does NOT show:**
- The evaluator's internal thresholds or configuration (these are in the actor's config, not in the event).
- Comparison with previous decisions (use batch review for side-by-side).

### 3. Resolution (Decision -> Strategy boundary)

**What it shows:** The strategy that was resolved from the decision.

**Data source:** `Strategy` domain entity.

| Field | Semantics |
|---|---|
| `type` | Strategy family (e.g., "mean_reversion_entry", "trend_following_entry") |
| `direction` | Positional intent: `long`, `short`, `flat` |
| `confidence` | Strategy-level confidence (may differ from decision confidence) |
| `decision_inputs` | The strategy's record of which decisions it consumed |
| `parameters` | Strategy-specific parameters (e.g., entry conditions, position sizing hints) |

**Semantics:**
- The strategy resolver combines one or more decisions into a directional intent.
- `decision_inputs` in the review shows what the strategy saw — each entry carries the decision's type, outcome, confidence, severity, and rationale.
- A `not_triggered` decision typically does not produce a strategy resolution, so this section will be absent.

**What it does NOT show:**
- Multi-decision composition logic (e.g., how multiple decisions are weighted). That logic lives in the strategy resolver actor.

### 4. Constraints (Strategy -> Risk boundary)

**What it shows:** The risk assessment applied to the strategy intent.

**Data source:** `RiskAssessment` domain entity.

| Field | Semantics |
|---|---|
| `type` | Risk evaluator family (e.g., "position_exposure") |
| `disposition` | Gate outcome: `approved`, `modified`, `rejected` |
| `confidence` | Risk evaluator's confidence in the assessment |
| `rationale` | Why this disposition (e.g., "within limits", "exposure exceeded") |
| `limits` | Active constraints: `max_position_size`, `max_exposure`, `stop_distance` |
| `strategy_context` | What the risk evaluator saw from the strategy stage |

**Semantics:**
- `disposition=approved` means the strategy intent passed all risk checks unchanged.
- `disposition=modified` means the risk evaluator adjusted parameters (e.g., reduced position size).
- `disposition=rejected` means the risk evaluator blocked the intent entirely.
- `limits` shows the constraints that were active at assessment time — these may have been dynamically computed based on strategy type and decision severity (S251 severity-aware scaling).
- `strategy_context` preserves the chain of reasoning from decision severity through strategy direction.

**What it does NOT show:**
- The evaluator's position state (current exposure, P&L). That is internal runtime state, not persisted in the event.
- Why limits were set to specific values (the scaling logic is in the risk evaluator actor).

### 5. Output (Risk -> Execution boundary)

**What it shows:** The execution intent and its lifecycle outcome.

**Data source:** `ExecutionIntent` domain entity.

| Field | Semantics |
|---|---|
| `type` | Execution type (e.g., "paper_order", "venue_market_order") |
| `side` | Order side: `buy`, `sell`, `none` |
| `quantity` | Requested quantity |
| `filled_quantity` | Actually filled quantity |
| `status` | Lifecycle status: `submitted`, `sent`, `accepted`, `filled`, `partially_filled`, `rejected`, `cancelled` |
| `final` | Whether the execution reached a terminal state |

**Semantics:**
- This section is only present if the decision chain produced an execution intent (requires triggered decision + approved/modified risk).
- `status` reflects the latest known lifecycle state at the time of the ClickHouse write.
- For real venue executions, use the session audit surface (`/session/audit`) for fill details.

**What it does NOT show:**
- Fill details (price, fees) — those are in the execution's `fills` array, available via `/analytical/execution/history` or the session audit bundle.
- Post-execution analysis (P&L, slippage). That is a downstream concern.

## Chain Completeness

Each review bundle carries `stage_count`, `chain_complete`, and `missing_stages`:

| Scenario | stage_count | chain_complete | missing_stages |
|---|---|---|---|
| Full chain (signal through filled execution) | 5 | true | [] |
| Decision triggered, execution pending | 3-4 | false | [execution] or [risk, execution] |
| Decision not_triggered | 1-2 | false | [strategy, risk, execution] |
| Signal event missing from ClickHouse | 4 | false | [signal] |

## Review Workflow

### "Why did this decision trigger?"

1. Call `GET /analytical/composite/decision/review?correlation_id=...&symbol=...`
2. Read the **Transform** section: `outcome`, `severity`, `rationale`
3. Read the **Inputs** section: which signals contributed and their values
4. If needed, drill into signal history via event_id

### "Why was this execution rejected by risk?"

1. Call the same endpoint with the chain's correlation_id
2. Read the **Constraints** section: `disposition=rejected`, `rationale`
3. Read `strategy_context` to see what the risk evaluator evaluated

### "Compare recent triggered decisions"

1. Call `GET /analytical/composite/decision/reviews?source=...&symbol=...&timeframe=...&outcome=triggered`
2. Compare **Transform** sections across bundles (severity, confidence, rationale)
3. Compare **Constraints** sections to see if risk disposition varies

## Limitations

1. **Batch mode is execution-rooted**: Decisions that never reached execution (e.g., `not_triggered`) are not returned in batch mode. Single-chain lookup by correlation_id works for any decision.

2. **No diff/comparison computation**: The surface returns individual bundles. Side-by-side comparison is left to the consumer.

3. **No raw signal drill-down**: The bundle carries `SignalInput` summaries, not raw signal values. Use the signal event_id to fetch the full signal via `/analytical/signal/history`.

4. **No evaluator config exposure**: The decision type names the evaluator family, but the evaluator's thresholds and configuration are not included in the event and thus not in the review.
