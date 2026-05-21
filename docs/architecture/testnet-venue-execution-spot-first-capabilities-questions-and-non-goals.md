# Testnet Venue Execution Proof (Spot-First) -- Capabilities, Questions, and Non-Goals

**Wave:** Testnet Venue Execution Proof (refreshed)
**Charter refresh stage:** S396
**Original charter:** S389
**Date frozen:** 2026-03-22

---

## 1. Governing Questions (Revised Targeting)

The 12 governing questions are preserved from the S389 charter. The only change
is the target stage, reflecting the Spot-first sequencing and the consumed
S390--S395 stage numbers.

### 1.1 Acceptance and Fill (S399 -- Spot)

| ID | Question | Target |
|---|---|---|
| **TV-Q1** | Does the `venue_live` write-path produce correct lifecycle transitions when the Spot venue accepts and fills a real order? | S399 |
| **TV-Q2** | Do fill records from a real Spot testnet response carry accurate prices, quantities, and fees? | S399 |

### 1.2 Rejection (S399 -- Spot)

| ID | Question | Target |
|---|---|---|
| **TV-Q3** | Does the lifecycle correctly transition to `rejected` when the Spot venue returns a real rejection (insufficient balance, invalid params, auth failure)? | S399 |
| **TV-Q4** | Does the `VenueOrderRejectedEvent` carry the real Spot venue rejection code, reason, and HTTP status? | S399 |

### 1.3 Partial Fill (S400 -- Spot)

| ID | Question | Target |
|---|---|---|
| **TV-Q5** | Can the system observe or structurally prove the `partially_filled` transition from a real Spot venue response? | S400 |
| **TV-Q6** | Does quantity monotonicity hold when real partial fills arrive from the Spot venue? | S400 |

### 1.4 Persistence and Read-Path (S400 -- Spot)

| ID | Question | Target |
|---|---|---|
| **TV-Q7** | Do KV projections, HTTP queries, and ClickHouse rows agree on terminal state after real Spot venue interactions? | S400 |
| **TV-Q8** | Is the ClickHouse rejection writer wired and producing correct rows (RG-1 closure)? | S400 |

### 1.5 Compose and Sustained Operation (S398 partial, S400 full)

| ID | Question | Target |
|---|---|---|
| **TV-Q9** | Does the full compose pipeline (derive -> execute -> store) operate correctly in `venue_live` mode against the Spot testnet? | S398 (infra), S400 (full) |
| **TV-Q10** | Does the system sustain correct behavior over multiple consecutive order cycles against the Spot testnet? | S400 |

### 1.6 Cross-Cutting (S399 -- Spot)

| ID | Question | Target |
|---|---|---|
| **TV-Q11** | Does the correlation chain (CorrelationID + CausationID) remain intact through real Spot venue interactions? | S399 |
| **TV-Q12** | Does the post-200 reconciliation path recover correctly when body-read fails after a real Spot 200? | S399 |

---

## 2. Capability Map (Revised)

Each capability links to a governing question and a target stage in the
Spot-first sequencing.

| ID | Capability | Question | Stage | Predecessor evidence |
|---|---|---|---|---|
| **TV-C1** | Real Spot venue acceptance lifecycle | TV-Q1 | S399 | S385: venue_live write-path; S394: Spot adapter |
| **TV-C2** | Real Spot venue fill record fidelity | TV-Q2 | S399 | S384: PriceSource; S394: Spot multi-fill aggregation |
| **TV-C3** | Real Spot venue rejection lifecycle | TV-Q3 | S399 | S386: rejection event path; S394: Spot auth error mapping |
| **TV-C4** | Real Spot venue rejection event fidelity | TV-Q4 | S399 | S386: VenueOrderRejectedEvent contract |
| **TV-C5** | Real Spot venue partial fill lifecycle | TV-Q5, TV-Q6 | S400 | S383: partially_filled at domain; S394: Spot multi-fill path |
| **TV-C6** | Lifecycle invariant fidelity under real Spot data | TV-Q1--TV-Q6 | S399--S400 | S383: 49/49 transitions; S384: 8 invariant categories |
| **TV-C7** | Persistence consistency under real Spot data | TV-Q7, TV-Q8 | S400 | S387: KV + HTTP; RG-1 open |
| **TV-C8** | Post-200 reconciliation under real Spot conditions | TV-Q12 | S399 | S322: Post200Reconciler |
| **TV-C9** | Compose E2E with real Spot testnet | TV-Q9, TV-Q10 | S398 (infra), S400 (full) | S380: E2E dry-run; S394: segmented compose |
| **TV-C10** | OMS read-path auditability under real Spot data | TV-Q7 | S400 | S387: composite status query |

---

## 3. Non-Goals

These items are explicitly excluded from the refreshed wave. The 22 original
non-goals from S389 are preserved. Additional non-goals are added to reflect
the post-segmentation context.

### 3.1 Venue and Market Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-1** | Mainnet execution | Testnet only. Mainnet requires separate activation ceremony. |
| **NG-2** | Multi-venue support | Only Binance testnet. No second exchange, no venue routing. |
| **NG-3** | Multi-symbol execution | Single symbol per binary instance. No symbol multiplexing. |
| **NG-4** | Advanced order types | Market orders only. No limit, stop-loss, OCO. |
| **NG-5** | WebSocket streaming fills | Synchronous REST-based fill retrieval only. |

