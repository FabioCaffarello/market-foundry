# Binance Spot/Futures Segmentation Foundation Wave — Charter and Scope Freeze

**Wave:** Binance Spot/Futures Segmentation Foundation
**Charter stage:** S390
**Date:** 2026-03-22
**Predecessor wave:** Testnet Venue Execution Proof (S389–S395, in progress)
**Authority:** This document freezes wave scope. Changes require a new stage.

---

## 1. Strategic Context

The Testnet Venue Execution Proof Wave (S389) opened with
`binance_futures_testnet` as the sole real venue adapter. The adapter, config
schema, credential model, and stream/KV naming all assume a single Binance
product: Futures.

Before proving execution against a real venue, the system must be able to
distinguish **Binance Spot** from **Binance Futures** as architecturally
separate segments with independent adapters, credentials, streams, and
configuration. Without this segmentation, any testnet proof locks the platform
into a single-product assumption that becomes expensive to unwind later.

This wave inserts a short, bounded segmentation phase between the S389 charter
and the first real venue execution stage. It does **not** execute trades, build
OMS features, or expand to other exchanges.

---

## 2. Problem Redefinition

### 2.1 Current State (Single-Product Assumption)

| Dimension | Current binding | Problem |
|---|---|---|
| VenueType enum | `paper_simulator`, `binance_futures_testnet` | No Spot type exists |
| Adapter implementation | `BinanceFuturesTestnetAdapter` only | Spot endpoint, auth, and response shape differ |
| Credential env vars | `MF_BINANCE_FUTURES_TESTNET_API_KEY/SECRET` | No naming convention for Spot credentials |
| NATS subjects | `execution.fill.venue_market_order.{source}.{symbol}.{tf}` | `source` is `"binancef"` — no Spot source value |
| Config schema | `venue.type` selects one adapter at startup | No segment concept, no enablement flag |
| Compose profile | Single execute binary instance | No Spot vs Futures binary segmentation |
| Activation surface | Reports single `AdapterState` | No per-segment activation visibility |

### 2.2 Target State (Segment-Aware)

| Dimension | Target |
|---|---|
| VenueType enum | `paper_simulator`, `binance_futures_testnet`, `binance_spot_testnet` |
| Adapter implementations | `BinanceFuturesTestnetAdapter` + `BinanceSpotTestnetAdapter` |
| Credential env vars | `MF_BINANCE_FUTURES_TESTNET_*` + `MF_BINANCE_SPOT_TESTNET_*` |
| NATS source values | `binancef` (Futures) + `binances` (Spot) |
| Config schema | `venue.type` extended, `venue.segment` explicit, validated |
| Compose profile | `execute-futures` + `execute-spot` service entries |
| Activation surface | Per-segment activation dimensions observable |

### 2.3 Architectural Approach: Multi-Binary per Segment

The system already proves multi-binary orchestration (S370–S375). Each segment
runs as an independent execute binary instance with its own:

- venue adapter (selected at startup via config);
- credential set (isolated env vars);
- NATS source value (distinct subject trees);
- control gate (independent halt/activate);
- activation surface (per-instance reporting).

This avoids introducing a venue routing layer inside a single binary, which
would violate the current single-symbol-per-instance model and add coupling
that is not warranted at this stage.

---

## 3. Wave Blocks (Ordered)

| Block | Stage | Title | Deliverable |
|---|---|---|---|
| B1 | S391 | Venue model refactor: exchange, market segment, environment | Domain types, source constants, config enum extension |
| B2 | S392 | Adapter boundary split: Binance Spot vs Binance Futures | `BinanceSpotTestnetAdapter` implementation + unit tests |
| B3 | S393 | Config-driven segment enablement with fail-closed semantics | Config schema extension, validation, compose wiring |
| B4 | S394 | Compose proof: segmented listening + dry-run on Spot and Futures paths | Compose-level E2E smoke with both segments |
| B5 | S395 | Evidence gate: Binance segmentation foundation | Matrix evaluation, residual gap registry, wave close |

### Block Details

#### B1 — Venue Model Refactor (S391)

Introduce canonical domain types that distinguish:

