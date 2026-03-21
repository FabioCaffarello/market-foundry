# Production Gap Map: Paper Execution → Venue Readiness

> **Stage:** S307 — Production Gap Map
> **Status:** DELIVERED
> **Date:** 2026-03-21
> **Scope:** Structural capability mapping — no implementation

---

## 1. Executive Summary

This document maps the structural gap between market-foundry's current paper execution envelope and the minimum venue readiness target defined in S306. The audit covers eight capability layers, classifying each as **existing** (reusable as-is), **partial** (exists but requires hardening), or **absent** (must be built). The goal is to reduce ambiguity about the paper → venue leap and prepare the ground for contracts and invariants in S308.

**Key Finding:** The architecture is sound and the domain model is well-factored. The gap is not in design but in **production-grade hardening** of the adapter layer, **fill model fidelity**, **failure classification**, and **operational observability**. No architectural redesign is required.

---

## 2. Paper Execution Envelope — Current State

The paper execution pipeline is complete and proven across single-symbol and multi-symbol scenarios (Phases 14, 28, 29). The flow is:

```
RiskAssessment event
  → PaperOrderEvaluator.Evaluate()         [risk primitives → ExecutionIntent]
  → PaperFillSimulator.SimulateFill()       [instant fill, Simulated=true]
  → Publisher.PublishExecution()             [NATS JetStream, dedup key]
  → Consumer → KV materialization           [EXECUTION_PAPER_ORDER_LATEST]
  → VenueAdapterActor.onIntent()            [SafetyGate check]
  → PaperVenueAdapter.SubmitOrder()         [instant receipt, paper-{uuid}]
  → Publisher.PublishFill()                  [EXECUTION_FILL_EVENTS]
  → ClickHouse writer                       [executions table]
  → Composite read model                    [4 analytical endpoints]
```

**What Paper Proves:**
- Domain model lifecycle (Submitted → Filled)
- Deduplication (JetStream message ID + KV monotonicity guard)
- Kill switch enforcement (SafetyGate blocks when halted)
- Staleness rejection (intents older than maxAge blocked)
- Multi-symbol isolation (partition keys, per-actor scoping)
- Analytical read path (ClickHouse composite reader)
- Causal chain integrity (correlation/causation IDs survive boundaries)

**What Paper Does NOT Prove:**
- Real venue HTTP round-trip behavior
- Real fill prices, quantities, fees
- Partial fills, rejections, expirations
- Network failure handling (timeout, DNS, connection reset)
- Authentication failure handling (401/403)
- Rate limiting behavior (429)
- Venue-sourced timestamps
- Concurrent HTTP requests to shared exchange endpoint

---

## 3. Capability Layer Analysis

### Layer 1: Execution Contracts

| Capability | Status | Detail |
|---|---|---|
| `VenuePort` interface | **Existing** | `SubmitOrder(ctx, req) → (receipt, *problem)` — clean, adapter-agnostic |
| `VenueOrderRequest` struct | **Existing** | Symbol, Side, Quantity, Type, Source, Timeframe, CorrelationID |
| `VenueOrderReceipt` struct | **Existing** | VenueOrderID, Status, Intent (with fills), TimestampUTC |
| `ExecutionIntent` domain model | **Existing** | 15-field model with validation, state machine, dedup key |
| `FillRecord` struct | **Existing** | Price, Quantity, Fee, Simulated, Timestamp — schema fits real data |
| `PaperOrderSubmittedEvent` | **Existing** | Event envelope with metadata, correlation/causation |
| `VenueOrderFilledEvent` | **Existing** | Fill event envelope, same schema |
| Order ID in ExecutionIntent | **Absent** | Venue order ID lives in VenueOrderFilledEvent, not in intent |
| Quantity format validation | **Absent** | Stored as string, no decimal validation |

**Assessment:** Contracts are well-designed and venue-agnostic. Minor gaps (order ID, quantity validation) are low-risk for testnet.

### Layer 2: Venue Adapter Semantics

