# Venue Activation Wave — Charter and Scope Freeze

> **Wave:** Venue Activation
> **Predecessor:** Live Stack Integration Wave (S332–S336) — FULL CLOSURE
> **Charter stage:** S337
> **Status:** OPEN — scope frozen

---

## 1. Wave Identity

The Venue Activation Wave converts the **proven technical pipeline** into
**operational capability** that can be activated in a controlled manner against
a real venue (Binance Futures testnet).

This wave is **not** about deploying to production, expanding to multiple
venues, or building management systems. It is about proving that the Foundry
can execute real orders on a single testnet venue, observe the results through
the existing analytical surface, and control the process via the existing
kill-switch.

### Strategic Position

```
Production Wiring Tranche (S327–S331) — CLOSED
  ↓  all components wired and tested in isolation
Live Stack Integration Wave (S332–S336) — CLOSED
  ↓  pipeline proven against live NATS/ClickHouse with paper simulator
Venue Activation Wave (S337–S34x) — THIS WAVE
  ↓  pipeline proven against real Binance testnet venue
[Future] Production Readiness Wave
  ↓  endurance, monitoring, alerting, operational playbooks
```

---

## 2. Definition: What "Venue Activation" Means

Venue activation is the controlled transition from **paper simulator** to
**real venue adapter** within the execute binary, such that:

1. The execute binary starts with `venue.type = binance_futures_testnet`.
2. Real HTTP requests reach the Binance Futures testnet API.
3. Fill events originating from the real venue propagate through the existing
   pipeline (NATS → writer → ClickHouse → gateway composite surface).
4. The kill-switch remains operational and can halt real venue execution.
5. The entire path is observable through existing smoke scripts and HTTP
   endpoints.

Activation is **configuration-driven** — the code path already exists
(`BinanceFuturesTestnetAdapter`). This wave proves it works end-to-end, not
that it is built.

---

## 3. Frozen Blocks

The wave is divided into five sequential blocks. Each block has a clear
deliverable and evidence criterion. No block may be started until its
predecessor is closed.

| Block | ID | Name | Purpose |
|-------|----|------|---------|
| 1 | VA-1 | Activation Policy and Rollout Model | Define how activation is governed, who can activate, and under what conditions |
| 2 | VA-2 | Canonical Activation Surface and Runtime Controls | Wire the activation configuration, credential injection, and runtime validation |
| 3 | VA-3 | Venue-Active Smoke and Acceptance Scenarios | Extend smoke scripts to validate real venue fills end-to-end |
| 4 | VA-4 | Controlled Live Activation Verification | Execute against Binance testnet and verify the full round-trip |
| 5 | VA-5 | Evidence Gate Final | Formal closure evaluation with evidence matrix |

### Block VA-1: Activation Policy and Rollout Model

**Scope:**
- Define the activation policy: what conditions must be met before the execute
  binary is allowed to start with a real venue adapter.
- Define the rollout model: single-symbol first (btcusdt), then multi-symbol.
- Define the credential model: how API key and secret are injected (env vars,
  not config files).
- Define the rollback model: how to revert to paper simulator.
- Document the operator checklist for activation.

**Evidence criterion:** Policy document exists, operator checklist is
executable, rollback path is tested.

### Block VA-2: Canonical Activation Surface and Runtime Controls

**Scope:**
- Validate that `cmd/execute/run.go` correctly bootstraps
  `BinanceFuturesTestnetAdapter` when `venue.type = binance_futures_testnet`.
- Validate credential injection via environment variables
  (`VENUE_API_KEY`, `VENUE_API_SECRET`).
- Validate that the composed decorator chain (Post200Reconciler →
  RetrySubmitter → BinanceFuturesTestnetAdapter) initializes correctly.
- Validate that the safety gate (kill-switch + staleness guard) operates
  identically with the real adapter.
- Add startup validation: execute binary must refuse to start if venue type is
  `binance_futures_testnet` and credentials are missing.

**Evidence criterion:** Execute binary starts with real adapter, refuses
without credentials, safety gates verified in integration tests.

### Block VA-3: Venue-Active Smoke and Acceptance Scenarios

**Scope:**
- Extend or create a smoke script (`smoke-venue-active.sh`) that:
  - Validates stack readiness with real venue adapter configured.
  - Submits a single order via the derive pipeline.
  - Waits for the fill to appear in ClickHouse.
  - Queries the composite surface for the real venue fill.
  - Exercises the kill-switch halt/resume cycle.
- Define acceptance scenarios:
  - AC-1: Single order submitted and filled on Binance testnet.
  - AC-2: Fill visible in ClickHouse with correct venue_market_order type.
  - AC-3: Composite surface returns venue fill with real exchange data.
  - AC-4: Kill-switch halts real venue execution.
  - AC-5: Resume after halt processes next intent correctly.

