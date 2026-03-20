# Stage S153 — Failure Handling and Overflow Hardening Report

Status: complete
Date: 2026-03-19

## 1. Executive Summary

S153 aligns the writer service implementation with its documented failure model. Before this stage, the writer had three critical divergences from its architecture: single-attempt INSERT (docs promised 5 retries with backoff), buffer clearing on INSERT failure (docs promised retention for retry), and silent buffer overflow (only WARN log, no distinct counter). All three are now resolved. The writer's failure handling is explicit, retry-capable, and operationally observable.

## 2. Problems Addressed

### 2.1 INSERT Single-Attempt (BLK-05)

**Before:** `flush()` made a single `InsertBatch` call. On failure, the batch was dropped with an ERROR log. No retry. The architecture document (`writer-service-failure-and-delivery-semantics.md`) specified "5 consecutive attempts with exponential backoff (1s-30s)."

**After:** `flush()` retries with configurable exponential backoff (default: 5 attempts, 1s initial, 30s cap). Buffer is retained during retries. Batch is only dropped after all retries are exhausted.

### 2.2 Buffer Clearing on INSERT Failure (BLK-06)

**Before:** `flush()` moved rows out of the buffer (`rows := a.buffer; a.buffer = make(...)`) before calling `InsertBatch`. On failure, rows were already removed from the buffer and permanently lost. This violated the at-least-once promise.

**After:** The buffer is only cleared after successful INSERT or after all retries are exhausted. During the retry window, rows remain in the buffer.

### 2.3 Silent Buffer Overflow

**Before:** `enforceMaxPending()` logged at WARN level. No distinct counter separated overflow from INSERT failure drops. Operators had no way to distinguish "ClickHouse was down" from "ingestion rate exceeded buffer capacity."

**After:** Overflow is logged at ERROR level. A new `events_overflowed` counter tracks overflow-specific data loss. `flush_failures` tracks INSERT-failure-specific drops. Both roll up to `events_dropped` for aggregate visibility.

### 2.4 Silent Mapper Fallbacks (BLK-07)

**Before:** `parseFloat("")` returned 0.0 silently. `marshalJSON(v)` returned `"{}"` on error silently. Zero-value injection was invisible to operators.

**After:** Both functions log at WARN level when a fallback value is used, with input/error context. This makes data quality issues visible without blocking ingestion.

## 3. Files Changed

### Code

| File | Change |
|------|--------|
| `cmd/writer/inserter.go` | Retry with exponential backoff in `flush()`; buffer retention; overflow at ERROR; `events_overflowed` and `flush_failures` counters; new config fields `maxRetries`, `initialBackoff` |
| `cmd/writer/inserter_test.go` | 4 new tests: retry on transient failure, drop after exhaustion, success on first attempt, buffer retained during retries; updated overflow test for `events_overflowed` counter |
| `cmd/writer/mappers.go` | `parseFloat()` and `marshalJSON()` now log WARN on fallback |
| `cmd/writer/supervisor.go` | Wires `maxRetries` and `initialBackoff` from config; logs them at startup |
| `internal/shared/settings/schema.go` | Added `MaxRetries` and `InitialBackoff` to `ClickHouseConfig`; `MaxRetriesOrDefault()` (5), `InitialBackoffOrDefault()` (1s); validation |

### Config

| File | Change |
|------|--------|
| `deploy/configs/writer.jsonc` | Added `max_retries: 5` and `initial_backoff: "1s"` |

### Documentation

| File | Content |
|------|---------|
| `docs/architecture/analytical-failure-handling-and-overflow-model.md` | Canonical failure model: categories, guarantees, counters |
| `docs/architecture/analytical-retry-backoff-and-loss-semantics.md` | Retry/backoff mechanics, loss semantics, tuning guidelines |

## 4. Resulting Semantics

### INSERT path

```
Event → Consumer → Inserter buffer → flush()
                                       ├─ Attempt 1 → success → clear buffer, count flushed
                                       ├─ Attempt 1 → fail → wait 1s
                                       ├─ Attempt 2 → fail → wait 2s
                                       ├─ Attempt 3 → fail → wait 4s
                                       ├─ Attempt 4 → fail → wait 8s
                                       └─ Attempt 5 → fail → drop batch, ERROR log, count dropped
```

### Overflow path

```
insertRowMsg → buffer append → enforceMaxPending()
                                ├─ under limit → no-op
                                └─ over limit → evict oldest (FIFO), ERROR log, count overflowed
```

### Counter semantics

| Counter | Cause |
|---------|-------|
| `events_flushed` | Successful INSERT |
| `events_dropped` | Any permanent loss (overflow + retry exhaustion) |
| `events_overflowed` | Buffer overflow eviction only |
| `flush_failures` | INSERT retry exhaustion only |

Invariant: `events_overflowed + (flush_failures * batch_size) ≈ events_dropped`
(approximate because batch sizes vary).

## 5. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| INSERT single-attempt no longer contradicts architecture | Done — 5 retries with backoff |
| Buffer overflow no longer silent | Done — ERROR log + `events_overflowed` counter |
| Failure semantics more explicit | Done — separate counters, structured logs |
| No exaggerated system complexity | Done — retry is inline, ~30 LOC added to flush |
| Base ready for recovery/supervision | Done — failure model is clean for S154 pipeline recovery |

## 6. Remaining Limits

1. **No dead-letter queue.** Batches that exhaust retries are permanently lost from the projection.
2. **No automatic replay.** Events remain in NATS JetStream but there is no re-consume mechanism.
3. **No pipeline-level recovery.** A poisoned consumer-inserter pair stays dead until process restart. Deferred to S154.
4. **No deduplication.** Retry may cause duplicate inserts if ClickHouse accepted but response was lost.
5. **Retry blocks the actor.** During backoff sleep, the inserter does not process new messages. Acceptable for batch I/O but limits throughput during extended outages.
6. **Mapper fallbacks degrade data silently at the row level.** WARN log is emitted but the degraded row is still inserted. No mechanism to reject bad rows.

## 7. S154 Preparation Recommendations

With failure handling now aligned, the next priorities are:

1. **Pipeline recovery (BLK-08).** Supervisor should restart failed consumer-inserter pairs with backoff instead of letting the entire writer die.
2. **Periodic operational summary.** Emit a structured log every N minutes with aggregate counters per family (`events_flushed`, `events_dropped`, `events_overflowed`, `flush_failures`).
3. **Config reference update.** `deploy/configs/CONFIG-REFERENCE.md` should document `max_retries` and `initial_backoff`.
4. **Mapper validation tightening.** Consider rejecting rows with critical fields at zero (e.g., candle with 0 open/high/low/close) instead of inserting degraded data.

## 8. Test Coverage

| Test | Covers |
|------|--------|
| `TestFlush_RetriesOnTransientFailure` | Retry succeeds after N failures |
| `TestFlush_DropsAfterRetriesExhausted` | Batch dropped, counters correct |
| `TestFlush_SucceedsOnFirstAttempt` | Happy path, single call |
| `TestFlush_BufferRetainedDuringRetries` | Buffer not cleared during retry |
| `TestEnforceMaxPending_TrackerCountsDrops` | `events_dropped` + `events_overflowed` |
| Existing mapper tests | WARN logs now emitted on fallback |

Total writer tests: 34 (up from 30 pre-S153).
