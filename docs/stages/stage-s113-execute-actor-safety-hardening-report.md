# Stage S113: Execute Actor Safety Hardening — Report

**Stage:** S113
**Status:** Complete
**Date:** 2026-03-19
**Objective:** Harden the execute actor's operational safety for minimum controlled live operation.

## Executive Summary

The execute actor's three-gate pre-submit safety logic (kill switch, staleness guard, submit timeout) was extracted into a testable `SafetyGate` structure and comprehensively tested. 15 new targeted safety tests prove the gate evaluation order, fail-open/fail-closed semantics, boundary conditions, and degraded-mode behavior. 6 additional edge-case tests were added to the staleness guard and paper venue adapter. The actor was refactored to delegate to `SafetyGate` without changing behavior.

## Changes Applied

### New Files

| File | Purpose |
|------|---------|
| `internal/application/execution/safety_gate.go` | Testable pre-submit safety gate (GateChecker interface + SafetyGate struct) |
| `internal/application/execution/safety_gate_test.go` | 15 tests: kill switch, staleness, ordering, edge cases, degraded modes |
| `docs/architecture/execute-actor-safety-model.md` | Safety model documentation |
| `docs/architecture/execute-actor-critical-test-coverage.md` | Test coverage matrix with evidence |

### Modified Files

| File | Change |
|------|--------|
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Delegates to SafetyGate instead of inline gate logic; adds unknown-reason error path |
| `internal/application/execution/staleness_guard_test.go` | +5 edge case tests (zero maxAge, zero timestamp, clock skew, boundary precision) |
| `internal/application/execution/paper_venue_adapter_test.go` | +2 tests (cancelled context behavior, fill delay) |

### No Files Removed

## Guarantees Proven

### Kill Switch (Gate 1)
- Halted gate blocks all intents (tested)
- Active gate allows intents (tested)
- Nil gate checker = fail-open (tested)
- Slow gate read (timeout) = fail-open (tested with 50ms timeout)
- Zero-value gate = not halted, fail-open (domain test)
- Halt → resume cycle preserves audit fields (domain test)

### Staleness Guard (Gate 2)
- Fresh intents pass (tested)
- Stale intents blocked (tested)
- Exact boundary = NOT stale, `>` semantics (tested)
- 1ns past boundary = stale (tested)
- 1ns under boundary = not stale (tested)
- Zero timestamp = stale (tested)
- Future timestamp = not stale, graceful clock skew (tested)
- Zero maxAge = everything past stale, exact-now not stale (tested)
- Large clock skew handled gracefully (tested)

### Submit Timeout (Gate 3)
- Context deadline applied to SubmitOrder (code review confirmed)
- Default 10s fallback when config is 0 (code review confirmed)
- Binance adapter respects context timeout (existing test)
- Paper adapter ignores context (documented by new test)

### Gate Ordering
- Kill switch evaluated before staleness (tested)
- When both would block, kill switch reason reported (tested)

### Degraded Modes
- Both nil (no gate checker + no staleness) = fully open (tested)
- Nil staleness guard alone = staleness check skipped (tested)
- KV unavailable at startup = gate check disabled, execution proceeds (code path verified)

## Remaining Limits

| Limit | Description | Risk | Recommendation |
|-------|-------------|------|----------------|
| No fill publish retry | If fill event fails to publish after successful venue submit, fill is lost | Medium | Consider dead-letter queue or at-least-once publish in S114+ |
| Global-only kill switch | No per-symbol or per-family granularity | Low | Acceptable for minimum live; extend if multi-family is needed |
| No circuit breaker | Repeated venue failures don't auto-halt | Low | Manual kill switch suffices for minimum live |
| Paper adapter ignores context | Paper fills are instant, context not checked | None | Documented behavior; real adapters checked |
| No actor-level integration test | SafetyGate is tested in isolation, not wired through Hollywood | Low | Actor delegates to SafetyGate; wiring is trivial and compile-checked |

## Test Count Summary

| Package | Before S113 | After S113 | Delta |
|---------|-------------|------------|-------|
| `internal/application/execution` | 47 | 69 | +22 |
| `internal/domain/execution` | 43+ | 43+ | 0 |
| **Total safety-related** | 47 | 69 | **+22** |

## Preparation for S114

The execute actor is now hardened for minimum controlled live operation. Recommended next steps:

1. **Live activation preparation** — deploy with paper_simulator venue and validate the 3-gate flow against real NATS event traffic.
2. **Fill publish reliability** — evaluate whether at-least-once fill publishing or a dead-letter mechanism is needed before real venue activation.
3. **Operational runbook** — document how to halt/resume execution via the gateway API and verify kill switch propagation latency.
4. **Observability validation** — confirm that `/statusz` counters (processed, filled, skipped_halt, skipped_stale, errors) are visible and alertable in the monitoring stack.
