# Venue Integration Activation Gate

> Formal activation gate for venue-integrated execution.
> Date: 2026-03-18 | Stage: S75
> Classification: DESIGN-ONLY — activation is deferred to S80.

---

## 1. Purpose

This document defines the activation gate that must be passed before any venue-integrated execution code enters the `main` branch. It consolidates all prerequisites from S68, S74, and S75 into a single verification checklist.

---

## 2. Gate Structure

The activation gate has three tiers. Each tier must be fully satisfied before proceeding to the next.

### Tier 1: Foundation (S76–S78)

These resolve the 5 hard blockers from S74.

| ID | Gate | Resolves | Verification |
|----|------|----------|-------------|
| G-1 | Publish retry with backoff | HB-3 | Unit test: transient failure → retry → success |
| G-2 | Projection NAK on write failure | HB-3 | Unit test: KV write error → NAK (not ACK) |
| G-3 | Lifecycle state machine | HB-1 | Unit test: all valid transitions pass, all invalid transitions rejected |
| G-4 | Fill tracking model | HB-2 | Unit test: fill validation, fill-to-intent linking |
| G-5 | Trace persistence in KV | HB-4 | Unit test: correlation_id + causation_id in KV payload |
| G-6 | Kill switch mechanism | HB-5 | Integration test: set halted → actors stop within 1 cycle |
| G-7 | Staleness guard | SR-3 | Unit test: stale assessment → rejected with logged reason |

### Tier 2: Verification (S79)

These resolve the operational gaps.

| ID | Gate | Resolves | Verification |
|----|------|----------|-------------|
| G-8 | Derive actor routing tests | A-2 (S68) | Unit test: risk → correct execution evaluator, no cross-symbol bleed |
| G-9 | Automated trace verification | B-1 (S68) | Integration test: synthetic trade → full chain correlation_id intact |
| G-10 | Operational smoke test | OR-1 | `make smoke-multi` passes with execution steps showing materialized data |

### Tier 3: Activation (S80)

| ID | Gate | Verification |
|----|------|-------------|
| G-11 | Execute binary compiles and starts | `go build ./cmd/execute` succeeds |
| G-12 | PaperVenueAdapter passes acceptance tests | Unit + integration tests for simulated venue |
| G-13 | Fill projection materializes correctly | Unit test: fill event → KV entry with correct fields |
| G-14 | Status change events propagate | Integration test: submitted → sent → accepted → filled |
| G-15 | Kill switch halts execute binary | Integration test: configctl halt → execute stops consuming |
| G-16 | Config symmetry enforced | raccoon-cli: derive ↔ store ↔ execute config alignment |
| G-17 | All drift rules pass | raccoon-cli: ED-1 through ED-9 all green |

---

## 3. Gate Enforcement

### Who Runs the Gate

The S80 readiness review (a dedicated stage) evaluates all 17 gates. Only if ALL gates pass does S80 proceed to implementation.

### What Happens on Failure

- Any Tier 1 failure: S80 is blocked. Resolve the failing gate in its designated stage.
- Any Tier 2 failure: S80 is blocked. Resolve in S79 or prior.
- Any Tier 3 failure: S80 is the implementation stage — fix and re-verify within S80.

### No Gate May Be Waived

The action boundary is the highest-stakes transition in Market Foundry. No gate may be skipped, deferred, or marked "acceptable risk." The cost of premature implementation far exceeds the cost of disciplined gating.

---

## 4. Activation Ceremony

When all 17 gates pass, venue-integrated execution is activated through this ceremony:

```
Step 1: Operator adds "venue_market_order" to derive.jsonc pipeline.execution_families
Step 2: Operator adds "venue_market_order" to store.jsonc pipeline.execution_families
Step 3: Operator creates execute.jsonc with venue configuration
Step 4: Operator runs raccoon-cli — all drift checks must pass
Step 5: Operator deploys derive, store, and execute binaries
Step 6: Operator verifies kill switch: configctl execution halt → configctl execution resume
Step 7: Operator monitors for 1 hour — no errors, no dead letters, fills materializing
Step 8: Activation complete — logged with timestamp
```

### Kill Switch Verification Before First Real Order

Before the first real venue adapter is deployed (replacing PaperVenueAdapter):

```
Step A: Deploy with PaperVenueAdapter — verify full flow works
Step B: Activate kill switch — verify all execution stops
Step C: Resume — verify execution resumes from last ACK
Step D: Only then: deploy with real venue adapter
```

---

## 5. Rollback Plan

If venue-integrated execution causes issues after activation:

```
Immediate (< 1 second): configctl execution halt
  → All execution actors stop
  → No new orders placed
  → In-flight orders complete

Short-term (minutes): Remove "venue_market_order" from configs
  → Restart derive + store + execute
  → paper_order continues unaffected

Full rollback (if needed): Stop execute binary
  → Remove execute.jsonc
  → Remove venue_market_order from all configs
  → System reverts to paper-only execution
```

---

## 6. Monitoring Requirements (S80)

Before venue activation, these monitoring capabilities must exist:

| Metric | Source | Alert Threshold |
|--------|--------|----------------|
| Orders placed / minute | execute stats | > 10 (anomaly for first step) |
| Fill rate | execute stats | < 50% (venue may be rejecting) |
| Dead letters | store stats | > 0 (any dead letter is incident) |
| Kill switch state | EXECUTION_CONTROL KV | Change = alert |
| Publish retry rate | derive/execute stats | > 10% (NATS health issue) |
| Consumer lag | JetStream | > 100 messages (processing bottleneck) |
| Staleness rejections | derive stats | > 20% (data feed latency issue) |

---

## 7. References

- [venue-integrated-execution-design.md](venue-integrated-execution-design.md) — S75 master design
- [venue-execution-family-01-contracts.md](venue-execution-family-01-contracts.md) — Family contracts
- [action-boundary-readiness-review.md](action-boundary-readiness-review.md) — S74 readiness review
- [venue-integration-entry-prerequisites.md](venue-integration-entry-prerequisites.md) — S74 prerequisites
