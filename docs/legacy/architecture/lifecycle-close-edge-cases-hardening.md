# Lifecycle Close Edge Cases Hardening

**Stage**: S500
**Status**: Complete
**Scope**: Hardening of lifecycle close boundary conditions, not OMS expansion

## 1. Objective

Reduce ambiguity and inconsistency at lifecycle close boundaries by hardening edge cases in session finalization, carry-forward eligibility, cross-session pairing, and reconciliation flagging.

## 2. Edge Cases Addressed

### 2.1 Session Double-Close (Idempotency Guard)

**Problem**: `Session.Close()` and `Session.Halt()` could be called on already-terminal sessions without error, silently overwriting counters and timestamps.

**Fix**: Both methods now return `*problem.Problem` and refuse to transition an already-terminal session. The execute supervisor handles this gracefully by logging and returning early.

**Impact**: Prevents counter/timestamp corruption from race conditions or redundant shutdown signals.

### 2.2 Temporal Ordering Invariant

**Problem**: `Session.Validate()` did not check whether `ClosedAt` precedes `StartedAt`, allowing structurally invalid sessions.

**Fix**: Added validation rule: `ClosedAt` must not be before `StartedAt`. Equal timestamps are valid (zero-duration sessions from instant operations).

**Impact**: Catches clock skew or programming errors at validation time rather than producing confusing query results.

### 2.3 In-Flight Orders at Session Close

**Problem**: When a session closes, orders in non-terminal state (submitted, sent, accepted, partially_filled) are silently abandoned. There was no counter to surface this at the session metadata level.

**Fix**: Added `InFlight` field to `SessionSegmentCounters` and `HasInFlightOrders()`/`TotalInFlight()` helper methods on `Session`. The execute supervisor can populate this from its tracker state.

**Impact**: Operators can now detect and triage sessions that closed with in-flight orders. Downstream reconciliation can use this to add the `non_terminal_at_close` flag.

### 2.4 Halted Session Origin Awareness

**Problem**: Round-trips with legs from halted sessions (kill-switch, error condition) were indistinguishable from normal round-trips in reconciliation output.

**Fix**: Added `FlagHaltedSessionOrigin` reconciliation flag and `HaltedOrigin` field in `ContinuityReconciliationResult`. Applied via the new `LifecycleCloseContext` parameter on `ReconcileCrossSessionRoundTrip`.

**Impact**: Operators can filter/flag round-trips from halted sessions for extra review. Carryover reliability is automatically degraded.

### 2.5 Non-Terminal at Close Reconciliation

**Problem**: When an execution intent was non-terminal at session close, the resulting leg data may be incomplete (no final fill, no accurate quantity). This was not flagged in reconciliation.

**Fix**: Added `FlagNonTerminalAtClose` reconciliation flag, applied via `LifecycleCloseContext`. Carryover reliability is automatically degraded when present.

**Impact**: Distinguishes legs with potentially incomplete data from fully resolved legs.

### 2.6 Cancelled-with-Partial-Fill Carry-Forward

**Problem**: The carry-forward eligibility rule R-CF5 correctly handles cancelled orders with partial fills, but this edge case had no explicit test coverage.

**Fix**: Added explicit test (`TestClassifyCarryForward_CancelledWithFills_IsEligible`) confirming R-CF5 behavior, plus `TestIntentToLeg_CancelledWithPartialFill_ProducesValidLeg` confirming correct quantity aggregation from fills (not from the original intent quantity).

**Impact**: Regression protection for a critical boundary condition.

### 2.7 Cross-Session Partial Remainder Cascade

**Problem**: A large entry (e.g., 0.3 BTC) that pairs partially across multiple sessions (0.1 in session 2, 0.1 in session 3) leaving a remainder — this scenario had no end-to-end test.

**Fix**: Added `TestMatchFIFO_CrossSession_PartialRemainderCascade` covering 3-session cascade with quantity splitting.

