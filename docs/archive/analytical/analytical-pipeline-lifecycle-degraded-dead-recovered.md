# Analytical Pipeline Lifecycle: Degraded, Dead, Recovered

## Purpose

This document defines the lifecycle states of an analytical pipeline family within the writer service. It establishes what each state means, how transitions happen, and what operators should expect in each state.

## Lifecycle States

### Active

The family's consumer and inserter are running normally. Events flow from NATS through the consumer, are buffered by the inserter, and batch-inserted into ClickHouse.

**Entry conditions:**
- Initial startup succeeds.
- Supervisor restart succeeds (consumer starts NATS subscription).

**Observable signals:**
- Consumer tracker: `event_count` incrementing.
- Inserter tracker: `events_flushed` incrementing.
- `/statusz` phase: `"active"` (if all families active).

### Restarting

The family's consumer failed to start (e.g., NATS unreachable). The supervisor has poisoned the old actors and scheduled a restart after a backoff delay. No events are consumed or inserted for this family during this window.

**Entry conditions:**
- Consumer sends `pipelineFailedMsg` to supervisor.
- Restart budget not yet exhausted.

**Duration:** Backoff delay (2s to 30s depending on restart count).

**Observable signals:**
- Consumer tracker: `pipeline_restarts` counter incremented.
- Consumer tracker: `error_count` incremented.
- Log: WARN with `"pipeline failure — scheduling restart"`.
- `/statusz` phase: may show `"warming"` (tracker awaiting first event after respawn).

**Exit conditions:**
- Restart succeeds → transitions to **Active**.
- Restart fails → re-enters **Restarting** (if budget remains) or transitions to **Degraded**.

### Degraded

The family exhausted its restart budget (5 attempts). The supervisor has stopped trying to recover it. No consumer or inserter actors exist for this family. Events for this family accumulate in NATS JetStream (bounded by stream retention, typically 72h).

**Entry conditions:**
- Restart count exceeds `maxPipelineRestarts` (5).

**Observable signals:**
- Consumer tracker: `pipeline_degraded` counter set to 1.
- Consumer tracker: `pipeline_restarts` = 6 (5 restarts + the triggering failure).
- Log: ERROR with `"pipeline degraded — restart budget exhausted"`.
- `/statusz` phase: `"degraded"`.
- `/diagz`: tracker shows no recent events, `pipeline_degraded` in counters.

**Exit conditions:**
- **None within the current process.** Degraded is a terminal state per process lifetime.

**Recovery path:**
1. Diagnose and fix the underlying issue (NATS configuration, network, etc.).
2. Restart the writer process (Docker restart or manual).
3. On restart, the family starts fresh with a full restart budget.
4. Unprocessed events are re-delivered from NATS durable consumer position.

## State Diagram

```
                    startup
                      │
                      ▼
                  ┌────────┐
                  │ Active │◄──────────────────┐
                  └───┬────┘                   │
                      │                        │
              consumer failure          restart succeeds
                      │                        │
                      ▼                        │
                ┌───────────┐                  │
           ┌───►│Restarting │──────────────────┘
           │    └─────┬─────┘
           │          │
     restart fails    │ budget exhausted
     (budget remains) │
           │          ▼
           │    ┌──────────┐
           └────│ Degraded │  (terminal per process)
                └──────────┘
```

## Operational Semantics

### What "Degraded" Means Operationally

A degraded family is **not dead** in the system-wide sense:

- Other families continue operating normally.
- The writer process stays alive and healthy for its remaining families.
- NATS retains unprocessed events (bounded by stream retention).
- ClickHouse already-inserted data remains queryable.

A degraded family means: **this specific projection pipeline has stopped writing new data until the process is restarted after the root cause is fixed.**

### What "Degraded" Does NOT Mean

- It does NOT mean the operational pipeline (ingest, derive, store, execute) is affected.
- It does NOT mean other analytical families are affected.
- It does NOT mean previously written data is lost or corrupted.
- It does NOT mean the family cannot be recovered (process restart recovers it).

### Distinguishing Degraded from Dead

| Attribute | Degraded | Dead (process crash) |
|-----------|----------|---------------------|
| Other families | Running | All stopped |
| Health endpoints | Responding | Unreachable |
| Recovery | Fix root cause + restart process | Docker auto-restart |
| NATS events | Accumulating (stream retention) | Accumulating (stream retention) |
| Restart budget | Exhausted | Reset on process restart |

### Why No Automatic Recovery from Degraded

The restart budget is intentionally fixed per process lifetime. Reasons:

1. **Prevents infinite restart storms.** If NATS is permanently misconfigured, unbounded restarts waste resources and flood logs.
2. **Forces operator attention.** Degraded state is loud (ERROR log, `/statusz` phase, tracker counter) and requires explicit action.
3. **Simple mental model.** Operators know exactly what happens: 5 attempts, then stop. Process restart resets everything.
4. **Docker restart is the escape valve.** If the underlying issue self-resolves (transient network partition), Docker's `restart: unless-stopped` policy eventually restarts the process with a fresh budget.

### Restart Budget Reset Consideration

A future enhancement could reset the restart budget after a "cooling period" (e.g., if a family runs successfully for 10 minutes after a restart, reset the counter). This is **not implemented** in S154 to maintain simplicity. The fixed-budget model is sufficient for the current stage.

## Counter Reference

| Counter | Type | Scope | Description |
|---------|------|-------|-------------|
| `pipeline_restarts` | Monotonic | Per consumer tracker | Total restart attempts for this family |
| `pipeline_degraded` | Flag (0 or 1) | Per consumer tracker | 1 if family is degraded, 0 otherwise |
| `events_flushed` | Monotonic | Per inserter tracker | Rows successfully inserted |
| `events_dropped` | Monotonic | Per inserter tracker | Rows permanently lost |
| `events_overflowed` | Monotonic | Per inserter tracker | Rows lost to buffer overflow |
| `flush_failures` | Monotonic | Per inserter tracker | INSERT operations exhausting retries |