| Capability | Status | Detail |
|---|---|---|
| `PaperVenueAdapter` | **Existing** | Reference implementation, instant fills, Simulated=true |
| `BinanceFuturesTestnetAdapter` | **Partial** | Exists (S90–S93), market orders only, HMAC signing, basic error mapping |
| HTTP request construction | **Partial** | POST to `/fapi/v1/order`, HMAC-SHA256, timestamp nonce — needs hardening |
| Response parsing | **Partial** | Maps orderId, avgPrice, executedQty, cumQuote, status — needs edge-case coverage |
| Binance status → domain mapping | **Partial** | NEW→Accepted, FILLED→Filled, PARTIALLY_FILLED→PartiallyFilled — needs EXPIRED, CANCELED, REJECTED |
| Error classification | **Partial** | 401→InvalidArgument, 429→Unavailable, 5xx→Unavailable — needs enrichment |
| Request timeout | **Partial** | Uses HTTP client timeout, no per-request deadline |
| Response body size cap | **Absent** | No `io.LimitReader` on response body |
| Credential leak prevention | **Existing** | `CredentialSet` never exposes values in logs or errors |
| Retry classification (retryable flag) | **Partial** | Problem types exist but retryable field not consistently set |

**Assessment:** Adapter skeleton exists and is structurally sound. Hardening needed for error classification, response validation, timeout, and body size limits. This is S307/S308 work per the charter.

### Layer 3: Order Lifecycle / OMS

| Capability | Status | Detail |
|---|---|---|
| State machine (valid transitions) | **Existing** | `validTransitions` map: Submitted→Sent→Accepted→Filled/PartiallyFilled/Cancelled |
| `IsTerminal()` check | **Existing** | Filled, Rejected, Cancelled are terminal |
| Partial fill tracking | **Partial** | `Fills []FillRecord` array exists, but no aggregation to `FilledQuantity` |
| Order amendment/cancellation | **Absent** | Not in scope (NG-1) — but state machine supports Cancelled terminal |
| Position state tracking | **Absent** | Not in scope (NG-1) — no open positions ledger |
| Order book / active orders | **Absent** | Not in scope (NG-1) — no tracking of in-flight orders |

**Assessment:** Lifecycle model is sufficient for market-order fire-and-forget within S306 non-goals. OMS features are explicitly deferred (NG-1).

### Layer 4: Idempotency / Deduplication

| Capability | Status | Detail |
|---|---|---|
| JetStream message ID dedup | **Existing** | `exec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}` — 24h server window |
| KV monotonicity guard | **Existing** | Rejects stale (older timestamp) and duplicate (equal timestamp) writes |
| Fill dedup key | **Existing** | `fill:{venueOrderID}:{timestamp_unix}` |
| Venue-side idempotency | **Absent** | No `newClientOrderId` sent to Binance — cannot safely retry on timeout |

**Assessment:** Internal dedup is production-grade. Venue-side idempotency (client order ID) is the critical gap — without it, retrying a timed-out request risks double-submission.

### Layer 5: Retries / Failure Handling

| Capability | Status | Detail |
|---|---|---|
| Problem type taxonomy | **Existing** | InvalidArgument, Unavailable, Internal, NotFound — well-defined |
| Retryable classification | **Partial** | 429 and 5xx marked retryable; others non-retryable — needs systematic tagging |
| Actor-level retry policy | **Absent** | NATS consumer has MaxDeliver=10 for redelivery, but no venue-specific retry |
| Circuit breaker | **Absent** | No per-venue circuit breaker (NG-12 defers retry infrastructure) |
| Timeout handling | **Partial** | HTTP client timeout exists; no context deadline per-request |
| Graceful degradation | **Partial** | Kill switch halts all; no per-symbol degradation on venue failure |

**Assessment:** Error taxonomy exists. Retry infrastructure is explicitly out of scope (NG-12) — adapter marks errors retryable, actor layer decides. Per-symbol degradation is the gap for S310/S311.

### Layer 6: Reconciliation

| Capability | Status | Detail |
|---|---|---|
| Venue order tracking | **Absent** | No venue order ID → internal intent mapping persisted |
| Fill completeness audit | **Absent** | No mechanism to detect missing fills |
| State divergence detection | **Absent** | No comparison of venue state vs. internal state |
| Async fill polling | **Absent** | Not in scope (NG-5) — synchronous fills only |

