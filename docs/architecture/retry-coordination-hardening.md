# Retry Coordination Hardening

> **Stage:** S323
> **Predecessor:** S319 (Retry Infrastructure), S320 (Venue Failure Path Verification), S322 (Post-200 Reconciliation)

## 1. Purpose

S320 identified two coordination gaps in the retry path:

- **R-S320-2**: No global deadline/budget on the retry sequence — retries could theoretically consume unbounded wall-clock time under adversarial backoff conditions.
- **R-S320-3**: The kill switch (execution control gate) is only checked before the first attempt. If the control plane signals halt while retries are in flight, the loop continues blind.

S323 closes both gaps with minimal, backward-compatible changes to `RetrySubmitter`.

## 2. Global Retry Deadline (Budget)

### Design

The `RetryPolicy` struct gains a `Deadline` field:

```go
type RetryPolicy struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Factor      float64
    Deadline    time.Duration  // NEW: global retry budget
}
```

When `Deadline > 0`, the retry loop records the start time and checks before each attempt whether the budget is exhausted. The check is **independent of context deadlines** — it governs the retry sequence as a whole, not individual submit calls.

### Semantics

| Condition | Behavior |
|-----------|----------|
| `Deadline == 0` | No global deadline; only MaxAttempts and context govern |
| `Deadline > 0`, budget remaining | Loop continues normally |
| `Deadline > 0`, budget exceeded | Loop aborts; Problem carries `retry_deadline_exceeded: true` |
| Deadline exceeded before first attempt | Immediate abort (no submission) |

### Default

`DefaultRetryPolicy()` sets `Deadline: 10s`. With 3 attempts and 2s max delay, this is generous but prevents runaway sequences.

## 3. Kill Switch Awareness Between Attempts

### Design

`RetrySubmitter` gains an optional `GateChecker` via `WithHaltChecker()`:

```go
rs := NewRetrySubmitter(inner, policy).WithHaltChecker(controlStore)
```

Between each pair of attempts (after a retryable failure, before the backoff sleep), the submitter checks `haltChecker.IsHalted(ctx)`. If halted, the loop aborts immediately.

### Semantics

| Condition | Behavior |
|-----------|----------|
| No GateChecker configured (nil) | Fail-open — no halt check (backward-compatible) |
| GateChecker returns `false` | Retry continues |
| GateChecker returns `true` | Abort; Problem carries `retry_halted: true` |
| GateChecker read times out (2s) | Fail-open (`false`) — consistent with SafetyGate behavior |

### Check Ordering in the Loop

```
for attempt := 1..MaxAttempts:
  1. Check ctx.Err()           → context abort
  2. Check global deadline     → deadline abort
  3. SubmitOrder()             → success or error
  4. (if retryable, not last)
     a. Check kill switch      → halt abort
     b. Backoff sleep          → ctx cancel abort
```

The kill switch is checked **after** a failed attempt and **before** backoff sleep. This ensures:
- The first attempt is never blocked by the halt check (SafetyGate handles pre-submit gate).
- If halt is signaled during retries, the system stops before investing more backoff time.

## 4. Abort Metadata

All abort paths produce structured metadata in Problem.Details:

| Key | Type | When Set |
|-----|------|----------|
| `retry_attempts` | int | Always (on any abort or exhaustion) |
| `retry_exhausted` | bool | When MaxAttempts are used up |
| `retry_halted` | bool | When kill switch triggered abort |
| `retry_deadline_exceeded` | bool | When global deadline exceeded |

Only one of `retry_exhausted`, `retry_halted`, or `retry_deadline_exceeded` is set per abort — they are mutually exclusive.

## 5. Backward Compatibility

- Existing `NewRetrySubmitter(inner, policy)` calls continue to work unchanged.
- `WithHaltChecker` is opt-in. Without it, behavior is identical to pre-S323.
- `Deadline: 0` (which existing `RetryPolicy{}` literal values produce) disables the deadline check.
- `DefaultRetryPolicy()` now includes `Deadline: 10s` — a safe default that does not change observable behavior for the 3-attempt/2s-cap policy (total worst-case is ~3.5s).
- All 9 existing retry tests pass without modification.

## 6. Integration with VenueAdapterActor

The `VenueAdapterActor` already checks the kill switch via `SafetyGate.Check()` before calling `Venue.SubmitOrder()`. With S323, the kill switch is also checked between retry attempts inside the submitter.

To wire the halt checker in production, the actor passes the same `controlStore` (which already implements `GateChecker`) to the `RetrySubmitter`:

```go
retrySubmitter := execution.NewRetrySubmitter(adapter, policy).
    WithHaltChecker(controlStore)
```

This is additive — SafetyGate continues to protect the pre-submit path, and RetrySubmitter now protects the mid-retry path.
