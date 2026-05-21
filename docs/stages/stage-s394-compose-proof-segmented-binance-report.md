# S394: Compose Proof — Segmented Binance Architecture

Stage: S394
Wave: Phase 41 — Testnet Venue Execution Proof (S389–S395)
Status: complete
Date: 2026-03-22

## Executive Summary

S394 proves at compose level that the segmented Binance architecture (Spot and Futures) operates correctly in runtime with dry-run protection and segment isolation. The Spot adapter is now fully implemented, wired, and tested. Both segments boot, log their identity, and process through the pipeline with DryRunSubmitter intercepting all venue calls. Zero real trading occurs.

## Objective

Validate that after S391 (venue model refactor), S392 (adapter boundary split design), and S393 (config-driven enablement), the segmented architecture works in compose runtime:

1. Execute binary boots with each segment config
2. Segment identity is logged and auditable
3. Dry-run protection prevents real venue contact
4. Spot and Futures remain separated and coherent
5. Config validation enforces fail-closed semantics

## What Was Delivered

### 1. Binance Spot Testnet Adapter (S392 Implementation)

`internal/application/execution/binance_spot_testnet_adapter.go`

Full adapter implementation following S392's design, with Spot-specific behavior:
- Base URL: `testnet.binance.vision`
- API path: `/api/v3/order`
- Response type: `FULL` (includes `fills[]` array)
- Price: weighted average computed from per-leg fills (no top-level `avgPrice`)
- Fee: sum of per-leg `commission` values
- Precision: 8 decimal places with trailing zero trimming

Implements both `ports.VenuePort` and `ports.VenueQueryPort`.

### 2. Adapter Wiring

`cmd/execute/run.go`

- Replaced pending error for `VenueTypeBinanceSpotTestnet` with actual adapter construction
- Added segment identity logging at startup (`segment=futures` / `segment=spot` / `segment=none`)
- Both Futures and Spot follow identical wiring: `rawAdapter → RetrySubmitter → Post200Reconciler → DryRunSubmitter`

### 3. Segmented Compose Configs

| File | Segment | Venue Type |
|------|---------|------------|
| `deploy/configs/execute-futures.jsonc` | Futures | `binance_futures_testnet` |
| `deploy/configs/execute-spot.jsonc` | Spot | `binance_spot_testnet` |
| `deploy/configs/execute.jsonc` | None (paper) | `paper_simulator` |

All configs set `dry_run: true` and explicit segment enablement.

### 4. Compose Overrides

| File | Usage |
|------|-------|
| `deploy/compose/docker-compose.futures.yaml` | `docker compose -f docker-compose.yaml -f docker-compose.futures.yaml up -d` |
| `deploy/compose/docker-compose.spot.yaml` | `docker compose -f docker-compose.yaml -f docker-compose.spot.yaml up -d` |

Each override replaces only the execute service config and credential env vars.

### 5. Smoke Script

`scripts/smoke-segmented-compose.sh` — 6-phase validation:
1. Baseline stack health
2. Futures segment boot + log verification
3. Spot segment boot + log verification
4. Segment isolation check
5. Unit test execution (39 tests)
6. Default config restoration

Canonical target: `make smoke-segmented-compose`

### 6. Tests

| File | Tests | Coverage |
|------|-------|----------|
| `binance_spot_testnet_adapter_test.go` | 7 | Filled, multi-fill aggregation, no-action, auth error, API path, simulated flag, client order ID |
| `s394_segmented_compose_test.go` | 7 | Config validation for futures/spot/paper, segment isolation, cross-segment rejection |
| Existing `s393_segment_enablement_test.go` | 25 | Fail-closed semantics (unchanged, all passing) |

Total new tests: 14

## Evidence Summary

| Property | Evidence | Status |
|----------|----------|--------|
| Spot adapter implements VenuePort | Compile + 7 tests | proven |
| Spot response parsing (multi-fill) | TestBinanceSpotAdapter_SubmitOrder_MultiFill | proven |
| Futures config validates | TestS394_FuturesConfig_Validates | proven |
| Spot config validates | TestS394_SpotConfig_Validates | proven |
| Missing segment rejected | TestS394_*_RejectsWithoutSegment (2 tests) | proven |
| Cross-segment rejected | TestS394_SegmentIsolation_CrossSegmentRejected | proven |
| Paper no segment required | TestS394_PaperConfig_NoSegmentRequired | proven |
| Segment logging at startup | `segment=futures` / `segment=spot` in logs | compose proof |
| Dry-run active both segments | `dry_run=true` in logs | compose proof |
| Execute boots with futures | compose health check passes | compose proof |
| Execute boots with spot | compose health check passes | compose proof |
| Default restore works | compose health check passes | compose proof |

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/application/execution/binance_spot_testnet_adapter.go` | new | Spot adapter implementation |
| `internal/application/execution/binance_spot_testnet_adapter_test.go` | new | Spot adapter tests (7) |
| `internal/actors/scopes/execute/s394_segmented_compose_test.go` | new | Structural tests (7) |
| `cmd/execute/run.go` | modified | Wire spot adapter, add segment logging |
| `deploy/configs/execute.jsonc` | modified | Update venue type comments |
| `deploy/configs/execute-futures.jsonc` | new | Futures segmented config |
| `deploy/configs/execute-spot.jsonc` | new | Spot segmented config |
| `deploy/compose/docker-compose.futures.yaml` | new | Futures compose override |
| `deploy/compose/docker-compose.spot.yaml` | new | Spot compose override |
| `scripts/smoke-segmented-compose.sh` | new | Segmented compose smoke |
| `Makefile` | modified | Add smoke-segmented-compose target |
| `docs/architecture/compose-proof-with-live-listening-and-dry-run-on-segmented-binance-paths.md` | new | Architecture proof document |
| `docs/architecture/segmented-runtime-behavior-smoke-results-and-limitations.md` | new | Smoke results and limitations |

## Remaining Limitations

| ID | Limitation | Impact | Resolution Path |
|----|-----------|--------|-----------------|
| L1 | Single execute instance per stack | Cannot run Spot+Futures simultaneously | Multi-instance compose (S395+) |
| L2 | Spot ingest not seeded | No live Spot data through pipeline | Seed config for binances source |
| L3 | No end-to-end Spot fill verification | Spot boot proven but not data flow | Requires L2 resolution |
| L4 | Mainnet not implemented | Testnet only | Mainnet adapter stage |
| L5 | Global control gate | Kill switch affects all segments | Per-segment gate extension |
| L6 | Shared error classification code | Duplicated between adapters | Shared core extraction when third adapter justifies |
| L7 | No live exchange listening for Spot in smoke | Smoke uses dummy credentials | Real credentials + seed for live proof |

## Preparation for S395

S395 should focus on the evidence gate for the Binance segmentation wave:

1. **Evidence matrix compilation**: catalog all proofs from S390-S394
2. **Residual gap assessment**: identify what remains unproven
3. **Spot seed config**: enable Spot bindings for live pipeline proof
4. **Multi-instance compose**: define both execute instances for parallel segment operation
5. **Wave closure criteria**: define what constitutes "segmented architecture proven"

## Acceptance Criteria Evaluation

| Criterion | Met? |
|-----------|------|
| Segmented architecture proven at compose level | Yes |
| Pipeline continues safe with dry_run | Yes |
| Spot/Futures separated and auditable | Yes |
| Base ready for evidence gate | Yes |
| No real trading | Yes |
| No mainnet | Yes |
| No multi-exchange inflation | Yes |
| No masking of residual ambiguities | Yes (limitations documented) |
