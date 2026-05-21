# Venue Segmentation Semantics: Valid/Invalid Configurations and Fail-Closed Rules

**Stage:** S391
**Date:** 2026-03-22
**Wave:** Binance Spot/Futures Segmentation Foundation (S390–S395)
**Companion to:** [venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md](venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md)
**Authority:** This document defines combination validity rules. Changes require a new stage.

---

## 1. Purpose

This document enumerates all valid and invalid combinations of the four venue
dimensions (exchange, market segment, environment, execution mode), defines
fail-closed semantics for ambiguous or missing values, and establishes the
invariants that config validation must enforce.

---

## 2. Dimension Value Sets

### 2.1 Exchange

| Value | Status | Description |
|---|---|---|
| `binance` | Active | Binance exchange (Spot + Futures) |
| (empty) | Valid only for `paper_simulator` | No external exchange |

### 2.2 Market Segment

| Value | Status | Description |
|---|---|---|
| `spot` | Active (S392) | Spot market — direct asset exchange |
| `futures` | Active | USD-M Futures — perpetual/delivery contracts |
| (empty) | Valid only for `paper_simulator` | No segment |

### 2.3 Environment

| Value | Status | Description |
|---|---|---|
| `testnet` | Active | Sandbox/test infrastructure, no real funds |
| `mainnet` | Defined, NOT activated | Production infrastructure, real funds |
| (empty) | Valid only for `paper_simulator` | No environment |

### 2.4 Execution Mode (Derived)

| Mode | Meaning | Real venue interaction? |
|---|---|---|
| `paper` | Paper simulator, no venue | No |
| `venue_halted` | Venue adapter loaded, gate closed | No |
| `venue_degraded` | Venue adapter loaded, credentials absent | No |
| `venue_live` | Venue adapter loaded, gate open, credentials present | **Yes** |

Execution mode is derived from `ActivationSurface`, not from config. The
`dry_run` config flag adds an additional interception layer: when `dry_run=true`
(the default), the `DryRunSubmitter` intercepts before any venue call, even in
`venue_live` mode.

---

## 3. Valid Combinations

### 3.1 Complete Validity Matrix

| # | VenueType | Exchange | Segment | Environment | dry_run | Status |
|---|---|---|---|---|---|---|
| 1 | `paper_simulator` | — | — | — | true (forced) | **Active** |
| 2 | `paper_simulator` | — | — | — | false | **REJECTED** |
| 3 | `binance_futures_testnet` | binance | futures | testnet | true | **Active** |
| 4 | `binance_futures_testnet` | binance | futures | testnet | false | **Active** (live execution) |
| 5 | `binance_spot_testnet` | binance | spot | testnet | true | **Active** (after S392) |
| 6 | `binance_spot_testnet` | binance | spot | testnet | false | **Active** (after S392, live execution) |
| 7 | `binance_futures_mainnet` | binance | futures | mainnet | true | **Registered, NOT activated** |
| 8 | `binance_futures_mainnet` | binance | futures | mainnet | false | **Registered, NOT activated** |
| 9 | `binance_spot_mainnet` | binance | spot | mainnet | true | **Registered, NOT activated** |
| 10 | `binance_spot_mainnet` | binance | spot | mainnet | false | **Registered, NOT activated** |

### 3.2 Active Combinations (This Wave)

Only rows 1, 3, 4, 5, and 6 are active. A combination is active when:

1. The VenueType is in the `knownVenueTypes` registry; **AND**
2. An adapter implementation exists for it; **AND**
3. Config validation passes all dimension-level checks.

### 3.3 Mainnet Combinations (Future, Gated)

Rows 7–10 are **structurally valid** (the naming convention and dimension
decomposition support them) but are NOT activated. Activation requires:

1. A new stage with activation gate ceremony;
2. Fund-safety review (credential governance, risk limits, kill-switch proof);
3. Registration in `knownVenueTypes`;
4. Adapter implementation.

---

## 4. Invalid Combinations

### 4.1 Structurally Invalid

These combinations are logically contradictory and must be rejected at config
validation time.

| # | Attempted config | Rejection reason |
|---|---|---|
| I1 | `paper_simulator` + `dry_run=false` | Paper is inherently simulated; dry_run=false is meaningless |
| I2 | Exchange set without segment | Incomplete identity — cannot select adapter |
| I3 | Segment set without exchange | Incomplete identity — cannot select adapter |
| I4 | Exchange set without environment | Incomplete identity — cannot determine infra tier |
| I5 | Unknown exchange value | Fail-closed: unrecognized exchange rejected |
| I6 | Unknown segment value | Fail-closed: unrecognized segment rejected |
| I7 | Unknown environment value | Fail-closed: unrecognized environment rejected |
| I8 | Valid dimensions but unregistered combination | e.g., `bybit_spot_testnet` — no adapter exists |

