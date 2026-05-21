# Stage S445: C-6 Controlled Execution Report

Stage: S445
Wave: Live Trading Enablement Ceremony (S444-S448)
Block: 1 (C-6 Controlled Execution)
Predecessor: S444 (Ceremony Charter)
Date: 2026-03-24

## Objective

Execute condition C-6 from the S443 evidence gate: remove the config validation rule that rejected `dry_run=false` for mainnet adapters, under controlled conditions with zero scope expansion beyond the minimum authorized surface.

## Executive Summary

C-6 has been executed. The S433 validation block in `schema.go:515-524` has been replaced with an authorization comment documenting the S443 provenance. A production config profile (`execute-mainnet-live.jsonc`) has been created for the minimum authorized scope. All tests pass with zero regressions. 11/12 safety invariants remain intact; SI-1 was intentionally modified per C-6 authorization.

## Deliverables

### 1. Schema Change (C-6)

**File:** `internal/shared/settings/schema.go:515-521`

The S433 validation block that rejected `dry_run=false` when `hasMainnetAdapter()` returned true has been replaced with an authorization comment. The change is a single, isolated, reviewable diff.

**Impact:** Config validation no longer rejects `dry_run=false` + mainnet. All other validation rules are unchanged.

### 2. Production Config Profile

**File:** `deploy/configs/execute-mainnet-live.jsonc`

Created for exactly the minimum authorized scope:
- Segment: Spot only (no Futures)
- Adapter: `binance_spot_mainnet`
- `dry_run: false`
- `credential_provider: "file"`
- `credential_path: "/run/secrets/market-foundry"`

### 3. Test Updates

| File | Change |
|------|--------|
| `s445_c6_controlled_removal_test.go` | NEW: 9 tests covering C-6 validity and fail-closed guards |
| `s433_mainnet_adapter_config_test.go` | UPDATED: `TestMainnetAdapter_DryRunEnforcement` -> `TestMainnetAdapter_DryRunFalse_NowValid` |
| `s436_mainnet_dryrun_config_test.go` | UPDATED: `TestS436_MainnetDryRunFalse_Rejected` -> `TestS436_MainnetDryRunFalse_NowValid` |

### 4. Architecture Documents

| Document | Path |
|----------|------|
| C-6 controlled removal | `docs/architecture/c6-controlled-dry-run-false-removal.md` |
| Scope guards and reversal plan | `docs/architecture/live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md` |

## Safety Invariant Verification

| # | Invariant | Status |
|---|-----------|--------|
| SI-1 | Config rejects dry_run=false + mainnet | **MODIFIED** (C-6 authorized) |
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

**11/12 INTACT. SI-1 intentionally modified per C-6.**

## Fail-Closed Guards Verification

| Guard | Test | Status |
|-------|------|--------|
| IsDryRun() defaults true when nil | `TestS445_FailClosed_IsDryRun_NilDefaultsTrue` | PASS |
| paper_simulator + dry_run=false rejected | `TestS445_FailClosed_PaperSimulator_DryRunFalse_StillRejected` | PASS |
| Mainnet + dry_run=true valid | `TestS445_FailClosed_MainnetDryRunTrue_StillValid` | PASS |
| Mainnet + dry_run omitted defaults true | `TestS445_FailClosed_MainnetDryRunOmitted_DefaultsToTrue` | PASS |
| Testnet + dry_run=false no regression | `TestS445_Testnet_DryRunFalse_NoRegression` | PASS |

## Test Results

| Package | Result | Tests |
|---------|--------|-------|
| internal/shared/settings | PASS | All (including 9 new S445 tests) |
| internal/shared/bootstrap | PASS | All |
| internal/application/execution | PASS | All |
| internal/domain/execution | PASS | All |
| internal/actors/scopes/execute | PASS | All |
| internal/adapters/nats/* | PASS | All |

**Zero regressions. Zero failures.**

## Governing Questions (S444 Block 1)

| ID | Question | Answer | Evidence |
|----|----------|--------|----------|
| GQ-1 | Has the `dry_run=false` rejection been removed from schema.go? | YES | schema.go:515-521 (S433 block replaced with authorization comment) |
| GQ-2 | Does the live config specify exactly the minimum authorized scope? | YES | execute-mainnet-live.jsonc: 1 segment (Spot), file credentials, dry_run=false |
| GQ-3 | Do all existing tests pass after the removal? | YES | All packages PASS, zero regressions |
| GQ-4 | Are all safety invariants (except SI-1) intact? | YES | 11/12 INTACT, SI-1 intentionally modified |

## Exit Criteria Assessment

| Criterion | Status |
|-----------|--------|
| C-6 executed | DONE |
| Config created for minimum scope | DONE |
| Zero regressions | VERIFIED |
| All safety invariants (except SI-1) intact | VERIFIED |
| Change is isolated and reviewable | VERIFIED (single schema.go change) |
| Reversal documented | VERIFIED (git revert) |

**Block 1 exit criteria: ALL MET.**

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/shared/settings/schema.go` | MODIFIED | C-6: removed S433 dry_run=false rejection for mainnet |
| `internal/shared/settings/s445_c6_controlled_removal_test.go` | NEW | 9 tests for C-6 and fail-closed guards |
| `internal/shared/settings/s433_mainnet_adapter_config_test.go` | MODIFIED | Updated test to expect validity |
| `internal/shared/settings/s436_mainnet_dryrun_config_test.go` | MODIFIED | Updated test to expect validity |
| `deploy/configs/execute-mainnet-live.jsonc` | NEW | Production live config for minimum scope |
| `docs/architecture/c6-controlled-dry-run-false-removal.md` | NEW | C-6 change documentation |
| `docs/architecture/live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md` | NEW | Scope guards and reversal plan |
| `docs/stages/stage-s445-c6-controlled-execution-report.md` | NEW | This report |

## Next Stage

**S446: Supervised Live Session Proof.**

Pre-condition: This report (S445) must be reviewed and accepted by the repository owner. The full pre-session checklist (PS-1 through PS-7) must pass before the live session begins.

## References

- [C-6 Controlled Removal](../architecture/c6-controlled-dry-run-false-removal.md)
- [Scope Guards and Reversal Plan](../architecture/live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md)
- [Enablement Ceremony Charter](../architecture/live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints](../architecture/live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [Live Trading Authorization Evidence Gate](../architecture/live-trading-authorization-evidence-gate.md) (S443)
