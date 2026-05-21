# S402: Single-Compose Coexistence Proof Report

**Stage**: S402
**Wave**: Phase 42 — Unified Segment Runtime Foundation (S398--S403)
**Type**: Validation / proof
**Status**: Complete

## Objective

Prove that Binance Spot and Futures coexist in a single compose stack, sharing
the same binaries and NATS infrastructure, governed by one unified config, with
segment isolation intact and write-path protected by dry_run.

## What Changed

### New Files

| File | Purpose |
|---|---|
| `internal/actors/scopes/execute/s402_unified_coexistence_test.go` | 8 coexistence invariant tests |
| `scripts/smoke-unified-coexistence.sh` | 7-phase compose-level coexistence proof |
| `docs/architecture/single-compose-coexistence-proof-for-spot-and-futures.md` | Coexistence proof architecture doc |
| `docs/architecture/unified-runtime-compose-behavior-isolation-and-limitations.md` | Runtime behavior, isolation, and limitations |

### Modified Files

| File | Change |
|---|---|
| `Makefile` | Added `smoke-unified-coexistence` target and smoke-help entry |

### No Production Code Changes

S402 is a proof stage. The unified config, SegmentRouter, and segment isolation
mechanisms were all delivered in S399--S401. This stage validates that the
existing architecture works end-to-end in a unified compose environment.

## Evidence Matrix

### Unit Tests (8 tests, all pass)

| Test | Invariant proven |
|---|---|
| `TestS402_UnifiedConfigBothSegmentsCoexist` | Both segments enabled, dry_run=true, config valid |
| `TestS402_CoexistentRouterDispatchesBothSegments` | Router routes spot/futures to correct adapters |
| `TestS402_CoexistentRouterRejectsCrossSegmentLeak` | Unknown sources rejected with both segments registered |
| `TestS402_UnifiedConsumerCoversBothSegments` | NATS consumer filter includes both segment sources |
| `TestS402_DryRunWrapsCoexistentRouterUniformly` | DryRunSubmitter wraps the router, both segments go through dry-run |
| `TestS402_EnabledSegmentsCanonicalOrderStable` | EnabledSegments() returns deterministic order |
| `TestS402_CoexistentConfigAdaptersAreDistinct` | Spot and Futures adapters are different types |
| `TestS402_CoexistentConfigHasUnifiedSegments` | HasUnifiedSegments() returns true |

### Compose-Level Smoke (7 phases)

| Phase | What it proves |
|---|---|
| 1. Baseline | Stack running with default paper config |
| 2. Unit tests | S402 + S401 + SegmentRouter tests pass |
| 3. Unified boot | Execute boots with unified config, both segments healthy |
| 4. Coexistence | multi_segment type, segment_count=2 in logs |
| 5. Write-path protection | dry_run=true, no real venue activity |
| 6. Config validation | Fail-closed for invalid configs |
| 7. Restore | Default paper config restored |

### Regression Baseline

S402 does not modify production code. All existing S399--S401 tests continue
to pass, as does the full `go test ./...` suite.

## Acceptance Criteria

| Criterion | Status |
|---|---|
| Spot and Futures coexist in the same compose and same services | Met |
| The config governs correctly the runtime | Met |
| Write-path continues protected when necessary (dry_run) | Met |
| Stage proves the unified architectural design | Met |

## Guard Rails Compliance

| Guard rail | Status |
|---|---|
| No multi-exchange opened | Compliant — Binance only |
| No mainnet opened | Compliant — testnet adapters only |
| No benchmark inflation | Compliant — proportional smoke and tests |
| No compose/runtime conflict masking | Compliant — all checks explicit |

## Promoted Architecture Documents

- [`single-compose-coexistence-proof-for-spot-and-futures.md`](../architecture/single-compose-coexistence-proof-for-spot-and-futures.md)
- [`unified-runtime-compose-behavior-isolation-and-limitations.md`](../architecture/unified-runtime-compose-behavior-isolation-and-limitations.md)

## Limitations Documented

- No per-segment dry_run, kill switch, or staleness override.
- Metrics not segmented (aggregate across segments).
- QueryOrder sequential across segments.
- Mainnet not supported.
- Only Binance source-segment mapping exists.

## What Comes Next

S403 (Unified Segment Runtime Evidence Gate) will evaluate the complete
S398--S402 wave delivery against the charter's acceptance criteria.
