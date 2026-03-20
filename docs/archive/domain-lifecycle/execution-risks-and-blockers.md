# Execution Risks and Blockers

> Concrete risks, blockers, and mitigation strategies for opening an `execution` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S68

## Purpose

This document catalogs every known risk and blocker that could prevent or compromise an `execution` domain implementation. Each entry is specific, actionable, and prioritized. No entry is aspirational or abstract — every item reflects a concrete gap or structural concern identified during the S68 readiness review.

---

## Hard Blockers

These must be resolved before any execution code is written. They are not negotiable.

### HB-1: Adapter Test Debt (6 domains × 2 adapter types)

**What**: NATS publisher and consumer adapters across all domains have zero unit tests. This debt has been carried since S59 and applies uniformly to observation, evidence, signal, decision, strategy, and risk.

**Why this blocks execution**: A message encoding bug in an analytical domain results in data quality issues detectable through query surfaces. A message encoding bug in an execution domain could produce malformed order intents, duplicate submissions, or lost execution signals. The blast radius is fundamentally different.

**Mitigation**: Dedicated test coverage sweep (S69) following established patterns — the registry, KV store, and projection tests demonstrate how to test NATS adapters without running infrastructure.

**Effort estimate**: MEDIUM — 12 adapter types need tests, but the pattern is repetitive.

### HB-2: No Automated Traceability Verification

**What**: The causal chain (correlation_id + causation_id) was hardened in S67, but verification is manual — visual log inspection or smoke test observation.

**Why this blocks execution**: For execution, the audit trail is not optional. A broken causation chain means an order cannot be traced back to the market data, signal, decision, strategy, and risk assessment that produced it. This is unacceptable for post-trade analysis, incident investigation, and regulatory compliance.

**Mitigation**: Integration test (S71) with running NATS that asserts the full chain. Pattern: publish synthetic trade → wait for chain completion → query all projections → verify correlation/causation integrity.

**Effort estimate**: MEDIUM — requires test infrastructure (embedded NATS or Docker NATS) and chain completion detection.

### HB-3: Trace Metadata Not Persisted in Projections

**What**: KV projections store only domain model fields. CorrelationID and CausationID are available in JetStream streams and structured logs but not in materialized state.

**Why this blocks execution**: Querying `GET /execution/:type/latest` would return an execution intent without its audit trail. Reconstructing the trace requires JetStream replay or log aggregation — neither is acceptable for real-time audit queries.

**Mitigation**: Design decision (S72) on persistence strategy. Options: embed in KV, separate audit bucket, dedicated audit stream, or hybrid.

**Effort estimate**: LOW (design) + MEDIUM (implementation).

### HB-4: Execution Domain Boundary Undefined

**What**: No design document exists for the execution domain. The term "execution" is used informally in the codebase and documentation without formal type definitions, lifecycle semantics, or ownership boundaries.

**Why this blocks execution**: Without a domain model, implementation would be speculative. Every previous domain (signal, decision, strategy, risk) was fully designed before code was written. Execution cannot be an exception — it is the highest-stakes domain.

**Mitigation**: Dedicated design stage (S73) producing `execution-domain-design.md` following the 2400+ line `strategy-domain-design.md` pattern.

**Effort estimate**: HIGH — execution is the most complex domain to define because it touches external systems.

---

## Structural Risks

These are not blockers but represent structural challenges that must be addressed during design.

### SR-1: Unidirectional Flow Model May Not Suffice

**Current model**:
```
configctl → ingest → derive → store → gateway
```

Every data flow is unidirectional. No binary sends data "upstream." Store never tells derive to do something. Gateway never pushes to store.

**Execution challenge**: An execution layer may need to:
1. Send orders to an external venue (outbound flow from derive or a new binary)
2. Receive fill confirmations from the venue (inbound flow into a new consumer)
3. Update execution state based on venue responses (feedback loop)

