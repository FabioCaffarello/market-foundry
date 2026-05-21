# Live Stack Capabilities, Governing Questions, and Non-Goals

> Companion document to the Live Stack Integration Wave Charter (S332)

## 1. Capability Target

The Live Stack Integration Wave targets a single systemic capability:

> **End-to-end proof that the composed venue execution pipeline operates correctly
> against running NATS infrastructure — consuming orders, submitting to venue,
> publishing fills, and respecting the kill-switch — in a reproducible local stack.**

This is a **verification** wave, not a **feature** wave. No new domain behavior is
introduced. The wave proves that existing, tested components work together when
connected to real infrastructure.

## 2. Governing Questions

Each block in the charter is anchored by governing questions that determine its
scope and define what "done" means.

### GQ-1: Consumer Flow (LSI-1)

| # | Question | Drives |
|---|----------|--------|
| GQ-1.1 | Does the durable consumer receive events published to `EXECUTION_EVENTS` within acceptable latency? | Consumer connection proof |
| GQ-1.2 | Does the actor's `onIntent()` handler execute when the consumer delivers a message? | End-to-end message path |
| GQ-1.3 | Does the health tracker reflect consumer delivery metrics accurately? | Observability verification |
| GQ-1.4 | Does consumer restart preserve durable state (no message loss, no duplicate delivery)? | Durability proof |

### GQ-2: Fill Round-Trip (LSI-2)

| # | Question | Drives |
|---|----------|--------|
| GQ-2.1 | Does `VenueOrderFilledEvent` appear on the NATS stream after a successful venue submit? | Publication proof |
| GQ-2.2 | Does the event subject follow the canonical pattern with correct source/symbol/timeframe routing? | Subject routing correctness |
| GQ-2.3 | Can a downstream consumer deserialize the published event without loss? | Serialization integrity |
| GQ-2.4 | Does the gateway return fill data after the store consumer persists the event? | Composite visibility |

### GQ-3: Kill-Switch Live (LSI-3)

| # | Question | Drives |
|---|----------|--------|
| GQ-3.1 | Does the control KV store connect to the live `EXECUTION_CONTROL` bucket? | KV connection proof |
| GQ-3.2 | Does setting the gate to `halted` block the next submit attempt? | Pre-submit gate proof |
| GQ-3.3 | Does setting the gate to `halted` stop an in-progress retry loop? | Halt checker proof |
| GQ-3.4 | Does returning the gate to `active` restore normal execution? | Recovery proof |
| GQ-3.5 | If the KV bucket is unavailable, does the gate default to active (fail-open)? | Fail-open verification |

### GQ-4: Wave Gate (LSI-4)

| # | Question | Drives |
|---|----------|--------|
| GQ-4.1 | Are all evidence items from LSI-1 through LSI-3 captured in the matrix? | Completeness |
| GQ-4.2 | Do all 202+ existing tests still pass? | Regression gate |
| GQ-4.3 | Does `make smoke-live-stack` pass as a reproducible ceremony? | Smoke canonicalization |
| GQ-4.4 | Are residual risks documented with explicit acceptance rationale? | Risk transparency |

## 3. Non-Goals

The following are **explicitly out of scope** for this wave. They are not deferred
bugs or forgotten items — they are intentional exclusions to keep the wave
focused on infrastructure verification.

### NG-1: Mainnet Activation

**What:** Connecting to Binance production (mainnet) API endpoints.

**Why excluded:** The Foundry is not yet validated for real-money execution. Live
stack integration uses testnet credentials only. Mainnet activation requires a
separate, dedicated ceremony with its own risk assessment, approval gates, and
rollback procedures.

**When appropriate:** After the Live Stack Integration Wave proves the full pipeline
against testnet, a Mainnet Readiness wave may be chartered.

---

### NG-2: Multi-Venue Expansion

**What:** Adding support for venues beyond Binance Futures Testnet (e.g., Bybit,
dYdX, other Binance products).

**Why excluded:** Multi-venue requires adapter abstraction, venue-specific error
mapping, credential management per venue, and routing logic. This is a breadth
expansion that should only happen after depth is fully proven on the single
canonical venue.

**When appropriate:** After live stack integration is proven and operationally stable.

---

### NG-3: OMS (Order Management System)

**What:** Order lifecycle tracking, position management, order amendment, cancel
flows, partial fill handling, order book state.

**Why excluded:** OMS is a large domain that requires its own design phase. The
current execution model is fire-and-forget with fill receipt — sufficient for
proving infrastructure integration but not for production trading.

**When appropriate:** After live stack integration proves the basic execution path,
OMS design may begin as a separate initiative.

---

### NG-4: Portfolio Risk Management

**What:** Position sizing, exposure limits, portfolio-level risk checks, margin
monitoring, drawdown limits.

**Why excluded:** Risk management is a cross-cutting concern that depends on OMS
and portfolio state — neither of which exist yet. Adding risk checks before the
basic execution path is proven against live infrastructure would be premature.

**When appropriate:** After OMS fundamentals are in place.

---

### NG-5: Broad Dashboards and Monitoring UI

**What:** Grafana dashboards, alerting rules, monitoring UIs, operational
dashboards beyond what is needed for smoke verification.

**Why excluded:** The wave focuses on proving infrastructure connectivity, not
operational excellence. Structured logs and health tracker counters provide
sufficient observability for verification purposes.

**When appropriate:** As part of operational hardening after the execution path is
proven end-to-end.

---

### NG-6: Config-Driven Retry and Reconciliation Policies

**What:** Making retry backoff, retry limits, reconciliation timeouts, and
deadline durations configurable via external configuration.

**Why excluded:** Current hardcoded values are sufficient for testnet verification.
Config-driven policies add complexity (config parsing, validation, hot-reload)
that is not needed to prove infrastructure connectivity.

**When appropriate:** When operational experience reveals that default values need
tuning for different environments or conditions.

---

### NG-7: Runtime Redesign

**What:** Changing the actor model, supervisor tree, message routing, or binary
composition architecture.

**Why excluded:** The current runtime architecture is proven and stable. Changing
it during an infrastructure verification wave would invalidate existing test
coverage and introduce unnecessary risk.

**When appropriate:** Only if the live stack integration reveals fundamental
architectural limitations that cannot be addressed incrementally.

---

### NG-8: New Breadth (New Domains or Scopes)

**What:** Adding new actor scopes (e.g., analytics, backtesting, portfolio),
new event streams, or new domain boundaries.

**Why excluded:** This wave is depth-first on the execution path. Adding new
domains would dilute focus and extend the wave beyond its verification purpose.

**When appropriate:** After the execution path is proven end-to-end against live
infrastructure.

## 4. Boundary Conditions

### What counts as "in scope" for this wave:

- Bug fixes discovered during live stack testing that affect existing components
- Minor hardening of smoke scripts to make them reproducible
- Log improvements needed to capture evidence for the gate evaluation
- Test additions that directly prove a governing question

### What requires a scope amendment:

- Any new domain capability
- Any new NATS stream or KV bucket
- Any new binary or actor scope
- Any change to the decorator composition chain
- Any dependency on external systems not already in the stack (Docker Compose)

## 5. Infrastructure Dependencies

The wave depends on infrastructure already defined in the project:

| Component | Source | Status |
|-----------|--------|--------|
| NATS JetStream | Docker Compose | Available |
| NATS KV (EXECUTION_CONTROL) | Seed script | Available |
| ClickHouse | Docker Compose | Available |
| Gateway HTTP | Execute binary | Available |
| Binance Futures Testnet | External | Requires credentials |

No new infrastructure is introduced by this wave.
