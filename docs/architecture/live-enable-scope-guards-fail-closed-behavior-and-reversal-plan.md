# Live Enable: Scope Guards, Fail-Closed Behavior, and Reversal Plan

> Authority: S445 | Date: 2026-03-24

## Purpose

This document catalogs every defense layer that prevents accidental live trading scope expansion after the C-6 removal. It demonstrates that removing the S433 validation block does not weaken any other safety mechanism, and documents the reversal plan.

## Defense Layers (Post-C-6)

### Layer 1: Fail-Closed Default (INTACT)

**Location:** `schema.go:679-684`

```go
func (v VenueConfig) IsDryRun() bool {
    return v.DryRun == nil || *v.DryRun
}
```

**Effect:** Any config that does not explicitly set `dry_run: false` operates in dry-run mode. This is the primary structural guard. It ensures that:
- New configs default to dry-run.
- Copy-pasted configs default to dry-run.
- Configs with missing or null `dry_run` fields default to dry-run.

**Test:** `TestS445_FailClosed_IsDryRun_NilDefaultsTrue`

### Layer 2: DryRunSubmitter Interception (INTACT)

**Location:** `cmd/execute/run.go:86-96`

```go
dryRunActive := config.Venue.IsDryRun()
if dryRunActive {
    drs := appexec.NewDryRunSubmitter(venueResult.submit)
    // ... wraps all venue calls
    venueResult.submit = drs
}
```

**Effect:** When `IsDryRun()` returns true, DryRunSubmitter wraps the venue adapter and intercepts 100% of `SubmitOrder` calls. No real HTTP call reaches any venue. This layer is completely independent of config validation -- it operates at runtime wiring.

### Layer 3: Paper Simulator Guard (INTACT)

**Location:** `schema.go:505-513`

```go
if v.DryRun != nil && !*v.DryRun {
    if v.Type == VenueTypePaperSimulator || (v.Type == "" && !v.HasUnifiedSegments()) {
        // reject: dry_run=false with paper_simulator is contradictory
    }
}
```

**Effect:** `dry_run=false` with `paper_simulator` is still rejected. Paper configs cannot accidentally become live.

**Test:** `TestS445_FailClosed_PaperSimulator_DryRunFalse_StillRejected`

### Layer 4: Credential Preflight (INTACT)

**Location:** `preflight.go:74-96`

**Effect:** Mainnet adapters require credentials at boot. If credentials are missing, the binary exits immediately. This prevents accidental mainnet adapter activation without proper secret management.

### Layer 5: Kill-Switch / SafetyGate (INTACT)

**Location:** `venue_adapter_actor.go` (SafetyGate check before every venue call)

**Effect:** Even with `dry_run=false` and real adapters wired, the SafetyGate checks the kill-switch state before every order submission. If the gate is halted, the order is rejected without reaching the venue.

### Layer 6: Config Profile Isolation (NEW)

**Location:** `deploy/configs/execute-mainnet-live.jsonc`

**Effect:** The live config is a separate, dedicated file. It specifies:
- Spot only (no Futures segment)
- File-based credential provider
- `dry_run: false`

Other config profiles (`execute.jsonc`, `execute-unified.jsonc`, `execute-mainnet-dry-run.jsonc`) are unchanged and retain `dry_run: true`.

## Scope Containment Matrix

| Scenario | Prevented By | Status |
|----------|-------------|--------|
| Accidental live from default config | Layer 1 (fail-closed default) | INTACT |
| Live with paper_simulator | Layer 3 (paper guard) | INTACT |
| Live without credentials | Layer 4 (credential preflight) | INTACT |
| Live with kill-switch halted | Layer 5 (SafetyGate) | INTACT |
| Live from non-live config profile | Layer 6 (config isolation) | INTACT |
| Any order without operator | Kill-switch protocol | INTACT |

## What C-6 Removed

Only Layer 0 (the S433 validation block) was removed. This was the administrative guard that prevented `dry_run=false` from being set in config when mainnet adapters were present. With C-6 executed, operators CAN now set `dry_run=false` with mainnet adapters.

**This is intentional and authorized by S443.**

The remaining 6 defense layers are independent and unmodified.

## Reversal Plan

### Trigger

Any of these conditions triggers reversal:

| ID | Condition | Action |
|----|-----------|--------|
| REV-1 | Test regression discovered after S445 | `git revert <S445-commit>` |
| REV-2 | Safety invariant SI-2 through SI-12 found broken | `git revert <S445-commit>`, full audit |
| REV-3 | Operator decision to re-lock mainnet live | `git revert <S445-commit>` |
| REV-4 | Stop condition triggered during S446 session | Kill-switch first, then `git revert` if required |

### Reversal Procedure

1. `git revert <S445-commit-hash>` -- single operation, restores S433 validation block.
2. Verify `go test ./internal/shared/settings/...` passes (original S433/S436 tests restored).
3. Remove `deploy/configs/execute-mainnet-live.jsonc` if no longer needed.
4. Confirm `dry_run=false` + mainnet is once again rejected.

**Estimated reversal time:** under 5 minutes.

### Post-Reversal State

After reversal, the system returns to exactly the state before S445:
- `dry_run=false` + mainnet rejected at config validation.
- All other defense layers unchanged (they were never modified).
- No data migration or runtime cleanup required.

## References

- [C-6 Controlled Removal](c6-controlled-dry-run-false-removal.md) (S445)
- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
