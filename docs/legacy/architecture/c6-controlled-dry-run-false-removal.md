# C-6: Controlled Removal of `dry_run=false` Rejection for Mainnet Adapters

> Authority: S445 | Date: 2026-03-24 | Predecessor: S443 (Live Trading Authorization Evidence Gate)

## Purpose

This document records the controlled execution of condition C-6 from the S443 evidence gate: removal of the config validation rule that rejected `dry_run=false` when mainnet adapters are configured.

## What Changed

### Single Point of Change

**File:** `internal/shared/settings/schema.go`
**Location:** Lines 515-524 (former S433 validation block)

**Before (S433):**
```go
// S433: Mainnet adapters require dry_run=true. Live trading on mainnet
// requires a separate authorization ceremony -- this wave only proves dry-run.
if v.DryRun != nil && !*v.DryRun {
    if v.hasMainnetAdapter() {
        issues = append(issues, problem.ValidationIssue{
            Field:   "venue.dry_run",
            Message: "dry_run=false is not authorized for mainnet adapters -- mainnet live trading requires a separate authorization ceremony",
        })
    }
}
```

**After (S445):**
```go
// S445: dry_run=false is now authorized for mainnet adapters.
// Authorization: S443 evidence gate, condition C-6.
// Scope: Binance Spot, BTCUSDT, market order, minimum quantity, supervised ceremony.
// Fail-closed behavior is preserved by IsDryRun() defaulting to true when omitted.
// DryRunSubmitter, SafetyGate, and kill-switch remain fully intact for all profiles
// where dry_run is not explicitly set to false.
// Reversal: restore the validation block from S433 (git revert).
```

### What This Change Does

- Allows `dry_run=false` in config when mainnet adapters are configured.
- When `dry_run=false` is set, `IsDryRun()` returns `false`, and the DryRunSubmitter is NOT wrapped around the venue adapter in `cmd/execute/run.go:86-96`.
- Real HTTP calls reach the mainnet venue endpoint.

### What This Change Does NOT Do

- Does not alter the fail-closed default: `IsDryRun()` still returns `true` when `DryRun` is `nil` (omitted).
- Does not modify DryRunSubmitter, SafetyGate, or kill-switch code.
- Does not change any other validation rule.
- Does not affect testnet or paper_simulator configs.
- Does not introduce new config fields, new adapters, or new runtime paths.

## Authorization Chain

```
S437 (Mainnet Authorization Evidence Gate)
  -> C-6 defined as explicit condition
    -> S438 (Live Trading Authorization Wave Charter)
      -> S439-S442 (C-1 through C-5 closed with evidence)
        -> S443 (Evidence Gate: AUTHORIZED -- CONDITIONAL)
          -> S444 (Enablement Ceremony Charter)
            -> S445 (C-6 executed)
```

## Safety Invariant Impact

| # | Invariant | Status After S445 |
|---|-----------|-------------------|
| SI-1 | Config rejects dry_run=false + mainnet | **INTENTIONALLY MODIFIED** (C-6) |
| SI-2 | DryRunSubmitter intercepts all SubmitOrder | INTACT |
| SI-3 | DryRunSubmitter has zero bypass paths | INTACT |
| SI-4 | SafetyGate before venue calls | INTACT |
| SI-5 | Kill-switch enforcement via IsHalted() | INTACT |
| SI-6 | gateReadTimeout = 2s | INTACT |
| SI-7 | MainnetCredentialCheck at preflight | INTACT |
| SI-8 | CredentialPathCheck at preflight | INTACT |
| SI-9 | Phase -1 credential provider wiring | INTACT |
| SI-10 | HTTP PUT /execution/control | INTACT |
| SI-11 | HTTP GET /execution/control | INTACT |
| SI-12 | Gateway composition connects control | INTACT |

**11/12 INTACT. SI-1 intentionally modified per C-6 authorization.**

## Artifacts

| Artifact | Path |
|----------|------|
| Schema change | `internal/shared/settings/schema.go:515-521` |
| Live config | `deploy/configs/execute-mainnet-live.jsonc` |
| S445 tests | `internal/shared/settings/s445_c6_controlled_removal_test.go` |
| Updated S433 tests | `internal/shared/settings/s433_mainnet_adapter_config_test.go` |
| Updated S436 tests | `internal/shared/settings/s436_mainnet_dryrun_config_test.go` |

## Reversal

To reverse this change:
1. Restore the S433 validation block in `schema.go` (the `git revert` of the S445 commit).
2. Revert the test updates in s433, s436, and s445 test files.
3. Remove `deploy/configs/execute-mainnet-live.jsonc`.

The reversal is a single `git revert` operation. No data migration, no config cleanup, no runtime state change required.

## References

- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md) (S443)
- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [Scope Guards and Fail-Closed Behavior](live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md) (S445)
