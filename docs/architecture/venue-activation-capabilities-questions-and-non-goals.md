# Venue Activation Wave — Capabilities, Questions, and Non-Goals

> **Wave:** Venue Activation
> **Companion to:** [venue-activation-wave-charter-and-scope-freeze.md](venue-activation-wave-charter-and-scope-freeze.md)
> **Charter stage:** S337

---

## 1. Target Capabilities

The Venue Activation Wave delivers exactly five capabilities. Each capability
maps to a wave block and has a clear, testable definition.

### CAP-VA-1: Activation Policy

**Definition:** A documented, executable policy that governs when and how the
execute binary transitions from paper simulator to real venue adapter.

**What it includes:**
- Pre-activation checklist (infrastructure health, credentials, kill-switch
  state).
- Activation trigger (configuration change + binary restart).
- Rollback procedure (revert configuration + binary restart).
- Operator responsibility matrix (who activates, who monitors, who reverts).

**What it does NOT include:**
- Automated activation (no CI/CD trigger, no API-driven switch).
- Gradual rollout (no percentage-based traffic splitting).
- Multi-venue activation sequencing.

### CAP-VA-2: Runtime Activation Surface

**Definition:** The execute binary correctly initializes the real venue adapter
when configured, refuses to start without credentials, and applies all
existing safety gates identically.

**What it includes:**
- `BinanceFuturesTestnetAdapter` bootstrap path validated.
- Credential injection via `VENUE_API_KEY` and `VENUE_API_SECRET`.
- Startup validation: missing credentials → binary exits with clear error.
- Composed decorator chain: Post200Reconciler → RetrySubmitter →
  BinanceFuturesTestnetAdapter.
- Safety gate (kill-switch + staleness guard) operates identically.

**What it does NOT include:**
- Hot-reload of venue type at runtime.
- Credential rotation without restart.
- Multiple simultaneous venue adapters.

### CAP-VA-3: Venue-Active Smoke Validation

**Definition:** A smoke script that proves the entire pipeline works with a
real venue adapter, from order submission through fill persistence to composite
surface visibility.

**What it includes:**
- Stack readiness validation with real adapter configured.
- Single-order round-trip: submit → fill → persist → read.
- Kill-switch exercise: halt → confirm → resume → confirm.
- Acceptance scenarios with pass/fail criteria.

**What it does NOT include:**
- Load testing or stress testing.
- Multi-symbol concurrent validation (single-symbol first).
- Automated regression suite against real venue (too slow, too flaky).

### CAP-VA-4: Controlled Activation Verification

**Definition:** End-to-end proof that a human operator can activate real venue
execution, observe results, exercise control, and revert cleanly.

**What it includes:**
- Full activation sequence: paper → venue → observe → control → revert.
- Behavioral delta documentation: paper vs. real venue differences.
- Evidence artifacts for each step.

**What it does NOT include:**
- Unattended activation.
- Multi-day continuous operation.
- Performance benchmarking.

### CAP-VA-5: Evidence Gate

**Definition:** Formal closure evaluation confirming all capabilities are
delivered with test evidence and all invariants hold.

---

## 2. Governing Questions

Each block has governing questions that must be answered with concrete
evidence. A question is answered when a test, script output, or documented
observation demonstrates the answer.

### Block VA-1: Activation Policy and Rollout Model

| ID | Question | Evidence Type |
|----|----------|---------------|
| GQ-VA-1.1 | Is the pre-activation checklist executable and complete? | Checklist document + dry-run execution |
| GQ-VA-1.2 | Can an operator activate venue execution by changing configuration alone? | Configuration change + binary restart observation |
| GQ-VA-1.3 | Can an operator revert to paper simulator cleanly? | Rollback execution + smoke validation |
| GQ-VA-1.4 | Are responsibilities clearly assigned for activation and revert? | Policy document with named roles |

### Block VA-2: Canonical Activation Surface and Runtime Controls

| ID | Question | Evidence Type |
|----|----------|---------------|
| GQ-VA-2.1 | Does the execute binary start with `BinanceFuturesTestnetAdapter` when configured? | Integration test or binary log |
| GQ-VA-2.2 | Does the execute binary refuse to start without credentials? | Integration test or binary exit observation |
| GQ-VA-2.3 | Does the composed decorator chain initialize correctly with the real adapter? | Integration test |
| GQ-VA-2.4 | Does the kill-switch block real venue execution identically to paper? | Integration test (LF-3 variant with real adapter) |
| GQ-VA-2.5 | Does the staleness guard operate identically with the real adapter? | Integration test |

### Block VA-3: Venue-Active Smoke and Acceptance Scenarios

| ID | Question | Evidence Type |
|----|----------|---------------|
| GQ-VA-3.1 | Does a single order reach Binance testnet and return a fill? | Smoke script output + NATS message |
| GQ-VA-3.2 | Does the fill persist to ClickHouse with correct `venue_market_order` type? | ClickHouse query result |
| GQ-VA-3.3 | Does the composite surface return real venue fill data? | HTTP response body |
| GQ-VA-3.4 | Does the kill-switch halt real venue execution during smoke? | Smoke script Phase 7 equivalent with real adapter |
| GQ-VA-3.5 | Do all five acceptance scenarios pass? | Smoke script exit code + scenario evidence |

### Block VA-4: Controlled Live Activation Verification

| ID | Question | Evidence Type |
|----|----------|---------------|
| GQ-VA-4.1 | Does the full activation sequence complete without errors? | Step-by-step execution log |
| GQ-VA-4.2 | Are behavioral differences between paper and venue fills documented? | Delta document |
| GQ-VA-4.3 | Does rollback to paper simulator restore previous behavior? | Smoke validation after rollback |
| GQ-VA-4.4 | Are there new residual gaps introduced by real venue interaction? | Gap analysis |

