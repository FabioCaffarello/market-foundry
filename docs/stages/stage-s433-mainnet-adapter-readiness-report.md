# Stage S433 -- Mainnet Adapter Readiness Report

## Stage Identity

| Field | Value |
|---|---|
| Stage | S433 |
| Type | Implementation (blocker resolution) |
| Wave | Mainnet Enablement (Phase 49) |
| Charter | S432 |
| Resolves | B-1 (no mainnet adapter implementation) |
| Predecessor | S432 (Mainnet Enablement Wave -- charter and scope freeze) |
| Date | 2026-03-23 |

## Executive Summary

S433 delivers mainnet adapter readiness for both Binance Spot and Futures segments. Two mainnet adapter types (`binance_spot_mainnet`, `binance_futures_mainnet`) are registered, config-validated, boot-wired, rate-limited, and tested. A config-level enforcement prevents `dry_run=false` with any mainnet adapter, guaranteeing that no real orders can be submitted without a future authorization ceremony.

**Blocker B-1 is resolved.** The architecture can now target mainnet endpoints under dry-run control.

## Capability Delivery

| ID | Capability | Evidence | Status |
|---|---|---|---|
| C-1 | Spot mainnet adapter implemented and interface-compliant | `binance_spot_mainnet_adapter.go`; `TestBinanceSpotMainnetAdapter_VenuePortInterface`; `TestBinanceSpotMainnetAdapter_BaseURL` | **DELIVERED** |
| C-2 | Futures mainnet adapter implemented and interface-compliant | `binance_futures_mainnet_adapter.go`; `TestBinanceFuturesMainnetAdapter_VenuePortInterface`; `TestBinanceFuturesMainnetAdapter_BaseURL` | **DELIVERED** |
| C-3 | Rate-limiter decorator integrated in mainnet adapter call chain | `rate_limiter.go`; `TestRateLimiter_PassesThrough`; `TestRateLimiter_RespectsContextCancellation` | **DELIVERED** |
| C-4 | Config-driven adapter selection supports mainnet variants | `schema.go` VenueType constants + knownVenueTypes + adapterSegmentCompatibility; `TestMainnetAdapter_KnownVenueType` | **DELIVERED** |

All 4 chartered capabilities delivered.

## Governing Question Answers

| ID | Question | Answer | Evidence |
|---|---|---|---|
| GQ-1 | Can mainnet adapters be instantiated from the same VenuePort interface without modifying the execution pipeline? | **YES** -- type aliases reuse the entire testnet adapter implementation; only base URL differs | Interface compliance tests pass; `buildVenueAdapterByType` adds 2 cases with no pipeline changes |
| GQ-2 | What are the concrete differences between testnet and mainnet Binance endpoints? | Base URL only (REST). Auth scheme, API paths, response schema are identical. Rate limits differ (mainnet stricter). | Documented in `mainnet-endpoint-selection-contracts-guards-and-limitations.md` Section 4 |
| GQ-3 | Is a token-bucket rate limiter sufficient for Binance mainnet API limits? | **YES** for single-symbol market-order scope. 10 burst / ~10 req/s steady state is well within Binance's 1200 weight/min (Spot) and 2400 weight/min (Futures) | `rate_limiter.go` implementation; Binance rate limit documentation |
| GQ-4 | Does config-driven adapter selection support mainnet variants without adding new config keys? | **YES** -- the existing `venue.segments.*.adapter` field accepts the new mainnet types. No new config keys introduced. | `TestMainnetAdapter_DualSegment_Valid`; config examples in arch doc |

## Implementation Details

### New Files

| File | Purpose | Lines |
|---|---|---|
| `internal/application/execution/binance_spot_mainnet_adapter.go` | Spot mainnet adapter (type alias + base URL) | 31 |
| `internal/application/execution/binance_futures_mainnet_adapter.go` | Futures mainnet adapter (type alias + base URL) | 31 |
| `internal/application/execution/rate_limiter.go` | Token-bucket VenuePort decorator | 91 |
| `internal/application/execution/s433_mainnet_adapter_readiness_test.go` | 10 adapter/rate-limiter/credential tests | ~240 |
| `internal/shared/settings/s433_mainnet_adapter_config_test.go` | 10 config validation tests | ~170 |

### Modified Files

| File | Change |
|---|---|
| `internal/shared/settings/schema.go` | +2 VenueType constants, +2 knownVenueTypes entries, +2 adapterSegmentCompatibility entries, +Environment/IsMainnet methods, +hasMainnetAdapter helper, +mainnet dry_run validation |
| `cmd/execute/run.go` | +2 cases in buildVenueAdapterByType for mainnet adapters with rate limiter wiring |

### Design Decisions

