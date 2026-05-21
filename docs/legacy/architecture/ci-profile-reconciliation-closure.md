# CI Profile Reconciliation Closure

**Date:** 2026-03-20
**Stage:** S229
**Status:** CLOSED — quality-gate-ci reconciled with current architecture

---

## 1. Problem Statement

After the S218–S224 restructuring tranche and the S223–S227 closure tranche,
the `ci` profile of `quality-gate` diverged from the actual repository state.

`make check` (fast profile) passed because it treats warnings as non-fatal.
`make quality-gate-ci` (ci profile) failed because it promotes warnings to
errors — and several analyzers were generating warnings based on assumptions
from a previous architectural state.

S228 recorded 40 errors across four steps: topology-doctor, contract-audit,
arch-guard, drift-detect.

---

## 2. Root Cause Analysis

The divergence had a single root cause: **raccoon-cli analyzers encoded
structural assumptions that were not updated during the S218–S224 restructuring.**

The restructuring changed:
- configctl control subject naming from `configctl.control.config.*` to `configctl.control.*`
- domain event surface from configctl-only to six domains (observation, signal, decision, strategy, execution, risk)
- request/reply type naming to include version segments (`v1`)
- writer service introduction with legitimate "consumer" usage in `cmd/writer/`

The analyzers were not updated to reflect these changes.

---

## 3. Reconciliation Applied

Six targeted fixes in five files:

### 3.1 topology.rs — subject prefix broadening
```
EXPECTED_SUBJECT_PREFIXES: "configctl.control.config" → "configctl.control."
```
Matches the flat control subject namespace introduced during restructuring.

### 3.2 contracts.rs — reply-type symmetry
New `control_operation_suffix()` helper strips `_request`/`_reply` suffixes
before comparing request and reply type names, making the check version-aware.

### 3.3 contracts/events.rs — multi-domain event scanning
- Added `domain` field to `DomainEventDef`
- Replaced hardcoded `internal/domain/configctl/events.go` scan with
  dynamic `internal/domain/*/events.go` scan
- Each event now carries its domain attribution

### 3.4 contracts.rs — event-registry alignment rewrite
Replaced rigid suffix-based matching with domain-aware tokenized matching:
- Groups domain events by domain
- Matches registry event subjects to their domain
- Uses tokenized subsequence matching to tolerate naming variations
- Preserves the alignment check without false positives

### 3.5 drift_detect.rs — defunct name correction
- Removed "consumer" from `DEFUNCT_NAMES` array
- Simplified `scan_stale_references` to only check genuinely defunct names
- "consumer" is legitimate in the current writer service topology

### 3.6 runtime_bindings/source.rs — documentation alignment
Updated doc comment and test to reflect current `configctl.control.*` pattern.

---

## 4. Validation Evidence

After applying the reconciliation:

```
make check        → PASS (6 passed, 0 failed, 1 skipped | 84 checks)
make quality-gate-ci → PASS (6 passed, 0 failed, 1 skipped | 84 checks)
cargo test (raccoon-cli) → 97 passed, 0 failed
```

All three gates now agree: the repository is structurally sound.

---

## 5. Profile Convergence

| Check | fast (check) | ci (quality-gate-ci) | Status |
|-------|:---:|:---:|:---:|
| doctor | PASS | PASS | converged |
| topology-doctor | PASS | PASS | converged |
| contract-audit | PASS | PASS | converged |
| runtime-bindings | PASS | PASS | converged |
| arch-guard | PASS | PASS | converged |
| drift-detect | PASS | PASS | converged |
| runtime-smoke | SKIP | SKIP | n/a (deep only) |

The fast and ci profiles now produce identical verdicts.

---

## 6. What Was NOT Changed

1. No guard rail was relaxed or removed
2. No new check was added
3. No profile behavior was changed (ci still promotes warnings to errors)
4. No architectural refactoring was performed
5. No documentation drift was addressed (deferred to S230)
6. No CI pipeline configuration was modified