This introduces a **bidirectional flow** for the first time. Options:
- **Option A**: Execution is purely outbound (fire-and-forget intent). Venue interaction is a separate system.
- **Option B**: Execution includes a venue adapter within the existing binary topology (e.g., a new `execute` binary).
- **Option C**: Execution intent is projected by store, and a separate system reads the projection to place orders.

**Recommendation**: Start with Option A (paper execution, fire-and-forget intent). The venue interaction question can be deferred until the domain model is proven.

**Impact if unaddressed**: Premature venue adapter integration could compromise the clean unidirectional mesh that is the foundation of the architecture.

### SR-2: Latest-Only Projection May Be Insufficient

**Current model**: All domains use latest-only KV projections. The risk assessment for `binancef.btcusdt.60` is overwritten by each new assessment. History is available only in JetStream streams (retention-limited).

**Execution challenge**: An execution intent needs lifecycle tracking:
- Intent created → submitted → filled → cancelled → expired
- Latest-only means only the current state is visible
- Previous intents are lost (no execution history)
- Concurrent intents for the same symbol are invisible

**Options**:
1. **History bucket**: Like candle history, execution keeps N most recent intents.
2. **State machine projection**: KV entry tracks lifecycle state, not just latest value.
3. **Separate state store**: Execution uses a different projection pattern (not just latest).

**Recommendation**: The first execution slice should use latest-only (consistent with all other domains). History and state tracking are second-slice concerns.

**Impact if unaddressed**: An execution layer without lifecycle tracking would be operationally blind to order state beyond the most recent intent.

### SR-3: Risk Assessment Staleness Window

**Current model**: `GET /risk/:type/latest` returns the most recent risk assessment. There is no concept of "how old is this assessment" or "is this still valid."

**Execution challenge**: An execution layer consuming risk assessments needs to know:
- Is this assessment based on current market data or stale data?
- Should execution proceed if the risk assessment is older than N seconds?
- What happens if the risk pipeline is delayed or stuck?

**Mitigation**: Add a staleness check at the execution boundary:
- Compare `risk_assessment.timestamp` against current time
- Reject assessments older than configurable threshold (e.g., 2× timeframe)
- Log staleness events for operational visibility

**Impact if unaddressed**: Execution could act on stale risk assessments that no longer reflect market conditions.

### SR-4: Single Risk Family Limits Execution Gate

**Current model**: Only `position_exposure` risk family is implemented. One risk evaluator, one disposition.

**Execution challenge**: Real execution gating typically requires multiple independent risk checks:
- Position exposure (implemented)
- Drawdown limits (not implemented)
- Concentration risk (not implemented)
- Market conditions (not implemented)

**Recommendation**: The first execution slice should depend on the single existing risk family. Multi-risk aggregation is a design concern for `execution-domain-design.md`, not a blocker.

**Impact if unaddressed**: Execution would proceed with a single risk dimension, which is acceptable for paper trading but insufficient for live trading.

---

## Operational Risks

### OR-1: No Kill Switch Mechanism

**What**: There is no way to halt execution without stopping the binary. Configuration changes require restart via `pipeline.execution_families` removal.

**Why this matters**: In a live execution scenario, the time between detecting a problem and halting execution could be critical. A restart takes seconds; a kill switch takes milliseconds.

**Mitigation**: S76 designs a configctl-driven kill switch:
- `configctl` publishes `execution_halted` event
- Execution actors subscribe and immediately stop processing
- State: "halted" is projected, visible via gateway

**Severity**: NOT blocking for paper execution. CRITICAL for live execution.

### OR-2: No Execution-Specific Health Metrics

**What**: Health tracking exists for all current domains (projection trackers, consumer trackers). Execution would need:
- Orders submitted per minute
- Fill rate (for live)
- Rejection rate
- Latency from risk assessment to execution intent
- Kill switch state

**Mitigation**: Follow existing `healthz.Tracker` pattern. Extend with execution-specific metrics.

