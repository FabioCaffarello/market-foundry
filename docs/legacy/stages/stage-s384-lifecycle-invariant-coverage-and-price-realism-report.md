# Stage S384 — Lifecycle Invariant Coverage and Price Realism

> **Status**: Complete
> **Wave**: OMS Foundation
> **Predecessor**: S383 (Canonical Order Model and Lifecycle State Machine)
> **Date**: 2026-03-22

## Executive Summary

S384 closes the 41 invariant coverage gaps identified in S383 and implements G1 (price realism via NATS KV lookup interface) for dry-run and paper execution modes. Lifecycle invariant test coverage moves from 16% (8/49) to 100% (49/49) across all 8 categories. The PriceSource interface enables realistic fill prices without inflating the ExecutionIntent model.

## What Was Delivered

### 1. Exhaustive Invariant Coverage (41 gaps → 0)

All 49 invariants across 8 categories now have explicit test evidence:

| Category | Before | After | Tests Added |
|----------|--------|-------|-------------|
| ST — State Transitions | 24% | 100% | 49 pairs (10 valid, 39 invalid) + completeness check |
| TERM — Terminal States | 40% | 100% | Absorbing (21 edges), identification, count, Final flag |
| FR — Fill Records | 11% | 100% | Presence, fields, simulated consistency, timestamps, multi-fill |
| IFC — Intent-Fill Consistency | 0% | 100% | Quantity sum, field preservation (side, symbol, source, timeframe, risk) |
| QM — Quantity Monotonicity | 0% | 100% | Monotonic increase, partial bounds, filled equality |
| SM — Status Monotonicity | 0% | 100% | Tier ordering, no backward, self-loop blocked, terminal→initial blocked |
| SAFE — Safety | 71% | 100% | 10 required fields, invalid side/status/timeframe |
| CORR — Correlation | 50% | 100% | CorrelationID, CausationID, PartitionKey stability, DeduplicationKey uniqueness |

Cross-mode consistency tests confirm dry_run, paper, and venue_live share the identical state machine.

### 2. G1 — Price Realism (Closed)

**Interface**: `ports.PriceSource` — single method `LastPrice(ctx, source, symbol, timeframe) → (price, problem)`.

**Integration**: Both `DryRunSubmitter` and `PaperVenueAdapter` accept optional `PriceSource` via `WithPriceSource()` builder. When provided, fills use the last observed close price. Fallback to `"0"` on nil source, error, or unknown symbol.

**Evidence**: 12 tests covering realistic price, fallback paths, no-action intents, field preservation, and backward compatibility.

**Backward compatibility**: Call sites that don't use `WithPriceSource()` produce identical behavior to pre-S384.

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/application/ports/price.go` | New | PriceSource interface |
| `internal/application/execution/dry_run_submitter.go` | Modified | Price injection via PriceSource |
| `internal/application/execution/paper_venue_adapter.go` | Modified | Price injection via PriceSource |
| `internal/domain/execution/s384_lifecycle_invariants_test.go` | New | 41 invariant gap tests (domain) |
| `internal/application/execution/s384_price_realism_test.go` | New | 12 G1 price realism tests |
| `docs/architecture/lifecycle-invariant-coverage-and-price-realism-hardening.md` | New | Design document |
| `docs/architecture/order-lifecycle-invariant-coverage-matrix-and-price-realism-findings.md` | New | Evidence matrix |

## Test Results

```
ok   internal/domain/execution       — 49 S384 tests PASS
ok   internal/application/execution  — 12 S384 tests PASS
     All pre-existing tests PASS (backward compatible)
     All binaries build clean (cmd/execute, internal/application, internal/domain)
```

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No OMS redesign | Compliant — ExecutionIntent unchanged |
| No ExecutionIntent schema changes | Compliant — no fields added/removed |
| No position tracking | Compliant |
| No portfolio risk | Compliant |
| No multi-venue | Compliant |
| G1 scoped to price only | Compliant — PriceSource is read-only, best-effort |

## Limitations and Honest Gaps

1. **Domain enforcement deferred**: Validate() does not enforce FilledQuantity ≤ Quantity or fill-sum consistency at the domain level. Tests document these as behavioral invariants; domain enforcement is a S385 candidate.
2. **Production wiring deferred**: PriceSource interface is defined and tested but not yet wired into `cmd/execute/run.go`. The NATS KV implementation (`CandleKVPriceSource` reading from `CANDLE_LATEST`) is straightforward and scoped for S385.
3. **Fee realism out of scope**: Fills still use Fee="0" in dry-run/paper modes. Not a lifecycle invariant concern.
4. **Final flag convention, not enforcement**: Terminal states should have Final=true, but this is a producer convention, not a Validate() rule.

## Preparation for S385

S385 should:
1. **Wire PriceSource in production**: implement `CandleKVPriceSource` reading from `CANDLE_LATEST` KV bucket, inject via `WithPriceSource()` in `cmd/execute/run.go`.
2. **Consider domain enforcement**: evaluate adding `FilledQuantity ≤ Quantity` and fill-sum checks to `Validate()` for defense-in-depth.
3. **Continue wave toward OMS**: with lifecycle fully covered and price realism in place, the foundation is ready for order lifecycle event sourcing or state persistence if the wave demands it.

## Conclusion

S384 transforms lifecycle invariant coverage from 16% to 100% and closes G1 with a minimal, backward-compatible design. The ExecutionIntent model and state machine remain unchanged. The wave is strengthened without scope inflation.