### Block VA-5: Evidence Gate Final

| ID | Question | Evidence Type |
|----|----------|---------------|
| GQ-VA-5.1 | Are all VA-1 through VA-4 governing questions answered? | Evidence matrix |
| GQ-VA-5.2 | Do all 202+ regression tests remain GREEN? | Test run output |
| GQ-VA-5.3 | Does `make smoke-live-stack` still pass? | Script output |
| GQ-VA-5.4 | Does the new venue-active smoke pass? | Script output |
| GQ-VA-5.5 | Are there BLOCKED items in the evidence matrix? | Matrix review |

---

## 3. Non-Goals

The following items are **explicitly excluded** from this wave. They are not
deferred — they are out of scope by design.

### NG-1: Mainnet Activation

**What:** Connecting to Binance Futures mainnet (real money).
**Why excluded:** Mainnet requires a separate risk assessment, compliance
review, and operational readiness certification that is outside the scope of
a technical verification wave. Testnet activation is the prerequisite.

### NG-2: Multi-Venue Expansion

**What:** Adding venue adapters for exchanges other than Binance Futures
testnet.
**Why excluded:** Multi-venue requires adapter abstraction validation, venue
discovery, routing policies, and cross-venue deduplication. Single-venue
activation must be proven first. The architecture supports multi-venue — this
wave proves the single-venue path.

### NG-3: Order Management System (OMS)

**What:** Order lifecycle management, position tracking, P&L calculation,
order amendment, order cancellation.
**Why excluded:** OMS is a large, separate domain that depends on venue
activation being proven. It introduces state management, reconciliation, and
reporting concerns that are orthogonal to activation verification.

### NG-4: Portfolio Risk Management

**What:** Position sizing, exposure limits, drawdown controls, portfolio-level
risk metrics.
**Why excluded:** Portfolio risk depends on OMS (which depends on venue
activation). It introduces cross-symbol aggregation and real-time risk
calculation that are out of scope.

### NG-5: Dashboards and Monitoring UI

**What:** Grafana dashboards, real-time monitoring UI, alerting rules,
operational dashboards.
**Why excluded:** Monitoring and dashboards are a production readiness concern.
This wave validates through smoke scripts and HTTP endpoints, not dashboards.
Dashboards add value after activation is proven and continuous operation begins.

### NG-6: New Breadth or Domain Expansion

**What:** Adding new signal families, new strategy types, new decision models,
new analytical surfaces.
**Why excluded:** This wave is depth-focused (proving the existing pipeline
with a real venue), not breadth-focused (expanding pipeline capabilities).

### NG-7: Runtime Redesign

**What:** Changing the actor model, replacing Hollywood, migrating from
NATS JetStream, restructuring the binary boundaries.
**Why excluded:** The runtime has been proven through two verification waves.
Redesign is not warranted and would invalidate existing evidence.

### NG-8: Automated Activation / Feature Flags

**What:** API-driven venue type switching, feature flag systems, gradual
rollout mechanisms, A/B testing between paper and venue.
**Why excluded:** Activation is a deliberate operator action (configuration +
restart). Automation adds complexity without proportional value at this stage.
The kill-switch provides sufficient runtime control.

### NG-9: Per-Symbol or Per-Family Gate Isolation

**What:** Independent kill-switches per symbol (btcusdt, ethusdt) or per
execution family (paper_order, venue_market_order).
**Why excluded:** The global gate is an intentional design decision (S336 G-6).
Per-symbol gates add complexity; global halt is the correct safety posture for
single-venue testnet activation.

### NG-10: WebSocket or Streaming Fills

**What:** Real-time fill streaming via WebSocket or Server-Sent Events.
**Why excluded:** REST polling through the composite surface is sufficient for
testnet activation verification. Streaming is a production readiness or
performance optimization concern.

---

## 4. Risk Registry

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Binance testnet API unavailable | Medium | Blocks VA-3/VA-4 | Document fallback; testnet outages are temporary |
| Testnet rate limits interfere with smoke | Low | Slows VA-3 | Smoke submits minimal orders; retry with backoff |
| Real fills have unexpected schema | Low | Blocks VA-3 | BRT-18/19 column alignment already verified; adapter maps fields |
| Credential leak in logs or traces | Medium | Security incident | Credentials injected via env vars; audit log output for secrets |
| Paper bridge migration introduces regression | Low | Breaks VA-2 | LF-1 through LF-4 tests serve as regression gate |
| Post200Reconciler fails against real API | Low | Blocks VA-4 | S322 adapter handles body-read-failure; integration test before live |

---

## 5. Dependencies

| Dependency | Owner | Status |
|------------|-------|--------|
| Binance Futures testnet API credentials | Operator | Required before S339 |
| `BinanceFuturesTestnetAdapter` implementation | Codebase | EXISTS (`internal/application/execution/binance_futures_testnet_adapter.go`) |
| Post200Reconciler implementation | Codebase | EXISTS (`internal/application/execution/post200_reconciler.go`) |
| Kill-switch control surface | Codebase | EXISTS and PROVEN (S335) |
| Smoke-live-stack script | Codebase | EXISTS and PROVEN (S335) |
| 202+ regression tests | Codebase | GREEN (S336) |

---

## 6. Success Metrics

The wave succeeds when:

1. At least one real order is submitted to Binance Futures testnet and a fill
   is returned.
2. The fill propagates through NATS → writer → ClickHouse → gateway composite
   surface without code changes to the pipeline.
3. The kill-switch halts and resumes real venue execution.
4. An operator can activate, observe, control, and revert venue execution
   using documented procedures.
5. All prior invariants and regression tests hold.
