# S21: Projection Model Hardening — Stage Report

## Summary

Hardened the store's projection pipeline to be explicitly safe under replay, restart, and reprocessing. The changes are structural (monotonicity guard, validation gate, observability counters) rather than functional — no new endpoints, no new features. The projection pattern is now documented as a reusable model for future evidence types.

## What Changed

### `CandleKVStore` — Monotonicity guard on latest projection

**Before:** `Put` was a blind overwrite. During replay, a stale candle could regress the latest projection to an older OpenTime.

**After:** `Put` reads the existing entry, compares OpenTime, and skips the write if the existing candle is newer or equal. Returns a `PutResult` (`PutWritten`, `PutSkippedStale`, `PutSkippedDuplicate`) so the caller knows the outcome.

```go
type PutResult int
const (
    PutWritten         PutResult = iota // new or newer candle materialized
    PutSkippedStale                     // existing is newer (stale replay)
    PutSkippedDuplicate                 // same OpenTime already exists
)
```

The guard is a read-then-write, safe under the single-writer invariant. No CAS needed.

### `CandleProjectionActor` — Validation gate + projection stats

**Before:** The actor only checked `Final=true`, then wrote blindly to both buckets.

**After:** Three gates before any KV write:
1. `Final=true` filter (unchanged)
2. `candle.Validate()` — domain validation rejects malformed candles
3. Monotonicity guard result from `Put` — skips stale/duplicate candles

The actor now tracks six counters via `projectionStats`:
- `materialized` — candles written to both buckets
- `skipped_stale` — latest skipped (existing is newer)
- `skipped_dedup` — latest skipped (same OpenTime)
- `skipped_non_final` — non-final candles dropped
- `rejected` — validation failures
- `errors` — KV write errors

Stats are logged at shutdown for operational visibility.

### `PutHistory` — Explicit idempotency documentation

The method already had natural idempotency (same candle → same key → same value). Added explicit documentation so the invariant is visible, not implicit.

### `StoreSupervisor` — Log both buckets on startup

Startup log now shows both `bucket_latest` and `bucket_history` for clarity.

### New unit tests — `candle_kv_store_test.go`

- `PutResult.String()` coverage
- `candleKey` format verification
- `candleHistoryKey` determinism (same OpenTime → same key)
- `candleHistoryKey` uniqueness (different OpenTime → different key)

## Architecture Documents Created

### `docs/architecture/projection-writer-pattern.md`

Defines the canonical projection pipeline: single-writer invariant, write ordering (latest then history), validation rules, observability counters, and the rationale for each design decision.

### `docs/architecture/replay-idempotency-rules.md`

Covers four replay scenarios (normal restart, consumer reset, duplicate delivery, out-of-order delivery) and how each projection target handles them. Formalizes four invariants:

1. History is idempotent by key design
2. Latest is idempotent via monotonicity guard
3. Non-final candles are never materialized
4. Domain validation precedes all writes

Documents known limitations (ack-before-projection window, no cross-bucket atomicity, single-writer assumption) with explicit rationale for why each is acceptable.

## Files Modified

| File | Change |
|------|--------|
| `internal/adapters/nats/candle_kv_store.go` | `PutResult` type, monotonicity guard in `Put`, idempotency docs on `PutHistory` |
| `internal/adapters/nats/candle_kv_store_test.go` | **New** — unit tests for PutResult and key helpers |
| `internal/actors/scopes/store/candle_projection_actor.go` | Validation gate, projection stats, structured logging |
| `internal/actors/scopes/store/store_supervisor.go` | Log both buckets on startup |
| `docs/architecture/projection-writer-pattern.md` | **New** — canonical projection writer reference |
| `docs/architecture/replay-idempotency-rules.md` | **New** — replay/idempotency/dedup rules |
| `docs/stages/stage-s21-projection-model-hardening-report.md` | **New** — this report |

## Invariants Consolidated

| # | Invariant | Enforced by |
|---|-----------|------------|
| 1 | Only `Final=true` candles are materialized | `CandleProjectionActor.onCandle` gate 1 |
| 2 | Every materialized candle passes domain validation | `CandleProjectionActor.onCandle` gate 2 |
| 3 | Latest projection is monotonically forward | `CandleKVStore.Put` monotonicity guard |
| 4 | History is idempotent by key design | Key format `{source}.{symbol}.{tf}.{open_time_unix}` |
| 5 | Single writer per KV bucket | `StoreSupervisor` spawns exactly one projection actor |
| 6 | Write order: latest before history | `CandleProjectionActor.onCandle` sequence |

## Remaining Limitations

1. **Ack-before-projection** — consumer acks before projection writes. On crash between ack and write, 0-1 candles are lost from projection. Acceptable for current use case; documented with mitigation path.
2. **No cross-bucket atomicity** — latest and history written sequentially. A crash between the two writes leaves an inconsistency that resolves on the next event.
3. **Single-writer assumption** — monotonicity guard is read-then-write, not CAS. Safe under the current single-store-instance topology.
4. **No projection lag metric** — we track event counts but not the delta between stream head and projection position. Future stages can add this.
5. **No integration test for monotonicity guard** — requires embedded NATS. Unit tests cover key design; the guard logic is simple (read, compare, skip/write).

## S22 Preparation

1. **Projection lag metric** — track the delta between JetStream stream sequence and last projected sequence. Surface in `/statusz`.
2. **Integration test with embedded NATS** — verify monotonicity guard, history dedup, and replay behavior end-to-end.
3. **Ack-after-projection option** — if candle loss during crash becomes unacceptable, implement synchronous ack by having the projection actor confirm back to the consumer before ack.
4. **Pattern reuse for new evidence types** — the projection-writer-pattern doc is designed to be the template when adding new evidence projections (e.g., order book snapshots, funding rates).
5. **CAS upgrade for multi-writer** — if horizontal scaling of the writer becomes necessary, upgrade `Put` to use NATS KV `Update` with revision-based CAS.