| Concept | Type | Values (this wave) |
|---|---|---|
| Exchange | `Exchange` | `binance` |
| Market segment | `MarketSegment` | `spot`, `futures` |
| Environment | `VenueEnvironment` | `testnet`, `mainnet` |
| Execution mode | (existing) | `paper`, `venue` |

The `VenueType` enum becomes a composite: `{exchange}_{segment}_{environment}`.

Extend the `source` field convention:
- `binancef` → Binance Futures (already in use)
- `binances` → Binance Spot (new)

Update `ExecutionIntent.Source` documentation to reflect the convention.

#### B2 — Adapter Boundary Split (S392)

Implement `BinanceSpotTestnetAdapter` satisfying the same `VenuePort` and
`VenueQueryPort` contracts as the Futures adapter, with:

| Difference | Futures | Spot |
|---|---|---|
| Base URL | `testnet.binancefuture.com/fapi/v1/order` | `testnet.binance.vision/api/v3/order` |
| Auth | HMAC-SHA256 (same scheme) | HMAC-SHA256 (same scheme) |
| Response shape | `fapi` response fields | `api` response fields |
| Order types | Market (this wave) | Market (this wave) |
| Source value | `binancef` | `binances` |

Share HMAC signing, HTTP plumbing, and error classification via internal
helpers — do not duplicate the full adapter. The decorator pipeline
(DryRunSubmitter → Post200Reconciler → RetrySubmitter) wraps both adapters
identically.

#### B3 — Config-Driven Segment Enablement (S393)

Extend `VenueConfig` schema:

```jsonc
{
  "venue": {
    "type": "binance_spot_testnet",   // new enum value
    "segment": "spot",                // explicit segment tag (derived from type, validated)
    "dry_run": true,                  // fail-closed default preserved
    "staleness_max_age": "120s",
    "submit_timeout": "10s"
  }
}
```

Validation rules:
- `type` must be in extended `knownVenueTypes`;
- `segment` must match `type` (e.g., `binance_spot_testnet` requires `segment: "spot"`);
- credential env vars must be present for the selected segment;
- `dry_run` defaults to `true` for all new types (fail-closed invariant preserved).

Compose wiring:
- `deploy/configs/execute-futures-testnet.jsonc` — Futures segment config;
- `deploy/configs/execute-spot-testnet.jsonc` — Spot segment config;
- `docker-compose.yml` gains `execute-futures` and `execute-spot` service entries;
- each service mounts its own config and credential env vars.

#### B4 — Compose Proof: Segmented Paths (S394)

End-to-end smoke script proving:

1. Both `execute-futures` and `execute-spot` services boot independently;
2. Each reports correct activation dimensions (segment, adapter state, credentials);
3. Dry-run intents flow through each segment's pipeline independently;
4. Fill/rejection events land on segment-specific NATS subjects;
5. KV projections are segment-isolated;
6. Control gate operates independently per segment.

Script: `scripts/smoke-segmented-spot-futures.sh`

#### B5 — Evidence Gate (S395)

Evaluate governing questions against collected evidence. Close wave or register
residual gaps for successor waves.

---

## 4. Governing Questions

| ID | Question | Target block |
|---|---|---|
| SEG-Q1 | Does the venue model cleanly separate exchange, market segment, and environment as orthogonal dimensions? | B1 |
| SEG-Q2 | Can a new market segment (e.g., Spot) be added without modifying existing adapter code? | B2 |
| SEG-Q3 | Does `BinanceSpotTestnetAdapter` satisfy `VenuePort` and `VenueQueryPort` with correct Spot-specific endpoint and response mapping? | B2 |
| SEG-Q4 | Does config validation reject invalid type/segment combinations at startup? | B3 |
| SEG-Q5 | Does `dry_run=true` remain the default for all new venue types (fail-closed preservation)? | B3 |
| SEG-Q6 | Can Spot and Futures execute binaries run concurrently in compose without stream/KV collision? | B4 |
| SEG-Q7 | Are activation dimensions segment-aware and independently observable? | B4 |
| SEG-Q8 | Does the control gate operate independently per segment instance? | B4 |
| SEG-Q9 | Is the credential isolation between segments enforced (no cross-segment credential leakage)? | B3, B4 |
| SEG-Q10 | Can the segmented architecture be extended to mainnet types without structural changes? | B5 |

