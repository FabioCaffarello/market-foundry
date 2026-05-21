# Venue Model Refactor: Exchange, Market Segment, Environment, and Execution Mode

**Stage:** S391
**Date:** 2026-03-22
**Wave:** Binance Spot/Futures Segmentation Foundation (S390–S395)
**Answers:** SEG-Q1 (venue model orthogonality)
**Authority:** This document defines the canonical venue model. Changes require a new stage.

---

## 1. Problem Statement

The current venue model uses a flat `VenueType` enum that encodes multiple
orthogonal dimensions into a single string:

```go
// Current (monolithic)
VenueTypePaperSimulator        VenueType = "paper_simulator"
VenueTypeBinanceFuturesTestnet VenueType = "binance_futures_testnet"
```

`binance_futures_testnet` collapses three independent concepts into one token:
**exchange** (binance), **market segment** (futures), and **environment**
(testnet). This makes it impossible to add Spot without introducing another
monolithic string (`binance_spot_testnet`) that duplicates the same encoding
pattern. It also prevents config validation from reasoning about dimensions
independently.

### 1.1 Specific Limitations

| Limitation | Impact |
|---|---|
| No explicit exchange dimension | Cannot validate exchange-level invariants (e.g., credential namespace) |
| No explicit market segment | Cannot distinguish Spot from Futures at the type level |
| No explicit environment | Cannot validate testnet-only restrictions independently |
| VenueType conflates adapter selection with identity | Adding a new combination requires a new enum value plus wiring everywhere |
| Credential env vars embed the monolithic name | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` — fragile naming convention |
| NATS source value convention is implicit | `binancef` encodes segment but has no formal mapping to the model |

---

## 2. Canonical Venue Model — Four Dimensions

The venue identity is decomposed into four **orthogonal, explicit** dimensions.
Each dimension has a bounded set of valid values. The composition of all four
dimensions defines the **effective venue identity** at runtime.

### 2.1 Dimension Definitions

#### Exchange

The external trading platform. Represents the counterparty infrastructure.

```
Type:     Exchange
Values:   "binance"
Future:   "bybit", "okx", etc. (NOT in scope for this wave)
```

Exchange determines:
- credential namespace (`MF_VENUE_{EXCHANGE}_{SEGMENT}_{ENVIRONMENT}_*`)
- API base URL family
- request signing scheme
- error classification taxonomy

#### Market Segment

The product class within an exchange. Each segment has independent API
endpoints, order models, and risk characteristics.

```
Type:     MarketSegment
Values:   "spot", "futures"
Future:   "margin", "options" (NOT in scope)
```

Market segment determines:
- API endpoint path (`/api/v3/order` vs `/fapi/v1/order`)
- response schema shape
- symbol normalization rules (same for Binance Spot/Futures today, may diverge)
- NATS source value convention (`binances`, `binancef`)

#### Environment

The infrastructure tier. Determines whether the venue interaction reaches
production or sandbox infrastructure.

```
Type:     VenueEnvironment
Values:   "testnet", "mainnet"
```

Environment determines:
- API base URL (`testnet.binance.vision` vs `api.binance.com`)
- credential isolation boundary (testnet keys never reach mainnet)
- risk classification (testnet = safe for experimentation, mainnet = fund exposure)

#### Execution Mode

How the pipeline processes execution intents. This dimension is **already
modeled** by the existing `ActivationSurface` (adapter + gate + credentials)
and the `DryRun` config flag. It is NOT part of the venue identity itself but
interacts with it.

```
Type:     (existing EffectiveMode)
Values:   "paper", "venue_halted", "venue_live", "venue_degraded"
```

Execution mode determines:
- whether intents reach the venue adapter at all
- whether the DryRunSubmitter intercepts
- whether fills are simulated or real

### 2.2 Composition: Venue Identity vs Execution Mode

```
┌─────────────────────────────────────────┐
│           Venue Identity                │
│  ┌──────────┐ ┌────────┐ ┌───────────┐ │
│  │ Exchange │ │Segment │ │Environment│ │
│  │ binance  │ │futures │ │ testnet   │ │
│  └──────────┘ └────────┘ └───────────┘ │
└─────────────────────────────────────────┘
                    ×
