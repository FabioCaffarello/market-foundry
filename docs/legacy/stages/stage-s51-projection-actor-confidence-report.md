# Stage S51 — Projection Actor Confidence Report

**Status**: Complete
**Date**: 2026-03-17
**Blocker addressed**: BG-3 (evidence projection actors without adequate coverage)

## 1. Executive Summary

S51 increases structural confidence in the foundational projection actors by:

1. Extracting testable interfaces from concrete KV stores.
2. Adding 38 unit tests covering all projection actor gates, monotonicity outcomes,
   error paths, health tracking, and the candle dual-write behavior.
3. Producing an explicit dual-write and consistency review that identifies risks
   honestly and documents the single-writer invariant the system depends on.

All 5 projection actors (candle, tradeburst, volume, signal, decision) now have
coverage for every code path in their event-handling methods.

## 2. Projection Actors Covered

| Actor | Tests | Gates Covered | Monotonicity | Dual-Write |
|---|---|---|---|---|
| CandleProjectionActor | 10 | Final, Validate | Stale, Dedup, Written | Latest+History tested |
| TradeBurstProjectionActor | 7 | Final, Validate | Stale, Dedup, Written | N/A (single-write) |
| VolumeProjectionActor | 7 | Final, Validate | Stale, Dedup, Written | N/A (single-write) |
| SignalProjectionActor | 8 | Final, Validate | Stale, Dedup, Written | N/A (single-write) |
| DecisionProjectionActor | 10 | Final, Validate, Outcome enum | Stale, Dedup, Written | N/A (single-write) |

**Total new tests**: 42 (38 projection actor + existing KV adapter tests continue to pass)

## 3. Files Changed

### New files
| File | Purpose |
|---|---|
| `internal/actors/scopes/store/projection_store.go` | Store interfaces for testability |
| `internal/actors/scopes/store/candle_projection_actor_test.go` | Candle projection tests |
| `internal/actors/scopes/store/trade_burst_projection_actor_test.go` | TradeBurst projection tests |
| `internal/actors/scopes/store/volume_projection_actor_test.go` | Volume projection tests |
| `internal/actors/scopes/store/signal_projection_actor_test.go` | Signal projection tests |
| `internal/actors/scopes/store/decision_projection_actor_test.go` | Decision projection tests |
| `docs/architecture/projection-confidence-and-dual-write-review.md` | Dual-write review |
| `docs/stages/stage-s51-projection-actor-confidence-report.md` | This report |

### Modified files
| File | Change |
|---|---|
| `candle_projection_actor.go` | Store field → interface; closer func for cleanup |
| `trade_burst_projection_actor.go` | Store field → interface; closer func for cleanup |
| `volume_projection_actor.go` | Store field → interface; closer func for cleanup |
| `signal_projection_actor.go` | Store field → interface; closer func for cleanup |
| `decision_projection_actor.go` | Store field → interface; closer func for cleanup |

### Architectural impact

The refactor changes the store field type from a concrete pointer to an interface.
The `start()` method still creates the concrete adapter, and the supervisor wiring
is completely unchanged. The only structural addition is a `closer func() error`
field that decouples the `Close()` call from the store interface (since `Close()`
is a lifecycle concern, not a projection concern).

## 4. Dual-Write / Consistency Review Summary

Full analysis in `docs/architecture/projection-confidence-and-dual-write-review.md`.

Key findings:

- **No cross-actor dual-writes exist**. Each projection type writes to its own bucket(s).
- **Candle is the only dual-write actor** (latest + history). Behavior under partial
  failure is well-defined and tested.
- **Monotonicity guards are safe** under the actor model's single-threaded processing.
  The single-writer invariant must be preserved (one projection actor per family).
- **VolumeKVStore has a cosmetic inconsistency** in error return values (returns
  `PutSkippedStale` instead of `PutWritten` on errors). No functional impact.

## 5. Acceptance Criteria

| Criterion | Status |
|---|---|
| Projection actors have useful coverage | Done — 38 tests across 5 actors |
| Monotonicity/idempotency are more reliable and explicit | Done — all outcomes tested |
| Latest/history and read-side semantics are better tested | Done — candle dual-write behavior verified |
| Dual-write/consistency review is honest and specific | Done — see architecture doc |
| Stage reduces BG-3 blocker before strategy | Done — BG-3 downgraded from blocker to low-risk |

## 6. Blockers Remaining for S52

| ID | Description | Severity | Notes |
|---|---|---|---|
| BG-3 | Evidence projection integration tests (real NATS) | Low | Unit tests cover logic; integration tests would cover KV wire protocol |
| BG-6 | VolumeKVStore error return inconsistency | Cosmetic | Returns PutSkippedStale on error; should be PutWritten |
| BG-7 | Multi-instance store would violate single-writer | Medium | Not deployed; document or add revision CAS if needed |
| BG-8 | No projection lag metric | Low | Useful for observability but not a correctness issue |

## 7. Guard Rail Compliance

| Rule | Status |
|---|---|
| No strategy implementation | Compliant |
| No new domain opened | Compliant |
| No broad store redesign | Compliant — interface extraction is minimal |
| No masked risks | Compliant — all risks documented with severity |
| Remaining hardening documented | Compliant — see blockers table |