### 4.2 Cross-Segment Invalid

| # | Attempted config | Rejection reason |
|---|---|---|
| C1 | Two segments in same binary | Each binary serves exactly one segment (multi-binary model) |
| C2 | Spot credentials used with Futures adapter | Credential namespace is segment-specific |
| C3 | Testnet credentials on mainnet endpoint | Environment mismatch — credential isolation violation |

### 4.3 Semantically Dangerous (Rejected by Default)

| # | Scenario | Rule |
|---|---|---|
| D1 | `mainnet` + `dry_run=false` without activation ceremony | Mainnet live execution requires explicit gate |
| D2 | Any unregistered VenueType | Default case in `buildVenueAdapter` returns error |
| D3 | Missing or empty `venue.type` | Defaults to `paper_simulator` (fail-safe) |

---

## 5. Fail-Closed Semantics

### 5.1 Core Principle

**When in doubt, do not execute.** Every ambiguous, missing, or unrecognized
configuration value must resolve to the safest possible state: either paper
simulation or startup rejection.

### 5.2 Fail-Closed Rules

| Rule | Trigger | Behavior |
|---|---|---|
| **FC-1:** Default venue type | `venue.type` omitted or empty | Default to `paper_simulator` |
| **FC-2:** Default dry_run | `venue.dry_run` omitted or null | Default to `true` (dry-run active) |
| **FC-3:** Unknown venue type | `venue.type` not in `knownVenueTypes` | **Reject at startup** — binary does not start |
| **FC-4:** Missing credentials | Required env vars absent for non-paper venue | **Reject at startup** — binary does not start |
| **FC-5:** Kill switch default | No gate state in KV | Default to `GateActive` (existing behavior, acceptable because dry_run defaults to true) |
| **FC-6:** Dimension parse failure | VenueType string does not decompose into valid dimensions | **Reject at startup** |
| **FC-7:** Mainnet without ceremony | Any mainnet VenueType without activation gate | **Reject at startup** (type not in knownVenueTypes) |

### 5.3 Fail-Closed Hierarchy

The system has multiple independent safety layers that compose:

```
Layer 1: Config validation       → rejects invalid type/dimension combos at startup
Layer 2: Credential loading      → rejects missing credentials at startup
Layer 3: DryRunSubmitter         → intercepts all venue calls when dry_run=true
Layer 4: SafetyGate / kill switch → blocks execution when gate is halted
Layer 5: Staleness guard         → rejects stale intents
Layer 6: Activation surface      → reports effective mode for observability
```

No single layer is responsible for all safety. The composition of all layers
ensures that **no real venue order can be submitted unless ALL of the following
are true:**

1. VenueType is registered and has an adapter implementation (config validation)
2. Credentials are present in environment (credential loading)
3. `dry_run=false` is explicitly set (DryRunSubmitter bypass)
4. Control gate is active (SafetyGate check)
5. Intent is not stale (Staleness guard)
6. Adapter is `venue` (not paper) — ActivationSurface reports `venue_live`

### 5.4 Fail-Closed Truth Table

| venue.type | dry_run | credentials | gate | Result |
|---|---|---|---|---|
| (empty) | (any) | (any) | (any) | Paper simulator |
| `paper_simulator` | true | (any) | (any) | Paper simulator |
| `paper_simulator` | false | (any) | (any) | **STARTUP REJECT** |
| `binance_futures_testnet` | true | present | active | Dry-run on Futures adapter |
| `binance_futures_testnet` | true | present | halted | Dry-run on Futures adapter (DryRun intercepts before gate) |
| `binance_futures_testnet` | false | absent | (any) | **STARTUP REJECT** |
| `binance_futures_testnet` | false | present | halted | Venue halted — no execution |
| `binance_futures_testnet` | false | present | active | **VENUE LIVE** — real Futures orders |
| `binance_spot_testnet` | true | present | active | Dry-run on Spot adapter |
| `binance_spot_testnet` | false | present | active | **VENUE LIVE** — real Spot orders |
| `binance_futures_mainnet` | (any) | (any) | (any) | **STARTUP REJECT** (not registered) |
| `unknown_value` | (any) | (any) | (any) | **STARTUP REJECT** |

---

## 6. Invariants

