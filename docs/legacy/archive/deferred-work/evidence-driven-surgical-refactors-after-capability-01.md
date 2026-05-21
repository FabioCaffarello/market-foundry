# Evidence-Driven Surgical Refactors After Capability 01

> Stage S123 — Targeted refactors anchored in S122 friction evidence.
> Date: 2026-03-19

---

## Purpose

This document records the surgical refactors executed in S123, each justified by specific friction evidence from S122 (capability-driven friction capture). No refactor was executed for aesthetic or speculative reasons.

---

## Refactors Executed

### R1: Per-Symbol Tracker Counters (CF-01)

**Friction:** `/statusz` tracker counters are aggregate-only. Under multi-symbol operation, the operator cannot answer "is ethusdt flowing?" without cross-referencing logs.

**Evidence:** CF-01 (P1), confirmed in S121 Phase 8. With 2 symbols, a single `event_count: 847` provides no per-symbol breakdown.

**Change:** Added per-symbol counter keys to all actors that use `healthz.Tracker`:

| Runtime | Actor | Counter Key Pattern |
|---------|-------|-------------------|
| store | CandleProjectionActor | `materialized:SYMBOL` |
| store | SignalProjectionActor | `materialized:SYMBOL` |
| store | DecisionProjectionActor | `materialized:SYMBOL` |
| store | StrategyProjectionActor | `materialized:SYMBOL` |
| store | RiskProjectionActor | `materialized:SYMBOL` |
| store | TradeBurstProjectionActor | `materialized:SYMBOL` |
| store | VolumeProjectionActor | `materialized:SYMBOL` |
| derive | EvidencePublisherActor | `published:SYMBOL` (candle, trade_burst, volume) |
| derive | SignalPublisherActor | `published:SYMBOL` |
| derive | DecisionPublisherActor | `published:SYMBOL` |
| derive | StrategyPublisherActor | `published:SYMBOL` |
| derive | RiskPublisherActor | `published:SYMBOL` |
| ingest | PublisherActor | `published:SYMBOL` |
| execute | VenueAdapterActor | `processed:SYMBOL`, `filled:SYMBOL` |

**Mechanism:** Uses the existing `tracker.Counter(name)` API — no infrastructure change. Each actor adds one `Counter("key:"+symbol).Add(1)` call alongside the existing `RecordEvent()` call. The aggregate `event_count` is preserved; per-symbol counters appear in the `counters` map of `/statusz` and `/diagz` responses.

**Example `/statusz` output after fix:**
```json
{
  "name": "store-candle",
  "event_count": 847,
  "counters": {
    "materialized:btcusdt": 423,
    "materialized:ethusdt": 424
  }
}
```

**Diagnostic payoff:** The operator can now answer "is ethusdt flowing?" from `/statusz` alone, without log correlation. This compounds with each additional symbol.

**Files changed:** 13 actor files (1 line added per tracker call site).

---

### R2: Automated Error-Level Log Scanning (CF-04)

**Friction:** The activation script validates health, readiness, diagnostics, and tracker activity, but does not scan for `level=error` log entries. Domain-level errors go undetected.

**Evidence:** CF-04 (P2), confirmed in S121. The script may report "all healthy" while error entries accumulate in container logs.

**Change:** Added a `grep -c '"level":"error"'` check to Phase 8 of `live-pipeline-activate.sh`. If any error-level entries are found, the script records a failure and directs the operator to inspect.

**Files changed:** `scripts/live-pipeline-activate.sh` (5 lines added).

---

### R3: Automated Memory Usage Snapshot (CF-05)

**Friction:** Memory usage under doubled multi-symbol load is not tracked automatically. A goroutine leak or buffer accumulation would go undetected.

**Evidence:** CF-05 (P2), confirmed in S121. The 30-minute soak procedure documents manual `docker stats` checks but does not automate them.

**Change:** Added a `docker stats --no-stream` snapshot to Phase 8 of `live-pipeline-activate.sh`. Prints per-container memory usage to the validation output.

**Files changed:** `scripts/live-pipeline-activate.sh` (4 lines added, alongside R2).

---

## Design Produced (No Code Change)

### D1: Correlation ID Injection Pattern (CF-03)

**Friction:** Correlation ID propagation is manual — each actor must copy the ID from incoming messages to outgoing events. If a new actor omits this, the correlation chain breaks silently.

**Evidence:** CF-03 (P1), confirmed across all actors in S121. Current actors are consistent, but the fragility scales with actor count.

**Design decision:** The recommended pattern is **envelope middleware** — a publish wrapper that automatically injects correlation ID from the incoming message context into outgoing events. This eliminates the manual copy requirement for new actors.

**Implementation deferred to:** Next actor addition (CC-02 or equivalent). The design is ready; implementing now would be premature since no new actors are being added in this stage.

**Pattern sketch:**
```go
// PublishMiddleware wraps a publisher function, automatically injecting
// correlation_id from the originating event's metadata.
func WithCorrelation(origin events.Metadata, fn PublishFunc) PublishFunc {
    return func(ctx context.Context, event any) *problem.Problem {
        // Inject correlation_id from origin into event metadata.
        setCorrelation(event, origin.CorrelationID)
        return fn(ctx, event)
    }
}
```

The exact API shape will be finalized when the first consumer (a new actor) validates the pattern in practice.

---

## Validation

| Check | Result |
|-------|--------|
| All Go packages compile | Pass |
| All existing tests pass | Pass (`internal/actors/...`, `internal/shared/...`) |
| No new abstractions introduced | Compliant — uses existing `tracker.Counter()` API |
| No horizontal refactoring opened | Compliant — changes are localized to tracker call sites |
| Per-symbol counters verified in `/statusz` contract | Structural — counters appear in existing `Counters()` map |

---

## Summary

3 refactors executed, 1 design produced. Total code delta: ~25 lines across 14 files. No new packages, no new abstractions, no behavioral changes. The architecture's diagnostic surface is measurably improved for multi-symbol operation.
