# Unified Runtime Smoke and Futures Preflight Proof

> S419 — Consolidated runtime validation and Futures Venue Execution Proof readiness gate.

## Purpose

This document captures the evidence that the post-S416/S417/S418 consolidated runtime surface is intact and ready to serve as the foundation for the Futures Venue Execution Proof Wave.

S416 consolidated the execute config surface from 6 files to 3 canonical configs. S417 consolidated the compose surface from 5 overlays to 3 canonical files. S418 removed transitional artifacts and corrected taxonomy labels. This stage (S419) exercises the consolidated surface end-to-end without a running stack, verifying that no regressions were introduced and that all Futures preconditions are satisfied.

## Evidence Summary

### 1. Build Integrity

All 8 binaries compile without errors:

| Binary | Status |
|--------|--------|
| configctl | PASS |
| derive | PASS |
| execute | PASS |
| gateway | PASS |
| ingest | PASS |
| migrate | PASS |
| store | PASS |
| writer | PASS |

### 2. Config Surface Coherence

**Canonical configs (3):**

| File | Mode | Segments | dry_run | Status |
|------|------|----------|---------|--------|
| `execute.jsonc` | Standalone (paper_simulator) | None | true (default) | Valid |
| `execute-unified.jsonc` | Segments-based | Spot + Futures | true | Valid |
| `execute-venue-live.jsonc` | Segments-based | Spot + Futures | false | Valid |

**Deprecated configs removed (4):**
- `execute-spot.jsonc` — subsumed by unified config with spot-only enablement
- `execute-futures.jsonc` — subsumed by unified config with futures-only enablement
- `execute-venue-live-spot.jsonc` — subsumed by venue-live config
- `execute-venue-live-futures.jsonc` — subsumed by venue-live config

### 3. Compose Surface Validity

**Canonical compose files (3):**

| File | Purpose | Validates |
|------|---------|-----------|
| `docker-compose.yaml` | Base topology (9 services) | PASS |
| `docker-compose.unified.yaml` | Overlay for segmented dry-run | PASS |
| `docker-compose.venue-live.yaml` | Overlay for real testnet execution | PASS |

**Deprecated compose overlays removed (4):**
- `docker-compose.spot.yaml`
- `docker-compose.futures.yaml`
- `docker-compose.unified-spot-live.yaml`
- `docker-compose.unified-futures-live.yaml`

### 4. Deprecated Reference Scan

Zero references to deprecated config or compose file names found in:
- `scripts/` (24 smoke scripts)
- `cmd/` (8 binary entrypoints)
- `internal/` (all application code)
- `deploy/` (configs + compose)

### 5. Taxonomy Verification

Zero stale `"legacy"` labels in Go code. S418 corrected 8 occurrences across 5 files, replacing the misleading "legacy" label with "standalone" for the Type-based config mode.

### 6. Test Suite Integrity

| Suite | Tests | Status |
|-------|-------|--------|
| S419 consolidated runtime preflight | 13 | PASS |
| S416 config consolidation | 8 | PASS |
| S401 segment isolation | n | PASS |
| S419 unified compose E2E Futures | 8 | PASS |
| S416-S418 Futures venue lifecycle | n | PASS |
| Full settings suite | 40+ | PASS |

### 7. Futures Preflight Readiness

| Precondition | Evidence | Status |
|-------------|----------|--------|
| Futures segment enabled in unified config | `execute-unified.jsonc` has `futures.enabled: true` | Ready |
| Futures segment enabled in venue-live config | `execute-venue-live.jsonc` has `futures.enabled: true` | Ready |
| Futures adapter implementation | `binance_futures_testnet_adapter.go` exists | Ready |
| SegmentRouter dispatches binancef | `SegmentForSource("binancef") == MarketSegmentFutures` | Proven |
| Compose overlays declare Futures credentials | Both overlays have `MF_VENUE_BINANCE_FUTURES_TESTNET_*` | Ready |
| Futures E2E smoke script | `smoke-e2e-unified-futures.sh` exists and executable | Ready |
| Futures venue acceptance/fill tests | S416 tests pass | Proven |
| Futures rejection/audit tests | S417-S418 tests pass | Proven |
| Source-to-segment mapping bijective | `binances` <-> Spot, `binancef` <-> Futures | Proven |
| Fail-closed validation | Adapter/segment mismatch rejected, no-enabled-segments rejected | Proven |

## Canonical Entrypoint

```bash
make smoke-runtime-preflight
```

Stackless — no compose infrastructure required. Exercises build, config, compose validation, deprecated reference scan, taxonomy check, and the full test suite.

## Scope Boundary

This proof does NOT:
- Start compose infrastructure
- Execute real orders against any exchange
- Run endurance/soak tests
- Open the Futures Venue Execution Proof wave

It exclusively validates that the consolidated surface is coherent and that the preconditions for the Futures wave are satisfied.

## Verdict

The consolidated runtime is **READY** for the Futures Venue Execution Proof Wave. No regressions were introduced by S416-S418, and all Futures preconditions are met.
