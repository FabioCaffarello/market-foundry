# OMS Foundation — Capabilities, Questions, and Non-Goals

> Companion document: [OMS Foundation Wave — Charter and Scope Freeze](oms-foundation-wave-charter-and-scope-freeze.md)

## Capabilities Under Proof

| ID | Capability | Acceptance criterion | Target stage |
|---|---|---|---|
| OMS-C1 | Lifecycle state machine enforces all transition invariants | Every valid and invalid transition pair covered by automated test | S383 |
| OMS-C2 | Terminal state finality (absorbing states) | No test or runtime path transitions out of `filled`, `rejected`, `cancelled` | S383 |
| OMS-C3 | Fill-status consistency (fills present iff status requires them) | FR-1 through FR-9 invariants from S309 covered by test | S383 |
| OMS-C4 | Quantity monotonicity (never decreases, never exceeds requested) | QM-1 through QM-3 invariants covered by test | S383 |
| OMS-C5 | Price realism in dry-run fills | DryRunSubmitter fills use last observed market price, not zero | S383 |
| OMS-C6 | Write-path correctness under dry_run mode | submitted → accepted → filled with Simulated=true and realistic price | S384 |
| OMS-C7 | Write-path correctness under paper_simulator mode | submitted → accepted → filled with Simulated=true | S384 |
| OMS-C8 | Write-path correctness under venue_live mode (testnet) | submitted → accepted → filled with Simulated=false and venue fill | S384 |
| OMS-C9 | Safety gate enforcement across all modes | Halted gate blocks submission; stale intent blocks submission | S384 |
| OMS-C10 | Correlation chain preservation across write-path | CorrelationID and CausationID survive from strategy event to fill event | S384 |
| OMS-C11 | KV materialization reflects terminal state and fills | Latest KV entry matches terminal ExecutionIntent | S385 |
| OMS-C12 | ClickHouse row reflects terminal state and fills | Analytical row matches terminal ExecutionIntent including all fill details | S385 |
| OMS-C13 | HTTP query returns consistent terminal view | Gateway query returns same data as KV and ClickHouse for terminal order | S385 |
| OMS-C14 | Fill model completeness (paper + venue + partial) | All fill shapes representable and queryable | S385 |
| OMS-C15 | End-to-end OMS lifecycle under live data | Live exchange → derive → execute → fill → persist → query lifecycle proven in compose | S386 |
| OMS-C16 | Correlation chain traceable from strategy to query | Compose smoke traces a single correlation chain through all binaries | S386 |
| OMS-C17 | Multi-binary sustained stability during OMS lifecycle | 5+ minutes of continuous operation with live triggers, no state inconsistency | S386 |

## Governing Questions

| ID | Question | How answered | Target stage |
|---|---|---|---|
| OMS-Q1 | Does the seven-state lifecycle, as implemented, enforce all S309 invariants without gaps? | Exhaustive transition-matrix tests + invariant coverage matrix | S383 |
| OMS-Q2 | Can dry-run fills carry realistic prices without introducing external API dependencies? | Implementation using last-observed price from NATS KV or in-memory cache | S383 |
| OMS-Q3 | Does the composed write-path (SafetyGate → DryRunSubmitter → RetrySubmitter → Adapter) produce correct state transitions for every execution mode? | Per-mode integration tests with explicit state-path assertions | S384 |
| OMS-Q4 | Do safety gates (kill switch + staleness) block correctly regardless of execution mode? | Cross-mode integration tests with gate manipulation | S384 |
| OMS-Q5 | Do the three persistence surfaces (KV, ClickHouse, HTTP) agree on terminal order state? | Consistency tests comparing all three surfaces for the same order | S385 |
| OMS-Q6 | Is the fill model sufficient to represent paper fills, venue fills, and partial fills without schema extension? | Evidence: all fill shapes persisted and queryable without migration | S385 |
| OMS-Q7 | Can the full OMS lifecycle execute end-to-end with live market data as the trigger? | Compose smoke: live aggTrade → terminal fill → queryable result | S386 |
| OMS-Q8 | Is the correlation chain intact from strategy event through fill event to query response? | Compose smoke with explicit correlation ID tracing | S386 |
| OMS-Q9 | Does the system maintain state consistency under sustained live operation (5+ minutes)? | Compose smoke duration test with post-run consistency check | S386 |

