# Lifecycle Invariant Coverage and Price Realism Hardening

> S384 — Exhaustive invariant test coverage + G1 price realism closure

## Purpose

This document records the design decisions and implementation details for S384, which closes 41 invariant gaps identified in S383 and implements G1 (price realism via NATS KV lookup) for dry-run and paper execution modes.

## Scope

- **In scope**: exhaustive lifecycle invariant tests across all 8 categories; PriceSource interface; DryRunSubmitter/PaperVenueAdapter price injection; backward compatibility.
- **Out of scope**: OMS redesign, position tracking, portfolio risk, multi-venue, ExecutionIntent schema changes.

## Design Decisions

### 1. Invariant Coverage Strategy

S383 cataloged 49 invariants across 8 categories. Only 8 had test coverage. S384 adds exhaustive domain-level tests that cover the full 7×7 transition matrix (49 pairs), all terminal state properties, fill record invariants, quantity monotonicity, status monotonicity, correlation preservation, and cross-mode consistency.

Tests are placed in `internal/domain/execution/s384_lifecycle_invariants_test.go` as a single cohesive file organized by invariant category (ST, TERM, FR, IFC, QM, SM, SAFE, CORR).

### 2. G1: Price Realism via PriceSource

**Problem**: DryRunSubmitter and PaperVenueAdapter hardcode `Price: "0"` in fill records, making paper and dry-run fills unrealistic for downstream analysis.

**Solution**: A minimal `PriceSource` interface injected via builder pattern:

```go
type PriceSource interface {
    LastPrice(ctx context.Context, source, symbol string, timeframe int) (string, *problem.Problem)
}
```

**Properties**:
- **Best-effort**: callers never fail on price lookup errors. Fallback is `"0"`.
- **Backward-compatible**: `WithPriceSource()` is optional. Existing call sites that don't use it produce identical behavior (Price="0").
- **No model inflation**: PriceSource is a port interface, not a domain model change. ExecutionIntent and FillRecord schemas are unchanged.
- **NATS KV integration path**: production implementation reads `CANDLE_LATEST` KV bucket using key `{source}.{symbol}.{timeframe}` and returns the `Close` field.

### 3. Adapter Changes

Both `DryRunSubmitter` and `PaperVenueAdapter` gain:
- A `priceSource` field (nil by default).
- A `WithPriceSource(ps)` builder method.
- A private `resolvePrice()` method that returns the last price or `"0"` on failure.

The fill record construction now uses `resolvePrice()` instead of hardcoded `"0"`.

### 4. What Was NOT Changed

- **ExecutionIntent struct**: no fields added or removed.
- **FillRecord struct**: no schema changes.
- **State machine**: no new states or transitions.
- **Validation logic**: Validate() unchanged.
- **Existing test files**: no modifications to pre-S384 tests.

## File Map

| File | Change |
|------|--------|
| `internal/application/ports/price.go` | New — PriceSource interface |
| `internal/application/execution/dry_run_submitter.go` | Modified — price injection |
| `internal/application/execution/paper_venue_adapter.go` | Modified — price injection |
| `internal/domain/execution/s384_lifecycle_invariants_test.go` | New — 41 invariant gap tests |
| `internal/application/execution/s384_price_realism_test.go` | New — G1 price realism tests |

## Limitations

1. **PriceSource not wired in production yet**: the interface is defined and tested, but the NATS KV implementation (`CandleKVPriceSource`) and wiring in `cmd/execute/run.go` are deferred to S385. This is intentional — S384 focuses on the contract and test coverage.
2. **Quantity invariants tested at value level, not enforced in domain**: the domain Validate() method does not enforce FilledQuantity <= Quantity or fill-sum consistency. These are tested as behavioral invariants in S384 but enforcement is a S385 candidate.
3. **Final flag not enforced by Validate()**: the test documents that terminal states should have Final=true, but this is a producer-side convention, not a domain validation rule.