**Impact**: Validates that FIFO matching + annotation + summarization handle cascading partial matches correctly.

### 2.8 Boundary Timestamp Equality

**Problem**: When entry and exit have identical timestamps, the M4 invariant (entry.Timestamp <= exit.Timestamp) should still allow pairing. This boundary condition was untested.

**Fix**: Added `TestMatchFIFO_SameTimestamp_EntryAndExit_Pair` confirming M4 accepts equal timestamps.

**Impact**: Documents and protects the boundary behavior for high-frequency scenarios.

## 3. Design Decisions

### 3.1 Variadic LifecycleCloseContext

The `ReconcileCrossSessionRoundTrip` function signature uses a variadic parameter for `LifecycleCloseContext`:

```go
func ReconcileCrossSessionRoundTrip(csrt CrossSessionRoundTrip, attr *effectiveness.Attribution, lcCtx ...*LifecycleCloseContext) ContinuityReconciliationResult
```

**Rationale**: This preserves backward compatibility — all existing callers pass 2 arguments and continue to work unchanged. The lifecycle close context is optional and only adds flags when provided.

### 3.2 Carryover Reliability Degradation

Both `halted_session_origin` and `non_terminal_at_close` flags automatically degrade `CarryoverReliable` to false. This is intentional: if the session close was abnormal, the data may be incomplete and P&L figures cannot be trusted for effectiveness analysis.

### 3.3 Session Close Returns Error

Changing `Close()`/`Halt()` from `void` to `*problem.Problem` is a breaking change at the type level. All callers were updated:
- Execute supervisor: handles error by logging and returning early
- Tests: check the error return

## 4. What Was NOT Changed

- **No write-path changes to execution pipeline**: All hardening is in domain types and read-path reconciliation.
- **No new ClickHouse tables or columns**: Flags are computed at read time.
- **No position engine**: Sessions remain isolated at runtime.
- **No OMS expansion**: Scope is strictly lifecycle close boundary conditions.
- **No changes to MatchFIFO algorithm**: The matching logic is correct; only its edge case test coverage was expanded.
- **No changes to carry-forward classification rules**: R-CF1 through R-CF5 are unchanged; only test coverage was added.

## 5. Files Changed

| File | Change Type | Description |
|------|------------|-------------|
| `internal/domain/execution/session.go` | Modified | Close/Halt return error; InFlight counter; temporal validation; HasInFlightOrders/TotalInFlight |
| `internal/domain/pairing/reconciliation.go` | Modified | FlagNonTerminalAtClose, FlagHaltedSessionOrigin |
| `internal/domain/pairing/continuity_reconciliation.go` | Modified | LifecycleCloseContext, HaltedOrigin, variadic lcCtx parameter |
| `internal/actors/scopes/execute/execute_supervisor.go` | Modified | Handle Close/Halt error return |
| `internal/domain/execution/s500_lifecycle_close_test.go` | New | Session close edge case tests |
| `internal/domain/pairing/s500_lifecycle_close_test.go` | New | Pairing lifecycle close edge case tests |
| `internal/domain/execution/session_test.go` | Modified | Adapted to new Close/Halt signatures |
| `internal/application/execution/s460_session_metadata_test.go` | Modified | Adapted to new Close signature |

## 6. Limitations

| ID | Description | Mitigation |
|----|------------|------------|
| L-S500-1 | `LifecycleCloseContext` is caller-provided, not automatically derived | Read-path callers must supply session metadata; gateway already has session data available |
| L-S500-2 | `InFlight` counter depends on tracker state at close time | May undercount if tracker is not fully flushed; acceptable for advisory counter |
| L-S500-3 | No retrospective fill ingestion for abandoned orders | By design — sessions are isolated; documented in open-fragments architecture |
| L-S500-4 | Halted session origin detection requires session status lookup | Not automatic in MatchFIFO; applied in read-path reconciliation |