---

## 5. What Enters (Scope Boundary)

1. Domain types for exchange, market segment, and venue environment.
2. `BinanceSpotTestnetAdapter` implementation and unit tests.
3. Config schema extension with segment-aware validation.
4. Compose wiring for concurrent Spot and Futures execute instances.
5. Segment-specific NATS source values and subject isolation.
6. Credential env var naming convention for Spot.
7. Activation surface extension for segment visibility.
8. Compose-level E2E smoke script for segmented paths.
9. Evidence gate evaluation.

---

## 6. What Does NOT Enter (Scope Freeze)

See companion document:
[`binance-segmentation-capabilities-questions-and-non-goals.md`](binance-segmentation-capabilities-questions-and-non-goals.md)

Summary of frozen exclusions:

| ID | Exclusion |
|---|---|
| NG-1 | Mainnet execution (testnet only) |
| NG-2 | Multi-exchange support (Binance only) |
| NG-3 | Full OMS (lifecycle, cancel, amend) |
| NG-4 | Portfolio risk management |
| NG-5 | Advanced order types (limit, stop-loss, OCO) |
| NG-6 | WebSocket fill streaming |
| NG-7 | Multi-symbol routing within a single binary |
| NG-8 | Real trading as wave focus |
| NG-9 | ClickHouse schema changes for segment dimension |
| NG-10 | Margin mode or leverage configuration |
| NG-11 | Cross-segment position management |
| NG-12 | Fee tier differentiation between Spot and Futures |
| NG-13 | Platform-wide redesign or actor topology changes |

---

## 7. Relationship to Testnet Venue Execution Proof Wave

The Testnet Venue Execution Proof Wave (S389–S395 original plan) is
**recalibrated**, not cancelled. The segmentation wave inserts between S389
(charter) and the first real venue execution stage:

| Original plan | Recalibrated plan |
|---|---|
| S389: Charter | S389: Charter (done) |
| S390: First real venue proof | **S390: Segmentation charter (this document)** |
| S391–S394: Execution stages | **S391–S394: Segmentation stages** |
| S395: Evidence gate | **S395: Segmentation evidence gate** |
| — | S396+: Resume testnet venue execution proof with segmented architecture |

The testnet venue governing questions (TV-Q1 through TV-Q12) remain valid and
will be answered in the resumed wave on top of the segmented architecture.

---

## 8. Success Criteria

The wave is complete when:

1. `BinanceSpotTestnetAdapter` exists and passes unit tests.
2. Config schema accepts `binance_spot_testnet` and `binance_futures_testnet`
   with segment-aware validation.
3. Compose profile runs both segments concurrently without collision.
4. Activation surface reports segment-specific dimensions.
5. Dry-run pipeline works on both segmented paths.
6. All 10 governing questions are answered at FULL or SUBSTANTIAL.
7. Evidence gate (S395) passes.

---

## 9. Dependencies and Preconditions

| Dependency | Status | Notes |
|---|---|---|
| Multi-binary orchestration (S370–S375) | Proven | Compose profile pattern reusable |
| Venue adapter port contracts | Stable | `VenuePort`, `VenueQueryPort` unchanged |
| Decorator pipeline | Stable | DryRunSubmitter, Post200Reconciler, RetrySubmitter |
| Activation surface (S339) | Stable | Extend, do not replace |
| Config validation framework | Stable | Extend `knownVenueTypes` |
| NATS subject convention | Stable | Add `binances` source, same pattern |

---

## 10. Risk Registry

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Binance Spot testnet API differs more than expected | Medium | Medium | Validate endpoint docs before B2; share HTTP/auth plumbing |
| Scope creep into margin/leverage | Low | High | NG-10 frozen; reject at review |
| Stream/KV collision between segments | Low | High | B4 smoke proves isolation |
| Credential cross-leak | Low | Critical | Env var naming convention enforced at config validation |