**Severity**: MEDIUM — operational visibility required before live execution.

### OR-3: No Rate Limiting Infrastructure

**What**: The system currently has no rate limiting. Each event in the chain is processed as fast as possible.

**Execution challenge**: Venue APIs have rate limits. Even paper execution should respect configurable rate limits to simulate real-world constraints.

**Mitigation**: Rate limiter at execution actor level, configurable per source/symbol.

**Severity**: LOW for paper execution. HIGH for live execution.

---

## Governance Risks

### GR-1: Risk Drift Rules Not Explicitly Verified

**What**: The risk domain was implemented in S59-S67, but the risk-specific drift rules in raccoon-cli have not been explicitly verified as passing in the current codebase.

**Mitigation**: S70 runs `raccoon-cli drift-detect` and fixes any failures.

**Severity**: MEDIUM — drift may exist silently.

### GR-2: Non-Goals NG-7 Phase Boundary

**What**: `docs/architecture/non-goals.md` (NG-7) states: "Market Foundry does not implement trading execution in Phase 2." Strategy and risk were classified as Phase 2 extensions. Execution is Phase 3.

**Implication**: Opening execution represents a phase transition. This must be acknowledged in the stage report. The phase boundary should be updated in non-goals.md to reflect that Phase 2 (observation→evidence→signal→decision→strategy→risk) is complete and Phase 3 (execution) is being gated.

**Severity**: LOW — administrative, not structural.

### GR-3: Execution Is the First Domain Without a MarketMonkey Reference

**What**: Signal, decision, strategy, and risk all have pattern references in MarketMonkey. Execution does not — MarketMonkey is a data processing pipeline, not a trading system.

**Implication**: Execution domain design cannot rely on MarketMonkey patterns. It must be designed from first principles, informed by Market Raccoon's domain authority.

**Severity**: MEDIUM — increases design risk. Mitigated by the structured readiness gate process.

---

## Risk Priority Matrix

| ID | Risk | Severity | Blocks Paper? | Blocks Live? | Resolution Stage |
|----|------|----------|---------------|--------------|-----------------|
| HB-1 | Adapter test debt | HIGH | YES | YES | S69 |
| HB-2 | No automated trace verification | HIGH | YES | YES | S71 |
| HB-3 | Trace not persisted in KV | HIGH | YES | YES | S72 |
| HB-4 | Domain boundary undefined | HIGH | YES | YES | S73 |
| SR-1 | Unidirectional flow model | MEDIUM | NO (paper=fire-and-forget) | YES | S73 design |
| SR-2 | Latest-only projection limits | MEDIUM | NO (acceptable for first slice) | YES | S73 design |
| SR-3 | Risk staleness window | MEDIUM | NO (paper tolerates staleness) | YES | S75 implementation |
| SR-4 | Single risk family | LOW | NO | YES (for production) | Future |
| OR-1 | No kill switch | LOW (paper) | NO | YES | S76 |
| OR-2 | No execution health metrics | MEDIUM | NO | YES | S75 |
| OR-3 | No rate limiting | LOW | NO | YES | Future |
| GR-1 | Risk drift unverified | MEDIUM | YES (governance gate) | YES | S70 |
| GR-2 | Phase boundary update | LOW | NO | NO | S73 |
| GR-3 | No MarketMonkey reference | MEDIUM | NO | NO | S73 design |

---

## Summary

**Total hard blockers**: 4 (HB-1 through HB-4)
**Total structural risks**: 4 (SR-1 through SR-4)
**Total operational risks**: 3 (OR-1 through OR-3)
**Total governance risks**: 3 (GR-1 through GR-3)

**Estimated stages to clear hard blockers**: 5 (S69-S73)
**Estimated stages to first paper execution**: 7 (S69-S75)
**Estimated stages to kill switch**: 8 (S69-S76)

The path to execution is clear but non-trivial. Every blocker has a concrete resolution path. No blocker requires architectural redesign — the foundation is sound.
