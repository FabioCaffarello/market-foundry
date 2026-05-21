# Testnet Venue Execution Proof — Capabilities, Questions, and Non-Goals

**Wave:** Testnet Venue Execution Proof
**Charter stage:** S389
**Date frozen:** 2026-03-22

---

## 1. Governing Questions

These questions define what the wave must answer. Every stage must advance
at least one question. The evidence gate (S395) will evaluate each.

| ID | Question | Target stage |
|---|---|---|
| **TV-Q1** | Does the `venue_live` write-path produce correct lifecycle transitions when the venue accepts and fills a real order? | S390 |
| **TV-Q2** | Do fill records from a real testnet response carry accurate prices, quantities, and fees? | S390 |
| **TV-Q3** | Does the lifecycle correctly transition to `rejected` when the venue returns a real rejection (insufficient margin, invalid params, auth failure)? | S391 |
| **TV-Q4** | Does the `VenueOrderRejectedEvent` carry the real venue rejection code, reason, and HTTP status? | S391 |
| **TV-Q5** | Can the system observe or structurally prove the `partially_filled` transition from a real venue response? | S392 |
| **TV-Q6** | Does quantity monotonicity hold when real partial fills arrive? | S392 |
| **TV-Q7** | Do KV projections, HTTP queries, and ClickHouse rows agree on terminal state after real venue interactions? | S393 |
| **TV-Q8** | Is the ClickHouse rejection writer wired and producing correct rows (RG-1 closure)? | S393 |
| **TV-Q9** | Does the full compose pipeline (derive → execute → store) operate correctly in `venue_live` mode against testnet? | S394 |
| **TV-Q10** | Does the system sustain correct behavior over multiple consecutive order cycles against testnet? | S394 |
| **TV-Q11** | Does the correlation chain (CorrelationID + CausationID) remain intact through real venue interactions? | S390, S391 |
| **TV-Q12** | Does the post-200 reconciliation path recover correctly when body-read fails after a real 200? | S390 |

---

## 2. Capability Map

Each capability links to a governing question and a target stage.

| ID | Capability | Question | Stage | Predecessor evidence |
|---|---|---|---|---|
| **TV-C1** | Real venue acceptance lifecycle | TV-Q1 | S390 | S385: venue_live write-path with httptest |
| **TV-C2** | Real venue fill record fidelity | TV-Q2 | S390 | S384: PriceSource; S385: fill record shape |
| **TV-C3** | Real venue rejection lifecycle | TV-Q3 | S391 | S386: rejection event path with simulated HTTP |
| **TV-C4** | Real venue rejection event fidelity | TV-Q4 | S391 | S386: VenueOrderRejectedEvent contract |
| **TV-C5** | Real venue partial fill lifecycle | TV-Q5, TV-Q6 | S392 | S383: partially_filled transition proven at domain level |
| **TV-C6** | Lifecycle invariant fidelity under real data | TV-Q1–TV-Q6 | S390–S392 | S383: 49/49 transitions; S384: 8 invariant categories |
| **TV-C7** | Persistence consistency under real data | TV-Q7, TV-Q8 | S393 | S387: KV + HTTP consistent; RG-1 open |
| **TV-C8** | Post-200 reconciliation under real conditions | TV-Q12 | S390 | S322: Post200Reconciler proven with httptest |
| **TV-C9** | Compose E2E with real testnet | TV-Q9, TV-Q10 | S394 | S380: E2E with dry-run; RG-7 open |
| **TV-C10** | OMS read-path auditability under real data | TV-Q7 | S393 | S387: composite status query |

---

## 3. Non-Goals

These items are explicitly excluded from the wave. Any work that drifts
toward a non-goal must be rejected or deferred.

