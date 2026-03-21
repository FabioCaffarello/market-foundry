# Venue Readiness Gap Matrix: Dependencies and Blockers

> **Stage:** S307 — Production Gap Map
> **Status:** DELIVERED
> **Date:** 2026-03-21
> **Companion:** `production-gap-map-from-paper-execution-to-venue-readiness.md`

---

## 1. Gap Matrix

The matrix below classifies every identified gap by capability layer, severity, stage assignment, and dependency chain. Severity uses three levels:

- **Blocker** — Cannot achieve venue readiness without resolving
- **Required** — Must be resolved for quality/safety but has workarounds
- **Deferred** — Explicitly out of scope per S306 non-goals

### Execution Contracts

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| EC-1 | Venue-side client order ID (`newClientOrderId`) | Not sent to Binance | UUID-based client order ID in every request | **Blocker** | S307 | — | Safe retry on timeout |
| EC-2 | Response body size cap | Unbounded `io.ReadAll` | `io.LimitReader(body, 64KB)` | **Required** | S307 | — | Malformed response safety |
| EC-3 | Per-request context deadline | HTTP client timeout only | `context.WithTimeout` per request | **Required** | S307 | — | Predictable timeout behavior |
| EC-4 | Quantity decimal validation | String, no validation | `shopspring/decimal` parse + positive check | Deferred | Post-S312 | — | — |

### Venue Adapter Semantics

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| VA-1 | Error classification completeness | 401, 429, 5xx partially mapped | All Binance error codes classified with retryable flag | **Blocker** | S307 | — | S310 failure envelope |
| VA-2 | Binance EXPIRED status mapping | Not handled | Maps to `Cancelled` or `Rejected` | **Blocker** | S308 | VA-1 | Fill model correctness |
| VA-3 | Binance CANCELED status mapping | Not handled | Maps to `Cancelled` | **Blocker** | S308 | VA-1 | Fill model correctness |
| VA-4 | Binance REJECTED status mapping | Not handled | Maps to `Rejected` | **Blocker** | S308 | VA-1 | Fill model correctness |
| VA-5 | Malformed response handling | No special handling | Classified as `Internal`, non-retryable, logged | **Required** | S307 | EC-2 | Adapter robustness |
| VA-6 | Credential refresh/rotation | Static env vars | Static env vars (acceptable for testnet) | Deferred | Post-S312 | — | — |

### Fill Model

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| FM-1 | Real fill price | Always "0" (paper) | `avgPrice` from Binance response | **Blocker** | S308 | VA-1 | S309 E2E |
| FM-2 | Real fill quantity | Echoes intent.Quantity | `executedQty` from Binance response | **Blocker** | S308 | VA-1 | S309 E2E |
| FM-3 | Real fill fee | Always "0" (paper) | `cumQuote` from Binance response | **Blocker** | S308 | VA-1 | S309 E2E |
| FM-4 | Real fill timestamp | `time.Now().UTC()` | `updateTime` from Binance response | **Blocker** | S308 | VA-1 | S309 E2E |
| FM-5 | Simulated flag | Always `true` | `false` for real venue fills | **Blocker** | S308 | — | S309 E2E |
| FM-6 | Partial fill aggregation | Fills array exists, no rollup | Aggregate FilledQuantity from Fills | **Required** | S308 | VA-2 | Partial fill accuracy |
| FM-7 | Venue order ID in intent | Only in VenueOrderFilledEvent | Propagated to ExecutionIntent for tracing | **Required** | S308 | — | Traceability |

### Order Lifecycle / OMS

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| OL-1 | Order amendment / cancellation | Not implemented | Not in scope (NG-1) | Deferred | — | — | — |
| OL-2 | Position state tracking | Not implemented | Not in scope (NG-1) | Deferred | — | — | — |
| OL-3 | Active orders ledger | Not implemented | Not in scope (NG-1) | Deferred | — | — | — |

### Idempotency / Deduplication

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| ID-1 | Venue-side idempotency | No `newClientOrderId` | UUID-based client order ID per submission | **Blocker** | S307 | EC-1 | Safe retry |
| ID-2 | JetStream dedup (internal) | Fully working | No changes needed | — | — | — | — |
| ID-3 | KV monotonicity guard (internal) | Fully working | No changes needed | — | — | — | — |

### Retries / Failure Handling

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| RF-1 | Retryable flag on all errors | Inconsistent | Every Problem has `Retryable` field set | **Blocker** | S307 | VA-1 | S310 containment |
| RF-2 | Venue-specific retry policy | None (NG-12) | Adapter marks retryable; actor layer decides | Deferred | Post-S312 | — | — |
| RF-3 | Circuit breaker per venue | None (NG-12) | Not in scope | Deferred | Post-S312 | — | — |
| RF-4 | Network failure classification | Partial | Timeout, DNS, connection reset all classified | **Blocker** | S310 | VA-1 | Failure envelope |

