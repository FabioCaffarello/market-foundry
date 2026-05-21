# Futures Venue Execution Proof -- Capabilities, Questions, and Non-Goals

**Wave:** Futures Venue Execution Proof
**Charter stage:** S415
**Date frozen:** 2026-03-23

---

## 1. Governing Questions

### 1.1 Acceptance and Fill (S416)

| ID | Question | Target |
|---|---|---|
| **FV-Q1** | Does the `venue_live` write-path produce correct lifecycle transitions when the Futures venue accepts and fills a real order? | S416 |
| **FV-Q2** | Do fill records from a real Futures testnet response carry accurate `avgPrice`, `executedQty`, `cumQuote`, and commission data? | S416 |

### 1.2 Rejection (S417)

| ID | Question | Target |
|---|---|---|
| **FV-Q3** | Does the lifecycle correctly transition to `rejected` when the Futures venue returns a real rejection (insufficient margin, invalid quantity, invalid symbol, auth failure)? | S417 |
| **FV-Q4** | Does the `VenueOrderRejectedEvent` carry the real Futures venue error code, reason, and HTTP status? | S417 |

### 1.3 Partial Fill (S417)

| ID | Question | Target |
|---|---|---|
| **FV-Q5** | Can the system observe or structurally prove the `partially_filled` transition from a real Futures venue response? | S417 |
| **FV-Q6** | Does quantity monotonicity hold when real partial fills arrive from the Futures venue? | S417 |

### 1.4 Persistence and Read-Path (S418)

| ID | Question | Target |
|---|---|---|
| **FV-Q7** | Do KV projections, HTTP queries, and ClickHouse rows agree on terminal state after real Futures venue interactions? | S418 |
| **FV-Q8** | Does the ClickHouse rejection writer (wired in S411) produce correct rows for Futures rejection events without code changes? | S418 |

### 1.5 Compose and Sustained Operation (S418, S419)

| ID | Question | Target |
|---|---|---|
| **FV-Q9** | Does the full compose pipeline (derive -> execute -> store) operate correctly in `venue_live` mode against the Futures testnet on the unified runtime? | S419 |
| **FV-Q10** | Does the system sustain correct behavior over multiple consecutive Futures order cycles? | S418 |

### 1.6 Cross-Cutting (S416)

| ID | Question | Target |
|---|---|---|
| **FV-Q11** | Does the correlation chain (CorrelationID + CausationID) remain intact through real Futures venue interactions? | S416 |
| **FV-Q12** | Does the post-200 reconciliation path recover correctly when body-read fails after a real Futures 200? | S416 |

---

## 2. Capability Map

Each capability links to a governing question and a target stage.

| ID | Capability | Question | Stage | Predecessor Evidence |
|---|---|---|---|---|
| **FV-C1** | Real Futures venue acceptance lifecycle | FV-Q1 | S416 | S385: venue_live write-path; S389: Futures adapter |
| **FV-C2** | Real Futures venue fill record fidelity | FV-Q2 | S416 | S384: PriceSource; Futures adapter unit tests |
| **FV-C3** | Real Futures venue rejection lifecycle | FV-Q3 | S417 | S386: rejection event path; adapter error mapping |
| **FV-C4** | Real Futures venue rejection event fidelity | FV-Q4 | S417 | S386: VenueOrderRejectedEvent contract |
| **FV-C5** | Real Futures venue partial fill lifecycle | FV-Q5, FV-Q6 | S417 | S383: partially_filled at domain level |
| **FV-C6** | Lifecycle invariant fidelity under real Futures data | FV-Q1--FV-Q6 | S416--S417 | S383: 49/49 transitions; S384: 8 invariant categories |
| **FV-C7** | Persistence consistency under real Futures data | FV-Q7, FV-Q8 | S418 | S387: KV + HTTP; S411: ClickHouse rejection writer |
| **FV-C8** | Read-path auditability and segment parity | FV-Q7 | S418 | S413: lifecycle list, composite status |
| **FV-C9** | Compose E2E with real Futures testnet | FV-Q9, FV-Q10 | S419 | S408: unified compose E2E Spot; S402: coexistence proof |
| **FV-C10** | Segment isolation under dual-segment live execution | FV-Q9 | S419 | S401: segment isolation; S408: compose E2E |

