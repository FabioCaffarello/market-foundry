# Stage S76 — Failure Recovery Hardening Report

**Status:** Complete
**Objective:** Harden failure semantics and recovery behavior in the execution domain's critical runtime paths.

## Executive Summary

S76 applies targeted failure/recovery hardening to the execution domain without inflating scope or introducing generic frameworks. The changes make failure behavior explicit and observable across the publish, consumer, and projection paths. The system moves from optimistic/silent failure handling to predictable, diagnosable behavior under transient failures.

## Changes Made

### 1. Health Tracker Error Visibility (`internal/shared/healthz/healthz.go`)

**Problem:** Tracker only recorded successful events. Failures left the tracker idle, making `/statusz` misleading — a component could be actively failing but appear idle.

**Change:**
- Added `RecordError()` method: updates `lastEventAt` (keeps tracker alive) while incrementing a separate `errorCount`
- Added `ErrorCount()` accessor
- Added `error_count` field to `/statusz` JSON response
- Heartbeat loop now considers error-only trackers as active and includes `error_count` in idle warnings

**Impact:** Operators can now distinguish "idle" from "actively failing" on the health surface.

### 2. Execution Tracker Initialization (`cmd/store/run.go`)

**Problem:** Execution pipeline trackers were never initialized in `run.go`, unlike risk, strategy, decision, and signal pipelines. When execution families were enabled, the store supervisor would pass nil trackers, causing missing health visibility.

**Change:** Added execution tracker initialization block following the same pattern as risk/strategy/decision/signal.

**Impact:** Execution pipeline now has full health visibility on `/statusz` when enabled.

### 3. Publisher Retry with Backoff (`internal/actors/scopes/derive/execution_publisher_actor.go`)

**Problem:** Single-attempt publish with `defer cancel()` (diverged from all other publishers that use immediate `cancel()`). No retry for transient failures. No publish/error counters.

**Changes:**
- Fixed context cancellation: immediate `cancel()` after publish (aligned with risk/strategy/decision publishers)
- Added `publishWithRetry`: single retry with 500ms backoff for `Unavailable` errors only
- Non-retryable errors (`InvalidArgument`, `Internal`) fail immediately without retry
- Added `published` and `errors` atomic counters, logged on actor stop
- `tracker.RecordError()` called on publish failure
- `tracker.RecordEvent()` called on publish success

**Impact:** Transient NATS failures get one retry opportunity. All failures are tracked and visible.

### 4. Projection Error Tracking (`internal/actors/scopes/store/execution_projection_actor.go`)

**Problem:** KV put failures were logged but not recorded on the health tracker, making projection failures invisible on `/statusz`.

**Changes:**
- Added `tracker.RecordError()` call on KV put failure
- Enriched error log with `code`, `side`, `status`, `correlation_id` for better diagnostics

**Impact:** Projection failures now visible on health surface. Richer error context aids debugging.

### 5. Consumer Failure Visibility (`internal/adapters/nats/execution_consumer.go`)

**Problem:** No visibility into delivery exhaustion (when NATS gives up redelivering), no delivery statistics.

**Changes:**
- Added `delivered`, `redelivered`, `terminated`, `nakked` atomic counters with `Stats()` accessor
- Added max-delivery exhaustion detection: logs ERROR when `NumDelivered >= MaxDeliver`
- Added `max_deliver` to redelivery warning log
- Added `code` to decode error log
- Terminal disposition logged at WARN with reason for post-incident analysis

**Impact:** Operators can detect when messages are being permanently dropped by NATS and identify decode error patterns.

### 6. Strategy Projection Consistency Fix (`internal/actors/scopes/store/strategy_projection_actor.go`)

**Problem:** Strategy projection was the only projection missing `checkStatsInvariant()` on stop, while risk and execution had it.

**Change:** Added `checkStatsInvariant()` call on `actor.Stopped` and the method implementation (identical pattern to risk/execution).

**Impact:** Strategy projection now detects accounting bugs on shutdown, consistent with all other projections.

## Architecture Documents Created

1. **`docs/architecture/execution-failure-recovery-model.md`** — Classifies all failure modes as recoverable, non-recoverable, or out-of-scope. Documents retry discipline, known gaps, and health observability.

2. **`docs/architecture/execution-projection-failure-semantics.md`** — Defines precise behavior at each projection gate, KV store error contracts, consumer-projection coupling semantics, and operational diagnostics.

## Files Changed

| File | Change |
|------|--------|
| `internal/shared/healthz/healthz.go` | `RecordError()`, `ErrorCount()`, `error_count` in statusz/heartbeat |
| `cmd/store/run.go` | Execution tracker initialization |
| `internal/actors/scopes/derive/execution_publisher_actor.go` | Retry with backoff, error tracking, context fix, publish stats |
| `internal/actors/scopes/store/execution_projection_actor.go` | Error tracking via `RecordError()`, enriched error context |
| `internal/adapters/nats/execution_consumer.go` | Delivery stats, max-delivery exhaustion warning, terminal logging |
| `internal/actors/scopes/store/strategy_projection_actor.go` | Added `checkStatsInvariant()` |
| `docs/architecture/execution-failure-recovery-model.md` | New |
| `docs/architecture/execution-projection-failure-semantics.md` | New |
| `docs/stages/stage-s76-failure-recovery-hardening-report.md` | New |

## Limitations Remaining

1. **No consumer-projection backpressure.** Consumer acks before projection completes KV write. Projection KV failures cause silent event loss (mitigated by error counters and latest-only semantics).

2. **No dead-letter queue for projection failures.** Events that fail KV put are counted but not stored for later retry.

3. **No application-level retry on KV put.** Retrying inside the projection actor would block the mailbox. Infrastructure-level KV failures are expected to be rare.

4. **No circuit breaker on publisher.** The publisher retries once per event. Sustained NATS unavailability will cause per-event retry overhead. A circuit breaker could be added if this becomes a problem.

5. **PutResult return value on error paths.** All KV stores return `PutWritten` when `prob != nil`. Callers correctly check `prob` first, so this is cosmetic. Changing it would touch all 8 KV stores for no behavioral benefit.

## Guard Rail Compliance

- No venue integration opened
- No OMS created
- No generic retry framework introduced — retry is inline, scoped to publisher only
- No failures hidden behind vague logs — all errors include structured context
- Limitations documented explicitly

## Preparation for S77

The hardening in S76 provides the foundation for the next steps:

1. **Action boundary readiness** is now more viable — failure semantics are explicit and observable.
2. **Venue integration** can be pursued knowing that the execution domain has defined recovery behavior for its internal paths.
3. **Consumer-projection backpressure** should be considered if execution volume increases or if event loss becomes unacceptable.
4. **Circuit breaker** on publisher path if sustained NATS unavailability is observed in production.
5. **Execution lifecycle** (cancel, amend, fill) can build on the existing retry/error tracking patterns.