### Reconciliation

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| RC-1 | Venue order → intent mapping | Not persisted | Synchronous receipt sufficient for testnet | Deferred | Post-S312 | — | — |
| RC-2 | Fill completeness audit | Not implemented | Not in scope (NG-5) | Deferred | Post-S312 | — | — |
| RC-3 | State divergence detection | Not implemented | Not in scope (NG-5) | Deferred | Post-S312 | — | — |
| RC-4 | Async fill polling | Not implemented | Not in scope (NG-5) | Deferred | Post-S312 | — | — |

### Production Control

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| PC-1 | Global kill switch | Fully working | Enforce under real venue calls | **Blocker** | S309 | — | S310 |
| PC-2 | Staleness guard | Fully working | Enforce under real venue calls | **Blocker** | S309 | — | S310 |
| PC-3 | Per-symbol kill switch | Not implemented | Not required for 3-symbol testnet | Deferred | Post-S312 | — | — |
| PC-4 | Outbound rate limiting | Not implemented | Not required for testnet | Deferred | Post-S312 | — | — |
| PC-5 | Order size circuit breaker | Not implemented | Not required for testnet | Deferred | Post-S312 | — | — |

### Observability

| ID | Gap | Current State | Target State | Severity | Stage | Depends On | Blocks |
|---|---|---|---|---|---|---|---|
| OB-1 | Venue response logging | Partial | Log venue response status, latency, error class (no credentials) | **Required** | S307 | — | Debugging |
| OB-2 | Health tracker for venue calls | Not instrumented | venue_submitted, venue_filled, venue_error counters | **Required** | S309 | — | Operational awareness |
| OB-3 | Distributed tracing (OTel) | Not instrumented | Not in scope (NG-6) | Deferred | Post-S312 | — | — |
| OB-4 | Prometheus metrics | Not implemented | Not in scope (NG-6) | Deferred | Post-S312 | — | — |
| OB-5 | Execution latency histograms | Not implemented | Not in scope (NG-6) | Deferred | Post-S312 | — | — |
| OB-6 | Alerting integration | Not implemented | Not in scope (NG-6) | Deferred | Post-S312 | — | — |

---

## 2. Blocker Summary

| ID | Description | Stage | Critical Path? |
|---|---|---|---|
| EC-1 | Client order ID for venue idempotency | S307 | Yes — blocks safe retry |
| VA-1 | Error classification completeness | S307 | Yes — blocks S308, S310 |
| VA-2 | EXPIRED status mapping | S308 | Yes — blocks fill correctness |
| VA-3 | CANCELED status mapping | S308 | Yes — blocks fill correctness |
| VA-4 | REJECTED status mapping | S308 | Yes — blocks fill correctness |
| FM-1 | Real fill price | S308 | Yes — blocks S309 E2E |
| FM-2 | Real fill quantity | S308 | Yes — blocks S309 E2E |
| FM-3 | Real fill fee | S308 | Yes — blocks S309 E2E |
| FM-4 | Real fill timestamp | S308 | Yes — blocks S309 E2E |
| FM-5 | Simulated=false | S308 | Yes — blocks S309 E2E |
| ID-1 | Venue-side idempotency | S307 | Yes — blocks safe retry |
| RF-1 | Retryable flag consistency | S307 | Yes — blocks S310 |
| RF-4 | Network failure classification | S310 | Yes — blocks failure envelope |
| PC-1 | Kill switch under real venue | S309 | Yes — blocks production control proof |
| PC-2 | Staleness guard under real venue | S309 | Yes — blocks production control proof |

**Total blockers:** 15
- S307 responsibility: 5 (EC-1, VA-1, RF-1, ID-1 overlap with EC-1, EC-2/EC-3 as required)
- S308 responsibility: 6 (VA-2, VA-3, VA-4, FM-1 through FM-5)
- S309 responsibility: 2 (PC-1, PC-2)
- S310 responsibility: 1 (RF-4)
- S311 responsibility: 1 (multi-symbol venue isolation — implicit blocker for S312)

---

## 3. Dependency Graph

```
                    ┌─────────────────┐
                    │  S306 (Charter)  │
                    │     DONE         │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S307 (Adapter  │
                    │  Hardening)     │
                    │                 │
                    │  EC-1, EC-2,    │
                    │  EC-3, VA-1,    │
                    │  VA-5, RF-1,    │
                    │  OB-1           │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S308 (Fill     │
                    │  Model)         │
                    │                 │
                    │  VA-2, VA-3,    │
                    │  VA-4, FM-1,    │
                    │  FM-2, FM-3,    │
                    │  FM-4, FM-5,    │
                    │  FM-6, FM-7     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S309 (E2E      │
                    │  Integration)   │
                    │                 │
                    │  PC-1, PC-2,    │
                    │  OB-2           │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S310 (Failure  │
                    │  Envelope)      │
                    │                 │
                    │  RF-4           │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S311 (Multi-   │
                    │  Symbol Venue)  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  S312 (Gate)    │
                    │  VQ1-VQ7        │
                    └─────────────────┘
```