**Assessment:** Reconciliation is entirely absent. For synchronous market orders on testnet, this is acceptable — the fill is returned in the HTTP response. Async reconciliation (NG-5) is deferred. A minimal "log and verify" reconciliation may surface as a need during E2E integration (S309).

### Layer 7: Production Control / Kill Switch

| Capability | Status | Detail |
|---|---|---|
| Global kill switch (ControlGate) | **Existing** | Active/Halted states, KV-backed, HTTP API, audit trail |
| Staleness guard | **Existing** | Configurable maxAge, rejects stale intents |
| SafetyGate orchestration | **Existing** | Composes kill switch + staleness, returns verdict with reason |
| Per-symbol kill switch | **Absent** | Only global halt — cannot isolate a failing symbol |
| Rate limiting (outbound) | **Absent** | No requests/second cap to venue |
| Order size circuit breaker | **Absent** | No max order size enforcement at adapter level |
| Emergency position liquidation | **Absent** | Not in scope — no OMS (NG-1) |

**Assessment:** Global kill switch is production-grade and proven. Per-symbol controls and rate limiting are gaps but may be post-venue-readiness concerns.

### Layer 8: Production Observability

| Capability | Status | Detail |
|---|---|---|
| Structured logging | **Existing** | slog with actor/symbol/source/timeframe context |
| Health tracker counters | **Existing** | processed, filled, skipped_halt, skipped_stale, errors per actor |
| `/healthz` endpoint | **Existing** | All services expose health |
| `/readyz` endpoint | **Existing** | Gateway checks configctl + evidence + store |
| ClickHouse analytical queries | **Existing** | Execution history, composite chain, funnel, disposition |
| Distributed tracing (OpenTelemetry) | **Absent** | OTel imported by ClickHouse SDK but not instrumented |
| Prometheus metrics | **Absent** | No metrics export endpoint |
| Execution latency histograms | **Absent** | No measurement of intent→fill time |
| Venue response time tracking | **Absent** | No per-request latency measurement |
| Alerting integration | **Absent** | No alert rules or webhook integration |

**Assessment:** Logging and health checks are good. Metrics and tracing are absent but explicitly out of scope (NG-6 — no operational dashboards). For venue readiness, the existing `/healthz` counters plus structured logs are the minimum viable observability.

---

## 4. Consolidated Gap Classification

### Existing & Reusable (No Changes Expected)

| # | Capability | Confidence |
|---|---|---|
| E1 | VenuePort interface + contracts | High |
| E2 | ExecutionIntent domain model + validation | High |
| E3 | FillRecord schema (fits real data) | High |
| E4 | State machine + valid transitions | High |
| E5 | JetStream dedup + KV monotonicity | High |
| E6 | Fill dedup key | High |
| E7 | Global kill switch (ControlGate) | High |
| E8 | Staleness guard | High |
| E9 | SafetyGate orchestration | High |
| E10 | CredentialSet (env var loading, no leaks) | High |
| E11 | Event publishing (NATS JetStream) | High |
| E12 | KV materialization pipeline | High |
| E13 | ClickHouse write path (executions table) | High |
| E14 | Composite read model (4 endpoints) | High |
| E15 | Structured logging | High |
| E16 | Health tracker counters | High |
| E17 | Problem type taxonomy | High |
| E18 | Multi-symbol partition isolation | High |

### Partial — Requires Hardening

| # | Capability | What Exists | What's Missing |
|---|---|---|---|
| P1 | BinanceFuturesTestnetAdapter | Market order submission, HMAC signing, basic status mapping | Edge-case response handling, body size cap, systematic error classification |
| P2 | Binance status → domain mapping | NEW, FILLED, PARTIALLY_FILLED | EXPIRED, CANCELED, REJECTED edge cases |
| P3 | Error classification | 401, 429, 5xx mapped | Retryable flag inconsistent, no malformed response handling |
| P4 | Request timeout | HTTP client-level timeout | No per-request context deadline |
| P5 | Partial fill tracking | Fills array in ExecutionIntent | No aggregation logic for FilledQuantity |
| P6 | Retry classification | Problem types exist | Actor layer has no venue-retry policy (NG-12 accepts this) |
| P7 | Per-symbol degradation | Global kill switch | No per-symbol halt (acceptable for 3-symbol testnet) |

