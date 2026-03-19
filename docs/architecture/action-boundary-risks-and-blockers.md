# Action Boundary Risks and Blockers

> Concrete risks, blockers, and mitigation strategies for crossing the action boundary in Market Foundry.
> Date: 2026-03-18 | Stage: S74

## Purpose

This document catalogs every known risk and blocker for transitioning from paper execution to venue-integrated execution. It builds on the S68 risk assessment but evaluates the current state after S69-S73 implementation.

---

## Hard Blockers

These must be resolved before any venue adapter code is written. They are non-negotiable.

### HB-1: Execution Domain Model Lacks Lifecycle

**What**: ExecutionIntent has only `StatusSubmitted`. No status for sent, accepted, filled, rejected, cancelled, or expired.

**Current impact**: Paper execution is unaffected — all intents are fire-and-forget. Venue integration requires tracking what happened after submission.

**Resolution**: Extend domain model with lifecycle enum and transition rules. Design in S75, implement in subsequent stage.

**Severity**: BLOCKING for venue integration.

### HB-2: No Fill Tracking Fields

**What**: No filled_quantity, average_fill_price, fill_timestamp, or venue_order_id in the domain model.

**Current impact**: The system generates intents but cannot record outcomes. Impossible to know if an order was filled, at what price, or how much.

**Resolution**: Design fill model in S75. Decide: extend ExecutionIntent or create ExecutionFill entity.

**Severity**: BLOCKING for venue integration.

### HB-3: Silent Data Loss on Failure Paths

**What**: Two critical paths silently lose data:
1. **Publish failure**: NATS publish error → log + drop. No retry, no DLQ. Event is lost permanently.
2. **Projection write failure**: KV write fails → log + ACK. Message is acknowledged as processed despite write failure. Projection has a gap.

**Current impact**: For paper execution with low volume, these failures are rare and consequence-free. For venue integration, a lost event means an untracked order or a phantom position.

**Resolution**:
- Publish: Add configurable retry with exponential backoff for transient errors. Circuit breaker for persistent failures.
- Projection: NAK on KV write failure to trigger redelivery. Only ACK after successful write.

**Severity**: BLOCKING for venue integration.

### HB-4: Trace Metadata Not Queryable

**What**: KV projections don't persist correlation_id or causation_id. The execution query surface returns intent data without audit trail. Full traceability requires JetStream replay or log aggregation.

**Current impact**: Acceptable for paper execution where audit is low-stakes. For venue integration, every order must be traceable from a simple HTTP query.

**Resolution**: Design trace persistence strategy in S75. Options: embed in KV payload, separate audit bucket, or dedicated audit stream.

**Severity**: BLOCKING for venue integration.

### HB-5: No Kill Switch

**What**: Execution can only be halted by removing `paper_order` from `pipeline.execution_families` and restarting the binary. Typical restart time: 5-10 seconds including NATS reconnection.

**Current impact**: Paper execution tolerates slow halt. Venue integration does not — a runaway order loop needs sub-second halt.

**Resolution**: Design kill switch in S75. Implementation: configctl `execution_halted` event → execution actors subscribe and immediately stop.

**Severity**: BLOCKING for venue integration. NOT blocking for paper execution.

---

## Structural Risks

These are not blockers but represent challenges that must be addressed during venue integration design.

### SR-1: Unidirectional Flow Model Extension

**Current model**: configctl → ingest → derive → store → gateway. Strictly unidirectional.

**Venue challenge**: Venue integration introduces bidirectional flow:
- Outbound: Execution intent → venue adapter → exchange API
- Inbound: Exchange API → fill confirmation → execution state update

**Options** (from S68, still valid):
- Option A: Execution remains unidirectional (fire-and-forget). Venue adapter is a separate system that reads execution projections.
- Option B: New binary (`execute`) that subscribes to EXECUTION_EVENTS and places orders, publishing fill events back.
- Option C: Derive publishes to EXECUTION_EVENTS, a venue adapter binary consumes and publishes EXECUTION_FILL_EVENTS, store materializes both.

**Recommendation**: Option C — cleanest separation. Derive owns intent generation. Venue adapter owns placement. Store materializes both. Gateway queries both.

**Impact if unaddressed**: Premature implementation could compromise the clean mesh topology.

### SR-2: Latest-Only Projection Limits

**Current model**: All domains use latest-only KV. Only the most recent execution intent per source/symbol/timeframe is visible.

**Venue challenge**:
- Multiple concurrent orders for the same symbol are invisible
- Previous intents are overwritten (no history)
- Order lifecycle transitions overwrite the KV entry

