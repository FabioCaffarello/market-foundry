# Venue Readiness — Capabilities, Questions, and Non-Goals

> Companion document to the Venue Readiness Charter (S306).
> Defines the minimum capabilities, governing questions, and explicit non-goals for Phase 30.
> Date: 2026-03-21

---

## 1. Minimum Capabilities

The Venue Readiness Wave delivers exactly six capabilities. Each capability is testable, has a clear definition of done, and maps to one or more governing questions.

### C1: Venue Adapter Contract Hardening

**What**: The `BinanceFuturesTestnetAdapter` is hardened with structured error classification, timeout handling, response validation, and credential safety guarantees.

**Definition of done**:
- All Binance REST API error codes mapped to `problem.Problem` categories (InvalidArgument, Unavailable, Internal).
- Retryable errors explicitly marked via `MarkRetryable()`.
- HTTP timeout configurable; default ≤ 10s.
- Response body size limited (already 64KB cap; validated).
- No credential material appears in error messages, logs, or problem details.
- Unit tests cover: success, auth failure (401/403), rate limit (429), venue rejection (400), server error (5xx), timeout, malformed response.

**Maps to**: VQ1, VQ5

### C2: Fill Model Validation

**What**: Real venue fills map correctly to the `ExecutionIntent` domain model. All Binance order statuses produce valid domain lifecycle transitions. Fill records carry real price, quantity, fee, and venue-sourced timestamp.

**Definition of done**:
- Every Binance order status (NEW, FILLED, PARTIALLY_FILLED, CANCELED, REJECTED, EXPIRED) maps to a valid `domainexec.Status`.
- `FillRecord` fields populated from venue response: `Price` = `avgPrice`, `Quantity` = `executedQty`, `Fee` = `cumQuote` (proxy), `Timestamp` = `updateTime`, `Simulated` = `false`.
- Lifecycle transitions validated: submitted → accepted → filled is the happy path; all terminal states reachable.
- Edge cases covered: zero-quantity fills (CANCELED before execution), partial fills.
- Unit tests for each Binance status → domain status mapping.

**Maps to**: VQ2

### C3: End-to-End Venue Integration

**What**: A complete path from execution intent to persisted real fill, queryable through the composite read model, with no schema changes.

**Definition of done**:
- Execution intent submitted to Binance testnet via `VenuePort.SubmitOrder`.
- Venue receipt parsed into `ExecutionIntent` with real fill data.
- Intent persisted to ClickHouse `execution_intents` table via existing write path.
- Composite read model queries (`/analytical/composite/chain`, `/analytical/composite/funnel`, `/analytical/composite/dispositions`) return real execution data.
- `Simulated` field is `false` in persisted and queried records.
- No ClickHouse schema migration required.

**Maps to**: VQ3, VQ4

### C4: Failure Envelope and Containment

**What**: All venue failure modes are classified, contained, and do not disrupt the pipeline for healthy symbols or non-execution stages.

**Definition of done**:
- Network failures (timeout, DNS, connection refused) classified as `Unavailable` + retryable.
- Authentication failures (401, 403) classified as `InvalidArgument` + non-retryable.
- Rate limit responses (429) classified as `Unavailable` + retryable.
- Venue rejections (400 + Binance error code) classified as `InvalidArgument` + non-retryable.
- Server errors (5xx) classified as `Unavailable` + retryable.
- Malformed responses (invalid JSON, missing fields) classified as `Internal`.
- No failure mode panics, blocks the actor tree, or contaminates other symbols.
- Failure classification logged with structured fields (status code, error code) but no credential material.

**Maps to**: VQ5

### C5: Production Guard Rails Under Real Venue

**What**: The existing safety gate (kill switch + staleness guard) remains enforced for real venue submissions. No test-only relaxation or "convenience" bypasses.

**Definition of done**:
- Kill switch (`ControlGate.IsHalted()`) blocks `SubmitOrder` call to real venue.
- Staleness guard rejects intents older than threshold before venue submission.
- Tests prove both gates block real venue calls, not just paper calls.
- No code path exists that bypasses the safety gate for venue submissions.
- Actor layer enforces gates before calling `VenuePort` — adapter itself has no bypass mechanism.

**Maps to**: VQ6

### C6: Multi-Symbol Venue Isolation