**Evidence criterion:** Smoke script passes with real venue adapter, all
acceptance scenarios have test evidence.

### Block VA-4: Controlled Live Activation Verification

**Scope:**
- Execute the full activation sequence:
  1. Start stack with paper simulator (baseline).
  2. Stop execute binary.
  3. Reconfigure to `binance_futures_testnet` with credentials.
  4. Restart execute binary.
  5. Submit orders and observe fills.
  6. Exercise kill-switch.
  7. Revert to paper simulator and confirm clean rollback.
- Record evidence for each step.
- Document any behavioral differences between paper and venue fills.

**Evidence criterion:** Full activation sequence executed successfully,
behavioral delta documented, rollback confirmed clean.

### Block VA-5: Evidence Gate Final

**Scope:**
- Evaluate the evidence matrix for all blocks (VA-1 through VA-4).
- Confirm all acceptance criteria met.
- Confirm all prior invariants held (202+ regression suite green).
- Confirm no new residual gaps that block closure.
- Issue formal wave closure verdict.

**Evidence criterion:** Evidence matrix complete, all tests green, verdict
issued.

---

## 4. Invariants Carried Forward

All 9 invariants from the Production Wiring Tranche remain active. This wave
must not break any of them:

| ID | Invariant |
|----|-----------|
| EC-1 | Deterministic client order ID |
| EC-3 | Correlation/causation preservation |
| F-1 | Fill event contract |
| F-4 | Venue column alignment |
| RF-1 | Round-trip fill visibility |
| PGR-08 | Paper gate registration |
| INV-REC-1 | No duplicate execution |
| INV-RC-1 | Deadline independence |
| INV-OBS-1 | Zero noise on success |

---

## 5. Residual Gaps Addressed by This Wave

Three residual gaps from S336 are naturally resolved by venue activation:

| Gap | S336 Severity | Resolution in This Wave |
|-----|---------------|------------------------|
| G-2: Partial fills with real venue data | Low | Real testnet may produce partial fills; verify handling |
| G-3: Commission uses cumQuote proxy | Low | Real venue fills include actual commission data |
| G-4: Paper bridge subject mapping | Low | Venue-specific intents replace paper bridge dependency |

Two medium-severity gaps (G-1: 24h+ observation, G-5: halt under load) remain
deferred to the Production Readiness Wave.

---

## 6. Scope Boundaries

### What Is In Scope

- Single venue: Binance Futures testnet only.
- Single symbol first (btcusdt), then multi-symbol (btcusdt + ethusdt).
- Configuration-driven activation via existing `venue.type` setting.
- Credential injection via environment variables.
- Existing kill-switch as the primary control mechanism.
- Existing smoke scripts as the validation framework.
- Existing composite surface as the observability surface.

### What Is NOT In Scope

See the companion document:
[venue-activation-capabilities-questions-and-non-goals.md](venue-activation-capabilities-questions-and-non-goals.md)

---

## 7. Stage Ordering

| Stage | Block | Title |
|-------|-------|-------|
| S337 | — | Venue Activation Wave Charter and Scope Freeze (this stage) |
| S338 | VA-1 | Activation Policy and Rollout Model |
| S339 | VA-2 | Canonical Activation Surface and Runtime Controls |
| S340 | VA-3 | Venue-Active Smoke and Acceptance Scenarios |
| S341 | VA-4 | Controlled Live Activation Verification |
| S342 | VA-5 | Evidence Gate Final |

---

## 8. Governance Model

This wave follows the same governance model proven in the Live Stack
Integration Wave:

- **Frozen scope:** No block may be added or removed after this charter.
- **Sequential execution:** Blocks execute in order; no block starts before
  its predecessor closes.
- **Evidence-based closure:** Each block closes with concrete test evidence,
  not assertions.
- **Governing questions:** Each block has explicit questions that must be
  answered with evidence.
- **Non-goals are binding:** Items listed as non-goals may not be pursued
  within this wave.
- **Invariant preservation:** All prior invariants must hold at every block
  boundary.

---

## 9. Entry Conditions

The following conditions must be true before S338 begins:

1. S336 evidence gate is FULL CLOSURE (confirmed).
2. All 202+ regression tests are GREEN.
3. `make smoke-live-stack` passes reproducibly.
4. Binance Futures testnet API credentials are available.
5. This charter is reviewed and accepted.

---

## 10. Exit Conditions

The wave closes when:

1. All 5 blocks have evidence-based closure.
2. At least one real order has been submitted and filled on Binance testnet.
3. The fill is visible through the existing composite surface.
4. The kill-switch has been exercised against real venue execution.
5. Rollback to paper simulator is confirmed clean.
6. All 202+ regression tests remain GREEN.
7. Evidence matrix is complete with no BLOCKED items.