### 6.1 Identity Invariants

| ID | Invariant | Enforcement point |
|---|---|---|
| **INV-1** | A non-paper VenueType decomposes into exactly three non-empty dimensions | `VenueConfig.Validate()` |
| **INV-2** | Each dimension value belongs to its known set | `VenueConfig.Validate()` |
| **INV-3** | The composed VenueType is in `knownVenueTypes` | `VenueConfig.Validate()` |
| **INV-4** | Paper has no exchange, segment, or environment | `VenueIdentity.IsPaper()` |

### 6.2 Isolation Invariants

| ID | Invariant | Enforcement point |
|---|---|---|
| **INV-5** | Each binary serves exactly one segment | Config: one `venue.type` per config file |
| **INV-6** | Credentials are namespaced by full identity | `LoadCredentials(venueType, ...)` |
| **INV-7** | NATS source values are distinct per segment | Source constant map: `spot→binances`, `futures→binancef` |
| **INV-8** | Testnet credentials never reach mainnet endpoints | VenueType encodes environment; adapter selects base URL from it |

### 6.3 Safety Invariants

| ID | Invariant | Enforcement point |
|---|---|---|
| **INV-9** | `dry_run` defaults to true when omitted | `IsDryRun()` nil-check |
| **INV-10** | `dry_run=false` with paper is rejected | `VenueConfig.Validate()` |
| **INV-11** | Unregistered types fail at startup, not at first order | `buildVenueAdapter()` default case |
| **INV-12** | Missing credentials fail at startup, not at first order | `LoadCredentials()` returns Problem |
| **INV-13** | Mainnet types are not in `knownVenueTypes` until activation ceremony | Registry in `schema.go` |

---

## 7. Segment-Specific Differences

### 7.1 Binance Spot vs Binance Futures

| Aspect | Spot | Futures |
|---|---|---|
| API base URL (testnet) | `https://testnet.binance.vision` | `https://testnet.binancefuture.com` |
| Order endpoint | `/api/v3/order` | `/fapi/v1/order` |
| Query endpoint | `/api/v3/order` (GET) | `/fapi/v1/order` (GET) |
| Symbol format | Uppercase (`BTCUSDT`) | Uppercase (`BTCUSDT`) |
| Order types (this wave) | MARKET only | MARKET only |
| Signing scheme | HMAC-SHA256 | HMAC-SHA256 |
| Response shape | Different field names for fills | Different field names for fills |
| NATS source | `binances` | `binancef` |
| Credential env prefix | `MF_VENUE_BINANCE_SPOT_TESTNET_` | `MF_VENUE_BINANCE_FUTURES_TESTNET_` |

### 7.2 Shared Infrastructure (Extractable to Common)

| Aspect | Shared? |
|---|---|
| HMAC-SHA256 signing | Yes — same algorithm, same key structure |
| Timestamp generation | Yes — millisecond Unix timestamp |
| HTTP client configuration | Yes — timeout, TLS, headers |
| Error classification (8 failure classes) | Yes — same HTTP status code mapping |
| Symbol normalization (lowercase→uppercase) | Yes — same convention for Binance |
| Request parameter encoding | Yes — query string with signature |

S392 will extract these to `internal/application/execution/binance_common.go`
while keeping adapters independent.

---

## 8. Interaction With Multi-Binary Orchestration

The venue segmentation model relies on the multi-binary orchestration proven
in S370–S375:

| Concern | How it maps |
|---|---|
| Per-segment binary | Each Compose service entry uses a different config file with a different `venue.type` |
| Stream isolation | Different source values produce different NATS subjects; no stream collision |
| KV isolation | Each binary reports its own `ActivationDimensions` to a unique key |
| Gate independence | Each binary reads the gate from the same KV bucket but can be halted independently (future: per-segment gate keys) |
| Failure isolation | One segment crashing does not affect the other's binary |

---

## 9. Limitations and Deferred Decisions

| Item | Status | Deferred to |
|---|---|---|
| Per-segment control gate keys | Global gate (`"global"` key) applies to all segments | Future stage when per-segment halt is needed |
| Mainnet activation ceremony | Mainnet types structurally defined but not registered | Future wave with fund-safety review |
| Multi-exchange support | Only `binance` exchange defined | Future wave |
| Config-level explicit dimensions | Config still uses flat `type` string; decomposition is internal | S393 may add explicit fields if needed |
| Advanced order types | Only MARKET orders in both Spot and Futures | Future wave |
| WebSocket streaming | REST-only for both segments | Future wave |