---

## 3. Non-Goals

These items are explicitly excluded from the wave. Non-goals from prior waves
are preserved and extended with Futures-specific exclusions.

### 3.1 Venue and Market Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-1** | Mainnet execution | Testnet only. Mainnet requires separate activation ceremony. |
| **NG-2** | Multi-venue support | Only Binance testnet. No second exchange, no venue routing beyond Binance. |
| **NG-3** | Multi-symbol execution | Single symbol per segment instance. No symbol multiplexing. |
| **NG-4** | Advanced order types | Market orders only. No limit, stop-loss, stop-market, OCO, trailing stop. |
| **NG-5** | WebSocket streaming fills | Synchronous REST-based fill retrieval only. |

### 3.2 OMS and Lifecycle Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-6** | Full OMS | No order book, no amendment, no cross-symbol aggregation. |
| **NG-7** | Cancel-order API | `cancelled` status mapped but no cancel HTTP call. |
| **NG-8** | Order amendment or replace | Fire-and-forget model. |
| **NG-9** | Lifecycle state machine extension | Seven-state machine is frozen. |
| **NG-10** | Multi-fill accumulation from WebSocket | REST-only fill observation. |

### 3.3 Risk, Portfolio, and Strategy Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-11** | Portfolio risk management | No position tracking, exposure limits, or margin management. |
| **NG-12** | P&L computation | No profit/loss. Fee realism deferred. |
| **NG-13** | Strategy optimization | Strategies produce intents; this wave validates execution. |

### 3.4 Infrastructure and Operations Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-14** | Dashboard or UI | No operational dashboards, no web UI. |
| **NG-15** | Alerting or paging | Observability via structured logs and NATS streams. |
| **NG-16** | CI/CD pipeline for testnet | Manual execution. |
| **NG-17** | Performance benchmarking | Correctness, not throughput. |
| **NG-18** | Credential rotation or vault integration | Env vars only. |

### 3.5 Architectural Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-19** | Lifecycle redesign | State machine proven and frozen. |
| **NG-20** | Actor topology changes | No new actors or supervision restructuring. |
| **NG-21** | New NATS streams or KV buckets | No new infrastructure beyond what exists. |
| **NG-22** | Domain model extension | `ExecutionIntent` is frozen. |

### 3.6 Segmentation Scope (preserved from S404)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-23** | Re-open segmentation wave | Segmentation is closed (S395 PASS). No venue model, adapter boundary, or config redesign. |
| **NG-24** | Per-segment control gate | Global gate is conservative and sufficient. |
| **NG-25** | Shared core extraction | ~120 lines duplication between adapters is acceptable. Extract when third adapter justifies. |
| **NG-26** | Activation surface segment queryability | Startup logging is sufficient. API/KV exposure is observability enhancement. |
| **NG-27** | Multi-exchange adapters | No second exchange. |

### 3.7 Runtime Scope (preserved from S404)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-28** | Runtime architecture redesign | Unified runtime is stable. This wave consumes it, not modifies it. |
| **NG-29** | Per-segment dry_run control | Global `dry_run` flag applies uniformly. |
| **NG-30** | Per-segment kill switch | Global kill switch sufficient. |
| **NG-31** | Concurrent `venue_live` on both segments simultaneously | Proving Futures `venue_live` with Spot in `dry_run` or disabled is sufficient. Simultaneous dual-segment `venue_live` is a follow-on concern. |
| **NG-32** | Schema changes to ClickHouse | 20-column schema is segment-agnostic. No DDL changes. |

