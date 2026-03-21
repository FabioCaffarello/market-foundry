# Multi-Symbol Risk/Execution Findings and Operational Limits

Stage: S304
Status: Validated
Date: 2026-03-21

## Findings

### 1. Risk Evaluators Are Fully Symbol-Isolated

Each PositionExposureEvaluator and DrawdownLimitEvaluator instance is constructed with a specific (source, symbol, timeframe) tuple. There is no shared mutable state between instances. Concurrent evaluation of btcusdt, ethusdt, and solusdt produces independent results with no cross-symbol contamination.

Evidence: RE-1 through RE-6, all passing.

### 2. Strategy-Type Scaling Is Correctly Applied Per Symbol

When different symbols use different strategy types (mean_reversion, trend_following, squeeze_breakout), each receives the correct strategy-specific confidence factor, stop distance factor, and severity multiplier. The scaling maps are package-level constants, not instance state, and are read-only.

Evidence: RE-2 (strategy-type confidence), RE-4 (stop distance diversity).

### 3. Disposition-Driven Execution Blocking Works Per Symbol

When one symbol's risk is rejected while others are approved/modified, only the rejected symbol's execution is blocked. The other symbols' paper orders proceed normally through the fill lifecycle.

Evidence: RE-3 (mixed dispositions), EX-1 (disposition→side mapping), EX-3 (rejection blocks per symbol).

### 4. Paper Fill Lifecycle Is Symbol-Scoped

The PaperFillSimulator and PaperVenueAdapter produce fill records and venue order IDs per symbol without interference. Fill quantities match risk-constrained positions. Venue order IDs are cryptographically unique.

Evidence: EX-2 (full lifecycle), EX-4 (modified quantity), EX-6 (venue adapter isolation).

### 5. Causal Context Survives Across Domain Boundaries

Strategy type and decision severity are preserved through the risk→execution boundary in both the RiskInput struct and the Parameters map. This enables end-to-end attribution from signal to execution for any symbol.

Evidence: EX-5 (causal context preservation), RX-2 (attribution diversity).

### 6. Composite Read Model Correctly Reflects Multi-Symbol Risk/Execution State

The composite read model assembles chains with correct stage counts, attribution, and execution presence/absence based on risk disposition. Cross-surface alignment (funnel, disposition breakdown, chain) is consistent per symbol.

Evidence: RX-1 through RX-5.

### 7. No Code Changes Required

The existing risk and execution code required zero modifications to support multi-symbol concurrency. All isolation was already in place by design (per-instance evaluators, symbol-scoped partition keys, stateless evaluation functions).

## Operational Limits

### L1: Paper Mode Only

All validation is in paper mode. Paper fills are instant (zero latency), use price "0", and have no market impact. Real venue behavior (latency, partial fills, rejections, slippage) is not covered.

### L2: No Portfolio-Level Risk Aggregation

Each symbol's risk is evaluated independently. There is no cross-symbol portfolio-level risk check (e.g., total portfolio exposure across all symbols). If btcusdt and ethusdt are both approved with maxExposure=10%, the combined exposure is 20% — no aggregate limit enforces this.

### L3: No Concurrent Actor-Level Testing

Tests validate evaluator logic and composite read model, but do not exercise the actor layer (RiskEvaluatorActor, PaperOrderEvaluatorActor) under true concurrent message delivery from multiple symbols. Actor scoping provides isolation, but message ordering under load is untested.

### L4: Single Risk Type Per Chain

Each chain carries a single risk assessment (position_exposure or drawdown_limit). The architecture does not yet compose multiple risk assessments for the same symbol in a single chain (e.g., requiring both position exposure approval AND drawdown limit approval before execution proceeds).

### L5: No Time-Window Risk Accumulation

Risk evaluation is stateless and per-event. There is no mechanism to track accumulated risk across time (e.g., "this symbol has been approved 10 times in 5 minutes — is that safe?"). This is a deliberate design choice for simplicity but limits operational risk management.

### L6: Scaling Factor Maps Are Static

Strategy-type confidence factors and severity multipliers are compile-time constants. They cannot be adjusted at runtime (e.g., through configuration or control plane commands) without redeployment.

## Remediation Priority

| Limit | Severity | When to Address |
|---|---|---|
| L1 (paper only) | Expected | Venue readiness wave |
| L2 (no portfolio aggregation) | Medium | Portfolio risk wave (post-venue) |
| L3 (no actor concurrency test) | Low | Integration hardening wave |
| L4 (single risk type per chain) | Low | Risk composition wave |
| L5 (no time-window accumulation) | Low | Operational risk wave |
| L6 (static scaling factors) | Low | Runtime config wave |