1. **Type alias over separate struct**: Mainnet adapters are `type X = TestnetX` aliases. This eliminates code duplication while maintaining type identity for documentation and config. All request/response/error logic is shared.

2. **Rate limiter in adapter chain**: The rate limiter sits between the raw adapter and RetrySubmitter. This means retries also consume tokens, which is the correct behavior -- a retry against a rate-limited venue should still respect the rate limit.

3. **Config-level dry_run enforcement**: Rather than a runtime guard, mainnet dry_run=true is enforced at config validation. This means the binary refuses to start with an invalid config, providing the strongest possible guarantee.

## Test Results

### New Tests (20 total)

**Config tests (10):**
- `TestVenueType_Segment_MainnetAdapters` (5 subtests)
- `TestVenueType_Environment` (5 subtests)
- `TestVenueType_IsMainnet`
- `TestMainnetAdapter_KnownVenueType`
- `TestMainnetAdapter_SegmentCompatibility`
- `TestMainnetAdapter_DryRunEnforcement`
- `TestMainnetAdapter_DryRunTrue_Valid`
- `TestMainnetAdapter_DryRunOmitted_Valid`
- `TestMainnetAdapter_DualSegment_Valid`
- `TestMainnetAdapter_MixedTestnetMainnet_Valid`
- `TestTestnetAdapter_DryRunFalse_StillValid` (regression guard)

**Adapter tests (10):**
- `TestBinanceSpotMainnetAdapter_BaseURL`
- `TestBinanceSpotMainnetAdapter_VenuePortInterface`
- `TestBinanceFuturesMainnetAdapter_BaseURL`
- `TestBinanceFuturesMainnetAdapter_VenuePortInterface`
- `TestRateLimiter_PassesThrough`
- `TestRateLimiter_RespectsContextCancellation`
- `TestMainnetCredentialLoading_Spot`
- `TestMainnetCredentialLoading_Futures`
- `TestMainnetCredentialLoading_FailClosed`
- `TestMainnetBaseURLs`

### Regression Verification

| Package | Result | Duration |
|---|---|---|
| `internal/shared/settings` | **PASS** | 0.14s |
| `internal/application/execution` | **PASS** | 32.0s |
| `internal/actors/scopes/execute` | **PASS** | 1.4s |
| `internal/domain/execution` | **PASS** | 0.4s |
| `internal/adapters/clickhouse/writerpipeline` | **PASS** | 0.6s |
| `internal/adapters/nats/natsexecution` | **PASS** | 0.7s |
| `cmd/execute` (build) | **BUILDS CLEAN** | -- |

**Zero regressions. All packages pass. Execute binary builds clean.**

## Non-Goal Compliance

| NG | Non-Goal | Compliance |
|---|---|---|
| NG-1 | Live trading on mainnet | **COMPLIANT** -- dry_run enforcement prevents real orders |
| NG-2 | OMS expansion | **COMPLIANT** -- market-order-only scope unchanged |
| NG-3 | Multi-exchange | **COMPLIANT** -- Binance only |
| NG-6 | Config surface re-expansion | **COMPLIANT** -- no new config keys; existing `adapter` field accepts new types |
| NG-7 | Large structural refactoring | **COMPLIANT** -- type aliases, no pipeline changes |
| NG-11 | Non-blocker resolution | **COMPLIANT** -- NB-1 (rate limiter) partially addressed by C-3; no other NB touched |

## Blocker Status

| ID | Blocker | Pre-S433 | Post-S433 |
|---|---|---|---|
| B-1 | No mainnet adapter implementation | **OPEN** (Critical) | **RESOLVED** |
| B-2 | No mainnet credential management | OPEN (Critical) | OPEN -- target S434 |
| B-3 | No ClickHouse backup/restore strategy | OPEN (High) | OPEN -- target S435 |

## Deliverables

| Artifact | Path | Status |
|---|---|---|
| Mainnet adapter readiness doc | [`mainnet-adapter-readiness-spot-and-futures.md`](../architecture/mainnet-adapter-readiness-spot-and-futures.md) | Delivered |
| Endpoint selection, contracts, guards | [`mainnet-endpoint-selection-contracts-guards-and-limitations.md`](../architecture/mainnet-endpoint-selection-contracts-guards-and-limitations.md) | Delivered |
| Stage report | This document | Delivered |

## Preparation for S434

The next stage (S434: Mainnet Secret Manager Integration) should:

1. Read the current `credentials.go` implementation (`LoadCredentials`, `CredentialSet`).
2. Design a `CredentialProvider` interface that abstracts credential retrieval.
3. Implement at least one concrete provider (Vault, file-based, or equivalent).
4. Wire mainnet adapters to use `CredentialProvider` instead of raw env vars.
5. Prove fail-closed behavior: adapter refuses to start if credentials are unavailable.
6. Retain env-var provider as fallback for testnet/development use.