┌─────────────────────────────────────────┐
│          Execution Mode                 │
│  ┌─────────┐ ┌──────┐ ┌────────────┐   │
│  │ Adapter │ │ Gate │ │ Credential │   │
│  │  venue  │ │active│ │  present   │   │
│  └─────────┘ └──────┘ └────────────┘   │
│  → EffectiveMode: venue_live            │
└─────────────────────────────────────────┘
```

**Key insight:** Venue identity (exchange × segment × environment) selects
*which adapter* is instantiated. Execution mode governs *whether that adapter
is allowed to execute*. These are independent concerns.

### 2.3 Backward-Compatible VenueType Derivation

The flat `VenueType` string remains as the **serialization key** in config
files and the adapter registry. It is now a **derived value** from the three
identity dimensions, not a primary concept:

```
VenueType = "{exchange}_{segment}_{environment}"
```

| Exchange | Segment | Environment | Derived VenueType |
|---|---|---|---|
| (none) | (none) | (none) | `paper_simulator` |
| binance | futures | testnet | `binance_futures_testnet` |
| binance | spot | testnet | `binance_spot_testnet` |
| binance | futures | mainnet | `binance_futures_mainnet` (future) |
| binance | spot | mainnet | `binance_spot_mainnet` (future) |

`paper_simulator` is a special case: it has no exchange, segment, or
environment — it is the null venue identity.

---

## 3. Domain Types

### 3.1 Type Definitions (Target)

```go
// Exchange represents the external trading platform.
type Exchange string

const (
    ExchangeBinance Exchange = "binance"
)

// MarketSegment represents the product class within an exchange.
type MarketSegment string

const (
    SegmentSpot    MarketSegment = "spot"
    SegmentFutures MarketSegment = "futures"
)

// VenueEnvironment represents the infrastructure tier.
type VenueEnvironment string

const (
    EnvironmentTestnet VenueEnvironment = "testnet"
    EnvironmentMainnet VenueEnvironment = "mainnet"
)

// VenueIdentity is the canonical decomposition of a venue type into its
// orthogonal dimensions. Paper simulator has zero-value identity.
type VenueIdentity struct {
    Exchange    Exchange         `json:"exchange,omitempty"`
    Segment     MarketSegment    `json:"segment,omitempty"`
    Environment VenueEnvironment `json:"environment,omitempty"`
}

// IsPaper reports whether this identity represents the paper simulator.
func (v VenueIdentity) IsPaper() bool {
    return v.Exchange == "" && v.Segment == "" && v.Environment == ""
}

// VenueType returns the flat string representation for config compatibility.
func (v VenueIdentity) VenueType() VenueType {
    if v.IsPaper() {
        return VenueTypePaperSimulator
    }
    return VenueType(string(v.Exchange) + "_" + string(v.Segment) + "_" + string(v.Environment))
}

// SourceValue returns the NATS source value for subject routing.
func (v VenueIdentity) SourceValue() string {
    if v.IsPaper() {
        return "" // paper uses derive's source, not venue's
    }
    return sourceValues[v.Segment]
}
```

### 3.2 Source Value Convention

| Segment | Source value | Subject example |
|---|---|---|
| futures | `binancef` | `execution.fill.venue_market_order.binancef.btcusdt.60` |
| spot | `binances` | `execution.fill.venue_market_order.binances.btcusdt.60` |

Source values encode **segment only** (not exchange or environment) because:
- within a single deployment, there is exactly one exchange;
- testnet vs mainnet is an infra concern, not a routing concern;
- source values partition NATS subject trees for stream isolation.

### 3.3 Credential Namespace Convention

```
MF_VENUE_{EXCHANGE}_{SEGMENT}_{ENVIRONMENT}_{CREDENTIAL}
```

| Identity | API Key env var |
|---|---|
| binance + futures + testnet | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` |
| binance + spot + testnet | `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` |
| binance + futures + mainnet | `MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY` |

