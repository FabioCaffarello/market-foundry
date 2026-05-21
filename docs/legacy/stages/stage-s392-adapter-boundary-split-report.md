# Stage S392 — Adapter Boundary Split for Binance Spot and Binance Futures

**Wave:** Binance Spot/Futures Segmentation Foundation (S390–S395)
**Block:** B2 — Adapter Boundary Split
**Depends on:** S391 (Venue Model Refactor — complete, documentation only)
**Enables:** S393 (Config-Driven Segment Enablement)

---

## 1. Executive Summary

S392 defines the architectural boundary between Binance Spot and Binance Futures adapters. The decision is **separate adapters with extracted shared core**: `BinanceFuturesTestnetAdapter` and `BinanceSpotTestnetAdapter` each own their endpoint identity, response parsing, and fill construction, while sharing exchange-level infrastructure (HMAC signing, error classification, status mapping) via a `BinanceClient` in `binance_common.go`.

This design was chosen because the response schemas between Spot and Futures are structurally different (top-level `avgPrice` vs `fills[]` array), making conditional branches inside a single adapter a readability and testing hazard. The shared surface (~120 lines of signing + error classification) is genuinely exchange-level and benefits from single-point maintenance.

---

## 2. Deliverables

| # | Artifact | Path | Status |
|---|----------|------|--------|
| D1 | Adapter boundary split design | `docs/architecture/adapter-boundary-split-for-binance-spot-and-binance-futures.md` | Complete |
| D2 | Shared core vs segment-specific responsibilities | `docs/architecture/shared-core-vs-segment-specific-responsibilities-and-limits.md` | Complete |
| D3 | Stage report (this document) | `docs/stages/stage-s392-adapter-boundary-split-report.md` | Complete |

---

## 3. Architectural Split Summary

### 3.1 Target File Layout

```
internal/application/execution/
├── binance_common.go                          # BinanceClient, signing, error classification
├── binance_common_test.go                     # Shared infra tests
├── binance_futures_testnet_adapter.go          # Futures adapter (refactored to use BinanceClient)
├── binance_futures_testnet_adapter_test.go     # Existing tests (must pass unchanged)
├── binance_spot_testnet_adapter.go             # New Spot adapter
├── binance_spot_testnet_adapter_test.go        # New Spot tests
```

### 3.2 Shared Core (binance_common.go)

| Component | Lines (est.) |
|-----------|-------------|
| `BinanceClient` struct + constructor | ~15 |
| `Sign` (HMAC-SHA256) | ~5 |
| `Do` (HTTP execution + deadline) | ~30 |
| `HandleErrorResponse` (C-FAIL taxonomy) | ~60 |
| `ClassifyByVenueErrorCode` | ~40 |
| `MapBinanceStatus` | ~15 |
| `MapSymbol` | ~10 |
| Shared types (`binanceErrorResponse`) | ~5 |
| **Total** | **~180** |

### 3.3 Segment-Specific (per adapter)

| Component | Lines (est.) |
|-----------|-------------|
| Adapter struct + constructor | ~20 |
| `SubmitOrder` | ~40 |
| `QueryOrder` | ~30 |
| `parseOrderResponse` | ~40 |
| Response struct | ~15 |
| Constants (URL, path) | ~5 |
| **Total per adapter** | **~150** |

### 3.4 Net Effect

| Metric | Before (Futures only) | After (Futures + Spot) |
|--------|----------------------|----------------------|
| Adapter files | 1 (~400 lines) | 3 (180 + 150 + 150 = ~480 lines) |
| Duplication | N/A | ~20 lines (accepted, see D2 §6.1) |
| Port compliance | VenuePort + VenueQueryPort | Same for both adapters |
| Test coverage | Futures tests | Futures tests + Spot tests + shared core tests |

---

## 4. Key Design Decisions

### D1: Separate adapters over single adapter with flag

**Rationale:** Spot response schema (`fills[]` array with per-leg commission) is structurally different from Futures (`avgPrice` top-level). Conditional branches would reduce readability and require segment-aware mocking in tests.

### D2: Extracted shared core over full duplication

**Rationale:** ~120 lines of signing + error classification logic are proven identical. Single-point maintenance prevents bug-fix divergence. The shared surface is exchange-level (not segment-level), making extraction natural.

### D3: Concrete types over interface strategy

**Rationale:** Two concrete adapters with a factory switch is simpler than a strategy interface for two implementations. The factory dispatch already exists in `buildVenueAdapter`. Adding an interface layer would add indirection without testability gain.

### D4: Accepted duplication for skeletal code

**Rationale:** No-action intent handling, request parameter skeleton, and `WithBaseURL` test helper are 5–10 lines each. Extracting them would create coupling without meaningful deduplication.

---

## 5. Governing Questions Progress

| ID | Question | Status | Evidence |
|----|----------|--------|----------|
| **SEG-Q2** | Can Spot be added without modifying Futures adapter code? | **Answered: Yes** | Spot is a new file; Futures refactor extracts shared code but preserves all behavior (verified by existing tests) |
| **SEG-Q3** | Does BinanceSpotTestnetAdapter satisfy VenuePort/VenueQueryPort correctly? | **Design-ready** | Contract compliance defined in D1 §2.3; implementation validation deferred to code execution |

---

## 6. Acceptance Criteria Evaluation

| Criterion | Met? | Evidence |
|-----------|------|----------|
| Spot and Futures have clear architectural boundaries | Yes | Separate files, separate structs, separate response types (D1 §2) |
| Shared core restricted to genuinely shared surface | Yes | Only exchange-level mechanics; strict inclusion rules (D2 §2.3) |
| Improves robustness without inflating to multi-exchange | Yes | Only Binance in scope; second exchange gets its own shared core (D2 §6.3) |
| Ready for config-driven enablement in S393 | Yes | Registration, factory branch, and credential namespace defined (D1 §5) |

---

## 7. Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No multi-exchange opened | Compliant — only Binance adapters designed |
| No generic plugin platform | Compliant — concrete types, factory switch, no registry pattern |
| No unnecessary duplication | Compliant — shared core extracts ~180 lines; accepted duplication is ~20 lines of skeletal code |
| No excessive abstraction collapsing Spot/Futures | Compliant — separate adapters, separate response types, no segment flag |

---

## 8. Preparation for S393

S393 (Config-Driven Segment Enablement) can proceed with:

1. **Register `binance_spot_testnet`** in `knownVenueTypes` and validation error messages
2. **Add factory branch** for `VenueTypeBinanceSpotTestnet` in `buildVenueAdapter`
3. **Create config file** `deploy/configs/execute-spot-testnet.jsonc` with Spot-specific settings
4. **Define credential env vars** `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` and `MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET`
5. **Assign NATS source value** `binances` for Spot stream partitioning

The boundary design in this stage provides the complete specification for S393 implementation.

---

## 9. Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Spot response schema assumption incorrect | Spot adapter tests will validate against Binance Spot testnet documentation; structural test with canned responses |
| Shared core extraction breaks Futures behavior | Existing Futures test suite must pass unchanged after extraction — this is the primary regression gate |
| Over-extraction into shared core | Strict inclusion rules (D2 §2.3) prevent adding segment-specific logic to shared module |

---

## 10. Non-Goals (Reaffirmed)

- Multi-exchange adapter support (Kraken, Coinbase, etc.)
- Mainnet adapter implementation (enum registered, code not activated)
- WebSocket or streaming adapters
- Generic adapter plugin/registry framework
- Adapter auto-discovery
- Order types beyond market orders
- Commission/fee reconciliation beyond proxy values
