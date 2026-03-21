# Retry Metrics, Logging Semantics, and Usage

> S324 — Operational reference for retry observability signals.

## Overview

This document is the operational reference for the retry observability signals
introduced in S324. It defines the exact semantics, payloads, and usage
patterns for each signal.

## Log Events Reference

All log events are emitted by `RetrySubmitter` via the configured `slog.Logger`.
They use the logger's existing fields (e.g., `component`, `actor`) for context.

### `retry attempt failed` (Warn)

Emitted after each non-terminal retryable failure — i.e., the attempt failed
but more attempts remain.

| Field | Type | Description |
|-------|------|-------------|
| `attempt` | int | Which attempt just failed (1-based) |
| `max_attempts` | int | Total allowed attempts from policy |
| `error` | string | Problem message from the venue |

**When NOT emitted:** On the final attempt (that triggers exhaustion) or on
non-retryable errors.

### `retry succeeded` (Info)

Emitted when a retry attempt succeeds after at least one prior failure.

| Field | Type | Description |
|-------|------|-------------|
| `attempts` | int | Total attempts made (always > 1) |

**When NOT emitted:** On first-attempt success.

### `retry exhausted` (Warn)

Emitted when all retry attempts are consumed without success.

| Field | Type | Description |
|-------|------|-------------|
| `attempts` | int | Total attempts made (equals MaxAttempts) |
| `max_attempts` | int | Policy limit |
| `last_error` | string | Problem message from the final attempt |

### `retry halted by kill switch` (Warn)

Emitted when the kill switch (`GateChecker`) aborts the retry loop between
attempts.

| Field | Type | Description |
|-------|------|-------------|
| `attempts` | int | Attempts made before halt |

### `retry deadline exceeded` (Warn)

Emitted when the global retry deadline budget is exceeded before the next
attempt can start.

| Field | Type | Description |
|-------|------|-------------|
| `attempts` | int | Attempts made before deadline |

## Counter Metrics Reference

All counters are incremented via `healthz.Tracker.Counter(name).Add(1)` and
exposed on `/statusz` in the tracker's counter map.

### `retry_attempts`

- **Semantics:** Count of individual retry attempts that failed with a
  retryable error and were followed by another attempt. Does NOT count the
  first attempt or the terminal attempt that triggers exhaustion.
- **Operational use:** High values indicate sustained venue instability.
  Compare with `retry_success_after_retry` to gauge recovery rate.

### `retry_success_after_retry`

- **Semantics:** Count of submissions that ultimately succeeded after at
  least one retryable failure.
- **Operational use:** Non-zero is normal under transient venue errors.
  If zero while `retry_attempts` is high, retries are not recovering.

### `retry_exhausted`

- **Semantics:** Count of submissions where all retry attempts were consumed
  without success.
- **Operational use:** Any non-zero value warrants investigation. Sustained
  values indicate persistent venue unavailability.

### `retry_halted`

- **Semantics:** Count of retry sequences aborted by the kill switch.
- **Operational use:** Expected to be non-zero only during controlled halts.
  Unexpected values may indicate kill switch misconfiguration.

### `retry_deadline_exceeded`

- **Semantics:** Count of retry sequences aborted because the global retry
  budget was exceeded.
- **Operational use:** Indicates that the retry budget (default 10s) is
  insufficient for the venue's recovery time. May require policy tuning.

## Actor-Level Error Enrichment

When the `VenueAdapterActor` logs `venue submit failed`, it now includes
retry metadata from `Problem.Details` when present:

```json
{
  "level": "ERROR",
  "msg": "venue submit failed",
  "error": "venue unavailable (HTTP 503)",
  "source": "binancef",
  "symbol": "BTCUSDT",
  "timeframe": 3600,
  "correlation_id": "abc-123",
  "retry_attempts": 3,
  "retry_exhausted": true
}
```

This provides a single log line with both intent context and retry outcome.

## Usage Patterns

### Monitoring retry health

```
# In /statusz response, check:
counters.retry_exhausted > 0        → venue may be down
counters.retry_success_after_retry  → retries recovering normally
counters.retry_halted               → kill switch is active
```

### Grep patterns for structured logs

```bash
# All retry events
grep '"msg":"retry ' logs.json

# Only terminal outcomes
grep -E '"msg":"retry (exhausted|halted|deadline)' logs.json

# Success-after-retry (good news)
grep '"msg":"retry succeeded"' logs.json
```

### Alerting candidates (future)

| Signal | Condition | Severity |
|--------|-----------|----------|
| `retry_exhausted` | > 0 in 5min window | Warning |
| `retry_deadline_exceeded` | > 0 in 5min window | Warning |
| `retry_halted` | > 0 when halt not expected | Info |

## Noise Budget

| Scenario | Logs emitted | Counters incremented |
|----------|-------------|---------------------|
| First-attempt success | 0 | 0 |
| Success on 2nd attempt | 2 (1 warn + 1 info) | 2 (`retry_attempts` + `retry_success_after_retry`) |
| Exhaustion at 3 attempts | 3 (2 warn attempts + 1 warn exhaustion) | 3 (`retry_attempts` x2 + `retry_exhausted`) |
| Halt after 1st attempt | 2 (1 warn attempt + 1 warn halt) | 2 (`retry_attempts` + `retry_halted`) |
| Deadline after 1st attempt | 2 (1 warn attempt + 1 warn deadline) | 2 (`retry_attempts` + `retry_deadline_exceeded`) |

The noise ceiling is bounded by `MaxAttempts` (default 3), producing at most
3 log lines per submission.