### Absent — Must Be Built

| # | Capability | Impact | Addressed By |
|---|---|---|---|
| A1 | Venue-side client order ID | Cannot safely retry timed-out orders | S307 (adapter hardening) |
| A2 | Response body size cap | Unbounded read on malformed response | S307 (adapter hardening) |
| A3 | Fill model validation (real prices/qty/fees) | Cannot trust fill data | S308 (fill model) |
| A4 | E2E venue integration test | No proof of real submission→fill→persist | S309 (E2E integration) |
| A5 | Failure injection tests | No proof of failure containment | S310 (failure envelope) |
| A6 | Multi-symbol venue isolation test | No proof under concurrent HTTP | S311 (isolation proof) |
| A7 | Reconciliation | No fill completeness audit | Post-venue (accepted gap) |
| A8 | Distributed tracing | No cross-service trace | Post-venue (NG-6) |
| A9 | Metrics export (Prometheus) | No quantitative observability | Post-venue (NG-6) |
| A10 | Per-symbol kill switch | Cannot isolate failing symbol | Post-venue |
| A11 | Outbound rate limiting | Risk of venue rate-limit cascade | Post-venue |

---

## 5. Dependency Chain: Paper → Venue

```
[EXISTING] Domain model + contracts + dedup + kill switch + NATS pipeline
     │
     ▼
[S307] Adapter contract hardening (P1-P4, A1-A2)
     │   - Error classification completeness
     │   - Client order ID for idempotent retry
     │   - Response body cap
     │   - Per-request timeout
     │
     ▼
[S308] Fill model validation (P2, P5, A3)
     │   - All Binance statuses mapped
     │   - Real price/qty/fee/timestamp populated
     │   - Simulated=false
     │   - Partial fill aggregation (if PARTIALLY_FILLED encountered)
     │
     ▼
[S309] E2E venue integration (A4)
     │   - Submit → real fill → ClickHouse → composite read
     │   - No schema changes
     │   - Safety gate enforced with real venue
     │
     ▼
[S310] Failure envelope & containment (A5)
     │   - Network/auth/rate/rejection all classified
     │   - Failure in one symbol doesn't contaminate others
     │
     ▼
[S311] Multi-symbol venue isolation (A6)
     │   - 3 symbols concurrent to testnet
     │   - Correct per-symbol fills
     │   - Composite read correct
     │
     ▼
[S312] Evidence gate — all VQ1-VQ7 answered
```

---

## 6. Explicit Limits of This Map

1. **No implementation** — This document identifies gaps, it does not close them.
2. **Testnet scope only** — Gaps are mapped against Binance Futures testnet, not mainnet.
3. **Market orders only** — Limit, stop, and conditional orders are out of scope (NG-4).
4. **No OMS** — Order tracking, amendment, cancellation are deferred (NG-1).
5. **No async reconciliation** — Synchronous fills only (NG-5).
6. **No dashboards** — Observability via existing endpoints only (NG-6).
7. **No retry infrastructure** — Adapter marks retryable, actor decides (NG-12).
8. **3 symbols maximum** — btcusdt, ethusdt, solusdt only (NG-7).

---

## 7. Preparation for S308

With this gap map in place, S308 (Fill Model Validation & Lifecycle Proof) should:

1. Harden every Binance status → domain state mapping with unit tests.
2. Define fill model validation rules: Price > 0, Quantity > 0, Fee ≥ 0, Timestamp from venue, Simulated=false.
3. Handle PARTIALLY_FILLED: aggregate fills into FilledQuantity.
4. Prove that real fill data fits existing ClickHouse `executions` schema without ALTER TABLE.
5. Validate that composite read model correctly surfaces real fills (non-simulated).

The contracts established in S308 become the invariants that S309 (E2E integration) must satisfy end-to-end.