### 3.2 OMS and Lifecycle Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-6** | Full OMS | No order book, no amendment, no cross-symbol aggregation. |
| **NG-7** | Cancel-order API | `cancelled` status mapped but no cancel HTTP call. |
| **NG-8** | Order amendment or replace | Fire-and-forget model. |
| **NG-9** | Lifecycle state machine extension | Seven-state machine is frozen. |
| **NG-10** | Multi-fill accumulation from WebSocket | REST-only partial fill observation. |

### 3.3 Risk, Portfolio, and Strategy Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-11** | Portfolio risk management | No position tracking, exposure limits, or margin management. |
| **NG-12** | P&L computation | No profit/loss. Fee realism deferred. |
| **NG-13** | Strategy optimization | Strategies produce intents; this wave validates execution. |

### 3.4 Infrastructure and Operations Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-14** | Dashboard or UI | No operational dashboards, no web UI. |
| **NG-15** | Alerting or paging | Observability via structured logs and NATS streams. |
| **NG-16** | CI/CD pipeline for testnet | Manual execution. |
| **NG-17** | Performance benchmarking | Correctness, not throughput. |
| **NG-18** | Credential rotation or vault integration | Env vars only. |

### 3.5 Architectural Scope (preserved from S389)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-19** | Lifecycle redesign | State machine proven and frozen. |
| **NG-20** | Actor topology changes | No new actors or supervision changes beyond RG-1. |
| **NG-21** | New NATS streams or KV buckets | No new infra beyond RG-1 wiring. |
| **NG-22** | Domain model extension | `ExecutionIntent` is frozen. |

### 3.6 Segmentation Scope (NEW -- post-S395)

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-23** | Parallel Futures testnet proof | Futures proof is deferred to a follow-on wave. This wave answers TV-Q1--TV-Q12 for Spot only. |
| **NG-24** | Re-open segmentation wave | Segmentation is closed (S395 PASS). No venue model, adapter boundary, or config enablement redesign. |
| **NG-25** | Per-segment control gate | S395 G2: global gate is conservative and sufficient. Per-segment gate is operational refinement. |
| **NG-26** | Shared core extraction | S395 G5: ~120 lines duplication is acceptable. Extract when third adapter justifies. |
| **NG-27** | Activation surface segment queryability | S395 G4: startup logging is sufficient. API/KV exposure is observability enhancement. |
| **NG-28** | Multi-exchange adapters | No second exchange. Segmentation was Binance-internal (Spot vs. Futures). |

---

## 4. Boundary Conditions (Revised)

### 4.1 Spot Testnet Credential Requirements

- `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` and `MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET`
  must be provisioned before S399.
- Credentials are never committed to the repository.
- Spot testnet account must have sufficient test balance for market order execution.

### 4.2 Spot Ingest Seed Prerequisite

- `binances` source bindings must be seeded in NATS before S399.
- S397 closes this prerequisite.

### 4.3 Partial Fill Feasibility (Spot)

Spot market orders typically fill instantly on testnet. The wave accepts:

- **Preferred:** Real partial fill observation from Spot testnet.
- **Acceptable:** Structural proof that the system handles partial fills
  correctly, combining domain-level evidence (S383) with Spot adapter multi-fill
  aggregation evidence (S394 unit tests), plus demonstration that
  `mapBinanceStatus("PARTIALLY_FILLED")` produces the correct lifecycle
  transition.

### 4.4 Post-200 Reconciliation (Spot)

Same policy as S389: structural proof (code inspection + existing httptest
coverage) is acceptable if the scenario cannot be reliably triggered against
the Spot testnet.

---

## 5. Differences from S389 Original

| Dimension | S389 Original | S396 Refresh |
|---|---|---|
| Venue target | Binance Futures testnet only | Spot-first; Futures deferred |
| Adapter | `BinanceFuturesTestnetAdapter` | `BinanceSpotTestnetAdapter` |
| Stage range | S390--S395 | S397--S401 |
| Blocks | 5 (S390--S394 + S395 gate) | 4 (S397--S400 + S401 gate) |
| Non-goals | 22 (NG-1--NG-22) | 28 (NG-1--NG-28) |
| Governing questions | 12 (unchanged) | 12 (unchanged, retargeted) |
| Capabilities | 10 (unchanged) | 10 (unchanged, retargeted) |
| Preconditions | OMS Foundation only | OMS Foundation + Segmentation Foundation |
| Compose model | Single-instance | Dual-instance (S398) |
| Ingest binding | `binancef` assumed seeded | `binances` must be seeded (S397) |

---

## 6. Links

- Charter refresh: [`testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md`](testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md)
- Original charter: [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md)
- Original capabilities and non-goals: [`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md)
- Segmentation evidence gate: [`binance-spot-futures-segmentation-evidence-gate.md`](binance-spot-futures-segmentation-evidence-gate.md)
- Stage report: [`../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md`](../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md)