### 3.1 Venue and Market Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-1** | Mainnet execution | This wave targets testnet only. Mainnet requires separate risk assessment, credential management, and operational procedures. |
| **NG-2** | Multi-venue support | Only Binance Futures testnet. No second venue adapter, no venue routing, no venue abstraction layer. |
| **NG-3** | Multi-symbol execution | Single symbol scope per execution binary instance. No symbol routing or multiplexing. |
| **NG-4** | Advanced order types | Market orders only. No limit, stop-loss, OCO, or conditional orders. |
| **NG-5** | WebSocket streaming fills | Synchronous REST-based fill retrieval only. Async WebSocket fill streaming is deferred. |

### 3.2 OMS and Lifecycle Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-6** | Full OMS (order management system) | No order book, no order amendment, no order state aggregation across symbols. |
| **NG-7** | Cancel-order API | The `cancelled` status is mapped in the lifecycle but no cancel-order HTTP call is implemented or tested. |
| **NG-8** | Order amendment or replace | Fire-and-forget model. No modify-order capability. |
| **NG-9** | Lifecycle state machine extension | The seven-state machine is frozen. No new states, no new transitions. |
| **NG-10** | Multi-fill accumulation from WebSocket | Partial fills are observed from synchronous REST response only. |

### 3.3 Risk, Portfolio, and Strategy Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-11** | Portfolio risk management | No position tracking, no exposure limits, no margin management. |
| **NG-12** | P&L computation | No profit/loss calculation. Fee realism (RG-3) remains deferred. |
| **NG-13** | Strategy optimization | Strategies produce intents; this wave validates execution of those intents, not strategy logic. |

### 3.4 Infrastructure and Operations Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-14** | Dashboard or UI | No operational dashboards, no web UI, no monitoring frontend. |
| **NG-15** | Alerting or paging | No alert configuration. Observability is via structured logs and NATS streams. |
| **NG-16** | CI/CD pipeline for testnet | Manual execution. No automated testnet deployment pipeline. |
| **NG-17** | Performance benchmarking | Correctness, not throughput. No latency benchmarks or load tests. |
| **NG-18** | Credential rotation or vault integration | Credentials via environment variables. No vault, no rotation, no secret management system. |

### 3.5 Architectural Scope

| ID | Non-goal | Rationale |
|---|---|---|
| **NG-19** | Lifecycle redesign | The lifecycle state machine is proven and frozen. This wave validates it; it does not redesign it. |
| **NG-20** | Actor topology changes | No new actors, no actor splits, no supervision tree changes beyond what is necessary for RG-1 closure. |
| **NG-21** | New NATS streams or KV buckets | No new infrastructure beyond wiring the already-specified ClickHouse rejection writer (RG-1). |
| **NG-22** | Domain model extension | `ExecutionIntent` is frozen. No new fields, no new event types beyond what exists. |

---

## 4. Boundary Conditions

### 4.1 Testnet Credential Requirements

- `MF_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_BINANCE_FUTURES_TESTNET_API_SECRET`
  must be provisioned before S390.
- Credentials are never committed to the repository.
- Testnet account must have sufficient test margin for market order execution.

### 4.2 Partial Fill Feasibility

Partial fills on Binance Futures testnet are difficult to trigger reliably
with market orders (they typically fill instantly). The wave accepts:

- **Preferred:** Real partial fill observation from testnet.
- **Acceptable:** Structural proof that the system handles partial fills
  correctly, combined with domain-level evidence from S383, plus
  demonstration that `mapBinanceStatus("PARTIALLY_FILLED")` produces the
  correct lifecycle transition.

### 4.3 Post-200 Reconciliation

The body-read-failure-after-200 scenario is rare in practice. The wave
accepts structural proof (code inspection + existing httptest coverage)
if the scenario cannot be reliably triggered against testnet.

---

## 5. Links

- Wave charter: [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md)
- Stage report: [`../stages/stage-s389-testnet-venue-execution-proof-charter-report.md`](../stages/stage-s389-testnet-venue-execution-proof-charter-report.md)
