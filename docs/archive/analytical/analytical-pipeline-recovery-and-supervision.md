# Analytical Pipeline Recovery and Supervision

## Purpose

This document defines the recovery and supervision semantics for the writer service's analytical pipelines. It establishes how the system responds to pipeline failures, what recovers automatically, and what requires manual intervention.

## Problem Statement

Prior to S154, a consumer startup failure (e.g., NATS unavailable during writer startup) called `Poison(self)`, permanently killing the consumer-inserter pair for that family. The family remained dead until the entire writer process was restarted. Other families sharing the same process were unaffected but the failed family could not recover.

## Recovery Model

### Supervisor-Managed Restart

The `writerSupervisor` actor owns all pipeline families. When a consumer reports a startup failure via `pipelineFailedMsg`, the supervisor:

1. Records the failure in the family's lifecycle state.
2. Increments the restart counter and records it in the consumer's health tracker.
3. Poisons the failed consumer and its paired inserter.
4. Schedules a restart after an exponential backoff delay.
5. On restart: spawns a fresh consumer-inserter pair for the family.

If the restart budget is exhausted (5 attempts), the family is marked **degraded** and no further restarts are attempted. Other families continue operating normally.

### Backoff Schedule

| Restart | Delay |
|---------|-------|
| 1       | 2s    |
| 2       | 4s    |
| 3       | 8s    |
| 4       | 16s   |
| 5       | 30s   |

Delays are capped at 30 seconds. The total time from first failure to degraded state (if all attempts fail) is approximately 60 seconds.

### Why Supervisor-Managed (Not Framework-Level)

Hollywood's built-in actor restart mechanism (`WithMaxRestarts`) handles panics via `recover()`. It does not handle voluntary shutdown or poison. Since consumer startup failures are detected through error returns (not panics), Hollywood's restart does not trigger. The supervisor-managed approach gives:

- Explicit control over backoff policy.
- Per-family lifecycle state tracking.
- Observability through health tracker counters.
- Clean separation: Hollywood handles actor mechanics, supervisor handles pipeline recovery.

## What Recovers Automatically

| Scenario | Recovery |
|----------|----------|
| Consumer startup fails (NATS temporarily unavailable) | Supervisor retries up to 5 times with exponential backoff |
| ClickHouse INSERT fails | Inserter retries up to 5 times with exponential backoff (per batch) |
| NATS connection drops during operation | NATS client reconnects automatically; durable consumer resumes |
| Brief ClickHouse outage (<60s) | Inserter buffers rows, retries, flushes on recovery |

## What Does NOT Recover Automatically

| Scenario | Behavior | Recovery Path |
|----------|----------|---------------|
| Consumer fails 5+ times on startup | Family marked degraded; no further restarts | Restart writer process |
| ClickHouse schema mismatch | INSERT retries exhaust; batch dropped | Fix schema, restart writer |
| NATS stream deleted | Consumer cannot resubscribe | Recreate stream, restart writer |
| Writer process crash | All families lost | Docker `restart: unless-stopped` restarts process |
| Inserter actor crash (panic) | Hollywood restarts up to 3 times (default) | If exceeded: family partially dead until process restart |

## Observability

### Health Tracker Counters (per family)

| Counter | Meaning |
|---------|---------|
| `pipeline_restarts` | Number of supervisor-initiated restart attempts |
| `pipeline_degraded` | Set to 1 when the family exhausts its restart budget |
| `events_flushed` | Rows successfully inserted into ClickHouse |
| `events_dropped` | Rows permanently lost (overflow + retry exhaustion) |
| `events_overflowed` | Rows lost to buffer overflow specifically |
| `flush_failures` | INSERT operations that exhausted all retries |

### Phase Derivation

The `/statusz` and `/diagz` endpoints derive an aggregate `phase` from tracker state. A new `"degraded"` phase is emitted when any tracker has `pipeline_degraded > 0`, providing immediate operational visibility.

### Log Signatures

| Event | Level | Key Fields |
|-------|-------|------------|
| Consumer startup failure | WARN | `family`, `error`, `restart`, `backoff` |
| Pipeline restart attempt | INFO | `family`, `attempt` |
| Pipeline degraded | ERROR | `family`, `restarts`, `last_error` |
| Pipeline recovery (successful restart) | INFO | `family`, `attempt` (via consumer "started" log) |

## Design Constraints

1. **No generic supervision framework.** Recovery logic is specific to the writer's consumer-inserter topology.
2. **Fixed restart budget per process lifetime.** The counter does not reset after successful restarts. This prevents infinite restart loops for intermittent failures.
3. **No cross-family coupling.** One family's failure or degradation never affects other families.
4. **Restart budget is hardcoded (5).** Configurable restart limits are deferred until there is evidence they are needed.
5. **Inserter restart is not independently managed.** The inserter is stopped and respawned alongside its consumer. It cannot fail independently in a way that triggers supervisor recovery (its errors are handled internally through retry+drop).

## Invariants Preserved

- **INV-01:** Writer failure never affects the operational pipeline.
- **INV-02:** NATS consumer never blocked by ClickHouse failure.
- **INV-03:** Writer never publishes to NATS.
- **R-01 through R-10:** All analytical optionality rules remain enforced.
