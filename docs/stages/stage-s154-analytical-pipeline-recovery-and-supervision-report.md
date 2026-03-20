# Stage S154 — Analytical Pipeline Recovery and Supervision Report

## Objective

Harden recovery and supervision for analytical pipelines so that a localized consumer failure no longer permanently kills the family until manual process restart.

## Executive Summary

S154 introduces supervisor-managed pipeline recovery to the writer service. When a consumer fails to start (e.g., NATS unavailable), the supervisor now retries with exponential backoff (2s→30s, up to 5 attempts) before marking the family as degraded. Other families continue operating normally throughout. The lifecycle state is observable via health tracker counters and the `/statusz` endpoint, which now surfaces a `"degraded"` phase.

## Recovery and Supervision Applied

### Before S154

- Consumer startup failure → `Poison(self)` → family permanently dead.
- Supervisor unaware of child failures.
- No lifecycle state tracking.
- No restart mechanism.
- `/statusz` phase could not distinguish between "idle" and "failed."

### After S154

- Consumer startup failure → `pipelineFailedMsg` sent to supervisor → supervisor manages restart.
- Supervisor tracks per-family lifecycle state: active → restarting → degraded.
- Exponential backoff: 2s, 4s, 8s, 16s, 30s (5 attempts max).
- Degraded state is terminal per process lifetime (restart resets).
- `/statusz` phase emits `"degraded"` when any tracker has `pipeline_degraded > 0`.
- Per-family counters: `pipeline_restarts`, `pipeline_degraded`.

### Recovery Flow

```
Consumer startup fails
  → consumer sends pipelineFailedMsg to supervisor
  → supervisor increments restart counter
  → supervisor poisons failed consumer + inserter
  → supervisor schedules restartPipelineMsg after backoff
  → supervisor spawns fresh consumer-inserter pair
  → if consumer starts successfully: family is active again
  → if consumer fails again: repeat (up to 5 times)
  → after 5 failures: family marked degraded, no more retries
```

## Files Changed

| File | Change |
|------|--------|
| `cmd/writer/supervisor.go` | Rewritten: lifecycle state tracking, `pipelineFailedMsg`/`restartPipelineMsg` handlers, exponential backoff, degraded state management, `spawnPipeline` / `poisonPipeline` / `calcBackoff` methods |
| `cmd/writer/consumer.go` | Added `supervisorPID` to config; on startup failure, sends `pipelineFailedMsg` to supervisor instead of self-poisoning |
| `cmd/writer/supervisor_test.go` | New: tests for `calcBackoff` (6 cases) and lifecycle state transitions |
| `internal/shared/healthz/healthz.go` | `computePhase` extended with `"degraded"` phase derived from `pipeline_degraded` counter |
| `internal/shared/healthz/healthz_test.go` | New: `TestHealthServer_Statusz_Phase_Degraded` |
| `docs/architecture/analytical-pipeline-recovery-and-supervision.md` | New: recovery model, backoff schedule, observability, design constraints |
| `docs/architecture/analytical-pipeline-lifecycle-degraded-dead-recovered.md` | New: lifecycle states (active/restarting/degraded), state diagram, operational semantics |

## Lifecycle Semantics

| State | Meaning | Entry | Exit |
|-------|---------|-------|------|
| **Active** | Consumer + inserter running, events flowing | Startup success or restart success | Consumer failure |
| **Restarting** | Supervisor scheduling restart after backoff | Consumer failure (budget remaining) | Restart succeeds → Active; Restart fails → Restarting or Degraded |
| **Degraded** | Restart budget exhausted, family stopped | 5+ failures | None (terminal per process lifetime) |

## Remaining Limits

1. **Restart budget is fixed per process lifetime.** No cooling-period reset. Process restart is the recovery path from degraded.
2. **Inserter failures are not supervisor-managed.** The inserter handles failures internally (retry + drop). If the inserter actor panics, Hollywood's default restart applies (3 attempts). Supervisor-level inserter recovery is deferred.
3. **Unexpected actor death (panics) is not supervisor-detected.** The supervisor reacts to `pipelineFailedMsg` from the consumer. If an actor dies from a panic and exhausts Hollywood's MaxRestarts, the supervisor is not notified. Event stream subscription for `ActorMaxRestartsExceededEvent` is a candidate for S155.
4. **No dead-letter queue.** Failed batches and dropped events are logged but not persisted for replay.
5. **Backoff parameters are hardcoded.** Initial backoff (2s), cap (30s), and max restarts (5) are constants. Configuration is deferred until there is evidence it is needed.
6. **Hollywood `cleanup(nil)` bug.** When an actor exhausts Hollywood's MaxRestarts via panic, the framework calls `cleanup(nil)` which panics on `defer cancel()`. This is a pre-existing library issue outside S154 scope.

## Test Coverage

| Test | File | Verifies |
|------|------|----------|
| `TestCalcBackoff` | `cmd/writer/supervisor_test.go` | Exponential backoff: 2s→4s→8s→16s→30s→30s (cap) |
| `TestPipelineLifecycleTransitions` | `cmd/writer/supervisor_test.go` | State transitions: active → restarting → degraded |
| `TestPipelineStateConstants` | `cmd/writer/supervisor_test.go` | State string values match expectations |
| `TestHealthServer_Statusz_Phase_Degraded` | `internal/shared/healthz/healthz_test.go` | `/statusz` returns `"degraded"` phase when `pipeline_degraded` counter > 0 |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Localized failure no longer kills family permanently | **Met.** Supervisor retries up to 5 times before marking degraded. |
| Recovery/supervision are clear and minimally robust | **Met.** Three states, exponential backoff, tracker counters, log signatures. |
| Pipeline behavior is more predictable | **Met.** Defined lifecycle with documented transitions and observable counters. |
| Solution maintains simplicity and boundaries | **Met.** No generic framework. Recovery is specific to writer consumer-inserter topology. |
| Analytical layer closer to minimum real reliability | **Met.** Transient failures auto-recover; persistent failures are loud and diagnosable. |

## Preparation for S155

S155 (Observability) should consider:

1. **Event stream subscription for `ActorMaxRestartsExceededEvent`** — detect actor deaths from panics and mark families as degraded.
2. **Per-family structured log counters** — periodic log emission of pipeline health summary.
3. **Query latency and row count observability** — reader-side metrics.
4. **Restart budget cooling period** — optional reset after sustained active period.
5. **Diagnostic script enhancements** — `scripts/diag-check.sh` should check for degraded pipelines.
