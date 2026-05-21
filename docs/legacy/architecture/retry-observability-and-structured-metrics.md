# Retry Observability and Structured Metrics

> S324 — Structured observability for the venue retry path.

## Purpose

The retry submitter (S320/S323) carries structured metadata on failure paths
via `Problem.Details`, but emitted no logs or counters. This left operators
unable to answer basic questions without parsing error payloads:

- How often do retries succeed vs. exhaust?
- Is the kill switch halting retry sequences?
- Are deadline aborts occurring under load?

S324 adds **minimal, structured observability** to the retry path: structured
logs via `slog` and atomic counters via `healthz.Tracker`, both optional and
nil-safe.

## Design Principles

1. **Proportional** — only signals with operational value; no per-field
   explosion or high-cardinality labels.
2. **Optional** — nil logger and nil tracker are safe; existing callers
   unchanged.
3. **Co-located** — observability lives in `RetrySubmitter`, where the retry
   decisions happen. No proxy layers or separate collectors.
4. **Low-noise** — first-attempt success emits nothing. Only retries,
   exhaustion, halt, and deadline generate signals.

## Integration Points

```
RetrySubmitter
├─ WithLogger(*slog.Logger)   → structured log events
└─ WithTracker(*healthz.Tracker) → atomic counter metrics
```

Both are wired at composition time (bootstrap/main). The `VenuePort` interface
is unchanged.

## Structured Log Events

| Event | Level | When | Key Fields |
|-------|-------|------|------------|
| `retry attempt failed` | Warn | Each non-terminal retryable failure | `attempt`, `max_attempts`, `error` |
| `retry succeeded` | Info | Success on attempt > 1 | `attempts` |
| `retry exhausted` | Warn | All attempts consumed | `attempts`, `max_attempts`, `last_error` |
| `retry halted by kill switch` | Warn | Kill switch aborted between attempts | `attempts` |
| `retry deadline exceeded` | Warn | Global deadline budget exceeded | `attempts` |

**Noise control:** first-attempt success produces zero log output.

## Counter Metrics

| Counter | Incremented When |
|---------|-----------------|
| `retry_attempts` | Each non-terminal retryable failure (counts individual retry attempts, excluding first attempt) |
| `retry_success_after_retry` | Success on attempt > 1 |
| `retry_exhausted` | MaxAttempts consumed without success |
| `retry_halted` | Kill switch aborted the retry loop |
| `retry_deadline_exceeded` | Global deadline budget exceeded |

Counters are exposed via `/statusz` under the tracker's counter map. They
accumulate monotonically for the process lifetime.

## Actor-Level Enrichment

The `VenueAdapterActor` error log now surfaces retry metadata from
`Problem.Details` when present:

```
logger.Error("venue submit failed",
    "error", prob.Message,
    ...
    "retry_attempts", 3,        // if present
    "retry_exhausted", true,    // if present
)
```

This provides correlated visibility at the actor level without duplicating
the retry-level logs.

## Composition Example

```go
retrySubmitter := execution.NewRetrySubmitter(adapter, execution.DefaultRetryPolicy()).
    WithHaltChecker(controlStore).
    WithLogger(logger.With("component", "retry-submitter")).
    WithTracker(tracker)
```

## Backward Compatibility

- All 17 existing retry tests pass without modification.
- `WithLogger(nil)` and `WithTracker(nil)` are no-ops (same as not calling them).
- The `VenuePort` interface is unchanged.
- Problem.Details metadata contract is unchanged.

## Invariants

- **INV-OBS-1**: First-attempt success emits zero observability signals.
- **INV-OBS-2**: Nil logger and nil tracker never panic.
- **INV-OBS-3**: Counter names are stable; renaming is a breaking change.
- **INV-OBS-4**: Log messages are stable identifiers for grep/alert rules.
- **INV-OBS-5**: Retry-level logs and actor-level logs are complementary, not duplicative — retry logs carry attempt-level detail, actor logs carry intent-level context.

## What This Does NOT Cover

- Per-symbol or per-venue retry breakdown (high cardinality).
- Dashboards or alerting rules.
- Distributed tracing spans.
- Retry-After header parsing (R-S320-6).
- Venue error code classification (R-S320-4).