**What**: Multi-symbol operation with real venue maintains the isolation proven in Phase 29. Concurrent venue submissions for different symbols do not interfere.

**Definition of done**:
- 3 symbols (btcusdt, ethusdt, solusdt) submit concurrent venue orders.
- Each symbol's fills are correctly attributed — no cross-symbol fill contamination.
- Composite read model returns correct per-symbol results under concurrent venue load.
- Venue failures for one symbol do not block or contaminate other symbols.
- Partition isolation (`{source}.{symbol}.{timeframe}`) maintained end-to-end with real data.

**Maps to**: VQ7

---

## 2. Governing Questions (VQ1–VQ7)

### VQ1: Does the adapter successfully submit orders and receive fills?

**Scope**: Authenticated REST API call to Binance Futures testnet → order response → fill data extraction.

**Evidence required**:
- Integration test that submits a real market order to testnet.
- Response contains non-zero `orderId`, `avgPrice`, `executedQty`.
- Adapter returns `VenueOrderReceipt` with `Status=Filled` and populated fill record.

**Answerable after**: S307 (adapter hardening) + S309 (E2E integration).

### VQ2: Does the ExecutionIntent lifecycle reflect venue states?

**Scope**: Mapping correctness between Binance order statuses and domain lifecycle.

**Evidence required**:
- Unit test for each Binance status value → domain status.
- Lifecycle transition validation: submitted → accepted → filled (happy path).
- Terminal state handling: CANCELED, REJECTED, EXPIRED all produce terminal domain states.
- PartiallyFilled handling: PARTIALLY_FILLED maps to `StatusPartiallyFilled`, transitions to Filled or Cancelled.

**Answerable after**: S308 (fill model validation).

### VQ3: Do real fills persist without schema changes?

**Scope**: ClickHouse write path accepts real fill data in existing schema.

**Evidence required**:
- Persisted `execution_intents` row contains `Simulated=false`, real price, real quantity, real fee.
- No ALTER TABLE or new columns required.
- Existing indexes and ORDER BY clauses work correctly with real data.

**Answerable after**: S309 (E2E integration).

### VQ4: Does the composite read model work with real data?

**Scope**: Existing 4 composite endpoints return correct results for real execution data.

**Evidence required**:
- `/analytical/composite/chain` returns chain with non-simulated execution stage.
- `/analytical/composite/chains` batch query correctly includes real executions.
- `/analytical/composite/funnel` counts are accurate with real fills.
- `/analytical/composite/dispositions` reflects real fill outcomes.

**Answerable after**: S309 (E2E integration).

### VQ5: Are venue failures classified and contained?

**Scope**: All failure modes produce structured errors; no failure disrupts the pipeline.

**Evidence required**:
- Failure injection tests for each category (network, auth, rate limit, rejection, server error, malformed).
- Each failure produces correct `problem.Problem` with appropriate type and retryable flag.
- Pipeline continues processing for healthy symbols during venue failure.
- No credential material in any error output.

**Answerable after**: S310 (failure envelope).

### VQ6: Does the safety gate remain enforced?

**Scope**: Kill switch and staleness guard block real venue submissions.

**Evidence required**:
- Test: kill switch halted → venue adapter `SubmitOrder` never called.
- Test: stale intent → venue adapter `SubmitOrder` never called.
- Test: active gate + fresh intent → venue adapter `SubmitOrder` called.
- No code path bypasses the safety gate for any adapter implementation.

**Answerable after**: S309 (E2E integration) + S310 (failure envelope).

### VQ7: Does multi-symbol venue operation maintain isolation?

**Scope**: Concurrent venue submissions across 3 symbols produce isolated results.

**Evidence required**:
- 3 symbols submit concurrent orders; each receives correct fills.
- Composite read model returns correct per-symbol results.
- Venue failure for one symbol does not affect other symbols.
- Fill attribution is symbol-correct (no cross-contamination in ClickHouse or read model).

**Answerable after**: S311 (multi-symbol venue isolation).

---

## 3. Explicit Non-Goals

Each non-goal includes a rationale explaining why it is excluded and which future wave, if any, would address it.

### NG-1: Order Management System (OMS)

No order tracking, order book, position state machine, order amendment, or cancellation flows. The adapter is synchronous fire-and-forget: submit → wait → receipt. OMS is a dedicated future wave that builds on proven venue connectivity.