### Cross-Stage Dependencies

| From | To | Dependency Type | Description |
|---|---|---|---|
| S307 | S308 | **Hard** | Adapter error classification (VA-1) must exist before fill mapping (VA-2/3/4) |
| S307 | S309 | **Hard** | Client order ID (EC-1) + error classification (VA-1) required for real submission |
| S308 | S309 | **Hard** | Fill model (FM-1 through FM-5) must be validated before E2E persistence |
| S309 | S310 | **Hard** | E2E working path required before testing failure injection |
| S309 | S311 | **Transitive** | Single-symbol E2E must work before multi-symbol |
| S310 | S311 | **Hard** | Failure containment must be proven before multi-symbol isolation |
| S311 | S312 | **Hard** | All capabilities proven before evidence gate |

### External Dependencies

| Dependency | Type | Impact | Mitigation |
|---|---|---|---|
| Binance Futures testnet availability | External | S309-S311 require live testnet | Skip/retry in CI; manual validation acceptable |
| Binance testnet API key/secret | Credential | Cannot run venue tests without | Env var convention exists (MF_VENUE_BINANCE_*) |
| NATS server | Infrastructure | All stages require running NATS | Docker Compose provides; CI has nats service |
| ClickHouse | Infrastructure | S309 persistence requires ClickHouse | Docker Compose provides; optional in gateway |

---

## 4. Risk Assessment

### High Risk

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Binance testnet returns unexpected status codes | Medium | Blocks S308 fill mapping | Unit tests with mocked HTTP responses |
| Testnet rate limiting during multi-symbol tests | Medium | Blocks S311 | Sequential submission with small delays |
| Partial fills on testnet market orders | Low | Requires FM-6 (partial fill aggregation) | Market orders on testnet typically fill instantly |

### Medium Risk

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Fill data doesn't fit ClickHouse schema | Low | Requires ALTER TABLE (violates NG-10) | Pre-validate with S308 unit tests |
| Kill switch enforcement gap under real HTTP latency | Low | Safety gate timing mismatch | S309 must test gate enforcement with real venue |
| Actor redelivery storm after venue timeout | Medium | Counters inflate, duplicate submissions | Client order ID (EC-1) prevents double-fill |

### Low Risk (Deferred)

| Risk | Probability | Impact | Stage |
|---|---|---|---|
| No reconciliation detects lost fills | Low (testnet) | Acceptable for testnet | Post-S312 |
| No per-symbol kill switch | Low (3 symbols) | Global halt is sufficient | Post-S312 |
| No outbound rate limiting | Low (testnet) | Testnet limits are generous | Post-S312 |

---

## 5. Deferred Gaps — Post-Venue-Readiness Backlog

These gaps are explicitly out of S306 scope but documented here for traceability:

| ID | Gap | Non-Goal | Future Wave |
|---|---|---|---|
| OL-1 | OMS (order tracking, amendment, cancellation) | NG-1 | OMS wave |
| OL-2 | Portfolio risk aggregation | NG-2 | Portfolio risk wave |
| VA-6 | Multi-venue routing / adapter registry | NG-3 | Multi-venue wave |
| EC-4 | Limit/stop/conditional orders | NG-4 | Advanced orders wave |
| RC-1–4 | Async fill reconciliation | NG-5 | Reconciliation wave |
| OB-3–6 | Operational dashboards, tracing, metrics | NG-6 | Operational maturity wave |
| — | New families/symbols | NG-7 | Symbol expansion wave |
| — | Compliance/regulatory | NG-8 | Compliance wave |
| — | Mainnet deployment | NG-9 | Mainnet wave |
| — | Schema changes | NG-10 | Schema evolution wave |
| RF-2–3 | Retry infrastructure / circuit breakers | NG-12 | Resilience wave |

---

## 6. Acceptance Criteria for This Document

- [x] Every identified gap has a unique ID
- [x] Every gap is classified as Blocker / Required / Deferred
- [x] Every blocker is assigned to a specific stage (S307–S312)
- [x] Dependencies between stages are explicit
- [x] External dependencies are listed with mitigation
- [x] Risks are assessed with probability and impact
- [x] Deferred gaps trace back to S306 non-goals
- [x] No implementation is included — this is a map, not a build