### 3.8 Futures-Specific Exclusions (NEW)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-33** | Leverage configuration | Default leverage applies. No leverage adjustment API calls. |
| **NG-34** | Position mode switching | Default one-way mode. No hedge mode. |
| **NG-35** | Margin type management | Default cross margin. No margin type switching API calls. |
| **NG-36** | Funding rate impact analysis | Funding rates exist but have no bearing on order lifecycle proof. |
| **NG-37** | Liquidation handling | Liquidation events are venue-initiated, not order-lifecycle events. Out of scope. |
| **NG-38** | Mark price or index price tracking | Not relevant to order lifecycle proof. |
| **NG-39** | Multi-asset margin mode | Default single-asset mode. No multi-asset margin. |
| **NG-40** | Income/trade history API | Execution proof uses order API only. No `/fapi/v1/income` or `/fapi/v1/userTrades`. |

---

## 4. Boundary Conditions

### 4.1 Futures Testnet Credential Requirements

- `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`
  must be provisioned before S416.
- Credentials are never committed to the repository.
- Futures testnet account must have sufficient USDT balance for market order execution.

### 4.2 Futures Testnet Account State

- Account leverage: default (20x for most symbols). No modification in this wave.
- Account margin type: cross margin (default). No modification in this wave.
- Account position mode: one-way (default). No modification in this wave.
- Sufficient USDT balance must be verified before each execution stage.

### 4.3 Partial Fill Feasibility (Futures)

Futures market orders may produce partial fills when order size exceeds
available liquidity at the best price level. This is more likely on Futures
testnet than Spot testnet. The wave accepts:

- **Preferred:** Real partial fill observation from Futures testnet.
- **Acceptable:** Structural proof combining domain-level evidence (S383) with
  Futures adapter response parsing tests and `mapBinanceStatus("PARTIALLY_FILLED")`
  mapping demonstration, if real partial fills cannot be reliably triggered.

### 4.4 Post-200 Reconciliation (Futures)

Same policy as S404: structural proof (code inspection + existing httptest
coverage via `Post200Reconciler`) is acceptable if the scenario cannot be
reliably triggered against the Futures testnet.

---

## 5. Relationship to Spot Proof Wave (S404--S409)

This wave mirrors the Spot proof wave structurally but targets the Futures
segment. The key differences are:

| Dimension | Spot Proof (S404--S409) | Futures Proof (S415--S420) |
|---|---|---|
| Venue target | Binance Spot testnet | Binance Futures testnet |
| Adapter | `BinanceSpotTestnetAdapter` | `BinanceFuturesTestnetAdapter` |
| Response model | `fills[]` array | Top-level `avgPrice`/`cumQuote` |
| Partial fill likelihood | Low (market fills atomically) | Medium (testnet liquidity gaps) |
| Margin relevance | None | Rejection source (insufficient margin) |
| Non-goals | 35 (NG-1--NG-35) | 40 (NG-1--NG-40) |
| Governing questions | 12 (TV-Q series) | 12 (FV-Q series, parallel structure) |
| Capabilities | 10 (TV-C series) | 10 (FV-C series, parallel structure) |
| New: segment parity | Not applicable (single segment proven) | Required (dual-segment coexistence) |

---

## 6. Links

- Charter: [`futures-venue-execution-proof-wave-charter-and-scope-freeze.md`](futures-venue-execution-proof-wave-charter-and-scope-freeze.md)
- Spot proof charter (S404): [`testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md`](testnet-venue-execution-proof-wave-charter-unified-runtime-spot-first.md)
- Spot proof evidence gate (S409): [`../stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md`](../stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md)
- Production hardening gate (S414): [`../stages/stage-s414-production-readiness-hardening-evidence-gate-report.md`](../stages/stage-s414-production-readiness-hardening-evidence-gate-report.md)
- Production hardening evidence matrix: [`production-readiness-hardening-evidence-matrix-residual-gaps-and-next-ceremony.md`](production-readiness-hardening-evidence-matrix-residual-gaps-and-next-ceremony.md)
- Canonical order model: [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md)
- Stage report: [`../stages/stage-s415-futures-venue-execution-proof-charter-report.md`](../stages/stage-s415-futures-venue-execution-proof-charter-report.md)