**Recommendation**: Latest-only is acceptable for the first venue integration step (one active order per symbol at a time). Multi-order and history support are second-slice concerns.

### SR-3: Risk Assessment Staleness

**Current model**: Execution evaluators accept risk assessments of any age. No staleness check.

**Venue challenge**: A risk assessment from 5 minutes ago may reflect market conditions that have dramatically changed. Placing an order based on stale risk data is dangerous.

**Recommendation**: Add configurable staleness threshold at execution evaluator. Default: 2× timeframe (e.g., 120s for 60s timeframe). Reject stale assessments with audit log.

### SR-4: Single Risk Family Gate

**Current model**: Only `position_exposure` risk family exists. Execution depends on a single risk dimension.

**Venue challenge**: Production order placement typically requires multiple independent risk checks (position limits, drawdown, concentration, volatility).

**Recommendation**: Acceptable for first venue integration (paper_order depends on single risk family). Multi-risk aggregation is a future design concern.

---

## Operational Risks

### OR-1: No Operational Validation

**What**: Execution has never been smoke-tested with a real pipeline. All 43 tests are unit/integration level. No evidence of `make smoke-multi` running with live Binance data and execution steps passing.

**Impact**: Unknown consumer lag, unknown end-to-end latency, unknown memory behavior under sustained operation.

**Mitigation**: Run operational smoke test before any venue integration work begins. Must see materialized (non-null) execution data for all symbols.

### OR-2: No Execution-Specific Health Metrics

**What**: Health tracking exists (healthz.Tracker) but provides only event count. No execution-specific metrics: orders/minute, rejection rate, latency, consumer lag.

**Mitigation**: Extend stats to be queryable via HTTP. Consider Prometheus export for operational dashboards.

### OR-3: No Rate Limiting

**What**: No rate limiting infrastructure. Events are processed as fast as possible.

**Impact**: Venue APIs have strict rate limits. Even initial venue integration needs configurable throttling.

**Mitigation**: Rate limiter at venue adapter level, not at execution evaluator level (evaluator should remain pure).

### OR-4: Derive Actor Test Gap

**What**: No unit tests for derive actor message routing or fan-out. Execution evaluator actor tested only at application layer (PaperOrderEvaluator), not at actor layer (message receive → fan-out → publish).

**Impact**: A routing bug could send risk data for btcusdt to ethusdt's execution evaluator. Multi-symbol isolation tests at domain level don't catch actor-level routing errors.

**Mitigation**: Dedicated derive actor test stage before venue integration.

---

## Risk Priority Matrix

| ID | Risk | Severity | Blocks Design? | Blocks Implementation? | Resolution |
|----|------|----------|----------------|----------------------|------------|
| HB-1 | No lifecycle state machine | HIGH | NO | YES | S75 design |
| HB-2 | No fill tracking | HIGH | NO | YES | S75 design |
| HB-3 | Silent data loss | HIGH | NO | YES | S75 design + impl |
| HB-4 | Trace not queryable | HIGH | NO | YES | S75 design |
| HB-5 | No kill switch | HIGH | NO | YES (for live) | S75 design |
| SR-1 | Bidirectional flow | MEDIUM | NO | YES | S75 design |
| SR-2 | Latest-only limits | MEDIUM | NO | NO (first step) | Future |
| SR-3 | Risk staleness | MEDIUM | NO | YES | S75 design + impl |
| SR-4 | Single risk family | LOW | NO | NO | Future |
| OR-1 | No operational validation | HIGH | NO | YES | Pre-impl smoke |
| OR-2 | No execution metrics | MEDIUM | NO | YES | Post-design |
| OR-3 | No rate limiting | MEDIUM | NO | YES (for live) | Venue adapter stage |
| OR-4 | Derive actor test gap | MEDIUM | NO | YES | Pre-impl tests |

---

## Summary

**Total hard blockers**: 5 (HB-1 through HB-5) — all designable in S75, none implementable yet
**Total structural risks**: 4 (SR-1 through SR-4) — designable in S75
**Total operational risks**: 4 (OR-1 through OR-4) — must be resolved before implementation

**Key insight**: None of the blockers prevent design work. All of them prevent implementation. This confirms that S75 should be **design-only**.

**Estimated stages to clear all blockers for first venue step:**
1. S75: Design-only (venue integration architecture)
2. S76: Failure recovery hardening (HB-3) + staleness guard (SR-3)
3. S77: Lifecycle + fill model implementation (HB-1, HB-2)
4. S78: Trace persistence + kill switch implementation (HB-4, HB-5)
5. S79: Derive actor tests + operational validation (OR-1, OR-4)
6. S80: First guarded venue adapter implementation