### NG-2: Portfolio-Level Risk Aggregation

No cross-symbol exposure limits, correlation-aware risk, or aggregate position management. Per-symbol risk assessment remains unchanged. Portfolio risk requires real positions and fills to be meaningful — it is blocked by this wave, not part of it.

### NG-3: Multi-Venue / Multi-Exchange

No adapter registry, exchange routing, or venue failover. Single exchange adapter (Binance Futures testnet) only. Multi-venue introduces normalization, routing, and failover concerns that are orthogonal to proving venue readiness.

### NG-4: Limit Orders, Stop Orders, or Conditional Orders

Market orders only. Advanced order types add order state complexity (pending, triggered, partially filled over time) that requires OMS-class infrastructure. Market orders prove the full lifecycle without this complexity.

### NG-5: Asynchronous Fill Reconciliation

No WebSocket user data stream, no fill polling, no async fill matching. Fills are received synchronously in the order response. Async reconciliation is a successor capability that addresses partial fills over time and missed responses.

### NG-6: Operational Dashboards

No Grafana boards, Prometheus metrics, or alerting rules. Observability is validated through existing composite HTTP endpoints. Dashboard work is meaningful only after venue data is flowing — it is a successor wave.

### NG-7: New Signal / Decision / Strategy / Risk Families

No new pipeline stages or families. Existing EMA, Trend, Squeeze families are sufficient to validate venue integration. Family expansion is independent of venue readiness.

### NG-8: Compliance, Regulatory, or Audit Trail

No KYC integration, trade reporting, regulatory audit trails, or compliance checks. These are production operational concerns, not venue connectivity concerns.

### NG-9: Mainnet Deployment

All work targets Binance Futures testnet exclusively. Mainnet promotion is a deployment and operational decision, not an architectural one. The adapter code is identical; the difference is configuration and credential set.

### NG-10: Write-Side Schema Changes

No ClickHouse schema migrations. Real fill data must fit within the existing `execution_intents` table schema. If the existing schema cannot accommodate real data, that is a blocker to be investigated — not a license to add columns.

### NG-11: Performance Optimization

No latency optimization, connection pooling, or throughput tuning. The wave proves correctness, not performance. Optimization is premature until the baseline is measured with real venue data.

### NG-12: Retry Infrastructure

No automatic retry queues, dead letter queues, or retry schedulers. The adapter marks retryable errors; the actor layer decides whether to retry. Building retry infrastructure is a successor capability.

---

## 4. Capability Dependency Matrix

```
C1: Adapter Contract Hardening
 └── C2: Fill Model Validation (depends on hardened adapter responses)
      └── C3: E2E Integration (depends on validated fill model)
           ├── C4: Failure Envelope (depends on working E2E path to inject failures)
           └── C5: Production Guard Rails (depends on working E2E path to test gates)
                └── C6: Multi-Symbol Venue Isolation (depends on all prior capabilities)
```

---

## 5. Gap Bridge: Paper → Venue

The following table maps the exact gaps between current paper operation and venue-ready operation. Each gap is addressed by a specific capability.

| Gap | Paper Behavior | Venue Target | Addressed By |
|-----|---------------|--------------|--------------|
| Fill price | Always `"0"` | Real `avgPrice` from exchange | C2, C3 |
| Fill quantity | Echoes `intent.Quantity` | Real `executedQty` from exchange | C2, C3 |
| Fill fee | Always `"0"` | Real `cumQuote` (proxy) from exchange | C2, C3 |
| Fill timestamp | `time.Now().UTC()` | Venue `updateTime` (exchange clock) | C2, C3 |
| Simulated flag | Always `true` | `false` for real fills | C2, C3 |
| Venue order ID | `paper-{random}` | Real Binance `orderId` | C1, C3 |
| Partial fills | Never occurs | `PARTIALLY_FILLED` status possible | C2 |
| Order rejection | Never occurs | `REJECTED` / `EXPIRED` possible | C2, C4 |
| Network failure | Never occurs | Timeout, DNS, connection errors possible | C4 |
| Auth failure | Never occurs | 401/403 from expired/invalid credentials | C4 |
| Rate limiting | Never occurs | 429 from Binance rate limits | C4 |
| Multi-symbol venue contention | No contention (instant fills) | Concurrent HTTP calls, shared rate limits | C6 |