## Non-Goals

### NG-1 — Full OMS

**What:** Complete order management system with order book, amendments, user-initiated cancellation, smart order routing, or allocation engine.

**Why out:** The Foundry is a signal-to-execution pipeline (GP-1 from S309). ExecutionIntent is fire-and-forget. Building broker/EMS capabilities is a different product.

### NG-2 — Portfolio Risk and Position Tracking

**What:** Portfolio-level aggregation, position sizing beyond per-intent risk, drawdown tracking across orders, or P&L computation.

**Why out:** Position tracking requires aggregation across intents, which contradicts the per-intent isolation model. Deferred to a dedicated wave after OMS foundation is proven.

### NG-3 — Multi-Venue

**What:** Adding exchange adapters beyond Binance Futures testnet, or routing orders across venues.

**Why out:** Architecture is parametric (proven by activation surface model), but multi-venue is mechanical extension, not foundation work. Premature before the single-venue OMS lifecycle is fully proven.

### NG-4 — Mainnet Trading

**What:** Any code path, configuration, or test that submits orders to Binance Futures mainnet or any production venue.

**Why out:** Four independent fail-closed layers (FC-1 through FC-4) prevent this by design. Mainnet enablement requires a separate safety ceremony.

### NG-5 — Advanced Order Types

**What:** Limit orders, stop orders, OCO, trailing stops, iceberg orders, or any type beyond market orders.

**Why out:** S309 explicitly limits scope to market orders (GP-5). Order type extension is post-foundation work.

### NG-6 — Order Amendments and Cancellations

**What:** Modifying or cancelling orders after submission, whether user-initiated or system-initiated.

**Why out:** Fire-and-forget model (GP-2 from S309). The Foundry does not initiate cancellations; only venue-side cancellation is represented.

### NG-7 — Async Order Lifecycle (WebSocket Fills)

**What:** WebSocket-based asynchronous fill reporting, order status streaming, or event-driven fill updates from venue.

**Why out:** Synchronous submit → response model is the current contract. Async fills are a post-S387 evolution candidate.

### NG-8 — Dashboards and UI

**What:** Web dashboards, monitoring UIs, trade blotters, or visual order tracking beyond the existing HTTP query endpoints.

**Why out:** The wave proves programmatic query correctness. Visual presentation is orthogonal.

### NG-9 — Operational Hardening (Latency, Throughput, Backpressure)

**What:** Latency measurement, throughput benchmarks, ingestion backpressure, or production-scale load testing.

**Why out:** These are accepted gaps (G4, G5, G6) from S381. Operational hardening is a parallel concern, not an OMS foundation concern.

### NG-10 — New Binaries, Streams, or Families

**What:** Introducing new service binaries, NATS JetStream streams, or codegen families.

**Why out:** The existing four-binary topology (derive, execute, store, gateway) and two execution stream families (EXECUTION_EVENTS, EXECUTION_FILL_EVENTS) are sufficient. Invariant OMS-I8, OMS-I9.

### NG-11 — Multi-Account

**What:** Support for multiple trading accounts, sub-accounts, or account-level isolation.

**Why out:** Single-account model is sufficient for foundation proof. Multi-account is post-foundation scope.

### NG-12 — Historical Order Search

**What:** Full-text search, filtered historical queries, or analytical reporting across order history beyond latest-state queries.

**Why out:** ClickHouse provides the analytical surface, but building a search/reporting layer is not foundation work.

### NG-13 — Retry Strategy Redesign

**What:** Redesigning the retry/backoff model in RetrySubmitter or introducing circuit-breaker patterns.

**Why out:** RetrySubmitter is proven (S321–S326). This wave exercises it, not redesigns it.

### NG-14 — State Machine Extension

**What:** Adding new states, transitions, or lifecycle phases beyond the seven-state model from S309.

**Why out:** The seven states and their transitions are frozen. If the existing model is insufficient, that is a finding, not a fix within this wave.