This convention is **already in use** (via `LoadCredentials(string(config.Venue.Type), ...)`).
The refactored model formalizes it as a derivable rule rather than a hardcoded string.

---

## 4. Config Schema Evolution

### 4.1 Current Config

```jsonc
{
    "venue": {
        "type": "paper_simulator",
        "dry_run": true,
        "staleness_max_age": "120s",
        "submit_timeout": "10s"
    }
}
```

### 4.2 Target Config (Backward Compatible)

```jsonc
{
    "venue": {
        "type": "binance_futures_testnet",
        "dry_run": true,
        "staleness_max_age": "120s",
        "submit_timeout": "10s"
    }
}
```

The `type` field remains the serialization key. The model internally parses it
into `VenueIdentity` dimensions for validation. This preserves backward
compatibility with existing config files and deployment scripts.

**No new config fields are required by S391.** The semantic decomposition is
internal to the domain model. S393 will extend the schema if needed for
explicit segment enablement flags.

### 4.3 Validation Rules (Extended)

The `VenueConfig.Validate()` method gains dimension-aware checks:

1. Parse `type` into `VenueIdentity` (exchange, segment, environment)
2. Validate each dimension is a known value
3. Validate the combination is in the allowed set (see S391 companion doc)
4. Existing rules (staleness, timeout, dry_run) remain unchanged

---

## 5. Adapter Registry Mapping

Each `VenueType` maps to exactly one adapter implementation. The mapping is
exhaustive and fail-closed:

| VenueType | Adapter | Query Port | Credentials |
|---|---|---|---|
| `paper_simulator` | `PaperVenueAdapter` | nil | none (CredentialAbsent) |
| `binance_futures_testnet` | `BinanceFuturesTestnetAdapter` | yes | API_KEY, API_SECRET |
| `binance_spot_testnet` | `BinanceSpotTestnetAdapter` (S392) | yes | API_KEY, API_SECRET |
| unknown | **reject at startup** | — | — |

---

## 6. Interaction With Existing Activation Surface

The `ActivationSurface` model remains unchanged. It answers "is this binary
allowed to execute right now?" — a question orthogonal to venue identity.

With the refactored model, `ActivationDimensions` can optionally carry the
`VenueIdentity` for per-segment observability:

```go
type ActivationDimensions struct {
    Adapter     AdapterState    `json:"adapter"`
    Credentials CredentialState `json:"credentials"`
    Venue       VenueIdentity   `json:"venue"`       // NEW: identity for observability
    ReportedAt  time.Time       `json:"reported_at"`
    ReportedBy  string          `json:"reported_by"`
}
```

This allows the store query responder to report which segment each execute
binary serves, enabling per-segment dashboards.

---

## 7. Ownership and Authority

| Aspect | Owner |
|---|---|
| `VenueIdentity` type and constants | `internal/domain/execution/` |
| `VenueConfig` validation with dimensions | `internal/shared/settings/` |
| Adapter registry and instantiation | `cmd/execute/run.go` |
| Credential loading convention | `internal/application/execution/credentials.go` |
| Source value convention | `internal/domain/execution/` (constants) |
| NATS subject routing | `internal/adapters/nats/natsexecution/` |

---

## 8. Limitations

| Limitation | Rationale |
|---|---|
| Only `binance` exchange defined | Multi-exchange is a future wave; segmentation proves the model within one exchange first |
| Only `testnet` environment active | Mainnet activation requires fund-safety ceremonies not in this wave |
| Source values encode segment only | Sufficient for current single-exchange, single-environment deployments |
| `paper_simulator` has no identity dimensions | Paper is inherently venue-less; forcing dimensions would add complexity without value |
| Config still uses flat `type` string | Backward compatibility; internal decomposition provides the semantic benefit |
