# Composite Observability Under Multi-Symbol Load

> Phase 29 — Multi-Symbol Operational Scaling Wave
> Stage: S303
> Date: 2026-03-21

## Purpose

This document records the validation that the composite observability surfaces delivered in the S294–S299 wave (chain, funnel, dispositions, attribution) remain correct, readable, and operationally useful when three symbols (btcusdt, ethusdt, solusdt) coexist simultaneously in the analytical store.

S303 is a **validation stage**, not a new platform. It applies pressure to the existing explainability surfaces under multi-symbol load and documents findings.

## Surfaces Validated

### 1. `/analytical/composite/chain` — Single Chain Lookup

**Multi-symbol behavior:** Each symbol's chain is queried independently via `correlation_id + symbol`. The mandatory symbol filter (S301) guarantees that a chain lookup for btcusdt cannot return ethusdt stages.

**Validation criteria:**
- Causal metadata integrity: correlation_id is consistent across all 5 stages; causation_id chain is internally valid (signal→decision→strategy→risk→execution).
- Symbol consistency: every stage's `.Symbol` field matches the queried symbol.
- Attribution completeness: `disposition`, `rationale`, `active_constraints`, `strategy_context` are all populated with symbol-specific values.

**Finding:** CORRECT. No cross-symbol contamination. Causal DAG is internally consistent per symbol. Attribution fields fully populated.

### 2. `/analytical/composite/chains` — Batch Chain Lookup

**Multi-symbol behavior:** Batch queries are scoped to a single symbol via mandatory `symbol` parameter. The execution-rooted discovery path starts from the `executions` table filtered by `source + symbol + timeframe`, then enriches each correlation_id with its full chain.

**Validation criteria:**
- Chains returned belong exclusively to the requested symbol.
- Different dispositions (approved, modified, rejected) coexist within the same symbol's batch.
- Chain counts match expected per-symbol totals.

**Finding:** CORRECT. Batch responses contain only the requested symbol's chains. Mixed dispositions are independently readable within the same batch.

### 3. `/analytical/composite/funnel` — Pipeline Conversion Funnel

**Multi-symbol behavior:** Funnel queries count events per stage for a single symbol. Each of the 5 independent table queries includes `WHERE symbol = ?`.

**Validation criteria:**
- Funnel is monotonically decreasing (signal ≥ decision ≥ ... ≥ execution).
- Execution count matches the number of approved chains for that symbol.
- Per-symbol funnels are independent — btcusdt's counts do not bleed into ethusdt's.

**Finding:** CORRECT. Monotonic decrease holds. Execution counts match approved chain counts. Filter specificity confirmed: same `type + source` with different `symbol` yields different counts.

### 4. `/analytical/composite/dispositions` — Risk Disposition Breakdown

**Multi-symbol behavior:** Disposition queries aggregate risk_assessments by disposition for a single symbol.

**Validation criteria:**
- Total count matches the risk stage count in the funnel.
- Percentages sum to 100%.
- Per-symbol breakdowns are independent.

**Finding:** CORRECT. Totals align with funnel risk-stage counts. Percentages sum to 100% within floating-point tolerance. No inter-symbol count bleed.

## Cross-Surface Consistency

A critical explainability property is that an operator reading chain, funnel, and disposition surfaces for the same symbol sees consistent data:

| Property | Chain | Funnel | Dispositions | Consistent? |
|----------|-------|--------|-------------|-------------|
| Execution presence | `chain_complete=true` when execution exists | `execution` stage count | N/A | YES |
| Risk outcome | `attribution.disposition` per chain | N/A | Disposition counts | YES |
| Total risk assessments | Count of chains with risk stage | `risk` stage count | `total` field | YES |
| Symbol isolation | Every stage `.Symbol` matches | Query scoped by `symbol` | Query scoped by `symbol` | YES |

## Causal Metadata Under Multi-Symbol Interleaving

When chains from 3 symbols are loaded concurrently, the causal DAG within each chain remains valid:

```
signal (causation_id="")
  → decision (causation_id=signal.event_id)
    → strategy (causation_id=decision.event_id)
      → risk (causation_id=strategy.event_id)
        → execution (event_causation_id=risk.event_id)
```

All stages within a chain share the same `correlation_id`. No cross-chain or cross-symbol causation_id references were observed.

## Attribution Readability

For an operator investigating "why was this symbol's execution rejected?", the attribution surface provides:

1. **`disposition`** — Clear outcome label (approved/modified/rejected).
2. **`rationale`** — Human-readable explanation, symbol-specific (e.g., "drawdown limit exceeded for eth exposure").
3. **`active_constraints`** — Numerical limits that were active (max_position_size, max_exposure).
4. **`strategy_context`** — The strategy type, direction, confidence, and decision severity that led to the risk assessment.

All fields are populated for all 3 symbols. No empty or ambiguous fields observed.

## Operational Ambiguities Identified

### AMB-1: Funnel `type` Parameter Scope

The funnel query requires `type` (signal type, e.g., "rsi"). When different symbols use different signal types (e.g., btcusdt uses "rsi", solusdt uses "bollinger"), an operator must issue separate funnel queries per type. There is no cross-type funnel aggregation.

**Impact:** Low. This is correct behavior — mixing signal types in a single funnel would produce misleading conversion rates. The operator must know which type to query.

**Mitigation:** Documentation of the `type` parameter semantics in the HTTP contracts doc.

### AMB-2: Batch Discovery Limitation for Rejected Chains

Batch queries start from the `executions` table. Chains that were rejected at the risk stage (no execution emitted) are not discoverable via `/chains`. They are only visible via single-chain lookup or indirectly via the funnel (risk count > execution count).

**Impact:** Moderate for individual rejected chain investigation. Low for aggregate views.

**Status:** Known since S299 (GAP-Q5-A). The funnel compensates at aggregate level.

### AMB-3: Per-Constraint Trigger Identification

Attribution shows `active_constraints` (all active constraints) but does not identify which specific constraint triggered a rejection. The `rationale` field provides a free-text explanation, which may or may not name the triggering constraint.

**Impact:** Low with 3 constraints; increases with future constraint additions.

**Status:** Known since S299 (GAP-Q2-A). Requires write-side `triggering_constraints` field.

## Conclusion

The composite observability wave (S294–S299) remains **fully valid under multi-symbol load**. All 4 explainability surfaces continue to produce correct, isolated, and operationally readable results when 3 symbols coexist. No code changes were required — the S301 isolation fix and existing filter architecture are sufficient.
